package state

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/catherinevee/driftmgr/internal/business/state"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/gorilla/mux"
)

// Handler handles HTTP requests for state management operations
type Handler struct {
	service *state.Service
}

// NewHandler creates a new state management handler
func NewHandler(service *state.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// ListStateFiles handles GET /api/v1/state/files
func (h *Handler) ListStateFiles(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	req := &models.StateFileListRequest{
		Limit:     50,
		Offset:    0,
		SortBy:    "last_modified",
		SortOrder: "desc",
	}

	// Parse backend_id filter
	if backendID := r.URL.Query().Get("backend_id"); backendID != "" {
		req.BackendID = &backendID
	}

	// Parse environment_id filter
	if environmentID := r.URL.Query().Get("environment_id"); environmentID != "" {
		req.EnvironmentID = &environmentID
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

	// List state files
	response, err := h.service.ListStateFiles(r.Context(), req)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to list state files", err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// GetStateFile handles GET /api/v1/state/files/{id}
func (h *Handler) GetStateFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stateFileID := vars["id"]

	if stateFileID == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid state file ID", "State file ID is required")
		return
	}

	// Get state file
	stateFile, err := h.service.GetStateFile(r.Context(), stateFileID)
	if err != nil {
		if err == models.ErrStateFileNotFound {
			WriteErrorResponse(w, http.StatusNotFound, "State file not found", err.Error())
			return
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get state file", err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, stateFile)
}

// ImportResource handles POST /api/v1/state/files/{id}/import
func (h *Handler) ImportResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stateFileID := vars["id"]

	if stateFileID == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid state file ID", "State file ID is required")
		return
	}

	// Parse import request
	var req models.ImportResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Import resource
	response, err := h.service.ImportResource(r.Context(), stateFileID, &req)
	if err != nil {
		if err == models.ErrStateFileNotFound {
			WriteErrorResponse(w, http.StatusNotFound, "State file not found", err.Error())
			return
		}
		if err == models.ErrStateFileLocked {
			WriteErrorResponse(w, http.StatusConflict, "State file is locked", err.Error())
			return
		}
		if err == models.ErrResourceExists {
			WriteErrorResponse(w, http.StatusConflict, "Resource already exists", err.Error())
			return
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to import resource", err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusCreated, response)
}

// RemoveResource handles DELETE /api/v1/state/files/{id}/resources/{resource}
func (h *Handler) RemoveResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stateFileID := vars["id"]
	resourceAddress := vars["resource"]

	if stateFileID == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid state file ID", "State file ID is required")
		return
	}

	if resourceAddress == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid resource address", "Resource address is required")
		return
	}

	// Parse remove request
	var req models.RemoveResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Set resource address from URL
	req.ResourceAddress = resourceAddress

	// Validate request
	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Remove resource
	response, err := h.service.RemoveResource(r.Context(), stateFileID, &req)
	if err != nil {
		if err == models.ErrStateFileNotFound {
			WriteErrorResponse(w, http.StatusNotFound, "State file not found", err.Error())
			return
		}
		if err == models.ErrResourceNotFound {
			WriteErrorResponse(w, http.StatusNotFound, "Resource not found", err.Error())
			return
		}
		if err == models.ErrStateFileLocked {
			WriteErrorResponse(w, http.StatusConflict, "State file is locked", err.Error())
			return
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to remove resource", err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// MoveResource handles POST /api/v1/state/files/{id}/move
func (h *Handler) MoveResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stateFileID := vars["id"]

	if stateFileID == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid state file ID", "State file ID is required")
		return
	}

	// Parse move request
	var req models.MoveResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Move resource
	response, err := h.service.MoveResource(r.Context(), stateFileID, &req)
	if err != nil {
		if err == models.ErrStateFileNotFound {
			WriteErrorResponse(w, http.StatusNotFound, "State file not found", err.Error())
			return
		}
		if err == models.ErrResourceNotFound {
			WriteErrorResponse(w, http.StatusNotFound, "Resource not found", err.Error())
			return
		}
		if err == models.ErrResourceExists {
			WriteErrorResponse(w, http.StatusConflict, "Resource already exists at target address", err.Error())
			return
		}
		if err == models.ErrStateFileLocked {
			WriteErrorResponse(w, http.StatusConflict, "State file is locked", err.Error())
			return
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to move resource", err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// ListResources handles GET /api/v1/state/resources
func (h *Handler) ListResources(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	req := &models.ResourceListRequest{
		Limit:     50,
		Offset:    0,
		SortBy:    "address",
		SortOrder: "asc",
	}

	// Parse state_file_id filter
	if stateFileID := r.URL.Query().Get("state_file_id"); stateFileID != "" {
		req.StateFileID = &stateFileID
	}

	// Parse resource_type filter
	if resourceType := r.URL.Query().Get("resource_type"); resourceType != "" {
		req.ResourceType = &resourceType
	}

	// Parse provider filter
	if provider := r.URL.Query().Get("provider"); provider != "" {
		req.Provider = &provider
	}

	// Parse module filter
	if module := r.URL.Query().Get("module"); module != "" {
		req.Module = &module
	}

	// Parse address filter
	if address := r.URL.Query().Get("address"); address != "" {
		req.Address = &address
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

	// List resources
	response, err := h.service.ListResources(r.Context(), req)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to list resources", err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// GetResource handles GET /api/v1/state/resources/{id}
func (h *Handler) GetResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["id"]

	if resourceID == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid resource ID", "Resource ID is required")
		return
	}

	// Get resource
	resource, err := h.service.GetResource(r.Context(), resourceID)
	if err != nil {
		if err == models.ErrResourceNotFound {
			WriteErrorResponse(w, http.StatusNotFound, "Resource not found", err.Error())
			return
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get resource", err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, resource)
}

// ExportResource handles POST /api/v1/state/resources/{id}/export
func (h *Handler) ExportResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["id"]

	if resourceID == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid resource ID", "Resource ID is required")
		return
	}

	// Parse export request
	var req models.ExportResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Export resource
	response, err := h.service.ExportResource(r.Context(), resourceID, &req)
	if err != nil {
		if err == models.ErrResourceNotFound {
			WriteErrorResponse(w, http.StatusNotFound, "Resource not found", err.Error())
			return
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to export resource", err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// ListBackends handles GET /api/v1/state/backends
func (h *Handler) ListBackends(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	req := &models.BackendListRequest{
		Limit:     50,
		Offset:    0,
		SortBy:    "name",
		SortOrder: "asc",
	}

	// Parse environment_id filter
	if environmentID := r.URL.Query().Get("environment_id"); environmentID != "" {
		req.EnvironmentID = &environmentID
	}

	// Parse type filter
	if backendType := r.URL.Query().Get("type"); backendType != "" {
		bt := models.BackendType(backendType)
		req.Type = &bt
	}

	// Parse is_active filter
	if isActiveStr := r.URL.Query().Get("is_active"); isActiveStr != "" {
		if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
			req.IsActive = &isActive
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

	// List backends
	response, err := h.service.ListBackends(r.Context(), req)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to list backends", err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// CreateBackend handles POST /api/v1/state/backends
func (h *Handler) CreateBackend(w http.ResponseWriter, r *http.Request) {
	// Parse create request
	var req models.BackendCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Create backend
	backend, err := h.service.CreateBackend(r.Context(), &req)
	if err != nil {
		if err == models.ErrBackendExists {
			WriteErrorResponse(w, http.StatusConflict, "Backend already exists", err.Error())
			return
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to create backend", err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusCreated, backend)
}

// GetBackend handles GET /api/v1/state/backends/{id}
func (h *Handler) GetBackend(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	backendID := vars["id"]

	if backendID == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid backend ID", "Backend ID is required")
		return
	}

	// Get backend
	backend, err := h.service.GetBackend(r.Context(), backendID)
	if err != nil {
		if err == models.ErrBackendNotFound {
			WriteErrorResponse(w, http.StatusNotFound, "Backend not found", err.Error())
			return
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to get backend", err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, backend)
}

// LockStateFile handles POST /api/v1/state/files/{id}/lock
func (h *Handler) LockStateFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stateFileID := vars["id"]

	if stateFileID == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid state file ID", "State file ID is required")
		return
	}

	// Parse lock request
	var req models.StateLockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Lock state file
	response, err := h.service.LockStateFile(r.Context(), stateFileID, &req)
	if err != nil {
		if err == models.ErrStateFileNotFound {
			WriteErrorResponse(w, http.StatusNotFound, "State file not found", err.Error())
			return
		}
		if err == models.ErrStateFileLocked {
			WriteErrorResponse(w, http.StatusConflict, "State file is already locked", err.Error())
			return
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to lock state file", err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// UnlockStateFile handles POST /api/v1/state/files/{id}/unlock
func (h *Handler) UnlockStateFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stateFileID := vars["id"]

	if stateFileID == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid state file ID", "State file ID is required")
		return
	}

	// Parse unlock request
	var req models.StateUnlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Unlock state file
	response, err := h.service.UnlockStateFile(r.Context(), stateFileID, &req)
	if err != nil {
		if err == models.ErrStateFileNotFound {
			WriteErrorResponse(w, http.StatusNotFound, "State file not found", err.Error())
			return
		}
		if err == models.ErrStateLockNotFound {
			WriteErrorResponse(w, http.StatusNotFound, "State lock not found", err.Error())
			return
		}
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to unlock state file", err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// Health handles GET /api/v1/state/health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	err := h.service.Health(r.Context())
	if err != nil {
		WriteErrorResponse(w, http.StatusServiceUnavailable, "Service unhealthy", err.Error())
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "state_management",
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
