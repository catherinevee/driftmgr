# DriftMgr Drift Simulation Guide

## Overview

DriftMgr's Drift Simulation feature allows you to create controlled, safe drift in your cloud infrastructure to test and demonstrate drift detection capabilities. This feature uses your existing cloud credentials to make real (but harmless and free) changes to resources.

## Key Features

- **Zero Cost**: All simulations use free-tier resources or cost $0
- **Safe**: Uses test IP ranges (192.0.2.0/32) and harmless modifications
- **Reversible**: All changes can be rolled back automatically
- **Multi-Cloud**: Works with AWS, Azure, and GCP
- **Realistic**: Uses actual cloud APIs, not mocks

## Installation

The drift simulation feature is built into DriftMgr. No additional installation required.

```bash
# Verify installation
driftmgr simulate-drift --help
```

## Quick Start

### 1. Basic Tag Drift Simulation (AWS)

```bash
# Simulate adding tags to AWS resources
driftmgr simulate-drift \
  --state terraform.tfstate \
  --provider aws \
  --type tag-change
```

This will:
- Read your terraform.tfstate file
- Find AWS resources in the state
- Add a "DriftSimulation" tag to a resource
- Detect the drift
- Optionally roll back the change

### 2. Unmanaged Resource Creation (Azure)

```bash
# Create a resource not in Terraform state
driftmgr simulate-drift \
  --state terraform.tfstate \
  --provider azure \
  --type resource-creation
```

This creates a small resource group with auto-delete tags.

### 3. Security Rule Addition (GCP)

```bash
# Add a firewall rule to simulate security drift
driftmgr simulate-drift \
  --state terraform.tfstate \
  --provider gcp \
  --type rule-addition
```

Creates a harmless deny rule for testing.

## Drift Types

### 1. Tag/Label Changes (`tag-change`)
- **AWS**: Adds tags to EC2 instances, S3 buckets, VPCs
- **Azure**: Adds tags to resource groups, VNets, NSGs
- **GCP**: Adds labels to compute instances, storage buckets
- **Cost**: $0.00
- **Risk**: None

### 2. Security Rule Additions (`rule-addition`)
- **AWS**: Adds security group rules (port 8443 from TEST-NET)
- **Azure**: Adds NSG rules (deny rule from TEST-NET)
- **GCP**: Creates firewall rules (deny rule from TEST-NET)
- **Cost**: $0.00
- **Risk**: None (uses RFC 5737 test networks)

### 3. Resource Creation (`resource-creation`)
- **AWS**: Creates S3 bucket with 1-day lifecycle
- **Azure**: Creates resource group with auto-delete tag
- **GCP**: Creates storage bucket with 1-day lifecycle
- **Cost**: $0.00 (auto-deletes within 24 hours)
- **Risk**: None

### 4. Attribute Changes (`attribute-change`)
- **AWS**: Modifies S3 bucket versioning
- **Azure**: Updates resource tags
- **GCP**: Changes resource labels
- **Cost**: $0.00
- **Risk**: None

### 5. Random (`random`)
- Randomly selects one of the above drift types
- Useful for testing detection capabilities

## Command Options

```bash
driftmgr simulate-drift [options]
```

| Option | Description | Default |
|--------|-------------|---------|
| `--state` | Path to Terraform state file | Required |
| `--provider` | Cloud provider (aws/azure/gcp) | Auto-detected |
| `--type` | Type of drift to simulate | random |
| `--target` | Specific resource to target | Auto-selected |
| `--auto-rollback` | Automatically rollback after detection | true |
| `--dry-run` | Preview changes without applying | false |
| `--rollback` | Rollback previous simulation | - |
| `--detect` | Run drift detection after simulation | true |
| `--verbose` | Show detailed output | false |

## Complete Workflow Example

### Step 1: Check Current State

```bash
# See what's in your state file
driftmgr state analyze --state terraform.tfstate
```

### Step 2: Simulate Drift

```bash
# Create some drift
driftmgr simulate-drift \
  --state terraform.tfstate \
  --provider aws \
  --type tag-change \
  --auto-rollback false
```

