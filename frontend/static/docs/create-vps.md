# Step 1: Create VPS

## 1. Create Hetzner Account

Go to [Hetzner Cloud](https://hetzner.cloud/?ref=OziePwGckx9o) and create an account.

## 2. Create New Server

1. Click **"Add Server"**
2. Choose **Ubuntu**
3. Select any CPU/RAM ($5/month - cheapest option)
4. Pick any location
5. Add your SSH key (see step 3)
6. Click **"Create & Buy Now"**

## 3. Setup SSH Keys

### Generate SSH Key

Use the Password app on Linux Mint or create your own key manually.

### Add to Hetzner

1. Open the Password app and copy the pub key
2. In Hetzner console: **SSH Keys** → **Add SSH Key**
3. Paste the key and save
4. Select it when creating your server

Done! You should be connected to your VPS.
