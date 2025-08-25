package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/catherinevee/driftmgr/internal/core/models"
)

// Config represents Azure discovery configuration
type Config struct {
	Regions      []string
	ResourceType string
}

// AzureDiscoverer provides comprehensive Azure resource discovery with all methods consolidated
type AzureDiscoverer struct {
	cred           azcore.TokenCredential
	subscriptionID string
	cliPath        string
	accountInfo    *AzureAccountInfo
	clients        *AzureClients
	mu             sync.RWMutex
	progressChan   chan DiscoveryProgress
	errorChan      chan AzureDiscoveryError
}

// AzureClients holds all Azure service clients
type AzureClients struct {
	Resources *armresources.Client
	Compute   *armcompute.VirtualMachinesClient
	Network   *armnetwork.VirtualNetworksClient
	Storage   *armstorage.AccountsClient
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

// AzureAccountInfo represents Azure account information
type AzureAccountInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	State     string `json:"state"`
	IsDefault bool   `json:"isDefault"`
}

// NewAzureDiscoverer creates a comprehensive Azure discoverer with all capabilities
func NewAzureDiscoverer(subscriptionID string) (*AzureDiscoverer, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	// Initialize Azure service clients
	resourceClient, err := armresources.NewClient(subscriptionID, cred, nil)
	if err != nil {
		log.Printf("Warning: Failed to create resources client: %v", err)
	}

	computeClient, err := armcompute.NewVirtualMachinesClient(subscriptionID, cred, nil)
	if err != nil {
		log.Printf("Warning: Failed to create compute client: %v", err)
	}

	networkClient, err := armnetwork.NewVirtualNetworksClient(subscriptionID, cred, nil)
	if err != nil {
		log.Printf("Warning: Failed to create network client: %v", err)
	}

	storageClient, err := armstorage.NewAccountsClient(subscriptionID, cred, nil)
	if err != nil {
		log.Printf("Warning: Failed to create storage client: %v", err)
	}

	clients := &AzureClients{
		Resources: resourceClient,
		Compute:   computeClient,
		Network:   networkClient,
		Storage:   storageClient,
	}

	// Get Azure CLI path and account info
	cliPath := getAzureCLIPath()
	var accountInfo *AzureAccountInfo
	if cliPath != "" {
		accountInfo, err = getAzureAccountInfo(cliPath)
		if err != nil {
			log.Printf("Warning: Failed to get Azure account info: %v", err)
		}
	}

	return &AzureDiscoverer{
		cred:           cred,
		subscriptionID: subscriptionID,
		cliPath:        cliPath,
		accountInfo:    accountInfo,
		clients:        clients,
		progressChan:   make(chan DiscoveryProgress, 100),
		errorChan:      make(chan AzureDiscoveryError, 100),
	}, nil
}

// Name returns the provider name
func (d *AzureDiscoverer) Name() string {
	return "Microsoft Azure"
}

// SupportedRegions returns the list of supported Azure regions
func (d *AzureDiscoverer) SupportedRegions() []string {
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
func (d *AzureDiscoverer) SupportedResourceTypes() []string {
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
		"azurerm_key_vault",
		"azurerm_container_registry",
		"azurerm_kubernetes_cluster",
	}
}

