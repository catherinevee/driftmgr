package terraform

import (
	"os"
	"path/filepath"
	"strings"
)

// Backend represents a Terraform backend configuration
type Backend struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// Config represents Terraform configuration
type Config struct {
	TerraformVersion string                 `json:"terraform_version"`
	Backend          *Backend               `json:"backend,omitempty"`
	Providers        map[string]interface{} `json:"providers"`
	Resources        []Resource             `json:"resources"`
}

// Resource represents a Terraform resource
type Resource struct {
	Type      string             `json:"type"`
	Name      string             `json:"name"`
	Provider  string             `json:"provider"`
	Instances []ResourceInstance `json:"instances"`
}

// ResourceInstance represents an instance of a Terraform resource
type ResourceInstance struct {
	Attributes map[string]interface{} `json:"attributes"`
	IndexKey   interface{}            `json:"index_key,omitempty"`
}

// FindStateFiles finds all Terraform state files in a directory
func FindStateFiles(rootPath string) ([]string, error) {
	var stateFiles []string

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}

		if info.IsDir() {
			// Skip hidden directories
			if filepath.Base(path)[0] == '.' && path != rootPath {
				return filepath.SkipDir
			}
			return nil
		}

		// Check for state files
		name := filepath.Base(path)
		if name == "terraform.tfstate" ||
			name == "terraform.tfstate.backup" ||
			filepath.Ext(name) == ".tfstate" {
			stateFiles = append(stateFiles, path)
		}

		return nil
	})

	return stateFiles, err
}

// DetectBackend detects Terraform backend configuration from files
func DetectBackend(dir string) (*Backend, error) {
	// Look for backend configuration in .tf files
	tfFiles, err := filepath.Glob(filepath.Join(dir, "*.tf"))
	if err != nil {
		return nil, err
	}

	for _, tfFile := range tfFiles {
		data, err := os.ReadFile(tfFile)
		if err != nil {
			continue
		}

		// Simple detection - look for backend configuration
		content := string(data)
		if strings.Contains(content, "backend \"s3\"") {
			return &Backend{Type: "s3", Config: make(map[string]interface{})}, nil
		}
		if strings.Contains(content, "backend \"azurerm\"") {
			return &Backend{Type: "azurerm", Config: make(map[string]interface{})}, nil
		}
		if strings.Contains(content, "backend \"gcs\"") {
			return &Backend{Type: "gcs", Config: make(map[string]interface{})}, nil
		}
		if strings.Contains(content, "backend \"remote\"") {
			return &Backend{Type: "remote", Config: make(map[string]interface{})}, nil
		}
	}

	return &Backend{Type: "local", Config: make(map[string]interface{})}, nil
}

// LoadConfig loads Terraform configuration from a directory
func LoadConfig(dir string) (*Config, error) {
	config := &Config{
		Providers: make(map[string]interface{}),
		Resources: []Resource{},
	}

	// Detect backend
	backend, err := DetectBackend(dir)
	if err == nil {
		config.Backend = backend
	}

	return config, nil
}
