package main

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/error_handling"
	"github.com/catherinevee/driftmgr/internal/events"
	internalModels "github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/pkg/models"
)

func main() {
	fmt.Println("Testing Enhanced Error Handling...")
	
	// Create event bus
	eventBus := events.NewEventBus()
	
	// Create error service configuration
	errorConfig := error_handling.ErrorConfig{
		MaxHistorySize:    1000,
		RetryAttempts:     3,
		RetryDelay:        1 * time.Second,
		EnableStackTraces: true,
		EnableMetrics:     true,
		SeverityThreshold: error_handling.ErrorSeverityLow,
	}
	
	// Create error service
	errorService := error_handling.NewErrorService(eventBus, errorConfig)
	
	// Create circuit breaker manager
	circuitBreakerManager := error_handling.NewCircuitBreakerManager()
	
	// Create drift error handler configuration
	driftErrorConfig := error_handling.DriftErrorConfig{
		MaxRetryAttempts:       3,
		RetryDelay:             2 * time.Second,
		CircuitBreakerThreshold: 5,
		CircuitBreakerTimeout:  30 * time.Second,
		EnablePartialResults:   true,
		ContinueOnError:        true,
	}
	
	// Create drift error handler
	driftErrorHandler := error_handling.NewDriftErrorHandler(errorService, eventBus, driftErrorConfig)
	
	// Test basic error handling
	fmt.Println("\n=== Testing Basic Error Handling ===")
	testBasicErrorHandling(errorService)
	
	// Test drift error handling
	fmt.Println("\n=== Testing Drift Error Handling ===")
	testDriftErrorHandling(driftErrorHandler)
	
	// Test circuit breaker
	fmt.Println("\n=== Testing Circuit Breaker ===")
	testCircuitBreaker(circuitBreakerManager)
	
	// Test error statistics
	fmt.Println("\n=== Testing Error Statistics ===")
	testErrorStatistics(errorService, driftErrorHandler)
	
	fmt.Println("\nEnhanced error handling testing completed!")
}

func testBasicErrorHandling(errorService *error_handling.ErrorService) {
	ctx := context.Background()
	
	// Test different types of errors
	testCases := []struct {
		name    string
		err     error
		context error_handling.ErrorContext
	}{
		{
			name: "Authentication Error",
			err:  fmt.Errorf("unauthorized access: invalid credentials"),
			context: error_handling.ErrorContext{
				Operation: "authentication",
				Provider:  "aws",
				Region:    "us-east-1",
			},
		},
		{
			name: "Network Error",
			err:  fmt.Errorf("connection timeout: unable to connect to AWS"),
			context: error_handling.ErrorContext{
				Operation: "resource_discovery",
				Provider:  "aws",
				Region:    "us-east-1",
			},
		},
		{
			name: "Validation Error",
			err:  fmt.Errorf("invalid resource configuration: missing required field"),
			context: error_handling.ErrorContext{
				Operation:    "validation",
				Provider:     "azure",
				Region:       "eastus",
				ResourceID:   "resource-123",
				ResourceType: "azurerm_storage_account",
			},
		},
		{
			name: "Critical Error",
			err:  fmt.Errorf("critical system failure: database connection lost"),
			context: error_handling.ErrorContext{
				Operation: "system_health_check",
			},
		},
	}
	
	for _, tc := range testCases {
		fmt.Printf("Testing %s...\n", tc.name)
		enhancedErr := errorService.HandleError(ctx, tc.err, tc.context)
		if enhancedErr != nil {
			fmt.Printf("  Severity: %s\n", enhancedErr.Severity)
			fmt.Printf("  Category: %s\n", enhancedErr.Category)
			fmt.Printf("  Retryable: %t\n", enhancedErr.Retryable)
			fmt.Printf("  Message: %s\n", enhancedErr.Message)
		}
	}
}

