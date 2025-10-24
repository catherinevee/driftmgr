package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// ResponseWriter wraps http.ResponseWriter with additional functionality
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// NewResponseWriter creates a new ResponseWriter
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader captures the status code
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// StatusCode returns the captured status code
func (rw *ResponseWriter) StatusCode() int {
	return rw.statusCode
}

// WriteJSON writes a JSON response with proper headers
func (rw *ResponseWriter) WriteJSON(statusCode int, data interface{}) error {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(statusCode)
	return json.NewEncoder(rw).Encode(data)
}

// WriteSuccess writes a success response
func (rw *ResponseWriter) WriteSuccess(data interface{}, meta *APIMeta) error {
	response := NewSuccessResponse(data, meta)
	return rw.WriteJSON(http.StatusOK, response)
}

// WriteCreated writes a created response
func (rw *ResponseWriter) WriteCreated(data interface{}) error {
	response := NewSuccessResponse(data, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	return rw.WriteJSON(http.StatusCreated, response)
}

// WriteError writes an error response
func (rw *ResponseWriter) WriteError(statusCode int, code, message, details string) error {
	response := NewErrorResponse(code, message, details)
	return rw.WriteJSON(statusCode, response)
}

// WriteBadRequest writes a bad request error response
func (rw *ResponseWriter) WriteBadRequest(message string) error {
	return rw.WriteError(http.StatusBadRequest, "BAD_REQUEST", message, "")
}

// WriteNotFound writes a not found error response
func (rw *ResponseWriter) WriteNotFound(resource string) error {
	return rw.WriteError(http.StatusNotFound, "NOT_FOUND", resource+" not found", "")
}

// WriteInternalError writes an internal server error response
func (rw *ResponseWriter) WriteInternalError(message string) error {
	return rw.WriteError(http.StatusInternalServerError, "INTERNAL_ERROR", message, "")
}

// WriteUnauthorized writes an unauthorized error response
func (rw *ResponseWriter) WriteUnauthorized(message string) error {
	return rw.WriteError(http.StatusUnauthorized, "UNAUTHORIZED", message, "")
}

// WriteForbidden writes a forbidden error response
func (rw *ResponseWriter) WriteForbidden(message string) error {
	return rw.WriteError(http.StatusForbidden, "FORBIDDEN", message, "")
}

// WriteConflict writes a conflict error response
func (rw *ResponseWriter) WriteConflict(message string) error {
	return rw.WriteError(http.StatusConflict, "CONFLICT", message, "")
}

// WriteValidationError writes a validation error response
func (rw *ResponseWriter) WriteValidationError(message string, details string) error {
	return rw.WriteError(http.StatusBadRequest, "VALIDATION_ERROR", message, details)
}

// WritePaginationResponse writes a paginated response
func (rw *ResponseWriter) WritePaginationResponse(data interface{}, page, limit, count int) error {
	meta := NewPaginationMeta(page, limit, count)
	return rw.WriteSuccess(data, meta)
}

// ParsePaginationParams parses pagination parameters from request
func ParsePaginationParams(r *http.Request) (page, limit int) {
	page = 1
	limit = 10

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	return page, limit
}

// ParseQueryParams parses query parameters into a map
func ParseQueryParams(r *http.Request) map[string]string {
	params := make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	return params
}

// SetCORSHeaders sets CORS headers
func SetCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
	w.Header().Set("Access-Control-Max-Age", "86400")
}

// HandleCORS handles CORS preflight requests
func HandleCORS(w http.ResponseWriter, r *http.Request) bool {
	SetCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return true
	}
	
	return false
}

// SetCacheHeaders sets appropriate cache headers
func SetCacheHeaders(w http.ResponseWriter, maxAge int) {
	w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(maxAge))
	w.Header().Set("Expires", time.Now().Add(time.Duration(maxAge)*time.Second).UTC().Format(time.RFC1123))
}

// SetNoCacheHeaders sets no-cache headers
func SetNoCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

// SetSecurityHeaders sets security headers
func SetSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

// SetCommonHeaders sets common headers for all responses
func SetCommonHeaders(w http.ResponseWriter) {
	SetCORSHeaders(w)
	SetSecurityHeaders(w)
	SetNoCacheHeaders(w)
}
