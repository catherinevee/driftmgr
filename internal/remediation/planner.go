package remediation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/graph"
	"github.com/catherinevee/driftmgr/internal/drift/detector"
	"github.com/catherinevee/driftmgr/internal/state"
	"github.com/google/uuid"
)

// RemediationPlanner creates remediation plans for drift
type RemediationPlanner struct {
	validator    *ActionValidator
	riskAnalyzer *RiskAnalyzer
	depGraph     *graph.DependencyGraph
	config       *PlannerConfig
}

// PlannerConfig contains configuration for the planner
type PlannerConfig struct {
	AutoApprove           bool          `json:"auto_approve"`
	MaxParallelActions    int           `json:"max_parallel_actions"`
	RequireApprovalFor    []ActionType  `json:"require_approval_for"`
	SafeMode              bool          `json:"safe_mode"`
	DryRun                bool          `json:"dry_run"`
	BackupBeforeAction    bool          `json:"backup_before_action"`
	MaxRetries            int           `json:"max_retries"`
	ActionTimeout         time.Duration `json:"action_timeout"`
}

// RemediationPlan represents a plan to remediate drift
type RemediationPlan struct {
	ID                string               `json:"id"`
	Name              string               `json:"name"`
	Description       string               `json:"description"`
	CreatedAt         time.Time            `json:"created_at"`
	Actions           []RemediationAction  `json:"actions"`
	EstimatedDuration time.Duration        `json:"estimated_duration"`
	RiskLevel         RiskLevel            `json:"risk_level"`
	RequiresApproval  bool                 `json:"requires_approval"`
	Dependencies      map[string][]string  `json:"dependencies"`
	ExecutionOrder    []string             `json:"execution_order"`
	RollbackPlan      *RollbackPlan        `json:"rollback_plan,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// RemediationAction represents a single remediation action
type RemediationAction struct {
	ID           string                 `json:"id"`
	Type         ActionType             `json:"type"`
	Resource     string                 `json:"resource"`
	ResourceType string                 `json:"resource_type"`
	Provider     string                 `json:"provider"`
	Description  string                 `json:"description"`
	Command      string                 `json:"command,omitempty"`
	Parameters   map[string]interface{} `json:"parameters"`
	PreChecks    []PreCheck             `json:"pre_checks,omitempty"`
	PostChecks   []PostCheck            `json:"post_checks,omitempty"`
	DependsOn    []string               `json:"depends_on,omitempty"`
	Timeout      time.Duration          `json:"timeout"`
	Retryable    bool                   `json:"retryable"`
	RiskLevel    RiskLevel              `json:"risk_level"`
	Rollback     *RollbackAction        `json:"rollback,omitempty"`
}

// ActionType defines the type of remediation action
type ActionType string

const (
	ActionTypeImport   ActionType = "import"
	ActionTypeUpdate   ActionType = "update"
	ActionTypeDelete   ActionType = "delete"
	ActionTypeCreate   ActionType = "create"
	ActionTypeRefresh  ActionType = "refresh"
	ActionTypeMove     ActionType = "move"
	ActionTypeReplace  ActionType = "replace"
	ActionTypeTaint    ActionType = "taint"
	ActionTypeUntaint  ActionType = "untaint"
)

// RiskLevel indicates the risk level of an action
type RiskLevel int

const (
	RiskLevelLow RiskLevel = iota
	RiskLevelMedium
	RiskLevelHigh
	RiskLevelCritical
)

// PreCheck defines a check to run before an action
type PreCheck struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Check       string `json:"check"`
	Required    bool   `json:"required"`
}

// PostCheck defines a check to run after an action
type PostCheck struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Check       string        `json:"check"`
	Timeout     time.Duration `json:"timeout"`
}

// RollbackPlan defines how to rollback a remediation plan
type RollbackPlan struct {
	ID          string            `json:"id"`
	Description string            `json:"description"`
	Actions     []RollbackAction  `json:"actions"`
	Automatic   bool              `json:"automatic"`
}

// RollbackAction defines a single rollback action
type RollbackAction struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Command     string                 `json:"command"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// NewRemediationPlanner creates a new remediation planner
func NewRemediationPlanner(depGraph *graph.DependencyGraph) *RemediationPlanner {
	return &RemediationPlanner{
		validator:    NewActionValidator(),
		riskAnalyzer: NewRiskAnalyzer(),
		depGraph:     depGraph,
		config: &PlannerConfig{
			AutoApprove:        false,
			MaxParallelActions: 5,
			SafeMode:           true,
			BackupBeforeAction: true,
			MaxRetries:         3,
			ActionTimeout:      5 * time.Minute,
			RequireApprovalFor: []ActionType{
				ActionTypeDelete,
				ActionTypeReplace,
			},
		},
	}
}

// CreatePlan creates a remediation plan from drift results
func (rp *RemediationPlanner) CreatePlan(ctx context.Context, driftReport *detector.DriftReport, 
	state *state.TerraformState) (*RemediationPlan, error) {
	
	plan := &RemediationPlan{
		ID:           uuid.New().String(),
		Name:         fmt.Sprintf("Remediation Plan %s", time.Now().Format("2006-01-02 15:04:05")),
		Description:  fmt.Sprintf("Plan to remediate %d drifted resources", len(driftReport.DriftResults)),
		CreatedAt:    time.Now(),
		Actions:      make([]RemediationAction, 0),
		Dependencies: make(map[string][]string),
	}

	// Generate actions for each drift result
	for _, drift := range driftReport.DriftResults {
		actions, err := rp.generateActions(drift, state)
		if err != nil {
			return nil, fmt.Errorf("failed to generate actions for %s: %w", drift.Resource, err)
		}
		plan.Actions = append(plan.Actions, actions...)
	}

	// Validate all actions
	for _, action := range plan.Actions {
		if err := rp.validator.Validate(action); err != nil {
			return nil, fmt.Errorf("action validation failed for %s: %w", action.Resource, err)
		}
	}

	// Build dependencies
	rp.buildDependencies(plan)

	// Determine execution order
	executionOrder, err := rp.determineExecutionOrder(plan)
	if err != nil {
		return nil, fmt.Errorf("failed to determine execution order: %w", err)
	}
	plan.ExecutionOrder = executionOrder

	// Analyze risk
	plan.RiskLevel = rp.riskAnalyzer.AnalyzePlan(plan)

	// Determine if approval is required
	plan.RequiresApproval = rp.requiresApproval(plan)

	// Estimate duration
	plan.EstimatedDuration = rp.estimateDuration(plan)

	// Create rollback plan
	if rp.config.SafeMode {
		plan.RollbackPlan = rp.createRollbackPlan(plan, state)
	}

	return plan, nil
}

// generateActions generates remediation actions for a drift result
func (rp *RemediationPlanner) generateActions(drift detector.DriftResult, 
	state *state.TerraformState) ([]RemediationAction, error) {
	
	actions := make([]RemediationAction, 0)

	switch drift.DriftType {
	case detector.ResourceMissing:
		// Resource exists in state but not in cloud - needs to be created
		action := rp.createImportAction(drift)
		actions = append(actions, action)

	case detector.ConfigurationDrift:
		// Resource exists but configuration differs
		if rp.shouldReplace(drift) {
			// Resource needs replacement
			actions = append(actions, rp.createReplaceActions(drift)...)
		} else {
			// Resource can be updated in place
			action := rp.createUpdateAction(drift)
			actions = append(actions, action)
		}

	case detector.ResourceUnmanaged:
		// Resource exists in cloud but not in state
		if rp.config.AutoApprove {
			action := rp.createImportAction(drift)
			actions = append(actions, action)
		} else {
			// Create import or delete action based on policy
			action := rp.createManageUnmanagedAction(drift)
			actions = append(actions, action)
		}

	case detector.ResourceOrphaned:
		// Resource in state has no corresponding cloud resource
		action := rp.createRemoveFromStateAction(drift)
		actions = append(actions, action)
	}

	return actions, nil
}

// createImportAction creates an import action
func (rp *RemediationPlanner) createImportAction(drift detector.DriftResult) RemediationAction {
	resourceID := ""
	if drift.ActualState != nil {
		if id, ok := drift.ActualState["id"].(string); ok {
			resourceID = id
		}
	}

	return RemediationAction{
		ID:           uuid.New().String(),
		Type:         ActionTypeImport,
		Resource:     drift.Resource,
		ResourceType: drift.ResourceType,
		Provider:     drift.Provider,
		Description:  fmt.Sprintf("Import %s into Terraform state", drift.Resource),
		Command:      fmt.Sprintf("terraform import %s %s", drift.Resource, resourceID),
		Parameters: map[string]interface{}{
			"resource_address": drift.Resource,
			"resource_id":      resourceID,
		},
		PreChecks: []PreCheck{
			{
				Name:        "resource_exists",
				Description: "Verify resource exists in cloud",
				Check:       fmt.Sprintf("cloud_resource_exists(%s)", resourceID),
				Required:    true,
			},
		},
		PostChecks: []PostCheck{
			{
				Name:        "state_updated",
				Description: "Verify resource imported to state",
				Check:       fmt.Sprintf("state_contains(%s)", drift.Resource),
				Timeout:     30 * time.Second,
			},
		},
		Timeout:   2 * time.Minute,
		Retryable: true,
		RiskLevel: RiskLevelLow,
		Rollback: &RollbackAction{
			ID:          uuid.New().String(),
			Type:        "state_rm",
			Description: fmt.Sprintf("Remove %s from state", drift.Resource),
			Command:     fmt.Sprintf("terraform state rm %s", drift.Resource),
		},
	}
}

// createUpdateAction creates an update action
func (rp *RemediationPlanner) createUpdateAction(drift detector.DriftResult) RemediationAction {
	return RemediationAction{
		ID:           uuid.New().String(),
		Type:         ActionTypeUpdate,
		Resource:     drift.Resource,
		ResourceType: drift.ResourceType,
		Provider:     drift.Provider,
		Description:  fmt.Sprintf("Update %s to match desired state", drift.Resource),
		Command:      fmt.Sprintf("terraform apply -target=%s", drift.Resource),
		Parameters: map[string]interface{}{
			"resource_address": drift.Resource,
			"differences":      drift.Differences,
			"desired_state":    drift.DesiredState,
		},
		PreChecks: []PreCheck{
			{
				Name:        "plan_review",
				Description: "Review terraform plan",
				Check:       fmt.Sprintf("terraform plan -target=%s", drift.Resource),
				Required:    true,
			},
		},
		PostChecks: []PostCheck{
			{
				Name:        "drift_resolved",
				Description: "Verify drift is resolved",
				Check:       fmt.Sprintf("no_drift(%s)", drift.Resource),
				Timeout:     1 * time.Minute,
			},
		},
		Timeout:   5 * time.Minute,
		Retryable: true,
		RiskLevel: rp.calculateUpdateRisk(drift),
		Rollback: &RollbackAction{
			ID:          uuid.New().String(),
			Type:        "restore",
			Description: fmt.Sprintf("Restore %s to previous state", drift.Resource),
			Parameters: map[string]interface{}{
				"previous_state": drift.ActualState,
			},
		},
	}
}

// createReplaceActions creates replace actions (delete + create)
func (rp *RemediationPlanner) createReplaceActions(drift detector.DriftResult) []RemediationAction {
	actions := make([]RemediationAction, 0)

	// Taint action
	taintAction := RemediationAction{
		ID:           uuid.New().String(),
		Type:         ActionTypeTaint,
		Resource:     drift.Resource,
		ResourceType: drift.ResourceType,
		Provider:     drift.Provider,
		Description:  fmt.Sprintf("Mark %s for replacement", drift.Resource),
		Command:      fmt.Sprintf("terraform taint %s", drift.Resource),
		Parameters: map[string]interface{}{
			"resource_address": drift.Resource,
		},
		Timeout:   30 * time.Second,
		Retryable: true,
		RiskLevel: RiskLevelMedium,
	}
	actions = append(actions, taintAction)

	// Apply action
	applyAction := RemediationAction{
		ID:           uuid.New().String(),
		Type:         ActionTypeReplace,
		Resource:     drift.Resource,
		ResourceType: drift.ResourceType,
		Provider:     drift.Provider,
		Description:  fmt.Sprintf("Replace %s with new resource", drift.Resource),
		Command:      fmt.Sprintf("terraform apply -target=%s -auto-approve", drift.Resource),
		Parameters: map[string]interface{}{
			"resource_address": drift.Resource,
			"desired_state":    drift.DesiredState,
		},
		DependsOn: []string{taintAction.ID},
		PreChecks: []PreCheck{
			{
				Name:        "backup_created",
				Description: "Ensure backup is created",
				Check:       fmt.Sprintf("backup_exists(%s)", drift.Resource),
				Required:    true,
			},
		},
		PostChecks: []PostCheck{
			{
				Name:        "resource_recreated",
				Description: "Verify resource is recreated",
				Check:       fmt.Sprintf("resource_healthy(%s)", drift.Resource),
				Timeout:     2 * time.Minute,
			},
		},
		Timeout:   10 * time.Minute,
		Retryable: false,
		RiskLevel: RiskLevelHigh,
		Rollback: &RollbackAction{
			ID:          uuid.New().String(),
			Type:        "untaint",
			Description: fmt.Sprintf("Untaint %s if replacement fails", drift.Resource),
			Command:     fmt.Sprintf("terraform untaint %s", drift.Resource),
		},
	}
	actions = append(actions, applyAction)

	return actions
}

// createManageUnmanagedAction creates action for unmanaged resources
func (rp *RemediationPlanner) createManageUnmanagedAction(drift detector.DriftResult) RemediationAction {
	// Default to import action for unmanaged resources
	action := rp.createImportAction(drift)
	action.Description = fmt.Sprintf("Import unmanaged resource %s", drift.Resource)
	action.RiskLevel = RiskLevelMedium
	
	// Add additional checks for unmanaged resources
	action.PreChecks = append(action.PreChecks, PreCheck{
		Name:        "no_conflicts",
		Description: "Ensure no naming conflicts",
		Check:       fmt.Sprintf("no_resource_conflicts(%s)", drift.Resource),
		Required:    true,
	})
	
	return action
}

// createRemoveFromStateAction creates action to remove orphaned resource from state
func (rp *RemediationPlanner) createRemoveFromStateAction(drift detector.DriftResult) RemediationAction {
	return RemediationAction{
		ID:           uuid.New().String(),
		Type:         ActionTypeDelete,
		Resource:     drift.Resource,
		ResourceType: drift.ResourceType,
		Provider:     drift.Provider,
		Description:  fmt.Sprintf("Remove orphaned resource %s from state", drift.Resource),
		Command:      fmt.Sprintf("terraform state rm %s", drift.Resource),
		Parameters: map[string]interface{}{
			"resource_address": drift.Resource,
		},
		PreChecks: []PreCheck{
			{
				Name:        "no_dependencies",
				Description: "Ensure no resources depend on this",
				Check:       fmt.Sprintf("no_dependencies(%s)", drift.Resource),
				Required:    true,
			},
		},
		Timeout:   30 * time.Second,
		Retryable: true,
		RiskLevel: RiskLevelLow,
	}
}

// shouldReplace determines if a resource should be replaced
func (rp *RemediationPlanner) shouldReplace(drift detector.DriftResult) bool {
	// Check if any differences require replacement
	forceNewFields := []string{
		"ami",
		"instance_type",
		"availability_zone",
		"subnet_id",
		"vpc_id",
		"engine",
		"engine_version",
		"node_type",
		"location",
		"region",
		"zone",
	}

	for _, diff := range drift.Differences {
		for _, field := range forceNewFields {
			if strings.Contains(strings.ToLower(diff.Path), field) {
				return true
			}
		}
	}

	return false
}

// calculateUpdateRisk calculates risk level for update action
func (rp *RemediationPlanner) calculateUpdateRisk(drift detector.DriftResult) RiskLevel {
	// Base risk on severity and type of changes
	if drift.Severity == detector.SeverityCritical {
		return RiskLevelHigh
	}

	// Check for high-risk fields
	highRiskFields := []string{
		"security_group",
		"iam",
		"policy",
		"encryption",
		"backup",
		"deletion_protection",
	}

	for _, diff := range drift.Differences {
		for _, field := range highRiskFields {
			if strings.Contains(strings.ToLower(diff.Path), field) {
				return RiskLevelHigh
			}
		}
	}

	if drift.Severity == detector.SeverityHigh {
		return RiskLevelMedium
	}

	return RiskLevelLow
}

// buildDependencies builds action dependencies
func (rp *RemediationPlanner) buildDependencies(plan *RemediationPlan) {
	// Build dependency map from resource dependencies
	if rp.depGraph != nil {
		for _, action := range plan.Actions {
			deps := rp.depGraph.GetDependencies(action.Resource)
			if len(deps) > 0 {
				plan.Dependencies[action.ID] = deps
			}
		}
	}

	// Add explicit action dependencies
	for _, action := range plan.Actions {
		if len(action.DependsOn) > 0 {
			if existing, ok := plan.Dependencies[action.ID]; ok {
				plan.Dependencies[action.ID] = append(existing, action.DependsOn...)
			} else {
				plan.Dependencies[action.ID] = action.DependsOn
			}
		}
	}
}

// determineExecutionOrder determines the order of action execution
func (rp *RemediationPlanner) determineExecutionOrder(plan *RemediationPlan) ([]string, error) {
	// Build action graph
	actionGraph := make(map[string][]string)
	for _, action := range plan.Actions {
		actionGraph[action.ID] = plan.Dependencies[action.ID]
	}

	// Topological sort
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	order := make([]string, 0)

	var visit func(id string) error
	visit = func(id string) error {
		visited[id] = true
		recStack[id] = true

		for _, dep := range actionGraph[id] {
			if !visited[dep] {
				if err := visit(dep); err != nil {
					return err
				}
			} else if recStack[dep] {
				return fmt.Errorf("circular dependency detected: %s -> %s", id, dep)
			}
		}

		recStack[id] = false
		order = append([]string{id}, order...)
		return nil
	}

	for _, action := range plan.Actions {
		if !visited[action.ID] {
			if err := visit(action.ID); err != nil {
				return nil, err
			}
		}
	}

	return order, nil
}

// requiresApproval checks if plan requires approval
func (rp *RemediationPlanner) requiresApproval(plan *RemediationPlan) bool {
	if rp.config.AutoApprove {
		return false
	}

	// Check risk level
	if plan.RiskLevel >= RiskLevelHigh {
		return true
	}

	// Check for specific action types
	for _, action := range plan.Actions {
		for _, requireApproval := range rp.config.RequireApprovalFor {
			if action.Type == requireApproval {
				return true
			}
		}
	}

	return false
}

// estimateDuration estimates plan execution duration
func (rp *RemediationPlanner) estimateDuration(plan *RemediationPlan) time.Duration {
	var totalDuration time.Duration

	// Group actions by parallel execution
	groups := rp.groupParallelActions(plan)

	for _, group := range groups {
		var maxDuration time.Duration
		for _, actionID := range group {
			for _, action := range plan.Actions {
				if action.ID == actionID {
					if action.Timeout > maxDuration {
						maxDuration = action.Timeout
					}
					break
				}
			}
		}
		totalDuration += maxDuration
	}

	// Add buffer for overhead
	totalDuration += time.Duration(len(plan.Actions)) * 5 * time.Second

	return totalDuration
}

// groupParallelActions groups actions that can run in parallel
func (rp *RemediationPlanner) groupParallelActions(plan *RemediationPlan) [][]string {
	groups := make([][]string, 0)
	processed := make(map[string]bool)

	for _, actionID := range plan.ExecutionOrder {
		if processed[actionID] {
			continue
		}

		// Find actions that can run in parallel
		group := []string{actionID}
		processed[actionID] = true

		for _, otherID := range plan.ExecutionOrder {
			if processed[otherID] {
				continue
			}

			// Check if actions can run in parallel
			if rp.canRunInParallel(actionID, otherID, plan) {
				group = append(group, otherID)
				processed[otherID] = true

				if len(group) >= rp.config.MaxParallelActions {
					break
				}
			}
		}

		groups = append(groups, group)
	}

	return groups
}

// canRunInParallel checks if two actions can run in parallel
func (rp *RemediationPlanner) canRunInParallel(id1, id2 string, plan *RemediationPlan) bool {
	// Check if either depends on the other
	deps1 := plan.Dependencies[id1]
	deps2 := plan.Dependencies[id2]

	for _, dep := range deps1 {
		if dep == id2 {
			return false
		}
	}

	for _, dep := range deps2 {
		if dep == id1 {
			return false
		}
	}

	return true
}

// createRollbackPlan creates a rollback plan
func (rp *RemediationPlanner) createRollbackPlan(plan *RemediationPlan, state *state.TerraformState) *RollbackPlan {
	rollback := &RollbackPlan{
		ID:          uuid.New().String(),
		Description: fmt.Sprintf("Rollback plan for %s", plan.Name),
		Actions:     make([]RollbackAction, 0),
		Automatic:   rp.config.SafeMode,
	}

	// Create rollback actions in reverse order
	for i := len(plan.Actions) - 1; i >= 0; i-- {
		action := plan.Actions[i]
		if action.Rollback != nil {
			rollback.Actions = append(rollback.Actions, *action.Rollback)
		}
	}

	return rollback
}

// ActionValidator validates remediation actions
type ActionValidator struct {
	rules []ValidationRule
}

// ValidationRule defines a validation rule
type ValidationRule struct {
	Name     string
	Validate func(RemediationAction) error
}

// NewActionValidator creates a new action validator
func NewActionValidator() *ActionValidator {
	validator := &ActionValidator{
		rules: make([]ValidationRule, 0),
	}

	// Add default validation rules
	validator.addDefaultRules()

	return validator
}

// Validate validates an action
func (av *ActionValidator) Validate(action RemediationAction) error {
	for _, rule := range av.rules {
		if err := rule.Validate(action); err != nil {
			return fmt.Errorf("validation failed for rule %s: %w", rule.Name, err)
		}
	}
	return nil
}

// addDefaultRules adds default validation rules
func (av *ActionValidator) addDefaultRules() {
	av.rules = append(av.rules, ValidationRule{
		Name: "required_fields",
		Validate: func(action RemediationAction) error {
			if action.ID == "" {
				return fmt.Errorf("action ID is required")
			}
			if action.Type == "" {
				return fmt.Errorf("action type is required")
			}
			if action.Resource == "" {
				return fmt.Errorf("resource is required")
			}
			return nil
		},
	})

	av.rules = append(av.rules, ValidationRule{
		Name: "timeout",
		Validate: func(action RemediationAction) error {
			if action.Timeout <= 0 {
				return fmt.Errorf("timeout must be positive")
			}
			if action.Timeout > 1*time.Hour {
				return fmt.Errorf("timeout exceeds maximum of 1 hour")
			}
			return nil
		},
	})
}

// RiskAnalyzer analyzes risk of remediation plans
type RiskAnalyzer struct {
	weights map[ActionType]float64
}

// NewRiskAnalyzer creates a new risk analyzer
func NewRiskAnalyzer() *RiskAnalyzer {
	return &RiskAnalyzer{
		weights: map[ActionType]float64{
			ActionTypeImport:  0.2,
			ActionTypeUpdate:  0.5,
			ActionTypeDelete:  0.8,
			ActionTypeCreate:  0.3,
			ActionTypeRefresh: 0.1,
			ActionTypeMove:    0.4,
			ActionTypeReplace: 0.9,
			ActionTypeTaint:   0.6,
			ActionTypeUntaint: 0.3,
		},
	}
}

// AnalyzePlan analyzes the risk of a remediation plan
func (ra *RiskAnalyzer) AnalyzePlan(plan *RemediationPlan) RiskLevel {
	if len(plan.Actions) == 0 {
		return RiskLevelLow
	}

	var totalRisk float64
	var maxRisk RiskLevel = RiskLevelLow

	for _, action := range plan.Actions {
		// Use action's risk level
		if action.RiskLevel > maxRisk {
			maxRisk = action.RiskLevel
		}

		// Calculate weighted risk
		if weight, ok := ra.weights[action.Type]; ok {
			totalRisk += weight
		}
	}

	// Average risk
	avgRisk := totalRisk / float64(len(plan.Actions))

	// Determine overall risk level
	if maxRisk >= RiskLevelCritical || avgRisk > 0.8 {
		return RiskLevelCritical
	} else if maxRisk >= RiskLevelHigh || avgRisk > 0.6 {
		return RiskLevelHigh
	} else if maxRisk >= RiskLevelMedium || avgRisk > 0.4 {
		return RiskLevelMedium
	}

	return RiskLevelLow
}