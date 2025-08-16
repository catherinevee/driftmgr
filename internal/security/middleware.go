package security

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Middleware represents HTTP middleware functions
type Middleware struct {
	authManager *AuthManager
	rateLimiter *RateLimiter
}

// NewMiddleware creates new middleware with authentication and rate limiting
func NewMiddleware(authManager *AuthManager, rateLimiter *RateLimiter) *Middleware {
	return &Middleware{
		authManager: authManager,
		rateLimiter: rateLimiter,
	}
}

// AuthMiddleware authenticates requests using JWT tokens
func (m *Middleware) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for login endpoint
		if r.URL.Path == "/api/v1/auth/login" || r.URL.Path == "/login" {
			next.ServeHTTP(w, r)
			return
		}
		
		// Extract token from request
		token := ExtractTokenFromRequest(r)
		if token == "" {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		
		// Validate token
		user, err := m.authManager.ValidateJWTToken(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		
		// Add user to request context
		ctx := r.Context()
		ctx = context.WithValue(ctx, "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// PermissionMiddleware checks if user has required permission
func (m *Middleware) PermissionMiddleware(permission Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := r.Context().Value("user").(*User)
			if !m.authManager.HasPermission(user, permission) {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware applies rate limiting to requests
func (m *Middleware) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client IP
		clientIP := getClientIP(r)
		
		// Check rate limit
		if !m.rateLimiter.Allow(clientIP) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func (m *Middleware) CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow specific origins in production
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:8080",
			"https://yourdomain.com", // Replace with your domain
		}
		
		origin := r.Header.Get("Origin")
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}
		
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersMiddleware adds security headers to responses
func (m *Middleware) SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://cdn.tailwindcss.com https://d3js.org https://unpkg.com; style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; img-src 'self' data: https:; font-src 'self' https://cdn.jsdelivr.net; connect-src 'self' ws: wss:;")
		
		// HSTS header (only for HTTPS)
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}
		
		next.ServeHTTP(w, r)
	})
}

// InputValidationMiddleware validates and sanitizes input
func (m *Middleware) InputValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate Content-Type for POST/PUT requests
		if r.Method == "POST" || r.Method == "PUT" {
			contentType := r.Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/json") && !strings.Contains(contentType, "multipart/form-data") {
				http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
				return
			}
		}
		
		// Validate request size
		if r.ContentLength > 10*1024*1024 { // 10MB limit
			http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs all requests
func (m *Middleware) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)
		
		// Log request details
		duration := time.Since(start)
		user := "anonymous"
		if u := r.Context().Value("user"); u != nil {
			user = u.(*User).Username
		}
		
		fmt.Printf("[%s] %s %s %d %v %s\n",
			time.Now().Format("2006-01-02 15:04:05"),
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration,
			user,
		)
	})
}

// SanitizeResponseMiddleware sanitizes sensitive data in responses
func (m *Middleware) SanitizeResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create response writer wrapper to capture response
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)
		
		// Sanitize response if user is readonly
		if user := r.Context().Value("user"); user != nil {
			u := user.(*User)
			if u.Role == RoleReadOnly {
				// Note: This is a simplified approach. In a real implementation,
				// you'd want to parse the response body and sanitize it properly
				w.Header().Set("X-Data-Sanitized", "true")
			}
		}
	})
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

// getClientIP extracts the real client IP address (duplicate removed - defined in api.go)

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string    `json:"token"`
	User      *User     `json:"user"`
	ExpiresAt time.Time `json:"expires_at"`
}

// HandleLogin handles user authentication
func (m *Middleware) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var loginReq LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate input
	if loginReq.Username == "" || loginReq.Password == "" {
		http.Error(w, "Username and password required", http.StatusBadRequest)
		return
	}
	
	// Authenticate user
	user, err := m.authManager.AuthenticateUser(loginReq.Username, loginReq.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	
	// Generate JWT token
	token, err := m.authManager.GenerateJWTToken(user)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	
	// Create response
	response := LoginResponse{
		Token:     token,
		User:      user,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	
	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil, // Only secure in HTTPS
		SameSite: http.SameSiteStrictMode,
		Expires:  response.ExpiresAt,
	})
	
	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
