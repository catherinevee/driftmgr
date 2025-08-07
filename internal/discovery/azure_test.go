package discovery

import (
	"context"
	"testing"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewAzureProvider(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful provider creation",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewAzureProvider()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				// Should not error even if Azure credentials are not available
				assert.NotNil(t, provider)
			}
		})
	}
}

func TestAzureProvider_Discover(t *testing.T) {
	provider, err := NewAzureProvider()
	assert.NoError(t, err)
	assert.NotNil(t, provider)

	tests := []struct {
		name         string
		config       models.DiscoveryConfig
		wantErr      bool
		minResources int
	}{
		{
			name: "discover with default config",
			config: models.DiscoveryConfig{
				Provider: "azure",
				Regions:  []string{"eastus"},
			},
			wantErr:      false,
			minResources: 0,
		},
		{
			name: "discover multiple regions",
			config: models.DiscoveryConfig{
				Provider: "azure",
				Regions:  []string{"eastus", "westus2"},
			},
			wantErr:      false,
			minResources: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources, err := provider.Discover(context.Background(), tt.config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, len(resources), tt.minResources)

				// Verify resource structure
				for _, resource := range resources {
					assert.NotEmpty(t, resource.ID)
					assert.NotEmpty(t, resource.Name)
					assert.NotEmpty(t, resource.Type)
					assert.NotEmpty(t, resource.Region)
					assert.Equal(t, "azure", resource.Provider)
				}
			}
		})
	}
}

func TestAzureProvider_GetMockData(t *testing.T) {
	provider, err := NewAzureProvider()
	assert.NoError(t, err)

	config := models.DiscoveryConfig{
		Provider: "azure",
		Regions:  []string{"eastus"},
	}

	resources := provider.getMockData(config)

	assert.GreaterOrEqual(t, len(resources), 3) // Should have at least 3 mock resources

	// Verify resource types
	resourceTypes := make(map[string]bool)
	for _, resource := range resources {
		resourceTypes[resource.Type] = true
	}

	expectedTypes := []string{
		"azurerm_virtual_machine",
		"azurerm_resource_group",
		"azurerm_storage_account",
	}

	for _, expectedType := range expectedTypes {
		assert.True(t, resourceTypes[expectedType], "Expected resource type %s not found", expectedType)
	}
}

func TestAzureProvider_AzureTypeToTerraformType(t *testing.T) {
	tests := []struct {
		azureType    string
		expectedType string
	}{
		{"Microsoft.Compute/virtualMachines", "azurerm_virtual_machine"},
		{"Microsoft.Storage/storageAccounts", "azurerm_storage_account"},
		{"Microsoft.Resources/resourceGroups", "azurerm_resource_group"},
		{"Microsoft.Network/virtualNetworks", "azurerm_virtual_network"},
		{"Microsoft.Unknown/unknownType", "azurerm_unknown_type"},
	}

	provider, err := NewAzureProvider()
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.azureType, func(t *testing.T) {
			result := provider.azureTypeToTerraformType(tt.azureType)
			assert.Equal(t, tt.expectedType, result)
		})
	}
}

// Integration test for real Azure API (only runs with credentials)
func TestAzureProvider_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	provider, err := NewAzureProvider()
	assert.NoError(t, err)

	config := models.DiscoveryConfig{
		Provider: "azure",
		Regions:  []string{"eastus"},
	}

	// This test will use real Azure credentials if available, mock data otherwise
	resources, err := provider.Discover(context.Background(), config)
	assert.NoError(t, err)
	assert.NotNil(t, resources)

	t.Logf("Discovered %d Azure resources", len(resources))

	// Log resource details for manual verification
	for i, resource := range resources {
		if i < 5 { // Limit output
			t.Logf("Resource %d: %s (%s) in %s", i+1, resource.Name, resource.Type, resource.Region)
		}
	}
}
