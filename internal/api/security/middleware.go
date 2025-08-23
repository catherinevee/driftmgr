package security

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/api/auth"
	"github.com/catherinevee/driftmgr/internal/api/validation"
	"github.com/catherinevee/driftmgr/internal/observability/logging"
	"github.com/catherinevee/driftmgr/internal/security/ratelimit"
	"github.com/catherinevee/driftmgr/internal/security/auth"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// SecurityMiddleware provides comprehensive security features
type SecurityMiddleware struct {
	authMiddleware     *auth.AuthMiddleware
	rateLimiter        *ratelimit.RateLimiter
	requestValidator   *validation.RequestValidator
	logger             *zerolog.Logger
	config             *Config
	corsAllowedOrigins map[string]bool
}

// Config for security middleware
type Config struct {
	// EnableAuth enables authentication
	EnableAuth bool `json:"enable_auth"`
	
	// EnableRateLimit enables rate limiting
	EnableRateLimit bool `json:"enable_rate_limit"`
	
	// EnableValidation enables input validation
	EnableValidation bool `json:"enable_validation"`
	
	// EnableCSRF enables CSRF protection
	EnableCSRF bool `json:"enable_csrf"`
	
	// EnableCORS enables CORS
	EnableCORS bool `json:"enable_cors"`
	
	// CORSOrigins is the list of allowed origins
	CORSOrigins []string `json:"cors_origins"`
	
	// EnableSecurityHeaders enables security headers
	EnableSecurityHeaders bool `json:"enable_security_headers"`
	
	// EnableAuditLog enables audit logging
	EnableAuditLog bool `json:"enable_audit_log"`
	
	// CSRFTokenHeader is the header name for CSRF tokens
	CSRFTokenHeader string `json:"csrf_token_header"`
	
	// SessionTimeout is the session timeout duration
	SessionTimeout time.Duration `json:"session_timeout"`
	
	// EnableIPWhitelist enables IP whitelisting
	EnableIPWhitelist bool `json:"enable_ip_whitelist"`
	
	// IPWhitelist is the list of allowed IPs
	IPWhitelist []string `json:"ip_whitelist"`
}

// DefaultConfig returns default security configuration
func DefaultConfig() *Config {
	return &Config{
		EnableAuth:            true,
		EnableRateLimit:       true,
		EnableValidation:      true,
		EnableCSRF:            true,
		EnableCORS:            true,
		CORSOrigins:           []string{"http://localhost:3000", "http://localhost:8080"},
		EnableSecurityHeaders: true,
		EnableAuditLog:        true,
		CSRFTokenHeader:       "X-CSRF-Token",
		SessionTimeout:        30 * time.Minute,
		EnableIPWhitelist:     false,
		IPWhitelist:           []string{},
	}
}

// NewSecurityMiddleware creates comprehensive security middleware
func NewSecurityMiddleware(authManager *security.AuthManager, config *Config) *SecurityMiddleware {
	if config == nil {
		config = DefaultConfig()
	}
	
	logger := logging.WithComponent("security-middleware")
	
	// Create CORS allowed origins map
	corsOrigins := make(map[string]bool)
	for _, origin := range config.CORSOrigins {
		corsOrigins[origin] = true
	}
	
	return &SecurityMiddleware{
		authMiddleware:     auth.NewAuthMiddleware(authManager, nil),
		rateLimiter:        ratelimit.NewRateLimiter(nil),
		requestValidator:   validation.NewRequestValidator(nil),
		logger:             &logger,
		config:             config,
		corsAllowedOrigins: corsOrigins,
	}
}

// Secure applies all security middleware to a handler
func (sm *SecurityMiddleware) Secure(handler http.HandlerFunc) http.HandlerFunc {
	// Build middleware chain (order matters!)
	secured := handler
	
	// Audit logging (innermost - logs the actual request)
	if sm.config.EnableAuditLog {
		secured = sm.auditLog(secured)
	}
	
	// Input validation
	if sm.config.EnableValidation {
		secured = sm.requestValidator.ValidateRequest(secured)
	}
	
	// CSRF protection
	if sm.config.EnableCSRF {
		secured = sm.csrfProtection(secured)
	}
	
	// Authentication
	if sm.config.EnableAuth {
		secured = sm.authMiddleware.Authenticate(secured)
	}
	
	// Rate limiting
	if sm.config.EnableRateLimit {
		secured = sm.rateLimiter.Middleware(secured)
	}
	
	// IP whitelist (if enabled)
	if sm.config.EnableIPWhitelist {
		secured = sm.ipWhitelist(secured)
	}
	
	// CORS
	if sm.config.EnableCORS {
		secured = sm.cors(secured)
	}
	
	// Security headers (outermost)
	if sm.config.EnableSecurityHeaders {
		secured = sm.securityHeaders(secured)
	}
	
	// Request ID and tracing
	secured = sm.requestID(secured)
	
	return secured
}

