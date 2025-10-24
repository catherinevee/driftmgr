package models

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// ResourceType represents the type of cloud resource
type ResourceType string

const (
	// AWS Resource Types
	ResourceTypeAWSEC2Instance    ResourceType = "aws_instance"
	ResourceTypeAWSS3Bucket       ResourceType = "aws_s3_bucket"
	ResourceTypeAWSRDSInstance    ResourceType = "aws_db_instance"
	ResourceTypeAWSLambdaFunction ResourceType = "aws_lambda_function"
	ResourceTypeAWSVPC            ResourceType = "aws_vpc"
	ResourceTypeAWSSecurityGroup  ResourceType = "aws_security_group"
	ResourceTypeAWSSubnet         ResourceType = "aws_subnet"
	ResourceTypeAWSIAMRole        ResourceType = "aws_iam_role"
	ResourceTypeAWSIAMPolicy      ResourceType = "aws_iam_policy"
	ResourceTypeAWSCloudFormation ResourceType = "aws_cloudformation_stack"

	// Azure Resource Types
	ResourceTypeAzureVM             ResourceType = "azurerm_virtual_machine"
	ResourceTypeAzureStorageAccount ResourceType = "azurerm_storage_account"
	ResourceTypeAzureSQLDatabase    ResourceType = "azurerm_sql_database"
	ResourceTypeAzureFunctionApp    ResourceType = "azurerm_function_app"
	ResourceTypeAzureVNet           ResourceType = "azurerm_virtual_network"
	ResourceTypeAzureNSG            ResourceType = "azurerm_network_security_group"
	ResourceTypeAzureSubnet         ResourceType = "azurerm_subnet"
	ResourceTypeAzureResourceGroup  ResourceType = "azurerm_resource_group"
	ResourceTypeAzureKeyVault       ResourceType = "azurerm_key_vault"
	ResourceTypeAzureAppService     ResourceType = "azurerm_app_service"

	// GCP Resource Types
	ResourceTypeGCPComputeInstance ResourceType = "google_compute_instance"
	ResourceTypeGCPStorageBucket   ResourceType = "google_storage_bucket"
	ResourceTypeGCPSQLInstance     ResourceType = "google_sql_database_instance"
	ResourceTypeGCPCloudFunction   ResourceType = "google_cloudfunctions_function"
	ResourceTypeGCPVPCNetwork      ResourceType = "google_compute_network"
	ResourceTypeGCPFirewall        ResourceType = "google_compute_firewall"
	ResourceTypeGCPSubnet          ResourceType = "google_compute_subnetwork"
	ResourceTypeGCPProject         ResourceType = "google_project"
	ResourceTypeGCPIAMRole         ResourceType = "google_project_iam_custom_role"
	ResourceTypeGCPIAMPolicy       ResourceType = "google_project_iam_policy"

	// DigitalOcean Resource Types
	ResourceTypeDODroplet      ResourceType = "digitalocean_droplet"
	ResourceTypeDOVolume       ResourceType = "digitalocean_volume"
	ResourceTypeDODatabase     ResourceType = "digitalocean_database"
	ResourceTypeDOLoadBalancer ResourceType = "digitalocean_loadbalancer"
	ResourceTypeDOVPC          ResourceType = "digitalocean_vpc"
	ResourceTypeDOFirewall     ResourceType = "digitalocean_firewall"
	ResourceTypeDOKubernetes   ResourceType = "digitalocean_kubernetes_cluster"
	ResourceTypeDODomain       ResourceType = "digitalocean_domain"
	ResourceTypeDORecord       ResourceType = "digitalocean_record"
	ResourceTypeDOSnapshot     ResourceType = "digitalocean_droplet_snapshot"
)

// String returns the string representation of ResourceType
func (rt ResourceType) String() string {
	return string(rt)
}

// ResourceCategory represents the category of a resource
type ResourceCategory string

