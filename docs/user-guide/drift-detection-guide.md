# DriftMgr - Terraform State Drift Detection Strategy

## Current State Analysis
We have a good foundation with:
- [OK] Remote state backend support (S3, Azure, GCS)
- [OK] State file parsing capabilities
- [OK] Cloud resource discovery (AWS, Azure, GCP, DigitalOcean)
- [OK] Resource models and structures
- [OK] TUI for visualization

## Implementation Strategy

### Phase 1: Backend Discovery
**Goal**: Automatically discover Terraform state backends

```go
// 1. Scan for terraform configuration files
- Search for *.tf files in workspace
- Parse backend {} blocks
- Extract backend configuration

// 2. Common backend patterns to detect:
- S3: bucket, key, region, dynamodb_table
- Azure: storage_account, container, key
- GCS: bucket, prefix
- Terraform Cloud: organization, workspaces
- Local: path

// 3. Multi-workspace support
- Detect terraform.tfstate.d/ directories
- Parse workspace configurations
- Handle env-based state paths
```

### Phase 2: State File Analysis
**Goal**: Parse and understand Terraform state contents

```go
// Structure we need to parse:
type TerraformState struct {
 Version int
 TerraformVersion string
 Serial int
 Lineage string
 Resources []StateResource
}

type StateResource struct {
 Module string
 Mode string // managed, data
 Type string // aws_instance, azure_vm, etc.
 Name string
 Provider string
 Instances []ResourceInstance
}

type ResourceInstance struct {
 SchemaVersion int
 Attributes map[string]interface{}
 Dependencies []string
}
```

### Phase 3: Cloud Resource Comparison
**Goal**: Compare state vs. actual deployed resources

```go
// Drift Detection Engine:
1. Load Terraform state
2. Discover actual cloud resources
3. Map state resources to cloud resources
4. Compare attributes
5. Identify:
 - Resources in state but not in cloud (deleted)
 - Resources in cloud but not in state (unmanaged)
 - Resources with different configurations (drifted)
```

### Phase 4: Drift Analysis Types

#### 1. **Configuration Drift**
```go
type ConfigDrift struct {
 ResourceID string
 ResourceType string
 Attribute string
 StateValue interface{}
 ActualValue interface{}
 Severity string // Critical, High, Medium, Low
}
```

#### 2. **Resource Drift**
```go
type ResourceDrift struct {
 Type string // Missing, Unmanaged, Modified
 ResourceID string
 Provider string
 Details map[string]interface{}
}
```

#### 3. **Cost Drift**
```go
type CostDrift struct {
 ExpectedCost float64 // Based on state
 ActualCost float64 // Based on discovery
 Difference float64
 Percentage float64
}
```

## Proposed Implementation

### 1. Create Terraform Backend Scanner
```go
// internal/terraform/backend_scanner.go
type BackendScanner struct {
 workDir string
 configs []BackendConfig
}

func (bs *BackendScanner) ScanDirectory(dir string) ([]BackendConfig, error)
func (bs *BackendScanner) ParseBackendBlock(content string) (*BackendConfig, error)
func (bs *BackendScanner) ValidateBackend(config *BackendConfig) error
```

### 2. Enhance State Loader
```go
// internal/state/terraform_state_parser.go
type TerraformStateParser struct {
 loader *StateLoader
}

func (tsp *TerraformStateParser) ParseStateFile(data []byte) (*TerraformState, error)
func (tsp *TerraformStateParser) ExtractResources(state *TerraformState) []ManagedResource
func (tsp *TerraformStateParser) GetResourceAttributes(resource StateResource) map[string]interface{}
```

### 3. Create Drift Detector
```go
// internal/drift/detector.go
type DriftDetector struct {
 stateParser *TerraformStateParser
 cloudDiscovery *discovery.EnhancedDiscovery
 comparisonRules map[string]ComparisonRule
}

func (dd *DriftDetector) DetectDrift(stateFile *StateFile, cloudResources []Resource) (*DriftReport, error)
func (dd *DriftDetector) CompareResource(stateRes, cloudRes Resource) []ConfigDrift
func (dd *DriftDetector) IdentifyUnmanagedResources(state, cloud []Resource) []Resource
```

### 4. Update TUI for Drift Visualization
```go
// Show drift in the Gobang-style TUI:

 State Files Resources (State vs Cloud) Drift Details

 > backend.tf vpc-main [MATCH] No drift detected
 prod.tfst ec2-web-01 [DRIFT] Instance Type Changed:
 staging rds-backup [MISSING] State: t2.micro
 + s3-logs [UNMGD] Cloud: t3.micro

```

## Integration Points

### 1. With Existing Discovery
```go
// Enhance discovery to tag resources with Terraform metadata
type EnhancedResource struct {
 Resource
 TerraformManaged bool
 TerraformID string
 StateFile string
}
```

### 2. With Existing Analysis
```go
// Add drift analysis to existing analysis engine
func (a *Analyzer) AnalyzeDrift(resources []Resource, stateFiles []StateFile) *DriftAnalysis
```

### 3. With Remediation
```go
// Generate Terraform code to fix drift
func (r *Remediator) GenerateTerraformPlan(drift *DriftReport) string
func (r *Remediator) ApplyStateCorrection(drift *DriftReport) error
```

## Workflow Example

```bash
# 1. Scan for Terraform backends
driftmgr scan --dir ./infrastructure

# 2. Load state files
driftmgr state list
driftmgr state show prod.tfstate

# 3. Detect drift
driftmgr drift detect --state prod.tfstate --provider aws

# 4. Analyze drift
driftmgr drift analyze --severity high

# 5. Generate remediation
driftmgr drift remediate --output terraform.tf
```

## Benefits of This Approach

1. **Complete**: Covers all major Terraform backends
2. **Accurate**: Direct comparison of state vs reality
3. **Actionable**: Provides clear drift details and fixes
4. **Integrated**: Works with existing discovery code
5. **Visual**: Clear representation in TUI

## Next Steps

1. Implement backend scanner
2. Enhance state parser for Terraform specifics
3. Create drift detection engine
4. Update TUI to show drift results
5. Add remediation capabilities
6. Test with real Terraform projects

## Resource Mapping Strategy

### AWS Mapping
```
Terraform Type -> AWS Resource Type
aws_instance -> EC2 Instance
aws_s3_bucket -> S3 Bucket
aws_rds_instance -> RDS DB Instance
aws_vpc -> VPC
```

### Azure Mapping
```
azurerm_virtual_machine -> Virtual Machine
azurerm_storage_account -> Storage Account
azurerm_sql_database -> SQL Database
```

### GCP Mapping
```
google_compute_instance -> Compute Instance
google_storage_bucket -> Storage Bucket
google_sql_database_instance -> Cloud SQL Instance
```

This approach leverages our existing code while adding the specific Terraform state drift detection capabilities that make DriftMgr valuable for Terraform users.