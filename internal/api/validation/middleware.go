package validation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/catherinevee/driftmgr/internal/observability/logging"
	"github.com/catherinevee/driftmgr/internal/security/validation"
	"github.com/catherinevee/driftmgr/internal/utils/errors"
	"github.com/rs/zerolog"
)

// RequestValidator validates HTTP requests
type RequestValidator struct {
	validator *validation.Validator
	logger    *zerolog.Logger
	config    *Config
}

// Config for request validation
type Config struct {
	// MaxBodySize is the maximum request body size in bytes
	MaxBodySize int64 `json:"max_body_size"`

	// MaxHeaderSize is the maximum header size
	MaxHeaderSize int `json:"max_header_size"`

	// AllowedMethods is the list of allowed HTTP methods
	AllowedMethods []string `json:"allowed_methods"`

	// AllowedContentTypes is the list of allowed content types
	AllowedContentTypes []string `json:"allowed_content_types"`

	// RequireContentType determines if Content-Type header is required
	RequireContentType bool `json:"require_content_type"`

	// ValidateHeaders determines if headers should be validated
	ValidateHeaders bool `json:"validate_headers"`

	// ValidateQueryParams determines if query parameters should be validated
	ValidateQueryParams bool `json:"validate_query_params"`
}

// DefaultConfig returns default request validation configuration
func DefaultConfig() *Config {
	return &Config{
		MaxBodySize:         10 * 1024 * 1024, // 10MB
		MaxHeaderSize:       8192,
		AllowedMethods:      []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
		AllowedContentTypes: []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		RequireContentType:  true,
		ValidateHeaders:     true,
		ValidateQueryParams: true,
	}
}

// NewRequestValidator creates a new request validator
func NewRequestValidator(config *Config) *RequestValidator {
	if config == nil {
		config = DefaultConfig()
	}

	logger := logging.WithComponent("request-validator")
	validator := validation.NewValidator(nil)

	return &RequestValidator{
		validator: validator,
		logger:    &logger,
		config:    config,
	}
}

// ValidateRequest is middleware that validates incoming requests
func (rv *RequestValidator) ValidateRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validate HTTP method
		if err := rv.validateMethod(r.Method); err != nil {
			rv.sendError(w, http.StatusMethodNotAllowed, err.Error())
			return
		}

		// Validate headers
		if rv.config.ValidateHeaders {
			if err := rv.validateHeaders(r); err != nil {
				rv.sendError(w, http.StatusBadRequest, err.Error())
				return
			}
		}

		// Validate query parameters
		if rv.config.ValidateQueryParams {
			if err := rv.validateQueryParams(r); err != nil {
				rv.sendError(w, http.StatusBadRequest, err.Error())
				return
			}
		}

		// Validate and sanitize request body
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
			body, err := rv.validateBody(r)
			if err != nil {
				rv.sendError(w, http.StatusBadRequest, err.Error())
				return
			}

			// Replace the body for downstream handlers
			r.Body = io.NopCloser(bytes.NewReader(body))
		}

		// Log successful validation
		rv.logger.Debug().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("ip", r.RemoteAddr).
			Msg("request validated")

		// Continue to next handler
		next(w, r)
	}
}

// ValidateJSON validates JSON request body
func (rv *RequestValidator) ValidateJSON(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check Content-Type
		contentType := r.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			rv.sendError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
			return
		}

		// Read body
		body, err := io.ReadAll(io.LimitReader(r.Body, rv.config.MaxBodySize))
		if err != nil {
			rv.sendError(w, http.StatusBadRequest, "failed to read request body")
			return
		}
		defer r.Body.Close()

		// Validate JSON
		validated, err := rv.validator.ValidateJSON(string(body))
		if err != nil {
			rv.sendError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
			return
		}

		// Re-encode sanitized JSON
		sanitized, err := json.Marshal(validated)
		if err != nil {
			rv.sendError(w, http.StatusInternalServerError, "failed to process request")
			return
		}

		// Replace body with sanitized version
		r.Body = io.NopCloser(bytes.NewReader(sanitized))

		next(w, r)
	}
}

