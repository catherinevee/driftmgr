package azure

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/catherinevee/driftmgr/internal/core/models"
)

// ComprehensiveAzureDiscoverer discovers ALL Azure resources
type ComprehensiveAzureDiscoverer struct {
	cred           azcore.TokenCredential
	subscriptionID string
	progress       chan AzureDiscoveryProgress
}

// AzureDiscoveryProgress tracks discovery progress for Azure
type AzureDiscoveryProgress struct {
	Service      string
	ResourceType string
	Count        int
	Message      string
}

// NewComprehensiveAzureDiscoverer creates a new comprehensive Azure discoverer
func NewComprehensiveAzureDiscoverer(subscriptionID string) (*ComprehensiveAzureDiscoverer, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	return &ComprehensiveAzureDiscoverer{
		cred:           cred,
		subscriptionID: subscriptionID,
		progress:       make(chan AzureDiscoveryProgress, 100),
	}, nil
}

// DiscoverAllAzureResources discovers all Azure resources
func (d *ComprehensiveAzureDiscoverer) DiscoverAllAzureResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	// Start progress reporter
	go d.reportProgress()

	var allResources []models.Resource
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Discover resources by type (Azure resources are not strictly region-based like AWS)
	resourceTypes := []func(context.Context) []models.Resource{
		d.discoverResourceGroups,
		d.discoverVirtualMachines,
		d.discoverStorageAccounts,
		d.discoverVirtualNetworks,
		d.discoverNetworkSecurityGroups,
		d.discoverLoadBalancers,
		// These require additional dependencies - can be enabled later
		// d.discoverSQLDatabases,
		// d.discoverCosmosDBAccounts,
		// d.discoverWebApps,
		// d.discoverFunctionApps,
		// d.discoverAKSClusters,
		// d.discoverKeyVaults,
		// d.discoverRedisCaches,
		d.discoverPublicIPs,
		d.discoverNetworkInterfaces,
		d.discoverDisks,
	}

	for _, discoveryFunc := range resourceTypes {
		wg.Add(1)
		go func(fn func(context.Context) []models.Resource) {
			defer wg.Done()
			resources := fn(ctx)
			mu.Lock()
			allResources = append(allResources, resources...)
			mu.Unlock()
		}(discoveryFunc)
	}

	wg.Wait()
	close(d.progress)

	log.Printf("Comprehensive Azure discovery completed: %d total resources found", len(allResources))
	return allResources, nil
}

// discoverResourceGroups discovers all resource groups
func (d *ComprehensiveAzureDiscoverer) discoverResourceGroups(ctx context.Context) []models.Resource {
	var resources []models.Resource
	client, err := armresources.NewResourceGroupsClient(d.subscriptionID, d.cred, nil)
	if err != nil {
		log.Printf("Failed to create resource groups client: %v", err)
		return resources
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get resource groups page: %v", err)
			break
		}

		for _, rg := range page.Value {
			if rg.Name != nil && rg.Location != nil {
				resources = append(resources, models.Resource{
					ID:       *rg.ID,
					Name:     *rg.Name,
					Type:     "azure_resource_group",
					Provider: "azure",
					Region:   *rg.Location,
					State:    "active",
					Tags:     convertAzureTags(rg.Tags),
					Properties: map[string]interface{}{
						"provisioning_state": safeStringPtr(rg.Properties.ProvisioningState),
					},
				})
			}
		}
	}

	d.progress <- AzureDiscoveryProgress{Service: "Resources", ResourceType: "Resource Groups", Count: len(resources)}
	return resources
}

