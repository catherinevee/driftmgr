package remediation

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SafetyCheckEngine performs safety checks before remediation
type SafetyCheckEngine struct {
	dryRunMode        bool
	rollbackEnabled   bool
	impactAnalyzer    *ImpactAnalyzer
	dependencyManager *DependencyManager
	changePreview     *ChangePreview
	validationRules   []ValidationRule
	safetyThresholds  SafetyThresholds
}

// ImpactAnalyzer analyzes the impact of remediation changes
type ImpactAnalyzer struct {
	criticalResources map[string]bool
	impactScores      map[string]float64
	cascadeEffects    map[string][]string
}

// DependencyManager manages resource dependencies for safe remediation
type DependencyManager struct {
	dependencies   map[string][]string
	reverseDeps    map[string][]string
	orderingRules  []OrderingRule
}

// ChangePreview provides detailed preview of changes
type ChangePreview struct {
	changes      []Change
	affectedAPIs []string
	estimatedTime time.Duration
	riskLevel    string
}

// Change represents a single remediation change
type Change struct {
	ResourceID   string
	ResourceType string
	ChangeType   string // create, update, delete, replace
	Before       map[string]interface{}
	After        map[string]interface{}
	Impact       ImpactAssessment
	Dependencies []string
}

// ImpactAssessment represents the impact of a change
type ImpactAssessment struct {
	Severity          string
	AffectedResources []string
	ServiceDisruption bool
	DataLossRisk      bool
	SecurityImpact    bool
	CostImpact        float64
	ComplianceImpact  []string
}

// ValidationRule defines a rule for validating remediation safety
type ValidationRule struct {
	Name        string
	Description string
	Validate    func(change Change) (bool, string)
}

// SafetyThresholds defines thresholds for safety checks
type SafetyThresholds struct {
	MaxChangesPerRun      int
	MaxCriticalChanges    int
	MaxCostIncrease       float64
	MaxDowntimeMinutes    int
	RequireApproval       []string // Resource types requiring approval
	BlockedResourceTypes  []string
	BlockedTimeWindows    []BlockedTimeWindow
}

// BlockedTimeWindow represents a time window for blocking changes
type BlockedTimeWindow struct {
	Start     time.Time
	End       time.Time
	Reason    string
	Recurring bool
}

// OrderingRule defines rules for ordering remediation actions
type OrderingRule struct {
	ResourceType string
	Priority     int
	MustBefore   []string
	MustAfter    []string
}

// RollbackPlan represents a plan to rollback changes
type RollbackPlan struct {
	ID           string
	OriginalState map[string]interface{}
	Changes      []Change
	Checkpoints  []Checkpoint
	CreatedAt    time.Time
}

// Checkpoint represents a rollback checkpoint
type Checkpoint struct {
	ID          string
	Timestamp   time.Time
	State       map[string]interface{}
	Reversible  bool
	Description string
}

// NewSafetyCheckEngine creates a new safety check engine
func NewSafetyCheckEngine() *SafetyCheckEngine {
	return &SafetyCheckEngine{
		dryRunMode:        false,
		rollbackEnabled:   true,
		impactAnalyzer:    NewImpactAnalyzer(),
		dependencyManager: NewDependencyManager(),
		changePreview:     &ChangePreview{},
		validationRules:   getDefaultValidationRules(),
		safetyThresholds:  getDefaultSafetyThresholds(),
	}
}

// NewImpactAnalyzer creates a new impact analyzer
func NewImpactAnalyzer() *ImpactAnalyzer {
	return &ImpactAnalyzer{
		criticalResources: make(map[string]bool),
		impactScores:      make(map[string]float64),
		cascadeEffects:    make(map[string][]string),
	}
}

// NewDependencyManager creates a new dependency manager
func NewDependencyManager() *DependencyManager {
	return &DependencyManager{
		dependencies:  make(map[string][]string),
		reverseDeps:   make(map[string][]string),
		orderingRules: getDefaultOrderingRules(),
	}
}

