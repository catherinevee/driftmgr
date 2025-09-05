package comparator

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// ResourceComparator compares resource states
type ResourceComparator struct {
	ignoreKeys  map[string]bool
	customRules map[string]CompareFunc
	normalizers map[string]NormalizeFunc
	config      *ComparatorConfig
}

// ComparatorConfig contains comparator configuration
type ComparatorConfig struct {
	IgnoreComputed     bool     `json:"ignore_computed"`
	IgnoreTags         bool     `json:"ignore_tags"`
	IgnoreMetadata     bool     `json:"ignore_metadata"`
	CaseSensitive      bool     `json:"case_sensitive"`
	DeepComparison     bool     `json:"deep_comparison"`
	CustomIgnoreFields []string `json:"custom_ignore_fields"`
}

// CompareFunc is a custom comparison function
type CompareFunc func(expected, actual interface{}) bool

// NormalizeFunc normalizes values before comparison
type NormalizeFunc func(value interface{}) interface{}

// Difference represents a difference between expected and actual values
type Difference struct {
	Path       string      `json:"path"`
	Type       DiffType    `json:"type"`
	Expected   interface{} `json:"expected"`
	Actual     interface{} `json:"actual"`
	Message    string      `json:"message"`
	Importance Importance  `json:"importance"`
}

// DiffType categorizes the type of difference
type DiffType string

const (
	DiffTypeAdded        DiffType = "added"
	DiffTypeRemoved      DiffType = "removed"
	DiffTypeModified     DiffType = "modified"
	DiffTypeTypeMismatch DiffType = "type_mismatch"
)

// Importance indicates the importance of a difference
type Importance int

const (
	ImportanceLow Importance = iota
	ImportanceMedium
	ImportanceHigh
	ImportanceCritical
)

// NewResourceComparator creates a new resource comparator
func NewResourceComparator() *ResourceComparator {
	comparator := &ResourceComparator{
		ignoreKeys:  make(map[string]bool),
		customRules: make(map[string]CompareFunc),
		normalizers: make(map[string]NormalizeFunc),
		config: &ComparatorConfig{
			IgnoreComputed: true,
			IgnoreTags:     false,
			IgnoreMetadata: true,
			CaseSensitive:  true,
			DeepComparison: true,
		},
	}

	// Add default ignore keys
	comparator.addDefaultIgnoreKeys()

	// Add default normalizers
	comparator.addDefaultNormalizers()

	// Add default custom rules
	comparator.addDefaultCustomRules()

	return comparator
}

// Compare compares expected and actual resource states
func (rc *ResourceComparator) Compare(expected, actual map[string]interface{}) []Difference {
	differences := make([]Difference, 0)

	// Normalize inputs
	expected = rc.normalizeMap(expected)
	actual = rc.normalizeMap(actual)

	// Perform comparison
	rc.compareRecursive("", expected, actual, &differences)

	// Sort differences by importance and path
	rc.sortDifferences(differences)

	return differences
}

