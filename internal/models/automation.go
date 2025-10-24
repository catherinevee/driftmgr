package models

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// WorkflowTrigger represents the trigger for a workflow
type WorkflowTrigger string

const (
	WorkflowTriggerManual    WorkflowTrigger = "manual"
	WorkflowTriggerScheduled WorkflowTrigger = "scheduled"
	WorkflowTriggerEvent     WorkflowTrigger = "event"
	WorkflowTriggerCondition WorkflowTrigger = "condition"
	WorkflowTriggerWebhook   WorkflowTrigger = "webhook"
)

// String returns the string representation of WorkflowTrigger
func (wt WorkflowTrigger) String() string {
	return string(wt)
}

// WorkflowStatus represents the status of a workflow
type WorkflowStatus string

const (
	WorkflowStatusDraft   WorkflowStatus = "draft"
	WorkflowStatusActive  WorkflowStatus = "active"
	WorkflowStatusPaused  WorkflowStatus = "paused"
	WorkflowStatusStopped WorkflowStatus = "stopped"
	WorkflowStatusError   WorkflowStatus = "error"
)

// String returns the string representation of WorkflowStatus
func (ws WorkflowStatus) String() string {
	return string(ws)
}

// WorkflowExecutionStatus represents the status of a workflow execution
type WorkflowExecutionStatus string

const (
	WorkflowExecutionStatusPending   WorkflowExecutionStatus = "pending"
	WorkflowExecutionStatusRunning   WorkflowExecutionStatus = "running"
	WorkflowExecutionStatusCompleted WorkflowExecutionStatus = "completed"
	WorkflowExecutionStatusFailed    WorkflowExecutionStatus = "failed"
	WorkflowExecutionStatusCancelled WorkflowExecutionStatus = "cancelled"
	WorkflowExecutionStatusSkipped   WorkflowExecutionStatus = "skipped"
)

// String returns the string representation of WorkflowExecutionStatus
func (wes WorkflowExecutionStatus) String() string {
	return string(wes)
}

