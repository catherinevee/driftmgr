package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/catherinevee/driftmgr/internal/cache"
	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/monitoring"
	"github.com/gorilla/mux"
)

const (
	serviceName = "discovery-service"
	servicePort = "8081"
)

var (
	discoverer   *discovery.EnhancedDiscoverer
	cacheManager = cache.GetGlobalManager()
	logger       = monitoring.GetGlobalLogger()
)

func main() {
	// Initialize the discoverer
	discoverer = discovery.NewEnhancedDiscoverer()

	// Set up router
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", handleHealth).Methods("GET")

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Discovery endpoints
	api.HandleFunc("/discover/{provider}/{region}", handleDiscoverRegion).Methods("GET")
	api.HandleFunc("/discover/{provider}/all", handleDiscoverAll).Methods("GET")
	api.HandleFunc("/resources/{provider}/{type}", handleGetResources).Methods("GET")
	api.HandleFunc("/resources/{provider}", handleGetAllResources).Methods("GET")

	// Monitoring endpoints
	api.HandleFunc("/monitor/start", handleStartMonitoring).Methods("POST")
	api.HandleFunc("/monitor/stop", handleStopMonitoring).Methods("POST")
	api.HandleFunc("/monitor/status", handleMonitoringStatus).Methods("GET")

	// Cache endpoints
	api.HandleFunc("/cache/stats", handleCacheStats).Methods("GET")
	api.HandleFunc("/cache/clear", handleCacheClear).Methods("POST")

	// Start server
	logger.Info("Starting discovery service on port " + servicePort)
	log.Fatal(http.ListenAndServe(":"+servicePort, router))
}

// handleHealth handles health check requests
func handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"service":   serviceName,
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDiscoverRegion handles discovery for a specific region
func handleDiscoverRegion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	provider := vars["provider"]
	region := vars["region"]

	// Check cache first
	cacheKey := fmt.Sprintf("discovery:%s:%s", provider, region)
	if cached, found := cacheManager.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Perform discovery
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	resources, err := discoverer.DiscoverResources(ctx, provider, region)
	if err != nil {
		http.Error(w, fmt.Sprintf("Discovery failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Cache the results
	cacheManager.Set(cacheKey, resources, 15*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resources)
}

// handleDiscoverAll handles discovery for all regions of a provider
func handleDiscoverAll(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	provider := vars["provider"]

	// Check cache first
	cacheKey := fmt.Sprintf("discovery:%s:all", provider)
	if cached, found := cacheManager.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Perform discovery for all regions
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	allResources := make(map[string][]models.Resource)

	// Get regions for the provider
	regions := getRegionsForProvider(provider)

	for _, region := range regions {
		resources, err := discoverer.DiscoverResources(ctx, provider, region)
		if err != nil {
			logger.Warn("Failed to discover resources in region " + region + ": " + err.Error())
			continue
		}
		allResources[region] = resources
	}

	// Cache the results
	cacheManager.Set(cacheKey, allResources, 15*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allResources)
}

// handleGetResources handles getting resources of a specific type
func handleGetResources(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	provider := vars["provider"]
	resourceType := vars["type"]

	// Check cache first
	cacheKey := fmt.Sprintf("resources:%s:%s", provider, resourceType)
	if cached, found := cacheManager.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Get resources by type
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	resources, err := discoverer.GetResourcesByType(ctx, provider, resourceType)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get resources: %v", err), http.StatusInternalServerError)
		return
	}

	// Cache the results
	cacheManager.Set(cacheKey, resources, 10*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resources)
}

// handleGetAllResources handles getting all resources for a provider
func handleGetAllResources(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	provider := vars["provider"]

	// Check cache first
	cacheKey := fmt.Sprintf("resources:%s:all", provider)
	if cached, found := cacheManager.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Get all resources
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	resources, err := discoverer.GetAllResources(ctx, provider)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get resources: %v", err), http.StatusInternalServerError)
		return
	}

	// Cache the results
	cacheManager.Set(cacheKey, resources, 10*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resources)
}

// handleStartMonitoring starts real-time monitoring
func handleStartMonitoring(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Provider string   `json:"provider"`
		Regions  []string `json:"regions"`
		Interval int      `json:"interval"` // seconds
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Start monitoring
	err := discoverer.StartMonitoring(request.Provider, request.Regions, time.Duration(request.Interval)*time.Second)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to start monitoring: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "monitoring_started",
		"message": fmt.Sprintf("Started monitoring %s in regions %v", request.Provider, request.Regions),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleStopMonitoring stops real-time monitoring
func handleStopMonitoring(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Provider string `json:"provider"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Stop monitoring
	err := discoverer.StopMonitoring(request.Provider)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to stop monitoring: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "monitoring_stopped",
		"message": fmt.Sprintf("Stopped monitoring %s", request.Provider),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMonitoringStatus gets the status of monitoring
func handleMonitoringStatus(w http.ResponseWriter, r *http.Request) {
	status := discoverer.GetMonitoringStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleCacheStats gets cache statistics
func handleCacheStats(w http.ResponseWriter, r *http.Request) {
	stats := cacheManager.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleCacheClear clears the cache
func handleCacheClear(w http.ResponseWriter, r *http.Request) {
	cacheManager.Clear()

	response := map[string]interface{}{
		"status":  "cache_cleared",
		"message": "Cache cleared successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getRegionsForProvider returns the list of regions for a given provider
func getRegionsForProvider(provider string) []string {
	switch provider {
	case "aws":
		return []string{
			"us-east-1", "us-east-2", "us-west-1", "us-west-2",
			"eu-west-1", "eu-west-2", "eu-central-1",
			"ap-southeast-1", "ap-southeast-2", "ap-northeast-1",
		}
	case "azure":
		return []string{
			"eastus", "westus", "centralus", "northeurope", "westeurope",
		}
	case "gcp":
		return []string{
			"us-central1", "us-east1", "us-west1", "europe-west1", "asia-east1",
		}
	default:
		return []string{}
	}
}
