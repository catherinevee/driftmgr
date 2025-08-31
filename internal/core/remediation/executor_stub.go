// +build !full_remediation

package remediation

import (
	"context"
	"fmt"
	"time"
)

// Executor handles remediation execution (stub version)
type Executor struct{}

// NewExecutor creates a new remediation executor
func NewExecutor() *Executor {
	return &Executor{}
}

// Execute executes a remediation plan
func (e *Executor) Execute(ctx context.Context, plan *Plan, options *Options) (*Results, error) {
	// Stub implementation
	results := &Results{
		Success:    true,
		ItemsFixed: len(plan.Actions),
		Duration:   1 * time.Second,
		Details:    make(map[string]interface{}),
	}
	return results, nil
}

// ValidatePlan validates a remediation plan
func (e *Executor) ValidatePlan(plan *Plan) error {
	if plan == nil {
		return fmt.Errorf("plan cannot be nil")
	}
	return nil
}

// GetProviderClient returns a provider client (stub)
func (e *Executor) GetProviderClient(provider string) (interface{}, error) {
	return nil, nil
}