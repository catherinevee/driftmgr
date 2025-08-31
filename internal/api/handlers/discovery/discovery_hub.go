package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	apimodels "github.com/catherinevee/driftmgr/internal/api/models"
	"github.com/catherinevee/driftmgr/internal/cache"
	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/catherinevee/driftmgr/internal/services"
	"github.com/google/uuid"
)

// JobStatus represents the status of a discovery job
type JobStatus struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"`
	Provider  string                 `json:"provider"`
	Regions   []string               `json:"regions"`
	StartTime time.Time              `json:"start_time"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
	Progress  int                    `json:"progress"`
	Total     int                    `json:"total"`
	Message   string                 `json:"message"`
	Error     string                 `json:"error,omitempty"`
	Resources int                    `json:"resources_found"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// DiscoveryHub manages discovery jobs
type DiscoveryHub struct {
	mu               sync.RWMutex
	jobs             map[string]*JobStatus
	results          map[string][]apimodels.Resource
	cache            []apimodels.Resource
	cacheTime        time.Time
	cacheTTL         time.Duration
	cacheVersion     int
	cacheMetadata    CacheMetadata
	discoveryService *discovery.Service
	serviceManager   *services.Manager
	globalCache      *cache.GlobalCache
	wsBroadcast      func(messageType string, data map[string]interface{})
}

// CacheMetadata tracks cache freshness and versioning
type CacheMetadata struct {
	Version       int       `json:"version"`
	LastUpdated   time.Time `json:"last_updated"`
	ResourceCount int       `json:"resource_count"`
	Sources       []string  `json:"sources"`
	Freshness     string    `json:"freshness"` // "fresh", "recent", "stale"
	TTL           int       `json:"ttl_seconds"`
}

// NewDiscoveryHub creates a new discovery hub
func NewDiscoveryHub(discoveryService *discovery.Service) *DiscoveryHub {
	hub := &DiscoveryHub{
		jobs:             make(map[string]*JobStatus),
		results:          make(map[string][]apimodels.Resource),
		cache:            []apimodels.Resource{},
		cacheTTL:         5 * time.Minute,
		discoveryService: discoveryService,
		serviceManager:   nil, // Will be initialized if needed
		globalCache:      cache.GetGlobalCache(),
		wsBroadcast:      nil, // Will be set later
	}
	
	// Load cache from disk if it exists
	hub.loadCacheFromDisk()
	
	return hub
}

// SetWebSocketBroadcast sets the WebSocket broadcast function
func (h *DiscoveryHub) SetWebSocketBroadcast(broadcast func(string, map[string]interface{})) {
	h.wsBroadcast = broadcast
}

