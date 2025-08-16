# Step 1: Create VPS

Set up your Virtual Private Server (VPS) on Hetzner and configure secure SSH access.

## Prerequisites

- A Hetzner Cloud account
- Terminal/Command line access
- Basic understanding of SSH

## 1. Create Server on Hetzner

### Sign Up for Hetzner Cloud

1. Go to [Hetzner Cloud Console](https://console.hetzner-cloud.com/)
2. Create an account or sign in
3. Verify your email and complete account setup

### Create a New Project

1. Click **"New Project"** in the Hetzner Cloud Console
2. Give your project a descriptive name (e.g., "my-pocketbase-app")
3. Click **"Create Project"**

### Launch Your Server

1. Click **"Add Server"** in your project dashboard
2. **Choose Location**: Select a data center close to your users
   - `Nuremberg` (Germany) - Good for Europe
   - `Helsinki` (Finland) - Good for Europe/Russia
   - `Ashburn` (USA) - Good for North America

3. **Choose Image**: 
   - Select **Ubuntu 22.04** (recommended for stability)

4. **Choose Type**:
   - **Shared vCPU**: Good for most applications
   - **CX11** (1 vCPU, 2GB RAM) - Minimum recommended
   - **CX21** (2 vCPU, 4GB RAM) - Better for production

5. **Choose Volume**: Leave default (no additional volumes needed)

6. **Choose Network**: Use default network

7. **SSH Keys**: We'll set this up in the next section

8. **Firewalls**: We'll configure this later

9. **Backups**: Enable automatic backups (recommended)

10. **Placement Groups**: Leave empty

11. **Labels**: Add labels if needed (optional)

12. **Cloud Config**: Leave empty for now

13. **Name**: Give your server a name (e.g., "pocketbase-prod")

14. Click **"Create & Buy Now"**

> **💡 Tip**: Start with CX11 for testing. You can always upgrade later through the Hetzner console.

## 2. Setup SSH Agent Keys

### Generate SSH Key Pair (if you don't have one)

On your local machine, generate an SSH key pair:

```bash
# Generate a new SSH key pair
ssh-keygen -t ed25519 -C "your-email@example.com"

# When prompted for file location, press Enter for default
# When prompted for passphrase, create a strong passphrase
```

This creates two files:
- `~/.ssh/id_ed25519` (private key - keep this secure!)
- `~/.ssh/id_ed25519.pub` (public key - this goes on the server)

### Add SSH Key to Hetzner

1. **Copy your public key**:
   ```bash
   cat ~/.ssh/id_ed25519.pub
   ```

2. **Add to Hetzner**:
   - In Hetzner Console, go to your project
   - Click **"SSH Keys"** in the left sidebar
   - Click **"Add SSH Key"**
   - Paste your public key content
   - Give it a name (e.g., "my-laptop")
   - Click **"Add SSH Key"**

3. **Assign to Server**:
   - Go back to your server creation process
   - In the **"SSH Keys"** section, select your newly added key
   - Complete server creation

### Start SSH Agent

Start the SSH agent and add your key:

```bash
# Start SSH agent
eval "$(ssh-agent -s)"

# Add your SSH key to the agent
ssh-add ~/.ssh/id_ed25519

# Verify the key is loaded
ssh-add -l
```

### Test SSH Connection

Once your server is created (usually takes 1-2 minutes):

1. **Get your server IP** from the Hetzner console
2. **Test the connection**:
   ```bash
   ssh root@YOUR_SERVER_IP
   ```

You should be connected without entering a password!

## Verification

Your VPS is ready when you can:

- ✅ SSH into your server without password prompt
- ✅ See the Ubuntu welcome message
- ✅ Run basic commands like `ls` and `pwd`

### Server Information

Check your server details:

```bash
# Check Ubuntu version
lsb_release -a

# Check available resources
free -h
df -h

# Check CPU info
nproc
```

## Security Notes

- 🔒 **Never share your private SSH key** (`id_ed25519`)
- 🔒 **Use strong passphrases** for your SSH keys
- 🔒 **Keep your SSH keys backed up** securely
- 🔒 **We'll disable password authentication** in Step 4

## Troubleshooting

### Can't connect via SSH

1. **Check server status** in Hetzner console
2. **Verify IP address** is correct
3. **Check SSH key** was added to server during creation
4. **Try verbose SSH** for debugging:
   ```bash
   ssh -v root@YOUR_SERVER_IP
   ```

### Permission denied

1. **Verify SSH key** is in SSH agent:
   ```bash
   ssh-add -l
   ```
2. **Re-add key** if needed:
   ```bash
   ssh-add ~/.ssh/id_ed25519
   ```

### Wrong SSH key format

- Make sure you copied the **public key** (`.pub` file)
- Public key should start with `ssh-ed25519` or `ssh-rsa`

## Next Steps

🎉 **Great!** Your VPS is ready. 

**Next**: Proceed to [Step 2: Cloudflare Proxy](#cloudflare-proxy) to set up a domain and SSL, or skip to [Step 3: Setup Server](#setup-server) if you want to use the IP address directly.