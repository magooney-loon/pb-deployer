# Installation

Get pb-deployer up and running on your system in just a few steps.

## Prerequisites

Before installing pb-deployer, make sure you have:

- **Node.js 18 or higher** - [Download from nodejs.org](https://nodejs.org/)
- **Git** - For version control and repository management
- **Docker** (optional) - For containerized deployments

## Install pb-deployer

### Global Installation (Recommended)

Install pb-deployer globally to use it from anywhere:

```bash
npm install -g pb-deployer
```

Verify the installation:

```bash
pb-deployer --version
```

### Local Installation

For project-specific installations:

```bash
npm install pb-deployer --save-dev
```

Then use with npx:

```bash
npx pb-deployer --version
```

## System Requirements

| Requirement | Minimum | Recommended |
|-------------|---------|-------------|
| Node.js | 18.0.0 | 20.0.0+ |
| RAM | 512MB | 1GB+ |
| Disk Space | 100MB | 500MB+ |

## Platform Support

pb-deployer works on all major platforms:

- **macOS** - Full support
- **Linux** - Full support  
- **Windows** - Full support (PowerShell recommended)

## Verification

After installation, verify everything is working:

```bash
# Check version
pb-deployer --version

# View available commands
pb-deployer --help

# Test configuration
pb-deployer doctor
```

## Update pb-deployer

Keep pb-deployer up to date for the latest features and fixes:

```bash
npm update -g pb-deployer
```

## Troubleshooting Installation

### Permission Issues (macOS/Linux)

If you get permission errors during global installation:

```bash
sudo npm install -g pb-deployer
```

Or configure npm to use a different directory:

```bash
mkdir ~/.npm-global
npm config set prefix '~/.npm-global'
export PATH=~/.npm-global/bin:$PATH
```

### Windows Issues

On Windows, you may need to run PowerShell as Administrator for global installations.

### Clear npm Cache

If you encounter installation issues:

```bash
npm cache clean --force
npm install -g pb-deployer
```

> **Ready for the next step?** Head over to [Configuration](#configuration) to set up your first project!