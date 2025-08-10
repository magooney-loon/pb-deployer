# SSH Troubleshooting Guide

This guide helps you diagnose and fix SSH connection issues, particularly after security lockdown has been applied to your servers.

## Overview

After security lockdown is applied to a server, several changes are made that affect SSH connectivity:

- **Root SSH access is disabled** (`PermitRootLogin no`)
- **Password authentication is disabled** (`PasswordAuthentication no`)
- **Only public key authentication is allowed**
- **Only the app user can connect via SSH**
- **Privileged operations must use sudo**

This is expected behavior and enhances security, but it can cause connection issues if not properly configured.

## Quick Diagnosis

### 1. Use the Connection Test Tool

First, test your server connection through the web interface:
1. Go to your server details page
2. Click "Test Connection"
3. Review the results

The test will show:
- ✅ **TCP Connection**: Basic network connectivity
- ⚠️ **Root SSH Connection**: Should show "Disabled (Security Locked)" after lockdown
- ✅ **App SSH Connection**: Should work for deployments

### 2. Expected Results After Security Lockdown

**Healthy Security-Locked Server:**
- TCP Connection: ✅ Success
- Root SSH Connection: ⚠️ Disabled (Expected)
- App SSH Connection: ✅ Success
- Overall Status: `healthy_secured`

**Problematic Server:**
- TCP Connection: ❌ or ✅
- Root SSH Connection: ❌ Failed (Expected)
- App SSH Connection: ❌ Failed (Problem!)
- Overall Status: `app_ssh_failed`

## Common Issues and Solutions

### Issue 1: "App SSH Connection Failed"

**Symptoms:**
- App user SSH connection fails
- Error messages about authentication or connection refused

**Likely Causes:**
- SSH keys not properly copied to app user
- Incorrect file permissions
- App user doesn't exist

**Solutions:**

#### Option A: Use the Troubleshooting Script
```bash
cd pb-deployer
chmod +x scripts/fix-post-security-ssh.sh
./scripts/fix-post-security-ssh.sh -h YOUR_SERVER_IP -f
```

#### Option B: Manual Fix via Root Access
If you still have root access through the console:

```bash
# 1. Connect as root (via console/VNC)
# 2. Check if app user exists
id pocketbase

# 3. Copy SSH keys from root to app user
cp /root/.ssh/authorized_keys /home/pocketbase/.ssh/authorized_keys

# 4. Fix ownership and permissions
chown -R pocketbase:pocketbase /home/pocketbase/.ssh
chmod 700 /home/pocketbase/.ssh
chmod 600 /home/pocketbase/.ssh/authorized_keys

# 5. Test the connection
exit
ssh pocketbase@YOUR_SERVER_IP "echo 'Connection successful'"
```

#### Option C: Use SSH Test Tool
```bash
cd pb-deployer
go build -o ssh-test cmd/ssh-test/main.go

# Pre-accept host key if needed
./ssh-test -host YOUR_SERVER_IP -accept-key

# Run comprehensive diagnostics
./ssh-test -host YOUR_SERVER_IP -post-security -security-locked

# Test app user connection
./ssh-test -host YOUR_SERVER_IP -test
```

### Issue 2: "Sudo Access Failed"

**Symptoms:**
- App user can connect but can't run deployment operations
- "sudo: no tty present" or permission denied errors

**Solution:**
```bash
# Connect as root and fix sudo configuration
ssh root@YOUR_SERVER_IP  # (via console if SSH disabled)

# Create or update sudoers file for app user
cat > /etc/sudoers.d/pocketbase << 'EOF'
pocketbase ALL=(ALL) NOPASSWD: /bin/systemctl, /usr/bin/systemctl, /bin/mkdir, /usr/bin/mkdir, /bin/chown, /usr/bin/chown, /bin/chmod, /usr/bin/chmod
EOF

# Test sudo access
su - pocketbase -c "sudo -n systemctl --version"
```

### Issue 3: "Host Key Unknown/Changed"

**Symptoms:**
- "WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!"
- "ssh: handshake failed: knownhosts: key is unknown"

**Solution:**
```bash
# Remove old host key
ssh-keygen -R YOUR_SERVER_IP

# Pre-accept new host key
cd pb-deployer
go build -o ssh-test cmd/ssh-test/main.go
./ssh-test -host YOUR_SERVER_IP -accept-key
```

### Issue 4: "Connection Refused" or "No Route to Host"

**Symptoms:**
- Cannot establish TCP connection
- Network timeouts

**Solutions:**
1. **Check server status**: Ensure server is running
2. **Verify IP/Port**: Confirm correct IP address and SSH port
3. **Check firewall**: Ensure SSH port (22) is open
4. **Test basic connectivity**: `ping YOUR_SERVER_IP`

## Step-by-Step Troubleshooting

### Step 1: Basic Connectivity
```bash
# Test if server is reachable
ping YOUR_SERVER_IP

# Test if SSH port is open
nc -zv YOUR_SERVER_IP 22
```

### Step 2: Test SSH Connection
```bash
# Test app user connection
ssh pocketbase@YOUR_SERVER_IP "echo 'App user works'"

# Test sudo access
ssh pocketbase@YOUR_SERVER_IP "sudo -n systemctl --version"
```

