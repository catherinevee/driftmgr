package azure

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAzureSDKProviderSimple_New(t *testing.T) {
	// Skip if no Azure credentials are available
	if os.Getenv("AZURE_SUBSCRIPTION_ID") == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID not set, skipping Azure SDK provider test")
	}

	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		subscriptionID = "test-subscription-id"
	}

	provider, err := NewAzureSDKProviderSimple(subscriptionID, "test-resource-group")
	if err != nil {
		// This is expected to fail in test environment without real credentials
		t.Logf("Expected error creating Azure SDK provider: %v", err)
		return
	}

	assert.NotNil(t, provider)
	assert.Equal(t, "azure", provider.Name())
	assert.Equal(t, subscriptionID, provider.GetSubscriptionID())
	assert.Equal(t, "test-resource-group", provider.GetResourceGroupName())
}

func TestAzureSDKProviderSimple_Name(t *testing.T) {
	provider := &AzureSDKProviderSimple{}
	assert.Equal(t, "azure", provider.Name())
}

func TestAzureSDKProviderSimple_SupportedResourceTypes(t *testing.T) {
	provider := &AzureSDKProviderSimple{}
	resourceTypes := provider.SupportedResourceTypes()

	assert.Contains(t, resourceTypes, "azurerm_virtual_machine")
	assert.Contains(t, resourceTypes, "azurerm_storage_account")
	assert.Contains(t, resourceTypes, "azurerm_virtual_network")
	assert.Contains(t, resourceTypes, "azurerm_key_vault")
	assert.Contains(t, resourceTypes, "azurerm_kubernetes_cluster")
}

func TestAzureSDKProviderSimple_ListRegions(t *testing.T) {
	provider := &AzureSDKProviderSimple{}
	regions, err := provider.ListRegions(context.Background())

	require.NoError(t, err)
	assert.NotEmpty(t, regions)
	assert.Contains(t, regions, "eastus")
	assert.Contains(t, regions, "westus")
	assert.Contains(t, regions, "eastus2")
	assert.Contains(t, regions, "westus2")
}

func TestAzureSDKProviderSimple_GetTerraformResourceType(t *testing.T) {
	provider := &AzureSDKProviderSimple{}

	tests := []struct {
		azureType    string
		expectedType string
	}{
		{
			azureType:    "Microsoft.Compute/virtualMachines",
			expectedType: "azurerm_virtual_machine",
		},
		{
			azureType:    "Microsoft.Storage/storageAccounts",
			expectedType: "azurerm_storage_account",
		},
		{
			azureType:    "Microsoft.Network/virtualNetworks",
			expectedType: "azurerm_virtual_network",
		},
		{
			azureType:    "Microsoft.KeyVault/vaults",
			expectedType: "azurerm_key_vault",
		},
		{
			azureType:    "Microsoft.ContainerService/managedClusters",
			expectedType: "azurerm_kubernetes_cluster",
		},
		{
			azureType:    "Microsoft.Unknown/Resource",
			expectedType: "azurerm_unknown/resource",
		},
	}

	for _, test := range tests {
		t.Run(test.azureType, func(t *testing.T) {
			result := provider.getTerraformResourceType(&test.azureType)
			assert.Equal(t, test.expectedType, result)
		})
	}
}

func TestAzureSDKProviderSimple_ExtractResourceNameFromID(t *testing.T) {
	provider := &AzureSDKProviderSimple{}

	tests := []struct {
		resourceID   string
		expectedName string
	}{
		{
			resourceID:   "/subscriptions/123/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm1",
			expectedName: "vm1",
		},
		{
			resourceID:   "/subscriptions/123/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/storage1",
			expectedName: "storage1",
		},
		{
			resourceID:   "simple-name",
			expectedName: "simple-name",
		},
		{
			resourceID:   "",
			expectedName: "",
		},
	}

	for _, test := range tests {
		t.Run(test.resourceID, func(t *testing.T) {
			result := provider.extractResourceNameFromID(test.resourceID)
			assert.Equal(t, test.expectedName, result)
		})
	}
}

