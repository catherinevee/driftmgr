package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// AzureProviderComplete implements CloudProvider for Azure using direct API calls
type AzureProviderComplete struct {
	subscriptionID string
	resourceGroup  string
	tenantID       string
	clientID       string
	clientSecret   string
	accessToken    string
	httpClient     *http.Client
	baseURL        string
	apiVersion     map[string]string
}

// AzureTokenResponse represents the OAuth token response
type AzureTokenResponse struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    string `json:"expires_in"`
	ExtExpiresIn string `json:"ext_expires_in"`
	AccessToken  string `json:"access_token"`
}

// NewAzureProviderComplete creates a new Azure provider with full implementation
func NewAzureProviderComplete(subscriptionID, resourceGroup string) *AzureProviderComplete {
	return &AzureProviderComplete{
		subscriptionID: subscriptionID,
		resourceGroup:  resourceGroup,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://management.azure.com",
		apiVersion: map[string]string{
			"Microsoft.Compute/virtualMachines":     "2023-03-01",
			"Microsoft.Network/virtualNetworks":     "2023-04-01",
			"Microsoft.Network/networkInterfaces":   "2023-04-01",
			"Microsoft.Network/networkSecurityGroups": "2023-04-01",
			"Microsoft.Storage/storageAccounts":     "2023-01-01",
			"Microsoft.Sql/servers":                 "2022-05-01-preview",
			"Microsoft.Sql/servers/databases":       "2022-05-01-preview",
			"Microsoft.KeyVault/vaults":             "2023-02-01",
			"Microsoft.Web/sites":                   "2023-01-01",
			"Microsoft.ContainerRegistry/registries": "2023-01-01-preview",
			"Microsoft.ContainerService/managedClusters": "2023-05-01",
		},
	}
}

// Name returns the provider name
func (p *AzureProviderComplete) Name() string {
	return "azure"
}

// Connect establishes connection to Azure
func (p *AzureProviderComplete) Connect(ctx context.Context) error {
	// Get credentials from environment or managed identity
	p.tenantID = os.Getenv("AZURE_TENANT_ID")
	p.clientID = os.Getenv("AZURE_CLIENT_ID")
	p.clientSecret = os.Getenv("AZURE_CLIENT_SECRET")

	if p.tenantID == "" || p.clientID == "" || p.clientSecret == "" {
		// Try managed identity
		return p.authenticateWithManagedIdentity(ctx)
	}

	// Authenticate with service principal
	return p.authenticateWithServicePrincipal(ctx)
}

// authenticateWithServicePrincipal gets access token using service principal
func (p *AzureProviderComplete) authenticateWithServicePrincipal(ctx context.Context) error {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", p.tenantID)

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("scope", "https://management.azure.com/.default")

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed: %s", body)
	}

	var tokenResp AzureTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	p.accessToken = tokenResp.AccessToken
	return nil
}

