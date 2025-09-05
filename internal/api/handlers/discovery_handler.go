package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// DiscoveryHandler handles discovery-related API requests
type DiscoveryHandler struct {
	hub *DiscoveryHub
}

// NewDiscoveryHandler creates a new discovery handler
func NewDiscoveryHandler(hub *DiscoveryHub) *DiscoveryHandler {
	return &DiscoveryHandler{
		hub: hub,
	}
}

// RegisterRoutes registers discovery routes
func (h *DiscoveryHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/discover", h.StartDiscovery).Methods("POST")
	router.HandleFunc("/api/v1/discover/status", h.GetDiscoveryStatus).Methods("GET")
	router.HandleFunc("/api/v1/discover/results", h.GetDiscoveryResults).Methods("GET")
	router.HandleFunc("/api/v1/discovery/cached", h.GetCachedResources).Methods("GET")
	router.HandleFunc("/api/v1/discovery/clear-cache", h.ClearCache).Methods("POST")
}

// StartDiscovery handles POST /api/v1/discover
func (h *DiscoveryHandler) StartDiscovery(w http.ResponseWriter, r *http.Request) {
	var req DiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set async to true for API calls
	req.Async = true

	jobID := h.hub.StartDiscovery(req)
	
	response := map[string]interface{}{
		"job_id": jobID,
		"status": "started",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetDiscoveryStatus handles GET /api/v1/discover/status
func (h *DiscoveryHandler) GetDiscoveryStatus(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		http.Error(w, "job_id is required", http.StatusBadRequest)
		return
	}

	status := h.hub.GetJobStatus(jobID)
	if status == nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// GetDiscoveryResults handles GET /api/v1/discover/results
func (h *DiscoveryHandler) GetDiscoveryResults(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		http.Error(w, "job_id is required", http.StatusBadRequest)
		return
	}

	results := h.hub.GetJobResults(jobID)
	if results == nil {
		http.Error(w, "Results not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// GetCachedResources handles GET /api/v1/discovery/cached
func (h *DiscoveryHandler) GetCachedResources(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	region := r.URL.Query().Get("region")

	if provider == "" {
		provider = "aws" // Default provider
	}
	if region == "" {
		region = "us-east-1" // Default region
	}

	resources := h.hub.GetCachedResources()
	// TODO: Filter by provider and region if needed

	response := map[string]interface{}{
		"resources": resources,
		"count":     len(resources),
		"cached":    true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ClearCache handles POST /api/v1/discovery/clear-cache
func (h *DiscoveryHandler) ClearCache(w http.ResponseWriter, r *http.Request) {
	h.hub.ClearCache()

	response := map[string]interface{}{
		"success": true,
		"message": "Cache cleared successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}