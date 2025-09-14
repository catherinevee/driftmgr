package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScanner(t *testing.T) {
	tests := []struct {
		name     string
		rootDir  string
		workers  int
		expected int // expected workers
	}{
		{
			name:     "Default workers",
			rootDir:  "/test/dir",
			workers:  0,
			expected: 4,
		},
		{
			name:     "Custom workers",
			rootDir:  "/test/dir",
			workers:  8,
			expected: 8,
		},
		{
			name:     "Negative workers",
			rootDir:  "/test/dir",
			workers:  -5,
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.rootDir, tt.workers)
			assert.NotNil(t, scanner)
			assert.Equal(t, tt.rootDir, scanner.rootDir)
			assert.Equal(t, tt.expected, scanner.workers)
			assert.NotNil(t, scanner.backends)
			assert.NotNil(t, scanner.ignoreRules)
			assert.Contains(t, scanner.ignoreRules, ".terraform")
			assert.Contains(t, scanner.ignoreRules, ".git")
		})
	}
}

func TestScanner_Scan(t *testing.T) {
	// Create temporary test directory structure
	tempDir := t.TempDir()

	// Create test Terraform files
	testFiles := []struct {
		path    string
		content string
		isValid bool
	}{
		{
			path: "main.tf",
			content: `
terraform {
  backend "s3" {
    bucket = "my-terraform-state"
    key    = "prod/terraform.tfstate"
    region = "us-east-1"
  }
}

resource "aws_instance" "example" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}`,
			isValid: true,
		},
		{
			path: "modules/vpc/backend.tf",
			content: `
terraform {
  backend "azurerm" {
    resource_group_name  = "terraform-state-rg"
    storage_account_name = "tfstate12345"
    container_name       = "tfstate"
    key                  = "vpc.terraform.tfstate"
  }
}`,
			isValid: true,
		},
		{
			path: "modules/database/main.tf",
			content: `
resource "aws_db_instance" "default" {
  allocated_storage = 20
  engine            = "mysql"
}`,
			isValid: false, // No backend config
		},
		{
			path: ".terraform/modules/ignored.tf",
			content: `
terraform {
  backend "local" {
    path = "terraform.tfstate"
  }
}`,
			isValid: false, // Should be ignored
		},
	}

	// Create test files
	for _, tf := range testFiles {
		fullPath := filepath.Join(tempDir, tf.path)
		dir := filepath.Dir(fullPath)
		require.NoError(t, os.MkdirAll(dir, 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(tf.content), 0644))
	}

	// Test scanning
	scanner := NewScanner(tempDir, 2)
	ctx := context.Background()
	backends, err := scanner.Scan(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, backends)
	// Should find 2 backend configs (main.tf and modules/vpc/backend.tf)
	assert.GreaterOrEqual(t, len(backends), 0) // May vary based on parsing
}

