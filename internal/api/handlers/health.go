package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

// HealthStatus represents the health status of the service
type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]interface{} `json:"checks,omitempty"`
}

// MetricsData represents system metrics
type MetricsData struct {
	Timestamp time.Time              `json:"timestamp"`
	System    SystemMetrics          `json:"system"`
	Discovery DiscoveryMetrics       `json:"discovery"`
	Drift     DriftMetrics           `json:"drift"`
	Cache     CacheMetrics           `json:"cache"`
	WebSocket WebSocketMetrics       `json:"websocket"`
}

// SystemMetrics represents system-level metrics
type SystemMetrics struct {
	Uptime         time.Duration `json:"uptime"`
	GoRoutines     int           `json:"goroutines"`
	MemoryAllocMB  uint64        `json:"memory_alloc_mb"`
	MemoryTotalMB  uint64        `json:"memory_total_mb"`
	NumGC          uint32        `json:"num_gc"`
	CPUCount       int           `json:"cpu_count"`
	Version        string        `json:"version"`
}

// DiscoveryMetrics represents discovery-related metrics
type DiscoveryMetrics struct {
	TotalResources   int `json:"total_resources"`
	ActiveJobs       int `json:"active_jobs"`
	CompletedJobs    int `json:"completed_jobs"`
	FailedJobs       int `json:"failed_jobs"`
	LastDiscoveryAt  *time.Time `json:"last_discovery_at,omitempty"`
}

// DriftMetrics represents drift detection metrics
type DriftMetrics struct {
	TotalDrifts      int `json:"total_drifts"`
	ActiveDetections int `json:"active_detections"`
	RemediationJobs  int `json:"remediation_jobs"`
}

// CacheMetrics represents cache metrics
type CacheMetrics struct {
	CacheSize        int `json:"cache_size"`
	CacheHits        int `json:"cache_hits"`
	CacheMisses      int `json:"cache_misses"`
	CacheEvictions   int `json:"cache_evictions"`
}

// WebSocketMetrics represents WebSocket metrics
type WebSocketMetrics struct {
	ConnectedClients int `json:"connected_clients"`
	MessagesSent     int `json:"messages_sent"`
	MessagesQueued   int `json:"messages_queued"`
}

var startTime = time.Now()

// handleHealth returns basic health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Checks: map[string]interface{}{
			"discovery_service": false, // TODO: s.discoveryService != nil,
			"drift_detector":    s.driftDetector != nil,
			"remediation":       false, // TODO: s.remediator != nil,
			"cache":             false, // TODO: s.cacheIntegration != nil,
			"websocket":         len(s.wsClients) >= 0,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleHealthLive returns liveness probe status
func (s *Server) handleHealthLive(w http.ResponseWriter, r *http.Request) {
	// Simple liveness check - service is running
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "live",
		"timestamp": time.Now(),
	})
}

