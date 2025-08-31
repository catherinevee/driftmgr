package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/cache"
	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/events"
	"github.com/catherinevee/driftmgr/internal/jobs"
	"github.com/catherinevee/driftmgr/internal/providers/cloud"
	"github.com/google/uuid"
)

// DiscoveryService provides a unified interface for resource discovery
// Used by both CLI and API to ensure consistent behavior
type DiscoveryService struct {
	discoveryEngine *discovery.Engine
	cache           cache.Cache
	eventBus        *events.EventBus
	jobQueue        *jobs.Queue
	providers       map[string]cloud.Provider
	mu              sync.RWMutex
}

// NewDiscoveryService creates a new discovery service instance
func NewDiscoveryService(
	discoveryEngine *discovery.Engine,
	cache cache.Cache,
	eventBus *events.EventBus,
	jobQueue *jobs.Queue,
) *DiscoveryService {
	return &DiscoveryService{
		discoveryEngine: discoveryEngine,
		cache:           cache,
		eventBus:        eventBus,
		jobQueue:        jobQueue,
		providers:       make(map[string]cloud.Provider),
	}
}

// DiscoveryRequest represents a request to discover resources
type DiscoveryRequest struct {
	Provider       string                 `json:"provider"`
	Regions        []string               `json:"regions"`
	ResourceTypes  []string               `json:"resource_types,omitempty"`
	AutoRemediate  bool                   `json:"auto_remediate"`
	Async          bool                   `json:"async"`
	Incremental    bool                   `json:"incremental"`
	LastSyncTime   *time.Time             `json:"last_sync_time,omitempty"`
	Options        map[string]interface{} `json:"options,omitempty"`
}

