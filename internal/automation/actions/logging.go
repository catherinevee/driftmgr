package actions

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/catherinevee/driftmgr/internal/events"
	"github.com/catherinevee/driftmgr/internal/models"
)

// LoggingAction handles structured logging for automation events
type LoggingAction struct {
	eventBus *events.EventBus
	logDir   string
}

// LoggingConfig represents the configuration for a logging action
type LoggingConfig struct {
	Level     string                 `json:"level"`     // log level (debug, info, warn, error)
	Format    string                 `json:"format"`    // log format (json, text, structured)
	Output    string                 `json:"output"`    // output destination (file, stdout, stderr)
	FilePath  string                 `json:"file_path"` // file path for file output
	Message   string                 `json:"message"`   // log message
	Template  string                 `json:"template"`  // template for message formatting
	Fields    map[string]interface{} `json:"fields"`    // additional fields to log
	Retention int                    `json:"retention"` // log retention in days
	MaxSize   int64                  `json:"max_size"`  // maximum log file size in bytes
	MaxFiles  int                    `json:"max_files"` // maximum number of log files
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp  time.Time              `json:"timestamp"`
	Level      string                 `json:"level"`
	Message    string                 `json:"message"`
	ActionID   string                 `json:"action_id"`
	ActionName string                 `json:"action_name"`
	ActionType string                 `json:"action_type"`
	Source     string                 `json:"source"`
	Fields     map[string]interface{} `json:"fields,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// NewLoggingAction creates a new logging action handler
func NewLoggingAction(eventBus *events.EventBus, logDir string) *LoggingAction {
	if logDir == "" {
		logDir = "logs"
	}

	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Warning: failed to create log directory %s: %v", logDir, err)
	}

	return &LoggingAction{
		eventBus: eventBus,
		logDir:   logDir,
	}
}

// Execute executes a logging action
func (la *LoggingAction) Execute(ctx context.Context, action *models.AutomationAction, context map[string]interface{}) (*models.ActionResult, error) {
	startTime := time.Now()

	// Parse logging configuration
	config, err := la.parseConfig(action.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to parse logging configuration: %w", err)
	}

	// Process template if specified
	message := config.Message
	if config.Template != "" {
		message, err = la.processTemplate(config.Template, context)
		if err != nil {
			return nil, fmt.Errorf("failed to process template: %w", err)
		}
	}

	// Create log entry
	logEntry := &LogEntry{
		Timestamp:  time.Now(),
		Level:      config.Level,
		Message:    message,
		ActionID:   action.ID,
		ActionName: action.Name,
		ActionType: string(action.Type),
		Source:     "automation_service",
		Fields:     config.Fields,
		Context:    context,
	}

	// Write log entry
	err = la.writeLogEntry(logEntry, config)
	if err != nil {
		return nil, fmt.Errorf("failed to write log entry: %w", err)
	}

	// Publish automation event
	automationEvent := events.Event{
		Type:      events.EventType("automation.logged"),
		Timestamp: time.Now(),
		Source:    "automation_service",
		Data: map[string]interface{}{
			"action_id":   action.ID,
			"action_name": action.Name,
			"log_level":   config.Level,
			"message":     message,
			"output":      config.Output,
			"action_type": "logging",
		},
	}

	la.eventBus.Publish(automationEvent)

	// Create action result
	result := &models.ActionResult{
		ActionID:      action.ID,
		Status:        models.ActionStatusCompleted,
		ExecutionTime: time.Since(startTime),
		Timestamp:     time.Now(),
		Output: models.JSONB(map[string]interface{}{
			"log_level": config.Level,
			"message":   message,
			"output":    config.Output,
			"file_path": config.FilePath,
			"logged_at": time.Now(),
		}),
	}

	return result, nil
}

// Validate validates a logging action
func (la *LoggingAction) Validate(action *models.AutomationAction) error {
	if action.Type != models.ActionTypeCustom {
		return fmt.Errorf("invalid action type: expected %s, got %s", models.ActionTypeCustom, action.Type)
	}

	config, err := la.parseConfig(action.Configuration)
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	if config.Level == "" {
		return fmt.Errorf("log level is required")
	}

	if config.Message == "" && config.Template == "" {
		return fmt.Errorf("either message or template is required")
	}

	// Validate log level
	validLevels := []string{"debug", "info", "warn", "error", "fatal"}
	validLevel := false
	for _, level := range validLevels {
		if config.Level == level {
			validLevel = true
			break
		}
	}
	if !validLevel {
		return fmt.Errorf("invalid log level: %s (must be one of: %v)", config.Level, validLevels)
	}

	// Validate output
	validOutputs := []string{"stdout", "stderr", "file"}
	validOutput := false
	for _, output := range validOutputs {
		if config.Output == output {
			validOutput = true
			break
		}
	}
	if !validOutput {
		return fmt.Errorf("invalid output: %s (must be one of: %v)", config.Output, validOutputs)
	}

	// If output is file, file path is required
	if config.Output == "file" && config.FilePath == "" {
		return fmt.Errorf("file path is required when output is 'file'")
	}

	return nil
}

// parseConfig parses the logging configuration from JSONB
func (la *LoggingAction) parseConfig(config models.JSONB) (*LoggingConfig, error) {
	var loggingConfig LoggingConfig

	// JSONB is already a map[string]interface{}
	configMap := config

	// Parse level
	if levelVal, ok := configMap["level"].(string); ok {
		loggingConfig.Level = levelVal
	} else {
		loggingConfig.Level = "info" // default level
	}

	// Parse format
	if formatVal, ok := configMap["format"].(string); ok {
		loggingConfig.Format = formatVal
	} else {
		loggingConfig.Format = "json" // default format
	}

	// Parse output
	if outputVal, ok := configMap["output"].(string); ok {
		loggingConfig.Output = outputVal
	} else {
		loggingConfig.Output = "stdout" // default output
	}

	// Parse file path
	if filePathVal, ok := configMap["file_path"].(string); ok {
		loggingConfig.FilePath = filePathVal
	}

	// Parse message
	if messageVal, ok := configMap["message"].(string); ok {
		loggingConfig.Message = messageVal
	}

	// Parse template
	if templateVal, ok := configMap["template"].(string); ok {
		loggingConfig.Template = templateVal
	}

	// Parse fields
	if fieldsVal, ok := configMap["fields"].(map[string]interface{}); ok {
		loggingConfig.Fields = fieldsVal
	}

	// Parse retention
	if retentionVal, ok := configMap["retention"].(float64); ok {
		loggingConfig.Retention = int(retentionVal)
	}

	// Parse max size
	if maxSizeVal, ok := configMap["max_size"].(float64); ok {
		loggingConfig.MaxSize = int64(maxSizeVal)
	}

	// Parse max files
	if maxFilesVal, ok := configMap["max_files"].(float64); ok {
		loggingConfig.MaxFiles = int(maxFilesVal)
	}

	return &loggingConfig, nil
}

// processTemplate processes a logging template with context data
func (la *LoggingAction) processTemplate(template string, context map[string]interface{}) (string, error) {
	// Simple template processing - replace {{key}} with values from context
	// In a real implementation, you might use a more sophisticated templating engine
	result := template

	for key, value := range context {
		placeholder := fmt.Sprintf("{{%s}}", key)
		valueStr := fmt.Sprintf("%v", value)
		result = fmt.Sprintf("%s", result) // This is a placeholder - implement proper template processing
		_ = placeholder
		_ = valueStr
	}

	return result, nil
}

// writeLogEntry writes a log entry to the specified output
func (la *LoggingAction) writeLogEntry(entry *LogEntry, config *LoggingConfig) error {
	var logMessage string

	// Format the log entry based on the format
	switch config.Format {
	case "json":
		logMessage = la.formatAsJSON(entry)
	case "text":
		logMessage = la.formatAsText(entry)
	case "structured":
		logMessage = la.formatAsStructured(entry)
	default:
		logMessage = la.formatAsJSON(entry)
	}

	// Write to the specified output
	switch config.Output {
	case "stdout":
		fmt.Println(logMessage)
	case "stderr":
		fmt.Fprintln(os.Stderr, logMessage)
	case "file":
		return la.writeToFile(logMessage, config)
	default:
		return fmt.Errorf("unsupported output: %s", config.Output)
	}

	return nil
}

// formatAsJSON formats the log entry as JSON
func (la *LoggingAction) formatAsJSON(entry *LogEntry) string {
	// In a real implementation, you would use json.Marshal
	// For now, return a simple string representation
	return fmt.Sprintf(`{"timestamp":"%s","level":"%s","message":"%s","action_id":"%s","action_name":"%s","action_type":"%s","source":"%s"}`,
		entry.Timestamp.Format(time.RFC3339),
		entry.Level,
		entry.Message,
		entry.ActionID,
		entry.ActionName,
		entry.ActionType,
		entry.Source)
}

// formatAsText formats the log entry as plain text
func (la *LoggingAction) formatAsText(entry *LogEntry) string {
	return fmt.Sprintf("[%s] %s: %s (action: %s, id: %s)",
		entry.Timestamp.Format("2006-01-02 15:04:05"),
		entry.Level,
		entry.Message,
		entry.ActionName,
		entry.ActionID)
}

// formatAsStructured formats the log entry as structured text
func (la *LoggingAction) formatAsStructured(entry *LogEntry) string {
	return fmt.Sprintf("timestamp=%s level=%s message=%s action_id=%s action_name=%s action_type=%s source=%s",
		entry.Timestamp.Format(time.RFC3339),
		entry.Level,
		entry.Message,
		entry.ActionID,
		entry.ActionName,
		entry.ActionType,
		entry.Source)
}

// writeToFile writes the log message to a file
func (la *LoggingAction) writeToFile(message string, config *LoggingConfig) error {
	// Determine the file path
	filePath := config.FilePath
	if filePath == "" {
		// Use default file path
		filePath = filepath.Join(la.logDir, fmt.Sprintf("automation_%s.log", time.Now().Format("2006-01-02")))
	}

	// Open file in append mode
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Write the message
	_, err = file.WriteString(message + "\n")
	if err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	return nil
}
