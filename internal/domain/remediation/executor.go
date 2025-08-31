package remediation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// RemediationType defines the type of remediation action
type RemediationType string

const (
	RemediationTypeUpdate  RemediationType = "update"
	RemediationTypeReplace RemediationType = "replace"
	RemediationTypeDelete  RemediationType = "delete"
	RemediationTypeImport  RemediationType = "import"
	RemediationTypeCreate  RemediationType = "create"
)

// RemediationPlan represents a plan for remediating drift
type RemediationPlan struct {
	ID               string                    `json:"id"`
	ResourceID       string                    `json:"resource_id"`
	ResourceType     string                    `json:"resource_type"`
	ResourceName     string                    `json:"resource_name"`
	Provider         string                    `json:"provider"`
	Type             RemediationType           `json:"type"`
	CurrentState     map[string]interface{}    `json:"current_state"`
	DesiredState     map[string]interface{}    `json:"desired_state"`
	Changes          []Change                  `json:"changes"`
	TerraformCode    string                    `json:"terraform_code"`
	ImportCommands   []string                  `json:"import_commands,omitempty"`
	ValidationErrors []string                  `json:"validation_errors,omitempty"`
	EstimatedImpact  *ImpactAssessment         `json:"estimated_impact,omitempty"`
	CreatedAt        time.Time                 `json:"created_at"`
	AppliedAt        *time.Time                `json:"applied_at,omitempty"`
	RollbackPlan     *RollbackPlan             `json:"rollback_plan,omitempty"`
	Metadata         map[string]string         `json:"metadata"`
}

// Change represents a specific change to be made
type Change struct {
	Path        string      `json:"path"`
	OldValue    interface{} `json:"old_value"`
	NewValue    interface{} `json:"new_value"`
	Action      string      `json:"action"` // add, remove, update
	Sensitivity string      `json:"sensitivity"` // low, medium, high, critical
}

// ImpactAssessment represents the estimated impact of remediation
type ImpactAssessment struct {
	Severity           string            `json:"severity"` // low, medium, high, critical
	EstimatedDowntime  time.Duration     `json:"estimated_downtime"`
	AffectedResources  []string          `json:"affected_resources"`
	CostImpact         *CostImpact       `json:"cost_impact,omitempty"`
	SecurityImpact     *SecurityImpact   `json:"security_impact,omitempty"`
	PerformanceImpact  *PerformanceImpact `json:"performance_impact,omitempty"`
	RiskScore          float64           `json:"risk_score"` // 0.0 to 10.0
	RequiresApproval   bool              `json:"requires_approval"`
	ApprovalReason     string            `json:"approval_reason,omitempty"`
}

// CostImpact represents financial impact
type CostImpact struct {
	CurrentMonthlyCost float64 `json:"current_monthly_cost"`
	NewMonthlyCost     float64 `json:"new_monthly_cost"`
	MonthlySavings     float64 `json:"monthly_savings"`
	OneTimeCost        float64 `json:"one_time_cost"`
	Currency           string  `json:"currency"`
}

// SecurityImpact represents security implications
type SecurityImpact struct {
	CurrentScore     float64  `json:"current_score"` // 0.0 to 10.0
	NewScore         float64  `json:"new_score"`
	Improvements     []string `json:"improvements"`
	NewRisks         []string `json:"new_risks"`
	ComplianceImpact []string `json:"compliance_impact"`
}

// PerformanceImpact represents performance implications
type PerformanceImpact struct {
	LatencyChange      string   `json:"latency_change"` // e.g., "-20%", "+5ms"
	ThroughputChange   string   `json:"throughput_change"`
	AvailabilityChange string   `json:"availability_change"`
	Metrics            []string `json:"affected_metrics"`
}

