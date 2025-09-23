package security

import (
	"context"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPolicyEngine tests the creation of a new policy engine
func TestNewPolicyEngine(t *testing.T) {
	eventBus := &MockEventBus{}
	engine := NewPolicyEngine(eventBus)

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.policies)
	assert.NotNil(t, engine.rules)
	assert.NotNil(t, engine.enforcements)
	assert.Equal(t, eventBus, engine.eventBus)
	assert.NotNil(t, engine.config)

	// Check default configuration
	config := engine.GetConfig()
	assert.Equal(t, "warn", config.DefaultEnforcement)
	assert.False(t, config.AutoRemediation)
	assert.True(t, config.NotificationEnabled)
	assert.True(t, config.AuditLogging)
	assert.Equal(t, 90*24*time.Hour, config.RetentionPeriod)
}

// TestPolicyEngine_CreatePolicy tests creating security policies
func TestPolicyEngine_CreatePolicy(t *testing.T) {
	eventBus := &MockEventBus{}
	engine := NewPolicyEngine(eventBus)
	ctx := context.Background()

	policy := &SecurityPolicy{
		Name:        "Test Security Policy",
		Description: "Test policy for unit testing",
		Category:    "test",
		Priority:    "medium",
		Rules:       []string{"rule1", "rule2"},
		Scope: PolicyScope{
			Regions: []string{"us-east-1"},
		},
		Enabled: true,
	}

	err := engine.CreatePolicy(ctx, policy)
	assert.NoError(t, err)

	// Verify policy was created
	assert.NotEmpty(t, policy.ID)
	assert.NotZero(t, policy.CreatedAt)
	assert.NotZero(t, policy.UpdatedAt)

	// Check that events were published
	events := eventBus.GetEvents()
	var policyCreatedEvent *ComplianceEvent
	for _, event := range events {
		if event.Type == "security_policy_created" {
			policyCreatedEvent = &event
			break
		}
	}
	assert.NotNil(t, policyCreatedEvent)
	assert.Equal(t, policy.ID, policyCreatedEvent.PolicyID)
	assert.Contains(t, policyCreatedEvent.Message, "Security policy")
	assert.Equal(t, "info", policyCreatedEvent.Severity)
}

