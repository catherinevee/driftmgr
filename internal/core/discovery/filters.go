package discovery

import (
	"regexp"
	"strings"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// FilterManager manages resource filtering
type FilterManager struct {
	filters []Filter
}

// Filter defines a resource filter interface
type Filter interface {
	Apply(resources []models.Resource) []models.Resource
	Name() string
}

// NewFilterManager creates a new filter manager
func NewFilterManager() *FilterManager {
	return &FilterManager{
		filters: make([]Filter, 0),
	}
}

// ApplyFilters applies all configured filters to resources
func (fm *FilterManager) ApplyFilters(resources []models.Resource, filterConfig map[string]interface{}) []models.Resource {
	filtered := resources

	// Apply tag filters
	if tags, ok := filterConfig["tags"].(map[string]string); ok && len(tags) > 0 {
		filtered = fm.filterByTags(filtered, tags)
	}

	// Apply region filter
	if regions, ok := filterConfig["regions"].([]string); ok && len(regions) > 0 {
		filtered = fm.filterByRegions(filtered, regions)
	}

	// Apply type filter
	if types, ok := filterConfig["types"].([]string); ok && len(types) > 0 {
		filtered = fm.filterByTypes(filtered, types)
	}

	// Apply state filter
	if states, ok := filterConfig["states"].([]string); ok && len(states) > 0 {
		filtered = fm.filterByStates(filtered, states)
	}

	// Apply name pattern filter
	if pattern, ok := filterConfig["name_pattern"].(string); ok && pattern != "" {
		filtered = fm.filterByNamePattern(filtered, pattern)
	}

	// Apply custom filters
	for _, filter := range fm.filters {
		filtered = filter.Apply(filtered)
	}

	return filtered
}

// filterByTags filters resources by tags
func (fm *FilterManager) filterByTags(resources []models.Resource, tags map[string]string) []models.Resource {
	filtered := make([]models.Resource, 0)

	for _, resource := range resources {
		matches := true
		resourceTags := resource.GetTagsAsMap()
		for key, value := range tags {
			if resourceTag, exists := resourceTags[key]; !exists || resourceTag != value {
				matches = false
				break
			}
		}
		if matches {
			filtered = append(filtered, resource)
		}
	}

	return filtered
}

// filterByRegions filters resources by regions
func (fm *FilterManager) filterByRegions(resources []models.Resource, regions []string) []models.Resource {
	regionMap := make(map[string]bool)
	for _, region := range regions {
		regionMap[strings.ToLower(region)] = true
	}

	filtered := make([]models.Resource, 0)
	for _, resource := range resources {
		if regionMap[strings.ToLower(resource.Region)] {
			filtered = append(filtered, resource)
		}
	}

	return filtered
}

// filterByTypes filters resources by types
func (fm *FilterManager) filterByTypes(resources []models.Resource, types []string) []models.Resource {
	typeMap := make(map[string]bool)
	for _, t := range types {
		typeMap[strings.ToLower(t)] = true
	}

	filtered := make([]models.Resource, 0)
	for _, resource := range resources {
		if typeMap[strings.ToLower(resource.Type)] {
			filtered = append(filtered, resource)
		}
	}

	return filtered
}

// filterByStates filters resources by states
func (fm *FilterManager) filterByStates(resources []models.Resource, states []string) []models.Resource {
	stateMap := make(map[string]bool)
	for _, state := range states {
		stateMap[strings.ToLower(state)] = true
	}

	filtered := make([]models.Resource, 0)
	for _, resource := range resources {
		// Handle State as interface{} - could be string or map
		if stateStr, ok := resource.State.(string); ok {
			if stateMap[strings.ToLower(stateStr)] {
				filtered = append(filtered, resource)
			}
		}
	}

	return filtered
}

// filterByNamePattern filters resources by name pattern
func (fm *FilterManager) filterByNamePattern(resources []models.Resource, pattern string) []models.Resource {
	re, err := regexp.Compile(pattern)
	if err != nil {
		// If pattern is invalid, return all resources
		return resources
	}

	filtered := make([]models.Resource, 0)
	for _, resource := range resources {
		if re.MatchString(resource.Name) {
			filtered = append(filtered, resource)
		}
	}

	return filtered
}

// AddFilter adds a custom filter
func (fm *FilterManager) AddFilter(filter Filter) {
	fm.filters = append(fm.filters, filter)
}

// RemoveFilter removes a filter by name
func (fm *FilterManager) RemoveFilter(name string) {
	newFilters := make([]Filter, 0)
	for _, filter := range fm.filters {
		if filter.Name() != name {
			newFilters = append(newFilters, filter)
		}
	}
	fm.filters = newFilters
}

// ClearFilters removes all custom filters
func (fm *FilterManager) ClearFilters() {
	fm.filters = make([]Filter, 0)
}

// Built-in filter implementations

// TagFilter filters resources by tags
type TagFilter struct {
	tags map[string]string
}

// NewTagFilter creates a new tag filter
func NewTagFilter(tags map[string]string) *TagFilter {
	return &TagFilter{tags: tags}
}

// Apply applies the tag filter
func (f *TagFilter) Apply(resources []models.Resource) []models.Resource {
	filtered := make([]models.Resource, 0)

	for _, resource := range resources {
		matches := true
		resourceTags := resource.GetTagsAsMap()
		for key, value := range f.tags {
			if resourceTag, exists := resourceTags[key]; !exists || resourceTag != value {
				matches = false
				break
			}
		}
		if matches {
			filtered = append(filtered, resource)
		}
	}

	return filtered
}

// Name returns the filter name
func (f *TagFilter) Name() string {
	return "tag_filter"
}

// RegionFilter filters resources by region
type RegionFilter struct {
	regions map[string]bool
}

// NewRegionFilter creates a new region filter
func NewRegionFilter(regions []string) *RegionFilter {
	regionMap := make(map[string]bool)
	for _, region := range regions {
		regionMap[strings.ToLower(region)] = true
	}
	return &RegionFilter{regions: regionMap}
}

// Apply applies the region filter
func (f *RegionFilter) Apply(resources []models.Resource) []models.Resource {
	filtered := make([]models.Resource, 0)
	for _, resource := range resources {
		if f.regions[strings.ToLower(resource.Region)] {
			filtered = append(filtered, resource)
		}
	}
	return filtered
}

// Name returns the filter name
func (f *RegionFilter) Name() string {
	return "region_filter"
}

// TypeFilter filters resources by type
type TypeFilter struct {
	types map[string]bool
}

// NewTypeFilter creates a new type filter
func NewTypeFilter(types []string) *TypeFilter {
	typeMap := make(map[string]bool)
	for _, t := range types {
		typeMap[strings.ToLower(t)] = true
	}
	return &TypeFilter{types: typeMap}
}

// Apply applies the type filter
func (f *TypeFilter) Apply(resources []models.Resource) []models.Resource {
	filtered := make([]models.Resource, 0)
	for _, resource := range resources {
		if f.types[strings.ToLower(resource.Type)] {
			filtered = append(filtered, resource)
		}
	}
	return filtered
}

// Name returns the filter name
func (f *TypeFilter) Name() string {
	return "type_filter"
}

// ExcludeFilter excludes resources matching certain criteria
type ExcludeFilter struct {
	excludeTags    map[string]string
	excludeTypes   map[string]bool
	excludeRegions map[string]bool
}

// NewExcludeFilter creates a new exclude filter
func NewExcludeFilter() *ExcludeFilter {
	return &ExcludeFilter{
		excludeTags:    make(map[string]string),
		excludeTypes:   make(map[string]bool),
		excludeRegions: make(map[string]bool),
	}
}

// AddExcludeTag adds a tag to exclude
func (f *ExcludeFilter) AddExcludeTag(key, value string) {
	f.excludeTags[key] = value
}

// AddExcludeType adds a type to exclude
func (f *ExcludeFilter) AddExcludeType(resourceType string) {
	f.excludeTypes[strings.ToLower(resourceType)] = true
}

// AddExcludeRegion adds a region to exclude
func (f *ExcludeFilter) AddExcludeRegion(region string) {
	f.excludeRegions[strings.ToLower(region)] = true
}

// Apply applies the exclude filter
func (f *ExcludeFilter) Apply(resources []models.Resource) []models.Resource {
	filtered := make([]models.Resource, 0)

	for _, resource := range resources {
		// Check if resource should be excluded
		exclude := false

		// Check tags
		resourceTags := resource.GetTagsAsMap()
		for key, value := range f.excludeTags {
			if resourceTag, exists := resourceTags[key]; exists && resourceTag == value {
				exclude = true
				break
			}
		}

		// Check type
		if !exclude && f.excludeTypes[strings.ToLower(resource.Type)] {
			exclude = true
		}

		// Check region
		if !exclude && f.excludeRegions[strings.ToLower(resource.Region)] {
			exclude = true
		}

		if !exclude {
			filtered = append(filtered, resource)
		}
	}

	return filtered
}

// Name returns the filter name
func (f *ExcludeFilter) Name() string {
	return "exclude_filter"
}