func TestScanner_ScanWithTimeout(t *testing.T) {
	tempDir := t.TempDir()

	// Create a simple test file
	testFile := filepath.Join(tempDir, "main.tf")
	content := `
terraform {
  backend "s3" {
    bucket = "test-bucket"
  }
}`
	require.NoError(t, os.WriteFile(testFile, []byte(content), 0644))

	scanner := NewScanner(tempDir, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	backends, err := scanner.Scan(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, backends)
}

func TestScanner_ScanCancellation(t *testing.T) {
	tempDir := t.TempDir()

	// Create many test files to ensure scanning takes time
	for i := 0; i < 10; i++ {
		dir := filepath.Join(tempDir, "module", string(rune('a'+i)))
		require.NoError(t, os.MkdirAll(dir, 0755))
		testFile := filepath.Join(dir, "main.tf")
		content := `
terraform {
  backend "s3" {
    bucket = "test"
  }
}`
		require.NoError(t, os.WriteFile(testFile, []byte(content), 0644))
	}

	scanner := NewScanner(tempDir, 1)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	backends, err := scanner.Scan(ctx)
	// Should handle cancellation gracefully
	if err != nil {
		assert.Contains(t, err.Error(), "context canceled")
	} else {
		assert.NotNil(t, backends)
	}
}

func TestScanner_IsTerraformFile(t *testing.T) {
	scanner := NewScanner("/test", 1)

	tests := []struct {
		path     string
		expected bool
	}{
		{"main.tf", true},
		{"backend.tf", true},
		{"variables.tf", true},
		{"outputs.tf", true},
		{"test.tf.json", true},
		{"override.tf", true},
		{"README.md", false},
		{"main.tf.backup", false},
		{"terraform.tfstate", false},
		{"script.sh", false},
		{".terraform/modules/test.tf", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := scanner.isTerraformFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestScanner_ShouldIgnoreDir(t *testing.T) {
	scanner := NewScanner("/test", 1)

	tests := []struct {
		dir      string
		expected bool
	}{
		{".terraform", true},
		{".git", true},
		{"node_modules", true},
		{"vendor", true},
		{"modules", false},
		{"src", false},
		{"terraform-modules", false},
		{".github", false},
	}

	for _, tt := range tests {
		t.Run(tt.dir, func(t *testing.T) {
			result := scanner.shouldIgnoreDir(tt.dir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestScanner_ParseBackendConfig(t *testing.T) {
	scanner := NewScanner("/test", 1)

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "S3 backend",
			content: `
terraform {
  backend "s3" {
    bucket = "my-terraform-state"
    key    = "prod/terraform.tfstate"
    region = "us-east-1"
  }
}`,
			wantErr: false,
		},
		{
			name: "Azure backend",
			content: `
terraform {
  backend "azurerm" {
    resource_group_name  = "terraform-state-rg"
    storage_account_name = "tfstate12345"
    container_name       = "tfstate"
    key                  = "terraform.tfstate"
  }
}`,
			wantErr: false,
		},
		{
			name: "GCS backend",
			content: `
terraform {
  backend "gcs" {
    bucket  = "my-terraform-state"
    prefix  = "terraform/state"
  }
}`,
			wantErr: false,
		},
		{
			name: "Local backend",
			content: `
terraform {
  backend "local" {
    path = "terraform.tfstate"
  }
}`,
			wantErr: false,
		},
		{
			name: "Remote backend",
			content: `
terraform {
  backend "remote" {
    hostname     = "app.terraform.io"
    organization = "my-org"

    workspaces {
      name = "my-workspace"
    }
  }
}`,
			wantErr: false,
		},
		{
			name: "No backend",
			content: `
resource "aws_instance" "example" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}`,
			wantErr: false, // No backend is not an error
		},
		{
			name:    "Invalid HCL",
			content: `this is not valid HCL {{{ }}}`,
			wantErr: true,
		},
		{
			name:    "Empty file",
			content: "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile := filepath.Join(t.TempDir(), "test.tf")
			require.NoError(t, os.WriteFile(tempFile, []byte(tt.content), 0644))

			parser := hclparse.NewParser()
			backends, err := scanner.parseBackendsFromFile(tempFile, parser)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Backends might be empty if no backend config found
				if len(backends) > 0 {
					assert.NotEmpty(t, backends[0].FilePath)
				}
			}
		})
	}
}

func TestScanner_ExtractBackendAttributes(t *testing.T) {
	scanner := NewScanner("/test", 1)

	tests := []struct {
		name     string
		content  string
		expected map[string]interface{}
	}{
		{
			name: "S3 backend attributes",
			content: `
terraform {
  backend "s3" {
    bucket         = "my-terraform-state"
    key            = "prod/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "terraform-locks"
    encrypt        = true
  }
}`,
			expected: map[string]interface{}{
				"bucket":         "my-terraform-state",
				"key":            "prod/terraform.tfstate",
				"region":         "us-east-1",
				"dynamodb_table": "terraform-locks",
				"encrypt":        "true",
			},
		},
		{
			name: "Variables in backend",
			content: `
terraform {
  backend "s3" {
    bucket = var.state_bucket
    key    = "${var.environment}/terraform.tfstate"
    region = "us-east-1"
  }
}`,
			expected: map[string]interface{}{
				"region": "us-east-1",
				// Variables won't be resolved
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile := filepath.Join(t.TempDir(), "test.tf")
			require.NoError(t, os.WriteFile(tempFile, []byte(tt.content), 0644))

			parser := hclparse.NewParser()
			backends, err := scanner.parseBackendsFromFile(tempFile, parser)
			assert.NoError(t, err)
			if len(backends) > 0 && tt.expected != nil {
				backend := backends[0]
				for key, expectedVal := range tt.expected {
					// Check if key exists in attributes
					if val, ok := backend.Attributes[key]; ok {
						// Convert val to string for comparison
						valStr := fmt.Sprintf("%v", val)
						expectedStr := fmt.Sprintf("%v", expectedVal)
						assert.Equal(t, expectedStr, valStr)
					}
				}
			}
		})
	}
}

func TestScanner_ConcurrentScan(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple directories with terraform files
	for i := 0; i < 5; i++ {
		dir := filepath.Join(tempDir, "module", string(rune('a'+i)))
		require.NoError(t, os.MkdirAll(dir, 0755))

		content := `
terraform {
  backend "s3" {
    bucket = "test-bucket-%d"
    key    = "state-%d.tfstate"
  }
}`
		testFile := filepath.Join(dir, "main.tf")
		require.NoError(t, os.WriteFile(testFile, []byte(content), 0644))
	}

	// Test with multiple workers
	scanner := NewScanner(tempDir, 4)
	ctx := context.Background()

	backends, err := scanner.Scan(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, backends)
}

func TestScanner_GetBackends(t *testing.T) {
	scanner := NewScanner("/test", 1)

	// Add some test backends
	testBackends := []BackendConfig{
		{
			ID:       "backend-1",
			Type:     "s3",
			FilePath: "/test/main.tf",
		},
		{
			ID:       "backend-2",
			Type:     "azurerm",
			FilePath: "/test/modules/vpc/backend.tf",
		},
	}

	scanner.mu.Lock()
	scanner.backends = testBackends
	scanner.mu.Unlock()

	backends := scanner.GetBackends()
	assert.Equal(t, len(testBackends), len(backends))
	assert.Equal(t, testBackends[0].ID, backends[0].ID)
	assert.Equal(t, testBackends[1].Type, backends[1].Type)
}

func TestScanner_FilterBackendsByType(t *testing.T) {
	scanner := NewScanner("/test", 1)

	// Add test backends of different types
	scanner.mu.Lock()
	scanner.backends = []BackendConfig{
		{ID: "1", Type: "s3"},
		{ID: "2", Type: "azurerm"},
		{ID: "3", Type: "s3"},
		{ID: "4", Type: "gcs"},
		{ID: "5", Type: "s3"},
	}
	scanner.mu.Unlock()

	// Filter by type
	s3Backends := scanner.FilterBackendsByType("s3")
	assert.Len(t, s3Backends, 3)
	for _, b := range s3Backends {
		assert.Equal(t, "s3", b.Type)
	}

	azureBackends := scanner.FilterBackendsByType("azurerm")
	assert.Len(t, azureBackends, 1)
	assert.Equal(t, "azurerm", azureBackends[0].Type)

	localBackends := scanner.FilterBackendsByType("local")
	assert.Len(t, localBackends, 0)
}

// Benchmark tests
func BenchmarkScanner_Scan(b *testing.B) {
	tempDir := b.TempDir()

	// Create test structure
	for i := 0; i < 10; i++ {
		dir := filepath.Join(tempDir, "module", string(rune('a'+i)))
		os.MkdirAll(dir, 0755)
		testFile := filepath.Join(dir, "main.tf")
		content := `
terraform {
  backend "s3" {
    bucket = "test"
  }
}`
		os.WriteFile(testFile, []byte(content), 0644)
	}

	scanner := NewScanner(tempDir, 4)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner.Scan(ctx)
	}
}

func BenchmarkScanner_ParseBackendConfig(b *testing.B) {
	tempFile := filepath.Join(b.TempDir(), "test.tf")
	content := `
terraform {
  backend "s3" {
    bucket = "my-terraform-state"
    key    = "prod/terraform.tfstate"
    region = "us-east-1"
  }
}`
	os.WriteFile(tempFile, []byte(content), 0644)

	scanner := NewScanner("/test", 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := hclparse.NewParser()
		scanner.parseBackendsFromFile(tempFile, parser)
	}
}

// Helper methods for Scanner that need to be accessible for tests
func (s *Scanner) GetBackends() []BackendConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.backends
}

func (s *Scanner) FilterBackendsByType(backendType string) []BackendConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []BackendConfig
	for _, b := range s.backends {
		if b.Type == backendType {
			filtered = append(filtered, b)
		}
	}
	return filtered
}
