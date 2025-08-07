package discovery

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/catherinevee/driftmgr/internal/models"
)

// AzureProvider implements the Provider interface for Azure
type AzureProvider struct {
	cred          azcore.TokenCredential
	ctx           context.Context
	clientFactory *armresources.ClientFactory
}

// NewAzureProvider creates a new Azure provider
func NewAzureProvider() (*AzureProvider, error) {
	ctx := context.Background()

	// Use default Azure credential chain (Azure CLI, environment variables, etc.)
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	// We'll need subscription ID from config or environment
	// For now, create without client factory and initialize it later
	return &AzureProvider{
		cred: cred,
		ctx:  ctx,
	}, nil
}

// Name returns the provider name
func (p *AzureProvider) Name() string {
	return "Microsoft Azure"
}

// SupportedRegions returns the list of supported Azure regions
func (p *AzureProvider) SupportedRegions() []string {
	return []string{
		"eastus", "eastus2", "westus", "westus2", "westus3",
		"centralus", "northcentralus", "southcentralus",
		"westeurope", "northeurope", "uksouth", "ukwest",
		"francecentral", "germanywestcentral", "norwayeast",
		"switzerlandnorth", "eastasia", "southeastasia",
		"japaneast", "japanwest", "koreacentral", "koreasouth",
		"australiaeast", "australiasoutheast", "brazilsouth",
		"canadacentral", "canadaeast", "southafricanorth",
		"uaenorth", "centralindia", "southindia", "westindia",
	}
}

// SupportedResourceTypes returns the list of supported Azure resource types
func (p *AzureProvider) SupportedResourceTypes() []string {
	return []string{
		"azurerm_virtual_machine",
		"azurerm_linux_virtual_machine",
		"azurerm_windows_virtual_machine",
		"azurerm_virtual_network",
		"azurerm_subnet",
		"azurerm_network_security_group",
		"azurerm_storage_account",
		"azurerm_sql_server",
		"azurerm_sql_database",
		"azurerm_app_service",
		"azurerm_app_service_plan",
		"azurerm_resource_group",
		"azurerm_public_ip",
		"azurerm_network_interface",
	}
}

// Discover discovers Azure resources
func (p *AzureProvider) Discover(config Config) ([]models.Resource, error) {
	fmt.Println("  [Azure] Discovering resources using Azure SDK...")

	// For Azure, we need a subscription ID. In a real implementation,
	// this would come from configuration or be discovered from the credential
	subscriptionID := p.getSubscriptionID(config)
	if subscriptionID == "" {
		fmt.Println("  [Azure] Warning: No subscription ID configured, using mock data")
		return p.getMockResources(config), nil
	}

	// Initialize client factory with subscription ID
	clientFactory, err := armresources.NewClientFactory(subscriptionID, p.cred, nil)
	if err != nil {
		fmt.Printf("  [Azure] Warning: Failed to create client factory: %v, using mock data\n", err)
		return p.getMockResources(config), nil
	}
	p.clientFactory = clientFactory

	// Discover resources
	var allResources []models.Resource

	// If specific regions are requested, use them
	regions := config.Regions
	if len(regions) == 0 {
		regions = []string{"eastus"} // Default region
	}

	for _, region := range regions {
		fmt.Printf("  [Azure] Scanning region: %s\n", region)

		resources, err := p.discoverResourcesInRegion(region, config)
		if err != nil {
			fmt.Printf("  [Azure] Warning: Failed to discover resources in %s: %v\n", region, err)
			continue
		}

		allResources = append(allResources, resources...)
	}

	fmt.Printf("  [Azure] Found %d resources\n", len(allResources))
	return allResources, nil
}

// getSubscriptionID extracts subscription ID from config or environment
func (p *AzureProvider) getSubscriptionID(config Config) string {
	// In a real implementation, this would come from:
	// 1. Configuration file
	// 2. Environment variable AZURE_SUBSCRIPTION_ID
	// 3. Azure CLI default subscription
	// 4. Discovery from credential

	// For demo purposes, return empty to trigger mock data
	return ""
}

// discoverResourcesInRegion discovers all resources in a specific region
func (p *AzureProvider) discoverResourcesInRegion(region string, config Config) ([]models.Resource, error) {
	var resources []models.Resource

	// Get all resources in the subscription
	client := p.clientFactory.NewClient()
	pager := client.NewListPager(&armresources.ClientListOptions{
		Filter: nil, // Could filter by region here
	})

	for pager.More() {
		page, err := pager.NextPage(p.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get resources page: %w", err)
		}

		for _, resource := range page.Value {
			if resource.Location != nil && strings.EqualFold(*resource.Location, region) {
				azureResource := p.convertAzureResource(resource)

				// Apply resource type filter
				if config.ResourceType != "" && azureResource.Type != config.ResourceType {
					continue
				}

				resources = append(resources, azureResource)
			}
		}
	}

	return resources, nil
}

