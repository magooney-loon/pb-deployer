# Step 2: Cloudflare Proxy

## 1. Buy Domain on Cloudflare

1. Go to [Cloudflare.com](https://cloudflare.com)
2. Create account and sign in
3. Go to **Domain Registration**
4. Search for your domain name
5. Buy the domain

## 2. Enable Cloudflare SSL and HTTPS

1. Add your domain to Cloudflare (if not already added during purchase)
2. Go to **SSL/TLS** → **Overview**
3. Set SSL mode to **"Full (strict)"**
4. Go to **SSL/TLS** → **Edge Certificates**
5. Enable **"Always Use HTTPS"**

## 3. Add A Record and CNAME for Domains

### Add A Record
1. Go to **DNS** → **Records**
2. Add A record:
   - **Type**: A
   - **Name**: @ 
   - **IPv4 address**: Your Hetzner server IP
   - **Proxy status**: Proxied (orange cloud)
   - Click **Save**

### Add CNAME Record
1. Add CNAME record:
   - **Type**: CNAME
   - **Name**: www
   - **Target**: yourdomain.com
   - **Proxy status**: Proxied (orange cloud)  
   - Click **Save**

Done! Your domain will now point to your VPS with SSL enabled.