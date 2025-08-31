package cqrs

import (
	"context"
	"fmt"
	"time"
)

// Command represents a command in the CQRS pattern
type Command interface {
	Name() string
	Validate() error
}

// CommandHandler handles commands
type CommandHandler interface {
	Handle(ctx context.Context, cmd Command) error
}

// CommandBus dispatches commands to handlers
type CommandBus struct {
	handlers map[string]CommandHandler
}

// NewCommandBus creates a new command bus
func NewCommandBus() *CommandBus {
	return &CommandBus{
		handlers: make(map[string]CommandHandler),
	}
}

// Register registers a command handler
func (b *CommandBus) Register(cmdName string, handler CommandHandler) {
	b.handlers[cmdName] = handler
}

// Dispatch dispatches a command to its handler
func (b *CommandBus) Dispatch(ctx context.Context, cmd Command) error {
	if err := cmd.Validate(); err != nil {
		return fmt.Errorf("command validation failed: %w", err)
	}

	handler, exists := b.handlers[cmd.Name()]
	if !exists {
		return fmt.Errorf("no handler registered for command: %s", cmd.Name())
	}

	return handler.Handle(ctx, cmd)
}

// Discovery Commands

// StartDiscoveryCommand starts resource discovery
type StartDiscoveryCommand struct {
	Provider      string
	Regions       []string
	ResourceTypes []string
	AutoRemediate bool
}

func (c StartDiscoveryCommand) Name() string { return "StartDiscovery" }

func (c StartDiscoveryCommand) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	return nil
}

// State Commands

// ImportStateCommand imports a Terraform state file
type ImportStateCommand struct {
	Path string
}

func (c ImportStateCommand) Name() string { return "ImportState" }

func (c ImportStateCommand) Validate() error {
	if c.Path == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}

// AnalyzeStateCommand analyzes state files
type AnalyzeStateCommand struct {
	FileIDs []string
}

func (c AnalyzeStateCommand) Name() string { return "AnalyzeState" }

func (c AnalyzeStateCommand) Validate() error {
	if len(c.FileIDs) == 0 {
		return fmt.Errorf("at least one file ID is required")
	}
	return nil
}

// Drift Commands

// DetectDriftCommand detects configuration drift
type DetectDriftCommand struct {
	Provider      string
	StateFileID   string
	StateFilePath string
	ResourceTypes []string
	Regions       []string
}

func (c DetectDriftCommand) Name() string { return "DetectDrift" }

func (c DetectDriftCommand) Validate() error {
	if c.Provider == "" && c.StateFileID == "" && c.StateFilePath == "" {
		return fmt.Errorf("provider, state file ID, or state file path required")
	}
	return nil
}

// Remediation Commands

// StartRemediationCommand starts remediation
type StartRemediationCommand struct {
	DriftReportID string
	DriftItemIDs  []string
	DryRun        bool
	Force         bool
	Strategy      string
}

func (c StartRemediationCommand) Name() string { return "StartRemediation" }

func (c StartRemediationCommand) Validate() error {
	if c.DriftReportID == "" && len(c.DriftItemIDs) == 0 {
		return fmt.Errorf("drift report ID or drift item IDs required")
	}
	return nil
}

// ApproveRemediationCommand approves a remediation plan
type ApproveRemediationCommand struct {
	PlanID   string
	Approver string
}

func (c ApproveRemediationCommand) Name() string { return "ApproveRemediation" }

func (c ApproveRemediationCommand) Validate() error {
	if c.PlanID == "" {
		return fmt.Errorf("plan ID is required")
	}
	if c.Approver == "" {
		return fmt.Errorf("approver is required")
	}
	return nil
}

// Workflow Commands

// ExecuteWorkflowCommand executes a predefined workflow
type ExecuteWorkflowCommand struct {
	WorkflowType string
	Parameters   map[string]interface{}
}

func (c ExecuteWorkflowCommand) Name() string { return "ExecuteWorkflow" }

func (c ExecuteWorkflowCommand) Validate() error {
	if c.WorkflowType == "" {
		return fmt.Errorf("workflow type is required")
	}
	
	validWorkflows := []string{"terraform_drift", "cleanup_unmanaged", "state_migration"}
	valid := false
	for _, w := range validWorkflows {
		if c.WorkflowType == w {
			valid = true
			break
		}
	}
	
	if !valid {
		return fmt.Errorf("invalid workflow type: %s", c.WorkflowType)
	}
	
	return nil
}

// Cache Commands

// ClearCacheCommand clears the cache
type ClearCacheCommand struct {
	Pattern string // Optional pattern to clear specific cache entries
}

func (c ClearCacheCommand) Name() string { return "ClearCache" }

func (c ClearCacheCommand) Validate() error {
	return nil
}

// Bulk Operations Commands

// BulkDeleteCommand deletes multiple resources
type BulkDeleteCommand struct {
	ResourceIDs []string
	Provider    string
	Force       bool
}

func (c BulkDeleteCommand) Name() string { return "BulkDelete" }

func (c BulkDeleteCommand) Validate() error {
	if len(c.ResourceIDs) == 0 {
		return fmt.Errorf("at least one resource ID is required")
	}
	if c.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	return nil
}

// Command Results

// CommandResult represents the result of a command execution
type CommandResult struct {
	Success   bool
	Message   string
	Data      interface{}
	Error     error
	Timestamp time.Time
}

// NewSuccessResult creates a successful command result
func NewSuccessResult(message string, data interface{}) CommandResult {
	return CommandResult{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewErrorResult creates an error command result
func NewErrorResult(err error) CommandResult {
	return CommandResult{
		Success:   false,
		Error:     err,
		Timestamp: time.Now(),
	}
}