Output:
```
=== Drift Simulation Plan ===
State File: terraform.tfstate
Provider: aws
Drift Type: tag-change
Auto Rollback: false

🔄 Simulating drift...

✅ Drift Simulation Successful!
Provider: aws
Resource Type: aws_instance
Resource ID: i-1234567890
Drift Type: tag-change
Cost Estimate: $0.00 (tags are free)

Changes Applied:
  • added_tag: {DriftSimulation: Created-2024-01-15-10:30:45}

💾 Rollback data saved (use --rollback to undo)
```

### Step 3: Detect the Drift

```bash
# Run DriftMgr's drift detection
driftmgr drift detect --provider aws
```

Output:
```
⚠️ Drift Detected! Found 1 drift(s):

1. i-1234567890 (aws_instance)
   Type: tag_addition
   Impact: Low - Tag addition detected
   Before:
     tags: {Name: web-server, Environment: production}
   After:
     tags: {Name: web-server, Environment: production, DriftSimulation: Created-2024-01-15-10:30:45}
```

### Step 4: Generate Import Commands

```bash
# If drift was an unmanaged resource
driftmgr import generate --unmanaged
```

### Step 5: Rollback Changes

```bash
# Clean up the drift
driftmgr simulate-drift --rollback
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Drift Detection Test

on:
  schedule:
    - cron: '0 9 * * 1'  # Weekly on Monday

jobs:
  drift-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Setup DriftMgr
        run: |
          curl -L https://github.com/catherinevee/driftmgr/releases/latest/download/driftmgr-linux -o driftmgr
          chmod +x driftmgr
      
      - name: Simulate Drift
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
        run: |
          ./driftmgr simulate-drift \
            --state terraform.tfstate \
            --provider aws \
            --type random \
            --dry-run
      
      - name: Verify Detection
        run: |
          ./driftmgr drift detect --provider aws --fail-on-drift
```

### Jenkins Pipeline Example

```groovy
pipeline {
    agent any
    
    stages {
        stage('Drift Simulation Test') {
            steps {
                script {
                    sh '''
                        # Simulate drift
                        ./driftmgr simulate-drift \
                          --state ${WORKSPACE}/terraform.tfstate \
                          --provider aws \
                          --type tag-change
                        
                        # Verify detection works
                        ./driftmgr drift detect --provider aws
                        
                        # Clean up
                        ./driftmgr simulate-drift --rollback
                    '''
                }
            }
        }
    }
}
```

## Safety Features

### 1. Automatic Expiration
All created resources include auto-delete mechanisms:
- S3 buckets: 1-day lifecycle rules
- Azure RGs: Auto-delete tags
- GCP buckets: 1-day lifecycle rules

### 2. Safe Network Rules
Security rules use RFC 5737 TEST-NET ranges:
- 192.0.2.0/24 (TEST-NET-1)
- 198.51.100.0/24 (TEST-NET-2)
- 203.0.113.0/24 (TEST-NET-3)

### 3. Rollback Tracking
Every simulation saves rollback data:
```bash
# Location: .driftmgr/rollback.json
{
  "provider": "aws",
  "resource_type": "aws_s3_bucket",
  "resource_id": "drift-simulation-1234567890",
  "action": "delete_resource",
  "timestamp": "2024-01-15T10:30:45Z"
}
```

### 4. Dry Run Mode
Test without making changes:
```bash
driftmgr simulate-drift --state terraform.tfstate --dry-run
```

## Multi-Account Testing

### AWS Cross-Account

```bash
# Use AWS profiles
export AWS_PROFILE=staging
driftmgr simulate-drift --state staging.tfstate --provider aws

export AWS_PROFILE=production
driftmgr simulate-drift --state prod.tfstate --provider aws
```

### Azure Subscriptions

```bash
# Switch subscriptions
az account set --subscription "Staging"
driftmgr simulate-drift --state staging.tfstate --provider azure

az account set --subscription "Production"
driftmgr simulate-drift --state prod.tfstate --provider azure
```

### GCP Projects