func testDriftErrorHandling(driftErrorHandler *error_handling.DriftErrorHandler) {
	ctx := context.Background()
	
	// Create mock resources
	resources := []models.Resource{
		{
			ID:         "resource-1",
			Name:       "test-bucket",
			Type:       "aws_s3_bucket",
			Provider:   "aws",
			Region:     "us-east-1",
			CreatedAt:  time.Now().Add(-24 * time.Hour),
		},
		{
			ID:         "resource-2",
			Name:       "test-instance",
			Type:       "aws_ec2_instance",
			Provider:   "aws",
			Region:     "us-east-1",
			CreatedAt:  time.Now().Add(-12 * time.Hour),
		},
	}
	
	// Mock drift detection function that simulates some failures
	detectionFunc := func(ctx context.Context, res []models.Resource) ([]internalModels.DriftRecord, error) {
		// Simulate some failures
		if len(res) > 1 {
			return nil, fmt.Errorf("batch processing failed: timeout")
		}
		
		// Return mock drift records
		return []internalModels.DriftRecord{
			{
				ID:           "drift-1",
				ResourceID:   res[0].ID,
				ResourceName: res[0].Name,
				ResourceType: res[0].Type,
				Provider:     res[0].Provider,
				Region:       res[0].Region,
				DriftType:    "configuration",
				Severity:     "high",
				Status:       "active",
				DetectedAt:   time.Now(),
				Description:  "S3 bucket has incorrect encryption settings",
			},
		}, nil
	}
	
	// Test drift detection with error handling
	result := driftErrorHandler.HandleDriftDetection(ctx, "aws", "us-east-1", detectionFunc, resources)
	
	fmt.Printf("Drift Detection Result:\n")
	fmt.Printf("  Success: %t\n", result.Success)
	fmt.Printf("  Total Resources: %d\n", result.TotalResources)
	fmt.Printf("  Processed Resources: %d\n", result.ProcessedResources)
	fmt.Printf("  Failed Resources: %d\n", result.FailedResources)
	fmt.Printf("  Drifts Found: %d\n", len(result.Drifts))
	fmt.Printf("  Errors: %d\n", len(result.Errors))
	fmt.Printf("  Partial Results: %t\n", result.PartialResults)
	fmt.Printf("  Duration: %v\n", result.Duration)
	
	// Test individual resource drift detection
	fmt.Println("\nTesting individual resource drift detection...")
	resource := resources[0]
	
	individualDetectionFunc := func(ctx context.Context, res models.Resource) (*internalModels.DriftRecord, error) {
		// Simulate success
		return &internalModels.DriftRecord{
			ID:           "drift-individual",
			ResourceID:   res.ID,
			ResourceName: res.Name,
			ResourceType: res.Type,
			Provider:     res.Provider,
			Region:       res.Region,
			DriftType:    "state",
			Severity:     "medium",
			Status:       "active",
			DetectedAt:   time.Now(),
			Description:  "Resource state mismatch detected",
		}, nil
	}
	
	drift, err := driftErrorHandler.HandleResourceDriftDetection(ctx, resource, individualDetectionFunc)
	if err != nil {
		fmt.Printf("  Error: %s\n", err.Message)
	} else if drift != nil {
		fmt.Printf("  Drift detected: %s\n", drift.Description)
	}
}

func testCircuitBreaker(circuitBreakerManager *error_handling.CircuitBreakerManager) {
	// Create a circuit breaker for AWS provider
	config := error_handling.CircuitBreakerConfig{
		Name:             "aws-provider",
		FailureThreshold: 3,
		Timeout:          10 * time.Second,
		OnStateChange: func(name string, from, to error_handling.CircuitState) {
			fmt.Printf("Circuit breaker %s changed from %s to %s\n", name, from, to)
		},
	}
	
	breaker := circuitBreakerManager.GetOrCreate("aws-provider", config)
	
	// Test circuit breaker with failing operations
	fmt.Println("Testing circuit breaker with failing operations...")
	
	for i := 0; i < 5; i++ {
		err := breaker.Execute(context.Background(), func() error {
			return fmt.Errorf("simulated failure %d", i+1)
		})
		
		if err != nil {
			fmt.Printf("  Attempt %d: %s (State: %s)\n", i+1, err.Error(), breaker.GetState())
		} else {
			fmt.Printf("  Attempt %d: Success (State: %s)\n", i+1, breaker.GetState())
		}
	}
	
	// Test circuit breaker with successful operation after timeout
	fmt.Println("\nWaiting for circuit breaker timeout...")
	time.Sleep(11 * time.Second)
	
	err := breaker.Execute(context.Background(), func() error {
		return nil // Success
	})
	
	if err != nil {
		fmt.Printf("  After timeout: %s (State: %s)\n", err.Error(), breaker.GetState())
	} else {
		fmt.Printf("  After timeout: Success (State: %s)\n", breaker.GetState())
	}
	
	// Test circuit breaker statistics
	stats := circuitBreakerManager.GetStatistics()
	fmt.Printf("\nCircuit Breaker Statistics:\n")
	for name, stat := range stats {
		fmt.Printf("  %s: %+v\n", name, stat)
	}
}

func testErrorStatistics(errorService *error_handling.ErrorService, driftErrorHandler *error_handling.DriftErrorHandler) {
	// Get general error statistics
	stats := errorService.GetErrorStatistics()
	fmt.Printf("General Error Statistics:\n")
	for key, value := range stats {
		fmt.Printf("  %s: %v\n", key, value)
	}
	
	// Get drift error statistics
	driftStats := driftErrorHandler.GetDriftDetectionStatistics()
	fmt.Printf("\nDrift Error Statistics:\n")
	for key, value := range driftStats {
		fmt.Printf("  %s: %v\n", key, value)
	}
	
	// Test error filtering
	fmt.Printf("\nError Filtering Tests:\n")
	
	// Get errors by severity
	highSeverityErrors := errorService.GetErrorsBySeverity(error_handling.ErrorSeverityHigh)
	fmt.Printf("  High severity errors: %d\n", len(highSeverityErrors))
	
	// Get errors by category
	networkErrors := errorService.GetErrorsByCategory(error_handling.ErrorCategoryNetwork)
	fmt.Printf("  Network errors: %d\n", len(networkErrors))
	
	// Get error history
	history := errorService.GetErrorHistory()
	fmt.Printf("  Total error history: %d\n", len(history))
}
