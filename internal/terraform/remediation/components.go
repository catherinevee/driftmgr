package remediation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// PlanValidator validates remediation plans
type PlanValidator struct {
	config    *RemediationConfig
	rules     map[string]ValidationRule
	mu        sync.RWMutex
}

// ValidationRule represents a validation rule
type ValidationRule struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Severity    string                    `json:"severity"` // low, medium, high, critical
	Validator   func(*RemediationPlan) error `json:"-"`
}

// NewPlanValidator creates a new plan validator
func NewPlanValidator(config *RemediationConfig) *PlanValidator {
	validator := &PlanValidator{
		config: config,
		rules:  make(map[string]ValidationRule),
	}
	
	// Add default validation rules
	validator.addDefaultRules()
	
	return validator
}

// addDefaultRules adds default validation rules
func (v *PlanValidator) addDefaultRules() {
	// Check for critical changes
	v.AddRule(ValidationRule{
		Name:        "critical_changes",
		Description: "Check for critical infrastructure changes",
		Severity:    "high",
		Validator: func(plan *RemediationPlan) error {
			criticalCount := 0
			for _, change := range plan.Changes {
				if change.Sensitivity == "critical" {
					criticalCount++
				}
			}
			if criticalCount > 5 {
				return fmt.Errorf("plan contains %d critical changes (max allowed: 5)", criticalCount)
			}
			return nil
		},
	})
	
	// Check for deletion of production resources
	v.AddRule(ValidationRule{
		Name:        "production_deletion",
		Description: "Prevent deletion of production resources",
		Severity:    "critical",
		Validator: func(plan *RemediationPlan) error {
			if plan.Type == RemediationTypeDelete {
				if tags, ok := plan.CurrentState["tags"].(map[string]interface{}); ok {
					if env, ok := tags["environment"].(string); ok && env == "production" {
						return fmt.Errorf("cannot delete production resource without explicit approval")
					}
				}
			}
			return nil
		},
	})
	
	// Check for missing required attributes
	v.AddRule(ValidationRule{
		Name:        "required_attributes",
		Description: "Ensure all required attributes are present",
		Severity:    "medium",
		Validator: func(plan *RemediationPlan) error {
			if plan.Type == RemediationTypeCreate || plan.Type == RemediationTypeUpdate {
				requiredAttrs := getRequiredAttributesForType(plan.ResourceType)
				for _, attr := range requiredAttrs {
					if _, ok := plan.DesiredState[attr]; !ok {
						return fmt.Errorf("missing required attribute: %s", attr)
					}
				}
			}
			return nil
		},
	})
}

// AddRule adds a validation rule
func (v *PlanValidator) AddRule(rule ValidationRule) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.rules[rule.Name] = rule
}

// Validate validates a remediation plan
func (v *PlanValidator) Validate(plan *RemediationPlan) *ValidationResult {
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	result := &ValidationResult{
		Valid:            true,
		Errors:           []string{},
		Warnings:         []string{},
		SecurityIssues:   []string{},
		BestPractices:    []string{},
		EstimatedChanges: len(plan.Changes),
	}
	
	// Run all validation rules
	for _, rule := range v.rules {
		if err := rule.Validator(plan); err != nil {
			switch rule.Severity {
			case "critical", "high":
				result.Valid = false
				result.Errors = append(result.Errors, err.Error())
			case "medium":
				result.Warnings = append(result.Warnings, err.Error())
			case "low":
				result.BestPractices = append(result.BestPractices, err.Error())
			}
		}
	}
	
	// Check for security issues
	result.SecurityIssues = v.checkSecurityIssues(plan)
	if len(result.SecurityIssues) > 0 {
		result.Valid = false
	}
	
	return result
}

// ValidateComprehensive performs comprehensive validation
func (v *PlanValidator) ValidateComprehensive(ctx context.Context, plan *RemediationPlan) (*ValidationResult, error) {
	// Basic validation
	result := v.Validate(plan)
	
	// Terraform validation
	if err := v.validateTerraformCode(ctx, plan); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Terraform validation failed: %v", err))
	}
	
	return result, nil
}

