package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

// CloudRegion represents a cloud region with metadata
type CloudRegion struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Provider    string `json:"provider"`
}

// RegionValidationResult contains validation results
type RegionValidationResult struct {
	Provider     string        `json:"provider"`
	TotalRegions int           `json:"total_regions"`
	ValidRegions []CloudRegion `json:"valid_regions"`
	Issues       []string      `json:"issues"`
	Updates      []CloudRegion `json:"updates"`
}

func main() {
	fmt.Println("🔍 Validating Cloud Provider Regions...")
	fmt.Println("=====================================")

	// Validate AWS regions
	awsResult := validateAWSRegions()
	printValidationResult(awsResult)

	// Validate Azure regions
	azureResult := validateAzureRegions()
	printValidationResult(azureResult)

	// Validate GCP regions
	gcpResult := validateGCPRegions()
	printValidationResult(gcpResult)

	// Generate updated region files
	generateUpdatedRegionFiles()

	fmt.Println("\n✅ Region validation complete!")
}

func validateAWSRegions() RegionValidationResult {
	result := RegionValidationResult{
		Provider: "aws",
		Issues:   []string{},
		Updates:  []CloudRegion{},
	}

	// Current AWS regions in the codebase
	currentRegions := []CloudRegion{
		{Name: "us-east-1", Description: "US East (N. Virginia)", Enabled: true, Provider: "aws"},
		{Name: "us-east-2", Description: "US East (Ohio)", Enabled: true, Provider: "aws"},
		{Name: "us-west-1", Description: "US West (N. California)", Enabled: true, Provider: "aws"},
		{Name: "us-west-2", Description: "US West (Oregon)", Enabled: true, Provider: "aws"},
		{Name: "af-south-1", Description: "Africa (Cape Town)", Enabled: true, Provider: "aws"},
		{Name: "ap-east-1", Description: "Asia Pacific (Hong Kong)", Enabled: true, Provider: "aws"},
		{Name: "ap-south-1", Description: "Asia Pacific (Mumbai)", Enabled: true, Provider: "aws"},
		{Name: "ap-northeast-1", Description: "Asia Pacific (Tokyo)", Enabled: true, Provider: "aws"},
		{Name: "ap-northeast-2", Description: "Asia Pacific (Seoul)", Enabled: true, Provider: "aws"},
		{Name: "ap-northeast-3", Description: "Asia Pacific (Osaka)", Enabled: true, Provider: "aws"},
		{Name: "ap-southeast-1", Description: "Asia Pacific (Singapore)", Enabled: true, Provider: "aws"},
		{Name: "ap-southeast-2", Description: "Asia Pacific (Sydney)", Enabled: true, Provider: "aws"},
		{Name: "ap-southeast-3", Description: "Asia Pacific (Jakarta)", Enabled: true, Provider: "aws"},
		{Name: "ap-southeast-4", Description: "Asia Pacific (Melbourne)", Enabled: true, Provider: "aws"},
		{Name: "ca-central-1", Description: "Canada (Central)", Enabled: true, Provider: "aws"},
		{Name: "eu-central-1", Description: "Europe (Frankfurt)", Enabled: true, Provider: "aws"},
		{Name: "eu-west-1", Description: "Europe (Ireland)", Enabled: true, Provider: "aws"},
		{Name: "eu-west-2", Description: "Europe (London)", Enabled: true, Provider: "aws"},
		{Name: "eu-west-3", Description: "Europe (Paris)", Enabled: true, Provider: "aws"},
		{Name: "eu-north-1", Description: "Europe (Stockholm)", Enabled: true, Provider: "aws"},
		{Name: "eu-south-1", Description: "Europe (Milan)", Enabled: true, Provider: "aws"},
		{Name: "eu-south-2", Description: "Europe (Spain)", Enabled: true, Provider: "aws"},
		{Name: "me-south-1", Description: "Middle East (Bahrain)", Enabled: true, Provider: "aws"},
		{Name: "me-central-1", Description: "Middle East (UAE)", Enabled: true, Provider: "aws"},
		{Name: "sa-east-1", Description: "South America (São Paulo)", Enabled: true, Provider: "aws"},
	}

	// Updated AWS regions (as of 2024)
	updatedRegions := []CloudRegion{
		{Name: "us-east-1", Description: "US East (N. Virginia)", Enabled: true, Provider: "aws"},
		{Name: "us-east-2", Description: "US East (Ohio)", Enabled: true, Provider: "aws"},
		{Name: "us-west-1", Description: "US West (N. California)", Enabled: true, Provider: "aws"},
		{Name: "us-west-2", Description: "US West (Oregon)", Enabled: true, Provider: "aws"},
		{Name: "af-south-1", Description: "Africa (Cape Town)", Enabled: true, Provider: "aws"},
		{Name: "ap-east-1", Description: "Asia Pacific (Hong Kong)", Enabled: true, Provider: "aws"},
		{Name: "ap-south-1", Description: "Asia Pacific (Mumbai)", Enabled: true, Provider: "aws"},
		{Name: "ap-south-2", Description: "Asia Pacific (Hyderabad)", Enabled: true, Provider: "aws"},
		{Name: "ap-northeast-1", Description: "Asia Pacific (Tokyo)", Enabled: true, Provider: "aws"},
		{Name: "ap-northeast-2", Description: "Asia Pacific (Seoul)", Enabled: true, Provider: "aws"},
		{Name: "ap-northeast-3", Description: "Asia Pacific (Osaka)", Enabled: true, Provider: "aws"},
		{Name: "ap-southeast-1", Description: "Asia Pacific (Singapore)", Enabled: true, Provider: "aws"},
		{Name: "ap-southeast-2", Description: "Asia Pacific (Sydney)", Enabled: true, Provider: "aws"},
		{Name: "ap-southeast-3", Description: "Asia Pacific (Jakarta)", Enabled: true, Provider: "aws"},
		{Name: "ap-southeast-4", Description: "Asia Pacific (Melbourne)", Enabled: true, Provider: "aws"},
		{Name: "ca-central-1", Description: "Canada (Central)", Enabled: true, Provider: "aws"},
		{Name: "eu-central-1", Description: "Europe (Frankfurt)", Enabled: true, Provider: "aws"},
		{Name: "eu-central-2", Description: "Europe (Zurich)", Enabled: true, Provider: "aws"},
		{Name: "eu-west-1", Description: "Europe (Ireland)", Enabled: true, Provider: "aws"},
		{Name: "eu-west-2", Description: "Europe (London)", Enabled: true, Provider: "aws"},
		{Name: "eu-west-3", Description: "Europe (Paris)", Enabled: true, Provider: "aws"},
		{Name: "eu-north-1", Description: "Europe (Stockholm)", Enabled: true, Provider: "aws"},
		{Name: "eu-south-1", Description: "Europe (Milan)", Enabled: true, Provider: "aws"},
		{Name: "eu-south-2", Description: "Europe (Spain)", Enabled: true, Provider: "aws"},
		{Name: "me-south-1", Description: "Middle East (Bahrain)", Enabled: true, Provider: "aws"},
		{Name: "me-central-1", Description: "Middle East (UAE)", Enabled: true, Provider: "aws"},
		{Name: "sa-east-1", Description: "South America (São Paulo)", Enabled: true, Provider: "aws"},
		{Name: "il-central-1", Description: "Israel (Tel Aviv)", Enabled: true, Provider: "aws"},
	}

	result.ValidRegions = updatedRegions
	result.TotalRegions = len(updatedRegions)

	// Check for missing regions
	currentMap := make(map[string]CloudRegion)
	for _, region := range currentRegions {
		currentMap[region.Name] = region
	}

	for _, region := range updatedRegions {
		if _, exists := currentMap[region.Name]; !exists {
			result.Updates = append(result.Updates, region)
			result.Issues = append(result.Issues, fmt.Sprintf("Missing region: %s (%s)", region.Name, region.Description))
		}
	}

	return result
}

