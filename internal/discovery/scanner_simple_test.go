package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBackendConfig(t *testing.T) {
	config := BackendConfig{
		Type:     "s3",
		FilePath: "/terraform/backend.tf",
		Module:   "main",
		Config: map[string]interface{}{
			"bucket": "my-terraform-state",
			"key":    "prod/terraform.tfstate",
			"region": "us-east-1",
		},
	}

	assert.Equal(t, "s3", config.Type)
	assert.Equal(t, "/terraform/backend.tf", config.FilePath)
	assert.Equal(t, "main", config.Module)
	assert.Equal(t, "my-terraform-state", config.Config["bucket"])
}

func TestNewScannerSimple(t *testing.T) {
	tests := []struct {
		name    string
		rootDir string
		workers int
		want    *Scanner
	}{
		{
			name:    "valid scanner",
			rootDir: "/terraform",
			workers: 4,
			want: &Scanner{
				rootDir:     "/terraform",
				workers:     4,
				ignoreRules: []string{".terraform", ".git", ".terragrunt-cache"},
			},
		},
		{
			name:    "zero workers defaults to 1",
			rootDir: "/terraform",
			workers: 0,
			want: &Scanner{
				rootDir:     "/terraform",
				workers:     1,
				ignoreRules: []string{".terraform", ".git", ".terragrunt-cache"},
			},
		},
		{
			name:    "negative workers defaults to 1",
			rootDir: "/terraform",
			workers: -5,
			want: &Scanner{
				rootDir:     "/terraform",
				workers:     1,
				ignoreRules: []string{".terraform", ".git", ".terragrunt-cache"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.rootDir, tt.workers)
			assert.Equal(t, tt.want.rootDir, scanner.rootDir)
			assert.Equal(t, tt.want.workers, scanner.workers)
			assert.Equal(t, tt.want.ignoreRules, scanner.ignoreRules)
			assert.NotNil(t, scanner.backends)
		})
	}
}

func TestScanner_GetBackendsSimple(t *testing.T) {
	scanner := NewScanner("/terraform", 4)

	// Add some test backends
	scanner.backends = []BackendConfig{
		{
			Type:     "s3",
			FilePath: "/terraform/backend.tf",
			Module:   "main",
			Config: map[string]interface{}{
				"bucket": "my-terraform-state",
			},
		},
		{
			Type:     "azurerm",
			FilePath: "/terraform/azure/backend.tf",
			Module:   "azure",
			Config: map[string]interface{}{
				"storage_account_name": "tfstate",
			},
		},
		{
			Type:     "gcs",
			FilePath: "/terraform/gcp/backend.tf",
			Module:   "gcp",
			Config: map[string]interface{}{
				"bucket": "gcp-terraform-state",
			},
		},
	}

	// Test GetBackendsByType
	s3Backends := scanner.GetBackendsByType("s3")
	assert.Len(t, s3Backends, 1)
	assert.Equal(t, "s3", s3Backends[0].Type)

	azureBackends := scanner.GetBackendsByType("azurerm")
	assert.Len(t, azureBackends, 1)
	assert.Equal(t, "azurerm", azureBackends[0].Type)

	gcsBackends := scanner.GetBackendsByType("gcs")
	assert.Len(t, gcsBackends, 1)
	assert.Equal(t, "gcs", gcsBackends[0].Type)

	// Test non-existent type
	localBackends := scanner.GetBackendsByType("local")
	assert.Len(t, localBackends, 0)
}

func TestBackendTypes(t *testing.T) {
	// Test that backend types are correctly handled
	validTypes := []string{"s3", "azurerm", "gcs", "remote", "consul", "etcd", "http"}

	for _, backendType := range validTypes {
		config := BackendConfig{
			Type: backendType,
		}
		assert.Equal(t, backendType, config.Type)
	}
}

func TestScanner_GetUniqueBackends(t *testing.T) {
	scanner := NewScanner("/terraform", 4)

	// Add duplicate backends
	scanner.backends = []BackendConfig{
		{
			Type:     "s3",
			FilePath: "/terraform/backend.tf",
			Module:   "main",
			Config: map[string]interface{}{
				"bucket": "my-terraform-state",
				"key":    "prod/terraform.tfstate",
			},
		},
		{
			Type:     "s3",
			FilePath: "/terraform/backend.tf",
			Module:   "main",
			Config: map[string]interface{}{
				"bucket": "my-terraform-state",
				"key":    "prod/terraform.tfstate",
			},
		},
		{
			Type:     "azurerm",
			FilePath: "/terraform/azure/backend.tf",
			Module:   "azure",
			Config: map[string]interface{}{
				"storage_account_name": "tfstate",
			},
		},
	}

	uniqueBackends := scanner.GetUniqueBackends()
	assert.Len(t, uniqueBackends, 2) // Should have 2 unique backends (s3 and azurerm)

	// Verify the unique backends
	types := make(map[string]bool)
	for _, backend := range uniqueBackends {
		types[backend.Type] = true
	}
	assert.True(t, types["s3"])
	assert.True(t, types["azurerm"])
}
