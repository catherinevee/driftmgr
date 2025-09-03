package config

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// DiscoveryJob represents a discovery job
type DiscoveryJob struct {
	ID          string     `json:"id"`
	JobType     string     `json:"job_type"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// DiscoveryHub manages discovery operations
type DiscoveryHub struct {
	cachedResources map[string]interface{}
	cacheHits       atomic.Int64
	cacheMisses     atomic.Int64
	cacheEvictions  atomic.Int64
	mu              sync.RWMutex
}

// DriftStore manages drift data
type DriftStore struct {
	drifts []interface{}
	mu     sync.RWMutex
}

// RemediationStore manages remediation jobs
type RemediationStore struct {
	jobs []interface{}
	mu   sync.RWMutex
}

// Server represents the config handler server
type Server struct {
	startTime        time.Time
	mu               sync.RWMutex
	discoveryService interface{}
	driftDetector    interface{}
	remediator       interface{}
	cacheIntegration interface{}
	authHandler      interface{}
	wsClients        map[string]interface{}
	wsClientsMu      sync.RWMutex
	broadcast        chan interface{}
	discoveryJobs    map[string]*DiscoveryJob
	discoveryMu      sync.RWMutex
	discoveryHub     *DiscoveryHub
	driftStore       *DriftStore
	remediationStore *RemediationStore
	wsMessagesSent   atomic.Int64
}

// NewServer creates a new config handler server
func NewServer() *Server {
	return &Server{
		startTime:        time.Now(),
		wsClients:        make(map[string]interface{}),
		broadcast:        make(chan interface{}, 100),
		discoveryJobs:    make(map[string]*DiscoveryJob),
		discoveryHub:     &DiscoveryHub{cachedResources: make(map[string]interface{})},
		driftStore:       &DriftStore{drifts: make([]interface{}, 0)},
		remediationStore: &RemediationStore{jobs: make([]interface{}, 0)},
	}
}

// respondJSON sends a JSON response
func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response
func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]string{"error": message})
}

// GetWSMessagesSent returns the count of WebSocket messages sent
func (s *Server) GetWSMessagesSent() int64 {
	return s.wsMessagesSent.Load()
}

// GetCachedResources returns cached resources
func (dh *DiscoveryHub) GetCachedResources() map[string]interface{} {
	dh.mu.RLock()
	defer dh.mu.RUnlock()
	return dh.cachedResources
}

// GetCacheMetrics returns cache metrics
func (dh *DiscoveryHub) GetCacheMetrics() (int64, int64, int64) {
	return dh.cacheHits.Load(), dh.cacheMisses.Load(), dh.cacheEvictions.Load()
}

// GetDriftCount returns the count of drifts
func (ds *DriftStore) GetDriftCount() int {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return len(ds.drifts)
}

// GetAllJobs returns all remediation jobs
func (rs *RemediationStore) GetAllJobs() []interface{} {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.jobs
}