// RollbackPlan represents a plan to rollback changes
type RollbackPlan struct {
	ID              string            `json:"id"`
	SnapshotID      string            `json:"snapshot_id"`
	BackupLocation  string            `json:"backup_location"`
	RollbackCode    string            `json:"rollback_code"`
	RollbackSteps   []RollbackStep    `json:"rollback_steps"`
	EstimatedTime   time.Duration     `json:"estimated_time"`
	AutoRollback    bool              `json:"auto_rollback"`
	RollbackTrigger string            `json:"rollback_trigger"`
	Metadata        map[string]string `json:"metadata"`
}

// RollbackStep represents a single rollback action
type RollbackStep struct {
	Order       int               `json:"order"`
	Description string            `json:"description"`
	Command     string            `json:"command"`
	Validation  string            `json:"validation"`
	OnFailure   string            `json:"on_failure"` // continue, abort, retry
	Metadata    map[string]string `json:"metadata"`
}

// ApplyResult represents the result of applying a remediation
type ApplyResult struct {
	Success          bool              `json:"success"`
	PlanID           string            `json:"plan_id"`
	AppliedChanges   []Change          `json:"applied_changes"`
	FailedChanges    []Change          `json:"failed_changes"`
	Errors           []string          `json:"errors"`
	Warnings         []string          `json:"warnings"`
	ExecutionTime    time.Duration     `json:"execution_time"`
	RollbackExecuted bool              `json:"rollback_executed"`
	FinalState       map[string]interface{} `json:"final_state"`
	Logs             []LogEntry        `json:"logs"`
}

// LogEntry represents a log entry during remediation
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // debug, info, warning, error
	Message   string    `json:"message"`
	Context   map[string]string `json:"context,omitempty"`
}

// ValidationResult represents the result of plan validation
type ValidationResult struct {
	Valid            bool     `json:"valid"`
	Errors           []string `json:"errors"`
	Warnings         []string `json:"warnings"`
	SecurityIssues   []string `json:"security_issues"`
	BestPractices    []string `json:"best_practices"`
	EstimatedChanges int      `json:"estimated_changes"`
}

// RemediationEngine handles the remediation process
type RemediationEngine struct {
	workDir          string
	stateManager     *StateManager
	codeGenerator    *CodeGenerator
	validator        *PlanValidator
	executor         *PlanExecutor
	rollbackManager  *RollbackManager
	impactAnalyzer   *ImpactAnalyzer
	approvalManager  *ApprovalManager
	auditLogger      *AuditLogger
	config           *RemediationConfig
}

// RemediationConfig holds configuration for remediation
type RemediationConfig struct {
	AutoApprove            bool              `json:"auto_approve"`
	DryRun                 bool              `json:"dry_run"`
	MaxParallelOperations  int               `json:"max_parallel_operations"`
	RequireApprovalAbove   float64           `json:"require_approval_above"` // risk score threshold
	AutoRollbackOnFailure  bool              `json:"auto_rollback_on_failure"`
	BackupBeforeApply      bool              `json:"backup_before_apply"`
	ValidateBeforeApply    bool              `json:"validate_before_apply"`
	GenerateImportCommands bool              `json:"generate_import_commands"`
	OutputFormat           string            `json:"output_format"` // hcl, json
	TerraformVersion       string            `json:"terraform_version"`
	ProviderVersions       map[string]string `json:"provider_versions"`
	CustomModules          []string          `json:"custom_modules"`
	VariableFiles          []string          `json:"variable_files"`
	BackendConfig          map[string]string `json:"backend_config"`
}

// NewRemediationEngine creates a new remediation engine
func NewRemediationEngine(workDir string, config *RemediationConfig) (*RemediationEngine, error) {
	if config == nil {
		config = DefaultRemediationConfig()
	}

	// Ensure work directory exists
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}

	engine := &RemediationEngine{
		workDir:         workDir,
		config:          config,
		stateManager:    NewStateManager(workDir),
		codeGenerator:   NewCodeGenerator(config),
		validator:       NewPlanValidator(config),
		executor:        NewPlanExecutor(workDir, config),
		rollbackManager: NewRollbackManager(workDir),
		impactAnalyzer:  NewImpactAnalyzer(),
		approvalManager: NewApprovalManager(config),
		auditLogger:     NewAuditLogger(filepath.Join(workDir, "audit.log")),
	}

	return engine, nil
}

