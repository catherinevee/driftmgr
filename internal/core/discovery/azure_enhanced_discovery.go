package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/catherinevee/driftmgr/internal/core/models"
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
	var allResources []models.Resource
	var mu sync.Mutex

	// Start progress monitoring
	go aed.monitorProgress()

	// Discover resources using Azure CLI for comprehensive coverage
	cliResources, err := aed.discoverViaCLI(ctx)
	if err != nil {
		log.Printf("Warning: CLI discovery failed: %v", err)
	} else {
		mu.Lock()
		allResources = append(allResources, cliResources...)
		mu.Unlock()
	}

	// Discover resources using SDK for detailed information
	sdkResources, err := aed.discoverViaSDK(ctx)
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
	var resources []models.Resource

	// Check if Azure CLI is available
	if !aed.isAzureCLIAvailable() {
		return resources, fmt.Errorf("Azure CLI not available")
	}

	// Discover resource groups
	resourceGroups, err := aed.discoverResourceGroupsViaCLI(ctx)
	if err != nil {
		log.Printf("Error discovering resource groups: %v", err)
	} else {
		resources = append(resources, resourceGroups...)
	}

	// Discover virtual machines
	vms, err := aed.discoverVirtualMachinesViaCLI(ctx)
	if err != nil {
		log.Printf("Error discovering virtual machines: %v", err)
	} else {
		resources = append(resources, vms...)
	}

	// Discover storage accounts
	storageAccounts, err := aed.discoverStorageAccountsViaCLI(ctx)
	if err != nil {
		log.Printf("Error discovering storage accounts: %v", err)
	} else {
		resources = append(resources, storageAccounts...)
	}

	return resources, nil
}

// discoverViaSDK discovers resources using Azure SDK
func (aed *AzureEnhancedDiscoverer) discoverViaSDK(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	// Discover resource groups using SDK
	resourceGroups, err := aed.discoverResourceGroupsViaSDK(ctx)
	if err != nil {
		log.Printf("Error discovering resource groups via SDK: %v", err)
	} else {
		resources = append(resources, resourceGroups...)
	}

	return resources, nil
}

// discoverResourceGroupsViaCLI discovers resource groups using Azure CLI
func (aed *AzureEnhancedDiscoverer) discoverResourceGroupsViaCLI(ctx context.Context) ([]models.Resource, error) {
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
		resource := models.Resource{
			ID:       getString(group, "id"),
			Name:     getString(group, "name"),
			Type:     "azure_resource_group",
			Provider: "azure",
			Region:   getString(group, "location"),
			Tags:     convertTags(getMap(group, "tags")),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// discoverVirtualMachinesViaCLI discovers virtual machines using Azure CLI
func (aed *AzureEnhancedDiscoverer) discoverVirtualMachinesViaCLI(ctx context.Context) ([]models.Resource, error) {
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
		resource := models.Resource{
			ID:       getString(vm, "id"),
			Name:     getString(vm, "name"),
			Type:     "azure_virtual_machine",
			Provider: "azure",
			Region:   getString(vm, "location"),
			Tags:     convertTags(getMap(vm, "tags")),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// discoverStorageAccountsViaCLI discovers storage accounts using Azure CLI
func (aed *AzureEnhancedDiscoverer) discoverStorageAccountsViaCLI(ctx context.Context) ([]models.Resource, error) {
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
		resource := models.Resource{
			ID:       getString(account, "id"),
			Name:     getString(account, "name"),
			Type:     "azure_storage_account",
			Provider: "azure",
			Region:   getString(account, "location"),
			Tags:     convertTags(getMap(account, "tags")),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// discoverResourceGroupsViaSDK discovers resource groups using Azure SDK
func (aed *AzureEnhancedDiscoverer) discoverResourceGroupsViaSDK(ctx context.Context) ([]models.Resource, error) {
	var resources []models.Resource

	pager := aed.clients.Resources.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get resource groups page: %w", err)
		}

		for _, group := range page.Value {
			resource := models.Resource{
				ID:       *group.ID,
				Name:     *group.Name,
				Type:     "azure_resource_group",
				Provider: "azure",
				Region:   *group.Location,
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
