package providers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/catherinevee/driftmgr/internal/providers/aws"
	"github.com/catherinevee/driftmgr/internal/providers/azure"
	"github.com/catherinevee/driftmgr/internal/providers/digitalocean"
	"github.com/catherinevee/driftmgr/internal/providers/gcp"
)

// NewProvider creates a new provider based on the provider name
func NewProvider(providerName string, config map[string]interface{}) (CloudProvider, error) {
	switch strings.ToLower(providerName) {
	case "aws":
		region := ""
		if r, ok := config["region"].(string); ok {
			region = r
		}
		return NewAWSProvider(region), nil
	case "azure":
		subscriptionID := ""
		resourceGroup := ""
		if s, ok := config["subscription_id"].(string); ok {
			subscriptionID = s
		}
		if r, ok := config["resource_group"].(string); ok {
			resourceGroup = r
		}
		return NewAzureProviderComplete(subscriptionID, resourceGroup, "", "", "", ""), nil
	case "gcp":
		projectID := ""
		if p, ok := config["project_id"].(string); ok {
			projectID = p
		}
		return gcp.NewGCPProviderComplete(projectID), nil
	case "digitalocean":
		region := ""
		if r, ok := config["region"].(string); ok {
			region = r
		}
		return digitalocean.NewDigitalOceanProvider(region), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}

// NewAWSProvider creates a new AWS provider with the specified region
func NewAWSProvider(region string) *aws.AWSProvider {
	if region == "" {
		region = "us-east-1"
	}
	return aws.NewAWSProvider(region)
}

// NewAzureProviderComplete creates a new Azure provider with full API implementation
func NewAzureProviderComplete(subscriptionID, resourceGroup, tenantID, clientID, clientSecret, region string) *azure.AzureProviderComplete {
	// Auto-detect credentials if not provided
	if subscriptionID == "" {
		subscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")
	}
	if tenantID == "" {
		tenantID = os.Getenv("AZURE_TENANT_ID")
	}
	if clientID == "" {
		clientID = os.Getenv("AZURE_CLIENT_ID")
	}
	if clientSecret == "" {
		clientSecret = os.Getenv("AZURE_CLIENT_SECRET")
	}
	if region == "" {
		region = "eastus"
	}

	// Create provider with available constructor
	provider := azure.NewAzureProviderComplete(subscriptionID, resourceGroup)

	// Set auth credentials through environment or other means
	// The provider will pick up credentials from environment variables during Initialize
	os.Setenv("AZURE_TENANT_ID", tenantID)
	os.Setenv("AZURE_CLIENT_ID", clientID)
	os.Setenv("AZURE_CLIENT_SECRET", clientSecret)

	// Initialize authentication
	ctx := context.Background()
	if err := provider.Connect(ctx); err != nil {
		// Log error but don't fail - authentication will be retried on first use
		fmt.Printf("Warning: Azure authentication failed during initialization: %v\n", err)
	}

	return provider
}

// NewGCPProviderComplete creates a new GCP provider with full API implementation
func NewGCPProviderComplete(projectID, region, credentialsPath string) *gcp.GCPProviderComplete {
	// Auto-detect project ID if not provided
	if projectID == "" {
		projectID = os.Getenv("GCP_PROJECT")
		if projectID == "" {
			projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
		}
	}

	// Auto-detect credentials path if not provided
	if credentialsPath == "" {
		credentialsPath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}

	if region == "" {
		region = "us-central1"
	}

	// Set credentials path in environment for provider to pick up
	if credentialsPath != "" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credentialsPath)
	}

	// Create provider with available constructor
	provider := gcp.NewGCPProviderComplete(projectID)

	// Initialize authentication
	ctx := context.Background()
	if err := provider.Connect(ctx); err != nil {
		// Log error but don't fail - authentication will be retried on first use
		fmt.Printf("Warning: GCP authentication failed during initialization: %v\n", err)
	}

	return provider
}

// NewDigitalOceanProvider creates a new DigitalOcean provider with the specified region
func NewDigitalOceanProvider(region string) *digitalocean.DigitalOceanProvider {
	if region == "" {
		region = "nyc1"
	}
	return digitalocean.NewDigitalOceanProvider(region)
}