// DefaultRemediationConfig returns default configuration
func DefaultRemediationConfig() *RemediationConfig {
	return &RemediationConfig{
		AutoApprove:            false,
		DryRun:                 false,
		MaxParallelOperations:  5,
		RequireApprovalAbove:   7.0,
		AutoRollbackOnFailure:  true,
		BackupBeforeApply:      true,
		ValidateBeforeApply:    true,
		GenerateImportCommands: true,
		OutputFormat:           "hcl",
		TerraformVersion:       "1.5.0",
		ProviderVersions:       make(map[string]string),
		CustomModules:          []string{},
		VariableFiles:          []string{},
		BackendConfig:          make(map[string]string),
	}
}

// GeneratePlan generates a remediation plan for drift
func (e *RemediationEngine) GeneratePlan(ctx context.Context, drift *models.DriftResult) (*RemediationPlan, error) {
	e.auditLogger.Log("info", "Generating remediation plan", map[string]string{
		"resource_id": drift.ResourceID,
		"resource_type": drift.ResourceType,
	})

	// Create base plan
	plan := &RemediationPlan{
		ID:           generatePlanID(),
		ResourceID:   drift.ResourceID,
		ResourceType: drift.ResourceType,
		ResourceName: drift.ResourceName,
		Provider:     drift.Provider,
		CreatedAt:    time.Now(),
		Metadata:     make(map[string]string),
	}

	// Determine remediation type
	plan.Type = e.determineRemediationType(drift)

	// Extract states from drift changes
	plan.CurrentState = make(map[string]interface{})
	plan.DesiredState = make(map[string]interface{})
	
	// Build states from drift changes
	for _, change := range drift.Changes {
		if change.NewValue != nil {
			plan.CurrentState[change.Field] = change.NewValue
		}
		if change.OldValue != nil {
			plan.DesiredState[change.Field] = change.OldValue
		}
	}

	// Generate changes list
	plan.Changes = e.generateChangesList(drift)

	// Generate Terraform code
	code, err := e.codeGenerator.GenerateCode(plan)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Terraform code: %w", err)
	}
	plan.TerraformCode = code

	// Generate import commands if needed
	if e.config.GenerateImportCommands && plan.Type == RemediationTypeImport {
		plan.ImportCommands = e.codeGenerator.GenerateImportCommands(plan)
	}

	// Perform impact analysis
	impact, err := e.impactAnalyzer.Analyze(ctx, plan)
	if err != nil {
		e.auditLogger.Log("warning", "Failed to analyze impact", map[string]string{
			"error": err.Error(),
		})
	} else {
		plan.EstimatedImpact = impact
	}

	// Create rollback plan
	if e.config.AutoRollbackOnFailure {
		rollback, err := e.rollbackManager.CreateRollbackPlan(plan)
		if err != nil {
			e.auditLogger.Log("warning", "Failed to create rollback plan", map[string]string{
				"error": err.Error(),
			})
		} else {
			plan.RollbackPlan = rollback
		}
	}

	// Validate the plan
	validation := e.validator.Validate(plan)
	plan.ValidationErrors = validation.Errors

	e.auditLogger.Log("info", "Remediation plan generated", map[string]string{
		"plan_id": plan.ID,
		"type": string(plan.Type),
		"changes_count": fmt.Sprintf("%d", len(plan.Changes)),
	})

	return plan, nil
}

// ValidatePlan validates a remediation plan
func (e *RemediationEngine) ValidatePlan(ctx context.Context, plan *RemediationPlan) (*ValidationResult, error) {
	return e.validator.ValidateComprehensive(ctx, plan)
}

