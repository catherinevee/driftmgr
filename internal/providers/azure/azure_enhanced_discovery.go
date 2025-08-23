package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/catherinevee/driftmgr/internal/models"
)

// AzureEnhancedDiscoverer provides comprehensive Azure resource discovery
type AzureEnhancedDiscoverer struct {
	cred           azcore.TokenCredential
	subscriptionID string
	clients        *AzureClients
	mu             sync.RWMutex
	progressChan   chan DiscoveryProgress
	errorChan      chan AzureDiscoveryError
}

// AzureClients holds all Azure service clients
type AzureClients struct {
	Resources *armresources.Client
	// Additional clients can be added as dependencies are available
}

// DiscoveryProgress represents discovery progress updates
type DiscoveryProgress struct {
	Service   string
	Progress  float64
	Resources int
	Errors    int
	Message   string
	Timestamp time.Time
}

// AzureDiscoveryError represents Azure discovery errors
type AzureDiscoveryError struct {
	Service    string
	Error      error
	ResourceID string
	Timestamp  time.Time
}

// NewAzureEnhancedDiscoverer creates a new enhanced Azure discoverer
func NewAzureEnhancedDiscoverer(subscriptionID string) (*AzureEnhancedDiscoverer, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	// Initialize Azure service clients
	client, err := armresources.NewClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create resources client: %w", err)
	}

	clients := &AzureClients{
		Resources: client,
	}

	return &AzureEnhancedDiscoverer{
		cred:           cred,
		subscriptionID: subscriptionID,
		clients:        clients,
		progressChan:   make(chan DiscoveryProgress, 100),
		errorChan:      make(chan AzureDiscoveryError, 100),
	}, nil
}

// DiscoverResources performs comprehensive Azure resource discovery
func (aed *AzureEnhancedDiscoverer) DiscoverResources(ctx context.Context) ([]models.Resource, error) {
	return aed.DiscoverResourcesForRegions(ctx, nil)
}

// DiscoverResourcesForRegions performs comprehensive Azure resource discovery for specific regions
func (aed *AzureEnhancedDiscoverer) DiscoverResourcesForRegions(ctx context.Context, regions []string) ([]models.Resource, error) {
	var allResources []models.Resource
	var mu sync.Mutex

	// Start progress monitoring
	go aed.monitorProgress()

	// Discover resources using Azure CLI for comprehensive coverage
	cliResources, err := aed.discoverViaCLIForRegions(ctx, regions)
	if err != nil {
		log.Printf("Warning: CLI discovery failed: %v", err)
	} else {
		mu.Lock()
		allResources = append(allResources, cliResources...)
		mu.Unlock()
	}

	// Discover resources using SDK for detailed information
	sdkResources, err := aed.discoverViaSDKForRegions(ctx, regions)
	if err != nil {
		log.Printf("Warning: SDK discovery failed: %v", err)
	} else {
		mu.Lock()
		allResources = append(allResources, sdkResources...)
		mu.Unlock()
	}

	return allResources, nil
}

// discoverViaCLI discovers resources using Azure CLI
func (aed *AzureEnhancedDiscoverer) discoverViaCLI(ctx context.Context) ([]models.Resource, error) {
	return aed.discoverViaCLIForRegions(ctx, nil)
}

// discoverViaCLIForRegions discovers resources using Azure CLI for specific regions
func (aed *AzureEnhancedDiscoverer) discoverViaCLIForRegions(ctx context.Context, regions []string) ([]models.Resource, error) {
	// Universal discoverer not implemented yet - use legacy discovery
	// universalDiscoverer, err := NewUniversalDiscoverer("azure")
	// if err != nil {
	//	log.Printf("Failed to create universal discoverer, falling back to legacy discovery: %v", err)
	//	return aed.discoverViaCLILegacy(ctx, regions)
	// }
	// return universalDiscoverer.DiscoverAllResources(ctx, regions)
	
	return aed.discoverViaCLILegacy(ctx, regions)
}

// discoverViaCLILegacy provides fallback to legacy discovery method
func (aed *AzureEnhancedDiscoverer) discoverViaCLILegacy(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// Check if Azure CLI is available
	if !aed.isAzureCLIAvailable() {
		return resources, fmt.Errorf("Azure CLI not available")
	}

	// Use generic resource discovery to capture all resource types
	allResources, err := aed.discoverAllResourcesViaCLIForRegions(ctx, regions)
	if err != nil {
		log.Printf("Error discovering all resources: %v", err)
	} else {
		resources = append(resources, allResources...)
	}

	return resources, nil
}

