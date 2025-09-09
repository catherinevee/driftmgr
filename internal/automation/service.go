package automation

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AutomationService provides a unified interface for intelligent automation
type AutomationService struct {
	workflowEngine *WorkflowEngine
	ruleEngine     *RuleEngine
	scheduler      *Scheduler
	mu             sync.RWMutex
	// eventBus removed for interface simplification
	config *AutomationConfig
}

// AutomationConfig represents configuration for the automation service
type AutomationConfig struct {
	AutoDiscovery       bool          `json:"auto_discovery"`
	AutoRemediation     bool          `json:"auto_remediation"`
	AutoScaling         bool          `json:"auto_scaling"`
	AutoBackup          bool          `json:"auto_backup"`
	AutoMonitoring      bool          `json:"auto_monitoring"`
	NotificationEnabled bool          `json:"notification_enabled"`
	AuditLogging        bool          `json:"audit_logging"`
	MaxConcurrentJobs   int           `json:"max_concurrent_jobs"`
	DefaultTimeout      time.Duration `json:"default_timeout"`
}

// NewAutomationService creates a new automation service
func NewAutomationService() *AutomationService {
	config := &AutomationConfig{
		AutoDiscovery:       true,
		AutoRemediation:     true,
		AutoScaling:         true,
		AutoBackup:          true,
		AutoMonitoring:      true,
		NotificationEnabled: true,
		AuditLogging:        true,
		MaxConcurrentJobs:   20,
		DefaultTimeout:      30 * time.Minute,
	}

	// Create components
	workflowEngine := NewWorkflowEngine()
	ruleEngine := NewRuleEngine()
	scheduler := NewScheduler()

	return &AutomationService{
		workflowEngine: workflowEngine,
		ruleEngine:     ruleEngine,
		scheduler:      scheduler,
		config:         config,
	}
}

// GetWorkflowEngine returns the workflow engine
func (s *AutomationService) GetWorkflowEngine() *WorkflowEngine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workflowEngine
}

// GetRuleEngine returns the rule engine
func (s *AutomationService) GetRuleEngine() *RuleEngine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ruleEngine
}

// GetScheduler returns the scheduler
func (s *AutomationService) GetScheduler() *Scheduler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scheduler
}

// Start starts the automation service
func (as *AutomationService) Start(ctx context.Context) error {
	// Start scheduler
	if err := as.scheduler.Start(ctx); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	// Create default workflows and rules
	if err := as.createDefaultAutomations(ctx); err != nil {
		return fmt.Errorf("failed to create default automations: %w", err)
	}

	// TODO: Implement event publishing

	return nil
}

// Stop stops the automation service
func (as *AutomationService) Stop(ctx context.Context) error {
	// Stop scheduler
	if err := as.scheduler.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop scheduler: %w", err)
	}

	// TODO: Implement event publishing

	return nil
}

// CreateWorkflow creates a new automation workflow
func (as *AutomationService) CreateWorkflow(ctx context.Context, workflow *Workflow) error {
	return as.workflowEngine.CreateWorkflow(ctx, workflow)
}

// ExecuteWorkflow executes a workflow
func (as *AutomationService) ExecuteWorkflow(ctx context.Context, workflowID string, input map[string]interface{}) (*WorkflowExecution, error) {
	return as.workflowEngine.ExecuteWorkflow(ctx, workflowID, input)
}

// CreateRule creates a new automation rule
func (as *AutomationService) CreateRule(ctx context.Context, rule *AutomationRule) error {
	return as.ruleEngine.CreateRule(ctx, rule)
}

