package detector

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/common/errors"
	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/catherinevee/driftmgr/internal/state/parser"
)

// EnhancedDetector is a drift detector with enhanced error handling
type EnhancedDetector struct {
	providers     map[string]providers.CloudProvider
	errorHandler  *errors.ErrorHandler
	recovery      *errors.RecoveryExecutor
	correlationID string
}

// NewEnhancedDetector creates a new enhanced drift detector
func NewEnhancedDetector() *EnhancedDetector {
	detector := &EnhancedDetector{
		providers:    make(map[string]providers.CloudProvider),
		errorHandler: errors.NewErrorHandler(),
		recovery:     errors.NewRecoveryExecutor(),
	}
	
	// Register error handlers for different error types
	detector.errorHandler.RegisterHandler(errors.ErrorTypeTransient, detector.handleTransientError)
	detector.errorHandler.RegisterHandler(errors.ErrorTypeValidation, detector.handleValidationError)
	detector.errorHandler.RegisterHandler(errors.ErrorTypeNotFound, detector.handleNotFoundError)
	
	return detector
}

// DetectDriftWithContext detects drift with enhanced error handling
func (d *EnhancedDetector) DetectDriftWithContext(ctx context.Context, state *parser.TerraformState) (*DriftReport, error) {
	// Create error context
	errCtx := errors.WithErrorContext(ctx)
	
	// Add trace ID to context
	ctx = context.WithValue(ctx, "trace_id", d.generateTraceID())
	ctx = context.WithValue(ctx, "operation", "drift_detection")
	
	report := &DriftReport{
		StartTime: time.Now(),
		Resources: make([]ResourceDrift, 0),
	}
	
	// Process each resource
	for _, resource := range state.Resources {
		if err := d.processResource(ctx, resource, report); err != nil {
			// Handle error with context
			driftErr := d.enrichError(ctx, err, resource)
			
			// Try recovery
			if recoveryErr := d.recovery.Execute(ctx, driftErr); recoveryErr != nil {
				// Recovery failed, add to error context
				errCtx.AddError(driftErr)
				
				// Decide whether to continue or fail fast
				if driftErr.Severity == errors.SeverityCritical {
					return nil, driftErr
				}
			}
		}
	}
	
	report.EndTime = time.Now()
	
	// Check if there were non-critical errors
	if errCtx.HasErrors() {
		// Return partial success with errors
		report.Errors = errCtx.GetErrors()
		return report, errCtx.GetFirstError()
	}
	
	return report, nil
}

// processResource processes a single resource with error handling
func (d *EnhancedDetector) processResource(ctx context.Context, resource parser.Resource, report *DriftReport) error {
	// Get provider for resource
	provider, err := d.getProvider(resource.Provider)
	if err != nil {
		return errors.NewError(errors.ErrorTypeSystem, "Failed to get provider").
			WithProvider(resource.Provider).
			WithResource(resource.ID).
			WithWrapped(err).
			WithContext(ctx).
			WithUserHelp("Ensure the provider is properly configured and credentials are valid").
			WithRecovery(errors.RecoveryStrategy{
				Strategy:    "fallback",
				Description: "Skip this resource and continue with others",
			}).
			Build()
	}
	
	// Discover actual resource with timeout
	discoverCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	actual, err := d.discoverResource(discoverCtx, provider, resource)
	if err != nil {
		// Check if it's a timeout
		if discoverCtx.Err() != nil {
			return errors.NewTimeoutError("resource_discovery", 30*time.Second).
				WithResource(resource.ID).
				WithProvider(resource.Provider).
				WithContext(ctx)
		}
		
		// Check if resource not found
		if isNotFoundError(err) {
			return errors.NewNotFoundError(resource.ID).
				WithProvider(resource.Provider).
				WithContext(ctx)
		}
		
		// Generic discovery error
		return errors.NewError(errors.ErrorTypeTransient, "Failed to discover resource").
			WithResource(resource.ID).
			WithProvider(resource.Provider).
			WithWrapped(err).
			WithContext(ctx).
			WithRetry(true, 5*time.Second).
			WithRecovery(errors.RecoveryStrategy{
				Strategy:    "retry_with_backoff",
				Description: "Retry discovery with exponential backoff",
				Params: map[string]interface{}{
					"max_retries": 3,
					"base_delay":  "2s",
				},
			}).
			Build()
	}
	
	// Compare resources
	drift := d.compareResources(resource, actual)
	if drift != nil {
		report.Resources = append(report.Resources, *drift)
	}
	
	return nil
}

