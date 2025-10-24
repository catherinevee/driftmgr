package remediation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/catherinevee/driftmgr/internal/business/remediation"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/gorilla/mux"
)

// Handler handles HTTP requests for remediation operations
type Handler struct {
	service *remediation.Service
}

// NewHandler creates a new remediation handler
func NewHandler(service *remediation.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// CreateJob handles POST /api/v1/remediation/jobs
func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req models.RemediationJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Create job
	job, err := h.service.CreateJob(r.Context(), &req)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to create job", err.Error())
		return
	}

	// Convert to response
	response := &models.RemediationJobResponse{
		ID:        job.ID,
		Status:    job.Status,
		Progress:  job.Progress,
		CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt,
	}

	WriteJSONResponse(w, http.StatusCreated, response)
}

// GetJob handles GET /api/v1/remediation/jobs/{id}
func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	if jobID == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid job ID", "Job ID is required")
		return
	}

	// Get job
	job, err := h.service.GetJob(r.Context(), jobID)
	if err != nil {
		if err == models.ErrRemediationJobNotFound {
			WriteErrorResponse(w, http.StatusNotFound, "Job not found", err.Error())
			return
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get job", err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, job)
}

// ListJobs handles GET /api/v1/remediation/jobs
func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	req := &models.RemediationJobListRequest{
		Limit:     50,
		Offset:    0,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	// Parse status filter
	if status := r.URL.Query().Get("status"); status != "" {
		jobStatus := models.JobStatus(status)
		req.Status = &jobStatus
	}

	// Parse priority filter
	if priority := r.URL.Query().Get("priority"); priority != "" {
		jobPriority := models.JobPriority(priority)
		req.Priority = &jobPriority
	}

	// Parse created_by filter
	if createdBy := r.URL.Query().Get("created_by"); createdBy != "" {
		req.CreatedBy = &createdBy
	}

	// Parse strategy_type filter
	if strategyType := r.URL.Query().Get("strategy_type"); strategyType != "" {
		strategy := models.StrategyType(strategyType)
		req.StrategyType = &strategy
	}

	// Parse date filters
	if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
		if startDate, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			req.StartDate = &startDate
		}
	}

	if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
		if endDate, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			req.EndDate = &endDate
		}
	}

	// Parse pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			req.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			req.Offset = offset
		}
	}

	// Parse sorting
	if sortBy := r.URL.Query().Get("sort_by"); sortBy != "" {
		req.SortBy = sortBy
	}

	if sortOrder := r.URL.Query().Get("sort_order"); sortOrder != "" {
		req.SortOrder = sortOrder
	}

	// Validate request
	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// List jobs
	response, err := h.service.ListJobs(r.Context(), req)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to list jobs", err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// CancelJob handles POST /api/v1/remediation/jobs/{id}/cancel
func (h *Handler) CancelJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	if jobID == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid job ID", "Job ID is required")
		return
	}

	// Parse cancel request
	var cancelReq models.JobCancelRequest
	if err := json.NewDecoder(r.Body).Decode(&cancelReq); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Validate request
	if err := cancelReq.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Cancel job
	err := h.service.CancelJob(r.Context(), jobID, cancelReq.Reason)
	if err != nil {
		if err == models.ErrRemediationJobNotFound {
			WriteErrorResponse(w, http.StatusNotFound, "Job not found", err.Error())
			return
		}
		if err == models.ErrJobCannotBeCancelled {
			WriteErrorResponse(w, http.StatusBadRequest, "Job cannot be cancelled", err.Error())
			return
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to cancel job", err.Error())
		return
	}

	// Get updated job
	job, err := h.service.GetJob(r.Context(), jobID)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get updated job", err.Error())
		return
	}

	// Create response
	response := &models.JobCancelResponse{
		JobID:       job.ID,
		Status:      job.Status,
		CancelledAt: *job.CompletedAt,
		Reason:      cancelReq.Reason,
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// GetJobProgress handles GET /api/v1/remediation/progress/{id}
func (h *Handler) GetJobProgress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	if jobID == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid job ID", "Job ID is required")
		return
	}

	// Get job
	job, err := h.service.GetJob(r.Context(), jobID)
	if err != nil {
		if err == models.ErrRemediationJobNotFound {
			WriteErrorResponse(w, http.StatusNotFound, "Job not found", err.Error())
			return
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get job", err.Error())
		return
	}

	// Return progress
	WriteJSONResponse(w, http.StatusOK, job.Progress)
}

