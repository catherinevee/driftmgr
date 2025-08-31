package fingerprint

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// ResourceFingerprinter identifies resource creation patterns
type ResourceFingerprinter struct {
	patterns           map[string]*Pattern
	signatures         map[string]*Signature
	namingConventions  map[string]*NamingConvention
	tagStrategies      map[string]*TagStrategy
	creationMethods    map[string]CreationMethod
	confidenceScores   map[string]float64
}

// Pattern represents a resource creation pattern
type Pattern struct {
	ID               string
	Name             string
	Type             PatternType
	Indicators       []string
	ConfidenceWeight float64
	Examples         []string
}

// PatternType defines the type of pattern
type PatternType string

const (
	PatternTypeTerraform   PatternType = "terraform"
	PatternTypeManual      PatternType = "manual"
	PatternTypeImported    PatternType = "imported"
	PatternTypeAutomation  PatternType = "automation"
	PatternTypeMigrated    PatternType = "migrated"
	PatternTypeUnknown     PatternType = "unknown"
)

// Signature represents a unique signature of resource creation
type Signature struct {
	Hash             string
	CreationMethod   CreationMethod
	ToolFingerprint  string
	TimestampPattern string
	UserAgent        string
	APIVersion       string
	Metadata         map[string]string
}

// CreationMethod identifies how a resource was created
type CreationMethod string

const (
	CreationMethodTerraform      CreationMethod = "terraform"
	CreationMethodConsole        CreationMethod = "console"
	CreationMethodCLI            CreationMethod = "cli"
	CreationMethodSDK            CreationMethod = "sdk"
	CreationMethodCloudFormation CreationMethod = "cloudformation"
	CreationMethodARM            CreationMethod = "arm"
	CreationMethodPulumi         CreationMethod = "pulumi"
	CreationMethodCrossplane     CreationMethod = "crossplane"
	CreationMethodManual         CreationMethod = "manual"
	CreationMethodUnknown        CreationMethod = "unknown"
)

// NamingConvention represents resource naming patterns
type NamingConvention struct {
	Pattern         string
	Regex           *regexp.Regexp
	Components      []NamingComponent
	Separator       string
	CaseStyle       CaseStyle
	Examples        []string
	MatchPercentage float64
}

// NamingComponent represents a component of a naming convention
type NamingComponent struct {
	Position    int
	Type        ComponentType
	Values      []string
	IsOptional  bool
	IsVariable  bool
}

// ComponentType defines the type of naming component
type ComponentType string

const (
	ComponentTypeEnvironment ComponentType = "environment"
	ComponentTypeRegion      ComponentType = "region"
	ComponentTypeService     ComponentType = "service"
	ComponentTypeType        ComponentType = "type"
	ComponentTypeIndex       ComponentType = "index"
	ComponentTypeRandom      ComponentType = "random"
	ComponentTypeDate        ComponentType = "date"
	ComponentTypeOwner       ComponentType = "owner"
)

// CaseStyle defines the case style of names
type CaseStyle string

const (
	CaseStyleKebab      CaseStyle = "kebab-case"
	CaseStyleSnake      CaseStyle = "snake_case"
	CaseStyleCamel      CaseStyle = "camelCase"
	CaseStylePascal     CaseStyle = "PascalCase"
	CaseStyleMixed      CaseStyle = "mixed"
)

// TagStrategy represents tagging patterns
type TagStrategy struct {
	RequiredTags      []string
	CommonTags        map[string]string
	AutomationTags    map[string]string
	EnvironmentTags   map[string]string
	ComplianceTags    map[string]string
	Consistency       float64
	Coverage          float64
}

// FingerprintResult contains the fingerprinting analysis
type FingerprintResult struct {
	ResourceID          string
	Fingerprint         string
	CreationMethod      CreationMethod
	Confidence          float64
	NamingConvention    *NamingConvention
	TagStrategy         *TagStrategy
	Pattern             *Pattern
	Signature           *Signature
	IsTerraformManaged  bool
	IsManuallyCreated   bool
	IsImported          bool
	CreationTimestamp   time.Time
	LastModified        time.Time
	Anomalies           []string
	Recommendations     []string
}

