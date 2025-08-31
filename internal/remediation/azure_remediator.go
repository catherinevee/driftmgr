package remediation

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/catherinevee/driftmgr/internal/core/models"
)

// AzureRemediator handles Azure-specific remediation
type AzureRemediator struct {
	cred               azcore.TokenCredential
	subscriptionID     string
	computeClient      *armcompute.VirtualMachinesClient
	networkClient      *armnetwork.SecurityGroupsClient
	vnetClient         *armnetwork.VirtualNetworksClient
	subnetClient       *armnetwork.SubnetsClient
	storageClient      *armstorage.AccountsClient
	sqlClient          *armsql.DatabasesClient
	resourceClient     *armresources.Client
}

// NewAzureRemediator creates a new Azure remediator
func NewAzureRemediator(ctx context.Context, subscriptionID string) (*AzureRemediator, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	computeClient, err := armcompute.NewVirtualMachinesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute client: %w", err)
	}

	networkClient, err := armnetwork.NewSecurityGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create network client: %w", err)
	}

	vnetClient, err := armnetwork.NewVirtualNetworksClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create vnet client: %w", err)
	}

	subnetClient, err := armnetwork.NewSubnetsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create subnet client: %w", err)
	}

	storageClient, err := armstorage.NewAccountsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	sqlClient, err := armsql.NewDatabasesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create SQL client: %w", err)
	}

	resourceClient, err := armresources.NewClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource client: %w", err)
	}

	return &AzureRemediator{
		cred:               cred,
		subscriptionID:     subscriptionID,
		computeClient:      computeClient,
		networkClient:      networkClient,
		vnetClient:         vnetClient,
		subnetClient:       subnetClient,
		storageClient:      storageClient,
		sqlClient:          sqlClient,
		resourceClient:     resourceClient,
	}, nil
}

// Remediate performs Azure resource remediation
func (r *AzureRemediator) Remediate(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	switch drift.ResourceType {
	case "azurerm_virtual_machine", "Microsoft.Compute/virtualMachines":
		return r.remediateVirtualMachine(ctx, drift, action)
	case "azurerm_network_security_group", "Microsoft.Network/networkSecurityGroups":
		return r.remediateNetworkSecurityGroup(ctx, drift, action)
	case "azurerm_virtual_network", "Microsoft.Network/virtualNetworks":
		return r.remediateVirtualNetwork(ctx, drift, action)
	case "azurerm_subnet", "Microsoft.Network/virtualNetworks/subnets":
		return r.remediateSubnet(ctx, drift, action)
	case "azurerm_storage_account", "Microsoft.Storage/storageAccounts":
		return r.remediateStorageAccount(ctx, drift, action)
	case "azurerm_sql_database", "Microsoft.Sql/servers/databases":
		return r.remediateSQLDatabase(ctx, drift, action)
	default:
		return fmt.Errorf("remediation not implemented for resource type: %s", drift.ResourceType)
	}
}

