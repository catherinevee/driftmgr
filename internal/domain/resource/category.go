package resource

import (
	"fmt"
	"strings"
	"time"
)

// ResourceCategory represents the management status of a resource
type ResourceCategory string

const (
	CategoryManaged      ResourceCategory = "MANAGED"      // In Terraform state
	CategoryManageable   ResourceCategory = "MANAGEABLE"   // Should be in Terraform
	CategoryUnmanageable ResourceCategory = "UNMANAGEABLE" // System/default resources
	CategoryUnknown      ResourceCategory = "UNKNOWN"      // Needs investigation
	CategoryShadowIT     ResourceCategory = "SHADOW_IT"    // Created outside process
	CategoryOrphaned     ResourceCategory = "ORPHANED"     // Was managed, now abandoned
	CategoryTemporary    ResourceCategory = "TEMPORARY"    // Short-lived resources
)

// ImportCandidate represents a resource that could be imported
type ImportCandidate struct {
	Resource         Resource
	Category         ResourceCategory
	Score            float64
	Reasons          []string
	TerraformType    string
	ImportCommand    string
	EstimatedImpact  string
	Dependencies     []string
	CreatedBy        string
	CreatedAt        time.Time
	LastModified     time.Time
	ComplianceIssues []string
	Cost             float64
}

// DiscoveryProfile defines rules for categorizing resources
type DiscoveryProfile struct {
	Name              string
	ManagedTags       map[string]string
	NamingPatterns    []string
	ExclusionPatterns []string
	RequiredTags      []string
	AgeThreshold      time.Duration
	CostThreshold     float64
}

// ResourceCategorizer categorizes and scores resources
type ResourceCategorizer struct {
	profiles        []DiscoveryProfile
	stateResources  map[string]bool
	knownExclusions map[string]bool
	tagStrategy     TagStrategy
}

// TagStrategy defines how to identify Terraform-managed resources
type TagStrategy struct {
	ManagedByTags    []string
	TerraformTags    []string
	EnvironmentTags  []string
	OwnershipTags    []string
	AutomationTags   []string
	ComplianceTags   []string
}

// NewResourceCategorizer creates a new categorizer
func NewResourceCategorizer() *ResourceCategorizer {
	return &ResourceCategorizer{
		profiles:        getDefaultProfiles(),
		stateResources:  make(map[string]bool),
		knownExclusions: getKnownExclusions(),
		tagStrategy:     getDefaultTagStrategy(),
	}
}

// CategorizeResource determines the category of a resource
func (rc *ResourceCategorizer) CategorizeResource(resource Resource, inState bool) ResourceCategory {
	// Check if in state
	if inState {
		return CategoryManaged
	}
	
	// Check if it's a known system resource
	if rc.isSystemResource(resource) {
		return CategoryUnmanageable
	}
	
	// Check if it's temporary
	if rc.isTemporaryResource(resource) {
		return CategoryTemporary
	}
	
	// Check for shadow IT indicators
	if rc.isShadowIT(resource) {
		return CategoryShadowIT
	}
	
	// Check if it was previously managed (orphaned)
	if rc.wasManaged(resource) {
		return CategoryOrphaned
	}
	
	// Check if it should be managed
	if rc.shouldBeManaged(resource) {
		return CategoryManageable
	}
	
	return CategoryUnknown
}