// NewResourceFingerprinter creates a new fingerprinter
func NewResourceFingerprinter() *ResourceFingerprinter {
	rf := &ResourceFingerprinter{
		patterns:          make(map[string]*Pattern),
		signatures:        make(map[string]*Signature),
		namingConventions: make(map[string]*NamingConvention),
		tagStrategies:     make(map[string]*TagStrategy),
		creationMethods:   make(map[string]CreationMethod),
		confidenceScores:  make(map[string]float64),
	}
	
	// Initialize default patterns
	rf.initializePatterns()
	rf.initializeNamingConventions()
	
	return rf
}

// FingerprintResource analyzes a resource to determine its creation pattern
func (rf *ResourceFingerprinter) FingerprintResource(resource models.Resource) *FingerprintResult {
	result := &FingerprintResult{
		ResourceID:        resource.ID,
		CreationTimestamp: resource.CreatedAt,
		LastModified:      resource.LastModified,
		Anomalies:         []string{},
		Recommendations:   []string{},
	}
	
	// Generate fingerprint hash
	result.Fingerprint = rf.generateFingerprint(resource)
	
	// Analyze naming convention
	result.NamingConvention = rf.analyzeNaming(resource.Name)
	
	// Analyze tags
	result.TagStrategy = rf.analyzeTags(resource.Tags)
	
	// Detect creation method
	result.CreationMethod = rf.detectCreationMethod(resource)
	
	// Match against patterns
	result.Pattern = rf.matchPattern(resource)
	
	// Generate signature
	result.Signature = rf.generateSignature(resource)
	
	// Calculate confidence
	result.Confidence = rf.calculateConfidence(result)
	
	// Determine management status
	rf.determineManagedStatus(result)
	
	// Detect anomalies
	result.Anomalies = rf.detectAnomalies(resource, result)
	
	// Generate recommendations
	result.Recommendations = rf.generateRecommendations(result)
	
	return result
}

