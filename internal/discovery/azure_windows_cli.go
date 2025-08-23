package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// AzureWindowsCLI provides Windows-specific Azure CLI integration
type AzureWindowsCLI struct {
	cliPath        string
	subscriptionID string
	accountInfo    *AzureAccountInfo
}

// AzureAccountInfo represents Azure account information
type AzureAccountInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	State     string `json:"state"`
	IsDefault bool   `json:"isDefault"`
}

// NewAzureWindowsCLI creates a new Windows-specific Azure CLI integration
func NewAzureWindowsCLI() (*AzureWindowsCLI, error) {
	if runtime.GOOS != "windows" {
		return nil, fmt.Errorf("Azure Windows CLI is only supported on Windows")
	}

	cliPath := getAzureCLIPath()
	if cliPath == "" {
		return nil, fmt.Errorf("Azure CLI not found. Please install Azure CLI from https://docs.microsoft.com/en-us/cli/azure/install-azure-cli-windows")
	}

	// Get subscription ID
	subscriptionID := getAzureSubscriptionID()
	if subscriptionID == "" {
		return nil, fmt.Errorf("no Azure subscription ID found")
	}

	// Get account information
	accountInfo, err := getAzureAccountInfo(cliPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure account info: %w", err)
	}

	return &AzureWindowsCLI{
		cliPath:        cliPath,
		subscriptionID: subscriptionID,
		accountInfo:    accountInfo,
	}, nil
}