// discoverViaSDK discovers resources using Azure SDK
func (aed *AzureEnhancedDiscoverer) discoverViaSDK(ctx context.Context) ([]models.Resource, error) {
	return aed.discoverViaSDKForRegions(ctx, nil)
}

// discoverViaSDKForRegions discovers resources using Azure SDK for specific regions
func (aed *AzureEnhancedDiscoverer) discoverViaSDKForRegions(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	// Discover resource groups using SDK
	resourceGroups, err := aed.discoverResourceGroupsViaSDKForRegions(ctx, regions)
	if err != nil {
		log.Printf("Error discovering resource groups via SDK: %v", err)
	} else {
		resources = append(resources, resourceGroups...)
	}

	return resources, nil
}

// discoverAllResourcesViaCLIForRegions discovers all resources using Azure CLI for specific regions
func (aed *AzureEnhancedDiscoverer) discoverAllResourcesViaCLIForRegions(ctx context.Context, regions []string) ([]models.Resource, error) {
	cmd := exec.CommandContext(ctx, "az", "resource", "list", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list all resources: %w", err)
	}

	var azResources []map[string]interface{}
	if err := json.Unmarshal(output, &azResources); err != nil {
		return nil, fmt.Errorf("failed to parse all resources: %w", err)
	}

	var resources []models.Resource
	for _, azResource := range azResources {
		location := getString(azResource, "location")

		// Filter by region if regions are specified
		if len(regions) > 0 {
			regionFound := false
			for _, region := range regions {
				if location == region {
					regionFound = true
					break
				}
			}
			if !regionFound {
				continue
			}
		}

		// Convert Azure resource type to driftmgr format
		azureType := getString(azResource, "type")
		driftmgrType := convertAzureTypeToDriftmgrType(azureType)

		resource := models.Resource{
			ID:       getString(azResource, "id"),
			Name:     getString(azResource, "name"),
			Type:     driftmgrType,
			Provider: "azure",
			Region:   location,
			Tags:     convertTags(getMap(azResource, "tags")),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// convertAzureTypeToDriftmgrType converts Azure resource types to driftmgr format
func convertAzureTypeToDriftmgrType(azureType string) string {
	// Convert Azure resource types to consistent driftmgr naming
	switch azureType {
	case "Microsoft.Compute/virtualMachines":
		return "azure_virtual_machine"
	case "Microsoft.Storage/storageAccounts":
		return "azure_storage_account"
	case "Microsoft.Resources/resourceGroups":
		return "azure_resource_group"
	case "Microsoft.ManagedIdentity/userAssignedIdentities":
		return "azure_managed_identity"
	case "Microsoft.Network/networkWatchers":
		return "azure_network_watcher"
	default:
		// For other types, convert to lowercase and replace separators
		return "azure_" + strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(azureType, "Microsoft.", ""), "/", "_"))
	}
}

// discoverResourceGroupsViaCLI discovers resource groups using Azure CLI
func (aed *AzureEnhancedDiscoverer) discoverResourceGroupsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return aed.discoverResourceGroupsViaCLIForRegions(ctx, nil)
}