// compareRecursive performs recursive comparison
func (rc *ResourceComparator) compareRecursive(path string, expected, actual interface{}, diffs *[]Difference) {
	// Check if path should be ignored
	if rc.shouldIgnore(path) {
		return
	}

	// Check for custom comparison rule
	if customRule, exists := rc.customRules[path]; exists {
		if !customRule(expected, actual) {
			*diffs = append(*diffs, Difference{
				Path:       path,
				Type:       DiffTypeModified,
				Expected:   expected,
				Actual:     actual,
				Message:    fmt.Sprintf("Custom rule failed for %s", path),
				Importance: rc.getFieldImportance(path),
			})
		}
		return
	}

	// Normalize values if normalizer exists
	if normalizer, exists := rc.normalizers[path]; exists {
		expected = normalizer(expected)
		actual = normalizer(actual)
	}

	// Handle nil cases
	if expected == nil && actual == nil {
		return
	}
	if expected == nil {
		*diffs = append(*diffs, Difference{
			Path:       path,
			Type:       DiffTypeAdded,
			Expected:   nil,
			Actual:     actual,
			Message:    fmt.Sprintf("Field %s added", path),
			Importance: rc.getFieldImportance(path),
		})
		return
	}
	if actual == nil {
		*diffs = append(*diffs, Difference{
			Path:       path,
			Type:       DiffTypeRemoved,
			Expected:   expected,
			Actual:     nil,
			Message:    fmt.Sprintf("Field %s removed", path),
			Importance: rc.getFieldImportance(path),
		})
		return
	}

	// Type checking
	expectedType := reflect.TypeOf(expected)
	actualType := reflect.TypeOf(actual)

	if expectedType != actualType {
		*diffs = append(*diffs, Difference{
			Path:       path,
			Type:       DiffTypeTypeMismatch,
			Expected:   expected,
			Actual:     actual,
			Message:    fmt.Sprintf("Type mismatch at %s: expected %T, got %T", path, expected, actual),
			Importance: ImportanceHigh,
		})
		return
	}

	// Compare based on type
	switch exp := expected.(type) {
	case map[string]interface{}:
		rc.compareMaps(path, exp, actual.(map[string]interface{}), diffs)
	case []interface{}:
		rc.compareSlices(path, exp, actual.([]interface{}), diffs)
	case string:
		rc.compareStrings(path, exp, actual.(string), diffs)
	case float64:
		rc.compareNumbers(path, exp, actual.(float64), diffs)
	case bool:
		if exp != actual.(bool) {
			*diffs = append(*diffs, Difference{
				Path:       path,
				Type:       DiffTypeModified,
				Expected:   exp,
				Actual:     actual,
				Message:    fmt.Sprintf("Boolean value changed at %s", path),
				Importance: rc.getFieldImportance(path),
			})
		}
	default:
		// Default comparison using DeepEqual
		if !reflect.DeepEqual(expected, actual) {
			*diffs = append(*diffs, Difference{
				Path:       path,
				Type:       DiffTypeModified,
				Expected:   expected,
				Actual:     actual,
				Message:    fmt.Sprintf("Value changed at %s", path),
				Importance: rc.getFieldImportance(path),
			})
		}
	}
}

// compareMaps compares two maps
func (rc *ResourceComparator) compareMaps(path string, expected, actual map[string]interface{}, diffs *[]Difference) {
	// Check for removed keys
	for key, expValue := range expected {
		newPath := rc.joinPath(path, key)
		if actValue, exists := actual[key]; exists {
			rc.compareRecursive(newPath, expValue, actValue, diffs)
		} else {
			*diffs = append(*diffs, Difference{
				Path:       newPath,
				Type:       DiffTypeRemoved,
				Expected:   expValue,
				Actual:     nil,
				Message:    fmt.Sprintf("Key %s removed", newPath),
				Importance: rc.getFieldImportance(newPath),
			})
		}
	}

	// Check for added keys
	for key, actValue := range actual {
		if _, exists := expected[key]; !exists {
			newPath := rc.joinPath(path, key)
			*diffs = append(*diffs, Difference{
				Path:       newPath,
				Type:       DiffTypeAdded,
				Expected:   nil,
				Actual:     actValue,
				Message:    fmt.Sprintf("Key %s added", newPath),
				Importance: rc.getFieldImportance(newPath),
			})
		}
	}
}

