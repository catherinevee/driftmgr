# DriftMgr - Terraform Drift Detection Examples

## Overview
DriftMgr is a powerful tool for detecting drift between your Terraform state files and actual deployed cloud resources. It helps identify when your infrastructure has diverged from your desired state as defined in Terraform.

## Key Features
- Automatic discovery of Terraform backend configurations
- Support for S3, Azure, GCS, and local backends
- Drift detection across AWS, Azure, GCP, and DigitalOcean
- Interactive TUI for visual drift analysis
- Remediation plan generation

## Installation
```bash
# Windows
go build -o driftmgr.exe ./cmd/driftmgr

# Linux/Mac
go build -o driftmgr ./cmd/driftmgr
```

## Usage Examples

### 1. Scanning for Terraform Backends
Discover all Terraform backend configurations in your infrastructure:

```bash
# Scan current directory
driftmgr scan --dir .

# Scan specific infrastructure directory
driftmgr scan --dir ./infrastructure

# Scan with output format
driftmgr scan --dir ./terraform --format json
```

Example output:
```
Found 3 Terraform backend configuration(s):

1. s3 backend in ./production
   Bucket: company-terraform-state
   Key: prod/terraform.tfstate
   Status: Initialized

2. azurerm backend in ./staging
   Storage Account: tfstateaccount
   Container: tfstate
   Key: staging.terraform.tfstate
   Status: Initialized

3. local backend in ./dev
   Path: terraform.tfstate
   Status: Initialized
```

### 2. Detecting Drift
Compare Terraform state with actual cloud resources:

```bash
# Detect drift for a specific state file
driftmgr drift detect --state s3://bucket/terraform.tfstate --provider aws

# Detect drift for Azure resources
driftmgr drift detect --state azurerm://account/container/state.tfstate --provider azure

# Detect drift with severity filter
driftmgr drift detect --state ./terraform.tfstate --severity high
```

Example output:
```
Drift Detection Report
======================
State File: s3://company-terraform-state/prod/terraform.tfstate
Provider: AWS
Scan Time: 2024-01-20 14:30:00

Summary:
- Total Resources: 25
- Drifted: 3 (12%)
- Missing: 1 (4%)
- Unmanaged: 2 (8%)

Critical Issues:
1. aws_security_group.web_sg - MODIFIED
   - Ingress rule added (port 8080)
   - Risk: Unauthorized port exposure

2. aws_instance.database - MISSING
   - Resource exists in state but not in cloud
   - Risk: Data loss, service disruption

High Priority:
1. aws_s3_bucket.logs - MODIFIED
   - Encryption disabled
   - Versioning disabled
```

### 3. Generating Drift Reports
Create detailed reports for documentation and review:

```bash
# Generate JSON report
driftmgr drift report --state ./terraform.tfstate --format json > drift-report.json

# Generate summary report
driftmgr drift report --state ./terraform.tfstate --format summary

# Generate table format
driftmgr drift report --state ./terraform.tfstate --format table
```

### 4. Creating Remediation Plans
Generate Terraform commands to fix drift:

```bash
# Generate fix for all drift
driftmgr drift fix --state ./terraform.tfstate

# Generate fix for critical issues only
driftmgr drift fix --state ./terraform.tfstate --severity critical

# Output to file
driftmgr drift fix --state ./terraform.tfstate > remediation.tf
```

Example remediation output:
```hcl
# Terraform Drift Remediation Plan
# Generated: 2024-01-20 14:45:00
# State File: ./terraform.tfstate

# Resource missing in cloud: aws_instance.database
# Option 1: Re-create resource
# terraform apply -target=aws_instance.database

# Option 2: Remove from state
# terraform state rm aws_instance.database

# Resource drifted: aws_security_group.web_sg
# Run: terraform plan -target=aws_security_group.web_sg
# Then: terraform apply -target=aws_security_group.web_sg
#   ingress.0.from_port: 8080 -> 80
#   ingress.0.to_port: 8080 -> 80

# Unmanaged resource found: i-0abc123def456
# To manage this resource, add it to your Terraform configuration:

resource "aws_instance" "unmanaged_server" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t2.micro"
  
  tags = {
    Name = "UnmanagedServer"
  }
}
```

### 5. Using the Interactive TUI
Launch the drift detection TUI for visual analysis:

