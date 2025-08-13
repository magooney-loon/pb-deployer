# Tunnel Package - Phase 3 Implementation Plan

## Overview
Phase 3 focuses on creating a modern, well-documented REST API layer with comprehensive Swagger/OpenAPI documentation. This phase will establish a new API package that provides a clean, versioned interface while maintaining backward compatibility with existing PocketBase handlers.

## Phase 1 & 2 Status ✅ COMPLETED
- ✅ **Phase 1**: Core SSH infrastructure, connection pooling, health monitoring
- ✅ **Phase 2**: Operational components (Security, Service, Deployment managers)
- ✅ **Current Handlers**: Basic REST endpoints using PocketBase router

## Phase 3 Goals

### Primary Objectives
1. **Modern API Layer**: Create a new `/internal/api` package with proper structure
2. **Swagger Documentation**: Comprehensive OpenAPI 3.0 specification
3. **Interactive UI**: Self-served Swagger UI for API exploration
4. **Type Safety**: Strongly typed request/response models
5. **API Versioning**: Future-proof versioning strategy
6. **Developer Experience**: Clear documentation and examples

### Secondary Objectives
1. **Validation**: Request/response validation with proper error handling
2. **Middleware**: Authentication, logging, rate limiting, CORS
3. **Testing**: API contract testing and documentation examples
4. **Monitoring**: API metrics and health endpoints
5. **Client Generation**: Support for auto-generated client libraries

## Architecture Design

### Package Structure
```
internal/api/
├── v1/                          # API version 1
│   ├── handlers/               # HTTP handlers
│   │   ├── server.go          # Server management endpoints
│   │   ├── app.go             # Application endpoints  
│   │   ├── version.go         # Version management endpoints
│   │   ├── deployment.go      # Deployment endpoints
│   │   └── health.go          # Health and monitoring endpoints
│   ├── models/                # Request/response models
│   │   ├── server.go          # Server-related types
│   │   ├── app.go             # App-related types
│   │   ├── version.go         # Version-related types
│   │   ├── deployment.go      # Deployment-related types
│   │   ├── common.go          # Common types (pagination, errors)
│   │   └── validation.go      # Validation helpers
│   ├── middleware/            # HTTP middleware
│   │   ├── auth.go           # Authentication middleware
│   │   ├── logging.go        # Request logging
│   │   ├── cors.go           # CORS handling
│   │   ├── ratelimit.go      # Rate limiting
│   │   └── recovery.go       # Panic recovery
│   └── router.go             # Route definitions
├── docs/                      # Generated documentation
│   ├── swagger.yaml          # OpenAPI specification
│   ├── swagger.json          # JSON format spec
│   └── examples/             # API usage examples
├── swagger/                   # Swagger UI assets
│   ├── index.html            # Custom Swagger UI
│   ├── swagger-ui-bundle.js  # Swagger UI JS
│   ├── swagger-ui-standalone-preset.js
│   └── swagger-ui.css        # Swagger UI CSS
├── client/                    # Generated client code
│   ├── go/                   # Go client
│   ├── typescript/           # TypeScript client
│   └── python/               # Python client
├── generator.go               # Code/docs generation
├── server.go                 # API server setup
└── README.md                 # API documentation
```

### API Design Principles

#### RESTful Design
- **Resource-based URLs**: `/api/v1/servers/{id}`, `/api/v1/apps/{id}/deployments`
- **HTTP Methods**: GET, POST, PUT, DELETE, PATCH for appropriate operations
- **Status Codes**: Proper HTTP status codes with consistent error responses
- **Content Negotiation**: JSON primary, support for other formats

#### Versioning Strategy
- **URL Versioning**: `/api/v1/`, `/api/v2/` for major version changes
- **Header Versioning**: `Accept: application/vnd.pb-deployer.v1+json` for minor versions
- **Backward Compatibility**: Maintain support for previous versions during transition

#### Error Handling
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "details": [
      {
        "field": "server.host",
        "message": "Host is required",
        "code": "REQUIRED"
      }
    ],
    "request_id": "req_123456789",
    "timestamp": "2023-12-07T10:30:00Z"
  }
}
```

#### Pagination Pattern
```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 150,
    "total_pages": 8,
    "has_next": true,
    "has_prev": false
  },
  "links": {
    "first": "/api/v1/deployments?page=1",
    "last": "/api/v1/deployments?page=8",
    "next": "/api/v1/deployments?page=2",
    "prev": null
  }
}
```

## OpenAPI Specification Structure

### API Information
```yaml
openapi: 3.0.3
info:
  title: PB-Deployer API
  description: |
    REST API for PocketBase application deployment and server management.
    
    ## Features
    - Server setup and security management
    - Application lifecycle management  
    - Version control and deployment
    - Real-time progress monitoring
    - Health monitoring and diagnostics
    
  version: 1.0.0
  contact:
    name: PB-Deployer Team
    url: https://github.com/username/pb-deployer
  license:
    name: MIT
    url: https://opensource.org/licenses/MIT

