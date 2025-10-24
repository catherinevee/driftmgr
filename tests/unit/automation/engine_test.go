package automation

import (
	"context"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/automation/engine"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWorkflowRepository is a mock implementation of the workflow repository
type MockWorkflowRepository struct {
	mock.Mock
}

func (m *MockWorkflowRepository) CreateWorkflow(ctx context.Context, workflow *models.AutomationWorkflow) error {
	args := m.Called(ctx, workflow)
	return args.Error(0)
}

func (m *MockWorkflowRepository) GetWorkflow(ctx context.Context, id uuid.UUID) (*models.AutomationWorkflow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.AutomationWorkflow), args.Error(1)
}

func (m *MockWorkflowRepository) UpdateWorkflow(ctx context.Context, workflow *models.AutomationWorkflow) error {
	args := m.Called(ctx, workflow)
	return args.Error(0)
}

func (m *MockWorkflowRepository) DeleteWorkflow(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockWorkflowRepository) ListWorkflows(ctx context.Context, filter engine.WorkflowFilter) ([]*models.AutomationWorkflow, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*models.AutomationWorkflow), args.Error(1)
}

func (m *MockWorkflowRepository) GetWorkflowStats(ctx context.Context, id uuid.UUID) (*engine.WorkflowStats, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*engine.WorkflowStats), args.Error(1)
}

// MockExecutionRepository is a mock implementation of the execution repository
type MockExecutionRepository struct {
	mock.Mock
}

func (m *MockExecutionRepository) CreateExecution(ctx context.Context, execution *models.AutomationJob) error {
	args := m.Called(ctx, execution)
	return args.Error(0)
}

func (m *MockExecutionRepository) UpdateExecution(ctx context.Context, execution *models.AutomationJob) error {
	args := m.Called(ctx, execution)
	return args.Error(0)
}

func (m *MockExecutionRepository) GetExecution(ctx context.Context, id uuid.UUID) (*models.AutomationJob, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.AutomationJob), args.Error(1)
}

func (m *MockExecutionRepository) ListExecutions(ctx context.Context, filter engine.ExecutionFilter) ([]*models.AutomationJob, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*models.AutomationJob), args.Error(1)
}

func (m *MockExecutionRepository) GetExecutionHistory(ctx context.Context, workflowID uuid.UUID, limit int) ([]*models.AutomationJob, error) {
	args := m.Called(ctx, workflowID, limit)
	return args.Get(0).([]*models.AutomationJob), args.Error(1)
}

func (m *MockExecutionRepository) GetExecutionStats(ctx context.Context, workflowID uuid.UUID) (*engine.ExecutionStats, error) {
	args := m.Called(ctx, workflowID)
	return args.Get(0).(*engine.ExecutionStats), args.Error(1)
}

// MockActionExecutor is a mock implementation of the action executor
type MockActionExecutor struct {
	mock.Mock
}

func (m *MockActionExecutor) ExecuteAction(ctx context.Context, action *models.AutomationAction, context map[string]interface{}) (*models.ActionResult, error) {
	args := m.Called(ctx, action, context)
	return args.Get(0).(*models.ActionResult), args.Error(1)
}

func (m *MockActionExecutor) ValidateAction(ctx context.Context, action *models.AutomationAction) error {
	args := m.Called(ctx, action)
	return args.Error(0)
}

func (m *MockActionExecutor) GetSupportedActionTypes() []models.ActionType {
	args := m.Called()
	return args.Get(0).([]models.ActionType)
}

// MockTriggerManager is a mock implementation of the trigger manager
type MockTriggerManager struct {
	mock.Mock
}

func (m *MockTriggerManager) RegisterTrigger(ctx context.Context, workflow *models.AutomationWorkflow) error {
	args := m.Called(ctx, workflow)
	return args.Error(0)
}

func (m *MockTriggerManager) UnregisterTrigger(ctx context.Context, workflowID uuid.UUID) error {
	args := m.Called(ctx, workflowID)
	return args.Error(0)
}

