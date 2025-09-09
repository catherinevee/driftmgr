# DriftMgr "Perspective" Feature - Out-of-Band Resource Detection

## Overview
DriftMgr includes a perspective feature that identifies resources that are "out of band" - resources that exist in your cloud infrastructure but are NOT managed by Terraform (not in your tfstate file).

## How It Works

### Resource Categories Detected by DriftMgr:

1. **Managed Resources** (In State + In Cloud) [OK]
 - Resources properly tracked in tfstate
 - Under Terraform control
 - Can be updated via `terraform apply`

2. **Missing Resources** (In State, Not in Cloud) [ERROR]
 - Resources that were deleted outside Terraform
 - Need to be recreated or removed from state
 - `missing_count` in drift report

3. **Unmanaged/Out-of-Band Resources** (Not in State, In Cloud) [WARNING]
 - Resources created manually via console/CLI
 - Not under Terraform control
 - Show as `unmanaged_count` in drift report
 - **This is the "perspective" feature**

4. **Modified Resources** (In State + In Cloud, Different Config)
 - Resources that exist but have been changed
 - Need reconciliation
 - `drifted_count` in drift report

## Using the Perspective Feature

### Command Structure:
```bash
driftmgr drift detect --state <tfstate-file> --provider <provider>
```

### Output Fields:
```json
{
 "unmanaged_count": 5, // Resources NOT in tfstate but IN cloud
 "missing_count": 2, // Resources IN tfstate but NOT in cloud
 "drifted_count": 3, // Resources with configuration changes
 "total_resources": 10 // Total resources discovered
}
```

## Example Scenarios

### Scenario 1: Empty State File
```bash
# Using minimal.tfstate (empty resources array)
driftmgr drift detect --state minimal.tfstate --provider aws

# Result: ALL cloud resources show as unmanaged
# unmanaged_count = total AWS resources
```

### Scenario 2: Partial State File
```bash
# State has only VPC, but AWS has VPC + S3 + EC2
driftmgr drift detect --state partial.tfstate --provider aws

# Result:
# - VPC: managed (or drifted if changed)
# - S3: unmanaged (out-of-band)
# - EC2: unmanaged (out-of-band)
```

### Scenario 3: Complete State with Extra Resources
```bash
# Someone created resources manually after terraform apply
driftmgr drift detect --state production.tfstate --provider aws

# Result shows resources created outside Terraform workflow
```

## Benefits of Perspective Feature

1. **Shadow IT Detection**
 - Find resources created outside approved workflows
 - Identify cost centers not tracked by IaC

2. **Compliance Auditing**
 - Ensure all resources are under Terraform management
 - Detect manual "hotfixes" that bypass change control

3. **Cost Management**
 - Discover forgotten or orphaned resources
 - Find resources not tagged by Terraform

4. **Migration Planning**
 - Identify resources to import into Terraform
 - Plan infrastructure-as-code adoption

## Remediation Options

### For Unmanaged Resources:

1. **Import into Terraform**
 ```bash
 driftmgr import --resource <resource-id> --type <resource-type>
 ```

2. **Delete if Unauthorized**
 ```bash
 driftmgr delete --resource <resource-id>
 ```

3. **Document as Exception**
 - Add to excluded_resources in auto-remediation config

## Auto-Remediation Rule

DriftMgr includes a built-in rule for unmanaged resources:

```yaml
- name: import-unmanaged-resources
 description: Import unmanaged resources into Terraform state
 drift_types:
 - unmanaged
 action:
 type: auto_fix
 strategy: terraform
```

## Best Practices

1. **Regular Scans**: Run perspective analysis weekly
2. **Alert on Unmanaged**: Set up notifications for new out-of-band resources
3. **Import Promptly**: Import legitimate resources immediately
4. **Document Exceptions**: Maintain list of intentionally unmanaged resources
5. **Enforce IaC**: Use perspective reports to enforce infrastructure-as-code policies

## Summary

The perspective feature provides complete visibility into your cloud infrastructure by comparing:
- What Terraform thinks exists (tfstate)
- What actually exists (cloud provider API)

This "outside perspective" helps maintain infrastructure integrity and ensures all resources are properly managed and tracked.