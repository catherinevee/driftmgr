package deletion

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// LogLevel represents the logging level
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// DeletionLogger provides comprehensive logging for deletion operations
type DeletionLogger struct {
	logFile   *os.File
	logLevel  LogLevel
	startTime time.Time
	sessionID string
	accountID string
	provider  string
}

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp    time.Time              `json:"timestamp"`
	Level        string                 `json:"level"`
	SessionID    string                 `json:"session_id"`
	AccountID    string                 `json:"account_id"`
	Provider     string                 `json:"provider"`
	Message      string                 `json:"message"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	ResourceType string                 `json:"resource_type,omitempty"`
	Duration     time.Duration          `json:"duration,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NewDeletionLogger creates a new deletion logger
func NewDeletionLogger(accountID, provider string) (*DeletionLogger, error) {
	// Create logs directory if it doesn't exist
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Generate session ID
	sessionID := fmt.Sprintf("deletion_%s_%s", time.Now().Format("20060102_150405"), accountID)

	// Create log file
	logFileName := filepath.Join(logsDir, fmt.Sprintf("%s.log", sessionID))
	logFile, err := os.Create(logFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	return &DeletionLogger{
		logFile:   logFile,
		logLevel:  INFO,
		startTime: time.Now(),
		sessionID: sessionID,
		accountID: accountID,
		provider:  provider,
	}, nil
}

// SetLogLevel sets the logging level
func (dl *DeletionLogger) SetLogLevel(level LogLevel) {
	dl.logLevel = level
}

// Close closes the logger
func (dl *DeletionLogger) Close() error {
	return dl.logFile.Close()
}

// Log logs a message with the specified level
func (dl *DeletionLogger) Log(level LogLevel, message string, metadata map[string]interface{}) {
	if level < dl.logLevel {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     dl.getLevelString(level),
		SessionID: dl.sessionID,
		AccountID: dl.accountID,
		Provider:  dl.provider,
		Message:   message,
		Metadata:  metadata,
	}

	dl.writeLogEntry(entry)
}

// LogResource logs a resource-specific message
func (dl *DeletionLogger) LogResource(level LogLevel, message, resourceID, resourceType string, metadata map[string]interface{}) {
	if level < dl.logLevel {
		return
	}

	entry := LogEntry{
		Timestamp:    time.Now(),
		Level:        dl.getLevelString(level),
		SessionID:    dl.sessionID,
		AccountID:    dl.accountID,
		Provider:     dl.provider,
		Message:      message,
		ResourceID:   resourceID,
		ResourceType: resourceType,
		Metadata:     metadata,
	}

	dl.writeLogEntry(entry)
}

// LogError logs an error message
func (dl *DeletionLogger) LogError(message, resourceID, resourceType, errorMsg string, metadata map[string]interface{}) {
	entry := LogEntry{
		Timestamp:    time.Now(),
		Level:        dl.getLevelString(ERROR),
		SessionID:    dl.sessionID,
		AccountID:    dl.accountID,
		Provider:     dl.provider,
		Message:      message,
		ResourceID:   resourceID,
		ResourceType: resourceType,
		Error:        errorMsg,
		Metadata:     metadata,
	}

	dl.writeLogEntry(entry)
}

// LogDeletionStart logs the start of a deletion operation
func (dl *DeletionLogger) LogDeletionStart(options DeletionOptions, resourceCount int) {
	metadata := map[string]interface{}{
		"dry_run":        options.DryRun,
		"force":          options.Force,
		"resource_types": options.ResourceTypes,
		"regions":        options.Regions,
		"timeout":        options.Timeout.String(),
		"batch_size":     options.BatchSize,
		"max_retries":    options.MaxRetries,
		"retry_delay":    options.RetryDelay.String(),
		"resource_count": resourceCount,
	}

	dl.Log(INFO, "Deletion operation started", metadata)
}

// LogDeletionComplete logs the completion of a deletion operation
func (dl *DeletionLogger) LogDeletionComplete(result *DeletionResult) {
	duration := time.Since(dl.startTime)

	metadata := map[string]interface{}{
		"total_resources":   result.TotalResources,
		"deleted_resources": result.DeletedResources,
		"failed_resources":  result.FailedResources,
		"skipped_resources": result.SkippedResources,
		"retried_resources": result.RetriedResources,
		"duration":          duration.String(),
		"success_rate":      fmt.Sprintf("%.2f%%", float64(result.DeletedResources)/float64(result.TotalResources)*100),
	}

	dl.Log(INFO, "Deletion operation completed", metadata)
}

// LogResourceDeletion logs the deletion of a specific resource
func (dl *DeletionLogger) LogResourceDeletion(resource models.Resource, duration time.Duration, success bool, errorMsg string) {
	metadata := map[string]interface{}{
		"region":   resource.Region,
		"state":    resource.State,
		"duration": duration.String(),
		"success":  success,
	}

	if success {
		dl.LogResource(INFO, "Resource deleted successfully", resource.ID, resource.Type, metadata)
	} else {
		dl.LogError("Resource deletion failed", resource.ID, resource.Type, errorMsg, metadata)
	}
}

// LogRetry logs a retry attempt
func (dl *DeletionLogger) LogRetry(resourceID, resourceType string, attempt int, maxRetries int, errorMsg string) {
	metadata := map[string]interface{}{
		"attempt":     attempt,
		"max_retries": maxRetries,
		"retry_delay": "5s",
	}

	dl.LogResource(WARN, fmt.Sprintf("Retrying resource deletion (attempt %d/%d)", attempt, maxRetries),
		resourceID, resourceType, metadata)
}

// LogSafetyCheck logs safety check results
func (dl *DeletionLogger) LogSafetyCheck(checkType string, passed bool, details map[string]interface{}) {
	level := INFO
	if !passed {
		level = WARN
	}

	metadata := map[string]interface{}{
		"check_type": checkType,
		"passed":     passed,
	}

	// Merge details into metadata
	for k, v := range details {
		metadata[k] = v
	}

	dl.Log(level, fmt.Sprintf("Safety check: %s", checkType), metadata)
}

// LogDependencyAnalysis logs dependency analysis results
func (dl *DeletionLogger) LogDependencyAnalysis(dependencies map[string][]string, deletionOrder []string) {
	metadata := map[string]interface{}{
		"dependency_count": len(dependencies),
		"deletion_order":   deletionOrder,
	}

	dl.Log(INFO, "Dependency analysis completed", metadata)
}

// LogBatchProgress logs batch deletion progress
func (dl *DeletionLogger) LogBatchProgress(batchNum, totalBatches, batchSize, processedCount int) {
	metadata := map[string]interface{}{
		"batch_num":       batchNum,
		"total_batches":   totalBatches,
		"batch_size":      batchSize,
		"processed_count": processedCount,
		"progress":        fmt.Sprintf("%.1f%%", float64(processedCount)/float64(totalBatches*batchSize)*100),
	}

	dl.Log(INFO, "Batch deletion progress", metadata)
}

// LogResourceDiscovery logs resource discovery results
func (dl *DeletionLogger) LogResourceDiscovery(provider string, resourceCount int, resourceTypes map[string]int) {
	metadata := map[string]interface{}{
		"provider":       provider,
		"resource_count": resourceCount,
		"resource_types": resourceTypes,
	}

	dl.Log(INFO, "Resource discovery completed", metadata)
}

// LogCredentialValidation logs credential validation results
func (dl *DeletionLogger) LogCredentialValidation(provider string, success bool, errorMsg string) {
	metadata := map[string]interface{}{
		"provider": provider,
		"success":  success,
	}

	if success {
		dl.Log(INFO, "Credential validation successful", metadata)
	} else {
		dl.LogError("Credential validation failed", "", "", errorMsg, metadata)
	}
}

// writeLogEntry writes a log entry to the log file
func (dl *DeletionLogger) writeLogEntry(entry LogEntry) {
	// Convert to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Failed to marshal log entry: %v", err)
		return
	}

	// Write to log file
	if _, err := dl.logFile.Write(append(jsonData, '\n')); err != nil {
		log.Printf("Failed to write log entry: %v", err)
	}

	// Also log to console for important messages
	if entry.Level == "ERROR" || entry.Level == "WARN" {
		log.Printf("[%s] %s: %s", entry.Level, entry.Message, entry.Error)
	}
}

