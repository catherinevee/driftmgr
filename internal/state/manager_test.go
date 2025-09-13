package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockBackend is a mock implementation of the Backend interface
type MockBackend struct {
	mock.Mock
	states map[string][]byte
	locks  map[string]bool
	mu     sync.Mutex
}

func NewMockBackend() *MockBackend {
	return &MockBackend{
		states: make(map[string][]byte),
		locks:  make(map[string]bool),
	}
}

func (m *MockBackend) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockBackend) Put(ctx context.Context, key string, data []byte) error {
	args := m.Called(ctx, key, data)
	return args.Error(0)
}

func (m *MockBackend) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockBackend) List(ctx context.Context, prefix string) ([]string, error) {
	args := m.Called(ctx, prefix)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockBackend) Lock(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockBackend) Unlock(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockBackend) ListStates(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockBackend) ListStateVersions(ctx context.Context, key string) ([]StateVersion, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]StateVersion), args.Error(1)
}

func (m *MockBackend) GetStateVersion(ctx context.Context, key string, version int) ([]byte, error) {
	args := m.Called(ctx, key, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func createValidState() *TerraformState {
	return &TerraformState{
		Version:          4,
		TerraformVersion: "1.5.0",
		Serial:           1,
		Lineage:          "test-lineage",
		Resources: []Resource{
			{
				Mode:     "managed",
				Type:     "aws_instance",
				Name:     "test",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []Instance{
					{
						Attributes: map[string]interface{}{
							"id":            "i-1234567890",
							"instance_type": "t2.micro",
						},
					},
				},
			},
		},
	}
}

func TestNewStateManager(t *testing.T) {
	backend := NewMockBackend()
	manager := NewStateManager(backend)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.backend)
	assert.NotNil(t, manager.parser)
	assert.NotNil(t, manager.validator)
	assert.NotNil(t, manager.cache)
}

func TestStateManager_GetState(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		setupMock   func(*MockBackend, string)
		wantErr     bool
		errContains string
		validate    func(*testing.T, *TerraformState)
	}{
		{
			name: "Get valid state",
			key:  "test-state",
			setupMock: func(m *MockBackend, key string) {
				state := createValidState()
				data, _ := json.Marshal(state)
				m.On("Get", mock.Anything, key).Return(data, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, state *TerraformState) {
				assert.Equal(t, 4, state.Version)
				assert.Equal(t, "1.5.0", state.TerraformVersion)
				assert.Len(t, state.Resources, 1)
			},
		},
		{
			name: "Get state from cache",
			key:  "cached-state",
			setupMock: func(m *MockBackend, key string) {
				// This should not be called if cached
				m.On("Get", mock.Anything, key).Maybe().Return(nil, errors.New("should use cache"))
			},
			wantErr: false,
			validate: func(t *testing.T, state *TerraformState) {
				assert.NotNil(t, state)
			},
		},
		{
			name: "Backend error",
			key:  "error-state",
			setupMock: func(m *MockBackend, key string) {
				m.On("Get", mock.Anything, key).Return(nil, errors.New("backend error"))
			},
			wantErr:     true,
			errContains: "failed to get state from backend",
		},
		{
			name: "Invalid JSON",
			key:  "invalid-json",
			setupMock: func(m *MockBackend, key string) {
				m.On("Get", mock.Anything, key).Return([]byte("not-json"), nil)
			},
			wantErr:     true,
			errContains: "failed to parse state",
		},
		{
			name: "Invalid state version",
			key:  "invalid-version",
			setupMock: func(m *MockBackend, key string) {
				state := map[string]interface{}{
					"version": 99,
					"lineage": "test",
				}
				data, _ := json.Marshal(state)
				m.On("Get", mock.Anything, key).Return(data, nil)
			},
			wantErr:     true,
			errContains: "state validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := NewMockBackend()
			manager := NewStateManager(backend)

			// Special handling for cache test
			if tt.name == "Get state from cache" {
				// First, populate the cache
				state := createValidState()
				manager.cache.Set(tt.key, state)
			}

			tt.setupMock(backend, tt.key)

			state, err := manager.GetState(context.Background(), tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				require.NotNil(t, state)
				if tt.validate != nil {
					tt.validate(t, state)
				}
			}

			backend.AssertExpectations(t)
		})
	}
}

