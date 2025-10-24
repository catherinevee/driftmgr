package auth

import (
	"context"
	"net/http"
)

// AuthMiddleware handles authentication and authorization
type AuthMiddleware struct {
	authService *Service
	jwtService  *JWTService
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authService *Service, jwtService *JWTService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		jwtService:  jwtService,
	}
}

// RequireAuth middleware that requires authentication
func (m *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		token, err := m.jwtService.ExtractTokenFromHeader(authHeader)
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", err.Error())
			return
		}

		// Validate token
		claims, err := m.jwtService.ValidateToken(token)
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid or expired token", err.Error())
			return
		}

		// Add user information to request context
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "username", claims.Username)
		ctx = context.WithValue(ctx, "email", claims.Email)
		ctx = context.WithValue(ctx, "roles", claims.Roles)
		ctx = context.WithValue(ctx, "is_admin", claims.IsAdmin)
		ctx = context.WithValue(ctx, "token", token)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// RequirePermission middleware that requires a specific permission
func (m *AuthMiddleware) RequirePermission(permission string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return m.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
			// Get user roles from context
			roles, ok := r.Context().Value("roles").([]string)
			if !ok {
				writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Access denied", "No roles found")
				return
			}

			// Check if user has required permission
			if !m.hasPermission(roles, permission) {
				writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Access denied", "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole middleware that requires a specific role
func (m *AuthMiddleware) RequireRole(role string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return m.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
			// Get user roles from context
			roles, ok := r.Context().Value("roles").([]string)
			if !ok {
				writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Access denied", "No roles found")
				return
			}

			// Check if user has required role
			if !m.hasRole(roles, role) {
				writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Access denied", "Insufficient role")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin middleware that requires admin privileges
func (m *AuthMiddleware) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return m.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		// Check if user is admin
		isAdmin, ok := r.Context().Value("is_admin").(bool)
		if !ok || !isAdmin {
			writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Access denied", "Admin privileges required")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// OptionalAuth middleware that adds user information if authenticated
func (m *AuthMiddleware) OptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try to extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			token, err := m.jwtService.ExtractTokenFromHeader(authHeader)
			if err == nil {
				// Validate token
				claims, err := m.jwtService.ValidateToken(token)
				if err == nil {
					// Add user information to request context
					ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
					ctx = context.WithValue(ctx, "username", claims.Username)
					ctx = context.WithValue(ctx, "email", claims.Email)
					ctx = context.WithValue(ctx, "roles", claims.Roles)
					ctx = context.WithValue(ctx, "is_admin", claims.IsAdmin)
					ctx = context.WithValue(ctx, "token", token)
					ctx = context.WithValue(ctx, "authenticated", true)

					// Call next handler with updated context
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}

		// No valid authentication, continue without user context
		ctx := context.WithValue(r.Context(), "authenticated", false)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// APIKeyAuth middleware that authenticates using API keys
func (m *AuthMiddleware) APIKeyAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract API key from header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "API key required", "X-API-Key header is missing")
			return
		}

		// Validate API key
		user, apiKeyObj, err := m.authService.ValidateAPIKey(apiKey)
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "INVALID_API_KEY", "Invalid API key", err.Error())
			return
		}

		// Add user and API key information to request context
		ctx := context.WithValue(r.Context(), "user_id", user.ID)
		ctx = context.WithValue(ctx, "username", user.Username)
		ctx = context.WithValue(ctx, "email", user.Email)
		ctx = context.WithValue(ctx, "is_admin", user.IsAdmin)
		ctx = context.WithValue(ctx, "api_key_id", apiKeyObj.ID)
		ctx = context.WithValue(ctx, "api_permissions", apiKeyObj.Permissions)
		ctx = context.WithValue(ctx, "auth_type", "api_key")

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// RequireAPIPermission middleware that requires a specific API permission
func (m *AuthMiddleware) RequireAPIPermission(permission string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return m.APIKeyAuth(func(w http.ResponseWriter, r *http.Request) {
			// Get API permissions from context
			permissions, ok := r.Context().Value("api_permissions").([]string)
			if !ok {
				writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Access denied", "No API permissions found")
				return
			}

			// Check if API key has required permission
			if !m.hasAPIPermission(permissions, permission) {
				writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Access denied", "Insufficient API permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORS middleware for handling cross-origin requests
func (m *AuthMiddleware) CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// Rate limiting middleware (basic implementation)
func (m *AuthMiddleware) RateLimit(requestsPerMinute int) func(http.HandlerFunc) http.HandlerFunc {
	// This is a basic implementation. In production, you'd use a proper rate limiting library
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// For now, just pass through. In production, implement proper rate limiting
			next.ServeHTTP(w, r)
		}
	}
}

// Helper methods

// hasPermission checks if the user has a specific permission
func (m *AuthMiddleware) hasPermission(roles []string, permission string) bool {
	// Check if user has admin role (admins have all permissions)
	for _, role := range roles {
		if role == RoleAdmin {
			return true
		}
	}

	// Check role-specific permissions
	for _, role := range roles {
		if permissions, exists := DefaultRoles[role]; exists {
			for _, perm := range permissions {
				if perm == permission {
					return true
				}
			}
		}
	}

	return false
}

// hasRole checks if the user has a specific role
func (m *AuthMiddleware) hasRole(roles []string, requiredRole string) bool {
	for _, role := range roles {
		if role == requiredRole {
			return true
		}
	}
	return false
}

// hasAPIPermission checks if the API key has a specific permission
func (m *AuthMiddleware) hasAPIPermission(permissions []string, requiredPermission string) bool {
	for _, permission := range permissions {
		if permission == requiredPermission {
			return true
		}
	}
	return false
}

// Context helper functions

// GetUserIDFromContext extracts user ID from request context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value("user_id").(string)
	return userID, ok
}

// GetUsernameFromContext extracts username from request context
func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value("username").(string)
	return username, ok
}

// GetEmailFromContext extracts email from request context
func GetEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value("email").(string)
	return email, ok
}

// GetRolesFromContext extracts roles from request context
func GetRolesFromContext(ctx context.Context) ([]string, bool) {
	roles, ok := ctx.Value("roles").([]string)
	return roles, ok
}

// GetIsAdminFromContext extracts admin status from request context
func GetIsAdminFromContext(ctx context.Context) (bool, bool) {
	isAdmin, ok := ctx.Value("is_admin").(bool)
	return isAdmin, ok
}

// GetTokenFromContext extracts token from request context
func GetTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value("token").(string)
	return token, ok
}

// GetAPIPermissionsFromContext extracts API permissions from request context
func GetAPIPermissionsFromContext(ctx context.Context) ([]string, bool) {
	permissions, ok := ctx.Value("api_permissions").([]string)
	return permissions, ok
}

// IsAuthenticated checks if the request is authenticated
func IsAuthenticated(ctx context.Context) bool {
	authenticated, ok := ctx.Value("authenticated").(bool)
	return ok && authenticated
}

// GetAuthType returns the authentication type used
func GetAuthType(ctx context.Context) string {
	authType, ok := ctx.Value("auth_type").(string)
	if !ok {
		return "jwt"
	}
	return authType
}
