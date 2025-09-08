package security

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// SecurityService provides a unified interface for security and compliance management
type SecurityService struct {
	complianceManager *ComplianceManager
	policyEngine      *PolicyEngine
	reportGenerator   *ComplianceReportGenerator
	mu                sync.RWMutex
	eventBus          EventBus
	config            *SecurityConfig
}

// SecurityConfig represents configuration for the security service
type SecurityConfig struct {
	AutoScanInterval    time.Duration `json:"auto_scan_interval"`
	ReportGeneration    bool          `json:"report_generation"`
	NotificationEnabled bool          `json:"notification_enabled"`
	AuditLogging        bool          `json:"audit_logging"`
	MaxConcurrentScans  int           `json:"max_concurrent_scans"`
}

// NewSecurityService creates a new security service
func NewSecurityService(eventBus EventBus) *SecurityService {
	config := &SecurityConfig{
		AutoScanInterval:    24 * time.Hour,
		ReportGeneration:    true,
		NotificationEnabled: true,
		AuditLogging:        true,
		MaxConcurrentScans:  10,
	}

	// Create managers
	complianceManager := NewComplianceManager(eventBus)
	policyEngine := NewPolicyEngine(eventBus)
	reportGenerator := NewComplianceReportGenerator(complianceManager)

	return &SecurityService{
		complianceManager: complianceManager,
		policyEngine:      policyEngine,
		reportGenerator:   reportGenerator,
		eventBus:          eventBus,
		config:            config,
	}
}

// Start starts the security service
func (ss *SecurityService) Start(ctx context.Context) error {
	// Start auto-scan
	go ss.autoScan(ctx)

	// Create default policies and rules
	if err := ss.createDefaultPolicies(ctx); err != nil {
		return fmt.Errorf("failed to create default policies: %w", err)
	}

	// Publish event
	if ss.eventBus != nil {
		event := ComplianceEvent{
			Type:      "security_service_started",
			Message:   "Security service started",
			Severity:  "info",
			Timestamp: time.Now(),
		}
		ss.eventBus.PublishComplianceEvent(event)
	}

	return nil
}

// Stop stops the security service
func (ss *SecurityService) Stop(ctx context.Context) error {
	// Publish event
	if ss.eventBus != nil {
		event := ComplianceEvent{
			Type:      "security_service_stopped",
			Message:   "Security service stopped",
			Severity:  "info",
			Timestamp: time.Now(),
		}
		ss.eventBus.PublishComplianceEvent(event)
	}

	return nil
}

// ScanResources performs a comprehensive security scan of resources
func (ss *SecurityService) ScanResources(ctx context.Context, resources []*models.Resource) (*SecurityScanResult, error) {
	result := &SecurityScanResult{
		ScanID:     fmt.Sprintf("scan_%d", time.Now().Unix()),
		StartTime:  time.Now(),
		Resources:  resources,
		Policies:   []*SecurityPolicy{},
		Compliance: []*ComplianceResult{},
		Violations: []PolicyViolation{},
		Summary:    make(map[string]interface{}),
		Metadata:   make(map[string]interface{}),
	}

	// Get all enabled policies
	ss.mu.RLock()
	policies := make([]*SecurityPolicy, 0, len(ss.policyEngine.policies))
	for _, policy := range ss.policyEngine.policies {
		if policy.Enabled {
			policies = append(policies, policy)
		}
	}
	ss.mu.RUnlock()

	result.Policies = policies

	// Scan each resource
	for _, resource := range resources {
		// Evaluate policies
		for _, policy := range policies {
			evaluation, err := ss.policyEngine.EvaluatePolicy(ctx, policy.ID, resource)
			if err != nil {
				// Log error but continue with other policies
				fmt.Printf("Warning: failed to evaluate policy %s for resource %s: %v\n", policy.ID, resource.ID, err)
				continue
			}

			// Add violations to result
			result.Violations = append(result.Violations, evaluation.Violations...)
		}

		// Run compliance checks
		complianceResults, err := ss.complianceManager.RunAllComplianceChecks(ctx, []*models.Resource{resource})
		if err != nil {
			// Log error but continue with other resources
			fmt.Printf("Warning: failed to run compliance checks for resource %s: %v\n", resource.ID, err)
			continue
		}

		result.Compliance = append(result.Compliance, complianceResults...)
	}

	// Generate summary
	ss.generateScanSummary(result)

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Publish event
	if ss.eventBus != nil {
		event := ComplianceEvent{
			Type:      "security_scan_completed",
			Message:   fmt.Sprintf("Security scan completed for %d resources", len(resources)),
			Severity:  "info",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"scan_id":         result.ScanID,
				"resource_count":  len(resources),
				"violation_count": len(result.Violations),
				"duration":        result.Duration,
			},
		}
		ss.eventBus.PublishComplianceEvent(event)
	}

	return result, nil
}