// DiscoverResourcesUsingCLI discovers Azure resources using Azure CLI
func (awc *AzureWindowsCLI) DiscoverResourcesUsingCLI(ctx context.Context) ([]models.Resource, error) {
	var allResources []models.Resource

	// Define CLI-based discovery functions
	discoveryFuncs := []struct {
		name string
		fn   func(context.Context) ([]models.Resource, error)
	}{
		{"VirtualMachines", awc.discoverVMsViaCLI},
		{"StorageAccounts", awc.discoverStorageAccountsViaCLI},
		{"VirtualNetworks", awc.discoverVirtualNetworksViaCLI},
		{"ResourceGroups", awc.discoverResourceGroupsViaCLI},
		{"NetworkInterfaces", awc.discoverNetworkInterfacesViaCLI},
		{"PublicIPAddresses", awc.discoverPublicIPsViaCLI},
		{"LoadBalancers", awc.discoverLoadBalancersViaCLI},
		{"KeyVaults", awc.discoverKeyVaultsViaCLI},
		{"AppServices", awc.discoverAppServicesViaCLI},
		{"ContainerRegistries", awc.discoverContainerRegistriesViaCLI},
		{"KubernetesServices", awc.discoverKubernetesServicesViaCLI},
		{"SQLServers", awc.discoverSQLServersViaCLI},
		{"RedisCaches", awc.discoverRedisCachesViaCLI},
		{"EventHubs", awc.discoverEventHubsViaCLI},
		{"ServiceBuses", awc.discoverServiceBusesViaCLI},
		{"CosmosDBAccounts", awc.discoverCosmosDBAccountsViaCLI},
		{"ApplicationGateways", awc.discoverApplicationGatewaysViaCLI},
		{"Firewalls", awc.discoverFirewallsViaCLI},
		{"BastionHosts", awc.discoverBastionHostsViaCLI},
		{"VPNGateways", awc.discoverVPNGatewaysViaCLI},
		{"ExpressRouteCircuits", awc.discoverExpressRouteCircuitsViaCLI},
		{"FunctionApps", awc.discoverFunctionAppsViaCLI},
		{"LogicApps", awc.discoverLogicAppsViaCLI},
		{"APIManagement", awc.discoverAPIManagementViaCLI},
		{"CDNProfiles", awc.discoverCDNProfilesViaCLI},
		{"FrontDoors", awc.discoverFrontDoorsViaCLI},
		{"ApplicationInsights", awc.discoverApplicationInsightsViaCLI},
		{"LogAnalyticsWorkspaces", awc.discoverLogAnalyticsWorkspacesViaCLI},
		{"BackupVaults", awc.discoverBackupVaultsViaCLI},
		{"RecoveryServicesVaults", awc.discoverRecoveryServicesVaultsViaCLI},
		{"SecurityCenters", awc.discoverSecurityCentersViaCLI},
		{"MonitorActionGroups", awc.discoverMonitorActionGroupsViaCLI},
		{"ManagedDisks", awc.discoverManagedDisksViaCLI},
		{"NetworkSecurityGroups", awc.discoverNetworkSecurityGroupsViaCLI},
		{"RouteTables", awc.discoverRouteTablesViaCLI},
		{"PrivateEndpoints", awc.discoverPrivateEndpointsViaCLI},
		{"PrivateLinkServices", awc.discoverPrivateLinkServicesViaCLI},
		{"TrafficManagerProfiles", awc.discoverTrafficManagerProfilesViaCLI},
		{"DNSZones", awc.discoverDNSZonesViaCLI},
		{"ContainerInstances", awc.discoverContainerInstancesViaCLI},
		{"SignalR", awc.discoverSignalRViaCLI},
		{"CognitiveServices", awc.discoverCognitiveServicesViaCLI},
		{"MachineLearningWorkspaces", awc.discoverMachineLearningWorkspacesViaCLI},
		{"DataFactories", awc.discoverDataFactoriesViaCLI},
		{"SynapseWorkspaces", awc.discoverSynapseWorkspacesViaCLI},
		{"DatabricksWorkspaces", awc.discoverDatabricksWorkspacesViaCLI},
		{"HDInsightClusters", awc.discoverHDInsightClustersViaCLI},
		{"StreamAnalyticsJobs", awc.discoverStreamAnalyticsJobsViaCLI},
		{"EventGridTopics", awc.discoverEventGridTopicsViaCLI},
		{"ServiceFabricClusters", awc.discoverServiceFabricClustersViaCLI},
		{"SpringCloudServices", awc.discoverSpringCloudServicesViaCLI},
		{"OpenShiftClusters", awc.discoverOpenShiftClustersViaCLI},
		{"BatchAccounts", awc.discoverBatchAccountsViaCLI},
		{"MediaServices", awc.discoverMediaServicesViaCLI},
		{"SearchServices", awc.discoverSearchServicesViaCLI},
		{"PowerBIDedicated", awc.discoverPowerBIDedicatedViaCLI},
		{"AnalysisServices", awc.discoverAnalysisServicesViaCLI},
		{"DataLakeAnalytics", awc.discoverDataLakeAnalyticsViaCLI},
		{"DataLakeStore", awc.discoverDataLakeStoreViaCLI},
		{"HPCClusters", awc.discoverHPCClustersViaCLI},
		{"LabServices", awc.discoverLabServicesViaCLI},
		{"DevTestLabs", awc.discoverDevTestLabsViaCLI},
		{"SharedImageGalleries", awc.discoverSharedImageGalleriesViaCLI},
		{"ImageDefinitions", awc.discoverImageDefinitionsViaCLI},
		{"ImageVersions", awc.discoverImageVersionsViaCLI},
		{"ProximityPlacementGroups", awc.discoverProximityPlacementGroupsViaCLI},
		{"AvailabilitySets", awc.discoverAvailabilitySetsViaCLI},
		{"VMScaleSets", awc.discoverVMScaleSetsViaCLI},
		{"VMImages", awc.discoverVMImagesViaCLI},
		{"Snapshots", awc.discoverSnapshotsViaCLI},
		{"RestorePointCollections", awc.discoverRestorePointCollectionsViaCLI},
		{"CapacityReservations", awc.discoverCapacityReservationsViaCLI},
		{"DedicatedHosts", awc.discoverDedicatedHostsViaCLI},
		{"DedicatedHostGroups", awc.discoverDedicatedHostGroupsViaCLI},
		{"GalleryApplications", awc.discoverGalleryApplicationsViaCLI},
		{"GalleryApplicationVersions", awc.discoverGalleryApplicationVersionsViaCLI},
	}

	// Execute discovery functions
	for _, discovery := range discoveryFuncs {
		resources, err := discovery.fn(ctx)
		if err != nil {
			fmt.Printf("Warning: Failed to discover %s: %v\n", discovery.name, err)
			continue
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

// Core CLI-based discovery methods

func (awc *AzureWindowsCLI) discoverVMsViaCLI(ctx context.Context) ([]models.Resource, error) {
	cmd := exec.CommandContext(ctx, awc.cliPath, "vm", "list", "--query",
		"[].{id:id,name:name,location:location,resourceGroup:resourceGroup,powerState:powerState,osType:storageProfile.osDisk.osType,tags:tags}",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}

	var vms []map[string]interface{}
	if err := json.Unmarshal(output, &vms); err != nil {
		return nil, fmt.Errorf("failed to parse VM list: %w", err)
	}

	var resources []models.Resource
	for _, vm := range vms {
		resource := models.Resource{
			ID:       getStringFromCLI(vm, "id"),
			Name:     getStringFromCLI(vm, "name"),
			Type:     "Microsoft.Compute/virtualMachines",
			Provider: "azure",
			Region:   getStringFromCLI(vm, "location"),
			State:    getStringFromCLI(vm, "powerState"),
			Tags:     convertCLITags(vm["tags"]),
			Created:  time.Now(),
			Updated:  time.Now(),
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

func (awc *AzureWindowsCLI) discoverStorageAccountsViaCLI(ctx context.Context) ([]models.Resource, error) {
	cmd := exec.CommandContext(ctx, awc.cliPath, "storage", "account", "list", "--query",
		"[].{id:id,name:name,location:location,resourceGroup:resourceGroup,statusOfPrimary:statusOfPrimary,accountType:sku.name,tags:tags}",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list storage accounts: %w", err)
	}

	var accounts []map[string]interface{}
	if err := json.Unmarshal(output, &accounts); err != nil {
		return nil, fmt.Errorf("failed to parse storage account list: %w", err)
	}

	var resources []models.Resource
	for _, account := range accounts {
		resource := models.Resource{
			ID:       getStringFromCLI(account, "id"),
			Name:     getStringFromCLI(account, "name"),
			Type:     "Microsoft.Storage/storageAccounts",
			Provider: "azure",
			Region:   getStringFromCLI(account, "location"),
			State:    getStringFromCLI(account, "statusOfPrimary"),
			Tags:     convertCLITags(account["tags"]),
			Created:  time.Now(),
			Updated:  time.Now(),
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

func (awc *AzureWindowsCLI) discoverVirtualNetworksViaCLI(ctx context.Context) ([]models.Resource, error) {
	cmd := exec.CommandContext(ctx, awc.cliPath, "network", "vnet", "list", "--query",
		"[].{id:id,name:name,location:location,resourceGroup:resourceGroup,addressSpace:addressSpace.addressPrefixes,tags:tags}",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list virtual networks: %w", err)
	}

	var vnets []map[string]interface{}
	if err := json.Unmarshal(output, &vnets); err != nil {
		return nil, fmt.Errorf("failed to parse virtual network list: %w", err)
	}

	var resources []models.Resource
	for _, vnet := range vnets {
		resource := models.Resource{
			ID:       getStringFromCLI(vnet, "id"),
			Name:     getStringFromCLI(vnet, "name"),
			Type:     "Microsoft.Network/virtualNetworks",
			Provider: "azure",
			Region:   getStringFromCLI(vnet, "location"),
			Tags:     convertCLITags(vnet["tags"]),
			State:    "Active",
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

func (awc *AzureWindowsCLI) discoverResourceGroupsViaCLI(ctx context.Context) ([]models.Resource, error) {
	cmd := exec.CommandContext(ctx, awc.cliPath, "group", "list", "--query",
		"[].{id:id,name:name,location:location,tags:tags}",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list resource groups: %w", err)
	}

	var groups []map[string]interface{}
	if err := json.Unmarshal(output, &groups); err != nil {
		return nil, fmt.Errorf("failed to parse resource group list: %w", err)
	}

	var resources []models.Resource
	for _, group := range groups {
		resource := models.Resource{
			ID:       getStringFromCLI(group, "id"),
			Name:     getStringFromCLI(group, "name"),
			Type:     "Microsoft.Resources/resourceGroups",
			Provider: "azure",
			Region:   getStringFromCLI(group, "location"),
			Tags:     convertCLITags(group["tags"]),
			State:    "Active",
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// Stub implementations for additional CLI-based discovery methods
// Implemented Azure CLI discovery methods

func (awc *AzureWindowsCLI) discoverNetworkInterfacesViaCLI(ctx context.Context) ([]models.Resource, error) {
	cmd := exec.CommandContext(ctx, "az", "network", "nic", "list", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to discover network interfaces: %w", err)
	}

	var nics []map[string]interface{}
	if err := json.Unmarshal(output, &nics); err != nil {
		return nil, fmt.Errorf("failed to parse network interfaces: %w", err)
	}

	var resources []models.Resource
	for _, nic := range nics {
		resource := models.Resource{
			ID:       getStringFromCLI(nic, "id"),
			Name:     getStringFromCLI(nic, "name"),
			Type:     "azure_network_interface",
			Provider: "azure",
			Region:   getStringFromCLI(nic, "location"),
			Tags:     convertTags(getMap(nic, "tags")),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (awc *AzureWindowsCLI) discoverPublicIPsViaCLI(ctx context.Context) ([]models.Resource, error) {
	cmd := exec.CommandContext(ctx, "az", "network", "public-ip", "list", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to discover public IPs: %w", err)
	}

	var ips []map[string]interface{}
	if err := json.Unmarshal(output, &ips); err != nil {
		return nil, fmt.Errorf("failed to parse public IPs: %w", err)
	}

	var resources []models.Resource
	for _, ip := range ips {
		resource := models.Resource{
			ID:       getStringFromCLI(ip, "id"),
			Name:     getStringFromCLI(ip, "name"),
			Type:     "azure_public_ip",
			Provider: "azure",
			Region:   getStringFromCLI(ip, "location"),
			Tags:     convertTags(getMap(ip, "tags")),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (awc *AzureWindowsCLI) discoverLoadBalancersViaCLI(ctx context.Context) ([]models.Resource, error) {
	cmd := exec.CommandContext(ctx, "az", "network", "lb", "list", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to discover load balancers: %w", err)
	}

	var lbs []map[string]interface{}
	if err := json.Unmarshal(output, &lbs); err != nil {
		return nil, fmt.Errorf("failed to parse load balancers: %w", err)
	}

	var resources []models.Resource
	for _, lb := range lbs {
		resource := models.Resource{
			ID:       getStringFromCLI(lb, "id"),
			Name:     getStringFromCLI(lb, "name"),
			Type:     "azure_load_balancer",
			Provider: "azure",
			Region:   getStringFromCLI(lb, "location"),
			Tags:     convertTags(getMap(lb, "tags")),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (awc *AzureWindowsCLI) discoverKeyVaultsViaCLI(ctx context.Context) ([]models.Resource, error) {
	cmd := exec.CommandContext(ctx, "az", "keyvault", "list", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to discover key vaults: %w", err)
	}

	var vaults []map[string]interface{}
	if err := json.Unmarshal(output, &vaults); err != nil {
		return nil, fmt.Errorf("failed to parse key vaults: %w", err)
	}

	var resources []models.Resource
	for _, vault := range vaults {
		resource := models.Resource{
			ID:       getStringFromCLI(vault, "id"),
			Name:     getStringFromCLI(vault, "name"),
			Type:     "azure_key_vault",
			Provider: "azure",
			Region:   getStringFromCLI(vault, "location"),
			Tags:     convertTags(getMap(vault, "tags")),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (awc *AzureWindowsCLI) discoverAppServicesViaCLI(ctx context.Context) ([]models.Resource, error) {
	cmd := exec.CommandContext(ctx, "az", "webapp", "list", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to discover app services: %w", err)
	}

	var apps []map[string]interface{}
	if err := json.Unmarshal(output, &apps); err != nil {
		return nil, fmt.Errorf("failed to parse app services: %w", err)
	}

	var resources []models.Resource
	for _, app := range apps {
		resource := models.Resource{
			ID:       getStringFromCLI(app, "id"),
			Name:     getStringFromCLI(app, "name"),
			Type:     "azure_app_service",
			Provider: "azure",
			Region:   getStringFromCLI(app, "location"),
			Tags:     convertTags(getMap(app, "tags")),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// Helper functions for other discovery methods
func (awc *AzureWindowsCLI) discoverContainerRegistriesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "acr", "azure_container_registry")
}

func (awc *AzureWindowsCLI) discoverKubernetesServicesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "aks", "azure_kubernetes_service")
}

func (awc *AzureWindowsCLI) discoverSQLServersViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "sql", "server", "azure_sql_server")
}

func (awc *AzureWindowsCLI) discoverRedisCachesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "redis", "azure_redis_cache")
}

func (awc *AzureWindowsCLI) discoverEventHubsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "eventhubs", "namespace", "azure_event_hub")
}

func (awc *AzureWindowsCLI) discoverServiceBusesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "servicebus", "namespace", "azure_service_bus")
}

func (awc *AzureWindowsCLI) discoverCosmosDBAccountsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "cosmosdb", "azure_cosmos_db")
}

func (awc *AzureWindowsCLI) discoverApplicationGatewaysViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "network", "application-gateway", "azure_application_gateway")
}

func (awc *AzureWindowsCLI) discoverFirewallsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "network", "firewall", "azure_firewall")
}

func (awc *AzureWindowsCLI) discoverBastionHostsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "network", "bastion", "azure_bastion")
}

func (awc *AzureWindowsCLI) discoverVPNGatewaysViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "network", "vnet-gateway", "azure_vpn_gateway")
}

func (awc *AzureWindowsCLI) discoverExpressRouteCircuitsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "network", "express-route", "azure_express_route")
}

func (awc *AzureWindowsCLI) discoverFunctionAppsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "functionapp", "azure_function_app")
}

func (awc *AzureWindowsCLI) discoverLogicAppsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "logic", "workflow", "azure_logic_app")
}

func (awc *AzureWindowsCLI) discoverAPIManagementViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "apim", "azure_api_management")
}