// compareSlices compares two slices
func (rc *ResourceComparator) compareSlices(path string, expected, actual []interface{}, diffs *[]Difference) {
	if len(expected) != len(actual) {
		*diffs = append(*diffs, Difference{
			Path:       path,
			Type:       DiffTypeModified,
			Expected:   expected,
			Actual:     actual,
			Message:    fmt.Sprintf("Array length changed at %s: %d -> %d", path, len(expected), len(actual)),
			Importance: rc.getFieldImportance(path),
		})
		return
	}

	// Try to match elements intelligently
	if rc.config.DeepComparison {
		// For deep comparison, compare element by element
		for i := 0; i < len(expected); i++ {
			newPath := fmt.Sprintf("%s[%d]", path, i)
			rc.compareRecursive(newPath, expected[i], actual[i], diffs)
		}
	} else {
		// Simple comparison
		if !reflect.DeepEqual(expected, actual) {
			*diffs = append(*diffs, Difference{
				Path:       path,
				Type:       DiffTypeModified,
				Expected:   expected,
				Actual:     actual,
				Message:    fmt.Sprintf("Array changed at %s", path),
				Importance: rc.getFieldImportance(path),
			})
		}
	}
}

// compareStrings compares two strings
func (rc *ResourceComparator) compareStrings(path string, expected, actual string, diffs *[]Difference) {
	compareExpected := expected
	compareActual := actual

	if !rc.config.CaseSensitive {
		compareExpected = strings.ToLower(expected)
		compareActual = strings.ToLower(actual)
	}

	if compareExpected != compareActual {
		*diffs = append(*diffs, Difference{
			Path:       path,
			Type:       DiffTypeModified,
			Expected:   expected,
			Actual:     actual,
			Message:    fmt.Sprintf("String value changed at %s", path),
			Importance: rc.getFieldImportance(path),
		})
	}
}

// compareNumbers compares two numbers
func (rc *ResourceComparator) compareNumbers(path string, expected, actual float64, diffs *[]Difference) {
	// Allow small floating point differences
	epsilon := 0.0001
	if diff := expected - actual; diff < -epsilon || diff > epsilon {
		*diffs = append(*diffs, Difference{
			Path:       path,
			Type:       DiffTypeModified,
			Expected:   expected,
			Actual:     actual,
			Message:    fmt.Sprintf("Number value changed at %s", path),
			Importance: rc.getFieldImportance(path),
		})
	}
}

// shouldIgnore checks if a path should be ignored
func (rc *ResourceComparator) shouldIgnore(path string) bool {
	// Check exact match
	if rc.ignoreKeys[path] {
		return true
	}

	// Check custom ignore fields
	for _, field := range rc.config.CustomIgnoreFields {
		if strings.Contains(path, field) {
			return true
		}
	}

	// Check patterns
	if rc.config.IgnoreComputed && strings.Contains(path, "computed_") {
		return true
	}

	if rc.config.IgnoreTags && (path == "tags" || strings.HasSuffix(path, ".tags")) {
		return true
	}

	if rc.config.IgnoreMetadata && strings.Contains(path, "metadata") {
		return true
	}

	return false
}

// normalizeMap normalizes a map for comparison
func (rc *ResourceComparator) normalizeMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}

	normalized := make(map[string]interface{})
	for k, v := range m {
		// Skip nil values
		if v == nil {
			continue
		}

		// Recursively normalize nested maps
		if nestedMap, ok := v.(map[string]interface{}); ok {
			normalized[k] = rc.normalizeMap(nestedMap)
		} else {
			normalized[k] = v
		}
	}

	return normalized
}

// addDefaultIgnoreKeys adds default keys to ignore
func (rc *ResourceComparator) addDefaultIgnoreKeys() {
	defaultIgnore := []string{
		"id",
		"arn",
		"self_link",
		"unique_id",
		"created_at",
		"updated_at",
		"etag",
		"last_modified",
		"generation",
		"resource_version",
		"uid",
	}

	for _, key := range defaultIgnore {
		rc.ignoreKeys[key] = true
	}
}

// addDefaultNormalizers adds default value normalizers
func (rc *ResourceComparator) addDefaultNormalizers() {
	// Normalize boolean strings
	rc.normalizers["enabled"] = func(v interface{}) interface{} {
		if str, ok := v.(string); ok {
			return str == "true" || str == "1" || str == "yes"
		}
		return v
	}

	// Normalize empty strings to nil
	rc.normalizers["description"] = func(v interface{}) interface{} {
		if str, ok := v.(string); ok && str == "" {
			return nil
		}
		return v
	}
}

