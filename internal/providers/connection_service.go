package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/events"
)

// ConnectionService provides high-level connection testing functionality
type ConnectionService struct {
	tester   ConnectionTester
	factory  *ProviderFactory
	eventBus *events.EventBus
	timeout  time.Duration
	results  map[string][]ConnectionTestResult
	mu       sync.RWMutex
}

// NewConnectionService creates a new connection service
func NewConnectionService(factory *ProviderFactory, eventBus *events.EventBus, timeout time.Duration) *ConnectionService {
	return &ConnectionService{
		tester:   NewConnectionTester(timeout),
		factory:  factory,
		eventBus: eventBus,
		timeout:  timeout,
		results:  make(map[string][]ConnectionTestResult),
	}
}

// TestProviderConnection tests connection to a specific provider
func (cs *ConnectionService) TestProviderConnection(ctx context.Context, providerName, region string) (*ConnectionTestResult, error) {
	// Create provider
	provider, err := cs.factory.CreateProvider(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", providerName, err)
	}

	// Test connection
	result, err := cs.tester.TestConnection(ctx, provider, region)
	if err != nil {
		return nil, fmt.Errorf("connection test failed: %w", err)
	}

	// Store result
	cs.mu.Lock()
	cs.results[providerName] = append(cs.results[providerName], *result)
	cs.mu.Unlock()

	// Publish event
	cs.publishConnectionEvent("connection.test.completed", providerName, region, result)

	return result, nil
}

// TestProviderService tests connection to a specific service of a provider
func (cs *ConnectionService) TestProviderService(ctx context.Context, providerName, region, service string) (*ConnectionTestResult, error) {
	// Create provider
	provider, err := cs.factory.CreateProvider(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", providerName, err)
	}

	// Test service connection
	result, err := cs.tester.TestServiceConnection(ctx, provider, region, service)
	if err != nil {
		return nil, fmt.Errorf("service connection test failed: %w", err)
	}

	// Store result
	cs.mu.Lock()
	cs.results[providerName] = append(cs.results[providerName], *result)
	cs.mu.Unlock()

	// Publish event
	cs.publishConnectionEvent("connection.service.test.completed", providerName, region, result)

	return result, nil
}

// TestAllProviders tests connection to all available providers
func (cs *ConnectionService) TestAllProviders(ctx context.Context, region string) (map[string]*ConnectionTestResult, error) {
	results := make(map[string]*ConnectionTestResult)

	// Get all available providers
	providers := []string{"aws", "azure", "gcp", "digitalocean"}

	// Test each provider concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, providerName := range providers {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			result, err := cs.TestProviderConnection(ctx, name, region)
			if err != nil {
				// Create error result
				result = &ConnectionTestResult{
					Provider: name,
					Region:   region,
					Success:  false,
					Error:    err.Error(),
					TestedAt: time.Now(),
				}
			}

			mu.Lock()
			results[name] = result
			mu.Unlock()
		}(providerName)
	}

	wg.Wait()

	// Publish summary event
	cs.publishConnectionSummaryEvent("connection.all.providers.tested", region, results)

	return results, nil
}

// TestProviderAllRegions tests connection to all regions of a provider
func (cs *ConnectionService) TestProviderAllRegions(ctx context.Context, providerName string) ([]ConnectionTestResult, error) {
	// Create provider
	provider, err := cs.factory.CreateProvider(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", providerName, err)
	}

	// Test all regions
	results, err := cs.tester.TestAllRegions(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("all regions test failed: %w", err)
	}

	// Store results
	cs.mu.Lock()
	cs.results[providerName] = append(cs.results[providerName], results...)
	cs.mu.Unlock()

	// Publish event
	cs.publishConnectionEvent("connection.all.regions.tested", providerName, "", &ConnectionTestResult{
		Provider: providerName,
		Success:  true,
		Details: map[string]interface{}{
			"regions_tested":     len(results),
			"successful_regions": cs.countSuccessfulResults(results),
		},
		TestedAt: time.Now(),
	})

	return results, nil
}

// TestProviderAllServices tests connection to all services of a provider in a region
func (cs *ConnectionService) TestProviderAllServices(ctx context.Context, providerName, region string) ([]ConnectionTestResult, error) {
	// Create provider
	provider, err := cs.factory.CreateProvider(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", providerName, err)
	}

	// Test all services
	results, err := cs.tester.TestAllServices(ctx, provider, region)
	if err != nil {
		return nil, fmt.Errorf("all services test failed: %w", err)
	}

	// Store results
	cs.mu.Lock()
	cs.results[providerName] = append(cs.results[providerName], results...)
	cs.mu.Unlock()

	// Publish event
	cs.publishConnectionEvent("connection.all.services.tested", providerName, region, &ConnectionTestResult{
		Provider: providerName,
		Region:   region,
		Success:  true,
		Details: map[string]interface{}{
			"services_tested":     len(results),
			"successful_services": cs.countSuccessfulResults(results),
		},
		TestedAt: time.Now(),
	})

	return results, nil
}

// GetConnectionResults returns stored connection test results
func (cs *ConnectionService) GetConnectionResults(providerName string) []ConnectionTestResult {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if results, exists := cs.results[providerName]; exists {
		return results
	}
	return []ConnectionTestResult{}
}

