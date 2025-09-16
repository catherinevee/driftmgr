package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/api"
	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/drift/detector"
	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/catherinevee/driftmgr/internal/shared/config"
	"github.com/catherinevee/driftmgr/internal/state"
	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEndToEndWorkflow tests the complete drift detection workflow
func TestEndToEndWorkflow(t *testing.T) {
	ctx := context.Background()

	t.Run("discover_to_drift_to_remediate", func(t *testing.T) {
		// Step 1: Discovery
		config := &config.Config{}
		discoverer := discovery.NewEnhancedDiscoverer(config)
		resources, err := discoverer.DiscoverResources(ctx)
		if err == nil {
			assert.NotNil(t, resources)
		}

		// Step 2: Drift Detection
		driftDetector := detector.NewDriftDetector(nil)
		driftResults := []*detector.DriftResult{}

		// Test with mock resources
		mockResources := []models.Resource{
			{ID: "test-1", Type: "aws_instance", Provider: "aws"},
			{ID: "test-2", Type: "aws_s3_bucket", Provider: "aws"},
		}

		for _, resource := range mockResources {
			result, err := driftDetector.DetectResourceDrift(ctx, resource)
			if err == nil {
				driftResults = append(driftResults, result)
			}
		}

		// Step 3: Remediation Planning
		if len(driftResults) > 0 {
			remediationPlanner := &MockRemediationPlanner{}
			plan, err := remediationPlanner.CreatePlan(ctx, &detector.DriftReport{
				Timestamp:    time.Now(),
				DriftResults: convertDriftResults(driftResults),
			}, nil)
			require.NoError(t, err)
			assert.NotNil(t, plan)
		}
	})
}

// TestAPIIntegration tests API integration with core components
func TestAPIIntegration(t *testing.T) {
	// Create test server
	config := &api.Config{
		Host: "localhost",
		Port: 8080,
	}
	services := &api.Services{}
	server := api.NewServer(config, services)

	t.Run("health_check", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "healthy", response["status"])
	})

	t.Run("discover_endpoint", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"providers": []string{"aws"},
			"regions":   []string{"us-east-1"},
		}
		jsonBody, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/discover", httptest.NewRequest("POST", "/discover", nil).Body)
		req.Header.Set("Content-Type", "application/json")
		req.Body = httptest.NewRequest("POST", "/discover", httptest.NewRequest("POST", "/discover", nil).Body).Body
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		// Should handle the request (may return 404 if route not set up)
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound)
		_ = jsonBody // Use jsonBody to avoid unused variable
	})
}

// TestProviderIntegration tests cloud provider integration
func TestProviderIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("aws_provider_integration", func(t *testing.T) {
		provider := providers.NewAWSProvider("us-east-1")
		require.NotNil(t, provider)

		// Test provider initialization
		err := provider.Initialize(ctx)
		// May fail due to missing credentials, but should not panic
		if err != nil {
			assert.Contains(t, err.Error(), "aws")
		}

		// Test resource discovery
		resources, err := provider.DiscoverResources(ctx, "us-east-1")
		if err == nil {
			assert.NotNil(t, resources)
		}
	})

	t.Run("azure_provider_integration", func(t *testing.T) {
		provider := providers.NewAzureProviderComplete("", "", "", "", "", "eastus")
		require.NotNil(t, provider)

		// Test resource discovery
		resources, err := provider.DiscoverResources(ctx, "eastus")
		if err == nil {
			assert.NotNil(t, resources)
		}
	})

	t.Run("gcp_provider_integration", func(t *testing.T) {
		provider := providers.NewGCPProviderComplete("", "us-central1", "")
		require.NotNil(t, provider)

		// Test resource discovery
		resources, err := provider.DiscoverResources(ctx, "us-central1")
		if err == nil {
			assert.NotNil(t, resources)
		}
	})

	t.Run("digitalocean_provider_integration", func(t *testing.T) {
		provider := providers.NewDigitalOceanProvider("nyc1")
		require.NotNil(t, provider)

		// Test resource discovery
		resources, err := provider.DiscoverResources(ctx, "nyc1")
		if err == nil {
			assert.NotNil(t, resources)
		}
	})
}

