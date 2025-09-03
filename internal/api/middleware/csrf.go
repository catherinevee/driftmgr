package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CSRFMiddleware provides CSRF protection
type CSRFMiddleware struct {
	tokens      map[string]*csrfToken
	mu          sync.RWMutex
	tokenLength int
	ttl         time.Duration
}

type csrfToken struct {
	token     string
	expiresAt time.Time
	sessionID string
}

// NewCSRFMiddleware creates a new CSRF middleware
func NewCSRFMiddleware() *CSRFMiddleware {
	m := &CSRFMiddleware{
		tokens:      make(map[string]*csrfToken),
		tokenLength: 32,
		ttl:         1 * time.Hour,
	}

	// Start cleanup routine
	go m.cleanupExpiredTokens()

	return m
}

// GenerateToken generates a new CSRF token
func (m *CSRFMiddleware) GenerateToken(sessionID string) (string, error) {
	tokenBytes := make([]byte, m.tokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)

	m.mu.Lock()
	m.tokens[token] = &csrfToken{
		token:     token,
		expiresAt: time.Now().Add(m.ttl),
		sessionID: sessionID,
	}
	m.mu.Unlock()

	return token, nil
}

// ValidateToken validates a CSRF token
func (m *CSRFMiddleware) ValidateToken(token string, sessionID string) bool {
	m.mu.RLock()
	csrfToken, exists := m.tokens[token]
	m.mu.RUnlock()

	if !exists {
		return false
	}

	// Check expiration
	if time.Now().After(csrfToken.expiresAt) {
		m.mu.Lock()
		delete(m.tokens, token)
		m.mu.Unlock()
		return false
	}

	// Check session match
	if csrfToken.sessionID != sessionID {
		return false
	}

	// Token is valid, remove it (one-time use)
	m.mu.Lock()
	delete(m.tokens, token)
	m.mu.Unlock()

	return true
}

// ProtectCSRF enforces CSRF protection on state-changing operations
func (m *CSRFMiddleware) ProtectCSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF for safe methods
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Skip for API key authentication (service-to-service)
		if c.GetHeader("X-API-Key") != "" {
			c.Next()
			return
		}

		// Get CSRF token from header or form
		csrfToken := c.GetHeader("X-CSRF-Token")
		if csrfToken == "" {
			csrfToken = c.PostForm("csrf_token")
		}

		if csrfToken == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "CSRF token required",
			})
			c.Abort()
			return
		}

		// Get session ID from context (set by auth middleware)
		sessionID, exists := c.Get("session_id")
		if !exists {
			// Try to get from claims
			if claims, exists := c.Get(string(ClaimsContextKey)); exists {
				if authClaims, ok := claims.(map[string]interface{}); ok {
					sessionID = authClaims.SessionID
				}
			}
		}

		if sessionID == nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Invalid session",
			})
			c.Abort()
			return
		}

		// Validate CSRF token
		if !m.ValidateToken(csrfToken, sessionID.(string)) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Invalid or expired CSRF token",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetCSRFToken endpoint handler to get a new CSRF token
func (m *CSRFMiddleware) GetCSRFToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get session ID from context
		sessionID, exists := c.Get("session_id")
		if !exists {
			// Try to get from claims
			if claims, exists := c.Get(string(ClaimsContextKey)); exists {
				if authClaims, ok := claims.(map[string]interface{}); ok {
					sessionID = authClaims.SessionID
				}
			}
		}

		if sessionID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			return
		}

		token, err := m.GenerateToken(sessionID.(string))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate CSRF token",
			})
			return
		}

		// Set CSRF cookie for browser clients
		c.SetCookie(
			"csrf_token",
			token,
			int(m.ttl.Seconds()),
			"/",
			"",
			true,  // Secure in production
			false, // Not HttpOnly - JavaScript needs access
		)

		c.JSON(http.StatusOK, gin.H{
			"csrf_token": token,
			"expires_in": int(m.ttl.Seconds()),
		})
	}
}

// cleanupExpiredTokens removes expired CSRF tokens
func (m *CSRFMiddleware) cleanupExpiredTokens() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for token, csrfToken := range m.tokens {
			if now.After(csrfToken.expiresAt) {
				delete(m.tokens, token)
			}
		}
		m.mu.Unlock()
	}
}

// DoubleSubmitCookie implements double-submit cookie CSRF protection
func (m *CSRFMiddleware) DoubleSubmitCookie() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF for safe methods
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Get token from header
		headerToken := c.GetHeader("X-CSRF-Token")
		if headerToken == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "CSRF token header required",
			})
			c.Abort()
			return
		}

		// Get token from cookie
		cookieToken, err := c.Cookie("csrf_token")
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "CSRF token cookie required",
			})
			c.Abort()
			return
		}

		// Compare tokens (constant time comparison)
		if !compareSecure(headerToken, cookieToken) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "CSRF token mismatch",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
// compareSecure performs a constant-time comparison of two strings
func compareSecure(a, b string) bool {
	return hmac.Equal([]byte(a), []byte(b))
}