// securityHeaders adds security headers to responses
func (sm *SecurityMiddleware) securityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		
		// Remove sensitive headers
		w.Header().Del("Server")
		w.Header().Del("X-Powered-By")
		
		next(w, r)
	}
}

// cors handles CORS headers
func (sm *SecurityMiddleware) cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		
		// Check if origin is allowed
		if sm.corsAllowedOrigins[origin] || sm.corsAllowedOrigins["*"] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token, X-Request-ID")
			w.Header().Set("Access-Control-Max-Age", "3600")
		}
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		
		next(w, r)
	}
}

// csrfProtection provides CSRF protection
func (sm *SecurityMiddleware) csrfProtection(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF for safe methods
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next(w, r)
			return
		}
		
		// Get CSRF token from header
		token := r.Header.Get(sm.config.CSRFTokenHeader)
		if token == "" {
			sm.sendError(w, http.StatusForbidden, "CSRF token required")
			return
		}
		
		// Validate CSRF token (simplified - in production use a proper CSRF library)
		expectedToken := sm.getExpectedCSRFToken(r)
		if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
			sm.logger.Warn().
				Str("ip", r.RemoteAddr).
				Str("path", r.URL.Path).
				Msg("CSRF token validation failed")
			sm.sendError(w, http.StatusForbidden, "invalid CSRF token")
			return
		}
		
		next(w, r)
	}
}

// ipWhitelist restricts access to whitelisted IPs
func (sm *SecurityMiddleware) ipWhitelist(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get client IP
		clientIP := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			clientIP = strings.Split(forwarded, ",")[0]
		}
		
		// Check if IP is whitelisted
		allowed := false
		for _, ip := range sm.config.IPWhitelist {
			if strings.HasPrefix(clientIP, ip) {
				allowed = true
				break
			}
		}
		
		if !allowed {
			sm.logger.Warn().
				Str("ip", clientIP).
				Msg("access denied - IP not whitelisted")
			sm.sendError(w, http.StatusForbidden, "access denied")
			return
		}
		
		next(w, r)
	}
}

// auditLog logs all API requests for compliance
func (sm *SecurityMiddleware) auditLog(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		
		// Create response writer wrapper to capture status
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		
		// Get request ID
		requestID := r.Context().Value("request_id").(string)
		
		// Get user info if authenticated
		var userID, username string
		if user := auth.GetUser(r.Context()); user != nil {
			userID = user.ID
			username = user.Username
		}
		
		// Process request
		next(wrapped, r)
		
		// Log audit entry
		duration := time.Since(startTime)
		
		sm.logger.Info().
			Str("request_id", requestID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("query", r.URL.RawQuery).
			Str("ip", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Str("user_id", userID).
			Str("username", username).
			Int("status", wrapped.statusCode).
			Dur("duration", duration).
			Msg("api_request")
	}
}

// requestID adds a unique request ID to the context
func (sm *SecurityMiddleware) requestID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get or generate request ID
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		// Add to context
		ctx := context.WithValue(r.Context(), "request_id", requestID)
		
		// Add to response header
		w.Header().Set("X-Request-ID", requestID)
		
		next(w, r.WithContext(ctx))
	}
}

// getExpectedCSRFToken generates the expected CSRF token
func (sm *SecurityMiddleware) getExpectedCSRFToken(r *http.Request) string {
	// In production, use a proper CSRF token generation/validation
	// This is simplified for demonstration
	if user := auth.GetUser(r.Context()); user != nil {
		return fmt.Sprintf("csrf_%s_%s", user.ID, user.Username)
	}
	return "csrf_anonymous"
}

// sendError sends an error response
func (sm *SecurityMiddleware) sendError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	response := map[string]interface{}{
		"error":     message,
		"status":    status,
		"timestamp": time.Now().Unix(),
	}
	
	json.NewEncoder(w).Encode(response)
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

// GetStats returns security middleware statistics
func (sm *SecurityMiddleware) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"auth_enabled":       sm.config.EnableAuth,
		"rate_limit_enabled": sm.config.EnableRateLimit,
		"validation_enabled": sm.config.EnableValidation,
		"csrf_enabled":       sm.config.EnableCSRF,
		"cors_enabled":       sm.config.EnableCORS,
		"audit_log_enabled":  sm.config.EnableAuditLog,
	}
	
	if sm.config.EnableRateLimit {
		stats["rate_limit_stats"] = sm.rateLimiter.GetStats()
	}
	
	return stats
}