# Deployment Manager API

REST API schema for the Deployment Manager service.

## Base URL
```
/api/v1/deployments
```

## Endpoints

### Deploy Application
Deploy a new version of an application.

**Endpoint:** `POST /api/v1/deployments`

**Request Body:**
```json
{
  "name": "string",
  "version": "string",
  "environment": "string",
  "strategy": "rolling|blue-green|canary|recreate",
  "artifact_path": "string",
  "working_directory": "string",
  "service_name": "string",
  "health_check_url": "string",
  "rollback_on_failure": true,
  "dependencies": ["string"],
  "pre_deploy_hooks": ["string"],
  "post_deploy_hooks": ["string"]
}
```

**Response:** `201 Created`
```json
{
  "version": "string",
  "start_time": "2024-01-01T00:00:00Z",
  "end_time": "2024-01-01T00:05:00Z",
  "duration": "5m0s",
  "success": true,
  "message": "string",
  "previous_version": "string",
  "steps": [
    {
      "name": "string",
      "status": "running|completed|failed",
      "message": "string",
      "start_time": "2024-01-01T00:00:00Z",
      "end_time": "2024-01-01T00:01:00Z",
      "error": "string"
    }
  ]
}
```

**Error Responses:**
- `400 Bad Request` - Invalid deployment specification
- `409 Conflict` - Deployment already in progress
- `500 Internal Server Error` - Deployment failed

---

### Rollback Deployment
Rollback a deployment to a previous version.

**Endpoint:** `POST /api/v1/deployments/{name}/rollback`

**Path Parameters:**
- `name` (string, required) - Deployment name

**Request Body:**
```json
{
  "version": "string"
}
```

**Response:** `200 OK`
```json
{
  "message": "Rollback completed successfully",
  "version": "string",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Deployment not found
- `400 Bad Request` - Invalid version or backup not available
- `500 Internal Server Error` - Rollback failed

---

### Get Deployment Status
Get the current status of a deployment.

**Endpoint:** `GET /api/v1/deployments/{name}`

**Path Parameters:**
- `name` (string, required) - Deployment name

**Response:** `200 OK`
```json
{
  "name": "string",
  "version": "string",
  "environment": "string",
  "state": "unknown|deploying|healthy|unhealthy|failed|rolling_back",
  "health": "unknown|healthy|unhealthy|degraded",
  "last_updated": "2024-01-01T00:00:00Z",
  "replicas": {
    "desired": 3,
    "available": 2,
    "ready": 2,
    "updated": 1
  },
  "configuration": {
    "key": "value"
  },
  "events": [
    {
      "timestamp": "2024-01-01T00:00:00Z",
      "type": "info|warning|error",
      "message": "string",
      "source": "string"
    }
  ]
}
```

**Error Responses:**
- `404 Not Found` - Deployment not found

---

### List Deployments
List all deployments.

**Endpoint:** `GET /api/v1/deployments`

**Query Parameters:**
- `environment` (string, optional) - Filter by environment
- `state` (string, optional) - Filter by state
- `limit` (integer, optional) - Limit number of results (default: 50)
- `offset` (integer, optional) - Offset for pagination (default: 0)

**Response:** `200 OK`
```json
{
  "deployments": [
    {
      "name": "string",
      "version": "string",
      "environment": "string",
      "state": "string",
      "health": "string",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 10,
  "limit": 50,
  "offset": 0
}
```

---

### Validate Deployment
Validate a deployment specification.

**Endpoint:** `POST /api/v1/deployments/validate`

**Request Body:**
```json
{
  "name": "string",
  "version": "string",
  "artifact_path": "string",
  "strategy": "string",
  "dependencies": ["string"]
}
```

**Response:** `200 OK`
```json
{
  "valid": true,
  "issues": [],
  "warnings": []
}
```

**Response:** `400 Bad Request`
```json
{
  "valid": false,
  "issues": [
    {
      "field": "string",
      "message": "string",
      "severity": "error|warning"
    }
  ],
  "warnings": [
    {
      "field": "string",
      "message": "string"
    }
  ]
}
```

---

### Health Check
Perform health check on a deployment.

**Endpoint:** `GET /api/v1/deployments/{name}/health`

**Path Parameters:**
- `name` (string, required) - Deployment name

**Response:** `200 OK`
```json
{
  "overall": "healthy|unhealthy|degraded|unknown",
  "components": {
    "service": {
      "name": "string",
      "status": "healthy|unhealthy|unknown",
      "message": "string",
      "checks": [
        {
          "name": "string",
          "status": "healthy|unhealthy|unknown",
          "message": "string",
          "duration": "100ms",
          "last_run": "2024-01-01T00:00:00Z"
        }
      ]
    }
  },
  "last_check": "2024-01-01T00:00:00Z",
  "message": "string"
}
```

**Error Responses:**
- `404 Not Found` - Deployment not found

---

## WebSocket Events

### Progress Updates
Real-time deployment progress updates.

**Endpoint:** `ws://host/api/v1/deployments/{name}/progress`

**Message Format:**
```json
{
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
      "deployment": "string",
      "operation": "string",
      "cause": "string"
    },
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

## Common Error Codes

- `DEPLOYMENT_NOT_FOUND` - Deployment does not exist
- `DEPLOYMENT_IN_PROGRESS` - Another deployment is already running
- `INVALID_DEPLOYMENT_SPEC` - Deployment specification is invalid
- `ARTIFACT_NOT_FOUND` - Deployment artifact not found
- `DEPENDENCY_FAILED` - Required dependency check failed
- `HEALTH_CHECK_FAILED` - Health check failed after deployment
- `ROLLBACK_FAILED` - Rollback operation failed
- `BACKUP_NOT_FOUND` - Backup for rollback not available

## Rate Limiting

- Deploy: 5 requests per minute per deployment
- Status/Health: 100 requests per minute
- List: 50 requests per minute