// Discover discovers Azure resources using the best available method (SDK preferred, CLI fallback)
func (d *AzureDiscoverer) Discover(config Config) ([]models.Resource, error) {
	log.Printf("[Azure] Starting comprehensive resource discovery...")

	var allResources []models.Resource
	var mu sync.Mutex

	// Start progress monitoring
	go d.monitorProgress()

	// Try SDK first for better performance and accuracy
	if d.clients.Resources != nil {
		sdkResources, err := d.discoverViaSDK(context.Background(), config)
		if err != nil {
			log.Printf("[Azure] SDK discovery failed: %v, falling back to CLI", err)
		} else {
			mu.Lock()
			allResources = append(allResources, sdkResources...)
			mu.Unlock()
			log.Printf("[Azure] SDK discovery completed: %d resources found", len(sdkResources))
		}
	}

	// Use CLI discovery as primary method or fallback
	if d.cliPath != "" {
		cliResources, err := d.discoverViaCLI(context.Background(), config)
		if err != nil {
			log.Printf("[Azure] CLI discovery failed: %v", err)
		} else {
			mu.Lock()
			// Remove duplicates by ID
			cliResources = d.removeDuplicates(allResources, cliResources)
			allResources = append(allResources, cliResources...)
			mu.Unlock()
			log.Printf("[Azure] CLI discovery completed: %d additional resources found", len(cliResources))
		}
	}

	// If both methods failed, return an error
	if len(allResources) == 0 {
		return nil, fmt.Errorf("both SDK and CLI discovery methods failed")
	}

	log.Printf("[Azure] Total resources discovered: %d", len(allResources))
	return allResources, nil
}

// DiscoverAllResources discovers all Azure resources without region filtering
func (d *AzureDiscoverer) DiscoverAllResources(ctx context.Context) ([]models.Resource, error) {
	return d.Discover(Config{})
}

// DiscoverForRegions discovers resources in specific regions
func (d *AzureDiscoverer) DiscoverForRegions(ctx context.Context, regions []string) ([]models.Resource, error) {
	return d.Discover(Config{Regions: regions})
}

// discoverViaSDK discovers resources using Azure SDK
func (d *AzureDiscoverer) discoverViaSDK(ctx context.Context, config Config) ([]models.Resource, error) {
	var allResources []models.Resource
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Discover different resource types concurrently
	resourceFuncs := []func(context.Context, Config) []models.Resource{
		d.discoverResourceGroupsSDK,
		d.discoverVirtualMachinesSDK,
		d.discoverStorageAccountsSDK,
		d.discoverVirtualNetworksSDK,
		d.discoverNetworkSecurityGroupsSDK,
		d.discoverLoadBalancersSDK,
		d.discoverPublicIPsSDK,
		d.discoverNetworkInterfacesSDK,
		d.discoverManagedDisksSDK,
	}

	for _, fn := range resourceFuncs {
		wg.Add(1)
		go func(discoveryFunc func(context.Context, Config) []models.Resource) {
			defer wg.Done()
			resources := discoveryFunc(ctx, config)
			mu.Lock()
			allResources = append(allResources, resources...)
			mu.Unlock()
		}(fn)
	}

	wg.Wait()
	return allResources, nil
}

// discoverViaCLI discovers resources using Azure CLI
func (d *AzureDiscoverer) discoverViaCLI(ctx context.Context, config Config) ([]models.Resource, error) {
	if !d.isAzureCLIAvailable() {
		return nil, fmt.Errorf("Azure CLI not available")
	}

	// Use the universal CLI discovery method for comprehensive coverage
	return d.discoverAllResourcesViaCLI(ctx, config.Regions)
}

