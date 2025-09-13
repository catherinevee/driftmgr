package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Adapter Creation
func TestNewAdapter(t *testing.T) {
	mockBackend := NewMockBackend()
	config := &BackendConfig{
		Type: "mock",
		Config: map[string]interface{}{
			"test": "value",
		},
	}

	adapter := NewAdapter(mockBackend, config)

	require.NotNil(t, adapter)
	assert.Equal(t, mockBackend, adapter.backend)
	assert.Equal(t, config, adapter.config)
}

// Test Adapter Operations
func TestAdapter_Operations(t *testing.T) {
	mockBackend := NewMockBackend()
	config := &BackendConfig{
		Type: "mock",
		Config: map[string]interface{}{
			"test": "value",
		},
	}

	adapter := NewAdapter(mockBackend, config)
	ctx := context.Background()

	t.Run("Get and Put operations", func(t *testing.T) {
		// Test getting non-existent key
		data, err := adapter.Get(ctx, "non-existent")
		require.NoError(t, err)
		assert.NotNil(t, data)

		// Verify it called the backend
		assert.Equal(t, 1, mockBackend.pullCalls)

		// Test putting data
		testStateData := map[string]interface{}{
			"version":           4,
			"terraform_version": "1.5.0",
			"serial":            1,
			"lineage":           "test-lineage",
			"resources":         []interface{}{},
			"outputs":           map[string]interface{}{},
		}

		stateBytes, err := json.Marshal(testStateData)
		require.NoError(t, err)

		err = adapter.Put(ctx, "terraform.tfstate", stateBytes)
		require.NoError(t, err)

		// Verify it called the backend
		assert.Equal(t, 1, mockBackend.pushCalls)

		// Test getting the data back
		retrievedData, err := adapter.Get(ctx, "terraform.tfstate")
		require.NoError(t, err)
		assert.NotNil(t, retrievedData)

		// Parse and verify the data
		var retrievedState map[string]interface{}
		err = json.Unmarshal(retrievedData, &retrievedState)
		require.NoError(t, err)
		assert.Equal(t, testStateData["version"], retrievedState["version"])
		assert.Equal(t, testStateData["serial"], retrievedState["serial"])
		assert.Equal(t, testStateData["lineage"], retrievedState["lineage"])
	})

	t.Run("Delete operation", func(t *testing.T) {
		// Put some data first
		testData := `{"version": 4, "serial": 1, "resources": [], "outputs": {}}`
		err := adapter.Put(ctx, "test.tfstate", []byte(testData))
		require.NoError(t, err)

		// Delete it
		err = adapter.Delete(ctx, "test.tfstate")
		require.NoError(t, err)

		// Verify deletion by trying to get it (should return empty state)
		data, err := adapter.Get(ctx, "test.tfstate")
		require.NoError(t, err)
		assert.NotNil(t, data)

		// Should be empty state
		var state map[string]interface{}
		err = json.Unmarshal(data, &state)
		require.NoError(t, err)
		assert.Equal(t, float64(0), state["serial"])
	})

	t.Run("List operation", func(t *testing.T) {
		keys, err := adapter.List(ctx, "terraform")
		require.NoError(t, err)
		assert.NotNil(t, keys)
		// Should at least contain default workspace
		assert.GreaterOrEqual(t, len(keys), 1)
	})

	t.Run("Lock and Unlock operations", func(t *testing.T) {
		// Test lock
		err := adapter.Lock(ctx, "terraform.tfstate")
		require.NoError(t, err)

		// Verify backend was called
		assert.Equal(t, 1, mockBackend.lockCalls)

		// Test unlock
		err = adapter.Unlock(ctx, "terraform.tfstate")
		require.NoError(t, err)

		// Verify backend was called
		assert.Equal(t, 1, mockBackend.unlockCalls)
	})

	t.Run("ListStates operation", func(t *testing.T) {
		states, err := adapter.ListStates(ctx)
		require.NoError(t, err)
		assert.NotNil(t, states)
		assert.Contains(t, states, "default")
	})

	t.Run("State versions operations", func(t *testing.T) {
		// Put a state to create versions
		testData := `{"version": 4, "serial": 1, "resources": [], "outputs": {}}`
		err := adapter.Put(ctx, "terraform.tfstate", []byte(testData))
		require.NoError(t, err)

		// List versions
		versions, err := adapter.ListStateVersions(ctx, "terraform.tfstate")
		require.NoError(t, err)
		assert.NotNil(t, versions)
		assert.GreaterOrEqual(t, len(versions), 1)

		// Get specific version
		if len(versions) > 0 {
			versionData, err := adapter.GetStateVersion(ctx, "terraform.tfstate", 0)
			require.NoError(t, err)
			assert.NotNil(t, versionData)
		}
	})
}

