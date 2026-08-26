# 🚀  RCEMaster v5.1 - 100% Complete, Unified & Ultra-Powered

**Ultimate Adaptive Remote Code Execution Engine.**


`RCEMaster` is an advanced, AI-inspired cybersecurity tool written in **Golang**. It automatically detects target tech stacks and launches adaptive RCE attacks using 200+ vectors including command injection, SSTI, deserialization, blind RCE, and header-based attacks.


  ____  _____ _____    __  __           _      
 |  _ \| ____|_   _|  |  \/  | __ _ _ __ | |_ ___
 | |_) |  _|   | |    | |\/| |/ _` | '_ \| __/ _ \
 |  _ <| |___  | |    | |  | | (_| | | | | ||  __/
 |_| \_\_____| |_|    |_|  |_|\__,_|_| |_|\__\___|
====================================================
 [!] RCEMaster v2.0: Ultimate RCE Engine
 [!] Human-Like Adaptive Thinking | 200+ Vectors
 [!] Command Injection | SSTI | Deserialization | Blind RCE
 [!] File Upload | Header Injection | WAF Evasion
====================================================

[*] Target: https://target.com
[*] Phase 1: Detecting Tech Stack & WAF...
[+] Detected: php  wordpress | WAF: cloudflare
[*] Phase 2: Starting Adaptive RCE Attack...
[*] Trying Command Injection...
[*] Trying SSTI (php)...
[*] Trying Header-Based RCE...
[*] Trying Blind RCE (Time-based & OOB)...
[*] Trying Deserialization Attacks...

========== RCE SUCCESSFUL ==========
[RCE] https://target.com/?template={{7*7}}
   -> Vector    : SSTI (twig)
   -> Payload   : {{7*7}}
   -> Evidence  : 49
   -> Confidence: High

=======================================
[+] Total RCE Vectors: 1


## ✨ Features

### 🔥 Human-Like Adaptive Thinking

- **Tech Stack Detection:** Automatically identifies PHP, Java, Python, Node.js, .NET
- **WAF Fingerprinting:** Detects Cloudflare, AWS WAF, Akamai, Imperva
- **Adaptive Vector Selection:** Chooses best attack vectors based on detected stack
- **Fallback Mechanisms:** If one method fails, automatically tries next best approach

### 💥 200+ Attack Vectors

#### Command Injection (50+ Payloads)

- Basic: `;id`, `|id`, `&&id`, `` `id` ``, `$(id)`
- Encoding Bypasses: Base64, variable expansion, `$IFS`
- Reverse Shells: Bash, Python, Perl, Netcat
- Time-based Blind: `sleep`, `ping`
- OOB/DNS Exfiltration: `curl`, `nslookup`, `dig`

#### SSTI (80+ Payloads across 10+ Engines)

- **Jinja2** (Python/Flask): `{{7*7}}`, `{{config}}`, RCE chains
- **Twig** (PHP): `{{7*7}}`, filter callbacks
- **Velocity** (Java): Runtime execution
- **Freemarker** (Java): Execute templates
- **Thymeleaf** (Java/Spring): T() expressions
- **Pebble**, **Mako**, **Smarty**, **Pug**, **Handlebars**

#### Deserialization Attacks

- **Java:** CommonsCollections1/5, Spring1 (ysoserial payloads)
- **PHP:** PHPInfo, DateTime objects
- **Python:** Pickle payloads
- **Ruby:** ERB deserialization

#### Header-Based RCE

- X-Forwarded-Host, X-Original-URL, X-Rewrite-URL
- Referer, User-Agent, X-Api-Version
- CF-Connecting-IP, True-Client-IP

#### Blind RCE Detection

- **Time-based:** Detects delays in responses
- **OOB/DNS:** Out-of-band exfiltration attempts

### 🛡️ Stealth & Evasion

- **Random User-Agents:** Rotates through 5+ realistic browsers
- **Configurable Delays:** `-delay` flag to avoid rate limiting
- **Proxy Support:** Burp Suite integration with `-proxy`
- **Silent Operation:** No verbose output by default

### 📊 Smart Detection

- **Evidence Extraction:** Shows exact proof of RCE
- **Confidence Levels:** High/Medium/Low based on indicators
- **JSON Export:** Professional reporting

## 📦 Installation

### Method 1: Go Install (Recommended)
```bash

go install github.com/supreamhacker/rcemaster@latest


Method 2: Build from Source


git clone https://github.com/supreamhacker/rcemaster.git
cd rcemaster
go mod tidy
go build -o rcemaster main.go


🛠️ Usage

Basic Scan

rcemaster -u https://target.com


With Output File

rcemaster -u https://target.com -o rce_results.json


Stealth Mode (with delay)

rcemaster -f targets.txt -delay 500ms -o results.json

rcemaster -u https://target.com -delay 100ms


With Proxy (Burp Suite)

rcemaster -u https://target.com -proxy http://127.0.0.1:8080


Complete Command

rcemaster -u https://target.com -o results.json -delay 100ms -proxy http://127.0.0.1:8080


Help Menu

rcemaster -h


🎯 Supported Attack Vectors


Category                       Count                 Examples


Command Injection              50+                   ;id, |id, &&id, Base64, Reverse shells

SSTI                           80+                   Jinja2, Twig, Velocity, Freemarker, Thymeleaf

Deserialization                20+                   Java, PHP, Python, Ruby

Header-Based                   15+                   X-Forwarded-Host, Referer, User-Agent

Blind RCE                      10+                   Time-based, OOB/DNS


⚠️ Disclaimer & Ethical Use

RCEMaster is intended strictly for Educational Purposes, Authorized Bug Bounty Hunting, and Internal Security Auditing.
DO NOT use this tool on systems you don't own or have explicit permission to test
RCE vulnerabilities are Critical severity - report them responsibly
Unauthorized access to computer systems is illegal
The authors are not responsible for any misuse
Use responsibly and ethically 🛡️

🤝 Contributing

Contributions, issues, and feature requests are welcome!


🙏 Credits


Inspired by:

ysoserial (Java deserialization)
tplmap (SSTI detection)
commix (Command injection)
Bug bounty reports from HackerOne & Bugcrowd

Built with ❤️ in Go for the cybersecurity community.


---

### 🚀 Installation & Usage

```bash
# Install
go install github.com/supreamhacker/rcemaster@latest

# Basic scan

rcemaster -u https://target.com

# With output & stealth

rcemaster -u https://target.com -o results.json -delay 100ms

# With Burp Suite

rcemaster -u https://target.com -proxy http://127.0.0.1:8080


💡 Key Features Added


200+ Attack Vectors: Command injection, SSTI (10+ engines), deserialization, blind RCE, header-based
Human-Like Thinking: Detects tech stack, chooses best vectors, falls back if one fails

WAF Detection:

Identifies Cloudflare, AWS WAF, Akamai, etc.

Stealth Mode: Random User-Agents, configurable delays

Blind RCE: Time-based detection

Professional Output: JSON export, evidence extraction, confidence levels


📜 License

MIT License

[![Go Report Card](https://goreportcard.com/badge/github.com/supreamhacker/rcemaster)](https://goreportcard.com/report/github.com/supreamhacker/rcemaster)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)


