package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/cache"
	"github.com/catherinevee/driftmgr/internal/core/remediation"
	"github.com/catherinevee/driftmgr/internal/events"
	"github.com/catherinevee/driftmgr/internal/jobs"
	"github.com/google/uuid"
)

// RemediationService provides unified remediation operations
type RemediationService struct {
	executor     *remediation.Executor
	driftService *DriftService
	cache        cache.Cache
	eventBus     *events.EventBus
	jobQueue     *jobs.Queue
	mu           sync.RWMutex
}

// NewRemediationService creates a new remediation service
func NewRemediationService(
	executor *remediation.Executor,
	driftService *DriftService,
	cache cache.Cache,
	eventBus *events.EventBus,
	jobQueue *jobs.Queue,
) *RemediationService {
	return &RemediationService{
		executor:     executor,
		driftService: driftService,
		cache:        cache,
		eventBus:     eventBus,
		jobQueue:     jobQueue,
	}
}

// RemediationRequest represents a request to remediate drift
type RemediationRequest struct {
	DriftReportID string   `json:"drift_report_id,omitempty"`
	DriftItemIDs  []string `json:"drift_item_ids,omitempty"`
	Provider      string   `json:"provider,omitempty"`
	DryRun        bool     `json:"dry_run"`
	Force         bool     `json:"force"`
	Async         bool     `json:"async"`
	Strategy      string   `json:"strategy,omitempty"` // auto, manual, selective
}

// RemediationResponse represents the response from remediation
type RemediationResponse struct {
	JobID       string              `json:"job_id,omitempty"`
	Status      string              `json:"status"`
	Progress    int                 `json:"progress"`
	Message     string              `json:"message"`
	Plan        *RemediationPlan    `json:"plan,omitempty"`
	Results     *RemediationResults `json:"results,omitempty"`
	StartedAt   time.Time           `json:"started_at"`
	EndedAt     *time.Time          `json:"ended_at,omitempty"`
}

// RemediationPlan represents a plan for remediation
type RemediationPlan struct {
	ID          string             `json:"id"`
	Actions     []RemediationAction `json:"actions"`
	TotalSteps  int                `json:"total_steps"`
	EstimatedTime string           `json:"estimated_time"`
	RiskLevel   string             `json:"risk_level"`
	Approval    *ApprovalRequest   `json:"approval,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
}

// RemediationAction represents a single remediation action
type RemediationAction struct {
	ID           string                 `json:"id"`
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	Action       string                 `json:"action"` // create, update, delete, import
	Description  string                 `json:"description"`
	Commands     []string               `json:"commands,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	RiskLevel    string                 `json:"risk_level"`
	Reversible   bool                   `json:"reversible"`
	Dependencies []string               `json:"dependencies,omitempty"`
}

// RemediationResults represents the results of remediation
type RemediationResults struct {
	TotalActions   int                    `json:"total_actions"`
	SuccessCount   int                    `json:"success_count"`
	FailureCount   int                    `json:"failure_count"`
	SkippedCount   int                    `json:"skipped_count"`
	Actions        []ActionResult         `json:"actions"`
	RollbackNeeded bool                   `json:"rollback_needed"`
	Errors         []string               `json:"errors,omitempty"`
	CompletedAt    time.Time              `json:"completed_at"`
}

// ActionResult represents the result of a single action
type ActionResult struct {
	ActionID    string    `json:"action_id"`
	Status      string    `json:"status"` // success, failed, skipped
	Message     string    `json:"message"`
	Error       string    `json:"error,omitempty"`
	ExecutedAt  time.Time `json:"executed_at"`
	Duration    string    `json:"duration"`
}

// ApprovalRequest represents an approval request
type ApprovalRequest struct {
	ID            string    `json:"id"`
	RequiredLevel string    `json:"required_level"` // admin, operator
	Reason        string    `json:"reason"`
	RequestedAt   time.Time `json:"requested_at"`
	ApprovedBy    string    `json:"approved_by,omitempty"`
	ApprovedAt    *time.Time `json:"approved_at,omitempty"`
	Status        string    `json:"status"` // pending, approved, rejected
}