func validateAzureRegions() RegionValidationResult {
	result := RegionValidationResult{
		Provider: "azure",
		Issues:   []string{},
		Updates:  []CloudRegion{},
	}

	// Current Azure regions in the codebase
	currentRegions := []CloudRegion{
		{Name: "eastus", Description: "East US", Enabled: true, Provider: "azure"},
		{Name: "eastus2", Description: "East US 2", Enabled: true, Provider: "azure"},
		{Name: "southcentralus", Description: "South Central US", Enabled: true, Provider: "azure"},
		{Name: "westus2", Description: "West US 2", Enabled: true, Provider: "azure"},
		{Name: "westus3", Description: "West US 3", Enabled: true, Provider: "azure"},
		{Name: "australiaeast", Description: "Australia East", Enabled: true, Provider: "azure"},
		{Name: "southeastasia", Description: "Southeast Asia", Enabled: true, Provider: "azure"},
		{Name: "northeurope", Description: "North Europe", Enabled: true, Provider: "azure"},
		{Name: "swedencentral", Description: "Sweden Central", Enabled: true, Provider: "azure"},
		{Name: "uksouth", Description: "UK South", Enabled: true, Provider: "azure"},
		{Name: "westeurope", Description: "West Europe", Enabled: true, Provider: "azure"},
		{Name: "centralus", Description: "Central US", Enabled: true, Provider: "azure"},
		{Name: "northcentralus", Description: "North Central US", Enabled: true, Provider: "azure"},
		{Name: "westus", Description: "West US", Enabled: true, Provider: "azure"},
		{Name: "southafricanorth", Description: "South Africa North", Enabled: true, Provider: "azure"},
		{Name: "centralindia", Description: "Central India", Enabled: true, Provider: "azure"},
		{Name: "eastasia", Description: "East Asia", Enabled: true, Provider: "azure"},
		{Name: "japaneast", Description: "Japan East", Enabled: true, Provider: "azure"},
		{Name: "japanwest", Description: "Japan West", Enabled: true, Provider: "azure"},
		{Name: "koreacentral", Description: "Korea Central", Enabled: true, Provider: "azure"},
		{Name: "canadacentral", Description: "Canada Central", Enabled: true, Provider: "azure"},
		{Name: "francecentral", Description: "France Central", Enabled: true, Provider: "azure"},
		{Name: "germanywestcentral", Description: "Germany West Central", Enabled: true, Provider: "azure"},
		{Name: "italynorth", Description: "Italy North", Enabled: true, Provider: "azure"},
		{Name: "norwayeast", Description: "Norway East", Enabled: true, Provider: "azure"},
		{Name: "polandcentral", Description: "Poland Central", Enabled: true, Provider: "azure"},
		{Name: "switzerlandnorth", Description: "Switzerland North", Enabled: true, Provider: "azure"},
		{Name: "uaenorth", Description: "UAE North", Enabled: true, Provider: "azure"},
		{Name: "brazilsouth", Description: "Brazil South", Enabled: true, Provider: "azure"},
	}

	// Updated Azure regions (as of 2024)
	updatedRegions := []CloudRegion{
		{Name: "eastus", Description: "East US", Enabled: true, Provider: "azure"},
		{Name: "eastus2", Description: "East US 2", Enabled: true, Provider: "azure"},
		{Name: "southcentralus", Description: "South Central US", Enabled: true, Provider: "azure"},
		{Name: "westus2", Description: "West US 2", Enabled: true, Provider: "azure"},
		{Name: "westus3", Description: "West US 3", Enabled: true, Provider: "azure"},
		{Name: "australiaeast", Description: "Australia East", Enabled: true, Provider: "azure"},
		{Name: "australiasoutheast", Description: "Australia Southeast", Enabled: true, Provider: "azure"},
		{Name: "southeastasia", Description: "Southeast Asia", Enabled: true, Provider: "azure"},
		{Name: "northeurope", Description: "North Europe", Enabled: true, Provider: "azure"},
		{Name: "swedencentral", Description: "Sweden Central", Enabled: true, Provider: "azure"},
		{Name: "uksouth", Description: "UK South", Enabled: true, Provider: "azure"},
		{Name: "ukwest", Description: "UK West", Enabled: true, Provider: "azure"},
		{Name: "westeurope", Description: "West Europe", Enabled: true, Provider: "azure"},
		{Name: "centralus", Description: "Central US", Enabled: true, Provider: "azure"},
		{Name: "northcentralus", Description: "North Central US", Enabled: true, Provider: "azure"},
		{Name: "westus", Description: "West US", Enabled: true, Provider: "azure"},
		{Name: "southafricanorth", Description: "South Africa North", Enabled: true, Provider: "azure"},
		{Name: "southafricawest", Description: "South Africa West", Enabled: true, Provider: "azure"},
		{Name: "centralindia", Description: "Central India", Enabled: true, Provider: "azure"},
		{Name: "southindia", Description: "South India", Enabled: true, Provider: "azure"},
		{Name: "westindia", Description: "West India", Enabled: true, Provider: "azure"},
		{Name: "eastasia", Description: "East Asia", Enabled: true, Provider: "azure"},
		{Name: "japaneast", Description: "Japan East", Enabled: true, Provider: "azure"},
		{Name: "japanwest", Description: "Japan West", Enabled: true, Provider: "azure"},
		{Name: "koreacentral", Description: "Korea Central", Enabled: true, Provider: "azure"},
		{Name: "koreasouth", Description: "Korea South", Enabled: true, Provider: "azure"},
		{Name: "canadacentral", Description: "Canada Central", Enabled: true, Provider: "azure"},
		{Name: "canadaeast", Description: "Canada East", Enabled: true, Provider: "azure"},
		{Name: "francecentral", Description: "France Central", Enabled: true, Provider: "azure"},
		{Name: "francesouth", Description: "France South", Enabled: true, Provider: "azure"},
		{Name: "germanywestcentral", Description: "Germany West Central", Enabled: true, Provider: "azure"},
		{Name: "germanynorth", Description: "Germany North", Enabled: true, Provider: "azure"},
		{Name: "italynorth", Description: "Italy North", Enabled: true, Provider: "azure"},
		{Name: "norwayeast", Description: "Norway East", Enabled: true, Provider: "azure"},
		{Name: "polandcentral", Description: "Poland Central", Enabled: true, Provider: "azure"},
		{Name: "switzerlandnorth", Description: "Switzerland North", Enabled: true, Provider: "azure"},
		{Name: "switzerlandwest", Description: "Switzerland West", Enabled: true, Provider: "azure"},
		{Name: "uaenorth", Description: "UAE North", Enabled: true, Provider: "azure"},
		{Name: "uaecentral", Description: "UAE Central", Enabled: true, Provider: "azure"},
		{Name: "brazilsouth", Description: "Brazil South", Enabled: true, Provider: "azure"},
		{Name: "brazilsoutheast", Description: "Brazil Southeast", Enabled: true, Provider: "azure"},
		{Name: "chilecentral", Description: "Chile Central", Enabled: true, Provider: "azure"},
		{Name: "mexicocentral", Description: "Mexico Central", Enabled: true, Provider: "azure"},
		{Name: "qatarcentral", Description: "Qatar Central", Enabled: true, Provider: "azure"},
	}

	result.ValidRegions = updatedRegions
	result.TotalRegions = len(updatedRegions)

	// Check for missing regions
	currentMap := make(map[string]CloudRegion)
	for _, region := range currentRegions {
		currentMap[region.Name] = region
	}

	for _, region := range updatedRegions {
		if _, exists := currentMap[region.Name]; !exists {
			result.Updates = append(result.Updates, region)
			result.Issues = append(result.Issues, fmt.Sprintf("Missing region: %s (%s)", region.Name, region.Description))
		}
	}

	return result
}