const (
	ResourceCategoryCompute    ResourceCategory = "compute"
	ResourceCategoryStorage    ResourceCategory = "storage"
	ResourceCategoryDatabase   ResourceCategory = "database"
	ResourceCategoryNetworking ResourceCategory = "networking"
	ResourceCategorySecurity   ResourceCategory = "security"
	ResourceCategoryMonitoring ResourceCategory = "monitoring"
	ResourceCategoryManagement ResourceCategory = "management"
	ResourceCategoryServerless ResourceCategory = "serverless"
	ResourceCategoryContainer  ResourceCategory = "container"
	ResourceCategoryAnalytics  ResourceCategory = "analytics"
	ResourceCategoryAI         ResourceCategory = "ai"
	ResourceCategoryIoT        ResourceCategory = "iot"
	ResourceCategoryOther      ResourceCategory = "other"
)

// String returns the string representation of ResourceCategory
func (rc ResourceCategory) String() string {
	return string(rc)
}

// ResourceStatus represents the status of a resource
type ResourceStatus string

const (
	ResourceStatusRunning     ResourceStatus = "running"
	ResourceStatusStopped     ResourceStatus = "stopped"
	ResourceStatusStarting    ResourceStatus = "starting"
	ResourceStatusStopping    ResourceStatus = "stopping"
	ResourceStatusTerminated  ResourceStatus = "terminated"
	ResourceStatusAvailable   ResourceStatus = "available"
	ResourceStatusUnavailable ResourceStatus = "unavailable"
	ResourceStatusCreating    ResourceStatus = "creating"
	ResourceStatusDeleting    ResourceStatus = "deleting"
	ResourceStatusUpdating    ResourceStatus = "updating"
	ResourceStatusUnknown     ResourceStatus = "unknown"
)

// String returns the string representation of ResourceStatus
func (rs ResourceStatus) String() string {
	return string(rs)
}

// ResourceMetadata represents additional metadata for a resource
type ResourceMetadata struct {
	ID                string                 `json:"id" db:"id" validate:"required,uuid"`
	ResourceID        string                 `json:"resource_id" db:"resource_id" validate:"required,uuid"`
	Category          ResourceCategory       `json:"category" db:"category"`
	Status            ResourceStatus         `json:"status" db:"status"`
	Size              string                 `json:"size,omitempty" db:"size"`
	InstanceType      string                 `json:"instance_type,omitempty" db:"instance_type"`
	OperatingSystem   string                 `json:"operating_system,omitempty" db:"operating_system"`
	Architecture      string                 `json:"architecture,omitempty" db:"architecture"`
	PublicIP          string                 `json:"public_ip,omitempty" db:"public_ip"`
	PrivateIP         string                 `json:"private_ip,omitempty" db:"private_ip"`
	Ports             []int                  `json:"ports,omitempty" db:"ports"`
	Protocols         []string               `json:"protocols,omitempty" db:"protocols"`
	Endpoints         []string               `json:"endpoints,omitempty" db:"endpoints"`
	Version           string                 `json:"version,omitempty" db:"version"`
	Runtime           string                 `json:"runtime,omitempty" db:"runtime"`
	Framework         string                 `json:"framework,omitempty" db:"framework"`
	Database          string                 `json:"database,omitempty" db:"database"`
	Engine            string                 `json:"engine,omitempty" db:"engine"`
	StorageType       string                 `json:"storage_type,omitempty" db:"storage_type"`
	StorageSize       string                 `json:"storage_size,omitempty" db:"storage_size"`
	Encryption        bool                   `json:"encryption" db:"encryption"`
	BackupEnabled     bool                   `json:"backup_enabled" db:"backup_enabled"`
	MonitoringEnabled bool                   `json:"monitoring_enabled" db:"monitoring_enabled"`
	LoggingEnabled    bool                   `json:"logging_enabled" db:"logging_enabled"`
	CustomFields      map[string]interface{} `json:"custom_fields" db:"custom_fields"`
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at" db:"updated_at"`
}

// ResourceTag represents a tag on a resource
type ResourceTag struct {
	ID         string    `json:"id" db:"id" validate:"required,uuid"`
	ResourceID string    `json:"resource_id" db:"resource_id" validate:"required,uuid"`
	Key        string    `json:"key" db:"key" validate:"required"`
	Value      string    `json:"value" db:"value" validate:"required"`
	Category   string    `json:"category,omitempty" db:"category"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// ResourceDependency represents a dependency between resources
type ResourceDependency struct {
	ID         string                 `json:"id" db:"id" validate:"required,uuid"`
	SourceID   string                 `json:"source_id" db:"source_id" validate:"required,uuid"`
	TargetID   string                 `json:"target_id" db:"target_id" validate:"required,uuid"`
	Type       DependencyType         `json:"type" db:"type" validate:"required"`
	Weight     int                    `json:"weight" db:"weight"`
	Properties map[string]interface{} `json:"properties" db:"properties"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at" db:"updated_at"`
}

