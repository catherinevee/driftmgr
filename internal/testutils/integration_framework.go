package testutils

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/drift/detector"
	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/catherinevee/driftmgr/pkg/models"
)

// IntegrationFramework provides a comprehensive testing framework
type IntegrationFramework struct {
	suite     *IntegrationTestSuite
	providers map[string]providers.CloudProvider
	detector  *detector.DriftDetector
	mu        sync.RWMutex
}

// IntegrationTestResult contains the result of an integration test
type IntegrationTestResult struct {
	TestName     string
	Success      bool
	Duration     time.Duration
	Error        error
	Resources    []models.Resource
	DriftResults []detector.DriftResult
	Metrics      map[string]interface{}
}

// NewIntegrationFramework creates a new integration testing framework
func NewIntegrationFramework(suite *IntegrationTestSuite) *IntegrationFramework {
	return &IntegrationFramework{
		suite:     suite,
		providers: make(map[string]providers.CloudProvider),
	}
}

// SetupProviders sets up cloud providers for testing
func (ifw *IntegrationFramework) SetupProviders(providerNames []string) error {
	ifw.mu.Lock()
	defer ifw.mu.Unlock()

	for _, providerName := range providerNames {
		provider, err := ifw.suite.GetTestProvider(providerName)
		if err != nil {
			return fmt.Errorf("failed to get provider %s: %w", providerName, err)
		}
		ifw.providers[providerName] = provider
	}

	// Initialize drift detector with providers
	ifw.detector = detector.NewDriftDetector(ifw.providers)

	return nil
}

// RunDiscoveryTest runs a resource discovery test
func (ifw *IntegrationFramework) RunDiscoveryTest(t *testing.T, providerName, region string) *IntegrationTestResult {
	start := time.Now()
	result := &IntegrationTestResult{
		TestName: fmt.Sprintf("discovery_%s_%s", providerName, region),
		Metrics:  make(map[string]interface{}),
	}

	provider, exists := ifw.providers[providerName]
	if !exists {
		result.Error = fmt.Errorf("provider %s not available", providerName)
		result.Success = false
		result.Duration = time.Since(start)
		return result
	}

	// Run discovery
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	resources, err := provider.DiscoverResources(ctx, region)
	if err != nil {
		result.Error = fmt.Errorf("discovery failed: %w", err)
		result.Success = false
		result.Duration = time.Since(start)
		return result
	}

	result.Resources = resources
	result.Success = true
	result.Duration = time.Since(start)
	result.Metrics["resource_count"] = len(resources)
	result.Metrics["provider"] = providerName
	result.Metrics["region"] = region

	return result
}

// RunDriftDetectionTest runs a drift detection test
func (ifw *IntegrationFramework) RunDriftDetectionTest(t *testing.T, stateFileName string) *IntegrationTestResult {
	start := time.Now()
	result := &IntegrationTestResult{
		TestName: fmt.Sprintf("drift_detection_%s", stateFileName),
		Metrics:  make(map[string]interface{}),
	}

	// Get test state
	state, err := ifw.suite.GetTestState(stateFileName)
	if err != nil {
		result.Error = fmt.Errorf("failed to get test state: %w", err)
		result.Success = false
		result.Duration = time.Since(start)
		return result
	}

	// Run drift detection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	report, err := ifw.detector.DetectDrift(ctx, state)
	if err != nil {
		result.Error = fmt.Errorf("drift detection failed: %w", err)
		result.Success = false
		result.Duration = time.Since(start)
		return result
	}

	result.DriftResults = report.DriftResults
	result.Success = true
	result.Duration = time.Since(start)
	result.Metrics["total_resources"] = report.TotalResources
	result.Metrics["drifted_resources"] = report.DriftedResources
	result.Metrics["missing_resources"] = report.MissingResources
	result.Metrics["unmanaged_resources"] = report.UnmanagedResources

	return result
}

