# Server Manager API

REST API schema for the Server Manager service.

## Base URL
```
/api/v1/servers
```

## Endpoints

### Create Server
Register a new server for management.

**Endpoint:** `POST /api/v1/servers`

**Request Body:**
```json
{
  "name": "string",
  "host": "string",
  "port": 22,
  "username": "string",
  "auth_method": "ssh_agent|ssh_key|password",
  "ssh_key_path": "string",
  "password": "string",
  "description": "string",
  "environment": "production|staging|development",
  "tags": ["string"]
}
```

**Response:** `201 Created`
```json
{
  "id": "string",
  "name": "string",
  "host": "string",
  "port": 22,
  "username": "string",
  "auth_method": "ssh_agent",
  "description": "string",
  "environment": "production",
  "tags": ["web", "api"],
  "status": "offline",
  "setup_complete": false,
  "security_locked": false,
  "created": "2024-01-01T00:00:00Z",
  "updated": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid server configuration
- `409 Conflict` - Server name or host already exists
- `500 Internal Server Error` - Server registration failed

---

### Get Server
Get server details and current status.

**Endpoint:** `GET /api/v1/servers/{id}`

**Path Parameters:**
- `id` (string, required) - Server ID

**Response:** `200 OK`
```json
{
  "id": "string",
  "name": "string",
  "host": "string",
  "port": 22,
  "username": "string",
  "auth_method": "string",
  "description": "string",
  "environment": "string",
  "tags": ["string"],
  "status": "online|offline|setup|security|error|unknown",
  "setup_complete": true,
  "security_locked": true,
  "last_seen": "2024-01-01T00:00:00Z",
  "uptime": "30d 12h 45m",
  "system_info": {
    "os": "Ubuntu 22.04.3 LTS",
    "kernel": "5.15.0-91-generic",
    "architecture": "x86_64",
    "memory_total": "8GB",
    "disk_total": "100GB",
    "cpu_cores": 4
  },
  "connection_health": {
    "tcp_reachable": true,
    "ssh_accessible": true,
    "last_check": "2024-01-01T00:00:00Z",
    "response_time": "25ms"
  },
  "applications_count": 5,
  "created": "2024-01-01T00:00:00Z",
  "updated": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Server not found

---

### List Servers
List all registered servers with optional filtering.

**Endpoint:** `GET /api/v1/servers`

**Query Parameters:**
- `status` (string, optional) - Filter by status
- `environment` (string, optional) - Filter by environment
- `tags` (string, optional) - Filter by tags (comma-separated)
- `setup_complete` (boolean, optional) - Filter by setup status
- `security_locked` (boolean, optional) - Filter by security status
- `limit` (integer, optional) - Limit number of results (default: 50)
- `offset` (integer, optional) - Offset for pagination (default: 0)

**Response:** `200 OK`
```json
{
  "servers": [
    {
      "id": "string",
      "name": "string",
      "host": "string",
      "environment": "string",
      "status": "string",
      "setup_complete": true,
      "security_locked": true,
      "applications_count": 5,
      "last_seen": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 25,
  "limit": 50,
  "offset": 0
}
```

---

### Update Server
Update server configuration.

**Endpoint:** `PUT /api/v1/servers/{id}`

**Path Parameters:**
- `id` (string, required) - Server ID

**Request Body:**
```json
{
  "name": "string",
  "description": "string",
  "environment": "string",
  "tags": ["string"],
  "auth_method": "string",
  "ssh_key_path": "string"
}
```

**Response:** `200 OK`
```json
{
  "message": "Server updated successfully",
  "server_id": "string",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid update data
- `404 Not Found` - Server not found
- `409 Conflict` - Name or host conflict
- `500 Internal Server Error` - Update failed

---

### Delete Server
Remove server from management and cleanup resources.

**Endpoint:** `DELETE /api/v1/servers/{id}`

**Path Parameters:**
- `id` (string, required) - Server ID

**Query Parameters:**
- `cleanup_apps` (boolean, optional) - Remove all applications (default: false)
- `force` (boolean, optional) - Force deletion even with active apps (default: false)

**Response:** `200 OK`
```json
{
  "message": "Server deleted successfully",
  "server_id": "string",
  "apps_removed": 3,
  "cleanup_performed": true,
  "deleted_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Server not found
- `409 Conflict` - Server has active applications and force is false
- `500 Internal Server Error` - Deletion failed

---

### Test Connection
Perform comprehensive connection test to server.

**Endpoint:** `POST /api/v1/servers/{id}/test`

**Path Parameters:**
- `id` (string, required) - Server ID

**Request Body:**
```json
{
  "test_types": ["tcp", "ssh", "sudo"],
  "timeout": "30s"
}
```

**Response:** `200 OK`
```json
{
  "success": true,
  "overall_status": "healthy|degraded|failed",
  "tests": {
    "tcp_connection": {
      "success": true,
      "latency": "15.23ms",
      "message": "TCP connection successful"
    },
    "ssh_connection": {
      "success": true,
      "username": "deploy",
      "auth_method": "ssh_agent",
      "message": "SSH authentication successful"
    },
    "sudo_access": {
      "success": true,
      "message": "Sudo access verified"
    }
  },
  "system_info": {
    "os": "Ubuntu 22.04.3 LTS",
    "uptime": "30 days",
    "load_average": "0.25, 0.30, 0.28"
  },
  "tested_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Server not found
- `408 Request Timeout` - Connection test timeout
- `500 Internal Server Error` - Test execution failed

---

### Setup Server
Perform initial server setup and configuration.

**Endpoint:** `POST /api/v1/servers/{id}/setup`

**Path Parameters:**
- `id` (string, required) - Server ID

**Request Body:**
```json
{
  "create_user": true,
  "username": "deploy",
  "ssh_keys": [
    "ssh-rsa AAAAB3NzaC1yc2EAAAA... user@example.com"
  ],
  "install_packages": ["curl", "git", "docker.io"],
  "create_directories": [
    {
      "path": "/opt/apps",
      "permissions": "755",
      "owner": "deploy"
    }
  ],
  "configure_sudo": true,
  "sudo_commands": [
    "/usr/bin/systemctl *",
    "/usr/bin/docker"
  ]
}
```

**Response:** `202 Accepted`
```json
{
  "message": "Server setup started",
  "server_id": "string",
  "setup_id": "string",
  "estimated_duration": "5m",
  "started_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Server not found
- `409 Conflict` - Server already setup or setup in progress
- `400 Bad Request` - Invalid setup configuration
- `500 Internal Server Error` - Setup initiation failed

---

### Apply Security
Apply security hardening configuration to server.

**Endpoint:** `POST /api/v1/servers/{id}/security`

**Path Parameters:**
- `id` (string, required) - Server ID

**Request Body:**
```json
{
  "disable_root_login": true,
  "disable_password_auth": true,
  "configure_firewall": true,
  "allowed_ports": [22, 80, 443],
  "setup_fail2ban": true,
  "fail2ban_config": {
    "max_retries": 5,
    "ban_time": "1h",
    "services": ["ssh"]
  },
  "enable_auto_updates": true,
  "additional_hardening": true
}
```

**Response:** `202 Accepted`
```json
{
  "message": "Security hardening started",
  "server_id": "string",
  "security_id": "string",
  "estimated_duration": "8m",
  "started_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Server not found
- `409 Conflict` - Security hardening already applied or in progress
- `400 Bad Request` - Invalid security configuration
- `500 Internal Server Error` - Security hardening initiation failed

---

### Get Server Status
Get real-time server status and health information.

**Endpoint:** `GET /api/v1/servers/{id}/status`

**Path Parameters:**
- `id` (string, required) - Server ID

**Response:** `200 OK`
```json
{
  "server_id": "string",
  "name": "string",
  "host": "string",
  "status": "online|offline|setup|security|error|unknown",
  "reachable": true,
  "setup_status": {
    "complete": true,
    "progress": 100,
    "current_step": "completed",
    "last_setup": "2024-01-01T00:00:00Z"
  },
  "security_status": {
    "locked": true,
    "score": 95,
    "last_audit": "2024-01-01T00:00:00Z",
    "issues": []
  },
  "system_health": {
    "uptime": "30d 12h 45m",
    "load_average": [0.25, 0.30, 0.28],
    "memory_usage": "65%",
    "disk_usage": "45%",
    "cpu_usage": "12%"
  },
  "applications": {
    "total": 5,
    "running": 4,
    "stopped": 1,
    "failed": 0
  },
  "last_check": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Server not found

---

### Get Server Health
Perform health check on server and its services.

**Endpoint:** `GET /api/v1/servers/{id}/health`

**Path Parameters:**
- `id` (string, required) - Server ID

**Query Parameters:**
- `include_apps` (boolean, optional) - Include application health (default: false)
- `timeout` (string, optional) - Health check timeout (default: "30s")

**Response:** `200 OK`
```json
{
  "server_id": "string",
  "overall": "healthy|degraded|unhealthy|unknown",
  "connection": {
    "status": "healthy",
    "response_time": "25ms",
    "last_successful": "2024-01-01T00:00:00Z"
  },
  "system": {
    "status": "healthy",
    "uptime": "30d 12h 45m",
    "load": "normal",
    "memory": "normal",
    "disk": "normal"
  },
  "services": {
    "ssh": "healthy",
    "fail2ban": "healthy",
    "firewall": "active"
  },
  "applications": [
    {
      "name": "my-app",
      "status": "healthy",
      "health_url": "https://myapp.example.com/health"
    }
  ],
  "checked_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Server not found
- `408 Request Timeout` - Health check timeout

---

### Get Server Logs
Retrieve server system logs.

**Endpoint:** `GET /api/v1/servers/{id}/logs`

**Path Parameters:**
- `id` (string, required) - Server ID

**Query Parameters:**
- `service` (string, optional) - Filter by service (sshd, fail2ban, etc.)
- `lines` (integer, optional) - Number of log lines (default: 100, max: 1000)
- `since` (string, optional) - Show logs since timestamp (ISO 8601)
- `level` (string, optional) - Filter by log level

**Response:** `200 OK`
```json
{
  "server_id": "string",
  "service": "sshd",
  "lines": 100,
  "logs": "string",
  "log_entries": [
    {
      "timestamp": "2024-01-01T00:00:00Z",
      "level": "info",
      "service": "sshd",
      "message": "Accepted publickey for deploy from 192.168.1.100",
      "source": "system"
    }
  ],
  "retrieved_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Server not found
- `400 Bad Request` - Invalid log parameters

---

### Get Server Metrics
Get server performance metrics and statistics.

**Endpoint:** `GET /api/v1/servers/{id}/metrics`

**Path Parameters:**
- `id` (string, required) - Server ID

**Query Parameters:**
- `period` (string, optional) - Time period: 1h|6h|24h|7d|30d (default: 1h)
- `metrics` (string, optional) - Comma-separated metrics to include

**Response:** `200 OK`
```json
{
  "server_id": "string",
  "period": "1h",
  "current": {
    "cpu_usage": "12%",
    "memory_usage": "4.2GB/8GB",
    "disk_usage": "45GB/100GB", 
    "network_in": "2.1MB/s",
    "network_out": "1.8MB/s",
    "load_average": [0.25, 0.30, 0.28],
    "connections": 45,
    "uptime": "30d 12h 45m"
  },
  "history": [
    {
      "timestamp": "2024-01-01T00:00:00Z",
      "cpu_usage": 12.5,
      "memory_usage": 4294967296,
      "disk_usage": 48318382080,
      "load_average": 0.25
    }
  ],
  "collected_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Server not found
- `400 Bad Request` - Invalid metrics parameters

---

## WebSocket Events

### Server Status Updates
Real-time server status change notifications.

**Endpoint:** `ws://host/api/v1/servers/{id}/status`

**Message Format:**
```json
{
  "server_id": "string",
  "event": "status_changed|health_changed|connection_lost|connection_restored",
  "data": {
    "old_status": "string",
    "new_status": "string",
    "reachable": true,
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

---

### Setup Progress Updates
Real-time server setup progress notifications.

**Endpoint:** `ws://host/api/v1/servers/{id}/setup/progress`

**Message Format:**
```json
{
  "server_id": "string",
  "setup_id": "string",
  "step": "create_user|install_packages|configure_sudo|create_directories",
  "status": "running|success|failed|warning",
  "message": "string",
  "progress_pct": 75,
  "timestamp": "2024-01-01T00:00:00Z"
}
```

---

### Security Progress Updates
Real-time security hardening progress notifications.

**Endpoint:** `ws://host/api/v1/servers/{id}/security/progress`

**Message Format:**
```json
{
  "server_id": "string",
  "security_id": "string",
  "step": "ssh_hardening|firewall_setup|fail2ban_config|system_updates",
  "status": "running|success|failed|warning",
  "message": "string",
  "progress_pct": 60,
  "timestamp": "2024-01-01T00:00:00Z"
}
```

---

## Error Response Format

All error responses follow this format:

```json
{
  "error": {
    "code": "string",
    "message": "string",
    "details": {
      "server": "string",
      "operation": "string",
      "cause": "string"
    },
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

## Common Error Codes

- `SERVER_NOT_FOUND` - Server does not exist
- `SERVER_ALREADY_EXISTS` - Server name or host already registered
- `INVALID_SERVER_CONFIG` - Server configuration is invalid
- `CONNECTION_FAILED` - Cannot connect to server
- `AUTHENTICATION_FAILED` - SSH authentication failed
- `SETUP_ALREADY_COMPLETE` - Server setup already completed
- `SETUP_IN_PROGRESS` - Server setup currently in progress
- `SECURITY_ALREADY_LOCKED` - Server security already applied
- `SECURITY_IN_PROGRESS` - Security hardening currently in progress
- `INSUFFICIENT_PERMISSIONS` - Insufficient permissions on server
- `SERVER_UNREACHABLE` - Server is not reachable
- `INVALID_SSH_KEY` - SSH key format is invalid
- `SUDO_ACCESS_DENIED` - Sudo access not available

## Rate Limiting

- Server operations: 20 requests per minute per server
- Connection tests: 10 requests per minute per server
- Setup/Security: 5 requests per minute per server
- Status/Health: 100 requests per minute
- Logs/Metrics: 50 requests per minute
- Real-time streams: No limit (WebSocket)

## Server Name Validation

Server names must:
- Be 1-50 characters long
- Contain only letters, numbers, hyphens, and underscores
- Start with a letter or number
- Not end with a hyphen or underscore
- Be unique across all servers

## Host Validation

Server hosts must:
- Be valid IP addresses or domain names
- Be reachable on specified port
- Support SSH connections
- Be unique across all servers

## Authentication Methods

Supported authentication methods:
- `ssh_agent` - Use SSH agent for authentication
- `ssh_key` - Use specific SSH key file
- `password` - Use password authentication (not recommended)

## Setup Requirements

Before server setup:
- Server must be reachable via SSH
- Must have sudo access or root access
- SSH connection must be stable
- Server OS must be supported (Ubuntu, CentOS, RHEL, Debian)

## Security Hardening

Security hardening includes:
- SSH configuration hardening
- Firewall configuration (ufw/iptables)
- Fail2ban intrusion prevention
- Automatic security updates
- User access restrictions
- System audit configuration