// discoverAllResourcesViaCLI discovers all resources using Azure CLI's generic resource list
func (d *AzureDiscoverer) discoverAllResourcesViaCLI(ctx context.Context, regions []string) ([]models.Resource, error) {
	cmd := exec.CommandContext(ctx, "az", "resource", "list", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list Azure resources: %w", err)
	}

	var azResources []struct {
		ID            string                 `json:"id"`
		Name          string                 `json:"name"`
		Type          string                 `json:"type"`
		Location      string                 `json:"location"`
		ResourceGroup string                 `json:"resourceGroup"`
		Tags          map[string]interface{} `json:"tags"`
	}

	if err := json.Unmarshal(output, &azResources); err != nil {
		return nil, fmt.Errorf("failed to parse Azure resources: %w", err)
	}

	var resources []models.Resource
	for _, azResource := range azResources {
		// Filter by region if specified
		if len(regions) > 0 {
			regionFound := false
			for _, region := range regions {
				if azResource.Location == region {
					regionFound = true
					break
				}
			}
			if !regionFound {
				continue
			}
		}

		// Convert Azure resource type to driftmgr format
		resourceType := d.azureTypeToTerraformType(azResource.Type)

		// Convert tags
		tags := make(map[string]string)
		for k, v := range azResource.Tags {
			if str, ok := v.(string); ok {
				tags[k] = str
			}
		}

		resource := models.Resource{
			ID:        azResource.ID,
			Name:      azResource.Name,
			Type:      resourceType,
			Provider:  "azure",
			Region:    azResource.Location,
			Tags:      tags,
			CreatedAt: time.Now(),
			Metadata: map[string]string{
				"terraform_type": resourceType,
				"import_id":      azResource.ID,
				"resource_group": azResource.ResourceGroup,
			},
			Attributes: map[string]interface{}{
				"resourceGroup": azResource.ResourceGroup,
			},
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// SDK Discovery Methods

func (d *AzureDiscoverer) discoverResourceGroupsSDK(ctx context.Context, config Config) []models.Resource {
	var resources []models.Resource
	if d.clients.Resources == nil {
		return resources
	}

	pager := d.clients.Resources.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get resource groups page: %v", err)
			break
		}

		for _, group := range page.Value {
			if group.Name != nil && group.Location != nil {
				// Filter by region if specified
				if len(config.Regions) > 0 && !contains(config.Regions, *group.Location) {
					continue
				}

				resources = append(resources, models.Resource{
					ID:        safeString(group.ID),
					Name:      safeString(group.Name),
					Type:      "azurerm_resource_group",
					Provider:  "azure",
					Region:    safeString(group.Location),
					Tags:      convertSDKTags(group.Tags),
					CreatedAt: time.Now(),
					Metadata: map[string]string{
						"terraform_type": "azurerm_resource_group",
						"import_id":      safeString(group.ID),
					},
				})
			}
		}
	}

	d.progressChan <- DiscoveryProgress{Service: "Resources", Resources: len(resources)}
	return resources
}

func (d *AzureDiscoverer) discoverVirtualMachinesSDK(ctx context.Context, config Config) []models.Resource {
	var resources []models.Resource
	if d.clients.Compute == nil {
		return resources
	}

	pager := d.clients.Compute.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get VMs page: %v", err)
			break
		}

		for _, vm := range page.Value {
			if vm.Name != nil && vm.Location != nil {
				// Filter by region if specified
				if len(config.Regions) > 0 && !contains(config.Regions, *vm.Location) {
					continue
				}

				state := "unknown"
				if vm.Properties != nil && vm.Properties.ProvisioningState != nil {
					state = *vm.Properties.ProvisioningState
				}

				properties := make(map[string]interface{})
				if vm.Properties != nil && vm.Properties.HardwareProfile != nil && vm.Properties.HardwareProfile.VMSize != nil {
					properties["vm_size"] = string(*vm.Properties.HardwareProfile.VMSize)
				}

				resources = append(resources, models.Resource{
					ID:         safeString(vm.ID),
					Name:       safeString(vm.Name),
					Type:       "azurerm_virtual_machine",
					Provider:   "azure",
					Region:     safeString(vm.Location),
					State:      state,
					Tags:       convertSDKTags(vm.Tags),
					CreatedAt:  time.Now(),
					Properties: properties,
					Metadata: map[string]string{
						"terraform_type": "azurerm_virtual_machine",
						"import_id":      safeString(vm.ID),
					},
				})
			}
		}
	}

	d.progressChan <- DiscoveryProgress{Service: "Compute", Resources: len(resources)}
	return resources
}