// EnableDryRun enables dry run mode
func (s *SafetyCheckEngine) EnableDryRun() {
	s.dryRunMode = true
}

// DisableDryRun disables dry run mode
func (s *SafetyCheckEngine) DisableDryRun() {
	s.dryRunMode = false
}

// PerformSafetyChecks performs all safety checks for a remediation plan
func (s *SafetyCheckEngine) PerformSafetyChecks(ctx context.Context, plan *RemediationPlan) (*EnhancedSafetyCheckResult, error) {
	result := &EnhancedSafetyCheckResult{
		Passed:       true,
		Warnings:     []string{},
		Errors:       []string{},
		RequiredApprovals: []ApprovalRequirement{},
	}

	// Check if in blocked time window
	if s.isInBlockedTimeWindow() {
		result.Passed = false
		result.Errors = append(result.Errors, "Remediation blocked during current time window")
		return result, nil
	}

	// Generate change preview
	preview, err := s.generateChangePreview(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("failed to generate change preview: %w", err)
	}
	result.Preview = preview

	// Check thresholds
	if err := s.checkThresholds(preview, result); err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, err.Error())
	}

	// Analyze impact
	impacts, err := s.analyzeImpact(ctx, preview)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze impact: %w", err)
	}
	result.ImpactAnalysis = impacts

	// Check dependencies
	depErrors := s.checkDependencies(preview)
	if len(depErrors) > 0 {
		result.Passed = false
		result.Errors = append(result.Errors, depErrors...)
	}

	// Validate changes
	validationErrors := s.validateChanges(preview)
	if len(validationErrors) > 0 {
		result.Passed = false
		result.Errors = append(result.Errors, validationErrors...)
	}

	// Check for required approvals
	approvals := s.checkApprovalRequirements(preview)
	if len(approvals) > 0 {
		result.RequiredApprovals = approvals
		if !s.dryRunMode {
			result.Passed = false
			result.Errors = append(result.Errors, "Manual approval required")
		}
	}

	// Generate rollback plan if needed
	if s.rollbackEnabled && result.Passed {
		rollbackPlan, err := s.generateRollbackPlan(ctx, preview)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to generate rollback plan: %v", err))
		} else {
			result.RollbackPlan = rollbackPlan
		}
	}

	return result, nil
}

// generateChangePreview generates a preview of changes
func (s *SafetyCheckEngine) generateChangePreview(ctx context.Context, plan *RemediationPlan) (*ChangePreview, error) {
	preview := &ChangePreview{
		changes:       []Change{},
		affectedAPIs:  []string{},
		estimatedTime: 0,
		riskLevel:     "low",
	}

	for _, action := range plan.Actions {
		change := Change{
			ResourceID:   action.ResourceID,
			ResourceType: action.ResourceType,
			ChangeType:   action.ActionType,
			Before:       action.CurrentState,
			After:        action.DesiredState,
			Dependencies: s.dependencyManager.GetDependencies(action.ResourceID),
		}

		// Analyze impact for this change
		impact := s.impactAnalyzer.AnalyzeChangeImpact(change)
		change.Impact = impact

		preview.changes = append(preview.changes, change)

		// Update estimated time
		preview.estimatedTime += s.estimateChangeTime(change)

		// Update risk level
		if impact.Severity == "critical" {
			preview.riskLevel = "high"
		} else if impact.Severity == "high" && preview.riskLevel != "high" {
			preview.riskLevel = "medium"
		}

		// Track affected APIs
		apis := s.getAffectedAPIs(change)
		preview.affectedAPIs = append(preview.affectedAPIs, apis...)
	}

	return preview, nil
}

