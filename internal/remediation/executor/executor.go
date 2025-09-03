package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/remediation/planner"
	"github.com/catherinevee/driftmgr/internal/state/manager"
	"github.com/catherinevee/driftmgr/internal/state/backup"
)

type RemediationExecutor struct {
	stateManager   *manager.StateManager
	backupManager  *backup.BackupManager
	terraformPath  string
	workDir        string
	dryRun         bool
	parallelism    int
	timeout        time.Duration
	hooks          ExecutionHooks
	rollbackOnFail bool
	mu             sync.RWMutex
	execHistory    []ExecutionResult
}

type ExecutionHooks struct {
	BeforeAction func(action *planner.RemediationAction) error
	AfterAction  func(action *planner.RemediationAction, result ActionResult) error
	OnError      func(action *planner.RemediationAction, err error) error
	OnRollback   func(plan *planner.RemediationPlan) error
}

type ExecutionResult struct {
	PlanID        string                   `json:"plan_id"`
	StartTime     time.Time                `json:"start_time"`
	EndTime       time.Time                `json:"end_time"`
	Status        ExecutionStatus          `json:"status"`
	ActionsTotal  int                      `json:"actions_total"`
	ActionsSucceeded int                   `json:"actions_succeeded"`
	ActionsFailed int                      `json:"actions_failed"`
	ActionsSkipped int                     `json:"actions_skipped"`
	ActionResults []ActionResult           `json:"action_results"`
	Error         string                   `json:"error,omitempty"`
	RollbackExecuted bool                 `json:"rollback_executed"`
}

type ActionResult struct {
	ActionID      string          `json:"action_id"`
	ResourceID    string          `json:"resource_id"`
	Action        string          `json:"action"`
	Status        ExecutionStatus `json:"status"`
	StartTime     time.Time       `json:"start_time"`
	EndTime       time.Time       `json:"end_time"`
	Output        string          `json:"output,omitempty"`
	Error         string          `json:"error,omitempty"`
	Changes       []string        `json:"changes,omitempty"`
}

type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusSuccess   ExecutionStatus = "success"
	StatusFailed    ExecutionStatus = "failed"
	StatusSkipped   ExecutionStatus = "skipped"
	StatusRolledBack ExecutionStatus = "rolled_back"
)

func NewRemediationExecutor(stateManager *manager.StateManager, workDir string) *RemediationExecutor {
	return &RemediationExecutor{
		stateManager:   stateManager,
		backupManager:  backup.NewBackupManager(filepath.Join(workDir, "backups")),
		terraformPath:  "terraform",
		workDir:        workDir,
		parallelism:    1,
		timeout:        30 * time.Minute,
		rollbackOnFail: true,
		execHistory:    make([]ExecutionResult, 0),
	}
}

