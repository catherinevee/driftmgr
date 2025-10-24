package api

import (
	"time"
)

// APIResponse represents a standardized API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *APIMeta    `json:"meta,omitempty"`
}

// APIError represents an API error
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// APIMeta represents response metadata
type APIMeta struct {
	Count      int    `json:"count,omitempty"`
	Page       int    `json:"page,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	TotalPages int    `json:"total_pages,omitempty"`
	Timestamp  string `json:"timestamp"`
}

// Backend represents a Terraform backend
type Backend struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Config      map[string]string `json:"config"`
	StateCount  int               `json:"state_count"`
	IsActive    bool              `json:"is_active"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// StateFile represents a Terraform state file
type StateFile struct {
	ID          string            `json:"id"`
	Path        string            `json:"path"`
	BackendID   string            `json:"backend_id"`
	BackendType string            `json:"backend_type"`
	Size        int64             `json:"size"`
	ResourceCount int             `json:"resource_count"`
	LastModified time.Time        `json:"last_modified"`
	IsLocked    bool              `json:"is_locked"`
	LockInfo    *LockInfo         `json:"lock_info,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// LockInfo represents state file lock information
type LockInfo struct {
	ID        string    `json:"id"`
	Operation string    `json:"operation"`
	Who       string    `json:"who"`
	Version   string    `json:"version"`
	Created   time.Time `json:"created"`
	Path      string    `json:"path"`
}

// Resource represents a cloud resource
type Resource struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Provider     string            `json:"provider"`
	Region       string            `json:"region"`
	AccountID    string            `json:"account_id"`
	State        string            `json:"state"`
	Tags         map[string]string `json:"tags"`
	Metadata     map[string]string `json:"metadata"`
	Cost         *ResourceCost     `json:"cost,omitempty"`
	Compliance   *ComplianceStatus `json:"compliance,omitempty"`
	DiscoveredAt time.Time         `json:"discovered_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// ResourceCost represents resource cost information
type ResourceCost struct {
	MonthlyCost   float64 `json:"monthly_cost"`
	DailyCost     float64 `json:"daily_cost"`
	Currency      string  `json:"currency"`
	LastCalculated time.Time `json:"last_calculated"`
}

// ComplianceStatus represents compliance information
type ComplianceStatus struct {
	Status      string            `json:"status"`
	Score       int               `json:"score"`
	Violations  []ComplianceViolation `json:"violations"`
	LastChecked time.Time         `json:"last_checked"`
}

// ComplianceViolation represents a compliance violation
type ComplianceViolation struct {
	Rule        string `json:"rule"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

// DriftResult represents a drift detection result
type DriftResult struct {
	ID           string            `json:"id"`
	ResourceID   string            `json:"resource_id"`
	ResourceName string            `json:"resource_name"`
	ResourceType string            `json:"resource_type"`
	Provider     string            `json:"provider"`
	Region       string            `json:"region"`
	Severity     string            `json:"severity"`
	Status       string            `json:"status"`
	DriftCount   int               `json:"drift_count"`
	Drifts       []DriftDetail     `json:"drifts"`
	DetectedAt   time.Time         `json:"detected_at"`
	ResolvedAt   *time.Time        `json:"resolved_at,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// DriftDetail represents a specific drift detail
type DriftDetail struct {
	Field       string      `json:"field"`
	Expected    interface{} `json:"expected"`
	Actual      interface{} `json:"actual"`
	Severity    string      `json:"severity"`
	Description string      `json:"description"`
}

// DiscoveryJob represents a resource discovery job
type DiscoveryJob struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Providers   []string          `json:"providers"`
	Regions     []string          `json:"regions"`
	Status      string            `json:"status"`
	Progress    int               `json:"progress"`
	ResourcesFound int            `json:"resources_found"`
	StartedAt   time.Time         `json:"started_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	Config      map[string]string `json:"config,omitempty"`
}

// BackendDiscoveryRequest represents a backend discovery request
type BackendDiscoveryRequest struct {
	Paths     []string `json:"paths"`
	Recursive bool     `json:"recursive"`
	Types     []string `json:"types,omitempty"`
}

// BackendDiscoveryResponse represents a backend discovery response
type BackendDiscoveryResponse struct {
	Count    int       `json:"count"`
	Backends []Backend `json:"backends"`
	Errors   []string  `json:"errors,omitempty"`
}

// StateFileListRequest represents a state file list request
type StateFileListRequest struct {
	BackendID string `json:"backend_id,omitempty"`
	Page      int    `json:"page,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

// ResourceListRequest represents a resource list request
type ResourceListRequest struct {
	Provider string            `json:"provider,omitempty"`
	Region   string            `json:"region,omitempty"`
	Type     string            `json:"type,omitempty"`
	Tags     map[string]string `json:"tags,omitempty"`
	Page     int               `json:"page,omitempty"`
	Limit    int               `json:"limit,omitempty"`
}

// DriftDetectionRequest represents a drift detection request
type DriftDetectionRequest struct {
	ResourceIDs []string `json:"resource_ids,omitempty"`
	Providers   []string `json:"providers,omitempty"`
	Regions     []string `json:"regions,omitempty"`
	Incremental bool     `json:"incremental"`
	UseCache    bool     `json:"use_cache"`
}

// DriftDetectionResponse represents a drift detection response
type DriftDetectionResponse struct {
	JobID      string        `json:"job_id"`
	Status     string        `json:"status"`
	DriftCount int           `json:"drift_count"`
	Results    []DriftResult `json:"results,omitempty"`
	Message    string        `json:"message"`
}

// Helper functions for creating standardized responses

// NewSuccessResponse creates a success response
func NewSuccessResponse(data interface{}, meta *APIMeta) *APIResponse {
	if meta == nil {
		meta = &APIMeta{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}
	return &APIResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	}
}

// NewErrorResponse creates an error response
func NewErrorResponse(code, message, details string) *APIResponse {
	return &APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: &APIMeta{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}
}

// NewPaginationMeta creates pagination metadata
func NewPaginationMeta(page, limit, count int) *APIMeta {
	totalPages := (count + limit - 1) / limit
	return &APIMeta{
		Count:      count,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
}
