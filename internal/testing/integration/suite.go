package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/drift"
	"github.com/catherinevee/driftmgr/internal/credentials"
	"github.com/catherinevee/driftmgr/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// IntegrationTestSuite provides comprehensive integration testing
type IntegrationTestSuite struct {
	suite.Suite
	providers        map[string]CloudProvider
	testAccounts     map[string]*TestAccount
	testResources    map[string][]TestResource
	cleanup          []func() error
	logger           *logging.Logger
	metricsCollector *MetricsCollector
}

// TestAccount represents a test account configuration
type TestAccount struct {
	Provider     string
	AccountID    string
	Credentials  map[string]string
	Region       string
	MaxResources int
}

// TestResource represents a test resource
type TestResource struct {
	ID       string
	Type     string
	Provider string
	Region   string
	Tags     map[string]string
	Created  time.Time
}

// CloudProvider interface for test providers
type CloudProvider interface {
	CreateTestResource(ctx context.Context, resourceType string) (*TestResource, error)
	DeleteTestResource(ctx context.Context, resourceID string) error
	ModifyTestResource(ctx context.Context, resourceID string, changes map[string]interface{}) error
	VerifyResource(ctx context.Context, resourceID string) (bool, error)
}

// MetricsCollector collects test metrics
type MetricsCollector struct {
	mu      sync.Mutex
	metrics map[string][]float64
}

// SetupSuite initializes the test suite
func (s *IntegrationTestSuite) SetupSuite() {
	s.logger = logging.GetLogger()
	s.metricsCollector = &MetricsCollector{
		metrics: make(map[string][]float64),
	}

	// Initialize test accounts for each provider
	s.initializeTestAccounts()

	// Create baseline test resources
	s.createBaselineResources()

	s.logger.Info("Integration test suite initialized", map[string]interface{}{
		"providers": len(s.providers),
		"accounts":  len(s.testAccounts),
	})
}

// TearDownSuite cleans up after all tests
func (s *IntegrationTestSuite) TearDownSuite() {
	s.logger.Info("Cleaning up test resources")

	// Execute cleanup in reverse order
	for i := len(s.cleanup) - 1; i >= 0; i-- {
		if err := s.cleanup[i](); err != nil {
			s.logger.Error("Cleanup failed", err, nil)
		}
	}

	// Generate test report
	s.generateTestReport()
}

// TestFullDiscoveryFlow tests end-to-end discovery
func (s *IntegrationTestSuite) TestFullDiscoveryFlow() {
	ctx := context.Background()

	// Phase 1: Create test resources
	s.T().Run("CreateResources", func(t *testing.T) {
		resources := s.createTestResourcesForAllProviders(ctx)
		assert.NotEmpty(t, resources, "Should create test resources")

		// Register cleanup
		s.cleanup = append(s.cleanup, func() error {
			return s.deleteTestResources(ctx, resources)
		})
	})

	// Phase 2: Run discovery
	s.T().Run("Discovery", func(t *testing.T) {
		discoverer := discovery.NewEnhancedDiscovery()
		results, err := discoverer.DiscoverAll(ctx)

		assert.NoError(t, err, "Discovery should succeed")
		assert.NotNil(t, results, "Should return discovery results")

		// Verify all test resources were discovered
		for provider, resources := range s.testResources {
			providerResults := results[provider]
			assert.GreaterOrEqual(t, len(providerResults), len(resources),
				"Should discover all test resources for %s", provider)
		}

		// Record metrics
		s.recordMetric("discovery.duration", time.Since(time.Now()).Seconds())
		s.recordMetric("discovery.resource_count", float64(len(results)))
	})

	// Phase 3: Test drift detection
	s.T().Run("DriftDetection", func(t *testing.T) {
		// Modify some resources
		s.modifyRandomResources(ctx, 0.3) // Modify 30% of resources

		// Detect drift
		detector := drift.NewDetector()
		driftResults, err := detector.DetectAll(ctx)

		assert.NoError(t, err, "Drift detection should succeed")
		assert.True(t, driftResults.HasDrift(), "Should detect drift")
		assert.GreaterOrEqual(t, len(driftResults.DriftedResources), 1,
			"Should find drifted resources")
	})

	// Phase 4: Test remediation
	s.T().Run("Remediation", func(t *testing.T) {
		// Create remediation plan
		planner := drift.NewRemediationPlanner()
		plan, err := planner.CreatePlan(ctx)

		assert.NoError(t, err, "Should create remediation plan")
		assert.NotNil(t, plan, "Plan should not be nil")

		// Execute remediation
		executor := drift.NewRemediationExecutor()
		results, err := executor.Execute(ctx, plan)

		assert.NoError(t, err, "Remediation should succeed")
		assert.True(t, results.Success, "Remediation should be successful")

		// Verify no drift remains
		driftResults, _ := drift.NewDetector().DetectAll(ctx)
		assert.False(t, driftResults.HasDrift(), "Should have no drift after remediation")
	})
}

