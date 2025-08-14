# Service Manager API

REST API schema for the Service Manager service.

## Base URL
```
/api/v1/services
```

## Endpoints

### Manage Service
Perform service management operations (start, stop, restart, reload).

**Endpoint:** `POST /api/v1/services/{name}/action`

**Path Parameters:**
- `name` (string, required) - Service name

**Request Body:**
```json
{
  "action": "start|stop|restart|reload"
}
```

**Response:** `200 OK`
```json
{
  "message": "Service action completed successfully",
  "service": "string",
  "action": "string",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Service not found
- `400 Bad Request` - Invalid action
- `500 Internal Server Error` - Service operation failed

---

### Get Service Status
Get the current status of a service.

**Endpoint:** `GET /api/v1/services/{name}`

**Path Parameters:**
- `name` (string, required) - Service name

**Response:** `200 OK`
```json
{
  "name": "string",
  "state": "running|stopped|failed|inactive",
  "active": true,
  "enabled": true,
  "description": "string",
  "since": "2024-01-01T00:00:00Z",
  "memory_usage": "50MB",
  "cpu_usage": "2.5%",
  "restart_count": 0
}
```

**Error Responses:**
- `404 Not Found` - Service not found
- `500 Internal Server Error` - Failed to get service status

---

### Get Service Logs
Retrieve service logs from journalctl.

**Endpoint:** `GET /api/v1/services/{name}/logs`

**Path Parameters:**
- `name` (string, required) - Service name

**Query Parameters:**
- `lines` (integer, optional) - Number of log lines to retrieve (default: 50, max: 1000)
- `follow` (boolean, optional) - Follow logs in real-time via WebSocket (default: false)
- `since` (string, optional) - Show logs since timestamp (ISO 8601 format)
- `level` (string, optional) - Filter by log level: debug|info|warning|error

**Response:** `200 OK`
```json
{
  "service": "string",
  "logs": "string",
  "lines": 50,
  "retrieved_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Service not found
- `400 Bad Request` - Invalid parameters
- `500 Internal Server Error` - Failed to retrieve logs

---

### Enable Service
Enable a service to start automatically on boot.

**Endpoint:** `POST /api/v1/services/{name}/enable`

**Path Parameters:**
- `name` (string, required) - Service name

**Response:** `200 OK`
```json
{
  "message": "Service enabled successfully",
  "service": "string",
  "enabled": true,
  "timestamp": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Service not found
- `500 Internal Server Error` - Failed to enable service

---

### Disable Service
Disable a service from starting automatically on boot.

**Endpoint:** `POST /api/v1/services/{name}/disable`

**Path Parameters:**
- `name` (string, required) - Service name

**Response:** `200 OK`
```json
{
  "message": "Service disabled successfully",
  "service": "string",
  "enabled": false,
  "timestamp": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Service not found
- `500 Internal Server Error` - Failed to disable service

---

### Create Service File
Create a new systemd service file.

**Endpoint:** `POST /api/v1/services`

**Request Body:**
```json
{
  "name": "string",
  "description": "string",
  "type": "simple|forking|oneshot|notify",
  "exec_start": "string",
  "exec_stop": "string",
  "exec_reload": "string",
  "user": "string",
  "group": "string",
  "working_directory": "string",
  "environment": {
    "KEY": "value"
  },
  "restart": "no|on-success|on-failure|on-abnormal|on-abort|always",
  "restart_sec": "5s",
  "after": ["network.target"],
  "requires": ["string"],
  "wanted_by": "multi-user.target",
  "enabled": true
}
```

**Response:** `201 Created`
```json
{
  "message": "Service file created successfully",
  "service": "string",
  "service_path": "/etc/systemd/system/service.service",
  "enabled": true,
  "created_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid service definition
- `409 Conflict` - Service already exists
- `500 Internal Server Error` - Failed to create service file

---

### Wait for Service
Wait for a service to reach the desired state with timeout.

**Endpoint:** `POST /api/v1/services/{name}/wait`

**Path Parameters:**
- `name` (string, required) - Service name

**Request Body:**
```json
{
  "timeout": "5m",
  "desired_state": "running|stopped"
}
```

**Response:** `200 OK`
```json
{
  "message": "Service reached desired state",
  "service": "string",
  "state": "string",
  "elapsed_time": "30s",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Service not found
- `408 Request Timeout` - Service did not reach desired state within timeout
- `500 Internal Server Error` - Wait operation failed

---

### List Services
List all services or filter by criteria.

**Endpoint:** `GET /api/v1/services`

**Query Parameters:**
- `state` (string, optional) - Filter by state: running|stopped|failed|inactive
- `enabled` (boolean, optional) - Filter by enabled status
- `pattern` (string, optional) - Filter by service name pattern
- `limit` (integer, optional) - Limit number of results (default: 50)
- `offset` (integer, optional) - Offset for pagination (default: 0)

**Response:** `200 OK`
```json
{
  "services": [
    {
      "name": "string",
      "state": "string",
      "active": true,
      "enabled": true,
      "description": "string"
    }
  ],
  "total": 25,
  "limit": 50,
  "offset": 0
}
```

---

### Delete Service File
Delete a systemd service file.

**Endpoint:** `DELETE /api/v1/services/{name}`

**Path Parameters:**
- `name` (string, required) - Service name

**Query Parameters:**
- `stop_first` (boolean, optional) - Stop service before deletion (default: true)

**Response:** `200 OK`
```json
{
  "message": "Service file deleted successfully",
  "service": "string",
  "stopped": true,
  "deleted_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Service not found
- `409 Conflict` - Service is running and stop_first is false
- `500 Internal Server Error` - Failed to delete service

---

### Get Service Configuration
Get current service manager configuration.

**Endpoint:** `GET /api/v1/services/config`

**Response:** `200 OK`
```json
{
  "action_timeout": "60s",
  "status_check_interval": "2s",
  "default_log_lines": 50,
  "max_log_lines": 1000
}
```

---

### Update Service Configuration
Update service manager configuration.

**Endpoint:** `PUT /api/v1/services/config`

**Request Body:**
```json
{
  "action_timeout": "90s",
  "status_check_interval": "5s",
  "default_log_lines": 100,
  "max_log_lines": 2000
}
```

**Response:** `200 OK`
```json
{
  "message": "Service configuration updated successfully",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

---

## WebSocket Endpoints

### Follow Service Logs
Stream service logs in real-time.

**Endpoint:** `ws://host/api/v1/services/{name}/logs/follow`

**Message Format:**
```json
{
  "timestamp": "2024-01-01T00:00:00.000Z",
  "level": "info|warning|error|debug",
  "message": "string",
  "service": "string"
}
```

---

### Service Status Updates
Real-time service status change notifications.

**Endpoint:** `ws://host/api/v1/services/status/watch`

**Query Parameters:**
- `services` (string, optional) - Comma-separated list of services to watch
- `states` (string, optional) - Comma-separated list of states to filter

**Message Format:**
```json
{
  "service": "string",
  "old_state": "string",
  "new_state": "string",
  "active": true,
  "timestamp": "2024-01-01T00:00:00Z"
}
```

---

### Progress Updates
Real-time service operation progress updates.

**Endpoint:** `ws://host/api/v1/services/{name}/progress`

**Message Format:**
```json
{
  "service": "string",
  "operation": "start|stop|restart|reload|enable|disable|create|wait",
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
      "service": "string",
      "operation": "string",
      "cause": "string"
    },
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

## Common Error Codes

- `SERVICE_NOT_FOUND` - Service does not exist
- `INVALID_SERVICE_NAME` - Service name is invalid
- `INVALID_ACTION` - Service action is not supported
- `SERVICE_ALREADY_RUNNING` - Service is already in requested state
- `SERVICE_OPERATION_FAILED` - Service operation failed to execute
- `SERVICE_DEFINITION_INVALID` - Service definition is invalid
- `PERMISSION_DENIED` - Insufficient permissions for operation
- `TIMEOUT_EXCEEDED` - Operation exceeded timeout limit
- `SYSTEMD_ERROR` - Systemd operation failed

## Rate Limiting

- Service operations: 30 requests per minute per service
- Status checks: 100 requests per minute
- Log retrieval: 50 requests per minute
- Real-time streams: No limit (WebSocket)

## Service Name Validation

Service names must:
- Be 1-64 characters long
- Contain only letters, numbers, hyphens, and underscores
- Start with a letter or number
- Not end with a hyphen or underscore
- Not contain consecutive special characters