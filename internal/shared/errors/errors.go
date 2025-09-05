package errors

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ErrorType represents the category of error
type ErrorType string

const (
	// Error types
	ErrorTypeTransient  ErrorType = "transient"  // Temporary errors that can be retried
	ErrorTypePermanent  ErrorType = "permanent"  // Errors that won't resolve with retry
	ErrorTypeUser       ErrorType = "user"       // User input/configuration errors
	ErrorTypeSystem     ErrorType = "system"     // System/infrastructure errors
	ErrorTypeValidation ErrorType = "validation" // Data validation errors
	ErrorTypeNotFound   ErrorType = "not_found"  // Resource not found errors
	ErrorTypeConflict   ErrorType = "conflict"   // Resource conflict errors
	ErrorTypeTimeout    ErrorType = "timeout"    // Operation timeout errors
)

// ErrorSeverity represents the severity level
type ErrorSeverity string

const (
	SeverityCritical ErrorSeverity = "critical"
	SeverityHigh     ErrorSeverity = "high"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityLow      ErrorSeverity = "low"
)

// DriftError is the enhanced error type with context
type DriftError struct {
	Type       ErrorType              `json:"type"`
	Severity   ErrorSeverity          `json:"severity"`
	Code       string                 `json:"code,omitempty"`
	Message    string                 `json:"message"`
	UserHelp   string                 `json:"user_help,omitempty"`
	Resource   string                 `json:"resource,omitempty"`
	Provider   string                 `json:"provider,omitempty"`
	Operation  string                 `json:"operation,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	TraceID    string                 `json:"trace_id,omitempty"`
	StackTrace string                 `json:"stack_trace,omitempty"`
	Wrapped    error                  `json:"-"`
	Retryable  bool                   `json:"retryable"`
	RetryAfter time.Duration          `json:"retry_after,omitempty"`
	Recovery   RecoveryStrategy       `json:"recovery,omitempty"`
}

// RecoveryStrategy defines how to recover from an error
type RecoveryStrategy struct {
	Strategy    string                 `json:"strategy"`
	Description string                 `json:"description"`
	Steps       []string               `json:"steps,omitempty"`
	Params      map[string]interface{} `json:"params,omitempty"`
}

// Error implements the error interface
func (e *DriftError) Error() string {
	var parts []string
	
	if e.Code != "" {
		parts = append(parts, fmt.Sprintf("[%s]", e.Code))
	}
	
	parts = append(parts, e.Message)
	
	if e.Resource != "" {
		parts = append(parts, fmt.Sprintf("(resource: %s)", e.Resource))
	}
	
	if e.Wrapped != nil {
		parts = append(parts, fmt.Sprintf("caused by: %v", e.Wrapped))
	}
	
	return strings.Join(parts, " ")
}

// Unwrap returns the wrapped error
func (e *DriftError) Unwrap() error {
	return e.Wrapped
}

// Is implements errors.Is interface
func (e *DriftError) Is(target error) bool {
	t, ok := target.(*DriftError)
	if !ok {
		return false
	}
	return e.Type == t.Type && e.Code == t.Code
}

// WithUserHelp adds user-friendly help text
func (e *DriftError) WithUserHelp(help string) *DriftError {
	e.UserHelp = help
	return e
}

// WithRecovery adds recovery strategy
func (e *DriftError) WithRecovery(strategy RecoveryStrategy) *DriftError {
	e.Recovery = strategy
	return e
}

// WithDetails adds additional context details
func (e *DriftError) WithDetails(key string, value interface{}) *DriftError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// ToJSON serializes error to JSON
func (e *DriftError) ToJSON() string {
	data, _ := json.MarshalIndent(e, "", "  ")
	return string(data)
}

// ErrorBuilder provides fluent API for building errors
type ErrorBuilder struct {
	err *DriftError
}

// NewError creates a new error builder
func NewError(errType ErrorType, message string) *ErrorBuilder {
	_, file, line, _ := runtime.Caller(1)
	
	return &ErrorBuilder{
		err: &DriftError{
			Type:      errType,
			Severity:  SeverityMedium,
			Message:   message,
			Timestamp: time.Now(),
			StackTrace: fmt.Sprintf("%s:%d", file, line),
			Details:   make(map[string]interface{}),
		},
	}
}

// WithCode sets error code
func (b *ErrorBuilder) WithCode(code string) *ErrorBuilder {
	b.err.Code = code
	return b
}

// WithSeverity sets error severity
func (b *ErrorBuilder) WithSeverity(severity ErrorSeverity) *ErrorBuilder {
	b.err.Severity = severity
	return b
}

// WithResource sets the affected resource
func (b *ErrorBuilder) WithResource(resource string) *ErrorBuilder {
	b.err.Resource = resource
	return b
}

// WithProvider sets the cloud provider
func (b *ErrorBuilder) WithProvider(provider string) *ErrorBuilder {
	b.err.Provider = provider
	return b
}

// WithOperation sets the operation that failed
func (b *ErrorBuilder) WithOperation(operation string) *ErrorBuilder {
	b.err.Operation = operation
	return b
}

// WithUserHelp adds user help text
func (b *ErrorBuilder) WithUserHelp(help string) *ErrorBuilder {
	b.err.UserHelp = help
	return b
}

// WithDetails adds context details
func (b *ErrorBuilder) WithDetails(key string, value interface{}) *ErrorBuilder {
	b.err.Details[key] = value
	return b
}

// WithWrapped wraps another error
func (b *ErrorBuilder) WithWrapped(err error) *ErrorBuilder {
	b.err.Wrapped = err
	return b
}

// WithRetry marks error as retryable
func (b *ErrorBuilder) WithRetry(retryable bool, retryAfter time.Duration) *ErrorBuilder {
	b.err.Retryable = retryable
	b.err.RetryAfter = retryAfter
	return b
}

// WithRecovery adds recovery strategy
func (b *ErrorBuilder) WithRecovery(strategy RecoveryStrategy) *ErrorBuilder {
	b.err.Recovery = strategy
	return b
}

// WithContext adds context values to error
func (b *ErrorBuilder) WithContext(ctx context.Context) *ErrorBuilder {
	// Extract trace ID if present
	if traceID := ctx.Value("trace_id"); traceID != nil {
		b.err.TraceID = fmt.Sprintf("%v", traceID)
	}
	
	// Extract other context values
	if userID := ctx.Value("user_id"); userID != nil {
		b.err.Details["user_id"] = userID
	}
	
	if requestID := ctx.Value("request_id"); requestID != nil {
		b.err.Details["request_id"] = requestID
	}
	
	return b
}

// Build returns the built error
func (b *ErrorBuilder) Build() *DriftError {
	return b.err
}

// Error returns the error interface
func (b *ErrorBuilder) Error() error {
	return b.err
}

// Common error constructors

// NewTransientError creates a transient error that can be retried
func NewTransientError(message string, retryAfter time.Duration) *DriftError {
	return NewError(ErrorTypeTransient, message).
		WithRetry(true, retryAfter).
		WithRecovery(RecoveryStrategy{
			Strategy:    "retry",
			Description: "Retry the operation after the specified delay",
		}).
		Build()
}

// NewValidationError creates a validation error
func NewValidationError(resource string, message string) *DriftError {
	return NewError(ErrorTypeValidation, message).
		WithResource(resource).
		WithSeverity(SeverityLow).
		WithUserHelp("Please check your input and try again").
		Build()
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) *DriftError {
	return NewError(ErrorTypeNotFound, fmt.Sprintf("Resource not found: %s", resource)).
		WithResource(resource).
		WithSeverity(SeverityLow).
		WithUserHelp("Ensure the resource exists and you have permission to access it").
		Build()
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(operation string, duration time.Duration) *DriftError {
	return NewError(ErrorTypeTimeout, fmt.Sprintf("Operation timed out after %v", duration)).
		WithOperation(operation).
		WithSeverity(SeverityMedium).
		WithRetry(true, 5*time.Second).
		WithRecovery(RecoveryStrategy{
			Strategy:    "retry_with_backoff",
			Description: "Retry with exponential backoff",
			Params: map[string]interface{}{
				"max_retries": 3,
				"base_delay":  "5s",
			},
		}).
		Build()
}

// ErrorHandler provides centralized error handling
type ErrorHandler struct {
	handlers map[ErrorType]func(*DriftError) error
	fallback func(*DriftError) error
}

// NewErrorHandler creates a new error handler
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		handlers: make(map[ErrorType]func(*DriftError) error),
		fallback: defaultFallbackHandler,
	}
}

// RegisterHandler registers a handler for specific error type
func (h *ErrorHandler) RegisterHandler(errType ErrorType, handler func(*DriftError) error) {
	h.handlers[errType] = handler
}

// SetFallback sets the fallback handler
func (h *ErrorHandler) SetFallback(handler func(*DriftError) error) {
	h.fallback = handler
}

// Handle processes an error
func (h *ErrorHandler) Handle(err error) error {
	driftErr, ok := err.(*DriftError)
	if !ok {
		// Wrap non-DriftError
		driftErr = NewError(ErrorTypeSystem, err.Error()).
			WithWrapped(err).
			Build()
	}
	
	// Try specific handler
	if handler, exists := h.handlers[driftErr.Type]; exists {
		return handler(driftErr)
	}
	
	// Use fallback
	return h.fallback(driftErr)
}

// defaultFallbackHandler is the default error handler
func defaultFallbackHandler(err *DriftError) error {
	// Log error based on severity
	switch err.Severity {
	case SeverityCritical, SeverityHigh:
		// Log to error tracking service
		fmt.Printf("CRITICAL ERROR: %s\n", err.ToJSON())
	case SeverityMedium:
		// Log warning
		fmt.Printf("WARNING: %s\n", err.Error())
	case SeverityLow:
		// Log info
		fmt.Printf("INFO: %s\n", err.Error())
	}
	
	return err
}

// ErrorContext adds error context to context.Context
type ErrorContext struct {
	context.Context
	errors []error
}

// WithErrorContext creates a new error context
func WithErrorContext(ctx context.Context) *ErrorContext {
	return &ErrorContext{
		Context: ctx,
		errors:  make([]error, 0),
	}
}

// AddError adds an error to the context
func (ec *ErrorContext) AddError(err error) {
	ec.errors = append(ec.errors, err)
}

// HasErrors returns true if there are errors
func (ec *ErrorContext) HasErrors() bool {
	return len(ec.errors) > 0
}

// GetErrors returns all errors
func (ec *ErrorContext) GetErrors() []error {
	return ec.errors
}

// GetFirstError returns the first error or nil
func (ec *ErrorContext) GetFirstError() error {
	if len(ec.errors) > 0 {
		return ec.errors[0]
	}
	return nil
}