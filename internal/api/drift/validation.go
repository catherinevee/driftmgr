package drift

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// Validator handles request validation
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateDriftResultID validates a drift result ID
func (v *Validator) ValidateDriftResultID(id string) error {
	if id == "" {
		return fmt.Errorf("drift result ID is required")
	}

	// Check if ID is a valid UUID format
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !uuidRegex.MatchString(id) {
		return fmt.Errorf("invalid drift result ID format")
	}

	return nil
}

// ValidateProvider validates a provider name
func (v *Validator) ValidateProvider(provider string) error {
	if provider == "" {
		return fmt.Errorf("provider is required")
	}

	validProviders := map[string]bool{
		"aws":          true,
		"azure":        true,
		"gcp":          true,
		"digitalocean": true,
	}

	if !validProviders[provider] {
		return fmt.Errorf("invalid provider: %s", provider)
	}

	return nil
}

// ValidateStatus validates a drift status
func (v *Validator) ValidateStatus(status string) error {
	if status == "" {
		return nil // Status is optional
	}

	validStatuses := map[string]bool{
		"completed": true,
		"failed":    true,
		"running":   true,
	}

	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s", status)
	}

	return nil
}

// ValidateLimit validates a limit parameter
func (v *Validator) ValidateLimit(limitStr string, maxLimit int) (int, error) {
	if limitStr == "" {
		return 50, nil // Default limit
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return 0, fmt.Errorf("invalid limit: %s", limitStr)
	}

	if limit < 0 {
		return 0, fmt.Errorf("limit must be non-negative")
	}

	if limit > maxLimit {
		return 0, fmt.Errorf("limit exceeds maximum of %d", maxLimit)
	}

	return limit, nil
}

// ValidateOffset validates an offset parameter
func (v *Validator) ValidateOffset(offsetStr string) (int, error) {
	if offsetStr == "" {
		return 0, nil // Default offset
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return 0, fmt.Errorf("invalid offset: %s", offsetStr)
	}

	if offset < 0 {
		return 0, fmt.Errorf("offset must be non-negative")
	}

	return offset, nil
}

// ValidateDate validates a date parameter
func (v *Validator) ValidateDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, nil // Empty date is valid
	}

	// Try different date formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
	}

	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return date, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid date format: %s", dateStr)
}

// ValidateDateRange validates a date range
func (v *Validator) ValidateDateRange(startDate, endDate time.Time) error {
	if !startDate.IsZero() && !endDate.IsZero() {
		if startDate.After(endDate) {
			return fmt.Errorf("start date must be before end date")
		}
	}

	return nil
}

// ValidateSortField validates a sort field
func (v *Validator) ValidateSortField(field string) error {
	if field == "" {
		return nil // Empty sort field is valid (will use default)
	}

	validFields := map[string]bool{
		"timestamp":   true,
		"drift_count": true,
		"provider":    true,
		"status":      true,
	}

	if !validFields[field] {
		return fmt.Errorf("invalid sort field: %s", field)
	}

	return nil
}

// ValidateSortOrder validates a sort order
func (v *Validator) ValidateSortOrder(order string) error {
	if order == "" {
		return nil // Empty sort order is valid (will use default)
	}

	validOrders := map[string]bool{
		"asc":  true,
		"desc": true,
	}

	if !validOrders[order] {
		return fmt.Errorf("invalid sort order: %s", order)
	}

	return nil
}

// ValidateDays validates a days parameter
func (v *Validator) ValidateDays(daysStr string, maxDays int) (int, error) {
	if daysStr == "" {
		return 30, nil // Default to 30 days
	}

	days, err := strconv.Atoi(daysStr)
	if err != nil {
		return 0, fmt.Errorf("invalid days: %s", daysStr)
	}

	if days < 1 {
		return 0, fmt.Errorf("days must be positive")
	}

	if days > maxDays {
		return 0, fmt.Errorf("days exceeds maximum of %d", maxDays)
	}

	return days, nil
}

// ValidateQueryParameters validates query parameters for drift results
func (v *Validator) ValidateQueryParameters(r *http.Request) (*models.DriftResultQuery, error) {
	query := &models.DriftResultQuery{
		Filter: models.DriftResultFilter{},
		Sort:   models.DriftResultSort{},
		Limit:  50,
		Offset: 0,
	}

	// Validate provider
	if provider := r.URL.Query().Get("provider"); provider != "" {
		if err := v.ValidateProvider(provider); err != nil {
			return nil, err
		}
		query.Filter.Provider = provider
	}

	// Validate status
	if status := r.URL.Query().Get("status"); status != "" {
		if err := v.ValidateStatus(status); err != nil {
			return nil, err
		}
		query.Filter.Status = status
	}

	// Validate dates
	if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
		startDate, err := v.ValidateDate(startDateStr)
		if err != nil {
			return nil, err
		}
		query.Filter.StartDate = startDate
	}

	if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
		endDate, err := v.ValidateDate(endDateStr)
		if err != nil {
			return nil, err
		}
		query.Filter.EndDate = endDate
	}

	// Validate date range
	if err := v.ValidateDateRange(query.Filter.StartDate, query.Filter.EndDate); err != nil {
		return nil, err
	}

	// Validate drift count filters
	if minDriftStr := r.URL.Query().Get("min_drift"); minDriftStr != "" {
		minDrift, err := strconv.Atoi(minDriftStr)
		if err != nil {
			return nil, fmt.Errorf("invalid min_drift: %s", minDriftStr)
		}
		if minDrift < 0 {
			return nil, fmt.Errorf("min_drift must be non-negative")
		}
		query.Filter.MinDrift = minDrift
	}

	if maxDriftStr := r.URL.Query().Get("max_drift"); maxDriftStr != "" {
		maxDrift, err := strconv.Atoi(maxDriftStr)
		if err != nil {
			return nil, fmt.Errorf("invalid max_drift: %s", maxDriftStr)
		}
		if maxDrift < 0 {
			return nil, fmt.Errorf("max_drift must be non-negative")
		}
		query.Filter.MaxDrift = maxDrift
	}

	// Validate sort parameters
	if sortField := r.URL.Query().Get("sort_field"); sortField != "" {
		if err := v.ValidateSortField(sortField); err != nil {
			return nil, err
		}
		query.Sort.Field = sortField
	}

	if sortOrder := r.URL.Query().Get("sort_order"); sortOrder != "" {
		if err := v.ValidateSortOrder(sortOrder); err != nil {
			return nil, err
		}
		query.Sort.Order = sortOrder
	}

	// Validate pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := v.ValidateLimit(limitStr, 1000)
		if err != nil {
			return nil, err
		}
		query.Limit = limit
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offset, err := v.ValidateOffset(offsetStr)
		if err != nil {
			return nil, err
		}
		query.Offset = offset
	}

	return query, nil
}

