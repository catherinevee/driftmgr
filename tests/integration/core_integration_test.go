package integration

import (
	"context"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCoreIntegration tests the core integration functionality
func TestCoreIntegration(t *testing.T) {
	// Create integration test suite
	config := &testutils.IntegrationTestConfig{
		UseRealProviders: false, // Use mock providers for now
		TestDataPath:     "../../testdata",
		Timeout:          5 * time.Minute,
		Parallel:         false,
	}

	suite, err := testutils.NewIntegrationTestSuite(config)
	require.NoError(t, err)
	defer suite.Cleanup()

	// Create integration framework
	framework := testutils.NewIntegrationFramework(suite)

	t.Run("BasicProviderSetup", func(t *testing.T) {
		// Test basic provider setup
		err := framework.SetupProviders([]string{"aws"})
		assert.NoError(t, err)
	})

	t.Run("TestResourceCreation", func(t *testing.T) {
		// Test creating test resources
		resources, err := suite.CreateTestResources("aws", 3)
		require.NoError(t, err)
		assert.Len(t, resources, 3)

		// Validate resource structure
		for i, resource := range resources {
			assert.Equal(t, "aws", resource.Provider)
			assert.Equal(t, "test_resource", resource.Type)
			assert.Equal(t, "us-east-1", resource.Region)
			assert.Equal(t, "test-account", resource.AccountID)
			// Note: Tags is interface{} so we can't directly index it
			assert.Equal(t, i, resource.Properties["test_id"])
		}
	})

	t.Run("TestStateFileLoading", func(t *testing.T) {
		// Test loading state files
		_, err := suite.GetTestState("terraform.tfstate.example")
		require.NoError(t, err)
		// Note: We can't access state fields directly due to interface{} types
	})

	t.Run("TestDriftDetection", func(t *testing.T) {
		// Set up providers for drift detection
		err := framework.SetupProviders([]string{"aws"})
		require.NoError(t, err)

		// Get test state
		_, err = suite.GetTestState("terraform.tfstate.example")
		require.NoError(t, err)

		// Run drift detection
		result := framework.RunDriftDetectionTest(t, "terraform.tfstate.example")
		assert.True(t, result.Success, "Drift detection should succeed")
		assert.NoError(t, result.Error)
		assert.NotNil(t, result.DriftResults)
		assert.Greater(t, result.Duration, time.Duration(0))
	})

	t.Run("TestEndToEndWorkflow", func(t *testing.T) {
		// Set up providers
		err := framework.SetupProviders([]string{"aws"})
		require.NoError(t, err)

		// Run end-to-end test
		result := framework.RunEndToEndTest(t, "basic_e2e_test")
		assert.True(t, result.Success, "End-to-end test should succeed")
		assert.NoError(t, result.Error)
		assert.Greater(t, result.Duration, time.Duration(0))
		assert.Contains(t, result.Metrics, "total_resources")
		assert.Contains(t, result.Metrics, "providers_tested")
	})
}

// TestProviderIntegration tests provider-specific integration
func TestProviderIntegration(t *testing.T) {
	config := &testutils.IntegrationTestConfig{
		UseRealProviders: false,
		TestDataPath:     "../../testdata",
		Timeout:          5 * time.Minute,
		Parallel:         false,
	}

	suite, err := testutils.NewIntegrationTestSuite(config)
	require.NoError(t, err)
	defer suite.Cleanup()

	framework := testutils.NewIntegrationFramework(suite)

	providers := []string{"aws", "azure", "gcp", "digitalocean"}

	for _, providerName := range providers {
		t.Run("Provider_"+providerName, func(t *testing.T) {
			// Set up single provider
			err := framework.SetupProviders([]string{providerName})
			require.NoError(t, err)

			// Test resource creation
			resources, err := suite.CreateTestResources(providerName, 2)
			require.NoError(t, err)
			assert.Len(t, resources, 2)

			// Validate provider-specific attributes
			for _, resource := range resources {
				assert.Equal(t, providerName, resource.Provider)
				assert.NotEmpty(t, resource.ID)
				assert.NotEmpty(t, resource.Name)
				assert.NotEmpty(t, resource.Type)
			}
		})
	}
}

// TestDriftScenarios tests various drift detection scenarios
func TestDriftScenarios(t *testing.T) {
	config := &testutils.IntegrationTestConfig{
		UseRealProviders: false,
		TestDataPath:     "../../testdata",
		Timeout:          5 * time.Minute,
		Parallel:         false,
	}

	suite, err := testutils.NewIntegrationTestSuite(config)
	require.NoError(t, err)
	defer suite.Cleanup()

	framework := testutils.NewIntegrationFramework(suite)

	// Set up providers
	err = framework.SetupProviders([]string{"aws"})
	require.NoError(t, err)

	stateFiles := []string{
		"terraform.tfstate.example",
		"drift-scenarios.tfstate",
	}

	for _, stateFile := range stateFiles {
		t.Run("DriftScenario_"+stateFile, func(t *testing.T) {
			// Test drift detection with different state files
			result := framework.RunDriftDetectionTest(t, stateFile)
			assert.True(t, result.Success, "Drift detection should succeed for %s", stateFile)
			assert.NoError(t, result.Error)
			assert.NotNil(t, result.DriftResults)
			assert.Contains(t, result.Metrics, "total_resources")
		})
	}
}

// TestParallelExecution tests parallel test execution
func TestParallelExecution(t *testing.T) {
	config := &testutils.IntegrationTestConfig{
		UseRealProviders: false,
		TestDataPath:     "../../testdata",
		Timeout:          5 * time.Minute,
		Parallel:         true,
	}

	suite, err := testutils.NewIntegrationTestSuite(config)
	require.NoError(t, err)
	defer suite.Cleanup()

	framework := testutils.NewIntegrationFramework(suite)

	// Set up providers
	err = framework.SetupProviders([]string{"aws", "azure"})
	require.NoError(t, err)

	// Define parallel tests
	tests := []func(*testing.T) *testutils.IntegrationTestResult{
		func(t *testing.T) *testutils.IntegrationTestResult {
			return framework.RunDiscoveryTest(t, "aws", "us-east-1")
		},
		func(t *testing.T) *testutils.IntegrationTestResult {
			return framework.RunDiscoveryTest(t, "azure", "eastus")
		},
		func(t *testing.T) *testutils.IntegrationTestResult {
			return framework.RunDriftDetectionTest(t, "terraform.tfstate.example")
		},
	}

	// Run tests in parallel
	results := framework.RunParallelTests(t, tests)

	// Validate all results
	for i, result := range results {
		assert.NotNil(t, result, "Result %d should not be nil", i)
		if result != nil {
			assert.True(t, result.Success, "Test %d should succeed", i)
			assert.NoError(t, result.Error)
		}
	}
}

// TestErrorHandling tests error handling scenarios
func TestErrorHandling(t *testing.T) {
	config := &testutils.IntegrationTestConfig{
		UseRealProviders: false,
		TestDataPath:     "../../testdata",
		Timeout:          5 * time.Minute,
		Parallel:         false,
	}

	suite, err := testutils.NewIntegrationTestSuite(config)
	require.NoError(t, err)
	defer suite.Cleanup()

	framework := testutils.NewIntegrationFramework(suite)

	t.Run("InvalidProvider", func(t *testing.T) {
		// Test with invalid provider
		err := framework.SetupProviders([]string{"invalid_provider"})
		assert.Error(t, err)
	})

	t.Run("InvalidStateFile", func(t *testing.T) {
		// Test with invalid state file
		_, err := suite.GetTestState("nonexistent.tfstate")
		assert.Error(t, err)
	})

	t.Run("TimeoutScenario", func(t *testing.T) {
		// Test timeout scenario
		err := suite.RunWithTimeout(1*time.Millisecond, func(ctx context.Context) error {
			// Simulate long-running operation
			time.Sleep(10 * time.Millisecond)
			return nil
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})
}

// TestValidationAndReporting tests validation and reporting functionality
func TestValidationAndReporting(t *testing.T) {
	config := &testutils.IntegrationTestConfig{
		UseRealProviders: false,
		TestDataPath:     "../../testdata",
		Timeout:          5 * time.Minute,
		Parallel:         false,
	}

	suite, err := testutils.NewIntegrationTestSuite(config)
	require.NoError(t, err)
	defer suite.Cleanup()

	framework := testutils.NewIntegrationFramework(suite)

	// Set up providers
	err = framework.SetupProviders([]string{"aws"})
	require.NoError(t, err)

	// Run multiple tests
	results := []*testutils.IntegrationTestResult{
		framework.RunDiscoveryTest(t, "aws", "us-east-1"),
		framework.RunDriftDetectionTest(t, "terraform.tfstate.example"),
		framework.RunEndToEndTest(t, "validation_test"),
	}

	// Validate all results
	for _, result := range results {
		err := framework.ValidateTestResult(result)
		assert.NoError(t, err)
	}

	// Generate test report
	report := framework.GenerateTestReport(results)
	assert.NotNil(t, report)
	assert.Equal(t, len(results), report.TotalTests)
	assert.Greater(t, report.PassedTests, 0)
	assert.Greater(t, report.TotalDuration, time.Duration(0))
	assert.Contains(t, report.Summary, "success_rate")
	assert.Contains(t, report.Summary, "average_duration")
}