// ScheduleWorkflow schedules a workflow for execution
func (as *AutomationService) ScheduleWorkflow(ctx context.Context, workflowID string, schedule string, input map[string]interface{}) (*ScheduledJob, error) {
	job := &ScheduledJob{
		ID:         fmt.Sprintf("job_%d", time.Now().Unix()),
		Name:       fmt.Sprintf("Scheduled workflow %s", workflowID),
		Type:       "workflow",
		Schedule:   schedule,
		WorkflowID: workflowID,
		Input:      input,
		Enabled:    true,
		CreatedAt:  time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	return as.scheduler.ScheduleJob(ctx, job)
}

// GetAutomationStatus returns the overall automation status
func (as *AutomationService) GetAutomationStatus(ctx context.Context) (*AutomationStatus, error) {
	status := &AutomationStatus{
		OverallStatus: "Unknown",
		Workflows:     make(map[string]int),
		Rules:         make(map[string]int),
		ScheduledJobs: make(map[string]int),
		LastActivity:  time.Time{},
		Metadata:      make(map[string]interface{}),
	}

	// Get workflow counts
	workflows, err := as.workflowEngine.ListWorkflows(ctx)
	if err == nil {
		for _, workflow := range workflows {
			status.Workflows[workflow.Category]++
		}
	}

	// Get rule counts
	rules, err := as.ruleEngine.ListRules(ctx)
	if err == nil {
		for _, rule := range rules {
			status.Rules[rule.Category]++
		}
	}

	// Get scheduled job counts
	jobs, err := as.scheduler.ListJobs(ctx)
	if err == nil {
		for _, job := range jobs {
			status.ScheduledJobs[job.Type]++
		}
	}

	// Determine overall status
	totalAutomations := len(workflows) + len(rules) + len(jobs)
	if totalAutomations > 0 {
		status.OverallStatus = "Active"
	} else {
		status.OverallStatus = "Inactive"
	}

	return status, nil
}

// TriggerAutomation triggers automation based on events
func (as *AutomationService) TriggerAutomation(ctx context.Context, event *AutomationEvent) error {
	// Find matching rules
	rules, err := as.ruleEngine.ListRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to list rules: %w", err)
	}

	for _, rule := range rules {
		if rule.Enabled && as.ruleMatches(rule, event) {
			// Execute rule actions
			for _, action := range rule.Actions {
				if err := as.executeRuleAction(ctx, action, event); err != nil {
					// Log error but continue with other actions
					fmt.Printf("Warning: failed to execute rule action: %v\n", err)
				}
			}
		}
	}

	return nil
}