// authenticateWithManagedIdentity gets access token using managed identity
func (p *AzureProviderComplete) authenticateWithManagedIdentity(ctx context.Context) error {
	// Azure Instance Metadata Service endpoint
	tokenURL := "http://169.254.169.254/metadata/identity/oauth2/token"
	
	params := url.Values{}
	params.Set("api-version", "2018-02-01")
	params.Set("resource", "https://management.azure.com/")

	req, err := http.NewRequestWithContext(ctx, "GET", tokenURL+"?"+params.Encode(), nil)
	if err != nil {
		return fmt.Errorf("failed to create MI token request: %w", err)
	}

	req.Header.Set("Metadata", "true")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get MI access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("managed identity authentication failed: %s", body)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresOn   string `json:"expires_on"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode MI token response: %w", err)
	}

	p.accessToken = tokenResp.AccessToken
	return nil
}

// makeAPIRequest makes an authenticated request to Azure API
func (p *AzureProviderComplete) makeAPIRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	url := p.baseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, respBody)
	}

	return respBody, nil
}

// DiscoverResources discovers resources in the specified region (implements CloudProvider interface)
func (p *AzureProviderComplete) DiscoverResources(ctx context.Context, region string) ([]models.Resource, error) {
	// Azure doesn't use regions in the same way as AWS, but we can use resource groups
	// For now, return empty list or implement actual discovery logic
	resources := []models.Resource{}
	
	// TODO: Implement actual resource discovery
	// This would involve listing various resource types from Azure
	
	return resources, nil
}

// GetResource retrieves a specific resource by ID (implements CloudProvider interface)
func (p *AzureProviderComplete) GetResource(ctx context.Context, resourceID string) (*models.Resource, error) {
	// Try to determine resource type from ID
	// Azure resource IDs typically contain the resource type in the path
	if strings.Contains(resourceID, "/virtualMachines/") {
		return p.GetResourceByType(ctx, "azurerm_virtual_machine", resourceID)
	} else if strings.Contains(resourceID, "/virtualNetworks/") {
		return p.GetResourceByType(ctx, "azurerm_virtual_network", resourceID)
	} else if strings.Contains(resourceID, "/networkSecurityGroups/") {
		return p.GetResourceByType(ctx, "azurerm_network_security_group", resourceID)
	} else if strings.Contains(resourceID, "/storageAccounts/") {
		return p.GetResourceByType(ctx, "azurerm_storage_account", resourceID)
	} else if strings.Contains(resourceID, "/servers/") && strings.Contains(resourceID, "Microsoft.Sql") {
		return p.GetResourceByType(ctx, "azurerm_sql_server", resourceID)
	} else if strings.Contains(resourceID, "/databases/") {
		return p.GetResourceByType(ctx, "azurerm_sql_database", resourceID)
	} else if strings.Contains(resourceID, "/vaults/") {
		return p.GetResourceByType(ctx, "azurerm_key_vault", resourceID)
	} else if strings.Contains(resourceID, "/sites/") {
		return p.GetResourceByType(ctx, "azurerm_app_service", resourceID)
	} else if strings.Contains(resourceID, "/registries/") {
		return p.GetResourceByType(ctx, "azurerm_container_registry", resourceID)
	} else if strings.Contains(resourceID, "/managedClusters/") {
		return p.GetResourceByType(ctx, "azurerm_kubernetes_cluster", resourceID)
	}
	
	// Extract the last part of the ID as resource name and try different types
	parts := strings.Split(resourceID, "/")
	if len(parts) > 0 {
		resourceName := parts[len(parts)-1]
		// Try common resource types
		if res, err := p.GetResourceByType(ctx, "azurerm_virtual_machine", resourceName); err == nil {
			return res, nil
		}
	}
	
	return nil, fmt.Errorf("unable to determine resource type for ID: %s", resourceID)
}

// ValidateCredentials checks if the provider credentials are valid (implements CloudProvider interface)
func (p *AzureProviderComplete) ValidateCredentials(ctx context.Context) error {
	return p.Connect(ctx)
}

// ListRegions returns available regions for the provider (implements CloudProvider interface)
func (p *AzureProviderComplete) ListRegions(ctx context.Context) ([]string, error) {
	// Return common Azure regions
	return []string{
		"eastus", "eastus2", "westus", "westus2", "centralus",
		"northeurope", "westeurope", "uksouth", "ukwest",
		"eastasia", "southeastasia", "japaneast", "japanwest",
		"australiaeast", "australiasoutheast", "canadacentral", "canadaeast",
	}, nil
}

// SupportedResourceTypes returns the list of supported resource types (implements CloudProvider interface)
func (p *AzureProviderComplete) SupportedResourceTypes() []string {
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
	}
}

// GetResourceByType retrieves a specific resource from Azure by type
func (p *AzureProviderComplete) GetResourceByType(ctx context.Context, resourceType string, resourceID string) (*models.Resource, error) {
	switch resourceType {
	case "azurerm_virtual_machine":
		return p.getVirtualMachine(ctx, resourceID)
	case "azurerm_virtual_network":
		return p.getVirtualNetwork(ctx, resourceID)
	case "azurerm_subnet":
		return p.getSubnet(ctx, resourceID)
	case "azurerm_network_security_group":
		return p.getNetworkSecurityGroup(ctx, resourceID)
	case "azurerm_storage_account":
		return p.getStorageAccount(ctx, resourceID)
	case "azurerm_sql_server":
		return p.getSQLServer(ctx, resourceID)
	case "azurerm_sql_database":
		return p.getSQLDatabase(ctx, resourceID)
	case "azurerm_key_vault":
		return p.getKeyVault(ctx, resourceID)
	case "azurerm_app_service":
		return p.getAppService(ctx, resourceID)
	case "azurerm_container_registry":
		return p.getContainerRegistry(ctx, resourceID)
	case "azurerm_kubernetes_cluster":
		return p.getKubernetesCluster(ctx, resourceID)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// getVirtualMachine retrieves a virtual machine
func (p *AzureProviderComplete) getVirtualMachine(ctx context.Context, vmName string) (*models.Resource, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s",
		p.subscriptionID, p.resourceGroup, vmName)
	
	apiVersion := p.apiVersion["Microsoft.Compute/virtualMachines"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM: %w", err)
	}

	var vm map[string]interface{}
	if err := json.Unmarshal(data, &vm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal VM: %w", err)
	}

	properties := vm["properties"].(map[string]interface{})
	
	return &models.Resource{
		ID:   vmName,
		Type: "azurerm_virtual_machine",
		Attributes: map[string]interface{}{
			"name":              vm["name"],
			"location":          vm["location"],
			"vm_size":           properties["hardwareProfile"].(map[string]interface{})["vmSize"],
			"tags":              vm["tags"],
			"zones":             vm["zones"],
			"provisioning_state": properties["provisioningState"],
		},
	}, nil
}

// getVirtualNetwork retrieves a virtual network
func (p *AzureProviderComplete) getVirtualNetwork(ctx context.Context, vnetName string) (*models.Resource, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s",
		p.subscriptionID, p.resourceGroup, vnetName)
	
	apiVersion := p.apiVersion["Microsoft.Network/virtualNetworks"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get VNet: %w", err)
	}

	var vnet map[string]interface{}
	if err := json.Unmarshal(data, &vnet); err != nil {
		return nil, fmt.Errorf("failed to unmarshal VNet: %w", err)
	}

	properties := vnet["properties"].(map[string]interface{})
	addressSpace := properties["addressSpace"].(map[string]interface{})
	
	return &models.Resource{
		ID:   vnetName,
		Type: "azurerm_virtual_network",
		Attributes: map[string]interface{}{
			"name":               vnet["name"],
			"location":           vnet["location"],
			"address_space":      addressSpace["addressPrefixes"],
			"tags":               vnet["tags"],
			"provisioning_state": properties["provisioningState"],
		},
	}, nil
}

// getSubnet retrieves a subnet
func (p *AzureProviderComplete) getSubnet(ctx context.Context, subnetID string) (*models.Resource, error) {
	// Parse subnet ID to extract vnet and subnet names
	parts := strings.Split(subnetID, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid subnet ID format")
	}
	vnetName := parts[0]
	subnetName := parts[1]

	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s",
		p.subscriptionID, p.resourceGroup, vnetName, subnetName)
	
	apiVersion := p.apiVersion["Microsoft.Network/virtualNetworks"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get subnet: %w", err)
	}

	var subnet map[string]interface{}
	if err := json.Unmarshal(data, &subnet); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subnet: %w", err)
	}

	properties := subnet["properties"].(map[string]interface{})
	
	return &models.Resource{
		ID:   subnetID,
		Type: "azurerm_subnet",
		Attributes: map[string]interface{}{
			"name":               subnet["name"],
			"address_prefixes":   []string{properties["addressPrefix"].(string)},
			"virtual_network_name": vnetName,
			"provisioning_state": properties["provisioningState"],
		},
	}, nil
}

// getNetworkSecurityGroup retrieves a network security group
func (p *AzureProviderComplete) getNetworkSecurityGroup(ctx context.Context, nsgName string) (*models.Resource, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkSecurityGroups/%s",
		p.subscriptionID, p.resourceGroup, nsgName)
	
	apiVersion := p.apiVersion["Microsoft.Network/networkSecurityGroups"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get NSG: %w", err)
	}

	var nsg map[string]interface{}
	if err := json.Unmarshal(data, &nsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal NSG: %w", err)
	}

	properties := nsg["properties"].(map[string]interface{})
	
	return &models.Resource{
		ID:   nsgName,
		Type: "azurerm_network_security_group",
		Attributes: map[string]interface{}{
			"name":               nsg["name"],
			"location":           nsg["location"],
			"security_rules":     properties["securityRules"],
			"tags":               nsg["tags"],
			"provisioning_state": properties["provisioningState"],
		},
	}, nil
}

// getStorageAccount retrieves a storage account
func (p *AzureProviderComplete) getStorageAccount(ctx context.Context, storageAccountName string) (*models.Resource, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s",
		p.subscriptionID, p.resourceGroup, storageAccountName)
	
	apiVersion := p.apiVersion["Microsoft.Storage/storageAccounts"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage account: %w", err)
	}

	var storage map[string]interface{}
	if err := json.Unmarshal(data, &storage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal storage account: %w", err)
	}

	properties := storage["properties"].(map[string]interface{})
	sku := storage["sku"].(map[string]interface{})
	
	return &models.Resource{
		ID:   storageAccountName,
		Type: "azurerm_storage_account",
		Attributes: map[string]interface{}{
			"name":                     storage["name"],
			"location":                 storage["location"],
			"account_tier":             sku["tier"],
			"account_replication_type": strings.TrimPrefix(sku["name"].(string), sku["tier"].(string)+"_"),
			"account_kind":             storage["kind"],
			"tags":                     storage["tags"],
			"provisioning_state":       properties["provisioningState"],
			"primary_endpoints":        properties["primaryEndpoints"],
		},
	}, nil
}

// getSQLServer retrieves a SQL server
func (p *AzureProviderComplete) getSQLServer(ctx context.Context, serverName string) (*models.Resource, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Sql/servers/%s",
		p.subscriptionID, p.resourceGroup, serverName)
	
	apiVersion := p.apiVersion["Microsoft.Sql/servers"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get SQL server: %w", err)
	}

	var server map[string]interface{}
	if err := json.Unmarshal(data, &server); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SQL server: %w", err)
	}

	properties := server["properties"].(map[string]interface{})
	
	return &models.Resource{
		ID:   serverName,
		Type: "azurerm_sql_server",
		Attributes: map[string]interface{}{
			"name":                       server["name"],
			"location":                   server["location"],
			"version":                    properties["version"],
			"administrator_login":        properties["administratorLogin"],
			"fully_qualified_domain_name": properties["fullyQualifiedDomainName"],
			"tags":                       server["tags"],
			"state":                      properties["state"],
		},
	}, nil
}

// getSQLDatabase retrieves a SQL database
func (p *AzureProviderComplete) getSQLDatabase(ctx context.Context, dbID string) (*models.Resource, error) {
	// Parse database ID to extract server and database names
	parts := strings.Split(dbID, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid database ID format")
	}
	serverName := parts[0]
	dbName := parts[1]

	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Sql/servers/%s/databases/%s",
		p.subscriptionID, p.resourceGroup, serverName, dbName)
	
	apiVersion := p.apiVersion["Microsoft.Sql/servers/databases"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get SQL database: %w", err)
	}

	var db map[string]interface{}
	if err := json.Unmarshal(data, &db); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SQL database: %w", err)
	}

	properties := db["properties"].(map[string]interface{})
	sku := db["sku"].(map[string]interface{})
	
	return &models.Resource{
		ID:   dbID,
		Type: "azurerm_sql_database",
		Attributes: map[string]interface{}{
			"name":                   db["name"],
			"server_name":            serverName,
			"location":               db["location"],
			"edition":                sku["tier"],
			"collation":              properties["collation"],
			"max_size_bytes":         properties["maxSizeBytes"],
			"tags":                   db["tags"],
			"status":                 properties["status"],
			"catalog_collation":      properties["catalogCollation"],
			"zone_redundant":         properties["zoneRedundant"],
		},
	}, nil
}

// getKeyVault retrieves a key vault
func (p *AzureProviderComplete) getKeyVault(ctx context.Context, vaultName string) (*models.Resource, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.KeyVault/vaults/%s",
		p.subscriptionID, p.resourceGroup, vaultName)
	
	apiVersion := p.apiVersion["Microsoft.KeyVault/vaults"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get key vault: %w", err)
	}

	var vault map[string]interface{}
	if err := json.Unmarshal(data, &vault); err != nil {
		return nil, fmt.Errorf("failed to unmarshal key vault: %w", err)
	}

	properties := vault["properties"].(map[string]interface{})
	
	return &models.Resource{
		ID:   vaultName,
		Type: "azurerm_key_vault",
		Attributes: map[string]interface{}{
			"name":                      vault["name"],
			"location":                  vault["location"],
			"tenant_id":                 properties["tenantId"],
			"sku_name":                  properties["sku"].(map[string]interface{})["name"],
			"vault_uri":                 properties["vaultUri"],
			"enabled_for_deployment":    properties["enabledForDeployment"],
			"enabled_for_disk_encryption": properties["enabledForDiskEncryption"],
			"enabled_for_template_deployment": properties["enabledForTemplateDeployment"],
			"enable_soft_delete":        properties["enableSoftDelete"],
			"soft_delete_retention_days": properties["softDeleteRetentionInDays"],
			"tags":                      vault["tags"],
		},
	}, nil
}

// getAppService retrieves an app service
func (p *AzureProviderComplete) getAppService(ctx context.Context, appName string) (*models.Resource, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Web/sites/%s",
		p.subscriptionID, p.resourceGroup, appName)
	
	apiVersion := p.apiVersion["Microsoft.Web/sites"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get app service: %w", err)
	}

	var app map[string]interface{}
	if err := json.Unmarshal(data, &app); err != nil {
		return nil, fmt.Errorf("failed to unmarshal app service: %w", err)
	}

	properties := app["properties"].(map[string]interface{})
	
	return &models.Resource{
		ID:   appName,
		Type: "azurerm_app_service",
		Attributes: map[string]interface{}{
			"name":                 app["name"],
			"location":             app["location"],
			"app_service_plan_id":  properties["serverFarmId"],
			"enabled":              properties["enabled"],
			"https_only":           properties["httpsOnly"],
			"client_cert_enabled":  properties["clientCertEnabled"],
			"default_host_name":    properties["defaultHostName"],
			"outbound_ip_addresses": properties["outboundIpAddresses"],
			"state":                properties["state"],
			"tags":                 app["tags"],
		},
	}, nil
}

// getContainerRegistry retrieves a container registry
func (p *AzureProviderComplete) getContainerRegistry(ctx context.Context, registryName string) (*models.Resource, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerRegistry/registries/%s",
		p.subscriptionID, p.resourceGroup, registryName)
	
	apiVersion := p.apiVersion["Microsoft.ContainerRegistry/registries"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get container registry: %w", err)
	}

	var registry map[string]interface{}
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal container registry: %w", err)
	}

	properties := registry["properties"].(map[string]interface{})
	sku := registry["sku"].(map[string]interface{})
	
	return &models.Resource{
		ID:   registryName,
		Type: "azurerm_container_registry",
		Attributes: map[string]interface{}{
			"name":                   registry["name"],
			"location":               registry["location"],
			"sku":                    sku["name"],
			"admin_enabled":          properties["adminUserEnabled"],
			"login_server":           properties["loginServer"],
			"public_network_access_enabled": properties["publicNetworkAccess"] == "Enabled",
			"zone_redundancy_enabled": properties["zoneRedundancy"] == "Enabled",
			"tags":                   registry["tags"],
			"provisioning_state":     properties["provisioningState"],
		},
	}, nil
}

// getKubernetesCluster retrieves an AKS cluster
func (p *AzureProviderComplete) getKubernetesCluster(ctx context.Context, clusterName string) (*models.Resource, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters/%s",
		p.subscriptionID, p.resourceGroup, clusterName)
	
	apiVersion := p.apiVersion["Microsoft.ContainerService/managedClusters"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get AKS cluster: %w", err)
	}

	var cluster map[string]interface{}
	if err := json.Unmarshal(data, &cluster); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AKS cluster: %w", err)
	}

	properties := cluster["properties"].(map[string]interface{})
	
	return &models.Resource{
		ID:   clusterName,
		Type: "azurerm_kubernetes_cluster",
		Attributes: map[string]interface{}{
			"name":                 cluster["name"],
			"location":             cluster["location"],
			"kubernetes_version":   properties["kubernetesVersion"],
			"dns_prefix":           properties["dnsPrefix"],
			"fqdn":                 properties["fqdn"],
			"node_resource_group":  properties["nodeResourceGroup"],
			"enable_rbac":          properties["enableRBAC"],
			"tags":                 cluster["tags"],
			"provisioning_state":   properties["provisioningState"],
		},
	}, nil
}

// ListResources lists all resources of a specific type
func (p *AzureProviderComplete) ListResources(ctx context.Context, resourceType string) ([]*models.Resource, error) {
	switch resourceType {
	case "azurerm_virtual_machine":
		return p.listVirtualMachines(ctx)
	case "azurerm_virtual_network":
		return p.listVirtualNetworks(ctx)
	case "azurerm_network_security_group":
		return p.listNetworkSecurityGroups(ctx)
	case "azurerm_storage_account":
		return p.listStorageAccounts(ctx)
	case "azurerm_sql_server":
		return p.listSQLServers(ctx)
	case "azurerm_key_vault":
		return p.listKeyVaults(ctx)
	case "azurerm_app_service":
		return p.listAppServices(ctx)
	case "azurerm_container_registry":
		return p.listContainerRegistries(ctx)
	case "azurerm_kubernetes_cluster":
		return p.listKubernetesClusters(ctx)
	default:
		return []*models.Resource{}, nil
	}
}

// listVirtualMachines lists all virtual machines
func (p *AzureProviderComplete) listVirtualMachines(ctx context.Context) ([]*models.Resource, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines",
		p.subscriptionID, p.resourceGroup)
	
	apiVersion := p.apiVersion["Microsoft.Compute/virtualMachines"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}

	var result struct {
		Value []map[string]interface{} `json:"value"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal VM list: %w", err)
	}

	resources := make([]*models.Resource, 0, len(result.Value))
	for _, vm := range result.Value {
		properties := vm["properties"].(map[string]interface{})
		resources = append(resources, &models.Resource{
			ID:   vm["name"].(string),
			Type: "azurerm_virtual_machine",
			Attributes: map[string]interface{}{
				"name":     vm["name"],
				"location": vm["location"],
				"vm_size":  properties["hardwareProfile"].(map[string]interface{})["vmSize"],
				"tags":     vm["tags"],
			},
		})
	}

	return resources, nil
}

