package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/observability/logging"
	"github.com/catherinevee/driftmgr/internal/security/auth"
	"github.com/catherinevee/driftmgr/internal/utils/errors"
	"github.com/rs/zerolog"
)

// contextKey for storing auth info in context
type contextKey string

const (
	userContextKey  contextKey = "user"
	tokenContextKey contextKey = "token"
)

// AuthMiddleware provides authentication and authorization for API endpoints
type AuthMiddleware struct {
	authManager *security.AuthManager
	logger      *zerolog.Logger
	config      *Config
}

// Config for authentication middleware
type Config struct {
	// RequireAuth determines if all endpoints require authentication
	RequireAuth bool `json:"require_auth"`

	// PublicPaths are paths that don't require authentication
	PublicPaths []string `json:"public_paths"`

	// APIKeyHeader is the header name for API key authentication
	APIKeyHeader string `json:"api_key_header"`

	// EnableJWT enables JWT token authentication
	EnableJWT bool `json:"enable_jwt"`

	// EnableAPIKey enables API key authentication
	EnableAPIKey bool `json:"enable_api_key"`

	// TokenExpiry is the JWT token expiry duration
	TokenExpiry time.Duration `json:"token_expiry"`

	// RefreshExpiry is the refresh token expiry duration
	RefreshExpiry time.Duration `json:"refresh_expiry"`

	// MaxFailedAttempts before account lockout
	MaxFailedAttempts int `json:"max_failed_attempts"`

	// LockoutDuration after max failed attempts
	LockoutDuration time.Duration `json:"lockout_duration"`
}

// DefaultConfig returns default authentication configuration
func DefaultConfig() *Config {
	return &Config{
		RequireAuth:       true,
		PublicPaths:       []string{"/health", "/health/ready", "/api/auth/login", "/api/auth/register"},
		APIKeyHeader:      "X-API-Key",
		EnableJWT:         true,
		EnableAPIKey:      true,
		TokenExpiry:       15 * time.Minute,
		RefreshExpiry:     7 * 24 * time.Hour,
		MaxFailedAttempts: 5,
		LockoutDuration:   30 * time.Minute,
	}
}

// NewAuthMiddleware creates new authentication middleware
func NewAuthMiddleware(authManager *security.AuthManager, config *Config) *AuthMiddleware {
	if config == nil {
		config = DefaultConfig()
	}

	logger := logging.WithComponent("auth-middleware")

	return &AuthMiddleware{
		authManager: authManager,
		logger:      &logger,
		config:      config,
	}
}

// Authenticate wraps an HTTP handler with authentication
func (m *AuthMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if path is public
		if m.isPublicPath(r.URL.Path) {
			next(w, r)
			return
		}

		// Skip auth if not required
		if !m.config.RequireAuth {
			next(w, r)
			return
		}

		// Try to authenticate the request
		user, token, err := m.authenticateRequest(r)
		if err != nil {
			m.handleAuthError(w, r, err)
			return
		}

		// Add user and token to context
		ctx := context.WithValue(r.Context(), userContextKey, user)
		ctx = context.WithValue(ctx, tokenContextKey, token)

		// Audit successful authentication
		m.logger.Info().
			Str("user_id", user.ID).
			Str("username", user.Username).
			Str("path", r.URL.Path).
			Str("method", r.Method).
			Str("ip", r.RemoteAddr).
			Msg("authenticated request")

		// Continue with authenticated context
		next(w, r.WithContext(ctx))
	}
}

// RequirePermission checks if user has required permission
func (m *AuthMiddleware) RequirePermission(permission security.Permission) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return m.Authenticate(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r.Context())
			if user == nil {
				m.sendError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			// Check if user has permission
			hasPermission := false
			for _, p := range user.Permissions {
				if p == permission {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				m.logger.Warn().
					Str("user_id", user.ID).
					Str("permission", string(permission)).
					Str("path", r.URL.Path).
					Msg("permission denied")

				m.sendError(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			next(w, r)
		})
	}
}

// RequireRole checks if user has required role
func (m *AuthMiddleware) RequireRole(role security.UserRole) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return m.Authenticate(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r.Context())
			if user == nil {
				m.sendError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			if user.Role != role && user.Role != security.RoleRoot {
				m.logger.Warn().
					Str("user_id", user.ID).
					Str("required_role", string(role)).
					Str("user_role", string(user.Role)).
					Str("path", r.URL.Path).
					Msg("role check failed")

				m.sendError(w, http.StatusForbidden, "insufficient role privileges")
				return
			}

			next(w, r)
		})
	}
}

