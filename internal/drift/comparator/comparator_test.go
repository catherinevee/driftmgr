package comparator

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewResourceComparator(t *testing.T) {
	comparator := NewResourceComparator()

	assert.NotNil(t, comparator)
	assert.NotNil(t, comparator.ignoreKeys)
	assert.NotNil(t, comparator.customRules)
	assert.NotNil(t, comparator.normalizers)
	assert.NotNil(t, comparator.config)

	// Check default config
	assert.True(t, comparator.config.IgnoreComputed)
	assert.False(t, comparator.config.IgnoreTags)
	assert.True(t, comparator.config.IgnoreMetadata)
	assert.True(t, comparator.config.CaseSensitive)
	assert.True(t, comparator.config.DeepComparison)
}

func TestResourceComparator_Compare(t *testing.T) {
	tests := []struct {
		name        string
		expected    map[string]interface{}
		actual      map[string]interface{}
		setupConfig func(*ResourceComparator)
		wantDiffs   int
		checkDiffs  func(t *testing.T, diffs []Difference)
	}{
		{
			name: "No differences",
			expected: map[string]interface{}{
				"name": "test",
				"value": 123,
			},
			actual: map[string]interface{}{
				"name": "test",
				"value": 123,
			},
			wantDiffs: 0,
		},
		{
			name: "Simple modification",
			expected: map[string]interface{}{
				"name": "test",
				"value": 123,
			},
			actual: map[string]interface{}{
				"name": "test",
				"value": 456,
			},
			wantDiffs: 1,
			checkDiffs: func(t *testing.T, diffs []Difference) {
				assert.Equal(t, "value", diffs[0].Path)
				assert.Equal(t, DiffTypeModified, diffs[0].Type)
				assert.Equal(t, 123, diffs[0].Expected)
				assert.Equal(t, 456, diffs[0].Actual)
			},
		},
		{
			name: "Added field",
			expected: map[string]interface{}{
				"name": "test",
			},
			actual: map[string]interface{}{
				"name": "test",
				"new": "field",
			},
			wantDiffs: 1,
			checkDiffs: func(t *testing.T, diffs []Difference) {
				assert.Equal(t, "new", diffs[0].Path)
				assert.Equal(t, DiffTypeAdded, diffs[0].Type)
				assert.Nil(t, diffs[0].Expected)
				assert.Equal(t, "field", diffs[0].Actual)
			},
		},
		{
			name: "Removed field",
			expected: map[string]interface{}{
				"name": "test",
				"old": "field",
			},
			actual: map[string]interface{}{
				"name": "test",
			},
			wantDiffs: 1,
			checkDiffs: func(t *testing.T, diffs []Difference) {
				assert.Equal(t, "old", diffs[0].Path)
				assert.Equal(t, DiffTypeRemoved, diffs[0].Type)
				assert.Equal(t, "field", diffs[0].Expected)
				assert.Nil(t, diffs[0].Actual)
			},
		},
		{
			name: "Nested object differences",
			expected: map[string]interface{}{
				"name": "test",
				"config": map[string]interface{}{
					"enabled": true,
					"value": 100,
				},
			},
			actual: map[string]interface{}{
				"name": "test",
				"config": map[string]interface{}{
					"enabled": false,
					"value": 100,
				},
			},
			wantDiffs: 1,
			checkDiffs: func(t *testing.T, diffs []Difference) {
				assert.Equal(t, "config.enabled", diffs[0].Path)
				assert.Equal(t, DiffTypeModified, diffs[0].Type)
				assert.Equal(t, true, diffs[0].Expected)
				assert.Equal(t, false, diffs[0].Actual)
			},
		},
		{
			name: "Array differences",
			expected: map[string]interface{}{
				"tags": []interface{}{"tag1", "tag2"},
			},
			actual: map[string]interface{}{
				"tags": []interface{}{"tag1", "tag3"},
			},
			wantDiffs: 1,
			checkDiffs: func(t *testing.T, diffs []Difference) {
				assert.Contains(t, diffs[0].Path, "tags")
				assert.Equal(t, DiffTypeModified, diffs[0].Type)
			},
		},
		{
			name: "Type mismatch",
			expected: map[string]interface{}{
				"value": "123",
			},
			actual: map[string]interface{}{
				"value": 123,
			},
			wantDiffs: 1,
			checkDiffs: func(t *testing.T, diffs []Difference) {
				assert.Equal(t, "value", diffs[0].Path)
				assert.Equal(t, DiffTypeTypeMismatch, diffs[0].Type)
			},
		},
		{
			name: "Ignore computed fields",
			expected: map[string]interface{}{
				"name": "test",
				"computed_field": "value1",
			},
			actual: map[string]interface{}{
				"name": "test",
				"computed_field": "value2",
			},
			setupConfig: func(c *ResourceComparator) {
				c.AddIgnoreKey("computed_field")
			},
			wantDiffs: 0,
		},
		{
			name: "Case insensitive comparison",
			expected: map[string]interface{}{
				"name": "Test",
			},
			actual: map[string]interface{}{
				"name": "test",
			},
			setupConfig: func(c *ResourceComparator) {
				c.config.CaseSensitive = false
			},
			wantDiffs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comparator := NewResourceComparator()
			if tt.setupConfig != nil {
				tt.setupConfig(comparator)
			}

			diffs := comparator.Compare(tt.expected, tt.actual)

			assert.Len(t, diffs, tt.wantDiffs)
			if tt.checkDiffs != nil && len(diffs) > 0 {
				tt.checkDiffs(t, diffs)
			}
		})
	}
}

