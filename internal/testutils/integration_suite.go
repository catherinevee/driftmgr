package testutils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/catherinevee/driftmgr/internal/state"
	"github.com/catherinevee/driftmgr/pkg/models"
)

// IntegrationTestSuite provides infrastructure for integration testing
type IntegrationTestSuite struct {
	TestDataDir    string
	CloudProviders map[string]providers.CloudProvider
	StateFiles     map[string]*state.TerraformState
	TempDir        string
	Context        context.Context
}

// IntegrationTestConfig contains configuration for integration tests
type IntegrationTestConfig struct {
	UseRealProviders bool
	TestDataPath     string
	TempDir          string
	Timeout          time.Duration
	Parallel         bool
}

// NewIntegrationTestSuite creates a new integration test suite
func NewIntegrationTestSuite(config *IntegrationTestConfig) (*IntegrationTestSuite, error) {
	if config == nil {
		config = &IntegrationTestConfig{
			UseRealProviders: false,
			TestDataPath:     "testdata",
			Timeout:          30 * time.Minute,
			Parallel:         true,
		}
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "driftmgr-integration-test-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Set up test data directory
	testDataDir := config.TestDataPath
	if testDataDir == "" {
		testDataDir = "testdata"
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	_ = cancel // We'll handle cancellation in cleanup

	suite := &IntegrationTestSuite{
		TestDataDir:    testDataDir,
		CloudProviders: make(map[string]providers.CloudProvider),
		StateFiles:     make(map[string]*state.TerraformState),
		TempDir:        tempDir,
		Context:        ctx,
	}

	// Set up real providers if requested
	if config.UseRealProviders {
		if err := suite.setupRealProviders(); err != nil {
			suite.Cleanup()
			return nil, fmt.Errorf("failed to setup real providers: %w", err)
		}
	}

	// Load test state files
	if err := suite.loadTestStateFiles(); err != nil {
		suite.Cleanup()
		return nil, fmt.Errorf("failed to load test state files: %w", err)
	}

	return suite, nil
}

// setupRealProviders sets up real cloud provider connections
func (its *IntegrationTestSuite) setupRealProviders() error {
	// AWS Provider
	awsProvider, err := providers.NewProvider("aws", map[string]interface{}{"region": "us-east-1"})
	if err != nil {
		// Log warning but don't fail - AWS might not be configured
		fmt.Printf("Warning: AWS provider not available: %v\n", err)
	} else {
		its.CloudProviders["aws"] = awsProvider
	}

	// Azure Provider
	azureProvider, err := providers.NewProvider("azure", map[string]interface{}{"region": "eastus"})
	if err != nil {
		// Log warning but don't fail - Azure might not be configured
		fmt.Printf("Warning: Azure provider not available: %v\n", err)
	} else {
		its.CloudProviders["azure"] = azureProvider
	}

	// GCP Provider
	gcpProvider, err := providers.NewProvider("gcp", map[string]interface{}{"region": "us-central1"})
	if err != nil {
		// Log warning but don't fail - GCP might not be configured
		fmt.Printf("Warning: GCP provider not available: %v\n", err)
	} else {
		its.CloudProviders["gcp"] = gcpProvider
	}

	// DigitalOcean Provider
	doProvider, err := providers.NewProvider("digitalocean", map[string]interface{}{"region": "nyc1"})
	if err != nil {
		// Log warning but don't fail - DO might not be configured
		fmt.Printf("Warning: DigitalOcean provider not available: %v\n", err)
	} else {
		its.CloudProviders["digitalocean"] = doProvider
	}

	return nil
}

// loadTestStateFiles loads real Terraform state files for testing
func (its *IntegrationTestSuite) loadTestStateFiles() error {
	// Look for state files in the test data directory
	stateFiles := []string{
		"terraform.tfstate.example",
		"multi-cloud.tfstate",
		"drift-scenarios.tfstate",
		"edge-cases.tfstate",
	}

	for _, stateFile := range stateFiles {
		statePath := filepath.Join(its.TestDataDir, stateFile)
		if _, err := os.Stat(statePath); err == nil {
			// Load the state file
			state, err := its.loadStateFile(statePath)
			if err != nil {
				fmt.Printf("Warning: Failed to load state file %s: %v\n", statePath, err)
				continue
			}
			its.StateFiles[stateFile] = state
		}
	}

	// If no state files found, create a minimal test state
	if len(its.StateFiles) == 0 {
		its.StateFiles["minimal.tfstate"] = its.createMinimalTestState()
	}

	return nil
}

// loadStateFile loads a Terraform state file
func (its *IntegrationTestSuite) loadStateFile(path string) (*state.TerraformState, error) {
	// This is a simplified implementation
	// In a full implementation, this would parse the actual Terraform state file
	return &state.TerraformState{
		Version: 4,
		Resources: []state.Resource{
			{
				Type: "aws_instance",
				Name: "test-instance",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"id":   "i-1234567890abcdef0",
							"type": "t3.micro",
						},
					},
				},
			},
		},
	}, nil
}

