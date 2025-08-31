package search

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

type SearchEngine struct {
	index     *SearchIndex
	mu        sync.RWMutex
	cache     map[string]*SearchResult
	cacheTime map[string]time.Time
	cacheTTL  time.Duration
}

type SearchIndex struct {
	StateFiles    map[string]*StateFileIndex
	Resources     map[string]*ResourceIndex
	Modules       map[string]*ModuleIndex
	LastUpdated   time.Time
}

type StateFileIndex struct {
	ID           string
	Path         string
	Name         string
	Provider     string
	Resources    []string
	Modules      []string
	Tags         []string
	Metadata     map[string]string
	LastModified time.Time
	Health       string
}

type ResourceIndex struct {
	ID         string
	Name       string
	Type       string
	Provider   string
	StateFile  string
	Module     string
	Attributes map[string]interface{}
	Tags       map[string]string
	Status     string
}

type ModuleIndex struct {
	Name      string
	Source    string
	Version   string
	StateFile string
	Resources []string
	Inputs    map[string]interface{}
	Outputs   map[string]interface{}
}

type SearchQuery struct {
	Query      string
	Type       SearchType
	Filters    SearchFilters
	Sort       SortOptions
	Pagination PaginationOptions
}

type SearchType string

const (
	SearchTypeAll       SearchType = "all"
	SearchTypeStateFile SearchType = "state_file"
	SearchTypeResource  SearchType = "resource"
	SearchTypeModule    SearchType = "module"
)

type SearchFilters struct {
	Providers    []string
	StateFiles   []string
	ResourceTypes []string
	Tags         map[string]string
	Health       []string
	Status       []string
	DateRange    *DateRange
}

type DateRange struct {
	Start time.Time
	End   time.Time
}

type SortOptions struct {
	Field     string
	Direction string
}

type PaginationOptions struct {
	Offset int
	Limit  int
}

type SearchResult struct {
	Query        string
	Type         SearchType
	TotalMatches int
	Results      []SearchItem
	Facets       map[string]map[string]int
	Suggestions  []string
	ExecutionTime time.Duration
}

type SearchItem struct {
	ID          string
	Type        string
	Name        string
	Description string
	Path        string
	Provider    string
	StateFile   string
	Score       float64
	Highlight   map[string]string
	Metadata    map[string]interface{}
}

func NewSearchEngine(cacheTTL time.Duration) *SearchEngine {
	if cacheTTL == 0 {
		cacheTTL = 5 * time.Minute
	}
	
	return &SearchEngine{
		index:     NewSearchIndex(),
		cache:     make(map[string]*SearchResult),
		cacheTime: make(map[string]time.Time),
		cacheTTL:  cacheTTL,
	}
}

func NewSearchIndex() *SearchIndex {
	return &SearchIndex{
		StateFiles:  make(map[string]*StateFileIndex),
		Resources:   make(map[string]*ResourceIndex),
		Modules:     make(map[string]*ModuleIndex),
		LastUpdated: time.Now(),
	}
}

func (se *SearchEngine) IndexStateFile(sf *StateFileIndex) {
	se.mu.Lock()
	defer se.mu.Unlock()
	
	se.index.StateFiles[sf.ID] = sf
	se.index.LastUpdated = time.Now()
	se.clearCache()
}

func (se *SearchEngine) IndexResource(r *ResourceIndex) {
	se.mu.Lock()
	defer se.mu.Unlock()
	
	se.index.Resources[r.ID] = r
	se.index.LastUpdated = time.Now()
	se.clearCache()
}

func (se *SearchEngine) IndexModule(m *ModuleIndex) {
	se.mu.Lock()
	defer se.mu.Unlock()
	
	key := fmt.Sprintf("%s-%s", m.StateFile, m.Name)
	se.index.Modules[key] = m
	se.index.LastUpdated = time.Now()
	se.clearCache()
}