// DiscoveryResponse represents the response from a discovery operation
type DiscoveryResponse struct {
	JobID     string                 `json:"job_id,omitempty"`
	Status    string                 `json:"status"`
	Progress  int                    `json:"progress"`
	Message   string                 `json:"message"`
	Resources []cloud.Resource       `json:"resources,omitempty"`
	Errors    []string               `json:"errors,omitempty"`
	StartedAt time.Time              `json:"started_at"`
	EndedAt   *time.Time             `json:"ended_at,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// StartDiscovery initiates a discovery operation
func (s *DiscoveryService) StartDiscovery(ctx context.Context, req DiscoveryRequest) (*DiscoveryResponse, error) {
	// Validate request
	if err := s.validateDiscoveryRequest(req); err != nil {
		return nil, fmt.Errorf("invalid discovery request: %w", err)
	}

	// Generate job ID
	jobID := uuid.New().String()

	// Emit discovery started event
	s.eventBus.Publish(events.Event{
		Type:      events.DiscoveryStarted,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"job_id":   jobID,
			"provider": req.Provider,
			"regions":  req.Regions,
		},
	})

	// If async, create a job and return immediately
	if req.Async {
		job := &jobs.Job{
			ID:        jobID,
			Type:      jobs.DiscoveryJob,
			Status:    jobs.StatusPending,
			CreatedAt: time.Now(),
			Data:      req,
		}

		if err := s.jobQueue.Enqueue(job); err != nil {
			return nil, fmt.Errorf("failed to enqueue discovery job: %w", err)
		}

		// Start processing in background
		go s.processDiscoveryJob(context.Background(), job)

		return &DiscoveryResponse{
			JobID:     jobID,
			Status:    "running",
			Progress:  0,
			Message:   "Discovery job started",
			StartedAt: time.Now(),
		}, nil
	}

	// Synchronous discovery
	return s.executeDiscovery(ctx, jobID, req)
}

// executeDiscovery performs the actual discovery operation
func (s *DiscoveryService) executeDiscovery(ctx context.Context, jobID string, req DiscoveryRequest) (*DiscoveryResponse, error) {
	startTime := time.Now()
	response := &DiscoveryResponse{
		JobID:     jobID,
		Status:    "running",
		Progress:  0,
		Message:   "Starting discovery",
		StartedAt: startTime,
		Resources: []cloud.Resource{},
		Errors:    []string{},
		Metadata:  make(map[string]interface{}),
	}

	// Check for incremental discovery
	if req.Incremental {
		// Try to get cached baseline for comparison
		cacheKey := fmt.Sprintf("discovery:baseline:%s", req.Provider)
		if cachedBaseline, found := s.cache.Get(cacheKey); found {
			if baseline, ok := cachedBaseline.([]cloud.Resource); ok {
				response.Metadata["baseline_count"] = len(baseline)
				response.Metadata["incremental"] = true
				
				// Store baseline for later comparison
				defer func() {
					if response.Status == "completed" {
						// Perform incremental analysis
						added, removed, modified := s.compareResources(baseline, response.Resources)
						response.Metadata["added"] = len(added)
						response.Metadata["removed"] = len(removed)
						response.Metadata["modified"] = len(modified)
						
						// Update baseline cache
						s.cache.Set(cacheKey, response.Resources, 1*time.Hour)
					}
				}()
			}
		}
	}

	// Get provider
	_, err := s.getProvider(req.Provider)
	if err != nil {
		response.Status = "failed"
		response.Errors = append(response.Errors, err.Error())
		endTime := time.Now()
		response.EndedAt = &endTime
		return response, err
	}

	// Discover resources for each region
	totalRegions := len(req.Regions)
	if totalRegions == 0 {
		req.Regions = []string{"us-east-1"} // Default region
		totalRegions = 1
	}

	for i, region := range req.Regions {
		// Update progress
		progress := (i * 100) / totalRegions
		response.Progress = progress
		response.Message = fmt.Sprintf("Discovering resources in %s", region)

		// Emit progress event
		s.eventBus.Publish(events.Event{
			Type:      events.DiscoveryProgress,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"job_id":   jobID,
				"progress": progress,
				"message":  response.Message,
			},
		})

		// Discover resources in region
		resources, err := s.discoveryEngine.DiscoverResourcesInRegion(ctx, req.Provider, region, req.ResourceTypes)
		if err != nil {
			response.Errors = append(response.Errors, fmt.Sprintf("Error in region %s: %v", region, err))
			continue
		}

		response.Resources = append(response.Resources, resources...)

		// Cache discovered resources
		for _, resource := range resources {
			cacheKey := fmt.Sprintf("resource:%s:%s:%s", req.Provider, region, resource.ID)
			s.cache.Set(cacheKey, resource, 15*time.Minute)
		}
	}

	// Final status
	endTime := time.Now()
	response.EndedAt = &endTime
	response.Progress = 100

	if len(response.Errors) > 0 {
		response.Status = "completed_with_errors"
		response.Message = fmt.Sprintf("Discovery completed with %d errors", len(response.Errors))
	} else {
		response.Status = "completed"
		response.Message = fmt.Sprintf("Successfully discovered %d resources", len(response.Resources))
	}

	// Cache the complete response
	s.cache.Set(fmt.Sprintf("discovery:job:%s", jobID), response, 1*time.Hour)

	// Emit discovery completed event
	s.eventBus.Publish(events.Event{
		Type:      events.DiscoveryCompleted,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"job_id":         jobID,
			"resource_count": len(response.Resources),
			"errors":         response.Errors,
		},
	})

	return response, nil
}

// processDiscoveryJob processes an async discovery job
func (s *DiscoveryService) processDiscoveryJob(ctx context.Context, job *jobs.Job) {
	req, ok := job.Data.(DiscoveryRequest)
	if !ok {
		job.Status = jobs.StatusFailed
		job.Error = fmt.Errorf("invalid job data")
		s.jobQueue.UpdateJob(job)
		return
	}

	job.Status = jobs.StatusRunning
	job.StartedAt = timePtr(time.Now())
	s.jobQueue.UpdateJob(job)

	response, err := s.executeDiscovery(ctx, job.ID, req)
	
	if err != nil {
		job.Status = jobs.StatusFailed
		job.Error = err
	} else {
		job.Status = jobs.StatusCompleted
		job.Result = response
	}
	
	job.CompletedAt = timePtr(time.Now())
	s.jobQueue.UpdateJob(job)
}

// GetDiscoveryStatus returns the status of a discovery job
func (s *DiscoveryService) GetDiscoveryStatus(ctx context.Context, jobID string) (*DiscoveryResponse, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("discovery:job:%s", jobID)
	if cached, found := s.cache.Get(cacheKey); found {
		if response, ok := cached.(*DiscoveryResponse); ok {
			return response, nil
		}
	}

	// Check job queue
	job, err := s.jobQueue.GetJob(jobID)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	response := &DiscoveryResponse{
		JobID:     job.ID,
		Status:    string(job.Status),
		StartedAt: job.CreatedAt,
	}

	if job.StartedAt != nil {
		response.StartedAt = *job.StartedAt
	}

	if job.CompletedAt != nil {
		response.EndedAt = job.CompletedAt
	}

	if job.Result != nil {
		if res, ok := job.Result.(*DiscoveryResponse); ok {
			return res, nil
		}
	}

	if job.Status == jobs.StatusRunning {
		response.Message = "Discovery in progress"
		// Try to get progress from cache
		if progress, found := s.cache.Get(fmt.Sprintf("discovery:progress:%s", jobID)); found {
			if p, ok := progress.(int); ok {
				response.Progress = p
			}
		}
	}

	return response, nil
}

// GetDiscoveryResults returns the results of a completed discovery job
func (s *DiscoveryService) GetDiscoveryResults(ctx context.Context, jobID string) (*DiscoveryResponse, error) {
	response, err := s.GetDiscoveryStatus(ctx, jobID)
	if err != nil {
		return nil, err
	}

	if response.Status != "completed" && response.Status != "completed_with_errors" {
		return nil, fmt.Errorf("job not completed, current status: %s", response.Status)
	}

	return response, nil
}

// GetCachedResources returns cached resources from previous discoveries
func (s *DiscoveryService) GetCachedResources(ctx context.Context, provider string, region string) ([]cloud.Resource, error) {
	pattern := fmt.Sprintf("resource:%s:%s:*", provider, region)
	keys := s.cache.Keys(pattern)
	
	resources := []cloud.Resource{}
	for _, key := range keys {
		if cached, found := s.cache.Get(key); found {
			if resource, ok := cached.(cloud.Resource); ok {
				resources = append(resources, resource)
			}
		}
	}

	return resources, nil
}

// compareResources compares two sets of resources for incremental discovery
func (s *DiscoveryService) compareResources(baseline, current []cloud.Resource) (added, removed, modified []cloud.Resource) {
	// Create maps for efficient lookup
	baselineMap := make(map[string]cloud.Resource)
	currentMap := make(map[string]cloud.Resource)
	
	for _, r := range baseline {
		baselineMap[r.ID] = r
	}
	
	for _, r := range current {
		currentMap[r.ID] = r
	}
	
	// Find added and modified resources
	for id, currentResource := range currentMap {
		if baselineResource, exists := baselineMap[id]; exists {
			// Check if modified by comparing key fields
			if s.isResourceModified(baselineResource, currentResource) {
				modified = append(modified, currentResource)
			}
		} else {
			// Resource not in baseline, so it's new
			added = append(added, currentResource)
		}
	}
	
	// Find removed resources
	for id, baselineResource := range baselineMap {
		if _, exists := currentMap[id]; !exists {
			removed = append(removed, baselineResource)
		}
	}
	
	return added, removed, modified
}

// isResourceModified checks if a resource has been modified
func (s *DiscoveryService) isResourceModified(baseline, current cloud.Resource) bool {
	// Compare key fields that indicate modification
	if baseline.Status != current.Status {
		return true
	}
	
	// Compare tags if both have them
	if baseline.Tags != nil && current.Tags != nil {
		// Tags are already map[string]string, no need for type assertion
		if len(baseline.Tags) != len(current.Tags) {
			return true
		}
		for k, v := range baseline.Tags {
			if current.Tags[k] != v {
				return true
			}
		}
	}
	
	// Compare modification times if available
	if baseline.ModifiedAt != "" && current.ModifiedAt != "" {
		return current.ModifiedAt > baseline.ModifiedAt
	}
	
	// Compare properties if they exist
	if baseline.Properties != nil && current.Properties != nil {
		// Simple comparison - could be enhanced
		return fmt.Sprintf("%v", baseline.Properties) != fmt.Sprintf("%v", current.Properties)
	}
	
	return false
}

// validateDiscoveryRequest validates a discovery request
func (s *DiscoveryService) validateDiscoveryRequest(req DiscoveryRequest) error {
	if req.Provider == "" {
		return fmt.Errorf("provider is required")
	}

	validProviders := []string{"aws", "azure", "gcp", "digitalocean"}
	valid := false
	for _, p := range validProviders {
		if req.Provider == p {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid provider: %s", req.Provider)
	}

	return nil
}

// getProvider returns a provider instance
func (s *DiscoveryService) getProvider(providerName string) (cloud.Provider, error) {
	s.mu.RLock()
	provider, exists := s.providers[providerName]
	s.mu.RUnlock()

	if exists {
		return provider, nil
	}

	// Create provider if not exists
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock
	if provider, exists := s.providers[providerName]; exists {
		return provider, nil
	}

	// Create new provider instance
	newProvider := cloud.Provider{
		Name: providerName,
		Type: providerName,
	}

	s.providers[providerName] = newProvider
	return newProvider, nil
}

// ClearCache clears the discovery cache
func (s *DiscoveryService) ClearCache(ctx context.Context) error {
	return s.cache.Clear()
}

// timePtr returns a pointer to a time.Time
func timePtr(t time.Time) *time.Time {
	return &t
}