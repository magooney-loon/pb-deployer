# Configuration

Configure pb-deployer to match your deployment needs with flexible configuration options.

## Configuration File

Create a `pb-deployer.config.js` file in your project root:

```javascript
export default {
  app: {
    name: "my-pocketbase-app",
    port: 8080,
    version: "1.0.0"
  },
  database: {
    path: "./pb_data",
    backupBeforeDeploy: true
  },
  deployment: {
    provider: "vercel",
    domain: "my-app.vercel.app",
    environment: "production"
  },
  build: {
    command: "npm run build",
    outputDir: "./dist"
  }
}
```

## Configuration Options

### App Settings

Configure your application basics:

- **name** - Your application name (used for deployment naming)
- **port** - Local development port (default: 8080)
- **version** - Application version for tracking deployments

### Database Configuration

PocketBase database settings:

- **path** - Path to your PocketBase data directory
- **backupBeforeDeploy** - Create backup before each deployment
- **migrations** - Path to migration files (optional)

### Deployment Settings

Choose your deployment target:

```javascript
deployment: {
  provider: "vercel",        // vercel, railway, digitalocean
  domain: "app.example.com", // Custom domain (optional)
  environment: "production", // production, staging, development
  region: "us-east-1"       // Deployment region
}
```

## Supported Providers

### Vercel

```javascript
deployment: {
  provider: "vercel",
  domain: "my-app.vercel.app",
  environment: "production"
}
```

### Railway

```javascript
deployment: {
  provider: "railway",
  projectId: "your-project-id",
  environment: "production"
}
```

### DigitalOcean

```javascript
deployment: {
  provider: "digitalocean",
  region: "nyc1",
  size: "basic"
}
```

## Environment Variables

Define environment-specific variables:

```javascript
env: {
  production: {
    PB_ENCRYPTION_KEY: "your-production-key",
    DATABASE_URL: "production-db-url"
  },
  staging: {
    PB_ENCRYPTION_KEY: "your-staging-key",
    DATABASE_URL: "staging-db-url"
  }
}
```

## Build Configuration

Customize the build process:

```javascript
build: {
  command: "npm run build",
  outputDir: "./dist",
  beforeBuild: ["npm run lint", "npm run test"],
  afterBuild: ["npm run optimize"]
}
```

## Advanced Options

### Custom Hooks

Run custom scripts during deployment:

```javascript
hooks: {
  beforeDeploy: "npm run pre-deploy",
  afterDeploy: "npm run post-deploy",
  onError: "npm run cleanup"
}
```

### Health Checks

Configure application monitoring:

```javascript
healthCheck: {
  enabled: true,
  endpoint: "/health",
  timeout: 30000,
  retries: 3
}
```

## Environment-Specific Configs

Create separate config files for different environments:

- `pb-deployer.config.js` - Default/production
- `pb-deployer.staging.config.js` - Staging environment
- `pb-deployer.dev.config.js` - Development environment

Use with:

```bash
pb-deployer deploy --config staging
```

## Configuration Validation

pb-deployer validates your configuration automatically. Common validation rules:

- **Required fields**: `app.name`, `deployment.provider`
- **Valid providers**: Must be a supported deployment provider
- **Port ranges**: Development ports must be between 1000-65535
- **Domain format**: Must be a valid domain name format

> **Pro Tip**: Use `pb-deployer validate` to check your configuration without deploying!