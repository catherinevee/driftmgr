package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/drift/detector"
	"github.com/catherinevee/driftmgr/internal/remediation"
	"github.com/catherinevee/driftmgr/internal/shared/cache"
	"github.com/catherinevee/driftmgr/internal/shared/events"
	"github.com/catherinevee/driftmgr/internal/state"
	"github.com/catherinevee/driftmgr/pkg/models"
)

// Server represents the main handler server
type Server struct {
	startTime        time.Time
	mu               sync.RWMutex
	// discoveryService will be added when discovery.Service is implemented
	driftDetector    *detector.DriftDetector
	// remediator and stateManager will be added when packages are implemented
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

// NewServer creates a new handler server
func NewServer(
	// discoveryService will be added when discovery.Service is implemented,
	driftDetector *detector.DriftDetector,
	// remediator and stateManager will be added later
) *Server {
	return &Server{
		startTime:        time.Now(),
		// discoveryService: discoveryService,
		driftDetector:    driftDetector,
		// remediator:       remediator,
		// stateManager:     stateManager,
		wsClients:        make(map[string]interface{}),
		broadcast:        make(chan interface{}, 100),
		discoveryJobs:    make(map[string]*DiscoveryJob),
		discoveryHub:     NewDiscoveryHub(nil), // Pass nil for now
		driftStore:       NewDriftStore(),
		remediationStore: NewRemediationStore(),
	}
}

// DiscoveryJob type is defined in discovery_jobs.go

// JobStatus is an alias for DiscoveryJob for backward compatibility
type JobStatus = DiscoveryJob

// DiscoveryHub manages discovery operations and caching
type DiscoveryHub struct {
	mu               sync.RWMutex
	jobs             map[string]*JobStatus
	results          map[string][]models.Resource
	cache            []models.Resource
	cacheTime        time.Time
	cacheTTL         time.Duration
	cacheVersion     int
	cacheMetadata    CacheMetadata
	cachedResources  map[string]interface{}
	wsBroadcast      func(messageType string, data map[string]interface{})
	eventBus         *events.EventBus
	globalCache      *cache.GlobalCache
	// discoveryService is removed - we handle discovery directly

	// Cache metrics
	cacheHits        atomic.Int64
	cacheMisses      atomic.Int64
	cacheEvictions   atomic.Int64
	metricsStartTime time.Time

	// Drift detection
	driftRecords   []*DriftRecord
	lastDriftCheck time.Time
	terraformState map[string]interface{}
	stateFilePath  string
}

// CacheMetadata tracks cache freshness and versioning
type CacheMetadata struct {
	Version       int       `json:"version"`
	LastUpdated   time.Time `json:"last_updated"`
	ResourceCount int       `json:"resource_count"`
	Sources       []string  `json:"sources"`
	Freshness     string    `json:"freshness"`
	TTL           int       `json:"ttl_seconds"`
}

// DriftRecord represents a drift detection result
type DriftRecord struct {
	ID           string                 `json:"id"`
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	Provider     string                 `json:"provider"`
	Region       string                 `json:"region"`
	DriftType    string                 `json:"drift_type"`
	Actual       map[string]interface{} `json:"actual"`
	Expected     map[string]interface{} `json:"expected"`
	Differences  []Difference           `json:"differences"`
	DetectedAt   time.Time              `json:"detected_at"`
	Severity     string                 `json:"severity"`
	Status       string                 `json:"status"`
}

// Difference represents a specific drift difference
type Difference struct {
	Path     string      `json:"path"`
	Actual   interface{} `json:"actual"`
	Expected interface{} `json:"expected"`
	Type     string      `json:"type"`
}

// DriftStore manages drift records
type DriftStore struct {
	mu     sync.RWMutex
	drifts []interface{}
}

// RemediationStore manages remediation jobs
type RemediationStore struct {
	mu   sync.RWMutex
	jobs []interface{}
}

// RemediationJob represents a remediation job
type RemediationJob struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	Resources   []models.Resource      `json:"resources"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Results     map[string]interface{} `json:"results,omitempty"`
	DryRun      bool                   `json:"dry_run"`
	AutoApprove bool                   `json:"auto_approve"`
}

// NewDiscoveryHub creates a new discovery hub
func NewDiscoveryHub(globalCache *cache.GlobalCache, eventBus *events.EventBus) *DiscoveryHub {
	return &DiscoveryHub{
		jobs:             make(map[string]*JobStatus),
		results:          make(map[string][]models.Resource),
		cache:            []models.Resource{},
		cacheTTL:         5 * time.Minute,
		globalCache:      globalCache,
		eventBus:         eventBus,
		cachedResources:  make(map[string]interface{}),
		metricsStartTime: time.Now(),
		driftRecords:     make([]*DriftRecord, 0),
		terraformState:   make(map[string]interface{}),
	}
}

// NewDriftStore creates a new drift store
func NewDriftStore() *DriftStore {
	return &DriftStore{
		drifts: make([]interface{}, 0),
	}
}

// NewRemediationStore creates a new remediation store
func NewRemediationStore() *RemediationStore {
	return &RemediationStore{
		jobs: make([]interface{}, 0),
	}
}

// Helper methods

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

// GetCachedResources and GetCacheMetrics are implemented in discovery_hub.go

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

// Discovery service types for handlers

// DiscoveryRequest represents a discovery request
type DiscoveryRequest struct {
	Provider      string                 `json:"provider"`
	Regions       []string               `json:"regions"`
	ResourceTypes []string               `json:"resource_types,omitempty"`
	Async         bool                   `json:"async"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// DiscoveryResponse represents a discovery response
type DiscoveryResponse struct {
	JobID     string            `json:"job_id,omitempty"`
	Resources []models.Resource `json:"resources,omitempty"`
	Error     string            `json:"error,omitempty"`
}

// StartDiscovery is implemented in discovery_hub.go