func TestStateManager_PutState(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		state       *TerraformState
		setupMock   func(*MockBackend)
		wantErr     bool
		errContains string
	}{
		{
			name:  "Put valid state",
			key:   "test-state",
			state: createValidState(),
			setupMock: func(m *MockBackend) {
				m.On("Lock", mock.Anything, "test-state").Return(nil)
				m.On("Put", mock.Anything, "test-state", mock.Anything).Return(nil)
				m.On("Unlock", mock.Anything, "test-state").Return(nil)
			},
			wantErr: false,
		},
		{
			name:  "Lock acquisition failure",
			key:   "locked-state",
			state: createValidState(),
			setupMock: func(m *MockBackend) {
				m.On("Lock", mock.Anything, "locked-state").Return(errors.New("lock already held"))
			},
			wantErr:     true,
			errContains: "failed to acquire lock",
		},
		{
			name:  "Backend put error",
			key:   "error-state",
			state: createValidState(),
			setupMock: func(m *MockBackend) {
				m.On("Lock", mock.Anything, "error-state").Return(nil)
				m.On("Put", mock.Anything, "error-state", mock.Anything).Return(errors.New("write error"))
				m.On("Unlock", mock.Anything, "error-state").Return(nil)
			},
			wantErr:     true,
			errContains: "failed to put state",
		},
		{
			name:  "Invalid state validation",
			key:   "invalid-state",
			state: &TerraformState{Version: 0},
			setupMock: func(m *MockBackend) {
				// Lock should not be called if validation fails
			},
			wantErr:     true,
			errContains: "state validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := NewMockBackend()
			manager := NewStateManager(backend)

			tt.setupMock(backend)

			err := manager.PutState(context.Background(), tt.key, tt.state)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				// Verify cache was updated
				cached := manager.cache.Get(tt.key)
				assert.NotNil(t, cached)
			}

			backend.AssertExpectations(t)
		})
	}
}

func TestStateManager_DeleteState(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		setupMock   func(*MockBackend)
		wantErr     bool
		errContains string
	}{
		{
			name: "Delete existing state",
			key:  "test-state",
			setupMock: func(m *MockBackend) {
				m.On("Lock", mock.Anything, "test-state").Return(nil)
				m.On("Delete", mock.Anything, "test-state").Return(nil)
				m.On("Unlock", mock.Anything, "test-state").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Delete non-existent state",
			key:  "missing-state",
			setupMock: func(m *MockBackend) {
				m.On("Lock", mock.Anything, "missing-state").Return(nil)
				m.On("Delete", mock.Anything, "missing-state").Return(errors.New("not found"))
				m.On("Unlock", mock.Anything, "missing-state").Return(nil)
			},
			wantErr:     true,
			errContains: "failed to delete state",
		},
		{
			name: "Lock failure",
			key:  "locked-state",
			setupMock: func(m *MockBackend) {
				m.On("Lock", mock.Anything, "locked-state").Return(errors.New("lock held"))
			},
			wantErr:     true,
			errContains: "failed to acquire lock",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := NewMockBackend()
			manager := NewStateManager(backend)

			// Pre-populate cache
			if tt.name == "Delete existing state" {
				manager.cache.Set(tt.key, createValidState())
			}

			tt.setupMock(backend)

			err := manager.DeleteState(context.Background(), tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				// Verify cache was cleared
				cached := manager.cache.Get(tt.key)
				assert.Nil(t, cached)
			}

			backend.AssertExpectations(t)
		})
	}
}

