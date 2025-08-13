# SSH Package Documentation

This package provides comprehensive SSH connection management, server setup, security hardening, and troubleshooting capabilities for the PocketBase deployer.

## Package Overview

The SSH package is designed to handle all SSH-related operations for server management, including:
- SSH connection pooling and health monitoring
- Server initial setup and configuration  
- Security hardening and lockdown procedures
- Post-security deployment operations
- Comprehensive troubleshooting and diagnostics

## Architecture Components

### Core Files

#### `manager.go` - Core SSH Manager
- **Purpose**: Foundation SSH client with connection management
- **Key Features**:
  - SSH connection establishment with retry logic
  - Command execution (regular and streaming)
  - Authentication via SSH agent or private keys
  - Host key verification and storage
  - Connection validation and reconnection
- **Usage**: Direct SSH operations and underlying component for other managers

#### `connection_pool.go` - Connection Pooling
- **Purpose**: Manages reusable SSH connections with health tracking
- **Key Features**:
  - Connection pool with automatic cleanup
  - Health monitoring and connection recovery
  - Thread-safe connection management
  - Singleton pattern (`GetConnectionPool()`, `GetConnectionManager()`)
- **Usage**: Efficient connection reuse across operations

#### `health_monitor.go` - Health Monitoring
- **Purpose**: Continuous health monitoring of SSH connections
- **Key Features**:
  - Periodic health checks (30-second intervals)
  - Connection status tracking (healthy/degraded/unhealthy/recovering/failed)
  - Metrics collection and reporting
  - Automatic connection recovery
- **Usage**: Background monitoring and connection reliability

#### `service.go` - High-Level Service Interface
- **Purpose**: Unified API orchestrating all SSH operations
- **Key Features**:
  - Single entry point for SSH operations
  - Integration with connection pool and health monitor
  - Context-aware operations
  - Server setup and security lockdown coordination
- **Usage**: Primary interface for external consumers

### Specialized Components

#### `post_security.go` - Post-Security Operations
- **Purpose**: Handles operations after security lockdown (root SSH disabled)
- **Key Features**:
  - Intelligent manager selection (root vs app user)
  - Sudo-based privileged command execution
  - Deployment capability validation
  - Security mode transitions
- **Usage**: Operations on security-hardened servers

#### `security_operations.go` - Security Hardening
- **Purpose**: Implements server security lockdown procedures
- **Key Features**:
  - UFW firewall configuration
  - fail2ban intrusion prevention setup
  - SSH hardening (disable root login, password auth, etc.)
  - Progress reporting for security steps
- **Usage**: Initial server security hardening

#### `setup_operations.go` - Server Setup
- **Purpose**: Initial server configuration and user setup
- **Key Features**:
  - App user creation and configuration
  - SSH key setup and permissions
  - Directory structure creation
  - Sudo access configuration
- **Usage**: New server initialization

#### `service_management.go` - Systemd Operations
- **Purpose**: Systemd service management utilities
- **Key Features**:
  - Service start/stop/restart/enable/disable
  - Service status checking
  - Log retrieval
  - Daemon reload operations
- **Usage**: Service lifecycle management

#### `troubleshooting.go` - Diagnostics & Troubleshooting
- **Purpose**: Comprehensive SSH connection diagnostics
- **Key Features**:
  - Multi-step connection testing
  - fail2ban detection and analysis
  - Authentication method testing
  - Host key and permissions validation
  - Automated issue fixing
- **Usage**: Debugging connection problems

## Data Flow & Dependencies

```
External API Calls
       ↓
   service.go (SSHService)
       ↓
connection_pool.go (ConnectionManager/Pool)
       ↓
   manager.go (SSHManager)
       ↓
   SSH Operations

Parallel:
health_monitor.go ← monitors → connection_pool.go
troubleshooting.go ← diagnoses → all components
```

### Key Interactions