// StartDiscovery starts a new discovery job
func (h *DiscoveryHub) StartDiscovery(req DiscoveryRequest) string {
	jobID := uuid.New().String()

	// Check global cache first
	cacheKey := fmt.Sprintf("discovery:%s:%v", req.Provider, req.Regions)
	if cachedData, found, age := h.globalCache.GetWithAge(cacheKey); found {
		if resources, ok := cachedData.([]models.Resource); ok {
			// Return cached results immediately
			job := &JobStatus{
				ID:        jobID,
				Status:    "completed",
				Provider:  req.Provider,
				Regions:   req.Regions,
				StartTime: time.Now().Add(-age),
				EndTime:   &[]time.Time{time.Now()}[0],
				Progress:  100,
				Total:     len(req.Regions),
				Message:   fmt.Sprintf("Using cached data (age: %v)", age.Round(time.Second)),
				Resources: len(resources),
				Metadata: map[string]interface{}{
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

	job := &JobStatus{
		ID:        jobID,
		Status:    "running",
		Provider:  req.Provider,
		Regions:   req.Regions,
		StartTime: time.Now(),
		Progress:  0,
		Total:     len(req.Regions),
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var allResources []apimodels.Resource

	// Send progress update
	h.sendTerminalOutput(jobID, fmt.Sprintf("Starting discovery for provider: %s", req.Provider), "info")

	// Always try to use real discovery service
	if h.discoveryService != nil {
		// Create discovery options
		options := discovery.DiscoveryOptions{
			Regions:       req.Regions,
			// ResourceTypes: req.ResourceTypes, // TODO: Add this field to DiscoveryRequest
			Parallel:      true,
			MaxWorkers:    5,
			Timeout:       5 * time.Minute,
		}

		h.sendTerminalOutput(jobID, fmt.Sprintf("Scanning regions: %v", req.Regions), "info")

		// Perform real discovery
		result, err := h.discoveryService.DiscoverProvider(ctx, req.Provider, options)
		if err != nil {
			h.sendTerminalOutput(jobID, fmt.Sprintf("Discovery failed: %v", err), "error")
			h.updateJobError(jobID, fmt.Sprintf("Discovery failed: %v", err))
			return
		}

		// Convert models.Resource to API Resource
		for _, r := range result.Resources {
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

			allResources = append(allResources, apimodels.Resource{
				ID:           r.ID,
				Name:         r.Name,
				Type:         r.Type,
				Provider:     r.Provider,
				Region:       r.Region,
				Status:       status,
				CreatedAt:    r.CreatedAt,
				ModifiedAt:   r.Updated,
				Tags:         tags,
				Properties:   r.Properties,
				Dependencies: r.Dependencies,
				Managed:      true, // Default to managed
			})
		}

		// Send progress update with resource count
		h.sendTerminalOutput(jobID, fmt.Sprintf("Found %d resources", len(allResources)), "success")
		h.sendDiscoveryProgress(jobID, 90, fmt.Sprintf("Processing %d resources...", len(allResources)))
	}

	// Store results
	h.mu.Lock()
	h.results[jobID] = allResources
	
	// Track previous cache size for logging
	prevCacheSize := len(h.cache)
	
	// Use map for efficient deduplication when merging
	cacheMap := make(map[string]apimodels.Resource)
	
	// First, add all existing cached resources to the map
	for _, resource := range h.cache {
		cacheMap[resource.ID] = resource
	}
	
	// Then, add/update with new resources (this automatically handles duplicates)
	for _, resource := range allResources {
		cacheMap[resource.ID] = resource
	}
	
	// Convert map back to slice
	h.cache = make([]apimodels.Resource, 0, len(cacheMap))
	for _, resource := range cacheMap {
		h.cache = append(h.cache, resource)
	}
	h.cacheTime = time.Now()
	
	// Store in global cache for CLI access
	cacheKey := fmt.Sprintf("discovery:%s:web", req.Provider)
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
	h.globalCache.Set(cacheKey, cacheResources, 5*time.Minute)
	
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
func (h *DiscoveryHub) GetJobResults(jobID string) []apimodels.Resource {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if results, ok := h.results[jobID]; ok {
		return results
	}
	return []apimodels.Resource{}
}

// GetCachedResults returns cached discovery results
func (h *DiscoveryHub) GetCachedResults() []apimodels.Resource {
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
func (h *DiscoveryHub) PrePopulateCache(resources []apimodels.Resource) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	// Deduplicate resources before adding to cache
	uniqueResources := make(map[string]apimodels.Resource)
	for _, resource := range resources {
		// Use the resource ID as the key for deduplication
		// If a resource with the same ID already exists, it will be overwritten
		// This ensures only unique resources are stored
		uniqueResources[resource.ID] = resource
	}
	
	// Convert map back to slice
	h.cache = make([]apimodels.Resource, 0, len(uniqueResources))
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
		Resources []apimodels.Resource `json:"resources"`
		Timestamp time.Time  `json:"timestamp"`
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
		Resources []apimodels.Resource `json:"resources"`
		Timestamp time.Time  `json:"timestamp"`
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
	uniqueResources := make(map[string]apimodels.Resource)
	for _, resource := range cacheData.Resources {
		uniqueResources[resource.ID] = resource
	}
	
	// Convert map to slice
	dedupedResources := make([]apimodels.Resource, 0, len(uniqueResources))
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
		job.EndTime = &now
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
		job.Resources = resourceCount
		now := time.Now()
		job.EndTime = &now

		// Update the cache with discovered resources using efficient deduplication
		if results, ok := h.results[jobID]; ok && len(results) > 0 {
			// Use map for efficient deduplication
			cacheMap := make(map[string]apimodels.Resource)
			
			// Add existing cache to map
			for _, resource := range h.cache {
				cacheMap[resource.ID] = resource
			}
			
			// Merge new resources (automatically handles duplicates)
			for _, resource := range results {
				cacheMap[resource.ID] = resource
			}
			
			// Convert back to slice
			h.cache = make([]apimodels.Resource, 0, len(cacheMap))
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
	h.cache = []apimodels.Resource{}
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
		if job.EndTime != nil && now.Sub(*job.EndTime) > maxAge {
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
func (h *DiscoveryHub) convertToAPIResources(resources []models.Resource) []apimodels.Resource {
	apiResources := make([]apimodels.Resource, 0, len(resources))
	for _, r := range resources {
		// Convert tags to map[string]string if needed
		var tags map[string]string
		if t, ok := r.Tags.(map[string]string); ok {
			tags = t
		} else {
			tags = make(map[string]string)
		}

		apiResources = append(apiResources, apimodels.Resource{
			ID:         r.ID,
			Name:       r.Name,
			Type:       r.Type,
			Provider:   r.Provider,
			Region:     r.Region,
			Status:     r.Status,
			Tags:       tags,
			ModifiedAt: r.LastModified,
			Properties: r.Properties,
		})
	}
	return apiResources
}

// GetAllResources returns all resources from the cache
func (h *DiscoveryHub) GetAllResources() []apimodels.Resource {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	result := make([]apimodels.Resource, len(h.cache))
	copy(result, h.cache)
	return result
}

// GetDriftRecords returns drift records for resources
func (h *DiscoveryHub) GetDriftRecords() []*DriftRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	// For now, return empty slice - will implement drift detection later
	return []*DriftRecord{}
}

// GetResourceByID returns a resource by ID
func (h *DiscoveryHub) GetResourceByID(id string) (*apimodels.Resource, bool) {
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
	// For now, return false - will implement drift detection later
	return false
}

// GetCachedResources returns cached resources
func (h *DiscoveryHub) GetCachedResources() []apimodels.Resource {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	if h.cache == nil {
		return []apimodels.Resource{}
	}
	
	return h.cache
}

// GetResourcesByIDs returns resources by their IDs
func (h *DiscoveryHub) GetResourcesByIDs(ids []string) []apimodels.Resource {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	idMap := make(map[string]bool)
	for _, id := range ids {
		idMap[id] = true
	}
	
	var resources []apimodels.Resource
	for _, resource := range h.cache {
		if idMap[resource.ID] {
			resources = append(resources, resource)
		}
	}
	
	return resources
}

// AddResource adds a single resource to the cache
func (h *DiscoveryHub) AddResource(resource apimodels.Resource) {
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
