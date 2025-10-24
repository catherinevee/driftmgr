package engine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/google/uuid"
)

// Engine represents the automation engine
type Engine struct {
	workflowRepo     WorkflowRepository
	executionRepo    ExecutionRepository
	actionExecutor   ActionExecutor
	triggerManager   TriggerManager
	eventBus         EventBus
	config           EngineConfig
	activeExecutions map[uuid.UUID]*ExecutionContext
	mu               sync.RWMutex
}

// WorkflowRepository defines the interface for workflow persistence
type WorkflowRepository interface {
	CreateWorkflow(ctx context.Context, workflow *models.AutomationWorkflow) error
	GetWorkflow(ctx context.Context, id uuid.UUID) (*models.AutomationWorkflow, error)
	UpdateWorkflow(ctx context.Context, workflow *models.AutomationWorkflow) error
	DeleteWorkflow(ctx context.Context, id uuid.UUID) error
	ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]*models.AutomationWorkflow, error)
}

// ExecutionRepository defines the interface for execution persistence
type ExecutionRepository interface {
	CreateExecution(ctx context.Context, execution *models.AutomationJob) error
	UpdateExecution(ctx context.Context, execution *models.AutomationJob) error
	GetExecution(ctx context.Context, id uuid.UUID) (*models.AutomationJob, error)
	ListExecutions(ctx context.Context, filter ExecutionFilter) ([]*models.AutomationJob, error)
	GetExecutionHistory(ctx context.Context, workflowID uuid.UUID, limit int) ([]*models.AutomationJob, error)
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

// EngineConfig holds configuration for the automation engine
type EngineConfig struct {
	MaxConcurrentExecutions int           `json:"max_concurrent_executions"`
	ExecutionTimeout        time.Duration `json:"execution_timeout"`
	RetryAttempts           int           `json:"retry_attempts"`
	RetryDelay              time.Duration `json:"retry_delay"`
	EnableEventLogging      bool          `json:"enable_event_logging"`
	EnableMetrics           bool          `json:"enable_metrics"`
}

// ExecutionContext holds the context for a workflow execution
type ExecutionContext struct {
	Execution  *models.AutomationJob
	Workflow   *models.AutomationWorkflow
	Context    map[string]interface{}
	StartTime  time.Time
	CancelFunc context.CancelFunc
	ResultChan chan *models.ActionResult
	ErrorChan  chan error
	mu         sync.RWMutex
}

// WorkflowFilter defines filters for workflow queries
type WorkflowFilter struct {
	Status      *models.WorkflowStatus `json:"status,omitempty"`
	TriggerType *models.TriggerType    `json:"trigger_type,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Limit       int                    `json:"limit,omitempty"`
	Offset      int                    `json:"offset,omitempty"`
}

// ExecutionFilter defines filters for execution queries
type ExecutionFilter struct {
	WorkflowID *uuid.UUID        `json:"workflow_id,omitempty"`
	Status     *models.JobStatus `json:"status,omitempty"`
	StartTime  *time.Time        `json:"start_time,omitempty"`
	EndTime    *time.Time        `json:"end_time,omitempty"`
	Limit      int               `json:"limit,omitempty"`
	Offset     int               `json:"offset,omitempty"`
}

// NewEngine creates a new automation engine
func NewEngine(
	workflowRepo WorkflowRepository,
	executionRepo ExecutionRepository,
	actionExecutor ActionExecutor,
	triggerManager TriggerManager,
	eventBus EventBus,
	config EngineConfig,
) *Engine {
	return &Engine{
		workflowRepo:     workflowRepo,
		executionRepo:    executionRepo,
		actionExecutor:   actionExecutor,
		triggerManager:   triggerManager,
		eventBus:         eventBus,
		config:           config,
		activeExecutions: make(map[uuid.UUID]*ExecutionContext),
	}
}

// CreateWorkflow creates a new automation workflow
func (e *Engine) CreateWorkflow(ctx context.Context, req *models.AutomationWorkflowRequest) (*models.AutomationWorkflow, error) {
	// Validate the workflow
	if err := e.validateWorkflow(req); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	// Create the workflow
	workflow := &models.AutomationWorkflow{
		ID:          uuid.New().String(),
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
	if err := e.workflowRepo.CreateWorkflow(ctx, workflow); err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	// Publish workflow created event
	if e.config.EnableEventLogging {
		event := &models.AutomationEvent{
			ID:         uuid.New().String(),
			Type:       models.EventTypeWorkflowCreated,
			WorkflowID: workflow.ID,
			Message:    fmt.Sprintf("Workflow '%s' created", workflow.Name),
			Data:       models.JSONB(workflow),
			Timestamp:  time.Now(),
		}
		if err := e.eventBus.PublishEvent(ctx, event); err != nil {
			log.Printf("Failed to publish workflow created event: %v", err)
		}
	}

	return workflow, nil
}

// GetWorkflow retrieves a workflow by ID
func (e *Engine) GetWorkflow(ctx context.Context, id uuid.UUID) (*models.AutomationWorkflow, error) {
	workflow, err := e.workflowRepo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}
	return workflow, nil
}

// UpdateWorkflow updates an existing workflow
func (e *Engine) UpdateWorkflow(ctx context.Context, id uuid.UUID, req *models.AutomationWorkflowRequest) (*models.AutomationWorkflow, error) {
	// Get existing workflow
	workflow, err := e.workflowRepo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// Validate the workflow
	if err := e.validateWorkflow(req); err != nil {
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
	if err := e.workflowRepo.UpdateWorkflow(ctx, workflow); err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	// Re-register trigger if workflow is active
	if workflow.Status == models.WorkflowStatusActive {
		if err := e.triggerManager.RegisterTrigger(ctx, workflow); err != nil {
			log.Printf("Failed to re-register trigger for workflow %s: %v", id, err)
		}
	}

	// Publish workflow updated event
	if e.config.EnableEventLogging {
		event := &models.AutomationEvent{
			ID:         uuid.New().String(),
			Type:       models.EventTypeWorkflowUpdated,
			WorkflowID: workflow.ID,
			Message:    fmt.Sprintf("Workflow '%s' updated", workflow.Name),
			Data:       models.JSONB(workflow),
			Timestamp:  time.Now(),
		}
		if err := e.eventBus.PublishEvent(ctx, event); err != nil {
			log.Printf("Failed to publish workflow updated event: %v", err)
		}
	}

	return workflow, nil
}

// DeleteWorkflow deletes a workflow
func (e *Engine) DeleteWorkflow(ctx context.Context, id uuid.UUID) error {
	// Get workflow to check if it exists
	workflow, err := e.workflowRepo.GetWorkflow(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	// Unregister trigger if workflow is active
	if workflow.Status == models.WorkflowStatusActive {
		if err := e.triggerManager.UnregisterTrigger(ctx, id); err != nil {
			log.Printf("Failed to unregister trigger for workflow %s: %v", id, err)
		}
	}

	// Delete the workflow
	if err := e.workflowRepo.DeleteWorkflow(ctx, id); err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	// Publish workflow deleted event
	if e.config.EnableEventLogging {
		event := &models.AutomationEvent{
			ID:         uuid.New().String(),
			Type:       models.EventTypeWorkflowDeleted,
			WorkflowID: id,
			Message:    fmt.Sprintf("Workflow '%s' deleted", workflow.Name),
			Data:       models.JSONB(workflow),
			Timestamp:  time.Now(),
		}
		if err := e.eventBus.PublishEvent(ctx, event); err != nil {
			log.Printf("Failed to publish workflow deleted event: %v", err)
		}
	}

	return nil
}

// ListWorkflows lists workflows with optional filtering
func (e *Engine) ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]*models.AutomationWorkflow, error) {
	workflows, err := e.workflowRepo.ListWorkflows(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	return workflows, nil
}

// ActivateWorkflow activates a workflow
func (e *Engine) ActivateWorkflow(ctx context.Context, id uuid.UUID) error {
	workflow, err := e.workflowRepo.GetWorkflow(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	if workflow.Status == models.WorkflowStatusActive {
		return fmt.Errorf("workflow is already active")
	}

	// Validate workflow before activation
	if err := e.validateWorkflowForActivation(workflow); err != nil {
		return fmt.Errorf("workflow validation failed: %w", err)
	}

	// Update workflow status
	workflow.Status = models.WorkflowStatusActive
	workflow.UpdatedAt = time.Now()

	if err := e.workflowRepo.UpdateWorkflow(ctx, workflow); err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	// Register trigger
	if err := e.triggerManager.RegisterTrigger(ctx, workflow); err != nil {
		// Rollback status change
		workflow.Status = models.WorkflowStatusDraft
		e.workflowRepo.UpdateWorkflow(ctx, workflow)
		return fmt.Errorf("failed to register trigger: %w", err)
	}

	// Publish workflow activated event
	if e.config.EnableEventLogging {
		event := &models.AutomationEvent{
			ID:         uuid.New().String(),
			Type:       models.EventTypeWorkflowActivated,
			WorkflowID: workflow.ID,
			Message:    fmt.Sprintf("Workflow '%s' activated", workflow.Name),
			Data:       models.JSONB(workflow),
			Timestamp:  time.Now(),
		}
		if err := e.eventBus.PublishEvent(ctx, event); err != nil {
			log.Printf("Failed to publish workflow activated event: %v", err)
		}
	}

	return nil
}

// DeactivateWorkflow deactivates a workflow
func (e *Engine) DeactivateWorkflow(ctx context.Context, id uuid.UUID) error {
	workflow, err := e.workflowRepo.GetWorkflow(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	if workflow.Status != models.WorkflowStatusActive {
		return fmt.Errorf("workflow is not active")
	}

	// Unregister trigger
	if err := e.triggerManager.UnregisterTrigger(ctx, id); err != nil {
		log.Printf("Failed to unregister trigger for workflow %s: %v", id, err)
	}

	// Update workflow status
	workflow.Status = models.WorkflowStatusDraft
	workflow.UpdatedAt = time.Now()

	if err := e.workflowRepo.UpdateWorkflow(ctx, workflow); err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	// Publish workflow deactivated event
	if e.config.EnableEventLogging {
		event := &models.AutomationEvent{
			ID:         uuid.New().String(),
			Type:       models.EventTypeWorkflowDeactivated,
			WorkflowID: workflow.ID,
			Message:    fmt.Sprintf("Workflow '%s' deactivated", workflow.Name),
			Data:       models.JSONB(workflow),
			Timestamp:  time.Now(),
		}
		if err := e.eventBus.PublishEvent(ctx, event); err != nil {
			log.Printf("Failed to publish workflow deactivated event: %v", err)
		}
	}

	return nil
}

// ExecuteWorkflow executes a workflow manually
func (e *Engine) ExecuteWorkflow(ctx context.Context, id uuid.UUID, input map[string]interface{}) (*models.AutomationJob, error) {
	workflow, err := e.workflowRepo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// Check if workflow is active
	if workflow.Status != models.WorkflowStatusActive {
		return nil, fmt.Errorf("workflow is not active")
	}

	// Check concurrent execution limit
	if len(e.activeExecutions) >= e.config.MaxConcurrentExecutions {
		return nil, fmt.Errorf("maximum concurrent executions reached")
	}

	// Create execution
	execution := &models.AutomationJob{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Status:     models.JobStatusPending,
		Input:      models.JSONB(input),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Save execution
	if err := e.executionRepo.CreateExecution(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// Start execution in background
	go e.executeWorkflowAsync(ctx, execution, workflow, input)

	return execution, nil
}

// GetExecution retrieves an execution by ID
func (e *Engine) GetExecution(ctx context.Context, id uuid.UUID) (*models.AutomationJob, error) {
	execution, err := e.executionRepo.GetExecution(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}
	return execution, nil
}

// ListExecutions lists executions with optional filtering
func (e *Engine) ListExecutions(ctx context.Context, filter ExecutionFilter) ([]*models.AutomationJob, error) {
	executions, err := e.executionRepo.ListExecutions(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	return executions, nil
}

// GetExecutionHistory retrieves execution history for a workflow
func (e *Engine) GetExecutionHistory(ctx context.Context, workflowID uuid.UUID, limit int) ([]*models.AutomationJob, error) {
	history, err := e.executionRepo.GetExecutionHistory(ctx, workflowID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution history: %w", err)
	}
	return history, nil
}

// CancelExecution cancels a running execution
func (e *Engine) CancelExecution(ctx context.Context, id uuid.UUID) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	executionCtx, exists := e.activeExecutions[id]
	if !exists {
		return fmt.Errorf("execution not found or not running")
	}

	// Cancel the execution context
	executionCtx.CancelFunc()

	// Update execution status
	executionCtx.Execution.Status = models.JobStatusCancelled
	executionCtx.Execution.UpdatedAt = time.Now()

	if err := e.executionRepo.UpdateExecution(ctx, executionCtx.Execution); err != nil {
		log.Printf("Failed to update execution status: %v", err)
	}

	// Remove from active executions
	delete(e.activeExecutions, id)

	// Publish execution cancelled event
	if e.config.EnableEventLogging {
		event := &models.AutomationEvent{
			ID:          uuid.New().String(),
			Type:        models.EventTypeExecutionCancelled,
			WorkflowID:  executionCtx.Workflow.ID,
			ExecutionID: id,
			Message:     fmt.Sprintf("Execution %s cancelled", id),
			Data:        models.JSONB(executionCtx.Execution),
			Timestamp:   time.Now(),
		}
		if err := e.eventBus.PublishEvent(ctx, event); err != nil {
			log.Printf("Failed to publish execution cancelled event: %v", err)
		}
	}

	return nil
}

// Start starts the automation engine
func (e *Engine) Start(ctx context.Context) error {
	// Start trigger monitoring
	if err := e.triggerManager.StartTriggerMonitoring(ctx); err != nil {
		return fmt.Errorf("failed to start trigger monitoring: %w", err)
	}

	log.Println("Automation engine started successfully")
	return nil
}

// Stop stops the automation engine
func (e *Engine) Stop(ctx context.Context) error {
	// Stop trigger monitoring
	if err := e.triggerManager.StopTriggerMonitoring(ctx); err != nil {
		log.Printf("Failed to stop trigger monitoring: %v", err)
	}

	// Cancel all active executions
	e.mu.Lock()
	for _, executionCtx := range e.activeExecutions {
		executionCtx.CancelFunc()
	}
	e.mu.Unlock()

	log.Println("Automation engine stopped successfully")
	return nil
}

// validateWorkflow validates a workflow request
func (e *Engine) validateWorkflow(req *models.AutomationWorkflowRequest) error {
	if req.Name == "" {
		return fmt.Errorf("workflow name is required")
	}

	if len(req.Actions) == 0 {
		return fmt.Errorf("workflow must have at least one action")
	}

	// Validate actions
	for i, action := range req.Actions {
		if err := e.actionExecutor.ValidateAction(context.Background(), action); err != nil {
			return fmt.Errorf("action %d validation failed: %w", i, err)
		}
	}

	return nil
}

// validateWorkflowForActivation validates a workflow before activation
func (e *Engine) validateWorkflowForActivation(workflow *models.AutomationWorkflow) error {
	if workflow.Trigger.Type == "" {
		return fmt.Errorf("workflow trigger is required")
	}

	if len(workflow.Actions) == 0 {
		return fmt.Errorf("workflow must have at least one action")
	}

	// Validate actions
	for i, action := range workflow.Actions {
		if err := e.actionExecutor.ValidateAction(context.Background(), action); err != nil {
			return fmt.Errorf("action %d validation failed: %w", i, err)
		}
	}

	return nil
}

// executeWorkflowAsync executes a workflow asynchronously
func (e *Engine) executeWorkflowAsync(ctx context.Context, execution *models.AutomationJob, workflow *models.AutomationWorkflow, input map[string]interface{}) {
	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, e.config.ExecutionTimeout)
	defer cancel()

	// Create execution context
	executionCtx := &ExecutionContext{
		Execution:  execution,
		Workflow:   workflow,
		Context:    input,
		StartTime:  time.Now(),
		CancelFunc: cancel,
		ResultChan: make(chan *models.ActionResult, len(workflow.Actions)),
		ErrorChan:  make(chan error, len(workflow.Actions)),
	}

	// Add to active executions
	e.mu.Lock()
	e.activeExecutions[execution.ID] = executionCtx
	e.mu.Unlock()

	// Remove from active executions when done
	defer func() {
		e.mu.Lock()
		delete(e.activeExecutions, execution.ID)
		e.mu.Unlock()
	}()

	// Update execution status to running
	execution.Status = models.JobStatusRunning
	execution.UpdatedAt = time.Now()
	if err := e.executionRepo.UpdateExecution(execCtx, execution); err != nil {
		log.Printf("Failed to update execution status: %v", err)
	}

	// Publish execution started event
	if e.config.EnableEventLogging {
		event := &models.AutomationEvent{
			ID:          uuid.New().String(),
			Type:        models.EventTypeExecutionStarted,
			WorkflowID:  workflow.ID,
			ExecutionID: execution.ID,
			Message:     fmt.Sprintf("Execution %s started", execution.ID),
			Data:        models.JSONB(execution),
			Timestamp:   time.Now(),
		}
		if err := e.eventBus.PublishEvent(execCtx, event); err != nil {
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
			e.executionRepo.UpdateExecution(execCtx, execution)
			return
		default:
			// Execute action
			result, err := e.actionExecutor.ExecuteAction(execCtx, action, executionCtx.Context)
			if err != nil {
				errors = append(errors, fmt.Errorf("action %d failed: %w", i, err))
				if workflow.Settings.StopOnError {
					break
				}
			} else {
				results = append(results, result)
				// Update context with action result
				if result.Output != nil {
					executionCtx.Context[fmt.Sprintf("action_%d_result", i)] = result.Output
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
	if err := e.executionRepo.UpdateExecution(execCtx, execution); err != nil {
		log.Printf("Failed to update execution results: %v", err)
	}

	// Publish execution completed event
	if e.config.EnableEventLogging {
		eventType := models.EventTypeExecutionCompleted
		if execution.Status == models.JobStatusFailed {
			eventType = models.EventTypeExecutionFailed
		}

		event := &models.AutomationEvent{
			ID:          uuid.New().String(),
			Type:        eventType,
			WorkflowID:  workflow.ID,
			ExecutionID: execution.ID,
			Message:     fmt.Sprintf("Execution %s %s", execution.ID, execution.Status),
			Data:        models.JSONB(execution),
			Timestamp:   time.Now(),
		}
		if err := e.eventBus.PublishEvent(execCtx, event); err != nil {
			log.Printf("Failed to publish execution completed event: %v", err)
		}
	}
}
