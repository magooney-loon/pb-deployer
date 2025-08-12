# API Reference

Complete command-line interface reference for pb-deployer.

## Core Commands

### `pb-deployer init`

Initialize a new pb-deployer project in the current directory.

```bash
pb-deployer init [options]
```

**Options:**
- `--template <name>` - Use a specific template (basic, advanced, full)
- `--provider <provider>` - Set default deployment provider
- `--yes` - Skip interactive prompts and use defaults

**Examples:**
```bash
# Interactive setup
pb-deployer init

# Quick setup with defaults
pb-deployer init --yes --provider vercel

# Use advanced template
pb-deployer init --template advanced
```

### `pb-deployer deploy`

Deploy your PocketBase application to the configured provider.

```bash
pb-deployer deploy [environment] [options]
```

**Arguments:**
- `environment` - Target environment (dev, staging, production)

**Options:**
- `--dry-run` - Preview changes without deploying
- `--watch` - Watch deployment progress in real-time
- `--force` - Force deployment even with warnings
- `--config <file>` - Use specific configuration file
- `--backup` - Create backup before deployment

**Examples:**
```bash
# Deploy to production
pb-deployer deploy production

# Preview deployment changes
pb-deployer deploy --dry-run

# Deploy with backup
pb-deployer deploy --backup --watch
```

### `pb-deployer status`

Check the status of your deployed application.

```bash
pb-deployer status [options]
```

**Options:**
- `--env <environment>` - Check specific environment
- `--detailed` - Show detailed status information
- `--json` - Output status as JSON

**Examples:**
```bash
# Check production status
pb-deployer status

# Detailed staging status
pb-deployer status --env staging --detailed
```

## Configuration Commands

### `pb-deployer config`

Manage configuration settings.

```bash
pb-deployer config <action> [options]
```

**Actions:**
- `get <key>` - Get configuration value
- `set <key> <value>` - Set configuration value
- `list` - List all configuration
- `validate` - Validate current configuration

**Examples:**
```bash
# Get app name
pb-deployer config get app.name

# Set deployment provider
pb-deployer config set deployment.provider railway

# Validate configuration
pb-deployer config validate
```

### `pb-deployer env`

Manage environment variables.

```bash
pb-deployer env <action> [options]
```

**Actions:**
- `list [environment]` - List environment variables
- `set <key> <value>` - Set environment variable
- `delete <key>` - Delete environment variable
- `import <file>` - Import from .env file

**Examples:**
```bash
# List production variables
pb-deployer env list production

# Set API key
pb-deployer env set API_KEY "your-key-here"

# Import from .env file
pb-deployer env import .env.production
```

## Database Commands

### `pb-deployer backup`

Manage database backups.

```bash
pb-deployer backup <action> [options]
```

**Actions:**
- `create` - Create a new backup
- `list` - List available backups
- `restore <backup-id>` - Restore from backup
- `delete <backup-id>` - Delete a backup

**Options:**
- `--name <name>` - Custom backup name
- `--compress` - Compress backup file
- `--remote` - Store backup remotely

**Examples:**
```bash
# Create named backup
pb-deployer backup create --name "pre-v2-deploy"

# List all backups
pb-deployer backup list

# Restore specific backup
pb-deployer backup restore backup-20240115-103045
```

### `pb-deployer migrate`

Run database migrations.

```bash
pb-deployer migrate <action> [options]
```

**Actions:**
- `up` - Run pending migrations
- `down` - Rollback last migration
- `status` - Show migration status
- `create <name>` - Create new migration

**Examples:**
```bash
# Run all pending migrations
pb-deployer migrate up

# Create new migration
pb-deployer migrate create "add-user-fields"

# Check migration status
pb-deployer migrate status
```

## Utility Commands

### `pb-deployer doctor`

Diagnose system and configuration issues.

```bash
pb-deployer doctor [options]
```

**Options:**
- `--fix` - Automatically fix common issues
- `--verbose` - Show detailed diagnostic information

### `pb-deployer logs`

View application and deployment logs.

```bash
pb-deployer logs [options]
```

**Options:**
- `--tail <number>` - Show last N log entries
- `--follow` - Follow log output in real-time
- `--level <level>` - Filter by log level (error, warn, info, debug)
- `--env <environment>` - Show logs for specific environment