// ScoreImportCandidate calculates import priority score
func (rc *ResourceCategorizer) ScoreImportCandidate(resource Resource) *ImportCandidate {
	candidate := &ImportCandidate{
		Resource: resource,
		Score:    0.0,
		Reasons:  []string{},
	}
	
	// Base score by resource type importance
	candidate.Score += rc.getResourceTypeScore(resource.Type)
	
	// Adjust for environment
	if env := rc.detectEnvironment(resource); env != "" {
		switch env {
		case "production":
			candidate.Score += 30
			candidate.Reasons = append(candidate.Reasons, "Production resource")
		case "staging":
			candidate.Score += 20
			candidate.Reasons = append(candidate.Reasons, "Staging resource")
		case "development":
			candidate.Score += 10
			candidate.Reasons = append(candidate.Reasons, "Development resource")
		}
	}
	
	// Check naming convention
	if rc.followsNamingConvention(resource) {
		candidate.Score += 15
		candidate.Reasons = append(candidate.Reasons, "Follows naming convention")
	}
	
	// Check for required tags
	missingTags := rc.getMissingRequiredTags(resource)
	if len(missingTags) == 0 {
		candidate.Score += 10
		candidate.Reasons = append(candidate.Reasons, "Has all required tags")
	} else {
		candidate.Score -= float64(len(missingTags)) * 2
		candidate.ComplianceIssues = append(candidate.ComplianceIssues, 
			fmt.Sprintf("Missing tags: %s", strings.Join(missingTags, ", ")))
	}
	
	// Check dependencies
	deps := rc.findDependencies(resource)
	if len(deps) > 0 {
		candidate.Score += float64(len(deps)) * 5
		candidate.Reasons = append(candidate.Reasons, 
			fmt.Sprintf("Has %d dependencies", len(deps)))
		candidate.Dependencies = deps
	}
	
	// Check age
	if age := rc.getResourceAge(resource); age != 0 {
		if age < 7*24*time.Hour {
			candidate.Score += 5
			candidate.Reasons = append(candidate.Reasons, "Recently created")
		} else if age > 90*24*time.Hour {
			candidate.Score += 10
			candidate.Reasons = append(candidate.Reasons, "Long-lived resource")
		}
	}
	
	// Check for automation indicators
	if rc.hasAutomationIndicators(resource) {
		candidate.Score -= 10
		candidate.Reasons = append(candidate.Reasons, "May be managed by other automation")
	}
	
	// Security considerations
	if rc.hasSecurityImplications(resource) {
		candidate.Score += 20
		candidate.Reasons = append(candidate.Reasons, "Security-sensitive resource")
	}
	
	// Cost considerations
	if cost := rc.estimateCost(resource); cost > 0 {
		candidate.Cost = cost
		if cost > 100 {
			candidate.Score += 15
			candidate.Reasons = append(candidate.Reasons, fmt.Sprintf("High cost: $%.2f/month", cost))
		}
	}
	
	// Cap score at 100
	if candidate.Score > 100 {
		candidate.Score = 100
	} else if candidate.Score < 0 {
		candidate.Score = 0
	}
	
	// Set import details
	candidate.TerraformType = rc.mapToTerraformType(resource)
	candidate.ImportCommand = rc.generateImportCommand(resource, candidate.TerraformType)
	candidate.EstimatedImpact = rc.estimateImportImpact(resource)
	
	return candidate
}

// isSystemResource checks if resource is a system/default resource
func (rc *ResourceCategorizer) isSystemResource(resource Resource) bool {
	// Check known exclusions
	if rc.knownExclusions[resource.Type] {
		return true
	}
	
	// AWS default resources
	if strings.HasPrefix(resource.Provider, "aws") {
		if strings.Contains(resource.Name, "default") ||
		   strings.HasPrefix(resource.ID, "vpc-00000000") ||
		   strings.HasPrefix(resource.Name, "aws-service-role") ||
		   strings.Contains(resource.Type, "::ServiceLinkedRole") {
			return true
		}
	}
	
	// Azure system resources
	if strings.HasPrefix(resource.Provider, "azure") {
		if strings.Contains(resource.Name, "AzureBackup") ||
		   strings.Contains(resource.Name, "DefaultWorkspace") ||
		   strings.HasPrefix(resource.Name, "SystemApplication") {
			return true
		}
	}
	
	// GCP system resources
	if strings.HasPrefix(resource.Provider, "gcp") {
		if strings.Contains(resource.Name, "default") ||
		   strings.HasPrefix(resource.Name, "gke-") ||
		   strings.Contains(resource.Name, "system") {
			return true
		}
	}
	
	return false
}

// isTemporaryResource checks if resource is temporary/ephemeral
func (rc *ResourceCategorizer) isTemporaryResource(resource Resource) bool {
	// Check for temporary indicators in name
	tempIndicators := []string{"temp", "tmp", "test", "demo", "ephemeral", "delete"}
	lowerName := strings.ToLower(resource.Name)
	for _, indicator := range tempIndicators {
		if strings.Contains(lowerName, indicator) {
			return true
		}
	}
	
	// Check age - if very new and matches patterns
	if age := rc.getResourceAge(resource); age > 0 && age < 24*time.Hour {
		if strings.Contains(lowerName, "lambda") && strings.Contains(lowerName, "eni") {
			return true // Lambda ENIs are temporary
		}
	}
	
	// Check tags for temporary indicators
	if tags, ok := resource.Tags.(map[string]string); ok {
		if ttl, exists := tags["ttl"]; exists && ttl != "" {
			return true
		}
		if temp, exists := tags["temporary"]; exists && temp == "true" {
			return true
		}
	}
	
	return false
}