// discoverVirtualMachines discovers all virtual machines
func (d *ComprehensiveAzureDiscoverer) discoverVirtualMachines(ctx context.Context) []models.Resource {
	var resources []models.Resource
	client, err := armcompute.NewVirtualMachinesClient(d.subscriptionID, d.cred, nil)
	if err != nil {
		log.Printf("Failed to create VM client: %v", err)
		return resources
	}

	pager := client.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get VMs page: %v", err)
			break
		}

		for _, vm := range page.Value {
			if vm.Name != nil && vm.Location != nil {
				state := "unknown"
				if vm.Properties != nil && vm.Properties.ProvisioningState != nil {
					state = *vm.Properties.ProvisioningState
				}

				properties := make(map[string]interface{})
				if vm.Properties != nil && vm.Properties.HardwareProfile != nil && vm.Properties.HardwareProfile.VMSize != nil {
					properties["vm_size"] = string(*vm.Properties.HardwareProfile.VMSize)
				}

				resources = append(resources, models.Resource{
					ID:         *vm.ID,
					Name:       *vm.Name,
					Type:       "azure_virtual_machine",
					Provider:   "azure",
					Region:     *vm.Location,
					State:      state,
					Tags:       convertAzureTags(vm.Tags),
					Properties: properties,
				})
			}
		}
	}

	d.progress <- AzureDiscoveryProgress{Service: "Compute", ResourceType: "Virtual Machines", Count: len(resources)}
	return resources
}

// discoverStorageAccounts discovers all storage accounts
func (d *ComprehensiveAzureDiscoverer) discoverStorageAccounts(ctx context.Context) []models.Resource {
	var resources []models.Resource
	client, err := armstorage.NewAccountsClient(d.subscriptionID, d.cred, nil)
	if err != nil {
		log.Printf("Failed to create storage client: %v", err)
		return resources
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get storage accounts page: %v", err)
			break
		}

		for _, account := range page.Value {
			if account.Name != nil && account.Location != nil {
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
					ID:         *account.ID,
					Name:       *account.Name,
					Type:       "azure_storage_account",
					Provider:   "azure",
					Region:     *account.Location,
					State:      "active",
					Tags:       convertAzureTags(account.Tags),
					Properties: properties,
				})
			}
		}
	}

	d.progress <- AzureDiscoveryProgress{Service: "Storage", ResourceType: "Storage Accounts", Count: len(resources)}
	return resources
}

// discoverVirtualNetworks discovers all virtual networks
func (d *ComprehensiveAzureDiscoverer) discoverVirtualNetworks(ctx context.Context) []models.Resource {
	var resources []models.Resource
	client, err := armnetwork.NewVirtualNetworksClient(d.subscriptionID, d.cred, nil)
	if err != nil {
		log.Printf("Failed to create vnet client: %v", err)
		return resources
	}

	pager := client.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get vnets page: %v", err)
			break
		}

		for _, vnet := range page.Value {
			if vnet.Name != nil && vnet.Location != nil {
				properties := make(map[string]interface{})
				if vnet.Properties != nil && vnet.Properties.AddressSpace != nil {
					properties["address_prefixes"] = vnet.Properties.AddressSpace.AddressPrefixes
				}

				resources = append(resources, models.Resource{
					ID:         *vnet.ID,
					Name:       *vnet.Name,
					Type:       "azure_virtual_network",
					Provider:   "azure",
					Region:     *vnet.Location,
					State:      "active",
					Tags:       convertAzureTags(vnet.Tags),
					Properties: properties,
				})
			}
		}
	}

	d.progress <- AzureDiscoveryProgress{Service: "Network", ResourceType: "Virtual Networks", Count: len(resources)}
	return resources
}

// discoverNetworkSecurityGroups discovers all NSGs
func (d *ComprehensiveAzureDiscoverer) discoverNetworkSecurityGroups(ctx context.Context) []models.Resource {
	var resources []models.Resource
	client, err := armnetwork.NewSecurityGroupsClient(d.subscriptionID, d.cred, nil)
	if err != nil {
		log.Printf("Failed to create NSG client: %v", err)
		return resources
	}

	pager := client.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get NSGs page: %v", err)
			break
		}

		for _, nsg := range page.Value {
			if nsg.Name != nil && nsg.Location != nil {
				properties := make(map[string]interface{})
				if nsg.Properties != nil && nsg.Properties.SecurityRules != nil {
					properties["rule_count"] = len(nsg.Properties.SecurityRules)
				}

				resources = append(resources, models.Resource{
					ID:         *nsg.ID,
					Name:       *nsg.Name,
					Type:       "azure_network_security_group",
					Provider:   "azure",
					Region:     *nsg.Location,
					State:      "active",
					Tags:       convertAzureTags(nsg.Tags),
					Properties: properties,
				})
			}
		}
	}

	d.progress <- AzureDiscoveryProgress{Service: "Network", ResourceType: "Security Groups", Count: len(resources)}
	return resources
}