// validateTerraformCode validates the generated Terraform code
func (v *PlanValidator) validateTerraformCode(ctx context.Context, plan *RemediationPlan) error {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "tf-validate-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Write Terraform code to file
	tfFile := filepath.Join(tempDir, "main.tf")
	if err := os.WriteFile(tfFile, []byte(plan.TerraformCode), 0644); err != nil {
		return fmt.Errorf("failed to write terraform file: %w", err)
	}
	
	// Run terraform init
	initCmd := exec.CommandContext(ctx, "terraform", "init", "-backend=false")
	initCmd.Dir = tempDir
	if output, err := initCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("terraform init failed: %w\nOutput: %s", err, output)
	}
	
	// Run terraform validate
	validateCmd := exec.CommandContext(ctx, "terraform", "validate")
	validateCmd.Dir = tempDir
	if output, err := validateCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("terraform validate failed: %w\nOutput: %s", err, output)
	}
	
	return nil
}

// checkSecurityIssues checks for security issues in the plan
func (v *PlanValidator) checkSecurityIssues(plan *RemediationPlan) []string {
	issues := []string{}
	
	// Check for hardcoded credentials
	for key, value := range plan.DesiredState {
		if strings.Contains(strings.ToLower(key), "password") ||
		   strings.Contains(strings.ToLower(key), "secret") ||
		   strings.Contains(strings.ToLower(key), "token") {
			if str, ok := value.(string); ok && str != "" && !strings.HasPrefix(str, "${") {
				issues = append(issues, fmt.Sprintf("Potential hardcoded credential in field: %s", key))
			}
		}
	}
	
	// Check for overly permissive security groups
	if plan.ResourceType == "aws_security_group" || plan.ResourceType == "azurerm_network_security_group" {
		if rules, ok := plan.DesiredState["ingress"].([]interface{}); ok {
			for _, rule := range rules {
				if ruleMap, ok := rule.(map[string]interface{}); ok {
					if cidr, ok := ruleMap["cidr_blocks"].([]interface{}); ok {
						for _, block := range cidr {
							if block == "0.0.0.0/0" {
								issues = append(issues, "Security group allows traffic from 0.0.0.0/0")
							}
						}
					}
				}
			}
		}
	}
	
	// Check for unencrypted storage
	if strings.Contains(plan.ResourceType, "bucket") || strings.Contains(plan.ResourceType, "storage") {
		if encrypted, ok := plan.DesiredState["server_side_encryption_configuration"]; !ok || encrypted == nil {
			issues = append(issues, "Storage resource is not configured with encryption")
		}
	}
	
	return issues
}

// PlanExecutor executes remediation plans
type PlanExecutor struct {
	workDir    string
	config     *RemediationConfig
	logFile    *os.File
	mu         sync.Mutex
}

// NewPlanExecutor creates a new plan executor
func NewPlanExecutor(workDir string, config *RemediationConfig) *PlanExecutor {
	logPath := filepath.Join(workDir, "execution.log")
	logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	
	return &PlanExecutor{
		workDir: workDir,
		config:  config,
		logFile: logFile,
	}
}

// Execute executes a remediation plan
func (e *PlanExecutor) Execute(ctx context.Context, plan *RemediationPlan) (*ApplyResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	startTime := time.Now()
	result := &ApplyResult{
		Success:        true,
		PlanID:         plan.ID,
		AppliedChanges: []Change{},
		FailedChanges:  []Change{},
		Errors:         []string{},
		Warnings:       []string{},
		Logs:           []LogEntry{},
	}
	
	// Log start
	e.log(result, "info", "Starting plan execution", map[string]string{
		"plan_id": plan.ID,
		"dry_run": fmt.Sprintf("%v", e.config.DryRun),
	})
	
	if e.config.DryRun {
		// Dry run mode - just simulate
		e.log(result, "info", "DRY RUN MODE - No actual changes will be made", nil)
		result.AppliedChanges = plan.Changes
		result.ExecutionTime = time.Since(startTime)
		return result, nil
	}
	
	// Create working directory for this execution
	execDir := filepath.Join(e.workDir, "executions", plan.ID)
	if err := os.MkdirAll(execDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create execution directory: %w", err)
	}
	
	// Write Terraform code
	tfFile := filepath.Join(execDir, "main.tf")
	if err := os.WriteFile(tfFile, []byte(plan.TerraformCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write terraform file: %w", err)
	}
	
	// Execute Terraform commands
	if err := e.executeTerraform(ctx, execDir, plan, result); err != nil {
		result.Success = false
		result.Errors = append(result.Errors, err.Error())
		e.log(result, "error", "Terraform execution failed", map[string]string{
			"error": err.Error(),
		})
	}
	
	result.ExecutionTime = time.Since(startTime)
	
	// Save execution result
	e.saveResult(plan.ID, result)
	
	return result, nil
}

// executeTerraform executes Terraform commands
func (e *PlanExecutor) executeTerraform(ctx context.Context, workDir string, plan *RemediationPlan, result *ApplyResult) error {
	// Initialize Terraform
	e.log(result, "info", "Running terraform init", nil)
	initCmd := exec.CommandContext(ctx, "terraform", "init")
	initCmd.Dir = workDir
	if output, err := initCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("terraform init failed: %w\nOutput: %s", err, output)
	}
	
	// Run terraform plan
	e.log(result, "info", "Running terraform plan", nil)
	planCmd := exec.CommandContext(ctx, "terraform", "plan", "-out=tfplan")
	planCmd.Dir = workDir
	if output, err := planCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("terraform plan failed: %w\nOutput: %s", err, output)
	}
	
	// Run terraform apply
	e.log(result, "info", "Running terraform apply", nil)
	applyCmd := exec.CommandContext(ctx, "terraform", "apply", "-auto-approve", "tfplan")
	applyCmd.Dir = workDir
	output, err := applyCmd.CombinedOutput()
	
	if err != nil {
		result.FailedChanges = plan.Changes
		return fmt.Errorf("terraform apply failed: %w\nOutput: %s", err, output)
	}
	
	result.AppliedChanges = plan.Changes
	e.log(result, "info", "Terraform apply completed successfully", nil)
	
	// Get final state
	stateCmd := exec.CommandContext(ctx, "terraform", "show", "-json")
	stateCmd.Dir = workDir
	if stateOutput, err := stateCmd.Output(); err == nil {
		var state map[string]interface{}
		if err := json.Unmarshal(stateOutput, &state); err == nil {
			result.FinalState = state
		}
	}
	
	return nil
}

