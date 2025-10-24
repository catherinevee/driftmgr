# 🛠️ DriftMgr Implementation Guide

## 🚀 **Quick Start - Phase 7: Multi-Cloud Provider Support**

This guide provides step-by-step instructions for implementing the remaining stub functionality in DriftMgr.

---

## 📋 **Phase 7.1: Azure Provider Implementation**

### **Step 1: Setup Azure Dependencies**

Add Azure SDK dependencies to `go.mod`:

```bash
go get github.com/Azure/azure-sdk-for-go/sdk/azidentity
go get github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources
go get github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute
go get github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage
go get github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork
```

### **Step 2: Implement Azure Provider**

Create `internal/providers/azure/provider.go`:

```go
package azure

import (
    "context"
    "fmt"
    "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
    "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
    "github.com/catherinevee/driftmgr/internal/providers/types"
)

type AzureProvider struct {
    subscriptionID string
    client         *armresources.Client
    credential     *azidentity.DefaultAzureCredential
}

func NewAzureProvider(subscriptionID string) (*AzureProvider, error) {
    credential, err := azidentity.NewDefaultAzureCredential(nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create Azure credential: %w", err)
    }

    client, err := armresources.NewClient(subscriptionID, credential, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create Azure client: %w", err)
    }

    return &AzureProvider{
        subscriptionID: subscriptionID,
        client:         client,
        credential:     credential,
    }, nil
}

func (p *AzureProvider) GetName() string {
    return "azure"
}

func (p *AzureProvider) GetVersion() string {
    return "1.0.0"
}

func (p *AzureProvider) GetCapabilities() []types.Capability {
    return []types.Capability{
        types.CapabilityResourceDiscovery,
        types.CapabilityStateManagement,
        types.CapabilityDriftDetection,
    }
}

func (p *AzureProvider) TestConnection(ctx context.Context) error {
    // Test connection by listing resource groups
    pager := p.client.NewListPager(nil)
    _, err := pager.NextPage(ctx)
    if err != nil {
        return fmt.Errorf("failed to connect to Azure: %w", err)
    }
    return nil
}
```

### **Step 3: Implement Azure Discovery**

Create `internal/providers/azure/discovery.go`:

```go
package azure

import (
    "context"
    "fmt"
    "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
    "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
    "github.com/catherinevee/driftmgr/internal/models"
)

func (p *AzureProvider) DiscoverResources(ctx context.Context) ([]models.CloudResource, error) {
    var resources []models.CloudResource

    // Discover Virtual Machines
    vmResources, err := p.discoverVirtualMachines(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to discover VMs: %w", err)
    }
    resources = append(resources, vmResources...)

    // Discover Storage Accounts
    storageResources, err := p.discoverStorageAccounts(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to discover storage accounts: %w", err)
    }
    resources = append(resources, storageResources...)

    return resources, nil
}

func (p *AzureProvider) discoverVirtualMachines(ctx context.Context) ([]models.CloudResource, error) {
    var resources []models.CloudResource

    // List all resource groups first
    rgClient, err := armresources.NewResourceGroupsClient(p.subscriptionID, p.credential, nil)
    if err != nil {
        return nil, err
    }

    rgPager := rgClient.NewListPager(nil)
    for rgPager.More() {
        page, err := rgPager.NextPage(ctx)
        if err != nil {
            return nil, err
        }

        for _, rg := range page.Value {
            // List VMs in each resource group
            vmClient, err := armcompute.NewVirtualMachinesClient(p.subscriptionID, p.credential, nil)
            if err != nil {
                continue
            }

            vmPager := vmClient.NewListPager(*rg.Name, nil)
            for vmPager.More() {
                vmPage, err := vmPager.NextPage(ctx)
                if err != nil {
                    break
                }

                for _, vm := range vmPage.Value {
                    resource := models.CloudResource{
                        ID:          *vm.ID,
                        Type:        "azurerm_virtual_machine",
                        Name:        *vm.Name,
                        Provider:    "azure",
                        Region:      *vm.Location,
                        ResourceGroup: *rg.Name,
                        Properties:  vm.Properties,
                        Tags:        convertTags(vm.Tags),
                        CreatedAt:   vm.SystemData.CreatedAt,
                        UpdatedAt:   vm.SystemData.LastModifiedAt,
                    }
                    resources = append(resources, resource)
                }
            }
        }
    }

    return resources, nil
}

func (p *AzureProvider) discoverStorageAccounts(ctx context.Context) ([]models.CloudResource, error) {
    var resources []models.CloudResource

    storageClient, err := armstorage.NewAccountsClient(p.subscriptionID, p.credential, nil)
    if err != nil {
        return nil, err
    }

    pager := storageClient.NewListPager(nil)
    for pager.More() {
        page, err := pager.NextPage(ctx)
        if err != nil {
            return nil, err
        }

        for _, account := range page.Value {
            resource := models.CloudResource{
                ID:          *account.ID,
                Type:        "azurerm_storage_account",
                Name:        *account.Name,
                Provider:    "azure",
                Region:      *account.Location,
                Properties:  account.Properties,
                Tags:        convertTags(account.Tags),
                CreatedAt:   account.SystemData.CreatedAt,
                UpdatedAt:   account.SystemData.LastModifiedAt,
            }
            resources = append(resources, resource)
        }
    }

    return resources, nil
}

func convertTags(tags map[string]*string) map[string]string {
    result := make(map[string]string)
    for k, v := range tags {
        if v != nil {
            result[k] = *v
        }
    }
    return result
}
```

