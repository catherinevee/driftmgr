package compliance

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/compliance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOPAEngine_LoadPolicies(t *testing.T) {
	// Create a temporary directory for test policies
	tempDir := t.TempDir()

	// Create test policy files
	testPolicies := map[string]string{
		"test1.rego": `package test.policy1

default allow = false

allow {
    input.action == "read"
}`,
		"test2.rego": `package test.policy2

default allow = false

allow {
    input.action == "write"
}

violations["missing_tag"] {
    not input.tags.Owner
    violation := {
        "rule": "required_tags",
        "message": "Missing required tag: Owner",
        "severity": "medium"
    }
}`,
	}

	for filename, content := range testPolicies {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Test loading policies
	config := compliance.OPAConfig{
		LocalPolicies: tempDir,
		CacheDuration: 5 * time.Minute,
		Timeout:       10 * time.Second,
	}

	engine := compliance.NewOPAEngine(config)
	ctx := context.Background()

	err := engine.LoadPolicies(ctx)
	assert.NoError(t, err)

	// Verify policies were loaded
	policies := engine.ListPolicies()
	assert.Len(t, policies, 2)

	// Check specific policies
	policy1, exists := engine.GetPolicy("test1.rego")
	assert.True(t, exists)
	assert.Equal(t, "test.policy1", policy1.Package)
	assert.Equal(t, "test1", policy1.Name)

	policy2, exists := engine.GetPolicy("test2.rego")
	assert.True(t, exists)
	assert.Equal(t, "test.policy2", policy2.Package)
	assert.Equal(t, "test2", policy2.Name)
}

func TestOPAEngine_Evaluate(t *testing.T) {
	// Create a temporary directory for test policies
	tempDir := t.TempDir()

	// Create a test policy
	policyContent := `package test.policy

default allow = false

allow {
    input.action == "read"
}

violations["missing_owner"] {
    not input.tags.Owner
    violation := {
        "rule": "required_tags",
        "message": "Missing required tag: Owner",
        "severity": "medium",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Add Owner tag to the resource"
    }
    violation
}`

	filePath := filepath.Join(tempDir, "test.rego")
	err := os.WriteFile(filePath, []byte(policyContent), 0644)
	require.NoError(t, err)

	// Setup engine
	config := compliance.OPAConfig{
		LocalPolicies: tempDir,
		CacheDuration: 5 * time.Minute,
		Timeout:       10 * time.Second,
	}

	engine := compliance.NewOPAEngine(config)
	ctx := context.Background()

	err = engine.LoadPolicies(ctx)
	require.NoError(t, err)

	tests := []struct {
		name               string
		input              compliance.PolicyInput
		expectedAllow      bool
		expectedViolations int
	}{
		{
			name: "allowed action with required tags",
			input: compliance.PolicyInput{
				Resource: map[string]interface{}{
					"type": "s3_bucket",
					"name": "test-bucket",
				},
				Action: "read",
				Tags: map[string]string{
					"Owner": "test-user",
				},
			},
			expectedAllow:      true,
			expectedViolations: 0,
		},
		{
			name: "denied action with violation",
			input: compliance.PolicyInput{
				Resource: map[string]interface{}{
					"type": "s3_bucket",
					"name": "test-bucket",
				},
				Action: "write",
				Tags:   map[string]string{},
			},
			expectedAllow:      false,
			expectedViolations: 1,
		},
		{
			name: "allowed action but missing tags",
			input: compliance.PolicyInput{
				Resource: map[string]interface{}{
					"type": "s3_bucket",
					"name": "test-bucket",
				},
				Action: "read",
				Tags:   map[string]string{},
			},
			expectedAllow:      false,
			expectedViolations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := engine.Evaluate(ctx, "test.policy", tt.input)

			assert.NoError(t, err)
			assert.NotNil(t, decision)
			assert.Equal(t, tt.expectedAllow, decision.Allow)
			assert.Len(t, decision.Violations, tt.expectedViolations)
			assert.NotZero(t, decision.EvaluatedAt)

			if tt.expectedViolations > 0 {
				assert.Equal(t, "required_tags", decision.Violations[0].Rule)
				assert.Equal(t, "medium", decision.Violations[0].Severity)
			}
		})
	}
}