func (m *MockTriggerManager) StartTriggerMonitoring(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTriggerManager) StopTriggerMonitoring(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockEventBus is a mock implementation of the event bus
type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) PublishEvent(ctx context.Context, event *models.AutomationEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventBus) SubscribeToEvents(ctx context.Context, eventType string, handler engine.EventHandler) error {
	args := m.Called(ctx, eventType, handler)
	return args.Error(0)
}

func (m *MockEventBus) UnsubscribeFromEvents(ctx context.Context, eventType string) error {
	args := m.Called(ctx, eventType)
	return args.Error(0)
}

func TestAutomationEngine_CreateWorkflow(t *testing.T) {
	// Setup
	mockWorkflowRepo := new(MockWorkflowRepository)
	mockExecutionRepo := new(MockExecutionRepository)
	mockActionExecutor := new(MockActionExecutor)
	mockTriggerManager := new(MockTriggerManager)
	mockEventBus := new(MockEventBus)

	config := engine.EngineConfig{
		MaxConcurrentExecutions: 10,
		ExecutionTimeout:        30 * time.Second,
		RetryAttempts:           3,
		RetryDelay:              5 * time.Second,
		EnableEventLogging:      true,
		EnableMetrics:           true,
	}

	automationEngine := engine.NewEngine(
		mockWorkflowRepo,
		mockExecutionRepo,
		mockActionExecutor,
		mockTriggerManager,
		mockEventBus,
		config,
	)

	req := &models.AutomationWorkflowRequest{
		Name:        "Test Workflow",
		Description: "Test workflow description",
		Trigger: models.AutomationTrigger{
			Type: models.TriggerTypeManual,
			Configuration: models.JSONB(map[string]interface{}{
				"manual": true,
			}),
		},
		Actions: []models.AutomationAction{
			{
				ID:   uuid.New(),
				Name: "Test Action",
				Type: models.ActionTypeTerraform,
				Configuration: models.JSONB(map[string]interface{}{
					"operation":   "plan",
					"config_path": "/path/to/config",
				}),
			},
		},
		Conditions: []models.AutomationCondition{},
		Settings: models.AutomationSettings{
			StopOnError:    true,
			RetryOnFailure: false,
		},
		Tags: []string{"test"},
	}

	// Mock expectations
	mockActionExecutor.On("ValidateAction", mock.Anything, mock.AnythingOfType("*models.AutomationAction")).Return(nil)
	mockWorkflowRepo.On("CreateWorkflow", mock.Anything, mock.AnythingOfType("*models.AutomationWorkflow")).Return(nil)
	mockEventBus.On("PublishEvent", mock.Anything, mock.AnythingOfType("*models.AutomationEvent")).Return(nil)

	// Execute
	workflow, err := automationEngine.CreateWorkflow(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, workflow)
	assert.Equal(t, req.Name, workflow.Name)
	assert.Equal(t, req.Description, workflow.Description)
	assert.Equal(t, models.WorkflowStatusDraft, workflow.Status)
	assert.Len(t, workflow.Actions, 1)

	mockWorkflowRepo.AssertExpectations(t)
	mockActionExecutor.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestAutomationEngine_ExecuteWorkflow(t *testing.T) {
	// Setup
	mockWorkflowRepo := new(MockWorkflowRepository)
	mockExecutionRepo := new(MockExecutionRepository)
	mockActionExecutor := new(MockActionExecutor)
	mockTriggerManager := new(MockTriggerManager)
	mockEventBus := new(MockEventBus)

	config := engine.EngineConfig{
		MaxConcurrentExecutions: 10,
		ExecutionTimeout:        30 * time.Second,
		RetryAttempts:           3,
		RetryDelay:              5 * time.Second,
		EnableEventLogging:      true,
		EnableMetrics:           true,
	}

	automationEngine := engine.NewEngine(
		mockWorkflowRepo,
		mockExecutionRepo,
		mockActionExecutor,
		mockTriggerManager,
		mockEventBus,
		config,
	)

	workflowID := uuid.New()
	workflow := &models.AutomationWorkflow{
		ID:     workflowID,
		Name:   "Test Workflow",
		Status: models.WorkflowStatusActive,
		Actions: []models.AutomationAction{
			{
				ID:   uuid.New(),
				Name: "Test Action",
				Type: models.ActionTypeTerraform,
				Configuration: models.JSONB(map[string]interface{}{
					"operation":   "plan",
					"config_path": "/path/to/config",
				}),
			},
		},
		Settings: models.AutomationSettings{
			StopOnError:    true,
			RetryOnFailure: false,
		},
	}

	input := map[string]interface{}{
		"test_input": "test_value",
	}

	// Mock expectations
	mockWorkflowRepo.On("GetWorkflow", mock.Anything, workflowID).Return(workflow, nil)
	mockExecutionRepo.On("CreateExecution", mock.Anything, mock.AnythingOfType("*models.AutomationJob")).Return(nil)
	mockEventBus.On("PublishEvent", mock.Anything, mock.AnythingOfType("*models.AutomationEvent")).Return(nil)

	// Execute
	execution, err := automationEngine.ExecuteWorkflow(context.Background(), workflowID, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, execution)
	assert.Equal(t, workflowID, execution.WorkflowID)
	assert.Equal(t, models.JobStatusPending, execution.Status)

	mockWorkflowRepo.AssertExpectations(t)
	mockExecutionRepo.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestAutomationEngine_ActivateWorkflow(t *testing.T) {
	// Setup
	mockWorkflowRepo := new(MockWorkflowRepository)
	mockExecutionRepo := new(MockExecutionRepository)
	mockActionExecutor := new(MockActionExecutor)
	mockTriggerManager := new(MockTriggerManager)
	mockEventBus := new(MockEventBus)

	config := engine.EngineConfig{
		MaxConcurrentExecutions: 10,
		ExecutionTimeout:        30 * time.Second,
		RetryAttempts:           3,
		RetryDelay:              5 * time.Second,
		EnableEventLogging:      true,
		EnableMetrics:           true,
	}

	automationEngine := engine.NewEngine(
		mockWorkflowRepo,
		mockExecutionRepo,
		mockActionExecutor,
		mockTriggerManager,
		mockEventBus,
		config,
	)

	workflowID := uuid.New()
	workflow := &models.AutomationWorkflow{
		ID:     workflowID,
		Name:   "Test Workflow",
		Status: models.WorkflowStatusDraft,
		Trigger: models.AutomationTrigger{
			Type: models.TriggerTypeScheduled,
			Configuration: models.JSONB(map[string]interface{}{
				"schedule": "0 0 * * *",
			}),
		},
		Actions: []models.AutomationAction{
			{
				ID:   uuid.New(),
				Name: "Test Action",
				Type: models.ActionTypeTerraform,
				Configuration: models.JSONB(map[string]interface{}{
					"operation":   "plan",
					"config_path": "/path/to/config",
				}),
			},
		},
		Settings: models.AutomationSettings{
			StopOnError:    true,
			RetryOnFailure: false,
		},
	}

	// Mock expectations
	mockWorkflowRepo.On("GetWorkflow", mock.Anything, workflowID).Return(workflow, nil)
	mockActionExecutor.On("ValidateAction", mock.Anything, mock.AnythingOfType("*models.AutomationAction")).Return(nil)
	mockWorkflowRepo.On("UpdateWorkflow", mock.Anything, mock.AnythingOfType("*models.AutomationWorkflow")).Return(nil)
	mockTriggerManager.On("RegisterTrigger", mock.Anything, workflow).Return(nil)
	mockEventBus.On("PublishEvent", mock.Anything, mock.AnythingOfType("*models.AutomationEvent")).Return(nil)

	// Execute
	err := automationEngine.ActivateWorkflow(context.Background(), workflowID)

	// Assert
	assert.NoError(t, err)

	mockWorkflowRepo.AssertExpectations(t)
	mockActionExecutor.AssertExpectations(t)
	mockTriggerManager.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestAutomationEngine_DeactivateWorkflow(t *testing.T) {
	// Setup
	mockWorkflowRepo := new(MockWorkflowRepository)
	mockExecutionRepo := new(MockExecutionRepository)
	mockActionExecutor := new(MockActionExecutor)
	mockTriggerManager := new(MockTriggerManager)
	mockEventBus := new(MockEventBus)

	config := engine.EngineConfig{
		MaxConcurrentExecutions: 10,
		ExecutionTimeout:        30 * time.Second,
		RetryAttempts:           3,
		RetryDelay:              5 * time.Second,
		EnableEventLogging:      true,
		EnableMetrics:           true,
	}

	automationEngine := engine.NewEngine(
		mockWorkflowRepo,
		mockExecutionRepo,
		mockActionExecutor,
		mockTriggerManager,
		mockEventBus,
		config,
	)

	workflowID := uuid.New()
	workflow := &models.AutomationWorkflow{
		ID:     workflowID,
		Name:   "Test Workflow",
		Status: models.WorkflowStatusActive,
		Trigger: models.AutomationTrigger{
			Type: models.TriggerTypeScheduled,
			Configuration: models.JSONB(map[string]interface{}{
				"schedule": "0 0 * * *",
			}),
		},
		Actions: []models.AutomationAction{
			{
				ID:   uuid.New(),
				Name: "Test Action",
				Type: models.ActionTypeTerraform,
				Configuration: models.JSONB(map[string]interface{}{
					"operation":   "plan",
					"config_path": "/path/to/config",
				}),
			},
		},
		Settings: models.AutomationSettings{
			StopOnError:    true,
			RetryOnFailure: false,
		},
	}

	// Mock expectations
	mockWorkflowRepo.On("GetWorkflow", mock.Anything, workflowID).Return(workflow, nil)
	mockTriggerManager.On("UnregisterTrigger", mock.Anything, workflowID).Return(nil)
	mockWorkflowRepo.On("UpdateWorkflow", mock.Anything, mock.AnythingOfType("*models.AutomationWorkflow")).Return(nil)
	mockEventBus.On("PublishEvent", mock.Anything, mock.AnythingOfType("*models.AutomationEvent")).Return(nil)

	// Execute
	err := automationEngine.DeactivateWorkflow(context.Background(), workflowID)

	// Assert
	assert.NoError(t, err)

	mockWorkflowRepo.AssertExpectations(t)
	mockTriggerManager.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestAutomationEngine_GetExecution(t *testing.T) {
	// Setup
	mockWorkflowRepo := new(MockWorkflowRepository)
	mockExecutionRepo := new(MockExecutionRepository)
	mockActionExecutor := new(MockActionExecutor)
	mockTriggerManager := new(MockTriggerManager)
	mockEventBus := new(MockEventBus)

	config := engine.EngineConfig{
		MaxConcurrentExecutions: 10,
		ExecutionTimeout:        30 * time.Second,
		RetryAttempts:           3,
		RetryDelay:              5 * time.Second,
		EnableEventLogging:      true,
		EnableMetrics:           true,
	}

	automationEngine := engine.NewEngine(
		mockWorkflowRepo,
		mockExecutionRepo,
		mockActionExecutor,
		mockTriggerManager,
		mockEventBus,
		config,
	)

	executionID := uuid.New()
	execution := &models.AutomationJob{
		ID:         executionID,
		WorkflowID: uuid.New(),
		Status:     models.JobStatusCompleted,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Mock expectations
	mockExecutionRepo.On("GetExecution", mock.Anything, executionID).Return(execution, nil)

	// Execute
	result, err := automationEngine.GetExecution(context.Background(), executionID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, executionID, result.ID)
	assert.Equal(t, models.JobStatusCompleted, result.Status)

	mockExecutionRepo.AssertExpectations(t)
}

func TestAutomationEngine_CancelExecution(t *testing.T) {
	// Setup
	mockWorkflowRepo := new(MockWorkflowRepository)
	mockExecutionRepo := new(MockExecutionRepository)
	mockActionExecutor := new(MockActionExecutor)
	mockTriggerManager := new(MockTriggerManager)
	mockEventBus := new(MockEventBus)

	config := engine.EngineConfig{
		MaxConcurrentExecutions: 10,
		ExecutionTimeout:        30 * time.Second,
		RetryAttempts:           3,
		RetryDelay:              5 * time.Second,
		EnableEventLogging:      true,
		EnableMetrics:           true,
	}

	automationEngine := engine.NewEngine(
		mockWorkflowRepo,
		mockExecutionRepo,
		mockActionExecutor,
		mockTriggerManager,
		mockEventBus,
		config,
	)

	executionID := uuid.New()
	workflowID := uuid.New()
	workflow := &models.AutomationWorkflow{
		ID:   workflowID,
		Name: "Test Workflow",
	}

	execution := &models.AutomationJob{
		ID:         executionID,
		WorkflowID: workflowID,
		Status:     models.JobStatusRunning,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Mock expectations
	mockExecutionRepo.On("GetExecution", mock.Anything, executionID).Return(execution, nil)
	mockWorkflowRepo.On("GetWorkflow", mock.Anything, workflowID).Return(workflow, nil)
	mockExecutionRepo.On("UpdateExecution", mock.Anything, mock.AnythingOfType("*models.AutomationJob")).Return(nil)
	mockEventBus.On("PublishEvent", mock.Anything, mock.AnythingOfType("*models.AutomationEvent")).Return(nil)

	// Execute
	err := automationEngine.CancelExecution(context.Background(), executionID)

	// Assert
	assert.NoError(t, err)

	mockExecutionRepo.AssertExpectations(t)
	mockWorkflowRepo.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestAutomationEngine_Start(t *testing.T) {
	// Setup
	mockWorkflowRepo := new(MockWorkflowRepository)
	mockExecutionRepo := new(MockExecutionRepository)
	mockActionExecutor := new(MockActionExecutor)
	mockTriggerManager := new(MockTriggerManager)
	mockEventBus := new(MockEventBus)

	config := engine.EngineConfig{
		MaxConcurrentExecutions: 10,
		ExecutionTimeout:        30 * time.Second,
		RetryAttempts:           3,
		RetryDelay:              5 * time.Second,
		EnableEventLogging:      true,
		EnableMetrics:           true,
	}

	automationEngine := engine.NewEngine(
		mockWorkflowRepo,
		mockExecutionRepo,
		mockActionExecutor,
		mockTriggerManager,
		mockEventBus,
		config,
	)

	// Mock expectations
	mockTriggerManager.On("StartTriggerMonitoring", mock.Anything).Return(nil)

	// Execute
	err := automationEngine.Start(context.Background())

	// Assert
	assert.NoError(t, err)

	mockTriggerManager.AssertExpectations(t)
}

func TestAutomationEngine_Stop(t *testing.T) {
	// Setup
	mockWorkflowRepo := new(MockWorkflowRepository)
	mockExecutionRepo := new(MockExecutionRepository)
	mockActionExecutor := new(MockActionExecutor)
	mockTriggerManager := new(MockTriggerManager)
	mockEventBus := new(MockEventBus)

	config := engine.EngineConfig{
		MaxConcurrentExecutions: 10,
		ExecutionTimeout:        30 * time.Second,
		RetryAttempts:           3,
		RetryDelay:              5 * time.Second,
		EnableEventLogging:      true,
		EnableMetrics:           true,
	}

	automationEngine := engine.NewEngine(
		mockWorkflowRepo,
		mockExecutionRepo,
		mockActionExecutor,
		mockTriggerManager,
		mockEventBus,
		config,
	)

	// Mock expectations
	mockTriggerManager.On("StopTriggerMonitoring", mock.Anything).Return(nil)

	// Execute
	err := automationEngine.Stop(context.Background())

	// Assert
	assert.NoError(t, err)

	mockTriggerManager.AssertExpectations(t)
}
