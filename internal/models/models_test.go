package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Validate(t *testing.T) {
	tests := []struct {
		name     string
		resource Resource
		wantErr  bool
	}{
		{
			name: "valid resource",
			resource: Resource{
				ID:       "test-id",
				Name:     "test-name",
				Type:     "aws_instance",
				Provider: "aws",
				Region:   "us-east-1",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			resource: Resource{
				Name:     "test-name",
				Type:     "aws_instance",
				Provider: "aws",
				Region:   "us-east-1",
			},
			wantErr: true,
		},
		{
			name: "missing Name",
			resource: Resource{
				ID:       "test-id",
				Type:     "aws_instance",
				Provider: "aws",
				Region:   "us-east-1",
			},
			wantErr: true,
		},
		{
			name: "missing Type",
			resource: Resource{
				ID:       "test-id",
				Name:     "test-name",
				Provider: "aws",
				Region:   "us-east-1",
			},
			wantErr: true,
		},
		{
			name: "missing Provider",
			resource: Resource{
				ID:     "test-id",
				Name:   "test-name",
				Type:   "aws_instance",
				Region: "us-east-1",
			},
			wantErr: true,
		},
		{
			name: "missing Region",
			resource: Resource{
				ID:       "test-id",
				Name:     "test-name",
				Type:     "aws_instance",
				Provider: "aws",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.resource.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDiscoveryConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  DiscoveryConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: DiscoveryConfig{
				Provider: "aws",
				Regions:  []string{"us-east-1"},
			},
			wantErr: false,
		},
		{
			name: "missing provider",
			config: DiscoveryConfig{
				Regions: []string{"us-east-1"},
			},
			wantErr: true,
		},
		{
			name: "empty regions",
			config: DiscoveryConfig{
				Provider: "aws",
				Regions:  []string{},
			},
			wantErr: true,
		},
		{
			name: "nil regions",
			config: DiscoveryConfig{
				Provider: "aws",
			},
			wantErr: true,
		},
		{
			name: "invalid provider",
			config: DiscoveryConfig{
				Provider: "invalid",
				Regions:  []string{"us-east-1"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestImportConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ImportConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ImportConfig{
				Resources:  []Resource{{ID: "test", Name: "test", Type: "aws_instance", Provider: "aws", Region: "us-east-1"}},
				OutputPath: "/path/to/output",
				Format:     "terraform",
			},
			wantErr: false,
		},
		{
			name: "empty resources",
			config: ImportConfig{
				OutputPath: "/path/to/output",
				Format:     "terraform",
			},
			wantErr: true,
		},
		{
			name: "missing output path",
			config: ImportConfig{
				Resources: []Resource{{ID: "test", Name: "test", Type: "aws_instance", Provider: "aws", Region: "us-east-1"}},
				Format:    "terraform",
			},
			wantErr: true,
		},
		{
			name: "missing format",
			config: ImportConfig{
				Resources:  []Resource{{ID: "test", Name: "test", Type: "aws_instance", Provider: "aws", Region: "us-east-1"}},
				OutputPath: "/path/to/output",
			},
			wantErr: true,
		},
		{
			name: "invalid resource",
			config: ImportConfig{
				Resources:  []Resource{{ID: "test"}}, // Invalid resource
				OutputPath: "/path/to/output",
				Format:     "terraform",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResource_String(t *testing.T) {
	resource := Resource{
		ID:       "i-1234567890abcdef0",
		Name:     "web-server",
		Type:     "aws_instance",
		Provider: "aws",
		Region:   "us-east-1",
	}

	str := resource.String()
	assert.Contains(t, str, "i-1234567890abcdef0")
	assert.Contains(t, str, "web-server")
	assert.Contains(t, str, "aws_instance")
	assert.Contains(t, str, "aws")
	assert.Contains(t, str, "us-east-1")
}

func TestDiscoveryConfig_String(t *testing.T) {
	config := DiscoveryConfig{
		Provider: "aws",
		Regions:  []string{"us-east-1", "us-west-2"},
	}

	str := config.String()
	assert.Contains(t, str, "aws")
	assert.Contains(t, str, "us-east-1")
	assert.Contains(t, str, "us-west-2")
}

func TestImportConfig_String(t *testing.T) {
	config := ImportConfig{
		Resources: []Resource{
			{ID: "test1", Name: "test1", Type: "aws_instance", Provider: "aws", Region: "us-east-1"},
			{ID: "test2", Name: "test2", Type: "aws_s3_bucket", Provider: "aws", Region: "us-east-1"},
		},
		OutputPath: "/path/to/output",
		Format:     "terraform",
	}

	str := config.String()
	assert.Contains(t, str, "/path/to/output")
	assert.Contains(t, str, "terraform")
	assert.Contains(t, str, "2 resources")
}

func TestResource_GetTerraformAddress(t *testing.T) {
	tests := []struct {
		name     string
		resource Resource
		expected string
	}{
		{
			name: "aws instance",
			resource: Resource{
				ID:   "i-1234567890abcdef0",
				Name: "web-server",
				Type: "aws_instance",
			},
			expected: "aws_instance.web_server",
		},
		{
			name: "azure vm",
			resource: Resource{
				ID:   "/subscriptions/.../virtualMachines/vm",
				Name: "test-vm",
				Type: "azurerm_virtual_machine",
			},
			expected: "azurerm_virtual_machine.test_vm",
		},
		{
			name: "gcp instance",
			resource: Resource{
				ID:   "projects/my-project/zones/us-central1-a/instances/my-instance",
				Name: "my-instance",
				Type: "google_compute_instance",
			},
			expected: "google_compute_instance.my_instance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.resource.GetTerraformAddress()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResource_NormalizeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "web-server",
			expected: "web_server",
		},
		{
			name:     "name with spaces",
			input:    "my web server",
			expected: "my_web_server",
		},
		{
			name:     "name with special characters",
			input:    "web-server@123!",
			expected: "web_server_123_",
		},
		{
			name:     "name starting with number",
			input:    "123-server",
			expected: "_123_server",
		},
		{
			name:     "uppercase name",
			input:    "WEB-SERVER",
			expected: "web_server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := Resource{Name: tt.input}
			result := resource.normalizeName()
			assert.Equal(t, tt.expected, result)
		})
	}
}
