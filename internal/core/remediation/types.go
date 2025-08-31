package remediation

import (
	"time"
)

// Plan represents a remediation plan
type Plan struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	DriftItems  []interface{}          `json:"drift_items"`
	Actions     []Action               `json:"actions"`
	Impact      Impact                 `json:"impact"`
	Approval    *ApprovalStatus        `json:"approval,omitempty"`
	Execution   *ExecutionStatus       `json:"execution,omitempty"`
	Results     *Results               `json:"results,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Action represents a remediation action
type Action struct {
	ID               string                 `json:"id"`
	Type             string                 `json:"type"`
	ActionType       string                 `json:"action_type"`
	ResourceID       string                 `json:"resource_id"`
	ResourceType     string                 `json:"resource_type"`
	ResourceAddress  string                 `json:"resource_address,omitempty"`
	Provider         string                 `json:"provider"`
	Region           string                 `json:"region,omitempty"`
	Description      string                 `json:"description"`
	Risk             string                 `json:"risk,omitempty"`
	Status           string                 `json:"status"`
	Error            string                 `json:"error,omitempty"`
	Parameters       map[string]interface{} `json:"parameters,omitempty"`
	TerraformConfig  string                 `json:"terraform_config,omitempty"`
	EstimatedTime    int                    `json:"estimated_time"`
	Dependencies     []string               `json:"dependencies"`
	StartTime        time.Time              `json:"start_time,omitempty"`
	EndTime          time.Time              `json:"end_time,omitempty"`
}

// Results represents remediation results
type Results struct {
	Success      bool                   `json:"success"`
	ItemsFixed   int                    `json:"items_fixed"`
	ItemsFailed  int                    `json:"items_failed"`
	Duration     time.Duration          `json:"duration"`
	Details      map[string]interface{} `json:"details,omitempty"`
	Errors       []string               `json:"errors,omitempty"`
	RollbackInfo *RollbackInfo          `json:"rollback_info,omitempty"`
}

// Impact describes remediation impact
type Impact struct {
	ResourcesAffected int                    `json:"resources_affected"`
	EstimatedDuration int                    `json:"estimated_duration"`
	RiskLevel         string                 `json:"risk_level"`
	CostImpact        float64                `json:"cost_impact"`
	ServiceImpact     []string               `json:"service_impact"`
	RequiresDowntime  bool                   `json:"requires_downtime"`
	Reversible        bool                   `json:"reversible"`
	Details           map[string]interface{} `json:"details"`
}

// ApprovalStatus represents approval status
type ApprovalStatus struct {
	Required     bool       `json:"required"`
	Status       string     `json:"status"`
	Approvers    []string   `json:"approvers"`
	ApprovedBy   []string   `json:"approved_by"`
	ApprovalTime *time.Time `json:"approval_time,omitempty"`
	Comments     []string   `json:"comments,omitempty"`
}

// ExecutionStatus represents execution status
type ExecutionStatus struct {
	Status         string     `json:"status"`
	StartTime      time.Time  `json:"start_time"`
	EndTime        *time.Time `json:"end_time,omitempty"`
	Progress       int        `json:"progress"`
	CurrentStep    string     `json:"current_step"`
	TotalSteps     int        `json:"total_steps"`
	CompletedSteps int        `json:"completed_steps"`
}

// RollbackInfo contains rollback information
type RollbackInfo struct {
	SnapshotID string                 `json:"snapshot_id"`
	PlanID     string                 `json:"plan_id"`
	CreatedAt  time.Time              `json:"created_at"`
	Available  bool                   `json:"available"`
	State      map[string]interface{} `json:"state,omitempty"`
}

// BulkOperationRequest represents a bulk operation request
type BulkOperationRequest struct {
	Operation   string                 `json:"operation"`
	ResourceIDs []string               `json:"resource_ids"`
	Provider    string                 `json:"provider"`
	Region      string                 `json:"region,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	DryRun      bool                   `json:"dry_run"`
}

// BulkOperationResult represents a bulk operation result
type BulkOperationResult struct {
	TotalResources   int                          `json:"total_resources"`
	SuccessfulCount  int                          `json:"successful_count"`
	FailedCount      int                          `json:"failed_count"`
	SkippedCount     int                          `json:"skipped_count"`
	ResourceResults  map[string]*ResourceResult  `json:"resource_results"`
	Duration         time.Duration                `json:"duration"`
}

// ResourceResult represents the result of an operation on a single resource
type ResourceResult struct {
	ResourceID string                 `json:"resource_id"`
	Success    bool                   `json:"success"`
	Error      string                 `json:"error,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
}