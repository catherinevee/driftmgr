package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/terraform/state"
)

// StateDiscoveryService provides intelligent state file discovery
type StateDiscoveryService struct {
	rootPaths           []string
	backendConfigs      []BackendConfig
	discoveredStates    map[string]*DiscoveredState
	stateFingerprints   map[string]StateFingerprint
	mu                  sync.RWMutex
	scanDepth           int
	includeBackups      bool
	includeLockTables   bool
	includeRemoteStates bool
}

// DiscoveredState represents a discovered Terraform state
type DiscoveredState struct {
	Path             string
	Type             StateType
	Backend          BackendType
	Workspace        string
	Environment      string
	LastModified     time.Time
	ResourceCount    int
	Provider         string
	TerraformVersion string
	Modules          []string
	RemoteConfig     *BackendConfig
	IsPartial        bool
	IsOrphaned       bool
	RelatedStates    []string
}

// StateType represents the type of state file
type StateType string

const (
	StateTypeLocal      StateType = "local"
	StateTypeRemote     StateType = "remote"
	StateTypeWorkspace  StateType = "workspace"
	StateTypeBackup     StateType = "backup"
	StateTypeTerragrunt StateType = "terragrunt"
	StateTypeModule     StateType = "module"
	StateTypeFragment   StateType = "fragment"
)

// BackendType represents the backend storage type
type BackendType string

const (
	BackendTypeLocal BackendType = "local"
	BackendTypeS3    BackendType = "s3"
	BackendTypeAzure BackendType = "azurerm"
	BackendTypeGCS   BackendType = "gcs"
	BackendTypeHTTP  BackendType = "http"
	BackendTypeConsul BackendType = "consul"
	BackendTypeEtcd  BackendType = "etcd"
)

// BackendConfig represents remote backend configuration
type BackendConfig struct {
	Type            BackendType
	Bucket          string
	Key             string
	Region          string
	DynamoDBTable   string
	ContainerName   string
	StorageAccount  string
	AccessKey       string
	SecretKey       string
	Token           string
	Endpoint        string
	EncryptionKey   string
	WorkspacePrefix string
}

// StateFingerprint identifies characteristics of a state file
type StateFingerprint struct {
	NamingConvention   string
	TagPatterns        map[string]string
	ResourcePatterns   []string
	ModulePatterns     []string
	ProviderVersions   map[string]string
	CreationPatterns   CreationPattern
	TeamSignature      string
	EnvironmentMarkers []string
}

// CreationPattern identifies how resources were created
type CreationPattern struct {
	IsManual           bool
	IsTerraform        bool
	IsImported         bool
	HasConsistentNaming bool
	HasAutomationTags  bool
	CreationTimeCluster []time.Time
}

// StateAggregator aggregates multiple state files
type StateAggregator struct {
	states              map[string]*state.State
	resourceIndex       map[string][]StateResourceMapping
	workspaceMap        map[string][]string
	environmentMap      map[string][]string
	moduleStateMap      map[string][]string
	crossStateRefs      map[string][]CrossStateReference
	partialStates       []PartialStateInfo
	orphanedResources   []OrphanedResource
}

// StateResourceMapping maps resources to their state files
type StateResourceMapping struct {
	StateFile    string
	ResourceID   string
	ResourceType string
	Workspace    string
	Environment  string
	LastModified time.Time
}

// CrossStateReference represents references between states
type CrossStateReference struct {
	SourceState      string
	TargetState      string
	ReferenceType    string
	ResourceID       string
	DataSourceQuery  string
}

// PartialStateInfo represents incomplete state management
type PartialStateInfo struct {
	StateFile         string
	MissingResources  []string
	FailedImports     []string
	IncompleteModules []string
}

// OrphanedResource represents a resource that was managed but isn't anymore
type OrphanedResource struct {
	ResourceID       string
	ResourceType     string
	LastSeenState    string
	LastSeenTime     time.Time
	RemovalDetected  time.Time
	LikelyReason     string
}