// getLevelString converts LogLevel to string
func (dl *DeletionLogger) getLevelString(level LogLevel) string {
	switch level {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "INFO"
	}
}

// GetLogFilePath returns the path to the log file
func (dl *DeletionLogger) GetLogFilePath() string {
	return dl.logFile.Name()
}

// GetSessionID returns the session ID
func (dl *DeletionLogger) GetSessionID() string {
	return dl.sessionID
}

// ExportLogs exports logs in various formats
func (dl *DeletionLogger) ExportLogs(format string) ([]byte, error) {
	// Read the log file
	logData, err := os.ReadFile(dl.logFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	switch format {
	case "json":
		return logData, nil
	case "csv":
		return dl.convertToCSV(logData)
	case "summary":
		return dl.generateSummary(logData)
	default:
		return logData, nil
	}
}

// convertToCSV converts JSON logs to CSV format
func (dl *DeletionLogger) convertToCSV(logData []byte) ([]byte, error) {
	var entries []LogEntry
	lines := strings.Split(string(logData), "\n")
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip malformed lines
		}
		entries = append(entries, entry)
	}
	
	// Create CSV header
	var csv strings.Builder
	csv.WriteString("Timestamp,Level,Message,ResourceID,ResourceType,Provider,Error\n")
	
	// Add data rows
	for _, entry := range entries {
		csv.WriteString(fmt.Sprintf("%s,%s,%q,%s,%s,%s,%q\n",
			entry.Timestamp.Format(time.RFC3339),
			entry.Level,
			entry.Message,
			entry.ResourceID,
			entry.ResourceType,
			entry.Provider,
			entry.Error,
		))
	}
	
	return []byte(csv.String()), nil
}

