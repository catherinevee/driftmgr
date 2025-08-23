package credentials

import (
	"context"
	"fmt"
	"os"
)

// Manager manages cloud provider credentials
type Manager struct {
	providers map[string]*ProviderCredentials
}

// ProviderCredentials holds credentials for a cloud provider
type ProviderCredentials struct {
	Provider    string
	Configured  bool
	Credentials map[string]string
}

// NewManager creates a new credentials manager
func NewManager() *Manager {
	return &Manager{
		providers: make(map[string]*ProviderCredentials),
	}
}

// LoadFromEnvironment loads credentials from environment variables
func (m *Manager) LoadFromEnvironment() {
	// AWS
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" {
		m.providers["aws"] = &ProviderCredentials{
			Provider:   "aws",
			Configured: true,
			Credentials: map[string]string{
				"access_key_id":     os.Getenv("AWS_ACCESS_KEY_ID"),
				"secret_access_key": os.Getenv("AWS_SECRET_ACCESS_KEY"),
				"session_token":     os.Getenv("AWS_SESSION_TOKEN"),
				"region":            os.Getenv("AWS_DEFAULT_REGION"),
			},
		}
	}

	// Azure
	if os.Getenv("AZURE_SUBSCRIPTION_ID") != "" {
		m.providers["azure"] = &ProviderCredentials{
			Provider:   "azure",
			Configured: true,
			Credentials: map[string]string{
				"subscription_id": os.Getenv("AZURE_SUBSCRIPTION_ID"),
				"tenant_id":       os.Getenv("AZURE_TENANT_ID"),
				"client_id":       os.Getenv("AZURE_CLIENT_ID"),
				"client_secret":   os.Getenv("AZURE_CLIENT_SECRET"),
			},
		}
	}

	// GCP
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		m.providers["gcp"] = &ProviderCredentials{
			Provider:   "gcp",
			Configured: true,
			Credentials: map[string]string{
				"credentials_file": os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
				"project_id":       os.Getenv("GCP_PROJECT_ID"),
			},
		}
	}

	// DigitalOcean
	if os.Getenv("DIGITALOCEAN_TOKEN") != "" {
		m.providers["digitalocean"] = &ProviderCredentials{
			Provider:   "digitalocean",
			Configured: true,
			Credentials: map[string]string{
				"token": os.Getenv("DIGITALOCEAN_TOKEN"),
			},
		}
	}
}

// IsConfigured checks if a provider has configured credentials
func (m *Manager) IsConfigured(ctx context.Context, provider string) (bool, error) {
	if creds, ok := m.providers[provider]; ok {
		return creds.Configured, nil
	}
	return false, nil
}

// GetProviders returns all configured providers
func (m *Manager) GetProviders() []string {
	var providers []string
	for name, creds := range m.providers {
		if creds.Configured {
			providers = append(providers, name)
		}
	}
	return providers
}

// ValidateCredentials validates credentials for a provider
func (m *Manager) ValidateCredentials(ctx context.Context, provider string) (bool, error) {
	if _, ok := m.providers[provider]; !ok {
		return false, fmt.Errorf("provider %s not configured", provider)
	}

	// Basic validation - just check if credentials exist
	// In a real implementation, this would make API calls to validate
	return true, nil
}

// GetCredentials returns credentials for a provider
func (m *Manager) GetCredentials(provider string) (*ProviderCredentials, error) {
	if creds, ok := m.providers[provider]; ok {
		return creds, nil
	}
	return nil, fmt.Errorf("provider %s not configured", provider)
}