// AutomationWorkflow represents an automation workflow
type AutomationWorkflow struct {
	ID             string                 `json:"id" db:"id" validate:"required,uuid"`
	Name           string                 `json:"name" db:"name" validate:"required"`
	Description    string                 `json:"description" db:"description"`
	Trigger        WorkflowTrigger        `json:"trigger" db:"trigger" validate:"required"`
	Steps          []WorkflowStep         `json:"steps" db:"steps" validate:"required"`
	Actions        []WorkflowAction       `json:"actions" db:"actions"`
	Conditions     []WorkflowCondition    `json:"conditions" db:"conditions"`
	Settings       map[string]interface{} `json:"settings" db:"settings"`
	Schedule       *WorkflowSchedule      `json:"schedule" db:"schedule"`
	EventConfig    *WorkflowEventConfig   `json:"event_config" db:"event_config"`
	WebhookConfig  *WorkflowWebhookConfig `json:"webhook_config" db:"webhook_config"`
	Status         WorkflowStatus         `json:"status" db:"status" validate:"required"`
	IsEnabled      bool                   `json:"is_enabled" db:"is_enabled"`
	LastExecuted   *time.Time             `json:"last_executed" db:"last_executed"`
	NextExecution  *time.Time             `json:"next_execution" db:"next_execution"`
	ExecutionCount int                    `json:"execution_count" db:"execution_count"`
	SuccessCount   int                    `json:"success_count" db:"success_count"`
	FailureCount   int                    `json:"failure_count" db:"failure_count"`
	Tags           map[string]string      `json:"tags" db:"tags"`
	UserID         string                 `json:"user_id" db:"user_id"`
	CreatedBy      string                 `json:"created_by" db:"created_by" validate:"required"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// WorkflowStep represents a step in a workflow
type WorkflowStep struct {
	ID          string              `json:"id" db:"id" validate:"required,uuid"`
	Name        string              `json:"name" db:"name" validate:"required"`
	Type        WorkflowStepType    `json:"type" db:"type" validate:"required"`
	Action      WorkflowAction      `json:"action" db:"action" validate:"required"`
	Conditions  []WorkflowCondition `json:"conditions" db:"conditions"`
	OnSuccess   *WorkflowStepConfig `json:"on_success" db:"on_success"`
	OnFailure   *WorkflowStepConfig `json:"on_failure" db:"on_failure"`
	RetryConfig *RetryConfig        `json:"retry_config" db:"retry_config"`
	Timeout     time.Duration       `json:"timeout" db:"timeout"`
	Parallel    bool                `json:"parallel" db:"parallel"`
	Order       int                 `json:"order" db:"order"`
}

// WorkflowStepType represents the type of a workflow step
type WorkflowStepType string

const (
	WorkflowStepTypeAction       WorkflowStepType = "action"
	WorkflowStepTypeCondition    WorkflowStepType = "condition"
	WorkflowStepTypeLoop         WorkflowStepType = "loop"
	WorkflowStepTypeParallel     WorkflowStepType = "parallel"
	WorkflowStepTypeWait         WorkflowStepType = "wait"
	WorkflowStepTypeNotification WorkflowStepType = "notification"
)

// String returns the string representation of WorkflowStepType
func (wst WorkflowStepType) String() string {
	return string(wst)
}

// WorkflowAction represents an action in a workflow
type WorkflowAction struct {
	Type         ActionType             `json:"type" db:"type" validate:"required"`
	Name         string                 `json:"name" db:"name" validate:"required"`
	Description  string                 `json:"description" db:"description"`
	Parameters   map[string]interface{} `json:"parameters" db:"parameters"`
	Inputs       map[string]interface{} `json:"inputs" db:"inputs"`
	Outputs      map[string]interface{} `json:"outputs" db:"outputs"`
	ResourceType string                 `json:"resource_type" db:"resource_type"`
	ResourceID   string                 `json:"resource_id" db:"resource_id"`
	Provider     CloudProvider          `json:"provider" db:"provider"`
	Region       string                 `json:"region" db:"region"`
}

// ActionType represents the type of an action
type ActionType string

const (
	ActionTypeTerraform    ActionType = "terraform"
	ActionTypeAPI          ActionType = "api"
	ActionTypeScript       ActionType = "script"
	ActionTypeNotification ActionType = "notification"
	ActionTypeWebhook      ActionType = "webhook"
	ActionTypeDatabase     ActionType = "database"
	ActionTypeFile         ActionType = "file"
	ActionTypeCustom       ActionType = "custom"
)

// String returns the string representation of ActionType
func (at ActionType) String() string {
	return string(at)
}

// WorkflowCondition represents a condition in a workflow
type WorkflowCondition struct {
	ID          string                 `json:"id" db:"id" validate:"required,uuid"`
	Name        string                 `json:"name" db:"name" validate:"required"`
	Type        ConditionType          `json:"type" db:"type" validate:"required"`
	Expression  string                 `json:"expression" db:"expression" validate:"required"`
	Parameters  map[string]interface{} `json:"parameters" db:"parameters"`
	Negate      bool                   `json:"negate" db:"negate"`
	Description string                 `json:"description" db:"description"`
}

// ConditionType represents the type of a condition
type ConditionType string

const (
	ConditionTypeExpression ConditionType = "expression"
	ConditionTypeResource   ConditionType = "resource"
	ConditionTypeTime       ConditionType = "time"
	ConditionTypeEvent      ConditionType = "event"
	ConditionTypeCustom     ConditionType = "custom"
)

// String returns the string representation of ConditionType
func (ct ConditionType) String() string {
	return string(ct)
}

// WorkflowStepConfig represents configuration for workflow step execution
type WorkflowStepConfig struct {
	NextStep         string                 `json:"next_step" db:"next_step"`
	SkipSteps        []string               `json:"skip_steps" db:"skip_steps"`
	EndWorkflow      bool                   `json:"end_workflow" db:"end_workflow"`
	SetVariables     map[string]interface{} `json:"set_variables" db:"set_variables"`
	SendNotification *NotificationConfig    `json:"send_notification" db:"send_notification"`
}

// RetryConfig represents retry configuration for workflow steps
type RetryConfig struct {
	MaxRetries        int           `json:"max_retries" db:"max_retries"`
	RetryDelay        time.Duration `json:"retry_delay" db:"retry_delay"`
	BackoffMultiplier float64       `json:"backoff_multiplier" db:"backoff_multiplier"`
	MaxRetryDelay     time.Duration `json:"max_retry_delay" db:"max_retry_delay"`
	RetryOn           []string      `json:"retry_on" db:"retry_on"`
}

// WorkflowSchedule represents the schedule for a workflow
type WorkflowSchedule struct {
	Frequency  ScheduleFrequency `json:"frequency" db:"frequency" validate:"required"`
	DayOfWeek  *int              `json:"day_of_week" db:"day_of_week"`
	DayOfMonth *int              `json:"day_of_month" db:"day_of_month"`
	Hour       int               `json:"hour" db:"hour"`
	Minute     int               `json:"minute" db:"minute"`
	Timezone   string            `json:"timezone" db:"timezone"`
	StartDate  *time.Time        `json:"start_date" db:"start_date"`
	EndDate    *time.Time        `json:"end_date" db:"end_date"`
	Enabled    bool              `json:"enabled" db:"enabled"`
}

// WorkflowEventConfig represents event configuration for a workflow
type WorkflowEventConfig struct {
	EventTypes    []string            `json:"event_types" db:"event_types"`
	Filters       []WorkflowCondition `json:"filters" db:"filters"`
	Source        string              `json:"source" db:"source"`
	ResourceTypes []string            `json:"resource_types" db:"resource_types"`
	Providers     []CloudProvider     `json:"providers" db:"providers"`
	Regions       []string            `json:"regions" db:"regions"`
}

// WorkflowWebhookConfig represents webhook configuration for a workflow
type WorkflowWebhookConfig struct {
	URL            string             `json:"url" db:"url" validate:"required"`
	Method         string             `json:"method" db:"method"`
	Headers        map[string]string  `json:"headers" db:"headers"`
	Authentication *WebhookAuth       `json:"authentication" db:"authentication"`
	Validation     *WebhookValidation `json:"validation" db:"validation"`
}

// WebhookAuth represents webhook authentication
type WebhookAuth struct {
	Type     string                 `json:"type" db:"type"`
	Token    string                 `json:"token" db:"token"`
	Username string                 `json:"username" db:"username"`
	Password string                 `json:"password" db:"password"`
	APIKey   string                 `json:"api_key" db:"api_key"`
	Custom   map[string]interface{} `json:"custom" db:"custom"`
}

// WebhookValidation represents webhook validation
type WebhookValidation struct {
	Secret          string   `json:"secret" db:"secret"`
	SignatureHeader string   `json:"signature_header" db:"signature_header"`
	Algorithm       string   `json:"algorithm" db:"algorithm"`
	RequiredFields  []string `json:"required_fields" db:"required_fields"`
}

// NotificationConfig represents notification configuration
type NotificationConfig struct {
	Channels   []NotificationChannel `json:"channels" db:"channels"`
	Template   string                `json:"template" db:"template"`
	Subject    string                `json:"subject" db:"subject"`
	Message    string                `json:"message" db:"message"`
	Priority   NotificationPriority  `json:"priority" db:"priority"`
	Recipients []string              `json:"recipients" db:"recipients"`
}

// NotificationChannel represents a notification channel
type NotificationChannel string

const (
	NotificationChannelEmail     NotificationChannel = "email"
	NotificationChannelSlack     NotificationChannel = "slack"
	NotificationChannelSMS       NotificationChannel = "sms"
	NotificationChannelWebhook   NotificationChannel = "webhook"
	NotificationChannelDashboard NotificationChannel = "dashboard"
)

// String returns the string representation of NotificationChannel
func (nc NotificationChannel) String() string {
	return string(nc)
}

// NotificationPriority represents notification priority
type NotificationPriority string

const (
	NotificationPriorityLow      NotificationPriority = "low"
	NotificationPriorityMedium   NotificationPriority = "medium"
	NotificationPriorityHigh     NotificationPriority = "high"
	NotificationPriorityCritical NotificationPriority = "critical"
)

// String returns the string representation of NotificationPriority
func (np NotificationPriority) String() string {
	return string(np)
}

// WorkflowExecution represents the execution of a workflow
type WorkflowExecution struct {
	ID          string                  `json:"id" db:"id" validate:"required,uuid"`
	WorkflowID  string                  `json:"workflow_id" db:"workflow_id" validate:"required,uuid"`
	Status      WorkflowExecutionStatus `json:"status" db:"status" validate:"required"`
	Trigger     WorkflowTrigger         `json:"trigger" db:"trigger"`
	TriggerData map[string]interface{}  `json:"trigger_data" db:"trigger_data"`
	Inputs      map[string]interface{}  `json:"inputs" db:"inputs"`
	Outputs     map[string]interface{}  `json:"outputs" db:"outputs"`
	Variables   map[string]interface{}  `json:"variables" db:"variables"`
	Steps       []WorkflowStepExecution `json:"steps" db:"steps"`
	StartedAt   time.Time               `json:"started_at" db:"started_at"`
	CompletedAt *time.Time              `json:"completed_at" db:"completed_at"`
	Duration    time.Duration           `json:"duration" db:"duration"`
	Error       *string                 `json:"error,omitempty" db:"error"`
	CreatedBy   string                  `json:"created_by" db:"created_by"`
	CreatedAt   time.Time               `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at" db:"updated_at"`
}

