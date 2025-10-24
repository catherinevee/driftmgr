package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// ActionExecutorImpl implements the ActionExecutor interface
type ActionExecutorImpl struct {
	terraformExecutor   TerraformExecutor
	apiClient           APIClient
	scriptExecutor      ScriptExecutor
	notificationService NotificationService
}

// TerraformExecutor defines the interface for executing Terraform commands
type TerraformExecutor interface {
	ExecuteCommand(ctx context.Context, command string, args []string, workingDir string) (*CommandResult, error)
	ValidateConfiguration(ctx context.Context, configPath string) error
	Plan(ctx context.Context, configPath string, variables map[string]string) (*PlanResult, error)
	Apply(ctx context.Context, configPath string, variables map[string]string) (*ApplyResult, error)
	Destroy(ctx context.Context, configPath string, variables map[string]string) (*DestroyResult, error)
}

// APIClient defines the interface for making API calls
type APIClient interface {
	MakeRequest(ctx context.Context, req *APIRequest) (*APIResponse, error)
	ValidateRequest(req *APIRequest) error
}

// ScriptExecutor defines the interface for executing scripts
type ScriptExecutor interface {
	ExecuteScript(ctx context.Context, script string, args []string, workingDir string) (*ScriptResult, error)
	ValidateScript(script string) error
}

// NotificationService defines the interface for sending notifications
type NotificationService interface {
	SendNotification(ctx context.Context, notification *NotificationRequest) error
	ValidateNotification(notification *NotificationRequest) error
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	ExitCode int           `json:"exit_code"`
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	Duration time.Duration `json:"duration"`
}

// PlanResult represents the result of a Terraform plan
type PlanResult struct {
	Changes  []ResourceChange `json:"changes"`
	Summary  PlanSummary      `json:"summary"`
	ExitCode int              `json:"exit_code"`
	Stdout   string           `json:"stdout"`
	Stderr   string           `json:"stderr"`
	Duration time.Duration    `json:"duration"`
}

// ApplyResult represents the result of a Terraform apply
type ApplyResult struct {
	Changes  []ResourceChange `json:"changes"`
	Summary  ApplySummary     `json:"summary"`
	ExitCode int              `json:"exit_code"`
	Stdout   string           `json:"stdout"`
	Stderr   string           `json:"stderr"`
	Duration time.Duration    `json:"duration"`
}

// DestroyResult represents the result of a Terraform destroy
type DestroyResult struct {
	Changes  []ResourceChange `json:"changes"`
	Summary  DestroySummary   `json:"summary"`
	ExitCode int              `json:"exit_code"`
	Stdout   string           `json:"stdout"`
	Stderr   string           `json:"stderr"`
	Duration time.Duration    `json:"duration"`
}

// ResourceChange represents a change to a resource
type ResourceChange struct {
	ResourceAddress string                 `json:"resource_address"`
	Action          string                 `json:"action"` // create, update, delete
	Before          map[string]interface{} `json:"before"`
	After           map[string]interface{} `json:"after"`
}

// PlanSummary represents the summary of a Terraform plan
type PlanSummary struct {
	Add     int `json:"add"`
	Change  int `json:"change"`
	Destroy int `json:"destroy"`
	NoOp    int `json:"no_op"`
}

// ApplySummary represents the summary of a Terraform apply
type ApplySummary struct {
	Added     int `json:"added"`
	Changed   int `json:"changed"`
	Destroyed int `json:"destroyed"`
}

// DestroySummary represents the summary of a Terraform destroy
type DestroySummary struct {
	Destroyed int `json:"destroyed"`
}

// APIRequest represents an API request
type APIRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    interface{}       `json:"body"`
	Timeout time.Duration     `json:"timeout"`
}

// APIResponse represents an API response
type APIResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       interface{}       `json:"body"`
	Duration   time.Duration     `json:"duration"`
}

// ScriptResult represents the result of a script execution
type ScriptResult struct {
	ExitCode int           `json:"exit_code"`
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	Duration time.Duration `json:"duration"`
}

// NotificationRequest represents a notification request
type NotificationRequest struct {
	Type       string                 `json:"type"` // email, slack, webhook, sms
	Recipients []string               `json:"recipients"`
	Subject    string                 `json:"subject"`
	Message    string                 `json:"message"`
	Data       map[string]interface{} `json:"data"`
	Priority   string                 `json:"priority"` // low, medium, high, critical
}

