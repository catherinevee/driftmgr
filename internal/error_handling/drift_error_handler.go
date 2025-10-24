package error_handling

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/events"
	internalModels "github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/pkg/models"
)

// DriftErrorHandler provides specialized error handling for drift detection
type DriftErrorHandler struct {
	errorService *ErrorService
	eventBus     *events.EventBus
	config       DriftErrorConfig
}

// DriftErrorConfig represents configuration for drift error handling
type DriftErrorConfig struct {
	MaxRetryAttempts    int           `json:"max_retry_attempts"`
	RetryDelay          time.Duration `json:"retry_delay"`
	CircuitBreakerThreshold int       `json:"circuit_breaker_threshold"`
	CircuitBreakerTimeout  time.Duration `json:"circuit_breaker_timeout"`
	EnablePartialResults bool         `json:"enable_partial_results"`
	ContinueOnError     bool         `json:"continue_on_error"`
}

// DriftDetectionResult represents the result of a drift detection operation
type DriftDetectionResult struct {
	Success        bool                    `json:"success"`
	TotalResources int                     `json:"total_resources"`
	ProcessedResources int                 `json:"processed_resources"`
	FailedResources   int                  `json:"failed_resources"`
	Drifts         []internalModels.DriftRecord    `json:"drifts"`
	Errors         []EnhancedError         `json:"errors"`
	PartialResults bool                    `json:"partial_results"`
	Duration       time.Duration           `json:"duration"`
	Metadata       map[string]interface{}  `json:"metadata"`
}

// NewDriftErrorHandler creates a new drift error handler
func NewDriftErrorHandler(errorService *ErrorService, eventBus *events.EventBus, config DriftErrorConfig) *DriftErrorHandler {
	return &DriftErrorHandler{
		errorService: errorService,
		eventBus:     eventBus,
		config:       config,
	}
}

// HandleDriftDetection handles drift detection with comprehensive error handling
func (deh *DriftErrorHandler) HandleDriftDetection(
	ctx context.Context,
	provider string,
	region string,
	detectionFunc func(ctx context.Context, resources []models.Resource) ([]internalModels.DriftRecord, error),
	resources []models.Resource,
) *DriftDetectionResult {
	start := time.Now()
	result := &DriftDetectionResult{
		TotalResources: len(resources),
		Drifts:         make([]internalModels.DriftRecord, 0),
		Errors:         make([]EnhancedError, 0),
		Metadata:       make(map[string]interface{}),
	}

	// Handle empty resources
	if len(resources) == 0 {
		result.Success = true
		result.Duration = time.Since(start)
		deh.publishDriftDetectionEvent(result, provider, region)
		return result
	}

	// Process resources in batches to handle errors gracefully
	batchSize := 10
	processedCount := 0
	failedCount := 0

	for i := 0; i < len(resources); i += batchSize {
		end := i + batchSize
		if end > len(resources) {
			end = len(resources)
		}

		batch := resources[i:end]
		batchResult := deh.processResourceBatch(ctx, provider, region, detectionFunc, batch)
		
		// Merge results
		result.Drifts = append(result.Drifts, batchResult.Drifts...)
		result.Errors = append(result.Errors, batchResult.Errors...)
		processedCount += batchResult.ProcessedResources
		failedCount += batchResult.FailedResources

		// Check if we should continue on error
		if !deh.config.ContinueOnError && len(batchResult.Errors) > 0 {
			// Stop processing if we have errors and continue_on_error is false
			result.PartialResults = true
			break
		}
	}

	result.ProcessedResources = processedCount
	result.FailedResources = failedCount
	result.Success = failedCount == 0 || (deh.config.EnablePartialResults && processedCount > 0)
	result.Duration = time.Since(start)

	// Publish drift detection event
	deh.publishDriftDetectionEvent(result, provider, region)

	return result
}

