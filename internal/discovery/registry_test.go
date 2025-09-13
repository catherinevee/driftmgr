package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackendType_Constants(t *testing.T) {
	// Verify backend type constants are defined
	assert.Equal(t, BackendType("local"), BackendLocal)
	assert.Equal(t, BackendType("s3"), BackendS3)
	assert.Equal(t, BackendType("azurerm"), BackendAzureBlob)
	assert.Equal(t, BackendType("gcs"), BackendGCS)
	assert.Equal(t, BackendType("remote"), BackendRemote)
}

func TestNewLocalBackend(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Simple path",
			path:     "/tmp/terraform.tfstate",
			expected: "/tmp/terraform.tfstate",
		},
		{
			name:     "Path with directory",
			path:     "/var/lib/terraform/state.tfstate",
			expected: "/var/lib/terraform/state.tfstate",
		},
		{
			name:     "Relative path",
			path:     "./terraform.tfstate",
			expected: "./terraform.tfstate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := NewLocalBackend(tt.path)
			assert.NotNil(t, backend)
			assert.Equal(t, tt.expected, backend.path)
			assert.Equal(t, tt.expected+".lock", backend.lockFile)
		})
	}
}

func TestLocalBackend_Connect(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "subdir", "terraform.tfstate")

	backend := NewLocalBackend(statePath)
	ctx := context.Background()

	err := backend.Connect(ctx)
	assert.NoError(t, err)

	// Verify directory was created
	dir := filepath.Dir(statePath)
	_, err = os.Stat(dir)
	assert.NoError(t, err)
}

func TestLocalBackend_GetState(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "terraform.tfstate")

	// Create test state file
	testData := []byte(`{"version": 4, "terraform_version": "1.0.0"}`)
	require.NoError(t, os.WriteFile(statePath, testData, 0644))

	backend := NewLocalBackend(statePath)
	ctx := context.Background()

	// Test getting state
	data, err := backend.GetState(ctx, "")
	assert.NoError(t, err)
	assert.Equal(t, testData, data)

	// Test getting non-existent state
	backend2 := NewLocalBackend(filepath.Join(tempDir, "nonexistent.tfstate"))
	_, err = backend2.GetState(ctx, "")
	assert.Error(t, err)
}

func TestLocalBackend_GetStateWithKey(t *testing.T) {
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "terraform.tfstate")

	// Create test state files with different keys
	testData1 := []byte(`{"version": 4, "key": "prod"}`)
	testData2 := []byte(`{"version": 4, "key": "staging"}`)

	prodPath := filepath.Join(tempDir, "prod.tfstate")
	stagingPath := filepath.Join(tempDir, "staging.tfstate")

	require.NoError(t, os.WriteFile(prodPath, testData1, 0644))
	require.NoError(t, os.WriteFile(stagingPath, testData2, 0644))

	backend := NewLocalBackend(basePath)
	ctx := context.Background()

	// Get state with key
	data, err := backend.GetState(ctx, "prod.tfstate")
	assert.NoError(t, err)
	assert.Equal(t, testData1, data)

	data, err = backend.GetState(ctx, "staging.tfstate")
	assert.NoError(t, err)
	assert.Equal(t, testData2, data)
}

func TestLocalBackend_PutState(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "terraform.tfstate")

	backend := NewLocalBackend(statePath)
	ctx := context.Background()

	// Test putting state
	testData := []byte(`{"version": 4, "serial": 1}`)
	err := backend.PutState(ctx, "", testData)
	assert.NoError(t, err)

	// Verify file was written
	data, err := os.ReadFile(statePath)
	assert.NoError(t, err)
	assert.Equal(t, testData, data)

	// Test updating state
	updatedData := []byte(`{"version": 4, "serial": 2}`)
	err = backend.PutState(ctx, "", updatedData)
	assert.NoError(t, err)

	data, err = os.ReadFile(statePath)
	assert.NoError(t, err)
	assert.Equal(t, updatedData, data)
}

func TestLocalBackend_PutStateWithKey(t *testing.T) {
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "terraform.tfstate")

	backend := NewLocalBackend(basePath)
	ctx := context.Background()

	// Put state with different keys
	testData1 := []byte(`{"version": 4, "env": "dev"}`)
	testData2 := []byte(`{"version": 4, "env": "test"}`)

	err := backend.PutState(ctx, "dev.tfstate", testData1)
	assert.NoError(t, err)

	err = backend.PutState(ctx, "test.tfstate", testData2)
	assert.NoError(t, err)

	// Verify files were written
	devPath := filepath.Join(tempDir, "dev.tfstate")
	testPath := filepath.Join(tempDir, "test.tfstate")

	data, err := os.ReadFile(devPath)
	assert.NoError(t, err)
	assert.Equal(t, testData1, data)

	data, err = os.ReadFile(testPath)
	assert.NoError(t, err)
	assert.Equal(t, testData2, data)
}

func TestLocalBackend_DeleteState(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "terraform.tfstate")

	// Create test state file
	testData := []byte(`{"version": 4}`)
	require.NoError(t, os.WriteFile(statePath, testData, 0644))

	backend := NewLocalBackend(statePath)
	ctx := context.Background()

	// Test deleting state
	err := backend.DeleteState(ctx, "")
	assert.NoError(t, err)

	// Verify file was deleted
	_, err = os.Stat(statePath)
	assert.True(t, os.IsNotExist(err))

	// Test deleting non-existent state (should not error)
	err = backend.DeleteState(ctx, "")
	assert.NoError(t, err)
}