// log adds a log entry
func (e *PlanExecutor) log(result *ApplyResult, level, message string, context map[string]string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Context:   context,
	}
	
	result.Logs = append(result.Logs, entry)
	
	// Also write to log file
	if e.logFile != nil {
		logLine := fmt.Sprintf("[%s] %s: %s\n", entry.Timestamp.Format("2006-01-02 15:04:05"), level, message)
		e.logFile.WriteString(logLine)
	}
}

// saveResult saves execution result
func (e *PlanExecutor) saveResult(planID string, result *ApplyResult) error {
	resultPath := filepath.Join(e.workDir, "results", planID+".json")
	os.MkdirAll(filepath.Dir(resultPath), 0755)
	
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(resultPath, data, 0644)
}

// RollbackManager manages rollback operations
type RollbackManager struct {
	workDir      string
	backupDir    string
	stateManager *StateManager
}

// NewRollbackManager creates a new rollback manager
func NewRollbackManager(workDir string) *RollbackManager {
	backupDir := filepath.Join(workDir, "rollback-backups")
	os.MkdirAll(backupDir, 0755)
	
	return &RollbackManager{
		workDir:      workDir,
		backupDir:    backupDir,
		stateManager: NewStateManager(workDir),
	}
}

// CreateRollbackPlan creates a rollback plan for a remediation
func (rm *RollbackManager) CreateRollbackPlan(plan *RemediationPlan) (*RollbackPlan, error) {
	rollback := &RollbackPlan{
		ID:             fmt.Sprintf("rollback-%s", plan.ID),
		SnapshotID:     fmt.Sprintf("snapshot-%s-%d", plan.ID, time.Now().Unix()),
		BackupLocation: filepath.Join(rm.backupDir, plan.ID),
		RollbackSteps:  []RollbackStep{},
		EstimatedTime:  5 * time.Minute,
		AutoRollback:   true,
		Metadata:       make(map[string]string),
	}
	
	// Create rollback steps based on remediation type
	switch plan.Type {
	case RemediationTypeCreate:
		rollback.RollbackSteps = append(rollback.RollbackSteps, RollbackStep{
			Order:       1,
			Description: "Delete created resource",
			Command:     fmt.Sprintf("terraform destroy -target=%s.%s -auto-approve", plan.ResourceType, plan.ResourceName),
			Validation:  "terraform state list | grep -v " + plan.ResourceName,
			OnFailure:   "abort",
		})
		
	case RemediationTypeDelete:
		rollback.RollbackSteps = append(rollback.RollbackSteps, RollbackStep{
			Order:       1,
			Description: "Restore deleted resource",
			Command:     "terraform apply -auto-approve",
			Validation:  fmt.Sprintf("terraform state show %s.%s", plan.ResourceType, plan.ResourceName),
			OnFailure:   "abort",
		})
		
	case RemediationTypeUpdate, RemediationTypeReplace:
		rollback.RollbackSteps = append(rollback.RollbackSteps, RollbackStep{
			Order:       1,
			Description: "Restore previous configuration",
			Command:     "terraform apply -auto-approve -backup",
			Validation:  "terraform plan -detailed-exitcode",
			OnFailure:   "retry",
		})
	}
	
	// Generate rollback code
	rollback.RollbackCode = rm.generateRollbackCode(plan)
	
	return rollback, nil
}

