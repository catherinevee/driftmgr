package cloud

import (
	"context"
	"fmt"
	"strings"
)

// Provider represents a cloud provider
type Provider struct {
	Name string
	Type string
}

// Resource represents a cloud resource
type Resource struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Provider   string                 `json:"provider"`
	Region     string                 `json:"region"`
	State      string                 `json:"state"`
	Tags       map[string]string      `json:"tags"`
	Properties map[string]interface{} `json:"properties"`
	AccountID  string                 `json:"account_id"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}

// GetResourceID returns the resource ID
func GetResourceID(r Resource) string {
	return r.ID
}

// GetResourceName returns the resource name
func GetResourceName(r Resource) string {
	return r.Name
}

// GetResourceType returns the resource type
func GetResourceType(r Resource) string {
	return r.Type
}

// GetResourceProvider returns the resource provider
func GetResourceProvider(r Resource) string {
	return r.Provider
}

// GetResourceRegion returns the resource region
func GetResourceRegion(r Resource) string {
	return r.Region
}

// GetResourceState returns the resource state
func GetResourceState(r Resource) string {
	return r.State
}

// GetResourceTags returns the resource tags
func GetResourceTags(r Resource) map[string]string {
	return r.Tags
}

// GetResourceProperties returns the resource properties
func GetResourceProperties(r Resource) map[string]interface{} {
	return r.Properties
}

// DiscoverResources discovers resources for a provider
func DiscoverResources(ctx context.Context, provider string) ([]Resource, error) {
	// Stub implementation
	return []Resource{}, nil
}

// NormalizeProviderName normalizes provider names
func NormalizeProviderName(provider string) string {
	switch strings.ToLower(provider) {
	case "aws", "amazon":
		return "aws"
	case "azure", "microsoft":
		return "azure"
	case "gcp", "google":
		return "gcp"
	case "do", "digitalocean":
		return "digitalocean"
	default:
		return strings.ToLower(provider)
	}
}

// IsValidProvider checks if a provider is valid
func IsValidProvider(provider string) bool {
	normalized := NormalizeProviderName(provider)
	switch normalized {
	case "aws", "azure", "gcp", "digitalocean":
		return true
	default:
		return false
	}
}

// GetProviderDisplayName returns display name for provider
func GetProviderDisplayName(provider string) string {
	switch NormalizeProviderName(provider) {
	case "aws":
		return "Amazon Web Services"
	case "azure":
		return "Microsoft Azure"
	case "gcp":
		return "Google Cloud Platform"
	case "digitalocean":
		return "DigitalOcean"
	default:
		return provider
	}
}

// FilterResourcesByType filters resources by type
func FilterResourcesByType(resources []Resource, resourceType string) []Resource {
	var filtered []Resource
	for _, r := range resources {
		if r.Type == resourceType {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// FilterResourcesByRegion filters resources by region
func FilterResourcesByRegion(resources []Resource, region string) []Resource {
	var filtered []Resource
	for _, r := range resources {
		if r.Region == region {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// GroupResourcesByProvider groups resources by provider
func GroupResourcesByProvider(resources []Resource) map[string][]Resource {
	grouped := make(map[string][]Resource)
	for _, r := range resources {
		grouped[r.Provider] = append(grouped[r.Provider], r)
	}
	return grouped
}

// GroupResourcesByType groups resources by type
func GroupResourcesByType(resources []Resource) map[string][]Resource {
	grouped := make(map[string][]Resource)
	for _, r := range resources {
		grouped[r.Type] = append(grouped[r.Type], r)
	}
	return grouped
}

// GroupResourcesByRegion groups resources by region
func GroupResourcesByRegion(resources []Resource) map[string][]Resource {
	grouped := make(map[string][]Resource)
	for _, r := range resources {
		grouped[r.Region] = append(grouped[r.Region], r)
	}
	return grouped
}

// GetRegionsForProvider returns regions for a provider
func GetRegionsForProvider(provider string) []string {
	switch NormalizeProviderName(provider) {
	case "aws":
		return []string{"us-east-1", "us-west-2", "eu-west-1"}
	case "azure":
		return []string{"eastus", "westus", "northeurope"}
	case "gcp":
		return []string{"us-central1", "us-east1", "europe-west1"}
	case "digitalocean":
		return []string{"nyc1", "sfo1", "lon1"}
	default:
		return []string{}
	}
}

// ValidateResource validates a resource
func ValidateResource(r Resource) error {
	if r.ID == "" {
		return fmt.Errorf("resource ID cannot be empty")
	}
	if r.Type == "" {
		return fmt.Errorf("resource type cannot be empty")
	}
	if r.Provider == "" {
		return fmt.Errorf("resource provider cannot be empty")
	}
	return nil
}
