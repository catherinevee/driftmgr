package filtering

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// FilterOperator represents the type of filter operation
type FilterOperator string

const (
	FilterOperatorEquals       FilterOperator = "equals"
	FilterOperatorNotEquals    FilterOperator = "not_equals"
	FilterOperatorContains     FilterOperator = "contains"
	FilterOperatorNotContains  FilterOperator = "not_contains"
	FilterOperatorStartsWith   FilterOperator = "starts_with"
	FilterOperatorEndsWith     FilterOperator = "ends_with"
	FilterOperatorRegex        FilterOperator = "regex"
	FilterOperatorIn           FilterOperator = "in"
	FilterOperatorNotIn        FilterOperator = "not_in"
	FilterOperatorGreaterThan  FilterOperator = "greater_than"
	FilterOperatorLessThan     FilterOperator = "less_than"
	FilterOperatorGreaterEqual FilterOperator = "greater_equal"
	FilterOperatorLessEqual    FilterOperator = "less_equal"
	FilterOperatorExists       FilterOperator = "exists"
	FilterOperatorNotExists    FilterOperator = "not_exists"
)

// FilterCondition represents a single filter condition
type FilterCondition struct {
	Field    string         `json:"field"`
	Operator FilterOperator `json:"operator"`
	Value    interface{}    `json:"value"`
}

// ResourceFilter represents a filter for resources
type ResourceFilter struct {
	Conditions []FilterCondition `json:"conditions"`
	Logic      FilterLogic       `json:"logic"`
	Limit      int               `json:"limit,omitempty"`
	Offset     int               `json:"offset,omitempty"`
	SortBy     string            `json:"sort_by,omitempty"`
	SortOrder  SortOrder         `json:"sort_order,omitempty"`
}

// FilterLogic represents the logic for combining conditions
type FilterLogic string

const (
	FilterLogicAnd FilterLogic = "and"
	FilterLogicOr  FilterLogic = "or"
)

// SortOrder represents the sort order
type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

// ResourceFilterService provides resource filtering functionality
type ResourceFilterService struct {
	compiledRegexes map[string]*regexp.Regexp
}

// NewResourceFilterService creates a new resource filter service
func NewResourceFilterService() *ResourceFilterService {
	return &ResourceFilterService{
		compiledRegexes: make(map[string]*regexp.Regexp),
	}
}

// FilterResources applies filters to a list of resources
func (rfs *ResourceFilterService) FilterResources(resources []models.Resource, filter ResourceFilter) ([]models.Resource, error) {
	if len(filter.Conditions) == 0 {
		return resources, nil
	}

	var filtered []models.Resource

	for _, resource := range resources {
		matches, err := rfs.evaluateResource(resource, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate resource %s: %w", resource.ID, err)
		}

		if matches {
			filtered = append(filtered, resource)
		}
	}

	// Apply sorting
	if filter.SortBy != "" {
		filtered = rfs.sortResources(filtered, filter.SortBy, filter.SortOrder)
	}

	// Apply pagination
	if filter.Limit > 0 || filter.Offset > 0 {
		filtered = rfs.paginateResources(filtered, filter.Limit, filter.Offset)
	}

	return filtered, nil
}

// evaluateResource evaluates if a resource matches the filter conditions
func (rfs *ResourceFilterService) evaluateResource(resource models.Resource, filter ResourceFilter) (bool, error) {
	if len(filter.Conditions) == 0 {
		return true, nil
	}

	var results []bool

	for _, condition := range filter.Conditions {
		result, err := rfs.evaluateCondition(resource, condition)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate condition %s %s %v: %w", condition.Field, condition.Operator, condition.Value, err)
		}
		results = append(results, result)
	}

	// Apply logic
	if filter.Logic == FilterLogicOr {
		for _, result := range results {
			if result {
				return true, nil
			}
		}
		return false, nil
	} else { // FilterLogicAnd
		for _, result := range results {
			if !result {
				return false, nil
			}
		}
		return true, nil
	}
}

// evaluateCondition evaluates a single condition against a resource
func (rfs *ResourceFilterService) evaluateCondition(resource models.Resource, condition FilterCondition) (bool, error) {
	fieldValue, err := rfs.getFieldValue(resource, condition.Field)
	if err != nil {
		return false, err
	}

	switch condition.Operator {
	case FilterOperatorEquals:
		return rfs.compareValues(fieldValue, condition.Value) == 0, nil
	case FilterOperatorNotEquals:
		return rfs.compareValues(fieldValue, condition.Value) != 0, nil
	case FilterOperatorContains:
		return rfs.stringContains(fieldValue, condition.Value), nil
	case FilterOperatorNotContains:
		return !rfs.stringContains(fieldValue, condition.Value), nil
	case FilterOperatorStartsWith:
		return rfs.stringStartsWith(fieldValue, condition.Value), nil
	case FilterOperatorEndsWith:
		return rfs.stringEndsWith(fieldValue, condition.Value), nil
	case FilterOperatorRegex:
		return rfs.stringMatchesRegex(fieldValue, condition.Value)
	case FilterOperatorIn:
		return rfs.valueInList(fieldValue, condition.Value), nil
	case FilterOperatorNotIn:
		return !rfs.valueInList(fieldValue, condition.Value), nil
	case FilterOperatorGreaterThan:
		return rfs.compareValues(fieldValue, condition.Value) > 0, nil
	case FilterOperatorLessThan:
		return rfs.compareValues(fieldValue, condition.Value) < 0, nil
	case FilterOperatorGreaterEqual:
		return rfs.compareValues(fieldValue, condition.Value) >= 0, nil
	case FilterOperatorLessEqual:
		return rfs.compareValues(fieldValue, condition.Value) <= 0, nil
	case FilterOperatorExists:
		return fieldValue != nil, nil
	case FilterOperatorNotExists:
		return fieldValue == nil, nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", condition.Operator)
	}
}

