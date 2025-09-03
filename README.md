```
     .___      .__  _____  __                         
   __| _/______|__|/ ____\/  |_  _____    ___________ 
  / __ |\_  __ \  \   __\\   __\/     \  / ___\_  __ \
 / /_/ | |  | \/  ||  |   |  | |  Y Y  \/ /_/  >  | \/
 \____ | |__|  |__||__|   |__| |__|_|  /\___  /|__|   
      \/                             \//_____/        
```

# DriftMgr - Terraform State Analyzer

Discover, analyze, and remediate drift in Terraform state files across multiple backends and cloud providers.

[![Go Report Card](https://goreportcard.com/badge/github.com/catherinevee/driftmgr)](https://goreportcard.com/report/github.com/catherinevee/driftmgr)
[![Release](https://img.shields.io/github/v/release/catherinevee/driftmgr?include_prereleases&sort=semver&color=blue)](https://github.com/catherinevee/driftmgr/releases)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fcatherinevee%2Fdriftmgr.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fcatherinevee%2Fdriftmgr?ref=badge_shield)
[![Multi-Cloud](https://img.shields.io/badge/☁️-AWS%20|%20Azure%20|%20GCP%20|%20DO-orange)](https://github.com/catherinevee/driftmgr#supported-providers)
[![Terraform](https://img.shields.io/badge/Terraform-0.11→1.x-7B42BC?logo=terraform)](https://github.com/catherinevee/driftmgr#terraform-compatibility)

## What It Does

DriftMgr automatically:
- **Discovers** Terraform state files in S3, Azure Storage, GCS, and local filesystems
- **Analyzes** state file health, dependencies, and resource relationships
- **Detects** configuration drift between state files and actual cloud resources
- **Generates** remediation plans and terraform import commands for unmanaged resources
- **Supports** Terragrunt configurations with full dependency resolution
- **Manages** state file operations including push/pull to remote backends

## Quick Start

```bash
# Clone and build
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr
go build -o driftmgr ./cmd/driftmgr

# Discover AWS resources
./driftmgr discover --provider aws --region us-east-1

# Analyze a state file
./driftmgr analyze --state terraform.tfstate

# Start web interface
./driftmgr serve web --port 8080
```

## Installation

### From Source
```bash
go install github.com/catherinevee/driftmgr/cmd/driftmgr@latest
```

### Using Docker
```bash
docker pull catherinevee/driftmgr:latest
docker run -it --rm \
  -v ~/.aws:/root/.aws:ro \
  -v ~/terraform:/terraform:ro \
  catherinevee/driftmgr discover --provider aws
```

## Core Commands

### Discovery & Analysis
```bash
# Discover cloud resources
driftmgr discover --provider aws --region us-west-2

# Analyze state file
driftmgr analyze --state terraform.tfstate

# Detect drift
driftmgr drift detect --state terraform.tfstate

# Analyze Terragrunt configurations
driftmgr terragrunt analyze --path ./infrastructure
```

### State Management
```bash
# Push state to remote backend
driftmgr state push terraform.tfstate s3 --bucket=my-states --key=prod.tfstate

# Pull state from remote backend
driftmgr state pull s3 terraform.tfstate --bucket=my-states --key=prod.tfstate

# List states in backend
driftmgr state list --backend s3 --bucket=my-states
```

### Remediation
```bash
# Import unmanaged resources
driftmgr import --provider aws --resource-type ec2_instance

# Generate remediation plan
driftmgr remediate --state terraform.tfstate --dry-run

# Delete unmanaged resources
driftmgr delete --provider aws --resource-id i-1234567890
```

### Web Interface
```bash
# Start web UI (default port 8080)
driftmgr serve web

# Custom port with authentication
driftmgr serve web --port 9090 --auth --jwt-secret=$SECRET
```

## Supported Providers

| Provider | Authentication | Implementation |
|----------|---------------|----------------|
| **AWS** | IAM roles, credentials file, environment vars | AWS SDK v2 |
| **Azure** | Service principal, Azure CLI, managed identity | Native HTTP (no SDK) |
| **GCP** | Service account, application default credentials | OAuth2 (no SDK) |
| **DigitalOcean** | API token | REST API |

## Configuration

Create `driftmgr.yaml` in your working directory:

```yaml
# Provider configuration
providers:
  aws:
    regions: [us-east-1, us-west-2]
    profile: production
  azure:
    subscription_id: ${AZURE_SUBSCRIPTION_ID}
  gcp:
    project_id: ${GCP_PROJECT_ID}

# State discovery settings
state_discovery:
  backends:
    s3:
      buckets: [terraform-states]
      region: us-east-1
    azurerm:
      storage_accounts: [tfstatestorage]
    gcs:
      buckets: [tf-state-bucket]

# Performance tuning
performance:
  workers: 10
  cache_ttl: 5m
  batch_size: 100

# Remediation settings
remediation:
  dry_run: true
  require_approval: true
```

## Environment Variables

```bash
# AWS
export AWS_PROFILE=production
export AWS_REGION=us-east-1

# Azure
export AZURE_SUBSCRIPTION_ID=xxx
export AZURE_TENANT_ID=xxx
export AZURE_CLIENT_ID=xxx
export AZURE_CLIENT_SECRET=xxx

# GCP
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json

# DigitalOcean
export DIGITALOCEAN_TOKEN=xxx

# DriftMgr settings
export DRIFTMGR_LOG_LEVEL=info
export DRIFTMGR_WORKERS=10
```

## Key Features

### State File Discovery
- Auto-discovers state files across multiple backends
- Supports S3, Azure Storage, GCS, and local filesystem
- Handles state locking (DynamoDB, Azure blob leases)
- Manages state versions and history

### Drift Detection
- Compares Terraform state with actual cloud resources
- Identifies missing, modified, and unmanaged resources
- Generates detailed drift reports
- Supports incremental discovery for large infrastructures

### Terragrunt Support
- Parses terragrunt.hcl configurations
- Resolves module dependencies
- Handles remote_state blocks
- Supports run-all operations

### Import Generation
- Automatically generates terraform import commands
- Discovers unmanaged resources
- Creates resource configurations
- Validates imports before execution

### Web Interface
- Real-time resource discovery
- Interactive drift visualization
- Remediation workflow management
- Export reports in JSON, YAML, HTML formats

## API Endpoints

The web server exposes a REST API:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/discover` | POST | Trigger resource discovery |
| `/api/drift` | GET | Get drift detection results |
| `/api/state` | GET/POST | Manage state files |
| `/api/resources` | GET | List discovered resources |
| `/api/remediate` | POST | Execute remediation |
| `/api/health` | GET | Health check |

## Docker Deployment

### Basic Usage
```bash
docker run --rm -it \
  -v ~/.aws:/root/.aws:ro \
  -v ~/terraform:/terraform:ro \
  catherinevee/driftmgr:latest \
  discover --provider aws --region us-east-1
```

### Docker Compose
```yaml
version: '3'
services:
  driftmgr:
    image: catherinevee/driftmgr:latest
    ports:
      - "8080:8080"
    volumes:
      - ~/.aws:/root/.aws:ro
      - ./terraform:/terraform:ro
    environment:
      - DRIFTMGR_LOG_LEVEL=info
    command: serve web
```

## Development

### Building
```bash
# Build for current platform
go build -o driftmgr ./cmd/driftmgr

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -o driftmgr-linux ./cmd/driftmgr

# Run tests
go test ./...

# Run with race detector
go run -race ./cmd/driftmgr discover --provider aws
```

### Project Structure
```
driftmgr/
├── cmd/driftmgr/          # CLI application
├── internal/
│   ├── providers/         # Cloud provider implementations
│   ├── state/            # State file management
│   ├── discovery/        # Resource discovery
│   ├── drift/            # Drift detection
│   ├── remediation/      # Remediation engine
│   └── api/              # Web server and API
├── configs/              # Configuration files
└── web/                  # Web UI assets
```

## Troubleshooting

### Common Issues

**Cannot find AWS credentials**
```bash
# Ensure credentials are configured
aws configure
# Or use environment variables
export AWS_ACCESS_KEY_ID=xxx
export AWS_SECRET_ACCESS_KEY=xxx
```

**State file locked**
```bash
# Force unlock (use with caution)
driftmgr state unlock --backend s3 --lock-id=xxx
```

**Discovery timeout**
```bash
# Increase timeout and reduce batch size
driftmgr discover --timeout 30m --batch-size 50
```

**High memory usage**
```bash
# Enable incremental discovery
driftmgr discover --incremental --cache-dir=/tmp/driftmgr
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) for details.

## Support

- Issues: [GitHub Issues](https://github.com/catherinevee/driftmgr/issues)
- Documentation: [docs/](docs/)
- Docker Hub: [catherinevee/driftmgr](https://hub.docker.com/r/catherinevee/driftmgr)