// NewActionExecutor creates a new action executor
func NewActionExecutor(
	terraformExecutor TerraformExecutor,
	apiClient APIClient,
	scriptExecutor ScriptExecutor,
	notificationService NotificationService,
) *ActionExecutorImpl {
	return &ActionExecutorImpl{
		terraformExecutor:   terraformExecutor,
		apiClient:           apiClient,
		scriptExecutor:      scriptExecutor,
		notificationService: notificationService,
	}
}

// ExecuteAction executes an automation action
func (e *ActionExecutorImpl) ExecuteAction(ctx context.Context, action *models.AutomationAction, context map[string]interface{}) (*models.ActionResult, error) {
	startTime := time.Now()

	// Validate action
	if err := e.ValidateAction(ctx, action); err != nil {
		return nil, fmt.Errorf("action validation failed: %w", err)
	}

	// Execute action based on type
	var result *models.ActionResult
	var err error

	switch action.Type {
	case models.ActionTypeTerraform:
		result, err = e.executeTerraformAction(ctx, action, context)
	case models.ActionTypeAPI:
		result, err = e.executeAPIAction(ctx, action, context)
	case models.ActionTypeScript:
		result, err = e.executeScriptAction(ctx, action, context)
	case models.ActionTypeNotification:
		result, err = e.executeNotificationAction(ctx, action, context)
	default:
		return nil, fmt.Errorf("unsupported action type: %s", action.Type)
	}

	if err != nil {
		return nil, err
	}

	// Set execution metadata
	result.ExecutionTime = time.Since(startTime)
	result.Timestamp = time.Now()

	return result, nil
}

// ValidateAction validates an automation action
func (e *ActionExecutorImpl) ValidateAction(ctx context.Context, action *models.AutomationAction) error {
	if action.Type == "" {
		return fmt.Errorf("action type is required")
	}

	if action.Name == "" {
		return fmt.Errorf("action name is required")
	}

	// Validate action type specific requirements
	switch action.Type {
	case models.ActionTypeTerraform:
		return e.validateTerraformAction(action)
	case models.ActionTypeAPI:
		return e.validateAPIAction(action)
	case models.ActionTypeScript:
		return e.validateScriptAction(action)
	case models.ActionTypeNotification:
		return e.validateNotificationAction(action)
	default:
		return fmt.Errorf("unsupported action type: %s", action.Type)
	}
}

// GetSupportedActionTypes returns the list of supported action types
func (e *ActionExecutorImpl) GetSupportedActionTypes() []models.ActionType {
	return []models.ActionType{
		models.ActionTypeTerraform,
		models.ActionTypeAPI,
		models.ActionTypeScript,
		models.ActionTypeNotification,
	}
}

// executeTerraformAction executes a Terraform action
func (e *ActionExecutorImpl) executeTerraformAction(ctx context.Context, action *models.AutomationAction, context map[string]interface{}) (*models.ActionResult, error) {
	// Parse Terraform configuration
	config, err := e.parseTerraformConfig(action.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Terraform configuration: %w", err)
	}

	// Execute Terraform command based on operation
	var result interface{}
	var exitCode int
	var stdout, stderr string

	switch config.Operation {
	case "plan":
		planResult, err := e.terraformExecutor.Plan(ctx, config.ConfigPath, config.Variables)
		if err != nil {
			return nil, fmt.Errorf("Terraform plan failed: %w", err)
		}
		result = planResult
		exitCode = planResult.ExitCode
		stdout = planResult.Stdout
		stderr = planResult.Stderr
	case "apply":
		applyResult, err := e.terraformExecutor.Apply(ctx, config.ConfigPath, config.Variables)
		if err != nil {
			return nil, fmt.Errorf("Terraform apply failed: %w", err)
		}
		result = applyResult
		exitCode = applyResult.ExitCode
		stdout = applyResult.Stdout
		stderr = applyResult.Stderr
	case "destroy":
		destroyResult, err := e.terraformExecutor.Destroy(ctx, config.ConfigPath, config.Variables)
		if err != nil {
			return nil, fmt.Errorf("Terraform destroy failed: %w", err)
		}
		result = destroyResult
		exitCode = destroyResult.ExitCode
		stdout = destroyResult.Stdout
		stderr = destroyResult.Stderr
	default:
		// Execute custom command
		cmdResult, err := e.terraformExecutor.ExecuteCommand(ctx, config.Operation, config.Args, config.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("Terraform command failed: %w", err)
		}
		result = cmdResult
		exitCode = cmdResult.ExitCode
		stdout = cmdResult.Stdout
		stderr = cmdResult.Stderr
	}

	// Create action result
	actionResult := &models.ActionResult{
		ActionID: action.ID,
		Status:   models.ActionStatusCompleted,
		Output:   models.JSONB(result.(map[string]interface{})),
		Metadata: map[string]interface{}{
			"exit_code": exitCode,
			"stdout":    stdout,
			"stderr":    stderr,
		},
	}

	// Set status based on exit code
	if exitCode != 0 {
		actionResult.Status = models.ActionStatusFailed
		actionResult.Error = stderr
	}

	return actionResult, nil
}