### Step 3: Verify Setup
```bash
# Check deployment directories
ssh pocketbase@YOUR_SERVER_IP "ls -la /opt/pocketbase"

# Test service management
ssh pocketbase@YOUR_SERVER_IP "sudo systemctl status ssh"
```

## Advanced Troubleshooting

### Using SSH Test Tool

The SSH test tool provides comprehensive diagnostics:

```bash
cd pb-deployer
go build -o ssh-test cmd/ssh-test/main.go

# Basic connection test
./ssh-test -host YOUR_SERVER_IP -test

# Test both users (shows security lockdown status)
./ssh-test -host YOUR_SERVER_IP -test-both -security-locked

# Comprehensive troubleshooting
./ssh-test -host YOUR_SERVER_IP -troubleshoot -security-locked

# Post-security specific diagnostics
./ssh-test -host YOUR_SERVER_IP -post-security -security-locked

# Auto-fix common issues
./ssh-test -host YOUR_SERVER_IP -fix
```

### Verbose Debugging

For detailed SSH debugging:
```bash
# Debug SSH connection
ssh -vvv pocketbase@YOUR_SERVER_IP

# Check SSH daemon logs on server (as root)
journalctl -u ssh -f
```

### Manual Key Recovery

If SSH keys are completely missing:

```bash
# Method 1: Via console access
# 1. Access server console (VNC/KVM)
# 2. Login as root
# 3. Copy your public key:
mkdir -p /home/pocketbase/.ssh
echo "YOUR_PUBLIC_KEY_HERE" > /home/pocketbase/.ssh/authorized_keys
chown -R pocketbase:pocketbase /home/pocketbase/.ssh
chmod 700 /home/pocketbase/.ssh
chmod 600 /home/pocketbase/.ssh/authorized_keys

# Method 2: Via cloud provider console
# Most cloud providers offer console access or key injection features
```

## Verification Commands

After fixing issues, verify everything works:

```bash
# 1. Test basic connection
ssh pocketbase@YOUR_SERVER_IP "whoami"

# 2. Test sudo access
ssh pocketbase@YOUR_SERVER_IP "sudo -n systemctl --version"

# 3. Test deployment directories
ssh pocketbase@YOUR_SERVER_IP "ls -la /opt/pocketbase"

# 4. Test service management
ssh pocketbase@YOUR_SERVER_IP "sudo systemctl status ssh"

# 5. Verify security settings
ssh pocketbase@YOUR_SERVER_IP "sudo grep 'PermitRootLogin no' /etc/ssh/sshd_config"
```

## When to Use Each Tool

| Problem | Tool | Command |
|---------|------|---------|
| Unknown connection issue | Web Interface | Server Details → Test Connection |
| First-time diagnosis | SSH Test Tool | `./ssh-test -host IP -diagnose` |
| Post-security issues | SSH Test Tool | `./ssh-test -host IP -post-security -security-locked` |
| Host key problems | SSH Test Tool | `./ssh-test -host IP -accept-key` |
| Manual debugging | Shell Script | `./scripts/fix-post-security-ssh.sh -h IP -v` |
| Quick fixes | SSH Test Tool | `./ssh-test -host IP -fix` |

## Understanding Security Lockdown

After security lockdown is applied:

### ✅ Expected Behavior
- Root SSH connections fail (security feature)
- App user SSH connections work
- Sudo commands work without password
- Deployment operations succeed

### ❌ Problematic Behavior  
- App user SSH connections fail
- Sudo commands require password or fail
- Deployment operations fail
- Cannot access deployment directories

### 🔒 Security Features Active
- Password authentication disabled
- Root login disabled
- Public key authentication only
- Firewall configured
- Fail2ban active

## Getting Help

If you're still having issues after following this guide:

1. **Gather Information:**
   ```bash
   # Run comprehensive diagnostics
   ./ssh-test -host YOUR_SERVER_IP -troubleshoot -security-locked -v
   
   # Run the shell script
   ./scripts/fix-post-security-ssh.sh -h YOUR_SERVER_IP -v
   ```

2. **Check Logs:**
   - Server deployment logs
   - SSH connection test results
   - System logs (if accessible)

3. **Provide Details:**
   - Server OS and version
   - Whether security lockdown was applied
   - Exact error messages
   - Steps that led to the issue

## Quick Reference

### Common Commands
```bash
# Test connection
ssh pocketbase@SERVER_IP "echo 'test'"

# Fix permissions
ssh root@SERVER_IP "chown -R pocketbase:pocketbase /home/pocketbase/.ssh && chmod 700 /home/pocketbase/.ssh && chmod 600 /home/pocketbase/.ssh/authorized_keys"

# Check security settings
ssh pocketbase@SERVER_IP "sudo grep -E '^(PermitRootLogin|PasswordAuthentication|PubkeyAuthentication)' /etc/ssh/sshd_config"

# Test sudo
ssh pocketbase@SERVER_IP "sudo -n systemctl --version"
```

### File Locations
- SSH keys: `/home/pocketbase/.ssh/authorized_keys`
- SSH config: `/etc/ssh/sshd_config`
- Sudo config: `/etc/sudoers.d/pocketbase`
- App directory: `/opt/pocketbase/`
- Logs: `/var/log/pocketbase/`

---

**Remember:** After security lockdown, root SSH access being disabled is **expected and correct** behavior. The goal is to have fully functional app user access for all deployment operations.