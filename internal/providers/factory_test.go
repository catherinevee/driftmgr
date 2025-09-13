package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		config       map[string]interface{}
		expectError  bool
	}{
		{
			name:         "AWS provider",
			providerName: "aws",
			config: map[string]interface{}{
				"region": "us-east-1",
			},
			expectError: false,
		},
		{
			name:         "AWS provider lowercase",
			providerName: "AWS",
			config: map[string]interface{}{
				"region": "us-west-2",
			},
			expectError: false,
		},
		{
			name:         "Azure provider",
			providerName: "azure",
			config: map[string]interface{}{
				"subscription_id": "12345-67890",
				"resource_group":  "my-rg",
			},
			expectError: false,
		},
		{
			name:         "GCP provider",
			providerName: "gcp",
			config: map[string]interface{}{
				"project_id": "my-project",
			},
			expectError: false,
		},
		{
			name:         "DigitalOcean provider",
			providerName: "digitalocean",
			config: map[string]interface{}{
				"region": "nyc1",
			},
			expectError: false,
		},
		{
			name:         "Unsupported provider",
			providerName: "unsupported",
			config:       map[string]interface{}{},
			expectError:  true,
		},
		{
			name:         "Empty provider name",
			providerName: "",
			config:       map[string]interface{}{},
			expectError:  true,
		},
		{
			name:         "AWS with empty config",
			providerName: "aws",
			config:       map[string]interface{}{},
			expectError:  false,
		},
		{
			name:         "AWS with nil config",
			providerName: "aws",
			config:       nil,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.providerName, tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, provider)
			}
		})
	}
}

func TestNewProvider_ConfigExtraction(t *testing.T) {
	t.Run("AWS region extraction", func(t *testing.T) {
		config := map[string]interface{}{
			"region": "eu-west-1",
			"profile": "default",
		}
		provider, err := NewProvider("aws", config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("Azure subscription extraction", func(t *testing.T) {
		config := map[string]interface{}{
			"subscription_id": "sub-12345",
			"resource_group": "test-rg",
			"tenant_id": "tenant-123",
		}
		provider, err := NewProvider("azure", config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("GCP project extraction", func(t *testing.T) {
		config := map[string]interface{}{
			"project_id": "gcp-project-123",
			"zone": "us-central1-a",
		}
		provider, err := NewProvider("gcp", config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("DigitalOcean region extraction", func(t *testing.T) {
		config := map[string]interface{}{
			"region": "sfo3",
			"token": "do-token",
		}
		provider, err := NewProvider("digitalocean", config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})
}

func TestNewProvider_CaseInsensitive(t *testing.T) {
	providers := []string{"AWS", "aws", "Aws", "Azure", "AZURE", "azure", "GCP", "gcp", "Gcp", "DigitalOcean", "digitalocean"}

	for _, name := range providers {
		t.Run(name, func(t *testing.T) {
			provider, err := NewProvider(name, nil)

			// These should all succeed (not be unsupported)
			if strings.ToLower(name) == "aws" || strings.ToLower(name) == "azure" ||
			   strings.ToLower(name) == "gcp" || strings.ToLower(name) == "digitalocean" {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
			}
		})
	}
}

func BenchmarkNewProvider(b *testing.B) {
	config := map[string]interface{}{
		"region": "us-east-1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewProvider("aws", config)
	}
}