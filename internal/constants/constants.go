package constants

import "time"

// Timeouts
const (
	DefaultTimeout            = 30 * time.Second
	DefaultDiscoveryTimeout   = 5 * time.Minute
	DefaultRemediationTimeout = 10 * time.Minute
	DefaultDeletionTimeout    = 30 * time.Minute
	DefaultAPITimeout         = 10 * time.Second
	DefaultContextTimeout     = 2 * time.Minute
)

// Retry configuration
const (
	DefaultMaxRetries    = 3
	DefaultRetryDelay    = 1 * time.Second
	DefaultRetryBackoff  = 2.0
	DefaultMaxRetryDelay = 30 * time.Second
)

// Concurrency limits
const (
	DefaultMaxConcurrency = 10
	DefaultBatchSize      = 50
	DefaultMaxWorkers     = 20
	DefaultQueueSize      = 1000
)

// Resource limits
const (
	MaxResourcesPerPage   = 100
	MaxResourcesPerBatch  = 500
	MaxResourcesWarning   = 1000
	MaxResourcesHardLimit = 10000
	DefaultResourceLimit  = 5000
)

// Cache configuration
const (
	DefaultCacheTTL             = 5 * time.Minute
	DefaultCacheMaxSize         = 1000
	DefaultCacheCleanupInterval = 10 * time.Minute
)

// Safety thresholds
const (
	CriticalDriftThreshold = 0.8
	HighDriftThreshold     = 0.6
	MediumDriftThreshold   = 0.4
	LowDriftThreshold      = 0.2
)

// File and path limits
const (
	MaxFileSize   = 100 * 1024 * 1024 // 100MB
	MaxPathLength = 4096
	MaxNameLength = 256
)

// API rate limits
const (
	DefaultRateLimit       = 100
	DefaultRateLimitWindow = 1 * time.Minute
	AWSRateLimit           = 100
	AzureRateLimit         = 120
	GCPRateLimit           = 100
)

// Log levels
const (
	LogLevelDebug   = "DEBUG"
	LogLevelInfo    = "INFO"
	LogLevelWarning = "WARNING"
	LogLevelError   = "ERROR"
	LogLevelFatal   = "FATAL"
)

// Provider names
const (
	ProviderAWS          = "aws"
	ProviderAzure        = "azure"
	ProviderGCP          = "gcp"
	ProviderKubernetes   = "kubernetes"
	ProviderDigitalOcean = "digitalocean"
)

// Resource states
const (
	ResourceStateActive   = "active"
	ResourceStatePending  = "pending"
	ResourceStateDeleting = "deleting"
	ResourceStateDeleted  = "deleted"
	ResourceStateFailed   = "failed"
	ResourceStateUnknown  = "unknown"
)

// Remediation states
const (
	RemediationStatePending    = "pending"
	RemediationStateInProgress = "in_progress"
	RemediationStateCompleted  = "completed"
	RemediationStateFailed     = "failed"
	RemediationStateRolledBack = "rolled_back"
)

// Drift types
const (
	DriftTypeAdded     = "ADDED"
	DriftTypeModified  = "MODIFIED"
	DriftTypeDeleted   = "DELETED"
	DriftTypeUnchanged = "UNCHANGED"
)

// Severity levels
const (
	SeverityCritical = "CRITICAL"
	SeverityHigh     = "HIGH"
	SeverityMedium   = "MEDIUM"
	SeverityLow      = "LOW"
	SeverityInfo     = "INFO"
)
