package testprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	
	"github.com/catherinevee/driftmgr/pkg/models"
)

// TestProvider is a provider implementation for testing
// It uses real data structures but loads from JSON fixtures
type TestProvider struct {
	name         string
	fixturesPath string
	resources    []models.Resource
}

// NewTestProvider creates a new test provider
func NewTestProvider(fixturesPath string) *TestProvider {
	return &TestProvider{
		name:         "test",
		fixturesPath: fixturesPath,
		resources:    []models.Resource{},
	}
}

// NewTestProviderWithData creates a test provider with predefined resources
func NewTestProviderWithData(resources []models.Resource) *TestProvider {
	return &TestProvider{
		name:      "test",
		resources: resources,
	}
}

// Name returns the provider name
func (p *TestProvider) Name() string {
	return p.name
}

// DiscoverResources returns test resources
func (p *TestProvider) DiscoverResources(ctx context.Context, region string) ([]models.Resource, error) {
	// If we have predefined resources, return them
	if len(p.resources) > 0 {
		// Filter by region if needed
		var filtered []models.Resource
		for _, r := range p.resources {
			if r.Region == region || region == "" {
				filtered = append(filtered, r)
			}
		}
		return filtered, nil
	}
	
	// Try to load from fixture file
	if p.fixturesPath != "" {
		fixturePath := filepath.Join(p.fixturesPath, fmt.Sprintf("%s.json", region))
		data, err := ioutil.ReadFile(fixturePath)
		if err != nil {
			// Return empty list if no fixture
			return []models.Resource{}, nil
		}
		
		var resources []models.Resource
		if err := json.Unmarshal(data, &resources); err != nil {
			return nil, fmt.Errorf("invalid fixture data: %w", err)
		}
		
		return resources, nil
	}
	
	// Return default test data if no fixtures
	return p.getDefaultTestResources(region), nil
}

// GetResource retrieves a specific resource
func (p *TestProvider) GetResource(ctx context.Context, resourceID string) (*models.Resource, error) {
	// Search in resources
	for _, r := range p.resources {
		if r.ID == resourceID {
			return &r, nil
		}
	}
	
	// Try to find in all regions
	if p.fixturesPath != "" {
		regions := []string{"us-east-1", "us-west-2", "eu-west-1"}
		for _, region := range regions {
			resources, _ := p.DiscoverResources(ctx, region)
			for _, r := range resources {
				if r.ID == resourceID {
					return &r, nil
				}
			}
		}
	}
	
	return nil, fmt.Errorf("resource not found: %s", resourceID)
}

// ValidateCredentials always returns success for test provider
func (p *TestProvider) ValidateCredentials(ctx context.Context) error {
	// Test provider always has valid "credentials"
	return nil
}

// ListRegions returns test regions
func (p *TestProvider) ListRegions(ctx context.Context) ([]string, error) {
	return []string{"us-east-1", "us-west-2", "eu-west-1"}, nil
}

// SupportedResourceTypes returns supported resource types
func (p *TestProvider) SupportedResourceTypes() []string {
	return []string{"instance", "bucket", "database", "network"}
}

// AddResource adds a resource to the test provider
func (p *TestProvider) AddResource(resource models.Resource) {
	p.resources = append(p.resources, resource)
}

// ClearResources clears all resources
func (p *TestProvider) ClearResources() {
	p.resources = []models.Resource{}
}

// getDefaultTestResources returns default test resources
func (p *TestProvider) getDefaultTestResources(region string) []models.Resource {
	return []models.Resource{
		{
			ID:       "test-instance-1",
			Type:     "instance",
			Provider: "test",
			Region:   region,
			Name:     "Test Instance 1",
			Properties: map[string]interface{}{
				"instance_type": "t2.micro",
				"state":        "running",
				"vpc_id":       "vpc-test123",
			},
		},
		{
			ID:       "test-bucket-1",
			Type:     "bucket",
			Provider: "test",
			Region:   region,
			Name:     "test-bucket-1",
			Properties: map[string]interface{}{
				"versioning": true,
				"encryption": "AES256",
			},
		},
		{
			ID:       "test-db-1",
			Type:     "database",
			Provider: "test",
			Region:   region,
			Name:     "test-database",
			Properties: map[string]interface{}{
				"engine":         "postgres",
				"engine_version": "13.7",
				"instance_class": "db.t3.micro",
			},
		},
	}
}