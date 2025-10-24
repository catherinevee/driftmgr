package auth

import (
	"net/http"
)

// SetupAuthRoutes sets up authentication routes
func SetupAuthRoutes(router Router, authHandlers *AuthHandlers, authMiddleware *AuthMiddleware) {
	// Public authentication routes
	router.POST("/api/v1/auth/login", authHandlers.Login)
	router.POST("/api/v1/auth/register", authHandlers.Register)
	router.POST("/api/v1/auth/refresh", authHandlers.RefreshToken)
	router.POST("/api/v1/auth/logout", authMiddleware.RequireAuth(authHandlers.Logout))

	// Protected user profile routes
	router.GET("/api/v1/auth/profile", authMiddleware.RequireAuth(authHandlers.GetProfile))
	router.PUT("/api/v1/auth/profile", authMiddleware.RequireAuth(authHandlers.UpdateProfile))
	router.POST("/api/v1/auth/change-password", authMiddleware.RequireAuth(authHandlers.ChangePassword))

	// API key management routes
	router.POST("/api/v1/auth/api-keys", authMiddleware.RequireAuth(authHandlers.CreateAPIKey))
	router.GET("/api/v1/auth/api-keys", authMiddleware.RequireAuth(authHandlers.ListAPIKeys))
	router.DELETE("/api/v1/auth/api-keys/{id}", authMiddleware.RequireAuth(authHandlers.DeleteAPIKey))

	// OAuth2 routes
	router.GET("/api/v1/auth/oauth2/providers", authHandlers.GetOAuth2Providers)
	router.GET("/api/v1/auth/oauth2/{provider}", authHandlers.OAuth2Auth)
	router.GET("/api/v1/auth/oauth2/{provider}/callback", authHandlers.OAuth2Callback)

	// Health check for auth service
	router.GET("/api/v1/auth/health", func(w http.ResponseWriter, r *http.Request) {
		WriteJSONResponse(w, http.StatusOK, map[string]string{"status": "healthy"}, nil)
	})
}

// Router interface for route registration
type Router interface {
	GET(path string, handler http.HandlerFunc)
	POST(path string, handler http.HandlerFunc)
	PUT(path string, handler http.HandlerFunc)
	DELETE(path string, handler http.HandlerFunc)
}

// WriteJSONResponse is a helper function for writing JSON responses
func WriteJSONResponse(w http.ResponseWriter, status int, data interface{}, metadata interface{}) {
	// This would use the api package's WriteJSONResponse
	// For now, we'll implement a simple version
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// In a real implementation, you'd use json.NewEncoder(w).Encode()
}