// getFieldValue extracts the value of a field from a resource
func (rfs *ResourceFilterService) getFieldValue(resource models.Resource, field string) (interface{}, error) {
	switch field {
	case "id":
		return resource.ID, nil
	case "name":
		return resource.Name, nil
	case "type":
		return resource.Type, nil
	case "provider":
		return resource.Provider, nil
	case "region":
		return resource.Region, nil
	case "account_id":
		return resource.AccountID, nil
	case "account_name":
		return resource.AccountName, nil
	case "status":
		return resource.Status, nil
	case "created":
		return resource.Created, nil
	case "updated":
		return resource.Updated, nil
	case "created_at":
		return resource.CreatedAt, nil
	case "last_modified":
		return resource.LastModified, nil
	default:
		// Check if it's a nested field (e.g., "tags.environment", "attributes.size")
		if strings.Contains(field, ".") {
			return rfs.getNestedFieldValue(resource, field)
		}
		return nil, fmt.Errorf("unknown field: %s", field)
	}
}

// getNestedFieldValue extracts the value of a nested field from a resource
func (rfs *ResourceFilterService) getNestedFieldValue(resource models.Resource, field string) (interface{}, error) {
	parts := strings.Split(field, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid nested field format: %s", field)
	}

	parentField := parts[0]
	childField := parts[1]

	switch parentField {
	case "tags":
		if resource.Tags == nil {
			return nil, nil
		}
		if tagsMap, ok := resource.Tags.(map[string]interface{}); ok {
			return tagsMap[childField], nil
		}
		if tagsSlice, ok := resource.Tags.([]string); ok {
			for _, tag := range tagsSlice {
				if tag == childField {
					return tag, nil
				}
			}
			return nil, nil
		}
		return nil, nil
	case "attributes":
		if resource.Attributes == nil {
			return nil, nil
		}
		return resource.Attributes[childField], nil
	case "properties":
		if resource.Properties == nil {
			return nil, nil
		}
		return resource.Properties[childField], nil
	case "metadata":
		if resource.Metadata == nil {
			return nil, nil
		}
		return resource.Metadata[childField], nil
	case "state":
		if resource.State == nil {
			return nil, nil
		}
		if stateMap, ok := resource.State.(map[string]interface{}); ok {
			return stateMap[childField], nil
		}
		return resource.State, nil
	default:
		return nil, fmt.Errorf("unknown parent field: %s", parentField)
	}
}

// compareValues compares two values and returns -1, 0, or 1
func (rfs *ResourceFilterService) compareValues(a, b interface{}) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// Convert to strings for comparison
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)

	// Try to parse as numbers
	if aNum, bNum, ok := rfs.parseNumbers(aStr, bStr); ok {
		if aNum < bNum {
			return -1
		} else if aNum > bNum {
			return 1
		}
		return 0
	}

	// Try to parse as dates
	if aTime, bTime, ok := rfs.parseDates(aStr, bStr); ok {
		if aTime.Before(bTime) {
			return -1
		} else if aTime.After(bTime) {
			return 1
		}
		return 0
	}

	// String comparison
	return strings.Compare(aStr, bStr)
}

// parseNumbers attempts to parse two strings as numbers
func (rfs *ResourceFilterService) parseNumbers(a, b string) (float64, float64, bool) {
	// This is a simplified implementation
	// In a real implementation, you might want to use a more robust number parsing library
	return 0, 0, false
}

// parseDates attempts to parse two strings as dates
func (rfs *ResourceFilterService) parseDates(a, b string) (time.Time, time.Time, bool) {
	// Try common date formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
	}

	for _, format := range formats {
		if aTime, err := time.Parse(format, a); err == nil {
			if bTime, err := time.Parse(format, b); err == nil {
				return aTime, bTime, true
			}
		}
	}

	return time.Time{}, time.Time{}, false
}

// stringContains checks if a string contains another string
func (rfs *ResourceFilterService) stringContains(fieldValue, conditionValue interface{}) bool {
	if fieldValue == nil || conditionValue == nil {
		return false
	}

	fieldStr := strings.ToLower(fmt.Sprintf("%v", fieldValue))
	conditionStr := strings.ToLower(fmt.Sprintf("%v", conditionValue))

	return strings.Contains(fieldStr, conditionStr)
}