**Examples:**
```bash
# View last 50 log entries
pb-deployer logs --tail 50

# Follow logs in real-time
pb-deployer logs --follow

# Show only errors
pb-deployer logs --level error
```

### `pb-deployer versions`

Manage application versions and deployments.

```bash
pb-deployer versions <action> [options]
```

**Actions:**
- `list` - List all deployed versions
- `info <version>` - Get version details
- `compare <v1> <v2>` - Compare two versions

### `pb-deployer rollback`

Rollback to a previous version.

```bash
pb-deployer rollback [version] [options]
```

**Arguments:**
- `version` - Specific version to rollback to (optional)

**Options:**
- `--confirm` - Skip confirmation prompt
- `--backup` - Create backup before rollback

## Global Options

These options work with all commands:

- `--help, -h` - Show help information
- `--version, -v` - Show version number
- `--config <file>` - Use specific configuration file
- `--verbose` - Enable verbose output
- `--quiet, -q` - Suppress non-essential output
- `--no-color` - Disable colored output

## Exit Codes

pb-deployer uses standard exit codes:

- `0` - Success
- `1` - General error
- `2` - Misuse of command
- `3` - Configuration error
- `4` - Network/connectivity error
- `5` - Deployment failed
- `6` - Validation error

## Environment Variables

Control pb-deployer behavior with environment variables:

- `PB_DEPLOYER_CONFIG` - Path to configuration file
- `PB_DEPLOYER_LOG_LEVEL` - Set log level (debug, info, warn, error)
- `PB_DEPLOYER_NO_COLOR` - Disable colored output
- `PB_DEPLOYER_API_KEY` - API key for cloud providers

## API Integration

### Programmatic Usage

Use pb-deployer in your Node.js applications:

```javascript
import { PBDeployer } from 'pb-deployer';

const deployer = new PBDeployer({
  configPath: './pb-deployer.config.js'
});

// Deploy programmatically
await deployer.deploy({
  environment: 'production',
  dryRun: false
});

// Check status
const status = await deployer.getStatus();
console.log('Deployment status:', status);
```

### Webhook Integration

Set up webhooks for deployment notifications:

```javascript
webhooks: {
  onDeploy: "https://api.slack.com/webhooks/...",
  onError: "https://api.discord.com/webhooks/...",
  onSuccess: "https://hooks.zapier.com/..."
}
```

## Configuration Schema

### Complete Configuration Reference

```javascript
{
  // Application settings
  app: {
    name: string,           // Required
    version: string,        // Default: "1.0.0"
    port: number,          // Default: 8080
    description: string    // Optional
  },

  // Database configuration
  database: {
    path: string,                    // Required
    backupBeforeDeploy: boolean,    // Default: true
    migrations: {
      enabled: boolean,             // Default: false
      path: string,                // Default: "./migrations"
      autoRun: boolean             // Default: false
    }
  },

  // Deployment settings
  deployment: {
    provider: string,              // Required: vercel, railway, digitalocean
    domain: string,               // Optional
    environment: string,          // Default: "production"
    region: string,              // Provider-specific
    strategy: string,            // rolling, blue-green
    timeout: number             // Default: 300000 (5 minutes)
  },

  // Build configuration
  build: {
    command: string,             // Default: "npm run build"
    outputDir: string,          // Default: "./dist"
    beforeBuild: string[],      // Pre-build commands
    afterBuild: string[]       // Post-build commands
  },

  // Environment variables
  env: {
    [environment: string]: {
      [key: string]: string
    }
  },

  // Deployment hooks
  hooks: {
    beforeDeploy: string | string[],
    afterDeploy: string | string[],
    onSuccess: string | string[],
    onError: string | string[]
  },

  // Health check settings
  healthCheck: {
    enabled: boolean,           // Default: true
    endpoint: string,          // Default: "/health"
    timeout: number,           // Default: 30000
    interval: number,          // Default: 10000
    retries: number,           // Default: 3
    successThreshold: number   // Default: 1
  },

  // Webhook notifications
  webhooks: {
    onDeploy: string,
    onSuccess: string,
    onError: string,
    onRollback: string
  }
}
```

> **Next**: Learn how to handle common issues in [Troubleshooting](#troubleshooting).