// convertAzureResource converts Azure SDK resource to internal resource model
func (p *AzureProvider) convertAzureResource(resource *armresources.GenericResource) models.Resource {
	resourceType := "unknown"
	terraformType := "unknown"

	if resource.Type != nil {
		azureType := *resource.Type
		resourceType = p.azureTypeToTerraformType(azureType)
		terraformType = resourceType
	}

	name := ""
	if resource.Name != nil {
		name = *resource.Name
	}

	location := ""
	if resource.Location != nil {
		location = *resource.Location
	}

	id := ""
	if resource.ID != nil {
		id = *resource.ID
	}

	tags := make(map[string]string)
	if resource.Tags != nil {
		for k, v := range resource.Tags {
			if v != nil {
				tags[k] = *v
			}
		}
	}

	return models.Resource{
		ID:            id,
		Name:          name,
		Type:          resourceType,
		TerraformType: terraformType,
		Provider:      "azure",
		Region:        location,
		Tags:          tags,
		ImportID:      id,
		CreatedAt:     time.Now(), // Azure doesn't provide creation time in list API
		Metadata:      make(map[string]interface{}),
	}
}

// azureTypeToTerraformType converts Azure resource type to Terraform resource type
func (p *AzureProvider) azureTypeToTerraformType(azureType string) string {
	typeMap := map[string]string{
		"Microsoft.Compute/virtualMachines":       "azurerm_virtual_machine",
		"Microsoft.Network/virtualNetworks":       "azurerm_virtual_network",
		"Microsoft.Network/networkSecurityGroups": "azurerm_network_security_group",
		"Microsoft.Storage/storageAccounts":       "azurerm_storage_account",
		"Microsoft.Sql/servers":                   "azurerm_sql_server",
		"Microsoft.Sql/servers/databases":         "azurerm_sql_database",
		"Microsoft.Web/sites":                     "azurerm_app_service",
		"Microsoft.Web/serverfarms":               "azurerm_app_service_plan",
		"Microsoft.Resources/resourceGroups":      "azurerm_resource_group",
		"Microsoft.Network/publicIPAddresses":     "azurerm_public_ip",
		"Microsoft.Network/networkInterfaces":     "azurerm_network_interface",
	}

	if terraformType, exists := typeMap[azureType]; exists {
		return terraformType
	}

	// Convert generic Azure type to terraform type
	parts := strings.Split(azureType, "/")
	if len(parts) >= 2 {
		provider := strings.ToLower(strings.Replace(parts[0], "Microsoft.", "", 1))
		resourceType := strings.ToLower(parts[1])
		return fmt.Sprintf("azurerm_%s_%s", provider, resourceType)
	}

	return "azurerm_unknown"
}

// getMockResources returns mock data when real Azure discovery fails
func (p *AzureProvider) getMockResources(config Config) []models.Resource {
	fmt.Println("  [Azure] Using mock resources for demonstration")

	resources := []models.Resource{
		{
			ID:            "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-prod/providers/Microsoft.Compute/virtualMachines/vm-web-01",
			Name:          "vm-web-01",
			Type:          "azurerm_linux_virtual_machine",
			TerraformType: "azurerm_linux_virtual_machine",
			Provider:      "azure",
			Region:        "eastus",
			Tags: map[string]string{
				"Environment": "production",
				"Team":        "platform",
				"Purpose":     "web-server",
			},
			ImportID:  "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-prod/providers/Microsoft.Compute/virtualMachines/vm-web-01",
			CreatedAt: time.Now().Add(-48 * time.Hour),
			Metadata: map[string]interface{}{
				"vm_size":        "Standard_B2s",
				"os_type":        "Linux",
				"publisher":      "Canonical",
				"offer":          "0001-com-ubuntu-server-focal",
				"sku":            "20_04-lts-gen2",
				"admin_username": "azureuser",
			},
		},
		{
			ID:            "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-prod/providers/Microsoft.Network/virtualNetworks/vnet-prod",
			Name:          "vnet-prod",
			Type:          "azurerm_virtual_network",
			TerraformType: "azurerm_virtual_network",
			Provider:      "azure",
			Region:        "eastus",
			Tags: map[string]string{
				"Environment": "production",
				"Team":        "platform",
			},
			ImportID:  "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-prod/providers/Microsoft.Network/virtualNetworks/vnet-prod",
			CreatedAt: time.Now().Add(-72 * time.Hour),
			Metadata: map[string]interface{}{
				"address_space": []string{"10.1.0.0/16"},
				"dns_servers":   []string{},
			},
		},
		{
			ID:            "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-prod/providers/Microsoft.Storage/storageAccounts/stprod001",
			Name:          "stprod001",
			Type:          "azurerm_storage_account",
			TerraformType: "azurerm_storage_account",
			Provider:      "azure",
			Region:        "eastus",
			Tags: map[string]string{
				"Environment": "production",
				"Purpose":     "data-storage",
			},
			ImportID:  "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/rg-prod/providers/Microsoft.Storage/storageAccounts/stprod001",
			CreatedAt: time.Now().Add(-120 * time.Hour),
			Metadata: map[string]interface{}{
				"account_tier":             "Standard",
				"account_replication_type": "LRS",
				"account_kind":             "StorageV2",
				"access_tier":              "Hot",
			},
		},
	}

	// Apply basic filtering
	var filtered []models.Resource
	for _, resource := range resources {
		if config.ResourceType != "" && resource.Type != config.ResourceType {
			continue
		}
		if len(config.Regions) > 0 && !contains(config.Regions, resource.Region) {
			continue
		}
		filtered = append(filtered, resource)
	}

	return filtered
}
