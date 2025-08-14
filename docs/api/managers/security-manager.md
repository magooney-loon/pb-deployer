# Security Manager API

REST API schema for the Security Manager service.

## Base URL
```
/api/v1/security
```

## Endpoints

### Apply Security Lockdown
Apply comprehensive security lockdown configuration to the system.

**Endpoint:** `POST /api/v1/security/lockdown`

**Request Body:**
```json
{
  "disable_root_login": true,
  "disable_password_auth": true,
  "firewall_rules": [
    {
      "port": 22,
      "protocol": "tcp",
      "action": "allow",
      "source": "192.168.1.0/24",
      "description": "SSH access from internal network"
    }
  ],
  "allowed_ports": [22, 80, 443],
  "allowed_users": ["admin", "deploy"],
  "fail2ban_config": {
    "enabled": true,
    "max_retries": 5,
    "ban_time": "1h",
    "find_time": "10m",
    "services": ["ssh", "apache", "nginx"]
  },
  "ssh_hardening_config": {
    "password_authentication": false,
    "pubkey_authentication": true,
    "permit_root_login": false,
    "x11_forwarding": false,
    "max_auth_tries": 3,
    "client_alive_interval": 300
  }
}
```

**Response:** `200 OK`
```json
{
  "message": "Security lockdown completed successfully",
  "applied_at": "2024-01-01T00:00:00Z",
  "components": {
    "ssh_hardened": true,
    "firewall_configured": true,
    "fail2ban_setup": true,
    "additional_security": true
  }
}
```

**Error Responses:**
- `400 Bad Request` - Invalid security configuration
- `500 Internal Server Error` - Lockdown operation failed

---

### Configure Firewall
Configure firewall rules using available firewall system.

**Endpoint:** `POST /api/v1/security/firewall`

**Request Body:**
```json
{
  "rules": [
    {
      "port": 80,
      "protocol": "tcp",
      "action": "allow",
      "source": "0.0.0.0/0",
      "description": "HTTP traffic"
    },
    {
      "port": 443,
      "protocol": "tcp", 
      "action": "allow",
      "source": "0.0.0.0/0",
      "description": "HTTPS traffic"
    }
  ],
  "reset_existing": false
}
```

