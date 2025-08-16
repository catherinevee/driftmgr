package terragrunt

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// TerragruntParser provides functionality to parse and manage Terragrunt configurations
type TerragruntParser struct {
	rootPath string
}

// NewTerragruntParser creates a new Terragrunt parser
func NewTerragruntParser(rootPath string) *TerragruntParser {
	return &TerragruntParser{
		rootPath: rootPath,
	}
}

// FindTerragruntFiles discovers all Terragrunt configuration files in the workspace
func (tp *TerragruntParser) FindTerragruntFiles() (*models.TerragruntDiscoveryResult, error) {
	var rootFiles []models.TerragruntFile
	var childFiles []models.TerragruntFile
	var environments []string
	var regions []string
	var accounts []string

	// Common Terragrunt directory patterns
	searchPaths := []string{
		tp.rootPath,
		filepath.Join(tp.rootPath, "terragrunt"),
		filepath.Join(tp.rootPath, "infrastructure"),
		filepath.Join(tp.rootPath, "iac"),
		filepath.Join(tp.rootPath, "environments"),
		filepath.Join(tp.rootPath, "stacks"),
		filepath.Join(tp.rootPath, "modules"),
		filepath.Join(tp.rootPath, "examples"),
	}

	log.Printf("Searching for Terragrunt files in paths: %v", searchPaths)

	for _, searchPath := range searchPaths {
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			log.Printf("Path does not exist: %s", searchPath)
			continue
		}

		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Printf("Error accessing path %s: %v", path, err)
				return nil
			}

			// Look for terragrunt.hcl files
			if !info.IsDir() && (info.Name() == "terragrunt.hcl" || strings.HasSuffix(info.Name(), ".hcl")) {
				// Skip .terraform.lock.hcl files as they're not Terragrunt configs
				if strings.Contains(info.Name(), ".terraform.lock.hcl") {
					return nil
				}

				log.Printf("Found Terragrunt file: %s", path)
				
				terragruntFile, err := tp.parseTerragruntFile(path)
				if err != nil {
					log.Printf("Warning: Failed to parse Terragrunt file %s: %v", path, err)
					return nil
				}

				// Determine if this is a root or child file
				if tp.isRootTerragruntFile(path) {
					rootFiles = append(rootFiles, *terragruntFile)
				} else {
					childFiles = append(childFiles, *terragruntFile)
				}

				// Extract environment, region, and account information
				if terragruntFile.Environment != "" {
					environments = appendIfNotExists(environments, terragruntFile.Environment)
				}
				if terragruntFile.Region != "" {
					regions = appendIfNotExists(regions, terragruntFile.Region)
				}
				if terragruntFile.Account != "" {
					accounts = appendIfNotExists(accounts, terragruntFile.Account)
				}
			}

			return nil
		})

		if err != nil {
			log.Printf("Warning: Error walking path %s: %v", searchPath, err)
		}
	}

	// Build parent-child relationships
	tp.buildTerragruntHierarchy(&rootFiles, &childFiles)

	result := &models.TerragruntDiscoveryResult{
		RootFiles:    rootFiles,
		ChildFiles:   childFiles,
		TotalFiles:   len(rootFiles) + len(childFiles),
		Environments: environments,
		Regions:      regions,
		Accounts:     accounts,
		Timestamp:    time.Now(),
	}

	log.Printf("Terragrunt discovery complete: found %d root files, %d child files", len(rootFiles), len(childFiles))
	return result, nil
}

