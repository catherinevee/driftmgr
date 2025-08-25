package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/google/uuid"
)

// RemediationPlanRequest represents a request to create a remediation plan
type RemediationPlanRequest struct {
	DriftIDs []string               `json:"driftIds"`
	Strategy string                 `json:"strategy"` // "auto", "manual", "hybrid"
	Priority string                 `json:"priority"` // "critical", "high", "medium", "low"
	DryRun   bool                   `json:"dryRun"`
	Schedule *RemediationSchedule   `json:"schedule,omitempty"`
	Approval *ApprovalRequirements  `json:"approval,omitempty"`
	Rollback bool                   `json:"rollback"`
	Options  map[string]interface{} `json:"options"`
}

// RemediationSchedule defines when remediation should occur
type RemediationSchedule struct {
	Type      string    `json:"type"` // "immediate", "scheduled", "maintenance_window"
	StartTime time.Time `json:"startTime,omitempty"`
	EndTime   time.Time `json:"endTime,omitempty"`
	Timezone  string    `json:"timezone,omitempty"`
}

// ApprovalRequirements defines approval requirements
type ApprovalRequirements struct {
	Required  bool     `json:"required"`
	Approvers []string `json:"approvers"`
	Timeout   int      `json:"timeout"` // minutes
	MinCount  int      `json:"minCount"`
}

// RemediationPlan represents a remediation plan
type RemediationPlan struct {
	ID         string              `json:"id"`
	Status     string              `json:"status"`
	CreatedAt  time.Time           `json:"createdAt"`
	UpdatedAt  time.Time           `json:"updatedAt"`
	DriftItems []models.DriftItem  `json:"driftItems"`
	Actions    []RemediationAction `json:"actions"`
	Impact     RemediationImpact   `json:"impact"`
	Approval   *ApprovalStatus     `json:"approval,omitempty"`
	Execution  *ExecutionStatus    `json:"execution,omitempty"`
	Results    *RemediationResults `json:"results,omitempty"`
}

// RemediationAction represents a single remediation action
type RemediationAction struct {
	ID            string                 `json:"id"`
	ResourceID    string                 `json:"resourceId"`
	ResourceType  string                 `json:"resourceType"`
	ActionType    string                 `json:"actionType"` // "update", "delete", "create", "rollback"
	Description   string                 `json:"description"`
	Parameters    map[string]interface{} `json:"parameters"`
	Risk          string                 `json:"risk"`          // "low", "medium", "high"
	EstimatedTime int                    `json:"estimatedTime"` // seconds
	Dependencies  []string               `json:"dependencies"`
	Status        string                 `json:"status"`
	Error         string                 `json:"error,omitempty"`
}

// RemediationImpact describes the impact of remediation
type RemediationImpact struct {
	ResourcesAffected int                    `json:"resourcesAffected"`
	EstimatedDuration int                    `json:"estimatedDuration"` // minutes
	RiskLevel         string                 `json:"riskLevel"`
	CostImpact        float64                `json:"costImpact"`
	ServiceImpact     []string               `json:"serviceImpact"`
	RequiresDowntime  bool                   `json:"requiresDowntime"`
	Reversible        bool                   `json:"reversible"`
	Details           map[string]interface{} `json:"details"`
}

// ApprovalStatus represents the approval status of a plan
type ApprovalStatus struct {
	Status        string           `json:"status"` // "pending", "approved", "rejected"
	Approvals     []ApprovalRecord `json:"approvals"`
	RequiredCount int              `json:"requiredCount"`
	ApprovedCount int              `json:"approvedCount"`
	Deadline      time.Time        `json:"deadline"`
}