**Response:** `200 OK`
```json
{
  "message": "Firewall configured successfully",
  "firewall_type": "ufw|iptables|firewalld",
  "rules_applied": 5,
  "configured_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid firewall rules
- `500 Internal Server Error` - Firewall configuration failed

---

### Setup Fail2ban
Configure fail2ban intrusion prevention system.

**Endpoint:** `POST /api/v1/security/fail2ban`

**Request Body:**
```json
{
  "enabled": true,
  "max_retries": 5,
  "ban_time": "1h",
  "find_time": "10m",
  "services": ["ssh", "apache", "nginx"],
  "whitelist_ips": ["192.168.1.0/24"]
}
```

**Response:** `200 OK`
```json
{
  "message": "Fail2ban configured successfully",
  "installed": true,
  "service_active": true,
  "jails_configured": ["ssh", "apache", "nginx"],
  "configured_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid fail2ban configuration
- `500 Internal Server Error` - Fail2ban setup failed

---

### Harden SSH Configuration
Apply SSH security hardening settings.

**Endpoint:** `POST /api/v1/security/ssh/harden`

**Request Body:**
```json
{
  "password_authentication": false,
  "pubkey_authentication": true,
  "permit_root_login": false,
  "x11_forwarding": false,
  "allow_agent_forwarding": false,
  "allow_tcp_forwarding": false,
  "client_alive_interval": 300,
  "client_alive_count_max": 2,
  "max_auth_tries": 3,
  "max_sessions": 10,
  "protocol": 2
}
```

**Response:** `200 OK`
```json
{
  "message": "SSH hardening completed successfully",
  "config_backup_created": true,
  "ssh_service_restarted": true,
  "hardened_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid SSH configuration
- `500 Internal Server Error` - SSH hardening failed

---

### Configure Automatic Updates
Setup automatic security updates.

**Endpoint:** `POST /api/v1/security/auto-updates`

**Request Body:**
```json
{
  "enable": true,
  "security_only": true,
  "reboot_if_required": false,
  "notification_email": "admin@example.com"
}
```

**Response:** `200 OK`
```json
{
  "message": "Automatic updates configured successfully",
  "package_manager": "apt|yum|dnf",
  "service_enabled": true,
  "configured_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid update configuration
- `500 Internal Server Error` - Auto-updates configuration failed

---

### Security Audit
Perform comprehensive security audit and compliance checking.

**Endpoint:** `GET /api/v1/security/audit`

**Query Parameters:**
- `include_recommendations` (boolean, optional) - Include security recommendations (default: true)
- `category` (string, optional) - Filter by category: ssh|firewall|system|intrusion_prevention

**Response:** `200 OK`
```json
{
  "timestamp": "2024-01-01T00:00:00Z",
  "overall": "excellent|good|fair|poor|critical",
  "score": 95,
  "checks": [
    {
      "name": "SSH Configuration",
      "category": "ssh",
      "status": "pass|warning|fail",
      "score": 100,
      "issues": [],
      "details": {
        "ssh_active": true,
        "password_auth_disabled": true,
        "root_login_disabled": true
      }
    },
    {
      "name": "Firewall Configuration", 
      "category": "firewall",
      "status": "pass",
      "score": 100,
      "issues": [],
      "details": {
        "firewall_active": true,
        "firewall_type": "ufw"
      }
    }
  ],
  "recommendations": [
    "Enable two-factor authentication for SSH",
    "Configure log monitoring and alerting"
  ]
}
```

---

### Get Security Status
Get current security configuration status.

**Endpoint:** `GET /api/v1/security/status`

**Response:** `200 OK`
```json
{
  "ssh": {
    "hardened": true,
    "password_auth": false,
    "root_login": false,
    "service_active": true
  },
  "firewall": {
    "type": "ufw|iptables|firewalld",
    "active": true,
    "rules_count": 5,
    "default_policy": "deny"
  },
  "fail2ban": {
    "installed": true,
    "active": true,
    "jails_active": ["ssh", "apache"],
    "banned_ips": 3
  },
  "auto_updates": {
    "enabled": true,
    "last_update": "2024-01-01T00:00:00Z",
    "pending_updates": 0
  },
  "last_audit": "2024-01-01T00:00:00Z",
  "overall_score": 95
}
```

---

### Get Security Configuration
Get current security manager configuration.

**Endpoint:** `GET /api/v1/security/config`

**Response:** `200 OK`
```json
{
  "disable_root_login": true,
  "disable_password_auth": true,
  "allowed_ports": [22, 80, 443],
  "allowed_users": ["admin"],
  "firewall_rules": [
    {
      "port": 22,
      "protocol": "tcp",
      "action": "allow",
      "description": "SSH access"
    }
  ],
  "fail2ban_config": {
    "enabled": true,
    "max_retries": 5,
    "ban_time": "1h",
    "services": ["ssh"]
  }
}
```

---

### Update Security Configuration
Update security manager configuration.

**Endpoint:** `PUT /api/v1/security/config`

**Request Body:**
```json
{
  "disable_root_login": true,
  "disable_password_auth": true,
  "allowed_ports": [22, 80, 443, 8080],
  "allowed_users": ["admin", "deploy"]
}
```

**Response:** `200 OK`
```json
{
  "message": "Security configuration updated successfully",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

---

## WebSocket Events

### Security Progress Updates
Real-time security operation progress updates.

**Endpoint:** `ws://host/api/v1/security/progress`

**Message Format:**
```json
{
  "operation": "lockdown|firewall|fail2ban|ssh_harden|audit",
  "step": "string",
  "status": "running|success|failed|warning", 
  "message": "string",
  "progress_pct": 75,
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
      "operation": "string",
      "component": "string",
      "cause": "string"
    },
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

## Common Error Codes

- `INVALID_SECURITY_CONFIG` - Security configuration is invalid
- `SSH_CONFIG_FAILED` - SSH configuration operation failed
- `FIREWALL_NOT_SUPPORTED` - Firewall system not supported
- `FAIL2BAN_INSTALL_FAILED` - Fail2ban installation failed
- `SERVICE_NOT_FOUND` - Required service not found
- `PERMISSION_DENIED` - Insufficient permissions for operation
- `BACKUP_FAILED` - Configuration backup failed
- `VALIDATION_FAILED` - Configuration validation failed

## Rate Limiting

- Configuration operations: 10 requests per minute
- Status/Audit: 30 requests per minute
- Real-time updates: No limit (WebSocket)