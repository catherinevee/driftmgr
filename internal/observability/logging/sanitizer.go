package logging

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/rs/zerolog"
)

// SensitiveFieldPatterns contains patterns for sensitive field names
var SensitiveFieldPatterns = []string{
	"password", "passwd", "pwd",
	"secret", "token", "key",
	"api_key", "apikey", "api-key",
	"access_key", "accesskey", "access-key",
	"secret_key", "secretkey", "secret-key",
	"private_key", "privatekey", "private-key",
	"credential", "cred",
	"authorization", "auth", "bearer",
	"session", "cookie",
	"ssn", "social_security",
	"credit_card", "card_number", "cvv", "cvc",
	"pin", "tax_id", "license",
	"encryption_key", "signing_key",
	"refresh_token", "oauth", "jwt",
	"x-api-key", "x-auth-token",
	"aws_access_key_id", "aws_secret_access_key", "aws_session_token",
	"azure_client_id", "azure_client_secret", "azure_tenant_id",
	"gcp_service_account", "google_application_credentials",
	"digitalocean_token", "do_api_token",
	"database_url", "connection_string", "conn_str",
	"redis_password", "mongodb_uri",
}

// SensitiveValuePatterns contains regex patterns for sensitive values
var SensitiveValuePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^AKIA[0-9A-Z]{16}$`),                     // AWS Access Key
	regexp.MustCompile(`(?i)^[0-9a-zA-Z/+=]{40}$`),                   // AWS Secret Key pattern
	regexp.MustCompile(`(?i)^sk-[a-zA-Z0-9]{48}$`),                   // OpenAI API key
	regexp.MustCompile(`(?i)^ghp_[a-zA-Z0-9]{36}$`),                  // GitHub personal access token
	regexp.MustCompile(`(?i)^gho_[a-zA-Z0-9]{36}$`),                  // GitHub OAuth token
	regexp.MustCompile(`(?i)^glpat-[a-zA-Z0-9\-_]{20,}$`),           // GitLab personal access token
	regexp.MustCompile(`(?i)^sq0[a-z]{3}-[a-zA-Z0-9\-_]{22,}$`),     // Square API token
	regexp.MustCompile(`(?i)^rzp_[a-zA-Z0-9]{14,}$`),                // Razorpay API key
	regexp.MustCompile(`(?i)^Bearer\s+[a-zA-Z0-9\-._~+/]+=*$`),      // Bearer token
	regexp.MustCompile(`(?i)^Basic\s+[a-zA-Z0-9+/]+=*$`),            // Basic auth
	regexp.MustCompile(`(?i)^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`), // UUID that might be secret
	regexp.MustCompile(`(?i)^\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}$`),                       // Credit card number
	regexp.MustCompile(`(?i)^\d{3}-\d{2}-\d{4}$`),                                            // SSN
	regexp.MustCompile(`(?i)^mongodb(\+srv)?://[^@]+:[^@]+@.+$`),                             // MongoDB connection string
	regexp.MustCompile(`(?i)^(postgres|postgresql|mysql)://[^@]+:[^@]+@.+$`),                 // Database connection string
	regexp.MustCompile(`(?i)^redis://[^@]+:[^@]+@.+$`),                                       // Redis connection string
}

// SanitizingHook is a zerolog hook that sanitizes sensitive data
type SanitizingHook struct {
	fieldPatterns []*regexp.Regexp
	valuePatterns []*regexp.Regexp
}

// NewSanitizingHook creates a new sanitizing hook
func NewSanitizingHook() *SanitizingHook {
	fieldPatterns := make([]*regexp.Regexp, len(SensitiveFieldPatterns))
	for i, pattern := range SensitiveFieldPatterns {
		fieldPatterns[i] = regexp.MustCompile(fmt.Sprintf("(?i)%s", pattern))
	}

	return &SanitizingHook{
		fieldPatterns: fieldPatterns,
		valuePatterns: SensitiveValuePatterns,
	}
}

// Run implements zerolog.Hook interface
func (h *SanitizingHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	// The event is already constructed, we can't modify it
	// Instead, we'll provide utility functions for sanitization
}

// SanitizeValue sanitizes a single value if it appears sensitive
func SanitizeValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case string:
		return sanitizeString(v)
	case map[string]interface{}:
		return sanitizeMap(v)
	case []interface{}:
		return sanitizeSlice(v)
	case map[string]string:
		return sanitizeStringMap(v)
	default:
		// Check if it's a struct using reflection
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Struct {
			return sanitizeStruct(rv)
		}
		return value
	}
}

// sanitizeString sanitizes a string value
func sanitizeString(s string) string {
	// Check if the string matches any sensitive patterns
	for _, pattern := range SensitiveValuePatterns {
		if pattern.MatchString(s) {
			return redactString(s)
		}
	}

	// Check for inline credentials in URLs
	if strings.Contains(s, "://") && strings.Contains(s, "@") {
		return sanitizeURL(s)
	}

	// Check for JSON with potential sensitive data
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(s), &data); err == nil {
			sanitized := sanitizeMap(data)
			if result, err := json.Marshal(sanitized); err == nil {
				return string(result)
			}
		}
	}

	return s
}

// sanitizeMap sanitizes a map
func sanitizeMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	
	for key, value := range m {
		// Check if the key is sensitive
		if isSensitiveField(key) {
			result[key] = "***REDACTED***"
		} else {
			// Recursively sanitize the value
			result[key] = SanitizeValue(value)
		}
	}
	
	return result
}

// sanitizeStringMap sanitizes a string map
func sanitizeStringMap(m map[string]string) map[string]string {
	result := make(map[string]string, len(m))
	
	for key, value := range m {
		if isSensitiveField(key) {
			result[key] = "***REDACTED***"
		} else {
			result[key] = sanitizeString(value)
		}
	}
	
	return result
}

