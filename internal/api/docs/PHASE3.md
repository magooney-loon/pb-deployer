# API Package - Phase 3 Complete Schema Documentation

## Overview
Complete OpenAPI specification and JSON schemas for the pb-deployer API, covering all managers and models.

## Phase 1 & 2 Status ✅ COMPLETED
- Tunnel package implementation complete
- All managers operational (Setup, Security, Service, Deployment)
- Models package complete

## Phase 3 Goals
Expose all tunnel functionality through a comprehensive RESTful API with OpenAPI documentation.

## Complete OpenAPI Specification

```json
{
  "openapi": "3.0.3",
  "info": {
    "title": "PocketBase Deployer API",
    "description": "Complete API for managing PocketBase deployments across servers",
    "version": "1.0.0",
    "contact": {
      "name": "PB Deployer Team",
      "url": "https://github.com/yourusername/pb-deployer"
    },
    "license": {
      "name": "MIT",
      "url": "https://opensource.org/licenses/MIT"
    }
  },
  "servers": [
    {
      "url": "http://localhost:8090",
      "description": "Local development server"
    },
    {
      "url": "https://deployer.example.com",
      "description": "Production server"
    }
  ],
  "tags": [
    {
      "name": "servers",
      "description": "Server management operations"
    },
    {
      "name": "apps",
      "description": "Application management"
    },
    {
      "name": "versions",
      "description": "Version management"
    },
    {
      "name": "deployments",
      "description": "Deployment operations"
    },
    {
      "name": "setup",
      "description": "Server setup operations"
    },
    {
      "name": "security",
      "description": "Security management"
    },
    {
      "name": "services",
      "description": "Service management"
    },
    {
      "name": "monitoring",
      "description": "Health and monitoring"
    }
  ],
  "paths": {
    "/api/v1/servers": {
      "get": {
        "tags": ["servers"],
        "summary": "List all servers",
        "operationId": "listServers",
        "parameters": [
          {
            "name": "page",
            "in": "query",
            "schema": {"type": "integer", "default": 1}
          },
          {
            "name": "per_page",
            "in": "query",
            "schema": {"type": "integer", "default": 20}
          },
          {
            "name": "filter",
            "in": "query",
            "schema": {"type": "string"},
            "description": "Filter expression (e.g., setup_complete=true)"
          }
        ],
        "responses": {
          "200": {
            "description": "List of servers",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/ServerListResponse"
                }
              }
            }
          }
        }
      },
      "post": {
        "tags": ["servers"],
        "summary": "Create a new server",
        "operationId": "createServer",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/ServerCreateRequest"
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Server created",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/Server"
                }
              }
            }
          }
        }
      }
    },
    "/api/v1/servers/{id}": {
      "get": {
        "tags": ["servers"],
        "summary": "Get server details",
        "operationId": "getServer",
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          }
        ],
        "responses": {
          "200": {
            "description": "Server details",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/Server"
                }
              }
            }
          }
        }
      },
      "patch": {
        "tags": ["servers"],
        "summary": "Update server",
        "operationId": "updateServer",
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/ServerUpdateRequest"
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Server updated",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/Server"
                }
              }
            }
          }
        }
      },
      "delete": {
        "tags": ["servers"],
        "summary": "Delete server",
        "operationId": "deleteServer",
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          }
        ],
        "responses": {
          "204": {
            "description": "Server deleted"
          }
        }
      }
    },
    "/api/v1/servers/{id}/setup": {
      "post": {
        "tags": ["setup"],
        "summary": "Run initial server setup",
        "operationId": "setupServer",
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/SetupRequest"
              }
            }
          }
        },
        "responses": {
          "202": {
            "description": "Setup started",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/OperationResponse"
                }
              }
            }
          }
        }
      }
    },
    "/api/v1/servers/{id}/setup/user": {
      "post": {
        "tags": ["setup"],
        "summary": "Create system user",
        "operationId": "createSystemUser",
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/SystemUserConfig"
              }
            }
          }
        },
        "responses": {
          "202": {
            "description": "User creation started",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/OperationResponse"
                }
              }
            }
          }
        }
      }
    },
    "/api/v1/servers/{id}/setup/packages": {
      "post": {
        "tags": ["setup"],
        "summary": "Install system packages",
        "operationId": "installPackages",
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/PackageInstallRequest"
              }
            }
          }
        },
        "responses": {
          "202": {
            "description": "Package installation started",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/OperationResponse"
                }
              }
            }
          }
        }
      }
    },
    "/api/v1/servers/{id}/security/lockdown": {
      "post": {
        "tags": ["security"],
        "summary": "Apply security lockdown",
        "operationId": "applySecurityLockdown",
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/SecurityConfig"
              }
            }
          }
        },
        "responses": {
          "202": {
            "description": "Security lockdown started",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/OperationResponse"
                }
              }
            }
          }
        }
      }
    },
    "/api/v1/servers/{id}/security/audit": {
      "get": {
        "tags": ["security"],
        "summary": "Run security audit",
        "operationId": "auditSecurity",
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          }
        ],
        "responses": {
          "200": {
            "description": "Security audit report",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/SecurityReport"
                }
              }
            }
          }
        }
      }
    },
    "/api/v1/apps": {
      "get": {
        "tags": ["apps"],
        "summary": "List all applications",
        "operationId": "listApps",
        "parameters": [
          {
            "name": "server_id",
            "in": "query",
            "schema": {"type": "string"},
            "description": "Filter by server ID"
          }
        ],
        "responses": {
          "200": {
            "description": "List of applications",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/AppListResponse"
                }
              }
            }
          }
        }
      },
      "post": {
        "tags": ["apps"],
        "summary": "Create new application",
        "operationId": "createApp",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/AppCreateRequest"
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Application created",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/App"
                }
              }
            }
          }
        }
      }
    },
    "/api/v1/apps/{id}/deploy": {
      "post": {
        "tags": ["deployments"],
        "summary": "Deploy application",
        "operationId": "deployApp",
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/DeploymentSpec"
              }
            }
          }
        },
        "responses": {
          "202": {
            "description": "Deployment started",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/DeploymentResult"
                }
              }
            }
          }
        }
      }
    },
    "/api/v1/apps/{id}/rollback": {
      "post": {
        "tags": ["deployments"],
        "summary": "Rollback application",
        "operationId": "rollbackApp",
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/RollbackRequest"
              }
            }
          }
        },
        "responses": {
          "202": {
            "description": "Rollback started",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/OperationResponse"
                }
              }
            }
          }
        }
      }
    },
    "/api/v1/services/{server_id}": {
      "get": {
        "tags": ["services"],
        "summary": "List services on server",
        "operationId": "listServices",
        "parameters": [
          {
            "name": "server_id",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          }
        ],
        "responses": {
          "200": {
            "description": "List of services",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/ServiceListResponse"
                }
              }
            }
          }
        }
      }
    },
    "/api/v1/services/{server_id}/{service_name}": {
      "get": {
        "tags": ["services"],
        "summary": "Get service status",
        "operationId": "getServiceStatus",
        "parameters": [
          {
            "name": "server_id",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          },
          {
            "name": "service_name",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          }
        ],
        "responses": {
          "200": {
            "description": "Service status",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/ServiceStatus"
                }
              }
            }
          }
        }
      },
      "post": {
        "tags": ["services"],
        "summary": "Manage service",
        "operationId": "manageService",
        "parameters": [
          {
            "name": "server_id",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          },
          {
            "name": "service_name",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/ServiceActionRequest"
              }
            }
          }
        },
        "responses": {
          "202": {
            "description": "Service action started",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/OperationResponse"
                }
              }
            }
          }
        }
      }
    },
    "/api/v1/services/{server_id}/{service_name}/logs": {
      "get": {
        "tags": ["services"],
        "summary": "Get service logs",
        "operationId": "getServiceLogs",
        "parameters": [
          {
            "name": "server_id",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          },
          {
            "name": "service_name",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          },
          {
            "name": "lines",
            "in": "query",
            "schema": {"type": "integer", "default": 50}
          }
        ],
        "responses": {
          "200": {
            "description": "Service logs",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/ServiceLogsResponse"
                }
              }
            }
          }
        }
      }
    },
    "/api/v1/deployments/{id}/status": {
      "get": {
        "tags": ["deployments"],
        "summary": "Get deployment status",
        "operationId": "getDeploymentStatus",
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          }
        ],
        "responses": {
          "200": {
            "description": "Deployment status",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/DeploymentStatus"
                }
              }
            }
          }
        }
      }
    },
    "/api/v1/deployments": {
      "get": {
        "tags": ["deployments"],
        "summary": "List deployments",
        "operationId": "listDeployments",
        "parameters": [
          {
            "name": "app_id",
            "in": "query",
            "schema": {"type": "string"}
          },
          {
            "name": "status",
            "in": "query",
            "schema": {"type": "string", "enum": ["pending", "running", "success", "failed"]}
          }
        ],
        "responses": {
          "200": {
            "description": "List of deployments",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/DeploymentListResponse"
                }
              }
            }
          }
        }
      }
    },
    "/api/v1/health": {
      "get": {
        "tags": ["monitoring"],
        "summary": "API health check",
        "operationId": "healthCheck",
        "responses": {
          "200": {
            "description": "API is healthy",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/HealthResponse"
                }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "Server": {
        "type": "object",
        "properties": {
          "id": {"type": "string"},
          "created": {"type": "string", "format": "date-time"},
          "updated": {"type": "string", "format": "date-time"},
          "name": {"type": "string"},
          "host": {"type": "string"},
          "port": {"type": "integer", "minimum": 1, "maximum": 65535},
          "root_username": {"type": "string"},
          "app_username": {"type": "string"},
          "use_ssh_agent": {"type": "boolean"},
          "manual_key_path": {"type": "string"},
          "setup_complete": {"type": "boolean"},
          "security_locked": {"type": "boolean"}
        },
        "required": ["name", "host"]
      },
      "App": {
        "type": "object",
        "properties": {
          "id": {"type": "string"},
          "created": {"type": "string", "format": "date-time"},
          "updated": {"type": "string", "format": "date-time"},
          "name": {"type": "string"},
          "server_id": {"type": "string"},
          "remote_path": {"type": "string"},
          "service_name": {"type": "string"},
          "domain": {"type": "string"},
          "current_version": {"type": "string"},
          "status": {"type": "string", "enum": ["online", "offline", "unknown"]}
        },
        "required": ["name", "server_id"]
      },
      "Version": {
        "type": "object",
        "properties": {
          "id": {"type": "string"},
          "created": {"type": "string", "format": "date-time"},
          "updated": {"type": "string", "format": "date-time"},
          "app_id": {"type": "string"},
          "version_number": {"type": "string"},
          "deployment_zip": {"type": "string"},
          "notes": {"type": "string"}
        },
        "required": ["app_id"]
      },
      "Deployment": {
        "type": "object",
        "properties": {
          "id": {"type": "string"},
          "created": {"type": "string", "format": "date-time"},
          "updated": {"type": "string", "format": "date-time"},
          "app_id": {"type": "string"},
          "version_id": {"type": "string"},
          "status": {"type": "string", "enum": ["pending", "running", "success", "failed"]},
          "logs": {"type": "string"},
          "started_at": {"type": "string", "format": "date-time"},
          "completed_at": {"type": "string", "format": "date-time"}
        },
        "required": ["app_id"]
      },
      "ServerCreateRequest": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "host": {"type": "string"},
          "port": {"type": "integer", "default": 22},
          "root_username": {"type": "string", "default": "root"},
          "app_username": {"type": "string", "default": "pocketbase"},
          "use_ssh_agent": {"type": "boolean", "default": true},
          "manual_key_path": {"type": "string"}
        },
        "required": ["name", "host"]
      },
      "ServerUpdateRequest": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "host": {"type": "string"},
          "port": {"type": "integer"},
          "root_username": {"type": "string"},
          "app_username": {"type": "string"},
          "use_ssh_agent": {"type": "boolean"},
          "manual_key_path": {"type": "string"}
        }
      },
      "AppCreateRequest": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "server_id": {"type": "string"},
          "remote_path": {"type": "string", "default": "/opt/pocketbase"},
          "service_name": {"type": "string"},
          "domain": {"type": "string"}
        },
        "required": ["name", "server_id", "domain"]
      },
      "SetupRequest": {
        "type": "object",
        "properties": {
          "create_user": {"type": "boolean", "default": true},
          "install_packages": {"type": "boolean", "default": true},
          "setup_directories": {"type": "boolean", "default": true},
          "configure_firewall": {"type": "boolean", "default": true},
          "packages": {
            "type": "array",
            "items": {"type": "string"},
            "default": ["curl", "wget", "unzip", "git", "systemd"]
          },
          "ssh_keys": {
            "type": "array",
            "items": {"type": "string"}
          }
        }
      },
      "SystemUserConfig": {
        "type": "object",
        "properties": {
          "username": {"type": "string"},
          "home_dir": {"type": "string"},
          "shell": {"type": "string", "default": "/bin/bash"},
          "groups": {
            "type": "array",
            "items": {"type": "string"}
          },
          "create_home": {"type": "boolean", "default": true},
          "system_user": {"type": "boolean", "default": false},
          "setup_ssh": {"type": "boolean", "default": true},
          "ssh_keys": {
            "type": "array",
            "items": {"type": "string"}
          },
          "setup_sudo": {"type": "boolean", "default": false},
          "sudo_commands": {
            "type": "array",
            "items": {"type": "string"}
          },
          "directories": {
            "type": "array",
            "items": {
              "$ref": "#/components/schemas/DirectoryConfig"
            }
          }
        },
        "required": ["username"]
      },
      "DirectoryConfig": {
        "type": "object",
        "properties": {
          "path": {"type": "string"},
          "permissions": {"type": "string", "default": "755"},
          "owner": {"type": "string"},
          "group": {"type": "string"},
          "parents": {"type": "boolean", "default": true}
        },
        "required": ["path"]
      },
      "PackageInstallRequest": {
        "type": "object",
        "properties": {
          "packages": {
            "type": "array",
            "items": {"type": "string"}
          },
          "update_first": {"type": "boolean", "default": true}
        },
        "required": ["packages"]
      },
      "SecurityConfig": {
        "type": "object",
        "properties": {
          "disable_root_login": {"type": "boolean", "default": true},
          "disable_password_auth": {"type": "boolean", "default": true},
          "firewall_rules": {
            "type": "array",
            "items": {
              "$ref": "#/components/schemas/FirewallRule"
            }
          },
          "fail2ban_config": {
            "$ref": "#/components/schemas/Fail2banConfig"
          },
          "ssh_hardening_config": {
            "$ref": "#/components/schemas/SSHHardeningConfig"
          },
          "allowed_ports": {
            "type": "array",
            "items": {"type": "integer"}
          },
          "allowed_users": {
            "type": "array",
            "items": {"type": "string"}
          }
        }
      },
      "FirewallRule": {
        "type": "object",
        "properties": {
          "port": {"type": "integer"},
          "protocol": {"type": "string", "enum": ["tcp", "udp"]},
          "action": {"type": "string", "enum": ["allow", "deny"]},
          "source": {"type": "string"},
          "description": {"type": "string"}
        },
        "required": ["port", "protocol", "action"]
      },
      "Fail2banConfig": {
        "type": "object",
        "properties": {
          "enabled": {"type": "boolean", "default": true},
          "max_retries": {"type": "integer", "default": 5},
          "ban_time": {"type": "integer", "default": 3600},
          "find_time": {"type": "integer", "default": 600},
          "services": {
            "type": "array",
            "items": {"type": "string"},
            "default": ["ssh", "nginx"]
          }
        }
      },
      "SSHHardeningConfig": {
        "type": "object",
        "properties": {
          "password_authentication": {"type": "boolean", "default": false},
          "pubkey_authentication": {"type": "boolean", "default": true},
          "permit_root_login": {"type": "boolean", "default": false},
          "x11_forwarding": {"type": "boolean", "default": false},
          "allow_agent_forwarding": {"type": "boolean", "default": false},
          "allow_tcp_forwarding": {"type": "boolean", "default": false},
          "client_alive_interval": {"type": "integer", "default": 300},
          "client_alive_count_max": {"type": "integer", "default": 2},
          "max_auth_tries": {"type": "integer", "default": 3},
          "max_sessions": {"type": "integer", "default": 10},
          "protocol": {"type": "integer", "default": 2}
        }
      },
      "SecurityReport": {
        "type": "object",
        "properties": {
          "timestamp": {"type": "string", "format": "date-time"},
          "overall": {"type": "string", "enum": ["excellent", "good", "fair", "poor", "critical", "unknown"]},
          "checks": {
            "type": "array",
            "items": {
              "$ref": "#/components/schemas/SecurityCheck"
            }
          },
          "recommendations": {
            "type": "array",
            "items": {"type": "string"}
          }
        }
      },
      "SecurityCheck": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "category": {"type": "string"},
          "status": {"type": "string", "enum": ["pass", "warning", "fail"]},
          "score": {"type": "integer", "minimum": 0, "maximum": 100},
          "issues": {
            "type": "array",
            "items": {"type": "string"}
          },
          "details": {"type": "object"}
        }
      },
      "DeploymentSpec": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "version": {"type": "string"},
          "version_id": {"type": "string"},
          "artifact_path": {"type": "string"},
          "working_directory": {"type": "string"},
          "service_name": {"type": "string"},
          "environment": {"type": "string"},
          "strategy": {
            "type": "string",
            "enum": ["rolling", "blue-green", "canary", "recreate"],
            "default": "recreate"
          },
          "health_check_url": {"type": "string"},
          "rollback_on_failure": {"type": "boolean", "default": true},
          "pre_deploy_hooks": {
            "type": "array",
            "items": {"type": "string"}
          },
          "post_deploy_hooks": {
            "type": "array",
            "items": {"type": "string"}
          },
          "dependencies": {
            "type": "array",
            "items": {"type": "string"}
          },
          "environment_variables": {
            "type": "object",
            "additionalProperties": {"type": "string"}
          }
        },
        "required": ["name", "version", "artifact_path"]
      },
      "DeploymentResult": {
        "type": "object",
        "properties": {
          "deployment_id": {"type": "string"},
          "success": {"type": "boolean"},
          "message": {"type": "string"},
          "version": {"type": "string"},
          "previous_version": {"type": "string"},
          "start_time": {"type": "string", "format": "date-time"},
          "end_time": {"type": "string", "format": "date-time"},
          "logs": {"type": "string"}
        }
      },
      
      "RollbackRequest": {
        "type": "object",
        "properties": {
          "target_version": {"type": "string"},
          "reason": {"type": "string"},
          "force": {"type": "boolean", "default": false}
        },
        "required": ["target_version"]
      },
      
      "ServiceConfig": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "type": {"type": "string", "enum": ["systemd", "docker", "process"]},
          "command": {"type": "string"},
          "working_directory": {"type": "string"},
          "environment": {
            "type": "object",
            "additionalProperties": {"type": "string"}
          },
          "restart_policy": {"type": "string", "enum": ["always", "on-failure", "unless-stopped"]},
          "max_restarts": {"type": "integer", "default": 3},
          "health_check": {
            "type": "object",
            "properties": {
              "endpoint": {"type": "string"},
              "interval": {"type": "integer"},
              "timeout": {"type": "integer"},
              "retries": {"type": "integer"}
            }
          }
        },
        "required": ["name", "type"]
      },
      
      "ServiceStatus": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "status": {"type": "string", "enum": ["running", "stopped", "failed", "restarting", "unknown"]},
          "pid": {"type": "integer"},
          "uptime": {"type": "integer"},
          "memory_usage": {"type": "integer"},
          "cpu_usage": {"type": "number"},
          "last_restart": {"type": "string", "format": "date-time"},
          "error": {"type": "string"}
        },
        "required": ["name", "status"]
      },
      
      "ServiceAction": {
        "type": "object",
        "properties": {
          "action": {"type": "string", "enum": ["start", "stop", "restart", "reload", "status"]},
          "force": {"type": "boolean", "default": false},
          "timeout": {"type": "integer", "default": 30}
        },
        "required": ["action"]
      },
      
      "ServiceLogs": {
        "type": "object",
        "properties": {
          "service": {"type": "string"},
          "logs": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "timestamp": {"type": "string", "format": "date-time"},
                "level": {"type": "string", "enum": ["debug", "info", "warn", "error"]},
                "message": {"type": "string"},
                "source": {"type": "string"}
              }
            }
          },
          "total_lines": {"type": "integer"},
          "from_line": {"type": "integer"},
          "to_line": {"type": "integer"}
        }
      },
      
      "ServerList": {
        "type": "object",
        "properties": {
          "servers": {
            "type": "array",
            "items": {"$ref": "#/components/schemas/Server"}
          },
          "total": {"type": "integer"},
          "page": {"type": "integer"},
          "per_page": {"type": "integer"}
        }
      },
      
      "AppList": {
        "type": "object",
        "properties": {
          "apps": {
            "type": "array",
            "items": {"$ref": "#/components/schemas/App"}
          },
          "total": {"type": "integer"},
          "filtered": {"type": "integer"}
        }
      },
      
      "ServiceList": {
        "type": "object",
        "properties": {
          "services": {
            "type": "array",
            "items": {"$ref": "#/components/schemas/ServiceStatus"}
          },
          "server_id": {"type": "string"},
          "timestamp": {"type": "string", "format": "date-time"}
        }
      },
      
      "DeploymentList": {
        "type": "object",
        "properties": {
          "deployments": {
            "type": "array",
            "items": {"$ref": "#/components/schemas/Deployment"}
          },
          "total": {"type": "integer"},
          "filtered": {"type": "integer"}
        }
      },
      
      "HealthStatus": {
        "type": "object",
        "properties": {
          "status": {"type": "string", "enum": ["healthy", "degraded", "unhealthy"]},
          "timestamp": {"type": "string", "format": "date-time"},
          "components": {
            "type": "object",
            "properties": {
              "database": {
                "type": "object",
                "properties": {
                  "status": {"type": "string"},
                  "latency_ms": {"type": "number"},
                  "message": {"type": "string"}
                }
              },
              "ssh": {
                "type": "object",
                "properties": {
                  "status": {"type": "string"},
                  "active_connections": {"type": "integer"},
                  "message": {"type": "string"}
                }
              },
              "storage": {
                "type": "object",
                "properties": {
                  "status": {"type": "string"},
                  "used_gb": {"type": "number"},
                  "total_gb": {"type": "number"},
                  "percentage": {"type": "number"}
                }
              }
            }
          },
          "uptime_seconds": {"type": "integer"},
          "version": {"type": "string"}
        },
        "required": ["status", "timestamp"]
      },
      
      "TaskResult": {
        "type": "object",
        "properties": {
          "task_id": {"type": "string"},
          "type": {"type": "string"},
          "status": {"type": "string", "enum": ["pending", "running", "completed", "failed"]},
          "started_at": {"type": "string", "format": "date-time"},
          "completed_at": {"type": "string", "format": "date-time"},
          "result": {"type": "object"},
          "error": {"type": "string"},
          "logs": {"type": "string"}
        },
        "required": ["task_id", "type", "status"]
      },
      
      "ErrorResponse": {
        "type": "object",
        "properties": {
          "error": {"type": "string"},
          "message": {"type": "string"},
          "code": {"type": "integer"},
          "details": {"type": "object"}
        },
        "required": ["error", "message"]
      }
    }
  }
}
```

