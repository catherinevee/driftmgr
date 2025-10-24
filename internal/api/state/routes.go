package state

import (
	"net/http"

	"github.com/catherinevee/driftmgr/internal/business/state"
	"github.com/gorilla/mux"
)

// RegisterRoutes registers all state management API routes
func RegisterRoutes(router *mux.Router, handler *Handler) {
	// Create a subrouter for state management API
	api := router.PathPrefix("/api/v1/state").Subrouter()

	// State file management routes
	api.HandleFunc("/files", handler.ListStateFiles).Methods("GET")
	api.HandleFunc("/files/{id}", handler.GetStateFile).Methods("GET")
	api.HandleFunc("/files/{id}/import", handler.ImportResource).Methods("POST")
	api.HandleFunc("/files/{id}/resources/{resource}", handler.RemoveResource).Methods("DELETE")
	api.HandleFunc("/files/{id}/move", handler.MoveResource).Methods("POST")
	api.HandleFunc("/files/{id}/lock", handler.LockStateFile).Methods("POST")
	api.HandleFunc("/files/{id}/unlock", handler.UnlockStateFile).Methods("POST")

	// Resource management routes
	api.HandleFunc("/resources", handler.ListResources).Methods("GET")
	api.HandleFunc("/resources/{id}", handler.GetResource).Methods("GET")
	api.HandleFunc("/resources/{id}/export", handler.ExportResource).Methods("POST")

	// Backend management routes
	api.HandleFunc("/backends", handler.ListBackends).Methods("GET")
	api.HandleFunc("/backends", handler.CreateBackend).Methods("POST")
	api.HandleFunc("/backends/{id}", handler.GetBackend).Methods("GET")

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
func NewHandlerFromService(service *state.Service) *Handler {
	return NewHandler(service)
}
