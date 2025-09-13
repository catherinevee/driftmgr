package backend

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test comprehensive backend functionality
func TestBackend_Comprehensive(t *testing.T) {
	backends := map[string]Backend{
		"Mock": NewMockBackend(),
	}

	// Add local backend
	if localBackend := createTestLocalBackend(t); localBackend != nil {
		backends["Local"] = localBackend
	}

	// Add GCS backend
	if gcsBackend := createTestGCSBackend(t); gcsBackend != nil {
		backends["GCS"] = gcsBackend
	}

	ctx := context.Background()

	for name, backend := range backends {
		t.Run(name, func(t *testing.T) {
			testBackendOperations(t, backend, ctx)
		})
	}
}

func createTestLocalBackend(t *testing.T) Backend {
	config := &BackendConfig{
		Type: "local",
		Config: map[string]interface{}{
			"path": t.TempDir(),
		},
	}
	backend, err := NewLocalBackend(config)
	if err != nil {
		t.Logf("Could not create local backend: %v", err)
		return nil
	}
	return backend
}

func createTestGCSBackend(t *testing.T) Backend {
	config := &BackendConfig{
		Type: "gcs",
		Config: map[string]interface{}{
			"bucket": "test-bucket",
			"prefix": "test-prefix",
		},
	}
	backend, err := NewGCSBackend(config)
	if err != nil {
		t.Logf("Could not create GCS backend: %v", err)
		return nil
	}
	return backend
}

func testBackendOperations(t *testing.T, backend Backend, ctx context.Context) {
	// Test basic state operations
	t.Run("StateOperations", func(t *testing.T) {
		// Pull initial state
		state, err := backend.Pull(ctx)
		require.NoError(t, err)
		assert.NotNil(t, state)
		assert.Equal(t, 4, state.Version)

		// Push new state
		newState := &StateData{
			Version:          4,
			TerraformVersion: "1.5.0",
			Serial:           1,
			Lineage:          "test-lineage",
			Data:             []byte(`{"version": 4, "serial": 1, "resources": [], "outputs": {}}`),
			LastModified:     time.Now(),
			Size:             100,
		}

		err = backend.Push(ctx, newState)
		require.NoError(t, err)

		// Pull updated state
		pulledState, err := backend.Pull(ctx)
		require.NoError(t, err)
		assert.Equal(t, newState.Version, pulledState.Version)
		assert.Equal(t, newState.Serial, pulledState.Serial)
	})

	// Test locking operations
	t.Run("LockingOperations", func(t *testing.T) {
		lockInfo := &LockInfo{
			ID:        "test-lock",
			Path:      "terraform.tfstate",
			Operation: "plan",
			Who:       "test-user",
			Created:   time.Now(),
		}

		// Acquire lock
		lockID, err := backend.Lock(ctx, lockInfo)
		require.NoError(t, err)
		assert.NotEmpty(t, lockID)

		// Get lock info
		info, err := backend.GetLockInfo(ctx)
		require.NoError(t, err)
		if info != nil { // Some backends might not support lock info
			assert.Equal(t, lockInfo.ID, info.ID)
		}

		// Release lock
		err = backend.Unlock(ctx, lockID)
		require.NoError(t, err)
	})

	// Test workspace operations
	t.Run("WorkspaceOperations", func(t *testing.T) {
		// List workspaces
		workspaces, err := backend.ListWorkspaces(ctx)
		require.NoError(t, err)
		assert.Contains(t, workspaces, "default")

		// Create workspace
		err = backend.CreateWorkspace(ctx, "test-workspace")
		require.NoError(t, err)

		// List workspaces again
		workspaces, err = backend.ListWorkspaces(ctx)
		require.NoError(t, err)
		assert.Contains(t, workspaces, "test-workspace")

		// Select workspace
		err = backend.SelectWorkspace(ctx, "test-workspace")
		require.NoError(t, err)

		// Switch back to default
		err = backend.SelectWorkspace(ctx, "default")
		require.NoError(t, err)

		// Delete workspace
		err = backend.DeleteWorkspace(ctx, "test-workspace")
		require.NoError(t, err)
	})

	// Test version operations
	t.Run("VersionOperations", func(t *testing.T) {
		// Push a state to create versions
		state := &StateData{
			Version:          4,
			TerraformVersion: "1.5.0",
			Serial:           1,
			Lineage:          "version-test",
			Data:             []byte(`{"version": 4, "serial": 1, "resources": []}`),
			LastModified:     time.Now(),
			Size:             50,
		}

		err := backend.Push(ctx, state)
		require.NoError(t, err)

		// Get versions
		versions, err := backend.GetVersions(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(versions), 1)

		// Get specific version
		if len(versions) > 0 {
			versionState, err := backend.GetVersion(ctx, versions[0].VersionID)
			require.NoError(t, err)
			assert.NotNil(t, versionState)
		}
	})

	// Test validation
	t.Run("Validation", func(t *testing.T) {
		err := backend.Validate(ctx)
		require.NoError(t, err)
	})

	// Test metadata
	t.Run("Metadata", func(t *testing.T) {
		metadata := backend.GetMetadata()
		require.NotNil(t, metadata)
		assert.NotEmpty(t, metadata.Type)
	})
}