// listVirtualNetworks lists all virtual networks
func (p *AzureProviderComplete) listVirtualNetworks(ctx context.Context) ([]*models.Resource, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks",
		p.subscriptionID, p.resourceGroup)
	
	apiVersion := p.apiVersion["Microsoft.Network/virtualNetworks"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list VNets: %w", err)
	}

	var result struct {
		Value []map[string]interface{} `json:"value"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal VNet list: %w", err)
	}

	resources := make([]*models.Resource, 0, len(result.Value))
	for _, vnet := range result.Value {
		properties := vnet["properties"].(map[string]interface{})
		addressSpace := properties["addressSpace"].(map[string]interface{})
		resources = append(resources, &models.Resource{
			ID:   vnet["name"].(string),
			Type: "azurerm_virtual_network",
			Attributes: map[string]interface{}{
				"name":          vnet["name"],
				"location":      vnet["location"],
				"address_space": addressSpace["addressPrefixes"],
				"tags":          vnet["tags"],
			},
		})
	}

	return resources, nil
}

// Additional list methods for other resource types...
func (p *AzureProviderComplete) listNetworkSecurityGroups(ctx context.Context) ([]*models.Resource, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkSecurityGroups",
		p.subscriptionID, p.resourceGroup)
	
	apiVersion := p.apiVersion["Microsoft.Network/networkSecurityGroups"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list NSGs: %w", err)
	}

	var result struct {
		Value []map[string]interface{} `json:"value"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal NSG list: %w", err)
	}

	resources := make([]*models.Resource, 0, len(result.Value))
	for _, nsg := range result.Value {
		resources = append(resources, &models.Resource{
			ID:   nsg["name"].(string),
			Type: "azurerm_network_security_group",
			Attributes: map[string]interface{}{
				"name":     nsg["name"],
				"location": nsg["location"],
				"tags":     nsg["tags"],
			},
		})
	}

	return resources, nil
}