// isShadowIT checks for shadow IT indicators
func (rc *ResourceCategorizer) isShadowIT(resource Resource) bool {
	shadowIndicators := 0
	
	// Check for manual creation patterns
	if !rc.hasAutomationTags(resource) {
		shadowIndicators++
	}
	
	// Check naming convention
	if !rc.followsNamingConvention(resource) {
		shadowIndicators++
	}
	
	// Check for required tags
	if len(rc.getMissingRequiredTags(resource)) > 2 {
		shadowIndicators++
	}
	
	// Check creation method (would need CloudTrail/Activity Log integration)
	if rc.isConsoleCreated(resource) {
		shadowIndicators += 2
	}
	
	// Check for isolated resources (no dependencies)
	if len(rc.findDependencies(resource)) == 0 {
		shadowIndicators++
	}
	
	return shadowIndicators >= 3
}

// wasManaged checks if resource was previously managed by Terraform
func (rc *ResourceCategorizer) wasManaged(resource Resource) bool {
	// Check for Terraform tags that indicate previous management
	if tags, ok := resource.Tags.(map[string]string); ok {
		// Look for Terraform lifecycle tags
		if _, exists := tags["terraform_managed"]; exists {
			return true
		}
		if workspace, exists := tags["terraform_workspace"]; exists && workspace != "" {
			return true
		}
		if _, exists := tags["tf_created"]; exists {
			return true
		}
	}
	
	// Check naming patterns that indicate Terraform
	if strings.Contains(resource.Name, "tf-") ||
	   strings.Contains(resource.Name, "terraform-") {
		return true
	}
	
	return false
}

// shouldBeManaged determines if resource should be in Terraform
func (rc *ResourceCategorizer) shouldBeManaged(resource Resource) bool {
	// Production resources should be managed
	if env := rc.detectEnvironment(resource); env == "production" {
		return true
	}
	
	// Resources with dependencies should be managed
	if len(rc.findDependencies(resource)) > 2 {
		return true
	}
	
	// High-cost resources should be managed
	if rc.estimateCost(resource) > 50 {
		return true
	}
	
	// Security-sensitive resources should be managed
	if rc.hasSecurityImplications(resource) {
		return true
	}
	
	// Resources following naming convention likely should be managed
	if rc.followsNamingConvention(resource) {
		return true
	}
	
	return false
}

// Helper methods

func (rc *ResourceCategorizer) getResourceTypeScore(resourceType string) float64 {
	// Critical infrastructure
	critical := []string{"VPC", "Network", "SecurityGroup", "Firewall", "Database", "RDS", "CosmosDB"}
	for _, t := range critical {
		if strings.Contains(resourceType, t) {
			return 40
		}
	}
	
	// Compute resources
	compute := []string{"Instance", "VM", "Container", "Function", "Lambda"}
	for _, t := range compute {
		if strings.Contains(resourceType, t) {
			return 30
		}
	}
	
	// Storage resources
	storage := []string{"Bucket", "Storage", "Volume", "Disk"}
	for _, t := range storage {
		if strings.Contains(resourceType, t) {
			return 25
		}
	}
	
	// Networking resources
	network := []string{"LoadBalancer", "CDN", "DNS", "Route"}
	for _, t := range network {
		if strings.Contains(resourceType, t) {
			return 20
		}
	}
	
	return 10
}

func (rc *ResourceCategorizer) detectEnvironment(resource Resource) string {
	// Check tags
	if tags, ok := resource.Tags.(map[string]string); ok {
		for _, envTag := range []string{"environment", "env", "stage"} {
			if env, exists := tags[envTag]; exists {
				return strings.ToLower(env)
			}
		}
	}
	
	// Check name patterns
	nameLower := strings.ToLower(resource.Name)
	if strings.Contains(nameLower, "prod") {
		return "production"
	}
	if strings.Contains(nameLower, "stag") {
		return "staging"
	}
	if strings.Contains(nameLower, "dev") {
		return "development"
	}
	if strings.Contains(nameLower, "test") || strings.Contains(nameLower, "qa") {
		return "testing"
	}
	
	return ""
}

func (rc *ResourceCategorizer) followsNamingConvention(resource Resource) bool {
	// Check common naming patterns
	nameLower := strings.ToLower(resource.Name)
	
	// Simple pattern checks
	if strings.Contains(nameLower, "-") {
		// Check for structured naming (at least 2 parts)
		parts := strings.Split(nameLower, "-")
		if len(parts) >= 2 {
			// Check if follows common patterns
			if len(parts) >= 3 {
				// Could be env-region-service or similar
				return true
			}
			// Check if has environment prefix
			envPrefixes := []string{"dev", "staging", "prod", "test", "qa"}
			for _, prefix := range envPrefixes {
				if parts[0] == prefix {
					return true
				}
			}
		}
	}
	
	return false
}

