package models

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// ProviderConfiguration represents the configuration for a cloud provider
type ProviderConfiguration struct {
	ID               string              `json:"id" db:"id" validate:"required,uuid"`
	Provider         CloudProvider       `json:"provider" db:"provider" validate:"required,oneof=aws azure gcp digitalocean"`
	Name             string              `json:"name" db:"name" validate:"required"`
	Description      string              `json:"description" db:"description"`
	AccountID        string              `json:"account_id" db:"account_id" validate:"required"`
	Region           string              `json:"region" db:"region" validate:"required"`
	Credentials      ProviderCredentials `json:"credentials" db:"credentials"`
	Settings         ProviderSettings    `json:"settings" db:"settings"`
	IsActive         bool                `json:"is_active" db:"is_active"`
	IsDefault        bool                `json:"is_default" db:"is_default"`
	LastConnected    *time.Time          `json:"last_connected" db:"last_connected"`
	ConnectionStatus ConnectionStatus    `json:"connection_status" db:"connection_status"`
	CreatedBy        string              `json:"created_by" db:"created_by" validate:"required"`
	CreatedAt        time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at" db:"updated_at"`
}

// ProviderCredentials represents credentials for a cloud provider
type ProviderCredentials struct {
	Type        CredentialType         `json:"type" db:"type" validate:"required"`
	AccessKey   string                 `json:"access_key,omitempty" db:"access_key"`
	SecretKey   string                 `json:"secret_key,omitempty" db:"secret_key"`
	Token       string                 `json:"token,omitempty" db:"token"`
	Region      string                 `json:"region,omitempty" db:"region"`
	Profile     string                 `json:"profile,omitempty" db:"profile"`
	RoleARN     string                 `json:"role_arn,omitempty" db:"role_arn"`
	ExternalID  string                 `json:"external_id,omitempty" db:"external_id"`
	SessionName string                 `json:"session_name,omitempty" db:"session_name"`
	Duration    int                    `json:"duration,omitempty" db:"duration"`
	Custom      map[string]interface{} `json:"custom,omitempty" db:"custom"`
}

// CredentialType represents the type of credentials
type CredentialType string

const (
	CredentialTypeAccessKey      CredentialType = "access_key"
	CredentialTypeIAMRole        CredentialType = "iam_role"
	CredentialTypeServiceAccount CredentialType = "service_account"
	CredentialTypeOAuth          CredentialType = "oauth"
	CredentialTypeAPIKey         CredentialType = "api_key"
	CredentialTypeToken          CredentialType = "token"
	CredentialTypeCertificate    CredentialType = "certificate"
)

// String returns the string representation of CredentialType
func (ct CredentialType) String() string {
	return string(ct)
}

// ProviderSettings represents settings for a cloud provider
type ProviderSettings struct {
	RateLimit          RateLimitSettings      `json:"rate_limit" db:"rate_limit"`
	RetrySettings      RetrySettings          `json:"retry_settings" db:"retry_settings"`
	TimeoutSettings    TimeoutSettings        `json:"timeout_settings" db:"timeout_settings"`
	DiscoverySettings  DiscoverySettings      `json:"discovery_settings" db:"discovery_settings"`
	SecuritySettings   SecuritySettings       `json:"security_settings" db:"security_settings"`
	MonitoringSettings MonitoringSettings     `json:"monitoring_settings" db:"monitoring_settings"`
	CustomSettings     map[string]interface{} `json:"custom_settings" db:"custom_settings"`
}

// RateLimitSettings represents rate limiting settings
type RateLimitSettings struct {
	RequestsPerSecond int `json:"requests_per_second" db:"requests_per_second"`
	BurstLimit        int `json:"burst_limit" db:"burst_limit"`
	RetryAfter        int `json:"retry_after" db:"retry_after"`
}

// RetrySettings represents retry settings
type RetrySettings struct {
	MaxRetries        int     `json:"max_retries" db:"max_retries"`
	InitialDelay      int     `json:"initial_delay" db:"initial_delay"`
	MaxDelay          int     `json:"max_delay" db:"max_delay"`
	BackoffMultiplier float64 `json:"backoff_multiplier" db:"backoff_multiplier"`
}