// StartRemediation initiates remediation
func (s *RemediationService) StartRemediation(ctx context.Context, req RemediationRequest) (*RemediationResponse, error) {
	// Validate request
	if err := s.validateRemediationRequest(req); err != nil {
		return nil, fmt.Errorf("invalid remediation request: %w", err)
	}

	// Generate job ID
	jobID := uuid.New().String()

	// Create remediation plan
	plan, err := s.createRemediationPlan(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create remediation plan: %w", err)
	}

	// Check if approval is needed
	if plan.Approval != nil && plan.Approval.Status == "pending" {
		// Return plan for approval
		return &RemediationResponse{
			JobID:     jobID,
			Status:    "pending_approval",
			Message:   "Remediation plan requires approval",
			Plan:      plan,
			StartedAt: time.Now(),
		}, nil
	}

	// Emit remediation started event
	s.eventBus.Publish(events.Event{
		Type:      events.RemediationStarted,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"job_id":       jobID,
			"total_actions": len(plan.Actions),
			"dry_run":      req.DryRun,
		},
	})

	// If dry run, return plan without executing
	if req.DryRun {
		return &RemediationResponse{
			JobID:     jobID,
			Status:    "dry_run_completed",
			Message:   "Dry run completed - no changes made",
			Plan:      plan,
			StartedAt: time.Now(),
		}, nil
	}

	// If async, create a job and return immediately
	if req.Async {
		job := &jobs.Job{
			ID:        jobID,
			Type:      jobs.Remediation,
			Status:    jobs.StatusPending,
			CreatedAt: time.Now(),
			Data: map[string]interface{}{
				"request": req,
				"plan":    plan,
			},
		}

		if err := s.jobQueue.Enqueue(job); err != nil {
			return nil, fmt.Errorf("failed to enqueue remediation job: %w", err)
		}

		// Start processing in background
		go s.processRemediationJob(context.Background(), job)

		return &RemediationResponse{
			JobID:     jobID,
			Status:    "running",
			Progress:  0,
			Message:   "Remediation job started",
			Plan:      plan,
			StartedAt: time.Now(),
		}, nil
	}

	// Synchronous remediation
	return s.executeRemediation(ctx, jobID, req, plan)
}

// createRemediationPlan creates a remediation plan
func (s *RemediationService) createRemediationPlan(ctx context.Context, req RemediationRequest) (*RemediationPlan, error) {
	plan := &RemediationPlan{
		ID:        uuid.New().String(),
		Actions:   []RemediationAction{},
		CreatedAt: time.Now(),
	}

	// Get drift items to remediate
	var driftItems []DriftItem

	if req.DriftReportID != "" {
		report, err := s.driftService.GetDriftReport(ctx, req.DriftReportID)
		if err != nil {
			return nil, fmt.Errorf("failed to get drift report: %w", err)
		}
		
		// Filter by drift item IDs if specified
		if len(req.DriftItemIDs) > 0 {
			idMap := make(map[string]bool)
			for _, id := range req.DriftItemIDs {
				idMap[id] = true
			}
			
			for _, drift := range report.Drifts {
				if idMap[drift.ResourceID] {
					driftItems = append(driftItems, drift)
				}
			}
		} else {
			// Use all remediable drifts
			for _, drift := range report.Drifts {
				if drift.Remediable {
					driftItems = append(driftItems, drift)
				}
			}
		}
	}

	// Create actions for each drift item
	for _, drift := range driftItems {
		action := s.createRemediationAction(drift)
		plan.Actions = append(plan.Actions, action)
	}

	// Calculate risk and estimates
	plan.TotalSteps = len(plan.Actions)
	plan.RiskLevel = s.calculatePlanRisk(plan.Actions)
	plan.EstimatedTime = s.estimateExecutionTime(plan.Actions)

	// Check if approval is needed
	if s.requiresApproval(plan) && !req.Force {
		plan.Approval = &ApprovalRequest{
			ID:            uuid.New().String(),
			RequiredLevel: s.getRequiredApprovalLevel(plan),
			Reason:        fmt.Sprintf("High risk remediation with %d actions", plan.TotalSteps),
			RequestedAt:   time.Now(),
			Status:        "pending",
		}
	}

	// Cache the plan
	s.cache.Set(fmt.Sprintf("remediation:plan:%s", plan.ID), plan, 1*time.Hour)

	return plan, nil
}

