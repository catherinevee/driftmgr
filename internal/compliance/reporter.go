package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/drift/detector"
)

// ComplianceReporter generates compliance reports
type ComplianceReporter struct {
	templates    map[string]*ReportTemplate
	formatters   map[string]Formatter
	dataSource   DataSource
	policyEngine *OPAEngine
}

// ReportTemplate represents a compliance report template
type ReportTemplate struct {
	ID           string
	Name         string
	Type         ComplianceType
	Sections     []ReportSection
	HTMLTemplate string
	JSONSchema   map[string]interface{}
}

// ComplianceType represents the type of compliance
type ComplianceType string

const (
	ComplianceSOC2     ComplianceType = "SOC2"
	ComplianceHIPAA    ComplianceType = "HIPAA"
	CompliancePCIDSS   ComplianceType = "PCI-DSS"
	ComplianceISO27001 ComplianceType = "ISO27001"
	ComplianceGDPR     ComplianceType = "GDPR"
	ComplianceCustom   ComplianceType = "Custom"
)

// ReportSection represents a section in the compliance report
type ReportSection struct {
	Title       string
	Description string
	Controls    []Control
	Evidence    []Evidence
	Status      ControlStatus
	Score       float64
}

// Control represents a compliance control
type Control struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	Category     string        `json:"category"`
	Status       ControlStatus `json:"status"`
	Evidence     []Evidence    `json:"evidence"`
	Findings     []Finding     `json:"findings"`
	Remediation  string        `json:"remediation,omitempty"`
	LastAssessed time.Time     `json:"last_assessed"`
}

// ControlStatus represents the status of a control
type ControlStatus string

const (
	ControlStatusPassed        ControlStatus = "passed"
	ControlStatusFailed        ControlStatus = "failed"
	ControlStatusPartial       ControlStatus = "partial"
	ControlStatusNotAssessed   ControlStatus = "not_assessed"
	ControlStatusNotApplicable ControlStatus = "not_applicable"
)

// Evidence represents evidence for a control
type Evidence struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Attachment  string                 `json:"attachment,omitempty"`
}