// TimeoutSettings represents timeout settings
type TimeoutSettings struct {
	ConnectionTimeout int `json:"connection_timeout" db:"connection_timeout"`
	RequestTimeout    int `json:"request_timeout" db:"request_timeout"`
	ReadTimeout       int `json:"read_timeout" db:"read_timeout"`
	WriteTimeout      int `json:"write_timeout" db:"write_timeout"`
}

// DiscoverySettings represents discovery settings
type DiscoverySettings struct {
	EnabledResourceTypes  []string          `json:"enabled_resource_types" db:"enabled_resource_types"`
	ExcludedResourceTypes []string          `json:"excluded_resource_types" db:"excluded_resource_types"`
	DiscoveryInterval     int               `json:"discovery_interval" db:"discovery_interval"`
	ParallelDiscovery     int               `json:"parallel_discovery" db:"parallel_discovery"`
	CacheTTL              int               `json:"cache_ttl" db:"cache_ttl"`
	IncludeDeleted        bool              `json:"include_deleted" db:"include_deleted"`
	TagFilter             map[string]string `json:"tag_filter" db:"tag_filter"`
}

// SecuritySettings represents security settings
type SecuritySettings struct {
	EncryptCredentials bool     `json:"encrypt_credentials" db:"encrypt_credentials"`
	AllowedRegions     []string `json:"allowed_regions" db:"allowed_regions"`
	DeniedRegions      []string `json:"denied_regions" db:"denied_regions"`
	RequireMFA         bool     `json:"require_mfa" db:"require_mfa"`
	SessionDuration    int      `json:"session_duration" db:"session_duration"`
	AuditLogging       bool     `json:"audit_logging" db:"audit_logging"`
}

// MonitoringSettings represents monitoring settings
type MonitoringSettings struct {
	EnableMetrics       bool               `json:"enable_metrics" db:"enable_metrics"`
	EnableLogging       bool               `json:"enable_logging" db:"enable_logging"`
	EnableTracing       bool               `json:"enable_tracing" db:"enable_tracing"`
	LogLevel            string             `json:"log_level" db:"log_level"`
	MetricsInterval     int                `json:"metrics_interval" db:"metrics_interval"`
	HealthCheckInterval int                `json:"health_check_interval" db:"health_check_interval"`
	AlertThresholds     map[string]float64 `json:"alert_thresholds" db:"alert_thresholds"`
}

// ConnectionStatus represents the connection status of a provider
type ConnectionStatus string

const (
	ConnectionStatusConnected    ConnectionStatus = "connected"
	ConnectionStatusDisconnected ConnectionStatus = "disconnected"
	ConnectionStatusConnecting   ConnectionStatus = "connecting"
	ConnectionStatusError        ConnectionStatus = "error"
	ConnectionStatusUnknown      ConnectionStatus = "unknown"
)

// String returns the string representation of ConnectionStatus
func (cs ConnectionStatus) String() string {
	return string(cs)
}

// ProviderRegion represents a region for a cloud provider
type ProviderRegion struct {
	ID                string        `json:"id" db:"id" validate:"required,uuid"`
	Provider          CloudProvider `json:"provider" db:"provider" validate:"required,oneof=aws azure gcp digitalocean"`
	RegionCode        string        `json:"region_code" db:"region_code" validate:"required"`
	RegionName        string        `json:"region_name" db:"region_name" validate:"required"`
	DisplayName       string        `json:"display_name" db:"display_name"`
	IsActive          bool          `json:"is_active" db:"is_active"`
	IsDefault         bool          `json:"is_default" db:"is_default"`
	Latitude          float64       `json:"latitude" db:"latitude"`
	Longitude         float64       `json:"longitude" db:"longitude"`
	AvailabilityZones []string      `json:"availability_zones" db:"availability_zones"`
	Services          []string      `json:"services" db:"services"`
	CreatedAt         time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at" db:"updated_at"`
}

// ProviderService represents a service for a cloud provider
type ProviderService struct {
	ID               string        `json:"id" db:"id" validate:"required,uuid"`
	Provider         CloudProvider `json:"provider" db:"provider" validate:"required,oneof=aws azure gcp digitalocean"`
	ServiceCode      string        `json:"service_code" db:"service_code" validate:"required"`
	ServiceName      string        `json:"service_name" db:"service_name" validate:"required"`
	DisplayName      string        `json:"display_name" db:"display_name"`
	Description      string        `json:"description" db:"description"`
	Category         string        `json:"category" db:"category"`
	IsActive         bool          `json:"is_active" db:"is_active"`
	IsSupported      bool          `json:"is_supported" db:"is_supported"`
	ResourceTypes    []string      `json:"resource_types" db:"resource_types"`
	APIVersion       string        `json:"api_version" db:"api_version"`
	DocumentationURL string        `json:"documentation_url" db:"documentation_url"`
	CreatedAt        time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at" db:"updated_at"`
}