func (rc *ResourceCategorizer) getMissingRequiredTags(resource Resource) []string {
	requiredTags := []string{"environment", "owner", "project", "cost-center"}
	missing := []string{}
	
	tags, ok := resource.Tags.(map[string]string)
	if !ok {
		return requiredTags
	}
	
	for _, required := range requiredTags {
		if _, exists := tags[required]; !exists {
			missing = append(missing, required)
		}
	}
	
	return missing
}

func (rc *ResourceCategorizer) findDependencies(resource Resource) []string {
	// This would need actual dependency analysis
	// For now, return placeholder based on resource type
	deps := []string{}
	
	if strings.Contains(resource.Type, "Instance") || strings.Contains(resource.Type, "VM") {
		deps = append(deps, "subnet", "security_group")
	}
	if strings.Contains(resource.Type, "Database") {
		deps = append(deps, "subnet_group", "parameter_group")
	}
	if strings.Contains(resource.Type, "LoadBalancer") {
		deps = append(deps, "target_group", "listener")
	}
	
	return deps
}

func (rc *ResourceCategorizer) getResourceAge(resource Resource) time.Duration {
	// Check if CreatedAt has a valid time
	if !resource.CreatedAt.IsZero() {
		return time.Since(resource.CreatedAt)
	}
	// Fallback to Created field
	if !resource.Created.IsZero() {
		return time.Since(resource.Created)
	}
	return 0
}

func (rc *ResourceCategorizer) hasAutomationTags(resource Resource) bool {
	if tags, ok := resource.Tags.(map[string]string); ok {
		automationTags := []string{"managed_by", "automation", "iac", "terraform", "created_by"}
		for _, tag := range automationTags {
			if _, exists := tags[tag]; exists {
				return true
			}
		}
	}
	return false
}

func (rc *ResourceCategorizer) hasAutomationIndicators(resource Resource) bool {
	// Check for CI/CD or automation tool indicators
	indicators := []string{"jenkins", "github", "gitlab", "circleci", "ansible", "puppet", "chef"}
	nameLower := strings.ToLower(resource.Name)
	
	for _, indicator := range indicators {
		if strings.Contains(nameLower, indicator) {
			return true
		}
	}
	
	return rc.hasAutomationTags(resource)
}

func (rc *ResourceCategorizer) hasSecurityImplications(resource Resource) bool {
	securityTypes := []string{
		"SecurityGroup", "NetworkACL", "Firewall", "IAMRole", "IAMPolicy",
		"Key", "Secret", "Certificate", "WAF", "Shield",
	}
	
	for _, secType := range securityTypes {
		if strings.Contains(resource.Type, secType) {
			return true
		}
	}
	
	return false
}

func (rc *ResourceCategorizer) estimateCost(resource Resource) float64 {
	// Simplified cost estimation based on resource type
	// In reality, would integrate with cloud pricing APIs
	
	costMap := map[string]float64{
		"Instance":     50.0,
		"Database":     100.0,
		"LoadBalancer": 25.0,
		"Storage":      10.0,
		"CDN":          20.0,
		"NAT":          45.0,
	}
	
	for key, cost := range costMap {
		if strings.Contains(resource.Type, key) {
			return cost
		}
	}
	
	return 0.0
}

func (rc *ResourceCategorizer) isConsoleCreated(resource Resource) bool {
	// Would need CloudTrail/Activity Log integration
	// For now, check for console-specific patterns
	
	if tags, ok := resource.Tags.(map[string]string); ok {
		if creator, exists := tags["created_by"]; exists {
			if strings.Contains(strings.ToLower(creator), "console") ||
			   strings.Contains(strings.ToLower(creator), "portal") {
				return true
			}
		}
	}
	
	return false
}

func (rc *ResourceCategorizer) mapToTerraformType(resource Resource) string {
	// Map cloud resource types to Terraform resource types
	provider := strings.ToLower(resource.Provider)
	
	switch provider {
	case "aws":
		return mapAWSToTerraform(resource.Type)
	case "azure":
		return mapAzureToTerraform(resource.Type)
	case "gcp":
		return mapGCPToTerraform(resource.Type)
	default:
		return strings.ToLower(resource.Type)
	}
}

func (rc *ResourceCategorizer) generateImportCommand(resource Resource, tfType string) string {
	return fmt.Sprintf("terraform import %s.%s %s",
		tfType,
		sanitizeResourceName(resource.Name),
		resource.ID)
}