1. **Service → Connection Pool**: High-level operations use pooled connections
2. **Pool → Health Monitor**: Health monitoring tracks pooled connections  
3. **Manager → Security/Setup**: Core manager provides base for specialized operations
4. **Post-Security → Manager**: Wraps managers for security-aware operations
5. **Troubleshooting → All**: Diagnoses issues across all components

## Current Issues & Technical Debt

### Architecture Problems
- **Multiple Singletons**: GlobalConnectionPool, GlobalHealthMonitor, GlobalSSHService create tight coupling
- **Circular Dependencies**: Health monitor and connection pool reference each other
- **Overlapping Responsibilities**: Service, ConnectionManager, and Pool have similar functions
- **Complex State Management**: Multiple layers of connection state tracking

### Code Quality Issues
- **Inconsistent Error Handling**: Mix of error types and handling strategies
- **Heavy Logging**: Excessive debug logging may impact performance
- **Resource Management**: Potential connection leaks in error scenarios
- **Testing Complexity**: Singleton patterns make unit testing difficult

### Functional Redundancy
- **Connection Testing**: Multiple ways to test connections across components
- **Service Management**: Duplicated service operations in multiple files
- **Progress Reporting**: Inconsistent progress update mechanisms

## Configuration & Authentication

### Supported Authentication Methods
1. **SSH Agent**: Via `SSH_AUTH_SOCK` environment variable
2. **Manual Key Path**: Specified in server configuration
3. **Default Keys**: `~/.ssh/id_rsa`, `~/.ssh/id_ed25519`, `~/.ssh/id_ecdsa`

### Host Key Management
- **Permissive Verification**: Accepts unknown hosts with logging
- **Automatic Storage**: Stores host keys in `~/.ssh/known_hosts`
- **Security Logging**: Comprehensive host key verification logging

### Connection Lifecycle
1. **Connection Creation**: Authentication and establishment
2. **Health Monitoring**: Periodic checks and status updates
3. **Command Execution**: Regular or streaming output
4. **Connection Recovery**: Automatic reconnection on failures
5. **Cleanup**: Stale connection removal and resource cleanup

## Usage Patterns

### Basic Operations
```go
service := GetSSHService()
result := service.TestConnection(server, false)
output, err := service.ExecuteCommand(server, false, "ls -la")
```

### Server Setup
```go
service := GetSSHService()
progressChan := make(chan SetupStep, 100)
err := service.RunServerSetup(server, progressChan)
```

### Security Lockdown
```go
service := GetSSHService()
progressChan := make(chan SetupStep, 100)
err := service.ApplySecurityLockdown(server, progressChan)
```

### Post-Security Operations
```go
psm, err := service.CreatePostSecurityManager(server)
output, err := psm.ExecutePrivilegedCommand("systemctl restart myapp")
```

## Performance Characteristics

### Connection Pool
- **Health Check Interval**: 30 seconds
- **Stale Cleanup**: 5 minutes (connections unused >15 minutes)
- **Connection Timeout**: 30 seconds
- **Retry Logic**: 3 attempts with exponential backoff

### Resource Usage
- **Memory**: Maintains active connections and health state
- **Network**: Background health checks and connection keep-alives
- **CPU**: Periodic monitoring goroutines

## Refactoring Opportunities

1. **Simplify Architecture**: Reduce singleton dependencies
2. **Consolidate Connection Management**: Single source of truth for connections
3. **Improve Error Handling**: Consistent error types and propagation
4. **Enhance Testing**: Dependency injection for better testability
5. **Reduce Complexity**: Streamline overlapping responsibilities
6. **Optimize Performance**: Reduce redundant health checks and logging

## Next Steps for Improvement

1. **Interface Definition**: Define clear interfaces for SSH operations
2. **Dependency Injection**: Replace singletons with injected dependencies  
3. **Connection Consolidation**: Single connection manager with clear responsibilities
4. **Error Strategy**: Consistent error handling and classification
5. **Resource Optimization**: Improve connection lifecycle management
6. **Testing Framework**: Enable comprehensive unit and integration testing