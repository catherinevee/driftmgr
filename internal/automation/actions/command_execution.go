package actions

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/events"
	"github.com/catherinevee/driftmgr/internal/models"
)

// CommandExecutionAction handles executing system commands
type CommandExecutionAction struct {
	eventBus *events.EventBus
}

// CommandExecutionConfig represents the configuration for a command execution action
type CommandExecutionConfig struct {
	Command       string                 `json:"command"`        // command to execute
	Args          []string               `json:"args"`           // command arguments
	WorkingDir    string                 `json:"working_dir"`    // working directory
	Environment   map[string]string      `json:"environment"`    // environment variables
	Timeout       time.Duration          `json:"timeout"`        // execution timeout
	Retries       int                    `json:"retries"`        // number of retries
	RetryDelay    time.Duration          `json:"retry_delay"`    // delay between retries
	CaptureOutput bool                   `json:"capture_output"` // whether to capture output
	ValidateExit  bool                   `json:"validate_exit"`  // whether to validate exit code
	ExpectedExit  int                    `json:"expected_exit"`  // expected exit code
	Shell         string                 `json:"shell"`          // shell to use (bash, cmd, powershell)
	Data          map[string]interface{} `json:"data"`           // additional data
}

// CommandExecutionResult represents the result of a command execution
type CommandExecutionResult struct {
	Command    string                 `json:"command"`
	Args       []string               `json:"args"`
	WorkingDir string                 `json:"working_dir"`
	Shell      string                 `json:"shell"`
	ExitCode   int                    `json:"exit_code"`
	Stdout     string                 `json:"stdout"`
	Stderr     string                 `json:"stderr"`
	Duration   time.Duration          `json:"duration"`
	Success    bool                   `json:"success"`
	Retries    int                    `json:"retries"`
	Data       map[string]interface{} `json:"data"`
}

// NewCommandExecutionAction creates a new command execution action handler
func NewCommandExecutionAction(eventBus *events.EventBus) *CommandExecutionAction {
	return &CommandExecutionAction{
		eventBus: eventBus,
	}
}

// Execute executes a command execution action
func (cea *CommandExecutionAction) Execute(ctx context.Context, action *models.AutomationAction, context map[string]interface{}) (*models.ActionResult, error) {
	startTime := time.Now()

	// Parse command execution configuration
	config, err := cea.parseConfig(action.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to parse command execution configuration: %w", err)
	}

	// Validate configuration
	err = cea.validateConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Process context variables in command and args
	command := cea.processContextVariables(config.Command, context)
	args := make([]string, len(config.Args))
	for i, arg := range config.Args {
		args[i] = cea.processContextVariables(arg, context)
	}

	// Execute the command with retries
	result, err := cea.executeWithRetries(ctx, command, args, config, context)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	// Publish automation event
	automationEvent := events.Event{
		Type:      events.EventType("automation.command_executed"),
		Timestamp: time.Now(),
		Source:    "automation_service",
		Data: map[string]interface{}{
			"action_id":   action.ID,
			"action_name": action.Name,
			"command":     result.Command,
			"exit_code":   result.ExitCode,
			"success":     result.Success,
			"duration":    result.Duration,
			"retries":     result.Retries,
			"action_type": "command_execution",
		},
	}

	cea.eventBus.Publish(automationEvent)

	// Create action result
	actionResult := &models.ActionResult{
		ActionID:      action.ID,
		Status:        models.ActionStatusCompleted,
		ExecutionTime: time.Since(startTime),
		Timestamp:     time.Now(),
		Output: models.JSONB(map[string]interface{}{
			"command":     result.Command,
			"args":        result.Args,
			"working_dir": result.WorkingDir,
			"shell":       result.Shell,
			"exit_code":   result.ExitCode,
			"stdout":      result.Stdout,
			"stderr":      result.Stderr,
			"duration":    result.Duration,
			"success":     result.Success,
			"retries":     result.Retries,
			"data":        result.Data,
		}),
	}

	return actionResult, nil
}