func (d *AzureDiscoverer) discoverStorageAccountsSDK(ctx context.Context, config Config) []models.Resource {
	var resources []models.Resource
	if d.clients.Storage == nil {
		return resources
	}

	pager := d.clients.Storage.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get storage accounts page: %v", err)
			break
		}

		for _, account := range page.Value {
			if account.Name != nil && account.Location != nil {
				// Filter by region if specified
				if len(config.Regions) > 0 && !contains(config.Regions, *account.Location) {
					continue
				}

				properties := make(map[string]interface{})
				if account.Properties != nil {
					if account.Kind != nil {
						properties["kind"] = string(*account.Kind)
					}
					if account.SKU != nil && account.SKU.Name != nil {
						properties["sku"] = string(*account.SKU.Name)
					}
					if account.Properties.AccessTier != nil {
						properties["access_tier"] = string(*account.Properties.AccessTier)
					}
				}

				resources = append(resources, models.Resource{
					ID:         safeString(account.ID),
					Name:       safeString(account.Name),
					Type:       "azurerm_storage_account",
					Provider:   "azure",
					Region:     safeString(account.Location),
					Tags:       convertSDKTags(account.Tags),
					CreatedAt:  time.Now(),
					Properties: properties,
					Metadata: map[string]string{
						"terraform_type": "azurerm_storage_account",
						"import_id":      safeString(account.ID),
					},
				})
			}
		}
	}

	d.progressChan <- DiscoveryProgress{Service: "Storage", Resources: len(resources)}
	return resources
}

func (d *AzureDiscoverer) discoverVirtualNetworksSDK(ctx context.Context, config Config) []models.Resource {
	var resources []models.Resource
	if d.clients.Network == nil {
		return resources
	}

	pager := d.clients.Network.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get vnets page: %v", err)
			break
		}

		for _, vnet := range page.Value {
			if vnet.Name != nil && vnet.Location != nil {
				// Filter by region if specified
				if len(config.Regions) > 0 && !contains(config.Regions, *vnet.Location) {
					continue
				}

				properties := make(map[string]interface{})
				if vnet.Properties != nil && vnet.Properties.AddressSpace != nil {
					properties["address_prefixes"] = vnet.Properties.AddressSpace.AddressPrefixes
				}

				resources = append(resources, models.Resource{
					ID:         safeString(vnet.ID),
					Name:       safeString(vnet.Name),
					Type:       "azurerm_virtual_network",
					Provider:   "azure",
					Region:     safeString(vnet.Location),
					Tags:       convertSDKTags(vnet.Tags),
					CreatedAt:  time.Now(),
					Properties: properties,
					Metadata: map[string]string{
						"terraform_type": "azurerm_virtual_network",
						"import_id":      safeString(vnet.ID),
					},
				})
			}
		}
	}

	d.progressChan <- DiscoveryProgress{Service: "Network", Resources: len(resources)}
	return resources
}

func (d *AzureDiscoverer) discoverNetworkSecurityGroupsSDK(ctx context.Context, config Config) []models.Resource {
	var resources []models.Resource
	// Implementation would use armnetwork.SecurityGroupsClient
	return resources
}

func (d *AzureDiscoverer) discoverLoadBalancersSDK(ctx context.Context, config Config) []models.Resource {
	var resources []models.Resource
	// Implementation would use armnetwork.LoadBalancersClient
	return resources
}

func (d *AzureDiscoverer) discoverPublicIPsSDK(ctx context.Context, config Config) []models.Resource {
	var resources []models.Resource
	// Implementation would use armnetwork.PublicIPAddressesClient
	return resources
}

func (d *AzureDiscoverer) discoverNetworkInterfacesSDK(ctx context.Context, config Config) []models.Resource {
	var resources []models.Resource
	// Implementation would use armnetwork.InterfacesClient
	return resources
}

func (d *AzureDiscoverer) discoverManagedDisksSDK(ctx context.Context, config Config) []models.Resource {
	var resources []models.Resource
	// Implementation would use armcompute.DisksClient
	return resources
}

// Helper Methods