### **Step 4: Update Discovery Engine**

Update `internal/discovery/engine.go`:

```go
// Replace the Azure TODO section with:
func (e *DiscoveryEngine) initializeAzureProvider(ctx context.Context, config ProviderConfig) error {
    provider, err := azure.NewAzureProvider(config.SubscriptionID)
    if err != nil {
        return fmt.Errorf("failed to initialize Azure provider: %w", err)
    }
    
    e.providers["azure"] = provider
    return nil
}
```

---

## 📋 **Phase 7.2: GCP Provider Implementation**

### **Step 1: Setup GCP Dependencies**

```bash
go get cloud.google.com/go/compute/apiv1
go get cloud.google.com/go/storage
go get cloud.google.com/go/iam/apiv1
go get google.golang.org/api/option
```

### **Step 2: Implement GCP Provider**

Create `internal/providers/gcp/provider.go`:

```go
package gcp

import (
    "context"
    "fmt"
    "cloud.google.com/go/compute/apiv1"
    "cloud.google.com/go/storage"
    "github.com/catherinevee/driftmgr/internal/providers/types"
)

type GCPProvider struct {
    projectID string
    computeClient *compute.InstancesClient
    storageClient *storage.Client
}

func NewGCPProvider(projectID string) (*GCPProvider, error) {
    ctx := context.Background()
    
    computeClient, err := compute.NewInstancesRESTClient(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to create GCP compute client: %w", err)
    }
    
    storageClient, err := storage.NewClient(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to create GCP storage client: %w", err)
    }
    
    return &GCPProvider{
        projectID:     projectID,
        computeClient: computeClient,
        storageClient: storageClient,
    }, nil
}

func (p *GCPProvider) GetName() string {
    return "gcp"
}

func (p *GCPProvider) GetVersion() string {
    return "1.0.0"
}

func (p *GCPProvider) GetCapabilities() []types.Capability {
    return []types.Capability{
        types.CapabilityResourceDiscovery,
        types.CapabilityStateManagement,
        types.CapabilityDriftDetection,
    }
}

func (p *GCPProvider) TestConnection(ctx context.Context) error {
    // Test connection by listing instances
    req := &computepb.ListInstancesRequest{
        Project: p.projectID,
        Zone:    "us-central1-a", // Default zone for testing
    }
    
    _, err := p.computeClient.List(ctx, req)
    if err != nil {
        return fmt.Errorf("failed to connect to GCP: %w", err)
    }
    return nil
}
```

---

## 📋 **Phase 7.3: DigitalOcean Provider Implementation**

### **Step 1: Setup DigitalOcean Dependencies**

```bash
go get github.com/digitalocean/godo
```

### **Step 2: Implement DigitalOcean Provider**

Create `internal/providers/digitalocean/provider.go`:

```go
package digitalocean

import (
    "context"
    "fmt"
    "github.com/digitalocean/godo"
    "github.com/catherinevee/driftmgr/internal/providers/types"
)

type DigitalOceanProvider struct {
    client *godo.Client
}

func NewDigitalOceanProvider(apiToken string) (*DigitalOceanProvider, error) {
    client := godo.NewFromToken(apiToken)
    
    return &DigitalOceanProvider{
        client: client,
    }, nil
}

func (p *DigitalOceanProvider) GetName() string {
    return "digitalocean"
}

func (p *DigitalOceanProvider) GetVersion() string {
    return "1.0.0"
}

func (p *DigitalOceanProvider) GetCapabilities() []types.Capability {
    return []types.Capability{
        types.CapabilityResourceDiscovery,
        types.CapabilityStateManagement,
        types.CapabilityDriftDetection,
    }
}

func (p *DigitalOceanProvider) TestConnection(ctx context.Context) error {
    // Test connection by listing droplets
    _, _, err := p.client.Droplets.List(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to connect to DigitalOcean: %w", err)
    }
    return nil
}
```

---

## 🧪 **Testing Implementation**

### **Unit Tests**

Create test files for each provider:

```go
// internal/providers/azure/provider_test.go
package azure

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestAzureProvider_New(t *testing.T) {
    // Mock Azure credentials for testing
    provider, err := NewAzureProvider("test-subscription-id")
    require.NoError(t, err)
    assert.NotNil(t, provider)
    assert.Equal(t, "azure", provider.GetName())
}

func TestAzureProvider_TestConnection(t *testing.T) {
    provider, err := NewAzureProvider("test-subscription-id")
    require.NoError(t, err)
    
    // This will fail in CI without real credentials, but tests the structure
    err = provider.TestConnection(context.Background())
    // We expect this to fail in test environment
    assert.Error(t, err)
}
```

