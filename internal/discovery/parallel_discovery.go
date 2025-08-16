package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/concurrency"
	"github.com/catherinevee/driftmgr/internal/models"
)

// ParallelDiscoverer provides parallel resource discovery capabilities
type ParallelDiscoverer struct {
	workerPool *concurrency.WorkerPool
	semaphore  *concurrency.Semaphore
	config     ParallelDiscoveryConfig
}

// ParallelDiscoveryConfig contains configuration for parallel discovery
type ParallelDiscoveryConfig struct {
	MaxWorkers     int
	MaxConcurrency int
	Timeout        time.Duration
	RetryAttempts  int
	RetryDelay     time.Duration
	BatchSize      int
	EnableMetrics  bool
}

// NewParallelDiscoverer creates a new parallel discoverer
func NewParallelDiscoverer(config ParallelDiscoveryConfig) *ParallelDiscoverer {
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = 10
	}
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = 5
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}
	if config.RetryAttempts <= 0 {
		config.RetryAttempts = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}

	return &ParallelDiscoverer{
		workerPool: concurrency.NewWorkerPool(config.MaxWorkers),
		semaphore:  concurrency.NewSemaphore(config.MaxConcurrency),
		config:     config,
	}
}

// DiscoverAllResources discovers resources across multiple providers and regions in parallel
func (pd *ParallelDiscoverer) DiscoverAllResources(ctx context.Context, providers []string, regions []string) ([]models.Resource, error) {
	start := time.Now()

	// Create channels for results and errors
	results := make(chan []models.Resource, len(providers)*len(regions))
	errors := make(chan error, len(providers)*len(regions))

	// Track completion
	var wg sync.WaitGroup
	completed := make(chan struct{})

	// Launch discovery goroutines
	for _, provider := range providers {
		for _, region := range regions {
			wg.Add(1)
			go func(p, r string) {
				defer wg.Done()
				pd.discoverProviderRegion(ctx, p, r, results, errors)
			}(provider, region)
		}
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(completed)
	}()

	// Collect results
	var allResources []models.Resource
	var discoveryErrors []error

	// Use a timeout context for result collection
	collectCtx, cancel := context.WithTimeout(ctx, pd.config.Timeout)
	defer cancel()

	for {
		select {
		case resources := <-results:
			allResources = append(allResources, resources...)
		case err := <-errors:
			discoveryErrors = append(discoveryErrors, err)
		case <-completed:
			goto done
		case <-collectCtx.Done():
			return nil, fmt.Errorf("discovery timed out after %v", pd.config.Timeout)
		}
	}

done:
	duration := time.Since(start)

	// Log metrics if enabled
	if pd.config.EnableMetrics {
		fmt.Printf("Parallel discovery completed in %v: %d resources, %d errors\n",
			duration, len(allResources), len(discoveryErrors))
	}

	// Return error if all discoveries failed
	if len(discoveryErrors) == len(providers)*len(regions) {
		return nil, fmt.Errorf("all discovery attempts failed: %v", discoveryErrors[0])
	}

	return allResources, nil
}

// discoverProviderRegion discovers resources for a specific provider and region
func (pd *ParallelDiscoverer) discoverProviderRegion(ctx context.Context, provider, region string,
	results chan<- []models.Resource, errors chan<- error) {

	// Acquire semaphore to limit concurrency
	pd.semaphore.Acquire()
	defer pd.semaphore.Release()

	// Create timeout context for this discovery
	discoveryCtx, cancel := context.WithTimeout(ctx, pd.config.Timeout)
	defer cancel()

	// Retry logic
	var lastErr error
	for attempt := 0; attempt < pd.config.RetryAttempts; attempt++ {
		select {
		case <-discoveryCtx.Done():
			errors <- fmt.Errorf("discovery timeout for %s/%s", provider, region)
			return
		default:
		}

		// Perform discovery with retry
		resources, err := pd.performDiscovery(discoveryCtx, provider, region)
		if err == nil {
			results <- resources
			return
		}

		lastErr = err

		// Wait before retry (except on last attempt)
		if attempt < pd.config.RetryAttempts-1 {
			select {
			case <-time.After(pd.config.RetryDelay):
			case <-discoveryCtx.Done():
				errors <- fmt.Errorf("discovery timeout for %s/%s", provider, region)
				return
			}
		}
	}

	errors <- fmt.Errorf("discovery failed for %s/%s after %d attempts: %v",
		provider, region, pd.config.RetryAttempts, lastErr)
}

