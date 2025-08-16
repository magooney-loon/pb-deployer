# Step 2: Cloudflare Proxy (Optional)

Set up a custom domain with Cloudflare for SSL/HTTPS and improved performance. This step is optional but highly recommended for production deployments.

## Prerequisites

- Completed [Step 1: Create VPS](#create-vps)
- Your server IP address from Hetzner
- Credit card for domain purchase (if needed)

## Why Use Cloudflare?

- 🔒 **Free SSL certificates** - Automatic HTTPS
- ⚡ **CDN & Caching** - Faster global performance  
- 🛡️ **DDoS Protection** - Enhanced security
- 📊 **Analytics** - Traffic insights
- 🌐 **Custom Domain** - Professional appearance

## 1. Buy Domain

### Choose a Domain Registrar

Popular options for domain registration:

- **Cloudflare Registrar** (recommended) - At-cost pricing
- **Namecheap** - User-friendly interface
- **Google Domains** - Simple setup
- **GoDaddy** - Widely known

### Register Your Domain

1. **Choose your domain name**:
   - Keep it short and memorable
   - Use `.com` for best compatibility
   - Examples: `myapp.com`, `pocketbase-app.com`

2. **Purchase the domain**:
   - Search for availability
   - Complete registration process
   - Keep your registration details secure

> **💡 Tip**: If using Cloudflare Registrar, you can register and manage everything in one place.

## 2. Set Up Cloudflare

### Create Cloudflare Account

1. Go to [Cloudflare.com](https://cloudflare.com)
2. Click **"Sign Up"**
3. Create your account with email and password
4. Verify your email address

### Add Your Domain to Cloudflare

1. **Add Site**:
   - Click **"Add a Site"** in Cloudflare dashboard
   - Enter your domain name (e.g., `myapp.com`)
   - Click **"Add Site"**

2. **Choose Plan**:
   - Select **"Free"** plan (sufficient for most needs)
   - Click **"Continue"**

3. **DNS Scan**:
   - Cloudflare will scan for existing DNS records
   - Review and verify the records
   - Click **"Continue"**

### Update Nameservers

1. **Get Cloudflare Nameservers**:
   - Cloudflare will show you 2 nameservers
   - Example: `ava.ns.cloudflare.com` and `ben.ns.cloudflare.com`

2. **Update at Your Domain Registrar**:
   - Log into your domain registrar account
   - Find DNS/Nameserver settings
   - Replace existing nameservers with Cloudflare's
   - Save changes

3. **Wait for Propagation**:
   - DNS changes can take 24-48 hours
   - Cloudflare will email you when active
   - Status will change to "Active" in dashboard

## 3. Set Up DNS Records

### Add A Record for Root Domain

1. **Go to DNS Settings**:
   - In Cloudflare dashboard, click **"DNS"**
   - Click **"Records"**

2. **Add A Record**:
   - **Type**: `A`
   - **Name**: `@` (represents your root domain)
   - **IPv4 address**: Your Hetzner server IP
   - **Proxy status**: 🟠 **Proxied** (orange cloud)
   - **TTL**: Auto
   - Click **"Save"**

### Add CNAME for www Subdomain

1. **Add CNAME Record**:
   - **Type**: `CNAME`
   - **Name**: `www`
   - **Target**: `yourdomain.com` (your root domain)
   - **Proxy status**: 🟠 **Proxied** (orange cloud)
   - **TTL**: Auto
   - Click **"Save"**

### Optional: Add API Subdomain

For API-specific subdomain (e.g., `api.yourdomain.com`):

1. **Add A Record**:
   - **Type**: `A`
   - **Name**: `api`
   - **IPv4 address**: Your Hetzner server IP
   - **Proxy status**: 🟠 **Proxied** (orange cloud)
   - **TTL**: Auto
   - Click **"Save"**

## 4. Configure SSL/HTTPS

### Enable SSL

1. **Go to SSL/TLS Settings**:
   - In Cloudflare dashboard, click **"SSL/TLS"**
   - Click **"Overview"**

2. **Set SSL Mode**:
   - Choose **"Full (strict)"** for maximum security
   - This encrypts traffic between visitors and Cloudflare, and between Cloudflare and your server

### Enable Always Use HTTPS

1. **Go to Edge Certificates**:
   - Click **"SSL/TLS"** → **"Edge Certificates"**

2. **Enable Always Use HTTPS**:
   - Toggle **"Always Use HTTPS"** to ON
   - This redirects all HTTP traffic to HTTPS

### Enable HSTS (Optional but Recommended)

1. **Enable HSTS**:
   - Toggle **"HTTP Strict Transport Security (HSTS)"** to ON
   - **Max Age**: 6 months
   - **Include subdomains**: ON
   - **No-sniff header**: ON
   - Click **"Save"**

## 5. Configure Security Settings

### Set Security Level

1. **Go to Security Settings**:
   - Click **"Security"** → **"Settings"**

2. **Set Security Level**:
   - Choose **"Medium"** for balanced protection
   - This challenges suspicious requests

### Enable Bot Fight Mode

1. **Enable Bot Protection**:
   - Toggle **"Bot Fight Mode"** to ON
   - This blocks known malicious bots

## 6. Performance Optimization

### Enable Caching

1. **Go to Caching**:
   - Click **"Caching"** → **"Configuration"**

2. **Set Caching Level**:
   - Choose **"Standard"** for most applications
   - This caches static content at Cloudflare edge

### Enable Minification

1. **Go to Speed**:
   - Click **"Speed"** → **"Optimization"**

2. **Enable Auto Minify**:
   - Toggle **JavaScript**, **CSS**, and **HTML** to ON
   - This reduces file sizes

## Verification

Your Cloudflare setup is complete when:

- ✅ Domain status shows "Active" in Cloudflare dashboard
- ✅ `https://yourdomain.com` loads (may show Cloudflare error initially)
- ✅ `http://yourdomain.com` redirects to HTTPS
- ✅ SSL certificate is valid (check browser lock icon)

### Test Your Setup

```bash
# Test DNS resolution
nslookup yourdomain.com

# Test HTTPS redirect
curl -I http://yourdomain.com

# Test SSL certificate
curl -I https://yourdomain.com
```

## Important Notes

- 🕐 **DNS Propagation**: Changes can take up to 48 hours to fully propagate
- 🔒 **Cloudflare Errors**: You'll see Cloudflare error pages until your server is configured (Step 3)
- 📊 **Analytics**: Check Cloudflare dashboard for traffic analytics
- 💰 **Costs**: Free plan covers most needs; paid plans offer advanced features

## Troubleshooting

### Domain Not Active

1. **Check nameservers**: Verify they're set correctly at your registrar
2. **Wait longer**: DNS propagation can take time
3. **Contact registrar**: Some registrars have delays

### SSL Errors

1. **Check SSL mode**: Should be "Full (strict)" for best security
2. **Wait for certificate**: SSL certificates can take a few minutes
3. **Clear browser cache**: Hard refresh your browser

### DNS Not Resolving

1. **Check A record**: Verify your server IP is correct
2. **Check proxy status**: Orange cloud should be enabled
3. **Test with different DNS**: Try `8.8.8.8` or `1.1.1.1`

### Still Getting HTTP

1. **Check "Always Use HTTPS"**: Should be enabled
2. **Clear browser cache**: Your browser might cache the HTTP version
3. **Test incognito mode**: Private browsing bypasses cache

## Next Steps

🎉 **Excellent!** Your domain and Cloudflare are configured.

**Next**: Proceed to [Step 3: Setup Server](#setup-server) to install and configure the necessary software on your VPS.

> **Note**: You may see Cloudflare error pages when visiting your domain until we complete server setup in the next step. This is normal!