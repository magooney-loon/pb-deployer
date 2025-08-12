# Getting Started

Welcome to **pb-deployer**! This tool helps you deploy PocketBase applications with ease and confidence.

## What is pb-deployer?

pb-deployer is a modern deployment tool specifically designed for PocketBase applications. It simplifies the deployment process by providing:

- **One-command deployment** to multiple cloud providers
- **Configuration management** with sensible defaults
- **Environment variable handling** for different stages
- **Database migration support** for seamless updates
- **Real-time deployment monitoring** and status tracking

## Quick Start

Get your PocketBase app deployed in under 5 minutes:

```bash
# Install pb-deployer globally
npm install -g pb-deployer

# Initialize in your PocketBase project
pb-deployer init

# Deploy to production
pb-deployer deploy
```

> **Tip**: Use `--dry-run` with any command to see what changes will be made without actually applying them.

## Key Features

- **Multi-provider support**: Deploy to Vercel, Railway, DigitalOcean, and more
- **Zero-config setup**: Smart defaults that work out of the box
- **Environment management**: Separate configs for dev, staging, and production
- **Database handling**: Automatic backup and migration support
- **Monitoring**: Built-in health checks and status monitoring

## Next Steps

1. Check out the [Installation](#installation) guide for detailed setup instructions
2. Learn about [Configuration](#configuration) options for your deployment
3. Follow the [Deployment](#deployment) workflow for your first deploy
4. Explore the [API Reference](#api-reference) for advanced usage

Ready to deploy? Let's get started! 🚀