// generateRollbackCode generates Terraform code for rollback
func (rm *RollbackManager) generateRollbackCode(plan *RemediationPlan) string {
	var code strings.Builder
	
	code.WriteString("# Rollback configuration for plan: " + plan.ID + "\n\n")
	
	// Generate resource block with original state
	code.WriteString(fmt.Sprintf("resource \"%s\" \"%s\" {\n", plan.ResourceType, plan.ResourceName))
	
	// Add original attributes
	for key, value := range plan.CurrentState {
		code.WriteString(fmt.Sprintf("  %s = %q\n", key, value))
	}
	
	code.WriteString("}\n")
	
	return code.String()
}

// CreateBackup creates a backup before applying changes
func (rm *RollbackManager) CreateBackup(plan *RemediationPlan) error {
	backupPath := filepath.Join(rm.backupDir, plan.ID)
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}
	
	// Save current state
	stateData, err := json.MarshalIndent(plan.CurrentState, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal current state: %w", err)
	}
	
	statePath := filepath.Join(backupPath, "current_state.json")
	if err := os.WriteFile(statePath, stateData, 0644); err != nil {
		return fmt.Errorf("failed to write state backup: %w", err)
	}
	
	// Save plan for reference
	planData, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}
	
	planPath := filepath.Join(backupPath, "plan.json")
	if err := os.WriteFile(planPath, planData, 0644); err != nil {
		return fmt.Errorf("failed to write plan backup: %w", err)
	}
	
	return nil
}

// ExecuteRollback executes a rollback plan
func (rm *RollbackManager) ExecuteRollback(ctx context.Context, rollback *RollbackPlan) (*ApplyResult, error) {
	result := &ApplyResult{
		Success: true,
		PlanID:  rollback.ID,
		Logs:    []LogEntry{},
	}
	
	// Execute each rollback step
	for _, step := range rollback.RollbackSteps {
		if err := rm.executeRollbackStep(ctx, step, result); err != nil {
			if step.OnFailure == "abort" {
				result.Success = false
				result.Errors = append(result.Errors, err.Error())
				return result, err
			} else if step.OnFailure == "retry" {
				// Retry once
				if err := rm.executeRollbackStep(ctx, step, result); err != nil {
					result.Success = false
					result.Errors = append(result.Errors, err.Error())
					return result, err
				}
			}
			// OnFailure == "continue" - just log and continue
			result.Warnings = append(result.Warnings, err.Error())
		}
	}
	
	return result, nil
}

// executeRollbackStep executes a single rollback step
func (rm *RollbackManager) executeRollbackStep(ctx context.Context, step RollbackStep, result *ApplyResult) error {
	// Log step execution
	result.Logs = append(result.Logs, LogEntry{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   fmt.Sprintf("Executing rollback step %d: %s", step.Order, step.Description),
	})
	
	// Execute command
	parts := strings.Fields(step.Command)
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = rm.workDir
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rollback step %d failed: %w\nOutput: %s", step.Order, err, output)
	}
	
	// Run validation if provided
	if step.Validation != "" {
		validationParts := strings.Fields(step.Validation)
		validationCmd := exec.CommandContext(ctx, validationParts[0], validationParts[1:]...)
		validationCmd.Dir = rm.workDir
		
		if _, err := validationCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("rollback validation failed for step %d: %w", step.Order, err)
		}
	}
	
	return nil
}

// ImpactAnalyzer analyzes the impact of remediation
type ImpactAnalyzer struct {
	costCalculator *CostCalculator
	riskAssessor   *RiskAssessor
}

// NewImpactAnalyzer creates a new impact analyzer
func NewImpactAnalyzer() *ImpactAnalyzer {
	return &ImpactAnalyzer{
		costCalculator: NewCostCalculator(),
		riskAssessor:   NewRiskAssessor(),
	}
}

