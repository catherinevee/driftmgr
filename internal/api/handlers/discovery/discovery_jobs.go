package discovery

import (
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// DiscoveryJob represents an active discovery job
type DiscoveryJob struct {
	ID          string                 `json:"id"`
	Status      string                 `json:"status"` // pending, running, completed, failed
	Progress    int                    `json:"progress"`
	Message     string                 `json:"message"`
	Providers   []string               `json:"providers"`
	Regions     []string               `json:"regions"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Resources   []models.Resource             `json:"resources,omitempty"`
	Summary     map[string]interface{} `json:"summary,omitempty"`
	mu          sync.RWMutex
}

// UpdateProgress updates the job progress
func (j *DiscoveryJob) UpdateProgress(progress int, message string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Progress = progress
	j.Message = message
}

// SetStatus updates the job status
func (j *DiscoveryJob) SetStatus(status string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = status
	if status == "completed" || status == "failed" {
		now := time.Now()
		j.CompletedAt = &now
	}
}

// SetError sets an error on the job
func (j *DiscoveryJob) SetError(err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = "failed"
	j.Error = err.Error()
	now := time.Now()
	j.CompletedAt = &now
}

// SetResources sets the discovered resources
func (j *DiscoveryJob) SetResources(resources []apimodels.Resource) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Resources = resources
	
	// Calculate summary
	j.Summary = map[string]interface{}{
		"total": len(resources),
		"by_provider": make(map[string]int),
		"by_type": make(map[string]int),
		"by_region": make(map[string]int),
	}
	
	for _, r := range resources {
		// Count by provider
		if providers, ok := j.Summary["by_provider"].(map[string]int); ok {
			providers[r.Provider]++
		}
		// Count by type
		if types, ok := j.Summary["by_type"].(map[string]int); ok {
			types[r.Type]++
		}
		// Count by region
		if regions, ok := j.Summary["by_region"].(map[string]int); ok {
			regions[r.Region]++
		}
	}
}

// GetSnapshot returns a snapshot of the job state
func (j *DiscoveryJob) GetSnapshot() DiscoveryJob {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return *j
}

/*
// handleDiscoveryStart starts a new discovery job
func (s *Server) handleDiscoveryStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Providers []string `json:"providers"`
		Regions   []string `json:"regions"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Default to all providers if none specified
	if len(req.Providers) == 0 {
		req.Providers = []string{"aws", "azure", "gcp", "digitalocean"}
	}
	
	// Create new discovery job
	job := &DiscoveryJob{
		ID:        fmt.Sprintf("discovery-%s", uuid.New().String()[:8]),
		Status:    "pending",
		Progress:  0,
		Message:   "Initializing discovery...",
		Providers: req.Providers,
		Regions:   req.Regions,
		StartedAt: time.Now(),
	}
	
	// Store job
	s.discoveryMu.Lock()
	s.discoveryJobs[job.ID] = job
	s.discoveryMu.Unlock()
	
	// Start discovery in background
	go s.runDiscoveryJob(job)
	
	// Send job ID back to client
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id": job.ID,
		"status": job.Status,
	})
	
	// Broadcast job start via WebSocket
	s.broadcastMessage(map[string]interface{}{
		"type":   "discovery_started",
		"job_id": job.ID,
		"status": job.Status,
	})
}
*/