// AnalyzeChangeImpact analyzes the impact of a single change
func (a *ImpactAnalyzer) AnalyzeChangeImpact(change Change) ImpactAssessment {
	impact := ImpactAssessment{
		Severity:          "low",
		AffectedResources: []string{},
		ServiceDisruption: false,
		DataLossRisk:      false,
		SecurityImpact:    false,
		CostImpact:        0.0,
		ComplianceImpact:  []string{},
	}

	// Check if critical resource
	if a.criticalResources[change.ResourceID] {
		impact.Severity = "critical"
		impact.ServiceDisruption = true
	}

	// Check for data loss risk
	if change.ChangeType == "delete" || change.ChangeType == "replace" {
		if isStatefulResource(change.ResourceType) {
			impact.DataLossRisk = true
			impact.Severity = "high"
		}
	}

	// Check security impact
	if isSecurityResource(change.ResourceType) {
		impact.SecurityImpact = true
		if impact.Severity == "low" {
			impact.Severity = "medium"
		}
	}

	// Analyze cascade effects
	if cascadeEffects, exists := a.cascadeEffects[change.ResourceID]; exists {
		impact.AffectedResources = cascadeEffects
		if len(cascadeEffects) > 5 {
			impact.Severity = "high"
		}
	}

	// Estimate cost impact
	impact.CostImpact = a.estimateCostImpact(change)

	// Check compliance impact
	impact.ComplianceImpact = a.checkComplianceImpact(change)

	return impact
}

// estimateCostImpact estimates the cost impact of a change
func (a *ImpactAnalyzer) estimateCostImpact(change Change) float64 {
	// Simplified cost estimation
	costMap := map[string]float64{
		"aws_instance":        100.0,
		"aws_rds_instance":    200.0,
		"aws_eks_cluster":     500.0,
		"azure_vm":            100.0,
		"azure_sql_database":  150.0,
		"gcp_compute_instance": 100.0,
	}

	baseCost := costMap[change.ResourceType]
	
	switch change.ChangeType {
	case "create":
		return baseCost
	case "delete":
		return -baseCost
	case "update":
		// Check if instance size is changing
		if change.Before["instance_type"] != change.After["instance_type"] {
			return baseCost * 0.5 // Assume 50% cost change
		}
		return 0
	case "replace":
		return baseCost * 0.1 // Temporary double cost during replacement
	default:
		return 0
	}
}

// checkComplianceImpact checks compliance impact of a change
func (a *ImpactAnalyzer) checkComplianceImpact(change Change) []string {
	var impacts []string

	// Check for compliance-related resources
	if change.ResourceType == "aws_security_group" || change.ResourceType == "azure_network_security_group" {
		impacts = append(impacts, "Network Security Compliance")
	}

	if change.ResourceType == "aws_iam_role" || change.ResourceType == "azure_role_assignment" {
		impacts = append(impacts, "Access Control Compliance")
	}

	// Check for encryption changes
	if before, ok := change.Before["encrypted"].(bool); ok {
		if after, ok := change.After["encrypted"].(bool); ok {
			if before && !after {
				impacts = append(impacts, "Data Encryption Compliance")
			}
		}
	}

	return impacts
}

// GetDependencies returns dependencies for a resource
func (d *DependencyManager) GetDependencies(resourceID string) []string {
	return d.dependencies[resourceID]
}

// AddDependency adds a dependency relationship
func (d *DependencyManager) AddDependency(resourceID, dependsOn string) {
	d.dependencies[resourceID] = append(d.dependencies[resourceID], dependsOn)
	d.reverseDeps[dependsOn] = append(d.reverseDeps[dependsOn], resourceID)
}

// GetRemediationOrder returns the order in which resources should be remediated
func (d *DependencyManager) GetRemediationOrder(changes []Change) ([]Change, error) {
	// Topological sort based on dependencies
	visited := make(map[string]bool)
	stack := []Change{}
	
	var visit func(change Change) error
	visit = func(change Change) error {
		if visited[change.ResourceID] {
			return nil
		}
		visited[change.ResourceID] = true

		// Visit dependencies first
		for _, dep := range d.dependencies[change.ResourceID] {
			for _, c := range changes {
				if c.ResourceID == dep {
					if err := visit(c); err != nil {
						return err
					}
				}
			}
		}

		stack = append(stack, change)
		return nil
	}

	for _, change := range changes {
		if err := visit(change); err != nil {
			return nil, err
		}
	}

	// Reverse the stack to get correct order
	ordered := make([]Change, len(stack))
	for i := range stack {
		ordered[len(stack)-1-i] = stack[i]
	}

	return ordered, nil
}

