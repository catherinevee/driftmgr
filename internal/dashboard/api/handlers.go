package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/catherinevee/driftmgr/internal/dashboard/widgets"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handler handles dashboard API requests
type Handler struct {
	widgetManager *widgets.Manager
}

// NewHandler creates a new dashboard handler
func NewHandler(widgetManager *widgets.Manager) *Handler {
	return &Handler{
		widgetManager: widgetManager,
	}
}

// CreateWidget creates a new dashboard widget
func (h *Handler) CreateWidget(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by authentication middleware)
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req models.DashboardWidgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Create widget
	widget, err := h.widgetManager.CreateWidget(r.Context(), userID, &req)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusCreated, widget)
}

// GetWidget retrieves a widget by ID
func (h *Handler) GetWidget(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get widget ID from URL
	vars := mux.Vars(r)
	widgetID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid widget ID")
		return
	}

	// Get widget
	widget, err := h.widgetManager.GetWidget(r.Context(), userID, widgetID)
	if err != nil {
		WriteErrorResponse(w, http.StatusNotFound, "widget not found")
		return
	}

	WriteJSONResponse(w, http.StatusOK, widget)
}

// UpdateWidget updates an existing widget
func (h *Handler) UpdateWidget(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get widget ID from URL
	vars := mux.Vars(r)
	widgetID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid widget ID")
		return
	}

	var req models.DashboardWidgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Update widget
	widget, err := h.widgetManager.UpdateWidget(r.Context(), userID, widgetID, &req)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, widget)
}

// DeleteWidget deletes a widget
func (h *Handler) DeleteWidget(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get widget ID from URL
	vars := mux.Vars(r)
	widgetID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid widget ID")
		return
	}

	// Delete widget
	if err := h.widgetManager.DeleteWidget(r.Context(), userID, widgetID); err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListWidgets lists widgets with optional filtering
func (h *Handler) ListWidgets(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Parse query parameters
	filter := parseWidgetFilter(r)

	// List widgets
	widgets, err := h.widgetManager.ListWidgets(r.Context(), userID, filter)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, widgets)
}

// GetWidgetData retrieves data for a widget
func (h *Handler) GetWidgetData(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get widget ID from URL
	vars := mux.Vars(r)
	widgetID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid widget ID")
		return
	}

	// Get widget data
	data, err := h.widgetManager.GetWidgetData(r.Context(), userID, widgetID)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, data)
}

// CreateDashboard creates a new dashboard
func (h *Handler) CreateDashboard(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req models.DashboardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Create dashboard
	dashboard, err := h.widgetManager.CreateDashboard(r.Context(), userID, &req)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusCreated, dashboard)
}

// GetDashboard retrieves a dashboard by ID
func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get dashboard ID from URL
	vars := mux.Vars(r)
	dashboardID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid dashboard ID")
		return
	}

	// Get dashboard
	dashboard, err := h.widgetManager.GetDashboard(r.Context(), userID, dashboardID)
	if err != nil {
		WriteErrorResponse(w, http.StatusNotFound, "dashboard not found")
		return
	}

	WriteJSONResponse(w, http.StatusOK, dashboard)
}

// UpdateDashboard updates an existing dashboard
func (h *Handler) UpdateDashboard(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get dashboard ID from URL
	vars := mux.Vars(r)
	dashboardID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid dashboard ID")
		return
	}

	var req models.DashboardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		WriteValidationError(w, err)
		return
	}

	// Update dashboard
	dashboard, err := h.widgetManager.UpdateDashboard(r.Context(), userID, dashboardID, &req)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, dashboard)
}

// DeleteDashboard deletes a dashboard
func (h *Handler) DeleteDashboard(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get dashboard ID from URL
	vars := mux.Vars(r)
	dashboardID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid dashboard ID")
		return
	}

	// Delete dashboard
	if err := h.widgetManager.DeleteDashboard(r.Context(), userID, dashboardID); err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListDashboards lists dashboards with optional filtering
func (h *Handler) ListDashboards(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Parse query parameters
	filter := parseDashboardFilter(r)

	// List dashboards
	dashboards, err := h.widgetManager.ListDashboards(r.Context(), userID, filter)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, dashboards)
}

// GetDashboardData retrieves data for a dashboard
func (h *Handler) GetDashboardData(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		WriteErrorResponse(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get dashboard ID from URL
	vars := mux.Vars(r)
	dashboardID, err := uuid.Parse(vars["id"])
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "invalid dashboard ID")
		return
	}

	// Get dashboard data
	data, err := h.widgetManager.GetDashboardData(r.Context(), userID, dashboardID)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSONResponse(w, http.StatusOK, data)
}

// parseWidgetFilter parses widget filter from query parameters
func parseWidgetFilter(r *http.Request) widgets.WidgetFilter {
	filter := widgets.WidgetFilter{}

	// Parse dashboard ID
	if dashboardIDStr := r.URL.Query().Get("dashboard_id"); dashboardIDStr != "" {
		if dashboardID, err := uuid.Parse(dashboardIDStr); err == nil {
			filter.DashboardID = &dashboardID
		}
	}

	// Parse widget type
	if typeStr := r.URL.Query().Get("type"); typeStr != "" {
		if widgetType := models.WidgetType(typeStr); widgetType != "" {
			filter.Type = &widgetType
		}
	}

	// Parse status
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		if status := models.WidgetStatus(statusStr); status != "" {
			filter.Status = &status
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

// parseDashboardFilter parses dashboard filter from query parameters
func parseDashboardFilter(r *http.Request) widgets.DashboardFilter {
	filter := widgets.DashboardFilter{}

	// Parse status
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		if status := models.DashboardStatus(statusStr); status != "" {
			filter.Status = &status
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