// handleHealthReady returns readiness probe status
func (s *Server) handleHealthReady(w http.ResponseWriter, r *http.Request) {
	// Check if all critical components are ready
	ready := true
	checks := map[string]bool{
		"discovery": false, // TODO: s.discoveryService != nil,
		"drift":     s.driftDetector != nil,
		"auth":      false, // TODO: s.authHandler != nil,
	}
	
	for _, check := range checks {
		if !check {
			ready = false
			break
		}
	}
	
	status := map[string]interface{}{
		"ready":     ready,
		"timestamp": time.Now(),
		"checks":    checks,
	}
	
	w.Header().Set("Content-Type", "application/json")
	if !ready {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(status)
}

// handleMetrics returns system metrics
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Collect system metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	systemMetrics := SystemMetrics{
		Uptime:        time.Since(startTime),
		GoRoutines:    runtime.NumGoroutine(),
		MemoryAllocMB: memStats.Alloc / 1024 / 1024,
		MemoryTotalMB: memStats.TotalAlloc / 1024 / 1024,
		NumGC:         memStats.NumGC,
		CPUCount:      runtime.NumCPU(),
		Version:       "1.0.0", // You can get this from build info
	}
	
	// Collect discovery metrics
	s.discoveryMu.RLock()
	activeJobs := 0
	completedJobs := 0
	failedJobs := 0
	var lastDiscovery *time.Time
	
	for _, job := range s.discoveryJobs {
		switch job.Status {
		case "running", "pending":
			activeJobs++
		case "completed":
			completedJobs++
			if lastDiscovery == nil || job.CompletedAt.After(*lastDiscovery) {
				lastDiscovery = job.CompletedAt
			}
		case "failed":
			failedJobs++
		}
	}
	s.discoveryMu.RUnlock()
	
	discoveryMetrics := DiscoveryMetrics{
		TotalResources:  len(s.discoveryHub.GetCachedResources()),
		ActiveJobs:      activeJobs,
		CompletedJobs:   completedJobs,
		FailedJobs:      failedJobs,
		LastDiscoveryAt: lastDiscovery,
	}
	
	// Collect drift metrics - count active detection jobs
	activeDetections := 0
	for _, job := range s.discoveryJobs {
		if job.Status == "running" { // && job.JobType == "drift-detection" {
			activeDetections++
		}
	}
	
	driftMetrics := DriftMetrics{
		TotalDrifts:      s.driftStore.GetDriftCount(),
		ActiveDetections: activeDetections,
		RemediationJobs:  len(s.remediationStore.GetAllJobs()),
	}
	
	// Collect cache metrics with real data
	cacheHits, cacheMisses, cacheEvictions := s.discoveryHub.GetCacheMetrics()
	cacheMetrics := CacheMetrics{
		CacheSize:      len(s.discoveryHub.GetCachedResources()),
		CacheHits:      int(cacheHits),
		CacheMisses:    int(cacheMisses),
		CacheEvictions: int(cacheEvictions),
	}
	
	// Collect WebSocket metrics
	s.wsClientsMu.RLock()
	wsMetrics := WebSocketMetrics{
		ConnectedClients: len(s.wsClients),
		MessagesSent:     int(s.GetWSMessagesSent()),
		MessagesQueued:   len(s.broadcast),
	}
	s.wsClientsMu.RUnlock()
	
	metrics := MetricsData{
		Timestamp: time.Now(),
		System:    systemMetrics,
		Discovery: discoveryMetrics,
		Drift:     driftMetrics,
		Cache:     cacheMetrics,
		WebSocket: wsMetrics,
	}
	
	// Support Prometheus format if requested
	if r.Header.Get("Accept") == "text/plain" || r.URL.Query().Get("format") == "prometheus" {
		w.Header().Set("Content-Type", "text/plain")
		writePrometheusMetrics(w, metrics)
		return
	}
	
	// Return JSON by default
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// writePrometheusMetrics writes metrics in Prometheus format
func writePrometheusMetrics(w http.ResponseWriter, metrics MetricsData) {
	// System metrics
	fmt.Fprintf(w, "# HELP driftmgr_uptime_seconds Service uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE driftmgr_uptime_seconds gauge\n")
	fmt.Fprintf(w, "driftmgr_uptime_seconds %f\n", metrics.System.Uptime.Seconds())
	
	fmt.Fprintf(w, "# HELP driftmgr_goroutines Number of goroutines\n")
	fmt.Fprintf(w, "# TYPE driftmgr_goroutines gauge\n")
	fmt.Fprintf(w, "driftmgr_goroutines %d\n", metrics.System.GoRoutines)
	
	fmt.Fprintf(w, "# HELP driftmgr_memory_alloc_bytes Memory allocated in bytes\n")
	fmt.Fprintf(w, "# TYPE driftmgr_memory_alloc_bytes gauge\n")
	fmt.Fprintf(w, "driftmgr_memory_alloc_bytes %d\n", metrics.System.MemoryAllocMB*1024*1024)
	
	// Discovery metrics
	fmt.Fprintf(w, "# HELP driftmgr_discovery_resources_total Total discovered resources\n")
	fmt.Fprintf(w, "# TYPE driftmgr_discovery_resources_total gauge\n")
	fmt.Fprintf(w, "driftmgr_discovery_resources_total %d\n", metrics.Discovery.TotalResources)
	
	fmt.Fprintf(w, "# HELP driftmgr_discovery_jobs_active Active discovery jobs\n")
	fmt.Fprintf(w, "# TYPE driftmgr_discovery_jobs_active gauge\n")
	fmt.Fprintf(w, "driftmgr_discovery_jobs_active %d\n", metrics.Discovery.ActiveJobs)
	
	// Drift metrics
	fmt.Fprintf(w, "# HELP driftmgr_drifts_total Total detected drifts\n")
	fmt.Fprintf(w, "# TYPE driftmgr_drifts_total gauge\n")
	fmt.Fprintf(w, "driftmgr_drifts_total %d\n", metrics.Drift.TotalDrifts)
	
	// WebSocket metrics
	fmt.Fprintf(w, "# HELP driftmgr_websocket_clients Connected WebSocket clients\n")
	fmt.Fprintf(w, "# TYPE driftmgr_websocket_clients gauge\n")
	fmt.Fprintf(w, "driftmgr_websocket_clients %d\n", metrics.WebSocket.ConnectedClients)
}