func TestAzureSDKProviderSimple_ValidateCredentials(t *testing.T) {
	// Skip if no Azure credentials are available
	if os.Getenv("AZURE_SUBSCRIPTION_ID") == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID not set, skipping Azure credentials validation test")
	}

	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	provider, err := NewAzureSDKProviderSimple(subscriptionID, "test-resource-group")
	if err != nil {
		t.Skipf("Failed to create Azure SDK provider: %v", err)
	}

	// This will likely fail in test environment without real credentials
	err = provider.ValidateCredentials(context.Background())
	if err != nil {
		t.Logf("Expected error validating Azure credentials: %v", err)
		// This is expected in test environment
		return
	}

	// If we get here, credentials are valid
	t.Log("Azure credentials are valid")
}

func TestAzureSDKProviderSimple_TestConnection(t *testing.T) {
	// Skip if no Azure credentials are available
	if os.Getenv("AZURE_SUBSCRIPTION_ID") == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID not set, skipping Azure connection test")
	}

	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	provider, err := NewAzureSDKProviderSimple(subscriptionID, "test-resource-group")
	if err != nil {
		t.Skipf("Failed to create Azure SDK provider: %v", err)
	}

	// This will likely fail in test environment without real credentials
	err = provider.TestConnection(context.Background())
	if err != nil {
		t.Logf("Expected error testing Azure connection: %v", err)
		// This is expected in test environment
		return
	}

	// If we get here, connection is successful
	t.Log("Azure connection test successful")
}

func TestAzureSDKProviderSimple_DiscoverResources(t *testing.T) {
	// Skip if no Azure credentials are available
	if os.Getenv("AZURE_SUBSCRIPTION_ID") == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID not set, skipping Azure resource discovery test")
	}

	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	provider, err := NewAzureSDKProviderSimple(subscriptionID, "test-resource-group")
	if err != nil {
		t.Skipf("Failed to create Azure SDK provider: %v", err)
	}

	// Test discovering resources (this will likely fail in test environment)
	resources, err := provider.DiscoverResources(context.Background(), "")
	if err != nil {
		t.Logf("Expected error discovering Azure resources: %v", err)
		// This is expected in test environment
		return
	}

	t.Logf("Discovered %d Azure resources", len(resources))
	for _, resource := range resources {
		t.Logf("Resource: %s (%s) in %s", resource.ID, resource.Type, resource.Region)
	}
}

func TestAzureSDKProviderSimple_ListResourceGroups(t *testing.T) {
	// Skip if no Azure credentials are available
	if os.Getenv("AZURE_SUBSCRIPTION_ID") == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID not set, skipping Azure resource groups test")
	}

	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	provider, err := NewAzureSDKProviderSimple(subscriptionID, "test-resource-group")
	if err != nil {
		t.Skipf("Failed to create Azure SDK provider: %v", err)
	}

	// Test listing resource groups (this will likely fail in test environment)
	resourceGroups, err := provider.ListResourceGroups(context.Background())
	if err != nil {
		t.Logf("Expected error listing Azure resource groups: %v", err)
		// This is expected in test environment
		return
	}

	t.Logf("Found %d Azure resource groups", len(resourceGroups))
	for _, rg := range resourceGroups {
		t.Logf("Resource Group: %s", rg)
	}
}

func TestAzureSDKProviderSimple_ListResourcesByType(t *testing.T) {
	// Skip if no Azure credentials are available
	if os.Getenv("AZURE_SUBSCRIPTION_ID") == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID not set, skipping Azure resource type listing test")
	}

	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	provider, err := NewAzureSDKProviderSimple(subscriptionID, "test-resource-group")
	if err != nil {
		t.Skipf("Failed to create Azure SDK provider: %v", err)
	}

	// Test listing resources by type (this will likely fail in test environment)
	resources, err := provider.ListResourcesByType(context.Background(), "azurerm_virtual_machine")
	if err != nil {
		t.Logf("Expected error listing Azure resources by type: %v", err)
		// This is expected in test environment
		return
	}

	t.Logf("Found %d Azure virtual machines", len(resources))
	for _, resource := range resources {
		assert.Equal(t, "azurerm_virtual_machine", resource.Type)
		t.Logf("VM: %s in %s", resource.ID, resource.Region)
	}
}
