package rest

// ProviderConfig represents provider configuration
type ProviderConfig struct {
	Provider    string                 `json:"provider"`
	Credentials map[string]interface{} `json:"credentials"`
	Regions     []string               `json:"regions,omitempty"`
	Services    []string               `json:"services,omitempty"`
}
