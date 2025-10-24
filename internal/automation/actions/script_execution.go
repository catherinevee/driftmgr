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

// ScriptExecutionAction handles executing scripts and commands
type ScriptExecutionAction struct {
	eventBus *events.EventBus
}

// ScriptExecutionConfig represents the configuration for a script execution action
type ScriptExecutionConfig struct {
	ScriptType    string                 `json:"script_type"`    // type of script (shell, python, powershell, etc.)
	Script        string                 `json:"script"`         // script content
	ScriptPath    string                 `json:"script_path"`    // path to script file
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
	Data          map[string]interface{} `json:"data"`           // additional data
}

// ScriptExecutionResult represents the result of a script execution
type ScriptExecutionResult struct {
	ScriptType string                 `json:"script_type"`
	Command    string                 `json:"command"`
	Args       []string               `json:"args"`
	WorkingDir string                 `json:"working_dir"`
	ExitCode   int                    `json:"exit_code"`
	Stdout     string                 `json:"stdout"`
	Stderr     string                 `json:"stderr"`
	Duration   time.Duration          `json:"duration"`
	Success    bool                   `json:"success"`
	Retries    int                    `json:"retries"`
	Data       map[string]interface{} `json:"data"`
}

// NewScriptExecutionAction creates a new script execution action handler
func NewScriptExecutionAction(eventBus *events.EventBus) *ScriptExecutionAction {
	return &ScriptExecutionAction{
		eventBus: eventBus,
	}
}