// ApplyPlan applies a remediation plan
func (e *RemediationEngine) ApplyPlan(ctx context.Context, plan *RemediationPlan) (*ApplyResult, error) {
	e.auditLogger.Log("info", "Starting plan application", map[string]string{
		"plan_id": plan.ID,
		"dry_run": fmt.Sprintf("%v", e.config.DryRun),
	})

	// Check if approval is required
	if plan.EstimatedImpact != nil && plan.EstimatedImpact.RequiresApproval && !e.config.AutoApprove {
		approved, err := e.approvalManager.RequestApproval(ctx, plan)
		if err != nil {
			return nil, fmt.Errorf("failed to get approval: %w", err)
		}
		if !approved {
			return nil, fmt.Errorf("plan requires approval but was not approved")
		}
	}

	// Validate before applying
	if e.config.ValidateBeforeApply {
		validation, err := e.ValidatePlan(ctx, plan)
		if err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
		if !validation.Valid {
			return nil, fmt.Errorf("plan validation failed: %v", validation.Errors)
		}
	}

	// Create backup if configured
	if e.config.BackupBeforeApply && !e.config.DryRun {
		if err := e.rollbackManager.CreateBackup(plan); err != nil {
			e.auditLogger.Log("warning", "Failed to create backup", map[string]string{
				"error": err.Error(),
			})
		}
	}

	// Execute the plan
	result, err := e.executor.Execute(ctx, plan)
	if err != nil {
		e.auditLogger.Log("error", "Plan execution failed", map[string]string{
			"error": err.Error(),
		})

		// Attempt rollback if configured
		if e.config.AutoRollbackOnFailure && !e.config.DryRun && plan.RollbackPlan != nil {
			rollbackResult, rollbackErr := e.rollbackManager.ExecuteRollback(ctx, plan.RollbackPlan)
			if rollbackErr != nil {
				e.auditLogger.Log("error", "Rollback failed", map[string]string{
					"error": rollbackErr.Error(),
				})
			} else {
				result.RollbackExecuted = true
				e.auditLogger.Log("info", "Rollback completed", map[string]string{
					"success": fmt.Sprintf("%v", rollbackResult.Success),
				})
			}
		}

		return result, err
	}

	// Update plan with application timestamp
	now := time.Now()
	plan.AppliedAt = &now

	e.auditLogger.Log("info", "Plan application completed", map[string]string{
		"plan_id": plan.ID,
		"success": fmt.Sprintf("%v", result.Success),
		"execution_time": result.ExecutionTime.String(),
	})

	return result, nil
}

// determineRemediationType determines the type of remediation needed
func (e *RemediationEngine) determineRemediationType(drift *models.DriftResult) RemediationType {
	// Check drift type
	if drift.DriftType == "created" {
		return RemediationTypeDelete
	}
	if drift.DriftType == "deleted" {
		return RemediationTypeCreate
	}

	// Check if resource needs to be imported into Terraform
	if drift.DriftType == "unmanaged" {
		return RemediationTypeImport
	}

	// Check severity of changes
	criticalChanges := 0
	for _, change := range drift.Changes {
		if e.isForceNewChange(drift.ResourceType, change.Field) {
			criticalChanges++
		}
	}

	// If there are critical changes, resource needs replacement
	if criticalChanges > 0 {
		return RemediationTypeReplace
	}

	// Otherwise, just update the resource
	return RemediationTypeUpdate
}

// generateChangesList generates a detailed list of changes
func (e *RemediationEngine) generateChangesList(drift *models.DriftResult) []Change {
	changes := []Change{}

	for _, driftChange := range drift.Changes {
		change := Change{
			Path:     driftChange.Field,
			OldValue: driftChange.OldValue,
			NewValue: driftChange.NewValue,
			Action:   driftChange.ChangeType,
			Sensitivity: e.determineChangeSensitivity(drift.ResourceType, driftChange.Field),
		}
		changes = append(changes, change)
	}

	return changes
}

