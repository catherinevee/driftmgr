package automation

import (
	"net/http"

	"github.com/catherinevee/driftmgr/internal/automation/workflows"
	"github.com/gorilla/mux"
)

// RegisterRoutes registers all automation-related routes
func RegisterRoutes(router *mux.Router, manager *workflows.Manager) {
	handler := NewHandler(manager)

	// Create automation subrouter
	automationRouter := router.PathPrefix("/api/v1/automation").Subrouter()

	// Workflow management routes
	automationRouter.HandleFunc("/workflows", handler.CreateWorkflow).Methods("POST")
	automationRouter.HandleFunc("/workflows", handler.ListWorkflows).Methods("GET")
	automationRouter.HandleFunc("/workflows/{id}", handler.GetWorkflow).Methods("GET")
	automationRouter.HandleFunc("/workflows/{id}", handler.UpdateWorkflow).Methods("PUT")
	automationRouter.HandleFunc("/workflows/{id}", handler.DeleteWorkflow).Methods("DELETE")
	automationRouter.HandleFunc("/workflows/{id}/activate", handler.ActivateWorkflow).Methods("POST")
	automationRouter.HandleFunc("/workflows/{id}/deactivate", handler.DeactivateWorkflow).Methods("POST")
	automationRouter.HandleFunc("/workflows/{id}/execute", handler.ExecuteWorkflow).Methods("POST")
	automationRouter.HandleFunc("/workflows/{id}/stats", handler.GetWorkflowStats).Methods("GET")
	automationRouter.HandleFunc("/workflows/{id}/execution-stats", handler.GetExecutionStats).Methods("GET")
	automationRouter.HandleFunc("/workflows/{id}/history", handler.GetExecutionHistory).Methods("GET")

	// Execution management routes
	automationRouter.HandleFunc("/executions", handler.ListExecutions).Methods("GET")
	automationRouter.HandleFunc("/executions/{id}", handler.GetExecution).Methods("GET")
	automationRouter.HandleFunc("/executions/{id}/cancel", handler.CancelExecution).Methods("POST")

	// Health check route
	automationRouter.HandleFunc("/health", handler.HealthCheck).Methods("GET")
}

// HealthCheck provides a health check endpoint for the automation service
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	// Simple health check - in a real implementation, you might check
	// database connectivity, external service availability, etc.
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "automation",
		"timestamp": "2024-01-01T00:00:00Z", // This would be time.Now() in real implementation
	}

	WriteJSONResponse(w, http.StatusOK, response)
}
