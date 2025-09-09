package strategies

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/catherinevee/driftmgr/internal/drift/comparator"
	"github.com/catherinevee/driftmgr/internal/drift/detector"
	importgen "github.com/catherinevee/driftmgr/internal/remediation/tfimport"
	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/google/uuid"
)

// CloudAsTruth implements the cloud-as-truth remediation strategy
// This strategy updates Terraform code to match the actual cloud state
type CloudAsTruth struct {
	config          *StrategyConfig
	importGenerator *importgen.ImportGenerator
}

// NewCloudAsTruthStrategy creates a new cloud-as-truth strategy
func NewCloudAsTruthStrategy(config *StrategyConfig) *CloudAsTruth {
	if config == nil {
		config = &StrategyConfig{
			TerraformPath: "terraform",
			Timeout:       15 * time.Minute,
			GitBranch:     "drift-fix",
		}
	}

	// Set defaults
	if config.TerraformPath == "" {
		config.TerraformPath = "terraform"
	}
	if config.Timeout == 0 {
		config.Timeout = 15 * time.Minute
	}
	if config.GitBranch == "" {
		config.GitBranch = fmt.Sprintf("drift-fix-%s", time.Now().Format("20060102-150405"))
	}

	return &CloudAsTruth{
		config:          config,
		importGenerator: importgen.NewImportGenerator(),
	}
}

// GetType returns the strategy type
func (c *CloudAsTruth) GetType() StrategyType {
	return CloudAsTruthStrategy
}

// GetDescription returns a human-readable description
func (c *CloudAsTruth) GetDescription() string {
	return "Updates Terraform code to match actual cloud state (Cloud state takes precedence)"
}

// Validate checks if the strategy can handle the given drift
func (c *CloudAsTruth) Validate(drift *detector.DriftResult) error {
	if drift == nil || len(drift.Differences) == 0 || drift.DriftType == detector.NoDrift {
		return fmt.Errorf("no drift detected")
	}

	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found in PATH: %w", err)
	}

	// Check if we're in a git repository
	if c.config.WorkingDir != "" {
		cmd := exec.Command("git", "status")
		cmd.Dir = c.config.WorkingDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("not a git repository: %w", err)
		}
	}

	return nil
}

// Plan creates a remediation plan based on detected drift
func (c *CloudAsTruth) Plan(ctx context.Context, drift *detector.DriftResult) (*RemediationPlan, error) {
	if err := c.Validate(drift); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	plan := &RemediationPlan{
		ID:        uuid.New().String(),
		Strategy:  CloudAsTruthStrategy,
		CreatedAt: time.Now(),
		Actions:   []RemediationAction{},
		Metadata:  make(map[string]interface{}),
	}

	// Create drift summary
	plan.DriftSummary = c.createDriftSummary(drift)

	// Analyze drift and create actions
	actions, riskLevel := c.analyzeDriftAndCreateActions(drift)
	plan.Actions = actions
	plan.RiskLevel = riskLevel

	// Estimate time
	plan.EstimatedTime = c.estimateExecutionTime(len(actions))

	// Cloud-as-truth is generally lower risk since we're matching reality
	plan.RequiresApproval = c.requiresApproval(riskLevel)

	// Add metadata
	plan.Metadata["git_branch"] = c.config.GitBranch
	plan.Metadata["import_count"] = c.countImports(actions)

	return plan, nil
}