func validateGCPRegions() RegionValidationResult {
	result := RegionValidationResult{
		Provider: "gcp",
		Issues:   []string{},
		Updates:  []CloudRegion{},
	}

	// Current GCP regions in the codebase
	currentRegions := []CloudRegion{
		{Name: "us-central1", Description: "Iowa", Enabled: true, Provider: "gcp"},
		{Name: "us-east1", Description: "South Carolina", Enabled: true, Provider: "gcp"},
		{Name: "us-east4", Description: "Northern Virginia", Enabled: true, Provider: "gcp"},
		{Name: "us-west1", Description: "Oregon", Enabled: true, Provider: "gcp"},
		{Name: "us-west2", Description: "Los Angeles", Enabled: true, Provider: "gcp"},
		{Name: "us-west3", Description: "Salt Lake City", Enabled: true, Provider: "gcp"},
		{Name: "us-west4", Description: "Las Vegas", Enabled: true, Provider: "gcp"},
		{Name: "europe-west1", Description: "Belgium", Enabled: true, Provider: "gcp"},
		{Name: "europe-west2", Description: "London", Enabled: true, Provider: "gcp"},
		{Name: "europe-west3", Description: "Frankfurt", Enabled: true, Provider: "gcp"},
		{Name: "europe-west4", Description: "Netherlands", Enabled: true, Provider: "gcp"},
		{Name: "europe-west6", Description: "Zurich", Enabled: true, Provider: "gcp"},
		{Name: "europe-west8", Description: "Milan", Enabled: true, Provider: "gcp"},
		{Name: "europe-west9", Description: "Paris", Enabled: true, Provider: "gcp"},
		{Name: "europe-west10", Description: "Berlin", Enabled: true, Provider: "gcp"},
		{Name: "europe-west12", Description: "Turin", Enabled: true, Provider: "gcp"},
		{Name: "europe-central2", Description: "Warsaw", Enabled: true, Provider: "gcp"},
		{Name: "europe-north1", Description: "Finland", Enabled: true, Provider: "gcp"},
		{Name: "europe-southwest1", Description: "Madrid", Enabled: true, Provider: "gcp"},
		{Name: "asia-east1", Description: "Taiwan", Enabled: true, Provider: "gcp"},
		{Name: "asia-northeast1", Description: "Tokyo", Enabled: true, Provider: "gcp"},
		{Name: "asia-northeast2", Description: "Osaka", Enabled: true, Provider: "gcp"},
		{Name: "asia-northeast3", Description: "Seoul", Enabled: true, Provider: "gcp"},
		{Name: "asia-south1", Description: "Mumbai", Enabled: true, Provider: "gcp"},
		{Name: "asia-south2", Description: "Delhi", Enabled: true, Provider: "gcp"},
		{Name: "asia-southeast1", Description: "Singapore", Enabled: true, Provider: "gcp"},
		{Name: "asia-southeast2", Description: "Jakarta", Enabled: true, Provider: "gcp"},
		{Name: "australia-southeast1", Description: "Sydney", Enabled: true, Provider: "gcp"},
		{Name: "australia-southeast2", Description: "Melbourne", Enabled: true, Provider: "gcp"},
		{Name: "southamerica-east1", Description: "São Paulo", Enabled: true, Provider: "gcp"},
		{Name: "northamerica-northeast1", Description: "Montreal", Enabled: true, Provider: "gcp"},
		{Name: "northamerica-northeast2", Description: "Toronto", Enabled: true, Provider: "gcp"},
	}

	// Updated GCP regions (as of 2024)
	updatedRegions := []CloudRegion{
		{Name: "us-central1", Description: "Iowa", Enabled: true, Provider: "gcp"},
		{Name: "us-east1", Description: "South Carolina", Enabled: true, Provider: "gcp"},
		{Name: "us-east4", Description: "Northern Virginia", Enabled: true, Provider: "gcp"},
		{Name: "us-east5", Description: "Columbus", Enabled: true, Provider: "gcp"},
		{Name: "us-west1", Description: "Oregon", Enabled: true, Provider: "gcp"},
		{Name: "us-west2", Description: "Los Angeles", Enabled: true, Provider: "gcp"},
		{Name: "us-west3", Description: "Salt Lake City", Enabled: true, Provider: "gcp"},
		{Name: "us-west4", Description: "Las Vegas", Enabled: true, Provider: "gcp"},
		{Name: "europe-west1", Description: "Belgium", Enabled: true, Provider: "gcp"},
		{Name: "europe-west2", Description: "London", Enabled: true, Provider: "gcp"},
		{Name: "europe-west3", Description: "Frankfurt", Enabled: true, Provider: "gcp"},
		{Name: "europe-west4", Description: "Netherlands", Enabled: true, Provider: "gcp"},
		{Name: "europe-west6", Description: "Zurich", Enabled: true, Provider: "gcp"},
		{Name: "europe-west8", Description: "Milan", Enabled: true, Provider: "gcp"},
		{Name: "europe-west9", Description: "Paris", Enabled: true, Provider: "gcp"},
		{Name: "europe-west10", Description: "Berlin", Enabled: true, Provider: "gcp"},
		{Name: "europe-west12", Description: "Turin", Enabled: true, Provider: "gcp"},
		{Name: "europe-central2", Description: "Warsaw", Enabled: true, Provider: "gcp"},
		{Name: "europe-north1", Description: "Finland", Enabled: true, Provider: "gcp"},
		{Name: "europe-southwest1", Description: "Madrid", Enabled: true, Provider: "gcp"},
		{Name: "asia-east1", Description: "Taiwan", Enabled: true, Provider: "gcp"},
		{Name: "asia-northeast1", Description: "Tokyo", Enabled: true, Provider: "gcp"},
		{Name: "asia-northeast2", Description: "Osaka", Enabled: true, Provider: "gcp"},
		{Name: "asia-northeast3", Description: "Seoul", Enabled: true, Provider: "gcp"},
		{Name: "asia-south1", Description: "Mumbai", Enabled: true, Provider: "gcp"},
		{Name: "asia-south2", Description: "Delhi", Enabled: true, Provider: "gcp"},
		{Name: "asia-southeast1", Description: "Singapore", Enabled: true, Provider: "gcp"},
		{Name: "asia-southeast2", Description: "Jakarta", Enabled: true, Provider: "gcp"},
		{Name: "australia-southeast1", Description: "Sydney", Enabled: true, Provider: "gcp"},
		{Name: "australia-southeast2", Description: "Melbourne", Enabled: true, Provider: "gcp"},
		{Name: "southamerica-east1", Description: "São Paulo", Enabled: true, Provider: "gcp"},
		{Name: "northamerica-northeast1", Description: "Montreal", Enabled: true, Provider: "gcp"},
		{Name: "northamerica-northeast2", Description: "Toronto", Enabled: true, Provider: "gcp"},
	}

	result.ValidRegions = updatedRegions
	result.TotalRegions = len(updatedRegions)

	// Check for missing regions
	currentMap := make(map[string]CloudRegion)
	for _, region := range currentRegions {
		currentMap[region.Name] = region
	}

	for _, region := range updatedRegions {
		if _, exists := currentMap[region.Name]; !exists {
			result.Updates = append(result.Updates, region)
			result.Issues = append(result.Issues, fmt.Sprintf("Missing region: %s (%s)", region.Name, region.Description))
		}
	}

	return result
}

