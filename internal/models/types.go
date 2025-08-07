package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Resource represents a cloud resource that can be imported into Terraform
type Resource struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Provider      string                 `json:"provider"`
	Region        string                 `json:"region"`
	Tags          map[string]string      `json:"tags"`
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
	ImportID      string                 `json:"import_id"`      // ID used for terraform import
	TerraformType string                 `json:"terraform_type"` // Terraform resource type
}

// ImportCommand represents a terraform import command
type ImportCommand struct {
	ResourceType  string   `json:"resource_type"`
	ResourceName  string   `json:"resource_name"`
	ResourceID    string   `json:"resource_id"`
	Configuration string   `json:"configuration"`
	Dependencies  []string `json:"dependencies"`
	Command       string   `json:"command"`
}

// ImportResult contains the results of an import operation
type ImportResult struct {
	Successful int             `json:"successful"`
	Failed     int             `json:"failed"`
	Errors     []ImportError   `json:"errors"`
	Commands   []ImportCommand `json:"commands"`
	Duration   time.Duration   `json:"duration"`
}

// ImportError represents an error that occurred during import
type ImportError struct {
	Resource string `json:"resource"`
	Error    string `json:"error"`
	Code     string `json:"code"`
}

// DiscoveryStats contains statistics about resource discovery
type DiscoveryStats struct {
	TotalResources int            `json:"total_resources"`
	ByProvider     map[string]int `json:"by_provider"`
	ByRegion       map[string]int `json:"by_region"`
	ByType         map[string]int `json:"by_type"`
	Duration       time.Duration  `json:"duration"`
}

// StateFileInfo contains information about a Terraform state file
type StateFileInfo struct {
	Path         string    `json:"path"`
	Version      int       `json:"version"`
	Resources    int       `json:"resources"`
	LastModified time.Time `json:"last_modified"`
	Size         int64     `json:"size"`
}

// DiscoveryConfig represents configuration for resource discovery
type DiscoveryConfig struct {
	Provider string   `json:"provider"`
	Regions  []string `json:"regions"`
}

// ImportConfig represents configuration for importing resources
type ImportConfig struct {
	Resources  []Resource `json:"resources"`
	OutputPath string     `json:"output_path"`
	Format     string     `json:"format"`
}

// Validate validates the Resource struct
func (r *Resource) Validate() error {
	if r.ID == "" {
		return errors.New("resource ID is required")
	}
	if r.Name == "" {
		return errors.New("resource name is required")
	}
	if r.Type == "" {
		return errors.New("resource type is required")
	}
	if r.Provider == "" {
		return errors.New("resource provider is required")
	}
	if r.Region == "" {
		return errors.New("resource region is required")
	}
	return nil
}

// String returns a string representation of the resource
func (r *Resource) String() string {
	return fmt.Sprintf("Resource{ID: %s, Name: %s, Type: %s, Provider: %s, Region: %s}",
		r.ID, r.Name, r.Type, r.Provider, r.Region)
}

// GetTerraformAddress returns the Terraform address for this resource
func (r *Resource) GetTerraformAddress() string {
	normalizedName := r.normalizeName()
	return fmt.Sprintf("%s.%s", r.Type, normalizedName)
}

// normalizeName normalizes the resource name for use in Terraform
func (r *Resource) normalizeName() string {
	// Convert to lowercase
	name := strings.ToLower(r.Name)

	// Replace non-alphanumeric characters with underscores
	re := regexp.MustCompile("[^a-z0-9_]")
	name = re.ReplaceAllString(name, "_")

	// Ensure it doesn't start with a number
	if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
		name = "_" + name
	}

	return name
}

// Validate validates the DiscoveryConfig struct
func (dc *DiscoveryConfig) Validate() error {
	if dc.Provider == "" {
		return errors.New("provider is required")
	}

	validProviders := map[string]bool{
		"aws":   true,
		"azure": true,
		"gcp":   true,
	}

	if !validProviders[dc.Provider] {
		return fmt.Errorf("invalid provider: %s", dc.Provider)
	}

	if len(dc.Regions) == 0 {
		return errors.New("at least one region is required")
	}

	return nil
}

// String returns a string representation of the discovery config
func (dc *DiscoveryConfig) String() string {
	return fmt.Sprintf("DiscoveryConfig{Provider: %s, Regions: %v}",
		dc.Provider, dc.Regions)
}

// Validate validates the ImportConfig struct
func (ic *ImportConfig) Validate() error {
	if len(ic.Resources) == 0 {
		return errors.New("at least one resource is required")
	}

	for _, resource := range ic.Resources {
		if err := resource.Validate(); err != nil {
			return fmt.Errorf("invalid resource: %w", err)
		}
	}

	if ic.OutputPath == "" {
		return errors.New("output path is required")
	}

	if ic.Format == "" {
		return errors.New("format is required")
	}

	return nil
}

// String returns a string representation of the import config
func (ic *ImportConfig) String() string {
	return fmt.Sprintf("ImportConfig{Resources: %d resources, OutputPath: %s, Format: %s}",
		len(ic.Resources), ic.OutputPath, ic.Format)
}