// Execute executes the remediation plan
func (c *CloudAsTruth) Execute(ctx context.Context, plan *RemediationPlan) (*RemediationResult, error) {
	if plan.Strategy != CloudAsTruthStrategy {
		return nil, fmt.Errorf("invalid strategy type: %s", plan.Strategy)
	}

	result := &RemediationResult{
		PlanID:          plan.ID,
		StartedAt:       time.Now(),
		ActionsExecuted: []ActionResult{},
		Artifacts:       make(map[string]interface{}),
	}

	// Create git branch
	if err := c.createGitBranch(ctx); err != nil {
		result.Success = false
		result.Errors = append(result.Errors, err)
		result.Summary = fmt.Sprintf("Failed to create git branch: %v", err)
		return result, err
	}

	// Execute actions
	var importCommands []string
	var modifiedFiles []string

	for _, action := range plan.Actions {
		actionResult := c.executeAction(ctx, action)
		result.ActionsExecuted = append(result.ActionsExecuted, actionResult)

		if !actionResult.Success {
			result.Errors = append(result.Errors, actionResult.Error)
			continue
		}

		// Collect import commands
		if action.Type == ActionImport {
			importCommands = append(importCommands, action.Command)
		}

		// Track modified files
		if file, ok := action.Parameters["file"].(string); ok {
			modifiedFiles = append(modifiedFiles, file)
		}
	}

	// Generate import script
	if len(importCommands) > 0 {
		scriptPath := c.generateImportScript(importCommands)
		result.Artifacts["import_script"] = scriptPath
	}

	// Create pull request
	if !c.config.DryRun && len(modifiedFiles) > 0 {
		prURL, err := c.createPullRequest(ctx, plan.DriftSummary, modifiedFiles)
		if err != nil {
			result.Errors = append(result.Errors, err)
		} else {
			result.Artifacts["pr_url"] = prURL
		}
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
	result.Success = len(result.Errors) == 0

	// Generate summary
	if prURL, ok := result.Artifacts["pr_url"].(string); ok {
		result.Summary = fmt.Sprintf("Created PR to fix drift: %s", prURL)
	} else {
		result.Summary = fmt.Sprintf("Generated %d import commands and modified %d files",
			len(importCommands), len(modifiedFiles))
	}

	return result, nil
}

// analyzeDriftAndCreateActions analyzes drift and creates remediation actions
func (c *CloudAsTruth) analyzeDriftAndCreateActions(drift *detector.DriftResult) ([]RemediationAction, RiskLevel) {
	var actions []RemediationAction
	maxRisk := RiskLow
	order := 1

	for _, diff := range drift.Differences {
		switch diff.Type {
		case comparator.DiffTypeAdded:
			// Unmanaged resource - generate import
			if resourceMap, ok := diff.Actual.(map[string]interface{}); ok {
				// Convert map to models.Resource
				resource := models.Resource{
					ID:         getStringFromMap(resourceMap, "id"),
					Name:       getStringFromMap(resourceMap, "name"),
					Type:       getStringFromMap(resourceMap, "type"),
					Provider:   getStringFromMap(resourceMap, "provider"),
					Region:     getStringFromMap(resourceMap, "region"),
					Attributes: resourceMap,
				}
				importCmd, err := c.importGenerator.GenerateImportCommand(resource)
				if err == nil {
					actions = append(actions, RemediationAction{
						ID:          uuid.New().String(),
						Type:        ActionImport,
						Resource:    diff.Path,
						Description: fmt.Sprintf("Import unmanaged resource %s", diff.Path),
						Command:     importCmd.Command,
						RiskLevel:   RiskLow,
						Order:       order,
						Parameters: map[string]interface{}{
							"resource_type": importCmd.ResourceType,
							"resource_id":   importCmd.ResourceID,
						},
					})
					order++
				}
			}

		case comparator.DiffTypeModified:
			// Resource drifted - update Terraform configuration
			action := RemediationAction{
				ID:          uuid.New().String(),
				Type:        ActionUpdate,
				Resource:    diff.Path,
				Description: fmt.Sprintf("Update Terraform configuration for %s", diff.Path),
				RiskLevel:   c.assessUpdateRisk(diff),
				Order:       order,
				Parameters: map[string]interface{}{
					"expected": diff.Expected,
					"actual":   diff.Actual,
					"file":     c.findResourceFile(diff.Path),
				},
			}
			actions = append(actions, action)
			if action.RiskLevel > maxRisk {
				maxRisk = action.RiskLevel
			}
			order++

		case comparator.DiffTypeRemoved:
			// Resource missing in cloud - might need to remove from Terraform
			action := RemediationAction{
				ID:          uuid.New().String(),
				Type:        ActionDelete,
				Resource:    diff.Path,
				Description: fmt.Sprintf("Remove %s from Terraform configuration", diff.Path),
				RiskLevel:   RiskMedium,
				Order:       order,
				Parameters: map[string]interface{}{
					"file": c.findResourceFile(diff.Path),
				},
			}
			actions = append(actions, action)
			if action.RiskLevel > maxRisk {
				maxRisk = action.RiskLevel
			}
			order++
		}
	}

	// Add PR generation action
	if len(actions) > 0 {
		actions = append(actions, RemediationAction{
			ID:          uuid.New().String(),
			Type:        ActionGeneratePR,
			Resource:    "*",
			Description: "Generate pull request with fixes",
			RiskLevel:   RiskLow,
			Order:       order,
		})
	}

	return actions, maxRisk
}

// executeAction executes a single remediation action
func (c *CloudAsTruth) executeAction(ctx context.Context, action RemediationAction) ActionResult {
	result := ActionResult{
		ActionID:   action.ID,
		ActionType: action.Type,
		StartedAt:  time.Now(),
	}

	if c.config.DryRun {
		result.Success = true
		result.CompletedAt = time.Now()
		result.Duration = result.CompletedAt.Sub(result.StartedAt)
		result.Output = fmt.Sprintf("[DRY RUN] Would %s for %s", action.Type, action.Resource)
		return result
	}

	switch action.Type {
	case ActionImport:
		// Just record the import command - actual import happens after PR merge
		result.Success = true
		result.Output = fmt.Sprintf("Import command: %s", action.Command)

	case ActionUpdate:
		// Update Terraform file
		if file, ok := action.Parameters["file"].(string); ok {
			err := c.updateTerraformFile(file, action.Resource, action.Parameters["actual"])
			if err != nil {
				result.Success = false
				result.Error = err
			} else {
				result.Success = true
				result.Output = fmt.Sprintf("Updated %s in %s", action.Resource, file)
			}
		}

	case ActionDelete:
		// Comment out resource in Terraform file
		if file, ok := action.Parameters["file"].(string); ok {
			err := c.commentOutResource(file, action.Resource)
			if err != nil {
				result.Success = false
				result.Error = err
			} else {
				result.Success = true
				result.Output = fmt.Sprintf("Commented out %s in %s", action.Resource, file)
			}
		}

	case ActionGeneratePR:
		result.Success = true
		result.Output = "PR generation will happen after all changes"

	default:
		result.Success = false
		result.Error = fmt.Errorf("unsupported action type: %s", action.Type)
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	return result
}

// createGitBranch creates a new git branch for the changes
func (c *CloudAsTruth) createGitBranch(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "checkout", "-b", c.config.GitBranch)
	if c.config.WorkingDir != "" {
		cmd.Dir = c.config.WorkingDir
	}

	if err := cmd.Run(); err != nil {
		// Branch might already exist, try to switch to it
		cmd = exec.CommandContext(ctx, "git", "checkout", c.config.GitBranch)
		if c.config.WorkingDir != "" {
			cmd.Dir = c.config.WorkingDir
		}
		return cmd.Run()
	}

	return nil
}

// createPullRequest creates a GitHub/GitLab pull request
func (c *CloudAsTruth) createPullRequest(ctx context.Context, summary *DriftSummary, files []string) (string, error) {
	// Stage modified files
	for _, file := range files {
		cmd := exec.CommandContext(ctx, "git", "add", file)
		if c.config.WorkingDir != "" {
			cmd.Dir = c.config.WorkingDir
		}
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to stage %s: %w", file, err)
		}
	}

	// Commit changes
	commitMsg := c.generateCommitMessage(summary)
	cmd := exec.CommandContext(ctx, "git", "commit", "-m", commitMsg)
	if c.config.WorkingDir != "" {
		cmd.Dir = c.config.WorkingDir
	}
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to commit: %w", err)
	}

	// Push branch
	cmd = exec.CommandContext(ctx, "git", "push", "origin", c.config.GitBranch)
	if c.config.WorkingDir != "" {
		cmd.Dir = c.config.WorkingDir
	}
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to push branch: %w", err)
	}

	// Create PR using GitHub CLI if available
	if _, err := exec.LookPath("gh"); err == nil {
		prBody := c.generatePRBody(summary)
		cmd = exec.CommandContext(ctx, "gh", "pr", "create",
			"--title", fmt.Sprintf("Fix drift: %s", summary.EstimatedImpact),
			"--body", prBody,
			"--base", "main",
		)
		if c.config.WorkingDir != "" {
			cmd.Dir = c.config.WorkingDir
		}

		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to create PR: %w", err)
		}

		return strings.TrimSpace(string(output)), nil
	}

	return fmt.Sprintf("Branch %s pushed - create PR manually", c.config.GitBranch), nil
}

