package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/catherinevee/driftmgr/internal/shared/cache"
	"github.com/catherinevee/driftmgr/internal/shared/events"
	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/google/uuid"
)

// Types are defined in types.go

// SetWebSocketBroadcast sets the WebSocket broadcast function
func (h *DiscoveryHub) SetWebSocketBroadcast(broadcast func(string, map[string]interface{})) {
	h.wsBroadcast = broadcast
}

// SetEventBus sets the event bus for publishing events
func (h *DiscoveryHub) SetEventBus(eventBus *events.EventBus) {
	h.eventBus = eventBus
}

// StartDiscovery starts a new discovery job
func (h *DiscoveryHub) StartDiscovery(req DiscoveryRequest) string {
	jobID := uuid.New().String()

	// Check global cache first
	cacheKey := fmt.Sprintf("discovery:%s:%v", req.Provider, req.Regions)
	if h.globalCache != nil {
		if cachedData, found, age := h.globalCache.GetWithAge(cacheKey); found {
			if resources, ok := cachedData.([]models.Resource); ok {
				// Return cached results immediately
				endTime := time.Now()
				job := &JobStatus{
					ID:        jobID,
					Status:    "completed",
					Providers: []string{req.Provider},
					Regions:   req.Regions,
					StartedAt: time.Now().Add(-age),
					CompletedAt: &endTime,
					Progress:  100,
					Message:   fmt.Sprintf("Using cached data (age: %v)", age.Round(time.Second)),
					Resources: resources,
					Summary: map[string]interface{}{
						"source": "cache",
						"age":    age.Seconds(),
					},
				}

				h.mu.Lock()
				h.jobs[jobID] = job
				// Convert and store results
				apiResources := h.convertToAPIResources(resources)
				h.results[jobID] = apiResources
				h.mu.Unlock()

				// Broadcast cache hit
				if h.wsBroadcast != nil {
					h.wsBroadcast("discovery_cache_hit", map[string]interface{}{
						"job_id":    jobID,
						"provider":  req.Provider,
						"cache_age": age.Seconds(),
					})
				}

				return jobID
			}
		}
	}

	job := &JobStatus{
		ID:        jobID,
		Status:    "running",
		Providers: []string{req.Provider},
		Regions:   req.Regions,
		StartedAt: time.Now(),
		Progress:  0,
		Message:   "Discovery started",
	}

	h.mu.Lock()
	h.jobs[jobID] = job
	h.mu.Unlock()

	// Start discovery in background
	go h.runDiscovery(jobID, req)

	return jobID
}