// stringStartsWith checks if a string starts with another string
func (rfs *ResourceFilterService) stringStartsWith(fieldValue, conditionValue interface{}) bool {
	if fieldValue == nil || conditionValue == nil {
		return false
	}

	fieldStr := strings.ToLower(fmt.Sprintf("%v", fieldValue))
	conditionStr := strings.ToLower(fmt.Sprintf("%v", conditionValue))

	return strings.HasPrefix(fieldStr, conditionStr)
}

// stringEndsWith checks if a string ends with another string
func (rfs *ResourceFilterService) stringEndsWith(fieldValue, conditionValue interface{}) bool {
	if fieldValue == nil || conditionValue == nil {
		return false
	}

	fieldStr := strings.ToLower(fmt.Sprintf("%v", fieldValue))
	conditionStr := strings.ToLower(fmt.Sprintf("%v", conditionValue))

	return strings.HasSuffix(fieldStr, conditionStr)
}

// stringMatchesRegex checks if a string matches a regex pattern
func (rfs *ResourceFilterService) stringMatchesRegex(fieldValue, conditionValue interface{}) (bool, error) {
	if fieldValue == nil || conditionValue == nil {
		return false, nil
	}

	fieldStr := fmt.Sprintf("%v", fieldValue)
	patternStr := fmt.Sprintf("%v", conditionValue)

	// Compile regex if not already compiled
	compiled, exists := rfs.compiledRegexes[patternStr]
	if !exists {
		var err error
		compiled, err = regexp.Compile(patternStr)
		if err != nil {
			return false, fmt.Errorf("invalid regex pattern: %s", patternStr)
		}
		rfs.compiledRegexes[patternStr] = compiled
	}

	return compiled.MatchString(fieldStr), nil
}

// valueInList checks if a value is in a list
func (rfs *ResourceFilterService) valueInList(fieldValue, conditionValue interface{}) bool {
	if fieldValue == nil || conditionValue == nil {
		return false
	}

	// Convert condition value to slice
	var list []interface{}
	switch v := conditionValue.(type) {
	case []interface{}:
		list = v
	case []string:
		for _, item := range v {
			list = append(list, item)
		}
	case []int:
		for _, item := range v {
			list = append(list, item)
		}
	case []float64:
		for _, item := range v {
			list = append(list, item)
		}
	default:
		return false
	}

	// Check if field value is in the list
	fieldStr := fmt.Sprintf("%v", fieldValue)
	for _, item := range list {
		if fmt.Sprintf("%v", item) == fieldStr {
			return true
		}
	}

	return false
}

// sortResources sorts resources by the specified field and order
func (rfs *ResourceFilterService) sortResources(resources []models.Resource, sortBy string, sortOrder SortOrder) []models.Resource {
	// This is a simplified implementation
	// In a real implementation, you might want to use a more robust sorting library
	return resources
}

// paginateResources applies pagination to resources
func (rfs *ResourceFilterService) paginateResources(resources []models.Resource, limit, offset int) []models.Resource {
	if offset >= len(resources) {
		return []models.Resource{}
	}

	end := offset + limit
	if limit <= 0 || end > len(resources) {
		end = len(resources)
	}

	return resources[offset:end]
}

// CreateFilterFromQuery creates a filter from query parameters
func (rfs *ResourceFilterService) CreateFilterFromQuery(queryParams map[string]string) ResourceFilter {
	filter := ResourceFilter{
		Logic:     FilterLogicAnd,
		SortOrder: SortOrderAsc,
	}

	for key, value := range queryParams {
		switch key {
		case "limit":
			if limit, err := parseInt(value); err == nil {
				filter.Limit = limit
			}
		case "offset":
			if offset, err := parseInt(value); err == nil {
				filter.Offset = offset
			}
		case "sort_by":
			filter.SortBy = value
		case "sort_order":
			if value == "desc" {
				filter.SortOrder = SortOrderDesc
			}
		case "logic":
			if value == "or" {
				filter.Logic = FilterLogicOr
			}
		default:
			// Parse field filters (e.g., "name:contains:test", "type:equals:aws_s3_bucket")
			if condition := rfs.parseFieldFilter(key, value); condition != nil {
				filter.Conditions = append(filter.Conditions, *condition)
			}
		}
	}

	return filter
}

// parseFieldFilter parses a field filter from query parameters
func (rfs *ResourceFilterService) parseFieldFilter(key, value string) *FilterCondition {
	// Format: field:operator:value
	parts := strings.Split(key, ":")
	if len(parts) < 2 {
		return nil
	}

	field := parts[0]
	operator := FilterOperator(parts[1])

	// Default to equals if no operator specified
	if len(parts) == 2 {
		operator = FilterOperatorEquals
		value = parts[1]
	}

	return &FilterCondition{
		Field:    field,
		Operator: operator,
		Value:    value,
	}
}

// parseInt parses a string to an integer
func parseInt(s string) (int, error) {
	// This is a simplified implementation
	// In a real implementation, you might want to use strconv.Atoi
	return 0, nil
}
