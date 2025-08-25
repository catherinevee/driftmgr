package drift

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeepDiff_CompareSimpleTypes(t *testing.T) {
	dd := NewDeepDiff()

	tests := []struct {
		name     string
		actual   interface{}
		expected interface{}
		hasDiff  bool
	}{
		{
			name:     "identical strings",
			actual:   "test",
			expected: "test",
			hasDiff:  false,
		},
		{
			name:     "different strings",
			actual:   "test1",
			expected: "test2",
			hasDiff:  true,
		},
		{
			name:     "identical numbers",
			actual:   42,
			expected: 42,
			hasDiff:  false,
		},
		{
			name:     "different numbers",
			actual:   42,
			expected: 43,
			hasDiff:  true,
		},
		{
			name:     "number vs string",
			actual:   "42",
			expected: 42,
			hasDiff:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := dd.Compare(tt.actual, tt.expected, "")
			assert.Equal(t, tt.hasDiff, len(diffs) > 0)
		})
	}
}

func TestDeepDiff_CompareNestedStructures(t *testing.T) {
	dd := NewDeepDiff()

	tests := []struct {
		name      string
		actual    interface{}
		expected  interface{}
		diffCount int
	}{
		{
			name: "identical nested maps",
			actual: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"value": "test",
					},
				},
			},
			expected: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"value": "test",
					},
				},
			},
			diffCount: 0,
		},
		{
			name: "different nested values",
			actual: map[string]interface{}{
				"config": map[string]interface{}{
					"instance_type": "t2.micro",
					"volume_size":   20,
				},
			},
			expected: map[string]interface{}{
				"config": map[string]interface{}{
					"instance_type": "t2.small",
					"volume_size":   30,
				},
			},
			diffCount: 2,
		},
		{
			name: "missing nested field",
			actual: map[string]interface{}{
				"config": map[string]interface{}{
					"instance_type": "t2.micro",
				},
			},
			expected: map[string]interface{}{
				"config": map[string]interface{}{
					"instance_type": "t2.micro",
					"volume_size":   20,
				},
			},
			diffCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := dd.Compare(tt.actual, tt.expected, "")
			assert.Equal(t, tt.diffCount, len(diffs))
		})
	}
}

func TestDeepDiff_CompareArrays(t *testing.T) {
	dd := NewDeepDiff()

	tests := []struct {
		name           string
		actual         interface{}
		expected       interface{}
		orderSensitive bool
		hasDiff        bool
	}{
		{
			name:           "identical arrays",
			actual:         []string{"a", "b", "c"},
			expected:       []string{"a", "b", "c"},
			orderSensitive: true,
			hasDiff:        false,
		},
		{
			name:           "different order - sensitive",
			actual:         []string{"a", "c", "b"},
			expected:       []string{"a", "b", "c"},
			orderSensitive: true,
			hasDiff:        true,
		},
		{
			name:           "different order - insensitive",
			actual:         []string{"a", "c", "b"},
			expected:       []string{"a", "b", "c"},
			orderSensitive: false,
			hasDiff:        false,
		},
		{
			name:           "missing element",
			actual:         []string{"a", "b"},
			expected:       []string{"a", "b", "c"},
			orderSensitive: false,
			hasDiff:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dd.OrderSensitive = map[string]bool{
				"": tt.orderSensitive,
			}
			diffs := dd.Compare(tt.actual, tt.expected, "")
			assert.Equal(t, tt.hasDiff, len(diffs) > 0)
		})
	}
}

func TestDeepDiff_IgnorePatterns(t *testing.T) {
	dd := NewDeepDiff()
	dd.IgnorePatterns = []string{
		"*.timestamp",
		"*.created_at",
		"metadata.*",
	}

	actual := map[string]interface{}{
		"id":         "resource-1",
		"name":       "test",
		"timestamp":  "2024-01-01T00:00:00Z",
		"created_at": "2024-01-01",
		"metadata": map[string]interface{}{
			"version": "1.0",
			"author":  "user1",
		},
		"config": map[string]interface{}{
			"enabled": true,
		},
	}

	expected := map[string]interface{}{
		"id":         "resource-1",
		"name":       "test-modified",        // This should be detected
		"timestamp":  "2024-01-02T00:00:00Z", // This should be ignored
		"created_at": "2024-01-02",           // This should be ignored
		"metadata": map[string]interface{}{
			"version": "2.0",   // This should be ignored
			"author":  "user2", // This should be ignored
		},
		"config": map[string]interface{}{
			"enabled": false, // This should be detected
		},
	}

	diffs := dd.Compare(actual, expected, "")
	assert.Equal(t, 2, len(diffs)) // Only name and config.enabled should differ

	// Verify the right fields were detected
	fieldPaths := make(map[string]bool)
	for _, diff := range diffs {
		fieldPaths[diff.Path] = true
	}
	assert.True(t, fieldPaths["name"])
	assert.True(t, fieldPaths["config.enabled"])
}

