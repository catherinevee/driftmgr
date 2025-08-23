# DriftMgr Verification Methods

## How DriftMgr Verifies Its Detection Accuracy

DriftMgr uses multiple layers of verification to ensure accurate resource detection and drift analysis:

## 1. **Native CLI Cross-Verification** (`internal/discovery/cli_verification.go`)

DriftMgr compares its discovery results against native cloud provider CLI tools to ensure accuracy:

### AWS Verification
- **AWS CLI Commands Used:**
  - `aws ec2 describe-instances` - Verify EC2 instances
  - `aws s3api list-buckets` - Verify S3 buckets  
  - `aws ec2 describe-vpcs` - Verify VPCs
  - `aws rds describe-db-instances` - Verify RDS databases
  - `aws lambda list-functions` - Verify Lambda functions
  - `aws iam list-users` - Verify IAM users
  - `aws iam list-roles` - Verify IAM roles

### Azure Verification
- **Azure CLI Commands Used:**
  - `az vm list` - Verify virtual machines
  - `az storage account list` - Verify storage accounts
  - `az network vnet list` - Verify virtual networks
  - `az sql server list` - Verify SQL servers
  - `az postgres server list` - Verify PostgreSQL servers

### GCP Verification  
- **GCloud CLI Commands Used:**
  - `gcloud compute instances list` - Verify compute instances
  - `gcloud storage buckets list` - Verify storage buckets
  - `gcloud compute networks list` - Verify networks
  - `gcloud sql instances list` - Verify SQL instances

### DigitalOcean Verification
- **doctl Commands Used:**
  - `doctl compute droplet list` - Verify droplets
  - `doctl compute volume list` - Verify volumes
  - `doctl database list` - Verify databases

## 2. **Resource Count Validation** (`internal/discovery/resource_count_validator.go`)

DriftMgr performs systematic validation by:

1. **Counting Resources**: Compares total counts between DriftMgr and native CLI
2. **Region-by-Region Validation**: Validates each region separately
3. **Resource Type Analysis**: Breaks down counts by resource type
4. **Discrepancy Detection**: Identifies:
   - Resources missing in DriftMgr (`MissingInDM`)
   - Extra resources in DriftMgr (`ExtraInDM`)
   - Mismatched attributes

### Validation Result Structure
```go
type ValidationResult struct {
    Provider       string          // Cloud provider
    Region         string          // Region being validated
    DriftmgrCount  int            // Resources found by DriftMgr
    CLICount       int            // Resources found by CLI
    Match          bool           // Whether counts match
    MissingInDM    []string       // Resources CLI found but DriftMgr didn't
    ExtraInDM      []string       // Resources DriftMgr found but CLI didn't
}
```

## 3. **Terraform State Comparison** (`internal/drift/terraform_drift_detector.go`)

For Terraform-managed infrastructure, DriftMgr verifies by:

1. **Loading Terraform State**: Reads `.tfstate` files (local or remote)
2. **Actual Resource Discovery**: Discovers current cloud resources
3. **Attribute-by-Attribute Comparison**: 
   - Compares each attribute in state vs actual
   - Identifies modified values
   - Detects missing resources
   - Finds unmanaged resources

### Drift Detection Process
```go
// For each resource in Terraform state:
1. Find corresponding cloud resource by ID/Name
2. Compare attributes:
   - State Value vs Actual Value
   - Calculate severity of differences
3. Classify drift type:
   - Modified: Resource exists but attributes differ
   - Missing: Resource in state but not in cloud
   - Unmanaged: Resource in cloud but not in state
```

## 4. **Multi-Level Discrepancy Analysis** (`internal/discovery/cli_verification.go`)

DriftMgr categorizes discrepancies by severity:

- **Critical**: Missing resources, wrong regions, security misconfigurations
- **Warning**: Size/capacity differences, tag mismatches
- **Info**: Timestamp differences, metadata variations

### Example Verification Flow
```go
type Discrepancy struct {
    Field       string      // e.g., "instance_type"
    DriftMgr    interface{} // e.g., "t2.micro"
    CLI         interface{} // e.g., "t3.micro"
    Severity    string      // "critical", "warning", "info"
}
```

## 5. **Account/Subscription Context Verification**

DriftMgr supports multi-account verification:
- AWS: Verifies across multiple AWS accounts
- Azure: Verifies across multiple subscriptions
- GCP: Verifies across multiple projects

## 6. **Smart Defaults Validation** (`internal/drift/smart_defaults.go`)

DriftMgr filters out "harmless" drift by:
- Ignoring auto-generated tags
- Filtering timestamp-only changes
- Excluding provider-managed attributes
- Environment-aware thresholds (prod/staging/dev)

## 7. **Continuous Verification Features**

### Retry Logic
- Automatic retries for transient API failures
- Configurable timeout settings
- Exponential backoff for rate limits

### Result Aggregation
```go
// Verification results are aggregated for reporting:
- Total resources discovered
- Verification success rate
- Discrepancy patterns
- Provider-specific issues
```

## Usage Examples

### Run Validation Command
```bash
# Validate AWS discovery
driftmgr validate --provider aws

# Validate all providers
driftmgr validate --provider all --output report.txt

# Validate specific region
driftmgr validate --provider azure --region eastus
```

### Enable CLI Verification in Discovery
```go
verifier := NewCLIVerifier(
    enabled: true,
    timeout: 2*time.Minute,
    maxRetries: 3,
    verbose: true,
)
results := verifier.VerifyResources(resources, "aws", "us-east-1")
```

## Verification Guarantees

1. **Accuracy**: Cross-references with native CLI tools
2. **Completeness**: Validates all resource types and regions
3. **Consistency**: Ensures state matches reality
4. **Performance**: Parallel verification for speed
5. **Reliability**: Retry logic for transient failures

## Key Benefits

- **Trust**: Users can trust DriftMgr's discovery results
- **Debugging**: Clear identification of discrepancies
- **Compliance**: Verification reports for auditing
- **Continuous Improvement**: Identifies areas needing enhancement