// GenerateComplianceReport generates a compliance report for a specific standard
func (ss *SecurityService) GenerateComplianceReport(ctx context.Context, standard string) (*ComplianceReport, error) {
	report, err := ss.reportGenerator.GenerateReport(ctx, standard)
	if err != nil {
		return nil, fmt.Errorf("failed to generate compliance report: %w", err)
	}

	// Publish event
	if ss.eventBus != nil {
		event := ComplianceEvent{
			Type:      "compliance_report_generated",
			Message:   fmt.Sprintf("Compliance report generated for %s", standard),
			Severity:  "info",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"report_id": report.ID,
				"standard":  standard,
			},
		}
		ss.eventBus.PublishComplianceEvent(event)
	}

	return report, nil
}

// CreateSecurityPolicy creates a new security policy
func (ss *SecurityService) CreateSecurityPolicy(ctx context.Context, policy *SecurityPolicy) error {
	return ss.policyEngine.CreatePolicy(ctx, policy)
}

// CreateSecurityRule creates a new security rule
func (ss *SecurityService) CreateSecurityRule(ctx context.Context, rule *SecurityRule) error {
	return ss.policyEngine.CreateRule(ctx, rule)
}

// CreateCompliancePolicy creates a new compliance policy
func (ss *SecurityService) CreateCompliancePolicy(ctx context.Context, policy *CompliancePolicy) error {
	return ss.complianceManager.CreatePolicy(ctx, policy)
}

// GetSecurityStatus returns the overall security status
func (ss *SecurityService) GetSecurityStatus(ctx context.Context) (*SecurityStatus, error) {
	status := &SecurityStatus{
		OverallStatus: "Unknown",
		SecurityScore: 0.0,
		Policies:      make(map[string]int),
		Compliance:    make(map[string]int),
		Violations:    make(map[string]int),
		LastScan:      time.Time{},
		Metadata:      make(map[string]interface{}),
	}

	// Get policy counts
	ss.mu.RLock()
	for _, policy := range ss.policyEngine.policies {
		if policy.Enabled {
			status.Policies[policy.Category]++
		}
	}
	ss.mu.RUnlock()

	// Get compliance counts
	ss.mu.RLock()
	for _, policy := range ss.complianceManager.policies {
		if policy.Enabled {
			status.Compliance[policy.Standard]++
		}
	}
	ss.mu.RUnlock()

	// Calculate security score (simplified)
	totalPolicies := 0
	for _, count := range status.Policies {
		totalPolicies += count
	}
	if totalPolicies > 0 {
		status.SecurityScore = float64(totalPolicies) * 10.0 // Simplified scoring
	}

	// Determine overall status
	if status.SecurityScore >= 80 {
		status.OverallStatus = "Good"
	} else if status.SecurityScore >= 60 {
		status.OverallStatus = "Fair"
	} else if status.SecurityScore >= 40 {
		status.OverallStatus = "Poor"
	} else {
		status.OverallStatus = "Critical"
	}

	return status, nil
}