// authenticateRequest tries various authentication methods
func (m *AuthMiddleware) authenticateRequest(r *http.Request) (*security.User, string, error) {
	// Try JWT authentication first
	if m.config.EnableJWT {
		if user, token, err := m.authenticateJWT(r); err == nil {
			return user, token, nil
		}
	}

	// Try API key authentication
	if m.config.EnableAPIKey {
		if user, key, err := m.authenticateAPIKey(r); err == nil {
			return user, key, nil
		}
	}

	// Try basic authentication
	if username, password, ok := r.BasicAuth(); ok {
		if user, err := m.authManager.Authenticate(username, password); err == nil {
			// Generate a JWT token for subsequent requests
			token, _ := m.authManager.GenerateToken(user)
			return user, token, nil
		}
	}

	return nil, "", errors.New(errors.ErrorTypeAuth, "no valid authentication provided")
}

// authenticateJWT validates JWT token
func (m *AuthMiddleware) authenticateJWT(r *http.Request) (*security.User, string, error) {
	// Get token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, "", errors.New(errors.ErrorTypeAuth, "no authorization header")
	}

	// Check for Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, "", errors.New(errors.ErrorTypeAuth, "invalid authorization header format")
	}

	token := parts[1]

	// Validate token
	user, err := m.authManager.ValidateToken(token)
	if err != nil {
		return nil, "", errors.Wrap(err, errors.ErrorTypeAuth, "token validation failed")
	}

	return user, token, nil
}

// authenticateAPIKey validates API key
func (m *AuthMiddleware) authenticateAPIKey(r *http.Request) (*security.User, string, error) {
	// Get API key from header
	apiKey := r.Header.Get(m.config.APIKeyHeader)
	if apiKey == "" {
		// Try query parameter as fallback
		apiKey = r.URL.Query().Get("api_key")
	}

	if apiKey == "" {
		return nil, "", errors.New(errors.ErrorTypeAuth, "no API key provided")
	}

	// Validate API key
	user, err := m.authManager.ValidateAPIKey(apiKey)
	if err != nil {
		return nil, "", errors.Wrap(err, errors.ErrorTypeAuth, "API key validation failed")
	}

	return user, apiKey, nil
}

// isPublicPath checks if path doesn't require authentication
func (m *AuthMiddleware) isPublicPath(path string) bool {
	for _, publicPath := range m.config.PublicPaths {
		if strings.HasPrefix(path, publicPath) {
			return true
		}
	}
	return false
}

// handleAuthError handles authentication errors
func (m *AuthMiddleware) handleAuthError(w http.ResponseWriter, r *http.Request, err error) {
	m.logger.Error().
		Err(err).
		Str("path", r.URL.Path).
		Str("method", r.Method).
		Str("ip", r.RemoteAddr).
		Msg("authentication failed")

	// Send appropriate error response
	if errors.Is(err, errors.ErrorTypeAuth) {
		m.sendError(w, http.StatusUnauthorized, "authentication failed")
	} else {
		m.sendError(w, http.StatusInternalServerError, "internal server error")
	}
}

// sendError sends an error response
func (m *AuthMiddleware) sendError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := map[string]interface{}{
		"error":     message,
		"status":    status,
		"timestamp": time.Now().Unix(),
	}

	json.NewEncoder(w).Encode(response)
}

// GetUser retrieves user from context
func GetUser(ctx context.Context) *security.User {
	if user, ok := ctx.Value(userContextKey).(*security.User); ok {
		return user
	}
	return nil
}

// GetToken retrieves token from context
func GetToken(ctx context.Context) string {
	if token, ok := ctx.Value(tokenContextKey).(string); ok {
		return token
	}
	return ""
}

// RateLimitMiddleware provides rate limiting per user/IP
func (m *AuthMiddleware) RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get identifier (user ID or IP)
		identifier := r.RemoteAddr
		if user := GetUser(r.Context()); user != nil {
			identifier = user.ID
		}

		// Check rate limit (implementation would use a rate limiter)
		// For now, just pass through
		next(w, r)
	}
}

// AuditMiddleware logs all API access for compliance
func (m *AuthMiddleware) AuditMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// Create response writer wrapper to capture status
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process request
		next(wrapped, r)

		// Log audit entry
		duration := time.Since(startTime)
		user := GetUser(r.Context())

		auditEntry := m.logger.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("ip", r.RemoteAddr).
			Int("status", wrapped.statusCode).
			Dur("duration", duration)

		if user != nil {
			auditEntry.
				Str("user_id", user.ID).
				Str("username", user.Username)
		}

		auditEntry.Msg("api_access")
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
