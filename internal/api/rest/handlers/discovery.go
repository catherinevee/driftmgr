package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/gorilla/mux"
)

// DiscoveryRequest represents a discovery request
type DiscoveryRequest struct {
	Providers   []string          `json:"providers"`
	AllAccounts bool              `json:"allAccounts"`
	Regions     []string          `json:"regions"`
	Filters     map[string]string `json:"filters"`
	Async       bool              `json:"async"`
}

// DiscoveryResult represents the result of a discovery operation
type DiscoveryResult struct {
	Resources     []interface{} `json:"resources"`
	ResourceCount int           `json:"resource_count"`
	Duration      string        `json:"duration"`
	Errors        []string      `json:"errors,omitempty"`
}

// DiscoveryResponse represents a discovery response
type DiscoveryResponse struct {
	JobID   string                      `json:"jobId,omitempty"`
	Results map[string]*DiscoveryResult `json:"results,omitempty"`
	Status  string                      `json:"status"`
	Message string                      `json:"message"`
}

// JobUpdate represents a job status update
type JobUpdate struct {
	JobID    string      `json:"jobId"`
	Status   string      `json:"status"`
	Progress float64     `json:"progress"`
	Message  string      `json:"message,omitempty"`
	Result   interface{} `json:"result,omitempty"`
}

// startDiscovery starts a discovery job
func (s *EnhancedDashboardServer) startDiscovery(w http.ResponseWriter, r *http.Request) {
	var req DiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Default to all providers if none specified
	if len(req.Providers) == 0 {
		req.Providers = []string{"aws", "azure", "gcp", "digitalocean"}
	}

	// Create a job for async discovery
	job := s.jobManager.CreateJob("discovery")

	// Start discovery in background
	go s.performDiscovery(job, req)

	// Return job ID for tracking
	resp := DiscoveryResponse{
		JobID:   job.ID,
		Status:  "started",
		Message: fmt.Sprintf("Discovery job started for providers: %s", strings.Join(req.Providers, ", ")),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// performDiscovery performs the actual discovery
func (s *EnhancedDashboardServer) performDiscovery(job *Job, req DiscoveryRequest) {
	ctx := context.Background()

	// Update job status
	s.jobManager.UpdateJob(job.ID, "running", 10)

	// Broadcast start event
	s.broadcast <- map[string]interface{}{
		"type":  "discovery_started",
		"jobId": job.ID,
		"data":  req,
	}

	var allResults = make(map[string]*DiscoveryResult)
	progressPerProvider := 80 / len(req.Providers) // Reserve 20% for finalization
	currentProgress := 10

	for _, provider := range req.Providers {
		s.jobManager.UpdateJob(job.ID, "running", float64(currentProgress))

		// Discover resources for this provider
		result, err := s.discoveryService.DiscoverProvider(ctx, provider, discovery.DiscoveryOptions{})
		if err != nil {
			// Log error but continue with other providers
			s.broadcast <- map[string]interface{}{
				"type":     "discovery_error",
				"jobId":    job.ID,
				"provider": provider,
				"error":    err.Error(),
			}
		} else {
			// Convert result to DiscoveryResult
			allResults[provider] = &DiscoveryResult{
				Resources:     convertResourcesToInterfaces(result.Resources),
				ResourceCount: len(result.Resources),
				Duration:      result.Duration.String(),
			}

			// Store resources
			resourceInterfaces := convertResourcesToInterfaces(result.Resources)
			for _, res := range resourceInterfaces {
				s.dataStore.SetResources(append(s.dataStore.GetResources(), res))
			}

			// Broadcast provider completion
			s.broadcast <- map[string]interface{}{
				"type":     "provider_discovered",
				"jobId":    job.ID,
				"provider": provider,
				"count":    allResults[provider].ResourceCount,
				"regions":  []string{},
			}
		}

		currentProgress += progressPerProvider
		s.jobManager.UpdateJob(job.ID, "running", float64(currentProgress))
	}

	// Finalize
	s.jobManager.UpdateJob(job.ID, "running", 90)

	// Calculate statistics
	totalResources := 0
	for _, result := range allResults {
		totalResources += result.ResourceCount
	}
	stats := map[string]interface{}{
		"TotalResources": totalResources,
		"Providers":      len(allResults),
	}

	// Store job result
	job.Result = map[string]interface{}{
		"results": allResults,
		"stats":   stats,
	}

	// Update job completion
	s.jobManager.UpdateJob(job.ID, "completed", 100)

	// Broadcast completion
	s.broadcast <- map[string]interface{}{
		"type":  "discovery_completed",
		"jobId": job.ID,
		"stats": stats,
	}
}

// listDiscoveryJobs lists all discovery jobs
func (s *EnhancedDashboardServer) listDiscoveryJobs(w http.ResponseWriter, r *http.Request) {
	jobs := s.jobManager.ListJobs()

	// Filter to only discovery jobs
	discoveryJobs := make([]*Job, 0)
	for _, job := range jobs {
		if job.Type == "discovery" {
			discoveryJobs = append(discoveryJobs, job)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(discoveryJobs)
}

// getDiscoveryJob gets a specific discovery job
func (s *EnhancedDashboardServer) getDiscoveryJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["jobId"]

	job, exists := s.jobManager.GetJob(jobID)
	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// cancelDiscoveryJob cancels a discovery job
func (s *EnhancedDashboardServer) cancelDiscoveryJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["jobId"]

	job, exists := s.jobManager.GetJob(jobID)
	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	if job.Status == "completed" || job.Status == "failed" {
		http.Error(w, "Job already finished", http.StatusBadRequest)
		return
	}

	// Update job status
	s.jobManager.UpdateJob(jobID, "cancelled", job.Progress)

	// Broadcast cancellation
	s.broadcast <- map[string]interface{}{
		"type":  "discovery_cancelled",
		"jobId": jobID,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

// streamDiscoveryProgress streams discovery progress using Server-Sent Events
func (s *EnhancedDashboardServer) streamDiscoveryProgress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["jobId"]

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create event channel
	eventChan := make(chan JobUpdate, 10)

	// Register for updates (simplified - in production, use proper pub/sub)
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				job, exists := s.jobManager.GetJob(jobID)
				if exists {
					eventChan <- JobUpdate{
						JobID:    jobID,
						Status:   job.Status,
						Progress: job.Progress,
						Message:  "",
					}

					if job.Status == "completed" || job.Status == "failed" || job.Status == "cancelled" {
						close(eventChan)
						return
					}
				}
			case <-r.Context().Done():
				close(eventChan)
				return
			}
		}
	}()

	// Send events
	for event := range eventChan {
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

// convertResourcesToInterfaces converts models.Resource to []interface{}
func convertResourcesToInterfaces(resources []models.Resource) []interface{} {
	interfaces := make([]interface{}, len(resources))
	for i, res := range resources {
		interfaces[i] = res
	}
	return interfaces
}
