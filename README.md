# Whizhook-Go (Sleeping God Protocol v1)

## Overview
Whizhook-Go is a single-binary Go CLI that automates:
1. Local IP detection
2. Cloudflare Tunnel setup
3. Payload generation (JS & XML)
4. Webhook + Dashboard server launch
5. Auto-RCE trigger with PHP reverse shell

## Installation
```bash
git clone https://github.com/nxneeraj/whizhook-go.git
cd whizhook-go
go mod tidy
go build -o whizhook cmd/whizhook/main.go
sudo mv whizhook /usr/local/bin