// Validate validates a command execution action
func (cea *CommandExecutionAction) Validate(action *models.AutomationAction) error {
	if action.Type != models.ActionTypeCustom {
		return fmt.Errorf("invalid action type: expected %s, got %s", models.ActionTypeCustom, action.Type)
	}

	config, err := cea.parseConfig(action.Configuration)
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return cea.validateConfig(config)
}

// parseConfig parses the command execution configuration from JSONB
func (cea *CommandExecutionAction) parseConfig(config models.JSONB) (*CommandExecutionConfig, error) {
	var commandExecutionConfig CommandExecutionConfig

	// JSONB is already a map[string]interface{}
	configMap := config

	// Parse command
	if commandVal, ok := configMap["command"].(string); ok {
		commandExecutionConfig.Command = commandVal
	}

	// Parse args
	if argsVal, ok := configMap["args"].([]interface{}); ok {
		commandExecutionConfig.Args = make([]string, len(argsVal))
		for i, arg := range argsVal {
			if argStr, ok := arg.(string); ok {
				commandExecutionConfig.Args[i] = argStr
			}
		}
	}

	// Parse working directory
	if workingDirVal, ok := configMap["working_dir"].(string); ok {
		commandExecutionConfig.WorkingDir = workingDirVal
	}

	// Parse environment
	if environmentVal, ok := configMap["environment"].(map[string]interface{}); ok {
		commandExecutionConfig.Environment = make(map[string]string)
		for key, value := range environmentVal {
			if valueStr, ok := value.(string); ok {
				commandExecutionConfig.Environment[key] = valueStr
			}
		}
	}

	// Parse timeout
	if timeoutVal, ok := configMap["timeout"].(float64); ok {
		commandExecutionConfig.Timeout = time.Duration(timeoutVal) * time.Second
	} else {
		commandExecutionConfig.Timeout = 30 * time.Second // default timeout
	}

	// Parse retries
	if retriesVal, ok := configMap["retries"].(float64); ok {
		commandExecutionConfig.Retries = int(retriesVal)
	}

	// Parse retry delay
	if retryDelayVal, ok := configMap["retry_delay"].(float64); ok {
		commandExecutionConfig.RetryDelay = time.Duration(retryDelayVal) * time.Second
	} else {
		commandExecutionConfig.RetryDelay = 5 * time.Second // default retry delay
	}

	// Parse capture output
	if captureOutputVal, ok := configMap["capture_output"].(bool); ok {
		commandExecutionConfig.CaptureOutput = captureOutputVal
	} else {
		commandExecutionConfig.CaptureOutput = true // default to capturing output
	}

	// Parse validate exit
	if validateExitVal, ok := configMap["validate_exit"].(bool); ok {
		commandExecutionConfig.ValidateExit = validateExitVal
	} else {
		commandExecutionConfig.ValidateExit = true // default to validating exit code
	}

	// Parse expected exit
	if expectedExitVal, ok := configMap["expected_exit"].(float64); ok {
		commandExecutionConfig.ExpectedExit = int(expectedExitVal)
	} else {
		commandExecutionConfig.ExpectedExit = 0 // default expected exit code
	}

	// Parse shell
	if shellVal, ok := configMap["shell"].(string); ok {
		commandExecutionConfig.Shell = shellVal
	} else {
		commandExecutionConfig.Shell = "bash" // default shell
	}

	// Parse data
	if dataVal, ok := configMap["data"].(map[string]interface{}); ok {
		commandExecutionConfig.Data = dataVal
	}

	return &commandExecutionConfig, nil
}

