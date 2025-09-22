package state

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockBackend implements Backend interface for testing
type MockBackend struct {
	stateData   []byte
	versions    []StateVersion
	states      []string
	lockError   error
	unlockError error
	getError    error
	putError    error
	deleteError error
	listError   error
}

func (m *MockBackend) Get(ctx context.Context, key string) ([]byte, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	return m.stateData, nil
}

func (m *MockBackend) Put(ctx context.Context, key string, data []byte) error {
	if m.putError != nil {
		return m.putError
	}
	m.stateData = data
	return nil
}

func (m *MockBackend) Delete(ctx context.Context, key string) error {
	if m.deleteError != nil {
		return m.deleteError
	}
	return nil
}

func (m *MockBackend) List(ctx context.Context, prefix string) ([]string, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	return m.states, nil
}

func (m *MockBackend) Lock(ctx context.Context, key string) error {
	if m.lockError != nil {
		return m.lockError
	}
	return nil
}

func (m *MockBackend) Unlock(ctx context.Context, key string) error {
	if m.unlockError != nil {
		return m.unlockError
	}
	return nil
}

func (m *MockBackend) ListStates(ctx context.Context) ([]string, error) {
	return m.states, nil
}

func (m *MockBackend) ListStateVersions(ctx context.Context, key string) ([]StateVersion, error) {
	return m.versions, nil
}

func (m *MockBackend) GetStateVersion(ctx context.Context, key string, version int) ([]byte, error) {
	for _, v := range m.versions {
		if v.Version == version {
			return m.stateData, nil
		}
	}
	return nil, fmt.Errorf("version %d not found", version)
}

// TestNewStateManager tests the creation of a new state manager
func TestNewStateManager(t *testing.T) {
	backend := &MockBackend{}
	manager := NewStateManager(backend)

	assert.NotNil(t, manager)
	assert.Equal(t, backend, manager.backend)
	assert.NotNil(t, manager.mu)
}

// TestStateManagerGetState tests getting state data
func TestStateManagerGetState(t *testing.T) {
	expectedData := []byte(`{"version": 4, "serial": 1, "lineage": "test-lineage-123"}`)

	backend := &MockBackend{
		stateData: expectedData,
	}

	manager := NewStateManager(backend)
	ctx := context.Background()

	state, err := manager.GetState(ctx, "test-key")

	require.NoError(t, err)
	assert.NotNil(t, state)
}

// TestStateManagerPutState tests putting state data
func TestStateManagerPutState(t *testing.T) {
	backend := &MockBackend{}
	manager := NewStateManager(backend)
	ctx := context.Background()

	state := &TerraformState{
		Version: 4,
		Serial:  1,
		Lineage: "test-lineage-123",
	}

	err := manager.PutState(ctx, "test-key", state)

	require.NoError(t, err)
}

// TestStateManagerListStates tests listing states
func TestStateManagerListStates(t *testing.T) {
	expectedStates := []string{"state1", "state2", "state3"}

	backend := &MockBackend{
		states: expectedStates,
	}

	manager := NewStateManager(backend)
	ctx := context.Background()

	states, err := manager.ListStates(ctx)

	require.NoError(t, err)
	assert.Equal(t, expectedStates, states)
}

// TestStateManagerListStateVersions tests listing state versions
func TestStateManagerListStateVersions(t *testing.T) {
	expectedVersions := []StateVersion{
		{
			Version:   1,
			Serial:    1,
			Timestamp: time.Now().Add(-24 * time.Hour),
			Checksum:  "abc123",
		},
		{
			Version:   2,
			Serial:    2,
			Timestamp: time.Now().Add(-1 * time.Hour),
			Checksum:  "def456",
		},
	}

	backend := &MockBackend{
		versions: expectedVersions,
	}

	manager := NewStateManager(backend)
	ctx := context.Background()

	versions, err := manager.ListStateVersions(ctx, "test-key")

	require.NoError(t, err)
	assert.Equal(t, expectedVersions, versions)
}