// discoverLoadBalancers discovers all load balancers
func (d *ComprehensiveAzureDiscoverer) discoverLoadBalancers(ctx context.Context) []models.Resource {
	var resources []models.Resource
	client, err := armnetwork.NewLoadBalancersClient(d.subscriptionID, d.cred, nil)
	if err != nil {
		log.Printf("Failed to create LB client: %v", err)
		return resources
	}

	pager := client.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get LBs page: %v", err)
			break
		}

		for _, lb := range page.Value {
			if lb.Name != nil && lb.Location != nil {
				properties := make(map[string]interface{})
				if lb.SKU != nil && lb.SKU.Name != nil {
					properties["sku"] = string(*lb.SKU.Name)
				}

				resources = append(resources, models.Resource{
					ID:         *lb.ID,
					Name:       *lb.Name,
					Type:       "azure_load_balancer",
					Provider:   "azure",
					Region:     *lb.Location,
					State:      "active",
					Tags:       convertAzureTags(lb.Tags),
					Properties: properties,
				})
			}
		}
	}

	d.progress <- AzureDiscoveryProgress{Service: "Network", ResourceType: "Load Balancers", Count: len(resources)}
	return resources
}

/* // Commented out - requires additional dependencies
// discoverSQLDatabases discovers all SQL databases
func (d *ComprehensiveAzureDiscoverer) discoverSQLDatabases(ctx context.Context) []models.Resource {
	var resources []models.Resource

	// First get SQL servers
	serverClient, err := armsql.NewServersClient(d.subscriptionID, d.cred, nil)
	if err != nil {
		log.Printf("Failed to create SQL server client: %v", err)
		return resources
	}

	serverPager := serverClient.NewListPager(nil)
	for serverPager.More() {
		serverPage, err := serverPager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get SQL servers page: %v", err)
			break
		}

		for _, server := range serverPage.Value {
			if server.Name != nil && server.Location != nil {
				// Add server as a resource
				resources = append(resources, models.Resource{
					ID:       *server.ID,
					Name:     *server.Name,
					Type:     "azure_sql_server",
					Provider: "azure",
					Region:   *server.Location,
					State:    "active",
					Tags:     convertAzureTags(server.Tags),
					Properties: map[string]interface{}{
						"version": safeStringPtr(server.Properties.Version),
					},
				})

				// Get databases for this server
				dbClient, err := armsql.NewDatabasesClient(d.subscriptionID, d.cred, nil)
				if err != nil {
					continue
				}

				// Extract resource group from server ID
				parts := strings.Split(*server.ID, "/")
				var resourceGroup string
				for i, part := range parts {
					if part == "resourceGroups" && i+1 < len(parts) {
						resourceGroup = parts[i+1]
						break
					}
				}

				if resourceGroup != "" {
					dbPager := dbClient.NewListByServerPager(resourceGroup, *server.Name, nil)
					for dbPager.More() {
						dbPage, err := dbPager.NextPage(ctx)
						if err != nil {
							break
						}

						for _, db := range dbPage.Value {
							if db.Name != nil && *db.Name != "master" { // Skip system databases
								resources = append(resources, models.Resource{
									ID:       *db.ID,
									Name:     *db.Name,
									Type:     "azure_sql_database",
									Provider: "azure",
									Region:   *db.Location,
									State:    "active",
									Tags:     convertAzureTags(db.Tags),
									Properties: map[string]interface{}{
										"server": *server.Name,
									},
								})
							}
						}
					}
				}
			}
		}
	}

	d.progress <- AzureDiscoveryProgress{Service: "SQL", ResourceType: "SQL Databases", Count: len(resources)}
	return resources
}
*/