servers:
  - url: /api/v1
    description: Version 1 API
  - url: https://demo.pb-deployer.com/api/v1
    description: Demo server

tags:
  - name: servers
    description: Server management operations
  - name: apps
    description: Application management
  - name: versions
    description: Version control
  - name: deployments
    description: Deployment operations
  - name: health
    description: Health and monitoring
```

### Security Schemes
```yaml
components:
  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key
      description: API key for authentication
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: JWT token authentication
    PocketBaseAuth:
      type: http
      scheme: bearer
      description: PocketBase session token

security:
  - ApiKeyAuth: []
  - BearerAuth: []
  - PocketBaseAuth: []
```

## Implementation Phases

### Phase 3.1: Foundation (Week 1) 🎯 **PRIORITY**
**Files to Create:**
- `internal/api/server.go` - API server setup
- `internal/api/v1/router.go` - Route definitions
- `internal/api/v1/models/common.go` - Common types
- `internal/api/v1/models/validation.go` - Validation helpers
- `internal/api/v1/middleware/recovery.go` - Basic middleware
- `internal/api/docs/swagger.yaml` - Initial OpenAPI spec

**Objectives:**
- ✅ Basic API server structure
- ✅ Route registration system
- ✅ Common response types
- ✅ Error handling patterns
- ✅ Basic OpenAPI specification
- ✅ Health endpoint implementation

**Deliverables:**
```go
// Basic API server that can:
GET  /api/v1/health           // API health check
GET  /api/v1/docs             // Swagger UI
GET  /api/v1/swagger.json     // OpenAPI spec
```

### Phase 3.2: Core Models (Week 1-2)
**Files to Create:**
- `internal/api/v1/models/server.go` - Server types
- `internal/api/v1/models/app.go` - Application types
- `internal/api/v1/models/version.go` - Version types
- `internal/api/v1/models/deployment.go` - Deployment types
- `internal/api/generator.go` - Documentation generator

**Objectives:**
- ✅ Complete type definitions
- ✅ JSON serialization tags
- ✅ Validation rules
- ✅ OpenAPI schema annotations
- ✅ Documentation examples

**Type Examples:**
```go
// Server management types
type ServerCreateRequest struct {
    Name          string `json:"name" validate:"required,min=1,max=50" example:"production-server"`
    Host          string `json:"host" validate:"required,hostname" example:"192.168.1.100"`
    Port          int    `json:"port" validate:"min=1,max=65535" example:"22"`
    Username      string `json:"username" validate:"required" example:"root"`
    UseSSHAgent   bool   `json:"use_ssh_agent" example:"true"`
    ManualKeyPath string `json:"manual_key_path,omitempty" example:"/home/user/.ssh/id_rsa"`
}

type ServerResponse struct {
    ID             string    `json:"id" example:"srv_123456789"`
    Name           string    `json:"name" example:"production-server"`
    Host           string    `json:"host" example:"192.168.1.100"`
    Port           int       `json:"port" example:"22"`
    Status         string    `json:"status" example:"online"`
    SetupComplete  bool      `json:"setup_complete" example:"true"`
    SecurityLocked bool      `json:"security_locked" example:"false"`
    CreatedAt      time.Time `json:"created_at" example:"2023-12-07T10:30:00Z"`
    UpdatedAt      time.Time `json:"updated_at" example:"2023-12-07T10:30:00Z"`
}
```

### Phase 3.3: Server Endpoints (Week 2)
**Files to Create:**
- `internal/api/v1/handlers/server.go` - Server handlers
- `internal/api/v1/middleware/auth.go` - Authentication
- `internal/api/v1/middleware/logging.go` - Request logging

**Endpoints:**
```yaml
# Server Management
GET    /api/v1/servers                    # List servers
POST   /api/v1/servers                    # Create server
GET    /api/v1/servers/{id}               # Get server
PUT    /api/v1/servers/{id}               # Update server
DELETE /api/v1/servers/{id}               # Delete server