// TestPolicyEngine_CreatePolicy_Validation tests policy validation
func TestPolicyEngine_CreatePolicy_Validation(t *testing.T) {
	eventBus := &MockEventBus{}
	engine := NewPolicyEngine(eventBus)
	ctx := context.Background()

	testCases := []struct {
		name    string
		policy  *SecurityPolicy
		wantErr bool
	}{
		{
			name: "Valid policy",
			policy: &SecurityPolicy{
				Name:        "Valid Policy",
				Description: "Valid policy",
				Category:    "test",
				Rules:       []string{"rule1"},
				Enabled:     true,
			},
			wantErr: false,
		},
		{
			name: "Missing name",
			policy: &SecurityPolicy{
				Description: "Policy without name",
				Category:    "test",
				Rules:       []string{"rule1"},
				Enabled:     true,
			},
			wantErr: true,
		},
		{
			name: "Missing category",
			policy: &SecurityPolicy{
				Name:        "Policy without category",
				Description: "Policy without category",
				Rules:       []string{"rule1"},
				Enabled:     true,
			},
			wantErr: true,
		},
		{
			name: "No rules",
			policy: &SecurityPolicy{
				Name:        "Policy without rules",
				Description: "Policy without rules",
				Category:    "test",
				Rules:       []string{},
				Enabled:     true,
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := engine.CreatePolicy(ctx, tc.policy)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPolicyEngine_CreateRule tests creating security rules
func TestPolicyEngine_CreateRule(t *testing.T) {
	eventBus := &MockEventBus{}
	engine := NewPolicyEngine(eventBus)
	ctx := context.Background()

	rule := &SecurityRule{
		Name:        "Test Security Rule",
		Description: "Test rule for unit testing",
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
				Description: "Enable encryption",
			},
		},
		Severity: "high",
		Enabled:  true,
	}

	err := engine.CreateRule(ctx, rule)
	assert.NoError(t, err)

	// Verify rule was created
	assert.NotEmpty(t, rule.ID)
	assert.NotZero(t, rule.CreatedAt)
	assert.NotZero(t, rule.UpdatedAt)

	// Check that events were published
	events := eventBus.GetEvents()
	var ruleCreatedEvent *ComplianceEvent
	for _, event := range events {
		if event.Type == "security_rule_created" {
			ruleCreatedEvent = &event
			break
		}
	}
	assert.NotNil(t, ruleCreatedEvent)
	assert.Contains(t, ruleCreatedEvent.Message, "Security rule")
	assert.Equal(t, "info", ruleCreatedEvent.Severity)
}

// TestPolicyEngine_CreateRule_Validation tests rule validation
func TestPolicyEngine_CreateRule_Validation(t *testing.T) {
	eventBus := &MockEventBus{}
	engine := NewPolicyEngine(eventBus)
	ctx := context.Background()

	testCases := []struct {
		name    string
		rule    *SecurityRule
		wantErr bool
	}{
		{
			name: "Valid rule",
			rule: &SecurityRule{
				Name:        "Valid Rule",
				Description: "Valid rule",
				Type:        "encryption",
				Category:    "test",
				Conditions: []RuleCondition{
					{
						Field:    "encryption",
						Operator: "equals",
						Value:    true,
						Type:     "boolean",
					},
				},
				Enabled: true,
			},
			wantErr: false,
		},
		{
			name: "Missing name",
			rule: &SecurityRule{
				Description: "Rule without name",
				Type:        "encryption",
				Category:    "test",
				Conditions: []RuleCondition{
					{
						Field:    "encryption",
						Operator: "equals",
						Value:    true,
						Type:     "boolean",
					},
				},
				Enabled: true,
			},
			wantErr: true,
		},
		{
			name: "Missing type",
			rule: &SecurityRule{
				Name:        "Rule without type",
				Description: "Rule without type",
				Category:    "test",
				Conditions: []RuleCondition{
					{
						Field:    "encryption",
						Operator: "equals",
						Value:    true,
						Type:     "boolean",
					},
				},
				Enabled: true,
			},
			wantErr: true,
		},
		{
			name: "No conditions",
			rule: &SecurityRule{
				Name:        "Rule without conditions",
				Description: "Rule without conditions",
				Type:        "encryption",
				Category:    "test",
				Conditions:  []RuleCondition{},
				Enabled:     true,
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := engine.CreateRule(ctx, tc.rule)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPolicyEngine_EvaluatePolicy tests policy evaluation
func TestPolicyEngine_EvaluatePolicy(t *testing.T) {
	eventBus := &MockEventBus{}
	engine := NewPolicyEngine(eventBus)
	ctx := context.Background()

	// Create a rule first
	rule := &SecurityRule{
		Name:        "Encryption Rule",
		Description: "Check encryption",
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
				Description: "Enable encryption",
			},
		},
		Severity: "high",
		Enabled:  true,
	}

	err := engine.CreateRule(ctx, rule)
	require.NoError(t, err)

	// Create a policy
	policy := &SecurityPolicy{
		Name:        "Test Policy",
		Description: "Test policy",
		Category:    "test",
		Rules:       []string{rule.ID},
		Scope: PolicyScope{
			Regions: []string{"us-east-1"},
		},
		Enabled: true,
	}

	err = engine.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Test resource that violates the policy
	resource := &models.Resource{
		ID:       "resource-1",
		Name:     "Test Resource",
		Type:     "aws_s3_bucket",
		Provider: "aws",
		Region:   "us-east-1",
		State:    "active",
		Attributes: map[string]interface{}{
			"encryption": false, // This should violate the rule
		},
	}

	evaluation, err := engine.EvaluatePolicy(ctx, policy.ID, resource)
	assert.NoError(t, err)
	assert.NotNil(t, evaluation)

	// Verify evaluation structure
	assert.Equal(t, policy.ID, evaluation.PolicyID)
	assert.Equal(t, resource.ID, evaluation.ResourceID)
	assert.Equal(t, "NON_COMPLIANT", evaluation.Status)
	assert.Contains(t, evaluation.Message, "violates")
	assert.NotZero(t, evaluation.Timestamp)
	assert.NotNil(t, evaluation.RuleResults)
	assert.NotNil(t, evaluation.Violations)

	// Check that violations were found
	assert.Greater(t, len(evaluation.Violations), 0)
	violation := evaluation.Violations[0]
	assert.Equal(t, rule.ID, violation.RuleID)
	assert.Equal(t, "encryption", violation.Type)
	assert.Equal(t, "high", violation.Severity)
	assert.NotEmpty(t, violation.Remediation)
}

// TestPolicyEngine_EvaluatePolicy_CompliantResource tests evaluation of compliant resource
func TestPolicyEngine_EvaluatePolicy_CompliantResource(t *testing.T) {
	eventBus := &MockEventBus{}
	engine := NewPolicyEngine(eventBus)
	ctx := context.Background()

	// Create a rule first
	rule := &SecurityRule{
		Name:        "Encryption Rule",
		Description: "Check encryption",
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
				Description: "Enable encryption",
			},
		},
		Severity: "high",
		Enabled:  true,
	}

	err := engine.CreateRule(ctx, rule)
	require.NoError(t, err)

	// Create a policy
	policy := &SecurityPolicy{
		Name:        "Test Policy",
		Description: "Test policy",
		Category:    "test",
		Rules:       []string{rule.ID},
		Scope: PolicyScope{
			Regions: []string{"us-east-1"},
		},
		Enabled: true,
	}

	err = engine.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Test resource that complies with the policy
	resource := &models.Resource{
		ID:       "resource-1",
		Name:     "Test Resource",
		Type:     "aws_s3_bucket",
		Provider: "aws",
		Region:   "us-east-1",
		State:    "active",
		Attributes: map[string]interface{}{
			"encryption": true, // This should comply with the rule
		},
	}

	evaluation, err := engine.EvaluatePolicy(ctx, policy.ID, resource)
	assert.NoError(t, err)
	assert.NotNil(t, evaluation)

	// Verify evaluation structure
	assert.Equal(t, policy.ID, evaluation.PolicyID)
	assert.Equal(t, resource.ID, evaluation.ResourceID)
	assert.Equal(t, "COMPLIANT", evaluation.Status)
	assert.Contains(t, evaluation.Message, "complies")
	assert.NotZero(t, evaluation.Timestamp)
	assert.NotNil(t, evaluation.RuleResults)
	assert.Equal(t, 0, len(evaluation.Violations))
}

// TestPolicyEngine_EvaluatePolicy_NotApplicable tests evaluation when policy doesn't apply
func TestPolicyEngine_EvaluatePolicy_NotApplicable(t *testing.T) {
	eventBus := &MockEventBus{}
	engine := NewPolicyEngine(eventBus)
	ctx := context.Background()

	// Create a policy with specific region scope
	policy := &SecurityPolicy{
		Name:        "Test Policy",
		Description: "Test policy",
		Category:    "test",
		Rules:       []string{"rule1"}, // Need at least one rule for validation
		Scope: PolicyScope{
			Regions: []string{"us-west-2"}, // Different region
		},
		Enabled: true,
	}

	err := engine.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Create the rule that the policy references
	rule := &SecurityRule{
		ID:          "rule1",
		Name:        "Test Rule",
		Description: "Test rule",
		Type:        "security",
		Conditions: []RuleCondition{
			{
				Field:    "encryption",
				Operator: "equals",
				Value:    true,
				Type:     "boolean",
			},
		},
		Enabled: true,
	}
	err = engine.CreateRule(ctx, rule)
	require.NoError(t, err)

	// Test resource in different region
	resource := &models.Resource{
		ID:       "resource-1",
		Name:     "Test Resource",
		Type:     "aws_s3_bucket",
		Provider: "aws",
		Region:   "us-east-1", // Different region
		State:    "active",
		Attributes: map[string]interface{}{
			"encryption": false,
		},
	}

	evaluation, err := engine.EvaluatePolicy(ctx, policy.ID, resource)
	assert.NoError(t, err)
	assert.NotNil(t, evaluation)

	// Verify evaluation structure
	assert.Equal(t, policy.ID, evaluation.PolicyID)
	assert.Equal(t, resource.ID, evaluation.ResourceID)
	assert.Equal(t, "NOT_APPLICABLE", evaluation.Status)
	assert.Contains(t, evaluation.Message, "does not apply")
	assert.NotZero(t, evaluation.Timestamp)
}

// TestPolicyEngine_EvaluatePolicy_NonExistentPolicy tests evaluation of non-existent policy
func TestPolicyEngine_EvaluatePolicy_NonExistentPolicy(t *testing.T) {
	eventBus := &MockEventBus{}
	engine := NewPolicyEngine(eventBus)
	ctx := context.Background()

	resource := &models.Resource{
		ID:       "resource-1",
		Name:     "Test Resource",
		Type:     "aws_s3_bucket",
		Provider: "aws",
		Region:   "us-east-1",
		State:    "active",
		Attributes: map[string]interface{}{
			"encryption": false,
		},
	}

	evaluation, err := engine.EvaluatePolicy(ctx, "non-existent-policy", resource)
	assert.Error(t, err)
	assert.Nil(t, evaluation)
	assert.Contains(t, err.Error(), "not found")
}

// TestPolicyEngine_SetConfig tests setting configuration
func TestPolicyEngine_SetConfig(t *testing.T) {
	eventBus := &MockEventBus{}
	engine := NewPolicyEngine(eventBus)

	newConfig := &PolicyConfig{
		DefaultEnforcement:  "block",
		AutoRemediation:     true,
		NotificationEnabled: false,
		AuditLogging:        false,
		RetentionPeriod:     30 * 24 * time.Hour,
	}

	engine.SetConfig(newConfig)

	config := engine.GetConfig()
	assert.Equal(t, newConfig.DefaultEnforcement, config.DefaultEnforcement)
	assert.Equal(t, newConfig.AutoRemediation, config.AutoRemediation)
	assert.Equal(t, newConfig.NotificationEnabled, config.NotificationEnabled)
	assert.Equal(t, newConfig.AuditLogging, config.AuditLogging)
	assert.Equal(t, newConfig.RetentionPeriod, config.RetentionPeriod)
}

// TestPolicyEngine_GetConfig tests getting configuration
func TestPolicyEngine_GetConfig(t *testing.T) {
	eventBus := &MockEventBus{}
	engine := NewPolicyEngine(eventBus)

	config := engine.GetConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "warn", config.DefaultEnforcement)
	assert.False(t, config.AutoRemediation)
	assert.True(t, config.NotificationEnabled)
	assert.True(t, config.AuditLogging)
	assert.Equal(t, 90*24*time.Hour, config.RetentionPeriod)
}

// TestPolicyEngine_ConcurrentAccess tests concurrent access to the policy engine
func TestPolicyEngine_ConcurrentAccess(t *testing.T) {
	eventBus := &MockEventBus{}
	engine := NewPolicyEngine(eventBus)
	ctx := context.Background()

	// Create a rule and policy
	rule := &SecurityRule{
		Name:        "Test Rule",
		Description: "Test rule",
		Type:        "encryption",
		Category:    "test",
		Conditions: []RuleCondition{
			{
				Field:    "encryption",
				Operator: "equals",
				Value:    true,
				Type:     "boolean",
			},
		},
		Enabled: true,
	}

	err := engine.CreateRule(ctx, rule)
	require.NoError(t, err)

	policy := &SecurityPolicy{
		Name:        "Test Policy",
		Description: "Test policy",
		Category:    "test",
		Rules:       []string{rule.ID},
		Enabled:     true,
	}

	err = engine.CreatePolicy(ctx, policy)
	require.NoError(t, err)

	// Create test resource
	resource := &models.Resource{
		ID:       "resource-1",
		Name:     "Test Resource",
		Type:     "aws_s3_bucket",
		Provider: "aws",
		Region:   "us-east-1",
		State:    "active",
		Attributes: map[string]interface{}{
			"encryption": false,
		},
	}

	// Run concurrent evaluations
	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := engine.EvaluatePolicy(ctx, policy.ID, resource)
			results <- err
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		assert.NoError(t, err)
	}
}