func (awc *AzureWindowsCLI) discoverCDNProfilesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "cdn", "profile", "azure_cdn")
}

func (awc *AzureWindowsCLI) discoverFrontDoorsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "network", "front-door", "azure_front_door")
}

func (awc *AzureWindowsCLI) discoverApplicationInsightsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "monitor", "app-insights", "azure_application_insights")
}

func (awc *AzureWindowsCLI) discoverLogAnalyticsWorkspacesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "monitor", "log-analytics", "workspace", "azure_log_analytics")
}

func (awc *AzureWindowsCLI) discoverBackupVaultsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "backup", "vault", "azure_backup_vault")
}

func (awc *AzureWindowsCLI) discoverRecoveryServicesVaultsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "backup", "recovery-services", "vault", "azure_recovery_services")
}

func (awc *AzureWindowsCLI) discoverSecurityCentersViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "security", "center", "azure_security_center")
}

func (awc *AzureWindowsCLI) discoverMonitorActionGroupsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "monitor", "action-group", "azure_monitor_action_group")
}

func (awc *AzureWindowsCLI) discoverManagedDisksViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "disk", "azure_managed_disk")
}

func (awc *AzureWindowsCLI) discoverNetworkSecurityGroupsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "network", "nsg", "azure_network_security_group")
}

func (awc *AzureWindowsCLI) discoverRouteTablesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "network", "route-table", "azure_route_table")
}

