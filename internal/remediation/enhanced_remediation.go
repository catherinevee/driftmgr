package remediation

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// EnhancedRemediationEngine provides advanced drift remediation capabilities
type EnhancedRemediationEngine struct {
	strategies map[string]EnhancedRemediationStrategy
	executors  map[string]EnhancedRemediationExecutor
	policies   map[string]EnhancedRemediationPolicy
	mu         sync.RWMutex
	history    []EnhancedRemediationAction
	metrics    *EnhancedRemediationMetrics
}

// EnhancedRemediation provides advanced drift remediation capabilities (alias for backward compatibility)
type EnhancedRemediation = EnhancedRemediationEngine

// EnhancedRemediationStrategy defines how to remediate a specific type of drift
type EnhancedRemediationStrategy interface {
	CanRemediate(drift *models.DriftResult) bool
	Remediate(ctx context.Context, drift *models.DriftResult) (*EnhancedRemediationResult, error)
	GetPriority() int
	GetName() string
}

// EnhancedRemediationExecutor executes remediation actions
type EnhancedRemediationExecutor interface {
	Execute(ctx context.Context, action *EnhancedRemediationAction) (*EnhancedRemediationResult, error)
	Validate(action *EnhancedRemediationAction) error
	GetSupportedActions() []string
}

// EnhancedRemediationPolicy defines remediation policies and rules
type EnhancedRemediationPolicy struct {
	ID          string                      `json:"id"`
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	Enabled     bool                        `json:"enabled"`
	Rules       []EnhancedRemediationRule   `json:"rules"`
	Actions     []EnhancedRemediationAction `json:"actions"`
	Conditions  map[string]interface{}      `json:"conditions"`
	Priority    int                         `json:"priority"`
	CreatedAt   time.Time                   `json:"created_at"`
	UpdatedAt   time.Time                   `json:"updated_at"`
}

// EnhancedRemediationRule defines a single remediation rule
type EnhancedRemediationRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Condition   string                 `json:"condition"` // JSONPath or expression
	Action      string                 `json:"action"`
	Parameters  map[string]interface{} `json:"parameters"`
	Enabled     bool                   `json:"enabled"`
	Priority    int                    `json:"priority"`
}

// EnhancedRemediationAction represents a remediation action to be executed
type EnhancedRemediationAction struct {
	ID          string                     `json:"id"`
	Type        string                     `json:"type"`
	ResourceID  string                     `json:"resource_id"`
	Provider    string                     `json:"provider"`
	Action      string                     `json:"action"`
	Parameters  map[string]interface{}     `json:"parameters"`
	Status      EnhancedRemediationStatus  `json:"status"`
	CreatedAt   time.Time                  `json:"created_at"`
	StartedAt   *time.Time                 `json:"started_at,omitempty"`
	CompletedAt *time.Time                 `json:"completed_at,omitempty"`
	Error       string                     `json:"error,omitempty"`
	Result      *EnhancedRemediationResult `json:"result,omitempty"`
}

// EnhancedRemediationStatus represents the status of a remediation action
type EnhancedRemediationStatus string

const (
	// EnhancedRemediationStatusPending represents a pending remediation action
	EnhancedRemediationStatusPending EnhancedRemediationStatus = "pending"
	// EnhancedRemediationStatusRunning represents a running remediation action
	EnhancedRemediationStatusRunning EnhancedRemediationStatus = "running"
	// EnhancedRemediationStatusCompleted represents a completed remediation action
	EnhancedRemediationStatusCompleted EnhancedRemediationStatus = "completed"
	// EnhancedRemediationStatusFailed represents a failed remediation action
	EnhancedRemediationStatusFailed EnhancedRemediationStatus = "failed"
	// EnhancedRemediationStatusCancelled represents a cancelled remediation action
	EnhancedRemediationStatusCancelled EnhancedRemediationStatus = "cancelled"
)

// EnhancedRemediationResult represents the result of a remediation action
type EnhancedRemediationResult struct {
	Success    bool                        `json:"success"`
	Message    string                      `json:"message"`
	Changes    []EnhancedRemediationChange `json:"changes,omitempty"`
	Metrics    map[string]interface{}      `json:"metrics,omitempty"`
	Duration   time.Duration               `json:"duration"`
	Timestamp  time.Time                   `json:"timestamp"`
	ResourceID string                      `json:"resource_id"`
	ActionID   string                      `json:"action_id"`
}

// EnhancedRemediationChange represents a change made during remediation
type EnhancedRemediationChange struct {
	Field       string      `json:"field"`
	OldValue    interface{} `json:"old_value,omitempty"`
	NewValue    interface{} `json:"new_value,omitempty"`
	ChangeType  string      `json:"change_type"`
	Description string      `json:"description"`
}

