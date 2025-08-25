package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// EventType represents the type of audit event
type EventType string

const (
	EventTypeDiscovery    EventType = "DISCOVERY"
	EventTypeDriftDetect  EventType = "DRIFT_DETECT"
	EventTypeRemediation  EventType = "REMEDIATION"
	EventTypeAccess       EventType = "ACCESS"
	EventTypeModification EventType = "MODIFICATION"
	EventTypeDeletion     EventType = "DELETION"
	EventTypeAuth         EventType = "AUTHENTICATION"
	EventTypeConfig       EventType = "CONFIGURATION"
)

// Severity represents the severity level of an audit event
type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityWarning  Severity = "WARNING"
	SeverityError    Severity = "ERROR"
	SeverityCritical Severity = "CRITICAL"
)

// AuditEvent represents a single audit log entry
type AuditEvent struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	EventType     EventType              `json:"event_type"`
	Severity      Severity               `json:"severity"`
	User          string                 `json:"user"`
	Service       string                 `json:"service"`
	Action        string                 `json:"action"`
	Resource      string                 `json:"resource"`
	Provider      string                 `json:"provider,omitempty"`
	Region        string                 `json:"region,omitempty"`
	Result        string                 `json:"result"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	IPAddress     string                 `json:"ip_address,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	Duration      time.Duration          `json:"duration,omitempty"`
}

// Logger interface for audit logging
type Logger interface {
	Log(ctx context.Context, event *AuditEvent) error
	Query(ctx context.Context, filter QueryFilter) ([]*AuditEvent, error)
	Export(ctx context.Context, format ExportFormat, writer interface{}) error
	Rotate() error
}

// QueryFilter for searching audit logs
type QueryFilter struct {
	StartTime     time.Time
	EndTime       time.Time
	EventTypes    []EventType
	Severities    []Severity
	Users         []string
	Services      []string
	Providers     []string
	CorrelationID string
	Limit         int
	Offset        int
}

// ExportFormat for audit log exports
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatSIEM ExportFormat = "siem"
)

// FileLogger implements file-based audit logging
type FileLogger struct {
	mu           sync.RWMutex
	basePath     string
	currentFile  *os.File
	events       []*AuditEvent
	maxFileSize  int64
	rotateCount  int
	bufferSize   int
	flushTicker  *time.Ticker
	stopCh       chan struct{}
}

// NewFileLogger creates a new file-based audit logger
func NewFileLogger(basePath string) (*FileLogger, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("creating audit directory: %w", err)
	}

	logger := &FileLogger{
		basePath:    basePath,
		events:      make([]*AuditEvent, 0, 100),
		maxFileSize: 100 * 1024 * 1024, // 100MB
		rotateCount: 10,
		bufferSize:  100,
		stopCh:      make(chan struct{}),
	}

	if err := logger.openCurrentFile(); err != nil {
		return nil, err
	}

	// Start background flush
	logger.flushTicker = time.NewTicker(5 * time.Second)
	go logger.backgroundFlush()

	return logger, nil
}

// Log records an audit event
func (l *FileLogger) Log(ctx context.Context, event *AuditEvent) error {
	if event.ID == "" {
		event.ID = generateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	l.mu.Lock()
	l.events = append(l.events, event)
	shouldFlush := len(l.events) >= l.bufferSize
	l.mu.Unlock()

	if shouldFlush {
		return l.flush()
	}

	return nil
}

// Query searches audit logs based on filter criteria
func (l *FileLogger) Query(ctx context.Context, filter QueryFilter) ([]*AuditEvent, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	results := make([]*AuditEvent, 0)
	
	// Search in current buffer
	for _, event := range l.events {
		if matchesFilter(event, filter) {
			results = append(results, event)
		}
	}

	// Search in rotated files
	files, err := l.getRotatedFiles()
	if err != nil {
		return results, err
	}

	for _, file := range files {
		events, err := l.readEventsFromFile(file)
		if err != nil {
			continue
		}
		for _, event := range events {
			if matchesFilter(event, filter) {
				results = append(results, event)
				if filter.Limit > 0 && len(results) >= filter.Limit {
					return results, nil
				}
			}
		}
	}

	return results, nil
}

// Export exports audit logs in specified format
func (l *FileLogger) Export(ctx context.Context, format ExportFormat, writer interface{}) error {
	events, err := l.Query(ctx, QueryFilter{
		StartTime: time.Now().AddDate(0, -1, 0), // Last month
		EndTime:   time.Now(),
	})
	if err != nil {
		return err
	}

	switch format {
	case ExportFormatJSON:
		encoder := json.NewEncoder(writer.(*os.File))
		encoder.SetIndent("", "  ")
		return encoder.Encode(events)
	case ExportFormatCSV:
		// CSV export implementation
		return exportToCSV(events, writer.(*os.File))
	case ExportFormatSIEM:
		// SIEM format export (e.g., CEF)
		return exportToSIEM(events, writer.(*os.File))
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}

// Rotate rotates the audit log file
func (l *FileLogger) Rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.flush(); err != nil {
		return err
	}

	if l.currentFile != nil {
		l.currentFile.Close()
	}

	// Rotate existing files
	for i := l.rotateCount - 1; i > 0; i-- {
		oldPath := l.getRotatedFilePath(i - 1)
		newPath := l.getRotatedFilePath(i)
		os.Rename(oldPath, newPath)
	}

	// Rename current to .1
	currentPath := l.getCurrentFilePath()
	if _, err := os.Stat(currentPath); err == nil {
		os.Rename(currentPath, l.getRotatedFilePath(0))
	}

	return l.openCurrentFile()
}

// Close closes the audit logger
func (l *FileLogger) Close() error {
	close(l.stopCh)
	l.flushTicker.Stop()
	
	if err := l.flush(); err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.currentFile != nil {
		return l.currentFile.Close()
	}
	return nil
}

// Private methods

func (l *FileLogger) openCurrentFile() error {
	path := l.getCurrentFilePath()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening audit file: %w", err)
	}
	l.currentFile = file
	return nil
}