func TestLocalBackend_ListStates(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple state files
	stateFiles := []string{
		"prod.tfstate",
		"staging.tfstate",
		"dev.tfstate",
		"test.tfstate.backup", // Should be excluded
		"README.md",           // Should be excluded
	}

	for _, file := range stateFiles {
		path := filepath.Join(tempDir, file)
		require.NoError(t, os.WriteFile(path, []byte(`{"version": 4}`), 0644))
	}

	backend := NewLocalBackend(filepath.Join(tempDir, "terraform.tfstate"))
	ctx := context.Background()

	states, err := backend.ListStates(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, states)

	// Should find .tfstate files
	expectedStates := 3 // prod, staging, dev
	assert.GreaterOrEqual(t, len(states), expectedStates)
}

func TestLocalBackend_LockState(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "terraform.tfstate")

	// Create state file
	require.NoError(t, os.WriteFile(statePath, []byte(`{"version": 4}`), 0644))

	backend := NewLocalBackend(statePath)
	ctx := context.Background()

	// Test locking state
	lockID, err := backend.LockState(ctx, "")
	assert.NoError(t, err)
	assert.NotEmpty(t, lockID)

	// Verify lock file exists
	_, err = os.Stat(backend.lockFile)
	assert.NoError(t, err)

	// Test locking already locked state
	_, err = backend.LockState(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already locked")
}

func TestLocalBackend_UnlockState(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "terraform.tfstate")

	// Create state file
	require.NoError(t, os.WriteFile(statePath, []byte(`{"version": 4}`), 0644))

	backend := NewLocalBackend(statePath)
	ctx := context.Background()

	// Lock state first
	lockID, err := backend.LockState(ctx, "")
	require.NoError(t, err)

	// Test unlocking with correct lock ID
	err = backend.UnlockState(ctx, "", lockID)
	assert.NoError(t, err)

	// Verify lock file was removed
	_, err = os.Stat(backend.lockFile)
	assert.True(t, os.IsNotExist(err))

	// Test unlocking already unlocked state
	err = backend.UnlockState(ctx, "", lockID)
	assert.NoError(t, err) // Should not error
}

func TestLocalBackend_UnlockStateWrongID(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "terraform.tfstate")

	// Create state file
	require.NoError(t, os.WriteFile(statePath, []byte(`{"version": 4}`), 0644))

	backend := NewLocalBackend(statePath)
	ctx := context.Background()

	// Lock state
	lockID, err := backend.LockState(ctx, "")
	require.NoError(t, err)

	// Try unlocking with wrong lock ID
	err = backend.UnlockState(ctx, "", "wrong-lock-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lock ID mismatch")

	// Lock should still exist
	_, err = os.Stat(backend.lockFile)
	assert.NoError(t, err)

	// Unlock with correct ID
	err = backend.UnlockState(ctx, "", lockID)
	assert.NoError(t, err)
}

func TestLocalBackend_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "terraform.tfstate")

	backend := NewLocalBackend(statePath)
	ctx := context.Background()

	// Simulate concurrent writes
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(n int) {
			data := []byte(fmt.Sprintf(`{"version": 4, "serial": %d}`, n))
			err := backend.PutState(ctx, "", data)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// State file should exist and be readable
	data, err := backend.GetState(ctx, "")
	assert.NoError(t, err)
	assert.NotNil(t, data)
}

func TestLocalBackend_LockTimeout(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "terraform.tfstate")

	// Create state file
	require.NoError(t, os.WriteFile(statePath, []byte(`{"version": 4}`), 0644))

	backend := NewLocalBackend(statePath)
	ctx := context.Background()

	// Lock state
	lockID, err := backend.LockState(ctx, "")
	require.NoError(t, err)

	// Try to lock with timeout context
	ctxTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	// This should timeout
	_, err = backend.LockState(ctxTimeout, "")
	assert.Error(t, err)

	// Unlock
	err = backend.UnlockState(ctx, "", lockID)
	assert.NoError(t, err)
}

// Benchmark tests
func BenchmarkLocalBackend_GetState(b *testing.B) {
	tempDir := b.TempDir()
	statePath := filepath.Join(tempDir, "terraform.tfstate")

	// Create test state
	testData := []byte(`{"version": 4, "serial": 1, "lineage": "test", "resources": []}`)
	os.WriteFile(statePath, testData, 0644)

	backend := NewLocalBackend(statePath)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = backend.GetState(ctx, "")
	}
}

func BenchmarkLocalBackend_PutState(b *testing.B) {
	tempDir := b.TempDir()
	statePath := filepath.Join(tempDir, "terraform.tfstate")

	backend := NewLocalBackend(statePath)
	ctx := context.Background()
	testData := []byte(`{"version": 4, "serial": 1}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = backend.PutState(ctx, "", testData)
	}
}

func BenchmarkLocalBackend_LockUnlock(b *testing.B) {
	tempDir := b.TempDir()
	statePath := filepath.Join(tempDir, "terraform.tfstate")
	os.WriteFile(statePath, []byte(`{"version": 4}`), 0644)

	backend := NewLocalBackend(statePath)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lockID, _ := backend.LockState(ctx, "")
		backend.UnlockState(ctx, "", lockID)
	}
}

// Test helper to verify the Backend interface is implemented
var _ Backend = (*LocalBackend)(nil)
