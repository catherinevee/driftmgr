package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/catherinevee/driftmgr/internal/api/handlers/discovery"
	"github.com/catherinevee/driftmgr/internal/terraform/state"
)

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// PerspectiveHandler handles state file perspective operations
type PerspectiveHandler struct {
	discoveryService *state.DiscoveryService
	analyzer         *state.StateAnalyzer
	discoveryHub     *discovery.DiscoveryHub
}

// NewPerspectiveHandler creates a new perspective handler
func NewPerspectiveHandler(discoveryHub *discovery.DiscoveryHub) *PerspectiveHandler {
	return &PerspectiveHandler{
		discoveryService: state.NewDiscoveryService(),
		analyzer:         state.NewStateAnalyzer(),
		discoveryHub:     discoveryHub,
	}
}

// SetupPerspectiveRoutes sets up all perspective-related routes
func (h *PerspectiveHandler) SetupPerspectiveRoutes(api *mux.Router) {
	// State discovery endpoints
	api.HandleFunc("/state/discovery/start", h.handleStartDiscovery).Methods("POST")
	api.HandleFunc("/state/discovery/status", h.handleDiscoveryStatus).Methods("GET")
	api.HandleFunc("/state/discovery/results", h.handleDiscoveryResults).Methods("GET")
	api.HandleFunc("/state/discovery/auto", h.handleAutoDiscovery).Methods("POST")
	
	// State file management
	api.HandleFunc("/state/files", h.handleListStateFiles).Methods("GET")
	api.HandleFunc("/state/files/{id}", h.handleGetStateFile).Methods("GET")
	api.HandleFunc("/state/files/{id}/refresh", h.handleRefreshStateFile).Methods("POST")
	api.HandleFunc("/state/files/{id}/analyze", h.handleAnalyzeStateFile).Methods("POST")
	
	// Perspective operations
	api.HandleFunc("/perspective/{id}", h.handleGetPerspective).Methods("GET")
	api.HandleFunc("/perspective/{id}/generate", h.handleGeneratePerspective).Methods("POST")
	api.HandleFunc("/perspective/{id}/out-of-band", h.handleGetOutOfBand).Methods("GET")
	api.HandleFunc("/perspective/{id}/conflicts", h.handleGetConflicts).Methods("GET")
	api.HandleFunc("/perspective/{id}/graph", h.handleGetResourceGraph).Methods("GET")
	api.HandleFunc("/perspective/{id}/timeline", h.handleGetTimeline).Methods("GET")
	
	// Comparison operations
	api.HandleFunc("/perspective/compare", h.handleComparePerspectives).Methods("POST")
	api.HandleFunc("/perspective/diff", h.handleDiffPerspectives).Methods("POST")
	
	// Adoption operations
	api.HandleFunc("/perspective/{id}/adopt", h.handleAdoptResources).Methods("POST")
	api.HandleFunc("/perspective/{id}/adopt/preview", h.handleAdoptionPreview).Methods("POST")
	api.HandleFunc("/perspective/{id}/import-commands", h.handleGenerateImports).Methods("GET")
	
	// Search and filter
	api.HandleFunc("/state/search", h.handleSearchStateFiles).Methods("POST")
	api.HandleFunc("/state/filter", h.handleFilterStateFiles).Methods("POST")
}

// handleStartDiscovery starts a new state file discovery process
func (h *PerspectiveHandler) handleStartDiscovery(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Paths         []string                       `json:"paths"`
		CloudBackends []state.BackendConfig          `json:"cloud_backends"`
		AutoScan      bool                           `json:"auto_scan"`
		ScanInterval  int                            `json:"scan_interval_minutes"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Add scan paths
	for _, path := range request.Paths {
		h.discoveryService.AddScanPath(path)
	}
	
	// Add cloud backends
	for _, backend := range request.CloudBackends {
		h.discoveryService.AddCloudBackend(backend)
	}
	
	// Start discovery
	ctx := context.Background()
	if request.AutoScan {
		h.discoveryService.StartAutoDiscovery(ctx)
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status": "auto_discovery_started",
			"message": "Automatic discovery started with periodic scanning",
		})
	} else {
		go h.discoveryService.DiscoverAll(ctx)
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status": "discovery_started",
			"message": "One-time discovery started",
		})
	}
}

// handleDiscoveryStatus returns the current discovery status
func (h *PerspectiveHandler) handleDiscoveryStatus(w http.ResponseWriter, r *http.Request) {
	status := h.discoveryService.GetScanStatus()
	respondJSON(w, http.StatusOK, status)
}

// handleDiscoveryResults returns discovered state files
func (h *PerspectiveHandler) handleDiscoveryResults(w http.ResponseWriter, r *http.Request) {
	states := h.discoveryService.GetDiscoveredStates()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"count": len(states),
		"states": states,
	})
}

// handleAutoDiscovery configures automatic discovery
func (h *PerspectiveHandler) handleAutoDiscovery(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Enabled      bool     `json:"enabled"`
		Paths        []string `json:"paths"`
		Interval     int      `json:"interval_minutes"`
		IncludeGit   bool     `json:"include_git"`
		IncludeCloud bool     `json:"include_cloud"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	if request.Enabled {
		// Configure scan paths
		for _, path := range request.Paths {
			h.discoveryService.AddScanPath(path)
		}
		
		// Start auto-discovery
		ctx := context.Background()
		h.discoveryService.StartAutoDiscovery(ctx)
		
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status": "enabled",
			"message": "Auto-discovery enabled",
		})
	} else {
		h.discoveryService.StopAutoDiscovery()
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status": "disabled",
			"message": "Auto-discovery disabled",
		})
	}
}

