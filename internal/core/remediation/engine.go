package remediation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// Engine provides unified remediation capabilities
type Engine struct {
	planner  *Planner
	executor *Executor
	rollback *RollbackManager
	safety   *SafetyManager
	mu       sync.RWMutex
}

// Use types from types.go file

// All types are now defined in types.go file

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
	// Convert drifts to interface slice
	driftInterfaces := make([]interface{}, len(drifts))
	for i, d := range drifts {
		driftInterfaces[i] = d
	}
	plan, err := e.planner.CreatePlan(ctx, driftInterfaces, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	// Validate plan with safety manager
	if err := e.safety.ValidatePlan(ctx, plan); err != nil {
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
		state := make(map[string]interface{})
		state["plan"] = plan
		s, err := e.rollback.CreateSnapshot(ctx, plan.ID, state)
		if err == nil {
			snapshot = s
		}
	}

	// Execute plan
	results, err := e.executor.Execute(ctx, plan, &options)
	if err != nil {
		if options.RollbackOnFail && snapshot != nil {
			e.rollback.Rollback(ctx, snapshot.SnapshotID)
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
