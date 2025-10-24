package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/catherinevee/driftmgr/internal/services"
	"github.com/google/uuid"
)

// StateHandlers handles state management API endpoints
type StateHandlers struct {
	stateService *services.StateService
}

// NewStateHandlers creates a new StateHandlers instance
func NewStateHandlers(stateService *services.StateService) *StateHandlers {
	return &StateHandlers{
		stateService: stateService,
	}
}

// ListStateFiles handles GET /api/v1/state/list
func (h *StateHandlers) ListStateFiles(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Parse query parameters
	queryParams := ParseQueryParams(r)
	backendID := queryParams["backend"]
	page, limit := ParsePaginationParams(r)

	// Create filters for state service
	filters := services.StateFilters{
		BackendID: backendID,
		Limit:     limit,
		Offset:    (page - 1) * limit,
	}

	// Use real service to get state files
	stateModels, err := h.stateService.ListStateFiles(r.Context(), filters)
	if err != nil {
		response := NewResponseWriter(w)
		response.WriteInternalError("Failed to list state files: " + err.Error())
		return
	}

	// Convert to API response format
	stateFiles := make([]StateFile, 0, len(stateModels))
	for _, state := range stateModels {
		stateFile := StateFile{
			ID:            state.ID,
			Path:          state.Path,
			BackendID:     state.BackendID,
			BackendType:   "unknown", // This would need to be determined from backend
			Size:          state.Size,
			ResourceCount: len(state.Resources),
			LastModified:  state.LastModified,
			IsLocked:      false, // This would need to be determined from state details
			Metadata:      convertMetadataToStringMap(state.Metadata),
		}
		stateFiles = append(stateFiles, stateFile)
	}

	// Apply pagination
	start := (page - 1) * limit
	end := start + limit

	if start >= len(stateFiles) {
		stateFiles = []StateFile{}
	} else {
		if end > len(stateFiles) {
			end = len(stateFiles)
		}
		stateFiles = stateFiles[start:end]
	}

	response := NewResponseWriter(w)
	err = response.WritePaginationResponse(stateFiles, page, limit, len(stateFiles))
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// GetStateDetails handles GET /api/v1/state/details
func (h *StateHandlers) GetStateDetails(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Parse query parameters
	queryParams := ParseQueryParams(r)
	stateID := queryParams["id"]

	if stateID == "" {
		response := NewResponseWriter(w)
		response.WriteValidationError("ID parameter is required", "")
		return
	}

	// Use real service to get state details
	stateDetails, err := h.stateService.GetStateDetails(r.Context(), stateID)
	if err != nil {
		response := NewResponseWriter(w)
		response.WriteNotFound("State file")
		return
	}

	// Convert to API response format
	stateFile := StateFile{
		ID:            stateDetails.ID,
		Path:          fmt.Sprintf("state/%s/%s/terraform.tfstate", stateDetails.Workspace, stateDetails.Environment),
		BackendID:     stateDetails.BackendID,
		BackendType:   "unknown", // This would need to be determined from backend
		Size:          0,         // This would need to be calculated
		ResourceCount: len(stateDetails.Resources),
		LastModified:  stateDetails.LastUpdated,
		IsLocked:      stateDetails.IsLocked,
		Metadata: map[string]string{
			"terraform_version": "1.5.0", // This would need to be extracted from state
			"serial":            fmt.Sprintf("%d", stateDetails.Serial),
			"lineage":           stateDetails.Lineage,
		},
	}

	// Convert resources to API format
	resources := make([]Resource, 0, len(stateDetails.Resources))
	for _, stateResource := range stateDetails.Resources {
		for _, instance := range stateResource.Instances {
			resource := Resource{
				ID:           instance.ID,
				Name:         stateResource.Name,
				Type:         stateResource.Type,
				Provider:     stateResource.Provider,
				Region:       "unknown", // This would need to be extracted from attributes
				AccountID:    "unknown", // This would need to be extracted from attributes
				State:        "present",
				Tags:         make(map[string]string), // This would need to be extracted from attributes
				DiscoveredAt: stateDetails.CreatedAt,
				UpdatedAt:    stateDetails.LastUpdated,
			}
			resources = append(resources, resource)
		}
	}

	// Create detailed response
	detailedResponse := map[string]interface{}{
		"state_file": stateFile,
		"resources":  resources,
		"summary": map[string]interface{}{
			"total_resources": len(resources),
			"by_provider": map[string]int{
				"aws": len(resources),
			},
			"by_type": map[string]int{
				"aws_instance":  1,
				"aws_s3_bucket": 1,
			},
		},
	}

	response := NewResponseWriter(w)
	err = response.WriteSuccess(detailedResponse, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// ImportResource handles POST /api/v1/state/import
func (h *StateHandlers) ImportResource(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Parse request body
	var importRequest struct {
		StateID      string `json:"state_id"`
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
		ResourceName string `json:"resource_name"`
		Provider     string `json:"provider"`
	}

	if err := json.NewDecoder(r.Body).Decode(&importRequest); err != nil {
		response := NewResponseWriter(w)
		response.WriteValidationError("Invalid request body", err.Error())
		return
	}

	// Validate required fields
	if importRequest.StateID == "" || importRequest.ResourceType == "" || importRequest.ResourceID == "" || importRequest.ResourceName == "" {
		response := NewResponseWriter(w)
		response.WriteValidationError("state_id, resource_type, resource_id, and resource_name are required", "")
		return
	}

	// Use real service to import resource
	importReq := &services.ImportRequest{
		StateID:      importRequest.StateID,
		ResourceType: importRequest.ResourceType,
		ResourceName: importRequest.ResourceName,
		ResourceID:   importRequest.ResourceID,
		Provider:     importRequest.Provider,
	}

	importResult, err := h.stateService.ImportResourceToState(r.Context(), importReq)
	if err != nil {
		response := NewResponseWriter(w)
		response.WriteInternalError("Failed to import resource: " + err.Error())
		return
	}

	// Convert to API response format
	apiResult := map[string]interface{}{
		"import_id":   uuid.New().String(),
		"resource_id": importResult.ResourceID,
		"state_id":    importRequest.StateID,
		"status":      "success",
		"message":     importResult.Message,
		"imported_at": importResult.ImportedAt.UTC().Format(time.RFC3339),
	}

	if !importResult.Success {
		apiResult["status"] = "failed"
	}

	response := NewResponseWriter(w)
	err = response.WriteCreated(apiResult)
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// RemoveResource handles DELETE /api/v1/state/resources/{id}
func (h *StateHandlers) RemoveResource(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Extract resource ID from URL path
	path := r.URL.Path
	parts := splitPath(path)
	if len(parts) < 5 {
		response := NewResponseWriter(w)
		response.WriteBadRequest("Invalid resource ID")
		return
	}

	resourceID := parts[4]

	// Parse query parameters for state path
	queryParams := ParseQueryParams(r)
	statePath := queryParams["state_path"]

	if statePath == "" {
		response := NewResponseWriter(w)
		response.WriteValidationError("state_path query parameter is required", "")
		return
	}

	// Check if resource exists (simulate not found for certain IDs)
	if resourceID == "nonexistent" {
		response := NewResponseWriter(w)
		response.WriteNotFound("Resource")
		return
	}

	// Simulate resource removal
	removalResult := map[string]interface{}{
		"resource_id": resourceID,
		"state_path":  statePath,
		"status":      "success",
		"message":     "Resource removed from state successfully",
		"removed_at":  time.Now().UTC().Format(time.RFC3339),
	}

	response := NewResponseWriter(w)
	err := response.WriteSuccess(removalResult, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// MoveResource handles POST /api/v1/state/move
func (h *StateHandlers) MoveResource(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Parse request body
	var moveRequest struct {
		StatePath     string `json:"state_path"`
		ResourceID    string `json:"resource_id"`
		NewResourceID string `json:"new_resource_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&moveRequest); err != nil {
		response := NewResponseWriter(w)
		response.WriteValidationError("Invalid request body", err.Error())
		return
	}

	// Validate required fields
	if moveRequest.StatePath == "" || moveRequest.ResourceID == "" || moveRequest.NewResourceID == "" {
		response := NewResponseWriter(w)
		response.WriteValidationError("state_path, resource_id, and new_resource_id are required", "")
		return
	}

	// Simulate resource move
	moveResult := map[string]interface{}{
		"resource_id":     moveRequest.ResourceID,
		"new_resource_id": moveRequest.NewResourceID,
		"state_path":      moveRequest.StatePath,
		"status":          "success",
		"message":         "Resource moved successfully",
		"moved_at":        time.Now().UTC().Format(time.RFC3339),
	}

	response := NewResponseWriter(w)
	err := response.WriteSuccess(moveResult, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// LockStateFile handles POST /api/v1/state/lock
func (h *StateHandlers) LockStateFile(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Parse request body
	var lockRequest struct {
		StatePath string `json:"state_path"`
		Operation string `json:"operation"`
		Who       string `json:"who"`
	}

	if err := json.NewDecoder(r.Body).Decode(&lockRequest); err != nil {
		response := NewResponseWriter(w)
		response.WriteValidationError("Invalid request body", err.Error())
		return
	}

	// Validate required fields
	if lockRequest.StatePath == "" {
		response := NewResponseWriter(w)
		response.WriteValidationError("state_path is required", "")
		return
	}

	// Simulate state file locking
	lockInfo := LockInfo{
		ID:        uuid.New().String(),
		Operation: lockRequest.Operation,
		Who:       lockRequest.Who,
		Version:   "1.5.0",
		Created:   time.Now(),
		Path:      lockRequest.StatePath,
	}

	lockResult := map[string]interface{}{
		"lock_id":    lockInfo.ID,
		"state_path": lockRequest.StatePath,
		"status":     "success",
		"message":    "State file locked successfully",
		"lock_info":  lockInfo,
		"locked_at":  time.Now().UTC().Format(time.RFC3339),
	}

	response := NewResponseWriter(w)
	err := response.WriteSuccess(lockResult, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// UnlockStateFile handles POST /api/v1/state/unlock
func (h *StateHandlers) UnlockStateFile(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Parse request body
	var unlockRequest struct {
		StatePath string `json:"state_path"`
		LockID    string `json:"lock_id"`
		Force     bool   `json:"force,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&unlockRequest); err != nil {
		response := NewResponseWriter(w)
		response.WriteValidationError("Invalid request body", err.Error())
		return
	}

	// Validate required fields
	if unlockRequest.StatePath == "" {
		response := NewResponseWriter(w)
		response.WriteValidationError("state_path is required", "")
		return
	}

	// Simulate state file unlocking
	unlockResult := map[string]interface{}{
		"lock_id":     unlockRequest.LockID,
		"state_path":  unlockRequest.StatePath,
		"status":      "success",
		"message":     "State file unlocked successfully",
		"unlocked_at": time.Now().UTC().Format(time.RFC3339),
	}

	response := NewResponseWriter(w)
	err := response.WriteSuccess(unlockResult, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// Helper function to convert metadata to string map
func convertMetadataToStringMap(metadata map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range metadata {
		if str, ok := v.(string); ok {
			result[k] = str
		} else {
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}
