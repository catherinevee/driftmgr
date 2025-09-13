package mock

import (
	"context"
	"fmt"
	"sync"

	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/catherinevee/driftmgr/pkg/models"
)

// MockProvider is a mock implementation of CloudProvider for testing
type MockProvider struct {
	name                 string
	resources            []models.Resource
	regions              []string
	supportedTypes       []string
	discoverError        error
	getResourceError     error
	validateError        error
	listRegionsError     error
	discoverCallCount    int
	getResourceCallCount int
	validateCallCount    int
	listRegionsCallCount int
	mu                   sync.Mutex
	resourceMap          map[string]*models.Resource
	discoverDelay        bool
	returnEmptyResources bool
}

// NewMockProvider creates a new mock provider
func NewMockProvider(name string) *MockProvider {
	return &MockProvider{
		name: name,
		resources: []models.Resource{
			{
				ID:         "mock-resource-1",
				Name:       "Mock Resource 1",
				Type:       "mock.instance",
				Provider:   name,
				Region:     "us-east-1",
				Status:     "running",
				Attributes: map[string]interface{}{"cpu": 2, "memory": 4096},
			},
			{
				ID:         "mock-resource-2",
				Name:       "Mock Resource 2",
				Type:       "mock.database",
				Provider:   name,
				Region:     "us-east-1",
				Status:     "available",
				Attributes: map[string]interface{}{"engine": "postgres", "version": "13.7"},
			},
			{
				ID:         "mock-resource-3",
				Name:       "Mock Resource 3",
				Type:       "mock.storage",
				Provider:   name,
				Region:     "us-west-2",
				Status:     "active",
				Attributes: map[string]interface{}{"size": 100, "type": "ssd"},
			},
		},
		regions: []string{"us-east-1", "us-west-2", "eu-west-1"},
		supportedTypes: []string{
			"mock.instance",
			"mock.database",
			"mock.storage",
			"mock.network",
		},
		resourceMap: make(map[string]*models.Resource),
	}
}

// Name returns the provider name
func (m *MockProvider) Name() string {
	return m.name
}

// DiscoverResources discovers resources in the specified region
func (m *MockProvider) DiscoverResources(ctx context.Context, region string) ([]models.Resource, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.discoverCallCount++

	if m.discoverError != nil {
		return nil, m.discoverError
	}

	if m.returnEmptyResources {
		return []models.Resource{}, nil
	}

	// Filter resources by region
	var filteredResources []models.Resource
	for _, resource := range m.resources {
		if resource.Region == region || region == "" {
			filteredResources = append(filteredResources, resource)
		}
	}

	return filteredResources, nil
}

// GetResource retrieves a specific resource by ID
func (m *MockProvider) GetResource(ctx context.Context, resourceID string) (*models.Resource, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.getResourceCallCount++

	if m.getResourceError != nil {
		return nil, m.getResourceError
	}

	// Check resourceMap first
	if resource, ok := m.resourceMap[resourceID]; ok {
		return resource, nil
	}

	// Then check default resources
	for _, resource := range m.resources {
		if resource.ID == resourceID {
			return &resource, nil
		}
	}

	return nil, &providers.NotFoundError{
		Provider:   m.name,
		ResourceID: resourceID,
		Region:     "unknown",
	}
}

// ValidateCredentials checks if the provider credentials are valid
func (m *MockProvider) ValidateCredentials(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.validateCallCount++

	if m.validateError != nil {
		return m.validateError
	}

	return nil
}

// ListRegions returns available regions for the provider
func (m *MockProvider) ListRegions(ctx context.Context) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.listRegionsCallCount++

	if m.listRegionsError != nil {
		return nil, m.listRegionsError
	}

	return m.regions, nil
}

// SupportedResourceTypes returns the list of supported resource types
func (m *MockProvider) SupportedResourceTypes() []string {
	return m.supportedTypes
}