func (p *AzureProviderComplete) listStorageAccounts(ctx context.Context) ([]*models.Resource, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts",
		p.subscriptionID, p.resourceGroup)
	
	apiVersion := p.apiVersion["Microsoft.Storage/storageAccounts"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list storage accounts: %w", err)
	}

	var result struct {
		Value []map[string]interface{} `json:"value"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal storage list: %w", err)
	}

	resources := make([]*models.Resource, 0, len(result.Value))
	for _, storage := range result.Value {
		sku := storage["sku"].(map[string]interface{})
		resources = append(resources, &models.Resource{
			ID:   storage["name"].(string),
			Type: "azurerm_storage_account",
			Attributes: map[string]interface{}{
				"name":         storage["name"],
				"location":     storage["location"],
				"account_tier": sku["tier"],
				"account_kind": storage["kind"],
				"tags":         storage["tags"],
			},
		})
	}

	return resources, nil
}

func (p *AzureProviderComplete) listSQLServers(ctx context.Context) ([]*models.Resource, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Sql/servers",
		p.subscriptionID, p.resourceGroup)
	
	apiVersion := p.apiVersion["Microsoft.Sql/servers"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list SQL servers: %w", err)
	}

	var result struct {
		Value []map[string]interface{} `json:"value"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SQL server list: %w", err)
	}

	resources := make([]*models.Resource, 0, len(result.Value))
	for _, server := range result.Value {
		properties := server["properties"].(map[string]interface{})
		resources = append(resources, &models.Resource{
			ID:   server["name"].(string),
			Type: "azurerm_sql_server",
			Attributes: map[string]interface{}{
				"name":                server["name"],
				"location":            server["location"],
				"version":             properties["version"],
				"administrator_login": properties["administratorLogin"],
				"tags":                server["tags"],
			},
		})
	}

	return resources, nil
}

