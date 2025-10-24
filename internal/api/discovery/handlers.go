package discovery

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/models"
)

// Handler represents the discovery API handler
type Handler struct {
	engine  *discovery.Engine
	manager *discovery.ResourceManager
}

// NewHandler creates a new discovery API handler
func NewHandler(engine *discovery.Engine, manager *discovery.ResourceManager) *Handler {
	return &Handler{
		engine:  engine,
		manager: manager,
	}
}

// CreateDiscoveryJob creates a new discovery job
func (h *Handler) CreateDiscoveryJob(w http.ResponseWriter, r *http.Request) {
	var req models.DiscoveryJobCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Create discovery job
	job := &models.DiscoveryJob{
		ID:            generateJobID(),
		Provider:      req.Provider,
		AccountID:     req.AccountID,
		Region:        req.Region,
		ResourceTypes: req.ResourceTypes,
		Status:        models.JobStatusPending,
		Configuration: req.Configuration,
		CreatedBy:     getCurrentUser(r),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Start discovery job
	ctx := r.Context()
	results, err := h.engine.DiscoverResources(ctx, job)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to start discovery job", err)
		return
	}

	job.Results = *results
	job.SetStatus(models.JobStatusCompleted)

	WriteJSONResponse(w, http.StatusCreated, job)
}

// ListDiscoveryJobs lists discovery jobs
func (h *Handler) ListDiscoveryJobs(w http.ResponseWriter, r *http.Request) {
	var req models.DiscoveryJobListRequest
	if err := parseListRequest(r, &req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}

	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Get jobs from scheduler
	jobs := h.engine.GetScheduledJobs()

	// Apply filters
	filteredJobs := h.filterJobs(jobs, &req)

	// Apply pagination
	total := len(filteredJobs)
	start := req.Offset
	end := start + req.Limit

	if start >= total {
		WriteJSONResponse(w, http.StatusOK, &models.DiscoveryJobListResponse{
			Jobs:   []models.DiscoveryJob{},
			Total:  total,
			Limit:  req.Limit,
			Offset: req.Offset,
		})
		return
	}

	if end > total {
		end = total
	}

	response := &models.DiscoveryJobListResponse{
		Jobs:   filteredJobs[start:end],
		Total:  total,
		Limit:  req.Limit,
		Offset: req.Offset,
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// GetDiscoveryJob retrieves a specific discovery job
func (h *Handler) GetDiscoveryJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	job, err := h.engine.GetScheduledJob(jobID)
	if err != nil {
		WriteErrorResponse(w, http.StatusNotFound, "Discovery job not found", err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, job)
}

// UpdateDiscoveryJob updates a discovery job
func (h *Handler) UpdateDiscoveryJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	var req models.DisedyJobUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Get existing job
	job, err := h.engine.GetScheduledJob(jobID)
	if err != nil {
		WriteErrorResponse(w, http.StatusNotFound, "Discovery job not found", err)
		return
	}

	// Update job fields
	if req.ResourceTypes != nil {
		job.ResourceTypes = *req.ResourceTypes
	}
	if req.Configuration != nil {
		job.Configuration = *req.Configuration
	}
	job.UpdatedAt = time.Now()

	WriteJSONResponse(w, http.StatusOK, job)
}

// DeleteDiscoveryJob deletes a discovery job
func (h *Handler) DeleteDiscoveryJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	if err := h.engine.CancelScheduledDiscovery(jobID); err != nil {
		WriteErrorResponse(w, http.StatusNotFound, "Discovery job not found", err)
		return
	}

	WriteJSONResponse(w, http.StatusNoContent, nil)
}

// StartDiscoveryJob starts a discovery job
func (h *Handler) StartDiscoveryJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	job, err := h.engine.GetScheduledJob(jobID)
	if err != nil {
		WriteErrorResponse(w, http.StatusNotFound, "Discovery job not found", err)
		return
	}

	if job.IsRunning() {
		WriteErrorResponse(w, http.StatusConflict, "Discovery job is already running", nil)
		return
	}

	// Start the job
	ctx := r.Context()
	results, err := h.engine.DiscoverResources(ctx, job)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to start discovery job", err)
		return
	}

	job.Results = *results
	job.SetStatus(models.JobStatusCompleted)

	WriteJSONResponse(w, http.StatusOK, job)
}