// azureTypeToTerraformType converts Azure resource type to Terraform resource type
func (d *AzureDiscoverer) azureTypeToTerraformType(azureType string) string {
	typeMap := map[string]string{
		"Microsoft.Compute/virtualMachines":          "azurerm_virtual_machine",
		"Microsoft.Network/virtualNetworks":          "azurerm_virtual_network",
		"Microsoft.Network/networkSecurityGroups":    "azurerm_network_security_group",
		"Microsoft.Storage/storageAccounts":          "azurerm_storage_account",
		"Microsoft.Sql/servers":                      "azurerm_sql_server",
		"Microsoft.Sql/servers/databases":            "azurerm_sql_database",
		"Microsoft.Web/sites":                        "azurerm_app_service",
		"Microsoft.Web/serverfarms":                  "azurerm_app_service_plan",
		"Microsoft.Resources/resourceGroups":         "azurerm_resource_group",
		"Microsoft.Network/publicIPAddresses":        "azurerm_public_ip",
		"Microsoft.Network/networkInterfaces":        "azurerm_network_interface",
		"Microsoft.KeyVault/vaults":                  "azurerm_key_vault",
		"Microsoft.ContainerRegistry/registries":     "azurerm_container_registry",
		"Microsoft.ContainerService/managedClusters": "azurerm_kubernetes_cluster",
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

// isAzureCLIAvailable checks if Azure CLI is available
func (d *AzureDiscoverer) isAzureCLIAvailable() bool {
	if d.cliPath == "" {
		return false
	}
	cmd := exec.Command(d.cliPath, "--version")
	return cmd.Run() == nil
}

// monitorProgress monitors discovery progress
func (d *AzureDiscoverer) monitorProgress() {
	for {
		select {
		case progress := <-d.progressChan:
			log.Printf("[Azure] %s discovery: %d resources found", progress.Service, progress.Resources)
		case err := <-d.errorChan:
			log.Printf("[Azure] Discovery error in %s: %v", err.Service, err.Error)
		}
	}
}

// removeDuplicates removes duplicate resources by ID
func (d *AzureDiscoverer) removeDuplicates(existing, new []models.Resource) []models.Resource {
	existingIDs := make(map[string]bool)
	for _, resource := range existing {
		existingIDs[resource.ID] = true
	}

	var filtered []models.Resource
	for _, resource := range new {
		if !existingIDs[resource.ID] {
			filtered = append(filtered, resource)
		}
	}
	return filtered
}

// Utility Functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func safeString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
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

func convertTags(tags map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range tags {
		if str, ok := v.(string); ok {
			result[k] = str
		}
	}
	return result
}

func getMap(data map[string]interface{}, key string) map[string]interface{} {
	if val, ok := data[key]; ok {
		if m, ok := val.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

// CLI Helper Functions

func getAzureCLIPath() string {
	// Try to find in PATH first
	if azPath, err := exec.LookPath("az"); err == nil {
		return azPath
	}

	// Windows-specific paths
	if runtime.GOOS == "windows" {
		paths := []string{
			`C:\Program Files (x86)\Microsoft SDKs\Azure\CLI2\wbin\az.cmd`,
			`C:\Program Files\Microsoft SDKs\Azure\CLI2\wbin\az.cmd`,
			`C:\Users\%USERNAME%\AppData\Local\Programs\Microsoft Azure CLI\az.cmd`,
		}

		for _, path := range paths {
			expandedPath := os.ExpandEnv(path)
			if _, err := os.Stat(expandedPath); err == nil {
				return expandedPath
			}
		}
	}

	return ""
}

func getAzureAccountInfo(cliPath string) (*AzureAccountInfo, error) {
	cmd := exec.Command(cliPath, "account", "show", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}

	var accountInfo AzureAccountInfo
	if err := json.Unmarshal(output, &accountInfo); err != nil {
		return nil, fmt.Errorf("failed to parse account info: %w", err)
	}

	return &accountInfo, nil
}