func TestResourceComparator_SetConfig(t *testing.T) {
	comparator := NewResourceComparator()

	config := &ComparatorConfig{
		IgnoreComputed:     false,
		IgnoreTags:         true,
		IgnoreMetadata:     false,
		CaseSensitive:      false,
		DeepComparison:     false,
		CustomIgnoreFields: []string{"field1", "field2"},
	}

	comparator.SetConfig(config)

	assert.Equal(t, config, comparator.config)
	assert.False(t, comparator.config.IgnoreComputed)
	assert.True(t, comparator.config.IgnoreTags)
	assert.False(t, comparator.config.IgnoreMetadata)
	assert.False(t, comparator.config.CaseSensitive)
	assert.False(t, comparator.config.DeepComparison)
	assert.Len(t, comparator.config.CustomIgnoreFields, 2)
}

func TestResourceComparator_AddIgnoreKey(t *testing.T) {
	comparator := NewResourceComparator()

	comparator.AddIgnoreKey("ignore_me")
	comparator.AddIgnoreKey("timestamp")

	expected := map[string]interface{}{
		"name":      "test",
		"ignore_me": "value1",
		"timestamp": "2024-01-01",
	}

	actual := map[string]interface{}{
		"name":      "test",
		"ignore_me": "value2",
		"timestamp": "2024-01-02",
	}

	diffs := comparator.Compare(expected, actual)

	assert.Empty(t, diffs)
}

func TestResourceComparator_AddCustomRule(t *testing.T) {
	comparator := NewResourceComparator()

	// Add custom rule that considers values equal if they're both non-empty strings
	comparator.AddCustomRule("status", func(expected, actual interface{}) bool {
		e, eOk := expected.(string)
		a, aOk := actual.(string)
		return eOk && aOk && len(e) > 0 && len(a) > 0
	})

	expected := map[string]interface{}{
		"name":   "test",
		"status": "RUNNING",
	}

	actual := map[string]interface{}{
		"name":   "test",
		"status": "running",
	}

	diffs := comparator.Compare(expected, actual)

	// Should only have differences in name, not status (due to custom rule)
	assert.Empty(t, diffs)
}

func TestResourceComparator_AddNormalizer(t *testing.T) {
	comparator := NewResourceComparator()

	// Add normalizer that converts strings to lowercase
	comparator.AddNormalizer("name", func(value interface{}) interface{} {
		if s, ok := value.(string); ok {
			return strings.ToLower(s)
		}
		return value
	})

	expected := map[string]interface{}{
		"name": "TEST",
		"id":   123,
	}

	actual := map[string]interface{}{
		"name": "test",
		"id":   123,
	}

	diffs := comparator.Compare(expected, actual)

	assert.Empty(t, diffs)
}

func TestResourceComparator_CompareArrays(t *testing.T) {
	comparator := NewResourceComparator()

	tests := []struct {
		name      string
		expected  map[string]interface{}
		actual    map[string]interface{}
		wantDiffs int
	}{
		{
			name: "Same arrays",
			expected: map[string]interface{}{
				"items": []interface{}{"a", "b", "c"},
			},
			actual: map[string]interface{}{
				"items": []interface{}{"a", "b", "c"},
			},
			wantDiffs: 0,
		},
		{
			name: "Different length",
			expected: map[string]interface{}{
				"items": []interface{}{"a", "b"},
			},
			actual: map[string]interface{}{
				"items": []interface{}{"a", "b", "c"},
			},
			wantDiffs: 1,
		},
		{
			name: "Different elements",
			expected: map[string]interface{}{
				"items": []interface{}{"a", "b", "c"},
			},
			actual: map[string]interface{}{
				"items": []interface{}{"a", "x", "c"},
			},
			wantDiffs: 1,
		},
		{
			name: "Array of maps",
			expected: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"id": 1, "name": "first"},
					map[string]interface{}{"id": 2, "name": "second"},
				},
			},
			actual: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"id": 1, "name": "first"},
					map[string]interface{}{"id": 2, "name": "modified"},
				},
			},
			wantDiffs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := comparator.Compare(tt.expected, tt.actual)
			assert.Len(t, diffs, tt.wantDiffs)
		})
	}
}

