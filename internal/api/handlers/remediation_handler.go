package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/catherinevee/driftmgr/internal/services"
	"github.com/gorilla/mux"
)

// RemediationHandler handles remediation-related API requests
type RemediationHandler struct {
	service *services.RemediationService
}

// NewRemediationHandler creates a new remediation handler
func NewRemediationHandler(service *services.RemediationService) *RemediationHandler {
	return &RemediationHandler{
		service: service,
	}
}

// RegisterRoutes registers remediation routes
func (h *RemediationHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/remediation/start", h.StartRemediation).Methods("POST")
	router.HandleFunc("/api/v1/remediation/status", h.GetRemediationStatus).Methods("GET")
	router.HandleFunc("/api/v1/remediation/plan", h.GetRemediationPlan).Methods("GET")
	router.HandleFunc("/api/v1/remediation/approve", h.ApproveRemediation).Methods("POST")
	router.HandleFunc("/api/v1/remediation/results", h.GetRemediationResults).Methods("GET")
}

// StartRemediation handles POST /api/v1/remediation/start
func (h *RemediationHandler) StartRemediation(w http.ResponseWriter, r *http.Request) {
	var req services.RemediationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set async to true for API calls
	req.Async = true

	response, err := h.service.StartRemediation(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetRemediationStatus handles GET /api/v1/remediation/status
func (h *RemediationHandler) GetRemediationStatus(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		http.Error(w, "job_id is required", http.StatusBadRequest)
		return
	}

	// This would get status from the job queue
	response := map[string]interface{}{
		"job_id":  jobID,
		"status":  "running",
		"message": "Remediation in progress",
		"progress": 50,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetRemediationPlan handles GET /api/v1/remediation/plan
func (h *RemediationHandler) GetRemediationPlan(w http.ResponseWriter, r *http.Request) {
	planID := r.URL.Query().Get("plan_id")
	if planID == "" {
		http.Error(w, "plan_id is required", http.StatusBadRequest)
		return
	}

	// This would fetch the plan from cache
	plan := &services.RemediationPlan{
		ID: planID,
		Actions: []services.RemediationAction{
			{
				ID:           "action-1",
				ResourceID:   "i-1234567890",
				ResourceType: "aws_instance",
				Action:       "update",
				Description:  "Update instance type from t2.micro to t2.small",
				RiskLevel:    "medium",
				Reversible:   true,
			},
		},
		TotalSteps:    1,
		EstimatedTime: "2 minutes",
		RiskLevel:     "medium",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plan)
}

// ApproveRemediation handles POST /api/v1/remediation/approve
func (h *RemediationHandler) ApproveRemediation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanID   string `json:"plan_id"`
		Approver string `json:"approver"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.PlanID == "" {
		http.Error(w, "plan_id is required", http.StatusBadRequest)
		return
	}

	if err := h.service.ApproveRemediation(r.Context(), req.PlanID, req.Approver); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Remediation plan approved",
		"plan_id": req.PlanID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetRemediationResults handles GET /api/v1/remediation/results
func (h *RemediationHandler) GetRemediationResults(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		http.Error(w, "job_id is required", http.StatusBadRequest)
		return
	}

	// This would fetch results from cache
	results := &services.RemediationResults{
		TotalActions: 5,
		SuccessCount: 4,
		FailureCount: 1,
		SkippedCount: 0,
		Actions: []services.ActionResult{
			{
				ActionID: "action-1",
				Status:   "success",
				Message:  "Successfully updated instance type",
				Duration: "30s",
			},
		},
		RollbackNeeded: false,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}