func (rc *ResourceCategorizer) estimateImportImpact(resource Resource) string {
	if rc.hasSecurityImplications(resource) {
		return "HIGH - Security-sensitive resource"
	}
	
	deps := rc.findDependencies(resource)
	if len(deps) > 3 {
		return "HIGH - Multiple dependencies"
	}
	
	if len(deps) > 0 {
		return "MEDIUM - Has dependencies"
	}
	
	return "LOW - Isolated resource"
}

// Helper functions

func mapAWSToTerraform(resourceType string) string {
	mapping := map[string]string{
		"EC2::Instance":       "aws_instance",
		"EC2::SecurityGroup":  "aws_security_group",
		"EC2::VPC":           "aws_vpc",
		"S3::Bucket":         "aws_s3_bucket",
		"RDS::DBInstance":    "aws_db_instance",
	}
	
	if tf, ok := mapping[resourceType]; ok {
		return tf
	}
	
	// Default conversion
	parts := strings.Split(resourceType, "::")
	if len(parts) == 2 {
		return fmt.Sprintf("aws_%s", strings.ToLower(parts[1]))
	}
	return "aws_" + strings.ToLower(strings.ReplaceAll(resourceType, "::", "_"))
}

func mapAzureToTerraform(resourceType string) string {
	mapping := map[string]string{
		"Microsoft.Compute/virtualMachines":       "azurerm_virtual_machine",
		"Microsoft.Network/virtualNetworks":       "azurerm_virtual_network",
		"Microsoft.Network/networkSecurityGroups": "azurerm_network_security_group",
		"Microsoft.Storage/storageAccounts":       "azurerm_storage_account",
	}
	
	if tf, ok := mapping[resourceType]; ok {
		return tf
	}
	
	// Default conversion
	parts := strings.Split(resourceType, "/")
	if len(parts) == 2 {
		return fmt.Sprintf("azurerm_%s", strings.ToLower(parts[1]))
	}
	return "azurerm_" + strings.ToLower(strings.ReplaceAll(resourceType, "/", "_"))
}

func mapGCPToTerraform(resourceType string) string {
	mapping := map[string]string{
		"compute.v1.instance": "google_compute_instance",
		"compute.v1.network":  "google_compute_network",
		"storage.v1.bucket":   "google_storage_bucket",
	}
	
	if tf, ok := mapping[resourceType]; ok {
		return tf
	}
	
	// Default conversion
	parts := strings.Split(resourceType, ".")
	if len(parts) >= 2 {
		return fmt.Sprintf("google_%s_%s", parts[0], parts[len(parts)-1])
	}
	return "google_" + strings.ReplaceAll(resourceType, ".", "_")
}

func sanitizeResourceName(name string) string {
	// Replace invalid characters with underscores
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "/", "_")
	
	// Ensure it starts with a letter
	if len(name) > 0 && (name[0] >= '0' && name[0] <= '9') {
		name = "resource_" + name
	}
	
	return strings.ToLower(name)
}

func getDefaultProfiles() []DiscoveryProfile {
	return []DiscoveryProfile{
		{
			Name: "production",
			ManagedTags: map[string]string{
				"environment": "production",
				"managed_by":  "terraform",
			},
			RequiredTags: []string{"owner", "project", "cost-center"},
			AgeThreshold: 30 * 24 * time.Hour,
		},
		{
			Name: "development",
			ManagedTags: map[string]string{
				"environment": "development",
			},
			RequiredTags: []string{"owner"},
			AgeThreshold: 7 * 24 * time.Hour,
		},
	}
}

func getKnownExclusions() map[string]bool {
	return map[string]bool{
		"AWS::EC2::DHCPOptions":              true,
		"AWS::EC2::InternetGateway::Default": true,
		"AWS::EC2::RouteTable::Default":      true,
		"AWS::EC2::NetworkAcl::Default":      true,
		"AWS::EC2::SecurityGroup::Default":   true,
		"Azure::Network::DefaultNSG":         true,
		"GCP::Compute::DefaultNetwork":       true,
	}
}

func getDefaultTagStrategy() TagStrategy {
	return TagStrategy{
		ManagedByTags:   []string{"managed_by", "managed-by", "managedBy"},
		TerraformTags:   []string{"terraform", "terraform_managed", "tf_managed"},
		EnvironmentTags: []string{"environment", "env", "stage"},
		OwnershipTags:   []string{"owner", "team", "department"},
		AutomationTags:  []string{"automation", "iac", "created_by"},
		ComplianceTags:  []string{"compliance", "data-classification", "pii"},
	}
}