// enrichError adds context to errors
func (d *EnhancedDetector) enrichError(ctx context.Context, err error, resource parser.Resource) *errors.DriftError {
	// If already a DriftError, enrich it
	if driftErr, ok := err.(*errors.DriftError); ok {
		return driftErr.
			WithDetails("resource_type", resource.Type).
			WithDetails("resource_name", resource.Name).
			WithDetails("correlation_id", d.correlationID)
	}
	
	// Wrap standard error
	return errors.NewError(errors.ErrorTypeSystem, err.Error()).
		WithWrapped(err).
		WithResource(resource.ID).
		WithContext(ctx).
		Build()
}

// Error handlers for different error types

func (d *EnhancedDetector) handleTransientError(err *errors.DriftError) error {
	fmt.Printf("Handling transient error: %s\n", err.Message)
	
	// Log to monitoring system
	d.logToMonitoring(err)
	
	// Attempt automatic recovery
	if err.Retryable {
		return errors.RetryWithExponentialBackoff(
			context.Background(),
			func() error {
				// Retry the operation
				return nil
			},
			errors.DefaultBackoffConfig(),
		)
	}
	
	return err
}

func (d *EnhancedDetector) handleValidationError(err *errors.DriftError) error {
	fmt.Printf("Validation error: %s\n", err.Message)
	
	// Log validation error
	d.logValidationError(err)
	
	// Return with user help
	if err.UserHelp != "" {
		fmt.Printf("User guidance: %s\n", err.UserHelp)
	}
	
	return err
}

func (d *EnhancedDetector) handleNotFoundError(err *errors.DriftError) error {
	fmt.Printf("Resource not found: %s\n", err.Resource)
	
	// Mark as drift (resource deleted)
	// This is expected in drift detection
	
	return nil // Continue processing
}

// Helper methods

func (d *EnhancedDetector) generateTraceID() string {
	return fmt.Sprintf("drift-%d", time.Now().UnixNano())
}

func (d *EnhancedDetector) getProvider(providerName string) (providers.CloudProvider, error) {
	provider, exists := d.providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", providerName)
	}
	return provider, nil
}

func (d *EnhancedDetector) discoverResource(ctx context.Context, provider providers.CloudProvider, resource parser.Resource) (interface{}, error) {
	// Simulate resource discovery
	// In real implementation, this would call provider.DiscoverResource()
	return nil, nil
}

func (d *EnhancedDetector) compareResources(desired parser.Resource, actual interface{}) *ResourceDrift {
	// Simulate resource comparison
	// In real implementation, this would do deep comparison
	return nil
}

func (d *EnhancedDetector) logToMonitoring(err *errors.DriftError) {
	// Send to monitoring system
	fmt.Printf("Monitoring: %s\n", err.ToJSON())
}

func (d *EnhancedDetector) logValidationError(err *errors.DriftError) {
	// Log validation errors for analysis
	fmt.Printf("Validation log: %s\n", err.ToJSON())
}

func isNotFoundError(err error) bool {
	// Check if error indicates resource not found
	// This would check provider-specific error codes
	return false
}

// DriftReport represents the drift detection report
type DriftReport struct {
	StartTime time.Time
	EndTime   time.Time
	Resources []ResourceDrift
	Errors    []error
}

// ResourceDrift represents drift for a single resource
type ResourceDrift struct {
	ResourceID   string
	ResourceType string
	DriftType    string
	Differences  map[string]interface{}
}