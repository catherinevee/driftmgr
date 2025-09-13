package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStateParser(t *testing.T) {
	parser := NewStateParser()
	assert.NotNil(t, parser, "Expected non-nil parser")
}

func TestStateParser_Parse(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		wantErr     bool
		errContains string
		validate    func(t *testing.T, state *StateFile)
	}{
		{
			name: "Valid Terraform state v4",
			input: map[string]interface{}{
				"version":          4,
				"terraform_version": "1.5.0",
				"serial":           1,
				"lineage":          "test-lineage",
				"outputs":          map[string]interface{}{},
				"resources": []interface{}{
					map[string]interface{}{
						"mode":     "managed",
						"type":     "aws_instance",
						"name":     "example",
						"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
						"instances": []interface{}{
							map[string]interface{}{
								"attributes": map[string]interface{}{
									"id":            "i-1234567890",
									"instance_type": "t2.micro",
									"ami":           "ami-12345678",
									"tags": map[string]interface{}{
										"Name": "test-instance",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, state *StateFile) {
				assert.Equal(t, 4, state.Version)
				assert.Equal(t, "1.5.0", state.TerraformVersion)
				assert.Equal(t, 1, state.Serial)
				assert.Equal(t, "test-lineage", state.Lineage)
				assert.Len(t, state.Resources, 1)
				assert.Equal(t, "aws_instance", state.Resources[0].Type)
				assert.Equal(t, "example", state.Resources[0].Name)
			},
		},
		{
			name: "Valid Terraform state with multiple resources",
			input: map[string]interface{}{
				"version":          4,
				"terraform_version": "1.0.0",
				"serial":           5,
				"lineage":          "multi-resource",
				"outputs": map[string]interface{}{
					"instance_id": map[string]interface{}{
						"value": "i-1234567890",
						"type":  "string",
					},
				},
				"resources": []interface{}{
					map[string]interface{}{
						"mode":     "managed",
						"type":     "aws_instance",
						"name":     "web",
						"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
						"instances": []interface{}{
							map[string]interface{}{
								"attributes": map[string]interface{}{
									"id":            "i-1234567890",
									"instance_type": "t3.medium",
								},
							},
						},
					},
					map[string]interface{}{
						"mode":     "managed",
						"type":     "aws_security_group",
						"name":     "web_sg",
						"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
						"instances": []interface{}{
							map[string]interface{}{
								"attributes": map[string]interface{}{
									"id":          "sg-1234567890",
									"name":        "web-security-group",
									"description": "Security group for web servers",
								},
							},
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, state *StateFile) {
				assert.Equal(t, 4, state.Version)
				assert.Equal(t, 5, state.Serial)
				assert.Len(t, state.Resources, 2)
				assert.Equal(t, "aws_instance", state.Resources[0].Type)
				assert.Equal(t, "aws_security_group", state.Resources[1].Type)
				assert.NotNil(t, state.Outputs)
			},
		},
		{
			name: "Empty state",
			input: map[string]interface{}{
				"version":          4,
				"terraform_version": "1.5.0",
				"serial":           0,
				"lineage":          "empty",
				"outputs":          map[string]interface{}{},
				"resources":        []interface{}{},
			},
			wantErr: false,
			validate: func(t *testing.T, state *StateFile) {
				assert.Equal(t, 4, state.Version)
				assert.Empty(t, state.Resources)
			},
		},
		{
			name: "State with data resources",
			input: map[string]interface{}{
				"version":          4,
				"terraform_version": "1.5.0",
				"serial":           1,
				"lineage":          "data-resources",
				"outputs":          map[string]interface{}{},
				"resources": []interface{}{
					map[string]interface{}{
						"mode":     "data",
						"type":     "aws_ami",
						"name":     "ubuntu",
						"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
						"instances": []interface{}{
							map[string]interface{}{
								"attributes": map[string]interface{}{
									"id":           "ami-ubuntu-latest",
									"name":         "ubuntu-20.04",
									"architecture": "x86_64",
								},
							},
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, state *StateFile) {
				assert.Len(t, state.Resources, 1)
				assert.Equal(t, "data", state.Resources[0].Mode)
				assert.Equal(t, "aws_ami", state.Resources[0].Type)
			},
		},
		{
			name:        "Invalid JSON",
			input:       "not-valid-json{",
			wantErr:     true,
			errContains: "failed to parse state",
		},
		{
			name:        "Nil data",
			input:       nil,
			wantErr:     true,
			errContains: "failed to parse state",
		},
		{
			name: "State with modules",
			input: map[string]interface{}{
				"version":          4,
				"terraform_version": "1.5.0",
				"serial":           1,
				"lineage":          "with-modules",
				"outputs":          map[string]interface{}{},
				"resources": []interface{}{
					map[string]interface{}{
						"module":   "module.vpc",
						"mode":     "managed",
						"type":     "aws_vpc",
						"name":     "main",
						"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
						"instances": []interface{}{
							map[string]interface{}{
								"attributes": map[string]interface{}{
									"id":         "vpc-12345",
									"cidr_block": "10.0.0.0/16",
								},
							},
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, state *StateFile) {
				assert.Len(t, state.Resources, 1)
				assert.Equal(t, "module.vpc", state.Resources[0].Module)
				assert.Equal(t, "aws_vpc", state.Resources[0].Type)
			},
		},
		{
			name: "State with multiple instances",
			input: map[string]interface{}{
				"version":          4,
				"terraform_version": "1.5.0",
				"serial":           1,
				"lineage":          "multi-instance",
				"outputs":          map[string]interface{}{},
				"resources": []interface{}{
					map[string]interface{}{
						"mode":     "managed",
						"type":     "aws_instance",
						"name":     "cluster",
						"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
						"instances": []interface{}{
							map[string]interface{}{
								"index_key": 0,
								"attributes": map[string]interface{}{
									"id":            "i-111111",
									"instance_type": "t2.micro",
								},
							},
							map[string]interface{}{
								"index_key": 1,
								"attributes": map[string]interface{}{
									"id":            "i-222222",
									"instance_type": "t2.micro",
								},
							},
							map[string]interface{}{
								"index_key": 2,
								"attributes": map[string]interface{}{
									"id":            "i-333333",
									"instance_type": "t2.micro",
								},
							},
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, state *StateFile) {
				assert.Len(t, state.Resources, 1)
				assert.Len(t, state.Resources[0].Instances, 3)
				assert.Equal(t, "i-111111", state.Resources[0].Instances[0].Attributes["id"])
				assert.Equal(t, "i-222222", state.Resources[0].Instances[1].Attributes["id"])
				assert.Equal(t, "i-333333", state.Resources[0].Instances[2].Attributes["id"])
			},
		},
		{
			name: "State with sensitive attributes",
			input: map[string]interface{}{
				"version":          4,
				"terraform_version": "1.5.0",
				"serial":           1,
				"lineage":          "sensitive",
				"outputs":          map[string]interface{}{},
				"resources": []interface{}{
					map[string]interface{}{
						"mode":     "managed",
						"type":     "aws_db_instance",
						"name":     "database",
						"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
						"instances": []interface{}{
							map[string]interface{}{
								"attributes": map[string]interface{}{
									"id":       "db-12345",
									"password": "supersecret",
									"username": "admin",
								},
								"sensitive_attributes": []interface{}{
									[]interface{}{"password"},
								},
							},
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, state *StateFile) {
				assert.Len(t, state.Resources, 1)
				assert.Equal(t, "aws_db_instance", state.Resources[0].Type)
				assert.Equal(t, "supersecret", state.Resources[0].Instances[0].Attributes["password"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewStateParser()

			var data []byte
			var err error

			if tt.input == nil {
				data = nil
			} else if str, ok := tt.input.(string); ok {
				data = []byte(str)
			} else {
				data, err = json.Marshal(tt.input)
				require.NoError(t, err, "Failed to marshal test input")
			}

			result, err := parser.Parse(data)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestStateParser_ParseFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		setupFile   func() string
		wantErr     bool
		errContains string
		validate    func(t *testing.T, state *StateFile)
	}{
		{
			name: "Valid state file",
			setupFile: func() string {
				filePath := filepath.Join(tempDir, "valid.tfstate")
				content := map[string]interface{}{
					"version":          4,
					"terraform_version": "1.5.0",
					"serial":           1,
					"lineage":          "test",
					"outputs":          map[string]interface{}{},
					"resources":        []interface{}{},
				}
				data, _ := json.Marshal(content)
				err := os.WriteFile(filePath, data, 0644)
				require.NoError(t, err)
				return filePath
			},
			wantErr: false,
			validate: func(t *testing.T, state *StateFile) {
				assert.NotEmpty(t, state.Path)
				assert.Equal(t, 4, state.Version)
				assert.Contains(t, state.Path, "valid.tfstate")
			},
		},
		{
			name: "File not found",
			setupFile: func() string {
				return filepath.Join(tempDir, "nonexistent.tfstate")
			},
			wantErr:     true,
			errContains: "failed to read state file",
		},
		{
			name: "Invalid JSON in file",
			setupFile: func() string {
				filePath := filepath.Join(tempDir, "invalid.tfstate")
				err := os.WriteFile(filePath, []byte("not-json{"), 0644)
				require.NoError(t, err)
				return filePath
			},
			wantErr:     true,
			errContains: "failed to parse state",
		},
		{
			name: "Empty file",
			setupFile: func() string {
				filePath := filepath.Join(tempDir, "empty.tfstate")
				err := os.WriteFile(filePath, []byte("{}"), 0644)
				require.NoError(t, err)
				return filePath
			},
			wantErr: false,
			validate: func(t *testing.T, state *StateFile) {
				assert.NotNil(t, state)
				assert.Contains(t, state.Path, "empty.tfstate")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewStateParser()
			filePath := tt.setupFile()

			result, err := parser.ParseFile(filePath)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestStateParser_ExtractResourceID(t *testing.T) {
	tests := []struct {
		name     string
		resource Resource
		expected string
	}{
		{
			name: "Simple resource ID",
			resource: Resource{
				Type: "aws_instance",
				Name: "web",
				Instances: []Instance{
					{
						Attributes: map[string]interface{}{
							"id": "i-1234567890",
						},
					},
				},
			},
			expected: "i-1234567890",
		},
		{
			name: "Resource without ID",
			resource: Resource{
				Type: "aws_instance",
				Name: "web",
				Instances: []Instance{
					{
						Attributes: map[string]interface{}{
							"name": "test",
						},
					},
				},
			},
			expected: "",
		},
		{
			name: "Resource with empty instances",
			resource: Resource{
				Type:      "aws_instance",
				Name:      "web",
				Instances: []Instance{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewStateParser()
			id := parser.ExtractResourceID(tt.resource)
			assert.Equal(t, tt.expected, id)
		})
	}
}

func TestStateParser_ExtractProviderFromResource(t *testing.T) {
	tests := []struct {
		name     string
		resource Resource
		expected string
	}{
		{
			name: "AWS provider",
			resource: Resource{
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
			},
			expected: "aws",
		},
		{
			name: "Azure provider",
			resource: Resource{
				Provider: "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
			},
			expected: "azurerm",
		},
		{
			name: "GCP provider",
			resource: Resource{
				Provider: "provider[\"registry.terraform.io/hashicorp/google\"]",
			},
			expected: "google",
		},
		{
			name: "Provider with alias",
			resource: Resource{
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"].west",
			},
			expected: "aws",
		},
		{
			name: "Short provider format",
			resource: Resource{
				Provider: "aws",
			},
			expected: "aws",
		},
		{
			name: "Empty provider",
			resource: Resource{
				Provider: "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewStateParser()
			provider := parser.ExtractProviderFromResource(tt.resource)
			assert.Equal(t, tt.expected, provider)
		})
	}
}

func TestStateParser_FilterResourcesByType(t *testing.T) {
	state := &StateFile{
		TerraformState: &TerraformState{
			Resources: []Resource{
				{Type: "aws_instance", Name: "web"},
				{Type: "aws_security_group", Name: "sg1"},
				{Type: "aws_instance", Name: "app"},
				{Type: "aws_s3_bucket", Name: "bucket"},
				{Type: "aws_security_group", Name: "sg2"},
			},
		},
	}

	tests := []struct {
		name         string
		resourceType string
		expectedLen  int
		expectedNames []string
	}{
		{
			name:         "Filter AWS instances",
			resourceType: "aws_instance",
			expectedLen:  2,
			expectedNames: []string{"web", "app"},
		},
		{
			name:         "Filter security groups",
			resourceType: "aws_security_group",
			expectedLen:  2,
			expectedNames: []string{"sg1", "sg2"},
		},
		{
			name:         "Filter S3 buckets",
			resourceType: "aws_s3_bucket",
			expectedLen:  1,
			expectedNames: []string{"bucket"},
		},
		{
			name:         "Filter non-existent type",
			resourceType: "aws_rds_instance",
			expectedLen:  0,
			expectedNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewStateParser()
			filtered := parser.FilterResourcesByType(state, tt.resourceType)

			assert.Len(t, filtered, tt.expectedLen)
			for i, name := range tt.expectedNames {
				assert.Equal(t, name, filtered[i].Name)
			}
		})
	}
}

func TestStateParser_CountResources(t *testing.T) {
	tests := []struct {
		name     string
		state    *StateFile
		expected map[string]int
	}{
		{
			name: "Count various resource types",
			state: &StateFile{
				TerraformState: &TerraformState{
					Resources: []Resource{
						{Type: "aws_instance"},
						{Type: "aws_instance"},
						{Type: "aws_security_group"},
						{Type: "aws_s3_bucket"},
						{Type: "aws_instance"},
						{Type: "aws_security_group"},
					},
				},
			},
			expected: map[string]int{
				"aws_instance":       3,
				"aws_security_group": 2,
				"aws_s3_bucket":      1,
			},
		},
		{
			name: "Empty state",
			state: &StateFile{
				TerraformState: &TerraformState{
					Resources: []Resource{},
				},
			},
			expected: map[string]int{},
		},
		{
			name: "Single resource type",
			state: &StateFile{
				TerraformState: &TerraformState{
					Resources: []Resource{
						{Type: "aws_instance"},
						{Type: "aws_instance"},
						{Type: "aws_instance"},
					},
				},
			},
			expected: map[string]int{
				"aws_instance": 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewStateParser()
			counts := parser.CountResources(tt.state)
			assert.Equal(t, tt.expected, counts)
		})
	}
}

func TestStateParser_ValidateStateVersion(t *testing.T) {
	tests := []struct {
		name    string
		state   *StateFile
		wantErr bool
	}{
		{
			name: "Valid version 4",
			state: &StateFile{
				TerraformState: &TerraformState{
					Version: 4,
				},
			},
			wantErr: false,
		},
		{
			name: "Valid version 3",
			state: &StateFile{
				TerraformState: &TerraformState{
					Version: 3,
				},
			},
			wantErr: false,
		},
		{
			name: "Unsupported version 2",
			state: &StateFile{
				TerraformState: &TerraformState{
					Version: 2,
				},
			},
			wantErr: true,
		},
		{
			name: "Future version 5",
			state: &StateFile{
				TerraformState: &TerraformState{
					Version: 5,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewStateParser()
			err := parser.ValidateStateVersion(tt.state)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}