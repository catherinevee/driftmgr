package cache

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// NewDataEnricher creates a new data enricher
func NewDataEnricher() *DataEnricher {
	enricher := &DataEnricher{
		enrichers: make([]Enricher, 0),
	}
	
	// Initialize default enrichers
	enricher.initializeEnrichers()
	
	return enricher
}

// initializeEnrichers sets up default enrichers
func (de *DataEnricher) initializeEnrichers() {
	// Add enrichers in order of execution
	de.AddEnricher(&MetadataEnricher{})
	de.AddEnricher(&TagEnricher{})
	de.AddEnricher(&CostEnricher{})
	de.AddEnricher(&ComplianceEnricher{})
	de.AddEnricher(&PerformanceEnricher{})
	de.AddEnricher(&OwnershipEnricher{})
	de.AddEnricher(&TerraformEnricher{})
	de.AddEnricher(&NetworkContextEnricher{})
	de.AddEnricher(&SecurityEnricher{})
	de.AddEnricher(&DriftAnalysisEnricher{})
}

// AddEnricher adds an enricher to the pipeline
func (de *DataEnricher) AddEnricher(enricher Enricher) {
	de.mu.Lock()
	defer de.mu.Unlock()
	
	de.enrichers = append(de.enrichers, enricher)
}

// Enrich applies all enrichers to a cache entry
func (de *DataEnricher) Enrich(entry *CacheEntry) error {
	de.mu.RLock()
	enrichers := make([]Enricher, len(de.enrichers))
	copy(enrichers, de.enrichers)
	de.mu.RUnlock()
	
	// Apply each enricher
	for _, enricher := range enrichers {
		if err := enricher.Enrich(entry); err != nil {
			// Log error but continue with other enrichers
			fmt.Printf("Enricher %s failed: %v\n", enricher.Type(), err)
		}
	}
	
	return nil
}

// MetadataEnricher enriches entries with additional metadata
type MetadataEnricher struct{}

func (e *MetadataEnricher) Enrich(entry *CacheEntry) error {
	if entry.Metadata == nil {
		entry.Metadata = &EntryMetadata{
			Custom: make(map[string]string),
		}
	}
	
	// Extract and enrich metadata from value
	if data, ok := entry.Value.(map[string]interface{}); ok {
		// Add resource name
		if name := extractField(data, "name", "resource_name", "id"); name != "" {
			entry.Metadata.Custom["name"] = name
		}
		
		// Add creation timestamp
		if created := extractTimeField(data, "created_at", "creation_time", "created"); created != nil {
			entry.Metadata.Custom["created_at"] = created.Format(time.RFC3339)
		}
		
		// Add last modified
		if modified := extractTimeField(data, "modified_at", "last_modified", "updated_at"); modified != nil {
			entry.Metadata.Custom["modified_at"] = modified.Format(time.RFC3339)
		}
		
		// Add state
		if state := extractField(data, "state", "status", "lifecycle_state"); state != "" {
			entry.Metadata.Custom["state"] = state
		}
		
		// Add size/capacity information
		if size := extractNumericField(data, "size", "capacity", "count"); size > 0 {
			entry.Metadata.Custom["size"] = fmt.Sprintf("%d", size)
		}
		
		// Add location information
		if location := extractField(data, "location", "region", "zone", "availability_zone"); location != "" {
			entry.Metadata.Custom["location"] = location
		}
		
		// Add environment
		entry.Metadata.Custom["environment"] = detectEnvironment(entry)
		
		// Add criticality score
		entry.Metadata.Custom["criticality"] = calculateCriticality(entry)
	}
	
	return nil
}

func (e *MetadataEnricher) Type() string {
	return "metadata"
}

// TagEnricher enriches entries with normalized tags
type TagEnricher struct{}

func (e *TagEnricher) Enrich(entry *CacheEntry) error {
	if entry.Tags == nil {
		entry.Tags = make(map[string]string)
	}
	
	// Extract tags from value
	if data, ok := entry.Value.(map[string]interface{}); ok {
		// Look for tag fields
		tagFields := []string{"tags", "labels", "metadata"}
		
		for _, field := range tagFields {
			if tags, exists := data[field]; exists {
				switch t := tags.(type) {
				case map[string]interface{}:
					for k, v := range t {
						entry.Tags[normalizeTagKey(k)] = fmt.Sprintf("%v", v)
					}
				case map[string]string:
					for k, v := range t {
						entry.Tags[normalizeTagKey(k)] = v
					}
				}
			}
		}
		
		// Add standard tags
		e.addStandardTags(entry)
		
		// Validate and clean tags
		e.validateTags(entry)
	}
	
	return nil
}

func (e *TagEnricher) Type() string {
	return "tags"
}

