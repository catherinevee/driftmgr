package remediation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// Engine provides unified remediation capabilities
type Engine struct {
	planner  *Planner
	executor *Executor
	rollback *RollbackManager
	safety   *SafetyManager
	mu       sync.RWMutex
}

// Plan represents a remediation plan
type Plan struct {
	ID         string                 `json:"id"`
	Status     string                 `json:"status"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	DriftItems []models.DriftItem     `json:"drift_items"`
	Actions    []Action               `json:"actions"`
	Impact     Impact                 `json:"impact"`
	Approval   *ApprovalStatus        `json:"approval,omitempty"`
	Execution  *ExecutionStatus       `json:"execution,omitempty"`
	Results    *Results               `json:"results,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Action represents a remediation action
type Action struct {
	ID            string                 `json:"id"`
	ResourceID    string                 `json:"resource_id"`
	ResourceType  string                 `json:"resource_type"`
	ActionType    string                 `json:"action_type"`
	Description   string                 `json:"description"`
	Parameters    map[string]interface{} `json:"parameters"`
	Risk          string                 `json:"risk"`
	EstimatedTime int                    `json:"estimated_time"`
	Dependencies  []string               `json:"dependencies"`
	Status        string                 `json:"status"`
	Error         string                 `json:"error,omitempty"`
}

// Impact describes remediation impact
type Impact struct {
	ResourcesAffected int                    `json:"resources_affected"`
	EstimatedDuration int                    `json:"estimated_duration"`
	RiskLevel         string                 `json:"risk_level"`
	CostImpact        float64                `json:"cost_impact"`
	ServiceImpact     []string               `json:"service_impact"`
	RequiresDowntime  bool                   `json:"requires_downtime"`
	Reversible        bool                   `json:"reversible"`
	Details           map[string]interface{} `json:"details"`
}

// ApprovalStatus represents approval status
type ApprovalStatus struct {
	Required     bool       `json:"required"`
	Status       string     `json:"status"`
	Approvers    []string   `json:"approvers"`
	ApprovedBy   []string   `json:"approved_by"`
	ApprovalTime *time.Time `json:"approval_time,omitempty"`
	Comments     []string   `json:"comments,omitempty"`
}

// ExecutionStatus represents execution status
type ExecutionStatus struct {
	Status         string     `json:"status"`
	StartTime      time.Time  `json:"start_time"`
	EndTime        *time.Time `json:"end_time,omitempty"`
	Progress       int        `json:"progress"`
	CurrentStep    string     `json:"current_step"`
	TotalSteps     int        `json:"total_steps"`
	CompletedSteps int        `json:"completed_steps"`
}

// Results represents remediation results
type Results struct {
	Success      bool                   `json:"success"`
	ItemsFixed   int                    `json:"items_fixed"`
	ItemsFailed  int                    `json:"items_failed"`
	Duration     time.Duration          `json:"duration"`
	Details      map[string]interface{} `json:"details"`
	RollbackInfo *RollbackInfo          `json:"rollback_info,omitempty"`
}

// RollbackInfo contains rollback information
type RollbackInfo struct {
	SnapshotID string                 `json:"snapshot_id"`
	PlanID     string                 `json:"plan_id"`
	CreatedAt  time.Time              `json:"created_at"`
	Available  bool                   `json:"available"`
	Snapshot   map[string]interface{} `json:"snapshot"`
	Steps      []string               `json:"steps"`
}

// Options configures remediation
type Options struct {
	Strategy       string                 `json:"strategy"`
	DryRun         bool                   `json:"dry_run"`
	AutoApprove    bool                   `json:"auto_approve"`
	Parallel       bool                   `json:"parallel"`
	MaxWorkers     int                    `json:"max_workers"`
	Timeout        time.Duration          `json:"timeout"`
	RollbackOnFail bool                   `json:"rollback_on_fail"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// NewEngine creates a new remediation engine
func NewEngine() *Engine {
	return &Engine{
		planner:  NewPlanner(),
		executor: NewExecutor(),
		rollback: NewRollbackManager(),
		safety:   NewSafetyManager(),
	}
}

// CreatePlan creates a remediation plan
func (e *Engine) CreatePlan(ctx context.Context, drifts []models.DriftItem, options Options) (*Plan, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Use planner to create plan
	plan, err := e.planner.CreatePlan(drifts, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	// Validate plan with safety manager
	if err := e.safety.ValidatePlan(plan); err != nil {
		return nil, fmt.Errorf("plan validation failed: %w", err)
	}

	// Calculate impact
	plan.Impact = e.calculateImpact(plan.Actions, drifts)

	return plan, nil
}

// ExecutePlan executes a remediation plan
func (e *Engine) ExecutePlan(ctx context.Context, plan *Plan, options Options) (*Results, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check approval if required
	if plan.Approval != nil && plan.Approval.Required && !options.AutoApprove {
		if plan.Approval.Status != "approved" {
			return nil, fmt.Errorf("plan requires approval")
		}
	}

	// Dry run if requested
	if options.DryRun {
		return e.simulateExecution(plan), nil
	}

	// Create snapshot for rollback
	var snapshot *RollbackInfo
	if options.RollbackOnFail {
		if e.rollback.CreateSnapshot("remediation", plan) == nil {
			snapshot = &RollbackInfo{
				SnapshotID: fmt.Sprintf("snapshot-%d", time.Now().Unix()),
				PlanID:     plan.ID,
				CreatedAt:  time.Now(),
				Available:  true,
			}
		}
	}

	// Execute plan
	results, err := e.executor.Execute(ctx, plan)
	if err != nil {
		if options.RollbackOnFail && snapshot != nil {
			e.rollback.Rollback(snapshot.SnapshotID)
		}
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	results.RollbackInfo = snapshot
	return results, nil
}

// calculateImpact calculates remediation impact
func (e *Engine) calculateImpact(actions []Action, drifts []models.DriftItem) Impact {
	impact := Impact{
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

	// Estimate cost impact
	impact.CostImpact = float64(len(actions)) * 10.0

	return impact
}

// simulateExecution simulates plan execution
func (e *Engine) simulateExecution(plan *Plan) *Results {
	return &Results{
		Success:     true,
		ItemsFixed:  len(plan.Actions),
		ItemsFailed: 0,
		Duration:    time.Duration(plan.Impact.EstimatedDuration) * time.Minute,
		Details: map[string]interface{}{
			"simulation": true,
			"message":    "Dry run completed successfully",
		},
	}
}