// ValidateHistoryParameters validates query parameters for drift history
func (v *Validator) ValidateHistoryParameters(r *http.Request) (*models.DriftHistoryRequest, error) {
	req := &models.DriftHistoryRequest{
		Limit:  50,
		Offset: 0,
	}

	// Validate provider
	if provider := r.URL.Query().Get("provider"); provider != "" {
		if err := v.ValidateProvider(provider); err != nil {
			return nil, err
		}
		req.Provider = provider
	}

	// Validate status
	if status := r.URL.Query().Get("status"); status != "" {
		if err := v.ValidateStatus(status); err != nil {
			return nil, err
		}
		req.Status = status
	}

	// Validate dates
	if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
		startDate, err := v.ValidateDate(startDateStr)
		if err != nil {
			return nil, err
		}
		req.StartDate = startDate
	}

	if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
		endDate, err := v.ValidateDate(endDateStr)
		if err != nil {
			return nil, err
		}
		req.EndDate = endDate
	}

	// Validate date range
	if err := v.ValidateDateRange(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	// Validate pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := v.ValidateLimit(limitStr, 1000)
		if err != nil {
			return nil, err
		}
		req.Limit = limit
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offset, err := v.ValidateOffset(offsetStr)
		if err != nil {
			return nil, err
		}
		req.Offset = offset
	}

	return req, nil
}

// ValidateTrendParameters validates query parameters for drift trend
func (v *Validator) ValidateTrendParameters(r *http.Request) (string, int, error) {
	provider := r.URL.Query().Get("provider")

	// Validate provider if provided
	if provider != "" {
		if err := v.ValidateProvider(provider); err != nil {
			return "", 0, err
		}
	}

	// Validate days
	daysStr := r.URL.Query().Get("days")
	days, err := v.ValidateDays(daysStr, 365)
	if err != nil {
		return "", 0, err
	}

	return provider, days, nil
}

// ValidateTopResourcesParameters validates query parameters for top drifted resources
func (v *Validator) ValidateTopResourcesParameters(r *http.Request) (int, error) {
	limitStr := r.URL.Query().Get("limit")
	limit, err := v.ValidateLimit(limitStr, 100)
	if err != nil {
		return 0, err
	}

	return limit, nil
}

// ValidateSeverityParameters validates query parameters for drift by severity
func (v *Validator) ValidateSeverityParameters(r *http.Request) (string, error) {
	provider := r.URL.Query().Get("provider")

	// Validate provider if provided
	if provider != "" {
		if err := v.ValidateProvider(provider); err != nil {
			return "", err
		}
	}

	return provider, nil
}

// SanitizeString sanitizes a string input
func (v *Validator) SanitizeString(input string) string {
	// Remove leading/trailing whitespace
	input = strings.TrimSpace(input)

	// Remove potentially dangerous characters
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#x27;")
	input = strings.ReplaceAll(input, "&", "&amp;")

	return input
}

// ValidateRequest validates an HTTP request
func (v *Validator) ValidateRequest(r *http.Request) error {
	// Check request method
	allowedMethods := map[string]bool{
		"GET":     true,
		"POST":    true,
		"PUT":     true,
		"DELETE":  true,
		"OPTIONS": true,
	}

	if !allowedMethods[r.Method] {
		return fmt.Errorf("method %s not allowed", r.Method)
	}

	// Check content type for POST/PUT requests
	if r.Method == "POST" || r.Method == "PUT" {
		contentType := r.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			return fmt.Errorf("content type must be application/json")
		}
	}

	return nil
}

// WriteValidationError writes a validation error response
func (v *Validator) WriteValidationError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	response := map[string]interface{}{
		"error":     "validation_error",
		"message":   err.Error(),
		"timestamp": time.Now(),
	}

	// This would typically use a JSON encoder
	// For now, we'll just write a simple error message
	w.Write([]byte(fmt.Sprintf(`{"error":"validation_error","message":"%s","timestamp":"%s"}`, err.Error(), time.Now().Format(time.RFC3339))))
	_ = response // Suppress unused variable warning
}
