package analysis

import (
	"fmt"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// ImpactAnalyzer analyzes the impact of resource changes
type ImpactAnalyzer struct {
	config *ImpactAnalysisConfig
}

// ImpactAnalysisConfig defines impact analysis configuration
type ImpactAnalysisConfig struct {
	EnableBusinessImpact bool
	EnableCostImpact     bool
	EnableSecurityImpact bool
	EnablePerformanceImpact bool
	EnableComplianceImpact bool
	RiskThresholds        map[string]float64
	BusinessRules         []BusinessRule
}

// BusinessRule defines a business rule for impact analysis
type BusinessRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	ResourceType string                `json:"resource_type"`
	Condition   string                 `json:"condition"`
	Impact      string                 `json:"impact"` // high, medium, low
	Weight      float64                `json:"weight"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ImpactAnalysisResult represents the result of impact analysis
type ImpactAnalysisResult struct {
	ResourceID        string                    `json:"resource_id"`
	ResourceName      string                    `json:"resource_name"`
	ResourceType      string                    `json:"resource_type"`
	ChangeType        string                    `json:"change_type"`
	BusinessImpact    BusinessImpact            `json:"business_impact"`
	CostImpact        CostImpact                `json:"cost_impact"`
	SecurityImpact    SecurityImpact            `json:"security_impact"`
	PerformanceImpact PerformanceImpact         `json:"performance_impact"`
	ComplianceImpact  ComplianceImpact          `json:"compliance_impact"`
	AffectedResources []string                  `json:"affected_resources"`
	Dependencies      []DependencyImpact        `json:"dependencies"`
	RiskScore         float64                   `json:"risk_score"`
	Recommendations   []string                  `json:"recommendations"`
	Timestamp         time.Time                 `json:"timestamp"`
}

// BusinessImpact represents business impact analysis
type BusinessImpact struct {
	Level       string   `json:"level"` // critical, high, medium, low
	Description string   `json:"description"`
	Services    []string `json:"services"`
	Users       int      `json:"users"`
	Revenue     float64  `json:"revenue"`
	Downtime    string   `json:"downtime"`
}

// CostImpact represents cost impact analysis
type CostImpact struct {
	Level        string  `json:"level"`
	MonthlyCost  float64 `json:"monthly_cost"`
	AnnualCost   float64 `json:"annual_cost"`
	CostChange   float64 `json:"cost_change"`
	Currency     string  `json:"currency"`
	Description  string  `json:"description"`
}

// SecurityImpact represents security impact analysis
type SecurityImpact struct {
	Level        string   `json:"level"`
	RiskFactors  []string `json:"risk_factors"`
	Vulnerabilities []string `json:"vulnerabilities"`
	Compliance   []string `json:"compliance"`
	Description  string   `json:"description"`
}

// PerformanceImpact represents performance impact analysis
type PerformanceImpact struct {
	Level        string  `json:"level"`
	Latency      float64 `json:"latency"`
	Throughput   float64 `json:"throughput"`
	Availability float64 `json:"availability"`
	Description  string  `json:"description"`
}

// ComplianceImpact represents compliance impact analysis
type ComplianceImpact struct {
	Level        string   `json:"level"`
	Standards    []string `json:"standards"`
	Violations   []string `json:"violations"`
	Requirements []string `json:"requirements"`
	Description  string   `json:"description"`
}

// DependencyImpact represents dependency impact analysis
type DependencyImpact struct {
	ResourceID   string  `json:"resource_id"`
	ResourceName string  `json:"resource_name"`
	ImpactLevel  string  `json:"impact_level"`
	Description  string  `json:"description"`
	RiskScore    float64 `json:"risk_score"`
}

// NewImpactAnalyzer creates a new impact analyzer
func NewImpactAnalyzer(config *ImpactAnalysisConfig) *ImpactAnalyzer {
	if config == nil {
		config = &ImpactAnalysisConfig{
			EnableBusinessImpact: true,
			EnableCostImpact:     true,
			EnableSecurityImpact: true,
			EnablePerformanceImpact: true,
			EnableComplianceImpact: true,
			RiskThresholds: map[string]float64{
				"critical": 0.9,
				"high":     0.7,
				"medium":   0.4,
				"low":      0.1,
			},
			BusinessRules: []BusinessRule{},
		}
	}

	return &ImpactAnalyzer{
		config: config,
	}
}

// AnalyzeImpact analyzes the impact of resource changes
func (ia *ImpactAnalyzer) AnalyzeImpact(drift models.DriftResult, stateFile *models.StateFile) (*ImpactAnalysisResult, error) {
	result := &ImpactAnalysisResult{
		ResourceID:   drift.ResourceID,
		ResourceName: drift.ResourceName,
		ResourceType: drift.ResourceType,
		ChangeType:   drift.DriftType,
		Timestamp:    time.Now(),
	}

	// Analyze business impact
	if ia.config.EnableBusinessImpact {
		result.BusinessImpact = ia.analyzeBusinessImpact(drift, stateFile)
	}

	// Analyze cost impact
	if ia.config.EnableCostImpact {
		result.CostImpact = ia.analyzeCostImpact(drift, stateFile)
	}

	// Analyze security impact
	if ia.config.EnableSecurityImpact {
		result.SecurityImpact = ia.analyzeSecurityImpact(drift, stateFile)
	}

	// Analyze performance impact
	if ia.config.EnablePerformanceImpact {
		result.PerformanceImpact = ia.analyzePerformanceImpact(drift, stateFile)
	}

	// Analyze compliance impact
	if ia.config.EnableComplianceImpact {
		result.ComplianceImpact = ia.analyzeComplianceImpact(drift, stateFile)
	}

	// Analyze dependencies
	result.Dependencies = ia.analyzeDependencies(drift, stateFile)

	// Calculate overall risk score
	result.RiskScore = ia.calculateRiskScore(result)

	// Generate recommendations
	result.Recommendations = ia.generateRecommendations(result)

	return result, nil
}

// analyzeBusinessImpact analyzes business impact of changes
func (ia *ImpactAnalyzer) analyzeBusinessImpact(drift models.DriftResult, stateFile *models.StateFile) BusinessImpact {
	impact := BusinessImpact{
		Level:       "low",
		Description: "Minimal business impact",
		Services:    []string{},
		Users:       0,
		Revenue:     0.0,
		Downtime:    "none",
	}

	// Determine impact based on resource type and change
	switch drift.ResourceType {
	case "aws_instance", "aws_ecs_service", "aws_lambda_function":
		impact.Level = "high"
		impact.Description = "Compute resource change may affect application availability"
		impact.Services = []string{"application", "api", "web"}
		impact.Users = 1000 // Estimate
		impact.Revenue = 10000.0 // Estimate
		impact.Downtime = "5-15 minutes"

	case "aws_rds_instance", "aws_dynamodb_table":
		impact.Level = "critical"
		impact.Description = "Database change may cause data loss or service interruption"
		impact.Services = []string{"database", "data", "storage"}
		impact.Users = 5000 // Estimate
		impact.Revenue = 50000.0 // Estimate
		impact.Downtime = "30-60 minutes"

	case "aws_vpc", "aws_subnet", "aws_security_group":
		impact.Level = "high"
		impact.Description = "Network change may affect connectivity and security"
		impact.Services = []string{"network", "security", "connectivity"}
		impact.Users = 2000 // Estimate
		impact.Revenue = 20000.0 // Estimate
		impact.Downtime = "10-30 minutes"

	case "aws_s3_bucket", "aws_cloudfront_distribution":
		impact.Level = "medium"
		impact.Description = "Storage/CDN change may affect data access and performance"
		impact.Services = []string{"storage", "cdn", "static-content"}
		impact.Users = 500 // Estimate
		impact.Revenue = 5000.0 // Estimate
		impact.Downtime = "2-10 minutes"
	}

	// Adjust based on change type
	switch drift.DriftType {
	case "missing":
		impact.Level = "critical"
		impact.Description += " - Resource is missing and may cause service failure"
	case "extra":
		impact.Level = "low"
		impact.Description += " - Extra resource may incur unnecessary costs"
	case "modified":
		// Level already set based on resource type
	}

	return impact
}

// analyzeCostImpact analyzes cost impact of changes
func (ia *ImpactAnalyzer) analyzeCostImpact(drift models.DriftResult, stateFile *models.StateFile) CostImpact {
	impact := CostImpact{
		Level:       "low",
		MonthlyCost: 0.0,
		AnnualCost:  0.0,
		CostChange:  0.0,
		Currency:    "USD",
		Description: "Minimal cost impact",
	}

	// Estimate costs based on resource type
	switch drift.ResourceType {
	case "aws_instance":
		impact.MonthlyCost = 100.0
		impact.AnnualCost = 1200.0
		impact.Level = "medium"

	case "aws_rds_instance":
		impact.MonthlyCost = 200.0
		impact.AnnualCost = 2400.0
		impact.Level = "high"

	case "aws_lambda_function":
		impact.MonthlyCost = 10.0
		impact.AnnualCost = 120.0
		impact.Level = "low"

	case "aws_s3_bucket":
		impact.MonthlyCost = 50.0
		impact.AnnualCost = 600.0
		impact.Level = "medium"

	case "aws_cloudfront_distribution":
		impact.MonthlyCost = 30.0
		impact.AnnualCost = 360.0
		impact.Level = "low"
	}

	// Adjust based on change type
	switch drift.DriftType {
	case "missing":
		impact.CostChange = -impact.MonthlyCost
		impact.Description = "Cost savings from missing resource"
	case "extra":
		impact.CostChange = impact.MonthlyCost
		impact.Description = "Additional cost from extra resource"
	case "modified":
		impact.CostChange = 0.0
		impact.Description = "No cost change for modification"
	}

	return impact
}

// analyzeSecurityImpact analyzes security impact of changes
func (ia *ImpactAnalyzer) analyzeSecurityImpact(drift models.DriftResult, stateFile *models.StateFile) SecurityImpact {
	impact := SecurityImpact{
		Level:         "low",
		RiskFactors:   []string{},
		Vulnerabilities: []string{},
		Compliance:    []string{},
		Description:   "Minimal security impact",
	}

	// Analyze security impact based on resource type
	switch drift.ResourceType {
	case "aws_security_group":
		impact.Level = "critical"
		impact.RiskFactors = []string{"network access", "firewall rules", "port exposure"}
		impact.Vulnerabilities = []string{"unauthorized access", "data breach"}
		impact.Compliance = []string{"PCI-DSS", "SOC2", "HIPAA"}
		impact.Description = "Security group changes may affect network security"

	case "aws_iam_role", "aws_iam_policy":
		impact.Level = "high"
		impact.RiskFactors = []string{"permissions", "access control", "privilege escalation"}
		impact.Vulnerabilities = []string{"unauthorized access", "privilege abuse"}
		impact.Compliance = []string{"SOC2", "ISO27001"}
		impact.Description = "IAM changes may affect access control"

	case "aws_s3_bucket":
		impact.Level = "medium"
		impact.RiskFactors = []string{"data access", "encryption", "public access"}
		impact.Vulnerabilities = []string{"data exposure", "unauthorized access"}
		impact.Compliance = []string{"GDPR", "CCPA", "SOC2"}
		impact.Description = "S3 bucket changes may affect data security"

	case "aws_instance":
		impact.Level = "medium"
		impact.RiskFactors = []string{"instance access", "network exposure"}
		impact.Vulnerabilities = []string{"unauthorized access", "malware"}
		impact.Compliance = []string{"SOC2"}
		impact.Description = "Instance changes may affect system security"
	}

	// Check for specific security issues in the drift
	if ia.hasSecurityIssue(drift) {
		impact.Level = "critical"
		impact.Description += " - Security vulnerability detected"
	}

	return impact
}

// analyzePerformanceImpact analyzes performance impact of changes
func (ia *ImpactAnalyzer) analyzePerformanceImpact(drift models.DriftResult, stateFile *models.StateFile) PerformanceImpact {
	impact := PerformanceImpact{
		Level:        "low",
		Latency:      0.0,
		Throughput:   0.0,
		Availability: 99.9,
		Description:  "Minimal performance impact",
	}

	// Analyze performance impact based on resource type
	switch drift.ResourceType {
	case "aws_instance":
		impact.Level = "medium"
		impact.Latency = 50.0 // ms
		impact.Throughput = 1000.0 // requests/sec
		impact.Availability = 99.5
		impact.Description = "Instance changes may affect application performance"

	case "aws_rds_instance":
		impact.Level = "high"
		impact.Latency = 100.0 // ms
		impact.Throughput = 500.0 // queries/sec
		impact.Availability = 99.0
		impact.Description = "Database changes may affect data access performance"

	case "aws_cloudfront_distribution":
		impact.Level = "medium"
		impact.Latency = 20.0 // ms
		impact.Throughput = 5000.0 // requests/sec
		impact.Availability = 99.9
		impact.Description = "CDN changes may affect content delivery performance"

	case "aws_lambda_function":
		impact.Level = "low"
		impact.Latency = 200.0 // ms
		impact.Throughput = 100.0 // invocations/sec
		impact.Availability = 99.9
		impact.Description = "Lambda changes may affect function performance"
	}

	return impact
}

// analyzeComplianceImpact analyzes compliance impact of changes
func (ia *ImpactAnalyzer) analyzeComplianceImpact(drift models.DriftResult, stateFile *models.StateFile) ComplianceImpact {
	impact := ComplianceImpact{
		Level:        "low",
		Standards:    []string{},
		Violations:   []string{},
		Requirements: []string{},
		Description:  "Minimal compliance impact",
	}

	// Analyze compliance impact based on resource type
	switch drift.ResourceType {
	case "aws_s3_bucket":
		impact.Level = "high"
		impact.Standards = []string{"GDPR", "CCPA", "SOC2", "ISO27001"}
		impact.Requirements = []string{"data encryption", "access logging", "versioning"}
		impact.Description = "S3 changes may affect data compliance"

	case "aws_rds_instance":
		impact.Level = "high"
		impact.Standards = []string{"SOC2", "ISO27001", "PCI-DSS"}
		impact.Requirements = []string{"encryption at rest", "encryption in transit", "backup retention"}
		impact.Description = "Database changes may affect data compliance"

	case "aws_iam_role", "aws_iam_policy":
		impact.Level = "medium"
		impact.Standards = []string{"SOC2", "ISO27001"}
		impact.Requirements = []string{"least privilege", "access review", "audit logging"}
		impact.Description = "IAM changes may affect access compliance"

	case "aws_security_group":
		impact.Level = "medium"
		impact.Standards = []string{"SOC2", "ISO27001", "PCI-DSS"}
		impact.Requirements = []string{"network segmentation", "firewall rules", "access control"}
		impact.Description = "Security group changes may affect network compliance"
	}

	return impact
}

// analyzeDependencies analyzes dependency impact
func (ia *ImpactAnalyzer) analyzeDependencies(drift models.DriftResult, stateFile *models.StateFile) []DependencyImpact {
	var dependencies []DependencyImpact

	// Find resources that depend on this resource
	for _, resource := range stateFile.Resources {
		if ia.hasDependency(resource, drift) {
			dependency := DependencyImpact{
				ResourceID:   resource.Name,
				ResourceName: resource.Name,
				ImpactLevel:  "medium",
				Description:  fmt.Sprintf("Depends on %s", drift.ResourceName),
				RiskScore:    0.5,
			}

			// Determine impact level based on resource type
			switch resource.Type {
			case "aws_instance", "aws_ecs_service":
				dependency.ImpactLevel = "high"
				dependency.RiskScore = 0.8
			case "aws_rds_instance":
				dependency.ImpactLevel = "critical"
				dependency.RiskScore = 0.9
			case "aws_lambda_function":
				dependency.ImpactLevel = "medium"
				dependency.RiskScore = 0.6
			}

			dependencies = append(dependencies, dependency)
		}
	}

	return dependencies
}

// hasDependency checks if a resource depends on the drifted resource
func (ia *ImpactAnalyzer) hasDependency(resource models.TerraformResource, drift models.DriftResult) bool {
	for _, instance := range resource.Instances {
		for _, value := range instance.Attributes {
			if strValue, ok := value.(string); ok {
				if strings.Contains(strValue, drift.ResourceName) || strings.Contains(strValue, drift.ResourceID) {
					return true
				}
			}
		}
	}
	return false
}

// hasSecurityIssue checks if the drift has security implications
func (ia *ImpactAnalyzer) hasSecurityIssue(drift models.DriftResult) bool {
	// Check for security-related changes
	securityKeywords := []string{
		"public", "open", "0.0.0.0/0", "::/0",
		"encryption", "ssl", "tls", "https",
		"permission", "policy", "role", "access",
	}

	// Check in changes
	for _, change := range drift.Changes {
		changeStr := fmt.Sprintf("%v", change.NewValue)
		for _, keyword := range securityKeywords {
			if strings.Contains(strings.ToLower(changeStr), keyword) {
				return true
			}
		}
	}

	return false
}

// calculateRiskScore calculates overall risk score
func (ia *ImpactAnalyzer) calculateRiskScore(result *ImpactAnalysisResult) float64 {
	score := 0.0
	weights := 0.0

	// Business impact weight
	if ia.config.EnableBusinessImpact {
		score += ia.getImpactScore(result.BusinessImpact.Level) * 0.3
		weights += 0.3
	}

	// Security impact weight
	if ia.config.EnableSecurityImpact {
		score += ia.getImpactScore(result.SecurityImpact.Level) * 0.25
		weights += 0.25
	}

	// Cost impact weight
	if ia.config.EnableCostImpact {
		score += ia.getImpactScore(result.CostImpact.Level) * 0.2
		weights += 0.2
	}

	// Performance impact weight
	if ia.config.EnablePerformanceImpact {
		score += ia.getImpactScore(result.PerformanceImpact.Level) * 0.15
		weights += 0.15
	}

	// Compliance impact weight
	if ia.config.EnableComplianceImpact {
		score += ia.getImpactScore(result.ComplianceImpact.Level) * 0.1
		weights += 0.1
	}

	if weights > 0 {
		return score / weights
	}

	return 0.0
}

// getImpactScore converts impact level to numeric score
func (ia *ImpactAnalyzer) getImpactScore(level string) float64 {
	switch strings.ToLower(level) {
	case "critical":
		return 1.0
	case "high":
		return 0.8
	case "medium":
		return 0.5
	case "low":
		return 0.2
	default:
		return 0.0
	}
}

// generateRecommendations generates recommendations based on impact analysis
func (ia *ImpactAnalyzer) generateRecommendations(result *ImpactAnalysisResult) []string {
	var recommendations []string

	// Business impact recommendations
	if result.BusinessImpact.Level == "critical" {
		recommendations = append(recommendations, 
			"Schedule maintenance window for this change")
		recommendations = append(recommendations, 
			"Notify stakeholders about potential service impact")
	}

	// Security impact recommendations
	if result.SecurityImpact.Level == "critical" {
		recommendations = append(recommendations, 
			"Review security implications before applying changes")
		recommendations = append(recommendations, 
			"Consider security team approval for this change")
	}

	// Cost impact recommendations
	if result.CostImpact.CostChange > 0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("This change will increase costs by $%.2f/month", result.CostImpact.CostChange))
	}

	// Performance impact recommendations
	if result.PerformanceImpact.Level == "high" {
		recommendations = append(recommendations, 
			"Monitor performance metrics after applying changes")
	}

	// Dependency recommendations
	if len(result.Dependencies) > 0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("This change affects %d dependent resources", len(result.Dependencies)))
	}

	// General recommendations
	if result.RiskScore > 0.7 {
		recommendations = append(recommendations, 
			"High-risk change - consider testing in non-production first")
	}

	return recommendations
}

// AddBusinessRule adds a custom business rule
func (ia *ImpactAnalyzer) AddBusinessRule(rule BusinessRule) {
	ia.config.BusinessRules = append(ia.config.BusinessRules, rule)
}

// GetBusinessRules returns all business rules
func (ia *ImpactAnalyzer) GetBusinessRules() []BusinessRule {
	return ia.config.BusinessRules
}