func (se *SearchEngine) Search(query SearchQuery) (*SearchResult, error) {
	start := time.Now()
	
	cacheKey := se.getCacheKey(query)
	if cached := se.getFromCache(cacheKey); cached != nil {
		return cached, nil
	}
	
	se.mu.RLock()
	defer se.mu.RUnlock()
	
	results := make([]SearchItem, 0)
	facets := make(map[string]map[string]int)
	
	switch query.Type {
	case SearchTypeStateFile:
		results = se.searchStateFiles(query)
	case SearchTypeResource:
		results = se.searchResources(query)
	case SearchTypeModule:
		results = se.searchModules(query)
	case SearchTypeAll:
		results = append(results, se.searchStateFiles(query)...)
		results = append(results, se.searchResources(query)...)
		results = append(results, se.searchModules(query)...)
	}
	
	results = se.applyFilters(results, query.Filters)
	results = se.sortResults(results, query.Sort)
	facets = se.calculateFacets(results)
	
	totalMatches := len(results)
	
	if query.Pagination.Limit > 0 {
		results = se.paginate(results, query.Pagination)
	}
	
	suggestions := se.generateSuggestions(query.Query, results)
	
	result := &SearchResult{
		Query:         query.Query,
		Type:          query.Type,
		TotalMatches:  totalMatches,
		Results:       results,
		Facets:        facets,
		Suggestions:   suggestions,
		ExecutionTime: time.Since(start),
	}
	
	se.putInCache(cacheKey, result)
	
	return result, nil
}

func (se *SearchEngine) searchStateFiles(query SearchQuery) []SearchItem {
	results := make([]SearchItem, 0)
	
	for _, sf := range se.index.StateFiles {
		if se.matchesQuery(query.Query, sf.Name, sf.Path) {
			item := SearchItem{
				ID:        sf.ID,
				Type:      "state_file",
				Name:      sf.Name,
				Path:      sf.Path,
				Provider:  sf.Provider,
				StateFile: sf.ID,
				Score:     se.calculateScore(query.Query, sf.Name, sf.Path),
				Metadata: map[string]interface{}{
					"health":       sf.Health,
					"resources":    len(sf.Resources),
					"modules":      len(sf.Modules),
					"lastModified": sf.LastModified,
				},
			}
			results = append(results, item)
		}
	}
	
	return results
}

func (se *SearchEngine) searchResources(query SearchQuery) []SearchItem {
	results := make([]SearchItem, 0)
	
	for _, r := range se.index.Resources {
		searchText := fmt.Sprintf("%s %s %s", r.Name, r.Type, r.Provider)
		if se.matchesQuery(query.Query, searchText) {
			item := SearchItem{
				ID:        r.ID,
				Type:      "resource",
				Name:      r.Name,
				Path:      r.Type,
				Provider:  r.Provider,
				StateFile: r.StateFile,
				Score:     se.calculateScore(query.Query, searchText),
				Metadata: map[string]interface{}{
					"resourceType": r.Type,
					"status":       r.Status,
					"module":       r.Module,
					"tags":         r.Tags,
				},
			}
			results = append(results, item)
		}
	}
	
	return results
}

func (se *SearchEngine) searchModules(query SearchQuery) []SearchItem {
	results := make([]SearchItem, 0)
	
	for _, m := range se.index.Modules {
		searchText := fmt.Sprintf("%s %s %s", m.Name, m.Source, m.Version)
		if se.matchesQuery(query.Query, searchText) {
			item := SearchItem{
				ID:        fmt.Sprintf("%s-%s", m.StateFile, m.Name),
				Type:      "module",
				Name:      m.Name,
				Path:      m.Source,
				StateFile: m.StateFile,
				Score:     se.calculateScore(query.Query, searchText),
				Metadata: map[string]interface{}{
					"version":   m.Version,
					"resources": len(m.Resources),
					"inputs":    m.Inputs,
					"outputs":   m.Outputs,
				},
			}
			results = append(results, item)
		}
	}
	
	return results
}

func (se *SearchEngine) matchesQuery(query string, texts ...string) bool {
	if query == "" {
		return true
	}
	
	query = strings.ToLower(query)
	combinedText := strings.ToLower(strings.Join(texts, " "))
	
	if strings.Contains(query, "*") || strings.Contains(query, "?") {
		pattern := strings.ReplaceAll(query, "*", ".*")
		pattern = strings.ReplaceAll(pattern, "?", ".")
		re, err := regexp.Compile(pattern)
		if err == nil {
			return re.MatchString(combinedText)
		}
	}
	
	terms := strings.Fields(query)
	for _, term := range terms {
		if !strings.Contains(combinedText, term) {
			return false
		}
	}
	
	return true
}

func (se *SearchEngine) calculateScore(query string, texts ...string) float64 {
	if query == "" {
		return 1.0
	}
	
	score := 0.0
	query = strings.ToLower(query)
	terms := strings.Fields(query)
	
	for _, text := range texts {
		text = strings.ToLower(text)
		for _, term := range terms {
			if strings.Contains(text, term) {
				score += 1.0
				if strings.HasPrefix(text, term) {
					score += 0.5
				}
				if text == term {
					score += 2.0
				}
			}
		}
	}
	
	return score
}