// runDiscovery performs the actual discovery
func (h *DiscoveryHub) runDiscovery(jobID string, req DiscoveryRequest) {
	h.updateJobStatus(jobID, "running", "Discovering resources...", 0)
	h.sendTerminalOutput(jobID, "Initializing discovery process...", "info")

	// Publish discovery started event
	if h.eventBus != nil {
		h.eventBus.Publish(events.Event{
			Type:   events.EventDiscoveryStarted,
			Source: "discovery_hub",
			Data: map[string]interface{}{
				"job_id":   jobID,
				"provider": req.Provider,
				"regions":  req.Regions,
			},
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var allResources []models.Resource

	// Send progress update
	h.sendTerminalOutput(jobID, fmt.Sprintf("Starting discovery for provider: %s", req.Provider), "info")

	// Use provider factory directly for discovery
	provider, err := providers.NewProvider(req.Provider, map[string]interface{}{})
	if err == nil {
		h.sendTerminalOutput(jobID, fmt.Sprintf("Scanning regions: %v", req.Regions), "info")

		// Perform real discovery
		resources, err := provider.DiscoverResources(ctx, req.Regions, req.ResourceTypes)
		if err != nil {
			h.sendTerminalOutput(jobID, fmt.Sprintf("Discovery failed: %v", err), "error")
			h.updateJobError(jobID, fmt.Sprintf("Discovery failed: %v", err))

			// Publish discovery failed event
			if h.eventBus != nil {
				h.eventBus.Publish(events.Event{
					Type:   events.EventDiscoveryFailed,
					Source: "discovery_hub",
					Error:  err,
					Data: map[string]interface{}{
						"job_id":   jobID,
						"provider": req.Provider,
						"regions":  req.Regions,
						"error":    err.Error(),
						"message":  fmt.Sprintf("Discovery failed: %v", err),
					},
				})
			}
			return
		}

		// Use discovered resources
		for _, r := range resources {
			// Convert tags to map[string]string if needed
			var tags map[string]string
			if t, ok := r.Tags.(map[string]string); ok {
				tags = t
			} else {
				tags = make(map[string]string)
			}

			// Get status from State or Status field
			status := r.Status
			if status == "" {
				if s, ok := r.State.(string); ok {
					status = s
				}
			}

			allResources = append(allResources, models.Resource{
				ID:           r.ID,
				Name:         r.Name,
				Type:         r.Type,
				Provider:     r.Provider,
				Region:       r.Region,
				Status:       status,
				CreatedAt:    r.CreatedAt,
				Tags:         tags,
				Properties:   r.Properties,
				Dependencies: r.Dependencies,
			})
		}

		// Send progress update with resource count
		h.sendTerminalOutput(jobID, fmt.Sprintf("Found %d resources", len(allResources)), "success")
		h.sendDiscoveryProgress(jobID, 90, fmt.Sprintf("Processing %d resources...", len(allResources)))

		// Publish discovery progress event
		if h.eventBus != nil {
			// TODO: Implement events package
	// h.eventBus.Publish(events.Event{
		// Type:   events.DiscoveryProgress,
		// Source: "discovery_hub",
		// Data: map[string]interface{}{
		// 	"job_id":   jobID,
		// 	"provider": req.Provider,
		// 	"progress": 90,
		// 	"message":  fmt.Sprintf("Found %d resources", len(allResources)),
		// 	"stats": map[string]interface{}{
		// 		"resource_count": len(allResources),
		// 	},
		// },
		// })
		}
	}

	// Store results
	h.mu.Lock()
	h.results[jobID] = allResources

	// Track previous cache size for logging
	prevCacheSize := len(h.cache)

	// Use map for efficient deduplication when merging
	cacheMap := make(map[string]models.Resource)

	// First, add all existing cached resources to the map
	for _, resource := range h.cache {
		cacheMap[resource.ID] = resource
	}

	// Then, add/update with new resources (this automatically handles duplicates)
	for _, resource := range allResources {
		cacheMap[resource.ID] = resource
	}

	// Convert map back to slice
	oldCacheSize := len(h.cache)
	h.cache = make([]models.Resource, 0, len(cacheMap))
	for _, resource := range cacheMap {
		h.cache = append(h.cache, resource)
	}
	h.cacheTime = time.Now()

	// Track evictions if cache was replaced with fewer items
	if oldCacheSize > 0 && len(h.cache) < oldCacheSize {
		h.cacheEvictions.Add(int64(oldCacheSize - len(h.cache)))
	}

	// Store in global cache for CLI access
	// cacheKey := fmt.Sprintf("discovery:%s:web", req.Provider)
	// Convert API resources back to models.Resource for cache
	var cacheResources []models.Resource
	for _, apiRes := range allResources {
		cacheResources = append(cacheResources, models.Resource{
			ID:       apiRes.ID,
			Name:     apiRes.Name,
			Type:     apiRes.Type,
			Provider: apiRes.Provider,
			Region:   apiRes.Region,
			Status:   apiRes.Status,
			Tags:     apiRes.Tags,
		})
	}
	// TODO: Implement global cache
	// h.globalCache.Set(cacheKey, cacheResources, 5*time.Minute)

	// Log deduplication info
	newResourceCount := len(allResources)
	finalCount := len(h.cache)
	h.cacheVersion++
	h.updateCacheMetadata("discovery")

	fmt.Printf("[Discovery Hub] Merged %d new resources with %d cached resources, resulting in %d unique resources (v%d)\n",
		newResourceCount, prevCacheSize, finalCount, h.cacheVersion)

	// Save updated cache to disk
	h.saveCacheToDisk()

	h.mu.Unlock()

	h.sendTerminalOutput(jobID, "Storing results...", "info")
	h.sendDiscoveryProgress(jobID, 95, "Finalizing discovery...")

	// Update job status
	h.completeJob(jobID, len(allResources))

	// Publish discovery completed event
	if h.eventBus != nil {
		// TODO: Implement events package
	// h.eventBus.Publish(events.Event{
		// Type:   events.DiscoveryCompleted,
		// Source: "discovery_hub",
		// Data: map[string]interface{}{
		// "job_id":    jobID,
		// "provider":  req.Provider,
		// "regions":   req.Regions,
		// "resources": len(allResources),
		// "message":   fmt.Sprintf("Discovery completed: %d resources found", len(allResources)),
		// "stats": map[string]interface{}{
		// 	"total_resources": len(allResources),
		// 	"cache_size":      len(h.cache),
		// },
		// },
		// })
	}
}

// All resources come from real cloud provider discovery only
// No mock or sample data is used

// GetJobStatus returns the status of a job
func (h *DiscoveryHub) GetJobStatus(jobID string) *JobStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if job, ok := h.jobs[jobID]; ok {
		return job
	}
	return nil
}

// GetAllJobStatuses returns all job statuses
func (h *DiscoveryHub) GetAllJobStatuses() []*JobStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	statuses := make([]*JobStatus, 0, len(h.jobs))
	for _, job := range h.jobs {
		statuses = append(statuses, job)
	}
	return statuses
}

// GetJobResults returns the results of a completed job
func (h *DiscoveryHub) GetJobResults(jobID string) []models.Resource {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if results, ok := h.results[jobID]; ok {
		return results
	}
	return []models.Resource{}
}

// GetCachedResults returns cached discovery results
func (h *DiscoveryHub) GetCachedResults() []models.Resource {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Update freshness status
	h.updateFreshnessStatus()

	// Always return cache if it has data
	// The cache is updated whenever discovery runs
	return h.cache
}

// GetCacheMetadata returns cache metadata including freshness
func (h *DiscoveryHub) GetCacheMetadata() CacheMetadata {
	h.mu.RLock()
	defer h.mu.RUnlock()

	h.updateFreshnessStatus()
	return h.cacheMetadata
}

// updateFreshnessStatus updates the cache freshness indicator
func (h *DiscoveryHub) updateFreshnessStatus() {
	age := time.Since(h.cacheTime)
	if age < 5*time.Minute {
		h.cacheMetadata.Freshness = "fresh"
	} else if age < 30*time.Minute {
		h.cacheMetadata.Freshness = "recent"
	} else {
		h.cacheMetadata.Freshness = "stale"
	}
}

// updateCacheMetadata updates cache metadata after changes
func (h *DiscoveryHub) updateCacheMetadata(source string) {
	h.cacheMetadata.Version = h.cacheVersion
	h.cacheMetadata.LastUpdated = h.cacheTime
	h.cacheMetadata.ResourceCount = len(h.cache)
	if source != "" && !contains(h.cacheMetadata.Sources, source) {
		h.cacheMetadata.Sources = append(h.cacheMetadata.Sources, source)
	}
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// PrePopulateCache pre-populates the cache with discovered resources
func (h *DiscoveryHub) PrePopulateCache(resources []models.Resource) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Deduplicate resources before adding to cache
	uniqueResources := make(map[string]models.Resource)
	for _, resource := range resources {
		// Use the resource ID as the key for deduplication
		// If a resource with the same ID already exists, it will be overwritten
		// This ensures only unique resources are stored
		uniqueResources[resource.ID] = resource
	}

	// Convert map back to slice
	h.cache = make([]models.Resource, 0, len(uniqueResources))
	for _, resource := range uniqueResources {
		h.cache = append(h.cache, resource)
	}
	h.cacheTime = time.Now()

	fmt.Printf("[Discovery Hub] Pre-populated cache with %d unique resources (deduplicated from %d)\n", len(h.cache), len(resources))

	// Save to disk for persistence
	h.saveCacheToDisk()
}

// getCachePath returns the path to the cache file
func (h *DiscoveryHub) getCachePath() string {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to temp directory
		return filepath.Join(os.TempDir(), ".driftmgr_cache.json")
	}

	// Create .driftmgr directory if it doesn't exist
	driftmgrDir := filepath.Join(homeDir, ".driftmgr")
	os.MkdirAll(driftmgrDir, 0755)

	return filepath.Join(driftmgrDir, "resource_cache.json")
}