// Test adapter functionality basic
func TestAdapter_BasicOperations(t *testing.T) {
	mockBackend := NewMockBackend()
	config := &BackendConfig{Type: "mock"}
	adapter := NewAdapter(mockBackend, config)

	ctx := context.Background()

	// Test get/put operations
	testData := []byte(`{"version": 4, "serial": 1, "resources": [], "outputs": {}}`)

	err := adapter.Put(ctx, "terraform.tfstate", testData)
	require.NoError(t, err)

	data, err := adapter.Get(ctx, "terraform.tfstate")
	require.NoError(t, err)
	assert.NotNil(t, data)

	// Test list operations
	keys, err := adapter.List(ctx, "terraform")
	require.NoError(t, err)
	assert.NotNil(t, keys)

	// Test locking
	err = adapter.Lock(ctx, "terraform.tfstate")
	require.NoError(t, err)

	err = adapter.Unlock(ctx, "terraform.tfstate")
	require.NoError(t, err)
}

// Test error scenarios
func TestBackend_ErrorScenarios(t *testing.T) {
	backend := NewMockBackend()
	ctx := context.Background()

	// Test workspace errors
	err := backend.CreateWorkspace(ctx, "default")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot create default workspace")

	err = backend.DeleteWorkspace(ctx, "default")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete default workspace")

	// Create workspace to test current workspace deletion
	err = backend.CreateWorkspace(ctx, "test")
	require.NoError(t, err)

	err = backend.SelectWorkspace(ctx, "test")
	require.NoError(t, err)

	err = backend.DeleteWorkspace(ctx, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete current workspace")
}

// Test concurrent operations (basic)
func TestBackend_BasicConcurrency(t *testing.T) {
	backend := NewMockBackend()
	ctx := context.Background()

	// Test concurrent pulls
	t.Run("ConcurrentPulls", func(t *testing.T) {
		const numGoroutines = 10
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				_, err := backend.Pull(ctx)
				done <- (err == nil)
			}()
		}

		successCount := 0
		for i := 0; i < numGoroutines; i++ {
			if <-done {
				successCount++
			}
		}

		assert.Equal(t, numGoroutines, successCount)
	})

	// Test concurrent workspace creation
	t.Run("ConcurrentWorkspaceCreation", func(t *testing.T) {
		const numWorkspaces = 5
		done := make(chan bool, numWorkspaces)

		for i := 0; i < numWorkspaces; i++ {
			go func(id int) {
				err := backend.CreateWorkspace(ctx, fmt.Sprintf("workspace-%d", id))
				done <- (err == nil)
			}(i)
		}

		successCount := 0
		for i := 0; i < numWorkspaces; i++ {
			if <-done {
				successCount++
			}
		}

		assert.Equal(t, numWorkspaces, successCount)
	})
}

// Helper function for basic benchmarking
func BenchmarkBackend_BasicOperations(b *testing.B) {
	backend := NewMockBackend()
	ctx := context.Background()

	state := &StateData{
		Version: 4,
		Serial:  1,
		Data:    []byte(`{"version": 4, "serial": 1}`),
	}

	b.Run("Pull", func(b *testing.B) {
		// Push initial state
		backend.Push(ctx, state)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := backend.Pull(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Push", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			testState := *state
			testState.Serial = uint64(i + 1)
			err := backend.Push(ctx, &testState)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

