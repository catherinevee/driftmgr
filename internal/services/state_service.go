package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/cache"
	"github.com/catherinevee/driftmgr/internal/events"
	"github.com/catherinevee/driftmgr/internal/jobs"
	"github.com/catherinevee/driftmgr/internal/terraform/state"
)

// StateService provides unified state management operations
type StateService struct {
	stateLoader  *state.Loader
	stateManager *state.Manager
	cache        cache.Cache
	eventBus     *events.EventBus
	jobQueue     *jobs.Queue
	mu           sync.RWMutex
}

// NewStateService creates a new state service
func NewStateService(
	stateLoader *state.Loader,
	cache cache.Cache,
	eventBus *events.EventBus,
	jobQueue *jobs.Queue,
) *StateService {
	return &StateService{
		stateLoader:  stateLoader,
		stateManager: state.NewManager(),
		cache:        cache,
		eventBus:     eventBus,
		jobQueue:     jobQueue,
	}
}

// StateFile represents a Terraform state file
type StateFile struct {
	ID           string                 `json:"id"`
	Path         string                 `json:"path"`
	Name         string                 `json:"name"`
	Backend      string                 `json:"backend"`
	Workspace    string                 `json:"workspace"`
	Version      int                    `json:"version"`
	Resources    []StateResource        `json:"resources"`
	Outputs      map[string]interface{} `json:"outputs,omitempty"`
	LastModified time.Time              `json:"last_modified"`
	Size         int64                  `json:"size"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// StateResource represents a resource in a state file
type StateResource struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Provider   string                 `json:"provider"`
	Module     string                 `json:"module,omitempty"`
	Attributes map[string]interface{} `json:"attributes"`
	Dependencies []string             `json:"dependencies,omitempty"`
}

// StateAnalysis represents the analysis of state files
type StateAnalysis struct {
	TotalFiles          int                       `json:"total_files"`
	TotalResources      int                       `json:"total_resources"`
	UniqueProviders     int                       `json:"unique_providers"`
	TerraformVersions   []string                  `json:"terraform_versions"`
	Modules             []string                  `json:"modules"`
	ResourceTypes       map[string]int            `json:"resource_types"`
	ProviderDistribution map[string]int           `json:"provider_distribution"`
	Workspaces          []string                  `json:"workspaces"`
	BackendTypes        map[string]int            `json:"backend_types"`
	AnalyzedAt          time.Time                 `json:"analyzed_at"`
}

// StateComparisonResult represents the comparison between two state files
type StateComparisonResult struct {
	File1           string          `json:"file1"`
	File2           string          `json:"file2"`
	AddedResources  []StateResource `json:"added_resources"`
	RemovedResources []StateResource `json:"removed_resources"`
	ModifiedResources []StateResource `json:"modified_resources"`
	UnchangedCount  int             `json:"unchanged_count"`
	ComparedAt      time.Time       `json:"compared_at"`
}

// DiscoverStateFiles discovers Terraform state files
func (s *StateService) DiscoverStateFiles(ctx context.Context, paths []string) ([]*StateFile, error) {
	var stateFiles []*StateFile
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Default paths if none provided
	if len(paths) == 0 {
		paths = []string{
			".",
			"terraform",
			"infrastructure",
			"deployments",
		}
	}

	// Emit discovery started event
	s.eventBus.Publish(events.Event{
		Type: events.StateImported,
		Data: map[string]interface{}{
			"action": "discovery_started",
			"paths":  paths,
		},
	})

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			
			files, err := s.discoverInPath(ctx, p)
			if err != nil {
				return
			}
			
			mu.Lock()
			stateFiles = append(stateFiles, files...)
			mu.Unlock()
		}(path)
	}

	wg.Wait()

	// Cache discovered files
	for _, file := range stateFiles {
		cacheKey := fmt.Sprintf("state:file:%s", file.ID)
		s.cache.Set(cacheKey, file, 1*time.Hour)
	}

	// Emit discovery completed event
	s.eventBus.Publish(events.Event{
		Type: events.StateImported,
		Data: map[string]interface{}{
			"action": "discovery_completed",
			"count":  len(stateFiles),
		},
	})

	return stateFiles, nil
}

// discoverInPath discovers state files in a specific path
func (s *StateService) discoverInPath(ctx context.Context, basePath string) ([]*StateFile, error) {
	var stateFiles []*StateFile

	// Look for common state file patterns
	patterns := []string{
		"*.tfstate",
		"*.tfstate.backup",
		"terraform.tfstate.d/*/*.tfstate",
		".terraform/*.tfstate",
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(basePath, pattern))
		if err != nil {
			continue
		}

		for _, match := range matches {
			stateFile, err := s.loadStateFile(match)
			if err != nil {
				continue
			}
			stateFiles = append(stateFiles, stateFile)
		}
	}

	// Also check for Terragrunt state files
	terragruntPattern := filepath.Join(basePath, "**", ".terragrunt-cache", "**", "*.tfstate")
	terragruntMatches, _ := filepath.Glob(terragruntPattern)
	for _, match := range terragruntMatches {
		stateFile, err := s.loadStateFile(match)
		if err != nil {
			continue
		}
		stateFile.Metadata["terragrunt"] = true
		stateFiles = append(stateFiles, stateFile)
	}

	return stateFiles, nil
}

// loadStateFile loads a single state file
func (s *StateService) loadStateFile(path string) (*StateFile, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tfState map[string]interface{}
	if err := json.Unmarshal(data, &tfState); err != nil {
		return nil, err
	}

	// Extract resources
	var resources []StateResource
	if res, ok := tfState["resources"].([]interface{}); ok {
		for _, r := range res {
			if resource, ok := r.(map[string]interface{}); ok {
				resources = append(resources, s.parseResource(resource))
			}
		}
	}

	// Determine backend type
	backend := "local"
	if b, ok := tfState["backend"].(map[string]interface{}); ok {
		if bType, ok := b["type"].(string); ok {
			backend = bType
		}
	}

	// Get file info
	fileInfo, _ := filepath.Abs(path)
	fileName := filepath.Base(path)

	stateFile := &StateFile{
		ID:           generateStateFileID(path),
		Path:         fileInfo,
		Name:         fileName,
		Backend:      backend,
		Workspace:    extractWorkspace(path),
		Version:      extractVersion(tfState),
		Resources:    resources,
		Outputs:      extractOutputs(tfState),
		LastModified: time.Now(),
		Size:         int64(len(data)),
		Metadata:     make(map[string]interface{}),
	}

	return stateFile, nil
}

// parseResource parses a resource from state
func (s *StateService) parseResource(resource map[string]interface{}) StateResource {
	sr := StateResource{
		Attributes: make(map[string]interface{}),
	}

	if id, ok := resource["id"].(string); ok {
		sr.ID = id
	}
	if rType, ok := resource["type"].(string); ok {
		sr.Type = rType
	}
	if name, ok := resource["name"].(string); ok {
		sr.Name = name
	}
	if provider, ok := resource["provider"].(string); ok {
		sr.Provider = provider
	}
	if module, ok := resource["module"].(string); ok {
		sr.Module = module
	}
	if attrs, ok := resource["instances"].([]interface{}); ok && len(attrs) > 0 {
		if inst, ok := attrs[0].(map[string]interface{}); ok {
			if attributes, ok := inst["attributes"].(map[string]interface{}); ok {
				sr.Attributes = attributes
			}
		}
	}

	return sr
}

// ImportStateFile imports a state file
func (s *StateService) ImportStateFile(ctx context.Context, path string) (*StateFile, error) {
	stateFile, err := s.loadStateFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load state file: %w", err)
	}

	// Cache the imported file
	cacheKey := fmt.Sprintf("state:file:%s", stateFile.ID)
	s.cache.Set(cacheKey, stateFile, 1*time.Hour)

	// Emit import event
	s.eventBus.Publish(events.Event{
		Type: events.StateImported,
		Data: map[string]interface{}{
			"file_id":   stateFile.ID,
			"path":      stateFile.Path,
			"resources": len(stateFile.Resources),
		},
	})

	return stateFile, nil
}

// AnalyzeStateFiles performs analysis on state files
func (s *StateService) AnalyzeStateFiles(ctx context.Context, fileIDs []string) (*StateAnalysis, error) {
	analysis := &StateAnalysis{
		ResourceTypes:        make(map[string]int),
		ProviderDistribution: make(map[string]int),
		BackendTypes:         make(map[string]int),
		TerraformVersions:    []string{},
		Modules:              []string{},
		Workspaces:           []string{},
		AnalyzedAt:           time.Now(),
	}

	providersSet := make(map[string]bool)
	versionsSet := make(map[string]bool)
	modulesSet := make(map[string]bool)
	workspacesSet := make(map[string]bool)

	for _, fileID := range fileIDs {
		cacheKey := fmt.Sprintf("state:file:%s", fileID)
		cached, found := s.cache.Get(cacheKey)
		if !found {
			continue
		}

		stateFile, ok := cached.(*StateFile)
		if !ok {
			continue
		}

		analysis.TotalFiles++
		analysis.TotalResources += len(stateFile.Resources)
		analysis.BackendTypes[stateFile.Backend]++
		
		if stateFile.Workspace != "" {
			workspacesSet[stateFile.Workspace] = true
		}

		// Analyze resources
		for _, resource := range stateFile.Resources {
			analysis.ResourceTypes[resource.Type]++
			
			if resource.Provider != "" {
				providersSet[resource.Provider] = true
				analysis.ProviderDistribution[resource.Provider]++
			}
			
			if resource.Module != "" {
				modulesSet[resource.Module] = true
			}
		}
	}

	// Convert sets to slices
	for range providersSet {
		analysis.UniqueProviders++
	}
	for version := range versionsSet {
		analysis.TerraformVersions = append(analysis.TerraformVersions, version)
	}
	for module := range modulesSet {
		analysis.Modules = append(analysis.Modules, module)
	}
	for workspace := range workspacesSet {
		analysis.Workspaces = append(analysis.Workspaces, workspace)
	}

	// Cache analysis
	s.cache.Set("state:analysis:latest", analysis, 30*time.Minute)

	// Emit analysis event
	s.eventBus.Publish(events.Event{
		Type: events.StateAnalyzed,
		Data: map[string]interface{}{
			"total_files":     analysis.TotalFiles,
			"total_resources": analysis.TotalResources,
			"providers":       analysis.UniqueProviders,
		},
	})

	return analysis, nil
}

// CompareStateFiles compares two state files
func (s *StateService) CompareStateFiles(ctx context.Context, fileID1, fileID2 string) (*StateComparisonResult, error) {
	// Get state files from cache
	var stateFile1, stateFile2 *StateFile
	
	if cached, found := s.cache.Get(fmt.Sprintf("state:file:%s", fileID1)); found {
		stateFile1, _ = cached.(*StateFile)
	}
	if cached, found := s.cache.Get(fmt.Sprintf("state:file:%s", fileID2)); found {
		stateFile2, _ = cached.(*StateFile)
	}

	if stateFile1 == nil || stateFile2 == nil {
		return nil, fmt.Errorf("state files not found")
	}

	result := &StateComparisonResult{
		File1:             stateFile1.Path,
		File2:             stateFile2.Path,
		AddedResources:    []StateResource{},
		RemovedResources:  []StateResource{},
		ModifiedResources: []StateResource{},
		ComparedAt:        time.Now(),
	}

	// Create maps for comparison
	resources1 := make(map[string]StateResource)
	resources2 := make(map[string]StateResource)

	for _, r := range stateFile1.Resources {
		key := fmt.Sprintf("%s.%s", r.Type, r.Name)
		resources1[key] = r
	}
	for _, r := range stateFile2.Resources {
		key := fmt.Sprintf("%s.%s", r.Type, r.Name)
		resources2[key] = r
	}

	// Find added and modified resources
	for key, r2 := range resources2 {
		if r1, exists := resources1[key]; exists {
			if !resourcesEqual(r1, r2) {
				result.ModifiedResources = append(result.ModifiedResources, r2)
			} else {
				result.UnchangedCount++
			}
		} else {
			result.AddedResources = append(result.AddedResources, r2)
		}
	}

	// Find removed resources
	for key, r1 := range resources1 {
		if _, exists := resources2[key]; !exists {
			result.RemovedResources = append(result.RemovedResources, r1)
		}
	}

	// Cache comparison result
	cacheKey := fmt.Sprintf("state:comparison:%s:%s", fileID1, fileID2)
	s.cache.Set(cacheKey, result, 15*time.Minute)

	return result, nil
}

// GetStateFile retrieves a state file by ID
func (s *StateService) GetStateFile(ctx context.Context, fileID string) (*StateFile, error) {
	cacheKey := fmt.Sprintf("state:file:%s", fileID)
	if cached, found := s.cache.Get(cacheKey); found {
		if stateFile, ok := cached.(*StateFile); ok {
			return stateFile, nil
		}
	}
	return nil, fmt.Errorf("state file not found: %s", fileID)
}

// ListStateFiles lists all cached state files
func (s *StateService) ListStateFiles(ctx context.Context) ([]*StateFile, error) {
	var stateFiles []*StateFile
	
	keys := s.cache.Keys("state:file:*")
	for _, key := range keys {
		if cached, found := s.cache.Get(key); found {
			if stateFile, ok := cached.(*StateFile); ok {
				stateFiles = append(stateFiles, stateFile)
			}
		}
	}

	return stateFiles, nil
}

// DeleteStateFile removes a state file from cache
func (s *StateService) DeleteStateFile(ctx context.Context, fileID string) error {
	cacheKey := fmt.Sprintf("state:file:%s", fileID)
	s.cache.Delete(cacheKey)

	// Emit delete event
	s.eventBus.Publish(events.Event{
		Type: events.StateDeleted,
		Data: map[string]interface{}{
			"file_id": fileID,
		},
	})

	return nil
}

// Helper functions

func generateStateFileID(path string) string {
	return fmt.Sprintf("state-%d-%s", time.Now().Unix(), filepath.Base(path))
}

func extractWorkspace(path string) string {
	if strings.Contains(path, "terraform.tfstate.d") {
		parts := strings.Split(path, string(filepath.Separator))
		for i, part := range parts {
			if part == "terraform.tfstate.d" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
	}
	return "default"
}

func extractVersion(tfState map[string]interface{}) int {
	if v, ok := tfState["version"].(float64); ok {
		return int(v)
	}
	return 0
}

func extractOutputs(tfState map[string]interface{}) map[string]interface{} {
	if outputs, ok := tfState["outputs"].(map[string]interface{}); ok {
		return outputs
	}
	return nil
}

func resourcesEqual(r1, r2 StateResource) bool {
	// Simple equality check - can be enhanced
	return r1.ID == r2.ID && r1.Type == r2.Type && r1.Name == r2.Name
}