// StopDiscoveryJob stops a discovery job
func (h *Handler) StopDiscoveryJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	if err := h.engine.CancelScheduledDiscovery(jobID); err != nil {
		WriteErrorResponse(w, http.StatusNotFound, "Discovery job not found", err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, map[string]string{"message": "Discovery job stopped"})
}

// GetDiscoveryResults retrieves results for a discovery job
func (h *Handler) GetDiscoveryResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	job, err := h.engine.GetScheduledJob(jobID)
	if err != nil {
		WriteErrorResponse(w, http.StatusNotFound, "Discovery job not found", err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, job.Results)
}

// ListResources lists discovered resources
func (h *Handler) ListResources(w http.ResponseWriter, r *http.Request) {
	var req models.ResourceListRequest
	if err := parseListRequest(r, &req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}

	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	ctx := r.Context()
	response, err := h.manager.ListResources(ctx, &req)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to list resources", err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// GetResource retrieves a specific resource
func (h *Handler) GetResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["id"]

	ctx := r.Context()
	resource, err := h.manager.GetResource(ctx, resourceID)
	if err != nil {
		WriteErrorResponse(w, http.StatusNotFound, "Resource not found", err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, resource)
}

// SearchResources searches for resources
func (h *Handler) SearchResources(w http.ResponseWriter, r *http.Request) {
	var req models.ResourceSearchRequest
	if err := parseSearchRequest(r, &req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}

	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	ctx := r.Context()
	response, err := h.manager.SearchResources(ctx, &req)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to search resources", err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// GetResourceRelationships retrieves relationships for a resource
func (h *Handler) GetResourceRelationships(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["id"]

	ctx := r.Context()
	relationships, err := h.manager.GetResourceRelationships(ctx, resourceID)
	if err != nil {
		WriteErrorResponse(w, http.StatusNotFound, "Resource not found", err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, relationships)
}

// UpdateResourceTags updates tags for a resource
func (h *Handler) UpdateResourceTags(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["id"]

	var req models.ResourceTagUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	ctx := r.Context()
	if err := h.manager.UpdateResourceTags(ctx, resourceID, req.Tags); err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to update resource tags", err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, map[string]string{"message": "Resource tags updated"})
}

// GetResourceCompliance retrieves compliance status for a resource
func (h *Handler) GetResourceCompliance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["id"]

	ctx := r.Context()
	compliance, err := h.manager.GetResourceCompliance(ctx, resourceID)
	if err != nil {
		WriteErrorResponse(w, http.StatusNotFound, "Resource not found", err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, compliance)
}

// GetResourceCost retrieves cost information for a resource
func (h *Handler) GetResourceCost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["id"]

	ctx := r.Context()
	cost, err := h.manager.GetResourceCost(ctx, resourceID)
	if err != nil {
		WriteErrorResponse(w, http.StatusNotFound, "Resource not found", err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, cost)
}

// GetDiscoveryStatistics retrieves discovery statistics
func (h *Handler) GetDiscoveryStatistics(w http.ResponseWriter, r *http.Request) {
	stats := h.engine.GetDiscoveryStatistics()
	WriteJSONResponse(w, http.StatusOK, stats)
}

// GetResourceStatistics retrieves resource statistics
func (h *Handler) GetResourceStatistics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stats, err := h.manager.GetResourceStatistics(ctx)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get resource statistics", err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, stats)
}

// Helper functions

func generateJobID() string {
	return "job-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func getCurrentUser(r *http.Request) string {
	// In a real implementation, this would extract the user from the request context
	return "system"
}

func parseListRequest(r *http.Request, req interface{}) error {
	// Parse query parameters
	query := r.URL.Query()

	// This is a simplified implementation
	// In production, you would use a proper query parameter parser
	return nil
}

func parseSearchRequest(r *http.Request, req interface{}) error {
	// Parse query parameters for search
	query := r.URL.Query()

	// This is a simplified implementation
	// In production, you would use a proper query parameter parser
	return nil
}

func (h *Handler) filterJobs(jobs []*models.DiscoveryJob, req *models.DiscoveryJobListRequest) []models.DiscoveryJob {
	var filtered []models.DiscoveryJob

	for _, job := range jobs {
		// Apply filters
		if req.Provider != nil && job.Provider != *req.Provider {
			continue
		}
		if req.AccountID != nil && job.AccountID != *req.AccountID {
			continue
		}
		if req.Region != nil && job.Region != *req.Region {
			continue
		}
		if req.Status != nil && job.Status != *req.Status {
			continue
		}

		filtered = append(filtered, *job)
	}

	return filtered
}
