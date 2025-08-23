package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// WorkspaceManager handles multiple Terraform workspaces and environment management
type WorkspaceManager struct {
	rootPath     string
	workspaces   map[string]*Workspace
	environments map[string]*Environment
	mu           sync.RWMutex
}

// Workspace represents a Terraform workspace
type Workspace struct {
	Name         string                 `json:"name"`
	Path         string                 `json:"path"`
	StatePath    string                 `json:"state_path"`
	Backend      string                 `json:"backend"`
	Config       map[string]interface{} `json:"config"`
	Environment  string                 `json:"environment"`
	Region       string                 `json:"region"`
	Account      string                 `json:"account"`
	LastModified time.Time              `json:"last_modified"`
	Status       WorkspaceStatus        `json:"status"`
	Resources    []models.Resource      `json:"resources"`
}

// Environment represents a deployment environment
type Environment struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Workspaces  []string               `json:"workspaces"`
	Promotion   *PromotionConfig       `json:"promotion"`
	Policies    map[string]interface{} `json:"policies"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// PromotionConfig defines how changes are promoted between environments
type PromotionConfig struct {
	SourceEnvironment string   `json:"source_environment"`
	TargetEnvironment string   `json:"target_environment"`
	ApprovalRequired  bool     `json:"approval_required"`
	Approvers         []string `json:"approvers"`
	AutoPromote       bool     `json:"auto_promote"`
	ValidationSteps   []string `json:"validation_steps"`
}

// WorkspaceStatus represents the status of a workspace
type WorkspaceStatus string

const (
	WorkspaceStatusActive    WorkspaceStatus = "active"
	WorkspaceStatusInactive  WorkspaceStatus = "inactive"
	WorkspaceStatusError     WorkspaceStatus = "error"
	WorkspaceStatusPromoting WorkspaceStatus = "promoting"
)

// NewWorkspaceManager creates a new workspace manager
func NewWorkspaceManager(rootPath string) *WorkspaceManager {
	return &WorkspaceManager{
		rootPath:     rootPath,
		workspaces:   make(map[string]*Workspace),
		environments: make(map[string]*Environment),
	}
}

// DiscoverWorkspaces discovers all Terraform workspaces in the root path
func (wm *WorkspaceManager) DiscoverWorkspaces(ctx context.Context) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Clear existing workspaces
	wm.workspaces = make(map[string]*Workspace)

	// Search for Terraform configurations
	err := filepath.Walk(wm.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Look for terraform.tf files or directories with .tf files
		if info.IsDir() {
			terraformFiles, err := filepath.Glob(filepath.Join(path, "*.tf"))
			if err == nil && len(terraformFiles) > 0 {
				workspace, err := wm.createWorkspaceFromPath(path)
				if err != nil {
					return nil // Skip this directory
				}
				wm.workspaces[workspace.Name] = workspace
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error discovering workspaces: %w", err)
	}

	// Discover environments from workspace patterns
	wm.discoverEnvironments()

	return nil
}

// createWorkspaceFromPath creates a workspace from a directory path
func (wm *WorkspaceManager) createWorkspaceFromPath(path string) (*Workspace, error) {
	relPath, err := filepath.Rel(wm.rootPath, path)
	if err != nil {
		return nil, err
	}

	// Extract environment and region from path
	parts := strings.Split(relPath, string(filepath.Separator))
	environment := ""
	region := ""
	account := ""

	// Common patterns: environments/dev/us-east-1, prod/eu-west-1, etc.
	for i, part := range parts {
		switch part {
		case "environments", "env", "envs":
			if i+1 < len(parts) {
				environment = parts[i+1]
			}
		case "regions", "region":
			if i+1 < len(parts) {
				region = parts[i+1]
			}
		case "accounts", "account":
			if i+1 < len(parts) {
				account = parts[i+1]
			}
		}
	}

	// If no explicit environment found, try to infer from path
	if environment == "" {
		for _, part := range parts {
			if part == "dev" || part == "staging" || part == "prod" || part == "test" {
				environment = part
				break
			}
		}
	}

	// If no explicit region found, try to infer from path
	if region == "" {
		for _, part := range parts {
			if strings.Contains(part, "-") && len(part) > 5 {
				region = part
				break
			}
		}
	}

	workspace := &Workspace{
		Name:         relPath,
		Path:         path,
		Environment:  environment,
		Region:       region,
		Account:      account,
		Status:       WorkspaceStatusActive,
		LastModified: time.Now(),
		Config:       make(map[string]interface{}),
	}

	// Try to determine backend configuration
	workspace.Backend = wm.detectBackend(path)

	return workspace, nil
}

// detectBackend detects the Terraform backend configuration
func (wm *WorkspaceManager) detectBackend(path string) string {
	// Check for backend configuration in terraform.tf files
	tfFiles := []string{"main.tf", "backend.tf", "terraform.tf"}

	for _, tfFile := range tfFiles {
		content, err := os.ReadFile(filepath.Join(path, tfFile))
		if err != nil {
			continue
		}

		contentStr := string(content)
		if strings.Contains(contentStr, "backend \"s3\"") {
			return "s3"
		} else if strings.Contains(contentStr, "backend \"azurerm\"") {
			return "azurerm"
		} else if strings.Contains(contentStr, "backend \"gcs\"") {
			return "gcs"
		} else if strings.Contains(contentStr, "backend \"local\"") {
			return "local"
		}
	}

	return "unknown"
}

// discoverEnvironments discovers environments from workspace patterns
func (wm *WorkspaceManager) discoverEnvironments() {
	envMap := make(map[string][]string)

	// Group workspaces by environment
	for _, workspace := range wm.workspaces {
		if workspace.Environment != "" {
			envMap[workspace.Environment] = append(envMap[workspace.Environment], workspace.Name)
		}
	}

	// Create environment objects
	for envName, workspaceNames := range envMap {
		environment := &Environment{
			Name:       envName,
			Workspaces: workspaceNames,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Policies:   make(map[string]interface{}),
		}

		// Set default promotion config
		environment.Promotion = &PromotionConfig{
			ApprovalRequired: true,
			AutoPromote:      false,
		}

		wm.environments[envName] = environment
	}
}

// ListWorkspaces returns all discovered workspaces
func (wm *WorkspaceManager) ListWorkspaces() []*Workspace {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	workspaces := make([]*Workspace, 0, len(wm.workspaces))
	for _, workspace := range wm.workspaces {
		workspaces = append(workspaces, workspace)
	}
	return workspaces
}

// ListEnvironments returns all discovered environments
func (wm *WorkspaceManager) ListEnvironments() []*Environment {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	environments := make([]*Environment, 0, len(wm.environments))
	for _, env := range wm.environments {
		environments = append(environments, env)
	}
	return environments
}

// GetWorkspace returns a specific workspace by name
func (wm *WorkspaceManager) GetWorkspace(name string) (*Workspace, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	workspace, exists := wm.workspaces[name]
	if !exists {
		return nil, fmt.Errorf("workspace not found: %s", name)
	}
	return workspace, nil
}

// GetEnvironment returns a specific environment by name
func (wm *WorkspaceManager) GetEnvironment(name string) (*Environment, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	env, exists := wm.environments[name]
	if !exists {
		return nil, fmt.Errorf("environment not found: %s", name)
	}
	return env, nil
}

// CompareEnvironments compares drift between two environments
func (wm *WorkspaceManager) CompareEnvironments(env1, env2 string) (*EnvironmentComparison, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	env1Obj, exists := wm.environments[env1]
	if !exists {
		return nil, fmt.Errorf("environment not found: %s", env1)
	}

	env2Obj, exists := wm.environments[env2]
	if !exists {
		return nil, fmt.Errorf("environment not found: %s", env2)
	}

	comparison := &EnvironmentComparison{
		SourceEnvironment: env1,
		TargetEnvironment: env2,
		ComparedAt:        time.Now(),
		Differences:       make([]EnvironmentDifference, 0),
	}

	// Compare workspaces between environments
	env1Workspaces := make(map[string]*Workspace)
	env2Workspaces := make(map[string]*Workspace)

	for _, wsName := range env1Obj.Workspaces {
		if ws, exists := wm.workspaces[wsName]; exists {
			env1Workspaces[wsName] = ws
		}
	}

	for _, wsName := range env2Obj.Workspaces {
		if ws, exists := wm.workspaces[wsName]; exists {
			env2Workspaces[wsName] = ws
		}
	}

	// Find differences
	for wsName, ws1 := range env1Workspaces {
		ws2, exists := env2Workspaces[wsName]
		if !exists {
			comparison.Differences = append(comparison.Differences, EnvironmentDifference{
				WorkspaceName: wsName,
				Type:          DifferenceTypeMissing,
				Description:   fmt.Sprintf("Workspace exists in %s but not in %s", env1, env2),
			})
			continue
		}

		// Compare workspace configurations
		if ws1.Region != ws2.Region {
			comparison.Differences = append(comparison.Differences, EnvironmentDifference{
				WorkspaceName: wsName,
				Type:          DifferenceTypeConfiguration,
				Description:   fmt.Sprintf("Region mismatch: %s vs %s", ws1.Region, ws2.Region),
			})
		}

		if ws1.Backend != ws2.Backend {
			comparison.Differences = append(comparison.Differences, EnvironmentDifference{
				WorkspaceName: wsName,
				Type:          DifferenceTypeConfiguration,
				Description:   fmt.Sprintf("Backend mismatch: %s vs %s", ws1.Backend, ws2.Backend),
			})
		}
	}

	// Check for workspaces in env2 that don't exist in env1
	for wsName := range env2Workspaces {
		if _, exists := env1Workspaces[wsName]; !exists {
			comparison.Differences = append(comparison.Differences, EnvironmentDifference{
				WorkspaceName: wsName,
				Type:          DifferenceTypeExtra,
				Description:   fmt.Sprintf("Workspace exists in %s but not in %s", env2, env1),
			})
		}
	}

	return comparison, nil
}

// PromoteEnvironment promotes changes from one environment to another
func (wm *WorkspaceManager) PromoteEnvironment(ctx context.Context, sourceEnv, targetEnv string, options *PromotionOptions) (*PromotionResult, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Validate environments exist
	sourceEnvObj, exists := wm.environments[sourceEnv]
	if !exists {
		return nil, fmt.Errorf("source environment not found: %s", sourceEnv)
	}

	targetEnvObj, exists := wm.environments[targetEnv]
	if !exists {
		return nil, fmt.Errorf("target environment not found: %s", targetEnv)
	}

	// Check promotion configuration
	if targetEnvObj.Promotion != nil && targetEnvObj.Promotion.ApprovalRequired && !options.AutoApprove {
		return nil, fmt.Errorf("approval required for promotion to %s", targetEnv)
	}

	result := &PromotionResult{
		SourceEnvironment: sourceEnv,
		TargetEnvironment: targetEnv,
		StartedAt:         time.Now(),
		Status:            PromotionStatusInProgress,
		Steps:             make([]PromotionStep, 0),
	}

	// Execute promotion steps
	for _, workspaceName := range sourceEnvObj.Workspaces {
		workspace, exists := wm.workspaces[workspaceName]
		if !exists {
			continue
		}

		step := PromotionStep{
			WorkspaceName: workspaceName,
			StartedAt:     time.Now(),
			Status:        StepStatusInProgress,
		}

		// Execute promotion for this workspace
		err := wm.promoteWorkspace(ctx, workspace, targetEnv, options)
		if err != nil {
			step.Status = StepStatusFailed
			step.Error = err.Error()
			result.Status = PromotionStatusFailed
		} else {
			step.Status = StepStatusCompleted
		}

		step.CompletedAt = time.Now()
		result.Steps = append(result.Steps, step)
	}

	if result.Status == PromotionStatusInProgress {
		result.Status = PromotionStatusCompleted
	}
	result.CompletedAt = time.Now()

	return result, nil
}

// promoteWorkspace promotes a single workspace to a target environment
func (wm *WorkspaceManager) promoteWorkspace(ctx context.Context, workspace *Workspace, targetEnv string, options *PromotionOptions) error {
	// This is a simplified implementation
	// In a real implementation, you would:
	// 1. Copy Terraform configurations
	// 2. Update environment-specific variables
	// 3. Run terraform plan
	// 4. Apply changes if approved

	// For now, we'll just update the workspace environment
	workspace.Environment = targetEnv
	workspace.LastModified = time.Now()

	return nil
}

// EnvironmentComparison represents a comparison between two environments
type EnvironmentComparison struct {
	SourceEnvironment string                  `json:"source_environment"`
	TargetEnvironment string                  `json:"target_environment"`
	ComparedAt        time.Time               `json:"compared_at"`
	Differences       []EnvironmentDifference `json:"differences"`
}

// EnvironmentDifference represents a difference between environments
type EnvironmentDifference struct {
	WorkspaceName string         `json:"workspace_name"`
	Type          DifferenceType `json:"type"`
	Description   string         `json:"description"`
}

// DifferenceType represents the type of difference
type DifferenceType string

const (
	DifferenceTypeMissing       DifferenceType = "missing"
	DifferenceTypeExtra         DifferenceType = "extra"
	DifferenceTypeConfiguration DifferenceType = "configuration"
	DifferenceTypeResource      DifferenceType = "resource"
)

// PromotionOptions defines options for environment promotion
type PromotionOptions struct {
	AutoApprove bool              `json:"auto_approve"`
	DryRun      bool              `json:"dry_run"`
	Parallel    bool              `json:"parallel"`
	Timeout     time.Duration     `json:"timeout"`
	Filters     map[string]string `json:"filters"`
}

// PromotionResult represents the result of an environment promotion
type PromotionResult struct {
	SourceEnvironment string          `json:"source_environment"`
	TargetEnvironment string          `json:"target_environment"`
	StartedAt         time.Time       `json:"started_at"`
	CompletedAt       time.Time       `json:"completed_at"`
	Status            PromotionStatus `json:"status"`
	Steps             []PromotionStep `json:"steps"`
}

// PromotionStatus represents the status of a promotion
type PromotionStatus string

const (
	PromotionStatusInProgress PromotionStatus = "in_progress"
	PromotionStatusCompleted  PromotionStatus = "completed"
	PromotionStatusFailed     PromotionStatus = "failed"
	PromotionStatusCancelled  PromotionStatus = "cancelled"
)

// PromotionStep represents a step in the promotion process
type PromotionStep struct {
	WorkspaceName string     `json:"workspace_name"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   time.Time  `json:"completed_at"`
	Status        StepStatus `json:"status"`
	Error         string     `json:"error,omitempty"`
}

// StepStatus represents the status of a promotion step
type StepStatus string

const (
	StepStatusInProgress StepStatus = "in_progress"
	StepStatusCompleted  StepStatus = "completed"
	StepStatusFailed     StepStatus = "failed"
	StepStatusSkipped    StepStatus = "skipped"
)