// discoverResourceGroupsViaCLIForRegions discovers resource groups using Azure CLI for specific regions
func (aed *AzureEnhancedDiscoverer) discoverResourceGroupsViaCLIForRegions(ctx context.Context, regions []string) ([]models.Resource, error) {
	cmd := exec.CommandContext(ctx, "az", "group", "list", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list resource groups: %w", err)
	}

	var groups []map[string]interface{}
	if err := json.Unmarshal(output, &groups); err != nil {
		return nil, fmt.Errorf("failed to parse resource groups: %w", err)
	}

	var resources []models.Resource
	for _, group := range groups {
		location := getString(group, "location")

		// Filter by region if regions are specified
		if len(regions) > 0 {
			regionFound := false
			for _, region := range regions {
				if location == region {
					regionFound = true
					break
				}
			}
			if !regionFound {
				continue
			}
		}

		resource := models.Resource{
			ID:       getString(group, "id"),
			Name:     getString(group, "name"),
			Type:     "azure_resource_group",
			Provider: "azure",
			Region:   location,
			Tags:     convertTags(getMap(group, "tags")),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// discoverVirtualMachinesViaCLI discovers virtual machines using Azure CLI
func (aed *AzureEnhancedDiscoverer) discoverVirtualMachinesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return aed.discoverVirtualMachinesViaCLIForRegions(ctx, nil)
}

// discoverVirtualMachinesViaCLIForRegions discovers virtual machines using Azure CLI for specific regions
func (aed *AzureEnhancedDiscoverer) discoverVirtualMachinesViaCLIForRegions(ctx context.Context, regions []string) ([]models.Resource, error) {
	cmd := exec.CommandContext(ctx, "az", "vm", "list", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list virtual machines: %w", err)
	}

	var vms []map[string]interface{}
	if err := json.Unmarshal(output, &vms); err != nil {
		return nil, fmt.Errorf("failed to parse virtual machines: %w", err)
	}

	var resources []models.Resource
	for _, vm := range vms {
		location := getString(vm, "location")

		// Filter by region if regions are specified
		if len(regions) > 0 {
			regionFound := false
			for _, region := range regions {
				if location == region {
					regionFound = true
					break
				}
			}
			if !regionFound {
				continue
			}
		}

		resource := models.Resource{
			ID:       getString(vm, "id"),
			Name:     getString(vm, "name"),
			Type:     "azure_virtual_machine",
			Provider: "azure",
			Region:   location,
			Tags:     convertTags(getMap(vm, "tags")),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// discoverStorageAccountsViaCLI discovers storage accounts using Azure CLI
func (aed *AzureEnhancedDiscoverer) discoverStorageAccountsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return aed.discoverStorageAccountsViaCLIForRegions(ctx, nil)
}

// discoverStorageAccountsViaCLIForRegions discovers storage accounts using Azure CLI for specific regions
func (aed *AzureEnhancedDiscoverer) discoverStorageAccountsViaCLIForRegions(ctx context.Context, regions []string) ([]models.Resource, error) {
	cmd := exec.CommandContext(ctx, "az", "storage", "account", "list", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list storage accounts: %w", err)
	}

	var accounts []map[string]interface{}
	if err := json.Unmarshal(output, &accounts); err != nil {
		return nil, fmt.Errorf("failed to parse storage accounts: %w", err)
	}

	var resources []models.Resource
	for _, account := range accounts {
		location := getString(account, "location")

		// Filter by region if regions are specified
		if len(regions) > 0 {
			regionFound := false
			for _, region := range regions {
				if location == region {
					regionFound = true
					break
				}
			}
			if !regionFound {
				continue
			}
		}

		resource := models.Resource{
			ID:       getString(account, "id"),
			Name:     getString(account, "name"),
			Type:     "azure_storage_account",
			Provider: "azure",
			Region:   location,
			Tags:     convertTags(getMap(account, "tags")),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// discoverResourceGroupsViaSDK discovers resource groups using Azure SDK
func (aed *AzureEnhancedDiscoverer) discoverResourceGroupsViaSDK(ctx context.Context) ([]models.Resource, error) {
	return aed.discoverResourceGroupsViaSDKForRegions(ctx, nil)
}

// discoverResourceGroupsViaSDKForRegions discovers resource groups using Azure SDK for specific regions
func (aed *AzureEnhancedDiscoverer) discoverResourceGroupsViaSDKForRegions(ctx context.Context, regions []string) ([]models.Resource, error) {
	var resources []models.Resource

	pager := aed.clients.Resources.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get next page: %w", err)
		}

		for _, group := range page.Value {
			if group == nil || group.Location == nil {
				continue
			}

			location := *group.Location

			// Filter by region if regions are specified
			if len(regions) > 0 {
				regionFound := false
				for _, region := range regions {
					if location == region {
						regionFound = true
						break
					}
				}
				if !regionFound {
					continue
				}
			}

			resource := models.Resource{
				ID:       *group.ID,
				Name:     *group.Name,
				Type:     "azure_resource_group",
				Provider: "azure",
				Region:   location,
				Tags:     convertSDKTags(group.Tags),
			}
			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// isAzureCLIAvailable checks if Azure CLI is available
func (aed *AzureEnhancedDiscoverer) isAzureCLIAvailable() bool {
	cmd := exec.Command("az", "--version")
	return cmd.Run() == nil
}

// monitorProgress monitors discovery progress
func (aed *AzureEnhancedDiscoverer) monitorProgress() {
	for {
		select {
		case progress := <-aed.progressChan:
			log.Printf("Discovery Progress - %s: %.1f%% (%d resources, %d errors)",
				progress.Service, progress.Progress, progress.Resources, progress.Errors)
		case err := <-aed.errorChan:
			log.Printf("Discovery Error - %s: %v", err.Service, err.Error)
		}
	}
}

// Helper functions
func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getMap(data map[string]interface{}, key string) map[string]interface{} {
	if val, ok := data[key]; ok {
		if m, ok := val.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

func convertTags(tags map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range tags {
		if str, ok := v.(string); ok {
			result[k] = str
		}
	}
	return result
}

func convertSDKTags(tags map[string]*string) map[string]string {
	result := make(map[string]string)
	for k, v := range tags {
		if v != nil {
			result[k] = *v
		}
	}
	return result
}
