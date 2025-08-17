<div align="center">
  <img src="frontend/static/favicon.svg" alt="Logo" width="200">
  <h1 align="center">pb-deployer</h1>
  <h3 align="center">Automates the lifecycle of deploying PocketBase apps to production</h3>
</div>

<div align="center">
    <a href="https://github.com/magooney-loon/pb-deployer/stargazers"><img src="https://img.shields.io/github/stars/magooney-loon/pb-deployer?style=for-the-badge&color=blue" alt="Stargazers"></a>
    <a href="https://github.com/magooney-loon/pb-deployer/graphs/contributors"><img src="https://img.shields.io/github/contributors/magooney-loon/pb-deployer?style=for-the-badge&color=blue" alt="Contributors"></a>
    <a href="https://github.com/magooney-loon/pb-deployer/blob/main/LICENSE"><img src="https://img.shields.io/github/license/magooney-loon/pb-deployer?style=for-the-badge&color=blue" alt="AGPL-3.0"></a>
    <br>
    <h5 align="center">**WARNING**HOBBY PROJECT**</h5>
  <hr>
</div>

**Server Setup**: Automated SSH user creation, directory structure, security hardening
<br>
**Deployment**: SFTP transfer protocol && systemd service management
<br>
**Security**: UFW firewall, fail2ban, SSH lockdown w/ specialized managers
<br>
**Observability**: Comprehensive tracing, connection pooling, health monitoring
<br>
**Configuration**: Type-safe config management w/ validation && retry logic

## 🚀 Quick Start

```bash
git clone https://github.com/magooney-loon/pb-deployer
cd pb-deployer
go run cmd/scripts/main.go --install
```

## Core Workflow

1. **Server Registration**: Add remote host connection details
2. **Server Setup**: Automated user creation and directory structure (`/opt/pocketbase/apps/`)
3. **Security Lockdown**: Firewall, fail2ban, disable root SSH
4. **App Deployment**: Upload prod dist, systemd service creation
5. **Version Management**: Rollback support with file storage

## Key Features

- Dependency injection, no singletons, clean interfaces
- Persistent connections with automatic health monitoring
- Domain-specific operations (setup, security, services, deployment)
- Full observability with structured logging and metrics
- Automatic transition from root to app user after lockdown
- Staging directory with atomic swaps
- WebSocket updates with detailed operation tracking
- Generic config management with validation

See `**/*/README.md` for detailed component documentation.

## Contribution
PRs are encouraged, but consider opening a discussion first for minor/major features.