// executeAPIAction executes an API action
func (e *ActionExecutorImpl) executeAPIAction(ctx context.Context, action *models.AutomationAction, context map[string]interface{}) (*models.ActionResult, error) {
	// Parse API configuration
	config, err := e.parseAPIConfig(action.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to parse API configuration: %w", err)
	}

	// Create API request
	req := &APIRequest{
		Method:  config.Method,
		URL:     config.URL,
		Headers: config.Headers,
		Body:    config.Body,
		Timeout: config.Timeout,
	}

	// Make API request
	resp, err := e.apiClient.MakeRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	// Create action result
	actionResult := &models.ActionResult{
		ActionID: action.ID,
		Status:   models.ActionStatusCompleted,
		Output: models.JSONB(map[string]interface{}{
			"status_code": resp.StatusCode,
			"headers":     resp.Headers,
			"body":        resp.Body,
			"duration":    resp.Duration,
		}),
		Metadata: map[string]interface{}{
			"status_code": resp.StatusCode,
			"headers":     resp.Headers,
			"duration":    resp.Duration,
		},
	}

	// Set status based on response code
	if resp.StatusCode >= 400 {
		actionResult.Status = models.ActionStatusFailed
		actionResult.Error = fmt.Sprintf("API request failed with status code: %d", resp.StatusCode)
	}

	return actionResult, nil
}

// executeScriptAction executes a script action
func (e *ActionExecutorImpl) executeScriptAction(ctx context.Context, action *models.AutomationAction, context map[string]interface{}) (*models.ActionResult, error) {
	// Parse script configuration
	config, err := e.parseScriptConfig(action.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to parse script configuration: %w", err)
	}

	// Execute script
	result, err := e.scriptExecutor.ExecuteScript(ctx, config.Script, config.Args, config.WorkingDir)
	if err != nil {
		return nil, fmt.Errorf("script execution failed: %w", err)
	}

	// Create action result
	actionResult := &models.ActionResult{
		ActionID: action.ID,
		Status:   models.ActionStatusCompleted,
		Output: models.JSONB(map[string]interface{}{
			"exit_code": result.ExitCode,
			"stdout":    result.Stdout,
			"stderr":    result.Stderr,
			"duration":  result.Duration,
		}),
		Metadata: map[string]interface{}{
			"exit_code": result.ExitCode,
			"stdout":    result.Stdout,
			"stderr":    result.Stderr,
			"duration":  result.Duration,
		},
	}

	// Set status based on exit code
	if result.ExitCode != 0 {
		actionResult.Status = models.ActionStatusFailed
		actionResult.Error = result.Stderr
	}

	return actionResult, nil
}

// executeNotificationAction executes a notification action
func (e *ActionExecutorImpl) executeNotificationAction(ctx context.Context, action *models.AutomationAction, context map[string]interface{}) (*models.ActionResult, error) {
	// Parse notification configuration
	config, err := e.parseNotificationConfig(action.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to parse notification configuration: %w", err)
	}

	// Create notification request
	req := &NotificationRequest{
		Type:       config.Type,
		Recipients: config.Recipients,
		Subject:    config.Subject,
		Message:    config.Message,
		Data:       config.Data,
		Priority:   config.Priority,
	}

	// Send notification
	err = e.notificationService.SendNotification(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("notification failed: %w", err)
	}

	// Create action result
	actionResult := &models.ActionResult{
		ActionID: action.ID,
		Status:   models.ActionStatusCompleted,
		Output: models.JSONB(map[string]interface{}{
			"type":       config.Type,
			"recipients": config.Recipients,
			"sent_at":    time.Now(),
		}),
		Metadata: map[string]interface{}{
			"notification_type": config.Type,
			"recipient_count":   len(config.Recipients),
		},
	}

	return actionResult, nil
}

