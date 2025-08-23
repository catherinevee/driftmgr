package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Storage interface for data persistence
type Storage interface {
	// State operations
	SaveState(ctx context.Context, key string, data interface{}) error
	LoadState(ctx context.Context, key string, dest interface{}) error
	DeleteState(ctx context.Context, key string) error
	ListStates(ctx context.Context, prefix string) ([]string, error)

	// Resource operations
	SaveResources(ctx context.Context, provider string, resources interface{}) error
	LoadResources(ctx context.Context, provider string) (interface{}, error)

	// Snapshot operations
	CreateSnapshot(ctx context.Context, data interface{}) (string, error)
	LoadSnapshot(ctx context.Context, snapshotID string) (interface{}, error)
	ListSnapshots(ctx context.Context) ([]SnapshotInfo, error)
}

// SnapshotInfo contains snapshot metadata
type SnapshotInfo struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Provider  string                 `json:"provider"`
	Size      int64                  `json:"size"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// MemoryStorage implements in-memory storage
type MemoryStorage struct {
	data map[string][]byte
	mu   sync.RWMutex
}

// NewMemoryStorage creates a new in-memory storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string][]byte),
	}
}

// SaveState saves state to memory
func (m *MemoryStorage) SaveState(ctx context.Context, key string, data interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	m.data[key] = bytes
	return nil
}

// LoadState loads state from memory
func (m *MemoryStorage) LoadState(ctx context.Context, key string, dest interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bytes, exists := m.data[key]
	if !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	return json.Unmarshal(bytes, dest)
}

// DeleteState deletes state from memory
func (m *MemoryStorage) DeleteState(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
	return nil
}

// ListStates lists all state keys with prefix
func (m *MemoryStorage) ListStates(ctx context.Context, prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0)
	for key := range m.data {
		if len(prefix) == 0 || len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

// SaveResources saves resources
func (m *MemoryStorage) SaveResources(ctx context.Context, provider string, resources interface{}) error {
	key := fmt.Sprintf("resources:%s", provider)
	return m.SaveState(ctx, key, resources)
}

// LoadResources loads resources
func (m *MemoryStorage) LoadResources(ctx context.Context, provider string) (interface{}, error) {
	key := fmt.Sprintf("resources:%s", provider)
	var resources interface{}
	err := m.LoadState(ctx, key, &resources)
	return resources, err
}

// CreateSnapshot creates a snapshot
func (m *MemoryStorage) CreateSnapshot(ctx context.Context, data interface{}) (string, error) {
	id := fmt.Sprintf("snapshot-%d", time.Now().Unix())
	return id, m.SaveState(ctx, id, data)
}

// LoadSnapshot loads a snapshot
func (m *MemoryStorage) LoadSnapshot(ctx context.Context, snapshotID string) (interface{}, error) {
	var data interface{}
	err := m.LoadState(ctx, snapshotID, &data)
	return data, err
}

// ListSnapshots lists all snapshots
func (m *MemoryStorage) ListSnapshots(ctx context.Context) ([]SnapshotInfo, error) {
	keys, err := m.ListStates(ctx, "snapshot-")
	if err != nil {
		return nil, err
	}

	snapshots := make([]SnapshotInfo, 0, len(keys))
	for _, key := range keys {
		snapshots = append(snapshots, SnapshotInfo{
			ID:        key,
			Timestamp: time.Now(), // In production, would parse from key or metadata
		})
	}

	return snapshots, nil
}

// DataStore provides in-memory storage for application data
type DataStore struct {
	mu                 sync.RWMutex
	resources          []interface{}
	drifts             []interface{}
	remediationHistory []interface{}
	data               map[string]interface{}
}

// NewDataStore creates a new data store
func NewDataStore() *DataStore {
	return &DataStore{
		resources:          []interface{}{},
		drifts:             []interface{}{},
		remediationHistory: []interface{}{},
		data:               make(map[string]interface{}),
	}
}

// SetResources sets the resources
func (ds *DataStore) SetResources(resources []interface{}) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.resources = resources
}

// GetResources gets the resources
func (ds *DataStore) GetResources() []interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.resources
}

// SetDrifts sets the drift items
func (ds *DataStore) SetDrifts(drifts []interface{}) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.drifts = drifts
}

// GetDrifts gets the drift items
func (ds *DataStore) GetDrifts() []interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.drifts
}

// Set stores a value with a key
func (ds *DataStore) Set(key string, value interface{}) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.data[key] = value
}

// Get retrieves a value by key
func (ds *DataStore) Get(key string) (interface{}, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	val, ok := ds.data[key]
	return val, ok
}

// AddRemediationHistory adds a remediation history item
func (ds *DataStore) AddRemediationHistory(item interface{}) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.remediationHistory = append(ds.remediationHistory, item)
}

// GetRemediationHistory gets the remediation history
func (ds *DataStore) GetRemediationHistory() []interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.remediationHistory
}