func printValidationResult(result RegionValidationResult) {
	fmt.Printf("\n🌐 %s Regions Validation\n", strings.ToUpper(result.Provider))
	fmt.Printf("   Total Regions: %d\n", result.TotalRegions)

	if len(result.Issues) > 0 {
		fmt.Printf("   ⚠️  Issues Found: %d\n", len(result.Issues))
		for _, issue := range result.Issues {
			fmt.Printf("      • %s\n", issue)
		}
	} else {
		fmt.Printf("   ✅ All regions are up to date\n")
	}
}

func generateUpdatedRegionFiles() {
	fmt.Println("\n📝 Generating updated region files...")

	// Generate AWS regions file
	awsRegions := []CloudRegion{
		{Name: "us-east-1", Description: "US East (N. Virginia)", Enabled: true, Provider: "aws"},
		{Name: "us-east-2", Description: "US East (Ohio)", Enabled: true, Provider: "aws"},
		{Name: "us-west-1", Description: "US West (N. California)", Enabled: true, Provider: "aws"},
		{Name: "us-west-2", Description: "US West (Oregon)", Enabled: true, Provider: "aws"},
		{Name: "af-south-1", Description: "Africa (Cape Town)", Enabled: true, Provider: "aws"},
		{Name: "ap-east-1", Description: "Asia Pacific (Hong Kong)", Enabled: true, Provider: "aws"},
		{Name: "ap-south-1", Description: "Asia Pacific (Mumbai)", Enabled: true, Provider: "aws"},
		{Name: "ap-south-2", Description: "Asia Pacific (Hyderabad)", Enabled: true, Provider: "aws"},
		{Name: "ap-northeast-1", Description: "Asia Pacific (Tokyo)", Enabled: true, Provider: "aws"},
		{Name: "ap-northeast-2", Description: "Asia Pacific (Seoul)", Enabled: true, Provider: "aws"},
		{Name: "ap-northeast-3", Description: "Asia Pacific (Osaka)", Enabled: true, Provider: "aws"},
		{Name: "ap-southeast-1", Description: "Asia Pacific (Singapore)", Enabled: true, Provider: "aws"},
		{Name: "ap-southeast-2", Description: "Asia Pacific (Sydney)", Enabled: true, Provider: "aws"},
		{Name: "ap-southeast-3", Description: "Asia Pacific (Jakarta)", Enabled: true, Provider: "aws"},
		{Name: "ap-southeast-4", Description: "Asia Pacific (Melbourne)", Enabled: true, Provider: "aws"},
		{Name: "ca-central-1", Description: "Canada (Central)", Enabled: true, Provider: "aws"},
		{Name: "eu-central-1", Description: "Europe (Frankfurt)", Enabled: true, Provider: "aws"},
		{Name: "eu-central-2", Description: "Europe (Zurich)", Enabled: true, Provider: "aws"},
		{Name: "eu-west-1", Description: "Europe (Ireland)", Enabled: true, Provider: "aws"},
		{Name: "eu-west-2", Description: "Europe (London)", Enabled: true, Provider: "aws"},
		{Name: "eu-west-3", Description: "Europe (Paris)", Enabled: true, Provider: "aws"},
		{Name: "eu-north-1", Description: "Europe (Stockholm)", Enabled: true, Provider: "aws"},
		{Name: "eu-south-1", Description: "Europe (Milan)", Enabled: true, Provider: "aws"},
		{Name: "eu-south-2", Description: "Europe (Spain)", Enabled: true, Provider: "aws"},
		{Name: "me-south-1", Description: "Middle East (Bahrain)", Enabled: true, Provider: "aws"},
		{Name: "me-central-1", Description: "Middle East (UAE)", Enabled: true, Provider: "aws"},
		{Name: "sa-east-1", Description: "South America (São Paulo)", Enabled: true, Provider: "aws"},
		{Name: "il-central-1", Description: "Israel (Tel Aviv)", Enabled: true, Provider: "aws"},
	}

	// Generate Azure regions file
	azureRegions := []CloudRegion{
		{Name: "eastus", Description: "East US", Enabled: true, Provider: "azure"},
		{Name: "eastus2", Description: "East US 2", Enabled: true, Provider: "azure"},
		{Name: "southcentralus", Description: "South Central US", Enabled: true, Provider: "azure"},
		{Name: "westus2", Description: "West US 2", Enabled: true, Provider: "azure"},
		{Name: "westus3", Description: "West US 3", Enabled: true, Provider: "azure"},
		{Name: "australiaeast", Description: "Australia East", Enabled: true, Provider: "azure"},
		{Name: "australiasoutheast", Description: "Australia Southeast", Enabled: true, Provider: "azure"},
		{Name: "southeastasia", Description: "Southeast Asia", Enabled: true, Provider: "azure"},
		{Name: "northeurope", Description: "North Europe", Enabled: true, Provider: "azure"},
		{Name: "swedencentral", Description: "Sweden Central", Enabled: true, Provider: "azure"},
		{Name: "uksouth", Description: "UK South", Enabled: true, Provider: "azure"},
		{Name: "ukwest", Description: "UK West", Enabled: true, Provider: "azure"},
		{Name: "westeurope", Description: "West Europe", Enabled: true, Provider: "azure"},
		{Name: "centralus", Description: "Central US", Enabled: true, Provider: "azure"},
		{Name: "northcentralus", Description: "North Central US", Enabled: true, Provider: "azure"},
		{Name: "westus", Description: "West US", Enabled: true, Provider: "azure"},
		{Name: "southafricanorth", Description: "South Africa North", Enabled: true, Provider: "azure"},
		{Name: "southafricawest", Description: "South Africa West", Enabled: true, Provider: "azure"},
		{Name: "centralindia", Description: "Central India", Enabled: true, Provider: "azure"},
		{Name: "southindia", Description: "South India", Enabled: true, Provider: "azure"},
		{Name: "westindia", Description: "West India", Enabled: true, Provider: "azure"},
		{Name: "eastasia", Description: "East Asia", Enabled: true, Provider: "azure"},
		{Name: "japaneast", Description: "Japan East", Enabled: true, Provider: "azure"},
		{Name: "japanwest", Description: "Japan West", Enabled: true, Provider: "azure"},
		{Name: "koreacentral", Description: "Korea Central", Enabled: true, Provider: "azure"},
		{Name: "koreasouth", Description: "Korea South", Enabled: true, Provider: "azure"},
		{Name: "canadacentral", Description: "Canada Central", Enabled: true, Provider: "azure"},
		{Name: "canadaeast", Description: "Canada East", Enabled: true, Provider: "azure"},
		{Name: "francecentral", Description: "France Central", Enabled: true, Provider: "azure"},
		{Name: "francesouth", Description: "France South", Enabled: true, Provider: "azure"},
		{Name: "germanywestcentral", Description: "Germany West Central", Enabled: true, Provider: "azure"},
		{Name: "germanynorth", Description: "Germany North", Enabled: true, Provider: "azure"},
		{Name: "italynorth", Description: "Italy North", Enabled: true, Provider: "azure"},
		{Name: "norwayeast", Description: "Norway East", Enabled: true, Provider: "azure"},
		{Name: "polandcentral", Description: "Poland Central", Enabled: true, Provider: "azure"},
		{Name: "switzerlandnorth", Description: "Switzerland North", Enabled: true, Provider: "azure"},
		{Name: "switzerlandwest", Description: "Switzerland West", Enabled: true, Provider: "azure"},
		{Name: "uaenorth", Description: "UAE North", Enabled: true, Provider: "azure"},
		{Name: "uaecentral", Description: "UAE Central", Enabled: true, Provider: "azure"},
		{Name: "brazilsouth", Description: "Brazil South", Enabled: true, Provider: "azure"},
		{Name: "brazilsoutheast", Description: "Brazil Southeast", Enabled: true, Provider: "azure"},
		{Name: "chilecentral", Description: "Chile Central", Enabled: true, Provider: "azure"},
		{Name: "mexicocentral", Description: "Mexico Central", Enabled: true, Provider: "azure"},
		{Name: "qatarcentral", Description: "Qatar Central", Enabled: true, Provider: "azure"},
	}

	// Generate GCP regions file
	gcpRegions := []CloudRegion{
		{Name: "us-central1", Description: "Iowa", Enabled: true, Provider: "gcp"},
		{Name: "us-east1", Description: "South Carolina", Enabled: true, Provider: "gcp"},
		{Name: "us-east4", Description: "Northern Virginia", Enabled: true, Provider: "gcp"},
		{Name: "us-east5", Description: "Columbus", Enabled: true, Provider: "gcp"},
		{Name: "us-west1", Description: "Oregon", Enabled: true, Provider: "gcp"},
		{Name: "us-west2", Description: "Los Angeles", Enabled: true, Provider: "gcp"},
		{Name: "us-west3", Description: "Salt Lake City", Enabled: true, Provider: "gcp"},
		{Name: "us-west4", Description: "Las Vegas", Enabled: true, Provider: "gcp"},
		{Name: "europe-west1", Description: "Belgium", Enabled: true, Provider: "gcp"},
		{Name: "europe-west2", Description: "London", Enabled: true, Provider: "gcp"},
		{Name: "europe-west3", Description: "Frankfurt", Enabled: true, Provider: "gcp"},
		{Name: "europe-west4", Description: "Netherlands", Enabled: true, Provider: "gcp"},
		{Name: "europe-west6", Description: "Zurich", Enabled: true, Provider: "gcp"},
		{Name: "europe-west8", Description: "Milan", Enabled: true, Provider: "gcp"},
		{Name: "europe-west9", Description: "Paris", Enabled: true, Provider: "gcp"},
		{Name: "europe-west10", Description: "Berlin", Enabled: true, Provider: "gcp"},
		{Name: "europe-west12", Description: "Turin", Enabled: true, Provider: "gcp"},
		{Name: "europe-central2", Description: "Warsaw", Enabled: true, Provider: "gcp"},
		{Name: "europe-north1", Description: "Finland", Enabled: true, Provider: "gcp"},
		{Name: "europe-southwest1", Description: "Madrid", Enabled: true, Provider: "gcp"},
		{Name: "asia-east1", Description: "Taiwan", Enabled: true, Provider: "gcp"},
		{Name: "asia-northeast1", Description: "Tokyo", Enabled: true, Provider: "gcp"},
		{Name: "asia-northeast2", Description: "Osaka", Enabled: true, Provider: "gcp"},
		{Name: "asia-northeast3", Description: "Seoul", Enabled: true, Provider: "gcp"},
		{Name: "asia-south1", Description: "Mumbai", Enabled: true, Provider: "gcp"},
		{Name: "asia-south2", Description: "Delhi", Enabled: true, Provider: "gcp"},
		{Name: "asia-southeast1", Description: "Singapore", Enabled: true, Provider: "gcp"},
		{Name: "asia-southeast2", Description: "Jakarta", Enabled: true, Provider: "gcp"},
		{Name: "australia-southeast1", Description: "Sydney", Enabled: true, Provider: "gcp"},
		{Name: "australia-southeast2", Description: "Melbourne", Enabled: true, Provider: "gcp"},
		{Name: "southamerica-east1", Description: "São Paulo", Enabled: true, Provider: "gcp"},
		{Name: "northamerica-northeast1", Description: "Montreal", Enabled: true, Provider: "gcp"},
		{Name: "northamerica-northeast2", Description: "Toronto", Enabled: true, Provider: "gcp"},
	}

	// Write AWS regions file
	writeRegionsFile("aws_regions.json", awsRegions)

	// Write Azure regions file
	writeRegionsFile("azure_regions.json", azureRegions)

	// Write GCP regions file
	writeRegionsFile("gcp_regions.json", gcpRegions)

	// Write combined regions file
	allRegions := map[string][]CloudRegion{
		"aws":   awsRegions,
		"azure": azureRegions,
		"gcp":   gcpRegions,
	}
	writeRegionsFile("all_regions.json", allRegions)
}

func writeRegionsFile(filename string, data interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("Error marshaling JSON for %s: %v", filename, err)
		return
	}

	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		log.Printf("Error writing file %s: %v", filename, err)
		return
	}

	fmt.Printf("   ✅ Generated %s\n", filename)
}
