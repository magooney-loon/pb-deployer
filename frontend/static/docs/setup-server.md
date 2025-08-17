# Step 3: Setup Server

Now you'll use our built-in UI to configure your server automatically.

## 1. Add New Server

1. In the pb-deployer dashboard, click **"Add Server"**
2. Enter your server details:
   - **Server IP**: Your Hetzner server IP address
   - **Domain** (optional): Your Cloudflare domain if you set one up
   - **SSH Key**: Agent (recommended) or manual keypath
3. Click **"Add Server"**

## 2. Click Setup

1. Find your server in the dashboard
2. Click the **"Setup"** button
3. Wait for the automated setup to complete

## 3. Optional: Lockdown

For production servers, click **"Lockdown"** to harden security:

- Disable password authentication
- Configure fail2ban
- Set up automatic security updates
- Create non-root user accounts
- Advanced firewall rules
- System hardening

## 4. Deploy and Manage Apps

Once setup is complete, you can:
- Deploy new applications
- Manage existing apps
- Monitor server resources
- View logs and analytics

Your server is now ready for production use!