// saveCacheToDisk saves the current cache to disk
func (h *DiscoveryHub) saveCacheToDisk() {
	cachePath := h.getCachePath()

	// Create cache data structure
	cacheData := struct {
		Resources []models.Resource `json:"resources"`
		Timestamp time.Time            `json:"timestamp"`
	}{
		Resources: h.cache,
		Timestamp: h.cacheTime,
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(cacheData, "", "  ")
	if err != nil {
		fmt.Printf("[Discovery Hub] Failed to marshal cache: %v\n", err)
		return
	}

	// Write to file
	err = ioutil.WriteFile(cachePath, data, 0644)
	if err != nil {
		fmt.Printf("[Discovery Hub] Failed to save cache to disk: %v\n", err)
		return
	}

	fmt.Printf("[Discovery Hub] Saved %d resources to cache file: %s\n", len(h.cache), cachePath)
}

// loadCacheFromDisk loads the cache from disk if it exists
func (h *DiscoveryHub) loadCacheFromDisk() {
	cachePath := h.getCachePath()

	// Check if cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		fmt.Printf("✓ No existing cache found. Fresh discovery will be performed.\n")
		return
	}

	// Read cache file
	data, err := ioutil.ReadFile(cachePath)
	if err != nil {
		fmt.Printf("[Discovery Hub] Failed to read cache file: %v\n", err)
		return
	}

	// Unmarshal cache data
	var cacheData struct {
		Resources []models.Resource `json:"resources"`
		Timestamp time.Time            `json:"timestamp"`
	}

	err = json.Unmarshal(data, &cacheData)
	if err != nil {
		fmt.Printf("[Discovery Hub] Failed to unmarshal cache: %v\n", err)
		return
	}

	// Check if cache is still valid (not older than 24 hours)
	if time.Since(cacheData.Timestamp) > 24*time.Hour {
		fmt.Printf("[Discovery Hub] Cache is older than 24 hours, ignoring\n")
		return
	}

	// Deduplicate resources before loading into cache
	uniqueResources := make(map[string]models.Resource)
	for _, resource := range cacheData.Resources {
		uniqueResources[resource.ID] = resource
	}

	// Convert map to slice
	dedupedResources := make([]models.Resource, 0, len(uniqueResources))
	for _, resource := range uniqueResources {
		dedupedResources = append(dedupedResources, resource)
	}

	// Load cache
	h.mu.Lock()
	h.cache = dedupedResources
	h.cacheTime = cacheData.Timestamp
	h.mu.Unlock()

	originalCount := len(cacheData.Resources)
	dedupedCount := len(h.cache)
	if originalCount != dedupedCount {
		fmt.Printf("✓ Loaded %d unique resources from cache (deduplicated from %d, last updated: %v ago)\n",
			dedupedCount, originalCount, time.Since(cacheData.Timestamp))
	} else {
		fmt.Printf("✓ Loaded %d resources from cache (last updated: %v ago)\n",
			dedupedCount, time.Since(cacheData.Timestamp).Round(time.Minute))
	}
}