// generateSummary generates a summary of the deletion operation
func (dl *DeletionLogger) generateSummary(logData []byte) ([]byte, error) {
	var entries []LogEntry
	lines := strings.Split(string(logData), "\n")
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	
	// Analyze logs for summary
	summary := struct {
		SessionID      string                 `json:"session_id"`
		StartTime      time.Time              `json:"start_time"`
		EndTime        time.Time              `json:"end_time"`
		Duration       string                 `json:"duration"`
		TotalResources int                    `json:"total_resources"`
		SuccessCount   int                    `json:"success_count"`
		FailureCount   int                    `json:"failure_count"`
		ErrorCount     int                    `json:"error_count"`
		ByProvider     map[string]int         `json:"by_provider"`
		ByResourceType map[string]int         `json:"by_resource_type"`
		Errors         []string               `json:"errors"`
	}{
		SessionID:      dl.sessionID,
		ByProvider:     make(map[string]int),
		ByResourceType: make(map[string]int),
		Errors:         []string{},
	}
	
	if len(entries) > 0 {
		summary.StartTime = entries[0].Timestamp
		summary.EndTime = entries[len(entries)-1].Timestamp
		summary.Duration = summary.EndTime.Sub(summary.StartTime).String()
	}
	
	// Count statistics
	resourcesSeen := make(map[string]bool)
	for _, entry := range entries {
		if entry.ResourceID != "" {
			resourcesSeen[entry.ResourceID] = true
			summary.ByProvider[entry.Provider]++
			summary.ByResourceType[entry.ResourceType]++
		}
		
		if entry.Level == "ERROR" {
			summary.ErrorCount++
			if entry.Error != "" {
				summary.Errors = append(summary.Errors, entry.Error)
			}
		}
		
		// Count success/failure based on message patterns
		if strings.Contains(entry.Message, "deleted successfully") {
			summary.SuccessCount++
		} else if strings.Contains(entry.Message, "failed") {
			summary.FailureCount++
		}
	}
	
	summary.TotalResources = len(resourcesSeen)
	
	return json.MarshalIndent(summary, "", "  ")
}
