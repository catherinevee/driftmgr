package aws

// OptInRegions are AWS regions that require manual activation
var OptInRegions = map[string]bool{
	"af-south-1":     true, // Africa (Cape Town)
	"ap-east-1":      true, // Asia Pacific (Hong Kong)
	"ap-south-2":     true, // Asia Pacific (Hyderabad)
	"ap-southeast-3": true, // Asia Pacific (Jakarta)
	"ap-southeast-4": true, // Asia Pacific (Melbourne)
	"ca-west-1":      true, // Canada West (Calgary)
	"eu-central-2":   true, // Europe (Zurich)
	"eu-south-1":     true, // Europe (Milan)
	"eu-south-2":     true, // Europe (Spain)
	"il-central-1":   true, // Israel (Tel Aviv)
	"me-central-1":   true, // Middle East (UAE)
	"me-south-1":     true, // Middle East (Bahrain)
}

// IsOptInRegion checks if a region requires opt-in
func IsOptInRegion(region string) bool {
	return OptInRegions[region]
}

// GetStandardRegions returns only standard (non-opt-in) AWS regions
func GetStandardRegions() []string {
	return []string{
		"us-east-1",      // US East (N. Virginia)
		"us-east-2",      // US East (Ohio)
		"us-west-1",      // US West (N. California)
		"us-west-2",      // US West (Oregon)
		"ca-central-1",   // Canada (Central)
		"eu-west-1",      // Europe (Ireland)
		"eu-west-2",      // Europe (London)
		"eu-west-3",      // Europe (Paris)
		"eu-central-1",   // Europe (Frankfurt)
		"eu-north-1",     // Europe (Stockholm)
		"ap-south-1",     // Asia Pacific (Mumbai)
		"ap-northeast-1", // Asia Pacific (Tokyo)
		"ap-northeast-2", // Asia Pacific (Seoul)
		"ap-northeast-3", // Asia Pacific (Osaka)
		"ap-southeast-1", // Asia Pacific (Singapore)
		"ap-southeast-2", // Asia Pacific (Sydney)
		"sa-east-1",      // South America (São Paulo)
	}
}

// GetAllRegions returns all AWS regions including opt-in ones
func GetAllRegions() []string {
	standardRegions := GetStandardRegions()
	optInRegions := []string{
		"af-south-1",
		"ap-east-1",
		"ap-south-2",
		"ap-southeast-3",
		"ap-southeast-4",
		"ca-west-1",
		"eu-central-2",
		"eu-south-1",
		"eu-south-2",
		"il-central-1",
		"me-central-1",
		"me-south-1",
	}
	
	return append(standardRegions, optInRegions...)
}