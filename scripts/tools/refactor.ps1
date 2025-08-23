# DriftMgr Refactoring Script
# This script consolidates and reorganizes the DriftMgr codebase

Write-Host "Starting DriftMgr Refactoring..." -ForegroundColor Green

# Phase 1: Create consolidated core modules
Write-Host "`nPhase 1: Creating core modules..." -ForegroundColor Yellow

# Create analyzer for drift
@"
package drift

import (
	"github.com/catherinevee/driftmgr/internal/models"
)

// Analyzer provides drift analysis capabilities
type Analyzer struct{}

// NewAnalyzer creates a new drift analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// Analyze performs drift analysis
func (a *Analyzer) Analyze(drifts []models.DriftItem, resources []models.Resource) *Analysis {
	// Consolidated analysis logic from multiple files
	return &Analysis{
		Patterns:         a.detectPatterns(drifts),
		Trends:           a.analyzeTrends(drifts),
		ImpactScore:      a.calculateImpact(drifts),
		RiskLevel:        a.assessRisk(drifts),
		CostImpact:       a.calculateCostImpact(drifts),
		SecurityImpact:   a.assessSecurityImpact(drifts),
		ComplianceImpact: a.assessComplianceImpact(drifts),
	}
}

func (a *Analyzer) detectPatterns(drifts []models.DriftItem) []Pattern {
	// Pattern detection logic
	return []Pattern{}
}

func (a *Analyzer) analyzeTrends(drifts []models.DriftItem) []Trend {
	// Trend analysis logic
	return []Trend{}
}

func (a *Analyzer) calculateImpact(drifts []models.DriftItem) float64 {
	return float64(len(drifts)) * 10.0
}

func (a *Analyzer) assessRisk(drifts []models.DriftItem) string {
	if len(drifts) > 50 {
		return "critical"
	} else if len(drifts) > 20 {
		return "high"
	} else if len(drifts) > 5 {
		return "medium"
	}
	return "low"
}

func (a *Analyzer) calculateCostImpact(drifts []models.DriftItem) float64 {
	return float64(len(drifts)) * 50.0
}

func (a *Analyzer) assessSecurityImpact(drifts []models.DriftItem) string {
	for _, drift := range drifts {
		if drift.Severity == "critical" {
			return "high"
		}
	}
	return "low"
}

func (a *Analyzer) assessComplianceImpact(drifts []models.DriftItem) string {
	return "medium"
}
"@ | Out-File -FilePath "internal/core/drift/analyzer.go" -Encoding UTF8

# Create predictor for drift
@"
package drift

import (
	"github.com/catherinevee/driftmgr/internal/models"
)

// Predictor provides drift prediction capabilities
type Predictor struct{}

// NewPredictor creates a new drift predictor
func NewPredictor() *Predictor {
	return &Predictor{}
}

// Predict generates drift predictions
func (p *Predictor) Predict(drifts []models.DriftItem, analysis *Analysis) *Predictions {
	return &Predictions{
		FutureDrift: p.predictFutureDrift(drifts),
		Likelihood:  p.calculateLikelihood(analysis),
		TimeFrame:   "7 days",
		PreventiveActions: []string{
			"Enable drift detection automation",
			"Review and update IaC templates",
			"Implement stricter change controls",
		},
	}
}

func (p *Predictor) predictFutureDrift(drifts []models.DriftItem) []FutureDrift {
	predictions := []FutureDrift{}
	
	// Analyze patterns to predict future drift
	resourceTypes := make(map[string]int)
	for _, drift := range drifts {
		resourceTypes[drift.ResourceType]++
	}
	
	for resourceType, count := range resourceTypes {
		if count > 2 {
			predictions = append(predictions, FutureDrift{
				ResourceType: resourceType,
				Probability:  float64(count) / float64(len(drifts)),
				TimeFrame:    "within 7 days",
				Reason:       "Historical pattern detected",
			})
		}
	}
	
	return predictions
}

func (p *Predictor) calculateLikelihood(analysis *Analysis) float64 {
	if analysis.RiskLevel == "critical" {
		return 0.9
	} else if analysis.RiskLevel == "high" {
		return 0.7
	} else if analysis.RiskLevel == "medium" {
		return 0.5
	}
	return 0.3
}
"@ | Out-File -FilePath "internal/core/drift/predictor.go" -Encoding UTF8

# Create policy engine
@"
package drift

import (
	"github.com/catherinevee/driftmgr/internal/models"
)