// TestConcurrentOperations tests concurrent user scenarios
func (s *IntegrationTestSuite) TestConcurrentOperations() {
	ctx := context.Background()
	numUsers := 10

	var wg sync.WaitGroup
	errors := make(chan error, numUsers)

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			// Each user performs operations
			if err := s.simulateUserOperations(ctx, userID); err != nil {
				errors <- fmt.Errorf("user %d: %w", userID, err)
			}
		}(i)
	}

	// Wait for all users
	wg.Wait()
	close(errors)

	// Check for errors
	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}

	assert.Empty(s.T(), errs, "Concurrent operations should succeed")
}

// TestFailureRecovery tests system recovery from failures
func (s *IntegrationTestSuite) TestFailureRecovery() {
	ctx := context.Background()

	s.T().Run("ProviderTimeout", func(t *testing.T) {
		// Simulate provider timeout
		ctx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
		defer cancel()

		discoverer := discovery.NewEnhancedDiscovery()
		results, err := discoverer.DiscoverAll(ctx)

		// Should handle timeout gracefully
		assert.Error(t, err, "Should return timeout error")
		assert.NotNil(t, results, "Should return partial results")
	})

	s.T().Run("RateLimitExceeded", func(t *testing.T) {
		// Simulate rate limit scenario
		for i := 0; i < 100; i++ {
			go func() {
				discovery.NewEnhancedDiscovery().DiscoverAll(ctx)
			}()
		}

		// System should handle rate limits without crashing
		time.Sleep(5 * time.Second)

		// Verify system is still responsive
		_, err := discovery.NewEnhancedDiscovery().DiscoverAll(ctx)
		assert.NoError(t, err, "System should remain responsive")
	})

	s.T().Run("InvalidCredentials", func(t *testing.T) {
		// Test with invalid credentials
		detector := credentials.NewCredentialDetector()

		// Temporarily corrupt credentials
		originalCreds := s.backupCredentials()
		s.corruptCredentials()
		defer s.restoreCredentials(originalCreds)

		// Should handle invalid credentials gracefully
		creds := detector.DetectAll()
		for _, cred := range creds {
			assert.NotEqual(t, "configured", cred.Status,
				"Should detect invalid credentials")
		}
	})
}

// TestPerformanceBaseline establishes performance baselines
func (s *IntegrationTestSuite) TestPerformanceBaseline() {
	ctx := context.Background()
	iterations := 10

	s.T().Run("DiscoveryLatency", func(t *testing.T) {
		latencies := []time.Duration{}

		for i := 0; i < iterations; i++ {
			start := time.Now()
			_, err := discovery.NewEnhancedDiscovery().DiscoverAll(ctx)
			latency := time.Since(start)

			assert.NoError(t, err)
			latencies = append(latencies, latency)
		}

		// Calculate statistics
		avg := s.calculateAverage(latencies)
		p95 := s.calculatePercentile(latencies, 95)
		p99 := s.calculatePercentile(latencies, 99)

		// Assert performance requirements
		assert.Less(t, avg, 5*time.Second, "Average latency should be < 5s")
		assert.Less(t, p95, 10*time.Second, "P95 latency should be < 10s")
		assert.Less(t, p99, 15*time.Second, "P99 latency should be < 15s")

		s.logger.Info("Discovery performance", map[string]interface{}{
			"avg_ms": avg.Milliseconds(),
			"p95_ms": p95.Milliseconds(),
			"p99_ms": p99.Milliseconds(),
		})
	})

	s.T().Run("Throughput", func(t *testing.T) {
		// Measure operations per second
		duration := 10 * time.Second
		operations := 0
		done := make(chan bool)

		go func() {
			start := time.Now()
			for time.Since(start) < duration {
				discovery.NewEnhancedDiscovery().DiscoverAll(ctx)
				operations++
			}
			done <- true
		}()

		<-done
		throughput := float64(operations) / duration.Seconds()

		assert.Greater(t, throughput, 1.0, "Should handle > 1 op/sec")

		s.logger.Info("Throughput test", map[string]interface{}{
			"ops_per_sec": throughput,
			"total_ops":   operations,
		})
	})
}