// HandleResourceDriftDetection handles drift detection for a single resource
func (deh *DriftErrorHandler) HandleResourceDriftDetection(
	ctx context.Context,
	resource models.Resource,
	detectionFunc func(ctx context.Context, resource models.Resource) (*internalModels.DriftRecord, error),
) (*internalModels.DriftRecord, *EnhancedError) {
	context := ErrorContext{
		Operation:    "drift_detection",
		Provider:     resource.Provider,
		Region:       resource.Region,
		ResourceID:   resource.ID,
		ResourceType: resource.Type,
	}

	// Attempt drift detection with retry
	var driftRecord *internalModels.DriftRecord
	var lastErr error

	for attempt := 0; attempt < deh.config.MaxRetryAttempts; attempt++ {
		if attempt > 0 {
			// Wait before retry
			time.Sleep(deh.config.RetryDelay * time.Duration(attempt))
		}

		drift, err := detectionFunc(ctx, resource)
		if err == nil {
			driftRecord = drift
			break
		}

		lastErr = err
		
		// Check if error is retryable
		enhancedErr := deh.errorService.HandleError(ctx, err, context)
		if !enhancedErr.Retryable {
			// Non-retryable error, return immediately
			return nil, enhancedErr
		}
	}

	// If all retries failed
	if lastErr != nil {
		enhancedErr := deh.errorService.HandleError(ctx, lastErr, context)
		return nil, enhancedErr
	}

	return driftRecord, nil
}

// HandleProviderError handles provider-specific errors during drift detection
func (deh *DriftErrorHandler) HandleProviderError(
	ctx context.Context,
	provider string,
	region string,
	err error,
	operation string,
) *EnhancedError {
	context := ErrorContext{
		Operation: operation,
		Provider:  provider,
		Region:    region,
	}

	enhancedErr := deh.errorService.HandleError(ctx, err, context)
	
	// Publish provider error event
	deh.publishProviderErrorEvent(enhancedErr, provider, region)
	
	return enhancedErr
}

// HandleValidationError handles validation errors during drift detection
func (deh *DriftErrorHandler) HandleValidationError(
	ctx context.Context,
	resource models.Resource,
	err error,
	validationType string,
) *EnhancedError {
	context := ErrorContext{
		Operation:    "validation",
		Provider:     resource.Provider,
		Region:       resource.Region,
		ResourceID:   resource.ID,
		ResourceType: resource.Type,
		Metadata: map[string]interface{}{
			"validation_type": validationType,
		},
	}

	enhancedErr := deh.errorService.HandleError(ctx, err, context)
	
	// Publish validation error event
	deh.publishValidationErrorEvent(enhancedErr, resource)
	
	return enhancedErr
}

// GetDriftDetectionStatistics returns statistics about drift detection errors
func (deh *DriftErrorHandler) GetDriftDetectionStatistics() map[string]interface{} {
	stats := make(map[string]interface{})
	
	// Get all errors from error service
	allErrors := deh.errorService.GetErrorHistory()
	
	// Filter drift detection related errors
	driftErrors := make([]EnhancedError, 0)
	for _, err := range allErrors {
		if err.Context.Operation == "drift_detection" ||
		   err.Context.Operation == "validation" ||
		   strings.Contains(err.Message, "drift") {
			driftErrors = append(driftErrors, err)
		}
	}
	
	stats["total_drift_errors"] = len(driftErrors)
	
	// Errors by provider
	providerErrors := make(map[string]int)
	for _, err := range driftErrors {
		if err.Context.Provider != "" {
			providerErrors[err.Context.Provider]++
		}
	}
	stats["errors_by_provider"] = providerErrors
	
	// Errors by resource type
	resourceTypeErrors := make(map[string]int)
	for _, err := range driftErrors {
		if err.Context.ResourceType != "" {
			resourceTypeErrors[err.Context.ResourceType]++
		}
	}
	stats["errors_by_resource_type"] = resourceTypeErrors
	
	// Errors by severity
	severityErrors := make(map[ErrorSeverity]int)
	for _, err := range driftErrors {
		severityErrors[err.Severity]++
	}
	stats["errors_by_severity"] = severityErrors
	
	// Recent errors (last 24 hours)
	recentCount := 0
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, err := range driftErrors {
		if err.Timestamp.After(cutoff) {
			recentCount++
		}
	}
	stats["recent_drift_errors_24h"] = recentCount
	
	return stats
}

// Helper methods

