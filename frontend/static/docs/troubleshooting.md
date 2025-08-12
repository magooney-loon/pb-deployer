# Troubleshooting

Common issues and solutions when using pb-deployer for your PocketBase deployments.

## Deployment Issues

### Deployment Fails to Start

**Symptoms:**
- Deployment command hangs or fails immediately
- Error: "Failed to connect to deployment provider"

**Solutions:**
1. Check your internet connection
2. Verify API credentials are correct
3. Ensure the deployment provider is not experiencing outages

```bash
# Test connectivity
pb-deployer doctor --verbose

# Validate configuration
pb-deployer config validate

# Check provider status
pb-deployer status --provider
```

### Authentication Errors

**Symptoms:**
- Error: "Invalid API key"
- Error: "Authentication failed"

**Solutions:**
1. Verify your API keys are set correctly
2. Check if keys have expired
3. Ensure proper permissions are granted

```bash
# Set API key
pb-deployer env set API_KEY "your-new-key"

# Test authentication
pb-deployer auth test
```

### Build Failures

**Symptoms:**
- Build process fails during deployment
- Error: "Build command failed"

**Solutions:**
1. Test build locally first
2. Check build dependencies
3. Verify build output directory exists

```bash
# Test build locally
npm run build

# Check build configuration
pb-deployer config get build

# Deploy with verbose logging
pb-deployer deploy --verbose
```

## Database Issues

### Migration Failures

**Symptoms:**
- Error: "Migration failed to apply"
- Database schema mismatch

**Solutions:**
1. Check migration file syntax
2. Verify database permissions
3. Ensure migration order is correct

```bash
# Check migration status
pb-deployer migrate status

# Test migration locally
pb-deployer migrate up --dry-run

# Create database backup first
pb-deployer backup create --name "pre-migration"
```

### Backup Issues

**Symptoms:**
- Error: "Failed to create backup"
- Backup restoration fails

**Solutions:**
1. Check disk space availability
2. Verify backup directory permissions
3. Ensure PocketBase is not running during backup

```bash
# Check backup directory
ls -la ./backups

# Test backup creation
pb-deployer backup create --test

# Verify backup integrity
pb-deployer backup verify <backup-id>
```

### Database Connection Errors

**Symptoms:**
- Error: "Cannot connect to database"
- Timeout errors during database operations

**Solutions:**
1. Verify database file exists and is accessible
2. Check file permissions
3. Ensure no other processes are using the database

```bash
# Check database file
ls -la ./pb_data/data.db

# Test database connection
pb-deployer db test

# Stop conflicting processes
pkill pocketbase
```

## Configuration Issues

### Invalid Configuration

**Symptoms:**
- Error: "Configuration validation failed"
- Missing required fields

**Solutions:**
1. Run configuration validation
2. Check configuration file syntax
3. Ensure all required fields are present

```bash
# Validate configuration
pb-deployer config validate

# Show configuration schema
pb-deployer config schema

# Reset to default configuration
pb-deployer init --reset
```

### Environment Variable Issues

**Symptoms:**
- Environment variables not loading
- Error: "Required environment variable missing"

**Solutions:**
1. Check .env file exists and is readable
2. Verify variable names match configuration
3. Ensure proper environment is selected

```bash
# List environment variables
pb-deployer env list

# Test environment loading
pb-deployer env test --env production

# Import from .env file
pb-deployer env import .env.production
```

## Network and Connectivity

### Timeout Errors

**Symptoms:**
- Operations timeout before completion
- Error: "Request timeout"

**Solutions:**
1. Increase timeout values in configuration
2. Check network stability
3. Verify provider service status

```javascript
// Increase timeouts in config
deployment: {
  timeout: 600000, // 10 minutes
  healthCheck: {
    timeout: 60000 // 1 minute
  }
}
```

### SSL/TLS Issues

**Symptoms:**
- Error: "SSL certificate verification failed"
- HTTPS connection errors

**Solutions:**
1. Update Node.js to latest version
2. Check system certificate store
3. Use `--insecure` flag for testing (not recommended for production)

```bash
# Update certificates (Ubuntu/Debian)
sudo apt-get update && sudo apt-get install ca-certificates

# Test with insecure flag (testing only)
pb-deployer deploy --insecure
```

## Performance Issues

### Slow Deployments

