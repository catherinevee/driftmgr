package validation

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/catherinevee/driftmgr/internal/utils/errors"
	"github.com/catherinevee/driftmgr/internal/observability/logging"
	"github.com/rs/zerolog"
)

// Validator provides input validation and sanitization
type Validator struct {
	logger *zerolog.Logger
	config *Config
}

// Config for validation rules
type Config struct {
	// MaxStringLength is the maximum allowed string length
	MaxStringLength int `json:"max_string_length"`
	
	// MaxJSONDepth is the maximum allowed JSON nesting depth
	MaxJSONDepth int `json:"max_json_depth"`
	
	// MaxArrayLength is the maximum allowed array length
	MaxArrayLength int `json:"max_array_length"`
	
	// AllowHTML determines if HTML tags are allowed
	AllowHTML bool `json:"allow_html"`
	
	// AllowScripts determines if script tags are allowed
	AllowScripts bool `json:"allow_scripts"`
	
	// StrictMode enables strict validation rules
	StrictMode bool `json:"strict_mode"`
	
	// CustomPatterns for additional validation
	CustomPatterns map[string]string `json:"custom_patterns"`
}

// DefaultConfig returns default validation configuration
func DefaultConfig() *Config {
	return &Config{
		MaxStringLength: 10000,
		MaxJSONDepth:    10,
		MaxArrayLength:  1000,
		AllowHTML:       false,
		AllowScripts:    false,
		StrictMode:      true,
		CustomPatterns:  make(map[string]string),
	}
}

// Common regex patterns for validation
var (
	emailRegex     = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	urlRegex       = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	alphaNumRegex  = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	phoneRegex     = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	ipv4Regex      = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	ipv6Regex      = regexp.MustCompile(`^(([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|::|::[0-9a-fA-F]{1,4})$`)
	uuidRegex      = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	awsArnRegex    = regexp.MustCompile(`^arn:aws:[a-zA-Z0-9\-]+:[a-zA-Z0-9\-]*:[0-9]{12}:.+$`)
	azureResIDRegex = regexp.MustCompile(`^/subscriptions/[0-9a-fA-F\-]+/resourceGroups/[^/]+/providers/.+$`)
	gcpResIDRegex  = regexp.MustCompile(`^projects/[a-z][a-z0-9\-]+/.*$`)
	
	// Security patterns
	sqlInjectionRegex = regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute|script|javascript|eval|setTimeout|setInterval)`)
	xssRegex         = regexp.MustCompile(`(?i)(<script|<iframe|javascript:|onerror=|onclick=|<img\s+src.*=)`)
	pathTraversalRegex = regexp.MustCompile(`\.\.[\\/]`)
	cmdInjectionRegex = regexp.MustCompile(`[;&|<>$\` + "`" + `]`)
)

// NewValidator creates a new validator
func NewValidator(config *Config) *Validator {
	if config == nil {
		config = DefaultConfig()
	}
	
	logger := logging.WithComponent("validator")
	
	return &Validator{
		logger: &logger,
		config: config,
	}
}

// ValidateString validates and sanitizes a string input
func (v *Validator) ValidateString(input string, fieldName string) (string, error) {
	// Check if string is valid UTF-8
	if !utf8.ValidString(input) {
		return "", errors.ValidationError(fmt.Sprintf("%s contains invalid UTF-8", fieldName))
	}
	
	// Check length
	if len(input) > v.config.MaxStringLength {
		return "", errors.ValidationError(fmt.Sprintf("%s exceeds maximum length of %d", fieldName, v.config.MaxStringLength))
	}
	
	// Sanitize the string
	sanitized := v.sanitizeString(input)
	
	// Check for dangerous patterns if in strict mode
	if v.config.StrictMode {
		if err := v.checkDangerousPatterns(sanitized, fieldName); err != nil {
			return "", err
		}
	}
	
	return sanitized, nil
}

