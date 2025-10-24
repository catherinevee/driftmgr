package azure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/catherinevee/driftmgr/pkg/models"
)

// AzureSDKProviderSimple implements CloudProvider for Azure using Azure SDK
type AzureSDKProviderSimple struct {
	subscriptionID    string
	credential        *azidentity.DefaultAzureCredential
	resourceClient    *armresources.Client
	resourceGroupName string
}

// NewAzureSDKProviderSimple creates a new Azure provider using Azure SDK
func NewAzureSDKProviderSimple(subscriptionID, resourceGroupName string) (*AzureSDKProviderSimple, error) {
	// Create credential
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	// Create resource client
	resourceClient, err := armresources.NewClient(subscriptionID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure resource client: %w", err)
	}

	return &AzureSDKProviderSimple{
		subscriptionID:    subscriptionID,
		credential:        credential,
		resourceClient:    resourceClient,
		resourceGroupName: resourceGroupName,
	}, nil
}

// Name returns the provider name
func (p *AzureSDKProviderSimple) Name() string {
	return "azure"
}

// DiscoverResources discovers resources in the specified region
func (p *AzureSDKProviderSimple) DiscoverResources(ctx context.Context, region string) ([]models.Resource, error) {
	// If region is specified, treat it as a resource group
	if region != "" {
		return p.discoverResourcesByResourceGroup(ctx, region)
	}

	// Otherwise, discover all resources in the subscription
	return p.discoverResourcesBySubscription(ctx)
}

// discoverResourcesByResourceGroup discovers resources in a specific resource group
func (p *AzureSDKProviderSimple) discoverResourcesByResourceGroup(ctx context.Context, resourceGroupName string) ([]models.Resource, error) {
	var resources []models.Resource

	// List all resources in the resource group
	pager := p.resourceClient.NewListByResourceGroupPager(resourceGroupName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list resources in resource group %s: %w", resourceGroupName, err)
		}

		for _, resource := range page.Value {
			azureResource := p.convertAzureResourceToModel(resource)
			resources = append(resources, azureResource)
		}
	}

	return resources, nil
}

// discoverResourcesBySubscription discovers all resources in the subscription
func (p *AzureSDKProviderSimple) discoverResourcesBySubscription(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	// List all resources in the subscription
	pager := p.resourceClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list resources in subscription: %w", err)
		}

		for _, resource := range page.Value {
			azureResource := p.convertAzureResourceToModel(resource)
			resources = append(resources, azureResource)
		}
	}

	return resources, nil
}

// convertAzureResourceToModel converts Azure resource to our model
func (p *AzureSDKProviderSimple) convertAzureResourceToModel(azureResource *armresources.GenericResourceExpanded) models.Resource {
	// Extract resource type from the full resource type
	resourceType := p.getTerraformResourceType(azureResource.Type)

	// Extract resource name from ID
	resourceName := p.extractResourceNameFromID(*azureResource.ID)

	// Convert tags
	tags := make(map[string]string)
	if azureResource.Tags != nil {
		for k, v := range azureResource.Tags {
			if v != nil {
				tags[k] = *v
			}
		}
	}

	// Create attributes map
	attributes := make(map[string]interface{})
	attributes["name"] = resourceName
	attributes["location"] = azureResource.Location
	attributes["tags"] = tags
	attributes["resource_type"] = *azureResource.Type
	attributes["resource_id"] = *azureResource.ID

	// Add provider-specific attributes
	if azureResource.Properties != nil {
		attributes["properties"] = azureResource.Properties
	}

	return models.Resource{
		ID:           resourceName,
		Type:         resourceType,
		Provider:     "azure",
		Region:       *azureResource.Location,
		Attributes:   attributes,
		Tags:         tags,
		CreatedAt:    time.Now(), // Use current time as fallback
		LastModified: time.Now(),
	}
}

// getTerraformResourceType converts Azure resource type to Terraform resource type
func (p *AzureSDKProviderSimple) getTerraformResourceType(azureType *string) string {
	if azureType == nil {
		return "unknown"
	}

	resourceType := *azureType
	switch {
	case strings.Contains(resourceType, "Microsoft.Compute/virtualMachines"):
		return "azurerm_virtual_machine"
	case strings.Contains(resourceType, "Microsoft.Network/virtualNetworks"):
		return "azurerm_virtual_network"
	case strings.Contains(resourceType, "Microsoft.Network/subnets"):
		return "azurerm_subnet"
	case strings.Contains(resourceType, "Microsoft.Network/networkSecurityGroups"):
		return "azurerm_network_security_group"
	case strings.Contains(resourceType, "Microsoft.Storage/storageAccounts"):
		return "azurerm_storage_account"
	case strings.Contains(resourceType, "Microsoft.Sql/servers"):
		return "azurerm_sql_server"
	case strings.Contains(resourceType, "Microsoft.Sql/servers/databases"):
		return "azurerm_sql_database"
	case strings.Contains(resourceType, "Microsoft.KeyVault/vaults"):
		return "azurerm_key_vault"
	case strings.Contains(resourceType, "Microsoft.Web/sites"):
		return "azurerm_app_service"
	case strings.Contains(resourceType, "Microsoft.ContainerRegistry/registries"):
		return "azurerm_container_registry"
	case strings.Contains(resourceType, "Microsoft.ContainerService/managedClusters"):
		return "azurerm_kubernetes_cluster"
	default:
		return "azurerm_" + strings.ToLower(strings.ReplaceAll(resourceType, "Microsoft.", ""))
	}
}

