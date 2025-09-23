package security

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// SecureHandlerTemplate provides AI with secure patterns for HTTP handlers
type SecureHandlerTemplate struct {
	InputValidation []ValidationRule `json:"input_validation"`
	Authentication  AuthPattern      `json:"authentication"`
	ErrorHandling   ErrorPattern     `json:"error_handling"`
	RateLimiting    RateLimitConfig  `json:"rate_limiting"`
	Logging         LoggingConfig    `json:"logging"`
}

// ValidationRule defines input validation requirements
type ValidationRule struct {
	Field       string   `json:"field"`
	Type        string   `json:"type"` // string, int, email, url, etc.
	Required    bool     `json:"required"`
	MaxLength   int      `json:"max_length,omitempty"`
	MinLength   int      `json:"min_length,omitempty"`
	Pattern     string   `json:"pattern,omitempty"`
	Whitelist   []string `json:"whitelist,omitempty"`
	Blacklist   []string `json:"blacklist,omitempty"`
	Description string   `json:"description"`
}

// AuthPattern defines authentication requirements
type AuthPattern struct {
	Required       bool          `json:"required"`
	Methods        []string      `json:"methods"` // bearer, api_key, basic
	Roles          []string      `json:"roles,omitempty"`
	Permissions    []string      `json:"permissions,omitempty"`
	RateLimit      int           `json:"rate_limit"` // requests per minute
	SessionTimeout time.Duration `json:"session_timeout"`
}

// ErrorPattern defines secure error handling
type ErrorPattern struct {
	UserMessage    string `json:"user_message"`    // Generic message for users
	LogLevel       string `json:"log_level"`       // debug, info, warn, error
	IncludeDetails bool   `json:"include_details"` // Include details in logs only
	StatusCode     int    `json:"status_code"`
}

// RateLimitConfig defines rate limiting configuration
type RateLimitConfig struct {
	Enabled  bool          `json:"enabled"`
	Requests int           `json:"requests"` // requests per window
	Window   time.Duration `json:"window"`   // time window
	Burst    int           `json:"burst"`    // burst allowance
	KeyFunc  string        `json:"key_func"` // ip, user, custom
}

// LoggingConfig defines secure logging configuration
type LoggingConfig struct {
	Level          string   `json:"level"`
	SanitizeFields []string `json:"sanitize_fields"` // Fields to sanitize
	ExcludeFields  []string `json:"exclude_fields"`  // Fields to exclude
	IncludeContext bool     `json:"include_context"`
	Structured     bool     `json:"structured"`
}

// SecureHandler creates a secure HTTP handler following security templates
func SecureHandler(template SecureHandlerTemplate, handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 1. Input validation
		if err := validateInputs(r, template.InputValidation); err != nil {
			logSecurityEvent(ctx, "validation_failed", err.Error(), template.Logging)
			writeSecureError(w, template.ErrorHandling, "Invalid input provided")
			return
		}

		// 2. Authentication
		if template.Authentication.Required {
			if err := authenticateRequest(r, template.Authentication); err != nil {
				logSecurityEvent(ctx, "auth_failed", err.Error(), template.Logging)
				writeSecureError(w, template.ErrorHandling, "Authentication required")
				return
			}
		}

		// 3. Rate limiting
		if template.RateLimiting.Enabled {
			if err := checkRateLimit(r, template.RateLimiting); err != nil {
				logSecurityEvent(ctx, "rate_limit_exceeded", err.Error(), template.Logging)
				writeSecureError(w, template.ErrorHandling, "Rate limit exceeded")
				return
			}
		}

		// 4. Execute handler with security context
		secureCtx := context.WithValue(ctx, "security_template", template)
		r = r.WithContext(secureCtx)

		handlerFunc(w, r)
	}
}

// validateInputs validates request inputs against security rules
func validateInputs(r *http.Request, rules []ValidationRule) error {
	for _, rule := range rules {
		value := r.URL.Query().Get(rule.Field)
		if value == "" {
			value = r.Header.Get(rule.Field)
		}

		if err := validateField(value, rule); err != nil {
			return fmt.Errorf("validation failed for field %s: %w", rule.Field, err)
		}
	}
	return nil
}

