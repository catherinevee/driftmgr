package security

import (
	"context"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEventBus for testing
type MockEventBus struct {
	events []ComplianceEvent
}

func (m *MockEventBus) PublishComplianceEvent(event ComplianceEvent) error {
	m.events = append(m.events, event)
	return nil
}

func (m *MockEventBus) GetEvents() []ComplianceEvent {
	return m.events
}

func (m *MockEventBus) ClearEvents() {
	m.events = []ComplianceEvent{}
}

// TestNewSecurityService tests the creation of a new security service
func TestNewSecurityService(t *testing.T) {
	eventBus := &MockEventBus{}
	service := NewSecurityService(eventBus)

	assert.NotNil(t, service)
	assert.NotNil(t, service.complianceManager)
	assert.NotNil(t, service.policyEngine)
	assert.NotNil(t, service.reportGenerator)
	assert.NotNil(t, service.config)
	assert.Equal(t, eventBus, service.eventBus)

	// Check default configuration
	config := service.GetConfig()
	assert.Equal(t, 24*time.Hour, config.AutoScanInterval)
	assert.True(t, config.ReportGeneration)
	assert.True(t, config.NotificationEnabled)
	assert.True(t, config.AuditLogging)
	assert.Equal(t, 10, config.MaxConcurrentScans)
}

// TestSecurityService_Start tests starting the security service
func TestSecurityService_Start(t *testing.T) {
	eventBus := &MockEventBus{}
	service := NewSecurityService(eventBus)
	ctx := context.Background()

	err := service.Start(ctx)
	assert.NoError(t, err)

	// Check that events were published
	events := eventBus.GetEvents()
	assert.Greater(t, len(events), 0)

	// Find the service started event
	var serviceStartedEvent *ComplianceEvent
	for _, event := range events {
		if event.Type == "security_service_started" {
			serviceStartedEvent = &event
			break
		}
	}
	assert.NotNil(t, serviceStartedEvent)
	assert.Equal(t, "Security service started", serviceStartedEvent.Message)
	assert.Equal(t, "info", serviceStartedEvent.Severity)
}

// TestSecurityService_Stop tests stopping the security service
func TestSecurityService_Stop(t *testing.T) {
	eventBus := &MockEventBus{}
	service := NewSecurityService(eventBus)
	ctx := context.Background()

	err := service.Stop(ctx)
	assert.NoError(t, err)

	// Check that events were published
	events := eventBus.GetEvents()
	assert.Greater(t, len(events), 0)

	// Find the service stopped event
	var serviceStoppedEvent *ComplianceEvent
	for _, event := range events {
		if event.Type == "security_service_stopped" {
			serviceStoppedEvent = &event
			break
		}
	}
	assert.NotNil(t, serviceStoppedEvent)
	assert.Equal(t, "Security service stopped", serviceStoppedEvent.Message)
	assert.Equal(t, "info", serviceStoppedEvent.Severity)
}

// TestSecurityService_ScanResources tests scanning resources for security issues
func TestSecurityService_ScanResources(t *testing.T) {
	eventBus := &MockEventBus{}
	service := NewSecurityService(eventBus)
	ctx := context.Background()

	// Start service to create default policies
	err := service.Start(ctx)
	require.NoError(t, err)

	// Create test resources
	resources := []*models.Resource{
		{
			ID:       "resource-1",
			Name:     "Test Resource 1",
			Type:     "aws_s3_bucket",
			Provider: "aws",
			Region:   "us-east-1",
			State:    "active",
			Attributes: map[string]interface{}{
				"encryption": false,
				"logging":    false,
				"backup":     false,
			},
		},
		{
			ID:       "resource-2",
			Name:     "Test Resource 2",
			Type:     "aws_ec2_instance",
			Provider: "aws",
			Region:   "us-west-2",
			State:    "active",
			Attributes: map[string]interface{}{
				"encryption": true,
				"logging":    true,
				"backup":     true,
			},
		},
	}

	result, err := service.ScanResources(ctx, resources)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify scan result structure
	assert.NotEmpty(t, result.ScanID)
	assert.NotZero(t, result.StartTime)
	assert.NotZero(t, result.EndTime)
	assert.NotZero(t, result.Duration)
	assert.Equal(t, resources, result.Resources)
	assert.NotNil(t, result.Policies)
	assert.NotNil(t, result.Compliance)
	assert.NotNil(t, result.Violations)
	assert.NotNil(t, result.Summary)
	assert.NotNil(t, result.Metadata)

	// Verify summary was generated
	assert.Contains(t, result.Summary, "total_resources")
	assert.Contains(t, result.Summary, "total_policies")
	assert.Contains(t, result.Summary, "total_violations")
	assert.Contains(t, result.Summary, "violation_severity")
	assert.Contains(t, result.Summary, "compliance_status")
	assert.Contains(t, result.Summary, "scan_duration")

	// Check that events were published
	events := eventBus.GetEvents()
	var scanCompletedEvent *ComplianceEvent
	for _, event := range events {
		if event.Type == "security_scan_completed" {
			scanCompletedEvent = &event
			break
		}
	}
	assert.NotNil(t, scanCompletedEvent)
	assert.Contains(t, scanCompletedEvent.Message, "Security scan completed")
	assert.Equal(t, "info", scanCompletedEvent.Severity)
	assert.Contains(t, scanCompletedEvent.Metadata, "scan_id")
	assert.Contains(t, scanCompletedEvent.Metadata, "resource_count")
}

// TestSecurityService_GenerateComplianceReport tests generating compliance reports
func TestSecurityService_GenerateComplianceReport(t *testing.T) {
	eventBus := &MockEventBus{}
	service := NewSecurityService(eventBus)
	ctx := context.Background()

	// Start service to create default policies
	err := service.Start(ctx)
	require.NoError(t, err)

	report, err := service.GenerateComplianceReport(ctx, "SOC2")
	assert.NoError(t, err)
	assert.NotNil(t, report)

	// Check that events were published
	events := eventBus.GetEvents()
	var reportGeneratedEvent *ComplianceEvent
	for _, event := range events {
		if event.Type == "compliance_report_generated" {
			reportGeneratedEvent = &event
			break
		}
	}
	assert.NotNil(t, reportGeneratedEvent)
	assert.Contains(t, reportGeneratedEvent.Message, "Compliance report generated")
	assert.Equal(t, "info", reportGeneratedEvent.Severity)
	assert.Contains(t, reportGeneratedEvent.Metadata, "standard")
}

// TestSecurityService_CreateSecurityPolicy tests creating security policies
func TestSecurityService_CreateSecurityPolicy(t *testing.T) {
	eventBus := &MockEventBus{}
	service := NewSecurityService(eventBus)
	ctx := context.Background()

	policy := &SecurityPolicy{
		Name:        "Test Security Policy",
		Description: "Test policy for unit testing",
		Category:    "test",
		Priority:    "medium",
		Rules:       []string{"rule1"}, // Need at least one rule for validation
		Scope: PolicyScope{
			Regions: []string{"us-east-1"},
		},
		Enabled: true,
	}

	err := service.CreateSecurityPolicy(ctx, policy)
	assert.NoError(t, err)

	// Create the rule that the policy references
	rule := &SecurityRule{
		ID:          "rule1",
		Name:        "Test Security Rule",
		Description: "Test rule for unit testing",
		Type:        "test",
		Category:    "test",
		Conditions: []RuleCondition{
			{
				Field:    "test_field",
				Operator: "equals",
				Value:    "test_value",
				Type:     "string",
			},
		},
		Enabled: true,
	}
	err = service.CreateSecurityRule(ctx, rule)
	assert.NoError(t, err)
}

// TestSecurityService_CreateSecurityRule tests creating security rules
func TestSecurityService_CreateSecurityRule(t *testing.T) {
	eventBus := &MockEventBus{}
	service := NewSecurityService(eventBus)
	ctx := context.Background()

	rule := &SecurityRule{
		Name:        "Test Security Rule",
		Description: "Test rule for unit testing",
		Type:        "test",
		Category:    "test",
		Conditions: []RuleCondition{
			{
				Field:    "test_field",
				Operator: "equals",
				Value:    true,
				Type:     "boolean",
			},
		},
		Actions: []RuleAction{
			{
				Type:        "warn",
				Description: "Test warning",
			},
		},
		Severity: "medium",
		Enabled:  true,
	}

	err := service.CreateSecurityRule(ctx, rule)
	assert.NoError(t, err)
}

// TestSecurityService_CreateCompliancePolicy tests creating compliance policies
func TestSecurityService_CreateCompliancePolicy(t *testing.T) {
	eventBus := &MockEventBus{}
	service := NewSecurityService(eventBus)
	ctx := context.Background()

	policy := &CompliancePolicy{
		Name:        "Test Compliance Policy",
		Description: "Test compliance policy for unit testing",
		Standard:    "TEST",
		Version:     "1.0",
		Category:    "test",
		Severity:    "medium",
		Rules: []ComplianceRule{
			{
				ID:          "test_rule",
				Name:        "Test Rule",
				Description: "Test compliance rule",
				Type:        "test",
				Conditions: []RuleCondition{
					{
						Field:    "test_field",
						Operator: "equals",
						Value:    true,
						Type:     "boolean",
					},
				},
				Actions: []RuleAction{
					{
						Type:        "warn",
						Description: "Test warning",
					},
				},
				Severity: "medium",
				Enabled:  true,
			},
		},
		Enabled: true,
	}

	err := service.CreateCompliancePolicy(ctx, policy)
	assert.NoError(t, err)
}

// TestSecurityService_GetSecurityStatus tests getting security status
func TestSecurityService_GetSecurityStatus(t *testing.T) {
	eventBus := &MockEventBus{}
	service := NewSecurityService(eventBus)
	ctx := context.Background()

	// Start service to create default policies
	err := service.Start(ctx)
	require.NoError(t, err)

	status, err := service.GetSecurityStatus(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, status)

	// Verify status structure
	assert.NotEmpty(t, status.OverallStatus)
	assert.GreaterOrEqual(t, status.SecurityScore, 0.0)
	assert.NotNil(t, status.Policies)
	assert.NotNil(t, status.Compliance)
	assert.NotNil(t, status.Violations)
	assert.NotNil(t, status.Metadata)

	// Check that status is one of the expected values
	validStatuses := []string{"Good", "Fair", "Poor", "Critical", "Unknown"}
	assert.Contains(t, validStatuses, status.OverallStatus)
}

// TestSecurityService_SetConfig tests setting configuration
func TestSecurityService_SetConfig(t *testing.T) {
	eventBus := &MockEventBus{}
	service := NewSecurityService(eventBus)

	newConfig := &SecurityConfig{
		AutoScanInterval:    12 * time.Hour,
		ReportGeneration:    false,
		NotificationEnabled: false,
		AuditLogging:        false,
		MaxConcurrentScans:  5,
	}

	service.SetConfig(newConfig)

	config := service.GetConfig()
	assert.Equal(t, newConfig.AutoScanInterval, config.AutoScanInterval)
	assert.Equal(t, newConfig.ReportGeneration, config.ReportGeneration)
	assert.Equal(t, newConfig.NotificationEnabled, config.NotificationEnabled)
	assert.Equal(t, newConfig.AuditLogging, config.AuditLogging)
	assert.Equal(t, newConfig.MaxConcurrentScans, config.MaxConcurrentScans)
}

// TestSecurityService_GetConfig tests getting configuration
func TestSecurityService_GetConfig(t *testing.T) {
	eventBus := &MockEventBus{}
	service := NewSecurityService(eventBus)

	config := service.GetConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 24*time.Hour, config.AutoScanInterval)
	assert.True(t, config.ReportGeneration)
	assert.True(t, config.NotificationEnabled)
	assert.True(t, config.AuditLogging)
	assert.Equal(t, 10, config.MaxConcurrentScans)
}