// NewStateDiscoveryService creates a new state discovery service
func NewStateDiscoveryService(rootPaths []string) *StateDiscoveryService {
	return &StateDiscoveryService{
		rootPaths:           rootPaths,
		discoveredStates:    make(map[string]*DiscoveredState),
		stateFingerprints:   make(map[string]StateFingerprint),
		scanDepth:           10,
		includeBackups:      true,
		includeLockTables:   true,
		includeRemoteStates: true,
	}
}

// DiscoverAllStates performs comprehensive state discovery
func (s *StateDiscoveryService) DiscoverAllStates(ctx context.Context) (map[string]*DiscoveredState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Phase 1: Local filesystem discovery
	if err := s.discoverLocalStates(ctx); err != nil {
		return nil, fmt.Errorf("local state discovery failed: %w", err)
	}
	
	// Phase 2: Backend state discovery
	if s.includeRemoteStates {
		if err := s.discoverRemoteStates(ctx); err != nil {
			// Don't fail completely if remote discovery fails
			fmt.Printf("Warning: Remote state discovery failed: %v\n", err)
		}
	}
	
	// Phase 3: CI/CD artifact discovery
	if err := s.discoverCIArtifacts(ctx); err != nil {
		fmt.Printf("Warning: CI artifact discovery failed: %v\n", err)
	}
	
	// Phase 4: Lock table discovery
	if s.includeLockTables {
		if err := s.discoverFromLockTables(ctx); err != nil {
			fmt.Printf("Warning: Lock table discovery failed: %v\n", err)
		}
	}
	
	// Phase 5: Terragrunt cache discovery
	if err := s.discoverTerragruntStates(ctx); err != nil {
		fmt.Printf("Warning: Terragrunt discovery failed: %v\n", err)
	}
	
	// Phase 6: Cross-reference and validate
	s.crossReferenceStates()
	
	return s.discoveredStates, nil
}

// discoverLocalStates searches filesystem for state files
func (s *StateDiscoveryService) discoverLocalStates(ctx context.Context) error {
	statePatterns := []string{
		"terraform.tfstate",
		"*.tfstate",
		"terraform.tfstate.backup",
		"*.tfstate.backup",
		".terraform/terraform.tfstate",
		"terraform.tfstate.d/*/terraform.tfstate",
		"**/*.tfstate",
	}
	
	for _, rootPath := range s.rootPaths {
		for _, pattern := range statePatterns {
			if err := s.searchPattern(ctx, rootPath, pattern); err != nil {
				continue // Don't fail on individual pattern errors
			}
		}
		
		// Deep recursive search
		if err := s.deepStateSearch(ctx, rootPath); err != nil {
			continue
		}
	}
	
	return nil
}

// deepStateSearch performs deep recursive search for state files
func (s *StateDiscoveryService) deepStateSearch(ctx context.Context, rootPath string) error {
	visited := make(map[string]bool)
	
	return filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip inaccessible directories
		}
		
		// Skip certain directories
		if d.IsDir() {
			dirName := d.Name()
			if dirName == ".git" || dirName == "node_modules" || dirName == ".idea" || dirName == "vendor" {
				return filepath.SkipDir
			}
			
			// Check depth
			depth := strings.Count(filepath.ToSlash(path), "/") - strings.Count(filepath.ToSlash(rootPath), "/")
			if depth > s.scanDepth {
				return filepath.SkipDir
			}
		}
		
		// Check if it's a state file
		if !d.IsDir() && s.isStateFile(d.Name()) {
			if !visited[path] {
				visited[path] = true
				s.processStateFile(ctx, path)
			}
		}
		
		return nil
	})
}

// isStateFile checks if a file is likely a Terraform state file
func (s *StateDiscoveryService) isStateFile(filename string) bool {
	// Check extensions
	if strings.HasSuffix(filename, ".tfstate") || 
	   strings.HasSuffix(filename, ".tfstate.backup") {
		return true
	}
	
	// Check for timestamped backups
	if matched, _ := regexp.MatchString(`\.tfstate\.\d+\.backup$`, filename); matched {
		return true
	}
	
	// Check for workspace states
	if strings.Contains(filename, "terraform.tfstate") {
		return true
	}
	
	return false
}

