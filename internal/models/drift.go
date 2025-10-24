package models

import (
	"time"
)

// DriftRecord represents a drift detection record
type DriftRecord struct {
	ID           string                 `json:"id"`
	ResourceID   string                 `json:"resource_id"`
	ResourceName string                 `json:"resource_name"`
	ResourceType string                 `json:"resource_type"`
	Provider     string                 `json:"provider"`
	Region       string                 `json:"region"`
	DriftType    string                 `json:"drift_type"`
	Severity     string                 `json:"severity"`
	Status       string                 `json:"status"`
	DetectedAt   time.Time              `json:"detected_at"`
	ResolvedAt   *time.Time             `json:"resolved_at,omitempty"`
	Description  string                 `json:"description"`
	Details      map[string]interface{} `json:"details,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// DriftType represents the type of drift detected
type DriftType string

const (
	DriftTypeConfiguration DriftType = "configuration"
	DriftTypeState         DriftType = "state"
	DriftTypeResource      DriftType = "resource"
	DriftTypePolicy        DriftType = "policy"
	DriftTypeSecurity      DriftType = "security"
	DriftTypeCompliance    DriftType = "compliance"
)

// DriftSeverity represents the severity of a drift
type DriftSeverity string

const (
	DriftSeverityCritical DriftSeverity = "critical"
	DriftSeverityHigh     DriftSeverity = "high"
	DriftSeverityMedium   DriftSeverity = "medium"
	DriftSeverityLow      DriftSeverity = "low"
	DriftSeverityMinimal  DriftSeverity = "minimal"
)

// DriftStatus represents the status of a drift
type DriftStatus string

const (
	DriftStatusActive     DriftStatus = "active"
	DriftStatusResolved   DriftStatus = "resolved"
	DriftStatusIgnored    DriftStatus = "ignored"
	DriftStatusSuppressed DriftStatus = "suppressed"
)

// DriftSummary represents a summary of drift detection results
type DriftSummary struct {
	ID                 string                 `json:"id"`
	Provider           string                 `json:"provider"`
	Region             string                 `json:"region"`
	TotalResources     int                    `json:"total_resources"`
	DriftedResources   int                    `json:"drifted_resources"`
	CompliantResources int                    `json:"compliant_resources"`
	DriftPercentage    float64                `json:"drift_percentage"`
	ComplianceRate     float64                `json:"compliance_rate"`
	DriftTypes         map[string]int         `json:"drift_types"`
	ResourceTypes      map[string]int         `json:"resource_types"`
	SeverityLevels     map[string]int         `json:"severity_levels"`
	GeneratedAt        time.Time              `json:"generated_at"`
	GeneratedBy        string                 `json:"generated_by"`
	Metadata           map[string]interface{} `json:"metadata"`
}

// DriftTrend represents a trend in drift detection over time
type DriftTrend struct {
	Date             time.Time `json:"date"`
	TotalResources   int       `json:"total_resources"`
	DriftedResources int       `json:"drifted_resources"`
	DriftPercentage  float64   `json:"drift_percentage"`
	NewDrifts        int       `json:"new_drifts"`
	ResolvedDrifts   int       `json:"resolved_drifts"`
}
