# DriftMgr Installation Guide

## Quick Start

### Build from Source (Currently Required)

Since DriftMgr is in active development, you'll need to build from source:

#### Windows
```powershell
# Clone repository
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr

# Build binary
go build -o driftmgr.exe ./cmd/driftmgr

# Run
./driftmgr.exe --help
```

#### Linux/macOS
```bash
# Clone repository
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr

# Build binary
go build -o driftmgr ./cmd/driftmgr

# Run
./driftmgr --help
```

## Installation Methods

### Method 1: Docker (Build Locally)
```bash
# Build Docker image
docker build -t catherinevee/driftmgr:latest .

# Run with Docker
docker run --rm -v ~/.aws:/root/.aws catherinevee/driftmgr:latest discover --provider aws

# Using Docker Compose
docker-compose up -d
```

### Method 2: Binary Installation

#### Build and Install Binary
After building from source, you can install the binary system-wide:

#### Windows Binary
```powershell
# Create directory
mkdir %LOCALAPPDATA%\DriftMgr

# Copy binary
copy driftmgr.exe %LOCALAPPDATA%\DriftMgr\

# Add to PATH (requires restart)
setx PATH "%PATH%;%LOCALAPPDATA%\DriftMgr"
```

#### Linux/macOS Binary
```bash
# Copy to local bin
sudo cp driftmgr /usr/local/bin/
sudo chmod +x /usr/local/bin/driftmgr

# Or user directory
mkdir -p ~/.local/bin
cp driftmgr ~/.local/bin/
chmod +x ~/.local/bin/driftmgr
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
```

### Method 3: Using Go Install

#### Prerequisites
- Go 1.23+ installed

#### Install Steps
```bash
# Install directly with Go
go install github.com/catherinevee/driftmgr/cmd/driftmgr@latest

# The binary will be in $GOPATH/bin or $HOME/go/bin
driftmgr --help

# Install
sudo mv driftmgr /usr/local/bin/
```

#### Windows Build
```powershell
# Clone repository
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr

# Build binary
go build -o driftmgr.exe ./cmd/driftmgr

# Add to PATH
mkdir %LOCALAPPDATA%\DriftMgr
move driftmgr.exe %LOCALAPPDATA%\DriftMgr\
```

## Configuration

### Initial Setup
```bash
# Run setup wizard
driftmgr setup

# Or manually create config
driftmgr config init
```

### Configuration File
Create `~/.driftmgr/config.yaml`:
```yaml
providers:
  aws:
    enabled: true
    regions: [us-east-1, us-west-2]
  azure:
    enabled: true
    subscriptions: [all]
  gcp:
    enabled: true
    projects: [all]

drift:
  smart_defaults: true
  environment: production
```

### Cloud Credentials

#### AWS
```bash
# Option 1: Environment variables
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret

# Option 2: AWS CLI configuration
aws configure

# Option 3: IAM Role (EC2/ECS)
# Automatic, no configuration needed
```

#### Azure
```bash
# Option 1: Azure CLI
az login

# Option 2: Service Principal
export AZURE_CLIENT_ID=your-client-id
export AZURE_CLIENT_SECRET=your-secret
export AZURE_TENANT_ID=your-tenant
export AZURE_SUBSCRIPTION_ID=your-subscription
```

#### GCP
```bash
# Option 1: gcloud CLI
gcloud auth application-default login

# Option 2: Service Account
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json
```

#### DigitalOcean
```bash
export DIGITALOCEAN_TOKEN=your-api-token
```

## Verification

### Check Installation
```bash
# Version check
driftmgr --version

# Health check
driftmgr health

# Credential verification
driftmgr credentials
```

### Test Discovery
```bash
# Quick test
driftmgr discover --provider aws --limit 10

# Full test
driftmgr status
```

## System Requirements

### Minimum Requirements
- **OS**: Windows 10+, macOS 10.15+, Linux (kernel 3.10+)
- **RAM**: 2GB
- **Disk**: 500MB free space
- **Network**: Internet connection for cloud API access

### Recommended Requirements
- **RAM**: 4GB+ for large infrastructures
- **CPU**: 2+ cores for parallel processing
- **Disk**: 2GB+ for caching and logs

### Docker Requirements
- Docker 20.10+
- Docker Compose 1.29+ (optional)
- 4GB RAM allocated to Docker

## Platform-Specific Notes

### Windows
- PowerShell 5.1+ required for installer
- May need to enable script execution: `Set-ExecutionPolicy RemoteSigned`
- Restart terminal after PATH changes

### macOS
- May need to allow app in Security settings
- Homebrew installation coming soon
- M1/M2 chips fully supported

### Linux
- Works on all major distributions
- Snap and APT packages coming soon
- SystemD service file available

## Troubleshooting

### Common Issues

#### Permission Denied
```bash
# Linux/macOS
chmod +x driftmgr
sudo chown $USER:$USER driftmgr
```

#### Command Not Found
```bash
# Check PATH
echo $PATH

# Add to PATH
export PATH=$PATH:/path/to/driftmgr
```

#### Cannot Connect to Cloud
```bash
# Check credentials
driftmgr credentials

# Test connectivity
driftmgr health --verbose
```

### Installation Logs
```bash
# View installation log
cat ~/.driftmgr/install.log

# Windows
type %LOCALAPPDATA%\DriftMgr\install.log
```

## Uninstallation

### Docker
```bash
docker-compose down -v
docker rmi driftmgr:latest
```

### Binary
```bash
# Linux/macOS
sudo rm /usr/local/bin/driftmgr
rm -rf ~/.driftmgr

# Windows
rmdir /s %LOCALAPPDATA%\DriftMgr
```

### Remove Configuration
```bash
rm -rf ~/.driftmgr
rm -rf ~/.config/driftmgr
```

## Advanced Installation

### Custom Installation Path
```bash
# Linux/macOS
PREFIX=/opt/driftmgr ./install.sh

# Windows
.\install.ps1 -InstallPath "C:\Program Files\DriftMgr"
```

### Silent Installation
```bash
# Linux/macOS
./install.sh --silent --skip-credentials

# Windows
.\install.ps1 -Silent -SkipCredentials
```

### Corporate Deployment
```bash
# Mass deployment script
for host in $(cat hosts.txt); do
  ssh $host 'curl -sSL https://install.driftmgr.io | bash'
done
```

## Next Steps

After installation:
1. [Configure cloud credentials](./docs/configuration.md)
2. [Run your first scan](./docs/quick-start.md)
3. [Set up the dashboard](./docs/dashboard.md)
4. [Configure auto-remediation](./docs/auto-remediation.md)

## Support

- **Documentation**: https://docs.driftmgr.io
- **Issues**: https://github.com/catherinevee/driftmgr/issues
- **Community**: https://discord.gg/driftmgr

---

*This document consolidates all installation-related documentation for DriftMgr.*