**Symptoms:**
- Deployments take much longer than expected
- Upload progress is very slow

**Solutions:**
1. Check file sizes in build output
2. Exclude unnecessary files
3. Use compression options

```javascript
// Optimize build output
build: {
  exclude: [
    "node_modules",
    "*.test.js",
    "docs/**",
    ".git"
  ],
  compress: true
}
```

### High Memory Usage

**Symptoms:**
- System becomes slow during deployment
- Out of memory errors

**Solutions:**
1. Increase system memory if possible
2. Deploy smaller chunks
3. Clear temporary files

```bash
# Clear pb-deployer cache
pb-deployer cache clear

# Deploy with memory optimization
pb-deployer deploy --optimize-memory
```

## Platform-Specific Issues

### Windows Issues

**Common Problems:**
- Path separator issues
- PowerShell execution policy
- Long path limitations

**Solutions:**
```powershell
# Set execution policy
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser

# Use forward slashes in paths
# Enable long paths in Windows 10+
```

### macOS Issues

**Common Problems:**
- Gatekeeper blocking execution
- Permission issues with Homebrew

**Solutions:**
```bash
# Allow pb-deployer through Gatekeeper
xattr -d com.apple.quarantine /usr/local/bin/pb-deployer

# Fix Homebrew permissions
sudo chown -R $(whoami) /usr/local/lib/node_modules
```

### Linux Issues

**Common Problems:**
- Permission denied errors
- Missing system dependencies

**Solutions:**
```bash
# Fix npm permissions
sudo chown -R $USER /usr/local/lib/node_modules

# Install missing dependencies (Ubuntu/Debian)
sudo apt-get install build-essential

# Install missing dependencies (CentOS/RHEL)
sudo yum groupinstall "Development Tools"
```

## Advanced Troubleshooting

### Debug Mode

Enable debug logging for detailed troubleshooting:

```bash
# Enable debug logging
export PB_DEPLOYER_LOG_LEVEL=debug
pb-deployer deploy

# Or use verbose flag
pb-deployer deploy --verbose --debug
```

### Log Analysis

Check log files for detailed error information:

```bash
# View deployment logs
pb-deployer logs --tail 100

# Export logs to file
pb-deployer logs --export deployment.log

# Filter error logs
pb-deployer logs --level error --since "1 hour ago"
```

### Configuration Debugging

Debug configuration issues:

```bash
# Show resolved configuration
pb-deployer config show --resolved

# Test configuration against schema
pb-deployer config validate --strict

# Show configuration inheritance
pb-deployer config trace
```

## Getting Help

### Self-Diagnosis

First, try these diagnostic commands:

```bash
# Run system diagnostics
pb-deployer doctor

# Validate everything
pb-deployer validate --all

# Check system requirements
pb-deployer system check
```

### Collecting Debug Information

When reporting issues, include:

```bash
# Generate debug report
pb-deployer debug report

# This creates a report.json with:
# - System information
# - Configuration (sanitized)
# - Recent logs
# - Error traces
```

### Community Support

Still need help? Here's where to get support:

- **GitHub Issues**: [Report bugs and request features](https://github.com/your-username/pb-deployer/issues)
- **Discussions**: [Community Q&A and discussions](https://github.com/your-username/pb-deployer/discussions)
- **Documentation**: Check this documentation for detailed guides
- **Examples**: Browse example configurations in the repository

### Creating a Good Issue Report

When creating an issue, include:

1. **pb-deployer version**: `pb-deployer --version`
2. **System information**: `pb-deployer system info`
3. **Configuration**: Your `pb-deployer.config.js` (remove sensitive data)
4. **Error logs**: Copy the full error message
5. **Steps to reproduce**: Clear steps to recreate the issue

**Issue Template:**
```markdown
**pb-deployer version**: 1.2.3
**OS**: macOS 14.0
**Node.js**: 20.5.0

**Configuration**:
(paste your sanitized config here)

**Error**:
(paste the full error message)

**Steps to reproduce**:
1. Run `pb-deployer init`
2. Configure deployment provider
3. Run `pb-deployer deploy`
4. Error occurs

**Expected behavior**:
Deployment should complete successfully

**Additional context**:
Any other relevant information
```

> **Remember**: Most issues can be resolved by updating to the latest version and validating your configuration!