// checkThresholds checks if changes exceed safety thresholds
func (s *SafetyCheckEngine) checkThresholds(preview *ChangePreview, result *EnhancedSafetyCheckResult) error {
	if len(preview.changes) > s.safetyThresholds.MaxChangesPerRun {
		return fmt.Errorf("too many changes (%d > %d)", len(preview.changes), s.safetyThresholds.MaxChangesPerRun)
	}

	criticalCount := 0
	totalCostIncrease := 0.0

	for _, change := range preview.changes {
		if change.Impact.Severity == "critical" {
			criticalCount++
		}
		totalCostIncrease += change.Impact.CostImpact
	}

	if criticalCount > s.safetyThresholds.MaxCriticalChanges {
		return fmt.Errorf("too many critical changes (%d > %d)", criticalCount, s.safetyThresholds.MaxCriticalChanges)
	}

	if totalCostIncrease > s.safetyThresholds.MaxCostIncrease {
		return fmt.Errorf("cost increase too high ($%.2f > $%.2f)", totalCostIncrease, s.safetyThresholds.MaxCostIncrease)
	}

	return nil
}

// analyzeImpact performs comprehensive impact analysis
func (s *SafetyCheckEngine) analyzeImpact(ctx context.Context, preview *ChangePreview) ([]ImpactAssessment, error) {
	var impacts []ImpactAssessment

	for _, change := range preview.changes {
		impacts = append(impacts, change.Impact)
	}

	return impacts, nil
}

// checkDependencies checks for dependency violations
func (s *SafetyCheckEngine) checkDependencies(preview *ChangePreview) []string {
	var errors []string

	// Check for circular dependencies
	for _, change := range preview.changes {
		if s.hasCircularDependency(change.ResourceID, change.Dependencies) {
			errors = append(errors, fmt.Sprintf("Circular dependency detected for resource %s", change.ResourceID))
		}
	}

	// Check for missing dependencies
	resourceMap := make(map[string]bool)
	for _, change := range preview.changes {
		resourceMap[change.ResourceID] = true
	}

	for _, change := range preview.changes {
		for _, dep := range change.Dependencies {
			if !resourceMap[dep] {
				errors = append(errors, fmt.Sprintf("Missing dependency %s for resource %s", dep, change.ResourceID))
			}
		}
	}

	return errors
}

// hasCircularDependency checks for circular dependencies
func (s *SafetyCheckEngine) hasCircularDependency(resourceID string, dependencies []string) bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(node string) bool
	hasCycle = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		deps := s.dependencyManager.GetDependencies(node)
		for _, dep := range deps {
			if !visited[dep] {
				if hasCycle(dep) {
					return true
				}
			} else if recStack[dep] {
				return true
			}
		}

		recStack[node] = false
		return false
	}

	return hasCycle(resourceID)
}

// validateChanges validates changes against rules
func (s *SafetyCheckEngine) validateChanges(preview *ChangePreview) []string {
	var errors []string

	for _, change := range preview.changes {
		for _, rule := range s.validationRules {
			if valid, err := rule.Validate(change); !valid {
				errors = append(errors, fmt.Sprintf("%s: %s", rule.Name, err))
			}
		}
	}

	return errors
}