// parseTerragruntFile parses a single Terragrunt configuration file
func (tp *TerragruntParser) parseTerragruntFile(filePath string) (*models.TerragruntFile, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Terragrunt file: %v", err)
	}

	config, err := tp.parseTerragruntConfig(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Terragrunt config: %v", err)
	}

	config.Path = filePath

	// Extract metadata from file path and configuration
	environment := tp.extractEnvironmentFromPath(filePath)
	region := tp.extractRegionFromPath(filePath)
	account := tp.extractAccountFromPath(filePath)
	moduleName := tp.extractModuleNameFromPath(filePath)

	// Try to extract additional information from config
	if region == "" && config.Inputs != nil {
		if regionVal, ok := config.Inputs["region"]; ok {
			if regionStr, ok := regionVal.(string); ok {
				region = regionStr
			}
		}
	}

	if environment == "" && config.Inputs != nil {
		if envVal, ok := config.Inputs["environment"]; ok {
			if envStr, ok := envVal.(string); ok {
				environment = envStr
			}
		}
	}

	terragruntFile := &models.TerragruntFile{
		Path:        filePath,
		Config:      config,
		IsRoot:      tp.isRootTerragruntFile(filePath),
		Environment: environment,
		Region:      region,
		Account:     account,
		ModuleName:  moduleName,
	}

	return terragruntFile, nil
}

// parseTerragruntConfig parses the content of a Terragrunt configuration file
func (tp *TerragruntParser) parseTerragruntConfig(content string) (*models.TerragruntConfig, error) {
	config := &models.TerragruntConfig{}

	// Extract source
	if source := tp.extractSource(content); source != "" {
		config.Source = source
	}

	// Extract inputs
	if inputs := tp.extractInputs(content); inputs != nil {
		config.Inputs = inputs
	}

	// Extract includes
	if includes := tp.extractIncludes(content); len(includes) > 0 {
		config.Include = includes
	}

	// Extract generate blocks
	if generates := tp.extractGenerates(content); len(generates) > 0 {
		config.Generate = generates
	}

	// Extract remote state configuration
	if remoteState := tp.extractRemoteState(content); remoteState != nil {
		config.RemoteState = remoteState
	}

	// Extract dependencies
	if dependencies := tp.extractDependencies(content); len(dependencies) > 0 {
		config.Dependencies = dependencies
	}

	// Extract hooks
	if beforeHooks := tp.extractHooks(content, "before_hook"); len(beforeHooks) > 0 {
		config.BeforeHooks = beforeHooks
	}
	if afterHooks := tp.extractHooks(content, "after_hook"); len(afterHooks) > 0 {
		config.AfterHooks = afterHooks
	}
	if errorHooks := tp.extractHooks(content, "error_hook"); len(errorHooks) > 0 {
		config.ErrorHooks = errorHooks
	}

	// Extract other configuration options
	config.TerragruntVersion = tp.extractTerragruntVersion(content)
	config.DownloadDir = tp.extractDownloadDir(content)
	config.PreventDestroy = tp.extractPreventDestroy(content)
	config.Skip = tp.extractSkip(content)
	config.IamRole = tp.extractIamRole(content)

	return config, nil
}

// extractSource extracts the source from Terragrunt configuration
func (tp *TerragruntParser) extractSource(content string) string {
	re := regexp.MustCompile(`terraform\s*\{\s*source\s*=\s*["']([^"']+)["']`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractInputs extracts inputs from Terragrunt configuration
func (tp *TerragruntParser) extractInputs(content string) map[string]interface{} {
	inputs := make(map[string]interface{})
	
	// Simple regex-based extraction for common input patterns
	re := regexp.MustCompile(`(\w+)\s*=\s*["']([^"']+)["']`)
	matches := re.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) > 2 {
			key := strings.TrimSpace(match[1])
			value := strings.TrimSpace(match[2])
			inputs[key] = value
		}
	}
	
	return inputs
}

// extractIncludes extracts include blocks from Terragrunt configuration
func (tp *TerragruntParser) extractIncludes(content string) []models.TerragruntInclude {
	var includes []models.TerragruntInclude
	
	re := regexp.MustCompile(`include\s*"([^"]+)"\s*\{\s*path\s*=\s*["']([^"']+)["']`)
	matches := re.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) > 2 {
			include := models.TerragruntInclude{
				Name: strings.TrimSpace(match[1]),
				Path: strings.TrimSpace(match[2]),
			}
			includes = append(includes, include)
		}
	}
	
	return includes
}

