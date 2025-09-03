# DriftMgr Examples

This directory contains practical examples demonstrating various use cases for DriftMgr.

## Examples Overview

### 1. Basic Usage
- [Simple Discovery](./basic/simple-discovery.md) - Basic resource discovery
- [Drift Detection](./basic/drift-detection.md) - Detecting configuration drift
- [State Management](./basic/state-management.md) - Working with Terraform states

### 2. Multi-Cloud Scenarios
- [AWS Multi-Account](./multi-cloud/aws-multi-account.md) - Managing multiple AWS accounts
- [Azure Subscriptions](./multi-cloud/azure-subscriptions.md) - Cross-subscription management
- [Hybrid Cloud](./multi-cloud/hybrid-cloud.md) - AWS + Azure + GCP together

### 3. CI/CD Integration
- [GitHub Actions](./cicd/github-actions.md) - Automated drift detection in CI
- [GitLab CI](./cicd/gitlab-ci.md) - GitLab pipeline integration
- [Jenkins](./cicd/jenkins.md) - Jenkins pipeline example

### 4. Advanced Features
- [OPA Policies](./advanced/opa-policies.md) - Policy enforcement examples
- [Custom Workflows](./advanced/workflows.md) - Complex automation workflows
- [Compliance](./advanced/compliance.md) - SOC2, HIPAA, PCI-DSS reporting

### 5. Troubleshooting
- [Common Issues](./troubleshooting/common-issues.md) - Frequently encountered problems
- [Performance Tuning](./troubleshooting/performance.md) - Optimization tips

## Quick Start Examples

### Example 1: Discover All Resources in AWS
```bash
# Discover all resources in current AWS account
driftmgr discover --provider aws --region us-east-1

# Export to JSON
driftmgr discover --provider aws --format json > aws-resources.json

# Filter specific resource types
driftmgr discover --provider aws --types "ec2,rds,s3"
```

### Example 2: Detect Drift from Terraform State
```bash
# Analyze local state file
driftmgr drift detect --state terraform.tfstate

# Analyze remote state in S3
driftmgr drift detect \
  --backend s3 \
  --bucket my-terraform-states \
  --key prod/terraform.tfstate

# Generate detailed report
driftmgr drift detect --state terraform.tfstate --format detailed > drift-report.md
```

### Example 3: Multi-Account Discovery
```bash
# Discover across all configured AWS accounts
driftmgr discover --all-accounts --provider aws

# Use specific profile
AWS_PROFILE=production driftmgr discover --provider aws

# Switch between accounts interactively
driftmgr use aws
# Select account from menu
driftmgr discover
```

### Example 4: Continuous Monitoring
```bash
# Start monitoring with webhooks
driftmgr monitor start \
  --interval 300 \
  --webhook https://hooks.slack.com/services/YOUR/WEBHOOK

# Monitor specific resources
driftmgr monitor start \
  --resources "prod-*" \
  --critical-only

# Check monitoring status
driftmgr monitor status
```

### Example 5: Compliance Reporting
```bash
# Generate SOC2 compliance report
driftmgr compliance report \
  --standard soc2 \
  --format pdf \
  --output soc2-report.pdf

# HIPAA compliance check
driftmgr compliance check --standard hipaa

# Custom policy validation
driftmgr policy validate \
  --policy policies/production.rego \
  --state terraform.tfstate
```

### Example 6: State Push/Pull Operations
```bash
# Pull state from S3 backend
driftmgr state pull \
  --backend s3 \
  --bucket terraform-states \
  --key prod/terraform.tfstate \
  --output local.tfstate

# Push state to Azure Storage
driftmgr state push \
  --backend azurerm \
  --storage-account tfstates \
  --container prod \
  --key terraform.tfstate \
  --input local.tfstate

# List all states in backend
driftmgr state list --backend s3 --bucket terraform-states
```

### Example 7: Remediation Workflows
```bash
# Generate import commands for unmanaged resources
driftmgr remediate generate-imports \
  --unmanaged-only \
  --output import-commands.sh

# Create Terraform code for discovered resources
driftmgr remediate generate-tf \
  --resources vpc-12345,sg-67890 \
  --output generated.tf

# Execute remediation with approval
driftmgr remediate apply \
  --plan remediation.json \
  --require-approval
```

### Example 8: Web UI and API Server
```bash
# Start web interface
driftmgr serve web --port 8080

# Start API server only
driftmgr serve api --port 3000

# Start with authentication
driftmgr serve web \
  --port 8080 \
  --auth-enabled \
  --auth-provider okta
```

