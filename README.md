![pb-deployer banner](frontend/static/deployer.png)

# <img src="frontend/static/favicon.svg" width="32" height="32" style="vertical-align: middle;"> pb-deployer

Automates the complete lifecycle of deploying PocketBase apps to production servers:
- **Server Setup**: Automated SSH user creation, directory structure, security hardening
- **Deployment**: SFTP transfer protocol && systemd service management
- **Security**: UFW firewall, fail2ban, SSH lockdown with specialized managers
- **Observability**: Comprehensive tracing, connection pooling, health monitoring
- **Configuration**: Type-safe config management with validation and retry logic

## Core Workflow

1. **Server Registration**: Add remote host connection details
2. **Server Setup**: Automated user creation and directory structure (`/opt/pocketbase/apps/`)
3. **Security Lockdown**: Firewall, fail2ban, disable root SSH
4. **App Deployment**: Upload binary + static files, systemd service creation
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
- Safe production deployments with version tracking

See `internal/*/README.md` for detailed component documentation.