// extractGenerates extracts generate blocks from Terragrunt configuration
func (tp *TerragruntParser) extractGenerates(content string) []models.TerragruntGenerate {
	var generates []models.TerragruntGenerate
	
	re := regexp.MustCompile(`generate\s*"([^"]+)"\s*\{\s*path\s*=\s*["']([^"']+)["']`)
	matches := re.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) > 2 {
			generate := models.TerragruntGenerate{
				Path: strings.TrimSpace(match[2]),
			}
			generates = append(generates, generate)
		}
	}
	
	return generates
}

// extractRemoteState extracts remote state configuration
func (tp *TerragruntParser) extractRemoteState(content string) *models.TerragruntRemoteState {
	re := regexp.MustCompile(`remote_state\s*\{\s*backend\s*=\s*["']([^"']+)["']`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return &models.TerragruntRemoteState{
			Backend: strings.TrimSpace(matches[1]),
		}
	}
	return nil
}

// extractDependencies extracts dependencies from Terragrunt configuration
func (tp *TerragruntParser) extractDependencies(content string) []string {
	var dependencies []string
	
	re := regexp.MustCompile(`dependency\s*"([^"]+)"\s*\{`)
	matches := re.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			dependencies = append(dependencies, strings.TrimSpace(match[1]))
		}
	}
	
	return dependencies
}

// extractHooks extracts hooks from Terragrunt configuration
func (tp *TerragruntParser) extractHooks(content, hookType string) []models.TerragruntHook {
	var hooks []models.TerragruntHook
	
	re := regexp.MustCompile(fmt.Sprintf(`%s\s*"([^"]+)"\s*\{`, hookType))
	matches := re.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			hook := models.TerragruntHook{
				Name: strings.TrimSpace(match[1]),
			}
			hooks = append(hooks, hook)
		}
	}
	
	return hooks
}

