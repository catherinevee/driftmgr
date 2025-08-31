package discovery

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/catherinevee/driftmgr/internal/core/models"
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

// Regions returns the list of available Azure regions
func (p *AzureProvider) Regions() []string {
	return p.SupportedRegions()
}

// Services returns the list of available Azure services
func (p *AzureProvider) Services() []string {
	return []string{
		"Virtual Machines", "Storage", "SQL Database", "App Service",
		"Functions", "AKS", "Virtual Network", "Load Balancer",
		"Application Gateway", "Traffic Manager", "CDN", "Event Hubs",
	}
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
func (p *AzureProvider) Discover(ctx context.Context, options DiscoveryOptions) (*Result, error) {
	fmt.Println("  [Azure] Discovering resources using Azure SDK...")

	// Convert options to config for backward compatibility
	config := Config{
		Regions: options.Regions,
	}
	if len(options.ResourceTypes) > 0 {
		config.ResourceType = options.ResourceTypes[0]
	}

	// For Azure, we need a subscription ID
	subscriptionID := p.getSubscriptionID(config)
	if subscriptionID == "" {
		return nil, fmt.Errorf("no Azure subscription ID found. Please set AZURE_SUBSCRIPTION_ID or run 'az login'")
	}

	// Initialize client factory with subscription ID
	clientFactory, err := armresources.NewClientFactory(subscriptionID, p.cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure client factory: %w", err)
	}
	p.clientFactory = clientFactory

	// Discover resources
	var allResources []models.Resource

	// If specific regions are requested, use them; otherwise scan all common regions
	regions := config.Regions
	if len(regions) == 0 {
		// Scan common regions where resources are likely to exist
		regions = []string{
			"eastus", "eastus2", "westus", "westus2", "centralus",
			"westeurope", "northeurope", "uksouth",
			"southeastasia", "australiaeast",
		}
		fmt.Printf("  [Azure] Scanning %d common regions for resources\n", len(regions))
	}

	totalRegions := len(regions)
	successfulRegions := 0
	failedRegions := []string{}

	for i, region := range regions {
		fmt.Printf("  [Azure] Scanning region %d/%d: %s\n", i+1, totalRegions, region)

		resources, err := p.discoverResourcesInRegion(region, config)
		if err != nil {
			fmt.Printf("  [Azure] Warning: Failed to discover resources in %s: %v\n", region, err)
			failedRegions = append(failedRegions, region)
			continue
		}

		if len(resources) > 0 {
			fmt.Printf("  [Azure]   Found %d resources in %s\n", len(resources), region)
			successfulRegions++
			allResources = append(allResources, resources...)
		}
	}

	fmt.Printf("  [Azure] Successfully scanned %d/%d regions\n", successfulRegions, totalRegions)
	if len(failedRegions) > 0 && len(failedRegions) <= 5 {
		fmt.Printf("  [Azure] Failed regions: %v\n", failedRegions)
	}

	fmt.Printf("  [Azure] Found %d resources\n", len(allResources))
	return &Result{
		Resources: allResources,
		Metadata: map[string]interface{}{
			"provider":       "azure",
			"resource_count": len(allResources),
			"regions":        regions,
		},
	}, nil
}

// GetAccountInfo returns Azure account information
func (p *AzureProvider) GetAccountInfo(ctx context.Context) (*AccountInfo, error) {
	// Get subscription ID
	subID := p.getSubscriptionID(Config{})
	if subID == "" {
		return nil, fmt.Errorf("no Azure subscription found")
	}

	return &AccountInfo{
		ID:       subID,
		Name:     "Azure Subscription",
		Type:     "azure",
		Provider: "azure",
		Regions:  p.SupportedRegions(),
	}, nil
}

// getSubscriptionID extracts subscription ID from config or environment
func (p *AzureProvider) getSubscriptionID(config Config) string {
	// Try environment variable first
	if subID := os.Getenv("AZURE_SUBSCRIPTION_ID"); subID != "" {
		return subID
	}

	// Try Azure CLI default subscription
	cmd := exec.Command("az", "account", "show", "--query", "id", "-o", "tsv")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return strings.TrimSpace(string(output))
	}

	// Return empty if no subscription found
	return ""
}