// ValidateEmail validates an email address
func (v *Validator) ValidateEmail(email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	
	if !emailRegex.MatchString(email) {
		return errors.ValidationError("invalid email format")
	}
	
	// Additional checks
	if strings.Count(email, "@") != 1 {
		return errors.ValidationError("email must contain exactly one @ symbol")
	}
	
	parts := strings.Split(email, "@")
	if len(parts[0]) < 1 || len(parts[1]) < 3 {
		return errors.ValidationError("invalid email format")
	}
	
	return nil
}

// ValidateURL validates a URL
func (v *Validator) ValidateURL(urlStr string) error {
	urlStr = strings.TrimSpace(urlStr)
	
	// Parse URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return errors.ValidationError("invalid URL format")
	}
	
	// Check scheme
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.ValidationError("URL must use http or https scheme")
	}
	
	// Check host
	if u.Host == "" {
		return errors.ValidationError("URL must have a valid host")
	}
	
	// Check for localhost/private IPs in production
	if v.config.StrictMode {
		if strings.Contains(u.Host, "localhost") || strings.Contains(u.Host, "127.0.0.1") {
			return errors.ValidationError("localhost URLs not allowed")
		}
	}
	
	return nil
}

// ValidateIP validates an IP address
func (v *Validator) ValidateIP(ip string) error {
	ip = strings.TrimSpace(ip)
	
	// Check if valid IP
	if net.ParseIP(ip) == nil {
		return errors.ValidationError("invalid IP address")
	}
	
	// Check for private/reserved IPs in strict mode
	if v.config.StrictMode {
		parsedIP := net.ParseIP(ip)
		if parsedIP.IsLoopback() || parsedIP.IsPrivate() || parsedIP.IsMulticast() {
			return errors.ValidationError("private/reserved IP addresses not allowed")
		}
	}
	
	return nil
}

// ValidateJSON validates JSON input
func (v *Validator) ValidateJSON(jsonStr string) (map[string]interface{}, error) {
	var result map[string]interface{}
	
	// Check JSON size
	if len(jsonStr) > v.config.MaxStringLength {
		return nil, errors.ValidationError("JSON exceeds maximum size")
	}
	
	// Parse JSON
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, errors.ValidationError("invalid JSON format")
	}
	
	// Check nesting depth
	if depth := v.getJSONDepth(result); depth > v.config.MaxJSONDepth {
		return nil, errors.ValidationError(fmt.Sprintf("JSON nesting depth %d exceeds maximum %d", depth, v.config.MaxJSONDepth))
	}
	
	// Sanitize all string values
	v.sanitizeJSONValues(result)
	
	return result, nil
}

// ValidateCloudResourceID validates cloud resource identifiers
func (v *Validator) ValidateCloudResourceID(resourceID string, provider string) error {
	resourceID = strings.TrimSpace(resourceID)
	
	switch strings.ToLower(provider) {
	case "aws":
		// Check for ARN format or resource ID
		if strings.HasPrefix(resourceID, "arn:") {
			if !awsArnRegex.MatchString(resourceID) {
				return errors.ValidationError("invalid AWS ARN format")
			}
		} else if !alphaNumRegex.MatchString(strings.ReplaceAll(resourceID, "-", "")) {
			return errors.ValidationError("invalid AWS resource ID")
		}
		
	case "azure":
		if !strings.HasPrefix(resourceID, "/subscriptions/") {
			return errors.ValidationError("Azure resource ID must start with /subscriptions/")
		}
		if !azureResIDRegex.MatchString(resourceID) {
			return errors.ValidationError("invalid Azure resource ID format")
		}
		
	case "gcp":
		if !strings.HasPrefix(resourceID, "projects/") {
			return errors.ValidationError("GCP resource ID must start with projects/")
		}
		if !gcpResIDRegex.MatchString(resourceID) {
			return errors.ValidationError("invalid GCP resource ID format")
		}
		
	default:
		// Generic validation for other providers
		if len(resourceID) < 1 || len(resourceID) > 500 {
			return errors.ValidationError("resource ID length invalid")
		}
	}
	
	return nil
}