func TestOPAEngine_Cache(t *testing.T) {
	// Create a temporary directory for test policies
	tempDir := t.TempDir()

	// Create a test policy
	policyContent := `package test.policy

default allow = false

allow {
    input.action == "read"
}`

	filePath := filepath.Join(tempDir, "test.rego")
	err := os.WriteFile(filePath, []byte(policyContent), 0644)
	require.NoError(t, err)

	// Setup engine with short cache duration
	config := compliance.OPAConfig{
		LocalPolicies: tempDir,
		CacheDuration: 100 * time.Millisecond,
		Timeout:       10 * time.Second,
	}

	engine := compliance.NewOPAEngine(config)
	ctx := context.Background()

	err = engine.LoadPolicies(ctx)
	require.NoError(t, err)

	input := compliance.PolicyInput{
		Resource: map[string]interface{}{
			"type": "s3_bucket",
			"name": "test-bucket",
		},
		Action: "read",
	}

	// First evaluation
	decision1, err := engine.Evaluate(ctx, "test.policy", input)
	assert.NoError(t, err)
	assert.True(t, decision1.Allow)

	// Second evaluation (should be cached)
	decision2, err := engine.Evaluate(ctx, "test.policy", input)
	assert.NoError(t, err)
	assert.True(t, decision2.Allow)

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third evaluation (cache should be expired)
	decision3, err := engine.Evaluate(ctx, "test.policy", input)
	assert.NoError(t, err)
	assert.True(t, decision3.Allow)

	// Clear cache
	engine.ClearCache()

	// Fourth evaluation (cache should be cleared)
	decision4, err := engine.Evaluate(ctx, "test.policy", input)
	assert.NoError(t, err)
	assert.True(t, decision4.Allow)
}

func TestOPAEngine_UploadPolicy(t *testing.T) {
	// Create a temporary directory for test policies
	tempDir := t.TempDir()

	// Setup engine
	config := compliance.OPAConfig{
		LocalPolicies: tempDir,
		CacheDuration: 5 * time.Minute,
		Timeout:       10 * time.Second,
	}

	engine := compliance.NewOPAEngine(config)
	ctx := context.Background()

	// Create a new policy
	policy := &compliance.Policy{
		ID:          "uploaded-policy",
		Name:        "Uploaded Policy",
		Description: "A policy uploaded via API",
		Package:     "test.uploaded",
		Rules: `package test.uploaded

default allow = false

allow {
    input.action == "read"
}`,
	}

	// Upload policy
	err := engine.UploadPolicy(ctx, policy)
	assert.NoError(t, err)

	// Verify policy was uploaded
	uploadedPolicy, exists := engine.GetPolicy("uploaded-policy")
	assert.True(t, exists)
	assert.Equal(t, policy.ID, uploadedPolicy.ID)
	assert.Equal(t, policy.Name, uploadedPolicy.Name)
	assert.Equal(t, policy.Package, uploadedPolicy.Package)

	// Verify file was created
	filePath := filepath.Join(tempDir, "uploaded-policy.rego")
	_, err = os.Stat(filePath)
	assert.NoError(t, err)
}

func TestOPAEngine_DeletePolicy(t *testing.T) {
	// Create a temporary directory for test policies
	tempDir := t.TempDir()

	// Create a test policy file
	policyContent := `package test.policy

default allow = false

allow {
    input.action == "read"
}`

	filePath := filepath.Join(tempDir, "test.rego")
	err := os.WriteFile(filePath, []byte(policyContent), 0644)
	require.NoError(t, err)

	// Setup engine
	config := compliance.OPAConfig{
		LocalPolicies: tempDir,
		CacheDuration: 5 * time.Minute,
		Timeout:       10 * time.Second,
	}

	engine := compliance.NewOPAEngine(config)
	ctx := context.Background()

	err = engine.LoadPolicies(ctx)
	require.NoError(t, err)

	// Verify policy exists
	_, exists := engine.GetPolicy("test.rego")
	assert.True(t, exists)

	// Delete policy (use just the base name, not the .rego extension)
	err = engine.DeletePolicy(ctx, "test")
	assert.NoError(t, err)

	// Verify policy was deleted
	_, exists = engine.GetPolicy("test.rego")
	assert.False(t, exists)

	// Verify file was deleted
	_, err = os.Stat(filePath)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestOPAEngine_InvalidPolicy(t *testing.T) {
	// Create a temporary directory for test policies
	tempDir := t.TempDir()

	// Create an invalid policy file
	invalidPolicyContent := `package test.policy

default allow = false

allow {
    input.action == "read"
    // Missing closing brace
}`

	filePath := filepath.Join(tempDir, "invalid.rego")
	err := os.WriteFile(filePath, []byte(invalidPolicyContent), 0644)
	require.NoError(t, err)

	// Setup engine
	config := compliance.OPAConfig{
		LocalPolicies: tempDir,
		CacheDuration: 5 * time.Minute,
		Timeout:       10 * time.Second,
	}

	engine := compliance.NewOPAEngine(config)
	ctx := context.Background()

	// Loading should fail due to invalid policy
	err = engine.LoadPolicies(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse policy")
}

func TestOPAEngine_EmptyDirectory(t *testing.T) {
	// Create an empty temporary directory
	tempDir := t.TempDir()

	// Setup engine
	config := compliance.OPAConfig{
		LocalPolicies: tempDir,
		CacheDuration: 5 * time.Minute,
		Timeout:       10 * time.Second,
	}

	engine := compliance.NewOPAEngine(config)
	ctx := context.Background()

	// Loading should succeed with no policies
	err := engine.LoadPolicies(ctx)
	assert.NoError(t, err)

	// No policies should be loaded
	policies := engine.ListPolicies()
	assert.Len(t, policies, 0)
}