/* // Commented out - requires additional dependencies
// discoverCosmosDBAccounts discovers all Cosmos DB accounts
func (d *ComprehensiveAzureDiscoverer) discoverCosmosDBAccounts(ctx context.Context) []models.Resource {
	var resources []models.Resource
	client, err := armcosmos.NewDatabaseAccountsClient(d.subscriptionID, d.cred, nil)
	if err != nil {
		log.Printf("Failed to create Cosmos DB client: %v", err)
		return resources
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get Cosmos DB accounts page: %v", err)
			break
		}

		for _, account := range page.Value {
			if account.Name != nil && account.Location != nil {
				properties := make(map[string]interface{})
				if account.Properties != nil && account.Properties.DatabaseAccountOfferType != nil {
					properties["offer_type"] = string(*account.Properties.DatabaseAccountOfferType)
				}

				resources = append(resources, models.Resource{
					ID:         *account.ID,
					Name:       *account.Name,
					Type:       "azure_cosmosdb_account",
					Provider:   "azure",
					Region:     *account.Location,
					State:      "active",
					Tags:       convertAzureTags(account.Tags),
					Properties: properties,
				})
			}
		}
	}

	d.progress <- AzureDiscoveryProgress{Service: "CosmosDB", ResourceType: "Accounts", Count: len(resources)}
	return resources
}
*/

/* // Commented out - requires additional dependencies
// discoverWebApps discovers all web apps
func (d *ComprehensiveAzureDiscoverer) discoverWebApps(ctx context.Context) []models.Resource {
	var resources []models.Resource
	client, err := armweb.NewStaticAppsClient(d.subscriptionID, d.cred, nil)
	if err != nil {
		log.Printf("Failed to create web apps client: %v", err)
		return resources
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get web apps page: %v", err)
			break
		}

		for _, app := range page.Value {
			if app.Name != nil && app.Location != nil {
				resources = append(resources, models.Resource{
					ID:       *app.ID,
					Name:     *app.Name,
					Type:     "azure_web_app",
					Provider: "azure",
					Region:   *app.Location,
					State:    "active",
					Tags:     convertAzureTags(app.Tags),
				})
			}
		}
	}

	d.progress <- AzureDiscoveryProgress{Service: "Web", ResourceType: "Web Apps", Count: len(resources)}
	return resources
}
*/

/* // Commented out - requires additional dependencies
// discoverFunctionApps discovers all function apps
func (d *ComprehensiveAzureDiscoverer) discoverFunctionApps(ctx context.Context) []models.Resource {
	var resources []models.Resource
	client, err := armweb.NewStaticAppsClient(d.subscriptionID, d.cred, nil)
	if err != nil {
		log.Printf("Failed to create function apps client: %v", err)
		return resources
	}

	// Note: Using WebAppsClient for function apps as they share the same API
	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get function apps page: %v", err)
			break
		}

		for _, app := range page.Value {
			if app.Name != nil && app.Location != nil {
				// Check if it's a function app by examining properties
				if app.Properties != nil {
					resources = append(resources, models.Resource{
						ID:       *app.ID,
						Name:     *app.Name,
						Type:     "azure_function_app",
						Provider: "azure",
						Region:   *app.Location,
						State:    "active",
						Tags:     convertAzureTags(app.Tags),
					})
				}
			}
		}
	}

	d.progress <- AzureDiscoveryProgress{Service: "Functions", ResourceType: "Function Apps", Count: len(resources)}
	return resources
}
*/

