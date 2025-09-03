```
     .___      .__  _____  __                         
   __| _/______|__|/ ____\/  |_  _____    ___________ 
  / __ |\_  __ \  \   __\\   __\/     \  / ___\_  __ \
 / /_/ | |  | \/  ||  |   |  | |  Y Y  \/ /_/  >  | \/
 \____ | |__|  |__||__|   |__| |__|_|  /\___  /|__|   
      \/                             \//_____/        
```

# DriftMgr - Terraform State Intelligence Platform

<div align="center">

[![Go Report Card](https://goreportcard.com/badge/github.com/catherinevee/driftmgr)](https://goreportcard.com/report/github.com/catherinevee/driftmgr)
[![Release](https://img.shields.io/github/v/release/catherinevee/driftmgr?include_prereleases&sort=semver&color=blue)](https://github.com/catherinevee/driftmgr/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/catherinevee/driftmgr)](https://hub.docker.com/r/catherinevee/driftmgr)
[![Multi-Cloud](https://img.shields.io/badge/☁️-AWS%20|%20Azure%20|%20GCP%20|%20DO-orange)](https://github.com/catherinevee/driftmgr#supported-providers)
[![Terraform](https://img.shields.io/badge/Terraform-0.11→1.x-7B42BC?logo=terraform)](https://github.com/catherinevee/driftmgr#terraform-compatibility)

**Enterprise-Grade Terraform State Intelligence & Drift Detection Platform**

[Quick Start](#quick-start) • [Features](#features) • [Documentation](docs/) • [Contributing](#contributing)

</div>

---

## Overview

**DriftMgr** is an advanced Terraform State Intelligence Platform that provides deep insights into your infrastructure-as-code deployments. It automatically discovers and analyzes Terraform state files across your organization, visualizes infrastructure relationships, identifies out-of-band changes, and provides intelligent remediation capabilities.

### Why DriftMgr?

- 🔍 **State-Centric Intelligence** - Automatic discovery and analysis of Terraform/Terragrunt state files
- 🌐 **Multi-Cloud Support** - Unified management for AWS, Azure, GCP, and DigitalOcean
- 📊 **Interactive Visualizations** - State galaxy view, dependency graphs, and coverage analytics
- 🎯 **Intelligent Drift Detection** - Context-aware drift analysis with 75-85% noise reduction
- 🔧 **Automated Remediation** - Safe, rollback-enabled fixes with approval workflows
- 🚀 **Out-of-Band Detection** - Identify resources created outside of Terraform
- 📈 **v3.0 Architecture** - 86% code reduction while maintaining all functionality

## Table of Contents

- [Quick Start](#quick-start)
- [Installation](#installation)
- [Features](#features)
- [Command Reference](#command-reference)
- [Web Interface](#web-interface)
- [Configuration](#configuration)
- [Architecture](#architecture)
- [API Reference](#api-reference)
- [Docker Deployment](#docker-deployment)
- [Production Features](#production-features)
- [Security](#security)
- [Performance](#performance)
- [Development](#development)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## Quick Start

Get DriftMgr running in under 2 minutes:

```bash
# Clone and build
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr
go build -o driftmgr ./cmd/driftmgr

# Start the web interface
./driftmgr serve web --port 8080

# Open your browser
open http://localhost:8080
```

That's it! DriftMgr will automatically detect your cloud credentials and begin discovering resources.

## Installation

### Prerequisites

- Go 1.21+ (for building from source)
- Cloud provider credentials configured (AWS, Azure, GCP, or DigitalOcean)
- Terraform state files to analyze

### Install from Source

```bash
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr
go build -o driftmgr ./cmd/driftmgr
sudo mv driftmgr /usr/local/bin/
```

### Install with Docker

```bash
docker pull catherinevee/driftmgr:latest
docker run -it --rm \
  -p 8080:8080 \
  -v ~/.aws:/root/.aws:ro \
  -v ~/terraform:/terraform:ro \
  catherinevee/driftmgr serve web
```

### Install with Homebrew (macOS)

```bash
# Coming soon
brew tap catherinevee/driftmgr
brew install driftmgr
```

## Features

### Core Capabilities

#### 🔍 State Discovery & Management
- **Automatic Discovery** - Finds state files across local, cloud, and VCS
- **State Push/Pull** - Sync states with remote backends (S3, Azure Storage, GCS)
- **Health Monitoring** - Proactive alerts for stale or oversized states
- **Version Tracking** - State file history and rollback capabilities

#### 🛡️ Governance & Compliance
- **Policy Enforcement** - OPA integration for governance rules
- **Compliance Reporting** - SOC2, HIPAA, PCI-DSS templates
- **Audit Logging** - Complete trail with retention policies
- **RBAC Support** - Role-based access control

#### 📊 Visualization & Analytics
- **State Galaxy View** - 3D force-directed graph visualization
- **Dependency Graphs** - Interactive resource relationships
- **Tree Maps** - Hierarchical views by resource count
- **Timeline Views** - Historical state modifications
- **Coverage Analytics** - Infrastructure coverage metrics

#### 🔧 Drift Detection & Remediation
- **Intelligent Filtering** - 75-85% noise reduction
- **Real-time Monitoring** - Webhooks and adaptive polling
- **Automated Remediation** - Safe fixes with approval workflows
- **Import Generation** - Automatic terraform import commands
- **Cost Impact Analysis** - Understand financial implications

### v3.0 Enhanced Features

- **Real-time Cloud Events** - AWS EventBridge, Azure Event Grid, GCP Pub/Sub
- **Incremental Discovery** - Bloom filters for efficient change detection
- **Multi-Level Caching** - Memory → Disk → Remote hierarchy
- **Enhanced Error Recovery** - Automatic retry with circuit breakers
- **Platform Optimizations** - Windows file unlock, Unix flock support

## Command Reference

### Essential Commands

```bash
# Start web interface
driftmgr serve web --port 8080

# Discover cloud resources
driftmgr discover --provider aws --region us-west-2

# Detect drift
driftmgr drift detect --state terraform.tfstate

# Analyze state files
driftmgr analyze --state terraform.tfstate

# Import unmanaged resources
driftmgr import --provider aws --resource-type ec2_instance
```

### State Management Commands

```bash
# Push/Pull state to/from remote backends
driftmgr state push terraform.tfstate s3 --bucket=my-state --key=prod.tfstate
driftmgr state pull s3 terraform.tfstate --bucket=my-state --key=prod.tfstate

# Backup management
driftmgr backup create --state terraform.tfstate
driftmgr backup list
driftmgr backup restore --id backup-123
```

### Advanced Commands

```bash
# Policy enforcement
driftmgr policy evaluate --state terraform.tfstate --package terraform.governance

# Compliance reporting
driftmgr compliance report --type soc2 --output report.html

# Continuous monitoring
driftmgr monitor start --enable-webhooks --port 8181

# Cost analysis
driftmgr cost-drift --state terraform.tfstate

# Terragrunt support
driftmgr terragrunt analyze --path ./infrastructure
```

## Web Interface

The web interface provides a comprehensive dashboard for managing your infrastructure:

### Key Pages

1. **Dashboard** - Overview with health metrics and statistics
2. **State Discovery** - Auto-detect and catalog state files
3. **Resources** - Browse all discovered cloud resources
4. **Drift Detection** - Identify configuration drift
5. **Remediation** - Fix drift with safety checks
6. **Visualizations** - Interactive graphs and charts

### Interactive Features

- **Split-Screen Views** - Compare state expectations vs cloud reality
- **Real-time Updates** - WebSocket-powered live data
- **Import Wizards** - Guided resource adoption workflows
- **Export Options** - Download reports in multiple formats

### Starting the Web Interface

```bash
# Default port 8080
driftmgr serve web

# Custom port
driftmgr serve web --port 9090

# With authentication
driftmgr serve web --auth --jwt-secret=$SECRET
```

## Configuration

### Configuration File

Create `driftmgr.yaml` in your working directory:

```yaml
# Basic Configuration
app:
  name: driftmgr
  environment: production
  log_level: info

# Provider Settings
providers:
  aws:
    enabled: true
    regions: [us-west-2, us-east-1]
    rate_limit: 20
  azure:
    enabled: true
    subscriptions: [production, staging]
  gcp:
    enabled: true
    projects: [my-project-123]

# State Discovery
state_discovery:
  auto_scan: true
  scan_interval: 5m
  scan_paths: [/terraform, ~/infrastructure]
  backends:
    s3:
      buckets: [terraform-states]
    azurerm:
      storage_accounts: [tfstatestorage]
    gcs:
      buckets: [terraform-state-bucket]

# Performance
performance:
  cache_ttl: 5m
  workers: 10
  max_connections: 100

# Security
security:
  enable_auth: true
  session_timeout: 24h
  audit_log: /var/log/driftmgr/
```

### Environment Variables

```bash
export DRIFTMGR_LOG_LEVEL=debug
export DRIFTMGR_AUTO_DISCOVER=true
export DRIFTMGR_WORKERS=10
export DRIFTMGR_CACHE_TTL=10m

# AWS
export AWS_PROFILE=default
export AWS_REGION=us-west-2

# Azure
export AZURE_SUBSCRIPTION_ID=xxx
export AZURE_TENANT_ID=xxx

# GCP
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json
```

## Architecture

### v3.0 Consolidated Architecture

DriftMgr v3.0 features a massively simplified architecture:

- **63 Go files** (reduced from 447) - Single source of truth
- **43 directories** (reduced from 186) - Clean module structure
- **Zero duplicates** - Eliminated redundant implementations
- **Production-ready** - No stubs or mocks

### Project Structure

```
driftmgr/
├── cmd/                    # Application entry points
│   ├── driftmgr/          # CLI application
│   └── driftmgr-server/   # Server mode
├── internal/              # Private application code
│   ├── providers/         # Cloud provider implementations
│   ├── discovery/         # Resource discovery logic
│   ├── state/            # State file management
│   ├── drift/            # Drift detection engine
│   ├── remediation/      # Remediation planning
│   ├── analysis/         # Cost and dependency analysis
│   ├── api/              # REST and WebSocket APIs
│   ├── monitoring/       # Real-time monitoring
│   ├── safety/           # Backup and compliance
│   └── terragrunt/       # Terragrunt support
├── pkg/                  # Public packages
│   └── models/           # Shared data models
├── web/                  # Web UI assets
├── configs/              # Configuration files
├── examples/             # Example files
└── docs/                 # Documentation
```

### Data Flow

1. **Discovery** → Cloud providers → Resource aggregation → Cache → Storage
2. **Drift Detection** → State parsing → Cloud comparison → Analysis → Reporting
3. **Remediation** → Plan generation → Safety checks → Execution → Verification
4. **Monitoring** → Event ingestion → Processing → Alerting → Dashboard updates

## API Reference

### REST API

Base URL: `http://localhost:8080/api/v1`

#### Key Endpoints

```bash
# State Discovery
POST   /state/discovery/start     # Start discovery
GET    /state/discovery/status    # Get status
GET    /state/files              # List state files

# Resources
GET    /resources                # List all resources
GET    /resources/{id}           # Get resource details
POST   /resources/import         # Import resources

# Drift Detection
POST   /drift/detect             # Run drift detection
GET    /drift/report/{id}        # Get drift report

# Remediation
POST   /remediation/plan         # Create plan
POST   /remediation/apply        # Apply fixes
```

### WebSocket API

Real-time updates via WebSocket:

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Event:', data.type, data.payload);
};
```

## Docker Deployment

### Quick Start

```bash
docker run -d \
  --name driftmgr \
  -p 8080:8080 \
  -v ~/.aws:/root/.aws:ro \
  catherinevee/driftmgr:latest
```

### Docker Compose

```yaml
version: '3.8'
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
      - DRIFTMGR_AUTO_DISCOVER=true
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: driftmgr
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: driftmgr
        image: catherinevee/driftmgr:latest
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health/live
        readinessProbe:
          httpGet:
            path: /health/ready
```

## Production Features

### Enterprise Capabilities

- **High Availability** - Multi-instance deployment support
- **Observability** - Metrics, tracing, and structured logging
- **Security** - Encryption, audit logging, RBAC
- **Compliance** - SOC2, HIPAA, PCI-DSS templates
- **Resilience** - Circuit breakers, rate limiting, graceful degradation

### Monitoring & Health

```bash
# Health endpoints
curl http://localhost:8080/health
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready

# Metrics (Prometheus format)
curl http://localhost:8080/metrics

# Cache statistics
curl http://localhost:8080/api/v1/cache/stats
```

## Security

### Best Practices

1. **Use Read-Only Credentials** for discovery operations
2. **Enable TLS** for production deployments
3. **Configure RBAC** with appropriate roles
4. **Enable Audit Logging** for compliance
5. **Use Secrets Management** (Vault, AWS Secrets Manager)

### Security Configuration

```bash
# Enable authentication
driftmgr serve web --auth --jwt-secret=$SECRET

# Use TLS
driftmgr serve web --tls-cert cert.pem --tls-key key.pem

# Restrict network access
driftmgr serve web --bind 127.0.0.1

# Enable audit logging
driftmgr serve web --audit-log /var/log/driftmgr/
```

## Performance

### Optimization Features

- **Smart Caching** - Multi-strategy caching (LRU, LFU, ARC, Predictive)
- **Incremental Discovery** - Only process changes
- **Parallel Processing** - Concurrent resource discovery
- **Connection Pooling** - Efficient API usage
- **Compression** - Automatic data compression

### Scaling Guidelines

| Component | Limit | Optimization |
|-----------|-------|--------------|
| State Files | 10,000+ | Pagination, filtering |
| Resources/State | 5,000+ | Incremental loading |
| Concurrent Users | 100+ | WebSocket pooling |
| API Requests | 1,000/sec | Rate limiting |

## Development

### Setup

```bash
# Clone repository
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o driftmgr ./cmd/driftmgr

# Run locally
./driftmgr serve web
```

### Testing

```bash
# Unit tests
go test ./...

# Integration tests
go test ./tests/integration -tags=integration

# Coverage
go test -cover ./...

# Race detection
go test -race ./...
```

### Building

```bash
# Production build
go build -ldflags="-s -w" -o driftmgr ./cmd/driftmgr

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o driftmgr-linux
GOOS=darwin GOARCH=amd64 go build -o driftmgr-mac
GOOS=windows GOARCH=amd64 go build -o driftmgr.exe

# Docker build
docker build -t driftmgr:latest .
```

## Troubleshooting

### Common Issues

#### State Files Not Discovered
```bash
# Check scan paths
driftmgr state discovery status

# Verify permissions
ls -la /terraform

# Manually trigger scan
curl -X POST http://localhost:8080/api/v1/state/discovery/start
```

#### Drift Detection Issues
```bash
# Verify credentials
driftmgr status

# Check specific provider
driftmgr discover --provider aws --debug

# Force cache refresh
curl -X POST http://localhost:8080/api/v1/cache/clear
```

#### Performance Problems
```bash
# Enable debug logging
export DRIFTMGR_LOG_LEVEL=debug

# Check resource usage
driftmgr debug metrics

# Adjust worker count
export DRIFTMGR_WORKERS=20
```

### Debug Mode

```bash
# Verbose output
driftmgr serve web --debug

# Trace specific components
export DRIFTMGR_TRACE=discovery,drift,cache

# Profile performance
driftmgr serve web --profile
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

### Development Guidelines

- Follow Go best practices
- Add tests for new features
- Update documentation
- Keep commits focused and atomic

## License

DriftMgr is licensed under the MIT License. See [LICENSE](LICENSE) for details.

## Support

- 📚 **Documentation**: [docs/](docs/)
- 🐛 **Issues**: [GitHub Issues](https://github.com/catherinevee/driftmgr/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/catherinevee/driftmgr/discussions)
- 📧 **Email**: support@driftmgr.io

## Acknowledgments

DriftMgr leverages these excellent projects:
- D3.js for visualizations
- Alpine.js for reactive UI
- AWS, Azure, and GCP SDKs
- Terraform state parsing libraries

---

<div align="center">

**Built for Infrastructure Engineers by Infrastructure Engineers**

*Transforming Terraform state management from reactive to proactive*

</div>