// DependencyType represents the type of dependency
type DependencyType string

const (
	DependencyTypeHard     DependencyType = "hard"
	DependencyTypeSoft     DependencyType = "soft"
	DependencyTypeNetwork  DependencyType = "network"
	DependencyTypeStorage  DependencyType = "storage"
	DependencyTypeSecurity DependencyType = "security"
	DependencyTypeData     DependencyType = "data"
)

// String returns the string representation of DependencyType
func (dt DependencyType) String() string {
	return string(dt)
}

// ResourcePolicy represents a policy applied to a resource
type ResourcePolicy struct {
	ID          string            `json:"id" db:"id" validate:"required,uuid"`
	ResourceID  string            `json:"resource_id" db:"resource_id" validate:"required,uuid"`
	PolicyID    string            `json:"policy_id" db:"policy_id" validate:"required"`
	PolicyName  string            `json:"policy_name" db:"policy_name" validate:"required"`
	PolicyType  PolicyType        `json:"policy_type" db:"policy_type" validate:"required"`
	Rules       []PolicyRule      `json:"rules" db:"rules"`
	IsActive    bool              `json:"is_active" db:"is_active"`
	AppliedAt   time.Time         `json:"applied_at" db:"applied_at"`
	LastChecked time.Time         `json:"last_checked" db:"last_checked"`
	NextCheck   time.Time         `json:"next_check" db:"next_check"`
	Violations  []PolicyViolation `json:"violations" db:"violations"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
}

// PolicyType represents the type of policy
type PolicyType string

const (
	PolicyTypeSecurity    PolicyType = "security"
	PolicyTypeCompliance  PolicyType = "compliance"
	PolicyTypeCost        PolicyType = "cost"
	PolicyTypePerformance PolicyType = "performance"
	PolicyTypeBackup      PolicyType = "backup"
	PolicyTypeAccess      PolicyType = "access"
	PolicyTypeData        PolicyType = "data"
	PolicyTypeNetwork     PolicyType = "network"
)

// String returns the string representation of PolicyType
func (pt PolicyType) String() string {
	return string(pt)
}

// PolicyRule represents a rule within a policy
type PolicyRule struct {
	ID          string `json:"id" db:"id"`
	RuleID      string `json:"rule_id" db:"rule_id"`
	RuleName    string `json:"rule_name" db:"rule_name"`
	Description string `json:"description" db:"description"`
	Severity    string `json:"severity" db:"severity"`
	Condition   string `json:"condition" db:"condition"`
	Action      string `json:"action" db:"action"`
	IsEnabled   bool   `json:"is_enabled" db:"is_enabled"`
}

// PolicyViolation represents a violation of a policy rule
type PolicyViolation struct {
	ID          string     `json:"id" db:"id"`
	RuleID      string     `json:"rule_id" db:"rule_id"`
	RuleName    string     `json:"rule_name" db:"rule_name"`
	Severity    string     `json:"severity" db:"severity"`
	Description string     `json:"description" db:"description"`
	Remediation string     `json:"remediation" db:"remediation"`
	DetectedAt  time.Time  `json:"detected_at" db:"detected_at"`
	ResolvedAt  *time.Time `json:"resolved_at" db:"resolved_at"`
	IsResolved  bool       `json:"is_resolved" db:"is_resolved"`
}

// ResourceMetrics represents metrics for a resource
type ResourceMetrics struct {
	ID         string                 `json:"id" db:"id" validate:"required,uuid"`
	ResourceID string                 `json:"resource_id" db:"resource_id" validate:"required,uuid"`
	MetricType MetricType             `json:"metric_type" db:"metric_type" validate:"required"`
	Value      float64                `json:"value" db:"value"`
	Unit       string                 `json:"unit" db:"unit"`
	Timestamp  time.Time              `json:"timestamp" db:"timestamp"`
	Dimensions map[string]string      `json:"dimensions" db:"dimensions"`
	Tags       map[string]string      `json:"tags" db:"tags"`
	Metadata   map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCPU          MetricType = "cpu"
	MetricTypeMemory       MetricType = "memory"
	MetricTypeDisk         MetricType = "disk"
	MetricTypeNetwork      MetricType = "network"
	MetricTypeLatency      MetricType = "latency"
	MetricTypeThroughput   MetricType = "throughput"
	MetricTypeError        MetricType = "error"
	MetricTypeCost         MetricType = "cost"
	MetricTypeAvailability MetricType = "availability"
	MetricTypePerformance  MetricType = "performance"
)

// String returns the string representation of MetricType
func (mt MetricType) String() string {
	return string(mt)
}

// ResourceAlert represents an alert for a resource
type ResourceAlert struct {
	ID               string        `json:"id" db:"id" validate:"required,uuid"`
	ResourceID       string        `json:"resource_id" db:"resource_id" validate:"required,uuid"`
	AlertType        AlertType     `json:"alert_type" db:"alert_type" validate:"required"`
	Severity         AlertSeverity `json:"severity" db:"severity" validate:"required"`
	Title            string        `json:"title" db:"title" validate:"required"`
	Description      string        `json:"description" db:"description"`
	Condition        string        `json:"condition" db:"condition" validate:"required"`
	Threshold        float64       `json:"threshold" db:"threshold"`
	IsActive         bool          `json:"is_active" db:"is_active"`
	IsTriggeredField bool          `json:"is_triggered" db:"is_triggered"`
	TriggeredAt      *time.Time    `json:"triggered_at" db:"triggered_at"`
	ResolvedAt       *time.Time    `json:"resolved_at" db:"resolved_at"`
	CreatedAt        time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at" db:"updated_at"`
}

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeThreshold  AlertType = "threshold"
	AlertTypeAnomaly    AlertType = "anomaly"
	AlertTypeStatus     AlertType = "status"
	AlertTypeSecurity   AlertType = "security"
	AlertTypeCost       AlertType = "cost"
	AlertTypeCompliance AlertType = "compliance"
)