// TestCacheEffectiveness tests caching performance
func (s *IntegrationTestSuite) TestCacheEffectiveness() {
	ctx := context.Background()

	// First call - cache miss
	start1 := time.Now()
	result1, err1 := discovery.NewEnhancedDiscovery().DiscoverAll(ctx)
	duration1 := time.Since(start1)

	assert.NoError(s.T(), err1)

	// Second call - cache hit
	start2 := time.Now()
	result2, err2 := discovery.NewEnhancedDiscovery().DiscoverAll(ctx)
	duration2 := time.Since(start2)

	assert.NoError(s.T(), err2)
	assert.Equal(s.T(), result1, result2, "Results should be identical")

	// Cache should significantly improve performance
	speedup := duration1.Seconds() / duration2.Seconds()
	assert.Greater(s.T(), speedup, 5.0, "Cache should provide >5x speedup")

	s.logger.Info("Cache effectiveness", map[string]interface{}{
		"cold_ms": duration1.Milliseconds(),
		"warm_ms": duration2.Milliseconds(),
		"speedup": speedup,
	})
}

// Helper methods

func (s *IntegrationTestSuite) initializeTestAccounts() {
	s.testAccounts = map[string]*TestAccount{
		"aws": {
			Provider:     "aws",
			AccountID:    "test-aws-account",
			Region:       "us-west-2",
			MaxResources: 10,
		},
		"azure": {
			Provider:     "azure",
			AccountID:    "test-azure-subscription",
			Region:       "westus2",
			MaxResources: 10,
		},
		"gcp": {
			Provider:     "gcp",
			AccountID:    "test-gcp-project",
			Region:       "us-central1",
			MaxResources: 10,
		},
	}
}

func (s *IntegrationTestSuite) createBaselineResources() {
	ctx := context.Background()
	s.testResources = make(map[string][]TestResource)

	for provider, account := range s.testAccounts {
		resources := []TestResource{}

		// Create VPC/Network
		vpc := TestResource{
			ID:       fmt.Sprintf("%s-vpc-test", provider),
			Type:     "network",
			Provider: provider,
			Region:   account.Region,
			Tags: map[string]string{
				"Environment": "test",
				"ManagedBy":   "driftmgr-integration",
			},
			Created: time.Now(),
		}
		resources = append(resources, vpc)

		// Create compute instance
		instance := TestResource{
			ID:       fmt.Sprintf("%s-instance-test", provider),
			Type:     "compute",
			Provider: provider,
			Region:   account.Region,
			Tags: map[string]string{
				"Environment": "test",
				"ManagedBy":   "driftmgr-integration",
			},
			Created: time.Now(),
		}
		resources = append(resources, instance)

		s.testResources[provider] = resources
	}

	s.logger.Info("Baseline resources created", map[string]interface{}{
		"total": len(s.testResources),
	})
}

func (s *IntegrationTestSuite) createTestResourcesForAllProviders(ctx context.Context) map[string][]TestResource {
	result := make(map[string][]TestResource)

	for provider := range s.providers {
		// Create test resources for each provider
		resources := s.createProviderResources(ctx, provider, 5)
		result[provider] = resources
	}

	return result
}

func (s *IntegrationTestSuite) createProviderResources(ctx context.Context, provider string, count int) []TestResource {
	resources := []TestResource{}

	for i := 0; i < count; i++ {
		resource := TestResource{
			ID:       fmt.Sprintf("%s-resource-%d-%d", provider, time.Now().Unix(), i),
			Type:     "test-resource",
			Provider: provider,
			Region:   s.testAccounts[provider].Region,
			Tags: map[string]string{
				"Test":      "true",
				"Timestamp": fmt.Sprintf("%d", time.Now().Unix()),
			},
			Created: time.Now(),
		}
		resources = append(resources, resource)
	}

	return resources
}