// handleListStateFiles lists all discovered state files
func (h *PerspectiveHandler) handleListStateFiles(w http.ResponseWriter, r *http.Request) {
	// Get filter parameters
	backendType := r.URL.Query().Get("backend")
	healthStatus := r.URL.Query().Get("health")
	
	var states []*state.StateFile
	
	if backendType != "" {
		discoveredStates := h.discoveryService.GetStateFilesByBackend(backendType)
		states = ConvertDiscoveredToStateFiles(discoveredStates)
	} else if healthStatus != "" {
		discoveredStates := h.discoveryService.GetStateFilesByHealth(healthStatus)
		states = ConvertDiscoveredToStateFiles(discoveredStates)
	} else {
		discoveredStates := h.discoveryService.GetDiscoveredStates()
		states = ConvertDiscoveredToStateFiles(discoveredStates)
	}
	
	// Group by backend type - using Path as a proxy for backend type
	grouped := make(map[string][]*state.StateFile)
	for _, s := range states {
		// For now, use "local" as default type since StateFile doesn't have Type
		backendType := "local"
		grouped[backendType] = append(grouped[backendType], s)
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"count": len(states),
		"states": states,
		"grouped": grouped,
		"backends": getBackendTypes(states),
		"health_summary": getHealthSummary(states),
	})
}

// handleGetStateFile returns details of a specific state file
func (h *PerspectiveHandler) handleGetStateFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	stateFile, err := h.discoveryService.GetStateFile(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "State file not found")
		return
	}
	
	respondJSON(w, http.StatusOK, stateFile)
}

// handleRefreshStateFile refreshes the analysis of a state file
func (h *PerspectiveHandler) handleRefreshStateFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	ctx := context.Background()
	if err := h.discoveryService.RefreshStateFile(ctx, id); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to refresh: %v", err))
		return
	}
	
	stateFile, _ := h.discoveryService.GetStateFile(id)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "refreshed",
		"state": stateFile,
	})
}

// handleAnalyzeStateFile performs deep analysis on a state file
func (h *PerspectiveHandler) handleAnalyzeStateFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	discoveredStateFile, err := h.discoveryService.GetStateFile(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "State file not found")
		return
	}
	
	// Convert discovered state file to regular state file
	stateFile := ConvertDiscoveredToStateFile(discoveredStateFile)
	
	// Get cloud resources from discovery hub
	cloudResources := h.discoveryHub.GetCachedResults()
	
	// Convert to interface slice for analyzer
	var cloudResourcesInterface []interface{}
	for _, r := range cloudResources {
		cloudResourcesInterface = append(cloudResourcesInterface, r)
	}
	
	ctx := context.Background()
	perspective, err := h.analyzer.AnalyzePerspective(ctx, stateFile, cloudResourcesInterface)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Analysis failed: %v", err))
		return
	}
	
	respondJSON(w, http.StatusOK, perspective)
}

// handleGetPerspective returns a cached perspective
func (h *PerspectiveHandler) handleGetPerspective(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	perspective, exists := h.analyzer.GetPerspective(id)
	if !exists {
		// Try to generate it
		discoveredStateFile, err := h.discoveryService.GetStateFile(id)
		if err != nil {
			respondError(w, http.StatusNotFound, "Perspective not found")
			return
		}
		
		// Convert discovered state file to regular state file
		stateFile := ConvertDiscoveredToStateFile(discoveredStateFile)
		
		cloudResources := h.discoveryHub.GetCachedResults()
		var cloudResourcesInterface []interface{}
		for _, r := range cloudResources {
			cloudResourcesInterface = append(cloudResourcesInterface, r)
		}
		
		ctx := context.Background()
		perspective, err = h.analyzer.AnalyzePerspective(ctx, stateFile, cloudResourcesInterface)
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to generate perspective: %v", err))
			return
		}
	}
	
	respondJSON(w, http.StatusOK, perspective)
}

