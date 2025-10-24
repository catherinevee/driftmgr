package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/services"
	"github.com/google/uuid"
)

// BackendHandlers handles backend discovery API endpoints
type BackendHandlers struct {
	backendService *services.BackendService
}

// NewBackendHandlers creates a new BackendHandlers instance
func NewBackendHandlers(backendService *services.BackendService) *BackendHandlers {
	return &BackendHandlers{
		backendService: backendService,
	}
}

// ListBackends handles GET /api/v1/backends/list
func (h *BackendHandlers) ListBackends(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Parse pagination parameters
	page, limit := ParsePaginationParams(r)

	// Parse filters from query parameters
	filters := services.BackendFilters{
		Limit:  limit,
		Offset: (page - 1) * limit,
	}

	// Get provider filter
	if provider := r.URL.Query().Get("provider"); provider != "" {
		filters.Provider = provider
	}

	// Get region filter
	if region := r.URL.Query().Get("region"); region != "" {
		filters.Region = region
	}

	// Get active filter
	if active := r.URL.Query().Get("active"); active != "" {
		isActive := active == "true"
		filters.IsActive = &isActive
	}

	// Use real service to get backends
	providerConfigs, err := h.backendService.ListBackends(r.Context(), filters)
	if err != nil {
		response := NewResponseWriter(w)
		response.WriteInternalError("Failed to list backends: " + err.Error())
		return
	}

	// Convert to API response format
	backends := make([]Backend, 0, len(providerConfigs))
	for _, config := range providerConfigs {
		backend := Backend{
			ID:          config.ID,
			Type:        string(config.Provider),
			Name:        config.Name,
			Description: config.Description,
			Config: map[string]string{
				"region":     config.Region,
				"account_id": config.AccountID,
			},
			StateCount: 0, // This would need to be calculated
			IsActive:   config.IsActive,
			CreatedAt:  config.CreatedAt,
			UpdatedAt:  config.UpdatedAt,
		}
		backends = append(backends, backend)
	}

	// If no backends found from service, return empty list
	if len(backends) == 0 {
		backends = []Backend{}
	}

	// Apply pagination
	start := (page - 1) * limit
	end := start + limit

	if start >= len(backends) {
		backends = []Backend{}
	} else {
		if end > len(backends) {
			end = len(backends)
		}
		backends = backends[start:end]
	}

	// Create response
	response := NewResponseWriter(w)
	err = response.WritePaginationResponse(backends, page, limit, len(backends))
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// DiscoverBackends handles POST /api/v1/backends/discover
func (h *BackendHandlers) DiscoverBackends(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Parse request body
	var req BackendDiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := NewResponseWriter(w)
		response.WriteValidationError("Invalid request body", err.Error())
		return
	}

	// Validate request
	if len(req.Paths) == 0 {
		response := NewResponseWriter(w)
		response.WriteValidationError("At least one path is required", "")
		return
	}

	// Simulate backend discovery process
	// In a real implementation, this would scan the filesystem for Terraform configurations
	discoveredBackends := []Backend{
		{
			ID:          uuid.New().String(),
			Type:        "s3",
			Name:        "Discovered S3 Backend",
			Description: "Auto-discovered S3 backend",
			Config: map[string]string{
				"bucket": "new-terraform-state",
				"key":    "terraform.tfstate",
				"region": "us-west-2",
			},
			StateCount: 0,
			IsActive:   false,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		{
			ID:          uuid.New().String(),
			Type:        "local",
			Name:        "Discovered Local Backend",
			Description: "Auto-discovered local backend",
			Config: map[string]string{
				"path": "./discovered.tfstate",
			},
			StateCount: 0,
			IsActive:   false,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}

	// Create discovery response
	discoveryResponse := BackendDiscoveryResponse{
		Count:    len(discoveredBackends),
		Backends: discoveredBackends,
		Errors:   []string{}, // No errors in this simulation
	}

	response := NewResponseWriter(w)
	err := response.WriteSuccess(discoveryResponse, &APIMeta{
		Count:     len(discoveredBackends),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// GetBackend handles GET /api/v1/backends/{id}
func (h *BackendHandlers) GetBackend(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Extract backend ID from URL path
	// This is a simplified extraction - in a real router, this would be handled by the router
	path := r.URL.Path
	parts := splitPath(path)
	if len(parts) < 4 {
		response := NewResponseWriter(w)
		response.WriteBadRequest("Invalid backend ID")
		return
	}

	backendID := parts[3]

	// Use real service to get backend
	providerConfig, err := h.backendService.GetBackend(r.Context(), backendID)
	if err != nil {
		response := NewResponseWriter(w)
		response.WriteNotFound("Backend")
		return
	}

	// Convert to API response format
	backend := Backend{
		ID:          providerConfig.ID,
		Type:        string(providerConfig.Provider),
		Name:        providerConfig.Name,
		Description: providerConfig.Description,
		Config: map[string]string{
			"region":     providerConfig.Region,
			"account_id": providerConfig.AccountID,
		},
		StateCount: 0, // This would need to be calculated from state service
		IsActive:   providerConfig.IsActive,
		CreatedAt:  providerConfig.CreatedAt,
		UpdatedAt:  providerConfig.UpdatedAt,
	}

	response := NewResponseWriter(w)
	err = response.WriteSuccess(backend, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// UpdateBackend handles PUT /api/v1/backends/{id}
func (h *BackendHandlers) UpdateBackend(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Extract backend ID from URL path
	path := r.URL.Path
	parts := splitPath(path)
	if len(parts) < 4 {
		response := NewResponseWriter(w)
		response.WriteBadRequest("Invalid backend ID")
		return
	}

	backendID := parts[3]

	// Parse request body
	var updateRequest models.ProviderConfigurationUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		response := NewResponseWriter(w)
		response.WriteValidationError("Invalid request body", err.Error())
		return
	}

	// Use real service to update backend
	updatedProviderConfig, err := h.backendService.UpdateBackend(r.Context(), backendID, &updateRequest)
	if err != nil {
		response := NewResponseWriter(w)
		response.WriteNotFound("Backend")
		return
	}

	// Convert to API response format
	updatedBackend := Backend{
		ID:          updatedProviderConfig.ID,
		Type:        string(updatedProviderConfig.Provider),
		Name:        updatedProviderConfig.Name,
		Description: updatedProviderConfig.Description,
		Config: map[string]string{
			"region":     updatedProviderConfig.Region,
			"account_id": updatedProviderConfig.AccountID,
		},
		StateCount: 0, // This would need to be calculated from state service
		IsActive:   updatedProviderConfig.IsActive,
		CreatedAt:  updatedProviderConfig.CreatedAt,
		UpdatedAt:  updatedProviderConfig.UpdatedAt,
	}

	response := NewResponseWriter(w)
	err = response.WriteSuccess(updatedBackend, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// DeleteBackend handles DELETE /api/v1/backends/{id}
func (h *BackendHandlers) DeleteBackend(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Extract backend ID from URL path
	path := r.URL.Path
	parts := splitPath(path)
	if len(parts) < 4 {
		response := NewResponseWriter(w)
		response.WriteBadRequest("Invalid backend ID")
		return
	}

	backendID := parts[3]

	// Use real service to delete backend
	err := h.backendService.DeleteBackend(r.Context(), backendID)
	if err != nil {
		response := NewResponseWriter(w)
		response.WriteNotFound("Backend")
		return
	}

	// Successful deletion
	response := NewResponseWriter(w)
	err = response.WriteSuccess(map[string]string{
		"message": "Backend deleted successfully",
		"id":      backendID,
	}, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// TestBackend handles POST /api/v1/backends/{id}/test
func (h *BackendHandlers) TestBackend(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Extract backend ID from URL path
	path := r.URL.Path
	parts := splitPath(path)
	if len(parts) < 5 {
		response := NewResponseWriter(w)
		response.WriteBadRequest("Invalid backend ID")
		return
	}

	backendID := parts[3]

	// Use real service to test backend connection
	connectionTest, err := h.backendService.TestBackendConnection(r.Context(), backendID)
	if err != nil {
		response := NewResponseWriter(w)
		response.WriteNotFound("Backend")
		return
	}

	// Convert to API response format
	testResult := map[string]interface{}{
		"backend_id": backendID,
		"status":     "success",
		"message":    connectionTest.Message,
		"details":    connectionTest.Details,
		"tested_at":  connectionTest.TestedAt.UTC().Format(time.RFC3339),
		"duration":   connectionTest.Duration.String(),
	}

	if !connectionTest.Success {
		testResult["status"] = "failed"
	}

	response := NewResponseWriter(w)
	err = response.WriteSuccess(testResult, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// Helper function to split URL path
func splitPath(path string) []string {
	var parts []string
	var current string

	for _, char := range path {
		if char == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}