// SecurityScanResult represents the result of a security scan
type SecurityScanResult struct {
	ScanID     string                 `json:"scan_id"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
	Duration   time.Duration          `json:"duration"`
	Resources  []*models.Resource     `json:"resources"`
	Policies   []*SecurityPolicy      `json:"policies"`
	Compliance []*ComplianceResult    `json:"compliance"`
	Violations []PolicyViolation      `json:"violations"`
	Summary    map[string]interface{} `json:"summary"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// SecurityStatus represents the overall security status
type SecurityStatus struct {
	OverallStatus string                 `json:"overall_status"`
	SecurityScore float64                `json:"security_score"`
	Policies      map[string]int         `json:"policies"`
	Compliance    map[string]int         `json:"compliance"`
	Violations    map[string]int         `json:"violations"`
	LastScan      time.Time              `json:"last_scan"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Helper methods

// autoScan performs automatic security scanning
func (ss *SecurityService) autoScan(ctx context.Context) {
	ticker := time.NewTicker(ss.config.AutoScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// In a real implementation, you would get resources from the system
			// For now, this is a placeholder
			fmt.Println("Auto-scan triggered (placeholder)")
		}
	}
}

// createDefaultPolicies creates default security policies
func (ss *SecurityService) createDefaultPolicies(ctx context.Context) error {
	// Create default security rules
	rules := []*SecurityRule{
		{
			Name:        "Encryption Required",
			Description: "All storage resources must have encryption enabled",
			Type:        "encryption",
			Category:    "data_protection",
			Conditions: []RuleCondition{
				{
					Field:    "encryption",
					Operator: "equals",
					Value:    true,
					Type:     "boolean",
				},
			},
			Actions: []RuleAction{
				{
					Type:        "warn",
					Description: "Enable encryption for this resource",
				},
			},
			Severity: "high",
			Enabled:  true,
		},
		{
			Name:        "Access Logging Required",
			Description: "All resources must have access logging enabled",
			Type:        "logging",
			Category:    "audit",
			Conditions: []RuleCondition{
				{
					Field:    "logging",
					Operator: "equals",
					Value:    true,
					Type:     "boolean",
				},
			},
			Actions: []RuleAction{
				{
					Type:        "warn",
					Description: "Enable access logging for this resource",
				},
			},
			Severity: "medium",
			Enabled:  true,
		},
		{
			Name:        "Backup Required",
			Description: "All critical resources must have backups enabled",
			Type:        "backup",
			Category:    "data_protection",
			Conditions: []RuleCondition{
				{
					Field:    "backup",
					Operator: "equals",
					Value:    true,
					Type:     "boolean",
				},
			},
			Actions: []RuleAction{
				{
					Type:        "warn",
					Description: "Enable backups for this resource",
				},
			},
			Severity: "medium",
			Enabled:  true,
		},
	}

	// Create rules
	for _, rule := range rules {
		if err := ss.policyEngine.CreateRule(ctx, rule); err != nil {
			return fmt.Errorf("failed to create rule %s: %w", rule.Name, err)
		}
	}

	// Create default security policy
	policy := &SecurityPolicy{
		Name:        "Default Security Policy",
		Description: "Default security policy for all resources",
		Category:    "general",
		Priority:    "high",
		Rules:       []string{rules[0].ID, rules[1].ID, rules[2].ID},
		Scope: PolicyScope{
			Regions: []string{"us-east-1", "us-west-2", "eu-west-1"},
		},
		Enabled: true,
	}

	if err := ss.policyEngine.CreatePolicy(ctx, policy); err != nil {
		return fmt.Errorf("failed to create default policy: %w", err)
	}

	// Create default compliance policies
	compliancePolicies := []*CompliancePolicy{
		{
			Name:        "SOC2 Type II Compliance",
			Description: "SOC2 Type II compliance requirements",
			Standard:    "SOC2",
			Version:     "2017",
			Category:    "security",
			Severity:    "high",
			Rules: []ComplianceRule{
				{
					ID:          "soc2_encryption",
					Name:        "Data Encryption",
					Description: "All data must be encrypted at rest and in transit",
					Type:        "encryption",
					Conditions: []RuleCondition{
						{
							Field:    "encryption",
							Operator: "equals",
							Value:    true,
							Type:     "boolean",
						},
					},
					Actions: []RuleAction{
						{
							Type:        "enforce",
							Description: "Enable encryption",
						},
					},
					Severity: "high",
					Enabled:  true,
				},
			},
			Enabled: true,
		},
		{
			Name:        "HIPAA Compliance",
			Description: "HIPAA compliance requirements",
			Standard:    "HIPAA",
			Version:     "2013",
			Category:    "privacy",
			Severity:    "critical",
			Rules: []ComplianceRule{
				{
					ID:          "hipaa_access_control",
					Name:        "Access Control",
					Description: "Implement proper access controls for PHI",
					Type:        "access_control",
					Conditions: []RuleCondition{
						{
							Field:    "access_control",
							Operator: "equals",
							Value:    true,
							Type:     "boolean",
						},
					},
					Actions: []RuleAction{
						{
							Type:        "enforce",
							Description: "Implement access controls",
						},
					},
					Severity: "critical",
					Enabled:  true,
				},
			},
			Enabled: true,
		},
	}

	// Create compliance policies
	for _, policy := range compliancePolicies {
		if err := ss.complianceManager.CreatePolicy(ctx, policy); err != nil {
			return fmt.Errorf("failed to create compliance policy %s: %w", policy.Name, err)
		}
	}

	return nil
}

// generateScanSummary generates a summary for the security scan result
func (ss *SecurityService) generateScanSummary(result *SecurityScanResult) {
	// Count violations by severity
	severityCount := make(map[string]int)
	for _, violation := range result.Violations {
		severityCount[violation.Severity]++
	}

	// Count compliance results by status
	complianceStatus := make(map[string]int)
	for _, compliance := range result.Compliance {
		complianceStatus[compliance.Status]++
	}

	result.Summary["total_resources"] = len(result.Resources)
	result.Summary["total_policies"] = len(result.Policies)
	result.Summary["total_violations"] = len(result.Violations)
	result.Summary["violation_severity"] = severityCount
	result.Summary["compliance_status"] = complianceStatus
	result.Summary["scan_duration"] = result.Duration
}

// SetConfig updates the security service configuration
func (ss *SecurityService) SetConfig(config *SecurityConfig) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.config = config
}

// GetConfig returns the current security service configuration
func (ss *SecurityService) GetConfig() *SecurityConfig {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.config
}