// ProviderFactory creates cloud providers based on configuration
type ProviderFactory struct {
	config map[string]interface{}
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(config map[string]interface{}) *ProviderFactory {
	return &ProviderFactory{
		config: config,
	}
}

// CreateProvider creates a cloud provider by name
func (pf *ProviderFactory) CreateProvider(providerName string) (CloudProvider, error) {
	switch providerName {
	case "aws":
		region := ""
		if r, ok := pf.config["aws_region"].(string); ok {
			region = r
		}
		provider := NewAWSProvider(region)
		if err := provider.Initialize(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to initialize AWS provider: %w", err)
		}
		return provider, nil

	case "azure", "azurerm":
		subscriptionID := ""
		resourceGroup := ""
		tenantID := ""
		clientID := ""
		clientSecret := ""
		region := ""

		if s, ok := pf.config["azure_subscription_id"].(string); ok {
			subscriptionID = s
		}
		if r, ok := pf.config["azure_resource_group"].(string); ok {
			resourceGroup = r
		}
		if t, ok := pf.config["azure_tenant_id"].(string); ok {
			tenantID = t
		}
		if c, ok := pf.config["azure_client_id"].(string); ok {
			clientID = c
		}
		if cs, ok := pf.config["azure_client_secret"].(string); ok {
			clientSecret = cs
		}
		if r, ok := pf.config["azure_region"].(string); ok {
			region = r
		}

		return NewAzureProviderComplete(subscriptionID, resourceGroup, tenantID, clientID, clientSecret, region), nil

	case "gcp", "google":
		projectID := ""
		region := ""
		credentialsPath := ""

		if p, ok := pf.config["gcp_project"].(string); ok {
			projectID = p
		}
		if r, ok := pf.config["gcp_region"].(string); ok {
			region = r
		}
		if c, ok := pf.config["gcp_credentials"].(string); ok {
			credentialsPath = c
		}

		return NewGCPProviderComplete(projectID, region, credentialsPath), nil

	case "digitalocean":
		region := ""
		if r, ok := pf.config["digitalocean_region"].(string); ok {
			region = r
		}

		return NewDigitalOceanProvider(region), nil

	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}

// DetectProviders detects available cloud providers based on environment and credentials
func DetectProviders() []string {
	var providers []string

	// Check for AWS credentials
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" || os.Getenv("AWS_PROFILE") != "" {
		providers = append(providers, "aws")
	} else if _, err := os.Stat(os.ExpandEnv("$HOME/.aws/credentials")); err == nil {
		providers = append(providers, "aws")
	}

	// Check for Azure credentials
	if os.Getenv("AZURE_CLIENT_ID") != "" && os.Getenv("AZURE_CLIENT_SECRET") != "" {
		providers = append(providers, "azure")
	}

	// Check for GCP credentials
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		providers = append(providers, "gcp")
	} else if os.Getenv("GCP_PROJECT") != "" || os.Getenv("GOOGLE_CLOUD_PROJECT") != "" {
		providers = append(providers, "gcp")
	}

	// Check for DigitalOcean credentials
	if os.Getenv("DIGITALOCEAN_TOKEN") != "" {
		providers = append(providers, "digitalocean")
	}

	return providers
}

// ValidateProviderCredentials validates that necessary credentials exist for a provider
func ValidateProviderCredentials(providerName string) error {
	switch providerName {
	case "aws":
		// Check for AWS credentials
		if os.Getenv("AWS_ACCESS_KEY_ID") == "" && os.Getenv("AWS_PROFILE") == "" {
			if _, err := os.Stat(os.ExpandEnv("$HOME/.aws/credentials")); err != nil {
				return fmt.Errorf("AWS credentials not found. Set AWS_ACCESS_KEY_ID or configure AWS CLI")
			}
		}
		return nil

	case "azure", "azurerm":
		// Check for Azure credentials
		if os.Getenv("AZURE_CLIENT_ID") == "" || os.Getenv("AZURE_CLIENT_SECRET") == "" {
			return fmt.Errorf("Azure credentials not found. Set AZURE_CLIENT_ID and AZURE_CLIENT_SECRET")
		}
		if os.Getenv("AZURE_TENANT_ID") == "" {
			return fmt.Errorf("Azure tenant ID not found. Set AZURE_TENANT_ID")
		}
		if os.Getenv("AZURE_SUBSCRIPTION_ID") == "" {
			return fmt.Errorf("Azure subscription ID not found. Set AZURE_SUBSCRIPTION_ID")
		}
		return nil

	case "gcp", "google":
		// Check for GCP credentials
		if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
			if os.Getenv("GCP_PROJECT") == "" && os.Getenv("GOOGLE_CLOUD_PROJECT") == "" {
				return fmt.Errorf("GCP credentials not found. Set GOOGLE_APPLICATION_CREDENTIALS or configure gcloud")
			}
		}
		return nil

	case "digitalocean":
		// Check for DigitalOcean credentials
		if os.Getenv("DIGITALOCEAN_TOKEN") == "" {
			return fmt.Errorf("DigitalOcean token not found. Set DIGITALOCEAN_TOKEN environment variable")
		}
		return nil

	default:
		return fmt.Errorf("unsupported provider: %s", providerName)
	}
}
