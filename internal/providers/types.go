package providers

import (
	"context"
	"fmt"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// CloudProvider defines the interface for cloud providers
type CloudProvider interface {
	// Name returns the provider name (e.g., "aws", "azure", "gcp")
	Name() string

	// DiscoverResources discovers resources in the specified region
	DiscoverResources(ctx context.Context, region string) ([]models.Resource, error)

	// GetResource retrieves a specific resource by ID
	GetResource(ctx context.Context, resourceID string) (*models.Resource, error)

	// ValidateCredentials checks if the provider credentials are valid
	ValidateCredentials(ctx context.Context) error

	// ListRegions returns available regions for the provider
	ListRegions(ctx context.Context) ([]string, error)

	// SupportedResourceTypes returns the list of supported resource types
	SupportedResourceTypes() []string
}

// ProviderConfig contains configuration for a provider
type ProviderConfig struct {
	// Provider name
	Name string

	// Credentials
	Credentials map[string]string

	// Region to operate in
	Region string

	// Additional provider-specific options
	Options map[string]interface{}
}

// NotFoundError represents an error when a resource is not found
type NotFoundError struct {
	Provider   string
	ResourceID string
	Region     string
	Message    string
}

// Error implements the error interface
func (e *NotFoundError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("resource %s not found in %s %s", e.ResourceID, e.Provider, e.Region)
}

// IsNotFound returns true if the error is a NotFoundError
func (e *NotFoundError) IsNotFound() bool {
	return true
}

// NewNotFoundError creates a new NotFoundError
func NewNotFoundError(provider, resourceID, region string) *NotFoundError {
	return &NotFoundError{
		Provider:   provider,
		ResourceID: resourceID,
		Region:     region,
	}
}

// IsNotFoundError checks if an error is a NotFoundError
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*NotFoundError)
	return ok
}