package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBackendConfig(t *testing.T) {
	config := BackendConfig{
		ID:       "backend-1",
		Type:     "s3",
		FilePath: "/terraform/main.tf",
		Module:   "vpc",
		Workspace: "production",
		ConfigPath: "/terraform",
		Attributes: map[string]interface{}{
			"bucket": "terraform-state",
			"key":    "vpc/terraform.tfstate",
			"region": "us-east-1",
		},
		Config: map[string]interface{}{
			"encrypt": true,
		},
	}

	assert.Equal(t, "backend-1", config.ID)
	assert.Equal(t, "s3", config.Type)
	assert.Equal(t, "/terraform/main.tf", config.FilePath)
	assert.Equal(t, "vpc", config.Module)
	assert.Equal(t, "production", config.Workspace)
	assert.NotNil(t, config.Attributes)
	assert.Equal(t, "terraform-state", config.Attributes["bucket"])
}

func TestNewScanner(t *testing.T) {
	tests := []struct {
		name            string
		rootDir         string
		workers         int
		expectedWorkers int
	}{
		{
			name:            "default workers",
			rootDir:         "/terraform",
			workers:         0,
			expectedWorkers: 4,
		},
		{
			name:            "negative workers",
			rootDir:         "/terraform",
			workers:         -1,
			expectedWorkers: 4,
		},
		{
			name:            "custom workers",
			rootDir:         "/terraform",
			workers:         8,
			expectedWorkers: 8,
		},
		{
			name:            "single worker",
			rootDir:         "/terraform",
			workers:         1,
			expectedWorkers: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.rootDir, tt.workers)

			assert.NotNil(t, scanner)
			assert.Equal(t, tt.rootDir, scanner.rootDir)
			assert.Equal(t, tt.expectedWorkers, scanner.workers)
			assert.NotNil(t, scanner.backends)
			assert.NotNil(t, scanner.ignoreRules)
			assert.Contains(t, scanner.ignoreRules, ".terraform")
			assert.Contains(t, scanner.ignoreRules, ".git")
		})
	}
}

func TestScanner_AddIgnoreRule(t *testing.T) {
	scanner := NewScanner("/terraform", 4)

	rules := []string{
		"*.backup",
		"*.tmp",
		"node_modules",
		"vendor",
	}

	for _, rule := range rules {
		scanner.AddIgnoreRule(rule)
	}

	// Check that default rules are still present
	assert.Contains(t, scanner.ignoreRules, ".terraform")
	assert.Contains(t, scanner.ignoreRules, ".git")

	// Check that new rules were added
	for _, rule := range rules {
		assert.Contains(t, scanner.ignoreRules, rule)
	}
}

func TestScanner_ShouldIgnore(t *testing.T) {
	scanner := NewScanner("/terraform", 4)
	scanner.AddIgnoreRule("*.backup")
	scanner.AddIgnoreRule("temp/")

	tests := []struct {
		name       string
		path       string
		shouldIgnore bool
	}{
		{
			name:       "terraform directory",
			path:       "/project/.terraform/modules",
			shouldIgnore: true,
		},
		{
			name:       "git directory",
			path:       "/project/.git/config",
			shouldIgnore: true,
		},
		{
			name:       "backup file",
			path:       "/project/main.tf.backup",
			shouldIgnore: true,
		},
		{
			name:       "temp directory",
			path:       "/project/temp/test.tf",
			shouldIgnore: true,
		},
		{
			name:       "valid terraform file",
			path:       "/project/main.tf",
			shouldIgnore: false,
		},
		{
			name:       "valid module",
			path:       "/project/modules/vpc/main.tf",
			shouldIgnore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scanner.shouldIgnore(tt.path)
			assert.Equal(t, tt.shouldIgnore, result)
		})
	}
}

func TestScanner_GetBackends(t *testing.T) {
	scanner := NewScanner("/terraform", 4)

	// Add some test backends
	testBackends := []BackendConfig{
		{
			ID:   "backend-1",
			Type: "s3",
		},
		{
			ID:   "backend-2",
			Type: "azurerm",
		},
		{
			ID:   "backend-3",
			Type: "gcs",
		},
	}

	scanner.mu.Lock()
	scanner.backends = testBackends
	scanner.mu.Unlock()

	backends := scanner.GetBackends()
	assert.Len(t, backends, 3)
	assert.Equal(t, "backend-1", backends[0].ID)
	assert.Equal(t, "s3", backends[0].Type)
}

func TestBackendTypes(t *testing.T) {
	backends := []struct {
		name     string
		backendType string
		attributes map[string]interface{}
	}{
		{
			name:     "S3 backend",
			backendType: "s3",
			attributes: map[string]interface{}{
				"bucket": "my-bucket",
				"key":    "terraform.tfstate",
				"region": "us-east-1",
			},
		},
		{
			name:     "Azure backend",
			backendType: "azurerm",
			attributes: map[string]interface{}{
				"storage_account_name": "mystorageaccount",
				"container_name":       "tfstate",
				"key":                  "terraform.tfstate",
			},
		},
		{
			name:     "GCS backend",
			backendType: "gcs",
			attributes: map[string]interface{}{
				"bucket": "my-gcs-bucket",
				"prefix": "terraform/state",
			},
		},
		{
			name:     "Local backend",
			backendType: "local",
			attributes: map[string]interface{}{
				"path": "./terraform.tfstate",
			},
		},
		{
			name:     "Remote backend",
			backendType: "remote",
			attributes: map[string]interface{}{
				"organization": "my-org",
				"workspaces": map[string]string{
					"name": "my-workspace",
				},
			},
		},
	}

	for _, backend := range backends {
		t.Run(backend.name, func(t *testing.T) {
			config := BackendConfig{
				Type:       backend.backendType,
				Attributes: backend.attributes,
			}

			assert.Equal(t, backend.backendType, config.Type)
			assert.NotNil(t, config.Attributes)

			// Verify essential attributes exist
			switch backend.backendType {
			case "s3":
				assert.NotNil(t, config.Attributes["bucket"])
				assert.NotNil(t, config.Attributes["key"])
			case "azurerm":
				assert.NotNil(t, config.Attributes["storage_account_name"])
				assert.NotNil(t, config.Attributes["container_name"])
			case "gcs":
				assert.NotNil(t, config.Attributes["bucket"])
			case "local":
				assert.NotNil(t, config.Attributes["path"])
			case "remote":
				assert.NotNil(t, config.Attributes["organization"])
			}
		})
	}
}

func BenchmarkNewScanner(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewScanner("/terraform", 4)
	}
}

func BenchmarkScanner_ShouldIgnore(b *testing.B) {
	scanner := NewScanner("/terraform", 4)
	scanner.AddIgnoreRule("*.backup")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scanner.shouldIgnore("/project/main.tf")
		_ = scanner.shouldIgnore("/project/.terraform/modules/vpc")
		_ = scanner.shouldIgnore("/project/backup.tf.backup")
	}
}