// extractResourceNameFromID extracts resource name from Azure resource ID
func (p *AzureSDKProviderSimple) extractResourceNameFromID(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return resourceID
}

// GetResource retrieves a specific resource by ID
func (p *AzureSDKProviderSimple) GetResource(ctx context.Context, resourceID string) (*models.Resource, error) {
	// Get the resource using Azure SDK
	resource, err := p.resourceClient.GetByID(ctx, resourceID, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource %s: %w", resourceID, err)
	}

	// Convert the resource to our model format
	// Create a GenericResourceExpanded from the response
	expandedResource := &armresources.GenericResourceExpanded{
		ID:         resource.ID,
		Name:       resource.Name,
		Type:       resource.Type,
		Location:   resource.Location,
		Tags:       resource.Tags,
		Properties: resource.Properties,
	}

	azureResource := p.convertAzureResourceToModel(expandedResource)
	return &azureResource, nil
}

// ValidateCredentials checks if the provider credentials are valid
func (p *AzureSDKProviderSimple) ValidateCredentials(ctx context.Context) error {
	// Test connection by listing resource groups
	pager := p.resourceClient.NewListPager(nil)
	_, err := pager.NextPage(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate Azure credentials: %w", err)
	}
	return nil
}

// ListRegions returns available regions for the provider
func (p *AzureSDKProviderSimple) ListRegions(ctx context.Context) ([]string, error) {
	// Return common Azure regions
	return []string{
		"eastus", "eastus2", "westus", "westus2", "centralus",
		"northeurope", "westeurope", "uksouth", "ukwest",
		"eastasia", "southeastasia", "japaneast", "japanwest",
		"australiaeast", "australiasoutheast", "canadacentral", "canadaeast",
		"brazilsouth", "southafricanorth", "southafricawest",
		"centralindia", "southindia", "westindia",
		"koreacentral", "koreasouth", "japaneast", "japanwest",
	}, nil
}

// SupportedResourceTypes returns the list of supported resource types
func (p *AzureSDKProviderSimple) SupportedResourceTypes() []string {
	return []string{
		"azurerm_virtual_machine",
		"azurerm_virtual_network",
		"azurerm_subnet",
		"azurerm_network_security_group",
		"azurerm_storage_account",
		"azurerm_sql_server",
		"azurerm_sql_database",
		"azurerm_key_vault",
		"azurerm_app_service",
		"azurerm_container_registry",
		"azurerm_kubernetes_cluster",
		"azurerm_resource_group",
		"azurerm_public_ip",
		"azurerm_network_interface",
		"azurerm_load_balancer",
		"azurerm_application_gateway",
		"azurerm_cosmosdb_account",
		"azurerm_redis_cache",
		"azurerm_service_bus_namespace",
		"azurerm_event_hub_namespace",
	}
}

// TestConnection tests the connection to Azure
func (p *AzureSDKProviderSimple) TestConnection(ctx context.Context) error {
	return p.ValidateCredentials(ctx)
}

// GetSubscriptionID returns the subscription ID
func (p *AzureSDKProviderSimple) GetSubscriptionID() string {
	return p.subscriptionID
}

// GetResourceGroupName returns the resource group name
func (p *AzureSDKProviderSimple) GetResourceGroupName() string {
	return p.resourceGroupName
}

// ListResourceGroups lists all resource groups in the subscription
func (p *AzureSDKProviderSimple) ListResourceGroups(ctx context.Context) ([]string, error) {
	var resourceGroups []string

	pager := p.resourceClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list resource groups: %w", err)
		}

		for _, resource := range page.Value {
			if resource.Name != nil {
				resourceGroups = append(resourceGroups, *resource.Name)
			}
		}
	}

	return resourceGroups, nil
}

// GetResourceByType retrieves a specific resource by type and name
func (p *AzureSDKProviderSimple) GetResourceByType(ctx context.Context, resourceType, resourceName string) (*models.Resource, error) {
	// Construct resource ID based on type and name
	var resourceID string

	switch resourceType {
	case "azurerm_virtual_machine":
		resourceID = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s",
			p.subscriptionID, p.resourceGroupName, resourceName)
	case "azurerm_storage_account":
		resourceID = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s",
			p.subscriptionID, p.resourceGroupName, resourceName)
	case "azurerm_virtual_network":
		resourceID = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s",
			p.subscriptionID, p.resourceGroupName, resourceName)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	return p.GetResource(ctx, resourceID)
}

// ListResourcesByType lists all resources of a specific type
func (p *AzureSDKProviderSimple) ListResourcesByType(ctx context.Context, resourceType string) ([]models.Resource, error) {
	var resources []models.Resource

	// Get all resources and filter by type
	allResources, err := p.DiscoverResources(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to discover resources: %w", err)
	}

	for _, resource := range allResources {
		if resource.Type == resourceType {
			resources = append(resources, resource)
		}
	}

	return resources, nil
}