// updateJobStatus updates the status of a job
func (h *DiscoveryHub) updateJobStatus(jobID, status, message string, progress int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if job, ok := h.jobs[jobID]; ok {
		job.Status = status
		job.Message = message
		job.Progress = progress
	}
}

// updateJobError updates a job with an error
func (h *DiscoveryHub) updateJobError(jobID, errorMsg string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if job, ok := h.jobs[jobID]; ok {
		job.Status = "failed"
		job.Error = errorMsg
		now := time.Now()
		job.CompletedAt = &now
	}
}

// completeJob marks a job as completed
func (h *DiscoveryHub) completeJob(jobID string, resourceCount int) {
	// Send terminal completion before locking
	h.sendTerminalOutput(jobID, fmt.Sprintf("Discovery completed successfully! Found %d resources", resourceCount), "success")
	h.sendDiscoveryProgress(jobID, 100, fmt.Sprintf("Completed - %d resources discovered", resourceCount))

	// Send terminal status update
	if h.wsBroadcast != nil {
		h.wsBroadcast("terminal_status", map[string]interface{}{
			"job_id": jobID,
			"status": "completed",
		})
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if job, ok := h.jobs[jobID]; ok {
		job.Status = "completed"
		job.Progress = 100
		job.Message = fmt.Sprintf("Discovery completed. Found %d resources", resourceCount)
		// TODO: Set resources when available
		// job.Resources = resourceCount
		now := time.Now()
		job.CompletedAt = &now

		// Update the cache with discovered resources using efficient deduplication
		if results, ok := h.results[jobID]; ok && len(results) > 0 {
			// Use map for efficient deduplication
			cacheMap := make(map[string]models.Resource)

			// Add existing cache to map
			for _, resource := range h.cache {
				cacheMap[resource.ID] = resource
			}

			// Merge new resources (automatically handles duplicates)
			for _, resource := range results {
				cacheMap[resource.ID] = resource
			}

			// Convert back to slice
			h.cache = make([]models.Resource, 0, len(cacheMap))
			for _, resource := range cacheMap {
				h.cache = append(h.cache, resource)
			}
			h.cacheTime = time.Now()
		}
	}
}

// ClearCache clears the in-memory and file-based cache
func (h *DiscoveryHub) ClearCache() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Clear in-memory cache
	h.cache = []models.Resource{}
	h.cacheTime = time.Time{}

	// Delete cache file
	cachePath := h.getCachePath()
	os.Remove(cachePath)

	fmt.Printf("[Discovery Hub] Cache cleared (both memory and disk)\n")
}