// WorkflowStepExecution represents the execution of a workflow step
type WorkflowStepExecution struct {
	ID          string                  `json:"id" db:"id" validate:"required,uuid"`
	StepID      string                  `json:"step_id" db:"step_id" validate:"required,uuid"`
	Status      WorkflowExecutionStatus `json:"status" db:"status" validate:"required"`
	Inputs      map[string]interface{}  `json:"inputs" db:"inputs"`
	Outputs     map[string]interface{}  `json:"outputs" db:"outputs"`
	StartedAt   time.Time               `json:"started_at" db:"started_at"`
	CompletedAt *time.Time              `json:"completed_at" db:"completed_at"`
	Duration    time.Duration           `json:"duration" db:"duration"`
	RetryCount  int                     `json:"retry_count" db:"retry_count"`
	Error       *string                 `json:"error,omitempty" db:"error"`
	Logs        []string                `json:"logs" db:"logs"`
}

// Request/Response Models

// AutomationWorkflowCreateRequest represents a request to create an automation workflow
type AutomationWorkflowCreateRequest struct {
	Name          string                 `json:"name" validate:"required"`
	Description   string                 `json:"description"`
	Trigger       WorkflowTrigger        `json:"trigger" validate:"required"`
	Steps         []WorkflowStep         `json:"steps" validate:"required"`
	Conditions    []WorkflowCondition    `json:"conditions"`
	Schedule      *WorkflowSchedule      `json:"schedule"`
	EventConfig   *WorkflowEventConfig   `json:"event_config"`
	WebhookConfig *WorkflowWebhookConfig `json:"webhook_config"`
	IsEnabled     bool                   `json:"is_enabled"`
	Tags          map[string]string      `json:"tags"`
}