// Analyze analyzes the impact of a remediation plan
func (ia *ImpactAnalyzer) Analyze(ctx context.Context, plan *RemediationPlan) (*ImpactAssessment, error) {
	assessment := &ImpactAssessment{
		Severity:          "low",
		EstimatedDowntime: 0,
		AffectedResources: []string{},
		RequiresApproval:  false,
	}
	
	// Assess risk
	assessment.RiskScore = ia.riskAssessor.AssessRisk(plan)
	
	// Determine severity based on risk score
	switch {
	case assessment.RiskScore >= 8.0:
		assessment.Severity = "critical"
		assessment.RequiresApproval = true
		assessment.ApprovalReason = "Critical risk level detected"
	case assessment.RiskScore >= 6.0:
		assessment.Severity = "high"
		assessment.RequiresApproval = true
		assessment.ApprovalReason = "High risk level detected"
	case assessment.RiskScore >= 4.0:
		assessment.Severity = "medium"
	default:
		assessment.Severity = "low"
	}
	
	// Estimate downtime
	assessment.EstimatedDowntime = ia.estimateDowntime(plan)
	
	// Identify affected resources
	assessment.AffectedResources = ia.identifyAffectedResources(plan)
	
	// Calculate cost impact
	assessment.CostImpact = ia.costCalculator.CalculateCostImpact(plan)
	
	// Assess security impact
	assessment.SecurityImpact = ia.assessSecurityImpact(plan)
	
	// Assess performance impact
	assessment.PerformanceImpact = ia.assessPerformanceImpact(plan)
	
	return assessment, nil
}

// estimateDowntime estimates the downtime for a remediation
func (ia *ImpactAnalyzer) estimateDowntime(plan *RemediationPlan) time.Duration {
	baseDowntime := time.Duration(0)
	
	switch plan.Type {
	case RemediationTypeReplace:
		baseDowntime = 5 * time.Minute
	case RemediationTypeDelete:
		baseDowntime = 1 * time.Minute
	case RemediationTypeCreate:
		baseDowntime = 3 * time.Minute
	case RemediationTypeUpdate:
		// Updates might not cause downtime
		for _, change := range plan.Changes {
			if change.Sensitivity == "critical" {
				baseDowntime = 2 * time.Minute
				break
			}
		}
	}
	
	return baseDowntime
}

// identifyAffectedResources identifies resources affected by the remediation
func (ia *ImpactAnalyzer) identifyAffectedResources(plan *RemediationPlan) []string {
	affected := []string{
		fmt.Sprintf("%s.%s", plan.ResourceType, plan.ResourceName),
	}
	
	// Add logic to identify dependent resources
	// This would analyze the resource graph to find dependencies
	
	return affected
}

// assessSecurityImpact assesses security impact
func (ia *ImpactAnalyzer) assessSecurityImpact(plan *RemediationPlan) *SecurityImpact {
	impact := &SecurityImpact{
		CurrentScore:     5.0,
		NewScore:         5.0,
		Improvements:     []string{},
		NewRisks:         []string{},
		ComplianceImpact: []string{},
	}
	
	// Analyze security improvements
	for _, change := range plan.Changes {
		if strings.Contains(change.Path, "encryption") && change.NewValue == true {
			impact.Improvements = append(impact.Improvements, "Enabled encryption")
			impact.NewScore += 1.0
		}
		
		if strings.Contains(change.Path, "public_access") && change.NewValue == false {
			impact.Improvements = append(impact.Improvements, "Disabled public access")
			impact.NewScore += 1.0
		}
	}
	
	return impact
}

// assessPerformanceImpact assesses performance impact
func (ia *ImpactAnalyzer) assessPerformanceImpact(plan *RemediationPlan) *PerformanceImpact {
	impact := &PerformanceImpact{
		LatencyChange:      "0%",
		ThroughputChange:   "0%",
		AvailabilityChange: "0%",
		Metrics:            []string{},
	}
	
	// Analyze performance changes
	for _, change := range plan.Changes {
		if strings.Contains(change.Path, "instance_type") || strings.Contains(change.Path, "size") {
			impact.Metrics = append(impact.Metrics, "compute_capacity")
		}
		
		if strings.Contains(change.Path, "storage") || strings.Contains(change.Path, "disk") {
			impact.Metrics = append(impact.Metrics, "storage_capacity")
		}
	}
	
	return impact
}

// CostCalculator calculates cost impacts
type CostCalculator struct{}

// NewCostCalculator creates a new cost calculator
func NewCostCalculator() *CostCalculator {
	return &CostCalculator{}
}

