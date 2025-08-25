```
     .___      .__  _____  __                         
   __| _/______|__|/ ____\/  |_  _____    ___________ 
  / __ |\_  __ \  \   __\\   __\/     \  / ___\_  __ \
 / /_/ | |  | \/  ||  |   |  | |  Y Y  \/ /_/  >  | \/
 \____ | |__|  |__||__|   |__| |__|_|  /\___  /|__|   
      \/                             \//_____/        
```

# DriftMgr - Enterprise Cloud Infrastructure Drift Detection & Management

[![Production Ready](https://img.shields.io/badge/Production-Ready-green.svg)](https://github.com/catherinevee/driftmgr)
[![CI/CD](https://img.shields.io/badge/CI%2FCD-Ready-brightgreen.svg)](https://github.com/catherinevee/driftmgr)
[![Multi-Cloud](https://img.shields.io/badge/Multi--Cloud-AWS%20%7C%20Azure%20%7C%20GCP%20%7C%20DO-blue.svg)](https://github.com/catherinevee/driftmgr)
[![Docker Support](https://img.shields.io/badge/Docker-Supported-blue.svg)](https://github.com/catherinevee/driftmgr)
[![Security](https://img.shields.io/badge/Security-AES--256--GCM-orange.svg)](https://github.com/catherinevee/driftmgr)

## Overview

**DriftMgr** is a production-ready, enterprise-grade cloud infrastructure drift detection and management platform. It provides intelligent monitoring, detection, and automated remediation capabilities across multiple cloud providers with advanced state file management for Terraform users.

### Key Capabilities
- **Intelligent Drift Detection** - Reduces noise by 75-85% while maintaining critical security visibility
- **Multi-Cloud Support** - Unified management for AWS, Azure, GCP, and DigitalOcean
- **Terraform State Management** - Analyze, visualize, and manage Terraform state files
- **Cost Impact Analysis** - Calculate financial impact of drift and remediation
- **Automated Remediation** - Fix drift automatically with safety checks and rollback capabilities
- **Enterprise Features** - Audit logging, RBAC, circuit breakers, health checks, rate limiting

## Table of Contents

- [Quick Start](#quick-start)
- [Installation](#installation)
- [Real-World Examples](#real-world-examples)
- [Managing Terraform State](#managing-terraform-state)
- [Fixing Drift](#fixing-drift)
- [Advanced Features](#advanced-features)
- [Command Reference](#command-reference)
- [Practical Workflows](#practical-workflows)
- [Production Features](#production-features)
- [Configuration](#configuration)
- [Web Dashboard](#web-dashboard)
- [Performance](#performance)
- [Security](#security)
- [Troubleshooting](#troubleshooting)

## Quick Start

### 1. Install DriftMgr

```bash
# Build from source (recommended)
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr
go build -o build/driftmgr ./cmd/driftmgr

# Or use Make
make build

# Windows: Add to PATH
set PATH=%PATH%;%CD%\build

# Linux/macOS: Add to PATH
export PATH=$PATH:$(pwd)/build
```

### 2. Check Cloud Credentials

```bash
$ driftmgr status

     .___      .__  _____  __                         
   __| _/______|__|/ ____\/  |_  _____    ___________ 
  / __ |\_  __ \  \   __\\   __\/     \  / ___\_  __ \
 / /_/ | |  | \/  ||  |   |  | |  Y Y  \/ /_/  >  | \/
 \____ | |__|  |__||__|   |__| |__|_|  /\___  /|__|   
      \/                             \//_____/        

DriftMgr System Status
============================================================

Detected Cloud Credentials:
├─ AWS (detected)
│  ├─ Profile: default (us-west-2)
│  ├─ Account: 123456789012
│  └─ User: admin@company.com
│
├─ Azure (detected) 
│  ├─ Subscription: Production (xxx-xxx-xxx)
│  ├─ Tenant: company.onmicrosoft.com
│  └─ Resources: 145 discovered
│
├─ GCP (detected)
│  ├─ Project: my-project-123
│  └─ Service Account: driftmgr@my-project.iam
│
└─ DigitalOcean (detected)
   └─ Account: team@company.com

Auto-discovering resources... [================----] 80%

Resources Discovered:
- AWS:           234 resources across 3 regions
- Azure:         145 resources across 2 subscriptions  
- GCP:           67 resources in 1 project
- DigitalOcean:  12 droplets and 5 databases

Total: 458 resources discovered in 12.3s
```

## Real-World Examples

### Example 1: Detecting Drift in Production

```bash
$ driftmgr drift detect --provider aws --environment production

Analyzing AWS Production Infrastructure...
============================================================

WARNING: DRIFT DETECTED - 18 changes found (73% noise filtered)

CRITICAL (2)
├─ sg-0abc123def: Security group exposed to 0.0.0.0/0
│  └─ Action: Remove inbound rule for port 22 from 0.0.0.0/0
│
└─ s3-prod-data: Bucket encryption disabled
   └─ Action: Re-enable AES-256 encryption

IMPORTANT (5)
├─ i-0def456abc: Instance type changed (t2.micro -> t2.small)
│  └─ Cost Impact: +$8.76/month
│
├─ rds-prod-db: Backup retention reduced (30 -> 7 days)
│  └─ Compliance: Violates SOC2 requirements
│
├─ alb-frontend: Health check interval modified
├─ asg-workers: Min capacity changed (3 -> 2)
└─ vpc-main: DNS hostnames disabled

INFORMATIONAL (11)
├─ Updated tags on 8 resources
├─ Modified descriptions on 2 security groups
└─ Changed metadata on 1 S3 bucket

Drift Analysis Summary:
- Security Issues:    2 (critical)
- Config Changes:     5 (important)
- Cosmetic Changes:   11 (filtered)
- Estimated Fix Time: 15 minutes
- Cost Impact:        +$8.76/month
```

### Example 2: Multi-Account Discovery

```bash
$ driftmgr discover --all-accounts

Discovering resources across all accounts...
============================================================

Detected Accounts:
├─ AWS: 3 profiles found
│  ├─ Production (123456789012)
│  ├─ Staging (234567890123)
│  └─ Development (345678901234)
│
├─ Azure: 2 subscriptions found
│  ├─ Production-Sub
│  └─ Staging-Sub
│
└─ GCP: 1 project found
   └─ my-project-123

Discovering resources...
[====================] 100% Complete

Discovery Results:
├─ AWS Production:   234 resources (us-west-2, us-east-1)
├─ AWS Staging:      156 resources (us-west-2)
├─ AWS Development:  89 resources (us-west-2)
├─ Azure Production: 145 resources
├─ Azure Staging:    78 resources
└─ GCP Project:      67 resources

Total: 769 resources discovered across all accounts

$ driftmgr export --format json --output discovery-report.json
Exported to: discovery-report.json
```

## Managing Terraform State

### Analyzing Terraform State Files

```bash
$ driftmgr state analyze --file terraform.tfstate

Terraform State Analysis
============================================================

State File: terraform.tfstate
├─ Version:     4
├─ Serial:      156
├─ Lineage:     a1b2c3d4-e5f6-7890-abcd-ef1234567890
└─ Backend:     s3://my-bucket/terraform/prod.tfstate

Resource Summary:
├─ Total Resources:     87
├─ Resource Types:      12
└─ Providers:           3 (aws, kubernetes, helm)
└─ Modules:            4

Resource Breakdown:
├─ AWS Resources (72)
│  ├─ EC2 Instances:        12
│  ├─ Security Groups:      18
│  ├─ Load Balancers:       3
│  ├─ RDS Instances:        2
│  ├─ S3 Buckets:           8
│  └─ Other:                29
│
├─ Kubernetes Resources (10)
│  ├─ Deployments:          4
│  ├─ Services:             4
│  └─ ConfigMaps:           2
│
└─ Helm Releases (5)
   ├─ nginx-ingress
   ├─ prometheus
   ├─ grafana
   ├─ elasticsearch
   └─ kibana

Potential Issues Found:
├─ 3 resources marked for recreation
├─ 5 resources with lifecycle prevent_destroy
└─ 2 resources with ignore_changes rules
```

### Analyzing State Files

```bash
$ driftmgr state analyze --file terraform.tfstate --detailed

Analyzing Terraform State File...
============================================================

State Analysis Results:

State Metadata:
├─ Version:     4
├─ Serial:      156
├─ Lineage:     a1b2c3d4-e5f6-7890-abcd-ef1234567890
└─ Backend:     s3://my-bucket/terraform/prod.tfstate

Resource Statistics:
├─ Total Resources:     87
├─ Resource Types:      12
├─ Providers Used:      3 (aws, kubernetes, helm)
└─ Modules:            4

Resource Breakdown by Type:
├─ aws_instance:              12
├─ aws_security_group:        18
├─ aws_security_group_rule:   24
├─ aws_s3_bucket:            8
├─ aws_db_instance:          2
├─ aws_alb:                  3
├─ kubernetes_deployment:     4
├─ kubernetes_service:        4
├─ kubernetes_config_map:     2
├─ helm_release:             5
└─ Other:                    5

State Health Check:
├─ Resources with lifecycle rules: 5
├─ Resources marked for recreation: 3
├─ Resources with ignore_changes: 2
└─ Orphaned resources: 0

Recommendations:
- Review resources marked for recreation
- Consider removing lifecycle prevent_destroy for non-critical resources
- Update ignore_changes rules to match current requirements
```

### Visualizing State Dependencies

```bash
$ driftmgr state visualize --file terraform.tfstate

Generating State Dependency Graph...
============================================================

Resource Dependencies:
├─ VPC (vpc-main)
│  ├─ Subnet (subnet-public-1a)
│  │  ├─ EC2 Instance (web-server-1)
│  │  └─ NAT Gateway (nat-1a)
│  ├─ Subnet (subnet-public-1b)
│  │  └─ EC2 Instance (web-server-2)
│  ├─ Internet Gateway (igw-main)
│  └─ Security Group (sg-web)
│     ├─ EC2 Instance (web-server-1)
│     └─ EC2 Instance (web-server-2)
│
├─ RDS Instance (database-primary)
│  ├─ Security Group (sg-database)
│  └─ DB Subnet Group (db-subnet-group)
│
└─ S3 Bucket (app-assets)
   └─ CloudFront Distribution (cdn-main)

Dependency Statistics:
- Root Resources:      3 (no dependencies)
- Leaf Resources:      15 (no dependents)
- Most Dependencies:   web-server-1 (6 dependencies)
- Most Dependents:     vpc-main (12 dependents)

Generated visualization: ./state-graph.html
Open in browser to view interactive graph.
```

## Fixing Drift

### Automated Drift Remediation

```bash
$ driftmgr drift fix --auto --safety-check

DriftMgr Automated Drift Remediation
============================================================

Analyzing drift items...
Found 18 drift items across 3 providers

Remediation Plan:
------------------------------------------------------------
Priority | Resource           | Action                    | Risk
------------------------------------------------------------
CRITICAL | sg-0abc123        | Remove 0.0.0.0/0 rule     | Low
CRITICAL | s3-prod-data      | Enable encryption         | None
HIGH     | rds-prod-db       | Restore 30-day backup     | Low
HIGH     | i-0def456abc      | Resize to t2.micro        | Med*
MEDIUM   | alb-frontend      | Reset health check        | Low
MEDIUM   | asg-workers       | Set min capacity to 3     | Low
LOW      | vpc-main          | Enable DNS hostnames      | None

* Warning: Resizing instance requires 2-minute downtime

Safety Checks:
- All changes reversible
- Snapshots will be created
- No data loss risk
- Estimated downtime: 2 minutes (instance resize)

Proceed with remediation? (y/N): y

Executing Remediation:
[1/7] Removing unsafe security group rule... DONE
[2/7] Enabling S3 bucket encryption... DONE
[3/7] Updating RDS backup retention... DONE
[4/7] Creating instance snapshot... DONE
[5/7] Resizing EC2 instance... DONE (1m 47s downtime)
[6/7] Resetting ALB health check... DONE
[7/7] Updating ASG configuration... DONE

Remediation Complete!
- Fixed: 7/7 drift items
- Time: 3m 22s
- Cost Savings: $8.76/month
- Compliance: SOC2 requirements restored

Post-Remediation Verification:
Running drift detection... No drift detected
```

### Drift Remediation Planning

```bash
$ driftmgr drift fix --plan

Generating Drift Remediation Plan...
============================================================

Remediation Plan Summary:

Priority | Resource           | Issue                    | Action
---------|-------------------|--------------------------|--------
CRITICAL | sg-0abc123        | Open to 0.0.0.0/0       | Remove rule
CRITICAL | s3-prod-data      | Encryption disabled      | Enable AES-256
HIGH     | rds-prod-db       | Backup retention 7 days  | Restore to 30
HIGH     | i-0def456abc      | Wrong instance type      | Resize
MEDIUM   | alb-frontend      | Health check modified    | Reset
LOW      | 8 resources       | Tag changes              | Update tags

Remediation Strategy:
├─ Total Changes: 14
├─ Estimated Time: 15 minutes
├─ Requires Downtime: Yes (2 minutes for instance resize)
├─ Risk Level: Medium
└─ Rollback Available: Yes

Safety Checks:
├─ Snapshot creation: Enabled
├─ Backup verification: Passed
├─ Dependencies checked: No conflicts
└─ Approval required: Yes (critical changes)

To proceed with remediation:
1. Review the plan above
2. Run: driftmgr drift fix --execute
3. Or run: driftmgr drift auto-remediate enable

For manual remediation:
- Use your existing Terraform workflow
- Or apply fixes individually through cloud console
```

### Auto-Remediation Configuration

```bash
$ driftmgr drift auto-remediate status

Auto-Remediation Status
============================================================

Status: ENABLED
Mode: Safe Mode (Critical issues only)
Last Run: 2024-01-15 10:30:00
Next Run: In 15 minutes

Configuration:
├─ Check Interval: 15 minutes
├─ Max Concurrent Fixes: 5
├─ Approval Required: Yes (for production)
└─ Rollback Enabled: Yes

Active Rules:
├─ auto-fix-security: Fix critical security issues
├─ auto-fix-encryption: Enable encryption on unencrypted resources
├─ auto-fix-backups: Restore backup configurations
└─ auto-tag: Apply required tags

Recent Actions:
├─ [10:30] Fixed: sg-0abc123 - Removed unsafe rule
├─ [10:15] Fixed: s3-logs - Enabled encryption
├─ [10:00] Detected: rds-prod - Backup retention drift (pending approval)
└─ [09:45] Fixed: 12 resources - Applied missing tags

To modify auto-remediation:
$ driftmgr drift auto-remediate enable --rules security,encryption
$ driftmgr drift auto-remediate disable
$ driftmgr drift auto-remediate test --dry-run
```

## Advanced Features

### Enterprise-Ready Capabilities

DriftMgr includes production-grade features for enterprise deployments.

#### Security & Compliance

**1. Credential Management**
- Encrypted credential storage using AES-256-GCM
- Integration with HashiCorp Vault
- Support for IAM roles and service principals
- Automatic credential rotation support

**2. Audit & Compliance**
- Complete audit trail of all operations with file-based logging
- SOC2, HIPAA, and PCI-DSS compliance modes with retention policies
- Role-based access control (RBAC) with predefined roles
- Detailed activity logging with CEF/SIEM export support

#### Performance Optimizations

**1. Intelligent Caching**
- TTL-based cache with automatic invalidation
- Distributed caching support
- Incremental discovery for large environments

**2. Parallel Processing**
- Concurrent resource discovery
- Batched API calls
- Rate limiting and backoff strategies

#### Resilience Features

**Circuit Breaker Pattern**
```bash
$ driftmgr discover --provider aws

Circuit Breaker Status:
=====================
Provider | State  | Failures | Last Success
---------|--------|----------|---------------
AWS      | CLOSED | 0/5      | 2 min ago
Azure    | CLOSED | 0/5      | 5 min ago
GCP      | OPEN   | 5/5      | 1 hour ago

Note: GCP discovery temporarily disabled due to repeated failures
Will retry in: 5 minutes
```

**Automatic Retry Logic**
- Exponential backoff for transient failures
- Provider-specific retry strategies
- Configurable retry limits and timeouts

## Command Reference

### Core Commands

| Command | Description | Example |
|---------|-------------|---------|
| `status` | Show system status and credentials | `driftmgr status` |
| `discover` | Discover cloud resources | `driftmgr discover --all` |
| `drift detect` | Detect infrastructure drift | `driftmgr drift detect --provider aws` |
| `drift fix` | Plan and execute drift remediation | `driftmgr drift fix --plan` |
| `state` | Manage Terraform state files | `driftmgr state analyze --file terraform.tfstate` |
| `accounts` | List all detected cloud accounts | `driftmgr accounts` |
| `use` | Select account/subscription to work with | `driftmgr use aws` |
| `delete` | Delete cloud resources | `driftmgr delete ec2 i-abc123` |
| `verify` | Verify drift detection results | `driftmgr verify` |
| `import` | Import resources into management | `driftmgr import` |
| `export` | Export discovery results | `driftmgr export --format json` |
| `serve` | Start web server or API | `driftmgr serve web` |

### Discovery Commands

```bash
# Discover all resources across all providers
driftmgr discover --all

# Discover with specific provider
driftmgr discover --provider aws

# Discover across all accounts/subscriptions
driftmgr discover --all-accounts

# Show credential status
driftmgr discover --show-credentials

# Export results after discovery
driftmgr export --format json --output discovery.json
```

### Drift Detection Commands

```bash
# Detect all drift
driftmgr drift detect --all

# Detect with smart filtering (default)
driftmgr drift detect --smart

# Detect without filtering (see all changes)
driftmgr drift detect --no-filter

# Detect with custom sensitivity
driftmgr drift detect --sensitivity high

# Schedule continuous monitoring
driftmgr drift monitor --interval 15m
```

### State Management Commands

```bash
# Analyze state file
driftmgr state analyze --file terraform.tfstate

# Compare state with reality
driftmgr state compare --file terraform.tfstate

# Import resources into state
driftmgr state import --resource aws_instance.web i-0abc123

# Clean unused resources from state
driftmgr state clean --file terraform.tfstate

# Backup state file
driftmgr state backup --file terraform.tfstate
```

### Remediation Commands

```bash
# Auto-fix all critical issues
driftmgr drift fix --priority critical --auto

# Generate remediation plan
driftmgr drift fix --plan-only

# Fix with approval workflow
driftmgr drift fix --require-approval

# Rollback last remediation
driftmgr drift rollback --last

# Schedule remediation
driftmgr drift fix --schedule "2024-01-20 02:00 UTC"
```

## Production Features

### Enterprise-Grade Capabilities

- **Circuit Breakers** - Prevent cascade failures with automatic recovery
- **Health Checks** - Kubernetes-ready liveness and readiness probes
- **Security Vault** - AES-256-GCM encryption for credentials at rest
- **Rate Limiting** - Provider-specific limits to prevent API throttling
- **Metrics Collection** - Performance and operational metrics
- **Graceful Shutdown** - Clean shutdown with resource cleanup
- **Audit Logging** - Structured logging with request tracing
- **Retry Logic** - Exponential backoff for transient failures

#### Experimental Features

- **Distributed State** - etcd integration (requires manual setup)
- **OpenTelemetry** - Distributed tracing (partial implementation)
- **WebSocket Support** - Real-time updates (basic implementation)

### Health Check Endpoints

**Note:** Health and metrics endpoints require the web server to be running.

```bash
# First, start the web server
driftmgr serve web --port 8080

# Then in another terminal, check system health
curl http://localhost:8080/health

{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "checks": {
    "database": "healthy",
    "cache": "healthy",
    "providers": {
      "aws": "healthy",
      "azure": "healthy",
      "gcp": "healthy"
    }
  },
  "metrics": {
    "uptime": "72h15m",
    "requests_total": 15234,
    "resources_discovered": 458,
    "drift_detected": 18
  }
}
```

## Configuration

### Configuration Files

DriftMgr looks for configuration files in the following order:
1. `./configs/driftmgr.yaml` (project directory)
2. `./driftmgr.yaml` (current directory)
3. `~/.driftmgr/config.yaml` (user home directory)
4. Environment variables (override file settings)

### Basic Configuration Example

```yaml
# configs/driftmgr.yaml
app:
  name: driftmgr
  environment: production
  log_level: info

# Provider Configuration
providers:
  aws:
    enabled: true
    regions:
      - us-west-2
      - us-east-1
    rate_limit: 20  # requests per second
    
  azure:
    enabled: true
    subscriptions:
      - production
      - staging
    rate_limit: 15
    
  gcp:
    enabled: true
    projects:
      - my-project-123
    rate_limit: 20

# Drift Detection Settings
drift:
  sensitivity: medium  # low, medium, high
  smart_filter: true   # Enable intelligent filtering
  ignore_tags:         # Tags to ignore in drift detection
    - LastModified
    - CreatedBy

# Remediation Settings
remediation:
  auto_fix: false          # Require manual approval
  safety_checks: true      # Always run safety checks
  create_snapshots: true   # Backup before changes
  rollback_on_failure: true
  max_parallel: 5          # Parallel remediation tasks (default)

# Performance Settings
performance:
  cache_ttl: 5m
  max_connections: 100
  discovery_timeout: 30s
  workers: 5               # Default concurrent workers (max: 10)
```

### Environment Variables

Override configuration with environment variables:

```bash
# Credential timeout
export DRIFTMGR_CREDENTIAL_TIMEOUT=60

# Log level
export DRIFTMGR_LOG_LEVEL=debug

# Provider settings
export DRIFTMGR_AWS_REGIONS=us-west-2,us-east-1
export DRIFTMGR_AZURE_SUBSCRIPTION=production-sub

# Performance tuning
export DRIFTMGR_WORKERS=10
export DRIFTMGR_CACHE_TTL=10m
```

## Web Dashboard

### Starting the Web Dashboard

```bash
$ driftmgr serve web --port 8080

DriftMgr Web Dashboard
============================================================

Starting web server...
- REST API:     http://localhost:8080/api/v1
- WebSocket:     ws://localhost:8080/ws
- Dashboard:     http://localhost:8080
- Health:        http://localhost:8080/health
- Metrics:       http://localhost:8080/metrics (JSON format, not Prometheus)

Real-time monitoring enabled
WebSocket connections: 0
Auto-refresh: Every 30 seconds

Press Ctrl+C to stop the server
```

### API Features

- **REST API** - Full REST API for programmatic access
- **Health Checks** - Kubernetes-ready health endpoints (`/health/live`, `/health/ready`)
- **Metrics Export** - JSON metrics at `/metrics` endpoint (not Prometheus format)
- **WebSocket Support** - Real-time updates (experimental)
- **Audit Logging** - Complete history of all operations

**Note:** All API endpoints require `driftmgr serve web` to be running.

## Performance

### Performance Characteristics

| Operation | Typical Performance | Notes |
|-----------|-------------------|--------|
| Resource Discovery | 50-200 resources/sec | Depends on API rate limits |
| Drift Detection | 100-500 resources/sec | Local state comparison |
| State Analysis | 500+ resources/sec | File parsing only |
| Parallel Discovery | Default: 5 concurrent | Max: 10 (configurable via workers flag) |

### Optimization Features

- **Parallel Processing** - Concurrent resource discovery
- **Intelligent Caching** - Reduces redundant API calls
- **Rate Limiting** - Respects provider API quotas
- **Incremental Discovery** - Scan only changed resources
- **Smart Filtering** - Reduces noise in drift detection

## Security

### Security Features

- **Encrypted Storage** - AES-256-GCM for credentials at rest
- **Audit Logging** - Complete audit trail of all operations
- **RBAC Support** - Role-based access control ready
- **Secret Management** - Integration with HashiCorp Vault
- **TLS Support** - Encrypted communication
- **Security Scanning** - Built-in security checks for drift

### Security Best Practices

```bash
# Use environment variables for credentials
export AWS_PROFILE=production
export DRIFTMGR_ENCRYPTION_KEY=$(openssl rand -base64 32)

# Enable audit logging
driftmgr --audit-log /var/log/driftmgr/audit.log

# Use read-only credentials when possible
driftmgr discover --read-only

# Encrypt sensitive outputs
driftmgr discover --export encrypted --password
```

## Troubleshooting

### Common Issues

#### Issue: Timeout during discovery
```bash
# Increase timeout
export DRIFTMGR_DISCOVERY_TIMEOUT=60s

# Reduce parallel workers
driftmgr discover --workers 5

# Use specific regions
driftmgr discover --region us-west-2
```

#### Issue: High memory usage
```bash
# Enable incremental discovery
driftmgr discover --incremental

# Clear cache
driftmgr cache clear

# Reduce batch size
driftmgr discover --batch-size 50
```

#### Issue: Rate limiting errors
```bash
# Check current limits
driftmgr config show rate-limits

# Adjust rate limits
driftmgr config set aws.rate_limit=10

# Use exponential backoff
driftmgr discover --retry-backoff
```

## Docker Deployment

### Using Docker

```bash
# Build the Docker image locally first
docker build -t driftmgr:latest .

# Then run with Docker
docker run -it --rm \
  -e AWS_PROFILE=default \
  -e AZURE_SUBSCRIPTION_ID=xxx \
  -v ~/.aws:/root/.aws:ro \
  driftmgr:latest \
  discover --all

# Or use Docker Compose
docker-compose up -d
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
        image: driftmgr:latest  # Build locally first: docker build -t driftmgr:latest .
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
        env:
        - name: DRIFTMGR_ENVIRONMENT
          value: production
```

## Practical Workflows

### 1. Initial Cloud Infrastructure Audit

Perform a complete audit of your cloud infrastructure:

```bash
# Check configured credentials
driftmgr status

# Discover all resources across all providers
driftmgr discover --all

# Export findings for review
driftmgr export --format json --output audit.json
driftmgr export --format csv --output audit.csv
```

### 2. Multi-Account Resource Discovery

Work with multiple cloud accounts:

```bash
# List all available accounts
driftmgr accounts

# Select specific AWS account
driftmgr use aws

# Discover resources in selected account
driftmgr discover --provider aws

# Switch to Azure subscription
driftmgr use azure
driftmgr discover --provider azure

# Or discover all accounts at once
driftmgr discover --all-accounts
```

### 3. Terraform State Drift Detection

Detect and manage drift in Terraform-managed infrastructure:

```bash
# Find all state files in current directory
driftmgr state scan --dir .

# Analyze a specific state file
driftmgr state analyze --file terraform.tfstate

# Detect drift between state and reality
driftmgr drift detect --state terraform.tfstate

# Generate drift report
driftmgr drift report --format json

# Plan remediation
driftmgr drift fix --plan
```

### 4. Automated Drift Monitoring

Set up continuous drift detection and auto-remediation:

```bash
# Check current auto-remediation config
driftmgr drift auto-remediate status

# Configure remediation rules
driftmgr drift auto-remediate configure --rules security,encryption

# Test rules without making changes
driftmgr drift auto-remediate test --dry-run

# Enable auto-remediation
driftmgr drift auto-remediate enable

# Disable when needed
driftmgr drift auto-remediate disable
```

### 5. Resource Cleanup

Clean up unused or unwanted resources:

```bash
# Discover all resources first
driftmgr discover --all

# Preview deletion (dry run)
driftmgr delete ec2 i-0abc123 --dry-run
driftmgr delete rds my-database --dry-run

# Actually delete resources
driftmgr delete ec2 i-0abc123

# Verify cleanup
driftmgr verify
```

### 6. Security Compliance Check

Focus on security-critical drift:

```bash
# Detect drift with security focus
driftmgr drift detect --smart-defaults

# Generate security report
driftmgr drift report --format json --output security.json

# Review security fixes
driftmgr drift fix --plan

# Enable auto-fix for security issues
driftmgr drift auto-remediate enable --rules security

# Apply security fixes
driftmgr drift fix --execute
```

### 7. Cross-Environment Comparison

Compare resources between environments:

```bash
# Select production account
driftmgr use aws  # Choose production profile
driftmgr discover --provider aws
driftmgr export --format json --output prod.json

# Switch to staging account
driftmgr use aws  # Choose staging profile
driftmgr discover --provider aws
driftmgr export --format json --output staging.json

# Compare JSON files with external tools
diff prod.json staging.json
```

### 8. Change Management Process

Manage infrastructure changes safely:

```bash
# Baseline before changes
driftmgr drift detect
driftmgr export --format json --output before.json

# Make your infrastructure changes
# ...

# Check what changed
driftmgr drift detect
driftmgr drift fix --plan    # Review remediation
driftmgr drift fix --execute  # Apply fixes
driftmgr verify              # Verify final state
```

### 9. API Integration

Set up API access for external tools:

```bash
# Start web server
driftmgr serve web --port 8080

# In another terminal, check endpoints:
curl http://localhost:8080/health
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
curl http://localhost:8080/metrics
```

### 10. Troubleshooting

Debug issues with DriftMgr:

```bash
# Check system status
driftmgr status

# Test single provider
driftmgr discover --provider aws

# Debug credential detection
driftmgr discover --show-credentials

# Run verification checks
driftmgr verify

# Start server for health checks
driftmgr serve web --port 8080
curl http://localhost:8080/health
```

## Monitoring & Observability

### Metrics Collection

```bash
# Start the web server to enable metrics
driftmgr serve web --port 8080

# Access metrics endpoint (JSON format)
curl http://localhost:8080/metrics
```

**Available Metrics:**
- `discovery_duration` - Time taken for resource discovery
- `resources_discovered_total` - Total resources found
- `drift_detected_total` - Total drift items detected
- `remediation_success_total` - Successful remediations
- `api_requests_total` - API request count
- `cache_hit_ratio` - Cache effectiveness

**Note:** Metrics are currently in JSON format. Prometheus export format is planned for future releases.

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

```bash
# Clone repository
git clone https://github.com/catherinevee/driftmgr.git
cd driftmgr

# Install dependencies
go mod download

# Run tests
make test-all

# Build locally
make build

# Run locally
./build/driftmgr status
```

## License

DriftMgr is licensed under the MIT License. See [LICENSE](LICENSE) for details.

## Support

- **Documentation**: [docs.driftmgr.io](https://docs.driftmgr.io)
- **Issues**: [GitHub Issues](https://github.com/catherinevee/driftmgr/issues)
- **Discussions**: [GitHub Discussions](https://github.com/catherinevee/driftmgr/discussions)
- **Security**: Report security issues to security@driftmgr.io

## Acknowledgments

DriftMgr is built with production-grade features including:
- Multiple cloud provider SDKs (AWS, Azure, GCP, DigitalOcean)
- Circuit breaker pattern for resilience
- Structured logging and audit trails
- Security-first design with encryption

---

**Built for DevOps and Cloud Engineers**

*Making cloud infrastructure drift a thing of the past*