// createRemediationAction creates a remediation action for a drift item
func (s *RemediationService) createRemediationAction(drift DriftItem) RemediationAction {
	action := RemediationAction{
		ID:           uuid.New().String(),
		ResourceID:   drift.ResourceID,
		ResourceType: drift.ResourceType,
		Parameters:   make(map[string]interface{}),
		Reversible:   true,
	}

	switch drift.DriftType {
	case "missing":
		action.Action = "create"
		action.Description = fmt.Sprintf("Create missing %s: %s", drift.ResourceType, drift.ResourceName)
		action.Parameters = drift.ExpectedState
		action.RiskLevel = "medium"
		
	case "modified":
		action.Action = "update"
		action.Description = fmt.Sprintf("Update %s: %s", drift.ResourceType, drift.ResourceName)
		action.Parameters = drift.StateDiff
		action.RiskLevel = s.calculateActionRisk(drift)
		
	case "unmanaged":
		action.Action = "import"
		action.Description = fmt.Sprintf("Import unmanaged %s: %s", drift.ResourceType, drift.ResourceName)
		action.Parameters = drift.ActualState
		action.RiskLevel = "low"
		action.Reversible = false
		
	default:
		action.Action = "unknown"
		action.RiskLevel = "high"
	}

	// Generate terraform commands if applicable
	action.Commands = s.generateTerraformCommands(action)

	return action
}

// executeRemediation executes the remediation plan
func (s *RemediationService) executeRemediation(ctx context.Context, jobID string, req RemediationRequest, plan *RemediationPlan) (*RemediationResponse, error) {
	startTime := time.Now()
	response := &RemediationResponse{
		JobID:     jobID,
		Status:    "running",
		Progress:  0,
		Message:   "Executing remediation plan",
		Plan:      plan,
		StartedAt: startTime,
	}

	results := &RemediationResults{
		TotalActions: len(plan.Actions),
		Actions:      []ActionResult{},
		Errors:       []string{},
	}

	// Execute each action
	for i, action := range plan.Actions {
		// Update progress
		progress := ((i + 1) * 100) / len(plan.Actions)
		response.Progress = progress
		
		// Emit progress event
		s.eventBus.Publish(events.Event{
			Type: events.RemediationStarted,
			Data: map[string]interface{}{
				"job_id":   jobID,
				"progress": progress,
				"action":   action.Description,
			},
		})

		// Execute action
		actionStart := time.Now()
		result := s.executeAction(ctx, action)
		result.Duration = time.Since(actionStart).String()

		results.Actions = append(results.Actions, result)

		if result.Status == "success" {
			results.SuccessCount++
		} else if result.Status == "failed" {
			results.FailureCount++
			results.Errors = append(results.Errors, result.Error)
			
			// Check if we should continue or rollback
			if !req.Force && s.shouldRollback(results) {
				results.RollbackNeeded = true
				break
			}
		} else {
			results.SkippedCount++
		}
	}

	// Complete remediation
	endTime := time.Now()
	results.CompletedAt = endTime
	response.EndedAt = &endTime
	response.Results = results
	response.Progress = 100

	if results.FailureCount > 0 {
		response.Status = "completed_with_errors"
		response.Message = fmt.Sprintf("Remediation completed with %d errors", results.FailureCount)
	} else {
		response.Status = "completed"
		response.Message = fmt.Sprintf("Successfully executed %d actions", results.SuccessCount)
	}

	// Cache results
	s.cache.Set(fmt.Sprintf("remediation:results:%s", jobID), results, 1*time.Hour)

	// Emit completion event
	s.eventBus.Publish(events.Event{
		Type:      events.RemediationCompleted,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"job_id":  jobID,
			"success": results.SuccessCount,
			"failed":  results.FailureCount,
		},
	})

	return response, nil
}

// executeAction executes a single remediation action
func (s *RemediationService) executeAction(ctx context.Context, action RemediationAction) ActionResult {
	result := ActionResult{
		ActionID:   action.ID,
		ExecutedAt: time.Now(),
	}

	// Simulate action execution
	// In real implementation, this would call the actual provider APIs
	switch action.Action {
	case "create":
		// Create resource
		result.Status = "success"
		result.Message = fmt.Sprintf("Created %s", action.ResourceID)
		
	case "update":
		// Update resource
		result.Status = "success"
		result.Message = fmt.Sprintf("Updated %s", action.ResourceID)
		
	case "delete":
		// Delete resource
		result.Status = "success"
		result.Message = fmt.Sprintf("Deleted %s", action.ResourceID)
		
	case "import":
		// Import resource
		result.Status = "success"
		result.Message = fmt.Sprintf("Imported %s", action.ResourceID)
		
	default:
		result.Status = "skipped"
		result.Message = fmt.Sprintf("Unknown action: %s", action.Action)
	}

	return result
}