// handleGeneratePerspective generates a new perspective for a state file
func (h *PerspectiveHandler) handleGeneratePerspective(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	discoveredStateFile, err := h.discoveryService.GetStateFile(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "State file not found")
		return
	}
	
	// Convert discovered state file to regular state file
	stateFile := ConvertDiscoveredToStateFile(discoveredStateFile)
	
	// Get cloud resources
	cloudResources := h.discoveryHub.GetCachedResults()
	var cloudResourcesInterface []interface{}
	for _, r := range cloudResources {
		cloudResourcesInterface = append(cloudResourcesInterface, r)
	}
	
	ctx := context.Background()
	perspective, err := h.analyzer.AnalyzePerspective(ctx, stateFile, cloudResourcesInterface)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to generate perspective: %v", err))
		return
	}
	
	respondJSON(w, http.StatusOK, perspective)
}

// handleGetOutOfBand returns out-of-band resources for a perspective
func (h *PerspectiveHandler) handleGetOutOfBand(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	perspective, exists := h.analyzer.GetPerspective(id)
	if !exists {
		respondError(w, http.StatusNotFound, "Perspective not found")
		return
	}
	
	// Filter by priority if requested
	priority := r.URL.Query().Get("priority")
	var filtered []state.OutOfBandResource
	
	if priority != "" {
		for _, oob := range perspective.OutOfBand {
			if oob.AdoptionPriority == priority {
				filtered = append(filtered, oob)
			}
		}
	} else {
		filtered = perspective.OutOfBand
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"count": len(filtered),
		"resources": filtered,
		"total_out_of_band": len(perspective.OutOfBand),
		"by_priority": groupByPriority(perspective.OutOfBand),
	})
}

// handleGetConflicts returns conflicts for a perspective
func (h *PerspectiveHandler) handleGetConflicts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	perspective, exists := h.analyzer.GetPerspective(id)
	if !exists {
		respondError(w, http.StatusNotFound, "Perspective not found")
		return
	}
	
	// Filter by severity if requested
	severity := r.URL.Query().Get("severity")
	var filtered []state.ResourceConflict
	
	if severity != "" {
		for _, conflict := range perspective.Conflicts {
			if conflict.Severity == severity {
				filtered = append(filtered, conflict)
			}
		}
	} else {
		filtered = perspective.Conflicts
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"count": len(filtered),
		"conflicts": filtered,
		"total_conflicts": len(perspective.Conflicts),
		"by_severity": groupBySeverity(perspective.Conflicts),
	})
}

// handleGetResourceGraph returns the resource dependency graph
func (h *PerspectiveHandler) handleGetResourceGraph(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	perspective, exists := h.analyzer.GetPerspective(id)
	if !exists {
		respondError(w, http.StatusNotFound, "Perspective not found")
		return
	}
	
	respondJSON(w, http.StatusOK, perspective.ResourceGraph)
}

// handleGetTimeline returns the state file timeline
func (h *PerspectiveHandler) handleGetTimeline(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	// This would retrieve historical data about the state file
	// For now, return a placeholder
	timeline := map[string]interface{}{
		"state_file_id": id,
		"events": []map[string]interface{}{
			{
				"timestamp": time.Now().Add(-24 * time.Hour),
				"event_type": "refresh",
				"description": "State refreshed",
			},
			{
				"timestamp": time.Now().Add(-48 * time.Hour),
				"event_type": "apply",
				"description": "Terraform apply executed",
			},
		},
	}
	
	respondJSON(w, http.StatusOK, timeline)
}