func (re *RemediationExecutor) Execute(ctx context.Context, plan *planner.RemediationPlan) (*ExecutionResult, error) {
	if plan == nil {
		return nil, errors.New("remediation plan is nil")
	}

	result := &ExecutionResult{
		PlanID:       plan.ID,
		StartTime:    time.Now(),
		Status:       StatusRunning,
		ActionsTotal: len(plan.Actions),
	}

	// Create backup before execution
	backupID := fmt.Sprintf("backup-%s-%d", plan.ID, time.Now().Unix())
	if err := re.createBackup(ctx, backupID); err != nil {
		result.Status = StatusFailed
		result.Error = fmt.Sprintf("failed to create backup: %v", err)
		result.EndTime = time.Now()
		return result, err
	}

	// Execute actions in order
	actionResults := make([]ActionResult, 0, len(plan.Actions))
	executedActions := make([]*planner.RemediationAction, 0)
	
	for _, action := range plan.Actions {
		// Check if action can be executed
		if !re.canExecuteAction(action, actionResults) {
			actionResult := ActionResult{
				ActionID:   action.ID,
				ResourceID: action.ResourceID,
				Action:     string(action.Type),
				Status:     StatusSkipped,
				StartTime:  time.Now(),
				EndTime:    time.Now(),
			}
			actionResults = append(actionResults, actionResult)
			result.ActionsSkipped++
			continue
		}

		// Execute pre-action hook
		if re.hooks.BeforeAction != nil {
			if err := re.hooks.BeforeAction(action); err != nil {
				actionResult := ActionResult{
					ActionID:   action.ID,
					ResourceID: action.ResourceID,
					Action:     string(action.Type),
					Status:     StatusFailed,
					Error:      fmt.Sprintf("pre-action hook failed: %v", err),
					StartTime:  time.Now(),
					EndTime:    time.Now(),
				}
				actionResults = append(actionResults, actionResult)
				result.ActionsFailed++
				
				if re.rollbackOnFail {
					re.executeRollback(ctx, plan, executedActions, backupID)
					result.RollbackExecuted = true
					result.Status = StatusRolledBack
					result.EndTime = time.Now()
					return result, fmt.Errorf("action %s failed: %v", action.ID, err)
				}
				continue
			}
		}

		// Execute the action
		actionResult := re.executeAction(ctx, action)
		actionResults = append(actionResults, actionResult)
		
		if actionResult.Status == StatusSuccess {
			result.ActionsSucceeded++
			executedActions = append(executedActions, action)
		} else if actionResult.Status == StatusFailed {
			result.ActionsFailed++
			
			if re.rollbackOnFail {
				re.executeRollback(ctx, plan, executedActions, backupID)
				result.RollbackExecuted = true
				result.Status = StatusRolledBack
				result.EndTime = time.Now()
				result.ActionResults = actionResults
				return result, fmt.Errorf("action %s failed: %s", action.ID, actionResult.Error)
			}
		}

		// Execute post-action hook
		if re.hooks.AfterAction != nil {
			if err := re.hooks.AfterAction(action, actionResult); err != nil {
				// Log but don't fail on post-action hook errors
				fmt.Fprintf(os.Stderr, "post-action hook error: %v\n", err)
			}
		}
	}

	result.ActionResults = actionResults
	result.EndTime = time.Now()
	
	if result.ActionsFailed > 0 {
		result.Status = StatusFailed
	} else {
		result.Status = StatusSuccess
	}

	// Store execution result
	re.mu.Lock()
	re.execHistory = append(re.execHistory, *result)
	re.mu.Unlock()

	return result, nil
}

func (re *RemediationExecutor) executeAction(ctx context.Context, action *planner.RemediationAction) ActionResult {
	result := ActionResult{
		ActionID:   action.ID,
		ResourceID: action.ResourceID,
		Action:     string(action.Type),
		StartTime:  time.Now(),
		Status:     StatusRunning,
	}

	if re.dryRun {
		result.Status = StatusSuccess
		result.EndTime = time.Now()
		result.Output = fmt.Sprintf("[DRY RUN] Would execute: %s on %s", action.Type, action.ResourceID)
		return result
	}

	var err error
	switch action.Type {
	case planner.ActionTypeCreate:
		err = re.executeCreateAction(ctx, action, &result)
	case planner.ActionTypeUpdate:
		err = re.executeUpdateAction(ctx, action, &result)
	case planner.ActionTypeDelete:
		err = re.executeDeleteAction(ctx, action, &result)
	case planner.ActionTypeImport:
		err = re.executeImportAction(ctx, action, &result)
	case planner.ActionTypeRefresh:
		err = re.executeRefreshAction(ctx, action, &result)
	case planner.ActionTypeTaint:
		err = re.executeTaintAction(ctx, action, &result)
	case planner.ActionTypeUntaint:
		err = re.executeUntaintAction(ctx, action, &result)
	case planner.ActionTypeMove:
		err = re.executeMoveAction(ctx, action, &result)
	default:
		err = fmt.Errorf("unsupported action type: %s", action.Type)
	}

	result.EndTime = time.Now()
	if err != nil {
		result.Status = StatusFailed
		result.Error = err.Error()
		
		if re.hooks.OnError != nil {
			re.hooks.OnError(action, err)
		}
	} else {
		result.Status = StatusSuccess
	}

	return result
}

