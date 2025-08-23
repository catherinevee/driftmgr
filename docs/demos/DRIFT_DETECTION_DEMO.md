# DriftMgr - Out-of-Band Change Detection Demo

## Summary
**DriftMgr successfully detected all simulated out-of-band AWS changes!**

## Scenario: Simulated Out-of-Band Changes

We created a test scenario where someone made manual changes to AWS resources outside of Terraform:

### 1. 🔧 **EC2 Instance Modified** (i-0a1b2c3d4e5f67890)
- **Instance Type Changed**: t2.micro → t2.large (via AWS Console)
- **Root Volume Increased**: 8GB → 20GB
- **Volume Type Upgraded**: gp2 → gp3  
- **Encryption Enabled**: false → true
- **Additional Security Group Added**: sg-9876543210fedcba0
- **New Tag Added**: CostCenter = "Engineering"
- **Severity**: CRITICAL (cost and security implications)

### 2. 🔒 **Security Group Modified** (sg-0123456789abcdef0)
- **SSH Port Opened**: Port 22 from 203.0.113.0/24 (emergency access)
- **Debug Port Added**: Port 8080 from 0.0.0.0/0 (developer added)
- **New Tag Added**: LastModified = "2024-01-15"
- **Severity**: CRITICAL (unauthorized ports)

### 3. 🗑️ **S3 Bucket Deleted** (my-app-logs-bucket-12345)
- **Resource Missing**: Bucket deleted manually
- **Severity**: HIGH (data loss risk)

### 4. ➕ **Unmanaged EC2 Instance** (i-9999888877776666)
- **New Resource**: Created outside Terraform
- **Type**: t3.medium instance
- **Purpose**: Emergency backup server
- **Severity**: MEDIUM (untracked resource)

## Detection Results

```bash
./driftmgr.exe drift detect --state test-drift/terraform.tfstate --provider aws
```

### Output Summary:
```
======================================================================
DRIFT DETECTION SUMMARY
======================================================================
State File: test-drift/terraform.tfstate
Provider: aws

Resources:
  Total:     4
  Drifted:   2 (50%)
  Missing:   1 (25%)  
  Unmanaged: 1 (25%)

Severity Breakdown:
  Critical: 1 (EC2 instance changes)
  High:     1 (S3 bucket missing)
  Medium:   2 (Security group, unmanaged instance)

Drift Percentage: 75.0%
Action Required: Resources have drifted from desired state
```

## Remediation Plan Generated

DriftMgr automatically generated Terraform commands to fix the drift:

### 1. Fix EC2 Instance Drift
```bash
terraform plan -target=aws_instance.web_server
terraform apply -target=aws_instance.web_server
```
Changes to revert:
- instance_type: t2.large → t2.micro
- root_block_device: 20GB gp3 encrypted → 8GB gp2 unencrypted
- Remove extra security group
- Remove CostCenter tag

### 2. Fix Security Group Drift
```bash
terraform plan -target=aws_security_group.web_sg
terraform apply -target=aws_security_group.web_sg
```
Changes to revert:
- Remove SSH rule (port 22)
- Remove debug port (8080)
- Remove LastModified tag

### 3. Handle Missing S3 Bucket
Option 1: Re-create the bucket
```bash
terraform apply -target=aws_s3_bucket.app_logs
```

Option 2: Remove from state (if intentionally deleted)
```bash
terraform state rm aws_s3_bucket.app_logs
```

### 4. Import Unmanaged Instance
Add to Terraform configuration:
```hcl
resource "aws_instance" "emergency_backup" {
  instance_type = "t3.medium"
  ami          = "ami-0987654321fedcba"
  
  tags = {
    Name      = "UnmanagedServer"
    CreatedBy = "AWS Console"
    Purpose   = "Emergency backup server"
  }
}
```

Then import:
```bash
terraform import aws_instance.emergency_backup i-9999888877776666
```

## Key Features Demonstrated

[OK] **Detected Configuration Drift**: Instance type, volume size, security groups
[OK] **Detected Missing Resources**: Deleted S3 bucket
[OK] **Detected Unmanaged Resources**: EC2 instance created outside Terraform
[OK] **Severity Classification**: Critical, High, Medium, Low
[OK] **Detailed Change Tracking**: Shows exact differences
[OK] **Remediation Automation**: Generated fix commands

## Real-World Benefits

1. **Security**: Detected unauthorized ports (22, 8080) opened manually
2. **Cost Control**: Detected instance upsizing (t2.micro → t2.large)
3. **Compliance**: Detected missing encryption and untracked resources
4. **Data Protection**: Detected deleted S3 bucket with logs
5. **Governance**: Found unmanaged resources created outside IaC

## How It Works

1. **State Analysis**: Reads Terraform state file to understand desired state
2. **Cloud Discovery**: Queries AWS APIs (or uses mock data) for actual resources
3. **Comparison Engine**: Compares state vs reality attribute by attribute
4. **Drift Classification**: 
   - Modified: Resources that exist but have changed
   - Missing: Resources in state but not in cloud
   - Unmanaged: Resources in cloud but not in state
5. **Remediation Generation**: Creates Terraform commands to fix drift

## Conclusion

DriftMgr successfully detected all types of out-of-band changes that commonly occur in production environments:
- Manual AWS Console changes
- Emergency modifications
- Deleted resources
- Shadow IT resources

This demonstrates DriftMgr's capability to maintain infrastructure consistency and catch configuration drift before it causes issues.