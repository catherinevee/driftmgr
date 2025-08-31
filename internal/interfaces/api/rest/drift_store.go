package api

import (
	"fmt"
	"sync"
	"time"
)

// Global drift store instance
var globalDriftStore *DriftStore
var driftStoreOnce sync.Once

// GetGlobalDriftStore returns the global drift store instance
func GetGlobalDriftStore() *DriftStore {
	driftStoreOnce.Do(func() {
		globalDriftStore = NewDriftStore()
	})
	return globalDriftStore
}

// DriftRecord represents a detected drift
type DriftRecord struct {
	ID           string                 `json:"id"`
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	Provider     string                 `json:"provider"`
	Region       string                 `json:"region"`
	DriftType    string                 `json:"drift_type"`
	Severity     string                 `json:"severity"`
	Changes      map[string]interface{} `json:"changes"`
	DetectedAt   time.Time              `json:"detected_at"`
	ResolvedAt   *time.Time             `json:"resolved_at,omitempty"`
	Status       string                 `json:"status"` // active, resolved, ignored
}

// DriftStore manages drift detection results
type DriftStore struct {
	mu     sync.RWMutex
	drifts map[string]*DriftRecord
}

// NewDriftStore creates a new drift store
func NewDriftStore() *DriftStore {
	return &DriftStore{
		drifts: make(map[string]*DriftRecord),
	}
}

// AddDrift adds a new drift record
func (ds *DriftStore) AddDrift(drift *DriftRecord) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	
	if drift.ID == "" {
		drift.ID = fmt.Sprintf("drift-%d", time.Now().Unix())
	}
	drift.DetectedAt = time.Now()
	drift.Status = "active"
	
	ds.drifts[drift.ID] = drift
}

// GetAllDrifts returns all active drifts
func (ds *DriftStore) GetAllDrifts() []*DriftRecord {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	var results []*DriftRecord
	for _, drift := range ds.drifts {
		if drift.Status == "active" {
			results = append(results, drift)
		}
	}
	return results
}

// GetDriftByID returns a specific drift by ID
func (ds *DriftStore) GetDriftByID(id string) (*DriftRecord, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	drift, exists := ds.drifts[id]
	return drift, exists
}

// GetDriftsByProvider returns drifts for a specific provider
func (ds *DriftStore) GetDriftsByProvider(provider string) []*DriftRecord {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	var results []*DriftRecord
	for _, drift := range ds.drifts {
		if drift.Provider == provider && drift.Status == "active" {
			results = append(results, drift)
		}
	}
	return results
}

// ResolveDrift marks a drift as resolved
func (ds *DriftStore) ResolveDrift(id string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	
	if drift, ok := ds.drifts[id]; ok {
		now := time.Now()
		drift.ResolvedAt = &now
		drift.Status = "resolved"
	}
}

// GetRecentDrifts returns the most recent drift records up to the specified limit
func (ds *DriftStore) GetRecentDrifts(limit int) []*DriftRecord {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	// Get all active drifts
	var allDrifts []*DriftRecord
	for _, drift := range ds.drifts {
		if drift.Status == "active" {
			allDrifts = append(allDrifts, drift)
		}
	}
	
	// Sort by detection time (most recent first) - simple implementation
	// In production, use a proper sorting algorithm
	if len(allDrifts) > limit {
		allDrifts = allDrifts[:limit]
	}
	
	return allDrifts
}

// GetDriftStats returns drift statistics
func (ds *DriftStore) GetDriftStats() map[string]interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total_active": 0,
		"by_severity": map[string]int{
			"critical": 0,
			"high":     0,
			"medium":   0,
			"low":      0,
		},
		"by_provider": make(map[string]int),
		"by_type": make(map[string]int),
	}
	
	for _, drift := range ds.drifts {
		if drift.Status != "active" {
			continue
		}
		
		stats["total_active"] = stats["total_active"].(int) + 1
		
		// By severity
		if severity, ok := stats["by_severity"].(map[string]int); ok {
			severity[drift.Severity]++
		}
		
		// By provider
		if provider, ok := stats["by_provider"].(map[string]int); ok {
			provider[drift.Provider]++
		}
		
		// By type
		if byType, ok := stats["by_type"].(map[string]int); ok {
			byType[drift.DriftType]++
		}
	}
	
	return stats
}

// ClearOldDrifts removes drifts older than the specified duration
func (ds *DriftStore) ClearOldDrifts(maxAge time.Duration) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	
	now := time.Now()
	for id, drift := range ds.drifts {
		if drift.ResolvedAt != nil && now.Sub(*drift.ResolvedAt) > maxAge {
			delete(ds.drifts, id)
		}
	}
}