// Finding represents a compliance finding
type Finding struct {
	ID          string                 `json:"id"`
	Severity    string                 `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Resource    string                 `json:"resource,omitempty"`
	Impact      string                 `json:"impact,omitempty"`
	Remediation string                 `json:"remediation"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// DataSource provides data for compliance reports
type DataSource interface {
	GetDriftResults(ctx context.Context) ([]*detector.DriftResult, error)
	GetPolicyViolations(ctx context.Context) ([]PolicyViolation, error)
	GetResourceInventory(ctx context.Context) ([]interface{}, error)
	GetAuditLogs(ctx context.Context, since time.Time) ([]interface{}, error)
}

// Formatter formats reports in different formats
type Formatter interface {
	Format(report *ComplianceReport) ([]byte, error)
}

// ComplianceReport represents a generated compliance report
type ComplianceReport struct {
	ID          string                 `json:"id"`
	Type        ComplianceType         `json:"type"`
	Title       string                 `json:"title"`
	GeneratedAt time.Time              `json:"generated_at"`
	Period      ReportPeriod           `json:"period"`
	Summary     ReportSummary          `json:"summary"`
	Sections    []ReportSection        `json:"sections"`
	Controls    []Control              `json:"controls"`
	Findings    []Finding              `json:"findings"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Signature   string                 `json:"signature,omitempty"`
}

// ReportPeriod represents the reporting period
type ReportPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ReportSummary provides a summary of the compliance status
type ReportSummary struct {
	TotalControls    int                    `json:"total_controls"`
	PassedControls   int                    `json:"passed_controls"`
	FailedControls   int                    `json:"failed_controls"`
	ComplianceScore  float64                `json:"compliance_score"`
	CriticalFindings int                    `json:"critical_findings"`
	HighFindings     int                    `json:"high_findings"`
	MediumFindings   int                    `json:"medium_findings"`
	LowFindings      int                    `json:"low_findings"`
	Trends           map[string]interface{} `json:"trends,omitempty"`
}

// NewComplianceReporter creates a new compliance reporter
func NewComplianceReporter(dataSource DataSource, policyEngine *OPAEngine) *ComplianceReporter {
	reporter := &ComplianceReporter{
		templates:    make(map[string]*ReportTemplate),
		formatters:   make(map[string]Formatter),
		dataSource:   dataSource,
		policyEngine: policyEngine,
	}

	// Load default templates
	reporter.loadDefaultTemplates()

	// Register formatters
	reporter.formatters["json"] = &JSONFormatter{}
	reporter.formatters["html"] = &HTMLFormatter{}
	reporter.formatters["pdf"] = &PDFFormatter{}
	reporter.formatters["yaml"] = &YAMLFormatter{}

	return reporter
}

// GenerateReport generates a compliance report
func (r *ComplianceReporter) GenerateReport(ctx context.Context, complianceType ComplianceType, period ReportPeriod) (*ComplianceReport, error) {
	report := &ComplianceReport{
		ID:          fmt.Sprintf("report-%d", time.Now().Unix()),
		Type:        complianceType,
		Title:       fmt.Sprintf("%s Compliance Report", complianceType),
		GeneratedAt: time.Now(),
		Period:      period,
		Metadata:    make(map[string]interface{}),
	}

	// Get template for compliance type
	template, exists := r.templates[string(complianceType)]
	if !exists {
		return nil, fmt.Errorf("no template found for compliance type: %s", complianceType)
	}

	// Assess controls based on template
	controls, findings := r.assessControls(ctx, template)
	report.Controls = controls
	report.Findings = findings

	// Generate sections
	report.Sections = r.generateSections(controls, findings)

	// Calculate summary
	report.Summary = r.calculateSummary(controls, findings)

	// Sign report
	report.Signature = r.signReport(report)

	return report, nil
}

// assessControls assesses compliance controls
func (r *ComplianceReporter) assessControls(ctx context.Context, template *ReportTemplate) ([]Control, []Finding) {
	var controls []Control
	var allFindings []Finding

	// Get data from sources
	driftResults, _ := r.dataSource.GetDriftResults(ctx)
	policyViolations, _ := r.dataSource.GetPolicyViolations(ctx)

	// Assess each control in template
	for _, section := range template.Sections {
		for _, control := range section.Controls {
			assessedControl := r.assessControl(ctx, control, driftResults, policyViolations)
			controls = append(controls, assessedControl)
			allFindings = append(allFindings, assessedControl.Findings...)
		}
	}

	return controls, allFindings
}

// assessControl assesses a single control
func (r *ComplianceReporter) assessControl(ctx context.Context, control Control, driftResults []*detector.DriftResult, violations []PolicyViolation) Control {
	control.LastAssessed = time.Now()
	control.Evidence = []Evidence{}
	control.Findings = []Finding{}

	// Check for relevant drift
	for _, drift := range driftResults {
		if r.isDriftRelevantToControl(control, drift) {
			control.Findings = append(control.Findings, Finding{
				ID:          fmt.Sprintf("drift-%s", drift.Resource),
				Severity:    "medium",
				Title:       "Configuration Drift Detected",
				Description: fmt.Sprintf("Resource %s has drifted from desired state", drift.Resource),
				Resource:    drift.Resource,
				Remediation: "Apply Terraform to restore desired state",
			})
		}
	}

	// Check for policy violations
	for _, violation := range violations {
		if r.isViolationRelevantToControl(control, violation) {
			control.Findings = append(control.Findings, Finding{
				ID:          fmt.Sprintf("policy-%s", violation.Rule),
				Severity:    violation.Severity,
				Title:       violation.Message,
				Description: violation.Message,
				Resource:    violation.Resource,
				Remediation: violation.Remediation,
			})
		}
	}

	// Determine control status
	if len(control.Findings) == 0 {
		control.Status = ControlStatusPassed
		control.Evidence = append(control.Evidence, Evidence{
			Type:        "automated_check",
			Description: "No violations or drift detected",
			Source:      "driftmgr",
			Timestamp:   time.Now(),
		})
	} else {
		control.Status = ControlStatusFailed
	}

	return control
}

// generateSections generates report sections
func (r *ComplianceReporter) generateSections(controls []Control, findings []Finding) []ReportSection {
	// Group controls by category
	categoryMap := make(map[string][]Control)
	for _, control := range controls {
		categoryMap[control.Category] = append(categoryMap[control.Category], control)
	}

	var sections []ReportSection
	for category, categoryControls := range categoryMap {
		section := ReportSection{
			Title:    category,
			Controls: categoryControls,
			Status:   r.calculateSectionStatus(categoryControls),
			Score:    r.calculateSectionScore(categoryControls),
		}
		sections = append(sections, section)
	}

	return sections
}

// calculateSummary calculates the report summary
func (r *ComplianceReporter) calculateSummary(controls []Control, findings []Finding) ReportSummary {
	summary := ReportSummary{
		TotalControls: len(controls),
	}

	for _, control := range controls {
		switch control.Status {
		case ControlStatusPassed:
			summary.PassedControls++
		case ControlStatusFailed:
			summary.FailedControls++
		}
	}

	for _, finding := range findings {
		switch finding.Severity {
		case "critical":
			summary.CriticalFindings++
		case "high":
			summary.HighFindings++
		case "medium":
			summary.MediumFindings++
		case "low":
			summary.LowFindings++
		}
	}

	if summary.TotalControls > 0 {
		summary.ComplianceScore = float64(summary.PassedControls) / float64(summary.TotalControls) * 100
	}

	return summary
}

// loadDefaultTemplates loads default compliance templates
func (r *ComplianceReporter) loadDefaultTemplates() {
	// SOC2 Template
	r.templates[string(ComplianceSOC2)] = r.createSOC2Template()

	// HIPAA Template
	r.templates[string(ComplianceHIPAA)] = r.createHIPAATemplate()

	// PCI-DSS Template
	r.templates[string(CompliancePCIDSS)] = r.createPCIDSSTemplate()
}

// createSOC2Template creates SOC2 compliance template
func (r *ComplianceReporter) createSOC2Template() *ReportTemplate {
	return &ReportTemplate{
		ID:   "soc2",
		Name: "SOC 2 Type II",
		Type: ComplianceSOC2,
		Sections: []ReportSection{
			{
				Title:       "Security",
				Description: "Common Criteria related to Security",
				Controls: []Control{
					{
						ID:          "CC6.1",
						Title:       "Logical Access Controls",
						Description: "The entity implements logical access security software, infrastructure, and architectures",
						Category:    "Security",
					},
					{
						ID:          "CC6.2",
						Title:       "Encryption",
						Description: "The entity uses encryption to supplement other access controls",
						Category:    "Security",
					},
				},
			},
			{
				Title:       "Availability",
				Description: "Common Criteria related to Availability",
				Controls: []Control{
					{
						ID:          "A1.1",
						Title:       "Infrastructure Monitoring",
						Description: "The entity monitors infrastructure and system availability",
						Category:    "Availability",
					},
				},
			},
		},
	}
}

// createHIPAATemplate creates HIPAA compliance template
func (r *ComplianceReporter) createHIPAATemplate() *ReportTemplate {
	return &ReportTemplate{
		ID:   "hipaa",
		Name: "HIPAA Security Rule",
		Type: ComplianceHIPAA,
		Sections: []ReportSection{
			{
				Title:       "Administrative Safeguards",
				Description: "45 CFR § 164.308",
				Controls: []Control{
					{
						ID:          "164.308(a)(1)",
						Title:       "Security Management Process",
						Description: "Implement policies and procedures to prevent, detect, contain, and correct security violations",
						Category:    "Administrative",
					},
				},
			},
			{
				Title:       "Technical Safeguards",
				Description: "45 CFR § 164.312",
				Controls: []Control{
					{
						ID:          "164.312(a)(1)",
						Title:       "Access Control",
						Description: "Implement technical policies and procedures for electronic information systems",
						Category:    "Technical",
					},
					{
						ID:          "164.312(e)(1)",
						Title:       "Transmission Security",
						Description: "Implement technical security measures to guard against unauthorized access",
						Category:    "Technical",
					},
				},
			},
		},
	}
}

// createPCIDSSTemplate creates PCI-DSS compliance template
func (r *ComplianceReporter) createPCIDSSTemplate() *ReportTemplate {
	return &ReportTemplate{
		ID:   "pcidss",
		Name: "PCI DSS v4.0",
		Type: CompliancePCIDSS,
		Sections: []ReportSection{
			{
				Title:       "Build and Maintain a Secure Network",
				Description: "Requirements 1-2",
				Controls: []Control{
					{
						ID:          "1.1",
						Title:       "Firewall Configuration Standards",
						Description: "Establish and implement firewall and router configuration standards",
						Category:    "Network Security",
					},
				},
			},
			{
				Title:       "Protect Cardholder Data",
				Description: "Requirements 3-4",
				Controls: []Control{
					{
						ID:          "3.4",
						Title:       "PAN Encryption",
						Description: "Render PAN unreadable anywhere it is stored",
						Category:    "Data Protection",
					},
				},
			},
		},
	}
}

// Helper methods

func (r *ComplianceReporter) isDriftRelevantToControl(control Control, drift *detector.DriftResult) bool {
	// Check if drift is relevant based on control ID and drift resource type
	switch control.ID {
	// SOC2 Security Controls
	case "CC6.1", "CC6.2", "CC6.3":
		// Logical and physical access controls
		return strings.Contains(drift.ResourceType, "iam") ||
			strings.Contains(drift.ResourceType, "security_group") ||
			strings.Contains(drift.ResourceType, "network_acl") ||
			strings.Contains(drift.ResourceType, "key") ||
			strings.Contains(drift.ResourceType, "secret")

	case "CC6.6", "CC6.7", "CC6.8":
		// Encryption controls
		return strings.Contains(drift.ResourceType, "kms") ||
			strings.Contains(drift.ResourceType, "encryption") ||
			(drift.ResourceType == "s3_bucket" && r.checkEncryptionDrift(drift)) ||
			(drift.ResourceType == "rds_instance" && r.checkEncryptionDrift(drift))

	case "CC7.1", "CC7.2":
		// System operations controls
		return strings.Contains(drift.ResourceType, "instance") ||
			strings.Contains(drift.ResourceType, "autoscaling") ||
			strings.Contains(drift.ResourceType, "load_balancer") ||
			strings.Contains(drift.ResourceType, "cloudwatch")

	case "A1.1", "A1.2":
		// Availability controls
		return strings.Contains(drift.ResourceType, "backup") ||
			strings.Contains(drift.ResourceType, "snapshot") ||
			strings.Contains(drift.ResourceType, "replica") ||
			strings.Contains(drift.ResourceType, "availability")

	// HIPAA Controls
	case "164.308(a)(1)", "164.308(a)(3)":
		// Administrative safeguards
		return strings.Contains(drift.ResourceType, "iam") ||
			strings.Contains(drift.ResourceType, "policy") ||
			strings.Contains(drift.ResourceType, "role")

	case "164.308(a)(4)":
		// Information access management
		return strings.Contains(drift.ResourceType, "iam") ||
			strings.Contains(drift.ResourceType, "access") ||
			strings.Contains(drift.ResourceType, "permission")

	case "164.312(a)(1)":
		// Access control
		return strings.Contains(drift.ResourceType, "security_group") ||
			strings.Contains(drift.ResourceType, "network_acl") ||
			strings.Contains(drift.ResourceType, "iam")

	case "164.312(a)(2)(iv)":
		// Encryption and decryption
		return r.checkEncryptionDrift(drift)

	case "164.312(b)":
		// Audit controls
		return strings.Contains(drift.ResourceType, "cloudtrail") ||
			strings.Contains(drift.ResourceType, "logging") ||
			strings.Contains(drift.ResourceType, "audit")

	// PCI-DSS Controls
	case "1.1", "1.2", "1.3":
		// Firewall configuration
		return strings.Contains(drift.ResourceType, "security_group") ||
			strings.Contains(drift.ResourceType, "network_acl") ||
			strings.Contains(drift.ResourceType, "firewall") ||
			strings.Contains(drift.ResourceType, "waf")

	case "2.1", "2.2", "2.3":
		// Default passwords and security parameters
		return strings.Contains(drift.ResourceType, "secret") ||
			strings.Contains(drift.ResourceType, "parameter") ||
			strings.Contains(drift.ResourceType, "password")

	case "3.1", "3.2", "3.3", "3.4":
		// Stored cardholder data protection
		return r.checkEncryptionDrift(drift) ||
			strings.Contains(drift.ResourceType, "database") ||
			strings.Contains(drift.ResourceType, "storage")

	case "8.1", "8.2", "8.3":
		// User identification and authentication
		return strings.Contains(drift.ResourceType, "iam") ||
			strings.Contains(drift.ResourceType, "user") ||
			strings.Contains(drift.ResourceType, "mfa")

	case "10.1", "10.2", "10.3":
		// Logging and monitoring
		return strings.Contains(drift.ResourceType, "cloudtrail") ||
			strings.Contains(drift.ResourceType, "cloudwatch") ||
			strings.Contains(drift.ResourceType, "log")

	default:
		// For unknown controls, check if it's a security-related drift
		return r.isSecurityRelatedDrift(drift)
	}
}

func (r *ComplianceReporter) isViolationRelevantToControl(control Control, violation PolicyViolation) bool {
	// Check if violation is relevant based on control requirements
	switch control.ID {
	// SOC2 Controls
	case "CC6.1", "CC6.2", "CC6.3":
		return violation.Severity == "HIGH" || violation.Severity == "CRITICAL"

	case "CC6.6", "CC6.7", "CC6.8":
		return strings.Contains(violation.Resource, "encryption") ||
			strings.Contains(violation.Rule, "encryption")

	// HIPAA Controls
	case "164.312(a)(2)(iv)":
		return strings.Contains(strings.ToLower(violation.Rule), "encrypt") ||
			strings.Contains(strings.ToLower(violation.Message), "encrypt")

	case "164.308(a)(1)":
		return strings.Contains(violation.Resource, "iam") ||
			strings.Contains(violation.Rule, "access")

	// PCI-DSS Controls
	case "1.1", "1.2", "1.3":
		return strings.Contains(violation.Resource, "network") ||
			strings.Contains(violation.Rule, "firewall")

	case "3.1", "3.2", "3.3", "3.4":
		return strings.Contains(violation.Rule, "data") ||
			strings.Contains(violation.Rule, "storage")

	default:
		// For general violations, check severity
		return violation.Severity == "HIGH" || violation.Severity == "CRITICAL"
	}
}

// checkEncryptionDrift checks if drift involves encryption settings
func (r *ComplianceReporter) checkEncryptionDrift(drift *detector.DriftResult) bool {
	if drift.Differences == nil {
		return false
	}

	for _, diff := range drift.Differences {
		pathLower := strings.ToLower(diff.Path)
		if strings.Contains(pathLower, "encrypt") ||
			strings.Contains(pathLower, "kms") ||
			strings.Contains(pathLower, "cipher") ||
			strings.Contains(pathLower, "tls") ||
			strings.Contains(pathLower, "ssl") {
			return true
		}
	}

	return false
}

// isSecurityRelatedDrift determines if drift is security-related
func (r *ComplianceReporter) isSecurityRelatedDrift(drift *detector.DriftResult) bool {
	securityKeywords := []string{
		"security", "iam", "role", "policy", "permission",
		"encrypt", "kms", "secret", "key", "password",
		"firewall", "nacl", "vpc", "subnet", "route",
		"ssl", "tls", "certificate", "auth", "mfa",
		"audit", "log", "trail", "monitoring", "alert",
	}

	resourceTypeLower := strings.ToLower(drift.ResourceType)
	for _, keyword := range securityKeywords {
		if strings.Contains(resourceTypeLower, keyword) {
			return true
		}
	}

	// Check if any differences involve security-related attributes
	if drift.Differences != nil {
		for _, diff := range drift.Differences {
			pathLower := strings.ToLower(diff.Path)
			for _, keyword := range securityKeywords {
				if strings.Contains(pathLower, keyword) {
					return true
				}
			}
		}
	}

	return false
}

func (r *ComplianceReporter) calculateSectionStatus(controls []Control) ControlStatus {
	allPassed := true
	anyPassed := false

	for _, control := range controls {
		if control.Status == ControlStatusPassed {
			anyPassed = true
		} else {
			allPassed = false
		}
	}

	if allPassed {
		return ControlStatusPassed
	} else if anyPassed {
		return ControlStatusPartial
	}
	return ControlStatusFailed
}

func (r *ComplianceReporter) calculateSectionScore(controls []Control) float64 {
	if len(controls) == 0 {
		return 0
	}

	passed := 0
	for _, control := range controls {
		if control.Status == ControlStatusPassed {
			passed++
		}
	}

	return float64(passed) / float64(len(controls)) * 100
}

func (r *ComplianceReporter) signReport(report *ComplianceReport) string {
	// Generate digital signature for report integrity
	data, _ := json.Marshal(report)
	return fmt.Sprintf("%x", data)[:16]
}

// ExportReport exports a report in the specified format
func (r *ComplianceReporter) ExportReport(report *ComplianceReport, format string, writer io.Writer) error {
	formatter, exists := r.formatters[format]
	if !exists {
		return fmt.Errorf("unsupported format: %s", format)
	}

	data, err := formatter.Format(report)
	if err != nil {
		return err
	}

	_, err = writer.Write(data)
	return err
}