// String returns the string representation of AlertType
func (at AlertType) String() string {
	return string(at)
}

// AlertSeverity is defined in common.go

// Request/Response Models

// ResourceMetadataUpdateRequest represents a request to update resource metadata
type ResourceMetadataUpdateRequest struct {
	Category          *ResourceCategory      `json:"category,omitempty"`
	Status            *ResourceStatus        `json:"status,omitempty"`
	Size              *string                `json:"size,omitempty"`
	InstanceType      *string                `json:"instance_type,omitempty"`
	OperatingSystem   *string                `json:"operating_system,omitempty"`
	Architecture      *string                `json:"architecture,omitempty"`
	PublicIP          *string                `json:"public_ip,omitempty"`
	PrivateIP         *string                `json:"private_ip,omitempty"`
	Ports             []int                  `json:"ports,omitempty"`
	Protocols         []string               `json:"protocols,omitempty"`
	Endpoints         []string               `json:"endpoints,omitempty"`
	Version           *string                `json:"version,omitempty"`
	Runtime           *string                `json:"runtime,omitempty"`
	Framework         *string                `json:"framework,omitempty"`
	Database          *string                `json:"database,omitempty"`
	Engine            *string                `json:"engine,omitempty"`
	StorageType       *string                `json:"storage_type,omitempty"`
	StorageSize       *string                `json:"storage_size,omitempty"`
	Encryption        *bool                  `json:"encryption,omitempty"`
	BackupEnabled     *bool                  `json:"backup_enabled,omitempty"`
	MonitoringEnabled *bool                  `json:"monitoring_enabled,omitempty"`
	LoggingEnabled    *bool                  `json:"logging_enabled,omitempty"`
	CustomFields      map[string]interface{} `json:"custom_fields,omitempty"`
}

// ResourceTagUpdateRequest represents a request to update resource tags
type ResourceTagUpdateRequest struct {
	Tags map[string]string `json:"tags" validate:"required"`
}

// ResourcePolicyApplyRequest represents a request to apply a policy to a resource
type ResourcePolicyApplyRequest struct {
	PolicyID string `json:"policy_id" validate:"required"`
}