func (s *IntegrationTestSuite) deleteTestResources(ctx context.Context, resources map[string][]TestResource) error {
	for provider, providerResources := range resources {
		for _, resource := range providerResources {
			s.logger.Debug("Deleting test resource", map[string]interface{}{
				"provider": provider,
				"resource": resource.ID,
			})
			// Actual deletion would happen here
		}
	}
	return nil
}

func (s *IntegrationTestSuite) modifyRandomResources(ctx context.Context, percentage float64) {
	for provider, resources := range s.testResources {
		numToModify := int(float64(len(resources)) * percentage)

		for i := 0; i < numToModify && i < len(resources); i++ {
			// Simulate resource modification
			resources[i].Tags["Modified"] = "true"
			resources[i].Tags["ModifiedAt"] = time.Now().Format(time.RFC3339)

			s.logger.Debug("Modified resource", map[string]interface{}{
				"provider": provider,
				"resource": resources[i].ID,
			})
		}
	}
}

func (s *IntegrationTestSuite) simulateUserOperations(ctx context.Context, userID int) error {
	// Simulate typical user workflow
	operations := []func(context.Context) error{
		func(ctx context.Context) error {
			_, err := discovery.NewEnhancedDiscovery().DiscoverAll(ctx)
			return err
		},
		func(ctx context.Context) error {
			_, err := drift.NewDetector().DetectAll(ctx)
			return err
		},
		func(ctx context.Context) error {
			// Simulate state query
			time.Sleep(100 * time.Millisecond)
			return nil
		},
	}

	for _, op := range operations {
		if err := op(ctx); err != nil {
			return err
		}

		// Random delay between operations
		time.Sleep(time.Duration(100+userID*10) * time.Millisecond)
	}

	return nil
}

func (s *IntegrationTestSuite) recordMetric(name string, value float64) {
	s.metricsCollector.mu.Lock()
	defer s.metricsCollector.mu.Unlock()

	s.metricsCollector.metrics[name] = append(s.metricsCollector.metrics[name], value)
}

func (s *IntegrationTestSuite) calculateAverage(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range durations {
		total += d
	}

	return total / time.Duration(len(durations))
}

func (s *IntegrationTestSuite) calculatePercentile(durations []time.Duration, percentile float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	index := int(float64(len(durations)) * percentile / 100)
	if index >= len(durations) {
		index = len(durations) - 1
	}

	return durations[index]
}

func (s *IntegrationTestSuite) backupCredentials() map[string]interface{} {
	// Backup current credentials
	return map[string]interface{}{
		"aws":   os.Getenv("AWS_ACCESS_KEY_ID"),
		"azure": os.Getenv("AZURE_CLIENT_ID"),
		"gcp":   os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
	}
}

func (s *IntegrationTestSuite) corruptCredentials() {
	os.Setenv("AWS_ACCESS_KEY_ID", "invalid")
	os.Setenv("AZURE_CLIENT_ID", "invalid")
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "invalid")
}

func (s *IntegrationTestSuite) restoreCredentials(original map[string]interface{}) {
	for key, value := range original {
		if v, ok := value.(string); ok {
			os.Setenv(key, v)
		}
	}
}

func (s *IntegrationTestSuite) generateTestReport() {
	report := TestReport{
		Timestamp: time.Now(),
		Duration:  time.Since(s.startTime),
		Metrics:   s.metricsCollector.metrics,
		Summary: map[string]interface{}{
			"total_tests":       s.T().Count(),
			"providers_tested":  len(s.providers),
			"resources_created": len(s.testResources),
		},
	}

	// Save report
	s.logger.Info("Test report generated", map[string]interface{}{
		"report": report,
	})
}

// TestReport represents the test execution report
type TestReport struct {
	Timestamp time.Time
	Duration  time.Duration
	Metrics   map[string][]float64
	Summary   map[string]interface{}
}

// Run the test suite
func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