func (e *TagEnricher) addStandardTags(entry *CacheEntry) {
	// Add provider tag
	if entry.Metadata != nil && entry.Metadata.Provider != "" {
		entry.Tags["provider"] = entry.Metadata.Provider
	}
	
	// Add resource type tag
	if entry.Metadata != nil && entry.Metadata.ResourceType != "" {
		entry.Tags["resource_type"] = entry.Metadata.ResourceType
	}
	
	// Add managed_by tag
	if _, exists := entry.Tags["managed_by"]; !exists {
		entry.Tags["managed_by"] = detectManagementTool(entry)
	}
	
	// Add cost_center if not present
	if _, exists := entry.Tags["cost_center"]; !exists {
		entry.Tags["cost_center"] = inferCostCenter(entry)
	}
}

func (e *TagEnricher) validateTags(entry *CacheEntry) {
	// Remove invalid tags
	for k, v := range entry.Tags {
		if !isValidTagKey(k) || !isValidTagValue(v) {
			delete(entry.Tags, k)
		}
	}
	
	// Ensure required tags
	requiredTags := []string{"environment", "owner", "project"}
	for _, tag := range requiredTags {
		if _, exists := entry.Tags[tag]; !exists {
			entry.Tags[tag] = "unknown"
		}
	}
}

// CostEnricher enriches entries with cost information
type CostEnricher struct {
	costCalculator CostCalculator
}

func (e *CostEnricher) Enrich(entry *CacheEntry) error {
	if entry.Metrics == nil {
		entry.Metrics = make(map[string]float64)
	}
	
	// Calculate cost based on resource type and configuration
	cost := e.calculateResourceCost(entry)
	entry.Metrics["hourly_cost"] = cost
	entry.Metrics["daily_cost"] = cost * 24
	entry.Metrics["monthly_cost"] = cost * 24 * 30
	
	// Add cost optimization score
	entry.Metrics["cost_optimization_score"] = e.calculateOptimizationScore(entry)
	
	// Add waste indicator
	entry.Metrics["waste_percentage"] = e.calculateWaste(entry)
	
	// Add cost trend
	entry.Enrichments["cost_trend"] = e.calculateCostTrend(entry)
	
	return nil
}

func (e *CostEnricher) Type() string {
	return "cost"
}

func (e *CostEnricher) calculateResourceCost(entry *CacheEntry) float64 {
	if entry.Metadata == nil {
		return 0.0
	}
	
	// Basic cost calculation based on resource type
	// This would be replaced with actual pricing API calls
	baseCosts := map[string]float64{
		"ec2_instance":     0.10,
		"rds_instance":     0.15,
		"load_balancer":    0.025,
		"storage_bucket":   0.023,
		"lambda_function":  0.0000002,
		"container":        0.05,
		"vm_instance":      0.08,
		"sql_database":     0.20,
	}
	
	resourceType := strings.ToLower(entry.Metadata.ResourceType)
	baseCost := 0.0
	
	for key, cost := range baseCosts {
		if strings.Contains(resourceType, key) {
			baseCost = cost
			break
		}
	}
	
	// Adjust based on size/capacity
	if sizeStr, exists := entry.Metadata.Custom["size"]; exists {
		if size := parseSize(sizeStr); size > 0 {
			baseCost *= float64(size) / 100
		}
	}
	
	return baseCost
}

func (e *CostEnricher) calculateOptimizationScore(entry *CacheEntry) float64 {
	score := 100.0
	
	// Check utilization metrics
	if util, exists := entry.Metrics["cpu_utilization"]; exists && util < 10 {
		score -= 30
	}
	
	// Check if resource is idle
	if lastUsed, exists := entry.Metadata.Custom["last_used"]; exists {
		if t, err := time.Parse(time.RFC3339, lastUsed); err == nil {
			if time.Since(t) > 7*24*time.Hour {
				score -= 40
			}
		}
	}
	
	// Check for reserved capacity
	if entry.Tags["reserved"] != "true" && entry.Metrics["monthly_cost"] > 100 {
		score -= 20
	}
	
	if score < 0 {
		score = 0
	}
	
	return score
}

func (e *CostEnricher) calculateWaste(entry *CacheEntry) float64 {
	waste := 0.0
	
	// Check for unused resources
	if state, exists := entry.Metadata.Custom["state"]; exists {
		if state == "stopped" || state == "idle" {
			waste = 100.0
		}
	}
	
	// Check for oversized resources
	if util, exists := entry.Metrics["avg_utilization"]; exists && util < 20 {
		waste = 80.0
	}
	
	return waste
}

func (e *CostEnricher) calculateCostTrend(entry *CacheEntry) map[string]interface{} {
	return map[string]interface{}{
		"direction": "stable",
		"change_percentage": 0.0,
		"projection_30d": entry.Metrics["monthly_cost"],
	}
}

// ComplianceEnricher enriches entries with compliance information
type ComplianceEnricher struct{}

func (e *ComplianceEnricher) Enrich(entry *CacheEntry) error {
	if entry.Enrichments == nil {
		entry.Enrichments = make(map[string]interface{})
	}
	
	// Check compliance violations
	violations := e.checkCompliance(entry)
	entry.Enrichments["compliance_violations"] = violations
	
	// Calculate compliance score
	score := 100.0 - float64(len(violations)*10)
	if score < 0 {
		score = 0
	}
	entry.Metrics["compliance_score"] = score
	
	// Add compliance frameworks
	entry.Enrichments["compliance_frameworks"] = e.applicableFrameworks(entry)
	
	// Add risk level
	entry.Enrichments["risk_level"] = e.calculateRiskLevel(violations)
	
	return nil
}

