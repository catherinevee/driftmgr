package errors

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// ErrorTypeValidation represents validation errors
	ErrorTypeValidation ErrorType = "validation"
	// ErrorTypeAuthentication represents authentication errors
	ErrorTypeAuthentication ErrorType = "authentication"
	// ErrorTypeAuthorization represents authorization errors
	ErrorTypeAuthorization ErrorType = "authorization"
	// ErrorTypeNotFound represents not found errors
	ErrorTypeNotFound ErrorType = "not_found"
	// ErrorTypeConflict represents conflict errors
	ErrorTypeConflict ErrorType = "conflict"
	// ErrorTypeRateLimit represents rate limit errors
	ErrorTypeRateLimit ErrorType = "rate_limit"
	// ErrorTypeTimeout represents timeout errors
	ErrorTypeTimeout ErrorType = "timeout"
	// ErrorTypeNetwork represents network errors
	ErrorTypeNetwork ErrorType = "network"
	// ErrorTypeInternal represents internal errors
	ErrorTypeInternal ErrorType = "internal"
	// ErrorTypeCloudProvider represents cloud provider errors
	ErrorTypeCloudProvider ErrorType = "cloud_provider"
	// ErrorTypeConfiguration represents configuration errors
	ErrorTypeConfiguration ErrorType = "configuration"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	// ErrorSeverityLow represents low severity errors
	ErrorSeverityLow ErrorSeverity = "low"
	// ErrorSeverityMedium represents medium severity errors
	ErrorSeverityMedium ErrorSeverity = "medium"
	// ErrorSeverityHigh represents high severity errors
	ErrorSeverityHigh ErrorSeverity = "high"
	// ErrorSeverityCritical represents critical severity errors
	ErrorSeverityCritical ErrorSeverity = "critical"
)