// extractTerragruntVersion extracts Terragrunt version requirement
func (tp *TerragruntParser) extractTerragruntVersion(content string) string {
	re := regexp.MustCompile(`terragrunt_version\s*=\s*["']([^"']+)["']`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractDownloadDir extracts download directory
func (tp *TerragruntParser) extractDownloadDir(content string) string {
	re := regexp.MustCompile(`download_dir\s*=\s*["']([^"']+)["']`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractPreventDestroy extracts prevent_destroy setting
func (tp *TerragruntParser) extractPreventDestroy(content string) bool {
	re := regexp.MustCompile(`prevent_destroy\s*=\s*true`)
	return re.MatchString(content)
}

// extractSkip extracts skip setting
func (tp *TerragruntParser) extractSkip(content string) bool {
	re := regexp.MustCompile(`skip\s*=\s*true`)
	return re.MatchString(content)
}

// extractIamRole extracts IAM role
func (tp *TerragruntParser) extractIamRole(content string) string {
	re := regexp.MustCompile(`iam_role\s*=\s*["']([^"']+)["']`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// isRootTerragruntFile determines if a Terragrunt file is a root configuration
func (tp *TerragruntParser) isRootTerragruntFile(filePath string) bool {
	// Root files are typically in the root directory or have specific patterns
	dir := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)
	
	// Check if it's in the root directory
	if dir == tp.rootPath {
		return true
	}
	
	// Check for common root file patterns
	rootPatterns := []string{
		"terragrunt.hcl",
		"root.hcl",
		"common.hcl",
		"account.hcl",
		"env.hcl",
	}
	
	for _, pattern := range rootPatterns {
		if fileName == pattern {
			return true
		}
	}
	
	return false
}

// buildTerragruntHierarchy builds parent-child relationships between Terragrunt files
func (tp *TerragruntParser) buildTerragruntHierarchy(rootFiles *[]models.TerragruntFile, childFiles *[]models.TerragruntFile) {
	// This is a simplified implementation
	// In a full implementation, you would parse the include blocks and build the actual hierarchy
	for i := range *rootFiles {
		rootFile := &(*rootFiles)[i]
		for j := range *childFiles {
			childFile := &(*childFiles)[j]
			
			// Check if child file includes the root file
			if tp.fileIncludesRoot(childFile, rootFile) {
				childFile.ParentPath = rootFile.Path
				rootFile.ChildPaths = append(rootFile.ChildPaths, childFile.Path)
			}
		}
	}
}

// fileIncludesRoot checks if a child file includes a root file
func (tp *TerragruntParser) fileIncludesRoot(childFile *models.TerragruntFile, rootFile *models.TerragruntFile) bool {
	// Simplified check - in reality, you'd parse the include blocks
	childDir := filepath.Dir(childFile.Path)
	rootDir := filepath.Dir(rootFile.Path)
	
	// Check if child is in a subdirectory of root
	return strings.HasPrefix(childDir, rootDir) && childDir != rootDir
}

// extractEnvironmentFromPath extracts environment from file path
func (tp *TerragruntParser) extractEnvironmentFromPath(filePath string) string {
	pathParts := strings.Split(filePath, string(os.PathSeparator))
	
	// Look for common environment patterns in path
	envPatterns := []string{"dev", "development", "staging", "prod", "production", "test", "qa"}
	
	for _, part := range pathParts {
		for _, pattern := range envPatterns {
			if strings.Contains(strings.ToLower(part), pattern) {
				return part
			}
		}
	}
	
	return ""
}

// extractRegionFromPath extracts region from file path
func (tp *TerragruntParser) extractRegionFromPath(filePath string) string {
	pathParts := strings.Split(filePath, string(os.PathSeparator))
	
	// Look for AWS region patterns in path
	regionPatterns := []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-central-1", "eu-north-1",
		"ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-northeast-2",
		"sa-east-1", "ca-central-1", "af-south-1", "me-south-1",
	}
	
	for _, part := range pathParts {
		for _, pattern := range regionPatterns {
			if strings.Contains(part, pattern) {
				return part
			}
		}
	}
	
	return ""
}

// extractAccountFromPath extracts account from file path
func (tp *TerragruntParser) extractAccountFromPath(filePath string) string {
	pathParts := strings.Split(filePath, string(os.PathSeparator))
	
	// Look for account patterns (typically numeric)
	accountRe := regexp.MustCompile(`^\d{12}$`)
	
	for _, part := range pathParts {
		if accountRe.MatchString(part) {
			return part
		}
	}
	
	return ""
}

// extractModuleNameFromPath extracts module name from file path
func (tp *TerragruntParser) extractModuleNameFromPath(filePath string) string {
	pathParts := strings.Split(filePath, string(os.PathSeparator))
	
	// Look for common module patterns
	modulePatterns := []string{"vpc", "ec2", "rds", "eks", "s3", "lambda", "alb", "nlb"}
	
	for _, part := range pathParts {
		for _, pattern := range modulePatterns {
			if strings.Contains(strings.ToLower(part), pattern) {
				return part
			}
		}
	}
	
	return ""
}

// appendIfNotExists appends a string to a slice if it doesn't already exist
func appendIfNotExists(slice []string, item string) []string {
	for _, existing := range slice {
		if existing == item {
			return slice
		}
	}
	return append(slice, item)
}

// FindTerragruntStateFiles finds Terragrunt-managed state files
func (tp *TerragruntParser) FindTerragruntStateFiles() []string {
	var stateFiles []string
	
	// Look for state files in common Terragrunt locations
	searchPaths := []string{
		filepath.Join(tp.rootPath, ".terragrunt-cache"),
		filepath.Join(tp.rootPath, "terragrunt"),
		filepath.Join(tp.rootPath, "infrastructure"),
		filepath.Join(tp.rootPath, "iac"),
	}
	
	for _, searchPath := range searchPaths {
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			continue
		}
		
		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".tfstate") {
				stateFiles = append(stateFiles, path)
			}
			
			return nil
		})
		
		if err != nil {
			log.Printf("Warning: Error walking path %s: %v", searchPath, err)
		}
	}
	
	return stateFiles
}