```bash
# Launch default Gobang-style TUI
driftmgr

# Launch with specific TUI style
driftmgr --modern-tui
driftmgr --simple-tui

# Launch drift-specific TUI (when available)
driftmgr --drift-tui
```

TUI Features:
- Three-panel layout: Backends | Drift Analysis | Remediation
- Keyboard navigation (vim-style)
- Real-time drift detection
- Color-coded severity levels
- Export capabilities

### 6. Multi-Workspace Support
Handle Terraform workspaces:

```bash
# Detect drift for specific workspace
driftmgr drift detect --state ./terraform.tfstate --workspace staging

# Scan all workspaces
driftmgr scan --dir . --all-workspaces
```

### 7. CI/CD Integration
Integrate drift detection into your pipeline:

```yaml
# GitHub Actions Example
- name: Detect Terraform Drift
  run: |
    driftmgr drift detect \
      --state s3://${{ secrets.STATE_BUCKET }}/terraform.tfstate \
      --provider aws \
      --format json > drift.json
    
    # Fail if critical drift detected
    if [ $(jq '.summary.critical_drifts' drift.json) -gt 0 ]; then
      echo "Critical drift detected!"
      exit 1
    fi
```

```groovy
// Jenkins Pipeline Example
stage('Drift Detection') {
    steps {
        sh '''
            driftmgr drift detect \
                --state ${STATE_FILE} \
                --provider ${CLOUD_PROVIDER} \
                --severity high
        '''
    }
}
```

### 8. Advanced Filtering
Filter resources during drift detection:

```bash
# Filter by resource type
driftmgr drift detect --state ./terraform.tfstate --resource-type aws_instance

# Filter by tags
driftmgr drift detect --state ./terraform.tfstate --tag Environment=Production

# Multiple filters
driftmgr drift detect \
  --state ./terraform.tfstate \
  --resource-type aws_s3_bucket \
  --tag Team=DevOps \
  --severity high
```

## Common Use Cases

### Daily Drift Check
```bash
#!/bin/bash
# daily-drift-check.sh

BACKENDS=$(driftmgr scan --dir ./infrastructure --format json)

for backend in $(echo $BACKENDS | jq -r '.backends[].path'); do
    echo "Checking drift for $backend"
    driftmgr drift detect --state $backend --format summary
done
```

### Pre-Deployment Validation
```bash
# Ensure no drift before deployment
driftmgr drift detect --state ./terraform.tfstate --severity critical
if [ $? -ne 0 ]; then
    echo "Critical drift detected. Please resolve before deployment."
    exit 1
fi
```

### Compliance Reporting
```bash
# Generate monthly compliance report
driftmgr drift report \
  --state s3://company-state/prod.tfstate \
  --format json \
  --output compliance-$(date +%Y%m).json
```

## Resource Type Mappings

DriftMgr automatically maps between Terraform and cloud provider resource types:

| Terraform Type | AWS Type | Azure Type | GCP Type |
|----------------|----------|------------|----------|
| aws_instance | EC2 Instance | - | - |
| azurerm_virtual_machine | - | Virtual Machine | - |
| google_compute_instance | - | - | Compute Instance |
| aws_s3_bucket | S3 Bucket | - | - |
| azurerm_storage_account | - | Storage Account | - |
| google_storage_bucket | - | - | Storage Bucket |

## Troubleshooting

### No backends found
- Ensure .tf files are in the scanned directory
- Check that backend blocks are properly formatted
- Verify file permissions

### Drift detection fails
- Verify cloud credentials are configured
- Ensure state file is accessible
- Check network connectivity to cloud APIs

### TUI not launching
- Verify terminal supports required features
- Try different TUI modes (--simple-tui)
- Check terminal size (minimum 80x24)

## Best Practices

1. **Regular Scanning**: Run drift detection daily or before deployments
2. **Severity Levels**: Focus on critical and high severity drift first
3. **Documentation**: Export reports for audit trails
4. **Automation**: Integrate into CI/CD pipelines
5. **Remediation**: Review remediation plans before applying
6. **Workspace Management**: Track drift across all workspaces
7. **Version Control**: Store drift reports in version control

## Conclusion

DriftMgr provides complete Terraform drift detection capabilities, helping you maintain infrastructure consistency and compliance. Use it as part of your regular infrastructure management workflow to catch and resolve drift before it causes issues.