// AutomationWorkflowListRequest represents a request to list automation workflows
type AutomationWorkflowListRequest struct {
	Trigger   *WorkflowTrigger `json:"trigger,omitempty"`
	Status    *WorkflowStatus  `json:"status,omitempty"`
	IsEnabled *bool            `json:"is_enabled,omitempty"`
	CreatedBy *string          `json:"created_by,omitempty"`
	Limit     int              `json:"limit" validate:"min=1,max=1000"`
	Offset    int              `json:"offset" validate:"min=0"`
	SortBy    string           `json:"sort_by" validate:"omitempty,oneof=name created_at updated_at last_executed"`
	SortOrder string           `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// AutomationWorkflowListResponse represents the response for listing automation workflows
type AutomationWorkflowListResponse struct {
	Workflows []AutomationWorkflow `json:"workflows"`
	Total     int                  `json:"total"`
	Limit     int                  `json:"limit"`
	Offset    int                  `json:"offset"`
}

// AutomationWorkflowExecuteRequest represents a request to execute an automation workflow
type AutomationWorkflowExecuteRequest struct {
	Inputs      map[string]interface{} `json:"inputs"`
	Variables   map[string]interface{} `json:"variables"`
	Async       bool                   `json:"async"`
	CallbackURL string                 `json:"callback_url"`
}

// WorkflowExecutionListRequest represents a request to list workflow executions
type WorkflowExecutionListRequest struct {
	WorkflowID *string                  `json:"workflow_id,omitempty"`
	Status     *WorkflowExecutionStatus `json:"status,omitempty"`
	Trigger    *WorkflowTrigger         `json:"trigger,omitempty"`
	StartTime  *time.Time               `json:"start_time,omitempty"`
	EndTime    *time.Time               `json:"end_time,omitempty"`
	Limit      int                      `json:"limit" validate:"min=1,max=1000"`
	Offset     int                      `json:"offset" validate:"min=0"`
	SortBy     string                   `json:"sort_by" validate:"omitempty,oneof=started_at completed_at duration"`
	SortOrder  string                   `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// WorkflowExecutionListResponse represents the response for listing workflow executions
type WorkflowExecutionListResponse struct {
	Executions []WorkflowExecution `json:"executions"`
	Total      int                 `json:"total"`
	Limit      int                 `json:"limit"`
	Offset     int                 `json:"offset"`
}

// Validation methods

// Validate validates the AutomationWorkflow struct
func (aw *AutomationWorkflow) Validate() error {
	validate := validator.New()
	return validate.Struct(aw)
}

// Validate validates the WorkflowStep struct
func (ws *WorkflowStep) Validate() error {
	validate := validator.New()
	return validate.Struct(ws)
}

// Validate validates the WorkflowAction struct
func (wa *WorkflowAction) Validate() error {
	validate := validator.New()
	return validate.Struct(wa)
}

// Validate validates the WorkflowCondition struct
func (wc *WorkflowCondition) Validate() error {
	validate := validator.New()
	return validate.Struct(wc)
}

// Validate validates the WorkflowExecution struct
func (we *WorkflowExecution) Validate() error {
	validate := validator.New()
	return validate.Struct(we)
}

// Validate validates the WorkflowStepExecution struct
func (wse *WorkflowStepExecution) Validate() error {
	validate := validator.New()
	return validate.Struct(wse)
}

// Validate validates the AutomationWorkflowCreateRequest struct
func (awcr *AutomationWorkflowCreateRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(awcr)
}

// Validate validates the AutomationWorkflowListRequest struct
func (awlr *AutomationWorkflowListRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(awlr)
}

// Validate validates the AutomationWorkflowExecuteRequest struct
func (awer *AutomationWorkflowExecuteRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(awer)
}

// Validate validates the WorkflowExecutionListRequest struct
func (welr *WorkflowExecutionListRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(welr)
}

// Helper methods

// IsActive returns true if the workflow is active
func (aw *AutomationWorkflow) IsActive() bool {
	return aw.Status == WorkflowStatusActive && aw.IsEnabled
}

// IsScheduled returns true if the workflow is scheduled
func (aw *AutomationWorkflow) IsScheduled() bool {
	return aw.Trigger == WorkflowTriggerScheduled && aw.Schedule != nil
}

// IsEventDriven returns true if the workflow is event-driven
func (aw *AutomationWorkflow) IsEventDriven() bool {
	return aw.Trigger == WorkflowTriggerEvent && aw.EventConfig != nil
}

// IsWebhookTriggered returns true if the workflow is webhook-triggered
func (aw *AutomationWorkflow) IsWebhookTriggered() bool {
	return aw.Trigger == WorkflowTriggerWebhook && aw.WebhookConfig != nil
}

// GetSuccessRate returns the success rate of the workflow
func (aw *AutomationWorkflow) GetSuccessRate() float64 {
	if aw.ExecutionCount == 0 {
		return 0.0
	}
	return float64(aw.SuccessCount) / float64(aw.ExecutionCount) * 100
}

// GetFailureRate returns the failure rate of the workflow
func (aw *AutomationWorkflow) GetFailureRate() float64 {
	if aw.ExecutionCount == 0 {
		return 0.0
	}
	return float64(aw.FailureCount) / float64(aw.ExecutionCount) * 100
}

// IsCompleted returns true if the execution is completed
func (we *WorkflowExecution) IsCompleted() bool {
	return we.Status == WorkflowExecutionStatusCompleted
}

// IsFailed returns true if the execution failed
func (we *WorkflowExecution) IsFailed() bool {
	return we.Status == WorkflowExecutionStatusFailed
}

// IsRunning returns true if the execution is running
func (we *WorkflowExecution) IsRunning() bool {
	return we.Status == WorkflowExecutionStatusRunning
}

// IsCancelled returns true if the execution was cancelled
func (we *WorkflowExecution) IsCancelled() bool {
	return we.Status == WorkflowExecutionStatusCancelled
}

// GetDuration returns the execution duration
func (we *WorkflowExecution) GetDuration() time.Duration {
	if we.CompletedAt != nil {
		return we.CompletedAt.Sub(we.StartedAt)
	}
	return time.Since(we.StartedAt)
}

// HasError returns true if the execution has an error
func (we *WorkflowExecution) HasError() bool {
	return we.Error != nil
}

// GetError returns the error message
func (we *WorkflowExecution) GetError() string {
	if we.Error == nil {
		return ""
	}
	return *we.Error
}

// SetError sets the error message
func (we *WorkflowExecution) SetError(err error) {
	if err != nil {
		errStr := err.Error()
		we.Error = &errStr
		we.Status = WorkflowExecutionStatusFailed
		now := time.Now()
		we.CompletedAt = &now
		we.Duration = we.GetDuration()
	}
}

// IsCompleted returns true if the step execution is completed
func (wse *WorkflowStepExecution) IsCompleted() bool {
	return wse.Status == WorkflowExecutionStatusCompleted
}

// IsFailed returns true if the step execution failed
func (wse *WorkflowStepExecution) IsFailed() bool {
	return wse.Status == WorkflowExecutionStatusFailed
}

// IsRunning returns true if the step execution is running
func (wse *WorkflowStepExecution) IsRunning() bool {
	return wse.Status == WorkflowExecutionStatusRunning
}

// GetDuration returns the step execution duration
func (wse *WorkflowStepExecution) GetDuration() time.Duration {
	if wse.CompletedAt != nil {
		return wse.CompletedAt.Sub(wse.StartedAt)
	}
	return time.Since(wse.StartedAt)
}

// HasError returns true if the step execution has an error
func (wse *WorkflowStepExecution) HasError() bool {
	return wse.Error != nil
}

// GetError returns the error message
func (wse *WorkflowStepExecution) GetError() string {
	if wse.Error == nil {
		return ""
	}
	return *wse.Error
}

// SetError sets the error message
func (wse *WorkflowStepExecution) SetError(err error) {
	if err != nil {
		errStr := err.Error()
		wse.Error = &errStr
		wse.Status = WorkflowExecutionStatusFailed
		now := time.Now()
		wse.CompletedAt = &now
		wse.Duration = wse.GetDuration()
	}
}

// JSONB represents a JSON binary type for database storage
type JSONB map[string]interface{}

// AutomationAction represents an automation action
type AutomationAction struct {
	ID            string                 `json:"id" db:"id"`
	Type          ActionType             `json:"type" db:"type" validate:"required"`
	Name          string                 `json:"name" db:"name" validate:"required"`
	Description   string                 `json:"description" db:"description"`
	Parameters    map[string]interface{} `json:"parameters" db:"parameters"`
	Configuration JSONB                  `json:"configuration" db:"configuration"`
	Inputs        map[string]interface{} `json:"inputs" db:"inputs"`
	Outputs       map[string]interface{} `json:"outputs" db:"outputs"`
	ResourceType  string                 `json:"resource_type" db:"resource_type"`
	ResourceID    string                 `json:"resource_id" db:"resource_id"`
	Provider      CloudProvider          `json:"provider" db:"provider"`
	Region        string                 `json:"region" db:"region"`
}

// ActionStatus represents the status of an action execution
type ActionStatus string

const (
	ActionStatusPending   ActionStatus = "pending"
	ActionStatusRunning   ActionStatus = "running"
	ActionStatusCompleted ActionStatus = "completed"
	ActionStatusFailed    ActionStatus = "failed"
	ActionStatusCancelled ActionStatus = "cancelled"
)

// String returns the string representation of ActionStatus
func (as ActionStatus) String() string {
	return string(as)
}

// ActionResult represents the result of an action execution
type ActionResult struct {
	ActionID      string                 `json:"action_id" db:"action_id"`
	Status        ActionStatus           `json:"status" db:"status"`
	ExecutionTime time.Duration          `json:"execution_time" db:"execution_time"`
	Timestamp     time.Time              `json:"timestamp" db:"timestamp"`
	Output        JSONB                  `json:"output" db:"output"`
	Error         string                 `json:"error,omitempty" db:"error"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

// AutomationJob represents a job in the automation system
type AutomationJob struct {
	ID          string                 `json:"id" db:"id" validate:"required,uuid"`
	Name        string                 `json:"name" db:"name" validate:"required"`
	Type        string                 `json:"type" db:"type" validate:"required"`
	Status      string                 `json:"status" db:"status" validate:"required"`
	Priority    int                    `json:"priority" db:"priority"`
	Data        map[string]interface{} `json:"data" db:"data"`
	Result      map[string]interface{} `json:"result" db:"result"`
	Error       string                 `json:"error,omitempty" db:"error"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
}

// AutomationEvent represents an event in the automation system
type AutomationEvent struct {
	ID        string                 `json:"id" db:"id" validate:"required,uuid"`
	Type      string                 `json:"type" db:"type" validate:"required"`
	Source    string                 `json:"source" db:"source"`
	Data      map[string]interface{} `json:"data" db:"data"`
	Timestamp time.Time              `json:"timestamp" db:"timestamp"`
}

// TriggerType represents the type of trigger
type TriggerType string

const (
	TriggerTypeManual    TriggerType = "manual"
	TriggerTypeScheduled TriggerType = "scheduled"
	TriggerTypeEvent     TriggerType = "event"
	TriggerTypeWebhook   TriggerType = "webhook"
)

// String returns the string representation of TriggerType
func (tt TriggerType) String() string {
	return string(tt)
}

// AutomationTrigger represents a trigger for automation
type AutomationTrigger struct {
	ID        string                 `json:"id" db:"id" validate:"required,uuid"`
	Name      string                 `json:"name" db:"name" validate:"required"`
	Type      TriggerType            `json:"type" db:"type" validate:"required"`
	Config    map[string]interface{} `json:"config" db:"config"`
	IsEnabled bool                   `json:"is_enabled" db:"is_enabled"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" db:"updated_at"`
}

// AutomationWorkflowRequest represents a request for automation workflow operations
type AutomationWorkflowRequest struct {
	WorkflowID  string                 `json:"workflow_id" validate:"required,uuid"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Trigger     WorkflowTrigger        `json:"trigger"`
	Steps       []WorkflowStep         `json:"steps"`
	Actions     []WorkflowAction       `json:"actions"`
	Conditions  []WorkflowCondition    `json:"conditions"`
	Settings    map[string]interface{} `json:"settings"`
	Inputs      map[string]interface{} `json:"inputs"`
	Variables   map[string]interface{} `json:"variables"`
	Async       bool                   `json:"async"`
}