func TestStateManager_ListStates(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*MockBackend)
		want        []string
		wantErr     bool
		errContains string
	}{
		{
			name: "List multiple states",
			setupMock: func(m *MockBackend) {
				m.On("ListStates", mock.Anything).Return([]string{
					"state1.tfstate",
					"state2.tfstate",
					"project/state3.tfstate",
				}, nil)
			},
			want: []string{
				"state1.tfstate",
				"state2.tfstate",
				"project/state3.tfstate",
			},
			wantErr: false,
		},
		{
			name: "Empty list",
			setupMock: func(m *MockBackend) {
				m.On("ListStates", mock.Anything).Return([]string{}, nil)
			},
			want:    []string{},
			wantErr: false,
		},
		{
			name: "Backend error",
			setupMock: func(m *MockBackend) {
				m.On("ListStates", mock.Anything).Return(nil, errors.New("list error"))
			},
			wantErr:     true,
			errContains: "failed to list states",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := NewMockBackend()
			manager := NewStateManager(backend)

			tt.setupMock(backend)

			states, err := manager.ListStates(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, states)
			}

			backend.AssertExpectations(t)
		})
	}
}

func TestStateManager_GetStateMetadata(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		setupMock   func(*MockBackend)
		wantErr     bool
		errContains string
		validate    func(*testing.T, *StateMetadata)
	}{
		{
			name: "Get metadata for valid state",
			key:  "test-state",
			setupMock: func(m *MockBackend) {
				state := createValidState()
				data, _ := json.Marshal(state)
				m.On("Get", mock.Anything, "test-state").Return(data, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, metadata *StateMetadata) {
				assert.Equal(t, "test-state", metadata.Key)
				assert.Equal(t, 4, metadata.Version)
				assert.Equal(t, 1, metadata.Serial)
				assert.Equal(t, "test-lineage", metadata.Lineage)
				assert.Equal(t, 1, metadata.ResourceCount)
				assert.NotEmpty(t, metadata.Checksum)
			},
		},
		{
			name: "Get metadata for non-existent state",
			key:  "missing-state",
			setupMock: func(m *MockBackend) {
				m.On("Get", mock.Anything, "missing-state").Return(nil, errors.New("not found"))
			},
			wantErr:     true,
			errContains: "failed to get state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := NewMockBackend()
			manager := NewStateManager(backend)

			tt.setupMock(backend)

			metadata, err := manager.GetStateMetadata(context.Background(), tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				require.NotNil(t, metadata)
				if tt.validate != nil {
					tt.validate(t, metadata)
				}
			}

			backend.AssertExpectations(t)
		})
	}
}

func TestStateManager_CompareStates(t *testing.T) {
	state1 := createValidState()
	state2 := createValidState()
	state2.Serial = 2
	state2.Resources = append(state2.Resources, Resource{
		Mode:     "managed",
		Type:     "aws_s3_bucket",
		Name:     "bucket",
		Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
		Instances: []Instance{
			{
				Attributes: map[string]interface{}{
					"id":     "test-bucket",
					"bucket": "test-bucket",
				},
			},
		},
	})

	tests := []struct {
		name     string
		state1   *TerraformState
		state2   *TerraformState
		expected *StateComparison
	}{
		{
			name:   "Identical states",
			state1: state1,
			state2: state1,
			expected: &StateComparison{
				AreEqual:      true,
				AddedResources:   []Resource{},
				RemovedResources: []Resource{},
				ModifiedResources: []Resource{},
			},
		},
		{
			name:   "Different states with added resource",
			state1: state1,
			state2: state2,
			expected: &StateComparison{
				AreEqual:      false,
				SerialDiff:    1,
				AddedResources: []Resource{
					state2.Resources[1],
				},
				RemovedResources:  []Resource{},
				ModifiedResources: []Resource{},
			},
		},
		{
			name:   "Different states with removed resource",
			state1: state2,
			state2: state1,
			expected: &StateComparison{
				AreEqual:      false,
				SerialDiff:    -1,
				AddedResources:   []Resource{},
				RemovedResources: []Resource{
					state2.Resources[1],
				},
				ModifiedResources: []Resource{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := NewMockBackend()
			manager := NewStateManager(backend)

			comparison := manager.CompareStates(tt.state1, tt.state2)

			assert.Equal(t, tt.expected.AreEqual, comparison.AreEqual)
			assert.Equal(t, tt.expected.SerialDiff, comparison.SerialDiff)
			assert.Equal(t, len(tt.expected.AddedResources), len(comparison.AddedResources))
			assert.Equal(t, len(tt.expected.RemovedResources), len(comparison.RemovedResources))
		})
	}
}

func TestStateManager_MoveResource(t *testing.T) {
	tests := []struct {
		name        string
		state       *TerraformState
		fromAddress string
		toAddress   string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *TerraformState)
	}{
		{
			name:        "Move existing resource",
			state:       createValidState(),
			fromAddress: "aws_instance.test",
			toAddress:   "aws_instance.renamed",
			wantErr:     false,
			validate: func(t *testing.T, state *TerraformState) {
				assert.Len(t, state.Resources, 1)
				assert.Equal(t, "renamed", state.Resources[0].Name)
			},
		},
		{
			name:        "Move non-existent resource",
			state:       createValidState(),
			fromAddress: "aws_instance.missing",
			toAddress:   "aws_instance.renamed",
			wantErr:     true,
			errContains: "resource not found",
		},
		{
			name: "Move to existing resource",
			state: &TerraformState{
				Version: 4,
				Lineage: "test",
				Resources: []Resource{
					{
						Type: "aws_instance",
						Name: "source",
						Mode: "managed",
						Instances: []Instance{
							{Attributes: map[string]interface{}{"id": "i-1"}},
						},
					},
					{
						Type: "aws_instance",
						Name: "target",
						Mode: "managed",
						Instances: []Instance{
							{Attributes: map[string]interface{}{"id": "i-2"}},
						},
					},
				},
			},
			fromAddress: "aws_instance.source",
			toAddress:   "aws_instance.target",
			wantErr:     true,
			errContains: "target resource already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := NewMockBackend()
			manager := NewStateManager(backend)

			err := manager.MoveResource(tt.state, tt.fromAddress, tt.toAddress)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, tt.state)
				}
			}
		})
	}
}