// Test Workspace Key Extraction
func TestAdapter_WorkspaceKeyExtraction(t *testing.T) {
	mockBackend := NewMockBackend()
	config := &BackendConfig{Type: "mock"}
	adapter := NewAdapter(mockBackend, config)

	tests := []struct {
		name              string
		key               string
		expectedWorkspace string
	}{
		{
			name:              "simple key",
			key:               "terraform.tfstate",
			expectedWorkspace: "terraform.tfstate",
		},
		{
			name:              "key with env prefix",
			key:               "env/production",
			expectedWorkspace: "production",
		},
		{
			name:              "key with workspaces prefix",
			key:               "workspaces/staging",
			expectedWorkspace: "staging",
		},
		{
			name:              "nested key",
			key:               "project/env/development",
			expectedWorkspace: "development",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace := adapter.extractWorkspaceFromKey(tt.key)
			assert.Equal(t, tt.expectedWorkspace, workspace)
		})
	}
}

// Test Backend Adapter Factory
func TestCreateBackendAdapter(t *testing.T) {
	t.Run("S3 backend", func(t *testing.T) {
		config := &BackendConfig{
			Type: "s3",
			Config: map[string]interface{}{
				"bucket": "test-bucket",
				"key":    "terraform.tfstate",
				"region": "us-west-2",
			},
		}

		// Skip actual backend creation since we don't have AWS clients
		// Just test config extraction
		bucket := getStringFromConfig(config.Config, "bucket")
		key := getStringFromConfig(config.Config, "key")
		region := getStringFromConfig(config.Config, "region")

		assert.Equal(t, "test-bucket", bucket)
		assert.Equal(t, "terraform.tfstate", key)
		assert.Equal(t, "us-west-2", region)
	})

	t.Run("Azure backend (disabled)", func(t *testing.T) {
		config := &BackendConfig{
			Type: "azurerm",
			Config: map[string]interface{}{
				"storage_account_name": "testaccount",
				"container_name":       "tfstate",
				"key":                  "terraform.tfstate",
			},
		}

		_, err := CreateBackendAdapter(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Azure backend temporarily disabled")
	})

	t.Run("GCS backend (not implemented)", func(t *testing.T) {
		config := &BackendConfig{
			Type: "gcs",
			Config: map[string]interface{}{
				"bucket": "test-bucket",
			},
		}

		_, err := CreateBackendAdapter(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GCS backend not yet implemented")
	})

	t.Run("Remote backend (not implemented)", func(t *testing.T) {
		config := &BackendConfig{
			Type: "remote",
			Config: map[string]interface{}{
				"organization": "test-org",
				"workspaces": map[string]interface{}{
					"name": "test-workspace",
				},
			},
		}

		_, err := CreateBackendAdapter(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Terraform Cloud backend not yet implemented")
	})

	t.Run("Unsupported backend", func(t *testing.T) {
		config := &BackendConfig{
			Type: "unsupported",
			Config: map[string]interface{}{
				"test": "value",
			},
		}

		_, err := CreateBackendAdapter(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported backend type")
	})
}

// Test Config Helper Functions
func TestConfigHelpers(t *testing.T) {
	config := map[string]interface{}{
		"string_value": "test",
		"bool_true":    true,
		"bool_false":   false,
		"int_value":    42,
		"float_value":  3.14,
	}

	t.Run("getStringFromConfig", func(t *testing.T) {
		assert.Equal(t, "test", getStringFromConfig(config, "string_value"))
		assert.Equal(t, "", getStringFromConfig(config, "nonexistent"))
		assert.Equal(t, "", getStringFromConfig(config, "int_value")) // Not a string
	})

	t.Run("getBoolFromConfig", func(t *testing.T) {
		assert.True(t, getBoolFromConfig(config, "bool_true"))
		assert.False(t, getBoolFromConfig(config, "bool_false"))
		assert.False(t, getBoolFromConfig(config, "nonexistent"))
		assert.False(t, getBoolFromConfig(config, "string_value")) // Not a bool
	})

	t.Run("getIntFromConfig", func(t *testing.T) {
		assert.Equal(t, 42, getIntFromConfig(config, "int_value"))
		assert.Equal(t, 3, getIntFromConfig(config, "float_value")) // Float to int conversion
		assert.Equal(t, 0, getIntFromConfig(config, "nonexistent"))
		assert.Equal(t, 0, getIntFromConfig(config, "string_value")) // Not an int
	})
}

// Test Backend Configuration Validation
func TestBackendConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		configType  string
		config      map[string]interface{}
		expectValid bool
	}{
		{
			name:       "valid S3 config",
			configType: "s3",
			config: map[string]interface{}{
				"bucket": "test-bucket",
				"key":    "terraform.tfstate",
				"region": "us-west-2",
			},
			expectValid: true,
		},
		{
			name:       "S3 config missing bucket",
			configType: "s3",
			config: map[string]interface{}{
				"key":    "terraform.tfstate",
				"region": "us-west-2",
			},
			expectValid: false,
		},
		{
			name:       "S3 config missing key",
			configType: "s3",
			config: map[string]interface{}{
				"bucket": "test-bucket",
				"region": "us-west-2",
			},
			expectValid: false,
		},
		{
			name:       "valid Azure config",
			configType: "azurerm",
			config: map[string]interface{}{
				"storage_account_name": "testaccount",
				"container_name":       "tfstate",
				"key":                  "terraform.tfstate",
			},
			expectValid: true,
		},
		{
			name:       "Azure config missing storage account",
			configType: "azurerm",
			config: map[string]interface{}{
				"container_name": "tfstate",
				"key":            "terraform.tfstate",
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate required fields based on backend type
			switch tt.configType {
			case "s3":
				bucket := getStringFromConfig(tt.config, "bucket")
				key := getStringFromConfig(tt.config, "key")
				valid := bucket != "" && key != ""
				assert.Equal(t, tt.expectValid, valid)

			case "azurerm":
				storageAccount := getStringFromConfig(tt.config, "storage_account_name")
				containerName := getStringFromConfig(tt.config, "container_name")
				key := getStringFromConfig(tt.config, "key")
				valid := storageAccount != "" && containerName != "" && key != ""
				assert.Equal(t, tt.expectValid, valid)
			}
		})
	}
}

// Test Adapter Error Handling
func TestAdapter_ErrorHandling(t *testing.T) {
	// Create mock backend that returns errors
	mockBackend := NewMockBackend()
	mockBackend.pullError = fmt.Errorf("pull error")
	mockBackend.pushError = fmt.Errorf("push error")
	mockBackend.lockError = fmt.Errorf("lock error")
	mockBackend.unlockError = fmt.Errorf("unlock error")

	config := &BackendConfig{Type: "mock"}
	adapter := NewAdapter(mockBackend, config)
	ctx := context.Background()

	t.Run("Get operation error", func(t *testing.T) {
		_, err := adapter.Get(ctx, "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pull error")
	})

	t.Run("Put operation error", func(t *testing.T) {
		testData := []byte(`{"version": 4}`)
		err := adapter.Put(ctx, "test", testData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "push error")
	})

	t.Run("Lock operation error", func(t *testing.T) {
		err := adapter.Lock(ctx, "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "lock error")
	})

	t.Run("Unlock operation error", func(t *testing.T) {
		err := adapter.Unlock(ctx, "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unlock error")
	})

	t.Run("Invalid JSON in Put", func(t *testing.T) {
		// Create backend without push error for this test
		goodBackend := NewMockBackend()
		goodAdapter := NewAdapter(goodBackend, config)

		invalidJSON := []byte(`{"invalid": json}`)
		err := goodAdapter.Put(ctx, "test", invalidJSON)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse state")
	})
}

// Test Adapter with Workspace Selection
func TestAdapter_WorkspaceSelection(t *testing.T) {
	mockBackend := NewMockBackend()
	config := &BackendConfig{Type: "mock"}
	adapter := NewAdapter(mockBackend, config)
	ctx := context.Background()

	t.Run("Workspace selection from key", func(t *testing.T) {
		// Test with workspace key
		err := adapter.selectWorkspaceFromKey(ctx, "env/production")
		require.NoError(t, err)

		// Verify workspace was selected
		assert.Equal(t, "production", mockBackend.metadata.Workspace)

		// Test with default key
		err = adapter.selectWorkspaceFromKey(ctx, "terraform.tfstate")
		require.NoError(t, err)

		// Should select the key itself as workspace for simple keys
		assert.Equal(t, "terraform.tfstate", mockBackend.metadata.Workspace)

		// Test with empty key
		err = adapter.selectWorkspaceFromKey(ctx, "")
		require.NoError(t, err)

		// Should default to "default"
		assert.Equal(t, "default", mockBackend.metadata.Workspace)
	})

	t.Run("Delete workspace via key", func(t *testing.T) {
		// Create a workspace first
		err := mockBackend.CreateWorkspace(ctx, "test-workspace")
		require.NoError(t, err)

		// Delete via adapter
		err = adapter.Delete(ctx, "env/test-workspace")
		require.NoError(t, err)

		// Verify workspace is gone
		workspaces, err := mockBackend.ListWorkspaces(ctx)
		require.NoError(t, err)
		assert.NotContains(t, workspaces, "test-workspace")
	})
}

// Benchmark Adapter Operations
func BenchmarkAdapter_Get(b *testing.B) {
	mockBackend := NewMockBackend()
	config := &BackendConfig{Type: "mock"}
	adapter := NewAdapter(mockBackend, config)

	// Prepare test data
	testData := []byte(`{"version": 4, "serial": 1, "resources": [], "outputs": {}}`)
	err := adapter.Put(context.Background(), "terraform.tfstate", testData)
	require.NoError(b, err)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := adapter.Get(ctx, "terraform.tfstate")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAdapter_Put(b *testing.B) {
	mockBackend := NewMockBackend()
	config := &BackendConfig{Type: "mock"}
	adapter := NewAdapter(mockBackend, config)

	testData := []byte(`{"version": 4, "serial": 1, "resources": [], "outputs": {}}`)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := adapter.Put(ctx, fmt.Sprintf("terraform-%d.tfstate", i), testData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAdapter_Lock(b *testing.B) {
	mockBackend := NewMockBackend()
	config := &BackendConfig{Type: "mock"}
	adapter := NewAdapter(mockBackend, config)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("terraform-%d.tfstate", i)
		err := adapter.Lock(ctx, key)
		if err != nil {
			b.Fatal(err)
		}

		err = adapter.Unlock(ctx, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}