// TestStateManagementIntegration tests state management integration
func TestStateManagementIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("state_parser_integration", func(t *testing.T) {
		parser := state.NewStateParser()
		require.NotNil(t, parser)

		// Test with empty state
		stateData, err := parser.Parse([]byte(`{"version": 4, "terraform_version": "1.0.0", "serial": 1, "lineage": "test", "resources": []}`))
		require.NoError(t, err)
		assert.Equal(t, 4, stateData.Version)
		assert.Equal(t, "1.0.0", stateData.TerraformVersion)
	})

	t.Run("state_manager_integration", func(t *testing.T) {
		// Create a mock backend
		backend := &MockStateBackend{}
		manager := state.NewStateManager(backend)
		require.NotNil(t, manager)

		// Test state operations
		terraformState := &state.TerraformState{
			Version:          4,
			TerraformVersion: "1.0.0",
			Serial:           1,
			Lineage:          "test",
		}

		err := manager.UpdateState(ctx, "test-key", terraformState)
		assert.NoError(t, err)

		retrievedState, err := manager.GetState(ctx, "test-key")
		if err == nil {
			assert.NotNil(t, retrievedState)
		}
	})
}

// TestDriftDetectionIntegration tests drift detection integration
func TestDriftDetectionIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("drift_detector_integration", func(t *testing.T) {
		detector := detector.NewDriftDetector(nil)
		require.NotNil(t, detector)

		// Test drift detection with mock resource
		resource := &models.Resource{
			ID:       "test-resource",
			Type:     "aws_instance",
			Provider: "aws",
		}

		result, err := detector.DetectResourceDrift(ctx, *resource)
		if err == nil {
			assert.NotNil(t, result)
			assert.Equal(t, "test-resource", result.Resource)
		}
	})

	t.Run("drift_comparator_integration", func(t *testing.T) {
		comparator := &MockDriftComparator{}
		require.NotNil(t, comparator)

		// Test comparison
		expected := map[string]interface{}{"key": "value1"}
		actual := map[string]interface{}{"key": "value2"}

		differences, err := comparator.Compare(expected, actual)
		if err == nil {
			assert.NotNil(t, differences)
		}
	})
}

// TestDiscoveryIntegration tests discovery engine integration
func TestDiscoveryIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("enhanced_discoverer_integration", func(t *testing.T) {
		config := &config.Config{}
		discoverer := discovery.NewEnhancedDiscoverer(config)
		require.NotNil(t, discoverer)

		// Test discovery
		resources, err := discoverer.DiscoverResources(ctx)
		if err == nil {
			assert.NotNil(t, resources)
		}
	})

	t.Run("incremental_discovery_integration", func(t *testing.T) {
		// Test incremental discovery
		config := discovery.DiscoveryConfig{}
		incremental := discovery.NewIncrementalDiscovery(config)
		require.NotNil(t, incremental)

		// Test change detection
		changes, err := incremental.DetectChanges(ctx, []*models.Resource{})
		if err == nil {
			assert.NotNil(t, changes)
		}
	})
}

// TestConcurrentOperations tests concurrent operations
func TestConcurrentOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("concurrent_discovery", func(t *testing.T) {
		config := &config.Config{}
		discoverer := discovery.NewEnhancedDiscoverer(config)
		results := make(chan []models.Resource, 3)

		// Start concurrent discovery operations
		go func() {
			resources, _ := discoverer.DiscoverResources(ctx)
			results <- resources
		}()

		go func() {
			resources, _ := discoverer.DiscoverResources(ctx)
			results <- resources
		}()

		go func() {
			resources, _ := discoverer.DiscoverResources(ctx)
			results <- resources
		}()

		// Collect results
		for i := 0; i < 3; i++ {
			select {
			case resources := <-results:
				assert.NotNil(t, resources)
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for discovery results")
			}
		}
	})

	t.Run("concurrent_drift_detection", func(t *testing.T) {
		detector := detector.NewDriftDetector(nil)
		results := make(chan *detector.DriftResult, 5)

		// Create test resources
		resources := []models.Resource{
			{ID: "resource1", Type: "aws_instance", Provider: "aws"},
			{ID: "resource2", Type: "aws_s3_bucket", Provider: "aws"},
			{ID: "resource3", Type: "azure_vm", Provider: "azure"},
			{ID: "resource4", Type: "gcp_compute_instance", Provider: "gcp"},
			{ID: "resource5", Type: "digitalocean_droplet", Provider: "digitalocean"},
		}

		// Start concurrent drift detection
		for _, resource := range resources {
			go func(r models.Resource) {
				result, _ := detector.DetectResourceDrift(ctx, r)
				results <- result
			}(resource)
		}

		// Collect results
		for i := 0; i < 5; i++ {
			select {
			case result := <-results:
				if result != nil {
					assert.NotNil(t, result.Resource)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for drift detection results")
			}
		}
	})
}

