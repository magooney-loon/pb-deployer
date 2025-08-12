# Deployment

Deploy your PocketBase application to production with confidence using pb-deployer's streamlined deployment process.

## Basic Deployment

### First Deployment

Deploy your app for the first time:

```bash
# Deploy to production
pb-deployer deploy

# Deploy with preview
pb-deployer deploy --preview

# Deploy to specific environment
pb-deployer deploy --env staging
```

### Deployment Process

pb-deployer follows a structured deployment workflow:

1. **Pre-deployment checks** - Validates configuration and dependencies
2. **Database backup** - Creates automatic backup (if enabled)
3. **Build process** - Compiles and optimizes your application
4. **Provider deployment** - Uploads to your chosen cloud provider
5. **Health checks** - Verifies deployment success
6. **Post-deployment hooks** - Runs any custom scripts

## Deployment Strategies

### Rolling Deployment

Default strategy for zero-downtime deployments:

```javascript
deployment: {
  strategy: "rolling",
  maxUnavailable: "25%",
  progressDeadline: "600s"
}
```

### Blue-Green Deployment

For critical applications requiring instant rollback:

```javascript
deployment: {
  strategy: "blue-green",
  autoSwitch: false,
  testEndpoint: "/health"
}
```

## Environment Management

### Multiple Environments

Deploy to different environments:

```bash
# Development
pb-deployer deploy --env dev

# Staging
pb-deployer deploy --env staging

# Production
pb-deployer deploy --env production
```

### Environment Configuration

```javascript
environments: {
  development: {
    provider: "vercel",
    domain: "dev-app.vercel.app",
    database: { path: "./dev_data" }
  },
  staging: {
    provider: "railway",
    domain: "staging-app.railway.app",
    database: { path: "./staging_data" }
  },
  production: {
    provider: "digitalocean",
    domain: "app.example.com",
    database: { path: "./prod_data" }
  }
}
```

## Database Handling

### Automatic Backups

pb-deployer can automatically backup your database before each deployment:

```javascript
database: {
  backupBeforeDeploy: true,
  backupRetention: 7, // days
  backupLocation: "./backups"
}
```

### Migration Support

Run database migrations during deployment:

```javascript
database: {
  migrations: {
    enabled: true,
    path: "./migrations",
    autoRun: true
  }
}
```

## Monitoring and Health Checks

### Health Check Configuration

Ensure your deployment is healthy:

```javascript
healthCheck: {
  enabled: true,
  endpoint: "/api/health",
  timeout: 30000,
  interval: 10000,
  retries: 5,
  successThreshold: 2
}
```

### Status Monitoring

Check deployment status:

```bash
# Check current status
pb-deployer status

# Watch deployment progress
pb-deployer deploy --watch

# Get detailed logs
pb-deployer logs --tail 100
```

## Rollback and Recovery

### Quick Rollback

If something goes wrong, rollback quickly:

```bash
# Rollback to previous version
pb-deployer rollback

# Rollback to specific version
pb-deployer rollback --version 1.2.3

# List available versions
pb-deployer versions
```

### Database Recovery

Restore from backup if needed:

```bash
# List available backups
pb-deployer backup list

# Restore from backup
pb-deployer backup restore --backup 2024-01-15-10-30
```

## Custom Deployment Hooks

### Pre and Post Deployment

Run custom scripts during deployment:

```javascript
hooks: {
  beforeDeploy: [
    "npm run lint",
    "npm run test",
    "npm run build:check"
  ],
  afterDeploy: [
    "npm run notify:slack",
    "npm run update:docs"
  ],
  onSuccess: "npm run celebrate",
  onError: "npm run alert:team"
}
```

## Deployment Best Practices

### 1. Test Before Deploy

Always test your deployment configuration:

```bash
pb-deployer deploy --dry-run --verbose
```

### 2. Use Staging Environment

Deploy to staging first:

```bash
pb-deployer deploy --env staging
pb-deployer test --env staging
pb-deployer deploy --env production
```

### 3. Monitor After Deployment

Keep an eye on your deployment:

```bash
pb-deployer status --watch
pb-deployer logs --follow
```

### 4. Database Safety

Always backup before major deployments:

```bash
pb-deployer backup create --name "pre-v2-deployment"
pb-deployer deploy
```

## Troubleshooting Deployments

### Common Issues

**Deployment Timeout**
- Increase timeout in configuration
- Check network connectivity
- Verify provider status

**Database Migration Errors**
- Review migration scripts
- Check database permissions
- Ensure backup exists

**Health Check Failures**
- Verify health endpoint is accessible
- Check application startup time
- Review application logs

> **Need help?** Check the [Troubleshooting](#troubleshooting) section or reach out on [GitHub Issues](https://github.com/your-username/pb-deployer/issues).