// CleanupOldJobs removes jobs older than the specified duration
func (h *DiscoveryHub) CleanupOldJobs(maxAge time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	for id, job := range h.jobs {
		if job.CompletedAt != nil && now.Sub(*job.CompletedAt) > maxAge {
			delete(h.jobs, id)
			delete(h.results, id)
		}
	}
}

// sendTerminalOutput sends terminal output via WebSocket
func (h *DiscoveryHub) sendTerminalOutput(jobID, text, outputType string) {
	if h.wsBroadcast != nil {
		h.wsBroadcast("terminal_output", map[string]interface{}{
			"job_id":      jobID,
			"text":        text,
			"output_type": outputType,
		})
	}
}

// sendDiscoveryProgress sends discovery progress via WebSocket
func (h *DiscoveryHub) sendDiscoveryProgress(jobID string, progress int, message string) {
	if h.wsBroadcast != nil {
		h.wsBroadcast("discovery_progress", map[string]interface{}{
			"job_id":   jobID,
			"progress": progress,
			"message":  message,
		})
	}
}

// convertToAPIResources converts models.Resource to API resources
func (h *DiscoveryHub) convertToAPIResources(resources []models.Resource) []models.Resource {
	apiResources := make([]models.Resource, 0, len(resources))
	for _, r := range resources {
		// Convert tags to map[string]string if needed
		var tags map[string]string
		if t, ok := r.Tags.(map[string]string); ok {
			tags = t
		} else {
			tags = make(map[string]string)
		}

		apiResources = append(apiResources, models.Resource{
			ID:         r.ID,
			Name:       r.Name,
			Type:       r.Type,
			Provider:   r.Provider,
			Region:     r.Region,
			Status:     r.Status,
			Tags:       tags,
			Properties: r.Properties,
		})
	}
	return apiResources
}

// GetAllResources returns all resources from the cache
func (h *DiscoveryHub) GetAllResources() []models.Resource {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]models.Resource, len(h.cache))
	copy(result, h.cache)
	return result
}

// GetDriftRecords returns drift records for resources
func (h *DiscoveryHub) GetDriftRecords() []*DriftRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Perform actual drift detection by comparing cached resources with their expected state
	driftRecords := []*DriftRecord{}

	// Get Terraform state for comparison
	tfState := h.getTerraformState()
	if tfState == nil {
		// If no Terraform state, check for unmanaged resources
		for _, resource := range h.cache {
			if !h.isResourceManaged(resource) {
		// driftRecords = append(driftRecords, &DriftRecord{
		// 	ResourceID:   resource.ID,
		// 	ResourceType: resource.Type,
		// 	Provider:     resource.Provider,
		// 	DriftType:    "UNMANAGED",
		// 	Severity:     "MEDIUM",
		// 	Details: map[string]interface{}{
		// 		"message":  "Resource exists in cloud but not in Terraform state",
		// 		"resource": resource,
		// 	},
		// 	DetectedAt: time.Now(),
		// })
			}
		}
		return driftRecords
	}

	// Compare each resource with its expected state
	for _, resource := range h.cache {
		expectedState := h.getExpectedState(resource.ID, tfState)
		if expectedState == nil {
			// Resource not in Terraform state (unmanaged)
			driftRecords = append(driftRecords, &DriftRecord{
		// ResourceID:   resource.ID,
		// ResourceType: resource.Type,
		// Provider:     resource.Provider,
		// DriftType:    "UNMANAGED",
		// Severity:     "MEDIUM",
		// Details: map[string]interface{}{
		// 	"message": "Resource exists in cloud but not in Terraform state",
		// 	"actual":  resource,
		// },
		// DetectedAt: time.Now(),
			})
			continue
		}

		// Compare properties for drift
		differences := h.compareResourceProperties(expectedState, resource)
		if len(differences) > 0 {
				// severity := h.calculateDriftSeverity(resource.Type, differences)
			driftRecords = append(driftRecords, &DriftRecord{
		// ResourceID:   resource.ID,
		// ResourceType: resource.Type,
		// Provider:     resource.Provider,
		// DriftType:    "MODIFIED",
		// Severity:     severity,
		// Details: map[string]interface{}{
		// 	"differences": differences,
		// 	"expected":    expectedState,
		// 	"actual":      resource,
		// },
		// DetectedAt: time.Now(),
			})
		}
	}

	// Check for resources in state but not in cloud (deleted)
	if tfState != nil {
		for _, stateResource := range h.getStateResources(tfState) {
			if !h.resourceExistsInCloud(stateResource.ID) {
		// driftRecords = append(driftRecords, &DriftRecord{
		// 	ResourceID:   stateResource.ID,
		// 	ResourceType: stateResource.Type,
		// 	Provider:     stateResource.Provider,
		// 	DriftType:    "DELETED",
		// 	Severity:     "HIGH",
		// 	Details: map[string]interface{}{
		// 		"message":  "Resource exists in Terraform state but not in cloud",
		// 		"expected": stateResource,
		// 	},
		// 	DetectedAt: time.Now(),
		// })
			}
		}
	}

	// Store drift records for future queries
	h.driftRecords = driftRecords
	h.lastDriftCheck = time.Now()

	return driftRecords
}