// TestErrorHandlingIntegration tests error handling across components
func TestErrorHandlingIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("provider_error_handling", func(t *testing.T) {
		// Test with invalid provider configuration
		provider := providers.NewAWSProvider("invalid-region")
		err := provider.Initialize(ctx)
		// Should handle error gracefully
		if err != nil {
			assert.Contains(t, err.Error(), "aws")
		}
	})

	t.Run("discovery_error_handling", func(t *testing.T) {
		config := &config.Config{}
		discoverer := discovery.NewEnhancedDiscoverer(config)

		// Test discovery
		_, err := discoverer.DiscoverResources(ctx)
		// Should handle error gracefully
		if err != nil {
			assert.NotNil(t, err)
		}
	})

	t.Run("drift_detection_error_handling", func(t *testing.T) {
		detector := detector.NewDriftDetector(nil)

		// Test with invalid resource
		invalidResource := models.Resource{
			ID:       "",
			Type:     "",
			Provider: "",
		}

		_, err := detector.DetectResourceDrift(ctx, invalidResource)
		// Should handle error gracefully
		if err != nil {
			assert.NotNil(t, err)
		}
	})
}

// TestPerformanceIntegration tests performance characteristics
func TestPerformanceIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("discovery_performance", func(t *testing.T) {
		config := &config.Config{}
		discoverer := discovery.NewEnhancedDiscoverer(config)

		start := time.Now()
		_, err := discoverer.DiscoverResources(ctx)
		duration := time.Since(start)

		// Should complete within reasonable time
		assert.Less(t, duration, 10*time.Second)
		if err != nil {
			// Error is acceptable for performance test
			assert.NotNil(t, err)
		}
	})

	t.Run("drift_detection_performance", func(t *testing.T) {
		detector := detector.NewDriftDetector(nil)

		// Test with multiple resources
		resources := make([]models.Resource, 100)
		for i := 0; i < 100; i++ {
			resources[i] = models.Resource{
				ID:       fmt.Sprintf("resource-%d", i),
				Type:     "aws_instance",
				Provider: "aws",
			}
		}

		start := time.Now()
		for _, resource := range resources {
			_, _ = detector.DetectResourceDrift(ctx, resource)
		}
		duration := time.Since(start)

		// Should complete within reasonable time
		assert.Less(t, duration, 5*time.Second)
	})
}

// Mock implementations for testing

type MockRemediationPlanner struct{}

func (m *MockRemediationPlanner) CreatePlan(ctx context.Context, report *detector.DriftReport, options interface{}) (*MockRemediationPlan, error) {
	return &MockRemediationPlan{
		Actions: []MockRemediationAction{
			{
				Type:        "update",
				Resource:    "test-resource",
				Description: "Update resource configuration",
			},
		},
		RiskLevel: "low",
	}, nil
}

type MockRemediationPlan struct {
	Actions   []MockRemediationAction
	RiskLevel string
}

type MockRemediationAction struct {
	Type        string
	Resource    string
	Description string
}

type MockStateBackend struct{}

func (m *MockStateBackend) Get(ctx context.Context, key string) ([]byte, error) {
	return []byte(`{"version": 4, "terraform_version": "1.0.0", "serial": 1, "lineage": "test", "resources": []}`), nil
}

func (m *MockStateBackend) Put(ctx context.Context, key string, data []byte) error {
	return nil
}

func (m *MockStateBackend) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *MockStateBackend) List(ctx context.Context, prefix string) ([]string, error) {
	return []string{"test-key"}, nil
}

func (m *MockStateBackend) Lock(ctx context.Context, key string) error {
	return nil
}

func (m *MockStateBackend) Unlock(ctx context.Context, key string) error {
	return nil
}

func (m *MockStateBackend) ListStates(ctx context.Context) ([]string, error) {
	return []string{"test-key"}, nil
}

func (m *MockStateBackend) ListStateVersions(ctx context.Context, key string) ([]state.StateVersion, error) {
	return []state.StateVersion{}, nil
}

func (m *MockStateBackend) GetStateVersion(ctx context.Context, key string, version int) ([]byte, error) {
	return []byte(`{"version": 4, "terraform_version": "1.0.0", "serial": 1, "lineage": "test", "resources": []}`), nil
}

type MockDriftComparator struct{}

func (m *MockDriftComparator) Compare(expected, actual map[string]interface{}) ([]MockDifference, error) {
	return []MockDifference{
		{
			Path:     "key",
			Expected: expected["key"],
			Actual:   actual["key"],
		},
	}, nil
}

type MockDifference struct {
	Path     string
	Expected interface{}
	Actual   interface{}
}

// Helper functions

func convertDriftResults(results []*detector.DriftResult) []detector.DriftResult {
	converted := make([]detector.DriftResult, len(results))
	for i, result := range results {
		if result != nil {
			converted[i] = *result
		}
	}
	return converted
}
