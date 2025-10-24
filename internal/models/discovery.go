package models

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// CloudProvider represents a cloud provider
type CloudProvider string

const (
	ProviderAWS          CloudProvider = "aws"
	ProviderAzure        CloudProvider = "azure"
	ProviderGCP          CloudProvider = "gcp"
	ProviderDigitalOcean CloudProvider = "digitalocean"
)

// String returns the string representation of CloudProvider
func (cp CloudProvider) String() string {
	return string(cp)
}

// CloudResource represents a discovered cloud resource
type CloudResource struct {
	ID             string                 `json:"id" db:"id" validate:"required,uuid"`
	Provider       CloudProvider          `json:"provider" db:"provider" validate:"required,oneof=aws azure gcp digitalocean"`
	Type           string                 `json:"type" db:"type" validate:"required"`
	Name           string                 `json:"name" db:"name" validate:"required"`
	Region         string                 `json:"region" db:"region" validate:"required"`
	AccountID      string                 `json:"account_id" db:"account_id" validate:"required"`
	ProjectID      string                 `json:"project_id,omitempty" db:"project_id"`
	ResourceGroup  string                 `json:"resource_group,omitempty" db:"resource_group"`
	Tags           map[string]string      `json:"tags" db:"tags"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	Configuration  map[string]interface{} `json:"configuration" db:"configuration"`
	Relationships  []ResourceRelationship `json:"relationships,omitempty" db:"relationships"`
	Compliance     ComplianceStatus       `json:"compliance,omitempty" db:"compliance"`
	Cost           CostInformation        `json:"cost,omitempty" db:"cost"`
	LastDiscovered time.Time              `json:"last_discovered" db:"last_discovered"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// ResourceRelationship represents a relationship between two resources
type ResourceRelationship struct {
	ID         string                 `json:"id" db:"id" validate:"required,uuid"`
	SourceID   string                 `json:"source_id" db:"source_id" validate:"required,uuid"`
	TargetID   string                 `json:"target_id" db:"target_id" validate:"required,uuid"`
	Type       RelationshipType       `json:"type" db:"type" validate:"required"`
	Direction  RelationshipDirection  `json:"direction" db:"direction" validate:"required"`
	Properties map[string]interface{} `json:"properties" db:"properties"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at" db:"updated_at"`
}

// RelationshipType represents the type of relationship between resources
type RelationshipType string

const (
	RelationshipTypeDependsOn   RelationshipType = "depends_on"
	RelationshipTypeContains    RelationshipType = "contains"
	RelationshipTypeConnectedTo RelationshipType = "connected_to"
	RelationshipTypeUses        RelationshipType = "uses"
	RelationshipTypeManagedBy   RelationshipType = "managed_by"
	RelationshipTypeSecuredBy   RelationshipType = "secured_by"
	RelationshipTypeBackedBy    RelationshipType = "backed_by"
	RelationshipTypeMonitoredBy RelationshipType = "monitored_by"
)

// String returns the string representation of RelationshipType
func (rt RelationshipType) String() string {
	return string(rt)
}

// RelationshipDirection represents the direction of a relationship
type RelationshipDirection string

const (
	RelationshipDirectionBidirectional  RelationshipDirection = "bidirectional"
	RelationshipDirectionUnidirectional RelationshipDirection = "unidirectional"
)

// String returns the string representation of RelationshipDirection
func (rd RelationshipDirection) String() string {
	return string(rd)
}

// ComplianceStatus represents the compliance status of a resource
type ComplianceStatus struct {
	Status      ComplianceLevel `json:"status" db:"status"`
	PolicyID    string          `json:"policy_id" db:"policy_id"`
	PolicyName  string          `json:"policy_name" db:"policy_name"`
	Violations  []Violation     `json:"violations" db:"violations"`
	LastChecked time.Time       `json:"last_checked" db:"last_checked"`
	NextCheck   time.Time       `json:"next_check" db:"next_check"`
	CheckedBy   string          `json:"checked_by" db:"checked_by"`
}

// ComplianceLevel represents the level of compliance
type ComplianceLevel string

const (
	ComplianceLevelCompliant    ComplianceLevel = "compliant"
	ComplianceLevelNonCompliant ComplianceLevel = "non_compliant"
	ComplianceLevelWarning      ComplianceLevel = "warning"
	ComplianceLevelUnknown      ComplianceLevel = "unknown"
)

// String returns the string representation of ComplianceLevel
func (cl ComplianceLevel) String() string {
	return string(cl)
}

// Violation represents a compliance violation
type Violation struct {
	ID          string    `json:"id" db:"id"`
	RuleID      string    `json:"rule_id" db:"rule_id"`
	RuleName    string    `json:"rule_name" db:"rule_name"`
	Severity    string    `json:"severity" db:"severity"`
	Description string    `json:"description" db:"description"`
	Remediation string    `json:"remediation" db:"remediation"`
	DetectedAt  time.Time `json:"detected_at" db:"detected_at"`
}

// CostInformation represents cost information for a resource
type CostInformation struct {
	MonthlyCost   float64            `json:"monthly_cost" db:"monthly_cost"`
	DailyCost     float64            `json:"daily_cost" db:"daily_cost"`
	HourlyCost    float64            `json:"hourly_cost" db:"hourly_cost"`
	Currency      string             `json:"currency" db:"currency"`
	CostBreakdown map[string]float64 `json:"cost_breakdown" db:"cost_breakdown"`
	LastUpdated   time.Time          `json:"last_updated" db:"last_updated"`
	BillingPeriod string             `json:"billing_period" db:"billing_period"`
}

// DiscoveryJob represents a resource discovery job
type DiscoveryJob struct {
	ID            string                 `json:"id" db:"id" validate:"required,uuid"`
	Provider      CloudProvider          `json:"provider" db:"provider" validate:"required,oneof=aws azure gcp digitalocean"`
	AccountID     string                 `json:"account_id" db:"account_id" validate:"required"`
	Region        string                 `json:"region" db:"region" validate:"required"`
	ResourceTypes []string               `json:"resource_types" db:"resource_types"`
	Status        JobStatus              `json:"status" db:"status" validate:"required"`
	Progress      JobProgress            `json:"progress" db:"progress"`
	Results       DiscoveryResults       `json:"results" db:"results"`
	Configuration map[string]interface{} `json:"configuration" db:"configuration"`
	StartedAt     time.Time              `json:"started_at" db:"started_at"`
	CompletedAt   *time.Time             `json:"completed_at" db:"completed_at"`
	Error         *string                `json:"error,omitempty" db:"error"`
	CreatedBy     string                 `json:"created_by" db:"created_by" validate:"required"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
}

// JobStatus is defined in common.go

// JobProgress is defined in common.go

// DiscoveryResults represents the results of a discovery job
type DiscoveryResults struct {
	TotalDiscovered   int                    `json:"total_discovered" db:"total_discovered"`
	ResourcesByType   map[string]int         `json:"resources_by_type" db:"resources_by_type"`
	ResourcesByRegion map[string]int         `json:"resources_by_region" db:"resources_by_region"`
	NewResources      []string               `json:"new_resources" db:"new_resources"`
	UpdatedResources  []string               `json:"updated_resources" db:"updated_resources"`
	DeletedResources  []string               `json:"deleted_resources" db:"deleted_resources"`
	Errors            []DiscoveryError       `json:"errors" db:"errors"`
	Summary           map[string]interface{} `json:"summary" db:"summary"`
}

// DiscoveryError represents an error during discovery
type DiscoveryError struct {
	ResourceType string    `json:"resource_type" db:"resource_type"`
	ResourceID   string    `json:"resource_id" db:"resource_id"`
	Error        string    `json:"error" db:"error"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
}

// ProviderAccount represents a cloud provider account
type ProviderAccount struct {
	ID            string                 `json:"id" db:"id" validate:"required,uuid"`
	Provider      CloudProvider          `json:"provider" db:"provider" validate:"required,oneof=aws azure gcp digitalocean"`
	Name          string                 `json:"name" db:"name" validate:"required"`
	AccountID     string                 `json:"account_id" db:"account_id" validate:"required"`
	Region        string                 `json:"region" db:"region" validate:"required"`
	Credentials   map[string]interface{} `json:"credentials" db:"credentials"`
	Configuration map[string]interface{} `json:"configuration" db:"configuration"`
	IsActive      bool                   `json:"is_active" db:"is_active"`
	LastConnected *time.Time             `json:"last_connected" db:"last_connected"`
	CreatedBy     string                 `json:"created_by" db:"created_by" validate:"required"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
}

// DiscoverySchedule represents a scheduled discovery job
type DiscoverySchedule struct {
	ID             string                 `json:"id" db:"id" validate:"required,uuid"`
	Name           string                 `json:"name" db:"name" validate:"required"`
	Provider       CloudProvider          `json:"provider" db:"provider" validate:"required,oneof=aws azure gcp digitalocean"`
	AccountID      string                 `json:"account_id" db:"account_id" validate:"required"`
	Region         string                 `json:"region" db:"region" validate:"required"`
	ResourceTypes  []string               `json:"resource_types" db:"resource_types"`
	CronExpression string                 `json:"cron_expression" db:"cron_expression" validate:"required"`
	IsActive       bool                   `json:"is_active" db:"is_active"`
	LastRun        *time.Time             `json:"last_run" db:"last_run"`
	NextRun        *time.Time             `json:"next_run" db:"next_run"`
	Configuration  map[string]interface{} `json:"configuration" db:"configuration"`
	CreatedBy      string                 `json:"created_by" db:"created_by" validate:"required"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// Request/Response Models

// DiscoveryJobCreateRequest represents a request to create a discovery job
type DiscoveryJobCreateRequest struct {
	Provider      CloudProvider          `json:"provider" validate:"required,oneof=aws azure gcp digitalocean"`
	AccountID     string                 `json:"account_id" validate:"required"`
	Region        string                 `json:"region" validate:"required"`
	ResourceTypes []string               `json:"resource_types"`
	Configuration map[string]interface{} `json:"configuration"`
}

// DiscoveryJobListRequest represents a request to list discovery jobs
type DiscoveryJobListRequest struct {
	Provider  *CloudProvider `json:"provider,omitempty"`
	AccountID *string        `json:"account_id,omitempty"`
	Region    *string        `json:"region,omitempty"`
	Status    *JobStatus     `json:"status,omitempty"`
	Limit     int            `json:"limit" validate:"min=1,max=1000"`
	Offset    int            `json:"offset" validate:"min=0"`
	SortBy    string         `json:"sort_by" validate:"omitempty,oneof=created_at started_at completed_at status"`
	SortOrder string         `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// DiscoveryJobListResponse represents the response for listing discovery jobs
type DiscoveryJobListResponse struct {
	Jobs   []DiscoveryJob `json:"jobs"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

// ResourceListRequest represents a request to list resources
type ResourceListRequest struct {
	Provider      *CloudProvider    `json:"provider,omitempty"`
	AccountID     *string           `json:"account_id,omitempty"`
	Region        *string           `json:"region,omitempty"`
	ResourceType  *string           `json:"resource_type,omitempty"`
	ResourceGroup *string           `json:"resource_group,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	Compliance    *ComplianceLevel  `json:"compliance,omitempty"`
	Limit         int               `json:"limit" validate:"min=1,max=1000"`
	Offset        int               `json:"offset" validate:"min=0"`
	SortBy        string            `json:"sort_by" validate:"omitempty,oneof=name last_discovered created_at updated_at"`
	SortOrder     string            `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// ResourceListResponse represents the response for listing resources
type ResourceListResponse struct {
	Resources []CloudResource `json:"resources"`
	Total     int             `json:"total"`
	Limit     int             `json:"limit"`
	Offset    int             `json:"offset"`
}

// ResourceSearchRequest represents a request to search resources
type ResourceSearchRequest struct {
	Query        string            `json:"query" validate:"required"`
	Provider     *CloudProvider    `json:"provider,omitempty"`
	AccountID    *string           `json:"account_id,omitempty"`
	Region       *string           `json:"region,omitempty"`
	ResourceType *string           `json:"resource_type,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
	Limit        int               `json:"limit" validate:"min=1,max=1000"`
	Offset       int               `json:"offset" validate:"min=0"`
}

// ResourceSearchResponse represents the response for searching resources
type ResourceSearchResponse struct {
	Resources []CloudResource `json:"resources"`
	Total     int             `json:"total"`
	Limit     int             `json:"limit"`
	Offset    int             `json:"offset"`
	Query     string          `json:"query"`
}

// ProviderAccountCreateRequest represents a request to create a provider account
type ProviderAccountCreateRequest struct {
	Provider      CloudProvider          `json:"provider" validate:"required,oneof=aws azure gcp digitalocean"`
	Name          string                 `json:"name" validate:"required"`
	AccountID     string                 `json:"account_id" validate:"required"`
	Region        string                 `json:"region" validate:"required"`
	Credentials   map[string]interface{} `json:"credentials" validate:"required"`
	Configuration map[string]interface{} `json:"configuration"`
}

// ProviderAccountListRequest represents a request to list provider accounts
type ProviderAccountListRequest struct {
	Provider  *CloudProvider `json:"provider,omitempty"`
	IsActive  *bool          `json:"is_active,omitempty"`
	Limit     int            `json:"limit" validate:"min=1,max=1000"`
	Offset    int            `json:"offset" validate:"min=0"`
	SortBy    string         `json:"sort_by" validate:"omitempty,oneof=name created_at last_connected"`
	SortOrder string         `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// ProviderAccountListResponse represents the response for listing provider accounts
type ProviderAccountListResponse struct {
	Accounts []ProviderAccount `json:"accounts"`
	Total    int               `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

// Validation methods

// Validate validates the CloudResource struct
func (cr *CloudResource) Validate() error {
	validate := validator.New()
	return validate.Struct(cr)
}

// Validate validates the ResourceRelationship struct
func (rr *ResourceRelationship) Validate() error {
	validate := validator.New()
	return validate.Struct(rr)
}

// Validate validates the DiscoveryJob struct
func (dj *DiscoveryJob) Validate() error {
	validate := validator.New()
	return validate.Struct(dj)
}

// Validate validates the ProviderAccount struct
func (pa *ProviderAccount) Validate() error {
	validate := validator.New()
	return validate.Struct(pa)
}

// Validate validates the DiscoverySchedule struct
func (ds *DiscoverySchedule) Validate() error {
	validate := validator.New()
	return validate.Struct(ds)
}

// Validate validates the DiscoveryJobCreateRequest struct
func (djcr *DiscoveryJobCreateRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(djcr)
}

// Validate validates the DiscoveryJobListRequest struct
func (djlr *DiscoveryJobListRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(djlr)
}

// Validate validates the ResourceListRequest struct
func (rlr *ResourceListRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(rlr)
}

// Validate validates the ResourceSearchRequest struct
func (rsr *ResourceSearchRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(rsr)
}

// Validate validates the ProviderAccountCreateRequest struct
func (pacr *ProviderAccountCreateRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(pacr)
}

// Validate validates the ProviderAccountListRequest struct
func (palr *ProviderAccountListRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(palr)
}

// Helper methods

// IsCompleted returns true if the discovery job is completed
func (dj *DiscoveryJob) IsCompleted() bool {
	return dj.Status == JobStatusCompleted || dj.Status == JobStatusFailed || dj.Status == JobStatusCancelled
}

// IsRunning returns true if the discovery job is running
func (dj *DiscoveryJob) IsRunning() bool {
	return dj.Status == JobStatusRunning
}

// UpdateProgress updates the job progress
func (dj *DiscoveryJob) UpdateProgress(total, discovered, failed int, currentResource string) {
	dj.Progress.TotalResources = total
	dj.Progress.DiscoveredResources = discovered
	dj.Progress.FailedResources = failed
	dj.Progress.CurrentResource = currentResource

	if total > 0 {
		dj.Progress.Percentage = float64(discovered) / float64(total) * 100
	}

	dj.UpdatedAt = time.Now()
}

// SetStatus updates the job status
func (dj *DiscoveryJob) SetStatus(status JobStatus) {
	dj.Status = status
	dj.UpdatedAt = time.Now()

	now := time.Now()
	switch status {
	case JobStatusRunning:
		dj.StartedAt = now
	case JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
		dj.CompletedAt = &now
	}
}

// SetError sets the job error
func (dj *DiscoveryJob) SetError(err error) {
	if err != nil {
		errStr := err.Error()
		dj.Error = &errStr
		dj.SetStatus(JobStatusFailed)
	}
}

// GetResourceType returns the resource type for display
func (cr *CloudResource) GetResourceType() string {
	return cr.Type
}

// GetDisplayName returns the display name for the resource
func (cr *CloudResource) GetDisplayName() string {
	if cr.Name != "" {
		return cr.Name
	}
	return cr.ID
}

// GetFullAddress returns the full resource address
func (cr *CloudResource) GetFullAddress() string {
	return cr.Provider.String() + "." + cr.Type + "." + cr.Name
}

// HasTag checks if the resource has a specific tag
func (cr *CloudResource) HasTag(key, value string) bool {
	if cr.Tags == nil {
		return false
	}
	return cr.Tags[key] == value
}

// GetTag returns a tag value
func (cr *CloudResource) GetTag(key string) (string, bool) {
	if cr.Tags == nil {
		return "", false
	}
	value, exists := cr.Tags[key]
	return value, exists
}

// SetTag sets a tag value
func (cr *CloudResource) SetTag(key, value string) {
	if cr.Tags == nil {
		cr.Tags = make(map[string]string)
	}
	cr.Tags[key] = value
	cr.UpdatedAt = time.Now()
}

// RemoveTag removes a tag
func (cr *CloudResource) RemoveTag(key string) {
	if cr.Tags != nil {
		delete(cr.Tags, key)
		cr.UpdatedAt = time.Now()
	}
}

// ProviderConfigurationRequest represents a request to create a provider configuration
type ProviderConfigurationRequest struct {
	Provider    CloudProvider       `json:"provider" validate:"required"`
	Name        string              `json:"name" validate:"required"`
	Description string              `json:"description,omitempty"`
	AccountID   string              `json:"account_id" validate:"required"`
	Region      string              `json:"region" validate:"required"`
	Credentials ProviderCredentials `json:"credentials" validate:"required"`
	Settings    *ProviderSettings   `json:"settings,omitempty"`
	IsDefault   bool                `json:"is_default,omitempty"`
}

// ProviderConfigurationUpdateRequest represents a request to update a provider configuration
type ProviderConfigurationUpdateRequest struct {
	Name        *string              `json:"name,omitempty"`
	Description *string              `json:"description,omitempty"`
	Credentials *ProviderCredentials `json:"credentials,omitempty"`
	Settings    *ProviderSettings    `json:"settings,omitempty"`
	IsActive    *bool                `json:"is_active,omitempty"`
	IsDefault   *bool                `json:"is_default,omitempty"`
}

// StateFileRequest represents a request to create a state file
type StateFileRequest struct {
	BackendID   string `json:"backend_id" validate:"required"`
	Workspace   string `json:"workspace" validate:"required"`
	Environment string `json:"environment" validate:"required"`
}

// StateFileUpdateRequest represents a request to update a state file
type StateFileUpdateRequest struct {
	Workspace   *string `json:"workspace,omitempty"`
	Environment *string `json:"environment,omitempty"`
}

// Validate validates the ProviderConfigurationRequest struct
func (pcr *ProviderConfigurationRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(pcr)
}

// Validate validates the StateFileRequest struct
func (sfr *StateFileRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(sfr)
}