// ResourceMetricsRequest represents a request for resource metrics
type ResourceMetricsRequest struct {
	ResourceID string      `json:"resource_id" validate:"required,uuid"`
	MetricType *MetricType `json:"metric_type,omitempty"`
	StartTime  *time.Time  `json:"start_time,omitempty"`
	EndTime    *time.Time  `json:"end_time,omitempty"`
	Limit      int         `json:"limit" validate:"min=1,max=1000"`
	Offset     int         `json:"offset" validate:"min=0"`
}

// ResourceMetricsResponse represents the response for resource metrics
type ResourceMetricsResponse struct {
	Metrics []ResourceMetrics `json:"metrics"`
	Total   int               `json:"total"`
	Limit   int               `json:"limit"`
	Offset  int               `json:"offset"`
}

// ResourceAlertCreateRequest represents a request to create a resource alert
type ResourceAlertCreateRequest struct {
	ResourceID  string        `json:"resource_id" validate:"required,uuid"`
	AlertType   AlertType     `json:"alert_type" validate:"required"`
	Severity    AlertSeverity `json:"severity" validate:"required"`
	Title       string        `json:"title" validate:"required"`
	Description string        `json:"description"`
	Condition   string        `json:"condition" validate:"required"`
	Threshold   float64       `json:"threshold"`
}

