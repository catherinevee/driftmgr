# Multi-Subscription CLI Verification Enhancement

## Overview

This document summarizes the enhancement of DriftMgr's CLI verification feature to support multiple subscriptions and accounts across cloud providers. The enhancement allows DriftMgr to iterate through multiple subscriptions for each provider and perform CLI-based verification with proper account context.

## Key Features Added

### 1. Multi-Subscription Support
- **Account Context Awareness**: CLI verification now understands and respects the account/subscription context when switching between different cloud accounts
- **Provider-Specific Account Handling**: 
  - AWS: Uses `--profile` parameter for account switching
  - Azure: Uses `--subscription` parameter for subscription switching
  - GCP: Uses `--project` parameter for project switching (framework ready)
  - DigitalOcean: Uses `--access-token` parameter for account switching (framework ready)

### 2. Enhanced CLI Verification Methods
- **`VerifyResourcesWithAccount()`**: New method that accepts account context
- **Account-Specific Verification**: Each resource type now has account-aware verification methods
- **Proper Account Switching**: CLI commands are executed with the correct account context

### 3. Multi-Subscription Discovery Integration
- **Seamless Integration**: CLI verification is automatically performed for each account/subscription during discovery
- **Account Information Tracking**: Resources are tagged with account information for proper verification
- **Progress Tracking**: Progress bar shows account-specific verification status

## Technical Implementation

### 1. Enhanced CLI Verification Structure

```go
// New methods added to CLIVerifier
func (cv *CLIVerifier) VerifyResourcesWithAccount(resources []models.Resource, provider, region, accountID string) []CLIVerificationResult

// Account-aware verification methods for each provider
func (cv *CLIVerifier) verifyAWSResourcesWithAccount(resources []models.Resource, resourceType, region, accountID string)
func (cv *CLIVerifier) verifyAzureResourcesWithAccount(resources []models.Resource, resourceType, region, subscriptionID string)
func (cv *CLIVerifier) verifyGCPResourcesWithAccount(resources []models.Resource, resourceType, region, projectID string)
func (cv *CLIVerifier) verifyDigitalOceanResourcesWithAccount(resources []models.Resource, resourceType, region, accountID string)
```

### 2. Account-Specific CLI Commands

#### AWS Account Support
```bash
# With account context
aws --profile <account-id> ec2 describe-instances --region <region>
aws --profile <account-id> s3api list-buckets
aws --profile <account-id> rds describe-db-instances --region <region>
```

#### Azure Subscription Support
```bash
# With subscription context
az --subscription <subscription-id> vm list --query "[?location=='<region>']"
az --subscription <subscription-id> storage account list --query "[?location=='<region>']"
az --subscription <subscription-id> network vnet list --query "[?location=='<region>']"
```

### 3. Enhanced Discovery Integration

```go
// Updated verification call in multi-account discovery
verificationResults := ed.verifyDiscoveredResourcesWithCLIAndAccount(validResources, provider, region, account.ID)
```

## Supported Resource Types

### AWS Resources (with account context)
- EC2 Instances
- S3 Buckets
- VPCs
- RDS Instances
- Lambda Functions
- IAM Users
- IAM Roles

### Azure Resources (with subscription context)
- Virtual Machines
- Storage Accounts
- Virtual Networks
- SQL Databases

### GCP Resources (framework ready)
- Compute Instances
- Storage Buckets
- VPC Networks
- Cloud SQL Instances

### DigitalOcean Resources (framework ready)
- Droplets
- Spaces Buckets
- VPC Networks
- Databases

## Usage Examples

### 1. Basic Multi-Subscription Discovery
```go
// The enhanced discovery automatically handles multiple subscriptions
discoverer := discovery.NewEnhancedDiscoverer(cfg)
resources, err := discoverer.DiscoverAllResourcesEnhanced(ctx, providers, regions)
```

### 2. CLI Verification Results by Account
```go
cliVerifier := discoverer.GetCLIVerifier()
verificationResults := cliVerifier.GetResults()

// Results are automatically grouped by account/subscription
for _, result := range verificationResults {
    // Each result contains account context information
    fmt.Printf("Account: %s, Resource: %s, CLI Found: %v\n", 
        result.CLIResource["OwnerId"], result.ResourceName, result.CLIFound)
}
```