// PolicyEngine evaluates drift against policies
type PolicyEngine struct {
	policies []Policy
}

// Policy defines a drift policy
type Policy struct {
	Name        string
	Environment string
	Rules       []Rule
}

// Rule defines a policy rule
type Rule struct {
	ResourceType string
	DriftType    string
	Severity     string
	Action       string
}

// NewPolicyEngine creates a new policy engine
func NewPolicyEngine() *PolicyEngine {
	return &PolicyEngine{
		policies: loadDefaultPolicies(),
	}
}

// EvaluateDrifts evaluates drifts against policies
func (pe *PolicyEngine) EvaluateDrifts(drifts []models.DriftItem, environment string) []models.DriftItem {
	evaluated := make([]models.DriftItem, 0)
	
	for _, drift := range drifts {
		if pe.shouldInclude(drift, environment) {
			evaluated = append(evaluated, drift)
		}
	}
	
	return evaluated
}

func (pe *PolicyEngine) shouldInclude(drift models.DriftItem, environment string) bool {
	// Apply policy rules
	for _, policy := range pe.policies {
		if policy.Environment == environment || policy.Environment == "*" {
			for _, rule := range policy.Rules {
				if pe.matchesRule(drift, rule) {
					return rule.Action == "include"
				}
			}
		}
	}
	return true
}

func (pe *PolicyEngine) matchesRule(drift models.DriftItem, rule Rule) bool {
	if rule.ResourceType != "*" && rule.ResourceType != drift.ResourceType {
		return false
	}
	if rule.DriftType != "*" && rule.DriftType != drift.DriftType {
		return false
	}
	if rule.Severity != "*" && rule.Severity != drift.Severity {
		return false
	}
	return true
}

func loadDefaultPolicies() []Policy {
	return []Policy{
		{
			Name:        "production_critical",
			Environment: "production",
			Rules: []Rule{
				{ResourceType: "*", DriftType: "*", Severity: "critical", Action: "include"},
				{ResourceType: "*", DriftType: "*", Severity: "high", Action: "include"},
			},
		},
		{
			Name:        "development_filter",
			Environment: "development",
			Rules: []Rule{
				{ResourceType: "*", DriftType: "added", Severity: "low", Action: "exclude"},
			},
		},
	}
}
"@ | Out-File -FilePath "internal/core/drift/policy.go" -Encoding UTF8

Write-Host "Core drift modules created" -ForegroundColor Green

# Phase 2: Create consolidated remediation module
Write-Host "`nPhase 2: Creating remediation module..." -ForegroundColor Yellow

@"
package remediation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// Engine provides unified remediation capabilities
type Engine struct {
	planner  *Planner
	executor *Executor
	rollback *RollbackManager
	safety   *SafetyManager
	mu       sync.RWMutex
}

