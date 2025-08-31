package state

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// DiscoveryService handles automatic detection of Terraform state files
type DiscoveryService struct {
	mu                sync.RWMutex
	discoveredStates  map[string]*DiscoveredStateFile
	scanPaths         []string
	cloudBackends     []BackendConfig
	lastScan          time.Time
	scanInterval      time.Duration
	stopChan          chan bool
	scanning          bool
	analyzer          *StateAnalyzer
}

// DiscoveredStateFile represents a discovered Terraform state file
type DiscoveredStateFile struct {
	ID               string                 `json:"id"`
	Path             string                 `json:"path"`
	Name             string                 `json:"name"`
	Type             string                 `json:"type"` // local, s3, azurerm, gcs, remote
	Backend          *BackendConfig `json:"backend,omitempty"`
	Size             int64                  `json:"size"`
	LastModified     time.Time              `json:"last_modified"`
	Version          int                    `json:"version"`
	TerraformVersion string                 `json:"terraform_version"`
	Serial           int                    `json:"serial"`
	Lineage          string                 `json:"lineage"`
	ResourceCount    int                    `json:"resource_count"`
	ProviderCounts   map[string]int         `json:"provider_counts"`
	ModuleCount      int                    `json:"module_count"`
	OutputCount      int                    `json:"output_count"`
	Resources        []StateResource        `json:"resources,omitempty"`
	Modules          []ModuleState          `json:"modules,omitempty"`
	IsTerragrunt     bool                   `json:"is_terragrunt"`
	TerragruntInfo   *TerragruntInfo        `json:"terragrunt_info,omitempty"`
	Health           StateHealth            `json:"health"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	DiscoveredAt     time.Time              `json:"discovered_at"`
	LastAnalyzed     time.Time              `json:"last_analyzed"`
}

// StateHealth represents the health status of a state file
type StateHealth struct {
	Status       string    `json:"status"` // healthy, warning, critical, unknown
	Age          string    `json:"age"`    // fresh, recent, stale, abandoned
	Issues       []string  `json:"issues,omitempty"`
	LastRefresh  time.Time `json:"last_refresh,omitempty"`
	Score        int       `json:"score"` // 0-100
}


// TerragruntInfo contains Terragrunt-specific information
type TerragruntInfo struct {
	ConfigPath    string   `json:"config_path"`
	Dependencies  []string `json:"dependencies,omitempty"`
	RemoteState   string   `json:"remote_state,omitempty"`
	IncludePaths  []string `json:"include_paths,omitempty"`
	ParentHCLPath string   `json:"parent_hcl_path,omitempty"`
}

// ModuleState represents a module within a state file
type ModuleState struct {
	Path      []string               `json:"path"`
	Resources map[string]interface{} `json:"resources"`
	Outputs   map[string]interface{} `json:"outputs"`
}

// NewDiscoveryService creates a new state discovery service
func NewDiscoveryService() *DiscoveryService {
	return &DiscoveryService{
		discoveredStates: make(map[string]*DiscoveredStateFile),
		scanPaths:        []string{},
		cloudBackends:    []BackendConfig{},
		scanInterval:     5 * time.Minute,
		stopChan:         make(chan bool),
		analyzer:         NewStateAnalyzer(),
	}
}

// AddScanPath adds a path to scan for state files
func (ds *DiscoveryService) AddScanPath(path string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	
	// Normalize and validate path
	absPath, err := filepath.Abs(path)
	if err == nil {
		ds.scanPaths = append(ds.scanPaths, absPath)
	}
}

// AddCloudBackend adds a cloud backend configuration to scan
func (ds *DiscoveryService) AddCloudBackend(backend BackendConfig) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.cloudBackends = append(ds.cloudBackends, backend)
}

// StartAutoDiscovery starts automatic discovery with periodic scanning
func (ds *DiscoveryService) StartAutoDiscovery(ctx context.Context) {
	go func() {
		// Initial scan
		ds.DiscoverAll(ctx)
		
		ticker := time.NewTicker(ds.scanInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ds.stopChan:
				return
			case <-ticker.C:
				ds.DiscoverAll(ctx)
			}
		}
	}()
}

// StopAutoDiscovery stops the automatic discovery process
func (ds *DiscoveryService) StopAutoDiscovery() {
	close(ds.stopChan)
}

// DiscoverAll performs a comprehensive discovery of all state files
func (ds *DiscoveryService) DiscoverAll(ctx context.Context) error {
	ds.mu.Lock()
	if ds.scanning {
		ds.mu.Unlock()
		return fmt.Errorf("discovery already in progress")
	}
	ds.scanning = true
	ds.lastScan = time.Now()
	ds.mu.Unlock()
	
	defer func() {
		ds.mu.Lock()
		ds.scanning = false
		ds.mu.Unlock()
	}()
	
	var wg sync.WaitGroup
	errChan := make(chan error, 10)
	
	// Scan local filesystem paths
	for _, path := range ds.scanPaths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			if err := ds.scanLocalPath(ctx, p); err != nil {
				errChan <- fmt.Errorf("error scanning %s: %w", p, err)
			}
		}(path)
	}
	
	// Scan cloud backends
	for _, backend := range ds.cloudBackends {
		wg.Add(1)
		go func(b BackendConfig) {
			defer wg.Done()
			if err := ds.scanCloudBackend(ctx, b); err != nil {
				errChan <- fmt.Errorf("error scanning backend %s: %w", b.Type, err)
			}
		}(backend)
	}
	
	// Scan git repositories
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := ds.scanGitRepositories(ctx); err != nil {
			errChan <- err
		}
	}()
	
	// Wait for all scans to complete
	wg.Wait()
	close(errChan)
	
	// Collect any errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("discovery completed with %d errors", len(errors))
	}
	
	return nil
}

// scanLocalPath scans a local filesystem path for state files
func (ds *DiscoveryService) scanLocalPath(ctx context.Context, basePath string) error {
	return filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue scanning other paths
		}
		
		// Check for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// Skip directories and non-state files
		if info.IsDir() {
			// Check for .terraform directories to find local state
			terraformDir := filepath.Join(path, ".terraform")
			if _, err := os.Stat(terraformDir); err == nil {
				ds.scanTerraformDirectory(ctx, terraformDir)
			}
			return nil
		}
		
		// Check for state files
		if isStateFile(path) {
			if stateFile := ds.analyzeStateFile(ctx, path, "local"); stateFile != nil {
				ds.addDiscoveredState(stateFile)
			}
		}
		
		// Check for Terragrunt configuration
		if isTerragruntFile(path) {
			ds.analyzeTerragruntConfig(ctx, path)
		}
		
		// Check for backend configuration
		if isBackendConfig(path) {
			ds.analyzeBackendConfig(ctx, path)
		}
		
		return nil
	})
}

// scanTerraformDirectory scans a .terraform directory for state files
func (ds *DiscoveryService) scanTerraformDirectory(ctx context.Context, terraformDir string) {
	stateFile := filepath.Join(terraformDir, "terraform.tfstate")
	if _, err := os.Stat(stateFile); err == nil {
		if state := ds.analyzeStateFile(ctx, stateFile, "local"); state != nil {
			ds.addDiscoveredState(state)
		}
	}
	
	// Check for environment-specific state files
	envStatePattern := filepath.Join(terraformDir, "*.tfstate")
	matches, _ := filepath.Glob(envStatePattern)
	for _, match := range matches {
		if state := ds.analyzeStateFile(ctx, match, "local"); state != nil {
			ds.addDiscoveredState(state)
		}
	}
}

// scanCloudBackend scans a cloud backend for state files
func (ds *DiscoveryService) scanCloudBackend(ctx context.Context, backend BackendConfig) error {
	switch backend.Type {
	case "s3":
		return ds.scanS3Backend(ctx, backend)
	case "azurerm":
		return ds.scanAzureBackend(ctx, backend)
	case "gcs":
		return ds.scanGCSBackend(ctx, backend)
	case "remote":
		return ds.scanRemoteBackend(ctx, backend)
	default:
		return fmt.Errorf("unsupported backend type: %s", backend.Type)
	}
}

// scanS3Backend scans an S3 backend for state files
func (ds *DiscoveryService) scanS3Backend(ctx context.Context, backend BackendConfig) error {
	// Implementation would use AWS SDK to scan S3 bucket
	// For now, return placeholder
	fmt.Printf("Scanning S3 backend: %v\n", backend.Config)
	return nil
}

// scanAzureBackend scans an Azure Storage backend for state files
func (ds *DiscoveryService) scanAzureBackend(ctx context.Context, backend BackendConfig) error {
	// Implementation would use Azure SDK to scan storage account
	fmt.Printf("Scanning Azure backend: %v\n", backend.Config)
	return nil
}

// scanGCSBackend scans a Google Cloud Storage backend for state files
func (ds *DiscoveryService) scanGCSBackend(ctx context.Context, backend BackendConfig) error {
	// Implementation would use GCP SDK to scan GCS bucket
	fmt.Printf("Scanning GCS backend: %v\n", backend.Config)
	return nil
}

// scanRemoteBackend scans a Terraform Cloud/Enterprise backend
func (ds *DiscoveryService) scanRemoteBackend(ctx context.Context, backend BackendConfig) error {
	// Implementation would use Terraform Cloud API
	fmt.Printf("Scanning remote backend: %v\n", backend.Config)
	return nil
}

// scanGitRepositories scans git repositories for state files
func (ds *DiscoveryService) scanGitRepositories(ctx context.Context) error {
	// Look for git repositories in scan paths
	for _, path := range ds.scanPaths {
		gitDir := filepath.Join(path, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			// This is a git repository
			ds.scanGitRepository(ctx, path)
		}
	}
	return nil
}

// scanGitRepository scans a single git repository
func (ds *DiscoveryService) scanGitRepository(ctx context.Context, repoPath string) {
	// Would use git commands to find state files in history
	// For now, just scan the working directory
	ds.scanLocalPath(ctx, repoPath)
}

// analyzeStateFile analyzes a discovered state file
func (ds *DiscoveryService) analyzeStateFile(ctx context.Context, path string, backendType string) *DiscoveredStateFile {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	
	// Read state file content
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil
	}
	
	// Parse state file
	var tfState TerraformState
	if err := json.Unmarshal(content, &tfState); err != nil {
		return nil
	}
	
	// Create state file record
	stateFile := &DiscoveredStateFile{
		ID:               generateStateFileID(path),
		Path:             path,
		Name:             filepath.Base(path),
		Type:             backendType,
		Size:             info.Size(),
		LastModified:     info.ModTime(),
		Version:          tfState.Version,
		TerraformVersion: tfState.TerraformVersion,
		Serial:           tfState.Serial,
		Lineage:          tfState.Lineage,
		DiscoveredAt:     time.Now(),
		LastAnalyzed:     time.Now(),
		Metadata:         make(map[string]interface{}),
	}
	
	// Count resources and providers
	providerCounts := make(map[string]int)
	for _, resource := range tfState.Resources {
		stateFile.ResourceCount++
		provider := extractProviderFromType(resource.Type)
		providerCounts[provider]++
	}
	stateFile.ProviderCounts = providerCounts
	
	// Check for Terragrunt
	stateFile.IsTerragrunt = ds.checkForTerragrunt(filepath.Dir(path))
	if stateFile.IsTerragrunt {
		stateFile.TerragruntInfo = ds.getTerragruntInfo(filepath.Dir(path))
	}
	
	// Calculate health
	stateFile.Health = ds.calculateStateHealth(stateFile)
	
	return stateFile
}

// analyzeTerragruntConfig analyzes a Terragrunt configuration file
func (ds *DiscoveryService) analyzeTerragruntConfig(ctx context.Context, path string) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	
	// Parse HCL content
	parser := hclparse.NewParser()
	_, diags := parser.ParseHCL(content, path)
	if diags.HasErrors() {
		return
	}
	
	// Extract Terragrunt configuration
	// This would parse the Terragrunt-specific blocks
	fmt.Printf("Found Terragrunt config: %s\n", path)
}

// analyzeBackendConfig analyzes a Terraform backend configuration
func (ds *DiscoveryService) analyzeBackendConfig(ctx context.Context, path string) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	
	// Parse HCL to extract backend configuration
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(content, path)
	if diags.HasErrors() {
		return
	}
	
	// Extract backend block
	// This would parse terraform { backend "type" { ... } } blocks
	_ = file // Use the parsed file to extract backend config
	fmt.Printf("Found backend config in: %s\n", path)
}

// checkForTerragrunt checks if a directory contains Terragrunt configuration
func (ds *DiscoveryService) checkForTerragrunt(dir string) bool {
	terragruntFile := filepath.Join(dir, "terragrunt.hcl")
	if _, err := os.Stat(terragruntFile); err == nil {
		return true
	}
	
	// Check for legacy terragrunt.tfvars
	terragruntLegacy := filepath.Join(dir, "terraform.tfvars")
	if content, err := ioutil.ReadFile(terragruntLegacy); err == nil {
		if strings.Contains(string(content), "terragrunt") {
			return true
		}
	}
	
	return false
}

// getTerragruntInfo extracts Terragrunt-specific information
func (ds *DiscoveryService) getTerragruntInfo(dir string) *TerragruntInfo {
	info := &TerragruntInfo{}
	
	terragruntFile := filepath.Join(dir, "terragrunt.hcl")
	if _, err := os.Stat(terragruntFile); err == nil {
		info.ConfigPath = terragruntFile
		// Parse the file to extract dependencies, includes, etc.
	}
	
	return info
}

// calculateStateHealth calculates the health score and status of a state file
func (ds *DiscoveryService) calculateStateHealth(state *DiscoveredStateFile) StateHealth {
	health := StateHealth{
		Status: "healthy",
		Issues: []string{},
		Score:  100,
	}
	
	// Calculate age
	age := time.Since(state.LastModified)
	switch {
	case age < 24*time.Hour:
		health.Age = "fresh"
	case age < 7*24*time.Hour:
		health.Age = "recent"
	case age < 30*24*time.Hour:
		health.Age = "stale"
		health.Score -= 20
		health.Issues = append(health.Issues, "State file is stale (>7 days old)")
	default:
		health.Age = "abandoned"
		health.Score -= 40
		health.Status = "warning"
		health.Issues = append(health.Issues, "State file appears abandoned (>30 days old)")
	}
	
	// Check size
	if state.Size > 10*1024*1024 { // 10MB
		health.Score -= 10
		health.Issues = append(health.Issues, "State file is large (>10MB)")
		if state.Size > 50*1024*1024 { // 50MB
			health.Status = "critical"
			health.Score -= 20
			health.Issues = append(health.Issues, "State file is very large (>50MB) - consider splitting")
		}
	}
	
	// Check resource count
	if state.ResourceCount > 500 {
		health.Score -= 10
		health.Issues = append(health.Issues, "High resource count (>500)")
		if state.ResourceCount > 1000 {
			health.Status = "warning"
			health.Score -= 20
			health.Issues = append(health.Issues, "Very high resource count (>1000) - consider splitting")
		}
	}
	
	// Ensure score doesn't go below 0
	if health.Score < 0 {
		health.Score = 0
	}
	
	// Set final status based on score
	switch {
	case health.Score >= 80:
		health.Status = "healthy"
	case health.Score >= 60:
		health.Status = "warning"
	case health.Score >= 40:
		health.Status = "critical"
	default:
		health.Status = "unknown"
	}
	
	return health
}

// addDiscoveredState adds a discovered state file to the service
func (ds *DiscoveryService) addDiscoveredState(state *DiscoveredStateFile) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.discoveredStates[state.ID] = state
}

// GetDiscoveredStates returns all discovered state files
func (ds *DiscoveryService) GetDiscoveredStates() []*DiscoveredStateFile {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	states := make([]*DiscoveredStateFile, 0, len(ds.discoveredStates))
	for _, state := range ds.discoveredStates {
		states = append(states, state)
	}
	return states
}

// GetStateFile returns a specific state file by ID
func (ds *DiscoveryService) GetStateFile(id string) (*DiscoveredStateFile, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	state, exists := ds.discoveredStates[id]
	if !exists {
		return nil, fmt.Errorf("state file not found: %s", id)
	}
	return state, nil
}

// RefreshStateFile refreshes the analysis of a specific state file
func (ds *DiscoveryService) RefreshStateFile(ctx context.Context, id string) error {
	state, err := ds.GetStateFile(id)
	if err != nil {
		return err
	}
	
	// Re-analyze the state file
	refreshed := ds.analyzeStateFile(ctx, state.Path, state.Type)
	if refreshed != nil {
		ds.addDiscoveredState(refreshed)
	}
	
	return nil
}

// GetStateFilesByBackend returns state files filtered by backend type
func (ds *DiscoveryService) GetStateFilesByBackend(backendType string) []*DiscoveredStateFile {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	var states []*DiscoveredStateFile
	for _, state := range ds.discoveredStates {
		if state.Type == backendType {
			states = append(states, state)
		}
	}
	return states
}

// GetStateFilesByHealth returns state files filtered by health status
func (ds *DiscoveryService) GetStateFilesByHealth(status string) []*DiscoveredStateFile {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	var states []*DiscoveredStateFile
	for _, state := range ds.discoveredStates {
		if state.Health.Status == status {
			states = append(states, state)
		}
	}
	return states
}

// ConvertToStateFile converts a DiscoveredStateFile to StateFile (State)
func ConvertDiscoveredToStateFile(discovered *DiscoveredStateFile) *StateFile {
	if discovered == nil {
		return nil
	}
	
	// Create a StateFile with basic fields
	state := &StateFile{
		ID:               discovered.ID,
		Path:             discovered.Path,
		Version:          discovered.Version,
		TerraformVersion: discovered.TerraformVersion,
		Serial:           discovered.Serial,
		Lineage:          discovered.Lineage,
		Resources:        []Resource{},
	}
	
	// Convert resources if needed
	for _, r := range discovered.Resources {
		resource := Resource{
			Module:   r.Module,
			Mode:     r.Mode,
			Type:     r.Type,
			Name:     r.Name,
			Provider: r.Provider,
		}
		state.Resources = append(state.Resources, resource)
	}
	
	return state
}

// ConvertDiscoveredSliceToStateFiles converts a slice of DiscoveredStateFile to StateFile
func ConvertDiscoveredSliceToStateFiles(discovered []*DiscoveredStateFile) []*StateFile {
	result := make([]*StateFile, len(discovered))
	for i, d := range discovered {
		result[i] = ConvertDiscoveredToStateFile(d)
	}
	return result
}

// GetScanStatus returns the current scanning status
func (ds *DiscoveryService) GetScanStatus() map[string]interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	
	return map[string]interface{}{
		"scanning":          ds.scanning,
		"last_scan":         ds.lastScan,
		"total_discovered":  len(ds.discoveredStates),
		"scan_paths":        ds.scanPaths,
		"cloud_backends":    len(ds.cloudBackends),
		"next_scan":         ds.lastScan.Add(ds.scanInterval),
	}
}

// DiscoverFromStateFile discovers resources from a Terraform state file
func (ds *DiscoveryService) DiscoverFromStateFile(ctx context.Context, path string) (map[string]*models.Resource, error) {
	// Load the state file
	stateData, err := LoadStateFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load state file: %w", err)
	}

	// Convert state resources to models.Resource
	resources := make(map[string]*models.Resource)
	for _, res := range stateData.Resources {
		for idx, instance := range res.Instances {
			// Generate unique ID for each resource instance
			resourceID := fmt.Sprintf("%s.%s[%d]", res.Type, res.Name, idx)
			
			resource := &models.Resource{
				ID:       resourceID,
				Name:     res.Name,
				Type:     res.Type,
				Provider: res.Provider,
			}

			// Extract additional properties from attributes
			if instance.Attributes != nil {
				if region, ok := instance.Attributes["region"].(string); ok {
					resource.Region = region
				} else if location, ok := instance.Attributes["location"].(string); ok {
					resource.Region = location
				}

				// Extract tags if present
				if tags, ok := instance.Attributes["tags"].(map[string]interface{}); ok {
					tagMap := make(map[string]string)
					for k, v := range tags {
						if strVal, ok := v.(string); ok {
							tagMap[k] = strVal
						}
					}
					resource.Tags = tagMap
				}
			}

			resources[resourceID] = resource
		}
	}

	return resources, nil
}

// Helper functions

func isStateFile(path string) bool {
	name := filepath.Base(path)
	return strings.HasSuffix(name, ".tfstate") || 
		strings.HasSuffix(name, ".tfstate.backup")
}

func isTerragruntFile(path string) bool {
	name := filepath.Base(path)
	return name == "terragrunt.hcl" || name == "terragrunt.tfvars"
}

func isBackendConfig(path string) bool {
	name := filepath.Base(path)
	return strings.HasSuffix(name, ".tf") || 
		strings.HasSuffix(name, ".tf.json") ||
		name == "backend.hcl"
}

func generateStateFileID(path string) string {
	// Generate a unique ID based on path
	return fmt.Sprintf("state_%x", hashString(path))
}

func hashString(s string) uint32 {
	var h uint32
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return h
}

func extractProviderFromType(resourceType string) string {
	parts := strings.Split(resourceType, "_")
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}