func (e *ComplianceEnricher) Type() string {
	return "compliance"
}

func (e *ComplianceEnricher) checkCompliance(entry *CacheEntry) []ComplianceViolation {
	violations := make([]ComplianceViolation, 0)
	
	// Check encryption
	if !e.isEncrypted(entry) {
		violations = append(violations, ComplianceViolation{
			Rule:     "encryption-at-rest",
			Severity: "high",
			Message:  "Resource is not encrypted at rest",
		})
	}
	
	// Check public access
	if e.isPubliclyAccessible(entry) {
		violations = append(violations, ComplianceViolation{
			Rule:     "no-public-access",
			Severity: "critical",
			Message:  "Resource is publicly accessible",
		})
	}
	
	// Check backup configuration
	if !e.hasBackup(entry) {
		violations = append(violations, ComplianceViolation{
			Rule:     "backup-required",
			Severity: "medium",
			Message:  "Resource does not have backup configured",
		})
	}
	
	// Check tags
	if !e.hasRequiredTags(entry) {
		violations = append(violations, ComplianceViolation{
			Rule:     "required-tags",
			Severity: "low",
			Message:  "Resource missing required tags",
		})
	}
	
	// Check password/key rotation
	if e.needsRotation(entry) {
		violations = append(violations, ComplianceViolation{
			Rule:     "credential-rotation",
			Severity: "high",
			Message:  "Credentials need rotation",
		})
	}
	
	return violations
}

func (e *ComplianceEnricher) isEncrypted(entry *CacheEntry) bool {
	if data, ok := entry.Value.(map[string]interface{}); ok {
		encrypted := extractBoolField(data, "encrypted", "encryption_enabled", "kms_key_id")
		return encrypted
	}
	return false
}

func (e *ComplianceEnricher) isPubliclyAccessible(entry *CacheEntry) bool {
	if data, ok := entry.Value.(map[string]interface{}); ok {
		return extractBoolField(data, "public", "publicly_accessible", "public_access")
	}
	return false
}

func (e *ComplianceEnricher) hasBackup(entry *CacheEntry) bool {
	if data, ok := entry.Value.(map[string]interface{}); ok {
		return extractBoolField(data, "backup_enabled", "backup_retention", "snapshot_enabled")
	}
	return false
}

func (e *ComplianceEnricher) hasRequiredTags(entry *CacheEntry) bool {
	requiredTags := []string{"environment", "owner", "cost_center", "project"}
	
	for _, tag := range requiredTags {
		if val, exists := entry.Tags[tag]; !exists || val == "unknown" || val == "" {
			return false
		}
	}
	
	return true
}

func (e *ComplianceEnricher) needsRotation(entry *CacheEntry) bool {
	if lastRotated, exists := entry.Metadata.Custom["last_rotated"]; exists {
		if t, err := time.Parse(time.RFC3339, lastRotated); err == nil {
			return time.Since(t) > 90*24*time.Hour
		}
	}
	
	// Check if resource type typically has credentials
	if entry.Metadata != nil {
		resourceType := strings.ToLower(entry.Metadata.ResourceType)
		credentialTypes := []string{"key", "secret", "password", "token", "certificate"}
		
		for _, ct := range credentialTypes {
			if strings.Contains(resourceType, ct) {
				return true // Assume needs rotation if we can't determine last rotation
			}
		}
	}
	
	return false
}

func (e *ComplianceEnricher) applicableFrameworks(entry *CacheEntry) []string {
	frameworks := make([]string, 0)
	
	// Determine applicable frameworks based on resource type and tags
	if entry.Tags["industry"] == "healthcare" || entry.Tags["data_classification"] == "phi" {
		frameworks = append(frameworks, "HIPAA")
	}
	
	if entry.Tags["industry"] == "finance" || entry.Tags["data_classification"] == "pci" {
		frameworks = append(frameworks, "PCI-DSS")
	}
	
	if entry.Tags["region"] == "eu" || strings.Contains(entry.Tags["location"], "eu-") {
		frameworks = append(frameworks, "GDPR")
	}
	
	// Default frameworks
	frameworks = append(frameworks, "SOC2", "ISO27001")
	
	return frameworks
}

func (e *ComplianceEnricher) calculateRiskLevel(violations []ComplianceViolation) string {
	criticalCount := 0
	highCount := 0
	
	for _, v := range violations {
		switch v.Severity {
		case "critical":
			criticalCount++
		case "high":
			highCount++
		}
	}
	
	if criticalCount > 0 {
		return "critical"
	} else if highCount > 2 {
		return "high"
	} else if len(violations) > 5 {
		return "medium"
	} else if len(violations) > 0 {
		return "low"
	}
	
	return "none"
}

// PerformanceEnricher enriches entries with performance metrics
type PerformanceEnricher struct{}