# Server Operations  
POST   /api/v1/servers/{id}/test          # Test connection
GET    /api/v1/servers/{id}/status        # Get status
GET    /api/v1/servers/{id}/health        # Health check
POST   /api/v1/servers/{id}/setup         # Run setup
POST   /api/v1/servers/{id}/security      # Apply security
POST   /api/v1/servers/{id}/troubleshoot  # Troubleshoot

# Real-time
GET    /api/v1/servers/{id}/setup/ws      # Setup progress WebSocket
GET    /api/v1/servers/{id}/security/ws   # Security progress WebSocket
```

### Phase 3.4: Application Endpoints (Week 2-3)
**Files to Create:**
- `internal/api/v1/handlers/app.go` - Application handlers
- `internal/api/v1/middleware/cors.go` - CORS handling

**Endpoints:**
```yaml
# Application Management
GET    /api/v1/apps                       # List apps
POST   /api/v1/apps                       # Create app
GET    /api/v1/apps/{id}                  # Get app
PUT    /api/v1/apps/{id}                  # Update app
DELETE /api/v1/apps/{id}                  # Delete app

# Application Operations
GET    /api/v1/apps/{id}/status           # Get status
POST   /api/v1/apps/{id}/health-check     # Health check
GET    /api/v1/apps/{id}/logs             # Get logs
POST   /api/v1/apps/{id}/start            # Start service
POST   /api/v1/apps/{id}/stop             # Stop service
POST   /api/v1/apps/{id}/restart          # Restart service

# Deployment Operations
POST   /api/v1/apps/{id}/deploy           # Deploy version
POST   /api/v1/apps/{id}/rollback         # Rollback version
GET    /api/v1/apps/{id}/deploy/ws        # Deployment progress WebSocket
```

### Phase 3.5: Version & Deployment Endpoints (Week 3)
**Files to Create:**
- `internal/api/v1/handlers/version.go` - Version handlers
- `internal/api/v1/handlers/deployment.go` - Deployment handlers
- `internal/api/v1/middleware/ratelimit.go` - Rate limiting

**Endpoints:**
```yaml
# Version Management
GET    /api/v1/versions                   # List versions
POST   /api/v1/versions                   # Create version
GET    /api/v1/versions/{id}              # Get version
PUT    /api/v1/versions/{id}              # Update version
DELETE /api/v1/versions/{id}              # Delete version

# Version Files
POST   /api/v1/versions/{id}/upload       # Upload files
GET    /api/v1/versions/{id}/download     # Download ZIP
POST   /api/v1/versions/{id}/validate     # Validate package

# App Versions
GET    /api/v1/apps/{id}/versions         # List app versions
POST   /api/v1/apps/{id}/versions         # Create app version

# Deployment Management  
GET    /api/v1/deployments                # List deployments
GET    /api/v1/deployments/{id}           # Get deployment
GET    /api/v1/deployments/{id}/status    # Get status
GET    /api/v1/deployments/{id}/logs      # Get logs
POST   /api/v1/deployments/{id}/cancel    # Cancel deployment
POST   /api/v1/deployments/{id}/retry     # Retry deployment
GET    /api/v1/deployments/{id}/ws        # Progress WebSocket