func TestDeepDiff_SemanticComparison(t *testing.T) {
	dd := NewDeepDiff()

	// Add semantic rule for security groups
	dd.SemanticRules = map[string]SemanticCompareFunc{
		"security_group": func(actual, expected interface{}) (bool, []Difference) {
			actualSG := actual.(map[string]interface{})
			expectedSG := expected.(map[string]interface{})

			// Compare ingress rules semantically
			actualRules := actualSG["ingress_rules"].([]interface{})
			expectedRules := expectedSG["ingress_rules"].([]interface{})

			// Check if rules are semantically equivalent
			if len(actualRules) != len(expectedRules) {
				return false, []Difference{{
					Path:     "security_group.ingress_rules",
					Type:     "count",
					Actual:   len(actualRules),
					Expected: len(expectedRules),
				}}
			}

			return true, nil
		},
	}

	tests := []struct {
		name     string
		actual   interface{}
		expected interface{}
		hasDiff  bool
	}{
		{
			name: "semantically equivalent security groups",
			actual: map[string]interface{}{
				"security_group": map[string]interface{}{
					"ingress_rules": []interface{}{
						map[string]interface{}{"port": 80, "protocol": "tcp"},
						map[string]interface{}{"port": 443, "protocol": "tcp"},
					},
				},
			},
			expected: map[string]interface{}{
				"security_group": map[string]interface{}{
					"ingress_rules": []interface{}{
						map[string]interface{}{"port": 443, "protocol": "tcp"},
						map[string]interface{}{"port": 80, "protocol": "tcp"},
					},
				},
			},
			hasDiff: false, // Order doesn't matter semantically
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := dd.Compare(tt.actual, tt.expected, "")
			assert.Equal(t, tt.hasDiff, len(diffs) > 0)
		})
	}
}

func TestDeepDiff_Normalization(t *testing.T) {
	dd := NewDeepDiff()

	// Add normalization for ARNs
	dd.NormalizationMap = map[string]NormalizeFunc{
		"*.arn": func(value interface{}) interface{} {
			// Normalize ARN format
			if str, ok := value.(string); ok {
				// Remove account ID from ARN for comparison
				return "arn:aws:service:region:ACCOUNT:resource"
			}
			return value
		},
		"*.size": func(value interface{}) interface{} {
			// Normalize size units
			if str, ok := value.(string); ok {
				if str == "10GB" || str == "10240MB" {
					return "10GB"
				}
			}
			return value
		},
	}

	actual := map[string]interface{}{
		"resource": map[string]interface{}{
			"arn":  "arn:aws:s3:us-east-1:123456789:bucket/test",
			"size": "10240MB",
		},
	}

	expected := map[string]interface{}{
		"resource": map[string]interface{}{
			"arn":  "arn:aws:s3:us-east-1:987654321:bucket/test",
			"size": "10GB",
		},
	}

	diffs := dd.Compare(actual, expected, "")
	assert.Equal(t, 0, len(diffs)) // Should be normalized to be identical
}

func TestDeepDiff_ComplexRealWorldScenario(t *testing.T) {
	dd := NewDeepDiff()
	dd.IgnorePatterns = []string{
		"*.last_modified",
		"*.etag",
	}

	// Simulate EC2 instance comparison
	actual := map[string]interface{}{
		"instance_id":   "i-1234567890abcdef0",
		"instance_type": "t2.micro",
		"state":         "running",
		"tags": map[string]interface{}{
			"Name":        "web-server",
			"Environment": "production",
		},
		"security_groups": []interface{}{"sg-1", "sg-2"},
		"network_interfaces": []interface{}{
			map[string]interface{}{
				"subnet_id":  "subnet-123",
				"private_ip": "10.0.1.10",
			},
		},
		"last_modified": "2024-01-01T00:00:00Z",
	}

	expected := map[string]interface{}{
		"instance_id":   "i-1234567890abcdef0",
		"instance_type": "t2.small", // Changed
		"state":         "running",
		"tags": map[string]interface{}{
			"Name":        "web-server",
			"Environment": "staging", // Changed
			"Owner":       "devops",  // Added
		},
		"security_groups": []interface{}{"sg-1", "sg-3"}, // sg-2 -> sg-3
		"network_interfaces": []interface{}{
			map[string]interface{}{
				"subnet_id":  "subnet-123",
				"private_ip": "10.0.1.10",
				"public_ip":  "54.1.2.3", // Added
			},
		},
		"last_modified": "2024-01-02T00:00:00Z", // Should be ignored
	}

	diffs := dd.Compare(actual, expected, "")

	// Should detect: instance_type, tags.Environment, tags.Owner (new),
	// security_groups change, network_interfaces.public_ip (new)
	assert.GreaterOrEqual(t, len(diffs), 4)

	// Verify specific changes were detected
	changeMap := make(map[string]bool)
	for _, diff := range diffs {
		changeMap[diff.Path] = true
	}

	assert.True(t, changeMap["instance_type"])
	assert.True(t, changeMap["tags.Environment"])
}

func TestDeepDiff_CircularReferences(t *testing.T) {
	dd := NewDeepDiff()

	// Create circular reference
	actual := make(map[string]interface{})
	actual["self"] = actual
	actual["value"] = "test"

	expected := make(map[string]interface{})
	expected["self"] = expected
	expected["value"] = "test"

	// Should handle circular references without infinite loop
	diffs := dd.Compare(actual, expected, "")
	assert.Equal(t, 0, len(diffs))
}

func TestDeepDiff_NilHandling(t *testing.T) {
	dd := NewDeepDiff()

	tests := []struct {
		name     string
		actual   interface{}
		expected interface{}
		hasDiff  bool
	}{
		{
			name:     "both nil",
			actual:   nil,
			expected: nil,
			hasDiff:  false,
		},
		{
			name:     "actual nil",
			actual:   nil,
			expected: "value",
			hasDiff:  true,
		},
		{
			name:     "expected nil",
			actual:   "value",
			expected: nil,
			hasDiff:  true,
		},
		{
			name: "nil in map",
			actual: map[string]interface{}{
				"field": nil,
			},
			expected: map[string]interface{}{
				"field": "value",
			},
			hasDiff: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := dd.Compare(tt.actual, tt.expected, "")
			assert.Equal(t, tt.hasDiff, len(diffs) > 0)
		})
	}
}