func (e *PerformanceEnricher) Enrich(entry *CacheEntry) error {
	// Add performance metrics
	e.addPerformanceMetrics(entry)
	
	// Add performance score
	entry.Metrics["performance_score"] = e.calculatePerformanceScore(entry)
	
	// Add performance trends
	entry.Enrichments["performance_trends"] = e.calculateTrends(entry)
	
	// Add optimization recommendations
	entry.Enrichments["performance_recommendations"] = e.generateRecommendations(entry)
	
	return nil
}

func (e *PerformanceEnricher) Type() string {
	return "performance"
}

func (e *PerformanceEnricher) addPerformanceMetrics(entry *CacheEntry) {
	if entry.Metrics == nil {
		entry.Metrics = make(map[string]float64)
	}
	
	// Simulate performance metrics (would be fetched from monitoring systems)
	if isComputeResource(entry.Metadata.ResourceType) {
		entry.Metrics["cpu_utilization"] = 45.5
		entry.Metrics["memory_utilization"] = 62.3
		entry.Metrics["disk_io"] = 120.5
		entry.Metrics["network_io"] = 85.2
		entry.Metrics["response_time_ms"] = 125.0
	}
	
	if isStorageResource(entry.Metadata.ResourceType) {
		entry.Metrics["read_iops"] = 1500.0
		entry.Metrics["write_iops"] = 800.0
		entry.Metrics["throughput_mbps"] = 125.0
		entry.Metrics["latency_ms"] = 2.5
	}
	
	if isDatabaseResource(entry.Metadata.ResourceType) {
		entry.Metrics["query_latency_ms"] = 15.0
		entry.Metrics["connections_active"] = 25.0
		entry.Metrics["transactions_per_second"] = 150.0
		entry.Metrics["deadlocks"] = 0.0
	}
}

func (e *PerformanceEnricher) calculatePerformanceScore(entry *CacheEntry) float64 {
	score := 100.0
	
	// Check CPU utilization
	if cpu, exists := entry.Metrics["cpu_utilization"]; exists {
		if cpu > 90 {
			score -= 30
		} else if cpu < 10 {
			score -= 20 // Underutilized
		}
	}
	
	// Check memory utilization
	if mem, exists := entry.Metrics["memory_utilization"]; exists {
		if mem > 90 {
			score -= 25
		}
	}
	
	// Check response time
	if rt, exists := entry.Metrics["response_time_ms"]; exists {
		if rt > 1000 {
			score -= 40
		} else if rt > 500 {
			score -= 20
		}
	}
	
	if score < 0 {
		score = 0
	}
	
	return score
}

func (e *PerformanceEnricher) calculateTrends(entry *CacheEntry) map[string]interface{} {
	return map[string]interface{}{
		"cpu_trend":        "stable",
		"memory_trend":     "increasing",
		"latency_trend":    "stable",
		"error_rate_trend": "decreasing",
	}
}

func (e *PerformanceEnricher) generateRecommendations(entry *CacheEntry) []string {
	recommendations := make([]string, 0)
	
	// Check for performance issues
	if cpu, exists := entry.Metrics["cpu_utilization"]; exists && cpu > 80 {
		recommendations = append(recommendations, "Consider scaling up CPU resources")
	}
	
	if mem, exists := entry.Metrics["memory_utilization"]; exists && mem > 85 {
		recommendations = append(recommendations, "Consider increasing memory allocation")
	}
	
	if rt, exists := entry.Metrics["response_time_ms"]; exists && rt > 500 {
		recommendations = append(recommendations, "Investigate high response times")
	}
	
	if cpu, exists := entry.Metrics["cpu_utilization"]; exists && cpu < 20 {
		recommendations = append(recommendations, "Consider downsizing to reduce costs")
	}
	
	return recommendations
}

// Additional enrichers...

// OwnershipEnricher enriches entries with ownership information
type OwnershipEnricher struct{}

func (e *OwnershipEnricher) Enrich(entry *CacheEntry) error {
	if entry.Enrichments == nil {
		entry.Enrichments = make(map[string]interface{})
	}
	
	// Determine ownership
	owner := e.determineOwner(entry)
	entry.Enrichments["owner"] = owner
	entry.Enrichments["team"] = e.determineTeam(entry)
	entry.Enrichments["contact"] = e.determineContact(owner)
	entry.Enrichments["escalation_path"] = e.getEscalationPath(owner)
	
	return nil
}

func (e *OwnershipEnricher) Type() string {
	return "ownership"
}

func (e *OwnershipEnricher) determineOwner(entry *CacheEntry) string {
	// Check tags first
	if owner, exists := entry.Tags["owner"]; exists && owner != "unknown" {
		return owner
	}
	
	// Check metadata
	if entry.Metadata != nil && entry.Metadata.Custom != nil {
		if owner, exists := entry.Metadata.Custom["created_by"]; exists {
			return owner
		}
	}
	
	// Infer from resource name
	if entry.Metadata != nil && entry.Metadata.Custom != nil {
		if name, exists := entry.Metadata.Custom["name"]; exists {
			// Extract team prefix from resource name
			parts := strings.Split(name, "-")
			if len(parts) > 0 {
				return parts[0] + "-team"
			}
		}
	}
	
	return "unknown"
}