// GetAllConnectionResults returns all stored connection test results
func (cs *ConnectionService) GetAllConnectionResults() map[string][]ConnectionTestResult {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// Create a copy to avoid race conditions
	results := make(map[string][]ConnectionTestResult)
	for provider, providerResults := range cs.results {
		results[provider] = make([]ConnectionTestResult, len(providerResults))
		copy(results[provider], providerResults)
	}

	return results
}

// ClearConnectionResults clears stored connection test results
func (cs *ConnectionService) ClearConnectionResults(providerName string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if providerName == "" {
		// Clear all results
		cs.results = make(map[string][]ConnectionTestResult)
	} else {
		// Clear specific provider results
		delete(cs.results, providerName)
	}
}

// GetConnectionSummary returns a summary of connection test results
func (cs *ConnectionService) GetConnectionSummary() map[string]interface{} {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	summary := make(map[string]interface{})

	for provider, results := range cs.results {
		providerSummary := map[string]interface{}{
			"total_tests":      len(results),
			"successful_tests": cs.countSuccessfulResults(results),
			"failed_tests":     len(results) - cs.countSuccessfulResults(results),
			"success_rate":     cs.calculateSuccessRate(results),
			"last_test":        cs.getLastTestTime(results),
		}
		summary[provider] = providerSummary
	}

	return summary
}

// RunHealthCheck runs a comprehensive health check on all providers
func (cs *ConnectionService) RunHealthCheck(ctx context.Context, region string) (*HealthCheckResult, error) {
	start := time.Now()

	// Test all providers
	results, err := cs.TestAllProviders(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	// Calculate health metrics
	healthResult := &HealthCheckResult{
		Region:      region,
		StartedAt:   start,
		CompletedAt: time.Now(),
		Duration:    time.Since(start),
		Results:     results,
		Summary:     cs.calculateHealthSummary(results),
	}

	// Publish health check event
	cs.publishHealthCheckEvent(healthResult)

	return healthResult, nil
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Region      string                           `json:"region"`
	StartedAt   time.Time                        `json:"started_at"`
	CompletedAt time.Time                        `json:"completed_at"`
	Duration    time.Duration                    `json:"duration"`
	Results     map[string]*ConnectionTestResult `json:"results"`
	Summary     map[string]interface{}           `json:"summary"`
}

// Helper methods

func (cs *ConnectionService) countSuccessfulResults(results []ConnectionTestResult) int {
	count := 0
	for _, result := range results {
		if result.Success {
			count++
		}
	}
	return count
}

func (cs *ConnectionService) calculateSuccessRate(results []ConnectionTestResult) float64 {
	if len(results) == 0 {
		return 0.0
	}
	successful := cs.countSuccessfulResults(results)
	return float64(successful) / float64(len(results)) * 100
}

func (cs *ConnectionService) getLastTestTime(results []ConnectionTestResult) *time.Time {
	if len(results) == 0 {
		return nil
	}

	var lastTime time.Time
	for _, result := range results {
		if result.TestedAt.After(lastTime) {
			lastTime = result.TestedAt
		}
	}
	return &lastTime
}

func (cs *ConnectionService) calculateHealthSummary(results map[string]*ConnectionTestResult) map[string]interface{} {
	totalProviders := len(results)
	successfulProviders := 0
	totalLatency := time.Duration(0)

	for _, result := range results {
		if result.Success {
			successfulProviders++
			totalLatency += result.Latency
		}
	}

	avgLatency := time.Duration(0)
	if successfulProviders > 0 {
		avgLatency = totalLatency / time.Duration(successfulProviders)
	}

	return map[string]interface{}{
		"total_providers":      totalProviders,
		"successful_providers": successfulProviders,
		"failed_providers":     totalProviders - successfulProviders,
		"success_rate":         float64(successfulProviders) / float64(totalProviders) * 100,
		"average_latency":      avgLatency.String(),
	}
}

func (cs *ConnectionService) publishConnectionEvent(eventType, provider, region string, result *ConnectionTestResult) {
	if cs.eventBus == nil {
		return
	}

	event := events.Event{
		Type:      events.EventType(eventType),
		Timestamp: time.Now(),
		Source:    "connection_service",
		Data: map[string]interface{}{
			"provider": provider,
			"region":   region,
			"success":  result.Success,
			"latency":  result.Latency.String(),
			"error":    result.Error,
			"details":  result.Details,
		},
	}

	cs.eventBus.Publish(event)
}

func (cs *ConnectionService) publishConnectionSummaryEvent(eventType, region string, results map[string]*ConnectionTestResult) {
	if cs.eventBus == nil {
		return
	}

	summary := cs.calculateHealthSummary(results)

	event := events.Event{
		Type:      events.EventType(eventType),
		Timestamp: time.Now(),
		Source:    "connection_service",
		Data: map[string]interface{}{
			"region":  region,
			"summary": summary,
			"results": results,
		},
	}

	cs.eventBus.Publish(event)
}

func (cs *ConnectionService) publishHealthCheckEvent(result *HealthCheckResult) {
	if cs.eventBus == nil {
		return
	}

	event := events.Event{
		Type:      events.EventType("connection.health_check.completed"),
		Timestamp: time.Now(),
		Source:    "connection_service",
		Data: map[string]interface{}{
			"region":   result.Region,
			"duration": result.Duration.String(),
			"summary":  result.Summary,
			"results":  result.Results,
		},
	}

	cs.eventBus.Publish(event)
}