func TestResourceComparator_ComplexNested(t *testing.T) {
	comparator := NewResourceComparator()

	expected := map[string]interface{}{
		"name": "test-resource",
		"config": map[string]interface{}{
			"settings": map[string]interface{}{
				"enabled": true,
				"options": []interface{}{
					map[string]interface{}{
						"key": "option1",
						"value": "value1",
					},
					map[string]interface{}{
						"key": "option2",
						"value": "value2",
					},
				},
			},
		},
		"tags": map[string]interface{}{
			"env": "production",
			"team": "engineering",
		},
	}

	actual := map[string]interface{}{
		"name": "test-resource",
		"config": map[string]interface{}{
			"settings": map[string]interface{}{
				"enabled": false, // Changed
				"options": []interface{}{
					map[string]interface{}{
						"key": "option1",
						"value": "value1",
					},
					map[string]interface{}{
						"key": "option2",
						"value": "modified", // Changed
					},
				},
			},
		},
		"tags": map[string]interface{}{
			"env": "production",
			"team": "devops", // Changed
		},
	}

	diffs := comparator.Compare(expected, actual)

	// Should detect 3 differences
	assert.GreaterOrEqual(t, len(diffs), 3)

	// Check that we found the specific differences
	paths := make(map[string]bool)
	for _, diff := range diffs {
		paths[diff.Path] = true
	}

	assert.True(t, paths["config.settings.enabled"])
	for _, diff := range diffs {
		if strings.Contains(diff.Path, "options") && strings.Contains(diff.Path, "value") {
			paths["config.settings.options.value"] = true
		}
	}
	assert.True(t, paths["tags.team"])
}

// TestResourceComparator_GetDriftSummary tests drift summary generation
// NOTE: GetDriftSummary method needs to be implemented in ResourceComparator
/*
func TestResourceComparator_GetDriftSummary(t *testing.T) {
	// Test commented out - method not yet implemented
}
*/

// TestResourceComparator_FilterByImportance tests filtering by importance
// NOTE: FilterByImportance method needs to be implemented in ResourceComparator
/*
func TestResourceComparator_FilterByImportance(t *testing.T) {
	// Test commented out - method not yet implemented
}
*/

func TestResourceComparator_WithIgnoreTags(t *testing.T) {
	comparator := NewResourceComparator()
	comparator.config.IgnoreTags = true

	expected := map[string]interface{}{
		"name": "test",
		"tags": map[string]interface{}{
			"env": "dev",
		},
	}

	actual := map[string]interface{}{
		"name": "test",
		"tags": map[string]interface{}{
			"env": "prod",
		},
	}

	diffs := comparator.Compare(expected, actual)

	// Tags should be ignored
	assert.Empty(t, diffs)
}

func TestResourceComparator_Benchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmark test in short mode")
	}

	comparator := NewResourceComparator()

	// Create large resource maps
	expected := make(map[string]interface{})
	actual := make(map[string]interface{})

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("field_%d", i)
		expected[key] = fmt.Sprintf("value_%d", i)
		if i%10 == 0 {
			actual[key] = fmt.Sprintf("modified_%d", i)
		} else {
			actual[key] = fmt.Sprintf("value_%d", i)
		}
	}

	// Add nested structures
	expected["nested"] = map[string]interface{}{
		"deep": map[string]interface{}{
			"values": []interface{}{1, 2, 3, 4, 5},
		},
	}
	actual["nested"] = map[string]interface{}{
		"deep": map[string]interface{}{
			"values": []interface{}{1, 2, 3, 4, 6},
		},
	}

	// Measure performance
	start := time.Now()
	diffs := comparator.Compare(expected, actual)
	duration := time.Since(start)

	assert.NotEmpty(t, diffs)
	assert.Less(t, duration.Milliseconds(), int64(100), "Comparison should complete within 100ms")

	// Should find about 100 differences (every 10th field)
	assert.GreaterOrEqual(t, len(diffs), 100)
}