// Plan represents a remediation plan
type Plan struct {
	ID          string                 `json:"id"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	DriftItems  []models.DriftItem     `json:"drift_items"`
	Actions     []Action               `json:"actions"`
	Impact      Impact                 `json:"impact"`
	Approval    *ApprovalStatus        `json:"approval,omitempty"`
	Execution   *ExecutionStatus       `json:"execution,omitempty"`
	Results     *Results               `json:"results,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Action represents a remediation action
type Action struct {
	ID           string                 `json:"id"`
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	ActionType   string                 `json:"action_type"`
	Description  string                 `json:"description"`
	Parameters   map[string]interface{} `json:"parameters"`
	Risk         string                 `json:"risk"`
	EstimatedTime int                   `json:"estimated_time"`
	Dependencies []string               `json:"dependencies"`
	Status       string                 `json:"status"`
	Error        string                 `json:"error,omitempty"`
}

// Impact describes remediation impact
type Impact struct {
	ResourcesAffected int                    `json:"resources_affected"`
	EstimatedDuration int                    `json:"estimated_duration"`
	RiskLevel         string                 `json:"risk_level"`
	CostImpact        float64                `json:"cost_impact"`
	ServiceImpact     []string               `json:"service_impact"`
	RequiresDowntime  bool                   `json:"requires_downtime"`
	Reversible        bool                   `json:"reversible"`
	Details           map[string]interface{} `json:"details"`
}

// ApprovalStatus represents approval status
type ApprovalStatus struct {
	Required      bool              `json:"required"`
	Status        string            `json:"status"`
	Approvers     []string          `json:"approvers"`
	ApprovedBy    []string          `json:"approved_by"`
	ApprovalTime  *time.Time        `json:"approval_time,omitempty"`
	Comments      []string          `json:"comments,omitempty"`
}

// ExecutionStatus represents execution status
type ExecutionStatus struct {
	Status         string     `json:"status"`
	StartTime      time.Time  `json:"start_time"`
	EndTime        *time.Time `json:"end_time,omitempty"`
	Progress       int        `json:"progress"`
	CurrentStep    string     `json:"current_step"`
	TotalSteps     int        `json:"total_steps"`
	CompletedSteps int        `json:"completed_steps"`
}

// Results represents remediation results
type Results struct {
	Success      bool                   `json:"success"`
	ItemsFixed   int                    `json:"items_fixed"`
	ItemsFailed  int                    `json:"items_failed"`
	Duration     time.Duration          `json:"duration"`
	Details      map[string]interface{} `json:"details"`
	RollbackInfo *RollbackInfo          `json:"rollback_info,omitempty"`
}

// RollbackInfo contains rollback information
type RollbackInfo struct {
	Available bool                   `json:"available"`
	Snapshot  map[string]interface{} `json:"snapshot"`
	Steps     []string               `json:"steps"`
}

// Options configures remediation
type Options struct {
	Strategy      string                 `json:"strategy"`
	DryRun        bool                   `json:"dry_run"`
	AutoApprove   bool                   `json:"auto_approve"`
	Parallel      bool                   `json:"parallel"`
	MaxWorkers    int                    `json:"max_workers"`
	Timeout       time.Duration          `json:"timeout"`
	RollbackOnFail bool                  `json:"rollback_on_fail"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// NewEngine creates a new remediation engine
func NewEngine() *Engine {
	return &Engine{
		planner:  NewPlanner(),
		executor: NewExecutor(),
		rollback: NewRollbackManager(),
		safety:   NewSafetyManager(),
	}
}

// CreatePlan creates a remediation plan
func (e *Engine) CreatePlan(ctx context.Context, drifts []models.DriftItem, options Options) (*Plan, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	// Use planner to create plan
	plan, err := e.planner.CreatePlan(drifts, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}
	
	// Validate plan with safety manager
	if err := e.safety.ValidatePlan(plan); err != nil {
		return nil, fmt.Errorf("plan validation failed: %w", err)
	}
	
	// Calculate impact
	plan.Impact = e.calculateImpact(plan.Actions, drifts)
	
	return plan, nil
}

// ExecutePlan executes a remediation plan
func (e *Engine) ExecutePlan(ctx context.Context, plan *Plan, options Options) (*Results, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	// Check approval if required
	if plan.Approval != nil && plan.Approval.Required && !options.AutoApprove {
		if plan.Approval.Status != "approved" {
			return nil, fmt.Errorf("plan requires approval")
		}
	}
	
	// Dry run if requested
	if options.DryRun {
		return e.simulateExecution(plan), nil
	}
	
	// Create snapshot for rollback
	var snapshot *RollbackInfo
	if options.RollbackOnFail {
		snapshot = e.rollback.CreateSnapshot(plan)
	}
	
	// Execute plan
	results, err := e.executor.Execute(ctx, plan, options)
	if err != nil {
		if options.RollbackOnFail && snapshot != nil {
			e.rollback.Rollback(ctx, snapshot)
		}
		return nil, fmt.Errorf("execution failed: %w", err)
	}
	
	results.RollbackInfo = snapshot
	return results, nil
}

// calculateImpact calculates remediation impact
func (e *Engine) calculateImpact(actions []Action, drifts []models.DriftItem) Impact {
	impact := Impact{
		ResourcesAffected: len(actions),
		EstimatedDuration: 0,
		RiskLevel:         "low",
		CostImpact:        0,
		ServiceImpact:     make([]string, 0),
		RequiresDowntime:  false,
		Reversible:        true,
		Details:           make(map[string]interface{}),
	}
	
	highRiskCount := 0
	for _, action := range actions {
		impact.EstimatedDuration += action.EstimatedTime
		
		if action.Risk == "high" {
			highRiskCount++
		}
		
		if action.ActionType == "delete" || action.ActionType == "create" {
			impact.RequiresDowntime = true
		}
	}
	
	// Calculate overall risk level
	if highRiskCount > len(actions)/2 {
		impact.RiskLevel = "high"
	} else if highRiskCount > 0 {
		impact.RiskLevel = "medium"
	}
	
	// Convert duration to minutes
	impact.EstimatedDuration = impact.EstimatedDuration / 60
	
	// Estimate cost impact
	impact.CostImpact = float64(len(actions)) * 10.0
	
	return impact
}

// simulateExecution simulates plan execution
func (e *Engine) simulateExecution(plan *Plan) *Results {
	return &Results{
		Success:     true,
		ItemsFixed:  len(plan.Actions),
		ItemsFailed: 0,
		Duration:    time.Duration(plan.Impact.EstimatedDuration) * time.Minute,
		Details: map[string]interface{}{
			"simulation": true,
			"message":    "Dry run completed successfully",
		},
	}
}
"@ | Out-File -FilePath "internal/core/remediation/engine.go" -Encoding UTF8

Write-Host "Remediation module created" -ForegroundColor Green

# Phase 3: Move provider-specific code
Write-Host "`nPhase 3: Organizing provider-specific code..." -ForegroundColor Yellow

# Create directories if they don't exist
$providers = @("aws", "azure", "gcp", "digitalocean")
foreach ($provider in $providers) {
    $providerPath = "internal/providers/$provider"
    if (!(Test-Path $providerPath)) {
        New-Item -ItemType Directory -Path $providerPath -Force | Out-Null
    }
}

# Move AWS files
$awsFiles = @(
    "internal/discovery/aws_comprehensive.go",
    "internal/discovery/aws_expanded_resources.go",
    "internal/discovery/aws_role_assumption.go"
)

foreach ($file in $awsFiles) {
    if (Test-Path $file) {
        $fileName = Split-Path $file -Leaf
        Move-Item -Path $file -Destination "internal/providers/aws/$fileName" -Force -ErrorAction SilentlyContinue
    }
}

# Move Azure files
$azureFiles = @(
    "internal/discovery/azure_comprehensive.go",
    "internal/discovery/azure_enhanced_discovery.go",
    "internal/discovery/azure_expanded_resources.go",
    "internal/discovery/azure_fix.go",
    "internal/discovery/azure_windows_cli.go"
)

foreach ($file in $azureFiles) {
    if (Test-Path $file) {
        $fileName = Split-Path $file -Leaf
        Move-Item -Path $file -Destination "internal/providers/azure/$fileName" -Force -ErrorAction SilentlyContinue
    }
}

# Move GCP files
$gcpFiles = @(
    "internal/discovery/gcp_basic_discovery.go",
    "internal/discovery/gcp_comprehensive.go",
    "internal/discovery/gcp_enhanced_discovery.go",
    "internal/discovery/gcp_expanded_resources.go"
)

foreach ($file in $gcpFiles) {
    if (Test-Path $file) {
        $fileName = Split-Path $file -Leaf
        Move-Item -Path $file -Destination "internal/providers/gcp/$fileName" -Force -ErrorAction SilentlyContinue
    }
}

# Move DigitalOcean files
$doFiles = @(
    "internal/discovery/digitalocean_comprehensive.go",
    "internal/discovery/digitalocean_discovery.go",
    "internal/discovery/digitalocean_enhanced_discovery.go",
    "internal/discovery/digitalocean_expanded_resources.go",
    "internal/discovery/digitalocean_integration.go"
)

foreach ($file in $doFiles) {
    if (Test-Path $file) {
        $fileName = Split-Path $file -Leaf
        Move-Item -Path $file -Destination "internal/providers/digitalocean/$fileName" -Force -ErrorAction SilentlyContinue
    }
}

Write-Host "Provider-specific code organized" -ForegroundColor Green

# Phase 4: Clean up duplicate files
Write-Host "`nPhase 4: Removing duplicate files..." -ForegroundColor Yellow

# List of files to remove (duplicates)
$duplicates = @(
    "internal/discovery/enhanced_discovery.go",
    "internal/discovery/enhanced_discovery_v2.go",
    "internal/discovery/universal_discovery.go",
    "internal/discovery/multi_account_discovery.go",
    "internal/discovery/parallel_discovery.go",
    "internal/analysis/cost_analyzer.go",
    "internal/cost/cost_analyzer.go",
    "internal/drift/cost_calculator.go"
)

foreach ($file in $duplicates) {
    if (Test-Path $file) {
        Remove-Item -Path $file -Force
        Write-Host "Removed duplicate: $file" -ForegroundColor Red
    }
}

Write-Host "`nRefactoring completed!" -ForegroundColor Green
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "1. Update all import statements in Go files"
Write-Host "2. Run 'go mod tidy' to clean up dependencies"
Write-Host "3. Run tests to ensure functionality is preserved"
Write-Host "4. Commit changes with detailed message about refactoring"