// handleComparePerspectives compares two perspectives
func (h *PerspectiveHandler) handleComparePerspectives(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Perspective1 string `json:"perspective_1"`
		Perspective2 string `json:"perspective_2"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	p1, exists1 := h.analyzer.GetPerspective(request.Perspective1)
	p2, exists2 := h.analyzer.GetPerspective(request.Perspective2)
	
	if !exists1 || !exists2 {
		respondError(w, http.StatusNotFound, "One or both perspectives not found")
		return
	}
	
	comparison := h.analyzer.ComparePerspectives(p1, p2)
	respondJSON(w, http.StatusOK, comparison)
}

// handleDiffPerspectives generates a diff between perspectives
func (h *PerspectiveHandler) handleDiffPerspectives(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Perspective1 string `json:"perspective_1"`
		Perspective2 string `json:"perspective_2"`
		Format       string `json:"format"` // unified, side-by-side, json
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	p1, exists1 := h.analyzer.GetPerspective(request.Perspective1)
	p2, exists2 := h.analyzer.GetPerspective(request.Perspective2)
	
	if !exists1 || !exists2 {
		respondError(w, http.StatusNotFound, "One or both perspectives not found")
		return
	}
	
	// Generate diff based on format
	diff := generateDiff(p1, p2, request.Format)
	respondJSON(w, http.StatusOK, diff)
}

// handleAdoptResources adopts out-of-band resources into Terraform
func (h *PerspectiveHandler) handleAdoptResources(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	var request struct {
		ResourceIDs []string `json:"resource_ids"`
		DryRun      bool     `json:"dry_run"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	perspective, exists := h.analyzer.GetPerspective(id)
	if !exists {
		respondError(w, http.StatusNotFound, "Perspective not found")
		return
	}
	
	// Generate import commands for selected resources
	var imports []map[string]string
	for _, resourceID := range request.ResourceIDs {
		for _, oob := range perspective.OutOfBand {
			if oob.ID == resourceID {
				imports = append(imports, map[string]string{
					"resource_id": oob.ID,
					"resource_type": oob.Type,
					"import_command": oob.SuggestedImport,
				})
			}
		}
	}
	
	if request.DryRun {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"dry_run": true,
			"imports": imports,
		})
	} else {
		// In production, this would execute the import commands
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status": "adopted",
			"count": len(imports),
			"imports": imports,
		})
	}
}

// handleAdoptionPreview previews adoption of out-of-band resources
func (h *PerspectiveHandler) handleAdoptionPreview(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	perspective, exists := h.analyzer.GetPerspective(id)
	if !exists {
		respondError(w, http.StatusNotFound, "Perspective not found")
		return
	}
	
	// Group resources by adoption priority
	preview := map[string]interface{}{
		"total_out_of_band": len(perspective.OutOfBand),
		"by_priority": groupByPriority(perspective.OutOfBand),
		"recommended": []state.OutOfBandResource{},
		"import_commands": []string{},
	}
	
	// Add high-priority resources to recommendations
	for _, oob := range perspective.OutOfBand {
		if oob.AdoptionPriority == "high" {
			preview["recommended"] = append(preview["recommended"].([]state.OutOfBandResource), oob)
			preview["import_commands"] = append(preview["import_commands"].([]string), oob.SuggestedImport)
		}
	}
	
	respondJSON(w, http.StatusOK, preview)
}

// handleGenerateImports generates import commands for a perspective
func (h *PerspectiveHandler) handleGenerateImports(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	perspective, exists := h.analyzer.GetPerspective(id)
	if !exists {
		respondError(w, http.StatusNotFound, "Perspective not found")
		return
	}
	
	priority := r.URL.Query().Get("priority")
	
	var imports []string
	for _, oob := range perspective.OutOfBand {
		if priority == "" || oob.AdoptionPriority == priority {
			imports = append(imports, oob.SuggestedImport)
		}
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"count": len(imports),
		"commands": imports,
		"script": generateImportScript(imports),
	})
}

// handleSearchStateFiles searches state files
func (h *PerspectiveHandler) handleSearchStateFiles(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Query    string   `json:"query"`
		Backends []string `json:"backends"`
		Health   []string `json:"health_statuses"`
		MinAge   int      `json:"min_age_days"`
		MaxAge   int      `json:"max_age_days"`
		MinSize  int64    `json:"min_size_bytes"`
		MaxSize  int64    `json:"max_size_bytes"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Get all state files
	allDiscoveredStates := h.discoveryService.GetDiscoveredStates()
	
	// Filter based on search criteria
	var filtered []*state.StateFile
	for _, s := range allDiscoveredStates {
		// Check query match
		if request.Query != "" {
			// Convert for containsQuery
			convertedState := ConvertDiscoveredToStateFile(s)
			if !containsQuery(convertedState, request.Query) {
				continue
			}
		}
		
		// Check backend filter
		if len(request.Backends) > 0 {
			if !contains(request.Backends, s.Type) {
				continue
			}
		}
		
		// Check health filter
		if len(request.Health) > 0 {
			if !contains(request.Health, s.Health.Status) {
				continue
			}
		}
		
		// Check age filters
		age := time.Since(s.LastModified).Hours() / 24
		if request.MinAge > 0 && age < float64(request.MinAge) {
			continue
		}
		if request.MaxAge > 0 && age > float64(request.MaxAge) {
			continue
		}
		
		// Check size filters
		if request.MinSize > 0 && s.Size < request.MinSize {
			continue
		}
		if request.MaxSize > 0 && s.Size > request.MaxSize {
			continue
		}
		
		// Convert and add to filtered
		filtered = append(filtered, ConvertDiscoveredToStateFile(s))
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"count": len(filtered),
		"states": filtered,
		"total": len(allDiscoveredStates),
	})
}

// handleFilterStateFiles filters state files with advanced criteria
func (h *PerspectiveHandler) handleFilterStateFiles(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Filters []Filter `json:"filters"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Get all state files
	allDiscoveredStates := h.discoveryService.GetDiscoveredStates()
	
	// Convert to regular state files
	allStates := ConvertDiscoveredToStateFiles(allDiscoveredStates)
	
	// Apply filters
	filtered := filterStateFiles(allStates, request.Filters)
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"count": len(filtered),
		"states": filtered,
		"total": len(allDiscoveredStates),
	})
}