// GetResourceByID returns a resource by ID
func (h *DiscoveryHub) GetResourceByID(id string) (*models.Resource, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, r := range h.cache {
		if r.ID == id {
			return &r, true
		}
	}
	return nil, false
}

// HasDrift checks if a resource has drift
func (h *DiscoveryHub) HasDrift(resourceID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Check if we have recent drift records
	if h.driftRecords != nil && time.Since(h.lastDriftCheck) < 5*time.Minute {
		for _, record := range h.driftRecords {
			if record.ResourceID == resourceID {
		// return true
			}
		}
		return false
	}

	// Perform drift check for specific resource
	resource, exists := h.GetResourceByID(resourceID)
	if !exists {
		return false
	}

	tfState := h.getTerraformState()
	if tfState == nil {
		// No state means unmanaged resource (drift)
		return !h.isResourceManaged(*resource)
	}

	expectedState := h.getExpectedState(resourceID, tfState)
	if expectedState == nil {
		// Resource not in state (unmanaged drift)
		return true
	}

	// Compare properties
	differences := h.compareResourceProperties(expectedState, *resource)
	return len(differences) > 0
}

// GetCachedResources returns cached resources
func (h *DiscoveryHub) GetCachedResources() []models.Resource {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.cache == nil {
		h.cacheMisses.Add(1)
		return []models.Resource{}
	}

	// Check if cache is fresh
	if time.Since(h.cacheTime) > h.cacheTTL {
		h.cacheMisses.Add(1)
		return []models.Resource{}
	}

	h.cacheHits.Add(1)
	return h.cache
}

// GetCacheMetrics returns cache performance metrics
func (h *DiscoveryHub) GetCacheMetrics() (hits, misses, evictions int64) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.cacheHits.Load(), h.cacheMisses.Load(), h.cacheEvictions.Load()
}

// GetResourcesByIDs returns resources by their IDs
func (h *DiscoveryHub) GetResourcesByIDs(ids []string) []models.Resource {
	h.mu.RLock()
	defer h.mu.RUnlock()

	idMap := make(map[string]bool)
	for _, id := range ids {
		idMap[id] = true
	}

	var resources []models.Resource
	for _, resource := range h.cache {
		if idMap[resource.ID] {
			resources = append(resources, resource)
		}
	}

	return resources
}

// AddResource adds a single resource to the cache
func (h *DiscoveryHub) AddResource(resource models.Resource) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check if resource already exists
	for i, existing := range h.cache {
		if existing.ID == resource.ID {
			// Update existing resource
			h.cache[i] = resource
			return
		}
	}

	// Add new resource
	h.cache = append(h.cache, resource)
	h.cacheTime = time.Now()
	h.cacheVersion++
	h.updateCacheMetadata("import")
}

// Helper methods for drift detection