/* // Commented out - requires additional dependencies
// discoverAKSClusters discovers all AKS clusters
func (d *ComprehensiveAzureDiscoverer) discoverAKSClusters(ctx context.Context) []models.Resource {
	var resources []models.Resource
	client, err := armcontainerservice.NewManagedClustersClient(d.subscriptionID, d.cred, nil)
	if err != nil {
		log.Printf("Failed to create AKS client: %v", err)
		return resources
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get AKS clusters page: %v", err)
			break
		}

		for _, cluster := range page.Value {
			if cluster.Name != nil && cluster.Location != nil {
				properties := make(map[string]interface{})
				if cluster.Properties != nil {
					if cluster.Properties.KubernetesVersion != nil {
						properties["k8s_version"] = *cluster.Properties.KubernetesVersion
					}
					if cluster.Properties.AgentPoolProfiles != nil {
						properties["node_count"] = len(cluster.Properties.AgentPoolProfiles)
					}
				}

				resources = append(resources, models.Resource{
					ID:         *cluster.ID,
					Name:       *cluster.Name,
					Type:       "azure_aks_cluster",
					Provider:   "azure",
					Region:     *cluster.Location,
					State:      "active",
					Tags:       convertAzureTags(cluster.Tags),
					Properties: properties,
				})
			}
		}
	}

	d.progress <- AzureDiscoveryProgress{Service: "AKS", ResourceType: "Clusters", Count: len(resources)}
	return resources
}
*/

/* // Commented out - requires additional dependencies
// discoverKeyVaults discovers all key vaults
func (d *ComprehensiveAzureDiscoverer) discoverKeyVaults(ctx context.Context) []models.Resource {
	var resources []models.Resource
	client, err := armkeyvault.NewVaultsClient(d.subscriptionID, d.cred, nil)
	if err != nil {
		log.Printf("Failed to create Key Vault client: %v", err)
		return resources
	}

	pager := client.NewListBySubscriptionPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get key vaults page: %v", err)
			break
		}

		for _, vault := range page.Value {
			if vault.Name != nil && vault.Location != nil {
				properties := make(map[string]interface{})
				if vault.Properties != nil && vault.Properties.SKU != nil && vault.Properties.SKU.Name != nil {
					properties["sku"] = string(*vault.Properties.SKU.Name)
				}

				resources = append(resources, models.Resource{
					ID:         *vault.ID,
					Name:       *vault.Name,
					Type:       "azure_key_vault",
					Provider:   "azure",
					Region:     *vault.Location,
					State:      "active",
					Tags:       convertAzureTags(vault.Tags),
					Properties: properties,
				})
			}
		}
	}

	d.progress <- AzureDiscoveryProgress{Service: "KeyVault", ResourceType: "Vaults", Count: len(resources)}
	return resources
}
*/

/* // Commented out - requires additional dependencies
// discoverRedisCaches discovers all Redis caches
func (d *ComprehensiveAzureDiscoverer) discoverRedisCaches(ctx context.Context) []models.Resource {
	var resources []models.Resource
	client, err := armredis.NewClient(d.subscriptionID, d.cred, nil)
	if err != nil {
		log.Printf("Failed to create Redis client: %v", err)
		return resources
	}

	pager := client.NewListBySubscriptionPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get Redis caches page: %v", err)
			break
		}

		for _, cache := range page.Value {
			if cache.Name != nil && cache.Location != nil {
				properties := make(map[string]interface{})
				if cache.Properties != nil && cache.Properties.SKU != nil {
					if cache.Properties.SKU.Name != nil {
						properties["sku"] = string(*cache.Properties.SKU.Name)
					}
					if cache.Properties.SKU.Capacity != nil {
						properties["capacity"] = *cache.Properties.SKU.Capacity
					}
				}

				resources = append(resources, models.Resource{
					ID:         *cache.ID,
					Name:       *cache.Name,
					Type:       "azure_redis_cache",
					Provider:   "azure",
					Region:     *cache.Location,
					State:      "active",
					Tags:       convertAzureTags(cache.Tags),
					Properties: properties,
				})
			}
		}
	}

	d.progress <- AzureDiscoveryProgress{Service: "Redis", ResourceType: "Caches", Count: len(resources)}
	return resources
}
*/