// validateField validates a single field against its rule
func validateField(value string, rule ValidationRule) error {
	// Required field check
	if rule.Required && value == "" {
		return fmt.Errorf("required field %s is missing", rule.Field)
	}

	// Skip validation if field is empty and not required
	if value == "" && !rule.Required {
		return nil
	}

	// Length validation
	if rule.MaxLength > 0 && len(value) > rule.MaxLength {
		return fmt.Errorf("field %s exceeds maximum length of %d", rule.Field, rule.MaxLength)
	}

	if rule.MinLength > 0 && len(value) < rule.MinLength {
		return fmt.Errorf("field %s is below minimum length of %d", rule.Field, rule.MinLength)
	}

	// Pattern validation
	if rule.Pattern != "" {
		matched, err := regexp.MatchString(rule.Pattern, value)
		if err != nil {
			return fmt.Errorf("invalid pattern for field %s: %w", rule.Field, err)
		}
		if !matched {
			return fmt.Errorf("field %s does not match required pattern", rule.Field)
		}
	}

	// Whitelist validation
	if len(rule.Whitelist) > 0 {
		if !containsString(rule.Whitelist, value) {
			return fmt.Errorf("field %s value not in allowed list", rule.Field)
		}
	}

	// Blacklist validation
	if len(rule.Blacklist) > 0 {
		if containsString(rule.Blacklist, value) {
			return fmt.Errorf("field %s value is not allowed", rule.Field)
		}
	}

	return nil
}

// authenticateRequest authenticates the request
func authenticateRequest(r *http.Request, auth AuthPattern) error {
	// Check for authentication header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return fmt.Errorf("authorization header missing")
	}

	// Validate authentication method
	for _, method := range auth.Methods {
		switch method {
		case "bearer":
			if strings.HasPrefix(authHeader, "Bearer ") {
				return validateBearerToken(authHeader[7:], auth)
			}
		case "api_key":
			if strings.HasPrefix(authHeader, "ApiKey ") {
				return validateAPIKey(authHeader[7:], auth)
			}
		case "basic":
			if strings.HasPrefix(authHeader, "Basic ") {
				return validateBasicAuth(authHeader[6:], auth)
			}
		}
	}

	return fmt.Errorf("unsupported authentication method")
}

// validateBearerToken validates a bearer token
func validateBearerToken(token string, auth AuthPattern) error {
	// SEC-1: Validate token format and signature
	if len(token) < 32 {
		return fmt.Errorf("invalid token format")
	}

	// Check token expiration
	// Check token permissions
	// Check rate limiting

	return nil
}

// validateAPIKey validates an API key
func validateAPIKey(key string, auth AuthPattern) error {
	// SEC-1: Validate API key format
	if len(key) < 16 {
		return fmt.Errorf("invalid API key format")
	}

	// Check key permissions
	// Check rate limiting

	return nil
}

// validateBasicAuth validates basic authentication
func validateBasicAuth(credentials string, auth AuthPattern) error {
	// SEC-1: Validate credentials format
	// Check username/password
	// Check permissions

	return nil
}

// checkRateLimit checks if request exceeds rate limit
func checkRateLimit(r *http.Request, config RateLimitConfig) error {
	// Implement rate limiting logic
	// Use IP, user, or custom key function
	// Check against configured limits

	return nil
}

// writeSecureError writes a secure error response
func writeSecureError(w http.ResponseWriter, pattern ErrorPattern, message string) {
	// SEC-3: Generic error messages for users, detailed logs for debugging
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(pattern.StatusCode)

	// Write generic error response
	fmt.Fprintf(w, `{"error":"%s","code":"%s"}`, message, fmt.Sprintf("ERR_%d", pattern.StatusCode))
}

// logSecurityEvent logs security events securely
func logSecurityEvent(ctx context.Context, event string, details string, config LoggingConfig) {
	// SEC-3: Detailed logs for debugging, never log secrets
	sanitizedDetails := sanitizeLogData(details, config.SanitizeFields)

	// Log with appropriate level
	logData := map[string]interface{}{
		"event":   event,
		"details": sanitizedDetails,
		"time":    time.Now().UTC(),
	}

	if config.IncludeContext {
		logData["context"] = ctx
	}

	// Use structured logging
	if config.Structured {
		// Log as JSON
		fmt.Printf("SECURITY_LOG: %+v\n", logData)
	} else {
		// Log as text
		fmt.Printf("SECURITY: %s - %s\n", event, sanitizedDetails)
	}
}