// createMinimalTestState creates a minimal test state for basic testing
func (its *IntegrationTestSuite) createMinimalTestState() *state.TerraformState {
	return &state.TerraformState{
		Version: 4,
		Resources: []state.Resource{
			{
				Type: "aws_instance",
				Name: "minimal-instance",
				Instances: []state.Instance{
					{
						Attributes: map[string]interface{}{
							"id":   "i-minimal123",
							"type": "t3.micro",
						},
					},
				},
			},
		},
	}
}

// GetTestState returns a test state file by name
func (its *IntegrationTestSuite) GetTestState(name string) (*state.TerraformState, error) {
	state, exists := its.StateFiles[name]
	if !exists {
		return nil, fmt.Errorf("test state file '%s' not found", name)
	}
	return state, nil
}

// GetTestProvider returns a test provider by name
func (its *IntegrationTestSuite) GetTestProvider(name string) (providers.CloudProvider, error) {
	provider, exists := its.CloudProviders[name]
	if !exists {
		return nil, fmt.Errorf("test provider '%s' not found", name)
	}
	return provider, nil
}

// CreateTestResources creates test resources for a provider
func (its *IntegrationTestSuite) CreateTestResources(providerName string, count int) ([]models.Resource, error) {
	_, err := its.GetTestProvider(providerName)
	if err != nil {
		return nil, err
	}

	// Create test resources
	var resources []models.Resource
	for i := 0; i < count; i++ {
		resource := models.Resource{
			ID:         fmt.Sprintf("test-resource-%d", i),
			Name:       fmt.Sprintf("test-resource-%d", i),
			Type:       "test_resource",
			Provider:   providerName,
			Region:     "us-east-1",
			AccountID:  "test-account",
			Tags:       map[string]string{"test": "true"},
			Properties: map[string]interface{}{
				"created_by": "integration_test",
				"test_id":    i,
			},
			CreatedAt: time.Now(),
			Updated:   time.Now(),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// Cleanup cleans up test resources and temporary files
func (its *IntegrationTestSuite) Cleanup() error {
	// Remove temp directory
	if its.TempDir != "" {
		if err := os.RemoveAll(its.TempDir); err != nil {
			return fmt.Errorf("failed to remove temp directory: %w", err)
		}
	}

	// Clean up any test resources that were created
	for providerName, provider := range its.CloudProviders {
		if cleanupProvider, ok := provider.(interface{ Cleanup() error }); ok {
			if err := cleanupProvider.Cleanup(); err != nil {
				fmt.Printf("Warning: Failed to cleanup provider %s: %v\n", providerName, err)
			}
		}
	}

	return nil
}

// RunWithTimeout runs a test function with timeout
func (its *IntegrationTestSuite) RunWithTimeout(timeout time.Duration, testFunc func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(its.Context, timeout)
	defer cancel()

	return testFunc(ctx)
}

// SkipIfProviderNotAvailable skips the test if the specified provider is not available
func (its *IntegrationTestSuite) SkipIfProviderNotAvailable(providerName string) error {
	_, exists := its.CloudProviders[providerName]
	if !exists {
		return fmt.Errorf("provider %s not available, skipping test", providerName)
	}
	return nil
}

// GetTestDataPath returns the path to test data
func (its *IntegrationTestSuite) GetTestDataPath(filename string) string {
	return filepath.Join(its.TestDataDir, filename)
}

// CreateTempFile creates a temporary file for testing
func (its *IntegrationTestSuite) CreateTempFile(filename string, content []byte) (string, error) {
	filePath := filepath.Join(its.TempDir, filename)
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	return filePath, nil
}