// ProviderHealth represents the health status of a provider
type ProviderHealth struct {
	ID                string        `json:"id" db:"id" validate:"required,uuid"`
	ProviderID        string        `json:"provider_id" db:"provider_id" validate:"required,uuid"`
	Status            HealthStatus  `json:"status" db:"status" validate:"required"`
	LastChecked       time.Time     `json:"last_checked" db:"last_checked"`
	ResponseTime      int           `json:"response_time" db:"response_time"`
	ErrorRate         float64       `json:"error_rate" db:"error_rate"`
	AvailableRegions  int           `json:"available_regions" db:"available_regions"`
	AvailableServices int           `json:"available_services" db:"available_services"`
	Issues            []HealthIssue `json:"issues" db:"issues"`
	Metrics           HealthMetrics `json:"metrics" db:"metrics"`
	CreatedAt         time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at" db:"updated_at"`
}

// HealthStatus represents the health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// String returns the string representation of HealthStatus
func (hs HealthStatus) String() string {
	return string(hs)
}

// HealthIssue represents a health issue
type HealthIssue struct {
	ID          string     `json:"id" db:"id"`
	Type        string     `json:"type" db:"type"`
	Severity    string     `json:"severity" db:"severity"`
	Description string     `json:"description" db:"description"`
	DetectedAt  time.Time  `json:"detected_at" db:"detected_at"`
	ResolvedAt  *time.Time `json:"resolved_at" db:"resolved_at"`
	IsResolved  bool       `json:"is_resolved" db:"is_resolved"`
}

// HealthMetrics represents health metrics
type HealthMetrics struct {
	TotalRequests       int        `json:"total_requests" db:"total_requests"`
	SuccessfulRequests  int        `json:"successful_requests" db:"successful_requests"`
	FailedRequests      int        `json:"failed_requests" db:"failed_requests"`
	AverageResponseTime float64    `json:"average_response_time" db:"average_response_time"`
	Uptime              float64    `json:"uptime" db:"uptime"`
	LastError           *string    `json:"last_error" db:"last_error"`
	LastErrorTime       *time.Time `json:"last_error_time" db:"last_error_time"`
}

// Request/Response Models

// ProviderConfigurationCreateRequest represents a request to create a provider configuration
type ProviderConfigurationCreateRequest struct {
	Provider    CloudProvider       `json:"provider" validate:"required,oneof=aws azure gcp digitalocean"`
	Name        string              `json:"name" validate:"required"`
	Description string              `json:"description"`
	AccountID   string              `json:"account_id" validate:"required"`
	Region      string              `json:"region" validate:"required"`
	Credentials ProviderCredentials `json:"credentials" validate:"required"`
	Settings    ProviderSettings    `json:"settings"`
	IsDefault   bool                `json:"is_default"`
}


