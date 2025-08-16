package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// ServiceConfig represents configuration for a discovery service
type ServiceConfig struct {
	Name        string            `json:"name"`
	Enabled     bool              `json:"enabled"`
	Priority    int               `json:"priority"`
	Regions     []string          `json:"regions"`
	Parameters  map[string]string `json:"parameters"`
	Description string            `json:"description"`
}

// ProviderServices represents services for a cloud provider
type ProviderServices struct {
	Provider string                    `json:"provider"`
	Services map[string]*ServiceConfig `json:"services"`
}

// DiscoveryServicesConfig manages service configurations
type DiscoveryServicesConfig struct {
	Providers  map[string]*ProviderServices `json:"providers"`
	mu         sync.RWMutex
	configPath string
}

// NewDiscoveryServicesConfig creates a new service configuration manager
func NewDiscoveryServicesConfig(configPath string) *DiscoveryServicesConfig {
	config := &DiscoveryServicesConfig{
		Providers:  make(map[string]*ProviderServices),
		configPath: configPath,
	}

	// Load default configuration
	config.loadDefaults()

	// Load from file if exists
	if err := config.LoadFromFile(); err != nil {
		fmt.Printf("Warning: Could not load service config from %s: %v\n", configPath, err)
	}

	return config
}

// loadDefaults loads default service configurations
func (dsc *DiscoveryServicesConfig) loadDefaults() {
	dsc.mu.Lock()
	defer dsc.mu.Unlock()

	// AWS Services
	awsServices := &ProviderServices{
		Provider: "aws",
		Services: map[string]*ServiceConfig{
			"ec2": {
				Name:        "ec2",
				Enabled:     true,
				Priority:    1,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon Elastic Compute Cloud instances",
			},
			"vpc": {
				Name:        "vpc",
				Enabled:     true,
				Priority:    2,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon Virtual Private Cloud",
			},
			"s3": {
				Name:        "s3",
				Enabled:     true,
				Priority:    3,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon Simple Storage Service",
			},
			"rds": {
				Name:        "rds",
				Enabled:     true,
				Priority:    4,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon Relational Database Service",
			},
			"lambda": {
				Name:        "lambda",
				Enabled:     true,
				Priority:    5,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Lambda functions",
			},
			"ecs": {
				Name:        "ecs",
				Enabled:     true,
				Priority:    6,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon Elastic Container Service",
			},
			"eks": {
				Name:        "eks",
				Enabled:     true,
				Priority:    7,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon Elastic Kubernetes Service",
			},
			"dynamodb": {
				Name:        "dynamodb",
				Enabled:     true,
				Priority:    8,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon DynamoDB",
			},
			"cloudwatch": {
				Name:        "cloudwatch",
				Enabled:     true,
				Priority:    9,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon CloudWatch",
			},
			"iam": {
				Name:        "iam",
				Enabled:     true,
				Priority:    10,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Identity and Access Management",
			},
		},
	}

	// Azure Services
	azureServices := &ProviderServices{
		Provider: "azure",
		Services: map[string]*ServiceConfig{
			"vm": {
				Name:        "vm",
				Enabled:     true,
				Priority:    1,
				Regions:     []string{"eastus", "westus2", "westeurope"},
				Parameters:  map[string]string{},
				Description: "Azure Virtual Machines",
			},
			"vnet": {
				Name:        "vnet",
				Enabled:     true,
				Priority:    2,
				Regions:     []string{"eastus", "westus2", "westeurope"},
				Parameters:  map[string]string{},
				Description: "Azure Virtual Networks",
			},
			"storage": {
				Name:        "storage",
				Enabled:     true,
				Priority:    3,
				Regions:     []string{"eastus", "westus2", "westeurope"},
				Parameters:  map[string]string{},
				Description: "Azure Storage Accounts",
			},
			"sql": {
				Name:        "sql",
				Enabled:     true,
				Priority:    4,
				Regions:     []string{"eastus", "westus2", "westeurope"},
				Parameters:  map[string]string{},
				Description: "Azure SQL Database",
			},
			"function": {
				Name:        "function",
				Enabled:     true,
				Priority:    5,
				Regions:     []string{"eastus", "westus2", "westeurope"},
				Parameters:  map[string]string{},
				Description: "Azure Functions",
			},
			"webapp": {
				Name:        "webapp",
				Enabled:     true,
				Priority:    6,
				Regions:     []string{"eastus", "westus2", "westeurope"},
				Parameters:  map[string]string{},
				Description: "Azure Web Apps",
			},
			"aks": {
				Name:        "aks",
				Enabled:     true,
				Priority:    7,
				Regions:     []string{"eastus", "westus2", "westeurope"},
				Parameters:  map[string]string{},
				Description: "Azure Kubernetes Service",
			},
			"cosmosdb": {
				Name:        "cosmosdb",
				Enabled:     true,
				Priority:    8,
				Regions:     []string{"eastus", "westus2", "westeurope"},
				Parameters:  map[string]string{},
				Description: "Azure Cosmos DB",
			},
			"monitor": {
				Name:        "monitor",
				Enabled:     true,
				Priority:    9,
				Regions:     []string{"eastus", "westus2", "westeurope"},
				Parameters:  map[string]string{},
				Description: "Azure Monitor",
			},
			"keyvault": {
				Name:        "keyvault",
				Enabled:     true,
				Priority:    10,
				Regions:     []string{"eastus", "westus2", "westeurope"},
				Parameters:  map[string]string{},
				Description: "Azure Key Vault",
			},
		},
	}

	// GCP Services
	gcpServices := &ProviderServices{
		Provider: "gcp",
		Services: map[string]*ServiceConfig{
			"compute": {
				Name:        "compute",
				Enabled:     true,
				Priority:    1,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Compute Engine",
			},
			"network": {
				Name:        "network",
				Enabled:     true,
				Priority:    2,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Networking",
			},
			"storage": {
				Name:        "storage",
				Enabled:     true,
				Priority:    3,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Storage",
			},
			"sql": {
				Name:        "sql",
				Enabled:     true,
				Priority:    4,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud SQL",
			},
			"function": {
				Name:        "function",
				Enabled:     true,
				Priority:    5,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Functions",
			},
			"run": {
				Name:        "run",
				Enabled:     true,
				Priority:    6,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Run",
			},
			"gke": {
				Name:        "gke",
				Enabled:     true,
				Priority:    7,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Kubernetes Engine",
			},
			"bigquery": {
				Name:        "bigquery",
				Enabled:     true,
				Priority:    8,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google BigQuery",
			},
			"monitoring": {
				Name:        "monitoring",
				Enabled:     true,
				Priority:    9,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Monitoring",
			},
			"iam": {
				Name:        "iam",
				Enabled:     true,
				Priority:    10,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud IAM",
			},
		},
	}

	dsc.Providers["aws"] = awsServices
	dsc.Providers["azure"] = azureServices
	dsc.Providers["gcp"] = gcpServices
}