// RunEndToEndTest runs a complete end-to-end test
func (ifw *IntegrationFramework) RunEndToEndTest(t *testing.T, testName string) *IntegrationTestResult {
	start := time.Now()
	result := &IntegrationTestResult{
		TestName: testName,
		Metrics:  make(map[string]interface{}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Step 1: Discover resources
	discoveryResults := make(map[string][]models.Resource)
	for providerName := range ifw.providers {
		provider := ifw.providers[providerName]
		resources, err := provider.DiscoverResources(ctx, "us-east-1")
		if err != nil {
			result.Error = fmt.Errorf("discovery failed for %s: %w", providerName, err)
			result.Success = false
			result.Duration = time.Since(start)
			return result
		}
		discoveryResults[providerName] = resources
	}

	// Step 2: Load state and detect drift
	state, err := ifw.suite.GetTestState("terraform.tfstate.example")
	if err != nil {
		result.Error = fmt.Errorf("failed to get test state: %w", err)
		result.Success = false
		result.Duration = time.Since(start)
		return result
	}

	report, err := ifw.detector.DetectDrift(ctx, state)
	if err != nil {
		result.Error = fmt.Errorf("drift detection failed: %w", err)
		result.Success = false
		result.Duration = time.Since(start)
		return result
	}

	// Step 3: Validate results
	if report.TotalResources == 0 {
		result.Error = fmt.Errorf("no resources found in state")
		result.Success = false
		result.Duration = time.Since(start)
		return result
	}

	result.Success = true
	result.Duration = time.Since(start)
	result.Metrics["total_resources"] = report.TotalResources
	result.Metrics["drifted_resources"] = report.DriftedResources
	result.Metrics["providers_tested"] = len(ifw.providers)
	result.Metrics["discovery_results"] = len(discoveryResults)

	return result
}

// RunParallelTests runs multiple tests in parallel
func (ifw *IntegrationFramework) RunParallelTests(t *testing.T, tests []func(*testing.T) *IntegrationTestResult) []*IntegrationTestResult {
	var wg sync.WaitGroup
	results := make([]*IntegrationTestResult, len(tests))
	resultChan := make(chan struct {
		index  int
		result *IntegrationTestResult
	}, len(tests))

	for i, test := range tests {
		wg.Add(1)
		go func(index int, testFunc func(*testing.T) *IntegrationTestResult) {
			defer wg.Done()
			result := testFunc(t)
			resultChan <- struct {
				index  int
				result *IntegrationTestResult
			}{index, result}
		}(i, test)
	}

	wg.Wait()
	close(resultChan)

	for result := range resultChan {
		results[result.index] = result.result
	}

	return results
}

// ValidateTestResult validates a test result and reports issues
func (ifw *IntegrationFramework) ValidateTestResult(result *IntegrationTestResult) error {
	if result == nil {
		return fmt.Errorf("test result is nil")
	}

	if !result.Success {
		return fmt.Errorf("test %s failed: %v", result.TestName, result.Error)
	}

	// Validate duration (should not be too long)
	if result.Duration > 30*time.Minute {
		return fmt.Errorf("test %s took too long: %v", result.TestName, result.Duration)
	}

	// Validate metrics
	if result.Metrics == nil {
		return fmt.Errorf("test %s has no metrics", result.TestName)
	}

	return nil
}

// GenerateTestReport generates a comprehensive test report
func (ifw *IntegrationFramework) GenerateTestReport(results []*IntegrationTestResult) *TestReport {
	report := &TestReport{
		Timestamp:     time.Now(),
		TotalTests:    len(results),
		PassedTests:   0,
		FailedTests:   0,
		TotalDuration: 0,
		TestResults:   results,
		Summary:       make(map[string]interface{}),
	}

	for _, result := range results {
		report.TotalDuration += result.Duration
		if result.Success {
			report.PassedTests++
		} else {
			report.FailedTests++
		}
	}

	// Calculate summary metrics
	report.Summary["success_rate"] = float64(report.PassedTests) / float64(report.TotalTests) * 100
	report.Summary["average_duration"] = report.TotalDuration / time.Duration(report.TotalTests)
	report.Summary["total_resources_tested"] = ifw.calculateTotalResources(results)
	report.Summary["total_drift_detected"] = ifw.calculateTotalDrift(results)

	return report
}

// calculateTotalResources calculates total resources tested across all results
func (ifw *IntegrationFramework) calculateTotalResources(results []*IntegrationTestResult) int {
	total := 0
	for _, result := range results {
		total += len(result.Resources)
	}
	return total
}

// calculateTotalDrift calculates total drift detected across all results
func (ifw *IntegrationFramework) calculateTotalDrift(results []*IntegrationTestResult) int {
	total := 0
	for _, result := range results {
		total += len(result.DriftResults)
	}
	return total
}

// TestReport contains a comprehensive test report
type TestReport struct {
	Timestamp     time.Time
	TotalTests    int
	PassedTests   int
	FailedTests   int
	TotalDuration time.Duration
	TestResults   []*IntegrationTestResult
	Summary       map[string]interface{}
}

// Cleanup cleans up the integration framework
func (ifw *IntegrationFramework) Cleanup() error {
	return ifw.suite.Cleanup()
}