func (p *AzureProviderComplete) listKeyVaults(ctx context.Context) ([]*models.Resource, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.KeyVault/vaults",
		p.subscriptionID, p.resourceGroup)
	
	apiVersion := p.apiVersion["Microsoft.KeyVault/vaults"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list key vaults: %w", err)
	}

	var result struct {
		Value []map[string]interface{} `json:"value"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal key vault list: %w", err)
	}

	resources := make([]*models.Resource, 0, len(result.Value))
	for _, vault := range result.Value {
		resources = append(resources, &models.Resource{
			ID:   vault["name"].(string),
			Type: "azurerm_key_vault",
			Attributes: map[string]interface{}{
				"name":     vault["name"],
				"location": vault["location"],
				"tags":     vault["tags"],
			},
		})
	}

	return resources, nil
}

func (p *AzureProviderComplete) listAppServices(ctx context.Context) ([]*models.Resource, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Web/sites",
		p.subscriptionID, p.resourceGroup)
	
	apiVersion := p.apiVersion["Microsoft.Web/sites"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list app services: %w", err)
	}

	var result struct {
		Value []map[string]interface{} `json:"value"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal app service list: %w", err)
	}

	resources := make([]*models.Resource, 0, len(result.Value))
	for _, app := range result.Value {
		properties := app["properties"].(map[string]interface{})
		resources = append(resources, &models.Resource{
			ID:   app["name"].(string),
			Type: "azurerm_app_service",
			Attributes: map[string]interface{}{
				"name":              app["name"],
				"location":          app["location"],
				"default_host_name": properties["defaultHostName"],
				"tags":              app["tags"],
			},
		})
	}

	return resources, nil
}