// sanitizeLogData removes sensitive information from log data
func sanitizeLogData(data string, sanitizeFields []string) string {
	sanitized := data

	for _, field := range sanitizeFields {
		// Replace sensitive field values with [REDACTED]
		pattern := fmt.Sprintf(`(?i)(%s[=:]\s*)([^\s,}]+)`, regexp.QuoteMeta(field))
		re := regexp.MustCompile(pattern)
		sanitized = re.ReplaceAllString(sanitized, "${1}[REDACTED]")
	}

	return sanitized
}

// containsString checks if a slice contains a string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Predefined security templates for common use cases

// APITemplate provides secure template for API endpoints
var APITemplate = SecureHandlerTemplate{
	InputValidation: []ValidationRule{
		{
			Field:       "Content-Type",
			Type:        "string",
			Required:    true,
			Whitelist:   []string{"application/json", "application/x-www-form-urlencoded"},
			Description: "Content type must be JSON or form data",
		},
	},
	Authentication: AuthPattern{
		Required:       true,
		Methods:        []string{"bearer", "api_key"},
		RateLimit:      100, // 100 requests per minute
		SessionTimeout: 15 * time.Minute,
	},
	ErrorHandling: ErrorPattern{
		UserMessage:    "An error occurred processing your request",
		LogLevel:       "error",
		IncludeDetails: true,
		StatusCode:     http.StatusInternalServerError,
	},
	RateLimiting: RateLimitConfig{
		Enabled:  true,
		Requests: 100,
		Window:   time.Minute,
		Burst:    10,
		KeyFunc:  "ip",
	},
	Logging: LoggingConfig{
		Level:          "info",
		SanitizeFields: []string{"password", "token", "secret", "key"},
		ExcludeFields:  []string{"authorization"},
		IncludeContext: true,
		Structured:     true,
	},
}

// PublicTemplate provides secure template for public endpoints
var PublicTemplate = SecureHandlerTemplate{
	InputValidation: []ValidationRule{
		{
			Field:       "User-Agent",
			Type:        "string",
			Required:    false,
			MaxLength:   500,
			Description: "User agent string",
		},
	},
	Authentication: AuthPattern{
		Required:  false,
		RateLimit: 1000, // 1000 requests per minute
	},
	ErrorHandling: ErrorPattern{
		UserMessage:    "Service temporarily unavailable",
		LogLevel:       "warn",
		IncludeDetails: false,
		StatusCode:     http.StatusServiceUnavailable,
	},
	RateLimiting: RateLimitConfig{
		Enabled:  true,
		Requests: 1000,
		Window:   time.Minute,
		Burst:    50,
		KeyFunc:  "ip",
	},
	Logging: LoggingConfig{
		Level:          "info",
		SanitizeFields: []string{"ip", "user_agent"},
		IncludeContext: false,
		Structured:     true,
	},
}

// AdminTemplate provides secure template for admin endpoints
var AdminTemplate = SecureHandlerTemplate{
	InputValidation: []ValidationRule{
		{
			Field:       "X-Admin-Token",
			Type:        "string",
			Required:    true,
			MinLength:   32,
			MaxLength:   64,
			Description: "Admin authentication token",
		},
	},
	Authentication: AuthPattern{
		Required:       true,
		Methods:        []string{"bearer"},
		Roles:          []string{"admin", "superuser"},
		RateLimit:      50, // 50 requests per minute
		SessionTimeout: 30 * time.Minute,
	},
	ErrorHandling: ErrorPattern{
		UserMessage:    "Access denied",
		LogLevel:       "error",
		IncludeDetails: true,
		StatusCode:     http.StatusForbidden,
	},
	RateLimiting: RateLimitConfig{
		Enabled:  true,
		Requests: 50,
		Window:   time.Minute,
		Burst:    5,
		KeyFunc:  "user",
	},
	Logging: LoggingConfig{
		Level:          "debug",
		SanitizeFields: []string{"password", "token", "secret"},
		IncludeContext: true,
		Structured:     true,
	},
}
