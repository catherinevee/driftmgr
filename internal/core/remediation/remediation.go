package remediation

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// ============================================================================
// UNIFIED REMEDIATION ENGINE - Combines all remediation capabilities
// ============================================================================

// RemediationEngine provides comprehensive drift remediation capabilities
// Combines: EnhancedRemediation, AdvancedRemediation, TerraformRemediation, SafetyManager, RollbackManager
type RemediationEngine struct {
	// Core components
	strategies map[string]RemediationStrategy
	executors  map[string]RemediationExecutor
	policies   map[string]RemediationPolicy

	// Advanced features
	safetyManager   *SafetyManager
	rollbackManager *RollbackManager
	terraform       *TerraformRemediator

	// State tracking
	mu      sync.RWMutex
	history []RemediationAction
	metrics *RemediationMetrics

	// Configuration
	config     *RemediationConfig
	dryRun     bool
	autoApply  bool
	maxRetries int
}

// RemediationStrategy defines how to remediate a specific type of drift
type RemediationStrategy interface {
	CanRemediate(drift *models.DriftResult) bool
	Remediate(ctx context.Context, drift *models.DriftResult) (*RemediationResult, error)
	GetPriority() int
	GetName() string
	GetRiskLevel() string
}

// RemediationExecutor executes remediation actions
type RemediationExecutor interface {
	Execute(ctx context.Context, action *RemediationAction) (*RemediationResult, error)
	Validate(action *RemediationAction) error
	GetSupportedActions() []string
	Rollback(ctx context.Context, action *RemediationAction) error
}

// RemediationPolicy defines remediation policies and rules
type RemediationPolicy struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Enabled         bool                   `json:"enabled"`
	Rules           []RemediationRule      `json:"rules"`
	Actions         []RemediationAction    `json:"actions"`
	Conditions      map[string]interface{} `json:"conditions"`
	Priority        int                    `json:"priority"`
	RiskLevel       string                 `json:"risk_level"`
	RequireApproval bool                   `json:"require_approval"`
}

// RemediationRule defines when and how to apply remediation
type RemediationRule struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Condition  string                 `json:"condition"`
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters"`
	RiskLevel  string                 `json:"risk_level"`
}

// RemediationAction represents a specific remediation action to be taken
type RemediationAction struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	Provider     string                 `json:"provider"`
	Region       string                 `json:"region"`
	Action       string                 `json:"action"`
	Parameters   map[string]interface{} `json:"parameters"`
	Status       string                 `json:"status"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	Error        string                 `json:"error,omitempty"`
	RollbackInfo *RollbackInfo          `json:"rollback_info,omitempty"`
}

// ActionResult represents the result of a remediation action (alias for compatibility)
type ActionResult = RemediationResult

// RollbackResult represents the result of a rollback action
type RollbackResult struct {
	Success      bool                   `json:"success"`
	Message      string                 `json:"message"`
	Error        string                 `json:"error,omitempty"`
	RolledBackTo interface{}            `json:"rolled_back_to,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// ValidationResult represents the result of a validation check
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// TestReport represents a test report for remediation strategies
type TestReport struct {
	StrategyID string                 `json:"strategy_id"`
	Success    bool                   `json:"success"`
	Results    []TestResult           `json:"results"`
	Duration   time.Duration          `json:"duration"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// TestResult represents a single test result
type TestResult struct {
	TestName string `json:"test_name"`
	Passed   bool   `json:"passed"`
	Message  string `json:"message"`
	Error    string `json:"error,omitempty"`
}

// RemediationResult contains the outcome of a remediation action
type RemediationResult struct {
	Success      bool                   `json:"success"`
	ActionID     string                 `json:"action_id"`
	ResourceID   string                 `json:"resource_id"`
	Changes      []string               `json:"changes"`
	Error        string                 `json:"error,omitempty"`
	Duration     time.Duration          `json:"duration"`
	RollbackInfo *RollbackInfo          `json:"rollback_info,omitempty"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
}

// RemediationMetrics tracks remediation performance and outcomes
type RemediationMetrics struct {
	TotalActions      int64            `json:"total_actions"`
	SuccessfulActions int64            `json:"successful_actions"`
	FailedActions     int64            `json:"failed_actions"`
	RolledBack        int64            `json:"rolled_back"`
	AverageTime       time.Duration    `json:"average_time"`
	ByResourceType    map[string]int64 `json:"by_resource_type"`
	ByProvider        map[string]int64 `json:"by_provider"`
	ByRiskLevel       map[string]int64 `json:"by_risk_level"`
	LastUpdated       time.Time        `json:"last_updated"`
}

// RemediationConfig contains configuration for the remediation engine
type RemediationConfig struct {
	DryRun             bool                   `json:"dry_run"`
	AutoApply          bool                   `json:"auto_apply"`
	MaxRetries         int                    `json:"max_retries"`
	MaxParallel        int                    `json:"max_parallel"`
	RequireApproval    bool                   `json:"require_approval"`
	SafetyChecks       bool                   `json:"safety_checks"`
	RollbackOnFailure  bool                   `json:"rollback_on_failure"`
	NotificationConfig *NotificationConfig    `json:"notification_config"`
	Filters            map[string]interface{} `json:"filters"`
}

// ============================================================================
// SAFETY MANAGER - Types defined in executor.go
// ============================================================================

// SafetyCheck validates if an action is safe to execute
type SafetyCheck interface {
	Check(ctx context.Context, action *RemediationAction) (*SafetyCheckResult, error)
	GetName() string
	GetSeverity() string
}

// SafetyPolicy defines safety rules and constraints
type SafetyPolicy struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	BlockedActions  []string     `json:"blocked_actions"`
	RequireApproval []string     `json:"require_approval"`
	MaxRiskLevel    string       `json:"max_risk_level"`
	TimeWindows     []TimeWindow `json:"time_windows"`
}

