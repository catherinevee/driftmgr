# Terraform State Drift Simulation Report

## Overview
We successfully simulated infrastructure drift by making manual changes to AWS resources via AWS CLI that differ from the Terraform state file.

## Initial State (drift-test.tfstate)
```json
{
  "bucket": "driftmgr-test-bucket-1755579790",
  "versioning": {
    "enabled": false
  },
  "tags": {
    "Name": "DriftMgr Test Bucket",
    "Environment": "test",
    "ManagedBy": "Terraform"
  },
  "lifecycle_rule": []
}
```

## Manual Changes via AWS CLI

### 1. **Modified Tags** [ERROR] DRIFT
```bash
aws s3api put-bucket-tagging --bucket driftmgr-test-bucket-1755579790 \
  --tagging 'TagSet=[
    {Key=Name,Value="Modified Bucket Name"},       # Changed from "DriftMgr Test Bucket"
    {Key=Environment,Value=production},            # Changed from "test"
    {Key=Team,Value=DevOps},                      # NEW TAG
    {Key=ModifiedBy,Value=AWS-CLI}                # NEW TAG
  ]'
```
**Result:** Tags no longer match Terraform state
- [ERROR] `Name`: "DriftMgr Test Bucket" → "Modified Bucket Name"
- [ERROR] `Environment`: "test" → "production"
- [ERROR] `ManagedBy`: "Terraform" → DELETED
- [ERROR] `Team`: Not in state → "DevOps" (ADDED)
- [ERROR] `ModifiedBy`: Not in state → "AWS-CLI" (ADDED)

### 2. **Enabled Versioning** [ERROR] DRIFT
```bash
aws s3api put-bucket-versioning --bucket driftmgr-test-bucket-1755579790 \
  --versioning-configuration Status=Enabled
```
**Result:** Versioning changed from `disabled` to `enabled`

### 3. **Added Lifecycle Rule** [ERROR] DRIFT
```bash
aws s3api put-bucket-lifecycle-configuration --bucket driftmgr-test-bucket-1755579790 \
  --lifecycle-configuration file://lifecycle.json
```
**Result:** Lifecycle rule added (was empty in state)

## Current AWS State vs Terraform State

| Property | Terraform State | Current AWS State | Drift? |
|----------|----------------|-------------------|--------|
| **Bucket Name** | driftmgr-test-bucket-1755579790 | driftmgr-test-bucket-1755579790 | [OK] No |
| **Versioning** | Disabled | **Enabled** | [ERROR] Yes |
| **Tag: Name** | "DriftMgr Test Bucket" | **"Modified Bucket Name"** | [ERROR] Yes |
| **Tag: Environment** | "test" | **"production"** | [ERROR] Yes |
| **Tag: ManagedBy** | "Terraform" | **[DELETED]** | [ERROR] Yes |
| **Tag: Team** | [none] | **"DevOps"** | [ERROR] Yes |
| **Tag: ModifiedBy** | [none] | **"AWS-CLI"** | [ERROR] Yes |
| **Lifecycle Rules** | [] (empty) | **[delete-old-objects]** | [ERROR] Yes |

## Verification Commands

### Check Current Tags:
```bash
aws s3api get-bucket-tagging --bucket driftmgr-test-bucket-1755579790
```

### Check Versioning Status:
```bash
aws s3api get-bucket-versioning --bucket driftmgr-test-bucket-1755579790
```

### Check Lifecycle Rules:
```bash
aws s3api get-bucket-lifecycle-configuration --bucket driftmgr-test-bucket-1755579790
```

## Summary

We successfully created **7 instances of drift** on a single S3 bucket:
1. Changed tag value: Name
2. Changed tag value: Environment
3. Deleted tag: ManagedBy
4. Added new tag: Team
5. Added new tag: ModifiedBy
6. Enabled versioning (was disabled)
7. Added lifecycle rule (was empty)

This demonstrates how manual changes outside of Terraform create configuration drift that needs to be detected and remediated to maintain infrastructure as code consistency.