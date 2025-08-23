package remediation

import (
	"fmt"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/google/uuid"
	"time"
)

// Planner creates remediation plans
type Planner struct {
	strategies map[string]Strategy
}

// Strategy defines a remediation strategy
type Strategy interface {
	CreateActions(drifts []models.DriftItem) []Action
	Name() string
}

// NewPlanner creates a new remediation planner
func NewPlanner() *Planner {
	return &Planner{
		strategies: map[string]Strategy{
			"auto":   &AutoStrategy{},
			"manual": &ManualStrategy{},
			"hybrid": &HybridStrategy{},
		},
	}
}

// CreatePlan creates a remediation plan from drift items
func (p *Planner) CreatePlan(drifts []models.DriftItem, options Options) (*Plan, error) {
	strategy, exists := p.strategies[options.Strategy]
	if !exists {
		strategy = p.strategies["auto"]
	}

	actions := strategy.CreateActions(drifts)

	plan := &Plan{
		ID:         uuid.New().String(),
		Status:     "created",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		DriftItems: drifts,
		Actions:    actions,
		Metadata:   options.Metadata,
	}

	// Set approval requirements based on risk
	if p.requiresApproval(actions) && !options.AutoApprove {
		plan.Approval = &ApprovalStatus{
			Required: true,
			Status:   "pending",
		}
	}

	return plan, nil
}

// requiresApproval determines if plan requires approval
func (p *Planner) requiresApproval(actions []Action) bool {
	for _, action := range actions {
		if action.Risk == "high" || action.Risk == "critical" {
			return true
		}
		if action.ActionType == "delete" {
			return true
		}
	}
	return false
}

// AutoStrategy implements automatic remediation strategy
type AutoStrategy struct{}

func (s *AutoStrategy) Name() string {
	return "auto"
}

func (s *AutoStrategy) CreateActions(drifts []models.DriftItem) []Action {
	actions := make([]Action, 0, len(drifts))

	for _, drift := range drifts {
		action := Action{
			ID:           uuid.New().String(),
			ResourceID:   drift.ResourceID,
			ResourceType: drift.ResourceType,
			Description:  fmt.Sprintf("Auto-remediate %s", drift.ResourceName),
			Parameters:   make(map[string]interface{}),
			Status:       "pending",
		}

		// Determine action type based on drift type
		switch drift.DriftType {
		case "added":
			action.ActionType = "delete"
			action.Risk = "medium"
			action.EstimatedTime = 60
		case "deleted":
			action.ActionType = "create"
			action.Risk = "high"
			action.EstimatedTime = 180
		case "modified", "state_drift":
			action.ActionType = "update"
			action.Risk = "low"
			action.EstimatedTime = 120
		default:
			action.ActionType = "update"
			action.Risk = "medium"
			action.EstimatedTime = 120
		}

		// Adjust risk based on resource type
		if drift.ResourceType == "aws_instance" || drift.ResourceType == "azure_virtual_machine" {
			action.EstimatedTime = 300
			if action.Risk == "low" {
				action.Risk = "medium"
			}
		}

		actions = append(actions, action)
	}

	return actions
}

// ManualStrategy implements manual remediation strategy
type ManualStrategy struct{}

func (s *ManualStrategy) Name() string {
	return "manual"
}

func (s *ManualStrategy) CreateActions(drifts []models.DriftItem) []Action {
	actions := make([]Action, 0, len(drifts))

	for _, drift := range drifts {
		action := Action{
			ID:            uuid.New().String(),
			ResourceID:    drift.ResourceID,
			ResourceType:  drift.ResourceType,
			ActionType:    "manual_review",
			Description:   fmt.Sprintf("Manual review required for %s", drift.ResourceName),
			Risk:          "low",
			EstimatedTime: 600,
			Parameters: map[string]interface{}{
				"requires_approval": true,
				"manual_steps": []string{
					"Review resource configuration",
					"Determine appropriate action",
					"Execute remediation manually",
					"Verify results",
				},
			},
			Status: "pending",
		}

		actions = append(actions, action)
	}

	return actions
}

// HybridStrategy implements hybrid remediation strategy
type HybridStrategy struct{}

func (s *HybridStrategy) Name() string {
	return "hybrid"
}

func (s *HybridStrategy) CreateActions(drifts []models.DriftItem) []Action {
	actions := make([]Action, 0, len(drifts))
	autoStrategy := &AutoStrategy{}
	manualStrategy := &ManualStrategy{}

	for _, drift := range drifts {
		// Use auto strategy for low-risk drifts
		if drift.Severity == "low" || drift.Severity == "medium" {
			autoActions := autoStrategy.CreateActions([]models.DriftItem{drift})
			if len(autoActions) > 0 {
				actions = append(actions, autoActions[0])
			}
		} else {
			// Use manual strategy for high-risk drifts
			manualActions := manualStrategy.CreateActions([]models.DriftItem{drift})
			if len(manualActions) > 0 {
				actions = append(actions, manualActions[0])
			}
		}
	}

	return actions
}
