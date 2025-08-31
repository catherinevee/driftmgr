package remediation

import (
	"context"
)

// Planner creates remediation plans
type Planner struct{}

// NewPlanner creates a new planner
func NewPlanner() *Planner {
	return &Planner{}
}

// CreatePlan creates a remediation plan
func (p *Planner) CreatePlan(ctx context.Context, drifts []interface{}, options Options) (*Plan, error) {
	plan := &Plan{
		ID:         "plan-1",
		Name:       "Remediation Plan",
		Status:     "pending",
		Actions:    []Action{},
		DriftItems: drifts,
		Metadata:   make(map[string]interface{}),
	}
	return plan, nil
}

// SafetyPolicy represents a safety policy
type SafetyPolicy struct {
	Name        string
	Description string
	Enabled     bool
}