func TestStateManager_RemoveResource(t *testing.T) {
	tests := []struct {
		name        string
		state       *TerraformState
		address     string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *TerraformState)
	}{
		{
			name:    "Remove existing resource",
			state:   createValidState(),
			address: "aws_instance.test",
			wantErr: false,
			validate: func(t *testing.T, state *TerraformState) {
				assert.Len(t, state.Resources, 0)
			},
		},
		{
			name:        "Remove non-existent resource",
			state:       createValidState(),
			address:     "aws_instance.missing",
			wantErr:     true,
			errContains: "resource not found",
		},
		{
			name: "Remove resource with index",
			state: &TerraformState{
				Version: 4,
				Lineage: "test",
				Resources: []Resource{
					{
						Type: "aws_instance",
						Name: "cluster",
						Mode: "managed",
						Instances: []Instance{
							{Attributes: map[string]interface{}{"id": "i-1"}},
							{Attributes: map[string]interface{}{"id": "i-2"}},
							{Attributes: map[string]interface{}{"id": "i-3"}},
						},
					},
				},
			},
			address: "aws_instance.cluster[1]",
			wantErr: false,
			validate: func(t *testing.T, state *TerraformState) {
				assert.Len(t, state.Resources, 1)
				assert.Len(t, state.Resources[0].Instances, 2)
				assert.Equal(t, "i-1", state.Resources[0].Instances[0].Attributes["id"])
				assert.Equal(t, "i-3", state.Resources[0].Instances[1].Attributes["id"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := NewMockBackend()
			manager := NewStateManager(backend)

			err := manager.RemoveResource(tt.state, tt.address)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, tt.state)
				}
			}
		})
	}
}

func TestStateManager_ConcurrentAccess(t *testing.T) {
	backend := NewMockBackend()
	manager := NewStateManager(backend)

	state := createValidState()
	stateData, _ := json.Marshal(state)

	// Setup mock for concurrent access
	backend.On("Get", mock.Anything, mock.Anything).Return(stateData, nil)
	backend.On("Lock", mock.Anything, mock.Anything).Return(nil)
	backend.On("Put", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	backend.On("Unlock", mock.Anything, mock.Anything).Return(nil)

	var wg sync.WaitGroup
	errors := make([]error, 10)

	// Concurrent reads and writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			if index%2 == 0 {
				// Read operation
				_, err := manager.GetState(context.Background(), fmt.Sprintf("state%d", index))
				errors[index] = err
			} else {
				// Write operation
				err := manager.PutState(context.Background(), fmt.Sprintf("state%d", index), state)
				errors[index] = err
			}
		}(i)
	}

	wg.Wait()

	// Check that no errors occurred
	for i, err := range errors {
		assert.NoError(t, err, "Operation %d failed", i)
	}
}