// ListStrategies handles GET /api/v1/remediation/strategies
func (h *Handler) ListStrategies(w http.ResponseWriter, r *http.Request) {
	// List strategies
	strategies, err := h.service.ListStrategies(r.Context())
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to list strategies", err.Error())
		return
	}

	// Convert to response
	response := &models.RemediationStrategyListResponse{
		Strategies: make([]models.RemediationStrategyResponse, len(strategies)),
		Total:      len(strategies),
	}

	for i, strategy := range strategies {
		response.Strategies[i] = models.RemediationStrategyResponse{
			ID:          strategy.ID,
			Type:        strategy.Type,
			Name:        strategy.Name,
			Description: strategy.Description,
			Parameters:  strategy.Parameters,
			Timeout:     strategy.Timeout,
			RetryCount:  strategy.RetryCount,
			IsCustom:    strategy.IsCustom,
			CreatedBy:   strategy.CreatedBy,
			CreatedAt:   strategy.CreatedAt,
		}
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// CreateStrategy handles POST /api/v1/remediation/strategies
func (h *Handler) CreateStrategy(w http.ResponseWriter, r *http.Request) {
	var req models.RemediationStrategyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Create strategy
	strategy, err := h.service.CreateStrategy(r.Context(), &req)
	if err != nil {
		if err == models.ErrRemediationStrategyExists {
			WriteErrorResponse(w, http.StatusConflict, "Strategy already exists", err.Error())
			return
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to create strategy", err.Error())
		return
	}

	// Convert to response
	response := &models.RemediationStrategyResponse{
		ID:          strategy.ID,
		Type:        strategy.Type,
		Name:        strategy.Name,
		Description: strategy.Description,
		Parameters:  strategy.Parameters,
		Timeout:     strategy.Timeout,
		RetryCount:  strategy.RetryCount,
		IsCustom:    strategy.IsCustom,
		CreatedBy:   strategy.CreatedBy,
		CreatedAt:   strategy.CreatedAt,
	}

	WriteJSONResponse(w, http.StatusCreated, response)
}

// GetStrategy handles GET /api/v1/remediation/strategies/{id}
func (h *Handler) GetStrategy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	strategyID := vars["id"]

	if strategyID == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid strategy ID", "Strategy ID is required")
		return
	}

	// Get strategy
	strategy, err := h.service.GetStrategy(r.Context(), strategyID)
	if err != nil {
		if err == models.ErrRemediationStrategyNotFound {
			WriteErrorResponse(w, http.StatusNotFound, "Strategy not found", err.Error())
			return
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get strategy", err.Error())
		return
	}

	// Convert to response
	response := &models.RemediationStrategyResponse{
		ID:          strategy.ID,
		Type:        strategy.Type,
		Name:        strategy.Name,
		Description: strategy.Description,
		Parameters:  strategy.Parameters,
		Timeout:     strategy.Timeout,
		RetryCount:  strategy.RetryCount,
		IsCustom:    strategy.IsCustom,
		CreatedBy:   strategy.CreatedBy,
		CreatedAt:   strategy.CreatedAt,
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// GetRemediationHistory handles GET /api/v1/remediation/history
func (h *Handler) GetRemediationHistory(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	req := &models.RemediationHistoryRequest{
		Limit:  50,
		Offset: 0,
	}

	// Parse status filter
	if status := r.URL.Query().Get("status"); status != "" {
		jobStatus := models.JobStatus(status)
		req.Status = &jobStatus
	}

	// Parse strategy filter
	if strategy := r.URL.Query().Get("strategy"); strategy != "" {
		strategyType := models.StrategyType(strategy)
		req.Strategy = &strategyType
	}

	// Parse date filters
	if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
		if startDate, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			req.StartDate = &startDate
		}
	}

	if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
		if endDate, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			req.EndDate = &endDate
		}
	}

	// Parse pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			req.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			req.Offset = offset
		}
	}

	// Validate request
	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Get history
	response, err := h.service.GetRemediationHistory(r.Context(), req)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get remediation history", err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// ApproveJob handles POST /api/v1/remediation/jobs/{id}/approve
func (h *Handler) ApproveJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	if jobID == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid job ID", "Job ID is required")
		return
	}

	// Parse approval request
	var approvalReq models.ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&approvalReq); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Validate request
	if err := approvalReq.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Approve job
	err := h.service.ApproveJob(r.Context(), jobID, approvalReq.Approved, approvalReq.Comments)
	if err != nil {
		if err == models.ErrRemediationJobNotFound {
			WriteErrorResponse(w, http.StatusNotFound, "Job not found", err.Error())
			return
		}
		if err == models.ErrJobNotApproved || err == models.ErrJobAlreadyApproved {
			WriteErrorResponse(w, http.StatusBadRequest, "Invalid approval request", err.Error())
			return
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to approve job", err.Error())
		return
	}

	// Get updated job
	job, err := h.service.GetJob(r.Context(), jobID)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get updated job", err.Error())
		return
	}

	// Create response
	response := &models.ApprovalResponse{
		JobID:      job.ID,
		Approved:   approvalReq.Approved,
		ApprovedBy: *job.ApprovedBy,
		ApprovedAt: *job.ApprovedAt,
		Comments:   approvalReq.Comments,
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// Health handles GET /api/v1/remediation/health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	err := h.service.Health(r.Context())
	if err != nil {
		WriteErrorResponse(w, http.StatusServiceUnavailable, "Service unhealthy", err.Error())
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "remediation",
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// Helper functions

// WriteJSONResponse writes a JSON response
func WriteJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// WriteErrorResponse writes an error response
func WriteErrorResponse(w http.ResponseWriter, statusCode int, message, details string) {
	response := map[string]interface{}{
		"error":   message,
		"details": details,
		"status":  statusCode,
	}
	WriteJSONResponse(w, statusCode, response)
}

// WriteValidationError writes a validation error response
func WriteValidationError(w http.ResponseWriter, err error) {
	response := map[string]interface{}{
		"error":   "Validation failed",
		"details": err.Error(),
		"status":  http.StatusBadRequest,
	}
	WriteJSONResponse(w, http.StatusBadRequest, response)
}
