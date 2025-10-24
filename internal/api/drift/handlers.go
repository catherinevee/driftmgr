package drift

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/catherinevee/driftmgr/internal/business/drift"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/gorilla/mux"
)

// Handler handles HTTP requests for drift management
type Handler struct {
	service drift.Service
}

// NewHandler creates a new drift handler
func NewHandler(service drift.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// GetDriftResult handles GET /api/v1/drift/results/{id}
func (h *Handler) GetDriftResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if id == "" {
		http.Error(w, "Missing drift result ID", http.StatusBadRequest)
		return
	}

	result, err := h.service.GetDriftResult(r.Context(), id)
	if err != nil {
		if err == models.ErrDriftResultNotFound {
			http.Error(w, "Drift result not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get drift result", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ListDriftResults handles GET /api/v1/drift/results
func (h *Handler) ListDriftResults(w http.ResponseWriter, r *http.Request) {
	query := &models.DriftResultQuery{
		Filter: models.DriftResultFilter{},
		Sort:   models.DriftResultSort{},
		Limit:  50,
		Offset: 0,
	}

	// Parse query parameters
	if provider := r.URL.Query().Get("provider"); provider != "" {
		query.Filter.Provider = provider
	}

	if status := r.URL.Query().Get("status"); status != "" {
		query.Filter.Status = status
	}

	if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
		if startDate, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			query.Filter.StartDate = startDate
		}
	}

	if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
		if endDate, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			query.Filter.EndDate = endDate
		}
	}

	if minDriftStr := r.URL.Query().Get("min_drift"); minDriftStr != "" {
		if minDrift, err := strconv.Atoi(minDriftStr); err == nil {
			query.Filter.MinDrift = minDrift
		}
	}

	if maxDriftStr := r.URL.Query().Get("max_drift"); maxDriftStr != "" {
		if maxDrift, err := strconv.Atoi(maxDriftStr); err == nil {
			query.Filter.MaxDrift = maxDrift
		}
	}

	if sortField := r.URL.Query().Get("sort_field"); sortField != "" {
		query.Sort.Field = sortField
	}

	if sortOrder := r.URL.Query().Get("sort_order"); sortOrder != "" {
		query.Sort.Order = sortOrder
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			query.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			query.Offset = offset
		}
	}

	results, err := h.service.ListDriftResults(r.Context(), query)
	if err != nil {
		http.Error(w, "Failed to list drift results", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// GetDriftHistory handles GET /api/v1/drift/history
func (h *Handler) GetDriftHistory(w http.ResponseWriter, r *http.Request) {
	req := &models.DriftHistoryRequest{
		Limit:  50,
		Offset: 0,
	}

	// Parse query parameters
	if provider := r.URL.Query().Get("provider"); provider != "" {
		req.Provider = provider
	}

	if status := r.URL.Query().Get("status"); status != "" {
		req.Status = status
	}

	if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
		if startDate, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			req.StartDate = startDate
		}
	}

	if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
		if endDate, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			req.EndDate = endDate
		}
	}

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

	history, err := h.service.GetDriftHistory(r.Context(), req)
	if err != nil {
		http.Error(w, "Failed to get drift history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// GetDriftSummary handles GET /api/v1/drift/summary
func (h *Handler) GetDriftSummary(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")

	summary, err := h.service.GetDriftSummary(r.Context(), provider)
	if err != nil {
		http.Error(w, "Failed to get drift summary", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// DeleteDriftResult handles DELETE /api/v1/drift/results/{id}
func (h *Handler) DeleteDriftResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if id == "" {
		http.Error(w, "Missing drift result ID", http.StatusBadRequest)
		return
	}

	err := h.service.DeleteDriftResult(r.Context(), id)
	if err != nil {
		if err == models.ErrDriftResultNotFound {
			http.Error(w, "Drift result not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete drift result", http.StatusInternalServerError)
		return
	}

	response := models.DriftResultResponse{
		ID:        id,
		Status:    "deleted",
		Message:   "Drift result deleted successfully",
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetDriftTrend handles GET /api/v1/drift/trend
func (h *Handler) GetDriftTrend(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	days := 30 // Default to 30 days

	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if parsedDays, err := strconv.Atoi(daysStr); err == nil && parsedDays > 0 {
			days = parsedDays
		}
	}

	trend, err := h.service.GetDriftTrend(r.Context(), provider, days)
	if err != nil {
		http.Error(w, "Failed to get drift trend", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trend)
}

// GetTopDriftedResources handles GET /api/v1/drift/top-resources
func (h *Handler) GetTopDriftedResources(w http.ResponseWriter, r *http.Request) {
	limit := 10 // Default to top 10

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	resources, err := h.service.GetTopDriftedResources(r.Context(), limit)
	if err != nil {
		http.Error(w, "Failed to get top drifted resources", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"resources": resources,
		"count":     len(resources),
	})
}

// GetDriftBySeverity handles GET /api/v1/drift/severity
func (h *Handler) GetDriftBySeverity(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")

	severity, err := h.service.GetDriftBySeverity(r.Context(), provider)
	if err != nil {
		http.Error(w, "Failed to get drift by severity", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(severity)
}

// Health handles GET /api/v1/drift/health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	err := h.service.Health(r.Context())
	if err != nil {
		http.Error(w, "Service unhealthy", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
	})
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// WriteError writes an error response
func (h *Handler) WriteError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := ErrorResponse{
		Error:     http.StatusText(status),
		Message:   message,
		Timestamp: time.Now(),
	}

	json.NewEncoder(w).Encode(response)
}

// WriteSuccess writes a success response
func (h *Handler) WriteSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
