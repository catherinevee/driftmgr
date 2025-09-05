package types

import "context"

// CloudResource represents a resource discovered from cloud provider
type CloudResource struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Provider   string                 `json:"provider"`
	Region     string                 `json:"region,omitempty"`
	Tags       map[string]string      `json:"tags,omitempty"`
	Attributes map[string]interface{} `json:"attributes"`
}

// CloudProvider defines the interface for cloud providers
type CloudProvider interface {
	Name() string
	Connect(ctx context.Context) error
	GetResource(ctx context.Context, resourceType string, resourceID string) (*CloudResource, error)
	ListResources(ctx context.Context, resourceType string) ([]*CloudResource, error)
	ResourceExists(ctx context.Context, resourceType string, resourceID string) (bool, error)
}