// CalculateCostImpact calculates the cost impact of a remediation
func (cc *CostCalculator) CalculateCostImpact(plan *RemediationPlan) *CostImpact {
	// This would integrate with cloud pricing APIs
	// For now, return estimated values
	return &CostImpact{
		CurrentMonthlyCost: 100.00,
		NewMonthlyCost:     120.00,
		MonthlySavings:     -20.00,
		OneTimeCost:        0.00,
		Currency:           "USD",
	}
}

// RiskAssessor assesses risk levels
type RiskAssessor struct{}

// NewRiskAssessor creates a new risk assessor
func NewRiskAssessor() *RiskAssessor {
	return &RiskAssessor{}
}

// AssessRisk assesses the risk of a remediation plan
func (ra *RiskAssessor) AssessRisk(plan *RemediationPlan) float64 {
	riskScore := 0.0
	
	// Base risk by remediation type
	switch plan.Type {
	case RemediationTypeDelete:
		riskScore = 7.0
	case RemediationTypeReplace:
		riskScore = 6.0
	case RemediationTypeUpdate:
		riskScore = 3.0
	case RemediationTypeCreate:
		riskScore = 2.0
	case RemediationTypeImport:
		riskScore = 1.0
	}
	
	// Adjust based on resource criticality
	criticalTypes := []string{"database", "rds", "sql", "storage", "network", "security"}
	for _, critical := range criticalTypes {
		if strings.Contains(strings.ToLower(plan.ResourceType), critical) {
			riskScore += 2.0
			break
		}
	}
	
	// Adjust based on environment
	if tags, ok := plan.CurrentState["tags"].(map[string]interface{}); ok {
		if env, ok := tags["environment"].(string); ok {
			switch env {
			case "production":
				riskScore += 3.0
			case "staging":
				riskScore += 1.0
			}
		}
	}
	
	// Cap at 10.0
	if riskScore > 10.0 {
		riskScore = 10.0
	}
	
	return riskScore
}

// ApprovalManager manages approval workflows
type ApprovalManager struct {
	config    *RemediationConfig
	approvers map[string]Approver
}

// Approver represents an approval authority
type Approver struct {
	Name      string   `json:"name"`
	Email     string   `json:"email"`
	MaxRisk   float64  `json:"max_risk"`
	Resources []string `json:"resources"`
}

// NewApprovalManager creates a new approval manager
func NewApprovalManager(config *RemediationConfig) *ApprovalManager {
	return &ApprovalManager{
		config:    config,
		approvers: make(map[string]Approver),
	}
}

// RequestApproval requests approval for a plan
func (am *ApprovalManager) RequestApproval(ctx context.Context, plan *RemediationPlan) (bool, error) {
	// In a real implementation, this would:
	// 1. Send notifications to approvers
	// 2. Wait for approval response
	// 3. Track approval status
	
	// For now, auto-approve if risk is below threshold
	if plan.EstimatedImpact != nil && plan.EstimatedImpact.RiskScore < am.config.RequireApprovalAbove {
		return true, nil
	}
	
	// Simulate approval process
	fmt.Printf("⚠️  Plan %s requires manual approval (risk score: %.1f)\n", plan.ID, plan.EstimatedImpact.RiskScore)
	fmt.Printf("Approval reason: %s\n", plan.EstimatedImpact.ApprovalReason)
	
	// In production, this would wait for actual approval
	return false, fmt.Errorf("manual approval required")
}

// AuditLogger logs audit events
type AuditLogger struct {
	logPath string
	mu      sync.Mutex
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logPath string) *AuditLogger {
	return &AuditLogger{
		logPath: logPath,
	}
}

// Log logs an audit event
func (al *AuditLogger) Log(level, message string, context map[string]string) {
	al.mu.Lock()
	defer al.mu.Unlock()
	
	entry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     level,
		"message":   message,
		"context":   context,
	}
	
	data, _ := json.Marshal(entry)
	
	file, err := os.OpenFile(al.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()
	
	file.WriteString(string(data) + "\n")
}

// Helper function to get required attributes for a resource type
func getRequiredAttributesForType(resourceType string) []string {
	requiredMap := map[string][]string{
		"aws_instance":          {"ami", "instance_type"},
		"aws_s3_bucket":         {"bucket"},
		"azurerm_virtual_machine": {"name", "location", "resource_group_name"},
		"google_compute_instance": {"name", "machine_type", "zone"},
	}
	
	if attrs, ok := requiredMap[resourceType]; ok {
		return attrs
	}
	
	return []string{}
}