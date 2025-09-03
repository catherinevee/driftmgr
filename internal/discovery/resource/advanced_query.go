package resource

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// AdvancedQuery provides advanced querying capabilities for discovered resources
type AdvancedQuery struct {
	mu          sync.RWMutex
	resources   []models.Resource
	indexes     map[string]map[string][]int // field -> value -> resource indices
	queryCache  map[string]*QueryResult
	cacheExpiry time.Duration
}

// QueryResult represents the result of a query
type QueryResult struct {
	Query      string
	Resources  []models.Resource
	Count      int
	ExecutionTime time.Duration
	Timestamp  time.Time
	Cached     bool
}

// QueryFilter defines filter criteria
type QueryFilter struct {
	Field    string
	Operator string
	Value    interface{}
}

// QueryOptions defines query options
type QueryOptions struct {
	Filters    []QueryFilter
	SortBy     string
	SortOrder  string // "asc" or "desc"
	Limit      int
	Offset     int
	GroupBy    string
	Aggregate  string // "count", "sum", "avg", "min", "max"
}

// NewAdvancedQuery creates a new advanced query engine
func NewAdvancedQuery() *AdvancedQuery {
	return &AdvancedQuery{
		resources:   make([]models.Resource, 0),
		indexes:     make(map[string]map[string][]int),
		queryCache:  make(map[string]*QueryResult),
		cacheExpiry: 5 * time.Minute,
	}
}

// AddResource adds a resource and updates indexes
func (aq *AdvancedQuery) AddResource(resource models.Resource) {
	aq.mu.Lock()
	defer aq.mu.Unlock()

	index := len(aq.resources)
	aq.resources = append(aq.resources, resource)

	// Update indexes
	aq.updateIndex("provider", resource.Provider, index)
	aq.updateIndex("region", resource.Region, index)
	aq.updateIndex("type", resource.Type, index)
	aq.updateIndex("status", resource.Status, index)
	aq.updateIndex("name", resource.Name, index)
	aq.updateIndex("id", resource.ID, index)

	// Index tags
	if tags, ok := resource.Tags.(map[string]string); ok {
		for key, value := range tags {
			aq.updateIndex(fmt.Sprintf("tag:%s", key), value, index)
		}
	}

	// Clear cache as data has changed
	aq.queryCache = make(map[string]*QueryResult)
}

// Query performs a query with the given options
func (aq *AdvancedQuery) Query(options QueryOptions) *QueryResult {
	startTime := time.Now()
	
	// Generate cache key
	cacheKey := aq.generateCacheKey(options)
	
	// Check cache
	aq.mu.RLock()
	if cached, exists := aq.queryCache[cacheKey]; exists {
		if time.Since(cached.Timestamp) < aq.cacheExpiry {
			aq.mu.RUnlock()
			cached.Cached = true
			return cached
		}
	}
	aq.mu.RUnlock()

	// Execute query
	results := aq.executeQuery(options)
	
	// Cache result
	queryResult := &QueryResult{
		Query:         cacheKey,
		Resources:     results,
		Count:         len(results),
		ExecutionTime: time.Since(startTime),
		Timestamp:     time.Now(),
		Cached:        false,
	}
	
	aq.mu.Lock()
	aq.queryCache[cacheKey] = queryResult
	aq.mu.Unlock()
	
	return queryResult
}

// QueryByJMESPath performs a JMESPath-like query
func (aq *AdvancedQuery) QueryByJMESPath(expression string) *QueryResult {
	startTime := time.Now()
	
	aq.mu.RLock()
	defer aq.mu.RUnlock()
	
	results := make([]models.Resource, 0)
	
	// Simple JMESPath-like implementation
	// Examples: "resources[?provider=='aws']", "resources[?region=='us-east-1' && type=='ec2_instance']"
	
	if strings.HasPrefix(expression, "resources[?") && strings.HasSuffix(expression, "]") {
		condition := expression[11 : len(expression)-1]
		results = aq.evaluateCondition(condition)
	}
	
	return &QueryResult{
		Query:         expression,
		Resources:     results,
		Count:         len(results),
		ExecutionTime: time.Since(startTime),
		Timestamp:     time.Now(),
		Cached:        false,
	}
}