func (e *OwnershipEnricher) determineTeam(entry *CacheEntry) string {
	// Map owners to teams
	owner := e.determineOwner(entry)
	
	teamMap := map[string]string{
		"platform-team":  "Platform Engineering",
		"data-team":      "Data Engineering",
		"security-team":  "Security",
		"devops-team":    "DevOps",
		"frontend-team":  "Frontend",
		"backend-team":   "Backend",
	}
	
	if team, exists := teamMap[owner]; exists {
		return team
	}
	
	// Check tags
	if team, exists := entry.Tags["team"]; exists {
		return team
	}
	
	return "Unknown Team"
}

func (e *OwnershipEnricher) determineContact(owner string) string {
	// Map owners to contact information
	contactMap := map[string]string{
		"platform-team": "platform@company.com",
		"data-team":     "data@company.com",
		"security-team": "security@company.com",
	}
	
	if contact, exists := contactMap[owner]; exists {
		return contact
	}
	
	return "ops@company.com"
}

func (e *OwnershipEnricher) getEscalationPath(owner string) []string {
	// Define escalation paths
	return []string{
		owner,
		"team-lead",
		"engineering-manager",
		"cto",
	}
}

// Helper functions

func normalizeTagKey(key string) string {
	// Convert to lowercase and replace spaces with underscores
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, " ", "_")
	key = strings.ReplaceAll(key, "-", "_")
	
	// Remove invalid characters
	re := regexp.MustCompile(`[^a-z0-9_]`)
	key = re.ReplaceAllString(key, "")
	
	return key
}