// generateFingerprint creates a unique fingerprint for the resource
func (rf *ResourceFingerprinter) generateFingerprint(resource models.Resource) string {
	data := fmt.Sprintf("%s:%s:%s:%s:%v",
		resource.Type,
		resource.Provider,
		resource.Name,
		resource.ID,
		resource.Tags)
	
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// analyzeNaming analyzes the resource naming convention
func (rf *ResourceFingerprinter) analyzeNaming(name string) *NamingConvention {
	bestMatch := &NamingConvention{
		Pattern:         "unknown",
		MatchPercentage: 0,
	}
	
	for _, convention := range rf.namingConventions {
		if convention.Regex != nil && convention.Regex.MatchString(name) {
			// Calculate match percentage
			matchScore := rf.calculateNamingMatchScore(name, convention)
			if matchScore > bestMatch.MatchPercentage {
				bestMatch = convention
				bestMatch.MatchPercentage = matchScore
			}
		}
	}
	
	// Detect case style
	bestMatch.CaseStyle = rf.detectCaseStyle(name)
	
	// Extract components
	bestMatch.Components = rf.extractNamingComponents(name, bestMatch)
	
	return bestMatch
}

// detectCaseStyle detects the case style of a name
func (rf *ResourceFingerprinter) detectCaseStyle(name string) CaseStyle {
	if strings.Contains(name, "-") && !strings.Contains(name, "_") {
		return CaseStyleKebab
	}
	if strings.Contains(name, "_") && !strings.Contains(name, "-") {
		return CaseStyleSnake
	}
	if name[0] >= 'A' && name[0] <= 'Z' {
		return CaseStylePascal
	}
	if name[0] >= 'a' && name[0] <= 'z' && strings.ContainsAny(name[1:], "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return CaseStyleCamel
	}
	return CaseStyleMixed
}

// extractNamingComponents extracts components from a resource name
func (rf *ResourceFingerprinter) extractNamingComponents(name string, convention *NamingConvention) []NamingComponent {
	components := []NamingComponent{}
	
	// Split by separator
	separator := "-"
	if convention.CaseStyle == CaseStyleSnake {
		separator = "_"
	}
	
	parts := strings.Split(name, separator)
	for i, part := range parts {
		component := NamingComponent{
			Position: i,
			Type:     rf.identifyComponentType(part),
		}
		
		// Check if it's a known value
		if component.Type == ComponentTypeEnvironment {
			component.Values = []string{"dev", "staging", "prod", "test", "qa"}
			component.IsVariable = false
		} else {
			component.IsVariable = true
		}
		
		components = append(components, component)
	}
	
	return components
}

// identifyComponentType identifies the type of a naming component
func (rf *ResourceFingerprinter) identifyComponentType(part string) ComponentType {
	// Environment markers
	envMarkers := []string{"dev", "development", "staging", "stage", "prod", "production", "test", "qa"}
	for _, marker := range envMarkers {
		if strings.EqualFold(part, marker) {
			return ComponentTypeEnvironment
		}
	}
	
	// Region markers
	if matched, _ := regexp.MatchString(`^[a-z]{2}-[a-z]+-\d+$`, part); matched {
		return ComponentTypeRegion
	}
	
	// Index/number
	if matched, _ := regexp.MatchString(`^\d+$`, part); matched {
		return ComponentTypeIndex
	}
	
	// Date pattern
	if matched, _ := regexp.MatchString(`^\d{8}$|^\d{4}-\d{2}-\d{2}$`, part); matched {
		return ComponentTypeDate
	}
	
	// Random string (hex, uuid part)
	if matched, _ := regexp.MatchString(`^[a-f0-9]{6,}$`, part); matched {
		return ComponentTypeRandom
	}
	
	return ComponentTypeService
}

// analyzeTags analyzes resource tagging strategy
func (rf *ResourceFingerprinter) analyzeTags(tags interface{}) *TagStrategy {
	strategy := &TagStrategy{
		RequiredTags:    []string{},
		CommonTags:      make(map[string]string),
		AutomationTags:  make(map[string]string),
		EnvironmentTags: make(map[string]string),
		ComplianceTags:  make(map[string]string),
	}
	
	tagMap, ok := tags.(map[string]string)
	if !ok || tagMap == nil {
		strategy.Coverage = 0
		return strategy
	}
	
	// Categorize tags
	automationKeys := []string{"managed_by", "terraform", "created_by", "provisioner", "iac", "automation"}
	environmentKeys := []string{"environment", "env", "stage"}
	complianceKeys := []string{"compliance", "data-classification", "pii", "gdpr", "hipaa"}
	
	for key, value := range tagMap {
		keyLower := strings.ToLower(key)
		
		// Check automation tags
		for _, autoKey := range automationKeys {
			if strings.Contains(keyLower, autoKey) {
				strategy.AutomationTags[key] = value
			}
		}
		
		// Check environment tags
		for _, envKey := range environmentKeys {
			if strings.Contains(keyLower, envKey) {
				strategy.EnvironmentTags[key] = value
			}
		}
		
		// Check compliance tags
		for _, compKey := range complianceKeys {
			if strings.Contains(keyLower, compKey) {
				strategy.ComplianceTags[key] = value
			}
		}
		
		strategy.CommonTags[key] = value
	}
	
	// Calculate coverage
	expectedTags := []string{"environment", "owner", "project", "cost-center"}
	presentCount := 0
	for _, expected := range expectedTags {
		if _, exists := tagMap[expected]; exists {
			presentCount++
		}
	}
	strategy.Coverage = float64(presentCount) / float64(len(expectedTags)) * 100
	
	// Calculate consistency (simplified)
	if len(strategy.AutomationTags) > 0 {
		strategy.Consistency = 80
	} else {
		strategy.Consistency = 40
	}
	
	return strategy
}

// detectCreationMethod detects how the resource was created
func (rf *ResourceFingerprinter) detectCreationMethod(resource models.Resource) CreationMethod {
	// Check tags for creation method
	if tags, ok := resource.Tags.(map[string]string); ok {
		// Terraform indicators
		if _, exists := tags["terraform"]; exists {
			return CreationMethodTerraform
		}
		if managedBy, exists := tags["managed_by"]; exists {
			switch strings.ToLower(managedBy) {
			case "terraform":
				return CreationMethodTerraform
			case "console":
				return CreationMethodConsole
			case "cli":
				return CreationMethodCLI
			case "cloudformation":
				return CreationMethodCloudFormation
			case "pulumi":
				return CreationMethodPulumi
			}
		}
		
		// Check for import markers
		if _, exists := tags["imported"]; exists {
			return CreationMethodTerraform // Imported into Terraform
		}
	}
	
	// Check naming patterns
	nameLower := strings.ToLower(resource.Name)
	
	// Terraform patterns
	if strings.Contains(nameLower, "tf-") || strings.Contains(nameLower, "terraform-") {
		return CreationMethodTerraform
	}
	
	// Console patterns (often have "test", "demo", or human-friendly names)
	if strings.Contains(nameLower, "test") || strings.Contains(nameLower, "demo") ||
	   strings.Contains(nameLower, "temp") || strings.Contains(nameLower, "my") {
		return CreationMethodConsole
	}
	
	// Check for consistent naming (likely automation)
	if rf.hasConsistentNaming(resource.Name) {
		return CreationMethodTerraform
	}
	
	return CreationMethodUnknown
}

// hasConsistentNaming checks if name follows consistent pattern
func (rf *ResourceFingerprinter) hasConsistentNaming(name string) bool {
	// Check for structured naming
	separators := 0
	if strings.Count(name, "-") > 1 {
		separators++
	}
	if strings.Count(name, "_") > 1 {
		separators++
	}
	
	return separators > 0
}

// matchPattern matches resource against known patterns
func (rf *ResourceFingerprinter) matchPattern(resource models.Resource) *Pattern {
	var bestMatch *Pattern
	maxScore := 0.0
	
	for _, pattern := range rf.patterns {
		score := rf.calculatePatternMatch(resource, pattern)
		if score > maxScore {
			maxScore = score
			bestMatch = pattern
		}
	}
	
	return bestMatch
}

// calculatePatternMatch calculates how well a resource matches a pattern
func (rf *ResourceFingerprinter) calculatePatternMatch(resource models.Resource, pattern *Pattern) float64 {
	score := 0.0
	matchedIndicators := 0
	
	for _, indicator := range pattern.Indicators {
		if rf.hasIndicator(resource, indicator) {
			matchedIndicators++
		}
	}
	
	if len(pattern.Indicators) > 0 {
		score = float64(matchedIndicators) / float64(len(pattern.Indicators)) * pattern.ConfidenceWeight
	}
	
	return score
}

// hasIndicator checks if resource has a specific indicator
func (rf *ResourceFingerprinter) hasIndicator(resource models.Resource, indicator string) bool {
	// Check in name
	if strings.Contains(strings.ToLower(resource.Name), strings.ToLower(indicator)) {
		return true
	}
	
	// Check in tags
	if tags, ok := resource.Tags.(map[string]string); ok {
		for key, value := range tags {
			if strings.Contains(strings.ToLower(key), strings.ToLower(indicator)) ||
			   strings.Contains(strings.ToLower(value), strings.ToLower(indicator)) {
				return true
			}
		}
	}
	
	// Check in type
	if strings.Contains(strings.ToLower(resource.Type), strings.ToLower(indicator)) {
		return true
	}
	
	return false
}

// generateSignature generates a signature for the resource
func (rf *ResourceFingerprinter) generateSignature(resource models.Resource) *Signature {
	sig := &Signature{
		Hash:            rf.generateFingerprint(resource),
		CreationMethod:  rf.detectCreationMethod(resource),
		Metadata:        make(map[string]string),
	}
	
	// Extract metadata
	if resource.Metadata != nil {
		for key, value := range resource.Metadata {
			sig.Metadata[key] = value
		}
	}
	
	// Detect tool fingerprint
	sig.ToolFingerprint = rf.detectToolFingerprint(resource)
	
	return sig
}

// detectToolFingerprint detects the tool used to create the resource
func (rf *ResourceFingerprinter) detectToolFingerprint(resource models.Resource) string {
	// Check for tool-specific patterns
	if tags, ok := resource.Tags.(map[string]string); ok {
		// Terraform
		if _, exists := tags["terraform"]; exists {
			return "terraform"
		}
		
		// CloudFormation
		if stackName, exists := tags["aws:cloudformation:stack-name"]; exists && stackName != "" {
			return "cloudformation"
		}
		
		// Check for tool-specific tag patterns
		for key := range tags {
			if strings.HasPrefix(key, "tf:") {
				return "terraform"
			}
			if strings.HasPrefix(key, "pulumi:") {
				return "pulumi"
			}
		}
	}
	
	return "unknown"
}

// calculateConfidence calculates confidence in the fingerprint
func (rf *ResourceFingerprinter) calculateConfidence(result *FingerprintResult) float64 {
	confidence := 0.0
	factors := 0
	
	// Naming convention match
	if result.NamingConvention != nil && result.NamingConvention.MatchPercentage > 0 {
		confidence += result.NamingConvention.MatchPercentage
		factors++
	}
	
	// Tag strategy coverage
	if result.TagStrategy != nil {
		confidence += result.TagStrategy.Coverage
		factors++
		
		// Bonus for automation tags
		if len(result.TagStrategy.AutomationTags) > 0 {
			confidence += 20
		}
	}
	
	// Pattern match
	if result.Pattern != nil {
		confidence += result.Pattern.ConfidenceWeight
		factors++
	}
	
	// Creation method certainty
	if result.CreationMethod != CreationMethodUnknown {
		confidence += 30
		factors++
	}
	
	if factors > 0 {
		confidence = confidence / float64(factors)
	}
	
	// Cap at 100
	if confidence > 100 {
		confidence = 100
	}
	
	return confidence
}

// determineManagedStatus determines if resource is managed by Terraform
func (rf *ResourceFingerprinter) determineManagedStatus(result *FingerprintResult) {
	// High confidence Terraform indicators
	if result.CreationMethod == CreationMethodTerraform {
		result.IsTerraformManaged = true
	}
	
	// Check for Terraform tags
	if result.TagStrategy != nil {
		for key := range result.TagStrategy.AutomationTags {
			if strings.Contains(strings.ToLower(key), "terraform") {
				result.IsTerraformManaged = true
				break
			}
		}
	}
	
	// Check pattern
	if result.Pattern != nil && result.Pattern.Type == PatternTypeTerraform {
		result.IsTerraformManaged = true
	}
	
	// Manual creation indicators
	if result.CreationMethod == CreationMethodConsole || result.CreationMethod == CreationMethodManual {
		result.IsManuallyCreated = true
	}
	
	// Import indicators
	if result.Pattern != nil && result.Pattern.Type == PatternTypeImported {
		result.IsImported = true
	}
	
	// Low confidence or unknown
	if result.Confidence < 30 && !result.IsTerraformManaged {
		result.IsManuallyCreated = true
	}
}

// detectAnomalies detects anomalies in the resource
func (rf *ResourceFingerprinter) detectAnomalies(resource models.Resource, result *FingerprintResult) []string {
	anomalies := []string{}
	
	// Naming anomalies
	if result.NamingConvention != nil && result.NamingConvention.MatchPercentage < 50 {
		anomalies = append(anomalies, "Non-standard naming convention")
	}
	
	// Tag anomalies
	if result.TagStrategy != nil {
		if result.TagStrategy.Coverage < 50 {
			anomalies = append(anomalies, "Missing required tags")
		}
		if len(result.TagStrategy.AutomationTags) == 0 && result.IsTerraformManaged {
			anomalies = append(anomalies, "Terraform-managed resource missing automation tags")
		}
	}
	
	// Creation time anomalies
	if !resource.CreatedAt.IsZero() {
		// Check for off-hours creation (potential manual/emergency change)
		hour := resource.CreatedAt.Hour()
		if hour < 6 || hour > 22 {
			anomalies = append(anomalies, "Created outside business hours")
		}
		
		// Check for weekend creation
		weekday := resource.CreatedAt.Weekday()
		if weekday == time.Saturday || weekday == time.Sunday {
			anomalies = append(anomalies, "Created on weekend")
		}
	}
	
	// Mixed patterns
	if result.IsTerraformManaged && result.IsManuallyCreated {
		anomalies = append(anomalies, "Mixed creation patterns detected")
	}
	
	return anomalies
}

// generateRecommendations generates recommendations based on fingerprint
func (rf *ResourceFingerprinter) generateRecommendations(result *FingerprintResult) []string {
	recommendations := []string{}
	
	// Low confidence recommendations
	if result.Confidence < 50 {
		recommendations = append(recommendations, 
			"Low confidence in detection - manual review recommended")
	}
	
	// Unmanaged resource recommendations
	if !result.IsTerraformManaged && result.CreationMethod != CreationMethodTerraform {
		recommendations = append(recommendations,
			"Consider importing this resource into Terraform")
	}
	
	// Tag recommendations
	if result.TagStrategy != nil && result.TagStrategy.Coverage < 100 {
		recommendations = append(recommendations,
			"Add missing required tags for compliance")
	}
	
	// Naming recommendations
	if result.NamingConvention != nil && result.NamingConvention.MatchPercentage < 70 {
		recommendations = append(recommendations,
			"Consider renaming to follow standard naming convention")
	}
	
	// Imported resource recommendations
	if result.IsImported {
		recommendations = append(recommendations,
			"Review imported resource configuration for completeness")
	}
	
	// Anomaly recommendations
	if len(result.Anomalies) > 2 {
		recommendations = append(recommendations,
			"Multiple anomalies detected - thorough review required")
	}
	
	return recommendations
}

// initializePatterns initializes default patterns
func (rf *ResourceFingerprinter) initializePatterns() {
	// Terraform pattern
	rf.patterns["terraform"] = &Pattern{
		ID:   "terraform",
		Name: "Terraform Managed",
		Type: PatternTypeTerraform,
		Indicators: []string{
			"terraform",
			"tf-",
			"managed_by",
			"workspace",
		},
		ConfidenceWeight: 90,
	}
	
	// Manual pattern
	rf.patterns["manual"] = &Pattern{
		ID:   "manual",
		Name: "Manually Created",
		Type: PatternTypeManual,
		Indicators: []string{
			"test",
			"demo",
			"temp",
			"my",
		},
		ConfidenceWeight: 70,
	}
	
	// Imported pattern
	rf.patterns["imported"] = &Pattern{
		ID:   "imported",
		Name: "Imported Resource",
		Type: PatternTypeImported,
		Indicators: []string{
			"imported",
			"migrated",
			"legacy",
		},
		ConfidenceWeight: 80,
	}
}

// initializeNamingConventions initializes naming conventions
func (rf *ResourceFingerprinter) initializeNamingConventions() {
	// Environment-prefixed convention
	rf.namingConventions["env-prefixed"] = &NamingConvention{
		Pattern:   "env-service-type",
		Regex:     regexp.MustCompile(`^(dev|staging|prod|test)-.+-.+$`),
		Separator: "-",
		CaseStyle: CaseStyleKebab,
		Examples:  []string{"prod-api-server", "dev-db-primary"},
	}
	
	// Service-based convention
	rf.namingConventions["service-based"] = &NamingConvention{
		Pattern:   "service-env-region-type",
		Regex:     regexp.MustCompile(`^[a-z]+-[a-z]+-[a-z]{2}-[a-z]+-\d+-[a-z]+$`),
		Separator: "-",
		CaseStyle: CaseStyleKebab,
		Examples:  []string{"api-prod-us-east-1-server"},
	}
	
	// AWS-style convention
	rf.namingConventions["aws-style"] = &NamingConvention{
		Pattern:   "resource-random",
		Regex:     regexp.MustCompile(`^[a-z]+-[a-f0-9]{8}$`),
		Separator: "-",
		CaseStyle: CaseStyleKebab,
		Examples:  []string{"instance-a1b2c3d4", "bucket-12345678"},
	}
}

// calculateNamingMatchScore calculates how well a name matches a convention
func (rf *ResourceFingerprinter) calculateNamingMatchScore(name string, convention *NamingConvention) float64 {
	score := 0.0
	
	// Regex match
	if convention.Regex != nil && convention.Regex.MatchString(name) {
		score += 50
	}
	
	// Case style match
	if rf.detectCaseStyle(name) == convention.CaseStyle {
		score += 25
	}
	
	// Separator match
	if strings.Contains(name, convention.Separator) {
		score += 25
	}
	
	return score
}

// CompareFingerprints compares two resource fingerprints
func (rf *ResourceFingerprinter) CompareFingerprints(fp1, fp2 *FingerprintResult) float64 {
	similarity := 0.0
	factors := 0
	
	// Compare creation methods
	if fp1.CreationMethod == fp2.CreationMethod {
		similarity += 30
	}
	factors++
	
	// Compare naming conventions
	if fp1.NamingConvention != nil && fp2.NamingConvention != nil {
		if fp1.NamingConvention.Pattern == fp2.NamingConvention.Pattern {
			similarity += 25
		}
	}
	factors++
	
	// Compare tag strategies
	if fp1.TagStrategy != nil && fp2.TagStrategy != nil {
		// Compare common tags
		commonTags := 0
		for key := range fp1.TagStrategy.CommonTags {
			if _, exists := fp2.TagStrategy.CommonTags[key]; exists {
				commonTags++
			}
		}
		
		if len(fp1.TagStrategy.CommonTags) > 0 {
			tagSimilarity := float64(commonTags) / float64(len(fp1.TagStrategy.CommonTags)) * 25
			similarity += tagSimilarity
		}
	}
	factors++
	
	// Compare patterns
	if fp1.Pattern != nil && fp2.Pattern != nil {
		if fp1.Pattern.Type == fp2.Pattern.Type {
			similarity += 20
		}
	}
	factors++
	
	if factors > 0 {
		similarity = similarity / float64(factors)
	}
	
	return similarity
}

// BatchFingerprint fingerprints multiple resources
func (rf *ResourceFingerprinter) BatchFingerprint(resources []models.Resource) map[string]*FingerprintResult {
	results := make(map[string]*FingerprintResult)
	
	for _, resource := range resources {
		result := rf.FingerprintResource(resource)
		results[resource.ID] = result
		
		// Store for pattern learning
		rf.confidenceScores[resource.ID] = result.Confidence
		rf.creationMethods[resource.ID] = result.CreationMethod
	}
	
	return results
}

// LearnPatterns learns patterns from a set of known resources
func (rf *ResourceFingerprinter) LearnPatterns(knownResources map[string]models.Resource, knownMethods map[string]CreationMethod) {
	// Group resources by creation method
	groups := make(map[CreationMethod][]models.Resource)
	for id, resource := range knownResources {
		if method, ok := knownMethods[id]; ok {
			groups[method] = append(groups[method], resource)
		}
	}
	
	// Learn patterns for each group
	for method, resources := range groups {
		pattern := rf.extractPattern(resources, method)
		if pattern != nil {
			patternID := fmt.Sprintf("learned_%s_%d", method, time.Now().Unix())
			rf.patterns[patternID] = pattern
		}
	}
}

// extractPattern extracts a pattern from a group of resources
func (rf *ResourceFingerprinter) extractPattern(resources []models.Resource, method CreationMethod) *Pattern {
	if len(resources) == 0 {
		return nil
	}
	
	// Find common indicators
	indicatorCounts := make(map[string]int)
	
	for _, resource := range resources {
		// Extract potential indicators from name
		nameParts := strings.FieldsFunc(resource.Name, func(r rune) bool {
			return r == '-' || r == '_' || r == '.'
		})
		
		for _, part := range nameParts {
			if len(part) > 2 { // Skip very short parts
				indicatorCounts[strings.ToLower(part)]++
			}
		}
		
		// Extract from tags
		if tags, ok := resource.Tags.(map[string]string); ok {
			for key, value := range tags {
				indicatorCounts[strings.ToLower(key)]++
				if len(value) < 50 { // Skip long values
					indicatorCounts[strings.ToLower(value)]++
				}
			}
		}
	}
	
	// Find indicators present in at least 50% of resources
	threshold := len(resources) / 2
	indicators := []string{}
	
	for indicator, count := range indicatorCounts {
		if count >= threshold {
			indicators = append(indicators, indicator)
		}
	}
	
	// Sort indicators for consistency
	sort.Strings(indicators)
	
	if len(indicators) == 0 {
		return nil
	}
	
	return &Pattern{
		ID:               fmt.Sprintf("learned_%s", method),
		Name:             fmt.Sprintf("Learned %s Pattern", method),
		Type:             PatternTypeUnknown,
		Indicators:       indicators,
		ConfidenceWeight: 70, // Lower weight for learned patterns
	}
}