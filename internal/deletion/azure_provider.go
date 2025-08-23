package deletion

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/apimanagement/armapimanagement"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/automation/armautomation"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/eventhub/armeventhub"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/logic/armlogic"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/machinelearning/armmachinelearning"
	// "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor" // Not used after removing Application Insights
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/notificationhubs/armnotificationhubs"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/operationalinsights/armoperationalinsights"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservices"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/relay/armrelay"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/search/armsearch"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/servicebus/armservicebus"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/signalr/armsignalr"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"

	"github.com/catherinevee/driftmgr/internal/models"
)

// AzureProvider implements CloudProvider for Azure
type AzureProvider struct {
	cred           azcore.TokenCredential
	client         *armresources.Client
	subscriptionID string
}

// convertTags converts Azure tags (map[string]*string) to map[string]string
func convertTags(tags map[string]*string) map[string]string {
	if tags == nil {
		return nil
	}
	result := make(map[string]string)
	for k, v := range tags {
		if v != nil {
			result[k] = *v
		}
	}
	return result
}

// NewAzureProvider creates a new Azure provider
func NewAzureProvider() (*AzureProvider, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	// Get subscription ID from environment or use default
	subscriptionID := getAzureSubscriptionID()
	if subscriptionID == "" {
		return nil, fmt.Errorf("no Azure subscription ID found. Set AZURE_SUBSCRIPTION_ID environment variable or configure Azure CLI")
	}

	client, err := armresources.NewClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure resources client: %w", err)
	}

	return &AzureProvider{
		cred:           cred,
		client:         client,
		subscriptionID: subscriptionID,
	}, nil
}

// getAzureSubscriptionID gets the Azure subscription ID from environment or Azure CLI
func getAzureSubscriptionID() string {
	// First try environment variable
	if subID := os.Getenv("AZURE_SUBSCRIPTION_ID"); subID != "" {
		return subID
	}

	// Try to get from Azure CLI
	cmd := exec.Command("az", "account", "show", "--query", "id", "-o", "tsv")
	output, err := cmd.Output()
	if err == nil {
		subID := strings.TrimSpace(string(output))
		if subID != "" {
			return subID
		}
	}

	return ""
}

// ValidateCredentials validates Azure credentials
func (ap *AzureProvider) ValidateCredentials(ctx context.Context, accountID string) error {
	// Test the credentials by listing resource groups
	pager := ap.client.NewListPager(nil)
	_, err := pager.NextPage(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate Azure credentials: %w", err)
	}

	return nil
}

// ListResources lists all Azure resources
func (ap *AzureProvider) ListResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Define resource discovery functions
	discoveryFuncs := []struct {
		name string
		fn   func(context.Context, string) ([]models.Resource, error)
	}{
		{"VirtualMachines", ap.discoverVirtualMachines},
		{"StorageAccounts", ap.discoverStorageAccounts},
		{"VirtualNetworks", ap.discoverVirtualNetworks},
		{"ResourceGroups", ap.discoverResourceGroups},
		{"NetworkInterfaces", ap.discoverNetworkInterfaces},
		{"PublicIPAddresses", ap.discoverPublicIPAddresses},
		{"LoadBalancers", ap.discoverLoadBalancers},
		{"ApplicationGateways", ap.discoverApplicationGateways},
		{"KeyVaults", ap.discoverKeyVaults},
		{"AppServices", ap.discoverAppServices},
		{"ContainerRegistries", ap.discoverContainerRegistries},
		{"KubernetesServices", ap.discoverKubernetesServices},
		{"SQLServers", ap.discoverSQLServers},
		{"SQLDatabases", ap.discoverSQLDatabases},
		{"RedisCaches", ap.discoverRedisCaches},
		{"CosmosDBAccounts", ap.discoverCosmosDBAccounts},
		{"EventHubs", ap.discoverEventHubs},
		{"ServiceBusNamespaces", ap.discoverServiceBusNamespaces},
		{"LogicApps", ap.discoverLogicApps},
		{"APIManagement", ap.discoverAPIManagement},
		{"SearchServices", ap.discoverSearchServices},
		{"MachineLearning", ap.discoverMachineLearning},
		{"AutomationAccounts", ap.discoverAutomationAccounts},
		{"RecoveryServices", ap.discoverRecoveryServices},
		{"ApplicationInsights", ap.discoverApplicationInsights},
		{"LogAnalyticsWorkspaces", ap.discoverLogAnalyticsWorkspaces},
		{"NotificationHubs", ap.discoverNotificationHubs},
		{"RelayNamespaces", ap.discoverRelayNamespaces},
		{"SignalR", ap.discoverSignalR},
		{"CommunicationServices", ap.discoverCommunicationServices},
		{"DesktopVirtualization", ap.discoverDesktopVirtualization},
		{"HealthcareAPIs", ap.discoverHealthcareAPIs},
		{"IoTCentral", ap.discoverIoTCentral},
		{"IoTSecurity", ap.discoverIoTSecurity},
		{"MapsAccounts", ap.discoverMapsAccounts},
		{"MixedReality", ap.discoverMixedReality},
		{"QuantumWorkspaces", ap.discoverQuantumWorkspaces},
		{"VisualStudio", ap.discoverVisualStudio},
		{"VMwareCloudSimple", ap.discoverVMwareCloudSimple},
		{"WindowsESU", ap.discoverWindowsESU},
		{"WindowsIoT", ap.discoverWindowsIoT},
	}

	for _, discovery := range discoveryFuncs {
		wg.Add(1)
		go func(d struct {
			name string
			fn   func(context.Context, string) ([]models.Resource, error)
		}) {
			defer wg.Done()

			res, err := d.fn(ctx, accountID)
			if err != nil {
				log.Printf("Error discovering %s resources: %v", d.name, err)
				return
			}

			mu.Lock()
			resources = append(resources, res...)
			mu.Unlock()
		}(discovery)
	}

	wg.Wait()
	return resources, nil
}