// QueryBySQL performs a SQL-like query
func (aq *AdvancedQuery) QueryBySQL(sql string) *QueryResult {
	startTime := time.Now()
	
	aq.mu.RLock()
	defer aq.mu.RUnlock()
	
	results := make([]models.Resource, 0)
	
	// Simple SQL-like parser
	// Example: "SELECT * FROM resources WHERE provider = 'aws' AND region = 'us-east-1'"
	
	sql = strings.ToLower(sql)
	if strings.Contains(sql, "where") {
		parts := strings.Split(sql, "where")
		if len(parts) == 2 {
			whereClause := strings.TrimSpace(parts[1])
			results = aq.evaluateWhereClause(whereClause)
		}
	} else if strings.Contains(sql, "select * from resources") {
		results = aq.resources
	}
	
	return &QueryResult{
		Query:         sql,
		Resources:     results,
		Count:         len(results),
		ExecutionTime: time.Since(startTime),
		Timestamp:     time.Now(),
		Cached:        false,
	}
}

// FindByTags finds resources with specific tags
func (aq *AdvancedQuery) FindByTags(tags map[string]string) []models.Resource {
	aq.mu.RLock()
	defer aq.mu.RUnlock()
	
	results := make([]models.Resource, 0)
	
	for _, resource := range aq.resources {
		matches := true
		if resourceTags, ok := resource.Tags.(map[string]string); ok {
			for key, value := range tags {
				if resourceValue, exists := resourceTags[key]; !exists || resourceValue != value {
					matches = false
					break
				}
			}
		} else {
			matches = false
		}
		if matches {
			results = append(results, resource)
		}
	}
	
	return results
}

// FindByRegex finds resources matching a regex pattern
func (aq *AdvancedQuery) FindByRegex(field, pattern string) []models.Resource {
	aq.mu.RLock()
	defer aq.mu.RUnlock()
	
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return []models.Resource{}
	}
	
	results := make([]models.Resource, 0)
	
	for _, resource := range aq.resources {
		value := aq.getFieldValue(resource, field)
		if regex.MatchString(value) {
			results = append(results, resource)
		}
	}
	
	return results
}

// GroupBy groups resources by a specific field
func (aq *AdvancedQuery) GroupBy(field string) map[string][]models.Resource {
	aq.mu.RLock()
	defer aq.mu.RUnlock()
	
	groups := make(map[string][]models.Resource)
	
	for _, resource := range aq.resources {
		value := aq.getFieldValue(resource, field)
		groups[value] = append(groups[value], resource)
	}
	
	return groups
}

// Aggregate performs aggregation operations
func (aq *AdvancedQuery) Aggregate(field, operation string) interface{} {
	aq.mu.RLock()
	defer aq.mu.RUnlock()
	
	switch operation {
	case "count":
		return len(aq.resources)
	case "distinct":
		distinct := make(map[string]bool)
		for _, resource := range aq.resources {
			value := aq.getFieldValue(resource, field)
			distinct[value] = true
		}
		return len(distinct)
	case "group_count":
		groups := make(map[string]int)
		for _, resource := range aq.resources {
			value := aq.getFieldValue(resource, field)
			groups[value]++
		}
		return groups
	default:
		return nil
	}
}

// GetStatistics returns query statistics
func (aq *AdvancedQuery) GetStatistics() map[string]interface{} {
	aq.mu.RLock()
	defer aq.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total_resources": len(aq.resources),
		"indexed_fields":  len(aq.indexes),
		"cached_queries":  len(aq.queryCache),
	}
	
	// Calculate cache hit rate
	hits := 0
	total := 0
	for _, result := range aq.queryCache {
		total++
		if result.Cached {
			hits++
		}
	}
	
	if total > 0 {
		stats["cache_hit_rate"] = float64(hits) / float64(total)
	}
	
	return stats
}

// ClearCache clears the query cache
func (aq *AdvancedQuery) ClearCache() {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	aq.queryCache = make(map[string]*QueryResult)
}

// Helper functions

func (aq *AdvancedQuery) updateIndex(field, value string, index int) {
	if aq.indexes[field] == nil {
		aq.indexes[field] = make(map[string][]int)
	}
	aq.indexes[field][value] = append(aq.indexes[field][value], index)
}

