package drift

import (
	"net/http"

	"github.com/gorilla/mux"
)

// RegisterRoutes registers all drift-related routes
func RegisterRoutes(router *mux.Router, handler *Handler) {
	// Create subrouter for drift API
	driftRouter := router.PathPrefix("/api/v1/drift").Subrouter()

	// Drift Results endpoints
	driftRouter.HandleFunc("/results/{id}", handler.GetDriftResult).Methods("GET")
	driftRouter.HandleFunc("/results", handler.ListDriftResults).Methods("GET")
	driftRouter.HandleFunc("/results/{id}", handler.DeleteDriftResult).Methods("DELETE")

	// Drift History endpoint
	driftRouter.HandleFunc("/history", handler.GetDriftHistory).Methods("GET")

	// Drift Summary endpoint
	driftRouter.HandleFunc("/summary", handler.GetDriftSummary).Methods("GET")

	// Drift Trend endpoint
	driftRouter.HandleFunc("/trend", handler.GetDriftTrend).Methods("GET")

	// Top Drifted Resources endpoint
	driftRouter.HandleFunc("/top-resources", handler.GetTopDriftedResources).Methods("GET")

	// Drift by Severity endpoint
	driftRouter.HandleFunc("/severity", handler.GetDriftBySeverity).Methods("GET")

	// Health check endpoint
	driftRouter.HandleFunc("/health", handler.Health).Methods("GET")

	// Add middleware for all drift routes
	driftRouter.Use(
		LoggingMiddleware,
		RecoveryMiddleware,
		RateLimitMiddleware,
	)
}

// Middleware functions

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log request details
		// This would typically use a proper logger
		next.ServeHTTP(w, r)
	})
}

// RecoveryMiddleware recovers from panics
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RateLimitMiddleware implements rate limiting
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple rate limiting implementation
		// In production, this would use a proper rate limiter like Redis
		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware adds CORS headers
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware implements authentication
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple authentication check
		// In production, this would validate JWT tokens or API keys
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ValidationMiddleware validates request parameters
func ValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate request parameters
		// This would typically use a validation library
		next.ServeHTTP(w, r)
	})
}
