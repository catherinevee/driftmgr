```
     .___      .__  _____  __                         
   __| _/______|__|/ ____\/  |_  _____    ___________ 
  / __ |\_  __ \  \   __\\   __\/     \  / ___\_  __ \
 / /_/ | |  | \/  ||  |   |  | |  Y Y  \/ /_/  >  | \/
 \____ | |__|  |__||__|   |__| |__|_|  /\___  /|__|   
      \/                             \//_____/        
```

# DriftMgr

Advanced Terraform drift detection and remediation for multi-cloud environments.

[![CI/CD Pipeline](https://github.com/catherinevee/driftmgr/actions/workflows/ci-cd.yml/badge.svg)](https://github.com/catherinevee/driftmgr/actions/workflows/ci-cd.yml)
[![Test Coverage](https://codecov.io/gh/catherinevee/driftmgr/branch/main/graph/badge.svg)](https://codecov.io/gh/catherinevee/driftmgr)
[![Security Scan](https://github.com/catherinevee/driftmgr/actions/workflows/security-compliance.yml/badge.svg)](https://github.com/catherinevee/driftmgr/actions/workflows/security-compliance.yml)
[![Go Format Check](https://github.com/catherinevee/driftmgr/actions/workflows/gofmt.yml/badge.svg)](https://github.com/catherinevee/driftmgr/actions/workflows/gofmt.yml)
[![Go Linting](https://github.com/catherinevee/driftmgr/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/catherinevee/driftmgr/actions/workflows/golangci-lint.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/catherinevee/driftmgr)](https://goreportcard.com/report/github.com/catherinevee/driftmgr)

## Table of Contents

- [Why DriftMgr](#why-driftmgr)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Core Concepts](#core-concepts)
- [Usage Guide](#usage-guide)
- [Configuration](#configuration)
- [Cloud Providers](#cloud-providers)
- [Advanced Features](#advanced-features)
- [API Reference](#api-reference)
- [Troubleshooting](#troubleshooting)

## Why DriftMgr

### The Problem

Infrastructure drift occurs when actual cloud resources diverge from Terraform state:
- Manual changes bypass version control
- Emergency fixes create undocumented modifications  
- Multiple teams cause configuration conflicts
- Untracked resources increase costs and security risks

### The Solution

DriftMgr provides automated drift detection and remediation with:
- **30-second quick scans** for CI/CD pipelines
- **Smart detection** that prioritizes critical resources
- **Automated remediation** with multiple strategies
- **Multi-cloud support** across AWS, Azure, GCP, and DigitalOcean

## Quick Start

Get drift detection running in under 2 minutes:

```bash
# Install
go install github.com/catherinevee/driftmgr/cmd/driftmgr@latest

# Detect drift in current directory
driftmgr drift detect --state terraform.tfstate

# Start web interface
driftmgr serve web --port 8080
```

### Example Output

```
Drift Detection Summary
----------------------
Resources Scanned: 47
Drift Detected: 3

MODIFIED: aws_security_group.web (critical)
  - ingress rule added outside Terraform
  
MISSING: aws_s3_bucket.logs
  - Resource deleted but exists in state

UNMANAGED: aws_ec2_instance.temp-debug
  - Resource created outside Terraform
```

## Installation

### Binary Installation

```bash
# macOS/Linux
curl -L https://github.com/catherinevee/driftmgr/releases/latest/download/driftmgr-$(uname -s)-$(uname -m) -o driftmgr
chmod +x driftmgr
sudo mv driftmgr /usr/local/bin/

# Windows
# Download from releases page
```

### From Source

```bash
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr
go build -o driftmgr ./cmd/driftmgr
```

### Docker

```bash
docker pull catherinevee/driftmgr:latest
docker run -v ~/.aws:/root/.aws:ro catherinevee/driftmgr discover --provider aws
```

## Core Concepts

### Detection Modes

| Mode | Duration | Use Case | What It Checks |
|------|----------|----------|----------------|
| **Quick** | <30s | CI/CD pipelines | Resource existence |
| **Deep** | 2-5min | Scheduled audits | All attributes |
| **Smart** | Adaptive | Production | Critical resources deep, others quick |

### Resource Criticality

DriftMgr automatically prioritizes resources:

- **Critical**: Databases, security groups, IAM roles
- **High**: Load balancers, encryption keys
- **Medium**: Compute instances, storage
- **Low**: Tags, metadata

### Remediation Strategies

| Strategy | Action | Use Case |
|----------|--------|----------|
| **Code-as-Truth** | Apply Terraform | Enforce desired state |
| **Cloud-as-Truth** | Update code | Accept cloud changes |
| **Manual** | Generate plan | Review before applying |

## Usage Guide

### Drift Detection

Basic drift detection:
```bash
# Quick scan (30 seconds)
driftmgr drift detect --state terraform.tfstate --mode quick

# Deep analysis
driftmgr drift detect --state terraform.tfstate --mode deep

# Smart mode (recommended for production)
driftmgr drift detect --state terraform.tfstate --mode smart
```

Filter by provider or resource:
```bash
# AWS only
driftmgr drift detect --state terraform.tfstate --provider aws

# Specific resource types
driftmgr drift detect --state terraform.tfstate --resource-type aws_security_group
```

### Resource Discovery

Discover all resources in your cloud accounts:
```bash
# Auto-discover across all configured providers
driftmgr discover

# Specific provider and region
driftmgr discover --provider aws --region us-east-1

# With filters
driftmgr discover --provider azure --filter "tag:Environment=production"
```

### State Management

Work with Terraform state files:
```bash
# Analyze state file
driftmgr analyze --state terraform.tfstate

# Pull from remote backend
driftmgr state pull s3 terraform.tfstate --bucket my-states --key prod.tfstate

# Push to remote backend  
driftmgr state push terraform.tfstate s3 --bucket my-states --key prod.tfstate

# List remote states
driftmgr state list --backend s3 --bucket my-states
```

### Remediation

Fix detected drift:
```bash
# Preview changes (dry run)
driftmgr remediate --state terraform.tfstate --dry-run

# Apply Terraform (code-as-truth)
driftmgr remediate --state terraform.tfstate --strategy code-as-truth

# Update code to match cloud (cloud-as-truth)
driftmgr remediate --state terraform.tfstate --strategy cloud-as-truth

# Interactive mode with approval
driftmgr remediate --state terraform.tfstate --interactive
```

### Import Resources

Import unmanaged resources:
```bash
# Auto-discover and generate imports
driftmgr import --provider aws --auto-discover

# Import specific resource type
driftmgr import --provider aws --resource-type aws_s3_bucket

# Bulk import from file
driftmgr import --from-file unmanaged-resources.json
```

## Configuration

### Configuration File

Create `driftmgr.yaml`:

```yaml
# Provider settings
providers:
  aws:
    regions: [us-east-1, us-west-2]
    profile: production
  azure:
    subscription_id: ${AZURE_SUBSCRIPTION_ID}
  gcp:
    project_id: ${GCP_PROJECT_ID}

# Detection settings  
detection:
  mode: smart
  workers: 10
  timeout: 5m

# State discovery
state_discovery:
  backends:
    s3:
      buckets: [terraform-states]
    azurerm:
      storage_accounts: [tfstates]

# Remediation
remediation:
  dry_run: true
  require_approval: true
  backup_state: true
```

### Environment Variables

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

# DriftMgr
export DRIFTMGR_LOG_LEVEL=info
export DRIFTMGR_WORKERS=10
```

## Cloud Providers

### AWS

Authentication methods:
- IAM roles (recommended for EC2)
- AWS credentials file
- Environment variables
- AssumeRole with MFA

Supported resources: EC2, VPC, S3, RDS, IAM, Lambda, ECS, EKS

### Azure

Authentication methods:
- Service principal
- Managed identity
- Azure CLI

Supported resources: VMs, VNets, Storage, SQL, AKS, Key Vault

### GCP

Authentication methods:
- Service account JSON
- Application default credentials
- Workload identity

Supported resources: Compute, Networks, Storage, CloudSQL, GKE

### DigitalOcean

Authentication methods:
- API token

Supported resources: Droplets, Volumes, Load Balancers, Databases

## Advanced Features

### Web Interface

Start the web server:
```bash
# Basic
driftmgr serve web

# With authentication
driftmgr serve web --auth --jwt-secret $SECRET

# Custom port
driftmgr serve web --port 9090
```

Access at `http://localhost:8080`

Features:
- Real-time drift detection
- Interactive resource explorer
- Visual dependency graphs
- Remediation workflows
- Export reports (JSON, CSV, HTML)

### API Server

Start API server:
```bash
driftmgr serve api --port 8081
```

Endpoints:
- `POST /api/discover` - Trigger discovery
- `GET /api/drift` - Get drift results
- `POST /api/remediate` - Execute remediation
- `GET /api/resources` - List resources
- `GET /api/health` - Health check

### Terragrunt Support

```bash
# Analyze Terragrunt project
driftmgr terragrunt analyze --path ./infrastructure

# Detect drift in Terragrunt modules
driftmgr terragrunt drift --path ./infrastructure

# Run-all operations
driftmgr terragrunt run-all plan --path ./infrastructure
```

### Continuous Monitoring

```bash
# Start monitoring daemon
driftmgr monitor start --interval 5m

# With webhook notifications
driftmgr monitor start --webhook https://slack.webhook.url

# Status
driftmgr monitor status
```

### Compliance & Reporting

```bash
# Generate compliance report
driftmgr compliance report --standard cis-aws

# Policy validation
driftmgr policy validate --policy-file policies.rego

# Audit trail
driftmgr audit export --format json --from 2024-01-01
```

## API Reference

### CLI Commands

```
driftmgr
├── discover        # Resource discovery
├── drift          
│   ├── detect      # Detect drift
│   └── report      # Generate reports
├── analyze         # Analyze state files
├── remediate       # Fix drift
├── import          # Import resources
├── state          
│   ├── pull        # Pull from backend
│   ├── push        # Push to backend
│   └── list        # List states
├── serve          
│   ├── web         # Web interface
│   └── api         # API server
├── monitor         # Continuous monitoring
├── compliance      # Compliance checks
└── terragrunt      # Terragrunt operations
```

### Go SDK

```go
import "github.com/catherinevee/driftmgr/pkg/drift"

// Create detector
detector := drift.NewDetector(drift.Config{
    Mode: drift.ModeSmart,
    Workers: 10,
})

// Detect drift
results, err := detector.Detect(ctx, stateFile)
```

## Troubleshooting

### Common Issues

**No credentials found**
```bash
# Check AWS credentials
aws sts get-caller-identity

# Set profile
export AWS_PROFILE=your-profile
```

**State file locked**
```bash
# Force unlock (use carefully)
driftmgr state unlock --force
```

**Timeout errors**
```bash
# Increase timeout
driftmgr drift detect --timeout 10m

# Reduce workers for rate limits
driftmgr discover --workers 5
```

**Memory issues with large states**
```bash
# Use streaming mode
driftmgr analyze --state terraform.tfstate --stream

# Increase memory limit
export GOGC=50
```

### Debug Mode

```bash
# Verbose logging
DRIFTMGR_LOG_LEVEL=debug driftmgr drift detect

# Trace HTTP requests  
DRIFTMGR_LOG_LEVEL=trace driftmgr discover

# Save debug output
driftmgr drift detect --debug 2> debug.log
```

## Performance Tuning

### Large Infrastructures

For environments with 1000+ resources:

```yaml
# driftmgr.yaml
performance:
  workers: 20
  batch_size: 200
  cache_ttl: 10m
  stream_mode: true
  
detection:
  incremental: true
  bloom_filter: true
```

### CI/CD Integration

```yaml
# .github/workflows/drift.yml
- name: Drift Detection
  run: |
    driftmgr drift detect \
      --state terraform.tfstate \
      --mode quick \
      --output json > drift.json
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md)

## License

MIT License - see [LICENSE](LICENSE)

## Support

- Documentation: [docs.driftmgr.io](https://docs.driftmgr.io)
- Issues: [GitHub Issues](https://github.com/catherinevee/driftmgr/issues)
- Discussions: [GitHub Discussions](https://github.com/catherinevee/driftmgr/discussions)# Trigger workflows
