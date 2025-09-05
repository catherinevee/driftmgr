```
     .___      .__  _____  __                         
   __| _/______|__|/ ____\/  |_  _____    ___________ 
  / __ |\_  __ \  \   __\\   __\/     \  / ___\_  __ \
 / /_/ | |  | \/  ||  |   |  | |  Y Y  \/ /_/  >  | \/
 \____ | |__|  |__||__|   |__| |__|_|  /\___  /|__|   
      \/                             \//_____/        
```

# DriftMgr - Advanced Terraform Drift Detection & Remediation

Enterprise-grade tool for discovering, analyzing, and remediating Terraform drift across multi-cloud environments with intelligent detection modes and automated remediation strategies.

[![Go Version](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go)](https://golang.org)
[![Go Report Card](https://goreportcard.com/badge/github.com/catherinevee/driftmgr)](https://goreportcard.com/report/github.com/catherinevee/driftmgr)
[![Tests](https://img.shields.io/github/actions/workflow/status/catherinevee/driftmgr/ci.yml?branch=main&label=tests)](https://github.com/catherinevee/driftmgr/actions)
[![Coverage](https://img.shields.io/codecov/c/github/catherinevee/driftmgr?token=CODECOV_TOKEN)](https://codecov.io/gh/catherinevee/driftmgr)
[![Release](https://img.shields.io/github/v/release/catherinevee/driftmgr?include_prereleases&sort=semver&color=blue)](https://github.com/catherinevee/driftmgr/releases)
[![License](https://img.shields.io/github/license/catherinevee/driftmgr)](LICENSE)
[![Multi-Cloud](https://img.shields.io/badge/☁️-AWS%20|%20Azure%20|%20GCP%20|%20DO-orange)](https://github.com/catherinevee/driftmgr#supported-providers)
[![Terraform](https://img.shields.io/badge/Terraform-0.11→1.x-7B42BC?logo=terraform)](https://github.com/catherinevee/driftmgr#terraform-compatibility)

## Key Capabilities

DriftMgr provides enterprise-grade infrastructure drift management:

### Core Features
- **Smart Discovery** - Auto-discovers Terraform states across S3, Azure Storage, GCS, and local filesystems
- **Multi-Mode Detection** - Quick (30s), Deep (full analysis), and Smart (adaptive) detection modes
- **Automated Remediation** - Multiple strategies: Code-as-Truth, Cloud-as-Truth, Manual Approval
- **Rich CLI** - Progress indicators, colored output, interactive prompts for better UX
- **Multi-Cloud** - Native support for AWS, Azure, GCP, and DigitalOcean
- **Quality Analytics** - Built-in code quality analysis and user acceptance testing

### Advanced Capabilities
- **Resource Criticality** - Prioritizes critical resources (databases, security groups) in Smart mode
- **Remediation Strategies** - Choose between applying Terraform or updating code to match cloud
- **Import Generation** - Automatically generates terraform import commands for unmanaged resources
- **Terragrunt Support** - Full dependency resolution and HCL parsing
- **State Management** - Push/pull operations with remote backend support
- **Compliance Reporting** - Policy validation and audit trail generation

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
docker pull catherinevee/driftmgr:v3.0.0
docker run -it --rm \
  -v ~/.aws:/root/.aws:ro \
  -v ~/terraform:/terraform:ro \
  catherinevee/driftmgr discover --provider aws
```

## Core Commands

### Discovery & Analysis
```bash
# Discover cloud resources with backend autodiscovery
driftmgr discover

# Analyze state file with dependency graph
driftmgr analyze --state terraform.tfstate

# Detect drift with different modes
driftmgr drift detect --state terraform.tfstate --mode quick    # < 30 seconds
driftmgr drift detect --state terraform.tfstate --mode deep     # Full analysis
driftmgr drift detect --state terraform.tfstate --mode smart    # Adaptive based on criticality

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
# Import unmanaged resources with automatic discovery
driftmgr import --provider aws --resource-type ec2_instance

# Generate remediation plan with different strategies
driftmgr remediate --state terraform.tfstate --strategy code-as-truth  # Apply Terraform
driftmgr remediate --state terraform.tfstate --strategy cloud-as-truth # Update code
driftmgr remediate --state terraform.tfstate --dry-run                # Preview changes

# Interactive remediation with approval workflow
driftmgr remediate --state terraform.tfstate --interactive

# Bulk delete unmanaged resources (with safety checks)
driftmgr bulk-delete --provider aws --filter "tag:Environment=dev"
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

## Detection Modes

DriftMgr offers three intelligent detection modes optimized for different scenarios:

| Mode | Speed | Use Case | Coverage | Resource Checks |
|------|-------|----------|----------|-----------------|
| **Quick** | < 30 seconds | Daily checks, CI/CD pipelines | Resource existence only | Basic validation |
| **Deep** | 2-5 minutes | Scheduled audits, pre-production | Full attribute comparison | Complete analysis |
| **Smart** | Adaptive | Production environments | Critical resources deep, others quick | Prioritized by criticality |

### Smart Mode Criticality Levels
- **Critical**: Databases, security groups, IAM roles (always deep scan)
- **High**: Load balancers, network ACLs, encryption keys (deep scan)
- **Medium**: Compute instances, storage buckets (quick scan unless flagged)
- **Low**: Tags, metadata, non-production resources (quick scan)

## Key Features

### State File Discovery
- Auto-discovers state files across multiple backends
- Supports S3, Azure Storage, GCS, and local filesystem
- Handles state locking (DynamoDB, Azure blob leases)
- Manages state versions and history

### Drift Detection
- Multi-mode detection (Quick, Deep, Smart) for different scenarios
- Resource criticality-based prioritization
- Compares Terraform state with actual cloud resources
- Identifies missing, modified, and unmanaged resources
- Generates detailed drift reports with remediation suggestions
- Supports incremental discovery for large infrastructures

### Terragrunt Support
- Parses terragrunt.hcl configurations
- Resolves module dependencies
- Handles remote_state blocks
- Supports run-all operations

### Remediation Strategies
- **Code-as-Truth**: Apply Terraform to fix drift (infrastructure matches code)
- **Cloud-as-Truth**: Update Terraform code to match cloud (code matches infrastructure)
- **Manual Approval**: Generate plans for review with risk assessment
- **Auto-Rollback**: Automatic rollback on drift detection with backup
- **Hybrid Strategy**: Combines multiple strategies based on resource criticality

### Import Generation
- Automatically generates terraform import commands
- Discovers unmanaged resources with intelligent matching
- Creates resource configurations from cloud state
- Validates imports before execution with dry-run support
- Bulk import with dependency ordering

### Rich CLI Experience
- **Progress Indicators**: Real-time progress bars with ETA
- **Colored Output**: Status-aware colored output for better readability
- **Interactive Prompts**: Safety confirmations for dangerous operations
- **Formatted Tables**: Clean table output for resource lists
- **Tree Visualization**: Hierarchical display of dependencies

### Web Interface
- Real-time resource discovery with WebSocket updates
- Interactive drift visualization with D3.js graphs
- Remediation workflow management with approval chains
- Export reports in JSON, YAML, HTML, CSV formats
- REST API for integration with CI/CD pipelines

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
  catherinevee/driftmgr:v3.0.0 \
  discover --provider aws --region us-east-1
```

### Docker Compose
```yaml
version: '3'
services:
  driftmgr:
    image: catherinevee/driftmgr:v3.0.0
    ports:
      - "8080:8080"
    volumes:
      - ~/.aws:/root/.aws:ro
      - ./terraform:/terraform:ro
    environment:
      - DRIFTMGR_LOG_LEVEL=info
    command: serve web
```

## Quality & Testing

### Code Quality
DriftMgr maintains high code quality standards:
- **Cyclomatic Complexity**: < 10 per function
- **Test Coverage**: > 80% across all packages
- **Documentation**: All exported functions documented
- **Quality Score**: Maintained above 85/100

### Testing
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run UAT tests for specific persona
go test ./tests/uat/journeys -run TestDevOpsEngineer

# Run quality checks
go run quality/cmd/analyze/main.go --project .
```

### CI/CD Pipeline
- **Quality Analysis**: Automated on every PR
- **UAT Testing**: Multi-persona journey tests
- **Security Scanning**: Gosec and vulnerability checks
- **Performance Tests**: Load testing with 50+ concurrent users
- **Auto-improvement**: Weekly automated refactoring PRs

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

### Project Structure (v3.0.0 - Optimized)
```
driftmgr/
├── cmd/
│   ├── driftmgr/         # Main CLI application
│   └── driftmgr-server/  # Server mode
├── internal/
│   ├── providers/        # Cloud provider implementations (AWS, Azure, GCP, DO)
│   ├── state/           # State file management (parser, cache, backup)
│   ├── discovery/       # Resource discovery (parallel, incremental)
│   ├── drift/           # Drift detection with modes
│   │   ├── detector/    # Multi-mode detection engine
│   │   └── comparator/  # Resource comparison logic
│   ├── remediation/     # Remediation strategies
│   │   └── strategies/  # Code-as-Truth, Cloud-as-Truth
│   ├── cli/            # Rich CLI components
│   ├── api/            # Web server and REST API
│   └── shared/         # Common utilities
├── quality/            # Code quality tools
│   ├── analyzer.go     # Quality metrics analyzer
│   ├── gates.go        # Quality gates enforcement
│   └── conciseness.go  # Code conciseness analyzer
├── tests/
│   └── uat/           # User acceptance tests
│       ├── personas.yaml    # User personas
│       └── journeys/       # User journey tests
├── configs/           # Configuration templates
└── web/              # Web UI assets (HTML, JS, CSS)
```

**Key Statistics:**
- **Go Files**: 63 (86% reduction from original)
- **Test Coverage**: > 80%
- **Quality Score**: 85+/100

## Performance

DriftMgr is optimized for speed and efficiency:

### Benchmarks
| Operation | Resources | Time | Memory |
|-----------|-----------|------|--------|
| Quick Scan | 100 | < 5s | 50MB |
| Quick Scan | 1000 | < 30s | 125MB |
| Deep Scan | 100 | < 45s | 75MB |
| Deep Scan | 1000 | < 5min | 200MB |
| Smart Scan | 1000 | < 2min | 150MB |

### Optimization Tips
- Use `--mode quick` for CI/CD pipelines
- Enable `--incremental` for large infrastructures
- Set `--workers` based on CPU cores (default: 10)
- Use `--cache-ttl` to reduce API calls
- Configure resource criticality for Smart mode

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

## What's New in v3.0.0

### Major Enhancements
- **Multi-Mode Detection**: Quick, Deep, and Smart modes for different use cases
- **Remediation Strategies**: Code-as-Truth and Cloud-as-Truth strategies
- **Rich CLI Experience**: Progress bars, colored output, interactive prompts
- **User Acceptance Testing**: Persona-based testing framework
- **Quality Analytics**: Built-in code quality analysis and reporting
- **Performance**: 80% faster with Smart mode, 86% less code

### Improvements
- Resource criticality configuration for prioritized scanning
- Automated import script generation for unmanaged resources
- Enhanced error messages with actionable suggestions
- Comprehensive test coverage (> 80%)
- Weekly automated quality improvements via CI/CD

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