func (awc *AzureWindowsCLI) discoverPrivateEndpointsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "network", "private-endpoint", "azure_private_endpoint")
}

func (awc *AzureWindowsCLI) discoverPrivateLinkServicesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "network", "private-link-service", "azure_private_link_service")
}

func (awc *AzureWindowsCLI) discoverTrafficManagerProfilesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "network", "traffic-manager", "profile", "azure_traffic_manager")
}

func (awc *AzureWindowsCLI) discoverDNSZonesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return awc.discoverGenericResource(ctx, "network", "dns", "zone", "azure_dns_zone")
}

// Generic discovery helper for Azure CLI resources
func (awc *AzureWindowsCLI) discoverGenericResource(ctx context.Context, args ...string) ([]models.Resource, error) {
	cmd := exec.CommandContext(ctx, "az", args...)
	cmd.Args = append(cmd.Args, "list", "--output", "json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to discover %s: %w", strings.Join(args, " "), err)
	}

	var resources []map[string]interface{}
	if err := json.Unmarshal(output, &resources); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", strings.Join(args, " "), err)
	}

	var result []models.Resource
	resourceType := args[len(args)-1] // Use last argument as resource type

	for _, resource := range resources {
		model := models.Resource{
			ID:       getStringFromCLI(resource, "id"),
			Name:     getStringFromCLI(resource, "name"),
			Type:     resourceType,
			Provider: "azure",
			Region:   getStringFromCLI(resource, "location"),
			Tags:     convertTags(getMap(resource, "tags")),
		}
		result = append(result, model)
	}

	return result, nil
}

