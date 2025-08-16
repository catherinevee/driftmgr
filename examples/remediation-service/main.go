package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/catherinevee/driftmgr/internal/cache"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/monitoring"
	"github.com/catherinevee/driftmgr/internal/remediation"
	"github.com/catherinevee/driftmgr/internal/workflow"
	"github.com/gorilla/mux"
)

const (
	serviceName = "remediation-service"
	servicePort = "8083"
)

var (
	remediator     *remediation.EnhancedRemediationEngine
	workflowEngine *workflow.WorkflowEngine
	cacheManager   = cache.GetGlobalManager()
	logger         = monitoring.GetGlobalLogger()
)

func main() {
	// Initialize the remediator and workflow engine
	remediator = remediation.NewEnhancedRemediationEngine()
	workflowEngine = workflow.NewWorkflowEngine()

	// Set up router
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", handleHealth).Methods("GET")

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Remediation endpoints
	api.HandleFunc("/remediate", handleRemediate).Methods("POST")
	api.HandleFunc("/remediate/batch", handleRemediateBatch).Methods("POST")
	api.HandleFunc("/remediate/history", handleRemediateHistory).Methods("GET")
	api.HandleFunc("/remediate/{id}", handleGetRemediation).Methods("GET")
	api.HandleFunc("/remediate/{id}/status", handleGetRemediationStatus).Methods("GET")
	api.HandleFunc("/remediate/{id}/rollback", handleRemediateRollback).Methods("POST")

	// Strategy endpoints
	api.HandleFunc("/strategies", handleGetStrategies).Methods("GET")
	api.HandleFunc("/strategies/generate", handleGenerateStrategies).Methods("POST")
	api.HandleFunc("/strategies/{id}", handleGetStrategy).Methods("GET")
	api.HandleFunc("/strategies/{id}/test", handleTestStrategy).Methods("POST")

	// Approval endpoints
	api.HandleFunc("/approval/request", handleRequestApproval).Methods("POST")
	api.HandleFunc("/approval/{id}/approve", handleApproveRemediation).Methods("POST")
	api.HandleFunc("/approval/{id}/reject", handleRejectRemediation).Methods("POST")
	api.HandleFunc("/approval/pending", handleGetPendingApprovals).Methods("GET")

	// Start server
	logger.Info("Starting remediation service on port " + servicePort)
	log.Fatal(http.ListenAndServe(":"+servicePort, router))
}