// processStateFile processes a discovered state file
func (s *StateDiscoveryService) processStateFile(ctx context.Context, path string) error {
	// Check if already processed
	if _, exists := s.discoveredStates[path]; exists {
		return nil
	}
	
	// Read state file
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	
	// Parse state
	var stateData map[string]interface{}
	if err := json.Unmarshal(data, &stateData); err != nil {
		return err // Not a valid JSON state file
	}
	
	// Validate it's a Terraform state
	if _, ok := stateData["version"]; !ok {
		return nil // Not a Terraform state file
	}
	
	// Extract metadata
	discovered := &DiscoveredState{
		Path:         path,
		Type:         s.determineStateType(path),
		LastModified: s.getFileModTime(path),
	}
	
	// Extract version
	if v, ok := stateData["terraform_version"].(string); ok {
		discovered.TerraformVersion = v
	}
	
	// Count resources
	if resources, ok := stateData["resources"].([]interface{}); ok {
		discovered.ResourceCount = len(resources)
		discovered.Provider = s.detectPrimaryProvider(resources)
	}
	
	// Detect workspace
	discovered.Workspace = s.detectWorkspace(path)
	
	// Detect environment
	discovered.Environment = s.detectEnvironment(path, stateData)
	
	// Extract modules
	discovered.Modules = s.extractModules(stateData)
	
	// Check if partial or orphaned
	discovered.IsPartial = s.isPartialState(stateData)
	discovered.IsOrphaned = s.isOrphanedState(path)
	
	// Find related states
	discovered.RelatedStates = s.findRelatedStates(path)
	
	// Store discovered state
	s.discoveredStates[path] = discovered
	
	// Generate fingerprint
	s.stateFingerprints[path] = s.generateFingerprint(stateData)
	
	return nil
}

// determineStateType determines the type of state file
func (s *StateDiscoveryService) determineStateType(path string) StateType {
	if strings.Contains(path, ".backup") {
		return StateTypeBackup
	}
	if strings.Contains(path, "terraform.tfstate.d") {
		return StateTypeWorkspace
	}
	if strings.Contains(path, ".terragrunt-cache") {
		return StateTypeTerragrunt
	}
	if strings.Contains(path, "modules/") {
		return StateTypeModule
	}
	if strings.Contains(path, ".terraform/") {
		return StateTypeLocal
	}
	return StateTypeLocal
}

// detectPrimaryProvider detects the primary cloud provider
func (s *StateDiscoveryService) detectPrimaryProvider(resources []interface{}) string {
	providerCounts := make(map[string]int)
	
	for _, r := range resources {
		if res, ok := r.(map[string]interface{}); ok {
			if typ, ok := res["type"].(string); ok {
				if strings.HasPrefix(typ, "aws_") {
					providerCounts["aws"]++
				} else if strings.HasPrefix(typ, "azurerm_") {
					providerCounts["azure"]++
				} else if strings.HasPrefix(typ, "google_") {
					providerCounts["gcp"]++
				} else if strings.HasPrefix(typ, "digitalocean_") {
					providerCounts["digitalocean"]++
				}
			}
		}
	}
	
	// Find provider with most resources
	maxProvider := ""
	maxCount := 0
	for provider, count := range providerCounts {
		if count > maxCount {
			maxProvider = provider
			maxCount = count
		}
	}
	
	return maxProvider
}

