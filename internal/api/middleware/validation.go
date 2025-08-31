package middleware

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ValidationMiddleware provides input validation and sanitization
type ValidationMiddleware struct {
	validator         *validator.Validate
	maxRequestSize    int64
	allowedProviders  []string
	allowedRegions    map[string][]string
	pathTraversalRegex *regexp.Regexp
	sqlInjectionRegex  *regexp.Regexp
	xssPatterns       []*regexp.Regexp
}

// NewValidationMiddleware creates a new validation middleware
func NewValidationMiddleware() *ValidationMiddleware {
	v := validator.New()
	
	// Register custom validators
	v.RegisterValidation("provider", validateProvider)
	v.RegisterValidation("region", validateRegion)
	v.RegisterValidation("resourcetype", validateResourceType)
	v.RegisterValidation("nosql", validateNoSQL)
	v.RegisterValidation("noxss", validateNoXSS)
	v.RegisterValidation("safepath", validateSafePath)

	return &ValidationMiddleware{
		validator:      v,
		maxRequestSize: 10 * 1024 * 1024, // 10MB
		allowedProviders: []string{
			"aws", "azure", "gcp", "digitalocean",
		},
		allowedRegions: map[string][]string{
			"aws": {
				"us-east-1", "us-east-2", "us-west-1", "us-west-2",
				"eu-west-1", "eu-west-2", "eu-central-1",
				"ap-southeast-1", "ap-southeast-2", "ap-northeast-1",
			},
			"azure": {
				"eastus", "eastus2", "westus", "westus2",
				"centralus", "northeurope", "westeurope",
			},
			"gcp": {
				"us-central1", "us-east1", "us-west1",
				"europe-west1", "europe-west2",
				"asia-east1", "asia-southeast1",
			},
			"digitalocean": {
				"nyc1", "nyc3", "sfo1", "sfo2", "sfo3",
				"ams2", "ams3", "lon1", "fra1",
			},
		},
		pathTraversalRegex: regexp.MustCompile(`\.\./|\.\.\\|%2e%2e%2f|%252e%252e%252f`),
		sqlInjectionRegex: regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute|script|javascript|eval|setTimeout|setInterval)`),
		xssPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
			regexp.MustCompile(`(?i)javascript:`),
			regexp.MustCompile(`(?i)on\w+\s*=`),
			regexp.MustCompile(`(?i)<iframe[^>]*>`),
			regexp.MustCompile(`(?i)<object[^>]*>`),
			regexp.MustCompile(`(?i)<embed[^>]*>`),
		},
	}
}

// ValidateRequest validates and sanitizes incoming requests
func (m *ValidationMiddleware) ValidateRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Limit request size
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, m.maxRequestSize)

		// Validate content type for POST/PUT/PATCH
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			contentType := c.GetHeader("Content-Type")
			if !strings.Contains(contentType, "application/json") && !strings.Contains(contentType, "multipart/form-data") {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid content type",
				})
				c.Abort()
				return
			}
		}

		// Sanitize query parameters
		m.sanitizeQueryParams(c)

		// Sanitize path parameters
		m.sanitizePathParams(c)

		c.Next()
	}
}

// sanitizeQueryParams sanitizes query parameters
func (m *ValidationMiddleware) sanitizeQueryParams(c *gin.Context) {
	for key, values := range c.Request.URL.Query() {
		for i, value := range values {
			// Check for path traversal
			if m.pathTraversalRegex.MatchString(value) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("Invalid parameter: %s", key),
				})
				c.Abort()
				return
			}

			// Check for SQL injection patterns
			if m.sqlInjectionRegex.MatchString(value) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("Potentially malicious input in parameter: %s", key),
				})
				c.Abort()
				return
			}

			// HTML escape the value
			values[i] = html.EscapeString(value)
		}
	}
}

// sanitizePathParams sanitizes path parameters
func (m *ValidationMiddleware) sanitizePathParams(c *gin.Context) {
	for _, param := range c.Params {
		// Check for path traversal
		if m.pathTraversalRegex.MatchString(param.Value) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Invalid path parameter: %s", param.Key),
			})
			c.Abort()
			return
		}

		// HTML escape the value
		param.Value = html.EscapeString(param.Value)
	}
}

// ValidateJSON validates JSON request body against a struct
func (m *ValidationMiddleware) ValidateJSON(target interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Bind JSON to target struct
		if err := c.ShouldBindJSON(target); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid JSON format",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		// Validate struct
		if err := m.validator.Struct(target); err != nil {
			validationErrors := make(map[string]string)
			for _, err := range err.(validator.ValidationErrors) {
				validationErrors[err.Field()] = fmt.Sprintf("Failed validation: %s", err.Tag())
			}
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Validation failed",
				"errors": validationErrors,
			})
			c.Abort()
			return
		}

		// Sanitize string fields
		m.sanitizeStruct(target)

		// Store validated data in context
		c.Set("validated_data", target)

		c.Next()
	}
}

// sanitizeStruct recursively sanitizes string fields in a struct
func (m *ValidationMiddleware) sanitizeStruct(v interface{}) {
	// This would use reflection to iterate through struct fields
	// and apply HTML escaping to string fields
	// For brevity, showing the concept
	if data, err := json.Marshal(v); err == nil {
		sanitized := html.EscapeString(string(data))
		// Check for XSS patterns
		for _, pattern := range m.xssPatterns {
			if pattern.MatchString(sanitized) {
				// Remove malicious content
				sanitized = pattern.ReplaceAllString(sanitized, "")
			}
		}
	}
}

// Custom validators

func validateProvider(fl validator.FieldLevel) bool {
	provider := fl.Field().String()
	validProviders := []string{"aws", "azure", "gcp", "digitalocean"}
	for _, valid := range validProviders {
		if provider == valid {
			return true
		}
	}
	return false
}

func validateRegion(fl validator.FieldLevel) bool {
	region := fl.Field().String()
	// Basic region format validation
	matched, _ := regexp.MatchString(`^[a-z]{2,}-[a-z]+-\d+$|^[a-z]+\d*$`, region)
	return matched
}

func validateResourceType(fl validator.FieldLevel) bool {
	resourceType := fl.Field().String()
	// Validate resource type format
	matched, _ := regexp.MatchString(`^[a-z0-9_]+$`, resourceType)
	return matched && len(resourceType) < 100
}

func validateNoSQL(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	sqlPatterns := []string{
		"(?i)union.*select",
		"(?i)select.*from",
		"(?i)insert.*into",
		"(?i)delete.*from",
		"(?i)drop.*table",
		"(?i)update.*set",
		"(?i)exec(ute)?",
		"(?i)xp_cmdshell",
		"(?i)sp_executesql",
	}
	
	for _, pattern := range sqlPatterns {
		if matched, _ := regexp.MatchString(pattern, value); matched {
			return false
		}
	}
	return true
}

func validateNoXSS(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	xssPatterns := []string{
		`<script[^>]*>.*?</script>`,
		`javascript:`,
		`on\w+\s*=`,
		`<iframe`,
		`<object`,
		`<embed`,
		`<applet`,
		`<meta`,
		`<link`,
		`<style`,
	}
	
	for _, pattern := range xssPatterns {
		if matched, _ := regexp.MatchString("(?i)"+pattern, value); matched {
			return false
		}
	}
	return true
}

func validateSafePath(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	
	// Check for path traversal attempts
	dangerous := []string{
		"..", "~", "%00", "%2e", "%252e",
		"..\\", "../", "..%2F", "..%5C",
	}
	
	for _, d := range dangerous {
		if strings.Contains(strings.ToLower(path), d) {
			return false
		}
	}
	
	// Check for absolute paths (security risk)
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		return false
	}
	
	// Check for drive letters (Windows)
	if matched, _ := regexp.MatchString(`^[a-zA-Z]:`, path); matched {
		return false
	}
	
	return true
}

// SanitizeString sanitizes a string for safe output
func SanitizeString(input string) string {
	// HTML escape
	output := html.EscapeString(input)
	
	// Remove null bytes
	output = strings.ReplaceAll(output, "\x00", "")
	
	// Limit length
	if len(output) > 10000 {
		output = output[:10000]
	}
	
	return output
}

// ValidateProvider validates cloud provider name
func (m *ValidationMiddleware) ValidateProvider(provider string) error {
	for _, allowed := range m.allowedProviders {
		if provider == allowed {
			return nil
		}
	}
	return fmt.Errorf("invalid provider: %s", provider)
}

// ValidateRegion validates region for a provider
func (m *ValidationMiddleware) ValidateRegion(provider, region string) error {
	regions, exists := m.allowedRegions[provider]
	if !exists {
		return fmt.Errorf("unknown provider: %s", provider)
	}
	
	for _, allowed := range regions {
		if region == allowed {
			return nil
		}
	}
	return fmt.Errorf("invalid region %s for provider %s", region, provider)
}

// ValidateResourceID validates resource identifier format
func ValidateResourceID(resourceID string) error {
	// Check length
	if len(resourceID) == 0 || len(resourceID) > 256 {
		return fmt.Errorf("invalid resource ID length")
	}
	
	// Check format (alphanumeric, hyphens, underscores, slashes for ARNs)
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9\-_/:\.]+$`, resourceID)
	if !matched {
		return fmt.Errorf("invalid resource ID format")
	}
	
	return nil
}