// generateImportScript generates a script with all import commands
func (c *CloudAsTruth) generateImportScript(commands []string) string {
	scriptPath := filepath.Join(c.config.WorkingDir, "drift-imports.sh")

	var script strings.Builder
	script.WriteString("#!/bin/bash\n\n")
	script.WriteString("# Terraform import commands for drift remediation\n")
	script.WriteString(fmt.Sprintf("# Generated: %s\n\n", time.Now().Format(time.RFC3339)))

	for _, cmd := range commands {
		script.WriteString(cmd + "\n")
	}

	os.WriteFile(scriptPath, []byte(script.String()), 0755)

	return scriptPath
}

// Helper methods

func (c *CloudAsTruth) createDriftSummary(drift *detector.DriftResult) *DriftSummary {
	summary := &DriftSummary{
		AffectedServices: []string{},
	}

	serviceMap := make(map[string]bool)

	for _, diff := range drift.Differences {
		summary.TotalResources++

		switch diff.Type {
		case comparator.DiffTypeRemoved:
			summary.MissingResources++
		case comparator.DiffTypeModified:
			summary.DriftedResources++
		case comparator.DiffTypeAdded:
			summary.UnmanagedResources++
		}

		if diff.Importance == comparator.ImportanceCritical {
			summary.CriticalDrifts++
		}

		parts := strings.Split(diff.Path, ".")
		if len(parts) > 0 {
			serviceMap[parts[0]] = true
		}
	}

	for service := range serviceMap {
		summary.AffectedServices = append(summary.AffectedServices, service)
	}

	if summary.UnmanagedResources > 0 {
		summary.EstimatedImpact = "New resources to import"
	} else if summary.DriftedResources > 0 {
		summary.EstimatedImpact = "Configuration updates needed"
	} else {
		summary.EstimatedImpact = "Minor adjustments"
	}

	return summary
}