// validateConfig validates the command execution configuration
func (cea *CommandExecutionAction) validateConfig(config *CommandExecutionConfig) error {
	if config.Command == "" {
		return fmt.Errorf("command is required")
	}

	// Validate shell
	validShells := []string{"bash", "sh", "cmd", "powershell", "zsh", "fish"}
	validShell := false
	for _, shell := range validShells {
		if config.Shell == shell {
			validShell = true
			break
		}
	}
	if !validShell {
		return fmt.Errorf("invalid shell: %s (must be one of: %v)", config.Shell, validShells)
	}

	return nil
}

// processContextVariables processes context variables in a string
func (cea *CommandExecutionAction) processContextVariables(str string, context map[string]interface{}) string {
	result := str
	for key, value := range context {
		placeholder := fmt.Sprintf("{{%s}}", key)
		valueStr := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, valueStr)
	}
	return result
}

// executeWithRetries executes the command with retries
func (cea *CommandExecutionAction) executeWithRetries(ctx context.Context, command string, args []string, config *CommandExecutionConfig, actionContext map[string]interface{}) (*CommandExecutionResult, error) {
	var lastErr error
	var result *CommandExecutionResult

	for attempt := 0; attempt <= config.Retries; attempt++ {
		result, lastErr = cea.executeCommand(ctx, command, args, config, actionContext)
		if lastErr == nil && result.Success {
			result.Retries = attempt
			return result, nil
		}

		// If this is not the last attempt, wait before retrying
		if attempt < config.Retries {
			time.Sleep(config.RetryDelay)
		}
	}

	// All retries failed
	if result == nil {
		result = &CommandExecutionResult{
			Command: command,
			Args:    args,
			Shell:   config.Shell,
			Success: false,
			Retries: config.Retries,
		}
	}
	result.Retries = config.Retries

	return result, fmt.Errorf("command execution failed after %d retries: %w", config.Retries, lastErr)
}

// executeCommand executes a single command
func (cea *CommandExecutionAction) executeCommand(ctx context.Context, command string, args []string, config *CommandExecutionConfig, actionContext map[string]interface{}) (*CommandExecutionResult, error) {
	startTime := time.Now()

	// Create command context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	// Determine the actual command to run based on shell
	var actualCommand string
	var actualArgs []string

	switch config.Shell {
	case "bash", "sh", "zsh", "fish":
		// For Unix shells, run the command through the shell
		actualCommand = config.Shell
		actualArgs = []string{"-c", command}
		if len(args) > 0 {
			actualArgs = append(actualArgs, args...)
		}
	case "cmd":
		// For Windows CMD
		actualCommand = "cmd"
		actualArgs = []string{"/c", command}
		if len(args) > 0 {
			actualArgs = append(actualArgs, args...)
		}
	case "powershell":
		// For PowerShell
		actualCommand = "powershell"
		actualArgs = []string{"-Command", command}
		if len(args) > 0 {
			actualArgs = append(actualArgs, args...)
		}
	default:
		// Direct execution
		actualCommand = command
		actualArgs = args
	}

	// Create command
	cmd := exec.CommandContext(cmdCtx, actualCommand, actualArgs...)

	// Set working directory
	if config.WorkingDir != "" {
		cmd.Dir = config.WorkingDir
	}

	// Set environment variables
	if len(config.Environment) > 0 {
		cmd.Env = os.Environ()
		for key, value := range config.Environment {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Execute command
	var stdout, stderr strings.Builder
	if config.CaptureOutput {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	err := cmd.Run()
	duration := time.Since(startTime)

	// Create result
	result := &CommandExecutionResult{
		Command:    command,
		Args:       args,
		WorkingDir: config.WorkingDir,
		Shell:      config.Shell,
		ExitCode:   cmd.ProcessState.ExitCode(),
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		Duration:   duration,
		Data:       config.Data,
	}

	// Determine success
	if err != nil {
		result.Success = false
	} else if config.ValidateExit {
		result.Success = (result.ExitCode == config.ExpectedExit)
	} else {
		result.Success = (result.ExitCode == 0)
	}

	return result, err
}