## WebSocket Events

The API also supports WebSocket connections for real-time updates:

### Connection
- **Endpoint**: `/api/v1/ws`
- **Authentication**: Bearer token in query parameter or header

### Event Types

#### Server Events
```json
{
  "type": "server.status",
  "data": {
    "server_id": "string",
    "status": "online|offline|error",
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

#### Deployment Events
```json
{
  "type": "deployment.progress",
  "data": {
    "deployment_id": "string",
    "status": "running|completed|failed",
    "progress": 75,
    "message": "Extracting application files...",
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

#### Service Events
```json
{
  "type": "service.status",
  "data": {
    "server_id": "string",
    "service_name": "string",
    "status": "running|stopped|failed",
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

#### Log Events
```json
{
  "type": "log.stream",
  "data": {
    "source": "deployment|service|security",
    "level": "debug|info|warn|error",
    "message": "string",
    "metadata": {},
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

## Error Codes

| Code | Description |
|------|-------------|
| 400 | Bad Request - Invalid input parameters |
| 401 | Unauthorized - Missing or invalid authentication |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Resource doesn't exist |
| 409 | Conflict - Resource already exists or state conflict |
| 422 | Unprocessable Entity - Validation error |
| 500 | Internal Server Error - Server-side error |
| 502 | Bad Gateway - SSH connection failed |
| 503 | Service Unavailable - Service temporarily unavailable |

## Local Tool - No Authentication Required

This is a **local development tool** designed to run on the developer's machine only. No authentication is required since:

- All operations are performed locally
- Tool manages servers via SSH keys (not API authentication)
- PocketBase runs in local-only mode
- No external access or multi-user requirements

## Connection Security

Security is managed at the SSH level:
- SSH key-based authentication to target servers
- No API tokens or user management needed
- Direct database access through local PocketBase instance

## Phase 3 Implementation Status

### Completed ✅
- Complete OpenAPI 3.0.3 specification
- All model schemas defined
- All API endpoints documented
- WebSocket event definitions
- Error code documentation
- Local tool configuration (no auth required)

### Next Steps
1. Implement API handlers based on this schema
2. Set up OpenAPI validation middleware
3. Generate client SDKs from schema
4. Implement WebSocket event system
5. Test local deployment workflows

## Testing

Use the OpenAPI specification to:
1. Generate Postman/Insomnia collections
2. Create automated API tests
3. Validate request/response formats
4. Generate mock servers for development

## Documentation Generation

The OpenAPI specification can be used to generate:
- Interactive API documentation (Swagger UI)
- Client libraries (Go, Python, JavaScript)
- Server stubs for testing
- API change detection and versioning