// ApprovalRecord represents an approval record
type ApprovalRecord struct {
	Approver  string    `json:"approver"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Comment   string    `json:"comment,omitempty"`
}

// ExecutionStatus represents the execution status
type ExecutionStatus struct {
	Status         string     `json:"status"` // "pending", "running", "completed", "failed", "cancelled"
	StartTime      time.Time  `json:"startTime"`
	EndTime        *time.Time `json:"endTime,omitempty"`
	Progress       int        `json:"progress"`
	CurrentStep    string     `json:"currentStep"`
	TotalSteps     int        `json:"totalSteps"`
	CompletedSteps int        `json:"completedSteps"`
}

// RemediationResults represents the results of remediation
type RemediationResults struct {
	Success      bool                   `json:"success"`
	ItemsFixed   int                    `json:"itemsFixed"`
	ItemsFailed  int                    `json:"itemsFailed"`
	Duration     int                    `json:"duration"` // seconds
	Details      map[string]interface{} `json:"details"`
	RollbackInfo *RollbackInfo          `json:"rollbackInfo,omitempty"`
}

// RollbackInfo contains information for rollback
type RollbackInfo struct {
	Available bool                   `json:"available"`
	Snapshot  map[string]interface{} `json:"snapshot"`
	Steps     []string               `json:"steps"`
}

// createRemediationPlan creates a remediation plan
func (s *EnhancedDashboardServer) createRemediationPlan(w http.ResponseWriter, r *http.Request) {
	var req RemediationPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get drift items
	allDrifts := s.dataStore.GetDrifts()
	selectedDrifts := make([]models.DriftItem, 0)

	for _, item := range allDrifts {
		if drift, ok := item.(models.DriftItem); ok {
			for _, id := range req.DriftIDs {
				if drift.ResourceID == id {
					selectedDrifts = append(selectedDrifts, drift)
					break
				}
			}
		}
	}

	if len(selectedDrifts) == 0 {
		http.Error(w, "No valid drift items found", http.StatusBadRequest)
		return
	}

	// Create remediation plan
	plan := s.buildRemediationPlan(selectedDrifts, req)

	// Store plan
	planKey := fmt.Sprintf("remediation:plan:%s", plan.ID)
	if err := s.storage.SaveState(context.Background(), planKey, plan); err != nil {
		s.logger.Printf("Failed to store remediation plan: %v", err)
	}

	// Also store in remediation history
	historyKey := fmt.Sprintf("remediation:history:%s", plan.ID)
	s.storage.SaveState(context.Background(), historyKey, map[string]interface{}{
		"planId":    plan.ID,
		"status":    plan.Status,
		"createdAt": plan.CreatedAt,
		"impact":    plan.Impact,
	})

	// If dry run, return plan without executing
	if req.DryRun {
		plan.Status = "dry_run"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(plan)
		return
	}

	// If approval required, set to pending
	if req.Approval != nil && req.Approval.Required {
		plan.Status = "pending_approval"
		plan.Approval = &ApprovalStatus{
			Status:        "pending",
			RequiredCount: req.Approval.MinCount,
			ApprovedCount: 0,
			Deadline:      time.Now().Add(time.Duration(req.Approval.Timeout) * time.Minute),
		}
	} else if req.Schedule != nil && req.Schedule.Type == "scheduled" {
		plan.Status = "scheduled"
	} else {
		// Execute immediately
		go s.executeRemediationPlan(plan)
		plan.Status = "executing"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plan)
}

// executeRemediation executes a remediation plan
func (s *EnhancedDashboardServer) executeRemediation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanID string `json:"planId"`
		Force  bool   `json:"force"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Retrieve plan from storage
	planKey := fmt.Sprintf("remediation:plan:%s", req.PlanID)
	var plan RemediationPlan
	if err := s.storage.LoadState(context.Background(), planKey, &plan); err != nil {
		http.Error(w, fmt.Sprintf("Plan not found: %s", req.PlanID), http.StatusNotFound)
		return
	}

	// Check if plan is approved (if required)
	if plan.Approval != nil && plan.Approval.Status != "approved" && !req.Force {
		http.Error(w, "Plan requires approval", http.StatusForbidden)
		return
	}

	// Create execution job
	job := s.jobManager.CreateJob("remediation")
	job.Result = map[string]interface{}{
		"planId": plan.ID,
		"force":  req.Force,
	}

	// Execute in background
	go s.executeRemediationPlan(&plan)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"jobId":  job.ID,
		"status": "started",
		"planId": plan.ID,
	})
}