### 3. Test Multi-Subscription CLI Verification
```bash
# Run the multi-subscription test
go run test_multi_subscription_cli_verification.go
```

## Configuration

### CLI Verification Settings
```yaml
discovery:
  cli_verification:
    enabled: true
    timeout_seconds: 30
    max_retries: 3
    verbose: false
```

### Multi-Subscription Discovery
The multi-subscription functionality is automatically enabled when:
- Multiple accounts/subscriptions are detected for a provider
- The credential manager successfully switches between accounts
- CLI verification is enabled in the configuration

## Output and Reporting

### 1. Console Output
```
🔍 Multi-Subscription CLI Verification Results:
=============================================
Total resources verified: 4
Resources found in CLI: 4
Resources not found in CLI: 0
Total discrepancies: 0
Accuracy rate: 100.00%

📋 Detailed Verification Results by Account/Subscription:
=======================================================

📁 Account/Subscription: 025066254478
   Resources verified: 4
   Found in CLI: 4
   Not found in CLI: 0
   Discrepancies: 0
   Accuracy: 100.0%
```

### 2. JSON Results File
```json
{
  "timestamp": "2025-08-17T12:48:18.123456789-07:00",
  "summary": {
    "total_resources": 4,
    "cli_found": 4,
    "cli_not_found": 0,
    "discrepancies": 0,
    "accuracy_rate": 100.0
  },
  "account_results": {
    "025066254478": [
      {
        "resource_id": "vpc-0c6f3f6997666e791",
        "resource_name": "vpc-0c6f3f6997666e791",
        "resource_type": "aws_vpc",
        "provider": "aws",
        "region": "us-east-1",
        "cli_found": true,
        "cli_resource": { ... },
        "verification_time": "2025-08-17T12:41:22.1535313-07:00"
      }
    ]
  }
}
```

## Benefits

### 1. Comprehensive Verification
- **Multi-Account Coverage**: Verifies resources across all accounts/subscriptions
- **Account-Specific Accuracy**: Ensures CLI verification matches the correct account context
- **Complete Resource Validation**: No resources are missed due to account switching

### 2. Enhanced Reliability
- **Context-Aware Verification**: CLI commands are executed with proper account context
- **Accurate Discrepancy Detection**: Identifies real discrepancies vs. account-related issues
- **Trustworthy Results**: Users can rely on verification results across all accounts

### 3. Improved User Experience
- **Clear Account Separation**: Results are clearly organized by account/subscription
- **Detailed Reporting**: Comprehensive breakdown of verification results per account
- **Easy Troubleshooting**: Account-specific discrepancies are clearly identified

## Testing

### Test Scripts Available
1. **`test_multi_subscription_cli_verification.go`**: Comprehensive multi-subscription test
2. **`test_cli_verification.go`**: Basic CLI verification test
3. **`count_azure_resources.go`**: Azure-specific resource counting

### Running Tests
```bash
# Test multi-subscription CLI verification
go run test_multi_subscription_cli_verification.go

# Test basic CLI verification
go run test_cli_verification.go

# Count Azure resources
go run count_azure_resources.go
```

## Future Enhancements

### 1. Additional Provider Support
- **GCP Implementation**: Complete GCP CLI verification with project context
- **DigitalOcean Implementation**: Complete DigitalOcean CLI verification with account context
- **Other Providers**: Extend to additional cloud providers

### 2. Enhanced Account Management
- **Account Grouping**: Group accounts by organization or project
- **Account Permissions**: Verify account-specific permissions and access
- **Account Health Checks**: Validate account connectivity and configuration

### 3. Advanced Verification Features
- **Cross-Account Verification**: Compare resources across accounts for consistency
- **Account-Specific Policies**: Apply different verification rules per account
- **Automated Account Discovery**: Automatically discover and configure new accounts

## Conclusion

The multi-subscription CLI verification enhancement significantly improves DriftMgr's ability to provide accurate and comprehensive resource verification across multiple cloud accounts and subscriptions. This enhancement ensures that users can trust the verification results regardless of their multi-account cloud infrastructure complexity.

The implementation maintains backward compatibility while adding powerful new capabilities for enterprise environments with multiple accounts and subscriptions across different cloud providers.