## Configuration Files

### Basic Configuration (config.yaml)
```yaml
providers:
  aws:
    regions:
      - us-east-1
      - us-west-2
    profile: default
  
  azure:
    subscription_id: ${AZURE_SUBSCRIPTION_ID}
    resource_groups:
      - production
      - staging
  
  gcp:
    project_id: my-project
    regions:
      - us-central1

discovery:
  parallel_workers: 10
  timeout: 300
  cache_ttl: 3600

drift:
  ignore_tags:
    - LastModified
    - CreatedBy
  severity_thresholds:
    critical: 0.8
    high: 0.6
    medium: 0.4

monitoring:
  enabled: true
  interval: 300
  webhooks:
    - url: ${SLACK_WEBHOOK_URL}
      events: [drift_detected, resource_created]
```

### Docker Compose Example
```yaml
version: '3.8'

services:
  driftmgr:
    image: catherinevee/driftmgr:latest
    ports:
      - "8080:8080"
    environment:
      - AWS_PROFILE=default
      - AZURE_SUBSCRIPTION_ID=${AZURE_SUBSCRIPTION_ID}
    volumes:
      - ~/.aws:/root/.aws:ro
      - ~/.azure:/root/.azure:ro
      - ./config.yaml:/app/config.yaml
    command: serve web --config /app/config.yaml

  localstack:
    image: localstack/localstack:latest
    ports:
      - "4566:4566"
    environment:
      - SERVICES=ec2,s3,iam,sts
      - DEBUG=1
    volumes:
      - ./localstack:/tmp/localstack
```

### GitHub Actions Workflow
```yaml
name: Drift Detection

on:
  schedule:
    - cron: '0 */6 * * *'  # Every 6 hours
  workflow_dispatch:

jobs:
  detect-drift:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup DriftMgr
        run: |
          curl -sSL https://github.com/catherinevee/driftmgr/releases/latest/download/driftmgr-linux-amd64.tar.gz | tar xz
          chmod +x driftmgr
      
      - name: Configure AWS
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ secrets.AWS_ROLE_ARN }}
          aws-region: us-east-1
      
      - name: Detect Drift
        run: |
          ./driftmgr drift detect \
            --backend s3 \
            --bucket ${{ secrets.TF_STATE_BUCKET }} \
            --key prod/terraform.tfstate \
            --format json > drift.json
      
      - name: Check Critical Drift
        run: |
          CRITICAL=$(jq '.summary.critical_count' drift.json)
          if [ "$CRITICAL" -gt 0 ]; then
            echo "❌ Critical drift detected!"
            exit 1
          fi
      
      - name: Upload Report
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: drift-report
          path: drift.json
```

## Best Practices

### 1. Resource Tagging Strategy
```bash
# Enforce tagging compliance
driftmgr policy validate --policy policies/tags.rego

# Find resources without required tags
driftmgr discover --filter 'tags.Environment == null'
```

### 2. Incremental Discovery
```bash
# First run - full discovery with cache
driftmgr discover --cache-enabled --cache-file .driftmgr-cache

# Subsequent runs - incremental only
driftmgr discover --incremental --cache-file .driftmgr-cache
```

### 3. Parallel Processing
```bash
# Optimize for large infrastructures
driftmgr discover \
  --parallel-workers 20 \
  --batch-size 100 \
  --timeout 600
```

### 4. Cost Optimization
```bash
# Find expensive drifted resources
driftmgr cost analyze \
  --threshold 100 \
  --drift-only

# Identify unused resources
driftmgr discover --filter 'state == "stopped" && age > 30d'
```

## Troubleshooting Examples

### Debug Mode
```bash
# Enable verbose logging
DRIFTMGR_LOG_LEVEL=debug driftmgr discover

# Trace AWS API calls
AWS_SDK_LOAD_CONFIG=1 AWS_SDK_LOG_LEVEL=debug driftmgr discover
```

### Performance Testing
```bash
# Benchmark discovery
time driftmgr discover --provider aws --metrics

# Profile CPU usage
driftmgr discover --cpu-profile cpu.prof
go tool pprof cpu.prof
```

### Handling Large States
```bash
# Stream large state files
driftmgr drift detect \
  --state s3://bucket/large-state.tfstate \
  --streaming \
  --memory-limit 512MB
```

## Contributing Examples

We welcome contributions! To add your own examples:

1. Create a new file in the appropriate subdirectory
2. Follow the existing format
3. Test your example
4. Submit a pull request

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.