// EnhancedRemediationMetrics tracks remediation performance and statistics
type EnhancedRemediationMetrics struct {
	TotalActions      int                         `json:"total_actions"`
	SuccessfulActions int                         `json:"successful_actions"`
	FailedActions     int                         `json:"failed_actions"`
	AverageDuration   time.Duration               `json:"average_duration"`
	SuccessRate       float64                     `json:"success_rate"`
	ByProvider        map[string]int              `json:"by_provider"`
	ByActionType      map[string]int              `json:"by_action_type"`
	ByResourceType    map[string]int              `json:"by_resource_type"`
	RecentActions     []EnhancedRemediationAction `json:"recent_actions"`
	PerformanceData   map[string]interface{}      `json:"performance_data"`
	mu                sync.RWMutex
}

// NewEnhancedRemediation creates a new enhanced remediation system
func NewEnhancedRemediation() *EnhancedRemediation {
	return &EnhancedRemediation{
		strategies: make(map[string]EnhancedRemediationStrategy),
		executors:  make(map[string]EnhancedRemediationExecutor),
		policies:   make(map[string]EnhancedRemediationPolicy),
		history:    make([]EnhancedRemediationAction, 0),
		metrics:    NewEnhancedRemediationMetrics(),
	}
}

// RegisterStrategy registers a remediation strategy
func (er *EnhancedRemediation) RegisterStrategy(strategy EnhancedRemediationStrategy) {
	er.mu.Lock()
	defer er.mu.Unlock()
	er.strategies[strategy.GetName()] = strategy
}

// RegisterExecutor registers a remediation executor
func (er *EnhancedRemediation) RegisterExecutor(name string, executor EnhancedRemediationExecutor) {
	er.mu.Lock()
	defer er.mu.Unlock()
	er.executors[name] = executor
}

// AddPolicy adds a remediation policy
func (er *EnhancedRemediation) AddPolicy(policy EnhancedRemediationPolicy) error {
	er.mu.Lock()
	defer er.mu.Unlock()

	if policy.ID == "" {
		return fmt.Errorf("policy ID is required")
	}

	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()
	er.policies[policy.ID] = policy

	return nil
}

// GetPolicy gets a remediation policy by ID
func (er *EnhancedRemediation) GetPolicy(policyID string) (*EnhancedRemediationPolicy, bool) {
	er.mu.RLock()
	defer er.mu.RUnlock()

	policy, exists := er.policies[policyID]
	return &policy, exists
}

// ListPolicies lists all remediation policies
func (er *EnhancedRemediation) ListPolicies() []EnhancedRemediationPolicy {
	er.mu.RLock()
	defer er.mu.RUnlock()

	policies := make([]EnhancedRemediationPolicy, 0, len(er.policies))
	for _, policy := range er.policies {
		policies = append(policies, policy)
	}

	return policies
}

// RemediateDrift remediates a specific drift
func (er *EnhancedRemediation) RemediateDrift(ctx context.Context, drift *models.DriftResult) (*EnhancedRemediationResult, error) {
	// Find applicable strategies
	applicableStrategies := er.findApplicableStrategies(drift)
	if len(applicableStrategies) == 0 {
		return nil, fmt.Errorf("no applicable remediation strategy found for drift: %s", drift.DriftType)
	}

	// Try each strategy until one succeeds
	for _, strategy := range applicableStrategies {
		result, err := strategy.Remediate(ctx, drift)
		if err != nil {
			log.Printf("Strategy %s failed: %v", strategy.GetName(), err)
			continue
		}

		if result.Success {
			// Record the successful remediation
			er.recordRemediationAction(drift, strategy, result)
			return result, nil
		}
	}

	return nil, fmt.Errorf("all applicable remediation strategies failed for drift: %s", drift.DriftType)
}

// findApplicableStrategies finds strategies that can remediate a drift
func (er *EnhancedRemediation) findApplicableStrategies(drift *models.DriftResult) []EnhancedRemediationStrategy {
	var applicable []EnhancedRemediationStrategy

	er.mu.RLock()
	defer er.mu.RUnlock()

	for _, strategy := range er.strategies {
		if strategy.CanRemediate(drift) {
			applicable = append(applicable, strategy)
		}
	}

	return applicable
}

// recordRemediationAction records a successful remediation action
func (er *EnhancedRemediation) recordRemediationAction(drift *models.DriftResult, strategy EnhancedRemediationStrategy, result *EnhancedRemediationResult) {
	now := time.Now()
	action := EnhancedRemediationAction{
		ID:          fmt.Sprintf("remediation_%d", now.UnixNano()),
		Type:        strategy.GetName(),
		ResourceID:  drift.ResourceID,
		Provider:    drift.Provider,
		Action:      strategy.GetName(),
		Status:      EnhancedRemediationStatusCompleted,
		CreatedAt:   now,
		CompletedAt: &now,
		Result:      result,
	}

	er.recordAction(&action)
}

// recordAction records a remediation action
func (er *EnhancedRemediation) recordAction(action *EnhancedRemediationAction) {
	er.mu.Lock()
	defer er.mu.Unlock()

	er.history = append(er.history, *action)

	// Update metrics
	er.metrics.recordAction(action)
}