func TestStateManager_StateVersioning(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		setupMock   func(*MockBackend)
		wantErr     bool
		errContains string
		validate    func(*testing.T, []StateVersion)
	}{
		{
			name: "List state versions",
			key:  "versioned-state",
			setupMock: func(m *MockBackend) {
				versions := []StateVersion{
					{Version: 3, Serial: 10, Timestamp: time.Now().Add(-2 * time.Hour)},
					{Version: 2, Serial: 9, Timestamp: time.Now().Add(-4 * time.Hour)},
					{Version: 1, Serial: 8, Timestamp: time.Now().Add(-6 * time.Hour)},
				}
				m.On("ListStateVersions", mock.Anything, "versioned-state").Return(versions, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, versions []StateVersion) {
				assert.Len(t, versions, 3)
				assert.Equal(t, 3, versions[0].Version)
				assert.Equal(t, 10, versions[0].Serial)
			},
		},
		{
			name: "No versions available",
			key:  "new-state",
			setupMock: func(m *MockBackend) {
				m.On("ListStateVersions", mock.Anything, "new-state").Return([]StateVersion{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, versions []StateVersion) {
				assert.Len(t, versions, 0)
			},
		},
		{
			name: "Backend error",
			key:  "error-state",
			setupMock: func(m *MockBackend) {
				m.On("ListStateVersions", mock.Anything, "error-state").Return(nil, errors.New("backend error"))
			},
			wantErr:     true,
			errContains: "failed to list state versions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := NewMockBackend()
			manager := NewStateManager(backend)

			tt.setupMock(backend)

			versions, err := manager.ListStateVersions(context.Background(), tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, versions)
				}
			}

			backend.AssertExpectations(t)
		})
	}
}

func TestStateManager_RestoreStateVersion(t *testing.T) {
	oldState := &TerraformState{
		Version:          4,
		TerraformVersion: "1.4.0",
		Serial:           5,
		Lineage:          "test-lineage",
		Resources: []Resource{
			{
				Type: "aws_instance",
				Name: "old",
				Mode: "managed",
				Instances: []Instance{
					{Attributes: map[string]interface{}{"id": "i-old"}},
				},
			},
		},
	}

	tests := []struct {
		name        string
		key         string
		version     int
		setupMock   func(*MockBackend)
		wantErr     bool
		errContains string
		validate    func(*testing.T, *TerraformState)
	}{
		{
			name:    "Restore previous version",
			key:     "test-state",
			version: 2,
			setupMock: func(m *MockBackend) {
				data, _ := json.Marshal(oldState)
				m.On("GetStateVersion", mock.Anything, "test-state", 2).Return(data, nil)
				m.On("Lock", mock.Anything, "test-state").Return(nil)
				m.On("Put", mock.Anything, "test-state", mock.Anything).Return(nil)
				m.On("Unlock", mock.Anything, "test-state").Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, state *TerraformState) {
				assert.Equal(t, "old", state.Resources[0].Name)
				assert.True(t, state.Serial > 5) // Serial should be incremented
			},
		},
		{
			name:    "Version not found",
			key:     "test-state",
			version: 99,
			setupMock: func(m *MockBackend) {
				m.On("GetStateVersion", mock.Anything, "test-state", 99).Return(nil, errors.New("version not found"))
			},
			wantErr:     true,
			errContains: "failed to get state version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := NewMockBackend()
			manager := NewStateManager(backend)

			tt.setupMock(backend)

			state, err := manager.RestoreStateVersion(context.Background(), tt.key, tt.version)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				require.NotNil(t, state)
				if tt.validate != nil {
					tt.validate(t, state)
				}
			}

			backend.AssertExpectations(t)
		})
	}
}