// processRemediationJob processes an async remediation job
func (s *RemediationService) processRemediationJob(ctx context.Context, job *jobs.Job) {
	data, ok := job.Data.(map[string]interface{})
	if !ok {
		job.Status = jobs.StatusFailed
		job.Error = fmt.Errorf("invalid job data")
		s.jobQueue.UpdateJob(job)
		return
	}

	req, _ := data["request"].(RemediationRequest)
	plan, _ := data["plan"].(*RemediationPlan)

	job.Status = jobs.StatusRunning
	job.StartedAt = timePtr(time.Now())
	s.jobQueue.UpdateJob(job)

	response, err := s.executeRemediation(ctx, job.ID, req, plan)
	
	if err != nil {
		job.Status = jobs.StatusFailed
		job.Error = err
		s.eventBus.Publish(events.Event{
			Type: events.RemediationFailed,
			Data: map[string]interface{}{
				"job_id": job.ID,
				"error":  err.Error(),
			},
		})
	} else {
		job.Status = jobs.StatusCompleted
		job.Result = response
	}
	
	job.CompletedAt = timePtr(time.Now())
	s.jobQueue.UpdateJob(job)
}

// ApproveRemediation approves a remediation plan
func (s *RemediationService) ApproveRemediation(ctx context.Context, planID string, approver string) error {
	cacheKey := fmt.Sprintf("remediation:plan:%s", planID)
	cached, found := s.cache.Get(cacheKey)
	if !found {
		return fmt.Errorf("remediation plan not found: %s", planID)
	}

	plan, ok := cached.(*RemediationPlan)
	if !ok {
		return fmt.Errorf("invalid plan data")
	}

	if plan.Approval == nil {
		return fmt.Errorf("plan does not require approval")
	}

	now := time.Now()
	plan.Approval.ApprovedBy = approver
	plan.Approval.ApprovedAt = &now
	plan.Approval.Status = "approved"

	// Update cache
	s.cache.Set(cacheKey, plan, 1*time.Hour)

	return nil
}

// Helper functions

func (s *RemediationService) validateRemediationRequest(req RemediationRequest) error {
	if req.DriftReportID == "" && len(req.DriftItemIDs) == 0 {
		return fmt.Errorf("drift_report_id or drift_item_ids required")
	}
	return nil
}

func (s *RemediationService) calculatePlanRisk(actions []RemediationAction) string {
	highRiskCount := 0
	for _, action := range actions {
		if action.RiskLevel == "high" {
			highRiskCount++
		}
	}
	
	if highRiskCount > 5 || float64(highRiskCount)/float64(len(actions)) > 0.3 {
		return "high"
	} else if highRiskCount > 0 {
		return "medium"
	}
	return "low"
}

func (s *RemediationService) estimateExecutionTime(actions []RemediationAction) string {
	// Estimate 30 seconds per action
	seconds := len(actions) * 30
	minutes := seconds / 60
	if minutes < 1 {
		return fmt.Sprintf("%d seconds", seconds)
	}
	return fmt.Sprintf("%d minutes", minutes)
}

func (s *RemediationService) requiresApproval(plan *RemediationPlan) bool {
	return plan.RiskLevel == "high" || plan.TotalSteps > 10
}

func (s *RemediationService) getRequiredApprovalLevel(plan *RemediationPlan) string {
	if plan.RiskLevel == "high" {
		return "admin"
	}
	return "operator"
}

func (s *RemediationService) calculateActionRisk(drift DriftItem) string {
	if drift.SecurityImpact {
		return "high"
	} else if drift.CostImpact > 100 {
		return "medium"
	}
	return "low"
}

func (s *RemediationService) generateTerraformCommands(action RemediationAction) []string {
	var commands []string
	
	switch action.Action {
	case "create":
		commands = append(commands, fmt.Sprintf("terraform apply -target=%s.%s", action.ResourceType, action.ResourceID))
	case "update":
		commands = append(commands, fmt.Sprintf("terraform apply -target=%s.%s", action.ResourceType, action.ResourceID))
	case "delete":
		commands = append(commands, fmt.Sprintf("terraform destroy -target=%s.%s", action.ResourceType, action.ResourceID))
	case "import":
		commands = append(commands, fmt.Sprintf("terraform import %s.%s %s", action.ResourceType, action.ResourceID, action.ResourceID))
	}
	
	return commands
}

func (s *RemediationService) shouldRollback(results *RemediationResults) bool {
	// Rollback if more than 20% of actions failed
	failureRate := float64(results.FailureCount) / float64(results.TotalActions)
	return failureRate > 0.2
}