// GetHistory returns remediation history
func (er *EnhancedRemediation) GetHistory() []EnhancedRemediationAction {
	er.mu.RLock()
	defer er.mu.RUnlock()

	// Return a copy to avoid race conditions
	history := make([]EnhancedRemediationAction, len(er.history))
	copy(history, er.history)

	return history
}

// GetMetrics returns remediation metrics
func (er *EnhancedRemediation) GetMetrics() *EnhancedRemediationMetrics {
	er.mu.RLock()
	defer er.mu.RUnlock()

	return er.metrics
}

// NewEnhancedRemediationMetrics creates new remediation metrics
func NewEnhancedRemediationMetrics() *EnhancedRemediationMetrics {
	return &EnhancedRemediationMetrics{
		ByProvider:      make(map[string]int),
		ByActionType:    make(map[string]int),
		ByResourceType:  make(map[string]int),
		RecentActions:   make([]EnhancedRemediationAction, 0),
		PerformanceData: make(map[string]interface{}),
	}
}

// recordAction records a remediation action in metrics
func (rm *EnhancedRemediationMetrics) recordAction(action *EnhancedRemediationAction) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.TotalActions++

	if action.Status == EnhancedRemediationStatusCompleted {
		rm.SuccessfulActions++
	} else if action.Status == EnhancedRemediationStatusFailed {
		rm.FailedActions++
	}

	// Update provider metrics
	rm.ByProvider[action.Provider]++

	// Update action type metrics
	rm.ByActionType[action.Type]++

	// Calculate success rate
	if rm.TotalActions > 0 {
		rm.SuccessRate = float64(rm.SuccessfulActions) / float64(rm.TotalActions) * 100
	}

	// Update recent actions (keep last 100)
	rm.RecentActions = append(rm.RecentActions, *action)
	if len(rm.RecentActions) > 100 {
		rm.RecentActions = rm.RecentActions[1:]
	}

	// Update average duration if action completed
	if action.CompletedAt != nil && action.StartedAt != nil {
		duration := action.CompletedAt.Sub(*action.StartedAt)
		if rm.AverageDuration == 0 {
			rm.AverageDuration = duration
		} else {
			rm.AverageDuration = (rm.AverageDuration + duration) / 2
		}
	}
}

// GetSuccessRate returns the current success rate
func (rm *EnhancedRemediationMetrics) GetSuccessRate() float64 {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.SuccessRate
}

// GetTotalActions returns the total number of actions
func (rm *EnhancedRemediationMetrics) GetTotalActions() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.TotalActions
}

// GetSuccessfulActions returns the number of successful actions
func (rm *EnhancedRemediationMetrics) GetSuccessfulActions() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.SuccessfulActions
}

// GetFailedActions returns the number of failed actions
func (rm *EnhancedRemediationMetrics) GetFailedActions() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.FailedActions
}

// ActionResult represents the result of a remediation action
type ActionResult struct {
	Success   bool                        `json:"success"`
	Message   string                      `json:"message"`
	Changes   []EnhancedRemediationChange `json:"changes,omitempty"`
	Metadata  map[string]interface{}      `json:"metadata,omitempty"`
	Timestamp time.Time                   `json:"timestamp"`
}

// RollbackResult represents the result of a rollback operation
type RollbackResult struct {
	Success    bool      `json:"success"`
	Message    string    `json:"message"`
	RolledBack []string  `json:"rolled_back,omitempty"`
	Failed     []string  `json:"failed,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// ValidationResult represents the result of a validation operation
type ValidationResult struct {
	Valid     bool      `json:"valid"`
	Message   string    `json:"message"`
	Issues    []string  `json:"issues,omitempty"`
	Warnings  []string  `json:"warnings,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// TestReport represents a test report
type TestReport struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Status    string       `json:"status"`
	Results   []TestResult `json:"results,omitempty"`
	Summary   TestSummary  `json:"summary"`
	Timestamp time.Time    `json:"timestamp"`
}

// TestResult represents a single test result
type TestResult struct {
	Name     string                 `json:"name"`
	Status   string                 `json:"status"`
	Message  string                 `json:"message"`
	Duration time.Duration          `json:"duration"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TestSummary represents a test summary
type TestSummary struct {
	Total    int           `json:"total"`
	Passed   int           `json:"passed"`
	Failed   int           `json:"failed"`
	Skipped  int           `json:"skipped"`
	Duration time.Duration `json:"duration"`
}

// NewEnhancedRemediationEngine creates a new enhanced remediation engine
func NewEnhancedRemediationEngine() *EnhancedRemediationEngine {
	return &EnhancedRemediationEngine{
		strategies: make(map[string]EnhancedRemediationStrategy),
		executors:  make(map[string]EnhancedRemediationExecutor),
		policies:   make(map[string]EnhancedRemediationPolicy),
		history:    make([]EnhancedRemediationAction, 0),
		metrics:    NewEnhancedRemediationMetrics(),
	}
}