func (aq *AdvancedQuery) generateCacheKey(options QueryOptions) string {
	parts := make([]string, 0)
	
	for _, filter := range options.Filters {
		parts = append(parts, fmt.Sprintf("%s%s%v", filter.Field, filter.Operator, filter.Value))
	}
	
	if options.SortBy != "" {
		parts = append(parts, fmt.Sprintf("sort:%s:%s", options.SortBy, options.SortOrder))
	}
	
	if options.Limit > 0 {
		parts = append(parts, fmt.Sprintf("limit:%d", options.Limit))
	}
	
	if options.Offset > 0 {
		parts = append(parts, fmt.Sprintf("offset:%d", options.Offset))
	}
	
	return strings.Join(parts, "|")
}

func (aq *AdvancedQuery) executeQuery(options QueryOptions) []models.Resource {
	aq.mu.RLock()
	defer aq.mu.RUnlock()
	
	// Start with all resources
	results := make([]models.Resource, len(aq.resources))
	copy(results, aq.resources)
	
	// Apply filters
	for _, filter := range options.Filters {
		filtered := make([]models.Resource, 0)
		for _, resource := range results {
			if aq.matchesFilter(resource, filter) {
				filtered = append(filtered, resource)
			}
		}
		results = filtered
	}
	
	// Apply limit and offset
	if options.Offset > 0 && options.Offset < len(results) {
		results = results[options.Offset:]
	}
	
	if options.Limit > 0 && options.Limit < len(results) {
		results = results[:options.Limit]
	}
	
	return results
}

func (aq *AdvancedQuery) matchesFilter(resource models.Resource, filter QueryFilter) bool {
	value := aq.getFieldValue(resource, filter.Field)
	filterValue := fmt.Sprintf("%v", filter.Value)
	
	switch filter.Operator {
	case "=", "==":
		return value == filterValue
	case "!=":
		return value != filterValue
	case "contains":
		return strings.Contains(value, filterValue)
	case "startswith":
		return strings.HasPrefix(value, filterValue)
	case "endswith":
		return strings.HasSuffix(value, filterValue)
	case "regex":
		matched, _ := regexp.MatchString(filterValue, value)
		return matched
	default:
		return false
	}
}

func (aq *AdvancedQuery) getFieldValue(resource models.Resource, field string) string {
	switch strings.ToLower(field) {
	case "provider":
		return resource.Provider
	case "region":
		return resource.Region
	case "type":
		return resource.Type
	case "name":
		return resource.Name
	case "id":
		return resource.ID
	case "status":
		return resource.Status
	default:
		if strings.HasPrefix(field, "tag:") {
			tagKey := field[4:]
			if tags, ok := resource.Tags.(map[string]string); ok {
				if value, exists := tags[tagKey]; exists {
					return value
				}
			}
		}
		return ""
	}
}

func (aq *AdvancedQuery) evaluateCondition(condition string) []models.Resource {
	results := make([]models.Resource, 0)
	
	// Parse simple conditions like "provider=='aws' && region=='us-east-1'"
	parts := strings.Split(condition, "&&")
	
	for _, resource := range aq.resources {
		matches := true
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if !aq.evaluateSimpleCondition(resource, part) {
				matches = false
				break
			}
		}
		if matches {
			results = append(results, resource)
		}
	}
	
	return results
}

func (aq *AdvancedQuery) evaluateSimpleCondition(resource models.Resource, condition string) bool {
	// Parse conditions like "provider=='aws'"
	if strings.Contains(condition, "==") {
		parts := strings.Split(condition, "==")
		if len(parts) == 2 {
			field := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
			return aq.getFieldValue(resource, field) == value
		}
	}
	return false
}

func (aq *AdvancedQuery) evaluateWhereClause(whereClause string) []models.Resource {
	results := make([]models.Resource, 0)
	
	// Parse simple WHERE clauses
	conditions := strings.Split(whereClause, "and")
	
	for _, resource := range aq.resources {
		matches := true
		for _, condition := range conditions {
			condition = strings.TrimSpace(condition)
			if !aq.evaluateSQLCondition(resource, condition) {
				matches = false
				break
			}
		}
		if matches {
			results = append(results, resource)
		}
	}
	
	return results
}

func (aq *AdvancedQuery) evaluateSQLCondition(resource models.Resource, condition string) bool {
	// Parse conditions like "provider = 'aws'"
	parts := strings.Split(condition, "=")
	if len(parts) == 2 {
		field := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
		return strings.ToLower(aq.getFieldValue(resource, field)) == strings.ToLower(value)
	}
	return false
}