// SetDiscoverError sets an error to be returned by DiscoverResources
func (m *MockProvider) SetDiscoverError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.discoverError = err
}

// SetGetResourceError sets an error to be returned by GetResource
func (m *MockProvider) SetGetResourceError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getResourceError = err
}

// SetValidateError sets an error to be returned by ValidateCredentials
func (m *MockProvider) SetValidateError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.validateError = err
}

// SetListRegionsError sets an error to be returned by ListRegions
func (m *MockProvider) SetListRegionsError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listRegionsError = err
}

// SetResources sets the resources to be returned by discovery
func (m *MockProvider) SetResources(resources []models.Resource) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resources = resources
}

// AddResource adds a resource to the provider
func (m *MockProvider) AddResource(resource models.Resource) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resources = append(m.resources, resource)
	m.resourceMap[resource.ID] = &resource
}

// SetRegions sets the regions to be returned by ListRegions
func (m *MockProvider) SetRegions(regions []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.regions = regions
}

// SetSupportedTypes sets the supported resource types
func (m *MockProvider) SetSupportedTypes(types []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.supportedTypes = types
}

// GetDiscoverCallCount returns the number of times DiscoverResources was called
func (m *MockProvider) GetDiscoverCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.discoverCallCount
}

// GetValidateCallCount returns the number of times ValidateCredentials was called
func (m *MockProvider) GetValidateCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.validateCallCount
}

// ResetCallCounts resets all call counts
func (m *MockProvider) ResetCallCounts() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.discoverCallCount = 0
	m.getResourceCallCount = 0
	m.validateCallCount = 0
	m.listRegionsCallCount = 0
}

// SetReturnEmpty sets whether to return empty resources
func (m *MockProvider) SetReturnEmpty(empty bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.returnEmptyResources = empty
}

// MockProviderWithDrift creates a mock provider with drift simulation
func MockProviderWithDrift(name string) *MockProvider {
	provider := NewMockProvider(name)
	provider.SetResources([]models.Resource{
		{
			ID:       "drift-resource-1",
			Name:     "Resource with Drift",
			Type:     "mock.instance",
			Provider: name,
			Region:   "us-east-1",
			Status:   "running",
			Attributes: map[string]interface{}{
				"cpu":           4,    // Changed from 2
				"memory":        8192, // Changed from 4096
				"modified_time": "2024-01-15T10:30:00Z",
			},
		},
		{
			ID:       "drift-resource-2",
			Name:     "Deleted Resource",
			Type:     "mock.database",
			Provider: name,
			Region:   "us-east-1",
			Status:   "deleted", // Resource deleted
			Attributes: map[string]interface{}{
				"engine":  "postgres",
				"version": "14.0", // Version changed
			},
		},
	})
	return provider
}

// MockProviderFactory creates mock providers for testing
type MockProviderFactory struct {
	providers map[string]providers.CloudProvider
	mu        sync.Mutex
}

// NewMockProviderFactory creates a new mock provider factory
func NewMockProviderFactory() *MockProviderFactory {
	return &MockProviderFactory{
		providers: make(map[string]providers.CloudProvider),
	}
}

// CreateProvider creates a provider based on configuration
func (f *MockProviderFactory) CreateProvider(config providers.ProviderConfig) (providers.CloudProvider, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if provider, exists := f.providers[config.Name]; exists {
		return provider, nil
	}

	// Create new mock provider
	provider := NewMockProvider(config.Name)
	f.providers[config.Name] = provider
	return provider, nil
}

// RegisterProvider registers a provider with the factory
func (f *MockProviderFactory) RegisterProvider(name string, provider providers.CloudProvider) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.providers[name] = provider
}

// GetProvider retrieves a registered provider
func (f *MockProviderFactory) GetProvider(name string) (providers.CloudProvider, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if provider, exists := f.providers[name]; exists {
		return provider, nil
	}
	return nil, fmt.Errorf("provider %s not found", name)
}
