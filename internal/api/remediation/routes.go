package remediation

import (
	"net/http"

	"github.com/catherinevee/driftmgr/internal/business/remediation"
	"github.com/gorilla/mux"
)

// RegisterRoutes registers all remediation API routes
func RegisterRoutes(router *mux.Router, handler *Handler) {
	// Create a subrouter for remediation API
	api := router.PathPrefix("/api/v1/remediation").Subrouter()

	// Job management routes
	api.HandleFunc("/jobs", handler.CreateJob).Methods("POST")
	api.HandleFunc("/jobs", handler.ListJobs).Methods("GET")
	api.HandleFunc("/jobs/{id}", handler.GetJob).Methods("GET")
	api.HandleFunc("/jobs/{id}/cancel", handler.CancelJob).Methods("POST")
	api.HandleFunc("/jobs/{id}/approve", handler.ApproveJob).Methods("POST")
	api.HandleFunc("/progress/{id}", handler.GetJobProgress).Methods("GET")

	// Strategy management routes
	api.HandleFunc("/strategies", handler.ListStrategies).Methods("GET")
	api.HandleFunc("/strategies", handler.CreateStrategy).Methods("POST")
	api.HandleFunc("/strategies/{id}", handler.GetStrategy).Methods("GET")

	// History and analytics routes
	api.HandleFunc("/history", handler.GetRemediationHistory).Methods("GET")

	// Health check route
	api.HandleFunc("/health", handler.Health).Methods("GET")

	// Add middleware
	api.Use(LoggingMiddleware)
	api.Use(CORSMiddleware)
	api.Use(RecoveryMiddleware)
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log request
		// This would typically use a proper logging library
		// For now, we'll just pass through
		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware handles CORS headers
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RecoveryMiddleware recovers from panics
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log panic
				// This would typically use a proper logging library

				// Write error response
				WriteErrorResponse(w, http.StatusInternalServerError, "Internal server error", "An unexpected error occurred")
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// NewHandlerFromService creates a new handler from a service
func NewHandlerFromService(service *remediation.Service) *Handler {
	return NewHandler(service)
}