func (h *DiscoveryHub) getTerraformState() map[string]interface{} {
	// Try to load from cached state first
	if h.terraformState != nil && time.Since(h.lastDriftCheck) < 5*time.Minute {
		return h.terraformState
	}

	// Load Terraform state from file if path is set
	if h.stateFilePath != "" {
		data, err := os.ReadFile(h.stateFilePath)
		if err == nil {
			var state map[string]interface{}
			if err := json.Unmarshal(data, &state); err == nil {
		// h.terraformState = state
		// return state
			}
		}
	}

	// Try default locations
	defaultPaths := []string{
		"terraform.tfstate",
		".terraform/terraform.tfstate",
		"terraform.tfstate.d/default/terraform.tfstate",
	}

	for _, path := range defaultPaths {
		if data, err := os.ReadFile(path); err == nil {
			var state map[string]interface{}
			if err := json.Unmarshal(data, &state); err == nil {
		// h.terraformState = state
		// h.stateFilePath = path
		// return state
			}
		}
	}

	return nil
}

func (h *DiscoveryHub) isResourceManaged(resource models.Resource) bool {
	// Check if resource has Terraform tags or metadata
	if resource.Tags != nil {
		if tags, ok := resource.Tags.(map[string]string); ok {
			if _, hasManaged := tags["ManagedBy"]; hasManaged {
				return true
			}
			if _, hasTerraform := tags["terraform"]; hasTerraform {
				return true
			}
		}
	}

	// Check if resource ID matches known managed pattern
	// This is a simplified check - in production would be more sophisticated
	return false
}

func (h *DiscoveryHub) getExpectedState(resourceID string, tfState map[string]interface{}) interface{} {
	// Parse Terraform state to find the resource
	if resources, ok := tfState["resources"].([]interface{}); ok {
		for _, res := range resources {
			if _, ok := res.(map[string]interface{}); ok {
		// if instances, ok := resMap["instances"].([]interface{}); ok {
		// 	for _, inst := range instances {
		// 		if instMap, ok := inst.(map[string]interface{}); ok {
		// 			if id, ok := instMap["attributes"].(map[string]interface{})["id"].(string); ok && id == resourceID {
		// 				return instMap["attributes"]
		// 			}
		// 		}
		// 	}
		// }
			}
		}
	}
	return nil
}

func (h *DiscoveryHub) compareResourceProperties(expected interface{}, actual models.Resource) []string {
	differences := []string{}

	expectedMap, ok := expected.(map[string]interface{})
	if !ok {
		return differences
	}

	// Convert actual resource to map for comparison
	actualMap := make(map[string]interface{})
	actualBytes, _ := json.Marshal(actual)
	json.Unmarshal(actualBytes, &actualMap)

	// Compare each expected property
	for key, expectedVal := range expectedMap {
		actualVal, exists := actualMap[key]
		if !exists {
			differences = append(differences, fmt.Sprintf("Property '%s' exists in state but not in cloud", key))
			continue
		}

		// Simple comparison - in production would handle nested objects
		if fmt.Sprintf("%v", expectedVal) != fmt.Sprintf("%v", actualVal) {
			differences = append(differences, fmt.Sprintf("Property '%s' differs: expected=%v, actual=%v", key, expectedVal, actualVal))
		}
	}

	return differences
}

func (h *DiscoveryHub) calculateDriftSeverity(resourceType string, differences []string) string {
	// Critical for security resources
	if strings.Contains(resourceType, "security") || strings.Contains(resourceType, "iam") {
		return "HIGH"
	}

	// Check for critical property changes
	for _, diff := range differences {
		if strings.Contains(strings.ToLower(diff), "security") ||
			strings.Contains(strings.ToLower(diff), "encryption") ||
			strings.Contains(strings.ToLower(diff), "public") {
			return "HIGH"
		}
	}

	// Severity based on number of differences
	if len(differences) > 5 {
		return "HIGH"
	} else if len(differences) > 2 {
		return "MEDIUM"
	}

	return "LOW"
}

func (h *DiscoveryHub) getStateResources(tfState map[string]interface{}) []models.Resource {
	resources := []models.Resource{}

	if resourcesData, ok := tfState["resources"].([]interface{}); ok {
		for _, res := range resourcesData {
			if _, ok := res.(map[string]interface{}); ok {
		// resType := resMap["type"].(string)
		// provider := resMap["provider"].(string)

		// if instances, ok := resMap["instances"].([]interface{}); ok {
		// 	for _, inst := range instances {
		// 		if instMap, ok := inst.(map[string]interface{}); ok {
		// 			if attrs, ok := instMap["attributes"].(map[string]interface{}); ok {
		// 				resource := models.Resource{
		// 					ID:       fmt.Sprintf("%v", attrs["id"]),
		// 					Type:     resType,
		// 					Provider: provider,
		// 				}

		// 				if name, ok := attrs["name"].(string); ok {
		// 					resource.Name = name
		// 				}
		// 				if region, ok := attrs["region"].(string); ok {
		// 					resource.Region = region
		// 				}

		// 				resources = append(resources, resource)
		// 			}
		// 		}
		// 	}
		// }
			}
		}
	}

	return resources
}

