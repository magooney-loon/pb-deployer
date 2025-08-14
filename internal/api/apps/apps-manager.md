# Apps Manager API

REST API schema for the Apps Manager service.

## Base URL
```
/api/v1/apps
```

## Endpoints

### Create Application
Create a new application on a server.

**Endpoint:** `POST /api/v1/apps`

**Request Body:**
```json
{
  "name": "string",
  "server_id": "string",
  "domain": "string",
  "remote_path": "string",
  "service_name": "string",
  "environment": "production|staging|development",
  "health_check_url": "string",
  "auto_start": true,
  "description": "string"
}
```

**Response:** `201 Created`
```json
{
  "id": "string",
  "name": "string",
  "server_id": "string",
  "server_name": "string",
  "domain": "string",
  "remote_path": "/opt/pocketbase/apps/my-app",
  "service_name": "pocketbase-my-app",
  "environment": "production",
  "status": "offline",
  "current_version": "string",
  "health_check_url": "https://myapp.example.com/api/health",
  "auto_start": true,
  "description": "string",
  "created": "2024-01-01T00:00:00Z",
  "updated": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid application specification
- `409 Conflict` - App name already exists on server
- `500 Internal Server Error` - App creation failed

---

### Get Application
Get application details and current status.

**Endpoint:** `GET /api/v1/apps/{name}`

**Path Parameters:**
- `name` (string, required) - Application name

**Response:** `200 OK`
```json
{
  "id": "string",
  "name": "string",
  "server_id": "string",
  "server_name": "string",
  "server_host": "string",
  "domain": "string",
  "remote_path": "string",
  "service_name": "string",
  "environment": "string",
  "status": "online|offline|deploying|failed|unknown",
  "health": "healthy|unhealthy|degraded|unknown",
  "current_version": "string",
  "last_deployment": "2024-01-01T00:00:00Z",
  "uptime": "5d 12h 30m",
  "replicas": {
    "desired": 1,
    "running": 1,
    "healthy": 1
  },
  "metrics": {
    "cpu_usage": "2.5%",
    "memory_usage": "128MB",
    "disk_usage": "1.2GB"
  },
  "health_check_url": "string",
  "auto_start": true,
  "description": "string",
  "created": "2024-01-01T00:00:00Z",
  "updated": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Application not found

---

### List Applications
List all applications with optional filtering.

**Endpoint:** `GET /api/v1/apps`

**Query Parameters:**
- `server_id` (string, optional) - Filter by server
- `status` (string, optional) - Filter by status
- `environment` (string, optional) - Filter by environment
- `limit` (integer, optional) - Limit number of results (default: 50)
- `offset` (integer, optional) - Offset for pagination (default: 0)

**Response:** `200 OK`
```json
{
  "applications": [
    {
      "id": "string",
      "name": "string",
      "server_name": "string",
      "domain": "string",
      "environment": "string",
      "status": "string",
      "health": "string",
      "current_version": "string",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 25,
  "limit": 50,
  "offset": 0
}
```

---

### Update Application
Update application configuration.

**Endpoint:** `PUT /api/v1/apps/{name}`

**Path Parameters:**
- `name` (string, required) - Application name

**Request Body:**
```json
{
  "domain": "string",
  "environment": "string",
  "health_check_url": "string",
  "auto_start": true,
  "description": "string"
}
```

**Response:** `200 OK`
```json
{
  "message": "Application updated successfully",
  "app_id": "string",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid update data
- `404 Not Found` - Application not found
- `409 Conflict` - Configuration conflict
- `500 Internal Server Error` - Update failed

---

### Delete Application
Delete an application and cleanup resources.

**Endpoint:** `DELETE /api/v1/apps/{name}`

**Path Parameters:**
- `name` (string, required) - Application name

**Query Parameters:**
- `cleanup_files` (boolean, optional) - Remove application files (default: false)
- `stop_service` (boolean, optional) - Stop service before deletion (default: true)

**Response:** `200 OK`
```json
{
  "message": "Application deleted successfully",
  "app_id": "string",
  "cleanup_performed": true,
  "deleted_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Application not found
- `409 Conflict` - Application is running and stop_service is false
- `500 Internal Server Error` - Deletion failed

---

### Deploy Application
Deploy a new version of the application.

**Endpoint:** `POST /api/v1/apps/{name}/deploy`

**Path Parameters:**
- `name` (string, required) - Application name

**Request Body:**
```json
{
  "version_id": "string",
  "strategy": "rolling|blue-green|recreate",
  "health_check_timeout": "5m",
  "rollback_on_failure": true,
  "superuser_email": "string",
  "superuser_password": "string",
  "environment_vars": {
    "key": "value"
  },
  "pre_deploy_hooks": ["string"],
  "post_deploy_hooks": ["string"],
  "notes": "string"
}
```

**Response:** `202 Accepted`
```json
{
  "message": "Deployment started",
  "deployment_id": "string",
  "app_id": "string",
  "version_id": "string",
  "strategy": "rolling",
  "is_first_deploy": false,
  "estimated_duration": "3m",
  "started_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid deployment specification
- `404 Not Found` - Application or version not found
- `409 Conflict` - Deployment already in progress
- `500 Internal Server Error` - Deployment initiation failed

---

### Rollback Application
Rollback application to a previous version.

**Endpoint:** `POST /api/v1/apps/{name}/rollback`

**Path Parameters:**
- `name` (string, required) - Application name

**Request Body:**
```json
{
  "version_id": "string",
  "strategy": "rolling|recreate",
  "notes": "string"
}
```

**Response:** `202 Accepted`
```json
{
  "message": "Rollback started",
  "deployment_id": "string",
  "app_id": "string",
  "version_id": "string",
  "rollback_from": "string",
  "started_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid rollback specification
- `404 Not Found` - Application or version not found
- `409 Conflict` - Operation already in progress
- `500 Internal Server Error` - Rollback initiation failed

---

### Start Application Service
Start the application service.

**Endpoint:** `POST /api/v1/apps/{name}/start`

**Path Parameters:**
- `name` (string, required) - Application name

**Request Body:**
```json
{
  "force": false,
  "wait_for_health": true,
  "timeout": "2m"
}
```

**Response:** `200 OK`
```json
{
  "message": "Service started successfully",
  "app_id": "string",
  "service_name": "string",
  "action": "start",
  "status": "running",
  "health": "healthy",
  "started_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Application not found
- `409 Conflict` - Service already running
- `500 Internal Server Error` - Service start failed

---

### Stop Application Service
Stop the application service.

**Endpoint:** `POST /api/v1/apps/{name}/stop`

**Path Parameters:**
- `name` (string, required) - Application name

**Request Body:**
```json
{
  "force": false,
  "graceful_timeout": "30s"
}
```

**Response:** `200 OK`
```json
{
  "message": "Service stopped successfully",
  "app_id": "string",
  "service_name": "string",
  "action": "stop",
  "status": "stopped",
  "stopped_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Application not found
- `500 Internal Server Error` - Service stop failed

---

### Restart Application Service
Restart the application service.

**Endpoint:** `POST /api/v1/apps/{name}/restart`

**Path Parameters:**
- `name` (string, required) - Application name

**Request Body:**
```json
{
  "force": false,
  "wait_for_health": true,
  "graceful_timeout": "30s",
  "startup_timeout": "2m"
}
```

**Response:** `200 OK`
```json
{
  "message": "Service restarted successfully",
  "app_id": "string",
  "service_name": "string",
  "action": "restart",
  "status": "running",
  "health": "healthy",
  "restarted_at": "2024-01-01T00:00:00Z",
  "downtime": "5.2s"
}
```

**Error Responses:**
- `404 Not Found` - Application not found
- `500 Internal Server Error` - Service restart failed

---

### Get Application Status
Get current application and service status.

**Endpoint:** `GET /api/v1/apps/{name}/status`

**Path Parameters:**
- `name` (string, required) - Application name

**Response:** `200 OK`
```json
{
  "app_id": "string",
  "name": "string",
  "status": "online|offline|deploying|failed|unknown",
  "health": "healthy|unhealthy|degraded|unknown",
  "service": {
    "name": "string",
    "status": "running|stopped|failed|unknown",
    "active": true,
    "enabled": true,
    "uptime": "5d 12h 30m",
    "restart_count": 0,
    "last_restart": "2024-01-01T00:00:00Z"
  },
  "deployment": {
    "current_version": "string",
    "last_deployment": "2024-01-01T00:00:00Z",
    "deployment_status": "success|failed|in_progress",
    "deployment_id": "string"
  },
  "health_check": {
    "url": "string",
    "status": "healthy|unhealthy|unknown",
    "last_check": "2024-01-01T00:00:00Z",
    "response_time": "150ms",
    "status_code": 200
  },
  "server": {
    "id": "string",
    "name": "string",
    "host": "string",
    "reachable": true
  },
  "checked_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Application not found

---

### Perform Health Check
Trigger an immediate health check for the application.

**Endpoint:** `POST /api/v1/apps/{name}/health-check`

**Path Parameters:**
- `name` (string, required) - Application name

**Request Body:**
```json
{
  "timeout": "10s",
  "update_status": true
}
```

**Response:** `200 OK`
```json
{
  "app_id": "string",
  "health_url": "string",
  "status": "healthy|unhealthy|unknown",
  "response_time": "150ms",
  "status_code": 200,
  "message": "string",
  "checks": [
    {
      "name": "http_response",
      "status": "healthy",
      "message": "HTTP 200 OK",
      "duration": "150ms"
    },
    {
      "name": "database_connection",
      "status": "healthy",
      "message": "Database accessible",
      "duration": "25ms"
    }
  ],
  "checked_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Application not found
- `400 Bad Request` - Health check configuration invalid

---

### Get Application Logs
Retrieve application service logs.

**Endpoint:** `GET /api/v1/apps/{name}/logs`

**Path Parameters:**
- `name` (string, required) - Application name

**Query Parameters:**
- `lines` (integer, optional) - Number of log lines (default: 100, max: 1000)
- `since` (string, optional) - Show logs since timestamp (ISO 8601)
- `level` (string, optional) - Filter by log level
- `follow` (boolean, optional) - Follow logs via WebSocket (default: false)

**Response:** `200 OK`
```json
{
  "app_id": "string",
  "service_name": "string",
  "lines": 100,
  "logs": "string",
  "log_entries": [
    {
      "timestamp": "2024-01-01T00:00:00Z",
      "level": "info",
      "message": "string",
      "source": "application"
    }
  ],
  "retrieved_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Application not found
- `400 Bad Request` - Invalid log parameters
- `500 Internal Server Error` - Log retrieval failed

---

### Get Application Metrics
Get application performance metrics.

**Endpoint:** `GET /api/v1/apps/{name}/metrics`

**Path Parameters:**
- `name` (string, required) - Application name

**Query Parameters:**
- `period` (string, optional) - Time period: 1h|6h|24h|7d|30d (default: 1h)
- `metrics` (string, optional) - Comma-separated metrics to include

**Response:** `200 OK`
```json
{
  "app_id": "string",
  "period": "1h",
  "current": {
    "cpu_usage": "2.5%",
    "memory_usage": "128MB",
    "disk_usage": "1.2GB",
    "network_in": "1.5MB/s",
    "network_out": "800KB/s",
    "response_time": "150ms",
    "requests_per_second": 45.2,
    "error_rate": "0.1%"
  },
  "history": [
    {
      "timestamp": "2024-01-01T00:00:00Z",
      "cpu_usage": 2.5,
      "memory_usage": 134217728,
      "response_time": 150
    }
  ],
  "collected_at": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Application not found
- `400 Bad Request` - Invalid metrics parameters

---

### Get Application Configuration
Get current application configuration.

**Endpoint:** `GET /api/v1/apps/{name}/config`

**Path Parameters:**
- `name` (string, required) - Application name

**Response:** `200 OK`
```json
{
  "app_id": "string",
  "name": "string",
  "domain": "string",
  "remote_path": "string",
  "service_name": "string",
  "environment": "string",
  "health_check_url": "string",
  "auto_start": true,
  "environment_vars": {
    "key": "value"
  },
  "service_config": {
    "restart_policy": "always",
    "user": "pocketbase",
    "working_directory": "/opt/pocketbase/apps/my-app"
  }
}
```

---

### Update Application Configuration
Update application configuration.

**Endpoint:** `PUT /api/v1/apps/{name}/config`

**Path Parameters:**
- `name` (string, required) - Application name

**Request Body:**
```json
{
  "domain": "string",
  "environment": "string",
  "health_check_url": "string",
  "auto_start": true,
  "environment_vars": {
    "key": "value"
  },
  "restart_service": true
}
```

**Response:** `200 OK`
```json
{
  "message": "Configuration updated successfully",
  "app_id": "string",
  "restart_required": true,
  "updated_at": "2024-01-01T00:00:00Z"
}
```

---

## WebSocket Events

### Application Status Updates
Real-time application status change notifications.

**Endpoint:** `ws://host/api/v1/apps/{name}/status`

**Message Format:**
```json
{
  "app_id": "string",
  "event": "status_changed|health_changed|deployment_progress",
  "data": {
    "old_status": "string",
    "new_status": "string",
    "health": "string",
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

---

### Deployment Progress Updates
Real-time deployment progress for application.

**Endpoint:** `ws://host/api/v1/apps/{name}/deploy/progress`

**Message Format:**
```json
{
  "app_id": "string",
  "deployment_id": "string",
  "step": "string",
  "status": "running|success|failed|warning",
  "message": "string",
  "progress_pct": 75,
  "timestamp": "2024-01-01T00:00:00Z"
}
```

---

### Live Logs
Real-time application logs streaming.

**Endpoint:** `ws://host/api/v1/apps/{name}/logs/live`

**Message Format:**
```json
{
  "app_id": "string",
  "service_name": "string",
  "timestamp": "2024-01-01T00:00:00Z",
  "level": "info|warning|error|debug",
  "message": "string",
  "source": "application|system"
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
      "app": "string",
      "operation": "string",
      "cause": "string"
    },
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

## Common Error Codes

- `APP_NOT_FOUND` - Application does not exist
- `APP_ALREADY_EXISTS` - Application name already exists on server
- `INVALID_APP_NAME` - Application name is invalid
- `SERVER_NOT_FOUND` - Target server does not exist
- `SERVER_UNREACHABLE` - Cannot connect to target server
- `SERVICE_ACTION_FAILED` - Service operation failed
- `DEPLOYMENT_IN_PROGRESS` - Another deployment is already running
- `VERSION_NOT_FOUND` - Specified version does not exist
- `HEALTH_CHECK_FAILED` - Application health check failed
- `INSUFFICIENT_RESOURCES` - Server lacks resources for operation
- `PERMISSION_DENIED` - Insufficient permissions for operation
- `CONFIGURATION_INVALID` - Application configuration is invalid

## Rate Limiting

- Application operations: 30 requests per minute per app
- Status/Health checks: 100 requests per minute
- Log retrieval: 50 requests per minute
- Deployment operations: 5 requests per minute per app
- Real-time streams: No limit (WebSocket)

## Application Name Validation

Application names must:
- Be 1-50 characters long
- Contain only letters, numbers, hyphens, and underscores
- Start with a letter or number
- Not end with a hyphen or underscore
- Be unique per server
- Not conflict with system service names

## Service Management

Service operations depend on server security status:
- **Pre-security servers**: Direct systemctl via root SSH
- **Post-security servers**: systemctl via app user with sudo permissions
- **Service naming**: Follows pattern `pocketbase-{app-name}`
- **Service files**: Located in `/etc/systemd/system/`

## Health Check Configuration

Health checks support:
- **HTTP/HTTPS endpoints**: Status code and response time validation
- **TCP connections**: Port availability checks
- **Custom scripts**: User-defined health validation
- **Timeout configuration**: Configurable timeout per check
- **Retry logic**: Automatic retry on transient failures

## Deployment Strategies

### Rolling Deployment
- **Description**: Gradual replacement of old version
- **Downtime**: Minimal (seconds)
- **Risk**: Low
- **Rollback**: Fast

### Blue-Green Deployment
- **Description**: Complete environment switch
- **Downtime**: Near zero
- **Risk**: Medium
- **Rollback**: Instant

### Recreate Deployment
- **Description**: Stop old, start new
- **Downtime**: High (minutes)
- **Risk**: Low
- **Rollback**: Manual

## Integration Points

### Server Manager Integration
- Validates server readiness before operations
- Uses server authentication configuration
- Respects server security status

### Version Manager Integration
- Validates version existence before deployment
- Downloads deployment packages
- Manages version lifecycle

### Service Manager Integration
- Controls systemd services
- Monitors service health
- Manages service configurations

### Deployment Manager Integration
- Orchestrates deployment process
- Tracks deployment history
- Handles rollback operations