// determineChangeAction determines the action for a change
func (e *RemediationEngine) determineChangeAction(change models.DriftChange) string {
	if change.OldValue == nil && change.NewValue != nil {
		return "add"
	}
	if change.OldValue != nil && change.NewValue == nil {
		return "remove"
	}
	return "update"
}

// determineChangeSensitivity determines the sensitivity of a change
func (e *RemediationEngine) determineChangeSensitivity(resourceType, field string) string {
	// Security-related fields are critical
	securityFields := []string{"security_group", "network_acl", "iam", "policy", "encryption", "ssl", "tls"}
	for _, secField := range securityFields {
		if strings.Contains(strings.ToLower(field), secField) {
			return "critical"
		}
	}

	// Network-related fields are high sensitivity
	networkFields := []string{"subnet", "vpc", "cidr", "ip", "dns", "route"}
	for _, netField := range networkFields {
		if strings.Contains(strings.ToLower(field), netField) {
			return "high"
		}
	}

	// Performance-related fields are medium sensitivity
	perfFields := []string{"instance_type", "size", "capacity", "memory", "cpu", "storage"}
	for _, perfField := range perfFields {
		if strings.Contains(strings.ToLower(field), perfField) {
			return "medium"
		}
	}

	// Tags and metadata are low sensitivity
	if strings.Contains(strings.ToLower(field), "tag") || strings.Contains(strings.ToLower(field), "metadata") {
		return "low"
	}

	// Default to medium
	return "medium"
}

// isForceNewChange checks if a change requires resource replacement
func (e *RemediationEngine) isForceNewChange(resourceType, field string) bool {
	// Define fields that require resource replacement for common resource types
	forceNewFields := map[string][]string{
		"aws_instance": {"ami", "instance_type", "availability_zone", "subnet_id"},
		"aws_rds_instance": {"engine", "engine_version", "allocated_storage"},
		"aws_s3_bucket": {"bucket", "region"},
		"azurerm_virtual_machine": {"location", "vm_size"},
		"google_compute_instance": {"machine_type", "zone", "boot_disk"},
	}

	if fields, ok := forceNewFields[resourceType]; ok {
		for _, f := range fields {
			if strings.EqualFold(f, field) {
				return true
			}
		}
	}

	return false
}

// generatePlanID generates a unique plan ID
func generatePlanID() string {
	return fmt.Sprintf("plan-%d-%s", time.Now().Unix(), generateRandomString(8))
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}

// GetPlan retrieves a remediation plan by ID
func (e *RemediationEngine) GetPlan(planID string) (*RemediationPlan, error) {
	planPath := filepath.Join(e.workDir, "plans", planID+".json")
	
	data, err := os.ReadFile(planPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan: %w", err)
	}

	var plan RemediationPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	return &plan, nil
}

// SavePlan saves a remediation plan
func (e *RemediationEngine) SavePlan(plan *RemediationPlan) error {
	plansDir := filepath.Join(e.workDir, "plans")
	if err := os.MkdirAll(plansDir, 0755); err != nil {
		return fmt.Errorf("failed to create plans directory: %w", err)
	}

	planPath := filepath.Join(plansDir, plan.ID+".json")
	
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	if err := os.WriteFile(planPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write plan: %w", err)
	}

	return nil
}

// ListPlans lists all remediation plans
func (e *RemediationEngine) ListPlans() ([]*RemediationPlan, error) {
	plansDir := filepath.Join(e.workDir, "plans")
	
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*RemediationPlan{}, nil
		}
		return nil, fmt.Errorf("failed to read plans directory: %w", err)
	}

	var plans []*RemediationPlan
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			planID := strings.TrimSuffix(entry.Name(), ".json")
			plan, err := e.GetPlan(planID)
			if err != nil {
				e.auditLogger.Log("warning", "Failed to load plan", map[string]string{
					"plan_id": planID,
					"error": err.Error(),
				})
				continue
			}
			plans = append(plans, plan)
		}
	}

	return plans, nil
}