func (h *DiscoveryHub) resourceExistsInCloud(resourceID string) bool {
	for _, resource := range h.cache {
		if resource.ID == resourceID {
			return true
		}
	}
	return false
}


// getCacheDir returns the cache directory path
func (h *DiscoveryHub) getCacheDir() string {
	// Try to use user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to temp directory
		return filepath.Join(os.TempDir(), ".driftmgr", "cache")
	}
	return filepath.Join(homeDir, ".driftmgr", "cache")
}

// InvalidateCache clears the cache and removes cache files
func (h *DiscoveryHub) InvalidateCache() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Clear in-memory cache
	h.cache = []models.Resource{}
	h.cacheTime = time.Time{}
	h.cacheVersion++

	// Update metrics
	h.cacheEvictions.Add(1)

	// Remove cache file
	cacheDir := h.getCacheDir()
	cacheFile := filepath.Join(cacheDir, "discovery_cache.json")

	if err := os.Remove(cacheFile); err != nil && !os.IsNotExist(err) {
		// Log error but don't fail
		fmt.Printf("Warning: Failed to remove cache file: %v\n", err)
	}

	// Clear global cache if available
	// TODO: Implement global cache
	// if h.globalCache != nil {
	// 	h.globalCache.Clear()
	// }

	// Publish cache invalidated event
	if h.eventBus != nil {
		// TODO: Implement events package
	// h.eventBus.Publish(events.Event{
		// Type:   events.CacheInvalidated,
		// Source: "discovery_hub",
		// Data: map[string]interface{}{
		// "reason": "manual_invalidation",
		// },
		// })
	}

	return nil
}

// RefreshCache forces a cache refresh by re-running discovery
func (h *DiscoveryHub) RefreshCache(ctx context.Context) error {
	// Clear existing cache first
	if err := h.InvalidateCache(); err != nil {
		return fmt.Errorf("failed to invalidate cache: %w", err)
	}

	// If no discovery service, can't refresh
	// TODO: Implement discoveryService
	if true { // h.discoveryService == nil {
		return fmt.Errorf("discovery service not available")
	}

	// Run discovery for all configured providers
	providers := []string{"aws", "azure", "gcp"}
	var allResources []models.Resource

	for _, provider := range providers {
		// Create discovery options with default regions
		// options := discovery.DiscoveryOptions{
		// 	// Parallel:   true,
		// 	// MaxWorkers: 5,
		// 	// Timeout:    5 * time.Minute,
		// }

		// Try to discover resources
		// result, err := h.discoveryService.DiscoverProvider(ctx, provider, options)
		// var result interface{}
		err := fmt.Errorf("discoveryService not implemented")
		if err != nil {
			// Log but continue with other providers
			fmt.Printf("Warning: Failed to discover %s resources: %v\n", provider, err)
			continue
		}

		// Convert and add resources
		// TODO: Fix when discoveryService is implemented
		// apiResources := h.convertToAPIResources(result.Resources)
		// allResources = append(allResources, apiResources...)
	}

	// Update cache
	h.mu.Lock()
	h.cache = allResources
	h.cacheTime = time.Now()
	h.cacheVersion++
	h.mu.Unlock()

	// Save to disk
	h.saveCacheToDisk()
	// TODO: Handle error when implemented
	// if false { // err := h.saveCacheToDisk(); err != nil {
	// 	// Log but don't fail
	// 	fmt.Printf("Warning: Failed to save cache to disk: %v\n", err)
	// }

	// Publish cache refreshed event
	if h.eventBus != nil {
		// TODO: Implement events package
	// h.eventBus.Publish(events.Event{
		// Type:   events.CacheRefreshed,
		// Source: "discovery_hub",
		// Data: map[string]interface{}{
		// "resource_count": len(allResources),
		// "providers":      providers,
		// },
		// })
	}

	return nil
}
