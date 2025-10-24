package compliance

import (
	"context"
	"fmt"
	"time"

	"github.com/open-policy-agent/opa/ast"
)

// ComplianceService provides compliance management functionality
type ComplianceService struct {
	policyEngine *OPAEngine
	reporter     *ComplianceReporter
}

// ComplianceRepository defines the interface for compliance data persistence
type ComplianceRepository interface {
	// Policy management
	SavePolicy(ctx context.Context, policy *Policy) error
	GetPolicy(ctx context.Context, id string) (*Policy, error)
	ListPolicies(ctx context.Context) ([]*Policy, error)
	DeletePolicy(ctx context.Context, id string) error

	// Report management
	SaveReport(ctx context.Context, report *ComplianceReport) error
	GetReport(ctx context.Context, id string) (*ComplianceReport, error)
	ListReports(ctx context.Context, limit, offset int) ([]*ComplianceReport, error)
	DeleteReport(ctx context.Context, id string) error

	// Evaluation history
	SaveEvaluation(ctx context.Context, evaluation *PolicyEvaluation) error
	GetEvaluationHistory(ctx context.Context, policyID string, limit, offset int) ([]*PolicyEvaluation, error)
}

// PolicyEvaluation represents a policy evaluation result
type PolicyEvaluation struct {
	ID          string                 `json:"id"`
	PolicyID    string                 `json:"policy_id"`
	Input       PolicyInput            `json:"input"`
	Decision    *PolicyDecision        `json:"decision"`
	EvaluatedAt time.Time              `json:"evaluated_at"`
	EvaluatedBy string                 `json:"evaluated_by,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Duration    time.Duration          `json:"duration"`
}

// NewComplianceService creates a new compliance service
func NewComplianceService(policyEngine *OPAEngine, reporter *ComplianceReporter) *ComplianceService {
	return &ComplianceService{
		policyEngine: policyEngine,
		reporter:     reporter,
	}
}

// EvaluatePolicy evaluates a policy against input
func (s *ComplianceService) EvaluatePolicy(ctx context.Context, policyPackage string, input PolicyInput) (*PolicyDecision, error) {
	start := time.Now()

	decision, err := s.policyEngine.Evaluate(ctx, policyPackage, input)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate policy: %w", err)
	}

	duration := time.Since(start)

	// Log the evaluation (in a real implementation, this would be stored)
	fmt.Printf("Policy evaluation completed in %v\n", duration)

	return decision, nil
}

// CreatePolicy creates a new policy
func (s *ComplianceService) CreatePolicy(ctx context.Context, policy *Policy) error {
	// Validate policy
	if policy.ID == "" {
		return fmt.Errorf("policy ID is required")
	}
	if policy.Rules == "" {
		return fmt.Errorf("policy rules are required")
	}
	if policy.Package == "" {
		return fmt.Errorf("policy package is required")
	}

	// Set timestamps
	now := time.Now()
	policy.CreatedAt = now
	policy.UpdatedAt = now

	// Upload to OPA engine
	if err := s.policyEngine.UploadPolicy(ctx, policy); err != nil {
		return fmt.Errorf("failed to upload policy to OPA: %w", err)
	}

	return nil
}

// UpdatePolicy updates an existing policy
func (s *ComplianceService) UpdatePolicy(ctx context.Context, policy *Policy) error {
	// Validate policy
	if policy.ID == "" {
		return fmt.Errorf("policy ID is required")
	}
	if policy.Rules == "" {
		return fmt.Errorf("policy rules are required")
	}
	if policy.Package == "" {
		return fmt.Errorf("policy package is required")
	}

	// Update timestamp
	policy.UpdatedAt = time.Now()

	// Upload to OPA engine
	if err := s.policyEngine.UploadPolicy(ctx, policy); err != nil {
		return fmt.Errorf("failed to update policy in OPA: %w", err)
	}

	return nil
}

// DeletePolicy deletes a policy
func (s *ComplianceService) DeletePolicy(ctx context.Context, policyID string) error {
	if policyID == "" {
		return fmt.Errorf("policy ID is required")
	}

	// Delete from OPA engine
	if err := s.policyEngine.DeletePolicy(ctx, policyID); err != nil {
		return fmt.Errorf("failed to delete policy from OPA: %w", err)
	}

	return nil
}

// GetPolicy retrieves a policy by ID
func (s *ComplianceService) GetPolicy(ctx context.Context, policyID string) (*Policy, bool) {
	return s.policyEngine.GetPolicy(policyID)
}

// ListPolicies returns all policies
func (s *ComplianceService) ListPolicies(ctx context.Context) []*Policy {
	return s.policyEngine.ListPolicies()
}

// GenerateComplianceReport generates a compliance report
func (s *ComplianceService) GenerateComplianceReport(ctx context.Context, complianceType ComplianceType, period ReportPeriod) (*ComplianceReport, error) {
	return s.reporter.GenerateReport(ctx, complianceType, period)
}

// ExportReport exports a report in the specified format
func (s *ComplianceService) ExportReport(ctx context.Context, report *ComplianceReport, format string) ([]byte, error) {
	formatter, exists := s.reporter.formatters[format]
	if !exists {
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	return formatter.Format(report)
}

// BatchEvaluatePolicies evaluates multiple policies against the same input
func (s *ComplianceService) BatchEvaluatePolicies(ctx context.Context, policyPackages []string, input PolicyInput) (map[string]*PolicyDecision, error) {
	results := make(map[string]*PolicyDecision)

	for _, packageName := range policyPackages {
		decision, err := s.EvaluatePolicy(ctx, packageName, input)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate policy %s: %w", packageName, err)
		}
		results[packageName] = decision
	}

	return results, nil
}

// ValidatePolicy validates a policy without uploading it
func (s *ComplianceService) ValidatePolicy(ctx context.Context, policy *Policy) error {
	// Parse the policy module
	module, err := ast.ParseModule(policy.ID, policy.Rules)
	if err != nil {
		return fmt.Errorf("failed to parse policy: %w", err)
	}

	// Create a temporary compiler
	compiler := ast.NewCompiler()
	modules := map[string]*ast.Module{
		policy.ID: module,
	}

	// Compile the policy
	compiler.Compile(modules)
	if compiler.Failed() {
		return fmt.Errorf("policy compilation failed: %v", compiler.Errors)
	}

	return nil
}

// GetPolicyStatistics returns statistics about policy evaluations
func (s *ComplianceService) GetPolicyStatistics(ctx context.Context, policyID string, since time.Time) (*PolicyStatistics, error) {
	// In a real implementation, this would query the repository for evaluation history
	// For now, return mock statistics
	return &PolicyStatistics{
		PolicyID:         policyID,
		TotalEvaluations: 100,
		AllowedCount:     85,
		DeniedCount:      15,
		AverageDuration:  50 * time.Millisecond,
		LastEvaluated:    time.Now(),
	}, nil
}

// PolicyStatistics represents statistics about policy evaluations
type PolicyStatistics struct {
	PolicyID         string        `json:"policy_id"`
	TotalEvaluations int           `json:"total_evaluations"`
	AllowedCount     int           `json:"allowed_count"`
	DeniedCount      int           `json:"denied_count"`
	AverageDuration  time.Duration `json:"average_duration"`
	LastEvaluated    time.Time     `json:"last_evaluated"`
}
