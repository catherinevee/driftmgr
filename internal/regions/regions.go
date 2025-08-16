package regions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Region represents a cloud region
type Region struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Provider    string `json:"provider"`
}

// RegionManager handles provider-specific region operations
type RegionManager struct {
	regions map[string][]Region // provider -> regions
}

// NewRegionManager creates a new region manager
func NewRegionManager() (*RegionManager, error) {
	rm := &RegionManager{
		regions: make(map[string][]Region),
	}

	// Load regions for each provider
	providers := []string{"aws", "azure", "gcp", "digitalocean"}
	for _, provider := range providers {
		if err := rm.loadRegions(provider); err != nil {
			return nil, fmt.Errorf("failed to load regions for %s: %w", provider, err)
		}
	}

	return rm, nil
}

// loadRegions loads regions from the provider-specific JSON file
func (rm *RegionManager) loadRegions(provider string) error {
	// Get the current working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Construct the path to the region file
	regionFile := filepath.Join(wd, fmt.Sprintf("%s_regions.json", provider))

	// Read the region file
	data, err := os.ReadFile(regionFile)
	if err != nil {
		return fmt.Errorf("failed to read region file %s: %w", regionFile, err)
	}

	// Parse the JSON
	var regions []Region
	if err := json.Unmarshal(data, &regions); err != nil {
		return fmt.Errorf("failed to parse region file %s: %w", regionFile, err)
	}

	// Store the regions
	rm.regions[provider] = regions

	return nil
}

// GetRegions returns all regions for a provider
func (rm *RegionManager) GetRegions(provider string) ([]Region, error) {
	regions, exists := rm.regions[provider]
	if !exists {
		return nil, fmt.Errorf("provider %s not supported", provider)
	}
	return regions, nil
}

// GetEnabledRegions returns only enabled regions for a provider
func (rm *RegionManager) GetEnabledRegions(provider string) ([]Region, error) {
	regions, err := rm.GetRegions(provider)
	if err != nil {
		return nil, err
	}

	var enabledRegions []Region
	for _, region := range regions {
		if region.Enabled {
			enabledRegions = append(enabledRegions, region)
		}
	}

	return enabledRegions, nil
}

// ValidateRegion validates if a region is valid for a specific provider
func (rm *RegionManager) ValidateRegion(provider, regionName string) (bool, error) {
	regions, err := rm.GetRegions(provider)
	if err != nil {
		return false, err
	}

	for _, region := range regions {
		if region.Name == regionName && region.Enabled {
			return true, nil
		}
	}

	return false, nil
}

// GetAllRegionNames returns all region names for a provider
func (rm *RegionManager) GetAllRegionNames(provider string) ([]string, error) {
	regions, err := rm.GetRegions(provider)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, region := range regions {
		names = append(names, region.Name)
	}

	return names, nil
}

// GetEnabledRegionNames returns enabled region names for a provider
func (rm *RegionManager) GetEnabledRegionNames(provider string) ([]string, error) {
	regions, err := rm.GetEnabledRegions(provider)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, region := range regions {
		names = append(names, region.Name)
	}

	return names, nil
}

// GetRegionDescription returns the description for a specific region
func (rm *RegionManager) GetRegionDescription(provider, regionName string) (string, error) {
	regions, err := rm.GetRegions(provider)
	if err != nil {
		return "", err
	}

	for _, region := range regions {
		if region.Name == regionName {
			return region.Description, nil
		}
	}

	return "", fmt.Errorf("region %s not found for provider %s", regionName, provider)
}

// IsValidRegionName performs basic validation on region names
func IsValidRegionName(region string) bool {
	// Special case: "all" is valid for discovering all regions
	if region == "all" {
		return true
	}

	// Basic validation: alphanumeric and hyphens only, reasonable length
	if len(region) < 3 || len(region) > 50 {
		return false
	}

	// Check for valid characters only
	for _, char := range region {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-') {
			return false
		}
	}

	// Common cloud region patterns
	validPatterns := []string{
		"us-", "eu-", "ap-", "sa-", "ca-", "af-", "me-",
		"east", "west", "north", "south", "central",
	}

	hasValidPattern := false
	for _, pattern := range validPatterns {
		if strings.Contains(region, pattern) {
			hasValidPattern = true
			break
		}
	}

	return hasValidPattern
}

// ValidateRegionsForProvider validates multiple regions for a specific provider
func (rm *RegionManager) ValidateRegionsForProvider(provider string, regions []string) ([]string, []string, error) {
	var validRegions []string
	var invalidRegions []string

	for _, region := range regions {
		if region == "all" {
			// For "all", get all enabled regions
			enabledRegions, err := rm.GetEnabledRegionNames(provider)
			if err != nil {
				return nil, nil, err
			}
			validRegions = append(validRegions, enabledRegions...)
		} else {
			// Validate individual region
			isValid, err := rm.ValidateRegion(provider, region)
			if err != nil {
				return nil, nil, err
			}
			if isValid {
				validRegions = append(validRegions, region)
			} else {
				invalidRegions = append(invalidRegions, region)
			}
		}
	}

	return validRegions, invalidRegions, nil
}