# Analytics
GET    /api/v1/deployments/stats          # Deployment statistics
POST   /api/v1/deployments/cleanup        # Cleanup old deployments
```

### Phase 3.6: Swagger UI & Documentation (Week 3-4)
**Files to Create:**
- `internal/api/swagger/index.html` - Custom Swagger UI
- `internal/api/swagger/swagger-ui-bundle.js` - Downloaded Swagger UI JS
- `internal/api/swagger/swagger-ui.css` - Downloaded Swagger UI CSS
- `internal/api/docs/examples/` - API usage examples
- `internal/api/v1/handlers/docs.go` - Documentation handlers

**Features:**
- ✅ Self-served Swagger UI
- ✅ Interactive API exploration
- ✅ Request/response examples
- ✅ Code generation support
- ✅ Download OpenAPI spec

**UI Customization:**
```html
<!DOCTYPE html>
<html>
<head>
    <title>PB-Deployer API Documentation</title>
    <link rel="stylesheet" type="text/css" href="./swagger-ui.css" />
    <style>
        .topbar { display: none; }
        .swagger-ui .info .title { color: #3b82f6; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="./swagger-ui-bundle.js"></script>
    <script>
        SwaggerUIBundle({
            url: '/api/v1/swagger.json',
            dom_id: '#swagger-ui',
            presets: [SwaggerUIBundle.presets.apis],
            layout: "BaseLayout",
            deepLinking: true,
            showExtensions: true,
            showCommonExtensions: true
        });
    </script>
</body>
</html>
```

### Phase 3.7: Advanced Features (Week 4)
**Files to Create:**
- `internal/api/client/generator.go` - Client code generation
- `internal/api/v1/handlers/health.go` - Advanced health endpoints
- `internal/api/v1/middleware/metrics.go` - Metrics collection
- `internal/api/testing/` - API testing utilities

**Advanced Features:**
- ✅ Client library generation (Go, TypeScript, Python)
- ✅ API metrics and monitoring
- ✅ Contract testing framework
- ✅ Performance monitoring
- ✅ API versioning support

## Integration Strategy

### PocketBase Integration
```go
// Integrate with existing PocketBase app
func RegisterAPIRoutes(app core.App) {
    // Mount new API alongside existing handlers
    app.OnServe().BindFunc(func(e *core.ServeEvent) error {
        // Register new API routes
        apiServer := api.NewServer(app)
        e.Router.Mount("/api/v1", apiServer.Handler())
        
        // Keep existing handlers for backward compatibility  
        handlers.RegisterHandlers(app) // Existing handlers
        
        return e.Next()
    })
}
```

### Backward Compatibility
- ✅ Keep existing handlers unchanged during transition
- ✅ Gradual migration path for clients
- ✅ Feature parity between old and new APIs
- ✅ Deprecation notices in old API responses

### Database Integration
```go
// Reuse existing PocketBase models and database
type ServerHandler struct {
    app    core.App
    logger slog.Logger
}

func (h *ServerHandler) ListServers(w http.ResponseWriter, r *http.Request) {
    // Use existing PocketBase database queries
    records, err := h.app.FindRecordsByFilter("servers", "", "-created", 50, 0, nil)
    if err != nil {
        h.writeError(w, err)
        return
    }
    
    // Convert to API response format
    servers := make([]models.ServerResponse, len(records))
    for i, record := range records {
        servers[i] = convertToServerResponse(record)
    }
    
    h.writeJSON(w, map[string]any{
        "data": servers,
        "pagination": calculatePagination(len(records), 1, 50),
    })
}
```

## Documentation Strategy

### API Documentation Structure
```
docs/
├── api/
│   ├── getting-started.md        # Quick start guide
│   ├── authentication.md         # Auth methods
│   ├── servers.md                # Server management guide
│   ├── apps.md                   # Application management
│   ├── deployments.md            # Deployment workflows
│   ├── troubleshooting.md        # Common issues
│   └── changelog.md              # API changes
├── examples/
│   ├── curl/                     # cURL examples
│   ├── javascript/               # JS/TS examples
│   ├── python/                   # Python examples
│   └── go/                       # Go examples
└── postman/
    └── pb-deployer.postman_collection.json
```

### Interactive Examples
```yaml
# OpenAPI example annotations
paths:
  /servers:
    post:
      summary: Create a new server
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ServerCreateRequest'
            examples:
              production:
                summary: Production server
                value:
                  name: "production-server"
                  host: "192.168.1.100"
                  port: 22
                  username: "root"
                  use_ssh_agent: true
              development:
                summary: Development server  
                value:
                  name: "dev-server"
                  host: "localhost"
                  port: 2222
                  username: "deploy"
                  manual_key_path: "/home/user/.ssh/dev_key"
```

## Testing Strategy

### API Contract Testing
```go
// Test API contract compliance
func TestServerAPI(t *testing.T) {
    suite := apitest.NewSuite(t)
    
    suite.TestCase("Create Server").
        Post("/api/v1/servers").
        JSON(validServerRequest).
        Expect(t).
        Status(http.StatusCreated).
        JSONSchema(serverResponseSchema).
        End()
        
    suite.TestCase("List Servers").
        Get("/api/v1/servers").
        Expect(t).
        Status(http.StatusOK).
        JSONPath("$.data").Array().Length().GreaterThan(0).
        JSONPath("$.pagination.total").Number().GreaterThan(0).
        End()
}
```

### Documentation Testing
```go
// Ensure examples in docs work
func TestDocumentationExamples(t *testing.T) {
    examples := loadOpenAPIExamples("docs/swagger.yaml")
    
    for path, methods := range examples {
        for method, example := range methods {
            t.Run(fmt.Sprintf("%s %s", method, path), func(t *testing.T) {
                // Test that example request/response works
                testAPIExample(t, method, path, example)
            })
        }
    }
}
```

## Monitoring and Analytics

### API Metrics
```go
// Metrics middleware
func MetricsMiddleware() middleware.Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            // Track request
            requestCounter.WithLabelValues(
                r.Method,
                getRoutePattern(r),
            ).Inc()
            
            // Wrap response writer to capture status
            wrapped := &responseWriter{ResponseWriter: w}
            next.ServeHTTP(wrapped, r)
            
            // Track duration and status
            duration := time.Since(start).Seconds()
            requestDuration.WithLabelValues(
                r.Method,
                getRoutePattern(r),
                strconv.Itoa(wrapped.statusCode),
            ).Observe(duration)
        })
    }
}
```

### Health Endpoints
```yaml
# Health monitoring endpoints
GET /api/v1/health           # API health
GET /api/v1/health/ready     # Readiness check  
GET /api/v1/health/live      # Liveness check
GET /api/v1/metrics          # Prometheus metrics
GET /api/v1/debug/pprof      # Performance profiling (dev only)
```

## Migration Plan

### Phase 3.1-3.2: Foundation (Weeks 1-2)
- ✅ Create new API package structure
- ✅ Implement basic server and routing
- ✅ Define core models and validation
- ✅ Basic OpenAPI specification
- ✅ Health endpoints

### Phase 3.3-3.5: Core Endpoints (Weeks 2-3)  
- ✅ Server management endpoints
- ✅ Application management endpoints
- ✅ Version and deployment endpoints
- ✅ WebSocket support for real-time updates
- ✅ Complete middleware stack

### Phase 3.6-3.7: Documentation & Polish (Weeks 3-4)
- ✅ Swagger UI integration
- ✅ Complete API documentation
- ✅ Client library generation
- ✅ Advanced monitoring and metrics
- ✅ Testing framework

### Post-Phase 3: Migration & Enhancement
- ✅ Gradual client migration to new API
- ✅ Performance optimization
- ✅ Advanced authentication options
- ✅ API rate limiting and throttling
- ✅ Deprecation of old handlers

## Success Metrics

### Functional Completeness
- ✅ **100% Feature Parity** - All existing handler functionality in new API
- ✅ **Interactive Documentation** - Complete Swagger UI with examples
- ✅ **Type Safety** - Strongly typed request/response models
- ✅ **Client Generation** - Auto-generated client libraries

### Developer Experience
- ✅ **API Discovery** - Self-documenting API with examples
- ✅ **Quick Start** - 5-minute setup guide
- ✅ **Testing Support** - Contract testing and mock servers
- ✅ **Error Clarity** - Clear error messages with solutions

### Performance & Reliability
- ✅ **Response Time** - API responses < 200ms (95th percentile)
- ✅ **Documentation Load** - Swagger UI loads < 2 seconds
- ✅ **Concurrent Users** - Support 100+ concurrent API users
- ✅ **Uptime** - 99.9% API availability

### Code Quality
- ✅ **Test Coverage** - 90%+ test coverage for API handlers
- ✅ **Documentation Coverage** - All endpoints documented with examples
- ✅ **Schema Validation** - All requests/responses validated
- ✅ **OpenAPI Compliance** - Valid OpenAPI 3.0 specification

## Conclusion

Phase 3 establishes a modern, well-documented API foundation that will serve as the primary interface for PB-Deployer. The implementation provides:

**Immediate Benefits:**
- ✅ **Professional API** - Industry-standard REST API with OpenAPI docs
- ✅ **Developer Experience** - Interactive documentation and client libraries
- ✅ **Type Safety** - Strongly typed interfaces prevent runtime errors
- ✅ **Future-Proof** - Versioning strategy supports API evolution

**Long-term Value:**
- ✅ **Ecosystem Growth** - Enables third-party integrations and tools
- ✅ **Client Diversity** - Supports web, mobile, CLI, and service clients
- ✅ **Operational Excellence** - Monitoring, metrics, and observability
- ✅ **Team Productivity** - Clear contracts reduce integration time

**Current State:** ✅ **READY TO START** - Foundation design complete
**Phase 3 Status:** 🎯 **STARTING** - Begin with foundation implementation
**Next Phase:** Advanced features, client libraries, and ecosystem tools

The architecture and implementation plan provide a robust, scalable foundation for the PB-Deployer API that will support the project's growth and adoption.