// ValidateFilePath validates a file path
func (v *Validator) ValidateFilePath(path string) error {
	path = strings.TrimSpace(path)
	
	// Check for path traversal
	if pathTraversalRegex.MatchString(path) {
		return errors.ValidationError("path traversal detected")
	}
	
	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return errors.ValidationError("null bytes not allowed in path")
	}
	
	// Check length
	if len(path) > 4096 {
		return errors.ValidationError("path too long")
	}
	
	// Check for dangerous characters
	if v.config.StrictMode {
		if cmdInjectionRegex.MatchString(path) {
			return errors.ValidationError("potentially dangerous characters in path")
		}
	}
	
	return nil
}

// ValidateCommand validates a command string
func (v *Validator) ValidateCommand(cmd string) error {
	cmd = strings.TrimSpace(cmd)
	
	// Check for command injection patterns
	if cmdInjectionRegex.MatchString(cmd) {
		return errors.ValidationError("potentially dangerous command characters detected")
	}
	
	// Check for known dangerous commands
	dangerousCommands := []string{"rm -rf", "format", "del /f", "drop table", "delete from"}
	lowerCmd := strings.ToLower(cmd)
	for _, dangerous := range dangerousCommands {
		if strings.Contains(lowerCmd, dangerous) {
			return errors.ValidationError("potentially dangerous command detected")
		}
	}
	
	return nil
}

// sanitizeString removes or escapes dangerous characters
func (v *Validator) sanitizeString(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	
	// Trim whitespace
	input = strings.TrimSpace(input)
	
	// Remove non-printable characters
	var builder strings.Builder
	for _, r := range input {
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			builder.WriteRune(r)
		}
	}
	input = builder.String()
	
	// HTML escape if needed
	if !v.config.AllowHTML {
		input = escapeHTML(input)
	}
	
	// Remove script tags if not allowed
	if !v.config.AllowScripts {
		input = xssRegex.ReplaceAllString(input, "")
	}
	
	return input
}

// checkDangerousPatterns checks for dangerous patterns
func (v *Validator) checkDangerousPatterns(input string, fieldName string) error {
	// Check for SQL injection
	if sqlInjectionRegex.MatchString(input) {
		v.logger.Warn().
			Str("field", fieldName).
			Msg("potential SQL injection detected")
		return errors.ValidationError("potentially dangerous SQL pattern detected")
	}
	
	// Check for XSS
	if xssRegex.MatchString(input) {
		v.logger.Warn().
			Str("field", fieldName).
			Msg("potential XSS detected")
		return errors.ValidationError("potentially dangerous script pattern detected")
	}
	
	return nil
}

// getJSONDepth calculates the maximum nesting depth of JSON
func (v *Validator) getJSONDepth(data interface{}) int {
	switch dt := data.(type) {
	case map[string]interface{}:
		maxDepth := 0
		for _, value := range dt {
			depth := v.getJSONDepth(value)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
		return maxDepth + 1
		
	case []interface{}:
		maxDepth := 0
		for _, item := range dt {
			depth := v.getJSONDepth(item)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
		return maxDepth + 1
		
	default:
		return 0
	}
}

// sanitizeJSONValues recursively sanitizes all string values in JSON
func (v *Validator) sanitizeJSONValues(data map[string]interface{}) {
	for key, value := range data {
		switch vt := value.(type) {
		case string:
			data[key] = v.sanitizeString(vt)
		case map[string]interface{}:
			v.sanitizeJSONValues(vt)
		case []interface{}:
			for i, item := range vt {
				if str, ok := item.(string); ok {
					vt[i] = v.sanitizeString(str)
				} else if m, ok := item.(map[string]interface{}); ok {
					v.sanitizeJSONValues(m)
				}
			}
		}
	}
}

// escapeHTML escapes HTML special characters
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// ValidateStruct validates a struct using tags
func (v *Validator) ValidateStruct(s interface{}) error {
	// This would use reflection to validate struct fields based on tags
	// For now, return nil
	return nil
}

// SanitizeStruct sanitizes all string fields in a struct
func (v *Validator) SanitizeStruct(s interface{}) error {
	// This would use reflection to sanitize struct fields
	// For now, return nil
	return nil
}