// addDefaultCustomRules adds default custom comparison rules
func (rc *ResourceComparator) addDefaultCustomRules() {
	// Custom rule for IP addresses (handle CIDR notation)
	rc.customRules["cidr_block"] = func(expected, actual interface{}) bool {
		expStr, expOk := expected.(string)
		actStr, actOk := actual.(string)

		if !expOk || !actOk {
			return expected == actual
		}

		// Normalize CIDR notation
		expStr = rc.normalizeCIDR(expStr)
		actStr = rc.normalizeCIDR(actStr)

		return expStr == actStr
	}

	// Custom rule for JSON strings
	rc.customRules["policy"] = func(expected, actual interface{}) bool {
		expStr, expOk := expected.(string)
		actStr, actOk := actual.(string)

		if !expOk || !actOk {
			return expected == actual
		}

		// Compare normalized JSON
		return rc.compareJSON(expStr, actStr)
	}
}

// normalizeCIDR normalizes CIDR notation
func (rc *ResourceComparator) normalizeCIDR(cidr string) string {
	// Simple normalization - in production, use proper IP parsing
	if !strings.Contains(cidr, "/") {
		return cidr + "/32"
	}
	return cidr
}

// compareJSON compares two JSON strings
func (rc *ResourceComparator) compareJSON(json1, json2 string) bool {
	// Remove whitespace for comparison
	normalize := func(s string) string {
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, "\n", "")
		s = strings.ReplaceAll(s, "\t", "")
		return s
	}

	return normalize(json1) == normalize(json2)
}

// getFieldImportance determines the importance of a field
func (rc *ResourceComparator) getFieldImportance(path string) Importance {
	criticalFields := []string{
		"deletion_protection",
		"encryption",
		"kms_key",
		"ssl",
		"https",
	}

	highFields := []string{
		"security_group",
		"subnet",
		"vpc",
		"network",
		"firewall",
		"iam",
		"role",
		"policy",
		"backup",
		"retention",
	}

	mediumFields := []string{
		"instance_type",
		"size",
		"capacity",
		"version",
		"engine",
	}

	pathLower := strings.ToLower(path)

	for _, field := range criticalFields {
		if strings.Contains(pathLower, field) {
			return ImportanceCritical
		}
	}

	for _, field := range highFields {
		if strings.Contains(pathLower, field) {
			return ImportanceHigh
		}
	}

	for _, field := range mediumFields {
		if strings.Contains(pathLower, field) {
			return ImportanceMedium
		}
	}

	return ImportanceLow
}

// sortDifferences sorts differences by importance and path
func (rc *ResourceComparator) sortDifferences(diffs []Difference) {
	sort.Slice(diffs, func(i, j int) bool {
		// Sort by importance first (descending)
		if diffs[i].Importance != diffs[j].Importance {
			return diffs[i].Importance > diffs[j].Importance
		}
		// Then by path (ascending)
		return diffs[i].Path < diffs[j].Path
	})
}

// joinPath joins path components
func (rc *ResourceComparator) joinPath(base, key string) string {
	if base == "" {
		return key
	}

	// Handle array indices
	if strings.HasPrefix(key, "[") {
		return base + key
	}

	return base + "." + key
}

// SetConfig updates the comparator configuration
func (rc *ResourceComparator) SetConfig(config *ComparatorConfig) {
	rc.config = config
}

// AddIgnoreKey adds a key to ignore during comparison
func (rc *ResourceComparator) AddIgnoreKey(key string) {
	rc.ignoreKeys[key] = true
}

// AddCustomRule adds a custom comparison rule
func (rc *ResourceComparator) AddCustomRule(path string, rule CompareFunc) {
	rc.customRules[path] = rule
}

// AddNormalizer adds a value normalizer
func (rc *ResourceComparator) AddNormalizer(path string, normalizer NormalizeFunc) {
	rc.normalizers[path] = normalizer
}