// Execute executes a script execution action
func (sea *ScriptExecutionAction) Execute(ctx context.Context, action *models.AutomationAction, context map[string]interface{}) (*models.ActionResult, error) {
	startTime := time.Now()

	// Parse script execution configuration
	config, err := sea.parseConfig(action.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to parse script execution configuration: %w", err)
	}

	// Validate configuration
	err = sea.validateConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Determine the command to execute
	command, args, err := sea.prepareCommand(config, context)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare command: %w", err)
	}

	// Execute the script/command with retries
	result, err := sea.executeWithRetries(ctx, command, args, config, context)
	if err != nil {
		return nil, fmt.Errorf("failed to execute script: %w", err)
	}

	// Publish automation event
	automationEvent := events.Event{
		Type:      events.EventType("automation.script_executed"),
		Timestamp: time.Now(),
		Source:    "automation_service",
		Data: map[string]interface{}{
			"action_id":   action.ID,
			"action_name": action.Name,
			"script_type": result.ScriptType,
			"command":     result.Command,
			"exit_code":   result.ExitCode,
			"success":     result.Success,
			"duration":    result.Duration,
			"retries":     result.Retries,
			"action_type": "script_execution",
		},
	}

	sea.eventBus.Publish(automationEvent)

	// Create action result
	actionResult := &models.ActionResult{
		ActionID:      action.ID,
		Status:        models.ActionStatusCompleted,
		ExecutionTime: time.Since(startTime),
		Timestamp:     time.Now(),
		Output: models.JSONB(map[string]interface{}{
			"script_type": result.ScriptType,
			"command":     result.Command,
			"args":        result.Args,
			"working_dir": result.WorkingDir,
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

// Validate validates a script execution action
func (sea *ScriptExecutionAction) Validate(action *models.AutomationAction) error {
	if action.Type != models.ActionTypeScript {
		return fmt.Errorf("invalid action type: expected %s, got %s", models.ActionTypeScript, action.Type)
	}

	config, err := sea.parseConfig(action.Configuration)
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return sea.validateConfig(config)
}

// parseConfig parses the script execution configuration from JSONB
func (sea *ScriptExecutionAction) parseConfig(config models.JSONB) (*ScriptExecutionConfig, error) {
	var scriptExecutionConfig ScriptExecutionConfig

	// JSONB is already a map[string]interface{}
	configMap := config

	// Parse script type
	if scriptTypeVal, ok := configMap["script_type"].(string); ok {
		scriptExecutionConfig.ScriptType = scriptTypeVal
	} else {
		scriptExecutionConfig.ScriptType = "shell" // default
	}

	// Parse script
	if scriptVal, ok := configMap["script"].(string); ok {
		scriptExecutionConfig.Script = scriptVal
	}

	// Parse script path
	if scriptPathVal, ok := configMap["script_path"].(string); ok {
		scriptExecutionConfig.ScriptPath = scriptPathVal
	}

	// Parse command
	if commandVal, ok := configMap["command"].(string); ok {
		scriptExecutionConfig.Command = commandVal
	}

	// Parse args
	if argsVal, ok := configMap["args"].([]interface{}); ok {
		scriptExecutionConfig.Args = make([]string, len(argsVal))
		for i, arg := range argsVal {
			if argStr, ok := arg.(string); ok {
				scriptExecutionConfig.Args[i] = argStr
			}
		}
	}

	// Parse working directory
	if workingDirVal, ok := configMap["working_dir"].(string); ok {
		scriptExecutionConfig.WorkingDir = workingDirVal
	}

	// Parse environment
	if environmentVal, ok := configMap["environment"].(map[string]interface{}); ok {
		scriptExecutionConfig.Environment = make(map[string]string)
		for key, value := range environmentVal {
			if valueStr, ok := value.(string); ok {
				scriptExecutionConfig.Environment[key] = valueStr
			}
		}
	}

	// Parse timeout
	if timeoutVal, ok := configMap["timeout"].(float64); ok {
		scriptExecutionConfig.Timeout = time.Duration(timeoutVal) * time.Second
	} else {
		scriptExecutionConfig.Timeout = 30 * time.Second // default timeout
	}

	// Parse retries
	if retriesVal, ok := configMap["retries"].(float64); ok {
		scriptExecutionConfig.Retries = int(retriesVal)
	}

	// Parse retry delay
	if retryDelayVal, ok := configMap["retry_delay"].(float64); ok {
		scriptExecutionConfig.RetryDelay = time.Duration(retryDelayVal) * time.Second
	} else {
		scriptExecutionConfig.RetryDelay = 5 * time.Second // default retry delay
	}

	// Parse capture output
	if captureOutputVal, ok := configMap["capture_output"].(bool); ok {
		scriptExecutionConfig.CaptureOutput = captureOutputVal
	} else {
		scriptExecutionConfig.CaptureOutput = true // default to capturing output
	}

	// Parse validate exit
	if validateExitVal, ok := configMap["validate_exit"].(bool); ok {
		scriptExecutionConfig.ValidateExit = validateExitVal
	} else {
		scriptExecutionConfig.ValidateExit = true // default to validating exit code
	}

	// Parse expected exit
	if expectedExitVal, ok := configMap["expected_exit"].(float64); ok {
		scriptExecutionConfig.ExpectedExit = int(expectedExitVal)
	} else {
		scriptExecutionConfig.ExpectedExit = 0 // default expected exit code
	}

	// Parse data
	if dataVal, ok := configMap["data"].(map[string]interface{}); ok {
		scriptExecutionConfig.Data = dataVal
	}

	return &scriptExecutionConfig, nil
}

// validateConfig validates the script execution configuration
func (sea *ScriptExecutionAction) validateConfig(config *ScriptExecutionConfig) error {
	// Must have either script content, script path, or command
	if config.Script == "" && config.ScriptPath == "" && config.Command == "" {
		return fmt.Errorf("either script, script_path, or command is required")
	}

	// Validate script type
	validScriptTypes := []string{"shell", "bash", "python", "powershell", "cmd", "custom"}
	validScriptType := false
	for _, scriptType := range validScriptTypes {
		if config.ScriptType == scriptType {
			validScriptType = true
			break
		}
	}
	if !validScriptType {
		return fmt.Errorf("invalid script type: %s (must be one of: %v)", config.ScriptType, validScriptTypes)
	}

	// If script path is provided, check if file exists
	if config.ScriptPath != "" {
		if _, err := os.Stat(config.ScriptPath); os.IsNotExist(err) {
			return fmt.Errorf("script file does not exist: %s", config.ScriptPath)
		}
	}

	return nil
}

// prepareCommand prepares the command and arguments to execute
func (sea *ScriptExecutionAction) prepareCommand(config *ScriptExecutionConfig, context map[string]interface{}) (string, []string, error) {
	var command string
	var args []string

	if config.Command != "" {
		// Use provided command
		command = config.Command
		args = config.Args
	} else if config.ScriptPath != "" {
		// Execute script file
		command = config.ScriptPath
		args = config.Args
	} else if config.Script != "" {
		// Execute inline script
		switch config.ScriptType {
		case "shell", "bash":
			command = "bash"
			args = []string{"-c", config.Script}
		case "python":
			command = "python"
			args = []string{"-c", config.Script}
		case "powershell":
			command = "powershell"
			args = []string{"-Command", config.Script}
		case "cmd":
			command = "cmd"
			args = []string{"/c", config.Script}
		default:
			return "", nil, fmt.Errorf("unsupported script type for inline execution: %s", config.ScriptType)
		}
	}

	// Process context variables in command and args
	command = sea.processContextVariables(command, context)
	for i, arg := range args {
		args[i] = sea.processContextVariables(arg, context)
	}

	return command, args, nil
}

// processContextVariables processes context variables in a string
func (sea *ScriptExecutionAction) processContextVariables(str string, context map[string]interface{}) string {
	result := str
	for key, value := range context {
		placeholder := fmt.Sprintf("{{%s}}", key)
		valueStr := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, valueStr)
	}
	return result
}

// executeWithRetries executes the command with retries
func (sea *ScriptExecutionAction) executeWithRetries(ctx context.Context, command string, args []string, config *ScriptExecutionConfig, actionContext map[string]interface{}) (*ScriptExecutionResult, error) {
	var lastErr error
	var result *ScriptExecutionResult

	for attempt := 0; attempt <= config.Retries; attempt++ {
		result, lastErr = sea.executeCommand(ctx, command, args, config, actionContext)
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
		result = &ScriptExecutionResult{
			ScriptType: config.ScriptType,
			Command:    command,
			Args:       args,
			Success:    false,
			Retries:    config.Retries,
		}
	}
	result.Retries = config.Retries

	return result, fmt.Errorf("script execution failed after %d retries: %w", config.Retries, lastErr)
}

// executeCommand executes a single command
func (sea *ScriptExecutionAction) executeCommand(ctx context.Context, command string, args []string, config *ScriptExecutionConfig, actionContext map[string]interface{}) (*ScriptExecutionResult, error) {
	startTime := time.Now()

	// Create command context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(cmdCtx, command, args...)

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
	result := &ScriptExecutionResult{
		ScriptType: config.ScriptType,
		Command:    command,
		Args:       args,
		WorkingDir: config.WorkingDir,
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