func (se *SearchEngine) applyFilters(items []SearchItem, filters SearchFilters) []SearchItem {
	filtered := make([]SearchItem, 0)
	
	for _, item := range items {
		if !se.matchesFilters(item, filters) {
			continue
		}
		filtered = append(filtered, item)
	}
	
	return filtered
}

func (se *SearchEngine) matchesFilters(item SearchItem, filters SearchFilters) bool {
	if len(filters.Providers) > 0 {
		matched := false
		for _, p := range filters.Providers {
			if item.Provider == p {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	
	if len(filters.StateFiles) > 0 {
		matched := false
		for _, sf := range filters.StateFiles {
			if item.StateFile == sf {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	
	if len(filters.ResourceTypes) > 0 && item.Type == "resource" {
		resourceType, _ := item.Metadata["resourceType"].(string)
		matched := false
		for _, rt := range filters.ResourceTypes {
			if resourceType == rt {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	
	if filters.DateRange != nil {
		if lastMod, ok := item.Metadata["lastModified"].(time.Time); ok {
			if lastMod.Before(filters.DateRange.Start) || lastMod.After(filters.DateRange.End) {
				return false
			}
		}
	}
	
	return true
}

func (se *SearchEngine) sortResults(items []SearchItem, sort SortOptions) []SearchItem {
	if sort.Field == "" {
		sort.Field = "score"
		sort.Direction = "desc"
	}
	
	return items
}

func (se *SearchEngine) paginate(items []SearchItem, pagination PaginationOptions) []SearchItem {
	start := pagination.Offset
	end := pagination.Offset + pagination.Limit
	
	if start >= len(items) {
		return []SearchItem{}
	}
	
	if end > len(items) {
		end = len(items)
	}
	
	return items[start:end]
}

func (se *SearchEngine) calculateFacets(items []SearchItem) map[string]map[string]int {
	facets := map[string]map[string]int{
		"type":     make(map[string]int),
		"provider": make(map[string]int),
		"status":   make(map[string]int),
		"health":   make(map[string]int),
	}
	
	for _, item := range items {
		facets["type"][item.Type]++
		
		if item.Provider != "" {
			facets["provider"][item.Provider]++
		}
		
		if status, ok := item.Metadata["status"].(string); ok {
			facets["status"][status]++
		}
		
		if health, ok := item.Metadata["health"].(string); ok {
			facets["health"][health]++
		}
	}
	
	return facets
}

func (se *SearchEngine) generateSuggestions(query string, results []SearchItem) []string {
	suggestions := make([]string, 0)
	seen := make(map[string]bool)
	
	for _, item := range results {
		if len(suggestions) >= 5 {
			break
		}
		
		suggestion := item.Name
		if !seen[suggestion] && suggestion != query {
			suggestions = append(suggestions, suggestion)
			seen[suggestion] = true
		}
	}
	
	return suggestions
}

func (se *SearchEngine) getCacheKey(query SearchQuery) string {
	data, _ := json.Marshal(query)
	return string(data)
}

func (se *SearchEngine) getFromCache(key string) *SearchResult {
	se.mu.RLock()
	defer se.mu.RUnlock()
	
	if result, ok := se.cache[key]; ok {
		if cacheTime, ok := se.cacheTime[key]; ok {
			if time.Since(cacheTime) < se.cacheTTL {
				return result
			}
		}
	}
	
	return nil
}

func (se *SearchEngine) putInCache(key string, result *SearchResult) {
	se.mu.Lock()
	defer se.mu.Unlock()
	
	se.cache[key] = result
	se.cacheTime[key] = time.Now()
	
	if len(se.cache) > 100 {
		oldest := time.Now()
		oldestKey := ""
		for k, t := range se.cacheTime {
			if t.Before(oldest) {
				oldest = t
				oldestKey = k
			}
		}
		if oldestKey != "" {
			delete(se.cache, oldestKey)
			delete(se.cacheTime, oldestKey)
		}
	}
}

func (se *SearchEngine) clearCache() {
	se.cache = make(map[string]*SearchResult)
	se.cacheTime = make(map[string]time.Time)
}

func (se *SearchEngine) GetStatistics() map[string]interface{} {
	se.mu.RLock()
	defer se.mu.RUnlock()
	
	return map[string]interface{}{
		"totalStateFiles": len(se.index.StateFiles),
		"totalResources":  len(se.index.Resources),
		"totalModules":    len(se.index.Modules),
		"lastUpdated":     se.index.LastUpdated,
		"cacheSize":       len(se.cache),
	}
}

func (se *SearchEngine) RebuildIndex() {
	se.mu.Lock()
	defer se.mu.Unlock()
	
	se.index = NewSearchIndex()
	se.clearCache()
}