### **Integration Tests**

Create integration test files:

```go
// tests/integration/providers/azure_test.go
package providers

import (
    "context"
    "os"
    "testing"
    "github.com/catherinevee/driftmgr/internal/providers/azure"
    "github.com/stretchr/testify/require"
)

func TestAzureProvider_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
    if subscriptionID == "" {
        t.Skip("AZURE_SUBSCRIPTION_ID not set")
    }
    
    provider, err := azure.NewAzureProvider(subscriptionID)
    require.NoError(t, err)
    
    err = provider.TestConnection(context.Background())
    require.NoError(t, err)
    
    resources, err := provider.DiscoverResources(context.Background())
    require.NoError(t, err)
    t.Logf("Discovered %d resources", len(resources))
}
```

---

## 🔧 **Configuration Updates**

### **Environment Variables**

Add to `.env.example`:

```bash
# Azure Configuration
AZURE_SUBSCRIPTION_ID=your-subscription-id
AZURE_TENANT_ID=your-tenant-id
AZURE_CLIENT_ID=your-client-id
AZURE_CLIENT_SECRET=your-client-secret

# GCP Configuration
GCP_PROJECT_ID=your-project-id
GOOGLE_APPLICATION_CREDENTIALS=path/to/service-account.json

# DigitalOcean Configuration
DIGITALOCEAN_TOKEN=your-api-token
```

### **Configuration File**

Update `configs/config.yaml`:

```yaml
providers:
  azure:
    enabled: true
    subscription_id: "${AZURE_SUBSCRIPTION_ID}"
    tenant_id: "${AZURE_TENANT_ID}"
    client_id: "${AZURE_CLIENT_ID}"
    client_secret: "${AZURE_CLIENT_SECRET}"
  
  gcp:
    enabled: true
    project_id: "${GCP_PROJECT_ID}"
    credentials_file: "${GOOGLE_APPLICATION_CREDENTIALS}"
  
  digitalocean:
    enabled: true
    api_token: "${DIGITALOCEAN_TOKEN}"
```

---

## 📚 **Documentation Updates**

### **API Documentation**

Update API documentation to include new provider endpoints:

```yaml
# docs/api/providers.yaml
paths:
  /api/v1/providers/azure:
    get:
      summary: List Azure resources
      parameters:
        - name: resource_group
          in: query
          description: Filter by resource group
        - name: resource_type
          in: query
          description: Filter by resource type
      responses:
        200:
          description: List of Azure resources
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/CloudResource'
```

### **User Guide**

Create provider-specific setup guides:

```markdown
# docs/providers/azure-setup.md
# Azure Provider Setup Guide

## Prerequisites
- Azure subscription
- Service Principal or Managed Identity
- Appropriate permissions

## Setup Steps
1. Create Service Principal
2. Assign permissions
3. Configure environment variables
4. Test connection
```

---

## 🚀 **Deployment Updates**

### **Docker Configuration**

Update `Dockerfile` to include provider-specific dependencies:

```dockerfile
# Add Azure CLI for debugging
RUN curl -sL https://aka.ms/InstallAzureCLIDeb | bash

# Add GCP CLI for debugging
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
RUN curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key --keyring /usr/share/keyrings/cloud.google.gpg add -
RUN apt-get update && apt-get install -y google-cloud-cli
```

### **Kubernetes Configuration**

Update Helm charts to include provider configurations:

```yaml
# charts/driftmgr/values.yaml
providers:
  azure:
    enabled: true
    config:
      subscriptionId: ""
      tenantId: ""
      clientId: ""
      clientSecret: ""
  
  gcp:
    enabled: true
    config:
      projectId: ""
      credentialsFile: ""
  
  digitalocean:
    enabled: true
    config:
      apiToken: ""
```

---

## ✅ **Validation Checklist**

### **Azure Provider**
- [ ] Provider initializes successfully
- [ ] Connection test passes
- [ ] Resource discovery works
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Documentation updated

### **GCP Provider**
- [ ] Provider initializes successfully
- [ ] Connection test passes
- [ ] Resource discovery works
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Documentation updated

### **DigitalOcean Provider**
- [ ] Provider initializes successfully
- [ ] Connection test passes
- [ ] Resource discovery works
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Documentation updated

---

## 🎯 **Next Steps**

After completing Phase 7:

1. **Test thoroughly** with real cloud accounts
2. **Update documentation** with provider-specific guides
3. **Create example configurations** for each provider
4. **Set up CI/CD** for multi-cloud testing
5. **Move to Phase 8** (Event Publishing System)

---

## 📞 **Support**

For questions or issues:
- Check the [troubleshooting guide](docs/troubleshooting.md)
- Review [provider-specific documentation](docs/providers/)
- Open an issue on GitHub
- Join the community discussions

---

**Happy coding! 🚀**