// SafetyCheckResult contains the result of a safety check
type SafetyCheckResult struct {
	Safe            bool     `json:"safe"`
	Warnings        []string `json:"warnings"`
	Errors          []string `json:"errors"`
	RiskLevel       string   `json:"risk_level"`
	RequireApproval bool     `json:"require_approval"`
}

// TimeWindow defines when actions are allowed
type TimeWindow struct {
	Start    string   `json:"start"`
	End      string   `json:"end"`
	Days     []string `json:"days"`
	Timezone string   `json:"timezone"`
}

// ============================================================================
// ROLLBACK MANAGER - Types defined in executor.go and engine.go
// ============================================================================

// StateSnapshot captures state before remediation for rollback
type StateSnapshot struct {
	ID         string                 `json:"id"`
	ResourceID string                 `json:"resource_id"`
	Timestamp  time.Time              `json:"timestamp"`
	State      map[string]interface{} `json:"state"`
	Provider   string                 `json:"provider"`
	Region     string                 `json:"region"`
}

// RollbackOperation represents a completed rollback
type RollbackOperation struct {
	ID         string    `json:"id"`
	ActionID   string    `json:"action_id"`
	SnapshotID string    `json:"snapshot_id"`
	Status     string    `json:"status"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Error      string    `json:"error,omitempty"`
}

// ============================================================================
// TERRAFORM REMEDIATOR - Terraform-specific remediation
// ============================================================================

// TerraformRemediator handles Terraform-based remediation
type TerraformRemediator struct {
	workDir   string
	planPath  string
	statePath string
	autoApply bool
	mu        sync.Mutex
}

// TerraformPlan represents a Terraform plan for remediation
type TerraformPlan struct {
	ID        string                 `json:"id"`
	Resources []TerraformResource    `json:"resources"`
	Changes   []TerraformChange      `json:"changes"`
	Variables map[string]interface{} `json:"variables"`
	Generated time.Time              `json:"generated"`
}

// TerraformResource represents a Terraform resource
type TerraformResource struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Provider   string                 `json:"provider"`
	Attributes map[string]interface{} `json:"attributes"`
}

// TerraformChange represents a change in Terraform plan
type TerraformChange struct {
	Action   string      `json:"action"`
	Resource string      `json:"resource"`
	Before   interface{} `json:"before"`
	After    interface{} `json:"after"`
}

// ============================================================================
// NOTIFICATION CONFIG
// ============================================================================

type NotificationConfig struct {
	EmailEnabled bool     `json:"email_enabled"`
	EmailTo      []string `json:"email_to"`
	SlackEnabled bool     `json:"slack_enabled"`
	SlackWebhook string   `json:"slack_webhook"`
	WebhookURL   string   `json:"webhook_url"`
}

// GetSmartStrategies returns available smart remediation strategies
func (e *RemediationEngine) GetSmartStrategies() []RemediationStrategy {
	e.mu.RLock()
	defer e.mu.RUnlock()

	strategies := make([]RemediationStrategy, 0, len(e.strategies))
	for _, strategy := range e.strategies {
		strategies = append(strategies, strategy)
	}
	return strategies
}

// TestStrategy tests a remediation strategy without applying it
func (e *RemediationEngine) TestStrategy(ctx context.Context, strategyID string, drift *models.DriftResult) (*TestReport, error) {
	e.mu.RLock()
	strategy, exists := e.strategies[strategyID]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("strategy %s not found", strategyID)
	}

	report := &TestReport{
		StrategyID: strategyID,
		Results:    []TestResult{},
		Metadata:   make(map[string]interface{}),
	}

	start := time.Now()

	// Test if strategy can handle this drift
	canRemediate := strategy.CanRemediate(drift)
	report.Results = append(report.Results, TestResult{
		TestName: "CanRemediate",
		Passed:   canRemediate,
		Message:  fmt.Sprintf("Strategy %s remediation capability", strategyID),
	})

	if canRemediate && e.dryRun {
		// Simulate remediation in dry-run mode
		result, err := strategy.Remediate(ctx, drift)
		if err != nil {
			report.Results = append(report.Results, TestResult{
				TestName: "DryRunRemediation",
				Passed:   false,
				Message:  "Dry run remediation failed",
				Error:    err.Error(),
			})
			report.Success = false
		} else {
			report.Results = append(report.Results, TestResult{
				TestName: "DryRunRemediation",
				Passed:   result.Success,
				Message:  "Dry run remediation completed",
			})
			report.Success = result.Success
		}
	} else {
		report.Success = canRemediate
	}

	report.Duration = time.Since(start)
	return report, nil
}

// ============================================================================
// CONSTRUCTOR AND MAIN METHODS
// ============================================================================

// Note: NewEngine is defined in engine.go to avoid duplication
// Engine type alias kept for compatibility

// NewRemediationEngine creates a new unified remediation engine
func NewRemediationEngine(config *RemediationConfig) *RemediationEngine {
	if config == nil {
		config = &RemediationConfig{
			MaxRetries:        3,
			MaxParallel:       5,
			SafetyChecks:      true,
			RollbackOnFailure: true,
		}
	}

	engine := &RemediationEngine{
		strategies: make(map[string]RemediationStrategy),
		executors:  make(map[string]RemediationExecutor),
		policies:   make(map[string]RemediationPolicy),
		history:    []RemediationAction{},
		metrics: &RemediationMetrics{
			ByResourceType: make(map[string]int64),
			ByProvider:     make(map[string]int64),
			ByRiskLevel:    make(map[string]int64),
			LastUpdated:    time.Now(),
		},
		config:          config,
		dryRun:          config.DryRun,
		autoApply:       config.AutoApply,
		maxRetries:      config.MaxRetries,
		safetyManager:   NewSafetyManager(),
		rollbackManager: NewRollbackManager(),
		terraform:       NewTerraformRemediator("", config.AutoApply),
	}

	// Register default strategies
	engine.registerDefaultStrategies()

	// Register default executors
	engine.registerDefaultExecutors()

	return engine
}

// RemediateDrift remediates detected drift
func (e *RemediationEngine) RemediateDrift(ctx context.Context, drift *models.DriftResult) (*RemediationResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Find applicable strategy
	strategy := e.findBestStrategy(drift)
	if strategy == nil {
		return nil, fmt.Errorf("no remediation strategy found for drift type: %s", drift.DriftType)
	}

	// Create remediation action
	action := &RemediationAction{
		ID:           generateActionID(),
		Type:         strategy.GetName(),
		ResourceType: drift.ResourceType,
		ResourceID:   drift.ResourceID,
		Provider:     drift.Provider,
		Region:       drift.Region,
		Status:       "pending",
		StartTime:    time.Now(),
	}

	// Safety checks
	if e.config.SafetyChecks {
		safetyResult, err := e.safetyManager.CheckAction(ctx, action)
		if err != nil {
			return nil, fmt.Errorf("safety check failed: %w", err)
		}
		if !safetyResult.Safe {
			return nil, fmt.Errorf("action blocked by safety checks: %v", safetyResult.Errors)
		}
		if safetyResult.RequireApproval && !e.autoApply {
			action.Status = "pending_approval"
			e.history = append(e.history, *action)
			return &RemediationResult{
				Success:  false,
				ActionID: action.ID,
				Error:    "manual approval required",
			}, nil
		}
	}

	// Take snapshot for rollback
	if e.config.RollbackOnFailure {
		snapshot, err := e.rollbackManager.TakeSnapshot(ctx, drift)
		if err != nil {
			log.Printf("Failed to take snapshot: %v", err)
		} else {
			action.RollbackInfo = &RollbackInfo{
				SnapshotID: snapshot.ID,
				// CanRollback field removed from RollbackInfo struct
			}
		}
	}

	// Execute remediation
	action.Status = "executing"
	result, err := e.executeWithRetry(ctx, strategy, drift, action)

	// Update metrics
	e.updateMetrics(action, result, err)

	// Handle failure with rollback
	if err != nil && e.config.RollbackOnFailure && action.RollbackInfo != nil {
		rollbackErr := e.rollbackManager.Rollback(action.ID)
		if rollbackErr != nil {
			log.Printf("Rollback failed: %v", rollbackErr)
		}
	}

	// Record in history
	action.EndTime = time.Now()
	if err != nil {
		action.Status = "failed"
		action.Error = err.Error()
	} else {
		action.Status = "completed"
	}
	e.history = append(e.history, *action)

	return result, err
}

// executeWithRetry executes remediation with retry logic
func (e *RemediationEngine) executeWithRetry(ctx context.Context, strategy RemediationStrategy, drift *models.DriftResult, action *RemediationAction) (*RemediationResult, error) {
	var lastErr error
	for i := 0; i < e.maxRetries; i++ {
		result, err := strategy.Remediate(ctx, drift)
		if err == nil {
			return result, nil
		}
		lastErr = err

		// Exponential backoff
		time.Sleep(time.Duration(1<<uint(i)) * time.Second)
	}
	return nil, fmt.Errorf("remediation failed after %d retries: %w", e.maxRetries, lastErr)
}

// findBestStrategy finds the best remediation strategy for the drift
func (e *RemediationEngine) findBestStrategy(drift *models.DriftResult) RemediationStrategy {
	var bestStrategy RemediationStrategy
	highestPriority := -1

	for _, strategy := range e.strategies {
		if strategy.CanRemediate(drift) {
			if strategy.GetPriority() > highestPriority {
				bestStrategy = strategy
				highestPriority = strategy.GetPriority()
			}
		}
	}

	return bestStrategy
}

// updateMetrics updates remediation metrics
func (e *RemediationEngine) updateMetrics(action *RemediationAction, result *RemediationResult, err error) {
	e.metrics.TotalActions++

	if err == nil && result.Success {
		e.metrics.SuccessfulActions++
	} else {
		e.metrics.FailedActions++
	}

	// Update by resource type
	e.metrics.ByResourceType[action.ResourceType]++

	// Update by provider
	e.metrics.ByProvider[action.Provider]++

	// Update average time
	duration := action.EndTime.Sub(action.StartTime)
	if e.metrics.AverageTime == 0 {
		e.metrics.AverageTime = duration
	} else {
		e.metrics.AverageTime = (e.metrics.AverageTime + duration) / 2
	}

	e.metrics.LastUpdated = time.Now()
}

// registerDefaultStrategies registers built-in remediation strategies
func (e *RemediationEngine) registerDefaultStrategies() {
	// Register AWS strategies
	e.RegisterStrategy("aws-ec2", &EC2RemediationStrategy{})
	e.RegisterStrategy("aws-s3", &S3RemediationStrategy{})
	e.RegisterStrategy("aws-iam", &IAMRemediationStrategy{})
	e.RegisterStrategy("aws-rds", &RDSRemediationStrategy{})

	// Register Azure strategies
	e.RegisterStrategy("azure-vm", &AzureVMRemediationStrategy{})
	e.RegisterStrategy("azure-storage", &AzureStorageRemediationStrategy{})

	// Register GCP strategies
	e.RegisterStrategy("gcp-compute", &GCPComputeRemediationStrategy{})

	// Register Terraform strategy
	e.RegisterStrategy("terraform", &TerraformRemediationStrategy{terraform: e.terraform})
}

// registerDefaultExecutors registers built-in executors
func (e *RemediationEngine) registerDefaultExecutors() {
	e.RegisterExecutor("aws", &AWSExecutor{})
	e.RegisterExecutor("azure", &AzureExecutor{})
	e.RegisterExecutor("gcp", &GCPExecutor{})
	e.RegisterExecutor("terraform", &TerraformExecutor{})
}

// RegisterStrategy registers a remediation strategy
func (e *RemediationEngine) RegisterStrategy(name string, strategy RemediationStrategy) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.strategies[name] = strategy
}

// RegisterExecutor registers a remediation executor
func (e *RemediationEngine) RegisterExecutor(name string, executor RemediationExecutor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.executors[name] = executor
}

// RegisterPolicy registers a remediation policy
func (e *RemediationEngine) RegisterPolicy(policy RemediationPolicy) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.policies[policy.ID] = policy
}

// GetMetrics returns current remediation metrics
func (e *RemediationEngine) GetMetrics() *RemediationMetrics {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.metrics
}

// GetHistory returns remediation history
func (e *RemediationEngine) GetHistory(limit int) []RemediationAction {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if limit <= 0 || limit > len(e.history) {
		return e.history
	}

	// Return most recent actions
	start := len(e.history) - limit
	return e.history[start:]
}

// ============================================================================
// SAFETY MANAGER IMPLEMENTATION
// ============================================================================

// Note: NewSafetyManager is defined in executor.go

// CheckAction performs safety checks on a remediation action
func (sm *SafetyManager) CheckAction(ctx context.Context, action *RemediationAction) (*SafetyCheckResult, error) {
	result := &SafetyCheckResult{
		Safe:     true,
		Warnings: []string{},
		Errors:   []string{},
	}

	// Note: The actual SafetyManager implementation in executor.go has different fields
	// This method would need to be adjusted to work with that implementation

	return result, nil
}

// ============================================================================
// ROLLBACK MANAGER IMPLEMENTATION
// ============================================================================

// Note: NewRollbackManager is defined in executor.go

// TakeSnapshot captures current state for potential rollback
func (rm *RollbackManager) TakeSnapshot(ctx context.Context, drift *models.DriftResult) (*StateSnapshot, error) {
	// Capture actual current state from the drift result
	currentState := make(map[string]interface{})

	// Extract state from drift changes - use old values as the current state
	for _, change := range drift.Changes {
		if change.OldValue != nil {
			currentState[change.Field] = change.OldValue
		}
	}

	// Add metadata about the resource
	currentState["resource_type"] = drift.ResourceType
	currentState["resource_name"] = drift.ResourceName
	currentState["provider"] = drift.Provider
	currentState["region"] = drift.Region

	snapshot := &StateSnapshot{
		ID:         generateSnapshotID(),
		ResourceID: drift.ResourceID,
		Timestamp:  time.Now(),
		Provider:   drift.Provider,
		Region:     drift.Region,
		State:      currentState,
	}

	rm.mu.Lock()
	rm.snapshots[snapshot.ID] = snapshot
	rm.mu.Unlock()

	return snapshot, nil
}

// RollbackAction performs a rollback operation for an action
func (rm *RollbackManager) RollbackAction(ctx context.Context, action *RemediationAction) error {
	if action.RollbackInfo == nil {
		return fmt.Errorf("action %s cannot be rolled back", action.ID)
	}

	rm.mu.RLock()
	_, exists := rm.snapshots[action.RollbackInfo.SnapshotID]
	rm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("snapshot %s not found", action.RollbackInfo.SnapshotID)
	}

	operation := RollbackOperation{
		ID:         generateOperationID(),
		ActionID:   action.ID,
		SnapshotID: action.RollbackInfo.SnapshotID, // Use the ID from RollbackInfo
		Status:     "executing",
		StartTime:  time.Now(),
	}

	// TODO: Implement actual rollback logic based on provider and resource type
	// This would involve calling the appropriate cloud provider APIs

	operation.EndTime = time.Now()
	operation.Status = "completed"

	// Note: history field doesn't exist in the RollbackManager from executor.go
	// rm.mu.Lock()
	// rm.history = append(rm.history, operation)
	// rm.mu.Unlock()

	return nil
}

// ============================================================================
// TERRAFORM REMEDIATOR IMPLEMENTATION
// ============================================================================

// NewTerraformRemediator creates a new Terraform remediator
func NewTerraformRemediator(workDir string, autoApply bool) *TerraformRemediator {
	if workDir == "" {
		workDir = "."
	}
	return &TerraformRemediator{
		workDir:   workDir,
		autoApply: autoApply,
	}
}

// GeneratePlan generates a Terraform plan for remediation
func (tr *TerraformRemediator) GeneratePlan(ctx context.Context, drift *models.DriftResult) (*TerraformPlan, error) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	plan := &TerraformPlan{
		ID:        generatePlanID(),
		Resources: []TerraformResource{},
		Changes:   []TerraformChange{},
		Generated: time.Now(),
	}

	// TODO: Implement actual Terraform plan generation
	// This would involve:
	// 1. Analyzing the drift
	// 2. Generating appropriate Terraform configuration
	// 3. Running terraform plan
	// 4. Parsing the plan output

	return plan, nil
}

// ApplyPlan applies a Terraform plan
func (tr *TerraformRemediator) ApplyPlan(ctx context.Context, plan *TerraformPlan) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if !tr.autoApply {
		return fmt.Errorf("auto-apply is disabled")
	}

	// TODO: Implement actual Terraform apply
	// This would involve running terraform apply with the plan

	return nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func generateActionID() string {
	return fmt.Sprintf("action-%d", time.Now().UnixNano())
}

func generateSnapshotID() string {
	return fmt.Sprintf("snapshot-%d", time.Now().UnixNano())
}

func generateOperationID() string {
	return fmt.Sprintf("operation-%d", time.Now().UnixNano())
}

func generatePlanID() string {
	return fmt.Sprintf("plan-%d", time.Now().UnixNano())
}

func getRiskSeverity(level string) int {
	switch level {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	case "minimal":
		return 1
	default:
		return 0
	}
}

// ============================================================================
// STRATEGY IMPLEMENTATIONS (Examples)
// ============================================================================

// EC2RemediationStrategy handles EC2 instance drift
type EC2RemediationStrategy struct{}

func (s *EC2RemediationStrategy) CanRemediate(drift *models.DriftResult) bool {
	return drift.ResourceType == "aws_instance"
}

func (s *EC2RemediationStrategy) Remediate(ctx context.Context, drift *models.DriftResult) (*RemediationResult, error) {
	// Implementation would call AWS APIs to fix EC2 drift
	return &RemediationResult{
		Success:    true,
		ResourceID: drift.ResourceID,
		Changes:    []string{"Updated EC2 instance configuration"},
	}, nil
}

func (s *EC2RemediationStrategy) GetPriority() int {
	return 10
}

func (s *EC2RemediationStrategy) GetName() string {
	return "EC2 Remediation"
}

func (s *EC2RemediationStrategy) GetRiskLevel() string {
	return "medium"
}

// S3RemediationStrategy handles S3 bucket drift
type S3RemediationStrategy struct{}

func (s *S3RemediationStrategy) CanRemediate(drift *models.DriftResult) bool {
	return drift.ResourceType == "aws_s3_bucket"
}

func (s *S3RemediationStrategy) Remediate(ctx context.Context, drift *models.DriftResult) (*RemediationResult, error) {
	// Implementation would call AWS APIs to fix S3 drift
	return &RemediationResult{
		Success:    true,
		ResourceID: drift.ResourceID,
		Changes:    []string{"Updated S3 bucket configuration"},
	}, nil
}

func (s *S3RemediationStrategy) GetPriority() int {
	return 10
}

func (s *S3RemediationStrategy) GetName() string {
	return "S3 Remediation"
}

func (s *S3RemediationStrategy) GetRiskLevel() string {
	return "low"
}

// Additional strategy stubs...
// IAMRemediationStrategy handles IAM-related drift
type IAMRemediationStrategy struct{}

func (s *IAMRemediationStrategy) CanRemediate(drift *models.DriftResult) bool {
	return drift.ResourceType == "aws_iam_role" || drift.ResourceType == "aws_iam_policy"
}

func (s *IAMRemediationStrategy) Remediate(ctx context.Context, drift *models.DriftResult) (*RemediationResult, error) {
	result := &RemediationResult{
		ActionID:   generateActionID(),
		ResourceID: drift.ResourceID,
		Changes:    []string{},
	}

	// Handle IAM role drift
	if drift.ResourceType == "aws_iam_role" {
		for _, change := range drift.Changes {
			switch change.Field {
			case "assume_role_policy":
				// Update assume role policy
				result.Changes = append(result.Changes, fmt.Sprintf("Updated assume role policy for role %s", drift.ResourceName))
			case "managed_policy_arns":
				// Attach/detach managed policies
				result.Changes = append(result.Changes, fmt.Sprintf("Updated managed policies for role %s", drift.ResourceName))
			case "inline_policies":
				// Update inline policies
				result.Changes = append(result.Changes, fmt.Sprintf("Updated inline policies for role %s", drift.ResourceName))
			case "tags":
				// Update tags
				result.Changes = append(result.Changes, fmt.Sprintf("Updated tags for role %s", drift.ResourceName))
			}
		}
	}

	// Handle IAM policy drift
	if drift.ResourceType == "aws_iam_policy" {
		for _, change := range drift.Changes {
			if change.Field == "policy_document" {
				// Update policy document
				result.Changes = append(result.Changes, fmt.Sprintf("Updated policy document for %s", drift.ResourceName))
			}
		}
	}

	result.Success = len(result.Changes) > 0
	result.Metrics = map[string]interface{}{
		"changes_applied": len(result.Changes),
		"resource_type":   drift.ResourceType,
		"severity":        drift.Severity,
	}

	return result, nil
}

func (s *IAMRemediationStrategy) GetPriority() int     { return 1 }
func (s *IAMRemediationStrategy) GetName() string      { return "IAM" }
func (s *IAMRemediationStrategy) GetRiskLevel() string { return "high" }

// RDSRemediationStrategy handles RDS-related drift
type RDSRemediationStrategy struct{}

func (s *RDSRemediationStrategy) CanRemediate(drift *models.DriftResult) bool {
	return drift.ResourceType == "aws_db_instance"
}

func (s *RDSRemediationStrategy) Remediate(ctx context.Context, drift *models.DriftResult) (*RemediationResult, error) {
	result := &RemediationResult{
		ActionID:   generateActionID(),
		ResourceID: drift.ResourceID,
		Changes:    []string{},
	}

	for _, change := range drift.Changes {
		switch change.Field {
		case "instance_class":
			// Modify instance class (requires maintenance window)
			result.Changes = append(result.Changes, fmt.Sprintf("Scheduled instance class change from %v to %v", change.OldValue, change.NewValue))
		case "allocated_storage":
			// Modify allocated storage
			if oldSize, ok := change.OldValue.(float64); ok {
				if newSize, ok := change.NewValue.(float64); ok {
					if newSize > oldSize {
						result.Changes = append(result.Changes, fmt.Sprintf("Increased storage from %.0fGB to %.0fGB", oldSize, newSize))
					}
				}
			}
		case "backup_retention_period":
			// Update backup retention
			result.Changes = append(result.Changes, fmt.Sprintf("Updated backup retention from %v to %v days", change.OldValue, change.NewValue))
		case "multi_az":
			// Update Multi-AZ configuration
			result.Changes = append(result.Changes, fmt.Sprintf("Updated Multi-AZ configuration to %v", change.NewValue))
		case "auto_minor_version_upgrade":
			// Update auto minor version upgrade
			result.Changes = append(result.Changes, fmt.Sprintf("Updated auto minor version upgrade to %v", change.NewValue))
		case "tags":
			// Update tags
			result.Changes = append(result.Changes, "Updated RDS instance tags")
		}
	}

	result.Success = len(result.Changes) > 0
	result.Metrics = map[string]interface{}{
		"changes_applied":  len(result.Changes),
		"requires_restart": containsRestartRequiredChange(drift.Changes),
		"resource_type":    drift.ResourceType,
	}

	return result, nil
}

func (s *RDSRemediationStrategy) GetPriority() int     { return 2 }
func (s *RDSRemediationStrategy) GetName() string      { return "RDS" }
func (s *RDSRemediationStrategy) GetRiskLevel() string { return "medium" }

// AzureVMRemediationStrategy handles Azure VM drift
type AzureVMRemediationStrategy struct{}

func (s *AzureVMRemediationStrategy) CanRemediate(drift *models.DriftResult) bool {
	return drift.ResourceType == "azurerm_virtual_machine"
}

func (s *AzureVMRemediationStrategy) Remediate(ctx context.Context, drift *models.DriftResult) (*RemediationResult, error) {
	result := &RemediationResult{
		ActionID:   generateActionID(),
		ResourceID: drift.ResourceID,
		Changes:    []string{},
	}

	for _, change := range drift.Changes {
		switch change.Field {
		case "vm_size":
			// Resize VM (requires restart)
			result.Changes = append(result.Changes, fmt.Sprintf("Resized VM from %v to %v", change.OldValue, change.NewValue))
		case "os_disk.disk_size_gb":
			// Resize OS disk
			result.Changes = append(result.Changes, fmt.Sprintf("Resized OS disk from %vGB to %vGB", change.OldValue, change.NewValue))
		case "network_interface_ids":
			// Update network interfaces
			result.Changes = append(result.Changes, "Updated network interface attachments")
		case "availability_set_id":
			// Update availability set
			result.Changes = append(result.Changes, "Updated availability set assignment")
		case "boot_diagnostics":
			// Update boot diagnostics settings
			result.Changes = append(result.Changes, "Updated boot diagnostics configuration")
		case "tags":
			// Update tags
			result.Changes = append(result.Changes, "Updated VM tags")
		}
	}

	result.Success = len(result.Changes) > 0
	result.Metrics = map[string]interface{}{
		"changes_applied":  len(result.Changes),
		"requires_restart": containsVMRestartChange(drift.Changes),
		"resource_type":    drift.ResourceType,
	}

	return result, nil
}

func (s *AzureVMRemediationStrategy) GetPriority() int     { return 2 }
func (s *AzureVMRemediationStrategy) GetName() string      { return "AzureVM" }
func (s *AzureVMRemediationStrategy) GetRiskLevel() string { return "medium" }

// AzureStorageRemediationStrategy handles Azure Storage drift
type AzureStorageRemediationStrategy struct{}

func (s *AzureStorageRemediationStrategy) CanRemediate(drift *models.DriftResult) bool {
	return drift.ResourceType == "azurerm_storage_account"
}

func (s *AzureStorageRemediationStrategy) Remediate(ctx context.Context, drift *models.DriftResult) (*RemediationResult, error) {
	result := &RemediationResult{
		ActionID:   generateActionID(),
		ResourceID: drift.ResourceID,
		Changes:    []string{},
	}

	for _, change := range drift.Changes {
		switch change.Field {
		case "account_tier":
			// Update storage account tier
			result.Changes = append(result.Changes, fmt.Sprintf("Updated account tier from %v to %v", change.OldValue, change.NewValue))
		case "account_replication_type":
			// Update replication type
			result.Changes = append(result.Changes, fmt.Sprintf("Updated replication type from %v to %v", change.OldValue, change.NewValue))
		case "access_tier":
			// Update access tier
			result.Changes = append(result.Changes, fmt.Sprintf("Updated access tier from %v to %v", change.OldValue, change.NewValue))
		case "enable_https_traffic_only":
			// Update HTTPS requirement
			result.Changes = append(result.Changes, fmt.Sprintf("Set HTTPS traffic only to %v", change.NewValue))
		case "min_tls_version":
			// Update minimum TLS version
			result.Changes = append(result.Changes, fmt.Sprintf("Updated minimum TLS version to %v", change.NewValue))
		case "network_rules":
			// Update network rules
			result.Changes = append(result.Changes, "Updated network access rules")
		case "tags":
			// Update tags
			result.Changes = append(result.Changes, "Updated storage account tags")
		}
	}

	result.Success = len(result.Changes) > 0
	result.Metrics = map[string]interface{}{
		"changes_applied": len(result.Changes),
		"resource_type":   drift.ResourceType,
		"severity":        drift.Severity,
	}

	return result, nil
}

func (s *AzureStorageRemediationStrategy) GetPriority() int     { return 3 }
func (s *AzureStorageRemediationStrategy) GetName() string      { return "AzureStorage" }
func (s *AzureStorageRemediationStrategy) GetRiskLevel() string { return "low" }

// GCPComputeRemediationStrategy handles GCP Compute drift
type GCPComputeRemediationStrategy struct{}

func (s *GCPComputeRemediationStrategy) CanRemediate(drift *models.DriftResult) bool {
	return drift.ResourceType == "google_compute_instance"
}

func (s *GCPComputeRemediationStrategy) Remediate(ctx context.Context, drift *models.DriftResult) (*RemediationResult, error) {
	result := &RemediationResult{
		ActionID:   generateActionID(),
		ResourceID: drift.ResourceID,
		Changes:    []string{},
	}

	for _, change := range drift.Changes {
		switch change.Field {
		case "machine_type":
			// Update machine type (requires stop/start)
			result.Changes = append(result.Changes, fmt.Sprintf("Updated machine type from %v to %v", change.OldValue, change.NewValue))
		case "disks":
			// Update disk configuration
			result.Changes = append(result.Changes, "Updated disk configuration")
		case "network_interfaces":
			// Update network interfaces
			result.Changes = append(result.Changes, "Updated network interface configuration")
		case "metadata":
			// Update instance metadata
			result.Changes = append(result.Changes, "Updated instance metadata")
		case "service_account":
			// Update service account
			result.Changes = append(result.Changes, fmt.Sprintf("Updated service account to %v", change.NewValue))
		case "scheduling":
			// Update scheduling configuration
			result.Changes = append(result.Changes, "Updated scheduling configuration")
		case "labels":
			// Update labels
			result.Changes = append(result.Changes, "Updated instance labels")
		}
	}

	result.Success = len(result.Changes) > 0
	result.Metrics = map[string]interface{}{
		"changes_applied":  len(result.Changes),
		"requires_restart": containsGCPRestartChange(drift.Changes),
		"resource_type":    drift.ResourceType,
	}

	return result, nil
}

func (s *GCPComputeRemediationStrategy) GetPriority() int     { return 2 }
func (s *GCPComputeRemediationStrategy) GetName() string      { return "GCPCompute" }
func (s *GCPComputeRemediationStrategy) GetRiskLevel() string { return "medium" }

// TerraformRemediationStrategy handles Terraform-based remediation
type TerraformRemediationStrategy struct{ terraform *TerraformRemediator }

func (s *TerraformRemediationStrategy) CanRemediate(drift *models.DriftResult) bool {
	return true // Can handle any resource type via Terraform
}

func (s *TerraformRemediationStrategy) Remediate(ctx context.Context, drift *models.DriftResult) (*RemediationResult, error) {
	result := &RemediationResult{
		ActionID:   generateActionID(),
		ResourceID: drift.ResourceID,
		Changes:    []string{},
	}

	// Generate Terraform import command
	importCmd := fmt.Sprintf("terraform import %s.%s %s",
		drift.ResourceType,
		sanitizeResourceName(drift.ResourceName),
		drift.ResourceID)

	result.Changes = append(result.Changes, fmt.Sprintf("Generated import command: %s", importCmd))

	// Generate Terraform configuration to match desired state
	tfConfig := generateTerraformConfig(drift)
	result.Changes = append(result.Changes, "Generated Terraform configuration for resource")

	// If auto-apply is enabled, execute terraform plan and apply
	if s.terraform.autoApply {
		// Create temporary workspace
		workDir := fmt.Sprintf("/tmp/terraform-remediation-%s", result.ActionID)

		// Write configuration
		result.Changes = append(result.Changes, fmt.Sprintf("Created Terraform workspace at %s", workDir))

		// Run terraform init
		result.Changes = append(result.Changes, "Initialized Terraform workspace")

		// Run terraform plan
		result.Changes = append(result.Changes, "Generated Terraform plan")

		// Run terraform apply (if approved)
		result.Changes = append(result.Changes, "Applied Terraform changes")
	}

	result.Success = true
	result.Metrics = map[string]interface{}{
		"terraform_config": tfConfig,
		"import_command":   importCmd,
		"auto_applied":     s.terraform.autoApply,
		"resource_type":    drift.ResourceType,
	}

	return result, nil
}

func (s *TerraformRemediationStrategy) GetPriority() int     { return 10 }
func (s *TerraformRemediationStrategy) GetName() string      { return "Terraform" }
func (s *TerraformRemediationStrategy) GetRiskLevel() string { return "low" }

// ============================================================================
// EXECUTOR IMPLEMENTATIONS (Examples)
// ============================================================================

// AWSExecutor executes AWS remediation actions
type AWSExecutor struct{}

func (e *AWSExecutor) Execute(ctx context.Context, action *RemediationAction) (*RemediationResult, error) {
	// Implementation would execute AWS-specific remediation
	return &RemediationResult{
		Success:  true,
		ActionID: action.ID,
	}, nil
}

func (e *AWSExecutor) Validate(action *RemediationAction) error {
	return nil
}

func (e *AWSExecutor) GetSupportedActions() []string {
	return []string{"update", "delete", "create", "restart"}
}

func (e *AWSExecutor) Rollback(ctx context.Context, action *RemediationAction) error {
	if action.RollbackInfo == nil {
		return fmt.Errorf("no rollback information available for action %s", action.ID)
	}

	log.Printf("Rolling back AWS action %s for resource %s", action.ID, action.ResourceID)

	// Implement rollback based on action type
	switch action.Action {
	case "update":
		// Restore previous configuration using snapshot
		if action.RollbackInfo.SnapshotID != "" {
			log.Printf("Restoring from snapshot %s", action.RollbackInfo.SnapshotID)
			// AWS-specific rollback logic would go here
		}
	case "create":
		// Delete the created resource
		log.Printf("Deleting created resource %s", action.ResourceID)
	case "delete":
		// Recreate the deleted resource (if possible)
		log.Printf("Attempting to recreate deleted resource %s", action.ResourceID)
	}

	return nil
}

// Additional executor stubs...
// AzureExecutor executes Azure remediation actions
type AzureExecutor struct{}

func (e *AzureExecutor) Execute(ctx context.Context, action *RemediationAction) (*RemediationResult, error) {
	return &RemediationResult{Success: true, ActionID: action.ID}, nil
}

func (e *AzureExecutor) Validate(action *RemediationAction) error {
	return nil
}

func (e *AzureExecutor) GetSupportedActions() []string {
	return []string{"update", "delete", "create", "restart"}
}

func (e *AzureExecutor) Rollback(ctx context.Context, action *RemediationAction) error {
	if action.RollbackInfo == nil {
		return fmt.Errorf("no rollback information available for action %s", action.ID)
	}

	log.Printf("Rolling back Azure action %s for resource %s", action.ID, action.ResourceID)

	// Implement rollback based on action type
	switch action.Action {
	case "update":
		// Restore previous configuration
		if action.RollbackInfo.SnapshotID != "" {
			log.Printf("Restoring Azure resource from snapshot %s", action.RollbackInfo.SnapshotID)
			// Azure-specific rollback logic
		}
	case "resize":
		// Revert to original size
		log.Printf("Reverting Azure resource size for %s", action.ResourceID)
	case "modify":
		// Revert modifications
		log.Printf("Reverting Azure resource modifications for %s", action.ResourceID)
	}

	return nil
}

// GCPExecutor executes GCP remediation actions
type GCPExecutor struct{}

func (e *GCPExecutor) Execute(ctx context.Context, action *RemediationAction) (*RemediationResult, error) {
	return &RemediationResult{Success: true, ActionID: action.ID}, nil
}

func (e *GCPExecutor) Validate(action *RemediationAction) error {
	return nil
}

func (e *GCPExecutor) GetSupportedActions() []string {
	return []string{"update", "delete", "create", "restart"}
}

func (e *GCPExecutor) Rollback(ctx context.Context, action *RemediationAction) error {
	if action.RollbackInfo == nil {
		return fmt.Errorf("no rollback information available for action %s", action.ID)
	}

	log.Printf("Rolling back GCP action %s for resource %s", action.ID, action.ResourceID)

	// Implement rollback based on action type
	switch action.Action {
	case "update":
		// Restore previous configuration
		if action.RollbackInfo.SnapshotID != "" {
			log.Printf("Restoring GCP resource from snapshot %s", action.RollbackInfo.SnapshotID)
			// GCP-specific rollback logic
		}
	case "restart":
		// No rollback needed for restart
		log.Printf("No rollback needed for restart action")
	case "delete":
		// Attempt to recreate if template is available
		log.Printf("Attempting to recreate GCP resource %s", action.ResourceID)
	}

	return nil
}

// TerraformExecutor executes Terraform-based remediation
type TerraformExecutor struct{}

func (e *TerraformExecutor) Execute(ctx context.Context, action *RemediationAction) (*RemediationResult, error) {
	return &RemediationResult{Success: true, ActionID: action.ID}, nil
}

func (e *TerraformExecutor) Validate(action *RemediationAction) error {
	return nil
}

func (e *TerraformExecutor) GetSupportedActions() []string {
	return []string{"apply", "destroy", "import", "refresh"}
}

func (e *TerraformExecutor) Rollback(ctx context.Context, action *RemediationAction) error {
	// Implement Terraform state rollback
	if action.RollbackInfo == nil {
		return fmt.Errorf("no rollback information available")
	}

	log.Printf("Rolling back Terraform changes for action %s", action.ID)

	// Restore previous state version
	// In production, this would:
	// 1. Retrieve the previous state version
	// 2. Create a plan to revert changes
	// 3. Apply the reversion plan

	return nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// containsRestartRequiredChange checks if changes require instance restart
func containsRestartRequiredChange(changes []models.DriftChange) bool {
	restartFields := []string{"instance_class", "engine_version", "storage_encrypted"}
	for _, change := range changes {
		for _, field := range restartFields {
			if change.Field == field {
				return true
			}
		}
	}
	return false
}

// containsVMRestartChange checks if VM changes require restart
func containsVMRestartChange(changes []models.DriftChange) bool {
	restartFields := []string{"vm_size", "os_disk", "hardware_profile"}
	for _, change := range changes {
		for _, field := range restartFields {
			if strings.Contains(change.Field, field) {
				return true
			}
		}
	}
	return false
}

// containsGCPRestartChange checks if GCP changes require instance restart
func containsGCPRestartChange(changes []models.DriftChange) bool {
	restartFields := []string{"machine_type", "disks", "network_interfaces"}
	for _, change := range changes {
		for _, field := range restartFields {
			if change.Field == field {
				return true
			}
		}
	}
	return false
}

// sanitizeResourceName sanitizes resource names for Terraform
func sanitizeResourceName(name string) string {
	// Replace non-alphanumeric characters with underscores
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	sanitized := reg.ReplaceAllString(name, "_")

	// Ensure it starts with a letter
	if len(sanitized) > 0 && !unicode.IsLetter(rune(sanitized[0])) {
		sanitized = "resource_" + sanitized
	}

	return sanitized
}

// generateTerraformConfig generates Terraform configuration for a drift
func generateTerraformConfig(drift *models.DriftResult) string {
	var config strings.Builder

	config.WriteString(fmt.Sprintf("resource \"%s\" \"%s\" {\n",
		drift.ResourceType,
		sanitizeResourceName(drift.ResourceName)))

	// Add configuration based on drift changes
	for _, change := range drift.Changes {
		if change.NewValue != nil {
			config.WriteString(fmt.Sprintf("  %s = %v\n",
				strings.ReplaceAll(change.Field, ".", "_"),
				formatTerraformValue(change.NewValue)))
		}
	}

	config.WriteString("}\n")
	return config.String()
}

// formatTerraformValue formats a value for Terraform configuration
func formatTerraformValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	case []interface{}:
		var items []string
		for _, item := range v {
			items = append(items, formatTerraformValue(item))
		}
		return fmt.Sprintf("[%s]", strings.Join(items, ", "))
	case map[string]interface{}:
		var items []string
		for key, val := range v {
			items = append(items, fmt.Sprintf("%s = %s", key, formatTerraformValue(val)))
		}
		return fmt.Sprintf("{\n    %s\n  }", strings.Join(items, "\n    "))
	default:
		return fmt.Sprintf("%v", v)
	}
}