// ResourceAlertListRequest represents a request to list resource alerts
type ResourceAlertListRequest struct {
	ResourceID  *string        `json:"resource_id,omitempty"`
	AlertType   *AlertType     `json:"alert_type,omitempty"`
	Severity    *AlertSeverity `json:"severity,omitempty"`
	IsActive    *bool          `json:"is_active,omitempty"`
	IsTriggered *bool          `json:"is_triggered,omitempty"`
	Limit       int            `json:"limit" validate:"min=1,max=1000"`
	Offset      int            `json:"offset" validate:"min=0"`
	SortBy      string         `json:"sort_by" validate:"omitempty,oneof=created_at triggered_at severity"`
	SortOrder   string         `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// ResourceAlertListResponse represents the response for listing resource alerts
type ResourceAlertListResponse struct {
	Alerts []ResourceAlert `json:"alerts"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// Validation methods

// Validate validates the ResourceMetadata struct
func (rm *ResourceMetadata) Validate() error {
	validate := validator.New()
	return validate.Struct(rm)
}

// Validate validates the ResourceTag struct
func (rt *ResourceTag) Validate() error {
	validate := validator.New()
	return validate.Struct(rt)
}

// Validate validates the ResourceDependency struct
func (rd *ResourceDependency) Validate() error {
	validate := validator.New()
	return validate.Struct(rd)
}

// Validate validates the ResourcePolicy struct
func (rp *ResourcePolicy) Validate() error {
	validate := validator.New()
	return validate.Struct(rp)
}

// Validate validates the ResourceMetrics struct
func (rm *ResourceMetrics) Validate() error {
	validate := validator.New()
	return validate.Struct(rm)
}

// Validate validates the ResourceAlert struct
func (ra *ResourceAlert) Validate() error {
	validate := validator.New()
	return validate.Struct(ra)
}

// Validate validates the ResourceMetadataUpdateRequest struct
func (rmur *ResourceMetadataUpdateRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(rmur)
}

// Validate validates the ResourceTagUpdateRequest struct
func (rtur *ResourceTagUpdateRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(rtur)
}

// Validate validates the ResourcePolicyApplyRequest struct
func (rpar *ResourcePolicyApplyRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(rpar)
}

// Validate validates the ResourceMetricsRequest struct
func (rmr *ResourceMetricsRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(rmr)
}

// Validate validates the ResourceAlertCreateRequest struct
func (racr *ResourceAlertCreateRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(racr)
}

// Validate validates the ResourceAlertListRequest struct
func (ralr *ResourceAlertListRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(ralr)
}

// Helper methods

// GetCategory returns the category for a resource type
func GetCategoryForResourceType(resourceType ResourceType) ResourceCategory {
	switch resourceType {
	case ResourceTypeAWSEC2Instance, ResourceTypeAzureVM, ResourceTypeGCPComputeInstance, ResourceTypeDODroplet:
		return ResourceCategoryCompute
	case ResourceTypeAWSS3Bucket, ResourceTypeAzureStorageAccount, ResourceTypeGCPStorageBucket, ResourceTypeDOVolume:
		return ResourceCategoryStorage
	case ResourceTypeAWSRDSInstance, ResourceTypeAzureSQLDatabase, ResourceTypeGCPSQLInstance, ResourceTypeDODatabase:
		return ResourceCategoryDatabase
	case ResourceTypeAWSVPC, ResourceTypeAWSSubnet, ResourceTypeAWSSecurityGroup, ResourceTypeAzureVNet, ResourceTypeAzureSubnet, ResourceTypeAzureNSG, ResourceTypeGCPVPCNetwork, ResourceTypeGCPSubnet, ResourceTypeGCPFirewall, ResourceTypeDOVPC, ResourceTypeDOFirewall:
		return ResourceCategoryNetworking
	case ResourceTypeAWSIAMRole, ResourceTypeAWSIAMPolicy, ResourceTypeAzureKeyVault, ResourceTypeGCPIAMRole, ResourceTypeGCPIAMPolicy:
		return ResourceCategorySecurity
	case ResourceTypeAWSLambdaFunction, ResourceTypeAzureFunctionApp, ResourceTypeGCPCloudFunction:
		return ResourceCategoryServerless
	case ResourceTypeDOKubernetes:
		return ResourceCategoryContainer
	default:
		return ResourceCategoryOther
	}
}

// IsComputeResource returns true if the resource type is a compute resource
func (rt ResourceType) IsComputeResource() bool {
	return GetCategoryForResourceType(rt) == ResourceCategoryCompute
}

// IsStorageResource returns true if the resource type is a storage resource
func (rt ResourceType) IsStorageResource() bool {
	return GetCategoryForResourceType(rt) == ResourceCategoryStorage
}

// IsDatabaseResource returns true if the resource type is a database resource
func (rt ResourceType) IsDatabaseResource() bool {
	return GetCategoryForResourceType(rt) == ResourceCategoryDatabase
}

// IsNetworkingResource returns true if the resource type is a networking resource
func (rt ResourceType) IsNetworkingResource() bool {
	return GetCategoryForResourceType(rt) == ResourceCategoryNetworking
}

// IsSecurityResource returns true if the resource type is a security resource
func (rt ResourceType) IsSecurityResource() bool {
	return GetCategoryForResourceType(rt) == ResourceCategorySecurity
}

// IsServerlessResource returns true if the resource type is a serverless resource
func (rt ResourceType) IsServerlessResource() bool {
	return GetCategoryForResourceType(rt) == ResourceCategoryServerless
}

// IsContainerResource returns true if the resource type is a container resource
func (rt ResourceType) IsContainerResource() bool {
	return GetCategoryForResourceType(rt) == ResourceCategoryContainer
}

// GetProviderForResourceType returns the provider for a resource type
func GetProviderForResourceType(resourceType ResourceType) CloudProvider {
	switch {
	case string(resourceType)[:3] == "aws":
		return ProviderAWS
	case string(resourceType)[:7] == "azurerm":
		return ProviderAzure
	case string(resourceType)[:6] == "google":
		return ProviderGCP
	case string(resourceType)[:12] == "digitalocean":
		return ProviderDigitalOcean
	default:
		return ProviderAWS // Default fallback
	}
}

// TriggerAlert triggers an alert
func (ra *ResourceAlert) TriggerAlert() {
	ra.IsTriggeredField = true
	now := time.Now()
	ra.TriggeredAt = &now
	ra.UpdatedAt = now
}

// ResolveAlert resolves an alert
func (ra *ResourceAlert) ResolveAlert() {
	ra.IsTriggeredField = false
	now := time.Now()
	ra.ResolvedAt = &now
	ra.UpdatedAt = now
}

// IsAlertActive returns true if the alert is active
func (ra *ResourceAlert) IsAlertActive() bool {
	return ra.IsActive && !ra.IsTriggeredField
}

// IsAlertTriggered returns true if the alert is triggered
func (ra *ResourceAlert) IsAlertTriggered() bool {
	return ra.IsTriggeredField
}

// IsResolved returns true if the alert is resolved
func (ra *ResourceAlert) IsResolved() bool {
	return ra.ResolvedAt != nil
}
