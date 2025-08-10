# SSH Test & Troubleshooting Tool

A comprehensive Go-based tool for SSH connection testing, pre-security setup, and post-security troubleshooting for PB-Deployer.

## Overview

This tool helps you:
- **Test SSH connections** to your servers
- **Prepare servers** for security lockdown (pre-security)
- **Verify and troubleshoot** after security lockdown (post-security)
- **Fix common SSH issues** automatically

## File Structure

```
ssh-test/
├── main.go           # Main orchestrator and CLI interface
├── pre-security.go   # Pre-lockdown preparation and setup
├── post-security.go  # Post-lockdown verification and troubleshooting
└── README.md         # This file
```

## Security Lockdown Workflow

```
1. [Pre-Security]  → 2. [Apply Lockdown] → 3. [Post-Security]
   Setup & Verify      Disable Root SSH     Verify & Troubleshoot
```

### 1. Pre-Security Phase
- Verify root SSH access (required for setup)
- Create and configure app user
- Set up SSH keys and sudo access
- Prepare deployment directories

### 2. Security Lockdown (Manual)
- Disable root SSH access in SSH config
- Apply firewall rules
- Remove root SSH keys (optional)

### 3. Post-Security Phase
- Verify app user SSH access works
- Test sudo permissions
- Check deployment directories
- Troubleshoot any issues

## Building

```bash
cd pb-deployer/cmd/ssh-test
go build -o ssh-test .
```

## Basic Usage

### Quick Connection Test
```bash
./ssh-test -host 91.99.196.153 -test
```

### Test Both Users
```bash
./ssh-test -host 91.99.196.153 -test-both
```

### Accept Host Key (fix "unknown host" errors)
```bash
./ssh-test -host 91.99.196.153 -accept-key
```

## Pre-Security Setup

### Check Readiness
```bash
./ssh-test -host 91.99.196.153 -pre-security
```

### Auto-Setup Everything
```bash
./ssh-test -host 91.99.196.153 -pre-security -setup
```

### Custom App User
```bash
./ssh-test -host 91.99.196.153 -pre-security -setup -app-user myapp
```

## Post-Security Verification

### Basic Post-Security Check
```bash
./ssh-test -host 91.99.196.153 -post-security -security-locked
```

### Post-Security with Auto-Fix
```bash
./ssh-test -host 91.99.196.153 -post-security -security-locked -auto-fix
```

### Verbose Post-Security Diagnostics
```bash
./ssh-test -host 91.99.196.153 -post-security -security-locked -v
```

## Troubleshooting

### Comprehensive Diagnostics
```bash
./ssh-test -host 91.99.196.153 -troubleshoot
```

### Fix Common Issues
```bash
./ssh-test -host 91.99.196.153 -fix
```

### Security-Locked Server Troubleshooting
```bash
./ssh-test -host 91.99.196.153 -troubleshoot -security-locked
```

## Command Line Options

### Required
- `-host` - Server hostname or IP address

### Server Configuration
- `-port` - SSH port (default: 22)
- `-app-user` - Application username (default: "pocketbase")
- `-root-user` - Root username (default: "root")
- `-key` - Path to private SSH key
- `-agent` - Use SSH agent (default: true)
- `-security-locked` - Server has security lockdown applied

### Operations
- `-test` - Test SSH connection
- `-test-both` - Test both root and app user connections
- `-root` - Connect as root user (for -test)
- `-accept-key` - Pre-accept host key
- `-fix` - Attempt to fix common issues
- `-diagnose` - Run connection diagnostics
- `-troubleshoot` - Comprehensive troubleshooting

### Pre-Security
- `-pre-security` - Run pre-security diagnostics
- `-setup` - Auto-setup prerequisites for lockdown

### Post-Security
- `-post-security` - Run post-security diagnostics
- `-auto-fix` - Attempt automatic fixes

### General
- `-v` - Verbose output

## Common Scenarios

### New Server Setup
```bash
# 1. Test initial connection
./ssh-test -host 91.99.196.153 -test -root

# 2. Prepare for security lockdown
./ssh-test -host 91.99.196.153 -pre-security -setup

# 3. Manually apply security lockdown (disable root SSH)

# 4. Verify post-security setup
./ssh-test -host 91.99.196.153 -post-security -security-locked
```

### Connection Issues
```bash
# Fix unknown host key
./ssh-test -host 91.99.196.153 -accept-key

# Comprehensive troubleshooting
./ssh-test -host 91.99.196.153 -troubleshoot

# Try automatic fixes
./ssh-test -host 91.99.196.153 -fix -test
```

### After Security Lockdown
```bash
# Quick verification
./ssh-test -host 91.99.196.153 -post-security -security-locked

# Fix permissions and test
./ssh-test -host 91.99.196.153 -post-security -security-locked -auto-fix

# Detailed diagnostics
./ssh-test -host 91.99.196.153 -post-security -security-locked -v
```

## Exit Codes

- `0` - Success, all operations completed
- `1` - Failure, some operations failed

## Output Indicators

- ✅ Success
- ❌ Error 
- ⚠️ Warning
- ℹ️ Information

## Tips

### SSH Key Management
- The tool will copy SSH keys from root to app user during setup
- Ensure your SSH keys are properly configured before running setup
- Use `-key` to specify a custom private key path

### Permissions
- The tool automatically sets correct SSH directory permissions (700/600)
- Use `-auto-fix` to fix permission issues after lockdown

### Debugging
- Use `-v` for verbose output and detailed information
- Check the tool's suggestions when operations fail
- Run `-troubleshoot` for comprehensive analysis

### Security Best Practices
- Always run pre-security setup before applying lockdown
- Test app user access thoroughly before disabling root SSH
- Keep a backup access method (console, etc.) in case of issues

## Examples for Different Scenarios

### Development Server
```bash
# Quick setup for development
./ssh-test -host dev.example.com -pre-security -setup -app-user developer
./ssh-test -host dev.example.com -post-security -security-locked -app-user developer
```

### Production Server
```bash
# Careful production setup
./ssh-test -host prod.example.com -pre-security -v
./ssh-test -host prod.example.com -test-both -v
# Apply lockdown manually with verification
./ssh-test -host prod.example.com -post-security -security-locked -v
```

### Custom Port
```bash
./ssh-test -host example.com -port 2222 -pre-security -setup
./ssh-test -host example.com -port 2222 -post-security -security-locked
```

### Using Custom SSH Key
```bash
./ssh-test -host example.com -key ~/.ssh/custom_key -pre-security -setup
./ssh-test -host example.com -key ~/.ssh/custom_key -post-security -security-locked
```

## Troubleshooting Common Issues

### "Permission denied (publickey)"
```bash
./ssh-test -host example.com -fix
./ssh-test -host example.com -accept-key
```

### "Host key verification failed"
```bash
./ssh-test -host example.com -accept-key
```

### App user sudo not working
```bash
./ssh-test -host example.com -pre-security -setup  # Re-setup sudo
```

### Deployment directories missing
```bash
./ssh-test -host example.com -pre-security -setup  # Re-create directories
```

---

For more help, run: `./ssh-test --help`
