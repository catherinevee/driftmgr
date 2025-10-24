package workflows

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/google/uuid"
)

// Manager manages automation workflows
type Manager struct {
	workflowRepo   WorkflowRepository
	executionRepo  ExecutionRepository
	actionExecutor ActionExecutor
	triggerManager TriggerManager
	eventBus       EventBus
	config         ManagerConfig
}

// WorkflowRepository defines the interface for workflow persistence
type WorkflowRepository interface {
	CreateWorkflow(ctx context.Context, workflow *models.AutomationWorkflow) error
	GetWorkflow(ctx context.Context, id uuid.UUID) (*models.AutomationWorkflow, error)
	UpdateWorkflow(ctx context.Context, workflow *models.AutomationWorkflow) error
	DeleteWorkflow(ctx context.Context, id uuid.UUID) error
	ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]*models.AutomationWorkflow, error)
	GetWorkflowStats(ctx context.Context, id uuid.UUID) (*WorkflowStats, error)
}

// ExecutionRepository defines the interface for execution persistence
type ExecutionRepository interface {
	CreateExecution(ctx context.Context, execution *models.AutomationJob) error
	UpdateExecution(ctx context.Context, execution *models.AutomationJob) error
	GetExecution(ctx context.Context, id uuid.UUID) (*models.AutomationJob, error)
	ListExecutions(ctx context.Context, filter ExecutionFilter) ([]*models.AutomationJob, error)
	GetExecutionHistory(ctx context.Context, workflowID uuid.UUID, limit int) ([]*models.AutomationJob, error)
	GetExecutionStats(ctx context.Context, workflowID uuid.UUID) (*ExecutionStats, error)
}

// ActionExecutor defines the interface for executing automation actions
type ActionExecutor interface {
	ExecuteAction(ctx context.Context, action *models.AutomationAction, context map[string]interface{}) (*models.ActionResult, error)
	ValidateAction(ctx context.Context, action *models.AutomationAction) error
	GetSupportedActionTypes() []models.ActionType
}

// TriggerManager defines the interface for managing workflow triggers
type TriggerManager interface {
	RegisterTrigger(ctx context.Context, workflow *models.AutomationWorkflow) error
	UnregisterTrigger(ctx context.Context, workflowID uuid.UUID) error
	StartTriggerMonitoring(ctx context.Context) error
	StopTriggerMonitoring(ctx context.Context) error
}

// EventBus defines the interface for event communication
type EventBus interface {
	PublishEvent(ctx context.Context, event *models.AutomationEvent) error
	SubscribeToEvents(ctx context.Context, eventType string, handler EventHandler) error
	UnsubscribeFromEvents(ctx context.Context, eventType string) error
}

// EventHandler defines the interface for handling automation events
type EventHandler interface {
	HandleEvent(ctx context.Context, event *models.AutomationEvent) error
}

// ManagerConfig holds configuration for the workflow manager
type ManagerConfig struct {
	MaxWorkflowsPerUser  int           `json:"max_workflows_per_user"`
	MaxExecutionsPerHour int           `json:"max_executions_per_hour"`
	ExecutionTimeout     time.Duration `json:"execution_timeout"`
	RetryAttempts        int           `json:"retry_attempts"`
	RetryDelay           time.Duration `json:"retry_delay"`
	EnableEventLogging   bool          `json:"enable_event_logging"`
	EnableMetrics        bool          `json:"enable_metrics"`
	EnableAuditLogging   bool          `json:"enable_audit_logging"`
}

