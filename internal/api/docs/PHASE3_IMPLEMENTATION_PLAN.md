# Phase 3 API Implementation Plan
**PB-Deployer API Handlers & OpenAPI Specification**

## Overview
This document outlines the implementation plan for Phase 3 of the PB-Deployer project, focusing on completing the API handlers and OpenAPI specification as documented in `HANDLERS.md` and `PHASE3.md`.

## Current State Analysis

### ✅ Already Implemented
- Basic handler structure in `/internal/handlers/`
- PocketBase integration framework
- Handler registration system
- Basic server connection handlers
- SSH connection management
- Database models structure

### 🔄 Partially Implemented
- Server handlers (connection testing exists, setup/security incomplete)
- App handlers (basic structure exists)
- Version handlers (basic structure exists)
- Deployment handlers (basic structure exists)

### ❌ Not Implemented
- Complete OpenAPI specification endpoints
- WebSocket real-time updates
- File upload/download handlers
- Background task processing
- Comprehensive error handling
- Service management handlers
- Security audit endpoints
- Deployment progress tracking

## Implementation Phases

### Phase 3.1: Core API Handler Completion
**Duration: 1-2 weeks**

#### 3.1.1 Server Management Handlers
**Priority: High**

```bash
Files to create/modify:
- internal/handlers/server/crud.go          # CRUD operations
- internal/handlers/server/setup.go         # Complete setup implementation
- internal/handlers/server/security.go      # Security lockdown & audit
- internal/handlers/server/service.go       # Service management
- internal/handlers/server/validation.go    # Request validation
```

**Tasks:**
1. **Complete Server CRUD Operations**
   - [ ] GET `/api/v1/servers` - List servers with pagination
   - [ ] POST `/api/v1/servers` - Create server
   - [ ] GET `/api/v1/servers/{id}` - Get server details
   - [ ] PATCH `/api/v1/servers/{id}` - Update server
   - [ ] DELETE `/api/v1/servers/{id}` - Delete server

2. **Server Setup Enhancement**
   - [ ] POST `/api/v1/servers/{id}/setup` - Complete setup flow
   - [ ] POST `/api/v1/servers/{id}/setup/user` - User creation
   - [ ] POST `/api/v1/servers/{id}/setup/packages` - Package installation
   - [ ] Implement background task processing
   - [ ] Add WebSocket progress notifications

3. **Security Management**
   - [ ] POST `/api/v1/servers/{id}/security/lockdown` - Security lockdown
   - [ ] GET `/api/v1/servers/{id}/security/audit` - Security audit
   - [ ] Implement firewall configuration
   - [ ] Implement SSH hardening
   - [ ] Implement fail2ban configuration

4. **Service Management**
   - [ ] GET `/api/v1/services/{server_id}` - List services
   - [ ] GET `/api/v1/services/{server_id}/{service_name}` - Get service status
   - [ ] POST `/api/v1/services/{server_id}/{service_name}` - Control service
   - [ ] GET `/api/v1/services/{server_id}/{service_name}/logs` - Get service logs

#### 3.1.2 Application Management Handlers
**Priority: High**

```bash
Files to create/modify:
- internal/handlers/apps/crud.go            # App CRUD operations
- internal/handlers/apps/deployment.go      # Deployment handlers
- internal/handlers/apps/service.go         # App service management
- internal/handlers/apps/validation.go      # Request validation
```

**Tasks:**
1. **App CRUD Operations**
   - [ ] GET `/api/v1/apps` - List apps with filtering
   - [ ] POST `/api/v1/apps` - Create app
   - [ ] GET `/api/v1/apps/{id}` - Get app details
   - [ ] PATCH `/api/v1/apps/{id}` - Update app
   - [ ] DELETE `/api/v1/apps/{id}` - Delete app

2. **Deployment Operations**
   - [ ] POST `/api/v1/apps/{id}/deploy` - Deploy app
   - [ ] POST `/api/v1/apps/{id}/rollback` - Rollback app
   - [ ] Implement deployment strategies (rolling, blue-green)
   - [ ] Add health checks during deployment
   - [ ] Implement pre/post deployment hooks

#### 3.1.3 Version Management Handlers
**Priority: Medium**

```bash
Files to create/modify:
- internal/handlers/version/crud.go         # Version CRUD
- internal/handlers/version/upload.go       # File upload/download
- internal/handlers/version/validation.go   # File validation
```

**Tasks:**
1. **Version Management**
   - [ ] GET `/api/v1/versions` - List versions
   - [ ] POST `/api/v1/versions` - Create version
   - [ ] GET `/api/v1/versions/{id}` - Get version details
   - [ ] DELETE `/api/v1/versions/{id}` - Delete version

