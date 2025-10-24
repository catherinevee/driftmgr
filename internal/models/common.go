package models

import "time"

// JobStatus represents the status of a job (used across multiple modules)
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// String returns the string representation of JobStatus
func (js JobStatus) String() string {
	return string(js)
}

// JobProgress represents the progress of a job (used across multiple modules)
type JobProgress struct {
	TotalResources      int        `json:"total_resources" db:"total_resources"`
	DiscoveredResources int        `json:"discovered_resources" db:"discovered_resources"`
	FailedResources     int        `json:"failed_resources" db:"failed_resources"`
	Percentage          float64    `json:"percentage" db:"percentage"`
	CurrentResource     string     `json:"current_resource" db:"current_resource"`
	EstimatedTimeLeft   string     `json:"estimated_time_left" db:"estimated_time_left"`
	StartedAt           *time.Time `json:"started_at" db:"started_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
}

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityInfo     AlertSeverity = "info"
)

// String returns the string representation of AlertSeverity
func (as AlertSeverity) String() string {
	return string(as)
}

// Common error types
var (
	ErrNotFound       = NewError("NOT_FOUND", "Resource not found")
	ErrInvalidInput   = NewError("INVALID_INPUT", "Invalid input provided")
	ErrUnauthorized   = NewError("UNAUTHORIZED", "Unauthorized access")
	ErrForbidden      = NewError("FORBIDDEN", "Access forbidden")
	ErrInternalError  = NewError("INTERNAL_ERROR", "Internal server error")
	ErrNotImplemented = NewError("NOT_IMPLEMENTED", "Feature not implemented")
)

// Error represents a structured error
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewError creates a new error
func NewError(code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Error implements the error interface
func (e *Error) Error() string {
	return e.Message
}

// WithDetails adds details to the error
func (e *Error) WithDetails(details string) *Error {
	e.Details = details
	return e
}