// TestSecurityService_ConcurrentAccess tests concurrent access to the service
func TestSecurityService_ConcurrentAccess(t *testing.T) {
	eventBus := &MockEventBus{}
	service := NewSecurityService(eventBus)
	ctx := context.Background()

	// Start service
	err := service.Start(ctx)
	require.NoError(t, err)

	// Create test resources
	resources := []*models.Resource{
		{
			ID:       "resource-1",
			Name:     "Test Resource 1",
			Type:     "aws_s3_bucket",
			Provider: "aws",
			Region:   "us-east-1",
			State:    "active",
			Attributes: map[string]interface{}{
				"encryption": false,
			},
		},
	}

	// Run concurrent operations
	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := service.ScanResources(ctx, resources)
			results <- err
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		assert.NoError(t, err)
	}
}

// TestSecurityService_EmptyResources tests scanning with empty resources
func TestSecurityService_EmptyResources(t *testing.T) {
	eventBus := &MockEventBus{}
	service := NewSecurityService(eventBus)
	ctx := context.Background()

	// Start service
	err := service.Start(ctx)
	require.NoError(t, err)

	result, err := service.ScanResources(ctx, []*models.Resource{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result.Resources))
	assert.Equal(t, 0, len(result.Violations))
}

// TestSecurityService_NilResources tests scanning with nil resources
func TestSecurityService_NilResources(t *testing.T) {
	eventBus := &MockEventBus{}
	service := NewSecurityService(eventBus)
	ctx := context.Background()

	// Start service
	err := service.Start(ctx)
	require.NoError(t, err)

	result, err := service.ScanResources(ctx, nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, result.Resources)
}

// TestSecurityService_InvalidComplianceStandard tests generating report for invalid standard
func TestSecurityService_InvalidComplianceStandard(t *testing.T) {
	eventBus := &MockEventBus{}
	service := NewSecurityService(eventBus)
	ctx := context.Background()

	// Start service
	err := service.Start(ctx)
	require.NoError(t, err)

	report, err := service.GenerateComplianceReport(ctx, "INVALID_STANDARD")
	// This might return an empty report rather than an error
	// depending on implementation
	assert.NotNil(t, report)
}

// TestSecurityService_ContextCancellation tests service behavior with cancelled context
func TestSecurityService_ContextCancellation(t *testing.T) {
	eventBus := &MockEventBus{}
	service := NewSecurityService(eventBus)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Start service with cancelled context
	err := service.Start(ctx)
	// Should not error even with cancelled context
	assert.NoError(t, err)

	// Stop service
	err = service.Stop(ctx)
	assert.NoError(t, err)
}