// validateMethod validates the HTTP method
func (rv *RequestValidator) validateMethod(method string) error {
	for _, allowed := range rv.config.AllowedMethods {
		if method == allowed {
			return nil
		}
	}
	return errors.ValidationError(fmt.Sprintf("method %s not allowed", method))
}

// validateHeaders validates request headers
func (rv *RequestValidator) validateHeaders(r *http.Request) error {
	// Check total header size
	headerSize := 0
	for key, values := range r.Header {
		headerSize += len(key)
		for _, value := range values {
			headerSize += len(value)
		}
	}

	if headerSize > rv.config.MaxHeaderSize {
		return errors.ValidationError("headers too large")
	}

	// Validate Content-Type if required
	if rv.config.RequireContentType && (r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH") {
		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			return errors.ValidationError("Content-Type header required")
		}

		// Check if content type is allowed
		allowed := false
		for _, ct := range rv.config.AllowedContentTypes {
			if strings.Contains(contentType, ct) {
				allowed = true
				break
			}
		}
		if !allowed {
			return errors.ValidationError(fmt.Sprintf("Content-Type %s not allowed", contentType))
		}
	}

	// Check for dangerous headers
	dangerousHeaders := []string{
		"X-Forwarded-Host",
		"X-Original-URL",
		"X-Rewrite-URL",
	}

	for _, header := range dangerousHeaders {
		if r.Header.Get(header) != "" {
			rv.logger.Warn().
				Str("header", header).
				Str("ip", r.RemoteAddr).
				Msg("potentially dangerous header detected")
		}
	}

	return nil
}

// validateQueryParams validates query parameters
func (rv *RequestValidator) validateQueryParams(r *http.Request) error {
	for key, values := range r.URL.Query() {
		// Validate parameter name
		if _, err := rv.validator.ValidateString(key, "query_param_key"); err != nil {
			return errors.ValidationError(fmt.Sprintf("invalid query parameter name: %s", key))
		}

		// Validate parameter values
		for _, value := range values {
			if _, err := rv.validator.ValidateString(value, fmt.Sprintf("query_param_%s", key)); err != nil {
				return errors.ValidationError(fmt.Sprintf("invalid value for parameter %s", key))
			}
		}
	}

	return nil
}

// validateBody validates request body
func (rv *RequestValidator) validateBody(r *http.Request) ([]byte, error) {
	// Check Content-Length
	if r.ContentLength > rv.config.MaxBodySize {
		return nil, errors.ValidationError(fmt.Sprintf("request body too large: %d bytes", r.ContentLength))
	}

	// Read body with size limit
	body, err := io.ReadAll(io.LimitReader(r.Body, rv.config.MaxBodySize))
	if err != nil {
		return nil, errors.ValidationError("failed to read request body")
	}
	defer r.Body.Close()

	// If empty body, return as is
	if len(body) == 0 {
		return body, nil
	}

	// Validate based on content type
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		// Validate JSON
		validated, err := rv.validator.ValidateJSON(string(body))
		if err != nil {
			return nil, err
		}

		// Re-encode
		return json.Marshal(validated)
	}

	// For other content types, just validate as string
	sanitized, err := rv.validator.ValidateString(string(body), "request_body")
	if err != nil {
		return nil, err
	}

	return []byte(sanitized), nil
}

// sendError sends an error response
func (rv *RequestValidator) sendError(w http.ResponseWriter, status int, message string) {
	rv.logger.Error().
		Int("status", status).
		Str("error", message).
		Msg("request validation failed")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := map[string]interface{}{
		"error":  message,
		"status": status,
	}

	json.NewEncoder(w).Encode(response)
}

// ValidateAPIRequest validates common API request patterns
func (rv *RequestValidator) ValidateAPIRequest(r *http.Request, schema interface{}) error {
	// This would validate against a JSON schema
	// For now, return nil
	return nil
}

// SanitizeResponse sanitizes response data before sending
func (rv *RequestValidator) SanitizeResponse(data interface{}) (interface{}, error) {
	// Convert to JSON and back to sanitize
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	validated, err := rv.validator.ValidateJSON(string(jsonBytes))
	if err != nil {
		return nil, err
	}

	return validated, nil
}