2. **File Management**
   - [ ] POST `/api/v1/versions/{id}/upload` - Upload deployment zip
   - [ ] GET `/api/v1/versions/{id}/download` - Download deployment zip
   - [ ] Implement file validation
   - [ ] Implement file size limits
   - [ ] Add file integrity checks

#### 3.1.4 Deployment Process Handlers
**Priority: High**

```bash
Files to create/modify:
- internal/handlers/deployment/process.go   # Deployment processing
- internal/handlers/deployment/status.go    # Status tracking
- internal/handlers/deployment/cancel.go    # Cancellation
- internal/handlers/deployment/stats.go     # Statistics
```

**Tasks:**
1. **Deployment Process Management**
   - [ ] GET `/api/v1/deployments` - List deployments
   - [ ] GET `/api/v1/deployments/{id}/status` - Get deployment status
   - [ ] POST `/api/v1/deployments/{id}/cancel` - Cancel deployment
   - [ ] GET `/api/v1/deployments/stats` - Get deployment statistics

2. **Background Processing**
   - [ ] Implement deployment queue system
   - [ ] Add deployment progress tracking
   - [ ] Implement deployment cancellation
   - [ ] Add deployment failure recovery

### Phase 3.2: WebSocket & Real-time Features
**Duration: 1 week**

#### 3.2.1 WebSocket Integration
**Priority: Medium**

```bash
Files to create:
- internal/websocket/manager.go             # WebSocket connection manager
- internal/websocket/events.go              # Event definitions
- internal/websocket/handlers.go            # WebSocket handlers
- internal/websocket/broadcaster.go         # Event broadcasting
```

**Tasks:**
1. **WebSocket Infrastructure**
   - [ ] Implement WebSocket connection manager
   - [ ] Add client connection tracking
   - [ ] Implement event broadcasting system
   - [ ] Add connection authentication (optional)

2. **Real-time Events**
   - [ ] Server status updates
   - [ ] Deployment progress updates
   - [ ] Service status changes
   - [ ] Log streaming
   - [ ] Setup progress notifications

#### 3.2.2 Background Task System
**Priority: High**

```bash
Files to create:
- internal/tasks/manager.go                 # Task manager
- internal/tasks/types.go                   # Task definitions
- internal/tasks/worker.go                  # Task worker
- internal/tasks/queue.go                   # Task queue
```

**Tasks:**
1. **Task Management System**
   - [ ] Implement task queue
   - [ ] Add task workers
   - [ ] Implement task status tracking
   - [ ] Add task cancellation
   - [ ] Implement task retry logic

2. **Task Types**
   - [ ] Server setup tasks
   - [ ] Security lockdown tasks
   - [ ] Deployment tasks
   - [ ] Package installation tasks

### Phase 3.3: Validation & Error Handling
**Duration: 3-5 days**

#### 3.3.1 Request Validation
**Priority: High**

```bash
Files to create:
- internal/validation/server.go             # Server validation
- internal/validation/app.go                # App validation
- internal/validation/deployment.go         # Deployment validation
- internal/validation/common.go             # Common validators
```

**Tasks:**
1. **Validation Functions**
   - [ ] Server creation/update validation
   - [ ] App creation/update validation
   - [ ] Deployment request validation
   - [ ] File upload validation
   - [ ] Domain name validation
   - [ ] Version number validation

#### 3.3.2 Error Handling
**Priority: Medium**

```bash
Files to create:
- internal/errors/types.go                  # Error types
- internal/errors/handlers.go               # Error handlers
- internal/errors/responses.go              # Error responses
```

**Tasks:**
1. **Consistent Error Handling**
   - [ ] Standardize error response format
   - [ ] Implement error logging
   - [ ] Add error code definitions
   - [ ] Create error recovery strategies

### Phase 3.4: Health & Monitoring
**Duration: 2-3 days**

#### 3.4.1 Health Endpoints
**Priority: Low**

```bash
Files to create:
- internal/handlers/health/system.go        # System health
- internal/handlers/health/components.go    # Component health
```

**Tasks:**
1. **Health Monitoring**
   - [ ] GET `/api/v1/health` - System health check
   - [ ] Database connectivity check
   - [ ] SSH connection pool check
   - [ ] Storage space monitoring
   - [ ] Service dependency checks

## Implementation Details

### Database Schema Requirements
Based on the OpenAPI spec, ensure these PocketBase collections exist:

