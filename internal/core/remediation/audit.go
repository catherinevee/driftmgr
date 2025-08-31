package remediation

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditLogger logs remediation actions for audit trail
type AuditLogger struct {
	mu       sync.Mutex
	logFile  *os.File
	logPath  string
	enabled  bool
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger() *AuditLogger {
	logger := &AuditLogger{
		enabled: true,
	}
	
	// Create audit log directory
	logDir := filepath.Join(os.TempDir(), "driftmgr", "audit")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Failed to create audit log directory: %v", err)
		logger.enabled = false
		return logger
	}
	
	// Create daily log file
	logFile := fmt.Sprintf("remediation_%s.log", time.Now().Format("2006-01-02"))
	logger.logPath = filepath.Join(logDir, logFile)
	
	// Open log file
	file, err := os.OpenFile(logger.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open audit log file: %v", err)
		logger.enabled = false
		return logger
	}
	
	logger.logFile = file
	return logger
}

// LogAction logs a remediation action
func (a *AuditLogger) LogAction(actionType, resourceID, provider string) {
	if !a.enabled || a.logFile == nil {
		return
	}
	
	a.mu.Lock()
	defer a.mu.Unlock()
	
	entry := map[string]interface{}{
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"action_type": actionType,
		"resource_id": resourceID,
		"provider":    provider,
		"user":        os.Getenv("USER"),
		"hostname":    getHostname(),
	}
	
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Failed to marshal audit log entry: %v", err)
		return
	}
	
	if _, err := a.logFile.Write(append(data, '\n')); err != nil {
		log.Printf("Failed to write audit log entry: %v", err)
	}
}

// LogPlan logs a remediation plan
func (a *AuditLogger) LogPlan(plan *Plan) {
	if !a.enabled || a.logFile == nil {
		return
	}
	
	a.mu.Lock()
	defer a.mu.Unlock()
	
	entry := map[string]interface{}{
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"event_type":    "plan_created",
		"plan_id":       plan.ID,
		"plan_name":     plan.Name,
		"action_count":  len(plan.Actions),
		"user":          os.Getenv("USER"),
		"hostname":      getHostname(),
	}
	
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Failed to marshal audit log entry: %v", err)
		return
	}
	
	if _, err := a.logFile.Write(append(data, '\n')); err != nil {
		log.Printf("Failed to write audit log entry: %v", err)
	}
}

// LogResult logs remediation results
func (a *AuditLogger) LogResult(planID string, results *Results) {
	if !a.enabled || a.logFile == nil {
		return
	}
	
	a.mu.Lock()
	defer a.mu.Unlock()
	
	entry := map[string]interface{}{
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"event_type":    "plan_executed",
		"plan_id":       planID,
		"success":       results.Success,
		"items_fixed":   results.ItemsFixed,
		"items_failed":  results.ItemsFailed,
		"duration_ms":   results.Duration.Milliseconds(),
		"user":          os.Getenv("USER"),
		"hostname":      getHostname(),
	}
	
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Failed to marshal audit log entry: %v", err)
		return
	}
	
	if _, err := a.logFile.Write(append(data, '\n')); err != nil {
		log.Printf("Failed to write audit log entry: %v", err)
	}
}

// Close closes the audit logger
func (a *AuditLogger) Close() error {
	if a.logFile != nil {
		return a.logFile.Close()
	}
	return nil
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}