func (deh *DriftErrorHandler) processResourceBatch(
	ctx context.Context,
	provider string,
	region string,
	detectionFunc func(ctx context.Context, resources []models.Resource) ([]internalModels.DriftRecord, error),
	resources []models.Resource,
) *DriftDetectionResult {
	result := &DriftDetectionResult{
		TotalResources: len(resources),
		Drifts:         make([]internalModels.DriftRecord, 0),
		Errors:         make([]EnhancedError, 0),
		Metadata:       make(map[string]interface{}),
	}

	// Attempt to process the batch
	drifts, err := detectionFunc(ctx, resources)
	if err != nil {
		// Handle batch error
		context := ErrorContext{
			Operation: "drift_detection_batch",
			Provider:  provider,
			Region:    region,
			Metadata: map[string]interface{}{
				"batch_size": len(resources),
			},
		}
		
		enhancedErr := deh.errorService.HandleError(ctx, err, context)
		result.Errors = append(result.Errors, *enhancedErr)
		result.FailedResources = len(resources)
		
		// If partial results are enabled, try to process resources individually
		if deh.config.EnablePartialResults {
			deh.processResourcesIndividually(ctx, provider, region, resources, result)
		}
	} else {
		// Success
		result.Drifts = drifts
		result.ProcessedResources = len(resources)
		result.Success = true
	}

	return result
}

func (deh *DriftErrorHandler) processResourcesIndividually(
	ctx context.Context,
	provider string,
	region string,
	resources []models.Resource,
	result *DriftDetectionResult,
) {
	for _, resource := range resources {
		// Create a simple detection function for individual resources
		detectionFunc := func(ctx context.Context, res models.Resource) (*internalModels.DriftRecord, error) {
			// This is a simplified implementation
			// In a real implementation, you would call the actual drift detection logic
			return nil, fmt.Errorf("individual resource drift detection not implemented")
		}
		
		drift, err := deh.HandleResourceDriftDetection(ctx, resource, detectionFunc)
		if err != nil {
			result.Errors = append(result.Errors, *err)
			result.FailedResources++
		} else if drift != nil {
			result.Drifts = append(result.Drifts, *drift)
			result.ProcessedResources++
		}
	}
}

func (deh *DriftErrorHandler) publishDriftDetectionEvent(result *DriftDetectionResult, provider, region string) {
	if deh.eventBus == nil {
		return
	}
	
	event := events.Event{
		Type:      events.EventType("drift.detection.completed"),
		Timestamp: time.Now(),
		Source:    "drift_error_handler",
		Data: map[string]interface{}{
			"provider":           provider,
			"region":             region,
			"success":            result.Success,
			"total_resources":    result.TotalResources,
			"processed_resources": result.ProcessedResources,
			"failed_resources":   result.FailedResources,
			"drifts_found":       len(result.Drifts),
			"errors_count":       len(result.Errors),
			"partial_results":    result.PartialResults,
			"duration_ms":        result.Duration.Milliseconds(),
		},
	}
	
	deh.eventBus.Publish(event)
}

func (deh *DriftErrorHandler) publishProviderErrorEvent(err *EnhancedError, provider, region string) {
	if deh.eventBus == nil {
		return
	}
	
	event := events.Event{
		Type:      events.EventType("drift.provider.error"),
		Timestamp: time.Now(),
		Source:    "drift_error_handler",
		Data: map[string]interface{}{
			"error_id":  err.ID,
			"provider":  provider,
			"region":    region,
			"severity":  string(err.Severity),
			"category":  string(err.Category),
			"message":   err.Message,
			"retryable": err.Retryable,
		},
	}
	
	deh.eventBus.Publish(event)
}

func (deh *DriftErrorHandler) publishValidationErrorEvent(err *EnhancedError, resource models.Resource) {
	if deh.eventBus == nil {
		return
	}
	
	event := events.Event{
		Type:      events.EventType("drift.validation.error"),
		Timestamp: time.Now(),
		Source:    "drift_error_handler",
		Data: map[string]interface{}{
			"error_id":      err.ID,
			"resource_id":   resource.ID,
			"resource_type": resource.Type,
			"provider":      resource.Provider,
			"region":        resource.Region,
			"severity":      string(err.Severity),
			"message":       err.Message,
		},
	}
	
	deh.eventBus.Publish(event)
}
