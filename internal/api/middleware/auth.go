package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/catherinevee/driftmgr/internal/auth"
	"github.com/gin-gonic/gin"
)

// ContextKey for storing auth information
type ContextKey string

const (
	UserContextKey   ContextKey = "user"
	ClaimsContextKey ContextKey = "claims"
)

// AuthMiddleware provides authentication middleware
type AuthMiddleware struct {
	authService *auth.AuthService
	skipPaths   []string
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authService *auth.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		skipPaths: []string{
			"/health",
			"/metrics",
			"/api/auth/login",
			"/api/auth/refresh",
			"/api/version",
		},
	}
}

// RequireAuth enforces authentication on routes
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for certain paths
		for _, path := range m.skipPaths {
			if strings.HasPrefix(c.Request.URL.Path, path) {
				c.Next()
				return
			}
		}

		// Check for API key first
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			apiKeyData, err := m.authService.ValidateAPIKey(apiKey)
			if err == nil {
				// Set API key context
				c.Set("api_key", apiKeyData)
				c.Set("role", string(apiKeyData.Role))
				c.Set("permissions", apiKeyData.Permissions)
				c.Next()
				return
			}
		}

		// Check for JWT token
		authHeader := c.GetHeader("Authorization")
		token, err := auth.ExtractBearerToken(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Missing or invalid authorization header",
			})
			c.Abort()
			return
		}

		// Validate token
		claims, err := m.authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			c.Abort()
			return
		}

		// Get user information
		user, err := m.authService.GetUser(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not found",
			})
			c.Abort()
			return
		}

		// Set user context
		c.Set(string(UserContextKey), user)
		c.Set(string(ClaimsContextKey), claims)
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", string(claims.Role))
		c.Set("permissions", claims.Permissions)

		c.Next()
	}
}

// RequireRole enforces role-based access control
func (m *AuthMiddleware) RequireRole(roles ...auth.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First ensure authentication
		claims, exists := c.Get(string(ClaimsContextKey))
		if !exists {
			// Check if API key was used
			if apiKey, exists := c.Get("api_key"); exists {
				apiKeyData := apiKey.(*auth.APIKey)
				for _, role := range roles {
					if apiKeyData.Role == role || apiKeyData.Role == auth.RoleAdmin {
						c.Next()
						return
					}
				}
			}

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		userClaims := claims.(*auth.Claims)
		
		// Admin always has access
		if userClaims.Role == auth.RoleAdmin {
			c.Next()
			return
		}

		// Check if user has required role
		for _, role := range roles {
			if userClaims.Role == role {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error": "Insufficient role permissions",
		})
		c.Abort()
	}
}

// RequirePermission enforces permission-based access control
func (m *AuthMiddleware) RequirePermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get(string(ClaimsContextKey))
		if !exists {
			// Check if API key was used
			if apiKey, exists := c.Get("api_key"); exists {
				apiKeyData := apiKey.(*auth.APIKey)
				// Create pseudo claims for API key
				pseudoClaims := &auth.Claims{
					Role:        apiKeyData.Role,
					Permissions: apiKeyData.Permissions,
				}
				if err := m.authService.Authorize(pseudoClaims, resource, action); err == nil {
					c.Next()
					return
				}
			}

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		userClaims := claims.(*auth.Claims)
		if err := m.authService.Authorize(userClaims, resource, action); err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error":    "Insufficient permissions",
				"resource": resource,
				"action":   action,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetUserFromContext retrieves user from context
func GetUserFromContext(c *gin.Context) (*auth.User, bool) {
	user, exists := c.Get(string(UserContextKey))
	if !exists {
		return nil, false
	}
	return user.(*auth.User), true
}

// GetClaimsFromContext retrieves claims from context
func GetClaimsFromContext(c *gin.Context) (*auth.Claims, bool) {
	claims, exists := c.Get(string(ClaimsContextKey))
	if !exists {
		return nil, false
	}
	return claims.(*auth.Claims), true
}

// StandardAuthMiddleware for non-Gin handlers
func StandardAuthMiddleware(authService *auth.AuthService, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for certain paths
		skipPaths := []string{"/health", "/metrics", "/api/auth/login"}
		for _, path := range skipPaths {
			if strings.HasPrefix(r.URL.Path, path) {
				next(w, r)
				return
			}
		}

		// Check for API key
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "" {
			apiKeyData, err := authService.ValidateAPIKey(apiKey)
			if err == nil {
				ctx := context.WithValue(r.Context(), "api_key", apiKeyData)
				ctx = context.WithValue(ctx, "role", string(apiKeyData.Role))
				next(w, r.WithContext(ctx))
				return
			}
		}

		// Check for JWT token
		authHeader := r.Header.Get("Authorization")
		token, err := auth.ExtractBearerToken(authHeader)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := authService.ValidateToken(token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Add claims to context
		ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
		ctx = context.WithValue(ctx, "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "role", string(claims.Role))

		next(w, r.WithContext(ctx))
	}
}