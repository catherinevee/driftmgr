package compliance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestComplianceTypes(t *testing.T) {
	types := []ComplianceType{
		ComplianceSOC2,
		ComplianceHIPAA,
		CompliancePCIDSS,
		ComplianceISO27001,
		ComplianceGDPR,
		ComplianceCustom,
	}

	expectedNames := []string{
		"SOC2",
		"HIPAA",
		"PCI-DSS",
		"ISO27001",
		"GDPR",
		"Custom",
	}

	for i, ct := range types {
		assert.Equal(t, ComplianceType(expectedNames[i]), ct)
		assert.NotEmpty(t, string(ct))
	}
}

func TestControlStatus(t *testing.T) {
	statuses := []ControlStatus{
		ControlStatus("compliant"),
		ControlStatus("non-compliant"),
		ControlStatus("partial"),
		ControlStatus("not-applicable"),
		ControlStatus("unknown"),
	}

	for _, status := range statuses {
		assert.NotEmpty(t, string(status))
	}
}

func TestComplianceReporter(t *testing.T) {
	reporter := &ComplianceReporter{
		templates:  make(map[string]*ReportTemplate),
		formatters: make(map[string]Formatter),
	}

	assert.NotNil(t, reporter)
	assert.NotNil(t, reporter.templates)
	assert.NotNil(t, reporter.formatters)
}

func TestReportTemplate(t *testing.T) {
	template := &ReportTemplate{
		ID:   "test-template",
		Name: "Test Template",
		Type: ComplianceCustom,
		Sections: []ReportSection{
			{
				Title:       "Security",
				Description: "Security controls",
				Status:      ControlStatus("compliant"),
			},
		},
	}

	assert.Equal(t, "test-template", template.ID)
	assert.Equal(t, "Test Template", template.Name)
	assert.Equal(t, ComplianceCustom, template.Type)
	assert.Len(t, template.Sections, 1)
}

func TestControl(t *testing.T) {
	control := Control{
		ID:          "ctrl-001",
		Title:       "Encryption at Rest",
		Description: "All data must be encrypted at rest",
		Category:    "Security",
		Status:      ControlStatus("compliant"),
	}

	assert.Equal(t, "ctrl-001", control.ID)
	assert.Equal(t, "Encryption at Rest", control.Title)
	assert.NotEmpty(t, control.Description)
	assert.Equal(t, "Security", control.Category)
	assert.Equal(t, ControlStatus("compliant"), control.Status)
}

func TestEvidence(t *testing.T) {
	evidence := Evidence{
		Type:        "log",
		Description: "CloudTrail audit logs",
		Source:      "AWS CloudTrail",
		Timestamp:   time.Now(),
		Data: map[string]interface{}{
			"event_count": 1000,
		},
	}

	assert.Equal(t, "log", evidence.Type)
	assert.NotEmpty(t, evidence.Description)
	assert.NotEmpty(t, evidence.Source)
	assert.NotZero(t, evidence.Timestamp)
	assert.NotNil(t, evidence.Data)
}
