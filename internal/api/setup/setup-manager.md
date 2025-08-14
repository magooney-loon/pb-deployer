# Setup Manager API

REST API schema for the Setup Manager service.

## Base URL
```
/api/v1/setup
```

## Endpoints

### Create User
Create a new user account on the server.

**Endpoint:** `POST /api/v1/setup/users`

**Request Body:**
```json
{
  "username": "string",
  "home_dir": "string",
  "shell": "/bin/bash",
  "groups": ["sudo", "docker"],
  "create_home": true,
  "system_user": false
}
```

**Response:** `201 Created`
```json
{
  "message": "User created successfully",
  "username": "string",
  "home_dir": "string",
  "shell": "string",
  "groups": ["string"],
  "created_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid user configuration
- `409 Conflict` - User already exists
- `500 Internal Server Error` - User creation failed

---

### Setup SSH Keys
Configure SSH keys for a user account.

**Endpoint:** `POST /api/v1/setup/users/{username}/ssh-keys`

**Path Parameters:**
- `username` (string, required) - Username

**Request Body:**
```json
{
  "keys": [
    "ssh-rsa AAAAB3NzaC1yc2EAAAA... user@example.com",
    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5... user2@example.com"
  ]
}
```

**Response:** `200 OK`
```json
{
  "message": "SSH keys configured successfully",
  "username": "string",
  "keys_count": 2,
  "authorized_keys_path": "/home/user/.ssh/authorized_keys",
  "configured_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - User not found
- `400 Bad Request` - Invalid SSH keys
- `500 Internal Server Error` - SSH key setup failed

---

### Create Directories
Create directories with specified permissions and ownership.

**Endpoint:** `POST /api/v1/setup/directories`

**Request Body:**
```json
{
  "directories": [
    {
      "path": "/opt/myapp",
      "permissions": "755",
      "owner": "appuser",
      "group": "appuser",
      "parents": true
    },
    {
      "path": "/var/log/myapp",
      "permissions": "750",
      "owner": "appuser",
      "group": "adm",
      "parents": true
    }
  ]
}
```

**Response:** `201 Created`
```json
{
  "message": "Directories created successfully",
  "created_count": 2,
  "directories": [
    {
      "path": "/opt/myapp",
      "permissions": "755",
      "owner": "appuser",
      "group": "appuser"
    }
  ],
  "created_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid directory configuration
- `500 Internal Server Error` - Directory creation failed

---

### Configure Sudo Access
Configure sudo access for a user with specific commands.

**Endpoint:** `POST /api/v1/setup/users/{username}/sudo`

**Path Parameters:**
- `username` (string, required) - Username

**Request Body:**
```json
{
  "commands": [
    "/usr/bin/systemctl restart nginx",
    "/usr/bin/systemctl reload nginx",
    "/usr/bin/docker"
  ]
}
```

**Response:** `200 OK`
```json
{
  "message": "Sudo access configured successfully",
  "username": "string",
  "sudo_file": "/etc/sudoers.d/username",
  "commands_count": 3,
  "configured_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - User not found
- `400 Bad Request` - Invalid sudo configuration
- `500 Internal Server Error` - Sudo configuration failed

---

### Install Packages
Install system packages using the detected package manager.

**Endpoint:** `POST /api/v1/setup/packages`

**Request Body:**
```json
{
  "packages": [
    "nginx",
    "docker.io",
    "git",
    "curl"
  ],
  "update_cache": true
}
```

**Response:** `200 OK`
```json
{
  "message": "Packages installed successfully",
  "package_manager": "apt|yum|dnf|pacman|zypper",
  "packages_installed": [
    "nginx",
    "docker.io",
    "git", 
    "curl"
  ],
  "packages_count": 4,
  "installed_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid package list
- `500 Internal Server Error` - Package installation failed

---

### Setup System User
Create a complete system user with all configurations in one operation.

**Endpoint:** `POST /api/v1/setup/system-users`

**Request Body:**
```json
{
  "username": "appuser",
  "home_dir": "/home/appuser",
  "shell": "/bin/bash",
  "groups": ["sudo", "docker"],
  "create_home": true,
  "system_user": false,
  "setup_ssh": true,
  "ssh_keys": [
    "ssh-rsa AAAAB3NzaC1yc2EAAAA... user@example.com"
  ],
  "setup_sudo": true,
  "sudo_commands": [
    "/usr/bin/systemctl restart nginx"
  ],
  "directories": [
    {
      "path": "/opt/myapp",
      "permissions": "755",
      "owner": "appuser",
      "group": "appuser",
      "parents": true
    }
  ]
}
```

**Response:** `201 Created`
```json
{
  "message": "System user setup completed successfully",
  "username": "string",
  "components": {
    "user_created": true,
    "ssh_configured": true,
    "sudo_configured": true,
    "directories_created": 1
  },
  "created_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid system user configuration
- `409 Conflict` - User already exists
- `500 Internal Server Error` - System user setup failed

---

### Get Setup Configuration
Get current setup manager configuration.

**Endpoint:** `GET /api/v1/setup/config`

**Response:** `200 OK`
```json
{
  "default_shell": "/bin/bash",
  "default_groups": ["users"],
  "package_manager": "auto|apt|yum|dnf|pacman|zypper",
  "create_home_by_default": true
}
```

---

### Update Setup Configuration
Update setup manager configuration.

**Endpoint:** `PUT /api/v1/setup/config`

**Request Body:**
```json
{
  "default_shell": "/bin/zsh",
  "default_groups": ["users", "wheel"],
  "package_manager": "apt",
  "create_home_by_default": true
}
```

**Response:** `200 OK`
```json
{
  "message": "Setup configuration updated successfully",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

---

### Get User Information
Get information about an existing user.

**Endpoint:** `GET /api/v1/setup/users/{username}`

**Path Parameters:**
- `username` (string, required) - Username

**Response:** `200 OK`
```json
{
  "username": "string",
  "uid": 1001,
  "gid": 1001,
  "home_dir": "/home/username",
  "shell": "/bin/bash",
  "groups": ["users", "sudo"],
  "ssh_keys_configured": true,
  "sudo_configured": true,
  "last_login": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - User not found

---

### List Users
List all users on the system.

**Endpoint:** `GET /api/v1/setup/users`

**Query Parameters:**
- `system_users` (boolean, optional) - Include system users (default: false)
- `limit` (integer, optional) - Limit number of results (default: 50)
- `offset` (integer, optional) - Offset for pagination (default: 0)

**Response:** `200 OK`
```json
{
  "users": [
    {
      "username": "string",
      "uid": 1001,
      "home_dir": "/home/username",
      "shell": "/bin/bash",
      "groups": ["users"],
      "system_user": false
    }
  ],
  "total": 10,
  "limit": 50,
  "offset": 0
}
```

---

### Check Package Manager
Detect and return information about the system package manager.

**Endpoint:** `GET /api/v1/setup/package-manager`

**Response:** `200 OK`
```json
{
  "detected": "apt|yum|dnf|pacman|zypper",
  "available": ["apt", "snap"],
  "update_command": "apt update",
  "install_command": "apt install -y",
  "last_update": "2024-01-01T00:00:00Z"
}
```

---

## WebSocket Events

### Setup Progress Updates
Real-time setup operation progress updates.

**Endpoint:** `ws://host/api/v1/setup/progress`

**Message Format:**
```json
{
  "operation": "create_user|setup_ssh|create_directories|configure_sudo|install_packages|setup_system_user",
  "step": "string",
  "status": "running|success|failed|warning",
  "message": "string",
  "progress_pct": 75,
  "timestamp": "2024-01-01T00:00:00Z",
  "details": {
    "username": "string",
    "component": "string"
  }
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
      "username": "string",
      "cause": "string"
    },
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

## Common Error Codes

- `INVALID_USERNAME` - Username is invalid or empty
- `USER_ALREADY_EXISTS` - User already exists on system
- `USER_NOT_FOUND` - User does not exist
- `INVALID_SSH_KEY` - SSH key format is invalid
- `INVALID_DIRECTORY_PATH` - Directory path is invalid
- `PERMISSION_DENIED` - Insufficient permissions for operation
- `PACKAGE_MANAGER_NOT_FOUND` - No supported package manager detected
- `PACKAGE_NOT_FOUND` - Specified package not available
- `SUDO_CONFIG_INVALID` - Sudo configuration is invalid
- `HOME_DIRECTORY_EXISTS` - Home directory already exists
- `SHELL_NOT_FOUND` - Specified shell not found on system

## Rate Limiting

- User operations: 10 requests per minute
- Package operations: 5 requests per minute
- Directory operations: 20 requests per minute
- Configuration: 15 requests per minute
- Status/Info: 50 requests per minute

## User Validation Rules

Usernames must:
- Be 1-32 characters long
- Contain only lowercase letters, numbers, hyphens, and underscores
- Start with a lowercase letter
- Not end with a hyphen or underscore
- Not be a reserved system username

## SSH Key Validation

SSH keys must:
- Be in valid OpenSSH format
- Support RSA (2048+ bits), Ed25519, ECDSA, or DSA key types
- Include valid base64 encoded key data
- Optionally include comment/identifier

## Directory Path Validation

Directory paths must:
- Be absolute paths starting with /
- Not contain special characters except hyphen, underscore, and dot
- Not exceed 4096 characters in length
- Not be system-critical paths (/, /etc, /usr, /var without subdirectories)