package models

import (
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// Resource represents a cloud resource with comprehensive details
type Resource struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Provider     string                 `json:"provider"`
	Region       string                 `json:"region"`
	Status       string                 `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	ModifiedAt   time.Time              `json:"modified_at"`
	Tags         map[string]string      `json:"tags"`
	Properties   map[string]interface{} `json:"properties"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Managed      bool                   `json:"managed"`
	DriftStatus  string                 `json:"drift_status,omitempty"`
	
	// Additional detailed information
	ARN          string                 `json:"arn,omitempty"`          // AWS ARN
	ResourceID   string                 `json:"resource_id,omitempty"`  // Azure Resource ID
	SelfLink     string                 `json:"self_link,omitempty"`    // GCP Self Link
	URN          string                 `json:"urn,omitempty"`          // DigitalOcean URN
	Cost         float64                `json:"cost,omitempty"`         // Estimated cost
	Owner        string                 `json:"owner,omitempty"`        // Resource owner
	Environment  string                 `json:"environment,omitempty"`  // Environment (prod/dev/staging)
	Compliance   map[string]bool        `json:"compliance,omitempty"`   // Compliance status
	Metrics      map[string]interface{} `json:"metrics,omitempty"`      // Performance metrics
	LastScanned  time.Time              `json:"last_scanned,omitempty"` // Last discovery scan time
	Account      string                 `json:"account,omitempty"`      // Account/Subscription ID
	Raw          interface{}            `json:"raw,omitempty"`          // Raw API response
}

// Perspective represents an infrastructure perspective
type Perspective struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	StateFilePath      string                 `json:"state_file_path"`
	Provider           string                 `json:"provider"`
	Region             string                 `json:"region,omitempty"`
	ManagedResources   []models.Resource      `json:"managed_resources"`
	OutOfBand          []models.Resource      `json:"out_of_band_resources"`
	StateResources     []models.Resource      `json:"state_resources,omitempty"`
	UnmanagedResources []models.Resource      `json:"unmanaged_resources,omitempty"`
	MissingResources   []models.Resource      `json:"missing_resources,omitempty"`
	Coverage           float64                `json:"coverage"`
	DriftPercentage    float64                `json:"drift_percentage"`
	Timestamp          time.Time              `json:"timestamp"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}