// discoverResourcesInRegion discovers all resources in a specific region
func (p *AzureProvider) discoverResourcesInRegion(region string, config Config) ([]models.Resource, error) {
	var resources []models.Resource

	// Get all resources in the subscription
	client := p.clientFactory.NewClient()
	
	// Use filter to get resources in specific region more efficiently
	filterStr := fmt.Sprintf("location eq '%s'", region)
	pager := client.NewListPager(&armresources.ClientListOptions{
		Filter: &filterStr,
		Expand: nil,
		Top:    nil,
	})

	for pager.More() {
		page, err := pager.NextPage(p.ctx)
		if err != nil {
			// Try without filter if filter fails
			pager = client.NewListPager(&armresources.ClientListOptions{})
			for pager.More() {
				page, err = pager.NextPage(p.ctx)
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

		for _, resource := range page.Value {
			// Double check location matches (case insensitive)
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
func (p *AzureProvider) convertAzureResource(resource *armresources.GenericResourceExpanded) models.Resource {
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
		ID:        id,
		Name:      name,
		Type:      resourceType,
		Provider:  "azure",
		Region:    location,
		Tags:      tags,
		CreatedAt: time.Now(), // Azure doesn't provide creation time in list API
		Metadata: map[string]string{
			"terraform_type": terraformType,
			"import_id":      id,
		},
		Attributes: map[string]interface{}{},
	}
}

// azureTypeToTerraformType converts Azure resource type to Terraform resource type
func (p *AzureProvider) azureTypeToTerraformType(azureType string) string {
	typeMap := map[string]string{
		"Microsoft.Compute/virtualMachines":          "azurerm_virtual_machine",
		"Microsoft.Network/virtualNetworks":          "azurerm_virtual_network",
		"Microsoft.Network/networkSecurityGroups":    "azurerm_network_security_group",
		"Microsoft.Network/networkWatchers":          "azurerm_network_watcher",
		"Microsoft.Network/networkWatchers/flowLogs": "azurerm_network_watcher_flow_log",
		"Microsoft.Storage/storageAccounts":          "azurerm_storage_account",
		"Microsoft.Sql/servers":                      "azurerm_sql_server",
		"Microsoft.Sql/servers/databases":            "azurerm_sql_database",
		"Microsoft.Web/sites":                        "azurerm_app_service",
		"Microsoft.Web/serverfarms":                  "azurerm_app_service_plan",
		"Microsoft.Resources/resourceGroups":         "azurerm_resource_group",
		"Microsoft.Network/publicIPAddresses":        "azurerm_public_ip",
		"Microsoft.Network/networkInterfaces":        "azurerm_network_interface",
		"Microsoft.Network/loadBalancers":            "azurerm_lb",
		"Microsoft.Network/applicationGateways":      "azurerm_application_gateway",
		"Microsoft.ContainerService/managedClusters": "azurerm_kubernetes_cluster",
		"Microsoft.KeyVault/vaults":                  "azurerm_key_vault",
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


// ValidateCredentials validates Azure credentials
func (p *AzureProvider) ValidateCredentials(ctx context.Context) error {
	// Check if we have a subscription ID
	subID := p.getSubscriptionID(Config{})
	if subID == "" {
		return fmt.Errorf("no Azure subscription configured")
	}
	
	// Try to create a client factory to validate credentials
	_, err := armresources.NewClientFactory(subID, p.cred, nil)
	if err != nil {
		return fmt.Errorf("failed to validate Azure credentials: %w", err)
	}
	
	return nil
}

// DiscoverRegion discovers resources in a specific region
func (p *AzureProvider) DiscoverRegion(ctx context.Context, region string) ([]models.Resource, error) {
	options := DiscoveryOptions{
		Regions: []string{region},
	}
	result, err := p.Discover(ctx, options)
	if err != nil {
		return nil, err
	}
	return result.Resources, nil
}