// WorkflowFilter defines filters for workflow queries
type WorkflowFilter struct {
	UserID      *uuid.UUID             `json:"user_id,omitempty"`
	Status      *models.WorkflowStatus `json:"status,omitempty"`
	TriggerType *models.TriggerType    `json:"trigger_type,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Search      string                 `json:"search,omitempty"`
	Limit       int                    `json:"limit,omitempty"`
	Offset      int                    `json:"offset,omitempty"`
}

// ExecutionFilter defines filters for execution queries
type ExecutionFilter struct {
	WorkflowID *uuid.UUID        `json:"workflow_id,omitempty"`
	UserID     *uuid.UUID        `json:"user_id,omitempty"`
	Status     *models.JobStatus `json:"status,omitempty"`
	StartTime  *time.Time        `json:"start_time,omitempty"`
	EndTime    *time.Time        `json:"end_time,omitempty"`
	Limit      int               `json:"limit,omitempty"`
	Offset     int               `json:"offset,omitempty"`
}

// WorkflowStats represents statistics for a workflow
type WorkflowStats struct {
	WorkflowID           uuid.UUID     `json:"workflow_id"`
	TotalExecutions      int           `json:"total_executions"`
	SuccessfulExecutions int           `json:"successful_executions"`
	FailedExecutions     int           `json:"failed_executions"`
	CancelledExecutions  int           `json:"cancelled_executions"`
	AverageDuration      time.Duration `json:"average_duration"`
	LastExecution        *time.Time    `json:"last_execution"`
	SuccessRate          float64       `json:"success_rate"`
}

// ExecutionStats represents execution statistics
type ExecutionStats struct {
	WorkflowID           uuid.UUID              `json:"workflow_id"`
	TotalExecutions      int                    `json:"total_executions"`
	SuccessfulExecutions int                    `json:"successful_executions"`
	FailedExecutions     int                    `json:"failed_executions"`
	CancelledExecutions  int                    `json:"cancelled_executions"`
	AverageDuration      time.Duration          `json:"average_duration"`
	LastExecution        *time.Time             `json:"last_execution"`
	SuccessRate          float64                `json:"success_rate"`
	ExecutionsByDay      []DailyExecutionCount  `json:"executions_by_day"`
	ExecutionsByHour     []HourlyExecutionCount `json:"executions_by_hour"`
}

// DailyExecutionCount represents execution count for a day
type DailyExecutionCount struct {
	Date   time.Time `json:"date"`
	Count  int       `json:"count"`
	Status string    `json:"status"`
}

// HourlyExecutionCount represents execution count for an hour
type HourlyExecutionCount struct {
	Hour   int    `json:"hour"`
	Count  int    `json:"count"`
	Status string `json:"status"`
}

// NewManager creates a new workflow manager
func NewManager(
	workflowRepo WorkflowRepository,
	executionRepo ExecutionRepository,
	actionExecutor ActionExecutor,
	triggerManager TriggerManager,
	eventBus EventBus,
	config ManagerConfig,
) *Manager {
	return &Manager{
		workflowRepo:   workflowRepo,
		executionRepo:  executionRepo,
		actionExecutor: actionExecutor,
		triggerManager: triggerManager,
		eventBus:       eventBus,
		config:         config,
	}
}

// CreateWorkflow creates a new automation workflow
func (m *Manager) CreateWorkflow(ctx context.Context, userID uuid.UUID, req *models.AutomationWorkflowRequest) (*models.AutomationWorkflow, error) {
	// Check workflow limit
	if err := m.checkWorkflowLimit(ctx, userID); err != nil {
		return nil, fmt.Errorf("workflow limit exceeded: %w", err)
	}

	// Validate the workflow
	if err := m.validateWorkflow(req); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	// Create the workflow
	workflow := &models.AutomationWorkflow{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Trigger:     req.Trigger,
		Actions:     req.Actions,
		Conditions:  req.Conditions,
		Settings:    req.Settings,
		Status:      models.WorkflowStatusDraft,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save the workflow
	if err := m.workflowRepo.CreateWorkflow(ctx, workflow); err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	// Log audit event
	if m.config.EnableAuditLogging {
		m.logAuditEvent(ctx, "workflow_created", userID, workflow.ID, map[string]interface{}{
			"workflow_name": workflow.Name,
			"trigger_type":  workflow.Trigger.Type,
			"action_count":  len(workflow.Actions),
		})
	}

	// Publish workflow created event
	if m.config.EnableEventLogging {
		event := &models.AutomationEvent{
			ID:         uuid.New().String(),
			Type:       models.EventTypeWorkflowCreated,
			UserID:     userID,
			WorkflowID: workflow.ID,
			Message:    fmt.Sprintf("Workflow '%s' created", workflow.Name),
			Data:       models.JSONB(workflow),
			Timestamp:  time.Now(),
		}
		if err := m.eventBus.PublishEvent(ctx, event); err != nil {
			log.Printf("Failed to publish workflow created event: %v", err)
		}
	}

	return workflow, nil
}

// GetWorkflow retrieves a workflow by ID
func (m *Manager) GetWorkflow(ctx context.Context, id uuid.UUID) (*models.AutomationWorkflow, error) {
	workflow, err := m.workflowRepo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}
	return workflow, nil
}

// UpdateWorkflow updates an existing workflow
func (m *Manager) UpdateWorkflow(ctx context.Context, userID uuid.UUID, id uuid.UUID, req *models.AutomationWorkflowRequest) (*models.AutomationWorkflow, error) {
	// Get existing workflow
	workflow, err := m.workflowRepo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// Check ownership
	if workflow.UserID != userID {
		return nil, fmt.Errorf("workflow not found or access denied")
	}

	// Validate the workflow
	if err := m.validateWorkflow(req); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	// Update workflow fields
	workflow.Name = req.Name
	workflow.Description = req.Description
	workflow.Trigger = req.Trigger
	workflow.Actions = req.Actions
	workflow.Conditions = req.Conditions
	workflow.Settings = req.Settings
	workflow.Tags = req.Tags
	workflow.UpdatedAt = time.Now()

	// Save the updated workflow
	if err := m.workflowRepo.UpdateWorkflow(ctx, workflow); err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	// Re-register trigger if workflow is active
	if workflow.Status == models.WorkflowStatusActive {
		if err := m.triggerManager.RegisterTrigger(ctx, workflow); err != nil {
			log.Printf("Failed to re-register trigger for workflow %s: %v", id, err)
		}
	}

	// Log audit event
	if m.config.EnableAuditLogging {
		m.logAuditEvent(ctx, "workflow_updated", userID, workflow.ID, map[string]interface{}{
			"workflow_name": workflow.Name,
			"trigger_type":  workflow.Trigger.Type,
			"action_count":  len(workflow.Actions),
		})
	}

	// Publish workflow updated event
	if m.config.EnableEventLogging {
		event := &models.AutomationEvent{
			ID:         uuid.New().String(),
			Type:       models.EventTypeWorkflowUpdated,
			UserID:     userID,
			WorkflowID: workflow.ID,
			Message:    fmt.Sprintf("Workflow '%s' updated", workflow.Name),
			Data:       models.JSONB(workflow),
			Timestamp:  time.Now(),
		}
		if err := m.eventBus.PublishEvent(ctx, event); err != nil {
			log.Printf("Failed to publish workflow updated event: %v", err)
		}
	}

	return workflow, nil
}

// DeleteWorkflow deletes a workflow
func (m *Manager) DeleteWorkflow(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	// Get workflow to check ownership
	workflow, err := m.workflowRepo.GetWorkflow(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	// Check ownership
	if workflow.UserID != userID {
		return fmt.Errorf("workflow not found or access denied")
	}

	// Unregister trigger if workflow is active
	if workflow.Status == models.WorkflowStatusActive {
		if err := m.triggerManager.UnregisterTrigger(ctx, id); err != nil {
			log.Printf("Failed to unregister trigger for workflow %s: %v", id, err)
		}
	}

	// Delete the workflow
	if err := m.workflowRepo.DeleteWorkflow(ctx, id); err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	// Log audit event
	if m.config.EnableAuditLogging {
		m.logAuditEvent(ctx, "workflow_deleted", userID, workflow.ID, map[string]interface{}{
			"workflow_name": workflow.Name,
		})
	}

	// Publish workflow deleted event
	if m.config.EnableEventLogging {
		event := &models.AutomationEvent{
			ID:         uuid.New().String(),
			Type:       models.EventTypeWorkflowDeleted,
			UserID:     userID,
			WorkflowID: id,
			Message:    fmt.Sprintf("Workflow '%s' deleted", workflow.Name),
			Data:       models.JSONB(workflow),
			Timestamp:  time.Now(),
		}
		if err := m.eventBus.PublishEvent(ctx, event); err != nil {
			log.Printf("Failed to publish workflow deleted event: %v", err)
		}
	}

	return nil
}

// ListWorkflows lists workflows with optional filtering
func (m *Manager) ListWorkflows(ctx context.Context, userID uuid.UUID, filter WorkflowFilter) ([]*models.AutomationWorkflow, error) {
	// Set user ID filter
	filter.UserID = &userID

	workflows, err := m.workflowRepo.ListWorkflows(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	return workflows, nil
}

// ActivateWorkflow activates a workflow
func (m *Manager) ActivateWorkflow(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	workflow, err := m.workflowRepo.GetWorkflow(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	// Check ownership
	if workflow.UserID != userID {
		return fmt.Errorf("workflow not found or access denied")
	}

	if workflow.Status == models.WorkflowStatusActive {
		return fmt.Errorf("workflow is already active")
	}

	// Validate workflow before activation
	if err := m.validateWorkflowForActivation(workflow); err != nil {
		return fmt.Errorf("workflow validation failed: %w", err)
	}

	// Update workflow status
	workflow.Status = models.WorkflowStatusActive
	workflow.UpdatedAt = time.Now()

	if err := m.workflowRepo.UpdateWorkflow(ctx, workflow); err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	// Register trigger
	if err := m.triggerManager.RegisterTrigger(ctx, workflow); err != nil {
		// Rollback status change
		workflow.Status = models.WorkflowStatusDraft
		m.workflowRepo.UpdateWorkflow(ctx, workflow)
		return fmt.Errorf("failed to register trigger: %w", err)
	}

	// Log audit event
	if m.config.EnableAuditLogging {
		m.logAuditEvent(ctx, "workflow_activated", userID, workflow.ID, map[string]interface{}{
			"workflow_name": workflow.Name,
		})
	}

	// Publish workflow activated event
	if m.config.EnableEventLogging {
		event := &models.AutomationEvent{
			ID:         uuid.New().String(),
			Type:       models.EventTypeWorkflowActivated,
			UserID:     userID,
			WorkflowID: workflow.ID,
			Message:    fmt.Sprintf("Workflow '%s' activated", workflow.Name),
			Data:       models.JSONB(workflow),
			Timestamp:  time.Now(),
		}
		if err := m.eventBus.PublishEvent(ctx, event); err != nil {
			log.Printf("Failed to publish workflow activated event: %v", err)
		}
	}

	return nil
}

// DeactivateWorkflow deactivates a workflow
func (m *Manager) DeactivateWorkflow(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	workflow, err := m.workflowRepo.GetWorkflow(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	// Check ownership
	if workflow.UserID != userID {
		return fmt.Errorf("workflow not found or access denied")
	}

	if workflow.Status != models.WorkflowStatusActive {
		return fmt.Errorf("workflow is not active")
	}

	// Unregister trigger
	if err := m.triggerManager.UnregisterTrigger(ctx, id); err != nil {
		log.Printf("Failed to unregister trigger for workflow %s: %v", id, err)
	}

	// Update workflow status
	workflow.Status = models.WorkflowStatusDraft
	workflow.UpdatedAt = time.Now()

	if err := m.workflowRepo.UpdateWorkflow(ctx, workflow); err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	// Log audit event
	if m.config.EnableAuditLogging {
		m.logAuditEvent(ctx, "workflow_deactivated", userID, workflow.ID, map[string]interface{}{
			"workflow_name": workflow.Name,
		})
	}

	// Publish workflow deactivated event
	if m.config.EnableEventLogging {
		event := &models.AutomationEvent{
			ID:         uuid.New().String(),
			Type:       models.EventTypeWorkflowDeactivated,
			UserID:     userID,
			WorkflowID: workflow.ID,
			Message:    fmt.Sprintf("Workflow '%s' deactivated", workflow.Name),
			Data:       models.JSONB(workflow),
			Timestamp:  time.Now(),
		}
		if err := m.eventBus.PublishEvent(ctx, event); err != nil {
			log.Printf("Failed to publish workflow deactivated event: %v", err)
		}
	}

	return nil
}

// ExecuteWorkflow executes a workflow manually
func (m *Manager) ExecuteWorkflow(ctx context.Context, userID uuid.UUID, id uuid.UUID, input map[string]interface{}) (*models.AutomationJob, error) {
	workflow, err := m.workflowRepo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// Check ownership
	if workflow.UserID != userID {
		return nil, fmt.Errorf("workflow not found or access denied")
	}

	// Check if workflow is active
	if workflow.Status != models.WorkflowStatusActive {
		return nil, fmt.Errorf("workflow is not active")
	}

	// Check execution rate limit
	if err := m.checkExecutionRateLimit(ctx, userID); err != nil {
		return nil, fmt.Errorf("execution rate limit exceeded: %w", err)
	}

	// Create execution
	execution := &models.AutomationJob{
		ID:         uuid.New(),
		UserID:     userID,
		WorkflowID: workflow.ID,
		Status:     models.JobStatusPending,
		Input:      models.JSONB(input),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Save execution
	if err := m.executionRepo.CreateExecution(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// Log audit event
	if m.config.EnableAuditLogging {
		m.logAuditEvent(ctx, "workflow_executed", userID, workflow.ID, map[string]interface{}{
			"execution_id":  execution.ID,
			"workflow_name": workflow.Name,
		})
	}

	// Start execution in background
	go m.executeWorkflowAsync(ctx, execution, workflow, input)

	return execution, nil
}

// GetExecution retrieves an execution by ID
func (m *Manager) GetExecution(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*models.AutomationJob, error) {
	execution, err := m.executionRepo.GetExecution(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	// Check ownership
	if execution.UserID != userID {
		return nil, fmt.Errorf("execution not found or access denied")
	}

	return execution, nil
}

// ListExecutions lists executions with optional filtering
func (m *Manager) ListExecutions(ctx context.Context, userID uuid.UUID, filter ExecutionFilter) ([]*models.AutomationJob, error) {
	// Set user ID filter
	filter.UserID = &userID

	executions, err := m.executionRepo.ListExecutions(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	return executions, nil
}

// GetExecutionHistory retrieves execution history for a workflow
func (m *Manager) GetExecutionHistory(ctx context.Context, userID uuid.UUID, workflowID uuid.UUID, limit int) ([]*models.AutomationJob, error) {
	// Check workflow ownership
	workflow, err := m.workflowRepo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	if workflow.UserID != userID {
		return nil, fmt.Errorf("workflow not found or access denied")
	}

	history, err := m.executionRepo.GetExecutionHistory(ctx, workflowID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution history: %w", err)
	}
	return history, nil
}

// GetWorkflowStats retrieves statistics for a workflow
func (m *Manager) GetWorkflowStats(ctx context.Context, userID uuid.UUID, workflowID uuid.UUID) (*WorkflowStats, error) {
	// Check workflow ownership
	workflow, err := m.workflowRepo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	if workflow.UserID != userID {
		return nil, fmt.Errorf("workflow not found or access denied")
	}

	stats, err := m.workflowRepo.GetWorkflowStats(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow stats: %w", err)
	}
	return stats, nil
}

// GetExecutionStats retrieves execution statistics for a workflow
func (m *Manager) GetExecutionStats(ctx context.Context, userID uuid.UUID, workflowID uuid.UUID) (*ExecutionStats, error) {
	// Check workflow ownership
	workflow, err := m.workflowRepo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	if workflow.UserID != userID {
		return nil, fmt.Errorf("workflow not found or access denied")
	}

	stats, err := m.executionRepo.GetExecutionStats(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution stats: %w", err)
	}
	return stats, nil
}

// CancelExecution cancels a running execution
func (m *Manager) CancelExecution(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	execution, err := m.executionRepo.GetExecution(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get execution: %w", err)
	}

	// Check ownership
	if execution.UserID != userID {
		return fmt.Errorf("execution not found or access denied")
	}

	// Check if execution can be cancelled
	if execution.Status != models.JobStatusRunning && execution.Status != models.JobStatusPending {
		return fmt.Errorf("execution cannot be cancelled")
	}

	// Update execution status
	execution.Status = models.JobStatusCancelled
	execution.UpdatedAt = time.Now()

	if err := m.executionRepo.UpdateExecution(ctx, execution); err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	// Log audit event
	if m.config.EnableAuditLogging {
		m.logAuditEvent(ctx, "execution_cancelled", userID, execution.WorkflowID, map[string]interface{}{
			"execution_id": execution.ID,
		})
	}

	// Publish execution cancelled event
	if m.config.EnableEventLogging {
		event := &models.AutomationEvent{
			ID:          uuid.New().String(),
			Type:        models.EventTypeExecutionCancelled,
			UserID:      userID,
			WorkflowID:  execution.WorkflowID,
			ExecutionID: id,
			Message:     fmt.Sprintf("Execution %s cancelled", id),
			Data:        models.JSONB(execution),
			Timestamp:   time.Now(),
		}
		if err := m.eventBus.PublishEvent(ctx, event); err != nil {
			log.Printf("Failed to publish execution cancelled event: %v", err)
		}
	}

	return nil
}

// checkWorkflowLimit checks if the user has reached the workflow limit
func (m *Manager) checkWorkflowLimit(ctx context.Context, userID uuid.UUID) error {
	if m.config.MaxWorkflowsPerUser <= 0 {
		return nil // No limit
	}

	filter := WorkflowFilter{
		UserID: &userID,
		Limit:  1,
	}

	workflows, err := m.workflowRepo.ListWorkflows(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to check workflow limit: %w", err)
	}

	if len(workflows) >= m.config.MaxWorkflowsPerUser {
		return fmt.Errorf("workflow limit exceeded: %d workflows", m.config.MaxWorkflowsPerUser)
	}

	return nil
}

// checkExecutionRateLimit checks if the user has reached the execution rate limit
func (m *Manager) checkExecutionRateLimit(ctx context.Context, userID uuid.UUID) error {
	if m.config.MaxExecutionsPerHour <= 0 {
		return nil // No limit
	}

	// Check executions in the last hour
	oneHourAgo := time.Now().Add(-time.Hour)
	filter := ExecutionFilter{
		UserID:    &userID,
		StartTime: &oneHourAgo,
		Limit:     m.config.MaxExecutionsPerHour + 1,
	}

	executions, err := m.executionRepo.ListExecutions(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to check execution rate limit: %w", err)
	}

	if len(executions) >= m.config.MaxExecutionsPerHour {
		return fmt.Errorf("execution rate limit exceeded: %d executions per hour", m.config.MaxExecutionsPerHour)
	}

	return nil
}

// validateWorkflow validates a workflow request
func (m *Manager) validateWorkflow(req *models.AutomationWorkflowRequest) error {
	if req.Name == "" {
		return fmt.Errorf("workflow name is required")
	}

	if len(req.Actions) == 0 {
		return fmt.Errorf("workflow must have at least one action")
	}

	// Validate actions
	for i, action := range req.Actions {
		if err := m.actionExecutor.ValidateAction(context.Background(), action); err != nil {
			return fmt.Errorf("action %d validation failed: %w", i, err)
		}
	}

	return nil
}

// validateWorkflowForActivation validates a workflow before activation
func (m *Manager) validateWorkflowForActivation(workflow *models.AutomationWorkflow) error {
	if workflow.Trigger.Type == "" {
		return fmt.Errorf("workflow trigger is required")
	}

	if len(workflow.Actions) == 0 {
		return fmt.Errorf("workflow must have at least one action")
	}

	// Validate actions
	for i, action := range workflow.Actions {
		if err := m.actionExecutor.ValidateAction(context.Background(), action); err != nil {
			return fmt.Errorf("action %d validation failed: %w", err)
		}
	}

	return nil
}

// executeWorkflowAsync executes a workflow asynchronously
func (m *Manager) executeWorkflowAsync(ctx context.Context, execution *models.AutomationJob, workflow *models.AutomationWorkflow, input map[string]interface{}) {
	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, m.config.ExecutionTimeout)
	defer cancel()

	// Update execution status to running
	execution.Status = models.JobStatusRunning
	execution.UpdatedAt = time.Now()
	if err := m.executionRepo.UpdateExecution(execCtx, execution); err != nil {
		log.Printf("Failed to update execution status: %v", err)
	}

	// Publish execution started event
	if m.config.EnableEventLogging {
		event := &models.AutomationEvent{
			ID:          uuid.New().String(),
			Type:        models.EventTypeExecutionStarted,
			UserID:      execution.UserID,
			WorkflowID:  workflow.ID,
			ExecutionID: execution.ID,
			Message:     fmt.Sprintf("Execution %s started", execution.ID),
			Data:        models.JSONB(execution),
			Timestamp:   time.Now(),
		}
		if err := m.eventBus.PublishEvent(execCtx, event); err != nil {
			log.Printf("Failed to publish execution started event: %v", err)
		}
	}

	// Execute actions
	var results []*models.ActionResult
	var errors []error

	for i, action := range workflow.Actions {
		select {
		case <-execCtx.Done():
			// Execution was cancelled
			execution.Status = models.JobStatusCancelled
			execution.UpdatedAt = time.Now()
			m.executionRepo.UpdateExecution(execCtx, execution)
			return
		default:
			// Execute action
			result, err := m.actionExecutor.ExecuteAction(execCtx, action, input)
			if err != nil {
				errors = append(errors, fmt.Errorf("action %d failed: %w", i, err))
				if workflow.Settings.StopOnError {
					break
				}
			} else {
				results = append(results, result)
				// Update input with action result
				if result.Output != nil {
					input[fmt.Sprintf("action_%d_result", i)] = result.Output
				}
			}
		}
	}

	// Update execution with results
	execution.Results = models.JSONB(results)
	execution.UpdatedAt = time.Now()

	if len(errors) > 0 {
		execution.Status = models.JobStatusFailed
		execution.Error = fmt.Sprintf("Execution failed with %d errors", len(errors))
	} else {
		execution.Status = models.JobStatusCompleted
	}

	// Save execution results
	if err := m.executionRepo.UpdateExecution(execCtx, execution); err != nil {
		log.Printf("Failed to update execution results: %v", err)
	}

	// Publish execution completed event
	if m.config.EnableEventLogging {
		eventType := models.EventTypeExecutionCompleted
		if execution.Status == models.JobStatusFailed {
			eventType = models.EventTypeExecutionFailed
		}

		event := &models.AutomationEvent{
			ID:          uuid.New().String(),
			Type:        eventType,
			UserID:      execution.UserID,
			WorkflowID:  workflow.ID,
			ExecutionID: execution.ID,
			Message:     fmt.Sprintf("Execution %s %s", execution.ID, execution.Status),
			Data:        models.JSONB(execution),
			Timestamp:   time.Now(),
		}
		if err := m.eventBus.PublishEvent(execCtx, event); err != nil {
			log.Printf("Failed to publish execution completed event: %v", err)
		}
	}
}

// logAuditEvent logs an audit event
func (m *Manager) logAuditEvent(ctx context.Context, action string, userID uuid.UUID, workflowID uuid.UUID, data map[string]interface{}) {
	// In a real implementation, this would log to an audit system
	log.Printf("AUDIT: %s by user %s for workflow %s: %+v", action, userID, workflowID, data)
}