// DriftMgrError represents a structured error with additional context
type DriftMgrError struct {
	Type      ErrorType              `json:"type"`
	Severity  ErrorSeverity          `json:"severity"`
	Message   string                 `json:"message"`
	Cause     error                  `json:"cause,omitempty"`
	Code      string                 `json:"code,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Stack     []StackFrame           `json:"stack,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// StackFrame represents a stack frame
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// New creates a new DriftMgrError
func New(errorType ErrorType, severity ErrorSeverity, message string) *DriftMgrError {
	return &DriftMgrError{
		Type:      errorType,
		Severity:  severity,
		Message:   message,
		Timestamp: time.Now(),
		Stack:     captureStack(),
		Details:   make(map[string]interface{}),
		Context:   make(map[string]interface{}),
	}
}

// NewValidationError creates a new validation error
func NewValidationError(message string) *DriftMgrError {
	return New(ErrorTypeValidation, ErrorSeverityMedium, message)
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(message string) *DriftMgrError {
	return New(ErrorTypeAuthentication, ErrorSeverityHigh, message)
}

// NewAuthorizationError creates a new authorization error
func NewAuthorizationError(message string) *DriftMgrError {
	return New(ErrorTypeAuthorization, ErrorSeverityHigh, message)
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(message string) *DriftMgrError {
	return New(ErrorTypeNotFound, ErrorSeverityMedium, message)
}

// NewConflictError creates a new conflict error
func NewConflictError(message string) *DriftMgrError {
	return New(ErrorTypeConflict, ErrorSeverityMedium, message)
}

// NewRateLimitError creates a new rate limit error
func NewRateLimitError(message string) *DriftMgrError {
	return New(ErrorTypeRateLimit, ErrorSeverityMedium, message)
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(message string) *DriftMgrError {
	return New(ErrorTypeTimeout, ErrorSeverityMedium, message)
}

// NewNetworkError creates a new network error
func NewNetworkError(message string) *DriftMgrError {
	return New(ErrorTypeNetwork, ErrorSeverityHigh, message)
}

// NewInternalError creates a new internal error
func NewInternalError(message string) *DriftMgrError {
	return New(ErrorTypeInternal, ErrorSeverityCritical, message)
}

// NewCloudProviderError creates a new cloud provider error
func NewCloudProviderError(message string) *DriftMgrError {
	return New(ErrorTypeCloudProvider, ErrorSeverityHigh, message)
}

// NewConfigurationError creates a new configuration error
func NewConfigurationError(message string) *DriftMgrError {
	return New(ErrorTypeConfiguration, ErrorSeverityHigh, message)
}

// WithCause sets the cause of the error
func (e *DriftMgrError) WithCause(cause error) *DriftMgrError {
	e.Cause = cause
	return e
}

// WithCode sets the error code
func (e *DriftMgrError) WithCode(code string) *DriftMgrError {
	e.Code = code
	return e
}

// WithDetail adds a detail to the error
func (e *DriftMgrError) WithDetail(key string, value interface{}) *DriftMgrError {
	e.Details[key] = value
	return e
}

// WithContext adds context to the error
func (e *DriftMgrError) WithContext(key string, value interface{}) *DriftMgrError {
	e.Context[key] = value
	return e
}

// WithDetails sets multiple details at once
func (e *DriftMgrError) WithDetails(details map[string]interface{}) *DriftMgrError {
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// WithContextMap sets multiple context values at once
func (e *DriftMgrError) WithContextMap(context map[string]interface{}) *DriftMgrError {
	for k, v := range context {
		e.Context[k] = v
	}
	return e
}

// Error implements the error interface
func (e *DriftMgrError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %s)", e.Type, e.Message, e.Cause.Error())
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the cause error
func (e *DriftMgrError) Unwrap() error {
	return e.Cause
}

// IsValidationError checks if the error is a validation error
func (e *DriftMgrError) IsValidationError() bool {
	return e.Type == ErrorTypeValidation
}

// IsAuthenticationError checks if the error is an authentication error
func (e *DriftMgrError) IsAuthenticationError() bool {
	return e.Type == ErrorTypeAuthentication
}

// IsAuthorizationError checks if the error is an authorization error
func (e *DriftMgrError) IsAuthorizationError() bool {
	return e.Type == ErrorTypeAuthorization
}

// IsNotFoundError checks if the error is a not found error
func (e *DriftMgrError) IsNotFoundError() bool {
	return e.Type == ErrorTypeNotFound
}

// IsConflictError checks if the error is a conflict error
func (e *DriftMgrError) IsConflictError() bool {
	return e.Type == ErrorTypeConflict
}

// IsRateLimitError checks if the error is a rate limit error
func (e *DriftMgrError) IsRateLimitError() bool {
	return e.Type == ErrorTypeRateLimit
}

// IsTimeoutError checks if the error is a timeout error
func (e *DriftMgrError) IsTimeoutError() bool {
	return e.Type == ErrorTypeTimeout
}

// IsNetworkError checks if the error is a network error
func (e *DriftMgrError) IsNetworkError() bool {
	return e.Type == ErrorTypeNetwork
}

// IsInternalError checks if the error is an internal error
func (e *DriftMgrError) IsInternalError() bool {
	return e.Type == ErrorTypeInternal
}

// IsCloudProviderError checks if the error is a cloud provider error
func (e *DriftMgrError) IsCloudProviderError() bool {
	return e.Type == ErrorTypeCloudProvider
}

// IsConfigurationError checks if the error is a configuration error
func (e *DriftMgrError) IsConfigurationError() bool {
	return e.Type == ErrorTypeConfiguration
}

// IsHighSeverity checks if the error is high severity or higher
func (e *DriftMgrError) IsHighSeverity() bool {
	return e.Severity == ErrorSeverityHigh || e.Severity == ErrorSeverityCritical
}

// IsCriticalSeverity checks if the error is critical severity
func (e *DriftMgrError) IsCriticalSeverity() bool {
	return e.Severity == ErrorSeverityCritical
}

// GetDetail returns a detail value
func (e *DriftMgrError) GetDetail(key string) interface{} {
	return e.Details[key]
}

// GetContext returns a context value
func (e *DriftMgrError) GetContext(key string) interface{} {
	return e.Context[key]
}

// captureStack captures the current stack trace
func captureStack() []StackFrame {
	var frames []StackFrame
	for i := 2; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		// Skip internal error handling functions
		if strings.Contains(fn.Name(), "errors.") || strings.Contains(fn.Name(), "runtime.") {
			continue
		}

		frames = append(frames, StackFrame{
			Function: fn.Name(),
			File:     file,
			Line:     line,
		})

		// Limit stack trace depth
		if len(frames) >= 10 {
			break
		}
	}

	return frames
}

// Wrap wraps an existing error with additional context
func Wrap(err error, errorType ErrorType, severity ErrorSeverity, message string) *DriftMgrError {
	if err == nil {
		return nil
	}

	// If it's already a DriftMgrError, just add context
	if driftErr, ok := err.(*DriftMgrError); ok {
		driftErr.WithContext("wrapped_message", message)
		return driftErr
	}

	return New(errorType, severity, message).WithCause(err)
}

// WrapValidation wraps an error as a validation error
func WrapValidation(err error, message string) *DriftMgrError {
	return Wrap(err, ErrorTypeValidation, ErrorSeverityMedium, message)
}

// WrapAuthentication wraps an error as an authentication error
func WrapAuthentication(err error, message string) *DriftMgrError {
	return Wrap(err, ErrorTypeAuthentication, ErrorSeverityHigh, message)
}

// WrapAuthorization wraps an error as an authorization error
func WrapAuthorization(err error, message string) *DriftMgrError {
	return Wrap(err, ErrorTypeAuthorization, ErrorSeverityHigh, message)
}

// WrapNotFound wraps an error as a not found error
func WrapNotFound(err error, message string) *DriftMgrError {
	return Wrap(err, ErrorTypeNotFound, ErrorSeverityMedium, message)
}

// WrapConflict wraps an error as a conflict error
func WrapConflict(err error, message string) *DriftMgrError {
	return Wrap(err, ErrorTypeConflict, ErrorSeverityMedium, message)
}

// WrapRateLimit wraps an error as a rate limit error
func WrapRateLimit(err error, message string) *DriftMgrError {
	return Wrap(err, ErrorTypeRateLimit, ErrorSeverityMedium, message)
}

// WrapTimeout wraps an error as a timeout error
func WrapTimeout(err error, message string) *DriftMgrError {
	return Wrap(err, ErrorTypeTimeout, ErrorSeverityMedium, message)
}

// WrapNetwork wraps an error as a network error
func WrapNetwork(err error, message string) *DriftMgrError {
	return Wrap(err, ErrorTypeNetwork, ErrorSeverityHigh, message)
}

// WrapInternal wraps an error as an internal error
func WrapInternal(err error, message string) *DriftMgrError {
	return Wrap(err, ErrorTypeInternal, ErrorSeverityCritical, message)
}

// WrapCloudProvider wraps an error as a cloud provider error
func WrapCloudProvider(err error, message string) *DriftMgrError {
	return Wrap(err, ErrorTypeCloudProvider, ErrorSeverityHigh, message)
}

// WrapConfiguration wraps an error as a configuration error
func WrapConfiguration(err error, message string) *DriftMgrError {
	return Wrap(err, ErrorTypeConfiguration, ErrorSeverityHigh, message)
}

// IsDriftMgrError checks if an error is a DriftMgrError
func IsDriftMgrError(err error) bool {
	_, ok := err.(*DriftMgrError)
	return ok
}

// GetDriftMgrError returns the DriftMgrError if the error is one
func GetDriftMgrError(err error) (*DriftMgrError, bool) {
	driftErr, ok := err.(*DriftMgrError)
	return driftErr, ok
}

// ErrorAggregator aggregates multiple errors
type ErrorAggregator struct {
	errors []*DriftMgrError
}

// NewErrorAggregator creates a new error aggregator
func NewErrorAggregator() *ErrorAggregator {
	return &ErrorAggregator{
		errors: make([]*DriftMgrError, 0),
	}
}

// Add adds an error to the aggregator
func (ea *ErrorAggregator) Add(err error) {
	if err == nil {
		return
	}

	if driftErr, ok := err.(*DriftMgrError); ok {
		ea.errors = append(ea.errors, driftErr)
	} else {
		ea.errors = append(ea.errors, WrapInternal(err, "Unknown error"))
	}
}

// Errors returns all collected errors
func (ea *ErrorAggregator) Errors() []*DriftMgrError {
	return ea.errors
}

// HasErrors returns true if there are any errors
func (ea *ErrorAggregator) HasErrors() bool {
	return len(ea.errors) > 0
}

// ErrorCount returns the number of errors
func (ea *ErrorAggregator) ErrorCount() int {
	return len(ea.errors)
}

// GetErrorsByType returns errors of a specific type
func (ea *ErrorAggregator) GetErrorsByType(errorType ErrorType) []*DriftMgrError {
	var result []*DriftMgrError
	for _, err := range ea.errors {
		if err.Type == errorType {
			result = append(result, err)
		}
	}
	return result
}

// GetErrorsBySeverity returns errors of a specific severity
func (ea *ErrorAggregator) GetErrorsBySeverity(severity ErrorSeverity) []*DriftMgrError {
	var result []*DriftMgrError
	for _, err := range ea.errors {
		if err.Severity == severity {
			result = append(result, err)
		}
	}
	return result
}

// GetCriticalErrors returns all critical errors
func (ea *ErrorAggregator) GetCriticalErrors() []*DriftMgrError {
	return ea.GetErrorsBySeverity(ErrorSeverityCritical)
}

// GetHighSeverityErrors returns all high severity errors
func (ea *ErrorAggregator) GetHighSeverityErrors() []*DriftMgrError {
	var result []*DriftMgrError
	for _, err := range ea.errors {
		if err.IsHighSeverity() {
			result = append(result, err)
		}
	}
	return result
}