// checkApprovalRequirements checks if manual approval is required
func (s *SafetyCheckEngine) checkApprovalRequirements(preview *ChangePreview) []ApprovalRequirement {
	var approvals []ApprovalRequirement

	for _, change := range preview.changes {
		// Check if resource type requires approval
		for _, requiredType := range s.safetyThresholds.RequireApproval {
			if change.ResourceType == requiredType {
				approvals = append(approvals, ApprovalRequirement{
					ResourceID:   change.ResourceID,
					ResourceType: change.ResourceType,
					ChangeType:   change.ChangeType,
					Reason:       "Resource type requires manual approval",
					Severity:     change.Impact.Severity,
				})
			}
		}

		// Check if critical impact requires approval
		if change.Impact.Severity == "critical" {
			approvals = append(approvals, ApprovalRequirement{
				ResourceID:   change.ResourceID,
				ResourceType: change.ResourceType,
				ChangeType:   change.ChangeType,
				Reason:       "Critical impact detected",
				Severity:     "critical",
			})
		}

		// Check if data loss risk requires approval
		if change.Impact.DataLossRisk {
			approvals = append(approvals, ApprovalRequirement{
				ResourceID:   change.ResourceID,
				ResourceType: change.ResourceType,
				ChangeType:   change.ChangeType,
				Reason:       "Potential data loss risk",
				Severity:     "high",
			})
		}
	}

	return approvals
}

// generateRollbackPlan generates a rollback plan
func (s *SafetyCheckEngine) generateRollbackPlan(ctx context.Context, preview *ChangePreview) (*RollbackPlan, error) {
	plan := &RollbackPlan{
		ID:            fmt.Sprintf("rollback-%d", time.Now().Unix()),
		OriginalState: make(map[string]interface{}),
		Changes:       preview.changes,
		Checkpoints:   []Checkpoint{},
		CreatedAt:     time.Now(),
	}

	// Save original state for each resource
	for _, change := range preview.changes {
		plan.OriginalState[change.ResourceID] = change.Before
		
		// Create checkpoint
		checkpoint := Checkpoint{
			ID:          fmt.Sprintf("checkpoint-%s-%d", change.ResourceID, time.Now().Unix()),
			Timestamp:   time.Now(),
			State:       change.Before,
			Reversible:  s.isReversible(change),
			Description: fmt.Sprintf("Checkpoint for %s before %s", change.ResourceID, change.ChangeType),
		}
		plan.Checkpoints = append(plan.Checkpoints, checkpoint)
	}

	return plan, nil
}

// isReversible checks if a change is reversible
func (s *SafetyCheckEngine) isReversible(change Change) bool {
	// Delete operations are generally not reversible
	if change.ChangeType == "delete" {
		return false
	}

	// Check for stateful resources
	if isStatefulResource(change.ResourceType) {
		return false
	}

	return true
}

// isInBlockedTimeWindow checks if current time is in a blocked window
func (s *SafetyCheckEngine) isInBlockedTimeWindow() bool {
	now := time.Now()

	for _, window := range s.safetyThresholds.BlockedTimeWindows {
		if now.After(window.Start) && now.Before(window.End) {
			return true
		}
	}

	return false
}

// estimateChangeTime estimates time required for a change
func (s *SafetyCheckEngine) estimateChangeTime(change Change) time.Duration {
	// Simplified estimation based on resource type and change type
	baseTime := 30 * time.Second

	switch change.ChangeType {
	case "create":
		baseTime = 5 * time.Minute
	case "delete":
		baseTime = 2 * time.Minute
	case "replace":
		baseTime = 7 * time.Minute
	}

	// Adjust for resource type
	if strings.Contains(change.ResourceType, "cluster") {
		baseTime *= 3
	} else if strings.Contains(change.ResourceType, "database") {
		baseTime *= 2
	}

	return baseTime
}

// getAffectedAPIs returns APIs affected by a change
func (s *SafetyCheckEngine) getAffectedAPIs(change Change) []string {
	// Map resource types to APIs
	apiMap := map[string][]string{
		"aws_instance":       {"EC2"},
		"aws_rds_instance":   {"RDS"},
		"aws_s3_bucket":      {"S3"},
		"azure_vm":           {"Compute"},
		"azure_sql_database": {"SQL"},
		"gcp_compute_instance": {"Compute Engine"},
	}

	if apis, exists := apiMap[change.ResourceType]; exists {
		return apis
	}

	return []string{"Unknown"}
}