// DeleteResources deletes Azure resources in the correct order
func (ap *AzureProvider) DeleteResources(ctx context.Context, accountID string, options DeletionOptions) (*DeletionResult, error) {
	startTime := time.Now()
	result := &DeletionResult{
		AccountID: accountID,
		Provider:  "azure",
		StartTime: startTime,
		Errors:    []DeletionError{},
		Warnings:  []string{},
		Details:   make(map[string]interface{}),
	}

	// List all resources first
	resources, err := ap.ListResources(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	result.TotalResources = len(resources)

	// Filter resources based on options
	filteredResources := ap.filterResources(resources, options)

	if options.DryRun {
		result.DeletedResources = len(filteredResources)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(startTime)
		return result, nil
	}

	// Delete resources in dependency order
	deletionOrder := []string{
		"microsoft.compute/virtualmachines",
		"microsoft.network/networkinterfaces",
		"microsoft.network/publicipaddresses",
		"microsoft.network/loadbalancers",
		"microsoft.network/applicationgateways",
		"microsoft.network/virtualnetworks",
		"microsoft.storage/storageaccounts",
		"microsoft.keyvault/vaults",
		"microsoft.web/sites",
		"microsoft.containerregistry/registries",
		"microsoft.containerservice/managedclusters",
		"microsoft.resources/resourcegroups",
	}

	// Group resources by type
	resourceGroups := make(map[string][]models.Resource)
	for _, resource := range filteredResources {
		resourceGroups[resource.Type] = append(resourceGroups[resource.Type], resource)
	}

	// Delete resources in order
	for _, resourceType := range deletionOrder {
		if resources, exists := resourceGroups[resourceType]; exists {
			for _, resource := range resources {
				if err := ap.deleteResource(ctx, resource, options); err != nil {
					result.Errors = append(result.Errors, DeletionError{
						ResourceID:   resource.ID,
						ResourceType: resource.Type,
						Error:        err.Error(),
						Timestamp:    time.Now(),
					})
					result.FailedResources++
				} else {
					result.DeletedResources++
				}

				// Send progress update
				if options.ProgressCallback != nil {
					options.ProgressCallback(ProgressUpdate{
						Type:      "deletion_progress",
						Message:   fmt.Sprintf("Deleted %s: %s", resource.Type, resource.Name),
						Progress:  result.DeletedResources + result.FailedResources,
						Total:     result.TotalResources,
						Current:   resource.Name,
						Timestamp: time.Now(),
					})
				}
			}
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(startTime)
	return result, nil
}

// deleteResource deletes a single Azure resource
// DeleteResource implements the CloudProvider interface for single resource deletion
func (ap *AzureProvider) DeleteResource(ctx context.Context, resource models.Resource) error {
	return ap.deleteResource(ctx, resource, DeletionOptions{})
}

func (ap *AzureProvider) deleteResource(ctx context.Context, resource models.Resource, options DeletionOptions) error {
	switch resource.Type {
	case "microsoft.compute/virtualmachines":
		return ap.deleteVirtualMachine(ctx, resource)
	case "microsoft.storage/storageaccounts":
		return ap.deleteStorageAccount(ctx, resource)
	case "microsoft.network/virtualnetworks":
		return ap.deleteVirtualNetwork(ctx, resource)
	case "microsoft.network/networkinterfaces":
		return ap.deleteNetworkInterface(ctx, resource)
	case "microsoft.network/publicipaddresses":
		return ap.deletePublicIPAddress(ctx, resource)
	case "microsoft.network/loadbalancers":
		return ap.deleteLoadBalancer(ctx, resource)
	case "microsoft.network/applicationgateways":
		return ap.deleteApplicationGateway(ctx, resource)
	case "microsoft.keyvault/vaults":
		return ap.deleteKeyVault(ctx, resource)
	case "microsoft.web/sites":
		return ap.deleteAppService(ctx, resource)
	case "microsoft.containerregistry/registries":
		return ap.deleteContainerRegistry(ctx, resource)
	case "microsoft.containerservice/managedclusters":
		return ap.deleteKubernetesService(ctx, resource)
	case "microsoft.resources/resourcegroups":
		return ap.deleteResourceGroup(ctx, resource)
	default:
		return fmt.Errorf("unsupported resource type: %s", resource.Type)
	}
}

// filterResources filters resources based on deletion options
func (ap *AzureProvider) filterResources(resources []models.Resource, options DeletionOptions) []models.Resource {
	var filtered []models.Resource

	for _, resource := range resources {
		// Check if resource should be excluded
		if ap.shouldExcludeResource(resource, options) {
			continue
		}

		// Check if resource should be included
		if len(options.IncludeResources) > 0 && !ap.shouldIncludeResource(resource, options) {
			continue
		}

		// Check resource type filter
		if len(options.ResourceTypes) > 0 && !ap.containsString(options.ResourceTypes, resource.Type) {
			continue
		}

		// Check region filter
		if len(options.Regions) > 0 && !ap.containsString(options.Regions, resource.Region) {
			continue
		}

		filtered = append(filtered, resource)
	}

	return filtered
}

// Helper methods for resource discovery
func (ap *AzureProvider) discoverVirtualMachines(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armcompute.NewVirtualMachinesClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			continue
		}

		for _, vm := range page.Value {
			if vm != nil && vm.ID != nil {
				resources = append(resources, models.Resource{
					ID:       *vm.ID,
					Name:     *vm.Name,
					Type:     "microsoft.compute/virtualmachines",
					Provider: "azure",
					Region:   *vm.Location,
					State:    string(*vm.Properties.ProvisioningState),
					Created:  time.Now(), // Azure doesn't provide creation time in this API
				})
			}
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverStorageAccounts(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armstorage.NewAccountsClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			continue
		}

		for _, account := range page.Value {
			if account != nil && account.ID != nil {
				resources = append(resources, models.Resource{
					ID:       *account.ID,
					Name:     *account.Name,
					Type:     "microsoft.storage/storageaccounts",
					Provider: "azure",
					Region:   *account.Location,
					State:    "Active", // Placeholder state
					Created:  time.Now(),
				})
			}
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverVirtualNetworks(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armnetwork.NewVirtualNetworksClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			continue
		}

		for _, vnet := range page.Value {
			if vnet != nil && vnet.ID != nil {
				resources = append(resources, models.Resource{
					ID:       *vnet.ID,
					Name:     *vnet.Name,
					Type:     "microsoft.network/virtualnetworks",
					Provider: "azure",
					Region:   *vnet.Location,
					State:    string(*vnet.Properties.ProvisioningState),
					Created:  time.Now(),
				})
			}
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverResourceGroups(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	pager := ap.client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			continue
		}

		for _, group := range page.Value {
			if group != nil && group.ID != nil {
				resources = append(resources, models.Resource{
					ID:       *group.ID,
					Name:     *group.Name,
					Type:     "microsoft.resources/resourcegroups",
					Provider: "azure",
					Region:   *group.Location,
					State:    "Succeeded", // Placeholder state
					Created:  time.Now(),
				})
			}
		}
	}

	return resources, nil
}

// Additional discovery methods would be implemented similarly for other Azure services
func (ap *AzureProvider) discoverNetworkInterfaces(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armnetwork.NewInterfacesClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, nic := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *nic.ID,
				Name:     *nic.Name,
				Type:     "microsoft.network/networkinterfaces",
				Provider: "azure",
				Region:   *nic.Location,
				Tags:     convertTags(nic.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverPublicIPAddresses(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armnetwork.NewPublicIPAddressesClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, pip := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *pip.ID,
				Name:     *pip.Name,
				Type:     "microsoft.network/publicipaddresses",
				Provider: "azure",
				Region:   *pip.Location,
				Tags:     convertTags(pip.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverLoadBalancers(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armnetwork.NewLoadBalancersClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, lb := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *lb.ID,
				Name:     *lb.Name,
				Type:     "microsoft.network/loadbalancers",
				Provider: "azure",
				Region:   *lb.Location,
				Tags:     convertTags(lb.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverApplicationGateways(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armnetwork.NewApplicationGatewaysClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, ag := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *ag.ID,
				Name:     *ag.Name,
				Type:     "microsoft.network/applicationgateways",
				Provider: "azure",
				Region:   *ag.Location,
				Tags:     convertTags(ag.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverKeyVaults(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armkeyvault.NewVaultsClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListBySubscriptionPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, vault := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *vault.ID,
				Name:     *vault.Name,
				Type:     "microsoft.keyvault/vaults",
				Provider: "azure",
				Region:   *vault.Location,
				Tags:     convertTags(vault.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverAppServices(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armappservice.NewWebAppsClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, app := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *app.ID,
				Name:     *app.Name,
				Type:     "microsoft.web/sites",
				Provider: "azure",
				Region:   *app.Location,
				Tags:     convertTags(app.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverContainerRegistries(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armcontainerregistry.NewRegistriesClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, registry := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *registry.ID,
				Name:     *registry.Name,
				Type:     "microsoft.containerregistry/registries",
				Provider: "azure",
				Region:   *registry.Location,
				Tags:     convertTags(registry.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverKubernetesServices(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armcontainerservice.NewManagedClustersClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, cluster := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *cluster.ID,
				Name:     *cluster.Name,
				Type:     "microsoft.containerservice/managedclusters",
				Provider: "azure",
				Region:   *cluster.Location,
				Tags:     convertTags(cluster.Tags),
			})
		}
	}

	return resources, nil
}

// Missing discovery methods
func (ap *AzureProvider) discoverSQLServers(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armsql.NewServersClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, server := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *server.ID,
				Name:     *server.Name,
				Type:     "microsoft.sql/servers",
				Provider: "azure",
				Region:   *server.Location,
				Tags:     convertTags(server.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverSQLDatabases(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	// First get all SQL servers
	serversClient, err := armsql.NewServersClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	serversPager := serversClient.NewListPager(nil)
	for serversPager.More() {
		serversPage, err := serversPager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, server := range serversPage.Value {
			// Get databases for each server
			dbClient, err := armsql.NewDatabasesClient(ap.subscriptionID, ap.cred, nil)
			if err != nil {
				continue
			}

			resourceGroup := ap.extractResourceGroupName(*server.ID)
			dbPager := dbClient.NewListByServerPager(resourceGroup, *server.Name, nil)
			
			for dbPager.More() {
				dbPage, err := dbPager.NextPage(ctx)
				if err != nil {
					break
				}

				for _, db := range dbPage.Value {
					resources = append(resources, models.Resource{
						ID:       *db.ID,
						Name:     *db.Name,
						Type:     "microsoft.sql/servers/databases",
						Provider: "azure",
						Region:   *db.Location,
						Tags:     convertTags(db.Tags),
					})
				}
			}
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverRedisCaches(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armredis.NewClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListBySubscriptionPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, cache := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *cache.ID,
				Name:     *cache.Name,
				Type:     "microsoft.cache/redis",
				Provider: "azure",
				Region:   *cache.Location,
				Tags:     convertTags(cache.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverCosmosDBAccounts(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armcosmos.NewDatabaseAccountsClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, account := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *account.ID,
				Name:     *account.Name,
				Type:     "microsoft.documentdb/databaseaccounts",
				Provider: "azure",
				Region:   *account.Location,
				Tags:     convertTags(account.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverEventHubs(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armeventhub.NewNamespacesClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, namespace := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *namespace.ID,
				Name:     *namespace.Name,
				Type:     "microsoft.eventhub/namespaces",
				Provider: "azure",
				Region:   *namespace.Location,
				Tags:     convertTags(namespace.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverServiceBusNamespaces(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armservicebus.NewNamespacesClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, namespace := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *namespace.ID,
				Name:     *namespace.Name,
				Type:     "microsoft.servicebus/namespaces",
				Provider: "azure",
				Region:   *namespace.Location,
				Tags:     convertTags(namespace.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverLogicApps(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armlogic.NewWorkflowsClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListBySubscriptionPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, workflow := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *workflow.ID,
				Name:     *workflow.Name,
				Type:     "microsoft.logic/workflows",
				Provider: "azure",
				Region:   *workflow.Location,
				Tags:     convertTags(workflow.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverAPIManagement(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armapimanagement.NewServiceClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, service := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *service.ID,
				Name:     *service.Name,
				Type:     "microsoft.apimanagement/service",
				Provider: "azure",
				Region:   *service.Location,
				Tags:     convertTags(service.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverSearchServices(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armsearch.NewServicesClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListBySubscriptionPager(nil, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, service := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *service.ID,
				Name:     *service.Name,
				Type:     "microsoft.search/searchservices",
				Provider: "azure",
				Region:   *service.Location,
				Tags:     convertTags(service.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverMachineLearning(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armmachinelearning.NewWorkspacesClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListBySubscriptionPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, workspace := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *workspace.ID,
				Name:     *workspace.Name,
				Type:     "microsoft.machinelearningservices/workspaces",
				Provider: "azure",
				Region:   *workspace.Location,
				Tags:     convertTags(workspace.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverAutomationAccounts(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armautomation.NewAccountClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, account := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *account.ID,
				Name:     *account.Name,
				Type:     "microsoft.automation/automationaccounts",
				Provider: "azure",
				Region:   *account.Location,
				Tags:     convertTags(account.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverRecoveryServices(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armrecoveryservices.NewVaultsClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListBySubscriptionIDPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, vault := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *vault.ID,
				Name:     *vault.Name,
				Type:     "microsoft.recoveryservices/vaults",
				Provider: "azure",
				Region:   *vault.Location,
				Tags:     convertTags(vault.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverApplicationInsights(ctx context.Context, accountID string) ([]models.Resource, error) {
	// Application Insights discovery not implemented yet
	// The armmonitor package doesn't have ComponentsClient
	// This would require the applicationinsights package
	return []models.Resource{}, nil
}

func (ap *AzureProvider) discoverLogAnalyticsWorkspaces(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armoperationalinsights.NewWorkspacesClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, workspace := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *workspace.ID,
				Name:     *workspace.Name,
				Type:     "microsoft.operationalinsights/workspaces",
				Provider: "azure",
				Region:   *workspace.Location,
				Tags:     convertTags(workspace.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverNotificationHubs(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armnotificationhubs.NewNamespacesClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager("", nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, namespace := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *namespace.ID,
				Name:     *namespace.Name,
				Type:     "microsoft.notificationhubs/namespaces",
				Provider: "azure",
				Region:   *namespace.Location,
				Tags:     convertTags(namespace.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverRelayNamespaces(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armrelay.NewNamespacesClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, namespace := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *namespace.ID,
				Name:     *namespace.Name,
				Type:     "microsoft.relay/namespaces",
				Provider: "azure",
				Region:   *namespace.Location,
				Tags:     convertTags(namespace.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverSignalR(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client, err := armsignalr.NewClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListBySubscriptionPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, signalr := range page.Value {
			resources = append(resources, models.Resource{
				ID:       *signalr.ID,
				Name:     *signalr.Name,
				Type:     "microsoft.signalrservice/signalr",
				Provider: "azure",
				Region:   *signalr.Location,
				Tags:     convertTags(signalr.Tags),
			})
		}
	}

	return resources, nil
}

func (ap *AzureProvider) discoverCommunicationServices(ctx context.Context, accountID string) ([]models.Resource, error) {
	// Communication Services requires specialized SDK
	return nil, fmt.Errorf("Azure Communication Services discovery requires specialized configuration")
}

func (ap *AzureProvider) discoverDesktopVirtualization(ctx context.Context, accountID string) ([]models.Resource, error) {
	// Desktop Virtualization requires specialized SDK
	return nil, fmt.Errorf("Azure Desktop Virtualization discovery requires specialized configuration")
}

func (ap *AzureProvider) discoverHealthcareAPIs(ctx context.Context, accountID string) ([]models.Resource, error) {
	// Healthcare APIs require specialized SDK
	return nil, fmt.Errorf("Azure Healthcare APIs discovery requires specialized configuration")
}

func (ap *AzureProvider) discoverIoTCentral(ctx context.Context, accountID string) ([]models.Resource, error) {
	// IoT Central requires specialized SDK
	return nil, fmt.Errorf("Azure IoT Central discovery requires specialized configuration")
}

func (ap *AzureProvider) discoverIoTSecurity(ctx context.Context, accountID string) ([]models.Resource, error) {
	// IoT Security requires specialized SDK
	return nil, fmt.Errorf("Azure IoT Security discovery requires specialized configuration")
}

func (ap *AzureProvider) discoverMapsAccounts(ctx context.Context, accountID string) ([]models.Resource, error) {
	// Maps requires specialized SDK
	return nil, fmt.Errorf("Azure Maps discovery requires specialized configuration")
}

func (ap *AzureProvider) discoverMixedReality(ctx context.Context, accountID string) ([]models.Resource, error) {
	// Mixed Reality requires specialized SDK
	return nil, fmt.Errorf("Azure Mixed Reality discovery requires specialized configuration")
}

func (ap *AzureProvider) discoverQuantumWorkspaces(ctx context.Context, accountID string) ([]models.Resource, error) {
	// Quantum requires specialized SDK
	return nil, fmt.Errorf("Azure Quantum discovery requires specialized configuration")
}

func (ap *AzureProvider) discoverVisualStudio(ctx context.Context, accountID string) ([]models.Resource, error) {
	// Visual Studio requires specialized SDK
	return nil, fmt.Errorf("Azure Visual Studio discovery requires specialized configuration")
}

func (ap *AzureProvider) discoverVMwareCloudSimple(ctx context.Context, accountID string) ([]models.Resource, error) {
	// VMware CloudSimple requires specialized SDK
	return nil, fmt.Errorf("Azure VMware CloudSimple discovery requires specialized configuration")
}

func (ap *AzureProvider) discoverWindowsESU(ctx context.Context, accountID string) ([]models.Resource, error) {
	// Windows ESU requires specialized SDK
	return nil, fmt.Errorf("Azure Windows ESU discovery requires specialized configuration")
}

func (ap *AzureProvider) discoverWindowsIoT(ctx context.Context, accountID string) ([]models.Resource, error) {
	// Windows IoT requires specialized SDK
	return nil, fmt.Errorf("Azure Windows IoT discovery requires specialized configuration")
}

// Helper methods for resource deletion
func (ap *AzureProvider) deleteVirtualMachine(ctx context.Context, resource models.Resource) error {
	client, err := armcompute.NewVirtualMachinesClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return err
	}

	// Extract resource group and VM name from resource ID
	resourceGroup, vmName, err := ap.extractResourceGroupAndName(resource.ID)
	if err != nil {
		return err
	}

	poller, err := client.BeginDelete(ctx, resourceGroup, vmName, nil)
	if err != nil {
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (ap *AzureProvider) deleteStorageAccount(ctx context.Context, resource models.Resource) error {
	client, err := armstorage.NewAccountsClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return err
	}

	resourceGroup, accountName, err := ap.extractResourceGroupAndName(resource.ID)
	if err != nil {
		return err
	}

	_, err = client.Delete(ctx, resourceGroup, accountName, nil)
	return err
}

func (ap *AzureProvider) deleteVirtualNetwork(ctx context.Context, resource models.Resource) error {
	client, err := armnetwork.NewVirtualNetworksClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return err
	}

	resourceGroup, vnetName, err := ap.extractResourceGroupAndName(resource.ID)
	if err != nil {
		return err
	}

	poller, err := client.BeginDelete(ctx, resourceGroup, vnetName, nil)
	if err != nil {
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (ap *AzureProvider) deleteResourceGroup(ctx context.Context, resource models.Resource) error {
	// For now, return nil as this is a placeholder implementation
	// In a real implementation, you would properly delete the resource group
	return nil
}

// Additional deletion methods would be implemented for other Azure services
func (ap *AzureProvider) deleteNetworkInterface(ctx context.Context, resource models.Resource) error {
	client, err := armnetwork.NewInterfacesClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return err
	}

	resourceGroup, resourceName, err := ap.extractResourceGroupAndName(resource.ID)
	if err != nil {
		return err
	}

	poller, err := client.BeginDelete(ctx, resourceGroup, resourceName, nil)
	if err != nil {
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (ap *AzureProvider) deletePublicIPAddress(ctx context.Context, resource models.Resource) error {
	client, err := armnetwork.NewPublicIPAddressesClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return err
	}

	resourceGroup, resourceName, err := ap.extractResourceGroupAndName(resource.ID)
	if err != nil {
		return err
	}

	poller, err := client.BeginDelete(ctx, resourceGroup, resourceName, nil)
	if err != nil {
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (ap *AzureProvider) deleteLoadBalancer(ctx context.Context, resource models.Resource) error {
	client, err := armnetwork.NewLoadBalancersClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return err
	}

	resourceGroup, resourceName, err := ap.extractResourceGroupAndName(resource.ID)
	if err != nil {
		return err
	}

	poller, err := client.BeginDelete(ctx, resourceGroup, resourceName, nil)
	if err != nil {
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (ap *AzureProvider) deleteApplicationGateway(ctx context.Context, resource models.Resource) error {
	client, err := armnetwork.NewApplicationGatewaysClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return err
	}

	resourceGroup, resourceName, err := ap.extractResourceGroupAndName(resource.ID)
	if err != nil {
		return err
	}

	poller, err := client.BeginDelete(ctx, resourceGroup, resourceName, nil)
	if err != nil {
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (ap *AzureProvider) deleteKeyVault(ctx context.Context, resource models.Resource) error {
	client, err := armkeyvault.NewVaultsClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return err
	}

	resourceGroup, resourceName, err := ap.extractResourceGroupAndName(resource.ID)
	if err != nil {
		return err
	}

	_, err = client.Delete(ctx, resourceGroup, resourceName, nil)
	return err
}

func (ap *AzureProvider) deleteAppService(ctx context.Context, resource models.Resource) error {
	client, err := armappservice.NewWebAppsClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return err
	}

	resourceGroup, resourceName, err := ap.extractResourceGroupAndName(resource.ID)
	if err != nil {
		return err
	}

	_, err = client.Delete(ctx, resourceGroup, resourceName, nil)
	return err
}

func (ap *AzureProvider) deleteContainerRegistry(ctx context.Context, resource models.Resource) error {
	client, err := armcontainerregistry.NewRegistriesClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return err
	}

	resourceGroup, resourceName, err := ap.extractResourceGroupAndName(resource.ID)
	if err != nil {
		return err
	}

	poller, err := client.BeginDelete(ctx, resourceGroup, resourceName, nil)
	if err != nil {
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (ap *AzureProvider) deleteKubernetesService(ctx context.Context, resource models.Resource) error {
	client, err := armcontainerservice.NewManagedClustersClient(ap.subscriptionID, ap.cred, nil)
	if err != nil {
		return err
	}

	resourceGroup, resourceName, err := ap.extractResourceGroupAndName(resource.ID)
	if err != nil {
		return err
	}

	poller, err := client.BeginDelete(ctx, resourceGroup, resourceName, nil)
	if err != nil {
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

// Helper utility methods
func (ap *AzureProvider) shouldExcludeResource(resource models.Resource, options DeletionOptions) bool {
	for _, excludeID := range options.ExcludeResources {
		if resource.ID == excludeID || resource.Name == excludeID {
			return true
		}
	}
	return false
}

func (ap *AzureProvider) shouldIncludeResource(resource models.Resource, options DeletionOptions) bool {
	for _, includeID := range options.IncludeResources {
		if resource.ID == includeID || resource.Name == includeID {
			return true
		}
	}
	return false
}

func (ap *AzureProvider) containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (ap *AzureProvider) extractResourceGroupAndName(resourceID string) (string, string, error) {
	parts := strings.Split(resourceID, "/")
	if len(parts) < 9 {
		return "", "", fmt.Errorf("invalid resource ID format")
	}

	// Azure resource ID format: /subscriptions/{subscriptionId}/resourceGroups/{resourceGroup}/providers/{provider}/...
	resourceGroup := parts[4]
	resourceName := parts[len(parts)-1]

	return resourceGroup, resourceName, nil
}

func (ap *AzureProvider) extractResourceGroupName(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	if len(parts) >= 5 {
		return parts[4]
	}
	return ""
}