func (p *AzureProviderComplete) listContainerRegistries(ctx context.Context) ([]*models.Resource, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerRegistry/registries",
		p.subscriptionID, p.resourceGroup)
	
	apiVersion := p.apiVersion["Microsoft.ContainerRegistry/registries"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list container registries: %w", err)
	}

	var result struct {
		Value []map[string]interface{} `json:"value"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal registry list: %w", err)
	}

	resources := make([]*models.Resource, 0, len(result.Value))
	for _, registry := range result.Value {
		sku := registry["sku"].(map[string]interface{})
		resources = append(resources, &models.Resource{
			ID:   registry["name"].(string),
			Type: "azurerm_container_registry",
			Attributes: map[string]interface{}{
				"name":     registry["name"],
				"location": registry["location"],
				"sku":      sku["name"],
				"tags":     registry["tags"],
			},
		})
	}

	return resources, nil
}

func (p *AzureProviderComplete) listKubernetesClusters(ctx context.Context) ([]*models.Resource, error) {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters",
		p.subscriptionID, p.resourceGroup)
	
	apiVersion := p.apiVersion["Microsoft.ContainerService/managedClusters"]
	fullPath := fmt.Sprintf("%s?api-version=%s", path, apiVersion)

	data, err := p.makeAPIRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list AKS clusters: %w", err)
	}

	var result struct {
		Value []map[string]interface{} `json:"value"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AKS list: %w", err)
	}

	resources := make([]*models.Resource, 0, len(result.Value))
	for _, cluster := range result.Value {
		properties := cluster["properties"].(map[string]interface{})
		resources = append(resources, &models.Resource{
			ID:   cluster["name"].(string),
			Type: "azurerm_kubernetes_cluster",
			Attributes: map[string]interface{}{
				"name":               cluster["name"],
				"location":           cluster["location"],
				"kubernetes_version": properties["kubernetesVersion"],
				"dns_prefix":         properties["dnsPrefix"],
				"tags":               cluster["tags"],
			},
		})
	}

	return resources, nil
}

// ResourceExists checks if a resource exists
func (p *AzureProviderComplete) ResourceExists(ctx context.Context, resourceType string, resourceID string) (bool, error) {
	_, err := p.GetResourceByType(ctx, resourceType, resourceID)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}