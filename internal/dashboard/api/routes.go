package api

import (
	"net/http"

	"github.com/catherinevee/driftmgr/internal/dashboard/widgets"
	"github.com/gorilla/mux"
)

// RegisterRoutes registers all dashboard-related routes
func RegisterRoutes(router *mux.Router, widgetManager *widgets.Manager) {
	handler := NewHandler(widgetManager)

	// Create dashboard subrouter
	dashboardRouter := router.PathPrefix("/api/v1/dashboard").Subrouter()

	// Widget management routes
	dashboardRouter.HandleFunc("/widgets", handler.CreateWidget).Methods("POST")
	dashboardRouter.HandleFunc("/widgets", handler.ListWidgets).Methods("GET")
	dashboardRouter.HandleFunc("/widgets/{id}", handler.GetWidget).Methods("GET")
	dashboardRouter.HandleFunc("/widgets/{id}", handler.UpdateWidget).Methods("PUT")
	dashboardRouter.HandleFunc("/widgets/{id}", handler.DeleteWidget).Methods("DELETE")
	dashboardRouter.HandleFunc("/widgets/{id}/data", handler.GetWidgetData).Methods("GET")

	// Dashboard management routes
	dashboardRouter.HandleFunc("/dashboards", handler.CreateDashboard).Methods("POST")
	dashboardRouter.HandleFunc("/dashboards", handler.ListDashboards).Methods("GET")
	dashboardRouter.HandleFunc("/dashboards/{id}", handler.GetDashboard).Methods("GET")
	dashboardRouter.HandleFunc("/dashboards/{id}", handler.UpdateDashboard).Methods("PUT")
	dashboardRouter.HandleFunc("/dashboards/{id}", handler.DeleteDashboard).Methods("DELETE")
	dashboardRouter.HandleFunc("/dashboards/{id}/data", handler.GetDashboardData).Methods("GET")

	// Health check route
	dashboardRouter.HandleFunc("/health", handler.HealthCheck).Methods("GET")
}

// HealthCheck provides a health check endpoint for the dashboard service
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	// Simple health check - in a real implementation, you might check
	// database connectivity, external service availability, etc.
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "dashboard",
		"timestamp": "2024-01-01T00:00:00Z", // This would be time.Now() in real implementation
	}

	WriteJSONResponse(w, http.StatusOK, response)
}