```javascript
// servers collection
{
  id: "string",
  created: "datetime",
  updated: "datetime", 
  name: "string",
  host: "string",
  port: "number",
  root_username: "string",
  app_username: "string",
  use_ssh_agent: "boolean",
  manual_key_path: "string",
  setup_complete: "boolean",
  security_locked: "boolean"
}

// apps collection  
{
  id: "string",
  created: "datetime",
  updated: "datetime",
  name: "string", 
  server_id: "relation(servers)",
  remote_path: "string",
  service_name: "string",
  domain: "string",
  current_version: "string",
  status: "select(idle,deploying,running,stopped,error)"
}

// versions collection
{
  id: "string",
  created: "datetime", 
  updated: "datetime",
  app_id: "relation(apps)",
  version_number: "string",
  deployment_zip: "file",
  notes: "text"
}

// deployments collection
{
  id: "string",
  created: "datetime",
  updated: "datetime", 
  app_id: "relation(apps)",
  version_id: "relation(versions)",
  status: "select(pending,running,completed,failed,cancelled)",
  logs: "json",
  started_at: "datetime",
  completed_at: "datetime"
}
```

### Configuration Files

#### 3.4.2 OpenAPI Specification
**Priority: Low**

```bash
Files to create:
- api/openapi.yaml                          # Complete OpenAPI spec
- internal/docs/generator.go                # Documentation generator
```

**Tasks:**
1. **API Documentation**
   - [ ] Generate complete OpenAPI 3.0.3 spec
   - [ ] Add example requests/responses
   - [ ] Document all error codes
   - [ ] Add WebSocket event documentation

### Testing Strategy

#### 3.4.3 Testing Implementation
**Priority: Medium**

```bash
Files to create:
- internal/handlers/server/server_test.go
- internal/handlers/apps/apps_test.go  
- internal/handlers/deployment/deployment_test.go
- internal/websocket/websocket_test.go
- test/integration/api_test.go
```

**Tasks:**
1. **Unit Tests**
   - [ ] Handler unit tests
   - [ ] Validation function tests
   - [ ] Error handling tests
   - [ ] WebSocket event tests

2. **Integration Tests**
   - [ ] End-to-end deployment flow
   - [ ] Server setup flow
   - [ ] Security lockdown flow
   - [ ] WebSocket integration tests

## Implementation Timeline

### Week 1: Core Handlers
- Days 1-2: Server CRUD and setup handlers
- Days 3-4: App CRUD and deployment handlers  
- Days 5-7: Version management and file handlers

### Week 2: Advanced Features  
- Days 1-2: Background task system
- Days 3-4: WebSocket integration
- Days 5-7: Error handling and validation

### Week 3: Polish & Testing
- Days 1-2: Health monitoring
- Days 3-4: Testing implementation
- Days 5-7: Documentation and cleanup

## Dependencies & Prerequisites

### Required Components
- [ ] PocketBase application setup
- [ ] Database collections created
- [ ] SSH key management system
- [ ] File storage configuration
- [ ] WebSocket support enabled

### External Dependencies
- [ ] Go 1.21+ with modules
- [ ] PocketBase framework
- [ ] WebSocket library (gorilla/websocket)
- [ ] File validation libraries
- [ ] SSH client libraries

## Success Criteria

### Functional Requirements
- [ ] All API endpoints respond correctly
- [ ] WebSocket events work in real-time
- [ ] File upload/download functions properly
- [ ] Background tasks process correctly
- [ ] Error handling is consistent
- [ ] Documentation is complete

### Performance Requirements  
- [ ] API responses under 500ms
- [ ] WebSocket latency under 100ms
- [ ] File uploads handle 100MB+ files
- [ ] Background tasks don't block API
- [ ] Database queries optimized

### Quality Requirements
- [ ] 80%+ test coverage
- [ ] No critical security vulnerabilities
- [ ] Proper error logging
- [ ] Graceful failure handling
- [ ] Clean, maintainable code

## Risk Mitigation

### Technical Risks
1. **WebSocket Complexity**: Start with simple events, expand gradually
2. **File Upload Limits**: Implement streaming and chunked uploads  
3. **Background Task Failures**: Add retry logic and failure recovery
4. **SSH Connection Issues**: Implement connection pooling and timeouts

### Timeline Risks
1. **Scope Creep**: Stick to documented API specification
2. **Integration Issues**: Test early and often
3. **Performance Problems**: Profile and optimize incrementally

## Next Steps

1. **Immediate (Day 1)**
   - Review existing handler implementations
   - Create missing database collections  
   - Set up development environment

2. **Week 1 Priority**
   - Implement server CRUD handlers
   - Complete setup process handlers
   - Add basic validation

3. **Ongoing**
   - Write tests as handlers are implemented  
   - Document API changes
   - Monitor performance metrics

---

**Status**: Ready for Implementation  
**Last Updated**: 2024-01-XX  
**Next Review**: After Phase 3.1 completion