// performDiscovery performs the actual resource discovery
func (pd *ParallelDiscoverer) performDiscovery(ctx context.Context, provider, region string) ([]models.Resource, error) {
	// This would integrate with the existing discovery logic
	// For now, we'll create a mock implementation

	switch provider {
	case "aws":
		return pd.discoverAWSResources(ctx, region)
	case "azure":
		return pd.discoverAzureResources(ctx, region)
	case "gcp":
		return pd.discoverGCPResources(ctx, region)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// discoverAWSResources discovers AWS resources with batching
func (pd *ParallelDiscoverer) discoverAWSResources(ctx context.Context, region string) ([]models.Resource, error) {
	var allResources []models.Resource

	// Define resource types to discover
	resourceTypes := []string{
		"ec2", "rds", "s3", "lambda", "iam", "cloudformation",
		"elasticache", "ecs", "eks", "route53", "sqs", "sns", "dynamodb",
	}

	// Discover resources in batches
	for i := 0; i < len(resourceTypes); i += pd.config.BatchSize {
		end := i + pd.config.BatchSize
		if end > len(resourceTypes) {
			end = len(resourceTypes)
		}

		batch := resourceTypes[i:end]
		resources, err := pd.discoverAWSBatch(ctx, region, batch)
		if err != nil {
			return nil, fmt.Errorf("failed to discover batch %d-%d: %w", i, end, err)
		}

		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

// discoverAWSBatch discovers a batch of AWS resource types
func (pd *ParallelDiscoverer) discoverAWSBatch(ctx context.Context, region string, resourceTypes []string) ([]models.Resource, error) {
	var allResources []models.Resource
	var mu sync.Mutex

	// Create a worker pool for this batch
	batchPool := concurrency.NewWorkerPool(len(resourceTypes))
	defer batchPool.Shutdown(30 * time.Second)

	// Submit discovery tasks
	for _, resourceType := range resourceTypes {
		rt := resourceType // Capture for closure
		err := batchPool.Submit(func() {
			resources, err := pd.discoverAWSResourceType(ctx, region, rt)
			if err != nil {
				// Log error but continue with other resource types
				fmt.Printf("Failed to discover %s in %s: %v\n", rt, region, err)
				return
			}

			mu.Lock()
			allResources = append(allResources, resources...)
			mu.Unlock()
		})

		if err != nil {
			return nil, fmt.Errorf("failed to submit discovery task for %s: %w", rt, err)
		}
	}

	return allResources, nil
}

// discoverAWSResourceType discovers a specific AWS resource type
func (pd *ParallelDiscoverer) discoverAWSResourceType(ctx context.Context, region, resourceType string) ([]models.Resource, error) {
	// This would integrate with the existing AWS discovery logic
	// For now, return mock data
	switch resourceType {
	case "ec2":
		return []models.Resource{
			{
				ID:       "i-mock1234567890",
				Name:     "mock-instance",
				Type:     "aws_instance",
				Provider: "aws",
				Region:   region,
				State:    "running",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			},
		}, nil
	case "s3":
		return []models.Resource{
			{
				ID:       "mock-bucket-123",
				Name:     "mock-bucket",
				Type:     "aws_s3_bucket",
				Provider: "aws",
				Region:   "global",
				State:    "active",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			},
		}, nil
	default:
		return []models.Resource{}, nil
	}
}

// discoverAzureResources discovers Azure resources
func (pd *ParallelDiscoverer) discoverAzureResources(ctx context.Context, region string) ([]models.Resource, error) {
	// Mock implementation for Azure
	return []models.Resource{
		{
			ID:       "azure-vm-123",
			Name:     "mock-azure-vm",
			Type:     "azurerm_virtual_machine",
			Provider: "azure",
			Region:   region,
			State:    "running",
			Created:  time.Now().Add(-24 * time.Hour),
			Updated:  time.Now(),
		},
	}, nil
}

// discoverGCPResources discovers GCP resources
func (pd *ParallelDiscoverer) discoverGCPResources(ctx context.Context, region string) ([]models.Resource, error) {
	// Mock implementation for GCP
	return []models.Resource{
		{
			ID:       "gcp-instance-123",
			Name:     "mock-gcp-instance",
			Type:     "google_compute_instance",
			Provider: "gcp",
			Region:   region,
			State:    "running",
			Created:  time.Now().Add(-24 * time.Hour),
			Updated:  time.Now(),
		},
	}, nil
}

// GetMetrics returns discovery metrics
func (pd *ParallelDiscoverer) GetMetrics() DiscoveryMetrics {
	return DiscoveryMetrics{
		TotalDiscoveries: 0, // Would track actual metrics
		AverageDuration:  0,
		SuccessRate:      0,
		ErrorCount:       0,
	}
}

// DiscoveryMetrics contains metrics about discovery operations
type DiscoveryMetrics struct {
	TotalDiscoveries int
	AverageDuration  time.Duration
	SuccessRate      float64
	ErrorCount       int
}

// Shutdown gracefully shuts down the parallel discoverer
func (pd *ParallelDiscoverer) Shutdown(timeout time.Duration) error {
	return pd.workerPool.Shutdown(timeout)
}