// validateTerraformAction validates a Terraform action
func (e *ActionExecutorImpl) validateTerraformAction(action *models.AutomationAction) error {
	config, err := e.parseTerraformConfig(action.Configuration)
	if err != nil {
		return fmt.Errorf("invalid Terraform configuration: %w", err)
	}

	if config.Operation == "" {
		return fmt.Errorf("Terraform operation is required")
	}

	if config.ConfigPath == "" {
		return fmt.Errorf("Terraform config path is required")
	}

	return nil
}

// validateAPIAction validates an API action
func (e *ActionExecutorImpl) validateAPIAction(action *models.AutomationAction) error {
	config, err := e.parseAPIConfig(action.Configuration)
	if err != nil {
		return fmt.Errorf("invalid API configuration: %w", err)
	}

	if config.Method == "" {
		return fmt.Errorf("HTTP method is required")
	}

	if config.URL == "" {
		return fmt.Errorf("URL is required")
	}

	// Validate request
	req := &APIRequest{
		Method:  config.Method,
		URL:     config.URL,
		Headers: config.Headers,
		Body:    config.Body,
		Timeout: config.Timeout,
	}

	return e.apiClient.ValidateRequest(req)
}

// validateScriptAction validates a script action
func (e *ActionExecutorImpl) validateScriptAction(action *models.AutomationAction) error {
	config, err := e.parseScriptConfig(action.Configuration)
	if err != nil {
		return fmt.Errorf("invalid script configuration: %w", err)
	}

	if config.Script == "" {
		return fmt.Errorf("script is required")
	}

	return e.scriptExecutor.ValidateScript(config.Script)
}

// validateNotificationAction validates a notification action
func (e *ActionExecutorImpl) validateNotificationAction(action *models.AutomationAction) error {
	config, err := e.parseNotificationConfig(action.Configuration)
	if err != nil {
		return fmt.Errorf("invalid notification configuration: %w", err)
	}

	if config.Type == "" {
		return fmt.Errorf("notification type is required")
	}

	if len(config.Recipients) == 0 {
		return fmt.Errorf("recipients are required")
	}

	if config.Message == "" {
		return fmt.Errorf("message is required")
	}

	// Validate notification
	req := &NotificationRequest{
		Type:       config.Type,
		Recipients: config.Recipients,
		Subject:    config.Subject,
		Message:    config.Message,
		Data:       config.Data,
		Priority:   config.Priority,
	}

	return e.notificationService.ValidateNotification(req)
}

// Configuration parsing structures
type TerraformConfig struct {
	Operation  string            `json:"operation"`
	ConfigPath string            `json:"config_path"`
	Variables  map[string]string `json:"variables"`
	Args       []string          `json:"args"`
}

type APIConfig struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    interface{}       `json:"body"`
	Timeout time.Duration     `json:"timeout"`
}

type ScriptConfig struct {
	Script     string   `json:"script"`
	Args       []string `json:"args"`
	WorkingDir string   `json:"working_dir"`
}

type NotificationConfig struct {
	Type       string                 `json:"type"`
	Recipients []string               `json:"recipients"`
	Subject    string                 `json:"subject"`
	Message    string                 `json:"message"`
	Data       map[string]interface{} `json:"data"`
	Priority   string                 `json:"priority"`
}

// parseTerraformConfig parses Terraform configuration from JSONB
func (e *ActionExecutorImpl) parseTerraformConfig(config models.JSONB) (*TerraformConfig, error) {
	var terraformConfig TerraformConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &terraformConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Terraform config: %w", err)
	}
	return &terraformConfig, nil
}

// parseAPIConfig parses API configuration from JSONB
func (e *ActionExecutorImpl) parseAPIConfig(config models.JSONB) (*APIConfig, error) {
	var apiConfig APIConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &apiConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal API config: %w", err)
	}
	return &apiConfig, nil
}

// parseScriptConfig parses script configuration from JSONB
func (e *ActionExecutorImpl) parseScriptConfig(config models.JSONB) (*ScriptConfig, error) {
	var scriptConfig ScriptConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &scriptConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal script config: %w", err)
	}
	return &scriptConfig, nil
}

// parseNotificationConfig parses notification configuration from JSONB
func (e *ActionExecutorImpl) parseNotificationConfig(config models.JSONB) (*NotificationConfig, error) {
	var notificationConfig NotificationConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &notificationConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal notification config: %w", err)
	}
	return &notificationConfig, nil
}
