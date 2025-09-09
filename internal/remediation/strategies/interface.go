package strategies

import (
	"context"
	"time"

	"github.com/catherinevee/driftmgr/internal/drift/detector"
)

// StrategyType defines the type of remediation strategy
type StrategyType string

const (
	// CodeAsTruthStrategy applies Terraform code to fix drift
	CodeAsTruthStrategy StrategyType = "code-as-truth"
	// CloudAsTruthStrategy updates Terraform to match actual state
	CloudAsTruthStrategy StrategyType = "cloud-as-truth"
	// ManualApprovalStrategy requires human approval before remediation
	ManualApprovalStrategy StrategyType = "manual-approval"
	// AutoRollbackStrategy automatically rolls back on drift detection
	AutoRollbackStrategy StrategyType = "auto-rollback"
	// HybridStrategy combines multiple strategies based on rules
	HybridStrategy StrategyType = "hybrid"
)

// RemediationStrategy defines the interface for all remediation strategies
type RemediationStrategy interface {
	// Plan creates a remediation plan based on detected drift
	Plan(ctx context.Context, drift *detector.DriftResult) (*RemediationPlan, error)

	// Execute executes the remediation plan
	Execute(ctx context.Context, plan *RemediationPlan) (*RemediationResult, error)

	// Validate checks if the strategy can handle the given drift
	Validate(drift *detector.DriftResult) error

	// GetType returns the strategy type
	GetType() StrategyType

	// GetDescription returns a human-readable description
	GetDescription() string
}

// RemediationPlan represents a plan to remediate drift
type RemediationPlan struct {
	ID               string                 `json:"id"`
	Strategy         StrategyType           `json:"strategy"`
	CreatedAt        time.Time              `json:"created_at"`
	DriftSummary     *DriftSummary          `json:"drift_summary"`
	Actions          []RemediationAction    `json:"actions"`
	RiskLevel        RiskLevel              `json:"risk_level"`
	EstimatedTime    time.Duration          `json:"estimated_time"`
	RequiresApproval bool                   `json:"requires_approval"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// RemediationAction represents a single remediation action
type RemediationAction struct {
	ID          string                 `json:"id"`
	Type        ActionType             `json:"type"`
	Resource    string                 `json:"resource"`
	Description string                 `json:"description"`
	Command     string                 `json:"command,omitempty"`
	RiskLevel   RiskLevel              `json:"risk_level"`
	Order       int                    `json:"order"`
	DependsOn   []string               `json:"depends_on,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ActionType defines the type of remediation action
type ActionType string

const (
	ActionApply      ActionType = "apply"       // Apply Terraform
	ActionImport     ActionType = "import"      // Import resource
	ActionDelete     ActionType = "delete"      // Delete resource
	ActionUpdate     ActionType = "update"      // Update resource
	ActionRefresh    ActionType = "refresh"     // Refresh state
	ActionMove       ActionType = "move"        // Move resource
	ActionGeneratePR ActionType = "generate_pr" // Generate pull request
	ActionNotify     ActionType = "notify"      // Send notification
	ActionRollback   ActionType = "rollback"    // Rollback to previous state
)

// RiskLevel defines the risk level of an action
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// RemediationResult represents the result of executing a remediation plan
type RemediationResult struct {
	PlanID          string                 `json:"plan_id"`
	Success         bool                   `json:"success"`
	StartedAt       time.Time              `json:"started_at"`
	CompletedAt     time.Time              `json:"completed_at"`
	Duration        time.Duration          `json:"duration"`
	ActionsExecuted []ActionResult         `json:"actions_executed"`
	Errors          []error                `json:"errors,omitempty"`
	RollbackNeeded  bool                   `json:"rollback_needed"`
	Summary         string                 `json:"summary"`
	Artifacts       map[string]interface{} `json:"artifacts,omitempty"`
}

// ActionResult represents the result of a single action
type ActionResult struct {
	ActionID    string        `json:"action_id"`
	ActionType  ActionType    `json:"action_type"`
	Success     bool          `json:"success"`
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt time.Time     `json:"completed_at"`
	Duration    time.Duration `json:"duration"`
	Output      string        `json:"output,omitempty"`
	Error       error         `json:"error,omitempty"`
}

// DriftSummary summarizes the drift to be remediated
type DriftSummary struct {
	TotalResources     int      `json:"total_resources"`
	DriftedResources   int      `json:"drifted_resources"`
	MissingResources   int      `json:"missing_resources"`
	UnmanagedResources int      `json:"unmanaged_resources"`
	CriticalDrifts     int      `json:"critical_drifts"`
	EstimatedImpact    string   `json:"estimated_impact"`
	AffectedServices   []string `json:"affected_services"`
}

// ApprovalRequest represents a request for manual approval
type ApprovalRequest struct {
	ID          string              `json:"id"`
	PlanID      string              `json:"plan_id"`
	RequestedAt time.Time           `json:"requested_at"`
	RequestedBy string              `json:"requested_by"`
	Description string              `json:"description"`
	RiskLevel   RiskLevel           `json:"risk_level"`
	Actions     []RemediationAction `json:"actions"`
	Status      ApprovalStatus      `json:"status"`
	ApprovedBy  string              `json:"approved_by,omitempty"`
	ApprovedAt  *time.Time          `json:"approved_at,omitempty"`
	Comments    string              `json:"comments,omitempty"`
}

// ApprovalStatus represents the status of an approval request
type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
	ApprovalExpired  ApprovalStatus = "expired"
)

// StrategyConfig contains configuration for remediation strategies
type StrategyConfig struct {
	// Common configuration
	DryRun      bool          `json:"dry_run"`
	AutoApprove bool          `json:"auto_approve"`
	MaxParallel int           `json:"max_parallel"`
	Timeout     time.Duration `json:"timeout"`

	// Strategy-specific configuration
	TerraformPath    string `json:"terraform_path,omitempty"`
	WorkingDir       string `json:"working_dir,omitempty"`
	BackupStateFirst bool   `json:"backup_state_first"`

	// Git configuration for cloud-as-truth
	GitRepo   string `json:"git_repo,omitempty"`
	GitBranch string `json:"git_branch,omitempty"`
	GitAuthor string `json:"git_author,omitempty"`
	GitEmail  string `json:"git_email,omitempty"`

	// Notification configuration
	NotifyOnStart    bool     `json:"notify_on_start"`
	NotifyOnComplete bool     `json:"notify_on_complete"`
	NotifyOnError    bool     `json:"notify_on_error"`
	NotifyChannels   []string `json:"notify_channels,omitempty"`

	// Risk thresholds
	MaxRiskLevel       RiskLevel   `json:"max_risk_level"`
	RequireApprovalFor []RiskLevel `json:"require_approval_for,omitempty"`
}

// StrategyFactory creates remediation strategies
type StrategyFactory interface {
	// CreateStrategy creates a strategy of the given type
	CreateStrategy(strategyType StrategyType, config *StrategyConfig) (RemediationStrategy, error)

	// GetAvailableStrategies returns all available strategy types
	GetAvailableStrategies() []StrategyType

	// GetStrategyInfo returns information about a strategy type
	GetStrategyInfo(strategyType StrategyType) (StrategyInfo, error)
}

// StrategyInfo provides information about a remediation strategy
type StrategyInfo struct {
	Type         StrategyType `json:"type"`
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	RiskLevel    RiskLevel    `json:"risk_level"`
	Requirements []string     `json:"requirements"`
	Limitations  []string     `json:"limitations"`
	BestFor      []string     `json:"best_for"`
}