// AutomationStatus represents the overall automation status
type AutomationStatus struct {
	OverallStatus string                 `json:"overall_status"`
	Workflows     map[string]int         `json:"workflows"`
	Rules         map[string]int         `json:"rules"`
	ScheduledJobs map[string]int         `json:"scheduled_jobs"`
	LastActivity  time.Time              `json:"last_activity"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AutomationEvent represents an automation event
type AutomationEvent struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Source     string                 `json:"source"`
	ResourceID string                 `json:"resource_id,omitempty"`
	Data       map[string]interface{} `json:"data"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Helper methods

// createDefaultAutomations creates default automation workflows and rules
func (as *AutomationService) createDefaultAutomations(ctx context.Context) error {
	// Create default workflows
	workflows := []*Workflow{
		{
			Name:        "Auto-Scale Resources",
			Description: "Automatically scale resources based on metrics",
			Category:    "scaling",
			Steps: []WorkflowStep{
				{
					ID:     "check_metrics",
					Name:   "Check Metrics",
					Type:   "condition",
					Action: "check_cpu_usage",
					Parameters: map[string]interface{}{
						"threshold": 80,
					},
					Timeout: 5 * time.Minute,
				},
				{
					ID:     "scale_up",
					Name:   "Scale Up",
					Type:   "resource",
					Action: "scale_instance",
					Parameters: map[string]interface{}{
						"scale_factor": 1.5,
					},
					Timeout: 10 * time.Minute,
				},
			},
			Triggers: []WorkflowTrigger{
				{
					ID:   "schedule_trigger",
					Type: "schedule",
					Parameters: map[string]interface{}{
						"cron": "*/5 * * * *", // Every 5 minutes
					},
					IsActive: true,
				},
			},
			IsActive: true,
		},
		{
			Name:        "Auto-Backup Resources",
			Description: "Automatically backup critical resources",
			Category:    "backup",
			Steps: []WorkflowStep{
				{
					ID:     "create_backup",
					Name:   "Create Backup",
					Type:   "resource",
					Action: "create_backup",
					Parameters: map[string]interface{}{
						"retention_days": 30,
					},
					Timeout: 15 * time.Minute,
				},
				{
					ID:     "verify_backup",
					Name:   "Verify Backup",
					Type:   "resource",
					Action: "verify_backup",
					Parameters: map[string]interface{}{
						"verify_integrity": true,
					},
					Timeout: 5 * time.Minute,
				},
			},
			Triggers: []WorkflowTrigger{
				{
					ID:   "daily_trigger",
					Type: "schedule",
					Parameters: map[string]interface{}{
						"cron": "0 2 * * *", // Daily at 2 AM
					},
					IsActive: true,
				},
			},
			IsActive: true,
		},
		{
			Name:        "Auto-Remediate Issues",
			Description: "Automatically remediate common issues",
			Category:    "remediation",
			Steps: []WorkflowStep{
				{
					ID:     "detect_issue",
					Name:   "Detect Issue",
					Type:   "condition",
					Action: "check_health",
					Parameters: map[string]interface{}{
						"health_threshold": 70,
					},
					Timeout: 2 * time.Minute,
				},
				{
					ID:     "remediate",
					Name:   "Remediate",
					Type:   "resource",
					Action: "auto_remediate",
					Parameters: map[string]interface{}{
						"remediation_type": "auto",
					},
					Timeout: 10 * time.Minute,
				},
			},
			Triggers: []WorkflowTrigger{
				{
					ID:   "event_trigger",
					Type: "event",
					Parameters: map[string]interface{}{
						"event_type": "health_check_failed",
					},
					IsActive: true,
				},
			},
			IsActive: true,
		},
	}

	// Create workflows
	for _, workflow := range workflows {
		if err := as.workflowEngine.CreateWorkflow(ctx, workflow); err != nil {
			return fmt.Errorf("failed to create workflow %s: %w", workflow.Name, err)
		}
	}

	// Create default rules
	rules := []*AutomationRule{
		{
			Name:        "High CPU Usage Rule",
			Description: "Trigger scaling when CPU usage is high",
			Category:    "scaling",
			Conditions: []RuleCondition{
				{
					Field:    "cpu_usage",
					Operator: "greater_than",
					Value:    80,
					Type:     "number",
				},
			},
			Actions: []RuleAction{
				{
					Type:        "execute_workflow",
					Description: "Execute auto-scale workflow",
					Parameters: map[string]interface{}{
						"workflow_id": "auto_scale_resources",
					},
				},
			},
			Enabled: true,
		},
		{
			Name:        "Low Disk Space Rule",
			Description: "Trigger cleanup when disk space is low",
			Category:    "maintenance",
			Conditions: []RuleCondition{
				{
					Field:    "disk_usage",
					Operator: "greater_than",
					Value:    90,
					Type:     "number",
				},
			},
			Actions: []RuleAction{
				{
					Type:        "execute_workflow",
					Description: "Execute cleanup workflow",
					Parameters: map[string]interface{}{
						"workflow_id": "cleanup_resources",
					},
				},
			},
			Enabled: true,
		},
		{
			Name:        "Security Violation Rule",
			Description: "Trigger remediation when security violations are detected",
			Category:    "security",
			Conditions: []RuleCondition{
				{
					Field:    "security_violations",
					Operator: "greater_than",
					Value:    0,
					Type:     "number",
				},
			},
			Actions: []RuleAction{
				{
					Type:        "execute_workflow",
					Description: "Execute security remediation workflow",
					Parameters: map[string]interface{}{
						"workflow_id": "security_remediation",
					},
				},
			},
			Enabled: true,
		},
	}

	// Create rules
	for _, rule := range rules {
		if err := as.ruleEngine.CreateRule(ctx, rule); err != nil {
			return fmt.Errorf("failed to create rule %s: %w", rule.Name, err)
		}
	}

	return nil
}

// ruleMatches checks if a rule matches an event
func (as *AutomationService) ruleMatches(rule *AutomationRule, event *AutomationEvent) bool {
	for _, condition := range rule.Conditions {
		if !as.conditionMatches(condition, event) {
			return false
		}
	}
	return true
}

// conditionMatches checks if a condition matches an event
func (as *AutomationService) conditionMatches(condition RuleCondition, event *AutomationEvent) bool {
	actualValue := event.Data[condition.Field]

	switch condition.Operator {
	case "equals":
		return actualValue == condition.Value
	case "not_equals":
		return actualValue != condition.Value
	case "greater_than":
		return as.compareValues(actualValue, condition.Value) > 0
	case "less_than":
		return as.compareValues(actualValue, condition.Value) < 0
	case "contains":
		if str, ok := actualValue.(string); ok {
			if val, ok := condition.Value.(string); ok {
				return contains(str, val)
			}
		}
		return false
	default:
		return false
	}
}

// compareValues compares two values
func (as *AutomationService) compareValues(a, b interface{}) int {
	// Simplified comparison - in reality, you'd handle different types
	if a == b {
		return 0
	}
	// This is a placeholder - real implementation would be more sophisticated
	return 1
}

// executeRuleAction executes a rule action
func (as *AutomationService) executeRuleAction(ctx context.Context, action RuleAction, event *AutomationEvent) error {
	switch action.Type {
	case "execute_workflow":
		workflowID, ok := action.Parameters["workflow_id"].(string)
		if !ok {
			return fmt.Errorf("workflow_id parameter not found")
		}

		input := make(map[string]interface{})
		input["event"] = event
		input["trigger"] = "rule"

		_, err := as.workflowEngine.ExecuteWorkflow(ctx, workflowID, input)
		return err

	case "send_notification":
		// Placeholder for notification action
		fmt.Printf("Sending notification: %s\n", action.Description)
		return nil

	case "log_event":
		// Placeholder for logging action
		fmt.Printf("Logging event: %s\n", action.Description)
		return nil

	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// SetConfig updates the automation service configuration
func (as *AutomationService) SetConfig(config *AutomationConfig) {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.config = config
}

// GetConfig returns the current automation service configuration
func (as *AutomationService) GetConfig() *AutomationConfig {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.config
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}