func (re *RemediationExecutor) executeCreateAction(ctx context.Context, action *planner.RemediationAction, result *ActionResult) error {
	// Generate Terraform configuration for the resource
	config, err := re.generateResourceConfig(action)
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	// Write configuration to temporary file
	configFile := filepath.Join(re.workDir, fmt.Sprintf("create_%s.tf", action.ResourceID))
	if err := os.WriteFile(configFile, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	defer os.Remove(configFile)

	// Run terraform apply
	cmd := exec.CommandContext(ctx, re.terraformPath, "apply", "-auto-approve", "-target", action.ResourceID)
	cmd.Dir = re.workDir
	output, err := cmd.CombinedOutput()
	result.Output = string(output)
	
	if err != nil {
		return fmt.Errorf("terraform apply failed: %w", err)
	}

	result.Changes = append(result.Changes, fmt.Sprintf("Created resource %s", action.ResourceID))
	return nil
}

func (re *RemediationExecutor) executeUpdateAction(ctx context.Context, action *planner.RemediationAction, result *ActionResult) error {
	// Run terraform apply for the specific resource
	cmd := exec.CommandContext(ctx, re.terraformPath, "apply", "-auto-approve", "-target", action.ResourceID)
	cmd.Dir = re.workDir
	output, err := cmd.CombinedOutput()
	result.Output = string(output)
	
	if err != nil {
		return fmt.Errorf("terraform apply failed: %w", err)
	}

	result.Changes = append(result.Changes, fmt.Sprintf("Updated resource %s", action.ResourceID))
	return nil
}

func (re *RemediationExecutor) executeDeleteAction(ctx context.Context, action *planner.RemediationAction, result *ActionResult) error {
	// Run terraform destroy for the specific resource
	cmd := exec.CommandContext(ctx, re.terraformPath, "destroy", "-auto-approve", "-target", action.ResourceID)
	cmd.Dir = re.workDir
	output, err := cmd.CombinedOutput()
	result.Output = string(output)
	
	if err != nil {
		return fmt.Errorf("terraform destroy failed: %w", err)
	}

	result.Changes = append(result.Changes, fmt.Sprintf("Deleted resource %s", action.ResourceID))
	return nil
}

func (re *RemediationExecutor) executeImportAction(ctx context.Context, action *planner.RemediationAction, result *ActionResult) error {
	importID, ok := action.Parameters["import_id"].(string)
	if !ok {
		return errors.New("import_id parameter not found")
	}

	// Run terraform import
	cmd := exec.CommandContext(ctx, re.terraformPath, "import", action.ResourceID, importID)
	cmd.Dir = re.workDir
	output, err := cmd.CombinedOutput()
	result.Output = string(output)
	
	if err != nil {
		return fmt.Errorf("terraform import failed: %w", err)
	}

	result.Changes = append(result.Changes, fmt.Sprintf("Imported resource %s with ID %s", action.ResourceID, importID))
	return nil
}

func (re *RemediationExecutor) executeRefreshAction(ctx context.Context, action *planner.RemediationAction, result *ActionResult) error {
	// Run terraform refresh
	cmd := exec.CommandContext(ctx, re.terraformPath, "refresh", "-target", action.ResourceID)
	cmd.Dir = re.workDir
	output, err := cmd.CombinedOutput()
	result.Output = string(output)
	
	if err != nil {
		return fmt.Errorf("terraform refresh failed: %w", err)
	}

	result.Changes = append(result.Changes, fmt.Sprintf("Refreshed resource %s", action.ResourceID))
	return nil
}

func (re *RemediationExecutor) executeTaintAction(ctx context.Context, action *planner.RemediationAction, result *ActionResult) error {
	// Run terraform taint
	cmd := exec.CommandContext(ctx, re.terraformPath, "taint", action.ResourceID)
	cmd.Dir = re.workDir
	output, err := cmd.CombinedOutput()
	result.Output = string(output)
	
	if err != nil {
		return fmt.Errorf("terraform taint failed: %w", err)
	}

	result.Changes = append(result.Changes, fmt.Sprintf("Tainted resource %s", action.ResourceID))
	return nil
}

func (re *RemediationExecutor) executeUntaintAction(ctx context.Context, action *planner.RemediationAction, result *ActionResult) error {
	// Run terraform untaint
	cmd := exec.CommandContext(ctx, re.terraformPath, "untaint", action.ResourceID)
	cmd.Dir = re.workDir
	output, err := cmd.CombinedOutput()
	result.Output = string(output)
	
	if err != nil {
		return fmt.Errorf("terraform untaint failed: %w", err)
	}

	result.Changes = append(result.Changes, fmt.Sprintf("Untainted resource %s", action.ResourceID))
	return nil
}

func (re *RemediationExecutor) executeMoveAction(ctx context.Context, action *planner.RemediationAction, result *ActionResult) error {
	newID, ok := action.Parameters["new_id"].(string)
	if !ok {
		return errors.New("new_id parameter not found")
	}

	// Run terraform state mv
	cmd := exec.CommandContext(ctx, re.terraformPath, "state", "mv", action.ResourceID, newID)
	cmd.Dir = re.workDir
	output, err := cmd.CombinedOutput()
	result.Output = string(output)
	
	if err != nil {
		return fmt.Errorf("terraform state mv failed: %w", err)
	}

	result.Changes = append(result.Changes, fmt.Sprintf("Moved resource from %s to %s", action.ResourceID, newID))
	return nil
}

func (re *RemediationExecutor) canExecuteAction(action *planner.RemediationAction, completedActions []ActionResult) bool {
	// Check dependencies
	for _, dep := range action.Dependencies {
		found := false
		for _, completed := range completedActions {
			if completed.ActionID == dep && completed.Status == StatusSuccess {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (re *RemediationExecutor) createBackup(ctx context.Context, backupID string) error {
	// Get current state
	state, err := re.stateManager.GetState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current state: %w", err)
	}

	// Create backup
	return re.backupManager.CreateBackup(backupID, state)
}

func (re *RemediationExecutor) executeRollback(ctx context.Context, plan *planner.RemediationPlan, executedActions []*planner.RemediationAction, backupID string) error {
	if re.hooks.OnRollback != nil {
		if err := re.hooks.OnRollback(plan); err != nil {
			fmt.Fprintf(os.Stderr, "rollback hook error: %v\n", err)
		}
	}

	// Restore from backup
	if err := re.backupManager.RestoreBackup(backupID); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	// Execute rollback actions in reverse order
	for i := len(executedActions) - 1; i >= 0; i-- {
		action := executedActions[i]
		if rollbackAction := plan.RollbackPlan[action.ID]; rollbackAction != nil {
			re.executeAction(ctx, rollbackAction)
		}
	}

	return nil
}

func (re *RemediationExecutor) generateResourceConfig(action *planner.RemediationAction) (string, error) {
	// Generate Terraform configuration based on action parameters
	config := strings.Builder{}
	
	resourceType, ok := action.Parameters["resource_type"].(string)
	if !ok {
		return "", errors.New("resource_type not specified")
	}

	resourceName := strings.ReplaceAll(action.ResourceID, ".", "_")
	
	config.WriteString(fmt.Sprintf("resource \"%s\" \"%s\" {\n", resourceType, resourceName))
	
	// Add resource attributes from parameters
	if attrs, ok := action.Parameters["attributes"].(map[string]interface{}); ok {
		for key, value := range attrs {
			config.WriteString(fmt.Sprintf("  %s = %s\n", key, re.formatValue(value)))
		}
	}
	
	config.WriteString("}\n")
	
	return config.String(), nil
}

func (re *RemediationExecutor) formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	case []interface{}:
		items := make([]string, len(v))
		for i, item := range v {
			items[i] = re.formatValue(item)
		}
		return fmt.Sprintf("[%s]", strings.Join(items, ", "))
	case map[string]interface{}:
		b, _ := json.Marshal(v)
		return string(b)
	default:
		return fmt.Sprintf("\"%v\"", v)
	}
}

func (re *RemediationExecutor) SetDryRun(dryRun bool) {
	re.dryRun = dryRun
}

func (re *RemediationExecutor) SetParallelism(parallelism int) {
	re.parallelism = parallelism
}

func (re *RemediationExecutor) SetTimeout(timeout time.Duration) {
	re.timeout = timeout
}

func (re *RemediationExecutor) SetHooks(hooks ExecutionHooks) {
	re.hooks = hooks
}

func (re *RemediationExecutor) GetExecutionHistory() []ExecutionResult {
	re.mu.RLock()
	defer re.mu.RUnlock()
	
	history := make([]ExecutionResult, len(re.execHistory))
	copy(history, re.execHistory)
	return history
}