// Helper functions

func isStatefulResource(resourceType string) bool {
	statefulTypes := []string{
		"database", "rds", "dynamodb", "cosmosdb",
		"storage", "s3", "blob", "persistent_volume",
		"statefulset", "elasticsearch", "redis", "kafka",
	}

	for _, t := range statefulTypes {
		if strings.Contains(strings.ToLower(resourceType), t) {
			return true
		}
	}
	return false
}

func isSecurityResource(resourceType string) bool {
	securityTypes := []string{
		"security_group", "firewall", "iam", "role", "policy",
		"network_acl", "waf", "certificate", "key", "secret",
	}

	for _, t := range securityTypes {
		if strings.Contains(strings.ToLower(resourceType), t) {
			return true
		}
	}
	return false
}

func getDefaultValidationRules() []ValidationRule {
	return []ValidationRule{
		{
			Name:        "no-production-delete",
			Description: "Prevent deletion of production resources",
			Validate: func(change Change) (bool, string) {
				if change.ChangeType == "delete" {
					if tags, ok := change.Before["tags"].(map[string]interface{}); ok {
						if env, exists := tags["environment"]; exists && env == "production" {
							return false, "Cannot delete production resources"
						}
					}
				}
				return true, ""
			},
		},
		{
			Name:        "encryption-required",
			Description: "Ensure encryption is not disabled",
			Validate: func(change Change) (bool, string) {
				if encrypted, ok := change.After["encrypted"].(bool); ok {
					if !encrypted {
						if beforeEncrypted, ok := change.Before["encrypted"].(bool); ok && beforeEncrypted {
							return false, "Cannot disable encryption"
						}
					}
				}
				return true, ""
			},
		},
	}
}

func getDefaultSafetyThresholds() SafetyThresholds {
	return SafetyThresholds{
		MaxChangesPerRun:   100,
		MaxCriticalChanges: 5,
		MaxCostIncrease:    10000.0,
		MaxDowntimeMinutes: 30,
		RequireApproval: []string{
			"aws_iam_role",
			"aws_iam_policy",
			"aws_rds_cluster",
			"azure_role_assignment",
			"gcp_project_iam_binding",
		},
		BlockedResourceTypes: []string{
			"aws_organizations_account",
			"azure_subscription",
		},
		BlockedTimeWindows: []BlockedTimeWindow{},
	}
}

func getDefaultOrderingRules() []OrderingRule {
	return []OrderingRule{
		{
			ResourceType: "network",
			Priority:     1,
			MustBefore:   []string{"compute", "database"},
		},
		{
			ResourceType: "iam",
			Priority:     2,
			MustBefore:   []string{"compute", "storage"},
		},
		{
			ResourceType: "database",
			Priority:     5,
			MustAfter:    []string{"network", "iam"},
		},
	}
}

// EnhancedSafetyCheckResult represents the result of safety checks
type EnhancedSafetyCheckResult struct {
	Passed            bool
	Warnings          []string
	Errors            []string
	Preview           *ChangePreview
	ImpactAnalysis    []ImpactAssessment
	RequiredApprovals []ApprovalRequirement
	RollbackPlan      *RollbackPlan
}

// ApprovalRequirement represents a required approval
type ApprovalRequirement struct {
	ResourceID   string
	ResourceType string
	ChangeType   string
	Reason       string
	Severity     string
}

// RemediationPlan represents a plan for remediation
type RemediationPlan struct {
	ID          string
	Name        string
	Description string
	Actions     []EnhancedRemediationAction
	CreatedAt   time.Time
	ExecutedAt  *time.Time
	Status      string
}

// EnhancedRemediationAction represents a single remediation action
type EnhancedRemediationAction struct {
	ResourceID   string
	ResourceType string
	ActionType   string
	CurrentState map[string]interface{}
	DesiredState map[string]interface{}
	Priority     int
}