// remediateVirtualMachine handles VM remediation
func (r *AzureRemediator) remediateVirtualMachine(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	// Parse resource ID to get resource group and VM name
	resourceGroup, vmName := r.parseResourceID(drift.ResourceID)

	switch action.Action {
	case "update":
		// Get current VM
		vm, err := r.computeClient.Get(ctx, resourceGroup, vmName, nil)
		if err != nil {
			return fmt.Errorf("failed to get VM: %w", err)
		}

		// Update VM size if changed
		if vmSize, ok := action.Parameters["vm_size"].(string); ok {
			vm.Properties.HardwareProfile.VMSize = to.Ptr(armcompute.VirtualMachineSizeTypes(vmSize))
		}

		// Update tags
		if tags, ok := action.Parameters["tags"].(map[string]interface{}); ok {
			vm.Tags = make(map[string]*string)
			for key, value := range tags {
				vm.Tags[key] = to.Ptr(fmt.Sprintf("%v", value))
			}
		}

		// Apply updates
		poller, err := r.computeClient.BeginCreateOrUpdate(ctx, resourceGroup, vmName, vm.VirtualMachine, nil)
		if err != nil {
			return fmt.Errorf("failed to update VM: %w", err)
		}

		_, err = poller.PollUntilDone(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to complete VM update: %w", err)
		}

		return nil

	case "delete":
		// Delete the VM
		poller, err := r.computeClient.BeginDelete(ctx, resourceGroup, vmName, nil)
		if err != nil {
			return fmt.Errorf("failed to delete VM: %w", err)
		}

		_, err = poller.PollUntilDone(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to complete VM deletion: %w", err)
		}

		return nil

	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}

// remediateNetworkSecurityGroup handles NSG remediation
func (r *AzureRemediator) remediateNetworkSecurityGroup(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	resourceGroup, nsgName := r.parseResourceID(drift.ResourceID)

	switch action.Action {
	case "update":
		// Get current NSG
		nsg, err := r.networkClient.Get(ctx, resourceGroup, nsgName, nil)
		if err != nil {
			return fmt.Errorf("failed to get NSG: %w", err)
		}

		// Update security rules
		if rules, ok := action.Parameters["security_rules"].([]interface{}); ok {
			var securityRules []*armnetwork.SecurityRule
			for i, rule := range rules {
				if ruleMap, ok := rule.(map[string]interface{}); ok {
					secRule := &armnetwork.SecurityRule{
						Name: to.Ptr(fmt.Sprintf("rule_%d", i)),
						Properties: &armnetwork.SecurityRulePropertiesFormat{
							Priority:                 to.Ptr(int32(100 + i*10)),
							Direction:                to.Ptr(armnetwork.SecurityRuleDirectionInbound),
							Access:                   to.Ptr(armnetwork.SecurityRuleAccessAllow),
							Protocol:                 to.Ptr(armnetwork.SecurityRuleProtocolTCP),
							SourcePortRange:          to.Ptr("*"),
							DestinationPortRange:     to.Ptr("*"),
							SourceAddressPrefix:      to.Ptr("*"),
							DestinationAddressPrefix: to.Ptr("*"),
						},
					}

					if direction, ok := ruleMap["direction"].(string); ok {
						secRule.Properties.Direction = to.Ptr(armnetwork.SecurityRuleDirection(direction))
					}
					if access, ok := ruleMap["access"].(string); ok {
						secRule.Properties.Access = to.Ptr(armnetwork.SecurityRuleAccess(access))
					}
					if protocol, ok := ruleMap["protocol"].(string); ok {
						secRule.Properties.Protocol = to.Ptr(armnetwork.SecurityRuleProtocol(protocol))
					}
					if sourcePort, ok := ruleMap["source_port_range"].(string); ok {
						secRule.Properties.SourcePortRange = to.Ptr(sourcePort)
					}
					if destPort, ok := ruleMap["destination_port_range"].(string); ok {
						secRule.Properties.DestinationPortRange = to.Ptr(destPort)
					}

					securityRules = append(securityRules, secRule)
				}
			}
			nsg.Properties.SecurityRules = securityRules
		}

		// Update tags
		if tags, ok := action.Parameters["tags"].(map[string]interface{}); ok {
			nsg.Tags = make(map[string]*string)
			for key, value := range tags {
				nsg.Tags[key] = to.Ptr(fmt.Sprintf("%v", value))
			}
		}

		// Apply updates
		poller, err := r.networkClient.BeginCreateOrUpdate(ctx, resourceGroup, nsgName, nsg.SecurityGroup, nil)
		if err != nil {
			return fmt.Errorf("failed to update NSG: %w", err)
		}

		_, err = poller.PollUntilDone(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to complete NSG update: %w", err)
		}

		return nil

	case "delete":
		// Delete the NSG
		poller, err := r.networkClient.BeginDelete(ctx, resourceGroup, nsgName, nil)
		if err != nil {
			return fmt.Errorf("failed to delete NSG: %w", err)
		}

		_, err = poller.PollUntilDone(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to complete NSG deletion: %w", err)
		}

		return nil

	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}

// remediateVirtualNetwork handles VNet remediation
func (r *AzureRemediator) remediateVirtualNetwork(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	resourceGroup, vnetName := r.parseResourceID(drift.ResourceID)

	switch action.Action {
	case "update":
		// Get current VNet
		vnet, err := r.vnetClient.Get(ctx, resourceGroup, vnetName, nil)
		if err != nil {
			return fmt.Errorf("failed to get VNet: %w", err)
		}

		// Update address space
		if addressSpace, ok := action.Parameters["address_space"].([]interface{}); ok {
			var addressPrefixes []string
			for _, prefix := range addressSpace {
				addressPrefixes = append(addressPrefixes, fmt.Sprintf("%v", prefix))
			}
			vnet.Properties.AddressSpace = &armnetwork.AddressSpace{
				AddressPrefixes: to.SliceOfPtrs(addressPrefixes...),
			}
		}

		// Update DNS servers
		if dnsServers, ok := action.Parameters["dns_servers"].([]interface{}); ok {
			var servers []string
			for _, server := range dnsServers {
				servers = append(servers, fmt.Sprintf("%v", server))
			}
			if vnet.Properties.DhcpOptions == nil {
				vnet.Properties.DhcpOptions = &armnetwork.DhcpOptions{}
			}
			vnet.Properties.DhcpOptions.DNSServers = to.SliceOfPtrs(servers...)
		}

		// Update tags
		if tags, ok := action.Parameters["tags"].(map[string]interface{}); ok {
			vnet.Tags = make(map[string]*string)
			for key, value := range tags {
				vnet.Tags[key] = to.Ptr(fmt.Sprintf("%v", value))
			}
		}

		// Apply updates
		poller, err := r.vnetClient.BeginCreateOrUpdate(ctx, resourceGroup, vnetName, vnet.VirtualNetwork, nil)
		if err != nil {
			return fmt.Errorf("failed to update VNet: %w", err)
		}

		_, err = poller.PollUntilDone(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to complete VNet update: %w", err)
		}

		return nil

	case "delete":
		// Delete the VNet
		poller, err := r.vnetClient.BeginDelete(ctx, resourceGroup, vnetName, nil)
		if err != nil {
			return fmt.Errorf("failed to delete VNet: %w", err)
		}

		_, err = poller.PollUntilDone(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to complete VNet deletion: %w", err)
		}

		return nil

	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}

// remediateSubnet handles subnet remediation
func (r *AzureRemediator) remediateSubnet(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	// Parse subnet resource ID
	parts := strings.Split(drift.ResourceID, "/")
	if len(parts) < 10 {
		return fmt.Errorf("invalid subnet resource ID")
	}
	
	resourceGroup := ""
	vnetName := ""
	subnetName := ""
	
	for i, part := range parts {
		if part == "resourceGroups" && i+1 < len(parts) {
			resourceGroup = parts[i+1]
		}
		if part == "virtualNetworks" && i+1 < len(parts) {
			vnetName = parts[i+1]
		}
		if part == "subnets" && i+1 < len(parts) {
			subnetName = parts[i+1]
		}
	}

	switch action.Action {
	case "update":
		// Get current subnet
		subnet, err := r.subnetClient.Get(ctx, resourceGroup, vnetName, subnetName, nil)
		if err != nil {
			return fmt.Errorf("failed to get subnet: %w", err)
		}

		// Update address prefix
		if addressPrefix, ok := action.Parameters["address_prefix"].(string); ok {
			subnet.Properties.AddressPrefix = to.Ptr(addressPrefix)
		}

		// Update NSG association
		if nsgID, ok := action.Parameters["network_security_group_id"].(string); ok {
			subnet.Properties.NetworkSecurityGroup = &armnetwork.SecurityGroup{
				ID: to.Ptr(nsgID),
			}
		}

		// Apply updates
		poller, err := r.subnetClient.BeginCreateOrUpdate(ctx, resourceGroup, vnetName, subnetName, subnet.Subnet, nil)
		if err != nil {
			return fmt.Errorf("failed to update subnet: %w", err)
		}

		_, err = poller.PollUntilDone(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to complete subnet update: %w", err)
		}

		return nil

	case "delete":
		// Delete the subnet
		poller, err := r.subnetClient.BeginDelete(ctx, resourceGroup, vnetName, subnetName, nil)
		if err != nil {
			return fmt.Errorf("failed to delete subnet: %w", err)
		}

		_, err = poller.PollUntilDone(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to complete subnet deletion: %w", err)
		}

		return nil

	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}

// remediateStorageAccount handles storage account remediation
func (r *AzureRemediator) remediateStorageAccount(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	resourceGroup, accountName := r.parseResourceID(drift.ResourceID)

	switch action.Action {
	case "update":
		// Get current storage account
		_, err := r.storageClient.GetProperties(ctx, resourceGroup, accountName, nil)
		if err != nil {
			return fmt.Errorf("failed to get storage account: %w", err)
		}

		updateParams := armstorage.AccountUpdateParameters{
			Properties: &armstorage.AccountPropertiesUpdateParameters{},
		}

		// Update encryption
		if encryption, ok := action.Parameters["encryption"].(map[string]interface{}); ok {
			if enabled, ok := encryption["enabled"].(bool); ok && enabled {
				updateParams.Properties.Encryption = &armstorage.Encryption{
					Services: &armstorage.EncryptionServices{
						Blob: &armstorage.EncryptionService{
							Enabled: to.Ptr(true),
						},
						File: &armstorage.EncryptionService{
							Enabled: to.Ptr(true),
						},
					},
					KeySource: to.Ptr(armstorage.KeySourceMicrosoftStorage),
				}
			}
		}

		// Update access tier
		if accessTier, ok := action.Parameters["access_tier"].(string); ok {
			updateParams.Properties.AccessTier = to.Ptr(armstorage.AccessTier(accessTier))
		}

		// Update tags
		if tags, ok := action.Parameters["tags"].(map[string]interface{}); ok {
			updateParams.Tags = make(map[string]*string)
			for key, value := range tags {
				updateParams.Tags[key] = to.Ptr(fmt.Sprintf("%v", value))
			}
		}

		// Apply updates
		_, err = r.storageClient.Update(ctx, resourceGroup, accountName, updateParams, nil)
		if err != nil {
			return fmt.Errorf("failed to update storage account: %w", err)
		}

		return nil

	case "delete":
		// Delete the storage account
		_, err := r.storageClient.Delete(ctx, resourceGroup, accountName, nil)
		if err != nil {
			return fmt.Errorf("failed to delete storage account: %w", err)
		}

		return nil

	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}

// remediateSQLDatabase handles SQL database remediation
func (r *AzureRemediator) remediateSQLDatabase(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	// Parse SQL database resource ID
	parts := strings.Split(drift.ResourceID, "/")
	if len(parts) < 12 {
		return fmt.Errorf("invalid SQL database resource ID")
	}
	
	resourceGroup := ""
	serverName := ""
	databaseName := ""
	
	for i, part := range parts {
		if part == "resourceGroups" && i+1 < len(parts) {
			resourceGroup = parts[i+1]
		}
		if part == "servers" && i+1 < len(parts) {
			serverName = parts[i+1]
		}
		if part == "databases" && i+1 < len(parts) {
			databaseName = parts[i+1]
		}
	}

	switch action.Action {
	case "update":
		// Get current database
		_, err := r.sqlClient.Get(ctx, resourceGroup, serverName, databaseName, nil)
		if err != nil {
			return fmt.Errorf("failed to get SQL database: %w", err)
		}

		updateParams := armsql.DatabaseUpdate{
			Properties: &armsql.DatabaseUpdateProperties{},
		}

		// Update SKU
		if sku, ok := action.Parameters["sku"].(map[string]interface{}); ok {
			updateParams.SKU = &armsql.SKU{}
			if name, ok := sku["name"].(string); ok {
				updateParams.SKU.Name = to.Ptr(name)
			}
			if tier, ok := sku["tier"].(string); ok {
				updateParams.SKU.Tier = to.Ptr(tier)
			}
		}

		// Update max size
		if maxSize, ok := action.Parameters["max_size_bytes"].(float64); ok {
			updateParams.Properties.MaxSizeBytes = to.Ptr(int64(maxSize))
		}

		// Update tags
		if tags, ok := action.Parameters["tags"].(map[string]interface{}); ok {
			updateParams.Tags = make(map[string]*string)
			for key, value := range tags {
				updateParams.Tags[key] = to.Ptr(fmt.Sprintf("%v", value))
			}
		}

		// Apply updates
		poller, err := r.sqlClient.BeginUpdate(ctx, resourceGroup, serverName, databaseName, updateParams, nil)
		if err != nil {
			return fmt.Errorf("failed to update SQL database: %w", err)
		}

		_, err = poller.PollUntilDone(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to complete SQL database update: %w", err)
		}

		return nil

	case "delete":
		// Delete the database
		poller, err := r.sqlClient.BeginDelete(ctx, resourceGroup, serverName, databaseName, nil)
		if err != nil {
			return fmt.Errorf("failed to delete SQL database: %w", err)
		}

		_, err = poller.PollUntilDone(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to complete SQL database deletion: %w", err)
		}

		return nil

	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}

// parseResourceID extracts resource group and resource name from Azure resource ID
func (r *AzureRemediator) parseResourceID(resourceID string) (string, string) {
	parts := strings.Split(resourceID, "/")
	resourceGroup := ""
	resourceName := ""
	
	for i, part := range parts {
		if part == "resourceGroups" && i+1 < len(parts) {
			resourceGroup = parts[i+1]
		}
	}
	
	// The resource name is typically the last part
	if len(parts) > 0 {
		resourceName = parts[len(parts)-1]
	}
	
	return resourceGroup, resourceName
}