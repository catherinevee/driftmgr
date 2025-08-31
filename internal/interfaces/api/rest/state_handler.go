package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/catherinevee/driftmgr/internal/services"
	"github.com/gorilla/mux"
)

// StateHandler handles state-related API requests
type StateHandler struct {
	service *services.StateService
}

// NewStateHandler creates a new state handler
func NewStateHandler(service *services.StateService) *StateHandler {
	return &StateHandler{
		service: service,
	}
}

// RegisterRoutes registers state routes
func (h *StateHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/state/discover", h.DiscoverStateFiles).Methods("GET", "POST")
	router.HandleFunc("/api/v1/state/list", h.ListStateFiles).Methods("GET")
	router.HandleFunc("/api/v1/state/import", h.ImportStateFile).Methods("POST")
	router.HandleFunc("/api/v1/state/analyze", h.AnalyzeStateFiles).Methods("POST")
	router.HandleFunc("/api/v1/state/compare", h.CompareStateFiles).Methods("POST")
	router.HandleFunc("/api/v1/state/details", h.GetStateFileDetails).Methods("GET")
	router.HandleFunc("/api/v1/state/{id}", h.DeleteStateFile).Methods("DELETE")
}

// DiscoverStateFiles handles GET/POST /api/v1/state/discover
func (h *StateHandler) DiscoverStateFiles(w http.ResponseWriter, r *http.Request) {
	var paths []string
	
	if r.Method == "POST" {
		var req struct {
			Paths []string `json:"paths"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		paths = req.Paths
	}

	stateFiles, err := h.service.DiscoverStateFiles(r.Context(), paths)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"files": stateFiles,
		"count": len(stateFiles),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListStateFiles handles GET /api/v1/state/list
func (h *StateHandler) ListStateFiles(w http.ResponseWriter, r *http.Request) {
	stateFiles, err := h.service.ListStateFiles(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"files": stateFiles,
		"count": len(stateFiles),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ImportStateFile handles POST /api/v1/state/import
func (h *StateHandler) ImportStateFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	stateFile, err := h.service.ImportStateFile(r.Context(), req.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stateFile)
}

// AnalyzeStateFiles handles POST /api/v1/state/analyze
func (h *StateHandler) AnalyzeStateFiles(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FileIDs []string `json:"file_ids"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// If no file IDs provided, analyze all cached files
	if len(req.FileIDs) == 0 {
		stateFiles, _ := h.service.ListStateFiles(r.Context())
		for _, sf := range stateFiles {
			req.FileIDs = append(req.FileIDs, sf.ID)
		}
	}

	analysis, err := h.service.AnalyzeStateFiles(r.Context(), req.FileIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// CompareStateFiles handles POST /api/v1/state/compare
func (h *StateHandler) CompareStateFiles(w http.ResponseWriter, r *http.Request) {
	var req struct {
		File1ID string `json:"file1_id"`
		File2ID string `json:"file2_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.File1ID == "" || req.File2ID == "" {
		http.Error(w, "file1_id and file2_id are required", http.StatusBadRequest)
		return
	}

	comparison, err := h.service.CompareStateFiles(r.Context(), req.File1ID, req.File2ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comparison)
}

// GetStateFileDetails handles GET /api/v1/state/details
func (h *StateHandler) GetStateFileDetails(w http.ResponseWriter, r *http.Request) {
	fileID := r.URL.Query().Get("id")
	filePath := r.URL.Query().Get("path")

	if fileID == "" && filePath == "" {
		http.Error(w, "id or path is required", http.StatusBadRequest)
		return
	}

	var stateFile *services.StateFile
	var err error

	if fileID != "" {
		stateFile, err = h.service.GetStateFile(r.Context(), fileID)
	} else {
		stateFile, err = h.service.ImportStateFile(r.Context(), filePath)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stateFile)
}

// DeleteStateFile handles DELETE /api/v1/state/{id}
func (h *StateHandler) DeleteStateFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]

	if fileID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteStateFile(r.Context(), fileID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "State file deleted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}