// LoadFromFile loads service configuration from file
func (dsc *DiscoveryServicesConfig) LoadFromFile() error {
	if dsc.configPath == "" {
		return fmt.Errorf("no config path specified")
	}

	data, err := os.ReadFile(dsc.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	dsc.mu.Lock()
	defer dsc.mu.Unlock()

	if err := json.Unmarshal(data, &dsc.Providers); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// SaveToFile saves service configuration to file
func (dsc *DiscoveryServicesConfig) SaveToFile() error {
	if dsc.configPath == "" {
		return fmt.Errorf("no config path specified")
	}

	dsc.mu.RLock()
	defer dsc.mu.RUnlock()

	data, err := json.MarshalIndent(dsc.Providers, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(dsc.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetEnabledServices returns enabled services for a provider
func (dsc *DiscoveryServicesConfig) GetEnabledServices(provider string) []string {
	dsc.mu.RLock()
	defer dsc.mu.RUnlock()

	providerServices, exists := dsc.Providers[provider]
	if !exists {
		return []string{}
	}

	var enabledServices []string
	for serviceName, serviceConfig := range providerServices.Services {
		if serviceConfig.Enabled {
			enabledServices = append(enabledServices, serviceName)
		}
	}

	return enabledServices
}

// GetServiceConfig returns configuration for a specific service
func (dsc *DiscoveryServicesConfig) GetServiceConfig(provider, service string) (*ServiceConfig, error) {
	dsc.mu.RLock()
	defer dsc.mu.RUnlock()

	providerServices, exists := dsc.Providers[provider]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", provider)
	}

	serviceConfig, exists := providerServices.Services[service]
	if !exists {
		return nil, fmt.Errorf("service %s not found for provider %s", service, provider)
	}

	return serviceConfig, nil
}

// EnableService enables a service for discovery
func (dsc *DiscoveryServicesConfig) EnableService(provider, service string) error {
	dsc.mu.Lock()
	defer dsc.mu.Unlock()

	providerServices, exists := dsc.Providers[provider]
	if !exists {
		return fmt.Errorf("provider %s not found", provider)
	}

	serviceConfig, exists := providerServices.Services[service]
	if !exists {
		return fmt.Errorf("service %s not found for provider %s", service, provider)
	}

	serviceConfig.Enabled = true
	return nil
}

// DisableService disables a service for discovery
func (dsc *DiscoveryServicesConfig) DisableService(provider, service string) error {
	dsc.mu.Lock()
	defer dsc.mu.Unlock()

	providerServices, exists := dsc.Providers[provider]
	if !exists {
		return fmt.Errorf("provider %s not found", provider)
	}

	serviceConfig, exists := providerServices.Services[service]
	if !exists {
		return fmt.Errorf("service %s not found for provider %s", service, provider)
	}

	serviceConfig.Enabled = false
	return nil
}

// GetServicesMap returns a map of provider to enabled services
func (dsc *DiscoveryServicesConfig) GetServicesMap() map[string][]string {
	dsc.mu.RLock()
	defer dsc.mu.RUnlock()

	servicesMap := make(map[string][]string)

	for provider, providerServices := range dsc.Providers {
		var enabledServices []string
		for serviceName, serviceConfig := range providerServices.Services {
			if serviceConfig.Enabled {
				enabledServices = append(enabledServices, serviceName)
			}
		}
		servicesMap[provider] = enabledServices
	}

	return servicesMap
}