// handleHealth handles health check requests
func handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"service":   serviceName,
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRemediate handles single remediation execution
func handleRemediate(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Provider        string                 `json:"provider"`
		Region          string                 `json:"region"`
		Drifts          []models.Drift         `json:"drifts"`
		Strategy        string                 `json:"strategy"`
		Options         map[string]interface{} `json:"options,omitempty"`
		RequireApproval bool                   `json:"require_approval"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if approval is required
	if request.RequireApproval {
		approvalID, err := remediator.RequestApproval(request.Provider, request.Region, request.Drifts, request.Strategy, request.Options)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to request approval: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"status":      "approval_required",
			"approval_id": approvalID,
			"message":     "Remediation requires approval",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Execute remediation
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Minute)
	defer cancel()

	result, err := remediator.ExecuteRemediation(ctx, request.Provider, request.Region, request.Drifts, request.Strategy, request.Options)
	if err != nil {
		http.Error(w, fmt.Sprintf("Remediation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleRemediateBatch handles batch remediation execution
func handleRemediateBatch(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Provider string                 `json:"provider"`
		Region   string                 `json:"region"`
		BatchID  string                 `json:"batch_id"`
		Drifts   []models.Drift         `json:"drifts"`
		Strategy string                 `json:"strategy"`
		Options  map[string]interface{} `json:"options,omitempty"`
		Parallel bool                   `json:"parallel"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Execute batch remediation
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()

	result, err := remediator.ExecuteBatchRemediation(ctx, request.BatchID, request.Provider, request.Region, request.Drifts, request.Strategy, request.Options, request.Parallel)
	if err != nil {
		http.Error(w, fmt.Sprintf("Batch remediation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleRemediateHistory retrieves remediation history
func handleRemediateHistory(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	provider := query.Get("provider")
	region := query.Get("region")
	limit := query.Get("limit")
	status := query.Get("status")

	history, err := remediator.GetRemediationHistory(provider, region, limit, status)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get history: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// handleGetRemediation retrieves a specific remediation by ID
func handleGetRemediation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	remediationID := vars["id"]

	remediation, err := remediator.GetRemediation(remediationID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Remediation not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(remediation)
}

// handleGetRemediationStatus retrieves the status of a remediation
func handleGetRemediationStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	remediationID := vars["id"]

	status, err := remediator.GetRemediationStatus(remediationID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Remediation not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleRemediateRollback handles remediation rollback
func handleRemediateRollback(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	remediationID := vars["id"]

	var request struct {
		Options map[string]interface{} `json:"options,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Execute rollback
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	result, err := remediator.RollbackRemediation(ctx, remediationID, request.Options)
	if err != nil {
		http.Error(w, fmt.Sprintf("Rollback failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleGetStrategies retrieves available remediation strategies
func handleGetStrategies(w http.ResponseWriter, r *http.Request) {
	strategies := remediator.GetAvailableStrategies()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(strategies)
}

// handleGenerateStrategies generates remediation strategies for drifts
func handleGenerateStrategies(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Provider string                 `json:"provider"`
		Region   string                 `json:"region"`
		Drifts   []models.Drift         `json:"drifts"`
		Options  map[string]interface{} `json:"options,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate strategies
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	strategies, err := remediator.GenerateStrategies(ctx, request.Provider, request.Region, request.Drifts, request.Options)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate strategies: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(strategies)
}

// handleGetStrategy retrieves a specific strategy
func handleGetStrategy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	strategyID := vars["id"]

	strategy, err := remediator.GetStrategy(strategyID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Strategy not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(strategy)
}

// handleTestStrategy tests a remediation strategy
func handleTestStrategy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	strategyID := vars["id"]

	var request struct {
		Provider string                 `json:"provider"`
		Region   string                 `json:"region"`
		Drifts   []models.Drift         `json:"drifts"`
		Options  map[string]interface{} `json:"options,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Test strategy
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	result, err := remediator.TestStrategy(ctx, strategyID, request.Provider, request.Region, request.Drifts, request.Options)
	if err != nil {
		http.Error(w, fmt.Sprintf("Strategy test failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleRequestApproval requests approval for a remediation
func handleRequestApproval(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Provider  string                 `json:"provider"`
		Region    string                 `json:"region"`
		Drifts    []models.Drift         `json:"drifts"`
		Strategy  string                 `json:"strategy"`
		Options   map[string]interface{} `json:"options,omitempty"`
		Approvers []string               `json:"approvers"`
		Priority  string                 `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	approvalID, err := remediator.RequestApproval(request.Provider, request.Region, request.Drifts, request.Strategy, request.Options)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to request approval: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"approval_id": approvalID,
		"status":      "pending_approval",
		"message":     "Approval request created successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleApproveRemediation approves a remediation
func handleApproveRemediation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	approvalID := vars["id"]

	var request struct {
		Approver string `json:"approver"`
		Comments string `json:"comments,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Approve remediation
	result, err := remediator.ApproveRemediation(approvalID, request.Approver, request.Comments)
	if err != nil {
		http.Error(w, fmt.Sprintf("Approval failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleRejectRemediation rejects a remediation
func handleRejectRemediation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	approvalID := vars["id"]

	var request struct {
		Approver string `json:"approver"`
		Reason   string `json:"reason"`
		Comments string `json:"comments,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Reject remediation
	result, err := remediator.RejectRemediation(approvalID, request.Approver, request.Reason, request.Comments)
	if err != nil {
		http.Error(w, fmt.Sprintf("Rejection failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleGetPendingApprovals retrieves pending approvals
func handleGetPendingApprovals(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	approver := query.Get("approver")
	priority := query.Get("priority")

	approvals, err := remediator.GetPendingApprovals(approver, priority)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get pending approvals: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(approvals)
}