// TestPolicyEngine_ConditionOperators tests different condition operators
func TestPolicyEngine_ConditionOperators(t *testing.T) {
	eventBus := &MockEventBus{}
	engine := NewPolicyEngine(eventBus)

	testCases := []struct {
		name     string
		operator string
		value    interface{}
		actual   interface{}
		expected bool
	}{
		{
			name:     "equals - true",
			operator: "equals",
			value:    true,
			actual:   true,
			expected: true,
		},
		{
			name:     "equals - false",
			operator: "equals",
			value:    true,
			actual:   false,
			expected: false,
		},
		{
			name:     "not_equals - true",
			operator: "not_equals",
			value:    true,
			actual:   false,
			expected: true,
		},
		{
			name:     "not_equals - false",
			operator: "not_equals",
			value:    true,
			actual:   true,
			expected: false,
		},
		{
			name:     "contains - true",
			operator: "contains",
			value:    "test",
			actual:   "this is a test",
			expected: true,
		},
		{
			name:     "contains - false",
			operator: "contains",
			value:    "missing",
			actual:   "this is a test",
			expected: false,
		},
		{
			name:     "not_contains - true",
			operator: "not_contains",
			value:    "missing",
			actual:   "this is a test",
			expected: true,
		},
		{
			name:     "not_contains - false",
			operator: "not_contains",
			value:    "test",
			actual:   "this is a test",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			condition := RuleCondition{
				Field:    "test_field",
				Operator: tc.operator,
				Value:    tc.value,
				Type:     "string",
			}

			resource := &models.Resource{
				ID:       "resource-1",
				Name:     "Test Resource",
				Type:     "aws_s3_bucket",
				Provider: "aws",
				Region:   "us-east-1",
				State:    "active",
				Attributes: map[string]interface{}{
					"test_field": tc.actual,
				},
			}

			result := engine.evaluateCondition(condition, resource)
			assert.Equal(t, tc.expected, result)
		})
	}
}