// dryRunRemediation performs a dry run of remediation
func (s *EnhancedDashboardServer) dryRunRemediation(w http.ResponseWriter, r *http.Request) {
	var req RemediationPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Force dry run
	req.DryRun = true

	// Get drift items
	allDrifts := s.dataStore.GetDrifts()
	selectedDrifts := make([]models.DriftItem, 0)

	for _, item := range allDrifts {
		if drift, ok := item.(models.DriftItem); ok {
			for _, id := range req.DriftIDs {
				if drift.ResourceID == id {
					selectedDrifts = append(selectedDrifts, drift)
					break
				}
			}
		}
	}

	// Build plan
	plan := s.buildRemediationPlan(selectedDrifts, req)

	// Simulate execution
	simulation := s.simulateRemediation(plan)

	response := map[string]interface{}{
		"plan":       plan,
		"simulation": simulation,
		"dryRun":     true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getRemediationHistory retrieves remediation history
func (s *EnhancedDashboardServer) getRemediationHistory(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	status := r.URL.Query().Get("status")

	// Get all remediation history keys from storage
	historyKeys, err := s.storage.ListStates(context.Background(), "remediation:history:")
	if err != nil {
		http.Error(w, "Failed to retrieve history", http.StatusInternalServerError)
		return
	}

	// Load history items
	history := make([]interface{}, 0)
	for i, key := range historyKeys {
		if limit > 0 && i >= limit {
			break
		}

		var histItem map[string]interface{}
		if err := s.storage.LoadState(context.Background(), key, &histItem); err != nil {
			continue
		}

		// Apply status filter if needed
		if status != "" {
			if itemStatus, ok := histItem["status"].(string); ok && itemStatus != status {
				continue
			}
		}

		history = append(history, histItem)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// Helper functions

func (s *EnhancedDashboardServer) buildRemediationPlan(drifts []models.DriftItem, req RemediationPlanRequest) *RemediationPlan {
	plan := &RemediationPlan{
		ID:         uuid.New().String(),
		Status:     "created",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		DriftItems: drifts,
		Actions:    make([]RemediationAction, 0),
	}

	// Build actions based on drift items
	for _, drift := range drifts {
		action := s.createRemediationAction(drift, req.Strategy)
		plan.Actions = append(plan.Actions, action)
	}

	// Calculate impact
	plan.Impact = s.calculateRemediationImpact(plan.Actions, drifts)

	// Set execution details
	if req.Schedule != nil {
		plan.Execution = &ExecutionStatus{
			Status:     "scheduled",
			TotalSteps: len(plan.Actions),
		}
	}

	return plan
}

func (s *EnhancedDashboardServer) createRemediationAction(drift models.DriftItem, strategy string) RemediationAction {
	action := RemediationAction{
		ID:           uuid.New().String(),
		ResourceID:   drift.ResourceID,
		ResourceType: drift.ResourceType,
		Description:  fmt.Sprintf("Fix drift for %s", drift.ResourceID),
		Parameters:   make(map[string]interface{}),
		Status:       "pending",
	}

	// Determine action type based on drift type
	switch drift.DriftType {
	case "added":
		action.ActionType = "delete"
		action.Risk = "medium"
	case "deleted":
		action.ActionType = "create"
		action.Risk = "high"
	case "modified", "state_drift":
		action.ActionType = "update"
		action.Risk = "low"
	default:
		action.ActionType = "update"
		action.Risk = "medium"
	}

	// Set estimated time based on resource type
	switch drift.ResourceType {
	case "aws_instance", "azure_virtual_machine":
		action.EstimatedTime = 300 // 5 minutes
	case "aws_s3_bucket", "azure_storage_account":
		action.EstimatedTime = 60 // 1 minute
	default:
		action.EstimatedTime = 120 // 2 minutes
	}

	// Add parameters based on strategy
	if strategy == "auto" {
		action.Parameters["automatic"] = true
		action.Parameters["validation"] = "post"
	}

	return action
}

func (s *EnhancedDashboardServer) calculateRemediationImpact(actions []RemediationAction, drifts []models.DriftItem) RemediationImpact {
	impact := RemediationImpact{
		ResourcesAffected: len(actions),
		EstimatedDuration: 0,
		RiskLevel:         "low",
		CostImpact:        0,
		ServiceImpact:     make([]string, 0),
		RequiresDowntime:  false,
		Reversible:        true,
		Details:           make(map[string]interface{}),
	}

	highRiskCount := 0
	for _, action := range actions {
		impact.EstimatedDuration += action.EstimatedTime

		if action.Risk == "high" {
			highRiskCount++
		}

		// Check if action requires downtime
		if action.ActionType == "delete" || action.ActionType == "create" {
			impact.RequiresDowntime = true
		}
	}

	// Calculate overall risk level
	if highRiskCount > len(actions)/2 {
		impact.RiskLevel = "high"
	} else if highRiskCount > 0 {
		impact.RiskLevel = "medium"
	}

	// Convert duration to minutes
	impact.EstimatedDuration = impact.EstimatedDuration / 60

	// Estimate cost impact (simplified)
	impact.CostImpact = float64(len(actions)) * 10.0

	// Identify affected services
	serviceMap := make(map[string]bool)
	for _, drift := range drifts {
		serviceMap[drift.Provider] = true
	}
	for service := range serviceMap {
		impact.ServiceImpact = append(impact.ServiceImpact, service)
	}

	return impact
}

func (s *EnhancedDashboardServer) executeRemediationPlan(plan *RemediationPlan) {
	ctx := context.Background()

	// Update status
	plan.Status = "executing"
	plan.Execution = &ExecutionStatus{
		Status:         "running",
		StartTime:      time.Now(),
		TotalSteps:     len(plan.Actions),
		CompletedSteps: 0,
	}

	// Broadcast start
	s.broadcast <- map[string]interface{}{
		"type":   "remediation_started",
		"planId": plan.ID,
	}

	// Execute each action
	successCount := 0
	failCount := 0

	for i, action := range plan.Actions {
		plan.Execution.CurrentStep = action.Description
		plan.Execution.Progress = (i * 100) / len(plan.Actions)

		// Execute action
		err := s.executeRemediationAction(ctx, &action)
		if err != nil {
			action.Status = "failed"
			action.Error = err.Error()
			failCount++
		} else {
			action.Status = "completed"
			successCount++
		}

		plan.Execution.CompletedSteps++

		// Broadcast progress
		s.broadcast <- map[string]interface{}{
			"type":     "remediation_progress",
			"planId":   plan.ID,
			"progress": plan.Execution.Progress,
			"action":   action,
		}
	}

	// Update final status
	endTime := time.Now()
	plan.Execution.EndTime = &endTime
	plan.Execution.Status = "completed"
	plan.Execution.Progress = 100

	plan.Results = &RemediationResults{
		Success:     failCount == 0,
		ItemsFixed:  successCount,
		ItemsFailed: failCount,
		Duration:    int(endTime.Sub(plan.Execution.StartTime).Seconds()),
		Details: map[string]interface{}{
			"actions_completed": successCount,
			"actions_failed":    failCount,
		},
	}

	if failCount == 0 {
		plan.Status = "completed"
	} else if successCount == 0 {
		plan.Status = "failed"
	} else {
		plan.Status = "partial"
	}

	// Store updated plan
	planKey := fmt.Sprintf("remediation:plan:%s", plan.ID)
	if err := s.storage.SaveState(context.Background(), planKey, plan); err != nil {
		s.logger.Printf("Failed to update remediation plan: %v", err)
	}

	// Update history
	historyKey := fmt.Sprintf("remediation:history:%s", plan.ID)
	s.storage.SaveState(context.Background(), historyKey, map[string]interface{}{
		"planId":    plan.ID,
		"status":    plan.Status,
		"updatedAt": time.Now(),
		"results":   plan.Results,
	})

	// Broadcast completion
	s.broadcast <- map[string]interface{}{
		"type":    "remediation_completed",
		"planId":  plan.ID,
		"status":  plan.Status,
		"results": plan.Results,
	}
}

func (s *EnhancedDashboardServer) executeRemediationAction(ctx context.Context, action *RemediationAction) error {
	// Simulate remediation action
	// In production, this would call actual remediation services

	switch action.ActionType {
	case "update":
		// Update resource to match expected state
		return s.updateResourceState(ctx, action.ResourceID, action.Parameters)
	case "delete":
		// Delete unexpected resource
		return s.deleteResourceForRemediation(ctx, action.ResourceID)
	case "create":
		// Create missing resource
		return s.createResourceForRemediation(ctx, action.ResourceType, action.Parameters)
	case "rollback":
		// Rollback to previous state
		return s.rollbackResource(ctx, action.ResourceID, action.Parameters)
	default:
		return fmt.Errorf("unknown action type: %s", action.ActionType)
	}
}

func (s *EnhancedDashboardServer) simulateRemediation(plan *RemediationPlan) map[string]interface{} {
	simulation := map[string]interface{}{
		"success_probability": 0.95,
		"estimated_duration":  plan.Impact.EstimatedDuration,
		"potential_issues": []string{
			"Network connectivity might be affected",
			"Some services may experience brief interruption",
		},
		"rollback_available": true,
		"validation_steps": []string{
			"Pre-flight checks",
			"Resource state verification",
			"Dependency validation",
			"Post-remediation testing",
		},
	}

	return simulation
}

func (s *EnhancedDashboardServer) updateResourceState(ctx context.Context, resourceID string, parameters map[string]interface{}) error {
	// Implement actual resource update logic
	// This would use the appropriate cloud SDK
	return nil
}

func (s *EnhancedDashboardServer) deleteResourceForRemediation(ctx context.Context, resourceID string) error {
	// Implement actual resource deletion logic
	return nil
}

func (s *EnhancedDashboardServer) createResourceForRemediation(ctx context.Context, resourceType string, parameters map[string]interface{}) error {
	// Implement actual resource creation logic
	return nil
}

func (s *EnhancedDashboardServer) rollbackResource(ctx context.Context, resourceID string, parameters map[string]interface{}) error {
	// Implement rollback logic
	return nil
}