func isValidTagKey(key string) bool {
	if len(key) == 0 || len(key) > 128 {
		return false
	}
	
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_.-]+$`, key)
	return matched
}

func isValidTagValue(value string) bool {
	return len(value) <= 256
}

func detectEnvironment(entry *CacheEntry) string {
	// Check tags
	envTags := []string{"environment", "env", "stage"}
	for _, tag := range envTags {
		if val, exists := entry.Tags[tag]; exists {
			return strings.ToLower(val)
		}
	}
	
	// Check resource name
	if entry.Metadata != nil && entry.Metadata.Custom != nil {
		if name, exists := entry.Metadata.Custom["name"]; exists {
			nameLower := strings.ToLower(name)
			
			envPatterns := map[string]string{
				"prod":    "production",
				"staging": "staging",
				"stg":     "staging",
				"dev":     "development",
				"test":    "testing",
				"qa":      "qa",
			}
			
			for pattern, env := range envPatterns {
				if strings.Contains(nameLower, pattern) {
					return env
				}
			}
		}
	}
	
	return "unknown"
}

func calculateCriticality(entry *CacheEntry) string {
	score := 0
	
	// Check environment
	env := detectEnvironment(entry)
	if env == "production" {
		score += 3
	} else if env == "staging" {
		score += 2
	} else {
		score += 1
	}
	
	// Check resource type
	if entry.Metadata != nil {
		resourceType := strings.ToLower(entry.Metadata.ResourceType)
		
		criticalTypes := []string{"database", "load_balancer", "gateway", "firewall"}
		for _, ct := range criticalTypes {
			if strings.Contains(resourceType, ct) {
				score += 2
				break
			}
		}
	}
	
	// Check tags
	if dataClass, exists := entry.Tags["data_classification"]; exists {
		if dataClass == "confidential" || dataClass == "restricted" {
			score += 3
		}
	}
	
	if score >= 6 {
		return "critical"
	} else if score >= 4 {
		return "high"
	} else if score >= 2 {
		return "medium"
	}
	
	return "low"
}

func detectManagementTool(entry *CacheEntry) string {
	// Check for Terraform management
	if entry.Tags["terraform"] == "true" || entry.Tags["managed_by"] == "terraform" {
		return "terraform"
	}
	
	// Check for CloudFormation
	if _, exists := entry.Tags["aws:cloudformation:stack-name"]; exists {
		return "cloudformation"
	}
	
	// Check for Ansible
	if entry.Tags["ansible_managed"] == "true" {
		return "ansible"
	}
	
	// Check for Kubernetes
	if entry.Tags["kubernetes.io/created-by"] != "" {
		return "kubernetes"
	}
	
	return "manual"
}

func inferCostCenter(entry *CacheEntry) string {
	// Check tags
	if cc, exists := entry.Tags["cost_center"]; exists && cc != "" {
		return cc
	}
	
	// Infer from team/owner
	if team, exists := entry.Tags["team"]; exists {
		costCenterMap := map[string]string{
			"platform":  "CC-1001",
			"data":      "CC-1002",
			"security":  "CC-1003",
			"frontend":  "CC-2001",
			"backend":   "CC-2002",
		}
		
		for key, cc := range costCenterMap {
			if strings.Contains(strings.ToLower(team), key) {
				return cc
			}
		}
	}
	
	// Infer from environment
	env := detectEnvironment(entry)
	if env == "production" {
		return "CC-PROD"
	} else if env == "development" {
		return "CC-DEV"
	}
	
	return "CC-GENERAL"
}

// Helper type definitions

type ComplianceViolation struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type CostCalculator interface {
	CalculateCost(resourceType string, config map[string]interface{}) float64
}

// Utility functions for field extraction

func extractField(data map[string]interface{}, fields ...string) string {
	for _, field := range fields {
		if val, exists := data[field]; exists {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}
	return ""
}

func extractTimeField(data map[string]interface{}, fields ...string) *time.Time {
	for _, field := range fields {
		if val, exists := data[field]; exists {
			switch v := val.(type) {
			case string:
				if t, err := time.Parse(time.RFC3339, v); err == nil {
					return &t
				}
			case time.Time:
				return &v
			}
		}
	}
	return nil
}

func extractNumericField(data map[string]interface{}, fields ...string) int {
	for _, field := range fields {
		if val, exists := data[field]; exists {
			switch v := val.(type) {
			case int:
				return v
			case int64:
				return int(v)
			case float64:
				return int(v)
			}
		}
	}
	return 0
}

func extractBoolField(data map[string]interface{}, fields ...string) bool {
	for _, field := range fields {
		if val, exists := data[field]; exists {
			switch v := val.(type) {
			case bool:
				return v
			case string:
				return v == "true" || v == "enabled" || v != ""
			}
		}
	}
	return false
}

func parseSize(sizeStr string) int {
	// Parse size strings like "100GB", "2TB", etc.
	// Simplified implementation
	var size int
	fmt.Sscanf(sizeStr, "%d", &size)
	return size
}

func isComputeResource(resourceType string) bool {
	computeTypes := []string{"instance", "vm", "container", "function", "compute"}
	resourceTypeLower := strings.ToLower(resourceType)
	
	for _, ct := range computeTypes {
		if strings.Contains(resourceTypeLower, ct) {
			return true
		}
	}
	
	return false
}

func isStorageResource(resourceType string) bool {
	storageTypes := []string{"storage", "bucket", "volume", "disk", "filesystem"}
	resourceTypeLower := strings.ToLower(resourceType)
	
	for _, st := range storageTypes {
		if strings.Contains(resourceTypeLower, st) {
			return true
		}
	}
	
	return false
}

func isDatabaseResource(resourceType string) bool {
	dbTypes := []string{"database", "rds", "sql", "nosql", "dynamodb", "cosmos"}
	resourceTypeLower := strings.ToLower(resourceType)
	
	for _, dt := range dbTypes {
		if strings.Contains(resourceTypeLower, dt) {
			return true
		}
	}
	
	return false
}

// TerraformEnricher enriches entries with Terraform-specific information
type TerraformEnricher struct{}

func (e *TerraformEnricher) Enrich(entry *CacheEntry) error {
	if entry.Enrichments == nil {
		entry.Enrichments = make(map[string]interface{})
	}
	
	// Add Terraform state information
	entry.Enrichments["terraform_managed"] = e.isTerraformManaged(entry)
	entry.Enrichments["terraform_module"] = e.extractModule(entry)
	entry.Enrichments["terraform_workspace"] = e.extractWorkspace(entry)
	entry.Enrichments["terraform_version"] = e.extractTerraformVersion(entry)
	
	return nil
}

func (e *TerraformEnricher) Type() string {
	return "terraform"
}

func (e *TerraformEnricher) isTerraformManaged(entry *CacheEntry) bool {
	return entry.Tags["terraform"] == "true" || 
	       entry.Tags["managed_by"] == "terraform" ||
	       strings.Contains(entry.Key, "terraform")
}

func (e *TerraformEnricher) extractModule(entry *CacheEntry) string {
	if module, exists := entry.Tags["terraform_module"]; exists {
		return module
	}
	
	// Try to extract from resource ID
	if entry.Metadata != nil && entry.Metadata.Custom != nil {
		if id, exists := entry.Metadata.Custom["terraform_id"]; exists {
			parts := strings.Split(id, ".")
			if len(parts) > 1 {
				return parts[0]
			}
		}
	}
	
	return ""
}

func (e *TerraformEnricher) extractWorkspace(entry *CacheEntry) string {
	if workspace, exists := entry.Tags["terraform_workspace"]; exists {
		return workspace
	}
	
	return "default"
}

func (e *TerraformEnricher) extractTerraformVersion(entry *CacheEntry) string {
	if version, exists := entry.Tags["terraform_version"]; exists {
		return version
	}
	
	return ""
}

// NetworkContextEnricher enriches entries with network context
type NetworkContextEnricher struct{}

func (e *NetworkContextEnricher) Enrich(entry *CacheEntry) error {
	if entry.Enrichments == nil {
		entry.Enrichments = make(map[string]interface{})
	}
	
	// Add network context
	entry.Enrichments["network_zone"] = e.determineNetworkZone(entry)
	entry.Enrichments["connectivity"] = e.analyzeConnectivity(entry)
	entry.Enrichments["exposure_level"] = e.calculateExposure(entry)
	
	return nil
}

func (e *NetworkContextEnricher) Type() string {
	return "network_context"
}

func (e *NetworkContextEnricher) determineNetworkZone(entry *CacheEntry) string {
	// Determine if resource is in public, private, or isolated zone
	if e.isPublicSubnet(entry) {
		return "public"
	}
	
	if e.hasInternetGateway(entry) {
		return "public"
	}
	
	if e.hasNATGateway(entry) {
		return "private"
	}
	
	return "isolated"
}

func (e *NetworkContextEnricher) analyzeConnectivity(entry *CacheEntry) map[string]interface{} {
	return map[string]interface{}{
		"ingress_rules":  e.getIngressRules(entry),
		"egress_rules":   e.getEgressRules(entry),
		"peering_connections": e.getPeeringConnections(entry),
	}
}

func (e *NetworkContextEnricher) calculateExposure(entry *CacheEntry) string {
	if e.isInternetFacing(entry) {
		return "internet"
	}
	
	if e.isInternalOnly(entry) {
		return "internal"
	}
	
	return "restricted"
}

func (e *NetworkContextEnricher) isPublicSubnet(entry *CacheEntry) bool {
	if data, ok := entry.Value.(map[string]interface{}); ok {
		return extractBoolField(data, "map_public_ip_on_launch", "public_subnet", "is_public")
	}
	return false
}

func (e *NetworkContextEnricher) hasInternetGateway(entry *CacheEntry) bool {
	if data, ok := entry.Value.(map[string]interface{}); ok {
		igw := extractField(data, "internet_gateway_id", "igw_id")
		return igw != ""
	}
	return false
}

func (e *NetworkContextEnricher) hasNATGateway(entry *CacheEntry) bool {
	if data, ok := entry.Value.(map[string]interface{}); ok {
		nat := extractField(data, "nat_gateway_id", "nat_id")
		return nat != ""
	}
	return false
}

func (e *NetworkContextEnricher) isInternetFacing(entry *CacheEntry) bool {
	if entry.Tags["scheme"] == "internet-facing" {
		return true
	}
	
	return e.isPublicSubnet(entry) || e.hasInternetGateway(entry)
}

func (e *NetworkContextEnricher) isInternalOnly(entry *CacheEntry) bool {
	return entry.Tags["scheme"] == "internal" || entry.Tags["access"] == "private"
}

func (e *NetworkContextEnricher) getIngressRules(entry *CacheEntry) []map[string]interface{} {
	// Extract ingress rules from security groups
	return []map[string]interface{}{}
}

func (e *NetworkContextEnricher) getEgressRules(entry *CacheEntry) []map[string]interface{} {
	// Extract egress rules from security groups
	return []map[string]interface{}{}
}

func (e *NetworkContextEnricher) getPeeringConnections(entry *CacheEntry) []string {
	// Extract VPC peering connections
	return []string{}
}

// SecurityEnricher enriches entries with security context
type SecurityEnricher struct{}

func (e *SecurityEnricher) Enrich(entry *CacheEntry) error {
	if entry.Enrichments == nil {
		entry.Enrichments = make(map[string]interface{})
	}
	
	// Add security assessment
	entry.Enrichments["security_score"] = e.calculateSecurityScore(entry)
	entry.Enrichments["vulnerabilities"] = e.detectVulnerabilities(entry)
	entry.Enrichments["security_recommendations"] = e.generateSecurityRecommendations(entry)
	
	return nil
}

func (e *SecurityEnricher) Type() string {
	return "security"
}

func (e *SecurityEnricher) calculateSecurityScore(entry *CacheEntry) float64 {
	score := 100.0
	
	// Check encryption
	if !e.hasEncryption(entry) {
		score -= 20
	}
	
	// Check access controls
	if e.hasBroadAccess(entry) {
		score -= 30
	}
	
	// Check for security patches
	if e.needsPatching(entry) {
		score -= 25
	}
	
	// Check for MFA
	if !e.hasMFA(entry) && e.requiresMFA(entry) {
		score -= 15
	}
	
	if score < 0 {
		score = 0
	}
	
	return score
}

func (e *SecurityEnricher) detectVulnerabilities(entry *CacheEntry) []map[string]interface{} {
	vulnerabilities := make([]map[string]interface{}, 0)
	
	// Check for known vulnerabilities
	if e.hasOpenPorts(entry) {
		vulnerabilities = append(vulnerabilities, map[string]interface{}{
			"type":     "open_ports",
			"severity": "high",
			"details":  "Unnecessary ports are open",
		})
	}
	
	if e.hasWeakCiphers(entry) {
		vulnerabilities = append(vulnerabilities, map[string]interface{}{
			"type":     "weak_ciphers",
			"severity": "medium",
			"details":  "Weak encryption ciphers detected",
		})
	}
	
	return vulnerabilities
}

func (e *SecurityEnricher) generateSecurityRecommendations(entry *CacheEntry) []string {
	recommendations := make([]string, 0)
	
	if !e.hasEncryption(entry) {
		recommendations = append(recommendations, "Enable encryption at rest")
	}
	
	if e.hasBroadAccess(entry) {
		recommendations = append(recommendations, "Restrict access with more specific security rules")
	}
	
	if e.needsPatching(entry) {
		recommendations = append(recommendations, "Apply latest security patches")
	}
	
	return recommendations
}

func (e *SecurityEnricher) hasEncryption(entry *CacheEntry) bool {
	if data, ok := entry.Value.(map[string]interface{}); ok {
		return extractBoolField(data, "encrypted", "encryption_enabled")
	}
	return false
}

func (e *SecurityEnricher) hasBroadAccess(entry *CacheEntry) bool {
	// Check for 0.0.0.0/0 in security rules
	if data, ok := entry.Value.(map[string]interface{}); ok {
		cidr := extractField(data, "cidr_blocks", "source_cidr")
		return cidr == "0.0.0.0/0"
	}
	return false
}

func (e *SecurityEnricher) needsPatching(entry *CacheEntry) bool {
	// Check last patch date
	if lastPatched, exists := entry.Metadata.Custom["last_patched"]; exists {
		if t, err := time.Parse(time.RFC3339, lastPatched); err == nil {
			return time.Since(t) > 30*24*time.Hour
		}
	}
	return true
}

func (e *SecurityEnricher) hasMFA(entry *CacheEntry) bool {
	return entry.Tags["mfa_enabled"] == "true"
}

func (e *SecurityEnricher) requiresMFA(entry *CacheEntry) bool {
	// Check if resource type typically requires MFA
	if entry.Metadata != nil {
		resourceType := strings.ToLower(entry.Metadata.ResourceType)
		return strings.Contains(resourceType, "iam") || 
		       strings.Contains(resourceType, "user") ||
		       strings.Contains(resourceType, "role")
	}
	return false
}

func (e *SecurityEnricher) hasOpenPorts(entry *CacheEntry) bool {
	// Check for unnecessary open ports
	if data, ok := entry.Value.(map[string]interface{}); ok {
		if ports, exists := data["open_ports"]; exists {
			if portList, ok := ports.([]interface{}); ok {
				return len(portList) > 5
			}
		}
	}
	return false
}

func (e *SecurityEnricher) hasWeakCiphers(entry *CacheEntry) bool {
	// Check for weak SSL/TLS ciphers
	if data, ok := entry.Value.(map[string]interface{}); ok {
		cipher := extractField(data, "ssl_cipher", "tls_version")
		weakCiphers := []string{"SSLv2", "SSLv3", "TLSv1.0", "RC4", "DES"}
		
		for _, weak := range weakCiphers {
			if strings.Contains(cipher, weak) {
				return true
			}
		}
	}
	return false
}

// DriftAnalysisEnricher enriches entries with drift analysis
type DriftAnalysisEnricher struct{}

func (e *DriftAnalysisEnricher) Enrich(entry *CacheEntry) error {
	if entry.Enrichments == nil {
		entry.Enrichments = make(map[string]interface{})
	}
	
	// Add drift analysis
	entry.Enrichments["drift_likelihood"] = e.calculateDriftLikelihood(entry)
	entry.Enrichments["drift_impact"] = e.assessDriftImpact(entry)
	entry.Enrichments["drift_prevention"] = e.suggestPreventiveMeasures(entry)
	
	return nil
}

func (e *DriftAnalysisEnricher) Type() string {
	return "drift_analysis"
}

func (e *DriftAnalysisEnricher) calculateDriftLikelihood(entry *CacheEntry) string {
	score := 0
	
	// Check if manually managed
	if entry.Tags["managed_by"] == "manual" {
		score += 3
	}
	
	// Check modification frequency
	if lastMod, exists := entry.Metadata.Custom["modified_at"]; exists {
		if t, err := time.Parse(time.RFC3339, lastMod); err == nil {
			if time.Since(t) < 24*time.Hour {
				score += 2
			}
		}
	}
	
	// Check for multiple owners
	if entry.Tags["shared"] == "true" {
		score += 2
	}
	
	if score >= 5 {
		return "high"
	} else if score >= 3 {
		return "medium"
	}
	
	return "low"
}

func (e *DriftAnalysisEnricher) assessDriftImpact(entry *CacheEntry) string {
	// Assess potential impact of drift
	if entry.Metadata.Custom["criticality"] == "critical" {
		return "severe"
	}
	
	if detectEnvironment(entry) == "production" {
		return "high"
	}
	
	return "moderate"
}

func (e *DriftAnalysisEnricher) suggestPreventiveMeasures(entry *CacheEntry) []string {
	measures := make([]string, 0)
	
	if entry.Tags["managed_by"] == "manual" {
		measures = append(measures, "Implement Infrastructure as Code")
	}
	
	if entry.Tags["change_tracking"] != "enabled" {
		measures = append(measures, "Enable change tracking and alerting")
	}
	
	if entry.Tags["locked"] != "true" {
		measures = append(measures, "Apply resource locks to prevent modifications")
	}
	
	return measures
}