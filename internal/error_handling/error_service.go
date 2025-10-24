package error_handling

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/events"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	ErrorSeverityCritical ErrorSeverity = "critical"
	ErrorSeverityHigh     ErrorSeverity = "high"
	ErrorSeverityMedium   ErrorSeverity = "medium"
	ErrorSeverityLow      ErrorSeverity = "low"
	ErrorSeverityInfo     ErrorSeverity = "info"
)

// ErrorCategory represents the category of an error
type ErrorCategory string

const (
	ErrorCategoryAuthentication ErrorCategory = "authentication"
	ErrorCategoryAuthorization  ErrorCategory = "authorization"
	ErrorCategoryNetwork        ErrorCategory = "network"
	ErrorCategoryTimeout        ErrorCategory = "timeout"
	ErrorCategoryValidation     ErrorCategory = "validation"
	ErrorCategoryConfiguration  ErrorCategory = "configuration"
	ErrorCategoryResource       ErrorCategory = "resource"
	ErrorCategoryProvider       ErrorCategory = "provider"
	ErrorCategoryInternal       ErrorCategory = "internal"
	ErrorCategoryExternal       ErrorCategory = "external"
)

// ErrorContext represents additional context for an error
type ErrorContext struct {
	Operation     string                 `json:"operation"`
	Provider      string                 `json:"provider,omitempty"`
	Region        string                 `json:"region,omitempty"`
	ResourceID    string                 `json:"resource_id,omitempty"`
	ResourceType  string                 `json:"resource_type,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	SessionID     string                 `json:"session_id,omitempty"`
	RequestID     string                 `json:"request_id,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// EnhancedError represents an enhanced error with additional context
type EnhancedError struct {
	ID          string                 `json:"id"`
	Message     string                 `json:"message"`
	OriginalErr error                  `json:"original_error,omitempty"`
	Severity    ErrorSeverity          `json:"severity"`
	Category    ErrorCategory          `json:"category"`
	Context     ErrorContext           `json:"context"`
	Stack       []string               `json:"stack,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Retryable   bool                   `json:"retryable"`
	RetryCount  int                    `json:"retry_count"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Error implements the error interface
func (e *EnhancedError) Error() string {
	return e.Message
}

// ErrorService provides enhanced error handling functionality
type ErrorService struct {
	eventBus     *events.EventBus
	config       ErrorConfig
	errorHistory []EnhancedError
}

// ErrorConfig represents configuration for the error service
type ErrorConfig struct {
	MaxHistorySize    int           `json:"max_history_size"`
	RetryAttempts     int           `json:"retry_attempts"`
	RetryDelay        time.Duration `json:"retry_delay"`
	EnableStackTraces bool          `json:"enable_stack_traces"`
	EnableMetrics     bool          `json:"enable_metrics"`
	SeverityThreshold ErrorSeverity `json:"severity_threshold"`
}

// NewErrorService creates a new error service
func NewErrorService(eventBus *events.EventBus, config ErrorConfig) *ErrorService {
	return &ErrorService{
		eventBus:     eventBus,
		config:       config,
		errorHistory: make([]EnhancedError, 0),
	}
}

// HandleError processes and handles an error with enhanced context
func (es *ErrorService) HandleError(ctx context.Context, err error, context ErrorContext) *EnhancedError {
	if err == nil {
		return nil
	}

	enhancedErr := es.createEnhancedError(err, context)
	
	// Add to history
	es.addToHistory(enhancedErr)
	
	// Publish error event
	es.publishErrorEvent(enhancedErr)
	
	// Log error based on severity
	es.logError(enhancedErr)
	
	return enhancedErr
}

// HandleErrorWithRetry handles an error with retry logic
func (es *ErrorService) HandleErrorWithRetry(ctx context.Context, err error, context ErrorContext, retryFunc func() error) error {
	if err == nil {
		return nil
	}

	enhancedErr := es.createEnhancedError(err, context)
	
	// Check if error is retryable
	if !enhancedErr.Retryable {
		es.HandleError(ctx, err, context)
		return err
	}
	
	// Attempt retry
	for i := 0; i < es.config.RetryAttempts; i++ {
		enhancedErr.RetryCount = i + 1
		
		// Wait before retry
		if i > 0 {
			time.Sleep(es.config.RetryDelay * time.Duration(i))
		}
		
		// Attempt the operation
		if retryErr := retryFunc(); retryErr == nil {
			// Success
			es.publishRetrySuccessEvent(enhancedErr)
			return nil
		} else {
			// Update error with retry information
			enhancedErr.OriginalErr = retryErr
			enhancedErr.Message = fmt.Sprintf("Retry %d failed: %v", i+1, retryErr)
		}
	}
	
	// All retries failed
	enhancedErr.Message = fmt.Sprintf("All %d retry attempts failed: %v", es.config.RetryAttempts, err)
	es.HandleError(ctx, enhancedErr, context)
	
	return enhancedErr
}

// CreateError creates a new enhanced error
func (es *ErrorService) CreateError(message string, severity ErrorSeverity, category ErrorCategory, context ErrorContext) *EnhancedError {
	return es.createEnhancedError(fmt.Errorf(message), context)
}

// GetErrorHistory returns the error history
func (es *ErrorService) GetErrorHistory() []EnhancedError {
	return es.errorHistory
}

// GetErrorsBySeverity returns errors filtered by severity
func (es *ErrorService) GetErrorsBySeverity(severity ErrorSeverity) []EnhancedError {
	var filtered []EnhancedError
	for _, err := range es.errorHistory {
		if err.Severity == severity {
			filtered = append(filtered, err)
		}
	}
	return filtered
}

// GetErrorsByCategory returns errors filtered by category
func (es *ErrorService) GetErrorsByCategory(category ErrorCategory) []EnhancedError {
	var filtered []EnhancedError
	for _, err := range es.errorHistory {
		if err.Category == category {
			filtered = append(filtered, err)
		}
	}
	return filtered
}

// GetErrorStatistics returns statistics about errors
func (es *ErrorService) GetErrorStatistics() map[string]interface{} {
	stats := make(map[string]interface{})
	
	// Total errors
	stats["total_errors"] = len(es.errorHistory)
	
	// Errors by severity
	severityCount := make(map[ErrorSeverity]int)
	for _, err := range es.errorHistory {
		severityCount[err.Severity]++
	}
	stats["errors_by_severity"] = severityCount
	
	// Errors by category
	categoryCount := make(map[ErrorCategory]int)
	for _, err := range es.errorHistory {
		categoryCount[err.Category]++
	}
	stats["errors_by_category"] = categoryCount
	
	// Errors by provider
	providerCount := make(map[string]int)
	for _, err := range es.errorHistory {
		if err.Context.Provider != "" {
			providerCount[err.Context.Provider]++
		}
	}
	stats["errors_by_provider"] = providerCount
	
	// Recent errors (last 24 hours)
	recentCount := 0
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, err := range es.errorHistory {
		if err.Timestamp.After(cutoff) {
			recentCount++
		}
	}
	stats["recent_errors_24h"] = recentCount
	
	return stats
}

// ClearErrorHistory clears the error history
func (es *ErrorService) ClearErrorHistory() {
	es.errorHistory = make([]EnhancedError, 0)
}

// Helper methods

func (es *ErrorService) createEnhancedError(err error, context ErrorContext) *EnhancedError {
	enhancedErr := &EnhancedError{
		ID:        generateErrorID(),
		Message:   err.Error(),
		OriginalErr: err,
		Context:   context,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	
	// Determine severity and category
	enhancedErr.Severity = es.determineSeverity(err)
	enhancedErr.Category = es.determineCategory(err)
	enhancedErr.Retryable = es.isRetryable(err)
	
	// Add stack trace if enabled
	if es.config.EnableStackTraces {
		enhancedErr.Stack = es.getStackTrace()
	}
	
	return enhancedErr
}

func (es *ErrorService) determineSeverity(err error) ErrorSeverity {
	errStr := strings.ToLower(err.Error())
	
	// Critical errors
	if strings.Contains(errStr, "critical") || 
	   strings.Contains(errStr, "fatal") ||
	   strings.Contains(errStr, "panic") {
		return ErrorSeverityCritical
	}
	
	// High severity errors
	if strings.Contains(errStr, "unauthorized") ||
	   strings.Contains(errStr, "forbidden") ||
	   strings.Contains(errStr, "access denied") ||
	   strings.Contains(errStr, "permission denied") {
		return ErrorSeverityHigh
	}
	
	// Medium severity errors
	if strings.Contains(errStr, "timeout") ||
	   strings.Contains(errStr, "connection") ||
	   strings.Contains(errStr, "network") {
		return ErrorSeverityMedium
	}
	
	// Low severity errors
	if strings.Contains(errStr, "not found") ||
	   strings.Contains(errStr, "invalid") ||
	   strings.Contains(errStr, "validation") {
		return ErrorSeverityLow
	}
	
	// Default to medium
	return ErrorSeverityMedium
}

func (es *ErrorService) determineCategory(err error) ErrorCategory {
	errStr := strings.ToLower(err.Error())
	
	// Authentication errors
	if strings.Contains(errStr, "unauthorized") ||
	   strings.Contains(errStr, "authentication") ||
	   strings.Contains(errStr, "credential") {
		return ErrorCategoryAuthentication
	}
	
	// Authorization errors
	if strings.Contains(errStr, "forbidden") ||
	   strings.Contains(errStr, "permission") ||
	   strings.Contains(errStr, "access denied") {
		return ErrorCategoryAuthorization
	}
	
	// Network errors
	if strings.Contains(errStr, "connection") ||
	   strings.Contains(errStr, "network") ||
	   strings.Contains(errStr, "dial") {
		return ErrorCategoryNetwork
	}
	
	// Timeout errors
	if strings.Contains(errStr, "timeout") ||
	   strings.Contains(errStr, "deadline") {
		return ErrorCategoryTimeout
	}
	
	// Validation errors
	if strings.Contains(errStr, "invalid") ||
	   strings.Contains(errStr, "validation") ||
	   strings.Contains(errStr, "malformed") {
		return ErrorCategoryValidation
	}
	
	// Resource errors
	if strings.Contains(errStr, "not found") ||
	   strings.Contains(errStr, "resource") {
		return ErrorCategoryResource
	}
	
	// Provider errors
	if strings.Contains(errStr, "provider") ||
	   strings.Contains(errStr, "aws") ||
	   strings.Contains(errStr, "azure") ||
	   strings.Contains(errStr, "gcp") {
		return ErrorCategoryProvider
	}
	
	// Default to internal
	return ErrorCategoryInternal
}

func (es *ErrorService) isRetryable(err error) bool {
	errStr := strings.ToLower(err.Error())
	
	// Retryable errors
	if strings.Contains(errStr, "timeout") ||
	   strings.Contains(errStr, "connection") ||
	   strings.Contains(errStr, "network") ||
	   strings.Contains(errStr, "temporary") ||
	   strings.Contains(errStr, "rate limit") {
		return true
	}
	
	// Non-retryable errors
	if strings.Contains(errStr, "unauthorized") ||
	   strings.Contains(errStr, "forbidden") ||
	   strings.Contains(errStr, "not found") ||
	   strings.Contains(errStr, "invalid") ||
	   strings.Contains(errStr, "malformed") {
		return false
	}
	
	// Default to retryable for unknown errors
	return true
}

func (es *ErrorService) getStackTrace() []string {
	var stack []string
	for i := 0; i < 10; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		stack = append(stack, fmt.Sprintf("%s:%d", file, line))
	}
	return stack
}

func (es *ErrorService) addToHistory(err *EnhancedError) {
	es.errorHistory = append(es.errorHistory, *err)
	
	// Maintain max history size
	if len(es.errorHistory) > es.config.MaxHistorySize {
		es.errorHistory = es.errorHistory[1:]
	}
}

func (es *ErrorService) publishErrorEvent(err *EnhancedError) {
	if es.eventBus == nil {
		return
	}
	
	event := events.Event{
		Type:      events.EventType("error.occurred"),
		Timestamp: time.Now(),
		Source:    "error_service",
		Data: map[string]interface{}{
			"error_id":    err.ID,
			"message":     err.Message,
			"severity":    string(err.Severity),
			"category":    string(err.Category),
			"operation":   err.Context.Operation,
			"provider":    err.Context.Provider,
			"region":      err.Context.Region,
			"resource_id": err.Context.ResourceID,
			"retryable":   err.Retryable,
			"retry_count": err.RetryCount,
		},
	}
	
	es.eventBus.Publish(event)
}

func (es *ErrorService) publishRetrySuccessEvent(err *EnhancedError) {
	if es.eventBus == nil {
		return
	}
	
	event := events.Event{
		Type:      events.EventType("error.retry_success"),
		Timestamp: time.Now(),
		Source:    "error_service",
		Data: map[string]interface{}{
			"error_id":    err.ID,
			"operation":   err.Context.Operation,
			"provider":    err.Context.Provider,
			"retry_count": err.RetryCount,
		},
	}
	
	es.eventBus.Publish(event)
}

func (es *ErrorService) logError(err *EnhancedError) {
	// This is a simplified logging implementation
	// In a real implementation, you would use a proper logging library
	logMessage := fmt.Sprintf("[%s] %s: %s (Provider: %s, Region: %s, Resource: %s)",
		err.Severity, err.Category, err.Message,
		err.Context.Provider, err.Context.Region, err.Context.ResourceID)
	
	// Log based on severity
	switch err.Severity {
	case ErrorSeverityCritical:
		// Log as critical error
		fmt.Printf("CRITICAL: %s\n", logMessage)
	case ErrorSeverityHigh:
		// Log as high severity error
		fmt.Printf("ERROR: %s\n", logMessage)
	case ErrorSeverityMedium:
		// Log as medium severity error
		fmt.Printf("WARN: %s\n", logMessage)
	case ErrorSeverityLow:
		// Log as low severity error
		fmt.Printf("INFO: %s\n", logMessage)
	default:
		// Log as info
		fmt.Printf("INFO: %s\n", logMessage)
	}
}

func generateErrorID() string {
	return fmt.Sprintf("err-%d", time.Now().UnixNano())
}
