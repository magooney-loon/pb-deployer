# pb-deployer

Production deployment automation for PocketBase applications with SSH server management and security hardening.

## Overview

Automates the complete lifecycle of deploying PocketBase apps to production servers:
- **Server Setup**: SSH user creation, directory structure, security hardening
- **Deployment**: rsync-based file transfer with systemd service management
- **Security**: UFW firewall, fail2ban, SSH hardening
- **Monitoring**: Health checks and service management

## Architecture

```
SvelteKit 5 UI → PocketBase API → SSH Manager → Remote Servers
                      ↓
              SQLite (metadata) + File Storage (versions)
```

## Core Workflow

1. **Server Registration**: Add SSH credentials and connection details
2. **Server Setup**: Automated user creation and directory structure (`/opt/pocketbase/apps/`)
3. **Security Lockdown**: Firewall, fail2ban, disable root SSH
4. **App Deployment**: Upload binary + static files, systemd service creation
5. **Version Management**: Rollback support with file storage

## Key Features

- **SSH Connection Pooling**: Persistent connections with health monitoring
- **Security-Aware Operations**: Automatic transition from root to app user after lockdown
- **Zero-Downtime Deployments**: Staging directory with quick swap
- **Real-time Progress**: WebSocket updates for all long-running operations
- **Automatic Backup/Rollback**: Safe production deployments

See `internal/*/README.md` for detailed component documentation.
