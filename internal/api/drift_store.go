package api

import (
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/drift/detector"
)

// DriftStore manages drift detection results
type DriftStore struct {
	results map[string]*detector.DriftResult
	mu      sync.RWMutex
}

var (
	globalDriftStore *DriftStore
	driftStoreOnce   sync.Once
)

// GetGlobalDriftStore returns the global drift store instance
func GetGlobalDriftStore() *DriftStore {
	driftStoreOnce.Do(func() {
		globalDriftStore = &DriftStore{
			results: make(map[string]*detector.DriftResult),
		}
	})
	return globalDriftStore
}

// Store stores a drift result
func (ds *DriftStore) Store(id string, result *detector.DriftResult) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.results[id] = result
}

// Get retrieves a drift result by ID
func (ds *DriftStore) Get(id string) (*detector.DriftResult, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	result, ok := ds.results[id]
	return result, ok
}

// List returns all stored drift results
func (ds *DriftStore) List() []*detector.DriftResult {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	results := make([]*detector.DriftResult, 0, len(ds.results))
	for _, result := range ds.results {
		results = append(results, result)
	}
	return results
}

// Delete removes a drift result
func (ds *DriftStore) Delete(id string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	delete(ds.results, id)
}

// Clear removes all drift results
func (ds *DriftStore) Clear() {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.results = make(map[string]*detector.DriftResult)
}

// CleanupOld removes drift results older than the specified duration
func (ds *DriftStore) CleanupOld(maxAge time.Duration) int {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	now := time.Now()
	removed := 0

	for id, result := range ds.results {
		if result.Timestamp.Before(now.Add(-maxAge)) {
			delete(ds.results, id)
			removed++
		}
	}

	return removed
}
