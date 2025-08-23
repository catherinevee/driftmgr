package cloud

// DiscoveryResult represents a cloud discovery result
type DiscoveryResult struct {
	Provider      string                 `json:"provider"`
	Account       string                 `json:"account"`
	Region        string                 `json:"region"`
	ResourceType  string                 `json:"resource_type"`
	ResourceCount int                    `json:"resource_count"`
	Resources     []interface{}          `json:"resources"`
	Metadata      map[string]interface{} `json:"metadata"`
}