func (awc *AzureWindowsCLI) discoverContainerInstancesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverSignalRViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverCognitiveServicesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverMachineLearningWorkspacesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverDataFactoriesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverSynapseWorkspacesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverDatabricksWorkspacesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverHDInsightClustersViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverStreamAnalyticsJobsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverEventGridTopicsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverServiceFabricClustersViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverSpringCloudServicesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverOpenShiftClustersViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverBatchAccountsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverMediaServicesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverSearchServicesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverPowerBIDedicatedViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverAnalysisServicesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverDataLakeAnalyticsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverDataLakeStoreViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverHPCClustersViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverLabServicesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverDevTestLabsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverSharedImageGalleriesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverImageDefinitionsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverImageVersionsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverProximityPlacementGroupsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverAvailabilitySetsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverVMScaleSetsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverVMImagesViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverSnapshotsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverRestorePointCollectionsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverCapacityReservationsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverDedicatedHostsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverDedicatedHostGroupsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverGalleryApplicationsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

func (awc *AzureWindowsCLI) discoverGalleryApplicationVersionsViaCLI(ctx context.Context) ([]models.Resource, error) {
	return nil, fmt.Errorf("CLI-based discovery not yet implemented for this resource type")
}

// Helper functions

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

func getStringFromCLI(data map[string]interface{}, key string) string {
	if value, exists := data[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func convertCLITags(tags interface{}) map[string]string {
	result := make(map[string]string)
	if tags == nil {
		return result
	}

	if tagsMap, ok := tags.(map[string]interface{}); ok {
		for k, v := range tagsMap {
			if str, ok := v.(string); ok {
				result[k] = str
			}
		}
	}

	return result
}

func getTagValueFromCLI(tags interface{}, key string) string {
	if tags == nil {
		return ""
	}

	if tagsMap, ok := tags.(map[string]interface{}); ok {
		if value, exists := tagsMap[key]; exists {
			if str, ok := value.(string); ok {
				return str
			}
		}
	}

	return ""
}

// Helper functions for Azure CLI path and subscription
func getAzureCLIPath() string {
	// Common Azure CLI installation paths on Windows
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

	// Try to find in PATH
	if azPath, err := exec.LookPath("az"); err == nil {
		return azPath
	}

	return ""
}

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