// sanitizeSlice sanitizes a slice
func sanitizeSlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))
	
	for i, value := range s {
		result[i] = SanitizeValue(value)
	}
	
	return result
}

// sanitizeStruct sanitizes a struct using reflection
func sanitizeStruct(v reflect.Value) interface{} {
	// Convert struct to map for sanitization
	result := make(map[string]interface{})
	t := v.Type()
	
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		
		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}
		
		fieldName := field.Name
		// Check JSON tag if present
		if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				fieldName = parts[0]
			}
		}
		
		if isSensitiveField(fieldName) {
			result[fieldName] = "***REDACTED***"
		} else {
			result[fieldName] = SanitizeValue(fieldValue.Interface())
		}
	}
	
	return result
}

// isSensitiveField checks if a field name is sensitive
func isSensitiveField(field string) bool {
	lowerField := strings.ToLower(field)
	
	for _, pattern := range SensitiveFieldPatterns {
		if strings.Contains(lowerField, pattern) {
			return true
		}
	}
	
	return false
}

// redactString redacts a sensitive string
func redactString(s string) string {
	if len(s) <= 8 {
		return "***REDACTED***"
	}
	
	// Show first 3 and last 3 characters for debugging
	return fmt.Sprintf("%s...%s", s[:3], s[len(s)-3:])
}

// sanitizeURL sanitizes credentials in URLs
func sanitizeURL(url string) string {
	// Find the protocol
	protocolEnd := strings.Index(url, "://")
	if protocolEnd == -1 {
		return url
	}
	
	protocol := url[:protocolEnd+3]
	rest := url[protocolEnd+3:]
	
	// Find @ symbol indicating credentials
	atIndex := strings.Index(rest, "@")
	if atIndex == -1 {
		return url
	}
	
	// Extract host part after @
	hostPart := rest[atIndex+1:]
	
	// Reconstruct URL with redacted credentials
	return protocol + "***REDACTED***@" + hostPart
}

// SafeLogger wraps zerolog.Logger with automatic sanitization
type SafeLogger struct {
	zerolog.Logger
}

// NewSafeLogger creates a logger that automatically sanitizes sensitive data
func NewSafeLogger(logger zerolog.Logger) *SafeLogger {
	return &SafeLogger{Logger: logger}
}

// WithField adds a field with automatic sanitization
func (l *SafeLogger) WithField(key string, value interface{}) *SafeLogger {
	var sanitizedValue interface{}
	
	if isSensitiveField(key) {
		sanitizedValue = "***REDACTED***"
	} else {
		sanitizedValue = SanitizeValue(value)
	}
	
	return &SafeLogger{
		Logger: l.Logger.With().Interface(key, sanitizedValue).Logger(),
	}
}

// WithFields adds multiple fields with automatic sanitization
func (l *SafeLogger) WithFields(fields map[string]interface{}) *SafeLogger {
	logger := l.Logger.With()
	
	for key, value := range fields {
		var sanitizedValue interface{}
		if isSensitiveField(key) {
			sanitizedValue = "***REDACTED***"
		} else {
			sanitizedValue = SanitizeValue(value)
		}
		logger = logger.Interface(key, sanitizedValue)
	}
	
	return &SafeLogger{Logger: logger.Logger()}
}

// Info logs an info message with sanitization
func (l *SafeLogger) Info(msg string, fields ...map[string]interface{}) {
	event := l.Logger.Info()
	
	if len(fields) > 0 {
		for key, value := range fields[0] {
			var sanitizedValue interface{}
			if isSensitiveField(key) {
				sanitizedValue = "***REDACTED***"
			} else {
				sanitizedValue = SanitizeValue(value)
			}
			event = event.Interface(key, sanitizedValue)
		}
	}
	
	event.Msg(msg)
}

// Debug logs a debug message with sanitization
func (l *SafeLogger) Debug(msg string, fields ...map[string]interface{}) {
	event := l.Logger.Debug()
	
	if len(fields) > 0 {
		for key, value := range fields[0] {
			var sanitizedValue interface{}
			if isSensitiveField(key) {
				sanitizedValue = "***REDACTED***"
			} else {
				sanitizedValue = SanitizeValue(value)
			}
			event = event.Interface(key, sanitizedValue)
		}
	}
	
	event.Msg(msg)
}

// Warn logs a warning message with sanitization
func (l *SafeLogger) Warn(msg string, fields ...map[string]interface{}) {
	event := l.Logger.Warn()
	
	if len(fields) > 0 {
		for key, value := range fields[0] {
			var sanitizedValue interface{}
			if isSensitiveField(key) {
				sanitizedValue = "***REDACTED***"
			} else {
				sanitizedValue = SanitizeValue(value)
			}
			event = event.Interface(key, sanitizedValue)
		}
	}
	
	event.Msg(msg)
}

// Error logs an error message with sanitization
func (l *SafeLogger) Error(msg string, err error, fields ...map[string]interface{}) {
	event := l.Logger.Error()
	
	if err != nil {
		// Sanitize error message if it contains sensitive data
		errMsg := sanitizeString(err.Error())
		event = event.Str("error", errMsg)
	}
	
	if len(fields) > 0 {
		for key, value := range fields[0] {
			var sanitizedValue interface{}
			if isSensitiveField(key) {
				sanitizedValue = "***REDACTED***"
			} else {
				sanitizedValue = SanitizeValue(value)
			}
			event = event.Interface(key, sanitizedValue)
		}
	}
	
	event.Msg(msg)
}

// GetSafeLogger returns a safe logger instance
func GetSafeLogger() *SafeLogger {
	return NewSafeLogger(Logger)
}