// discoverPublicIPs discovers all public IP addresses
func (d *ComprehensiveAzureDiscoverer) discoverPublicIPs(ctx context.Context) []models.Resource {
	var resources []models.Resource
	client, err := armnetwork.NewPublicIPAddressesClient(d.subscriptionID, d.cred, nil)
	if err != nil {
		log.Printf("Failed to create public IP client: %v", err)
		return resources
	}

	pager := client.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get public IPs page: %v", err)
			break
		}

		for _, ip := range page.Value {
			if ip.Name != nil && ip.Location != nil {
				properties := make(map[string]interface{})
				if ip.Properties != nil {
					if ip.Properties.IPAddress != nil {
						properties["ip_address"] = *ip.Properties.IPAddress
					}
					if ip.Properties.PublicIPAllocationMethod != nil {
						properties["allocation"] = string(*ip.Properties.PublicIPAllocationMethod)
					}
				}

				resources = append(resources, models.Resource{
					ID:         *ip.ID,
					Name:       *ip.Name,
					Type:       "azure_public_ip",
					Provider:   "azure",
					Region:     *ip.Location,
					State:      "active",
					Tags:       convertAzureTags(ip.Tags),
					Properties: properties,
				})
			}
		}
	}

	d.progress <- AzureDiscoveryProgress{Service: "Network", ResourceType: "Public IPs", Count: len(resources)}
	return resources
}

// discoverNetworkInterfaces discovers all network interfaces
func (d *ComprehensiveAzureDiscoverer) discoverNetworkInterfaces(ctx context.Context) []models.Resource {
	var resources []models.Resource
	client, err := armnetwork.NewInterfacesClient(d.subscriptionID, d.cred, nil)
	if err != nil {
		log.Printf("Failed to create network interface client: %v", err)
		return resources
	}

	pager := client.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get network interfaces page: %v", err)
			break
		}

		for _, nic := range page.Value {
			if nic.Name != nil && nic.Location != nil {
				resources = append(resources, models.Resource{
					ID:       *nic.ID,
					Name:     *nic.Name,
					Type:     "azure_network_interface",
					Provider: "azure",
					Region:   *nic.Location,
					State:    "active",
					Tags:     convertAzureTags(nic.Tags),
				})
			}
		}
	}

	d.progress <- AzureDiscoveryProgress{Service: "Network", ResourceType: "Network Interfaces", Count: len(resources)}
	return resources
}

// discoverDisks discovers all managed disks
func (d *ComprehensiveAzureDiscoverer) discoverDisks(ctx context.Context) []models.Resource {
	var resources []models.Resource
	client, err := armcompute.NewDisksClient(d.subscriptionID, d.cred, nil)
	if err != nil {
		log.Printf("Failed to create disks client: %v", err)
		return resources
	}

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to get disks page: %v", err)
			break
		}

		for _, disk := range page.Value {
			if disk.Name != nil && disk.Location != nil {
				properties := make(map[string]interface{})
				if disk.Properties != nil {
					if disk.Properties.DiskSizeGB != nil {
						properties["size_gb"] = *disk.Properties.DiskSizeGB
					}
					if disk.Properties.DiskState != nil {
						properties["state"] = string(*disk.Properties.DiskState)
					}
				}

				resources = append(resources, models.Resource{
					ID:         *disk.ID,
					Name:       *disk.Name,
					Type:       "azure_managed_disk",
					Provider:   "azure",
					Region:     *disk.Location,
					State:      "active",
					Tags:       convertAzureTags(disk.Tags),
					Properties: properties,
				})
			}
		}
	}

	d.progress <- AzureDiscoveryProgress{Service: "Compute", ResourceType: "Managed Disks", Count: len(resources)}
	return resources
}

// Helper functions
func (d *ComprehensiveAzureDiscoverer) reportProgress() {
	for progress := range d.progress {
		log.Printf("[Azure] %s: Discovered %d %s", progress.Service, progress.Count, progress.ResourceType)
	}
}

func convertAzureTags(tags map[string]*string) map[string]string {
	result := make(map[string]string)
	for k, v := range tags {
		if v != nil {
			result[k] = *v
		}
	}
	return result
}

func safeStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