```bash
# Set project
export GOOGLE_CLOUD_PROJECT=staging-project
driftmgr simulate-drift --state staging.tfstate --provider gcp

export GOOGLE_CLOUD_PROJECT=prod-project
driftmgr simulate-drift --state prod.tfstate --provider gcp
```

## Troubleshooting

### Issue: "No suitable resources found in state file"

**Solution**: The state file might not contain resources of the correct type. Try:
```bash
# List resources in state
driftmgr state list --state terraform.tfstate

# Target a specific resource
driftmgr simulate-drift --state terraform.tfstate --target aws_instance.web
```

### Issue: "Failed to initialize provider"

**Solution**: Check cloud credentials:
```bash
# AWS
aws sts get-caller-identity

# Azure
az account show

# GCP
gcloud config list
```

### Issue: "Rollback failed"

**Solution**: Manually clean up resources:
```bash
# AWS - Delete S3 bucket
aws s3 rb s3://drift-simulation-xxxxx --force

# Azure - Delete resource group
az group delete --name drift-simulation-xxxxx --yes

# GCP - Delete bucket
gsutil rm -r gs://drift-simulation-xxxxx
```

## Best Practices

1. **Always use dry-run first** to preview changes
2. **Keep auto-rollback enabled** for safety
3. **Use specific resource targeting** in production
4. **Save simulation logs** for audit purposes
5. **Run in non-production first** to test
6. **Monitor costs** (should always be $0)

## Cost Analysis

All drift simulations are designed to be completely free:

| Provider | Resource | Cost | Notes |
|----------|----------|------|-------|
| AWS | Tags | $0.00 | Tags are free |
| AWS | S3 Bucket | $0.00 | Deleted within 1 day |
| AWS | Security Group Rules | $0.00 | Rules are free |
| Azure | Tags | $0.00 | Tags are free |
| Azure | Resource Groups | $0.00 | RGs are free |
| Azure | NSG Rules | $0.00 | Rules are free |
| GCP | Labels | $0.00 | Labels are free |
| GCP | Storage Buckets | $0.00 | Deleted within 1 day |
| GCP | Firewall Rules | $0.00 | Rules are free |

## Advanced Usage

### Custom Drift Patterns

Create specific drift scenarios:

```bash
# Simulate compliance violation
driftmgr simulate-drift \
  --state terraform.tfstate \
  --provider aws \
  --type rule-addition \
  --target aws_security_group.database

# Simulate cost-impacting drift
driftmgr simulate-drift \
  --state terraform.tfstate \
  --provider aws \
  --type attribute-change \
  --target aws_s3_bucket.backups
```

### Batch Simulation

Test multiple drift types:

```bash
#!/bin/bash
for drift_type in tag-change rule-addition resource-creation; do
  echo "Testing $drift_type drift..."
  driftmgr simulate-drift \
    --state terraform.tfstate \
    --provider aws \
    --type $drift_type \
    --auto-rollback true
  sleep 5
done
```

### Automated Testing

```python
import subprocess
import json

def test_drift_detection():
    """Test that DriftMgr detects simulated drift"""
    
    # Simulate drift
    result = subprocess.run([
        "driftmgr", "simulate-drift",
        "--state", "terraform.tfstate",
        "--provider", "aws",
        "--type", "tag-change",
        "--output", "json"
    ], capture_output=True, text=True)
    
    simulation = json.loads(result.stdout)
    assert simulation["success"] == True
    
    # Detect drift
    result = subprocess.run([
        "driftmgr", "drift", "detect",
        "--provider", "aws",
        "--output", "json"
    ], capture_output=True, text=True)
    
    drifts = json.loads(result.stdout)
    assert len(drifts) > 0
    
    # Rollback
    subprocess.run(["driftmgr", "simulate-drift", "--rollback"])
    
    print("✅ Drift detection test passed!")

if __name__ == "__main__":
    test_drift_detection()
```

## Conclusion

DriftMgr's drift simulation feature provides a safe, cost-free way to:
- Test drift detection accuracy
- Demonstrate DriftMgr capabilities
- Validate monitoring and alerting
- Train teams on drift remediation
- Verify CI/CD drift detection pipelines

All while using real cloud APIs and your actual infrastructure configuration.