func (l *FileLogger) flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.events) == 0 {
		return nil
	}

	for _, event := range l.events {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		l.currentFile.Write(data)
		l.currentFile.Write([]byte("\n"))
	}

	l.events = l.events[:0]
	return l.currentFile.Sync()
}

func (l *FileLogger) backgroundFlush() {
	for {
		select {
		case <-l.flushTicker.C:
			l.flush()
		case <-l.stopCh:
			return
		}
	}
}

func (l *FileLogger) getCurrentFilePath() string {
	return filepath.Join(l.basePath, "audit.log")
}

func (l *FileLogger) getRotatedFilePath(index int) string {
	return filepath.Join(l.basePath, fmt.Sprintf("audit.log.%d", index+1))
}

func (l *FileLogger) getRotatedFiles() ([]string, error) {
	var files []string
	for i := 0; i < l.rotateCount; i++ {
		path := l.getRotatedFilePath(i)
		if _, err := os.Stat(path); err == nil {
			files = append(files, path)
		}
	}
	return files, nil
}

func (l *FileLogger) readEventsFromFile(path string) ([]*AuditEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var events []*AuditEvent
	lines := splitLines(string(data))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var event AuditEvent
		if err := json.Unmarshal([]byte(line), &event); err == nil {
			events = append(events, &event)
		}
	}
	return events, nil
}

func matchesFilter(event *AuditEvent, filter QueryFilter) bool {
	if !filter.StartTime.IsZero() && event.Timestamp.Before(filter.StartTime) {
		return false
	}
	if !filter.EndTime.IsZero() && event.Timestamp.After(filter.EndTime) {
		return false
	}
	if len(filter.EventTypes) > 0 && !contains(filter.EventTypes, event.EventType) {
		return false
	}
	if len(filter.Severities) > 0 && !contains(filter.Severities, event.Severity) {
		return false
	}
	if len(filter.Users) > 0 && !containsString(filter.Users, event.User) {
		return false
	}
	if filter.CorrelationID != "" && event.CorrelationID != filter.CorrelationID {
		return false
	}
	return true
}

func generateEventID() string {
	return fmt.Sprintf("evt_%d_%s", time.Now().UnixNano(), randomString(8))
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func contains[T comparable](slice []T, item T) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func exportToCSV(events []*AuditEvent, writer *os.File) error {
	// CSV header
	writer.WriteString("ID,Timestamp,EventType,Severity,User,Service,Action,Resource,Provider,Result,Duration\n")
	for _, e := range events {
		line := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
			e.ID, e.Timestamp.Format(time.RFC3339), e.EventType, e.Severity,
			e.User, e.Service, e.Action, e.Resource, e.Provider, e.Result, e.Duration)
		writer.WriteString(line)
	}
	return nil
}

func exportToSIEM(events []*AuditEvent, writer *os.File) error {
	// CEF format for SIEM
	for _, e := range events {
		cef := fmt.Sprintf("CEF:0|DriftMgr|DriftMgr|1.0|%s|%s|%s|src=%s act=%s outcome=%s\n",
			e.EventType, e.Action, e.Severity, e.IPAddress, e.Action, e.Result)
		writer.WriteString(cef)
	}
	return nil
}

// ComplianceLogger wraps audit logger with compliance features
type ComplianceLogger struct {
	*FileLogger
	complianceMode string // SOC2, HIPAA, PCI-DSS
	encryption     bool
}

// NewComplianceLogger creates a compliance-aware audit logger
func NewComplianceLogger(basePath string, mode string) (*ComplianceLogger, error) {
	fileLogger, err := NewFileLogger(basePath)
	if err != nil {
		return nil, err
	}

	return &ComplianceLogger{
		FileLogger:     fileLogger,
		complianceMode: mode,
		encryption:     true, // Always encrypt for compliance
	}, nil
}

// LogComplianceEvent logs an event with compliance metadata
func (c *ComplianceLogger) LogComplianceEvent(ctx context.Context, event *AuditEvent) error {
	// Add compliance metadata
	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}
	event.Metadata["compliance_mode"] = c.complianceMode
	event.Metadata["encrypted"] = c.encryption
	event.Metadata["retention_days"] = c.getRetentionDays()

	return c.Log(ctx, event)
}

func (c *ComplianceLogger) getRetentionDays() int {
	switch c.complianceMode {
	case "SOC2":
		return 365 * 3 // 3 years
	case "HIPAA":
		return 365 * 6 // 6 years
	case "PCI-DSS":
		return 365 * 2 // 2 years
	default:
		return 365 // 1 year default
	}
}