/*
// runDiscoveryJob runs a discovery job in the background
func (s *Server) runDiscoveryJob(job *DiscoveryJob) {
	// Set job as running
	job.SetStatus("running")
	job.UpdateProgress(10, "Starting discovery process...")
	
	// Broadcast status update
	s.broadcastMessage(map[string]interface{}{
		"type":     "discovery_update",
		"job_id":   job.ID,
		"status":   "running",
		"progress": job.Progress,
		"message":  job.Message,
	})
	
	// Use the real discovery service
	ctx := context.Background()
	allResources := []Resource{}
	
	totalProviders := len(job.Providers)
	for i, provider := range job.Providers {
		// Calculate progress
		baseProgress := (i * 80) / totalProviders
		job.UpdateProgress(baseProgress+10, fmt.Sprintf("Discovering %s resources...", provider))
		
		// Broadcast progress
		s.broadcastMessage(map[string]interface{}{
			"type":     "discovery_update",
			"job_id":   job.ID,
			"progress": job.Progress,
			"message":  job.Message,
		})
		
		// Run discovery for this provider
		if s.discoveryService != nil {
			options := &discovery.Options{
				Regions:  job.Regions,
				Parallel: true,
			}
			
			resources, err := s.discoveryService.DiscoverResources(ctx, provider, options)
			if err != nil {
				// Log error but continue with other providers
				fmt.Printf("Discovery error for %s: %v\n", provider, err)
				continue
			}
			
			// Convert to API resources
			for _, res := range resources {
				apiResource := Resource{
					ID:       res.ID,
					Name:     res.Name,
					Type:     res.Type,
					Provider: provider,
					Region:   res.Region,
					Status:   "discovered",
					Tags:     res.Tags,
				}
				allResources = append(allResources, apiResource)
			}
		}
	}
	
	// Set final results
	job.SetResources(allResources)
	job.SetStatus("completed")
	job.UpdateProgress(100, fmt.Sprintf("Discovery completed: %d resources found", len(allResources)))
	
	// Store results in discovery hub cache
	if s.discoveryHub != nil {
		s.discoveryHub.PrePopulateCache(allResources)
	}
	
	// Persist discovery results to database
	if s.persistence != nil {
		if err := s.persistence.SaveDiscoveryResults(job.ID, allResources); err != nil {
			fmt.Printf("Failed to persist discovery results: %v\n", err)
		}
	}
	
	// Broadcast completion
	s.broadcastMessage(map[string]interface{}{
		"type":     "discovery_completed",
		"job_id":   job.ID,
		"status":   "completed",
		"progress": 100,
		"message":  job.Message,
		"summary":  job.Summary,
	})
}

// handleDiscoveryStatus returns the status of a discovery job
func (s *Server) handleDiscoveryStatus(w http.ResponseWriter, r *http.Request) {
	// Get job ID from path
	vars := mux.Vars(r)
	jobID := vars["id"]
	
	// If no specific job ID, check query parameter
	if jobID == "" {
		jobID = r.URL.Query().Get("job_id")
	}
	
	if jobID == "" {
		// Return all jobs
		s.discoveryMu.RLock()
		jobs := make([]DiscoveryJob, 0, len(s.discoveryJobs))
		for _, job := range s.discoveryJobs {
			jobs = append(jobs, job.GetSnapshot())
		}
		s.discoveryMu.RUnlock()
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobs)
		return
	}
	
	// Get specific job
	s.discoveryMu.RLock()
	job, exists := s.discoveryJobs[jobID]
	s.discoveryMu.RUnlock()
	
	if !exists {
		sendJSONError(w, "Job not found", http.StatusNotFound)
		return
	}
	
	// Return job status
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job.GetSnapshot())
}

// handleDiscoveryResults returns the results of a completed discovery job
func (s *Server) handleDiscoveryResults(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		// Return cached results from discovery hub
		if s.discoveryHub != nil {
			resources := s.discoveryHub.GetCachedResources()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"resources": resources,
				"total":     len(resources),
			})
			return
		}
		sendJSONError(w, "No discovery results available", http.StatusNotFound)
		return
	}
	
	// Get specific job results
	s.discoveryMu.RLock()
	job, exists := s.discoveryJobs[jobID]
	s.discoveryMu.RUnlock()
	
	if !exists {
		sendJSONError(w, "Job not found", http.StatusNotFound)
		return
	}
	
	snapshot := job.GetSnapshot()
	if snapshot.Status != "completed" {
		sendJSONError(w, "Job not completed yet", http.StatusBadRequest)
		return
	}
	
	// Return results
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id":    snapshot.ID,
		"resources": snapshot.Resources,
		"summary":   snapshot.Summary,
		"total":     len(snapshot.Resources),
	})
}

// broadcastMessage sends a message to all WebSocket clients
func (s *Server) broadcastMessage(message interface{}) {
	// Use the existing broadcast channel
	select {
	case s.broadcast <- message:
	default:
		// Channel is full, skip
	}
}

// CleanupOldJobs removes jobs older than specified duration
func (s *Server) CleanupOldJobs(maxAge time.Duration) {
	s.discoveryMu.Lock()
	defer s.discoveryMu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	for id, job := range s.discoveryJobs {
		if job.CompletedAt != nil && job.CompletedAt.Before(cutoff) {
			delete(s.discoveryJobs, id)
		}
	}
}
*/