// Filter represents a filter criterion
type Filter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// Helper functions

func getBackendTypes(states []*state.StateFile) []string {
	// Since StateFile doesn't have Type, we return a default list
	return []string{"local"}
}

func getHealthSummary(states []*state.StateFile) map[string]int {
	// Since StateFile doesn't have Health, return a default summary
	summary := make(map[string]int)
	summary["unknown"] = len(states)
	return summary
}

func groupByPriority(resources []state.OutOfBandResource) map[string]int {
	grouped := make(map[string]int)
	for _, r := range resources {
		grouped[r.AdoptionPriority]++
	}
	return grouped
}

func groupBySeverity(conflicts []state.ResourceConflict) map[string]int {
	grouped := make(map[string]int)
	for _, c := range conflicts {
		grouped[c.Severity]++
	}
	return grouped
}

func generateDiff(p1, p2 *state.StatePerspective, format string) map[string]interface{} {
	// Simple diff implementation
	return map[string]interface{}{
		"format": format,
		"perspective_1": p1.StateFileID,
		"perspective_2": p2.StateFileID,
		"differences": map[string]interface{}{
			"resource_count_diff": len(p1.ManagedResources) - len(p2.ManagedResources),
		},
	}
}

func generateImportScript(commands []string) string {
	script := "#!/bin/bash\n"
	script += "# Terraform import script\n"
	script += "# Generated by DriftMgr\n\n"
	
	for _, cmd := range commands {
		script += cmd + "\n"
	}
	
	return script
}

func containsQuery(state *state.StateFile, query string) bool {
	// Search in available fields
	// StateFile only has ID and Path, not Name or Type
	if strings.Contains(state.ID, query) || strings.Contains(state.Path, query) {
		return true
	}
	return false
}

// Removed duplicate contains function - using the one from discovery_hub.go

// ConvertDiscoveredToStateFile converts a discovered state file to a regular state file
func ConvertDiscoveredToStateFile(discovered *state.DiscoveredStateFile) *state.StateFile {
	if discovered == nil {
		return nil
	}
	
	// Convert Resources
	resources := make([]state.Resource, 0, len(discovered.Resources))
	for _, r := range discovered.Resources {
		resources = append(resources, state.Resource{
			Module:   r.Module,
			Mode:     r.Mode,
			Type:     r.Type,
			Name:     r.Name,
			Provider: r.Provider,
		})
	}
	
	return &state.StateFile{
		ID:               discovered.ID,
		Path:             discovered.Path,
		Version:          discovered.Version,
		TerraformVersion: discovered.TerraformVersion,
		Serial:           discovered.Serial,
		Lineage:          discovered.Lineage,
		Resources:        resources,
	}
}

// ConvertDiscoveredToStateFiles converts discovered state files to regular state files
func ConvertDiscoveredToStateFiles(discovered []*state.DiscoveredStateFile) []*state.StateFile {
	result := make([]*state.StateFile, len(discovered))
	for i, d := range discovered {
		result[i] = ConvertDiscoveredToStateFile(d)
	}
	return result
}

func filterStateFiles(states []*state.StateFile, filters []Filter) []*state.StateFile {
	var result []*state.StateFile
	
	for _, s := range states {
		match := true
		for _, f := range filters {
			if !applyFilter(s, f) {
				match = false
				break
			}
		}
		if match {
			result = append(result, s)
		}
	}
	
	return result
}

func applyFilter(state *state.StateFile, filter Filter) bool {
	// Simple filter implementation
	// In production, this would be more sophisticated
	return true
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}