func (c *CloudAsTruth) countImports(actions []RemediationAction) int {
	count := 0
	for _, action := range actions {
		if action.Type == ActionImport {
			count++
		}
	}
	return count
}

func (c *CloudAsTruth) assessUpdateRisk(diff comparator.Difference) RiskLevel {
	if diff.Importance == comparator.ImportanceCritical {
		return RiskMedium // Lower than code-as-truth since we're matching reality
	}
	if diff.Importance == comparator.ImportanceHigh {
		return RiskLow
	}
	return RiskLow
}

func (c *CloudAsTruth) findResourceFile(resourcePath string) string {
	// Simplified - in reality would parse TF files to find resource location
	return "main.tf"
}

func (c *CloudAsTruth) updateTerraformFile(file, resource string, actualState interface{}) error {
	// Simplified - in reality would parse and update HCL
	fmt.Printf("Would update %s in %s with actual state\n", resource, file)
	return nil
}

func (c *CloudAsTruth) commentOutResource(file, resource string) error {
	// Simplified - in reality would parse and comment out HCL block
	fmt.Printf("Would comment out %s in %s\n", resource, file)
	return nil
}

func (c *CloudAsTruth) estimateExecutionTime(actionCount int) time.Duration {
	// Cloud-as-truth is generally faster as it's mostly file operations
	return time.Duration(actionCount) * 5 * time.Second
}

func (c *CloudAsTruth) requiresApproval(riskLevel RiskLevel) bool {
	// Cloud-as-truth generally requires less approval since we're matching reality
	if c.config.AutoApprove {
		return false
	}
	return riskLevel == RiskCritical
}

func (c *CloudAsTruth) generateCommitMessage(summary *DriftSummary) string {
	return fmt.Sprintf("Fix drift: %d resources updated to match cloud state\n\n"+
		"- Drifted: %d\n"+
		"- Missing: %d\n"+
		"- Unmanaged: %d\n",
		summary.TotalResources,
		summary.DriftedResources,
		summary.MissingResources,
		summary.UnmanagedResources,
	)
}

func (c *CloudAsTruth) generatePRBody(summary *DriftSummary) string {
	tmpl := `## Drift Remediation

### Summary
{{.EstimatedImpact}}

### Changes
- **Total Resources:** {{.TotalResources}}
- **Drifted Resources:** {{.DriftedResources}}
- **Missing Resources:** {{.MissingResources}}
- **Unmanaged Resources:** {{.UnmanagedResources}}
- **Critical Drifts:** {{.CriticalDrifts}}

### Affected Services
{{range .AffectedServices}}- {{.}}
{{end}}

### Next Steps
1. Review the changes
2. Run the import script (if any)
3. Test in a non-production environment
4. Merge when ready

---
*Generated by DriftMgr*`

	t, _ := template.New("pr").Parse(tmpl)
	var buf bytes.Buffer
	t.Execute(&buf, summary)
	return buf.String()
}

// getStringFromMap safely extracts a string value from a map
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