// ProviderConfigurationListRequest represents a request to list provider configurations
type ProviderConfigurationListRequest struct {
	Provider  *CloudProvider `json:"provider,omitempty"`
	IsActive  *bool          `json:"is_active,omitempty"`
	IsDefault *bool          `json:"is_default,omitempty"`
	Limit     int            `json:"limit" validate:"min=1,max=1000"`
	Offset    int            `json:"offset" validate:"min=0"`
	SortBy    string         `json:"sort_by" validate:"omitempty,oneof=name created_at last_connected"`
	SortOrder string         `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// ProviderConfigurationListResponse represents the response for listing provider configurations
type ProviderConfigurationListResponse struct {
	Configurations []ProviderConfiguration `json:"configurations"`
	Total          int                     `json:"total"`
	Limit          int                     `json:"limit"`
	Offset         int                     `json:"offset"`
}

// ProviderTestConnectionRequest represents a request to test a provider connection
type ProviderTestConnectionRequest struct {
	Provider    CloudProvider       `json:"provider" validate:"required,oneof=aws azure gcp digitalocean"`
	Credentials ProviderCredentials `json:"credentials" validate:"required"`
	Region      string              `json:"region" validate:"required"`
}

// ProviderTestConnectionResponse represents the response for testing a provider connection
type ProviderTestConnectionResponse struct {
	Success      bool                   `json:"success"`
	Status       ConnectionStatus       `json:"status"`
	ResponseTime int                    `json:"response_time"`
	Message      string                 `json:"message"`
	Details      map[string]interface{} `json:"details"`
	TestedAt     time.Time              `json:"tested_at"`
}

// ProviderRegionListRequest represents a request to list provider regions
type ProviderRegionListRequest struct {
	Provider  *CloudProvider `json:"provider,omitempty"`
	IsActive  *bool          `json:"is_active,omitempty"`
	Limit     int            `json:"limit" validate:"min=1,max=1000"`
	Offset    int            `json:"offset" validate:"min=0"`
	SortBy    string         `json:"sort_by" validate:"omitempty,oneof=region_code region_name"`
	SortOrder string         `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// ProviderRegionListResponse represents the response for listing provider regions
type ProviderRegionListResponse struct {
	Regions []ProviderRegion `json:"regions"`
	Total   int              `json:"total"`
	Limit   int              `json:"limit"`
	Offset  int              `json:"offset"`
}

// ProviderServiceListRequest represents a request to list provider services
type ProviderServiceListRequest struct {
	Provider    *CloudProvider `json:"provider,omitempty"`
	Category    *string        `json:"category,omitempty"`
	IsActive    *bool          `json:"is_active,omitempty"`
	IsSupported *bool          `json:"is_supported,omitempty"`
	Limit       int            `json:"limit" validate:"min=1,max=1000"`
	Offset      int            `json:"offset" validate:"min=0"`
	SortBy      string         `json:"sort_by" validate:"omitempty,oneof=service_code service_name category"`
	SortOrder   string         `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// ProviderServiceListResponse represents the response for listing provider services
type ProviderServiceListResponse struct {
	Services []ProviderService `json:"services"`
	Total    int               `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

// Validation methods

// Validate validates the ProviderConfiguration struct
func (pc *ProviderConfiguration) Validate() error {
	validate := validator.New()
	return validate.Struct(pc)
}

// Validate validates the ProviderCredentials struct
func (pcr *ProviderCredentials) Validate() error {
	validate := validator.New()
	return validate.Struct(pcr)
}

// Validate validates the ProviderRegion struct
func (pr *ProviderRegion) Validate() error {
	validate := validator.New()
	return validate.Struct(pr)
}

// Validate validates the ProviderService struct
func (ps *ProviderService) Validate() error {
	validate := validator.New()
	return validate.Struct(ps)
}

// Validate validates the ProviderHealth struct
func (ph *ProviderHealth) Validate() error {
	validate := validator.New()
	return validate.Struct(ph)
}

// Validate validates the ProviderConfigurationCreateRequest struct
func (pccr *ProviderConfigurationCreateRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(pccr)
}

// Validate validates the ProviderConfigurationUpdateRequest struct
func (pcur *ProviderConfigurationUpdateRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(pcur)
}

// Validate validates the ProviderConfigurationListRequest struct
func (pclr *ProviderConfigurationListRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(pclr)
}

// Validate validates the ProviderTestConnectionRequest struct
func (ptcr *ProviderTestConnectionRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(ptcr)
}

// Validate validates the ProviderRegionListRequest struct
func (prlr *ProviderRegionListRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(prlr)
}

// Validate validates the ProviderServiceListRequest struct
func (pslr *ProviderServiceListRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(pslr)
}

// Helper methods

// IsConnected returns true if the provider is connected
func (pc *ProviderConfiguration) IsConnected() bool {
	return pc.ConnectionStatus == ConnectionStatusConnected
}

// IsHealthy returns true if the provider is healthy
func (ph *ProviderHealth) IsHealthy() bool {
	return ph.Status == HealthStatusHealthy
}

// UpdateConnectionStatus updates the connection status
func (pc *ProviderConfiguration) UpdateConnectionStatus(status ConnectionStatus) {
	pc.ConnectionStatus = status
	pc.UpdatedAt = time.Now()

	if status == ConnectionStatusConnected {
		now := time.Now()
		pc.LastConnected = &now
	}
}

// UpdateHealth updates the health status
func (ph *ProviderHealth) UpdateHealth(status HealthStatus, responseTime int, errorRate float64) {
	ph.Status = status
	ph.ResponseTime = responseTime
	ph.ErrorRate = errorRate
	ph.LastChecked = time.Now()
	ph.UpdatedAt = time.Now()
}

// AddHealthIssue adds a health issue
func (ph *ProviderHealth) AddHealthIssue(issue HealthIssue) {
	ph.Issues = append(ph.Issues, issue)
	ph.UpdatedAt = time.Now()
}

// ResolveHealthIssue resolves a health issue
func (ph *ProviderHealth) ResolveHealthIssue(issueID string) {
	for i, issue := range ph.Issues {
		if issue.ID == issueID {
			ph.Issues[i].IsResolved = true
			now := time.Now()
			ph.Issues[i].ResolvedAt = &now
			ph.UpdatedAt = time.Now()
			break
		}
	}
}

// GetDefaultSettings returns default settings for a provider
func GetDefaultSettings(provider CloudProvider) ProviderSettings {
	return ProviderSettings{
		RateLimit: RateLimitSettings{
			RequestsPerSecond: 10,
			BurstLimit:        20,
			RetryAfter:        60,
		},
		RetrySettings: RetrySettings{
			MaxRetries:        3,
			InitialDelay:      1000,
			MaxDelay:          10000,
			BackoffMultiplier: 2.0,
		},
		TimeoutSettings: TimeoutSettings{
			ConnectionTimeout: 30000,
			RequestTimeout:    60000,
			ReadTimeout:       30000,
			WriteTimeout:      30000,
		},
		DiscoverySettings: DiscoverySettings{
			DiscoveryInterval: 3600,
			ParallelDiscovery: 5,
			CacheTTL:          1800,
			IncludeDeleted:    false,
		},
		SecuritySettings: SecuritySettings{
			EncryptCredentials: true,
			RequireMFA:         false,
			SessionDuration:    3600,
			AuditLogging:       true,
		},
		MonitoringSettings: MonitoringSettings{
			EnableMetrics:       true,
			EnableLogging:       true,
			EnableTracing:       false,
			LogLevel:            "info",
			MetricsInterval:     300,
			HealthCheckInterval: 60,
		},
	}
}

// GetSupportedRegions returns supported regions for a provider
func GetSupportedRegions(provider CloudProvider) []string {
	switch provider {
	case ProviderAWS:
		return []string{
			"us-east-1", "us-east-2", "us-west-1", "us-west-2",
			"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1",
			"ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-northeast-2",
		}
	case ProviderAzure:
		return []string{
			"eastus", "eastus2", "westus", "westus2",
			"centralus", "northcentralus", "southcentralus", "westcentralus",
			"northeurope", "westeurope", "uksouth", "ukwest",
		}
	case ProviderGCP:
		return []string{
			"us-central1", "us-east1", "us-east4", "us-west1", "us-west2", "us-west3", "us-west4",
			"europe-west1", "europe-west2", "europe-west3", "europe-west4", "europe-west6",
			"asia-east1", "asia-east2", "asia-northeast1", "asia-northeast2", "asia-south1", "asia-southeast1",
		}
	case ProviderDigitalOcean:
		return []string{
			"nyc1", "nyc2", "nyc3", "sfo1", "sfo2", "sfo3",
			"ams2", "ams3", "fra1", "lon1", "sgp1", "blr1",
		}
	default:
		return []string{}
	}
}

// GetSupportedServices returns supported services for a provider
func GetSupportedServices(provider CloudProvider) []string {
	switch provider {
	case ProviderAWS:
		return []string{
			"ec2", "s3", "rds", "lambda", "iam", "vpc", "cloudformation",
			"cloudwatch", "sns", "sqs", "dynamodb", "elasticache", "route53",
		}
	case ProviderAzure:
		return []string{
			"compute", "storage", "sql", "functions", "keyvault", "network",
			"monitor", "servicebus", "cosmosdb", "redis", "dns", "resourcemanager",
		}
	case ProviderGCP:
		return []string{
			"compute", "storage", "sql", "cloudfunctions", "iam", "network",
			"monitoring", "pubsub", "firestore", "memorystore", "dns", "resourcemanager",
		}
	case ProviderDigitalOcean:
		return []string{
			"droplets", "volumes", "databases", "loadbalancers", "networking",
			"monitoring", "spaces", "kubernetes", "dns", "firewalls",
		}
	default:
		return []string{}
	}
}