// detectWorkspace detects the Terraform workspace
func (s *StateDiscoveryService) detectWorkspace(path string) string {
	// Check if in workspace directory
	if strings.Contains(path, "terraform.tfstate.d") {
		parts := strings.Split(filepath.ToSlash(path), "/")
		for i, part := range parts {
			if part == "terraform.tfstate.d" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
	}
	
	// Check for environment markers in path
	pathLower := strings.ToLower(path)
	if strings.Contains(pathLower, "prod") {
		return "production"
	}
	if strings.Contains(pathLower, "staging") || strings.Contains(pathLower, "stage") {
		return "staging"
	}
	if strings.Contains(pathLower, "dev") {
		return "development"
	}
	
	return "default"
}

// detectEnvironment detects the environment from path and state
func (s *StateDiscoveryService) detectEnvironment(path string, stateData map[string]interface{}) string {
	// Check path
	pathLower := strings.ToLower(path)
	envMarkers := map[string]string{
		"prod":    "production",
		"staging": "staging",
		"stage":   "staging",
		"dev":     "development",
		"test":    "testing",
		"qa":      "qa",
	}
	
	for marker, env := range envMarkers {
		if strings.Contains(pathLower, marker) {
			return env
		}
	}
	
	// Check resources for environment tags
	if resources, ok := stateData["resources"].([]interface{}); ok {
		for _, r := range resources {
			if res, ok := r.(map[string]interface{}); ok {
				if instances, ok := res["instances"].([]interface{}); ok {
					for _, inst := range instances {
						if instance, ok := inst.(map[string]interface{}); ok {
							if attrs, ok := instance["attributes"].(map[string]interface{}); ok {
								if tags, ok := attrs["tags"].(map[string]interface{}); ok {
									if env, ok := tags["environment"].(string); ok {
										return env
									}
									if env, ok := tags["env"].(string); ok {
										return env
									}
								}
							}
						}
					}
				}
			}
		}
	}
	
	return "unknown"
}

// extractModules extracts module information from state
func (s *StateDiscoveryService) extractModules(stateData map[string]interface{}) []string {
	modules := []string{}
	seen := make(map[string]bool)
	
	if resources, ok := stateData["resources"].([]interface{}); ok {
		for _, r := range resources {
			if res, ok := r.(map[string]interface{}); ok {
				if module, ok := res["module"].(string); ok && module != "" {
					if !seen[module] {
						seen[module] = true
						modules = append(modules, module)
					}
				}
			}
		}
	}
	
	return modules
}

// isPartialState checks if state is incomplete
func (s *StateDiscoveryService) isPartialState(stateData map[string]interface{}) bool {
	// Check for signs of incomplete state
	if resources, ok := stateData["resources"].([]interface{}); ok {
		for _, r := range resources {
			if res, ok := r.(map[string]interface{}); ok {
				// Check for tainted resources
				if mode, ok := res["mode"].(string); ok && mode == "tainted" {
					return true
				}
				
				// Check for resources without instances
				if instances, ok := res["instances"].([]interface{}); ok && len(instances) == 0 {
					return true
				}
			}
		}
	}
	
	return false
}

// isOrphanedState checks if state file is orphaned
func (s *StateDiscoveryService) isOrphanedState(path string) bool {
	// Check if corresponding .tf files exist
	dir := filepath.Dir(path)
	tfFiles, _ := filepath.Glob(filepath.Join(dir, "*.tf"))
	
	// If no .tf files in same directory, might be orphaned
	if len(tfFiles) == 0 {
		// Check parent directory
		parentDir := filepath.Dir(dir)
		tfFiles, _ = filepath.Glob(filepath.Join(parentDir, "*.tf"))
		if len(tfFiles) == 0 {
			return true
		}
	}
	
	return false
}

// findRelatedStates finds related state files
func (s *StateDiscoveryService) findRelatedStates(path string) []string {
	related := []string{}
	dir := filepath.Dir(path)
	
	// Find other state files in same directory
	stateFiles, _ := filepath.Glob(filepath.Join(dir, "*.tfstate*"))
	for _, file := range stateFiles {
		if file != path {
			related = append(related, file)
		}
	}
	
	// Check for workspace states
	workspaceDir := filepath.Join(dir, "terraform.tfstate.d")
	if info, err := os.Stat(workspaceDir); err == nil && info.IsDir() {
		workspaceStates, _ := filepath.Glob(filepath.Join(workspaceDir, "*", "terraform.tfstate"))
		related = append(related, workspaceStates...)
	}
	
	return related
}

// generateFingerprint generates a fingerprint for the state
func (s *StateDiscoveryService) generateFingerprint(stateData map[string]interface{}) StateFingerprint {
	fingerprint := StateFingerprint{
		TagPatterns:      make(map[string]string),
		ResourcePatterns: []string{},
		ModulePatterns:   []string{},
		ProviderVersions: make(map[string]string),
	}
	
	// Analyze naming patterns
	resourceNames := []string{}
	if resources, ok := stateData["resources"].([]interface{}); ok {
		for _, r := range resources {
			if res, ok := r.(map[string]interface{}); ok {
				if name, ok := res["name"].(string); ok {
					resourceNames = append(resourceNames, name)
				}
			}
		}
	}
	
	fingerprint.NamingConvention = s.detectNamingConvention(resourceNames)
	
	// Analyze tag patterns
	fingerprint.TagPatterns = s.analyzeTagPatterns(stateData)
	
	// Detect creation patterns
	fingerprint.CreationPatterns = s.detectCreationPatterns(stateData)
	
	return fingerprint
}

// detectNamingConvention detects the naming convention used
func (s *StateDiscoveryService) detectNamingConvention(names []string) string {
	if len(names) == 0 {
		return "unknown"
	}
	
	// Check for common patterns
	patterns := map[string]*regexp.Regexp{
		"kebab-case":     regexp.MustCompile(`^[a-z]+(-[a-z]+)*$`),
		"snake_case":     regexp.MustCompile(`^[a-z]+(_[a-z]+)*$`),
		"env-prefixed":   regexp.MustCompile(`^(dev|staging|prod|test)-`),
		"service-based":  regexp.MustCompile(`^[a-z]+-[a-z]+-[a-z]+$`),
	}
	
	patternCounts := make(map[string]int)
	for _, name := range names {
		for patternName, regex := range patterns {
			if regex.MatchString(strings.ToLower(name)) {
				patternCounts[patternName]++
			}
		}
	}
	
	// Find most common pattern
	maxPattern := "mixed"
	maxCount := 0
	for pattern, count := range patternCounts {
		if count > maxCount {
			maxPattern = pattern
			maxCount = count
		}
	}
	
	return maxPattern
}

// analyzeTagPatterns analyzes tagging patterns in state
func (s *StateDiscoveryService) analyzeTagPatterns(stateData map[string]interface{}) map[string]string {
	tagPatterns := make(map[string]int)
	
	if resources, ok := stateData["resources"].([]interface{}); ok {
		for _, r := range resources {
			if res, ok := r.(map[string]interface{}); ok {
				if instances, ok := res["instances"].([]interface{}); ok {
					for _, inst := range instances {
						if instance, ok := inst.(map[string]interface{}); ok {
							if attrs, ok := instance["attributes"].(map[string]interface{}); ok {
								if tags, ok := attrs["tags"].(map[string]interface{}); ok {
									for tagKey := range tags {
										tagPatterns[tagKey]++
									}
								}
							}
						}
					}
				}
			}
		}
	}
	
	// Determine consistent tags
	consistentTags := make(map[string]string)
	for tag, count := range tagPatterns {
		if count > 1 {
			consistentTags[tag] = "consistent"
		}
	}
	
	return consistentTags
}

// detectCreationPatterns detects how resources were created
func (s *StateDiscoveryService) detectCreationPatterns(stateData map[string]interface{}) CreationPattern {
	pattern := CreationPattern{}
	
	// Check for import markers
	if resources, ok := stateData["resources"].([]interface{}); ok {
		importCount := 0
		totalCount := len(resources)
		
		for _, r := range resources {
			if res, ok := r.(map[string]interface{}); ok {
				// Check for import metadata (would be in resource metadata)
				if _, ok := res["imported"]; ok {
					importCount++
				}
			}
		}
		
		if importCount > totalCount/2 {
			pattern.IsImported = true
		}
	}
	
	// Check for consistent naming
	pattern.HasConsistentNaming = s.hasConsistentNaming(stateData)
	
	// Check for automation tags
	pattern.HasAutomationTags = s.hasAutomationTags(stateData)
	
	// Determine if Terraform-created
	pattern.IsTerraform = pattern.HasConsistentNaming && pattern.HasAutomationTags
	pattern.IsManual = !pattern.IsTerraform && !pattern.IsImported
	
	return pattern
}

// hasConsistentNaming checks for consistent resource naming
func (s *StateDiscoveryService) hasConsistentNaming(stateData map[string]interface{}) bool {
	names := []string{}
	
	if resources, ok := stateData["resources"].([]interface{}); ok {
		for _, r := range resources {
			if res, ok := r.(map[string]interface{}); ok {
				if name, ok := res["name"].(string); ok {
					names = append(names, name)
				}
			}
		}
	}
	
	if len(names) < 2 {
		return true // Not enough to determine
	}
	
	// Check if names follow similar pattern
	convention := s.detectNamingConvention(names)
	return convention != "mixed" && convention != "unknown"
}

// hasAutomationTags checks for automation-related tags
func (s *StateDiscoveryService) hasAutomationTags(stateData map[string]interface{}) bool {
	automationTags := []string{
		"managed_by",
		"terraform",
		"created_by",
		"automation",
		"iac",
		"provisioner",
	}
	
	if resources, ok := stateData["resources"].([]interface{}); ok {
		for _, r := range resources {
			if res, ok := r.(map[string]interface{}); ok {
				if instances, ok := res["instances"].([]interface{}); ok {
					for _, inst := range instances {
						if instance, ok := inst.(map[string]interface{}); ok {
							if attrs, ok := instance["attributes"].(map[string]interface{}); ok {
								if tags, ok := attrs["tags"].(map[string]interface{}); ok {
									for _, autoTag := range automationTags {
										if _, exists := tags[autoTag]; exists {
											return true
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	
	return false
}

// getFileModTime gets file modification time
func (s *StateDiscoveryService) getFileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// searchPattern searches for files matching pattern
func (s *StateDiscoveryService) searchPattern(ctx context.Context, rootPath, pattern string) error {
	matches, err := filepath.Glob(filepath.Join(rootPath, pattern))
	if err != nil {
		return err
	}
	
	for _, match := range matches {
		s.processStateFile(ctx, match)
	}
	
	return nil
}

// discoverRemoteStates discovers states in remote backends
func (s *StateDiscoveryService) discoverRemoteStates(ctx context.Context) error {
	// This would integrate with cloud storage APIs
	// For now, we'll look for backend configurations
	
	for _, rootPath := range s.rootPaths {
		backendConfigs := s.findBackendConfigurations(rootPath)
		for _, config := range backendConfigs {
			s.backendConfigs = append(s.backendConfigs, config)
			// Would connect to remote backend and list states
		}
	}
	
	return nil
}

// findBackendConfigurations finds Terraform backend configurations
func (s *StateDiscoveryService) findBackendConfigurations(rootPath string) []BackendConfig {
	configs := []BackendConfig{}
	
	// Search for .tf files with backend configurations
	tfFiles, _ := filepath.Glob(filepath.Join(rootPath, "**/*.tf"))
	for _, tfFile := range tfFiles {
		if config := s.extractBackendConfig(tfFile); config != nil {
			configs = append(configs, *config)
		}
	}
	
	return configs
}

// extractBackendConfig extracts backend configuration from .tf file
func (s *StateDiscoveryService) extractBackendConfig(tfFile string) *BackendConfig {
	data, err := os.ReadFile(tfFile)
	if err != nil {
		return nil
	}
	
	content := string(data)
	
	// Simple regex to find backend blocks (would use HCL parser in production)
	if strings.Contains(content, "backend \"s3\"") {
		return &BackendConfig{Type: BackendTypeS3}
	}
	if strings.Contains(content, "backend \"azurerm\"") {
		return &BackendConfig{Type: BackendTypeAzure}
	}
	if strings.Contains(content, "backend \"gcs\"") {
		return &BackendConfig{Type: BackendTypeGCS}
	}
	
	return nil
}

// discoverCIArtifacts discovers states in CI/CD artifacts
func (s *StateDiscoveryService) discoverCIArtifacts(ctx context.Context) error {
	ciDirs := []string{
		".github",
		".gitlab",
		".circleci",
		"jenkins",
		".azure-pipelines",
		"buildkite",
	}
	
	for _, rootPath := range s.rootPaths {
		for _, ciDir := range ciDirs {
			ciPath := filepath.Join(rootPath, ciDir)
			if info, err := os.Stat(ciPath); err == nil && info.IsDir() {
				s.searchCIArtifacts(ctx, ciPath)
			}
		}
	}
	
	return nil
}

// searchCIArtifacts searches CI directories for state artifacts
func (s *StateDiscoveryService) searchCIArtifacts(ctx context.Context, ciPath string) {
	filepath.WalkDir(ciPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		
		if !d.IsDir() && s.isStateFile(d.Name()) {
			s.processStateFile(ctx, path)
		}
		
		return nil
	})
}

// discoverFromLockTables discovers states from DynamoDB/CosmosDB lock tables
func (s *StateDiscoveryService) discoverFromLockTables(ctx context.Context) error {
	// This would connect to DynamoDB/CosmosDB and query lock tables
	// For now, we'll look for lock table references in configurations
	
	return nil
}

// discoverTerragruntStates discovers Terragrunt-managed states
func (s *StateDiscoveryService) discoverTerragruntStates(ctx context.Context) error {
	for _, rootPath := range s.rootPaths {
		// Find terragrunt.hcl files
		terragruntFiles, _ := filepath.Glob(filepath.Join(rootPath, "**/terragrunt.hcl"))
		for _, tgFile := range terragruntFiles {
			dir := filepath.Dir(tgFile)
			
			// Check for .terragrunt-cache
			cacheDir := filepath.Join(dir, ".terragrunt-cache")
			if info, err := os.Stat(cacheDir); err == nil && info.IsDir() {
				s.searchTerragruntCache(ctx, cacheDir)
			}
		}
	}
	
	return nil
}

// searchTerragruntCache searches Terragrunt cache for states
func (s *StateDiscoveryService) searchTerragruntCache(ctx context.Context, cacheDir string) {
	filepath.WalkDir(cacheDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		
		if !d.IsDir() && s.isStateFile(d.Name()) {
			s.processStateFile(ctx, path)
		}
		
		return nil
	})
}

// crossReferenceStates cross-references discovered states
func (s *StateDiscoveryService) crossReferenceStates() {
	// Build relationships between states
	for path1, state1 := range s.discoveredStates {
		for path2, state2 := range s.discoveredStates {
			if path1 != path2 {
				// Check if states are related
				if s.areStatesRelated(state1, state2) {
					state1.RelatedStates = append(state1.RelatedStates, path2)
				}
			}
		}
	}
}

// areStatesRelated checks if two states are related
func (s *StateDiscoveryService) areStatesRelated(state1, state2 *DiscoveredState) bool {
	// Same workspace
	if state1.Workspace == state2.Workspace && state1.Workspace != "default" {
		return true
	}
	
	// Same environment
	if state1.Environment == state2.Environment && state1.Environment != "unknown" {
		return true
	}
	
	// Share modules
	for _, mod1 := range state1.Modules {
		for _, mod2 := range state2.Modules {
			if mod1 == mod2 {
				return true
			}
		}
	}
	
	// Same directory hierarchy
	dir1 := filepath.Dir(state1.Path)
	dir2 := filepath.Dir(state2.Path)
	if dir1 == dir2 || strings.HasPrefix(dir1, dir2) || strings.HasPrefix(dir2, dir1) {
		return true
	}
	
	return false
}

// GetAllDiscoveredStates returns all discovered states
func (s *StateDiscoveryService) GetAllDiscoveredStates() map[string]*DiscoveredState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.discoveredStates
}

// GetStateFingerprints returns state fingerprints
func (s *StateDiscoveryService) GetStateFingerprints() map[string]StateFingerprint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stateFingerprints
}