package discovery

import (
	"context"
	"fmt"
	"time"
)

// ErrorHandler handles errors during discovery operations
type ErrorHandler struct {
	retryConfig *RetryConfig
	errors      []DiscoveryErrorDetail
}

// RetryConfig defines retry behavior for failed operations
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
}

// DiscoveryErrorDetail represents detailed error information during discovery
type DiscoveryErrorDetail struct {
	Provider  string
	Region    string
	Resource  string
	Error     error
	Timestamp time.Time
	Retries   int
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(config *RetryConfig) *ErrorHandler {
	if config == nil {
		config = &RetryConfig{
			MaxRetries:     3,
			InitialBackoff: 1 * time.Second,
			MaxBackoff:     30 * time.Second,
			BackoffFactor:  2.0,
		}
	}
	return &ErrorHandler{
		retryConfig: config,
		errors:      make([]DiscoveryErrorDetail, 0),
	}
}

// HandleError processes a discovery error
func (eh *ErrorHandler) HandleError(ctx context.Context, err error, provider, region, resource string) error {
	discoveryErr := DiscoveryErrorDetail{
		Provider:  provider,
		Region:    region,
		Resource:  resource,
		Error:     err,
		Timestamp: time.Now(),
	}

	eh.errors = append(eh.errors, discoveryErr)
	return err
}

// GetErrors returns all recorded errors
func (eh *ErrorHandler) GetErrors() []DiscoveryErrorDetail {
	return eh.errors
}

// ClearErrors clears all recorded errors
func (eh *ErrorHandler) ClearErrors() {
	eh.errors = make([]DiscoveryErrorDetail, 0)
}

// EnhancedErrorReporting provides detailed error reporting
type EnhancedErrorReporting struct {
	errors  map[string][]DiscoveryErrorDetail
	summary ErrorSummary
}

// ErrorSummary provides a summary of errors
type ErrorSummary struct {
	TotalErrors    int
	ErrorsByType   map[string]int
	ErrorsByRegion map[string]int
	CriticalErrors []DiscoveryErrorDetail
}

// NewEnhancedErrorReporting creates a new enhanced error reporter
func NewEnhancedErrorReporting() *EnhancedErrorReporting {
	return &EnhancedErrorReporting{
		errors: make(map[string][]DiscoveryErrorDetail),
		summary: ErrorSummary{
			ErrorsByType:   make(map[string]int),
			ErrorsByRegion: make(map[string]int),
			CriticalErrors: make([]DiscoveryErrorDetail, 0),
		},
	}
}

// RecordError records an error for reporting
func (eer *EnhancedErrorReporting) RecordError(provider string, err DiscoveryErrorDetail) {
	if _, exists := eer.errors[provider]; !exists {
		eer.errors[provider] = make([]DiscoveryErrorDetail, 0)
	}
	eer.errors[provider] = append(eer.errors[provider], err)

	// Update summary
	eer.summary.TotalErrors++
	eer.summary.ErrorsByRegion[err.Region]++

	// Check if critical
	if isCriticalError(err.Error) {
		eer.summary.CriticalErrors = append(eer.summary.CriticalErrors, err)
	}
}

// GetReport generates an error report
func (eer *EnhancedErrorReporting) GetReport() string {
	report := fmt.Sprintf("Discovery Error Report\n")
	report += fmt.Sprintf("=====================\n")
	report += fmt.Sprintf("Total Errors: %d\n", eer.summary.TotalErrors)
	report += fmt.Sprintf("Critical Errors: %d\n", len(eer.summary.CriticalErrors))

	if len(eer.errors) > 0 {
		report += "\nErrors by Provider:\n"
		for provider, errors := range eer.errors {
			report += fmt.Sprintf("  %s: %d errors\n", provider, len(errors))
		}
	}

	if len(eer.summary.ErrorsByRegion) > 0 {
		report += "\nErrors by Region:\n"
		for region, count := range eer.summary.ErrorsByRegion {
			report += fmt.Sprintf("  %s: %d errors\n", region, count)
		}
	}

	return report
}

// isCriticalError determines if an error is critical
func isCriticalError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	criticalPatterns := []string{
		"unauthorized",
		"forbidden",
		"access denied",
		"invalid credentials",
		"quota exceeded",
	}

	for _, pattern := range criticalPatterns {
		if containsString(errMsg, pattern) {
			return true
		}
	}

	return false
}

// containsString checks if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
