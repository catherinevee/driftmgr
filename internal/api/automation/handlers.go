package automation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/catherinevee/driftmgr/internal/automation/workflows"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handler handles automation API requests
type Handler struct {
	manager *workflows.Manager
}

// NewHandler creates a new automation handler
func NewHandler(manager *workflows.Manager) *Handler {
	return &Handler{
		manager: manager,
	}
}

// CreateWorkflow creates a new automation workflow
func (h *Handler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by authentication middleware)
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req models.AutomationWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Create workflow
	workflow, err := h.manager.CreateWorkflow(r.Context(), userID, &req)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusCreated, workflow)
}

// GetWorkflow retrieves a workflow by ID
func (h *Handler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get workflow ID from URL
	vars := mux.Vars(r)
	workflowID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	// Get workflow
	workflow, err := h.manager.GetWorkflow(r.Context(), workflowID)
	if err != nil {
		WriteErrorResponse(w, http.StatusNotFound, "workflow not found")
		return
	}

	// Check ownership
	if workflow.UserID != userID {
		WriteErrorResponse(w, http.StatusForbidden, "access denied")
		return
	}

	WriteJSONResponse(w, http.StatusOK, workflow)
}

// UpdateWorkflow updates an existing workflow
func (h *Handler) UpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get workflow ID from URL
	vars := mux.Vars(r)
	workflowID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	var req models.AutomationWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Update workflow
	workflow, err := h.manager.UpdateWorkflow(r.Context(), userID, workflowID, &req)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, workflow)
}

// DeleteWorkflow deletes a workflow
func (h *Handler) DeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get workflow ID from URL
	vars := mux.Vars(r)
	workflowID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	// Delete workflow
	if err := h.manager.DeleteWorkflow(r.Context(), userID, workflowID); err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListWorkflows lists workflows with optional filtering
func (h *Handler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Parse query parameters
	filter := parseWorkflowFilter(r)

	// List workflows
	workflows, err := h.manager.ListWorkflows(r.Context(), userID, filter)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, workflows)
}

// ActivateWorkflow activates a workflow
func (h *Handler) ActivateWorkflow(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get workflow ID from URL
	vars := mux.Vars(r)
	workflowID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	// Activate workflow
	if err := h.manager.ActivateWorkflow(r.Context(), userID, workflowID); err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeactivateWorkflow deactivates a workflow
func (h *Handler) DeactivateWorkflow(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get workflow ID from URL
	vars := mux.Vars(r)
	workflowID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	// Deactivate workflow
	if err := h.manager.DeactivateWorkflow(r.Context(), userID, workflowID); err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ExecuteWorkflow executes a workflow manually
func (h *Handler) ExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get workflow ID from URL
	vars := mux.Vars(r)
	workflowID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	var req models.AutomationJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Execute workflow
	execution, err := h.manager.ExecuteWorkflow(r.Context(), userID, workflowID, req.Input)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusAccepted, execution)
}

// GetExecution retrieves an execution by ID
func (h *Handler) GetExecution(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get execution ID from URL
	vars := mux.Vars(r)
	executionID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid execution ID")
		return
	}

	// Get execution
	execution, err := h.manager.GetExecution(r.Context(), userID, executionID)
	if err != nil {
		WriteErrorResponse(w, http.StatusNotFound, "execution not found")
		return
	}

	WriteJSONResponse(w, http.StatusOK, execution)
}

// ListExecutions lists executions with optional filtering
func (h *Handler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Parse query parameters
	filter := parseExecutionFilter(r)

	// List executions
	executions, err := h.manager.ListExecutions(r.Context(), userID, filter)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, executions)
}

// GetExecutionHistory retrieves execution history for a workflow
func (h *Handler) GetExecutionHistory(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get workflow ID from URL
	vars := mux.Vars(r)
	workflowID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	// Parse limit parameter
	limit := 50 // Default limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	// Get execution history
	history, err := h.manager.GetExecutionHistory(r.Context(), userID, workflowID, limit)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, history)
}

// GetWorkflowStats retrieves statistics for a workflow
func (h *Handler) GetWorkflowStats(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get workflow ID from URL
	vars := mux.Vars(r)
	workflowID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	// Get workflow stats
	stats, err := h.manager.GetWorkflowStats(r.Context(), userID, workflowID)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, stats)
}

// GetExecutionStats retrieves execution statistics for a workflow
func (h *Handler) GetExecutionStats(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get workflow ID from URL
	vars := mux.Vars(r)
	workflowID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid workflow ID")
		return
	}

	// Get execution stats
	stats, err := h.manager.GetExecutionStats(r.Context(), userID, workflowID)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, stats)
}

// CancelExecution cancels a running execution
func (h *Handler) CancelExecution(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get execution ID from URL
	vars := mux.Vars(r)
	executionID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid execution ID")
		return
	}

	// Cancel execution
	if err := h.manager.CancelExecution(r.Context(), userID, executionID); err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// parseWorkflowFilter parses workflow filter from query parameters
func parseWorkflowFilter(r *http.Request) workflows.WorkflowFilter {
	filter := workflows.WorkflowFilter{}

	// Parse status
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		if status := models.WorkflowStatus(statusStr); status != "" {
			filter.Status = &status
		}
	}

	// Parse trigger type
	if triggerTypeStr := r.URL.Query().Get("trigger_type"); triggerTypeStr != "" {
		if triggerType := models.TriggerType(triggerTypeStr); triggerType != "" {
			filter.TriggerType = &triggerType
		}
	}

	// Parse tags
	if tagsStr := r.URL.Query().Get("tags"); tagsStr != "" {
		// Split comma-separated tags
		filter.Tags = []string{tagsStr} // Simplified for now
	}

	// Parse search
	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = search
	}

	// Parse pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	return filter
}

// parseExecutionFilter parses execution filter from query parameters
func parseExecutionFilter(r *http.Request) workflows.ExecutionFilter {
	filter := workflows.ExecutionFilter{}

	// Parse workflow ID
	if workflowIDStr := r.URL.Query().Get("workflow_id"); workflowIDStr != "" {
		if workflowID, err := uuid.Parse(workflowIDStr); err == nil {
			filter.WorkflowID = &workflowID
		}
	}

	// Parse status
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		if status := models.JobStatus(statusStr); status != "" {
			filter.Status = &status
		}
	}

	// Parse start time
	if startTimeStr := r.URL.Query().Get("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = &startTime
		}
	}

	// Parse end time
	if endTimeStr := r.URL.Query().Get("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = &endTime
		}
	}

	// Parse pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	return filter
}
