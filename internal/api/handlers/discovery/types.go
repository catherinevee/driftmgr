package discovery

import "time"

// DiscoveryRequest represents a discovery job request
type DiscoveryRequest struct {
	Provider        string   `json:"provider"`
	Regions         []string `json:"regions"`
	ResourceTypes   []string `json:"resource_types,omitempty"` // Filter by specific resource types
	StateFilePath   string   `json:"state_file_path,omitempty"`
	RemediationMode string   `json:"remediation_mode,omitempty"`
	DriftDetection  bool     `json:"drift_detection"`
}

// DriftRecord represents a drift detection record
type DriftRecord struct {
	ID           string                 `json:"id"`
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	Provider     string                 `json:"provider"`
	Region       string                 `json:"region"`
	DriftType    string                 `json:"drift_type"`
	Description  string                 `json:"description"`
	Expected     interface{}            `json:"expected"`
	Actual       interface{}            `json:"actual"`
	Severity     string                 `json:"severity"`
	DetectedAt   time.Time              `json:"detected_at"`
	Status       string                 `json:"status"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"` // Detailed drift information
}
