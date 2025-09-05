package detector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/drift/comparator"
	"github.com/catherinevee/driftmgr/pkg/models"
)

// DetectionMode defines how thorough the drift detection should be
type DetectionMode string

const (
	// QuickMode only checks if resources exist (fastest)
	QuickMode DetectionMode = "quick"
	// DeepMode performs full attribute-level comparison (slowest)
	DeepMode DetectionMode = "deep"
	// SmartMode adapts based on resource criticality (balanced)
	SmartMode DetectionMode = "smart"
)

// ResourceCriticality defines how critical a resource type is
type ResourceCriticality string

const (
	CriticalPriority ResourceCriticality = "critical"
	HighPriority     ResourceCriticality = "high"
	MediumPriority   ResourceCriticality = "medium"
	LowPriority      ResourceCriticality = "low"
)

// CriticalityConfig maps resource types to their criticality levels
type CriticalityConfig struct {
	Rules map[string]ResourceCriticality
}

// DefaultCriticalityConfig returns the default criticality configuration
func DefaultCriticalityConfig() *CriticalityConfig {
	return &CriticalityConfig{
		Rules: map[string]ResourceCriticality{
			// Critical infrastructure
			"aws_db_instance":                 CriticalPriority,
			"aws_rds_cluster":                 CriticalPriority,
			"aws_kms_key":                     CriticalPriority,
			"aws_iam_role":                    CriticalPriority,
			"aws_iam_policy":                  CriticalPriority,
			"aws_security_group":              CriticalPriority,
			"aws_vpc":                         CriticalPriority,
			"azure_key_vault":                 CriticalPriority,
			"azure_sql_database":              CriticalPriority,
			"google_sql_database_instance":    CriticalPriority,
			"google_kms_crypto_key":           CriticalPriority,
			
			// High priority
			"aws_instance":                    HighPriority,
			"aws_ecs_service":                 HighPriority,
			"aws_eks_cluster":                 HighPriority,
			"aws_lambda_function":             HighPriority,
			"aws_alb":                         HighPriority,
			"aws_elb":                         HighPriority,
			"azure_virtual_machine":           HighPriority,
			"azure_kubernetes_cluster":        HighPriority,
			"google_compute_instance":         HighPriority,
			"google_container_cluster":        HighPriority,
			
			// Medium priority
			"aws_s3_bucket":                   MediumPriority,
			"aws_sqs_queue":                   MediumPriority,
			"aws_sns_topic":                   MediumPriority,
			"aws_dynamodb_table":              MediumPriority,
			"azure_storage_account":           MediumPriority,
			"google_storage_bucket":           MediumPriority,
			
			// Low priority (tags, metadata, etc)
			"aws_ec2_tag":                     LowPriority,
			"aws_ssm_parameter":               LowPriority,
			"aws_cloudwatch_log_group":        LowPriority,
		},
	}
}

// GetCriticality returns the criticality level for a resource type
func (c *CriticalityConfig) GetCriticality(resourceType string) ResourceCriticality {
	if criticality, exists := c.Rules[resourceType]; exists {
		return criticality
	}
	
	// Check for partial matches (e.g., any IAM resource is critical)
	lowerType := strings.ToLower(resourceType)
	if strings.Contains(lowerType, "iam") || strings.Contains(lowerType, "kms") {
		return CriticalPriority
	}
	if strings.Contains(lowerType, "security") || strings.Contains(lowerType, "firewall") {
		return CriticalPriority
	}
	if strings.Contains(lowerType, "database") || strings.Contains(lowerType, "sql") {
		return HighPriority
	}
	
	return MediumPriority // Default to medium if unknown
}

// ModeDetector extends the base Detector with mode-aware detection
type ModeDetector struct {
	*Detector
	mode              DetectionMode
	criticalityConfig *CriticalityConfig
	stats             *DetectionStats
}

// DetectionStats tracks statistics about the detection process
type DetectionStats struct {
	StartTime         time.Time
	EndTime           time.Time
	ResourcesScanned  int
	ResourcesWithDrift int
	QuickScanned      int
	DeepScanned       int
	Skipped           int
	Errors            []error
}

// NewModeDetector creates a new mode-aware drift detector
func NewModeDetector(mode DetectionMode, config *CriticalityConfig) *ModeDetector {
	if config == nil {
		config = DefaultCriticalityConfig()
	}
	
	return &ModeDetector{
		Detector:          &Detector{},
		mode:              mode,
		criticalityConfig: config,
		stats: &DetectionStats{
			Errors: make([]error, 0),
		},
	}
}

// DetectDriftWithMode performs drift detection based on the configured mode
func (md *ModeDetector) DetectDriftWithMode(ctx context.Context, desired, actual *models.State) (*DriftResult, error) {
	md.stats.StartTime = time.Now()
	defer func() {
		md.stats.EndTime = time.Now()
	}()

	switch md.mode {
	case QuickMode:
		return md.quickDetection(ctx, desired, actual)
	case DeepMode:
		return md.deepDetection(ctx, desired, actual)
	case SmartMode:
		return md.smartDetection(ctx, desired, actual)
	default:
		return nil, fmt.Errorf("unknown detection mode: %s", md.mode)
	}
}

// quickDetection performs fast resource existence checks only
func (md *ModeDetector) quickDetection(ctx context.Context, desired, actual *models.State) (*DriftResult, error) {
	result := &DriftResult{
		HasDrift:    false,
		Differences: []comparator.Difference{},
		Timestamp:   time.Now(),
	}

	// Build resource maps for O(1) lookup
	actualResources := make(map[string]*models.Resource)
	for _, r := range actual.Resources {
		key := fmt.Sprintf("%s.%s", r.Type, r.Name)
		actualResources[key] = &r
	}

	// Check for missing resources (exist in desired but not in actual)
	for _, desiredResource := range desired.Resources {
		md.stats.ResourcesScanned++
		key := fmt.Sprintf("%s.%s", desiredResource.Type, desiredResource.Name)
		
		if _, exists := actualResources[key]; !exists {
			md.stats.ResourcesWithDrift++
			result.HasDrift = true
			result.Differences = append(result.Differences, comparator.Difference{
				Path:       key,
				Type:       comparator.DiffTypeRemoved,
				Expected:   desiredResource,
				Actual:     nil,
				Message:    fmt.Sprintf("Resource %s is missing from actual state", key),
				Importance: comparator.ImportanceHigh,
			})
		}
		md.stats.QuickScanned++
	}

	// Check for unmanaged resources (exist in actual but not in desired)
	desiredResourceMap := make(map[string]bool)
	for _, r := range desired.Resources {
		key := fmt.Sprintf("%s.%s", r.Type, r.Name)
		desiredResourceMap[key] = true
	}

	for _, actualResource := range actual.Resources {
		key := fmt.Sprintf("%s.%s", actualResource.Type, actualResource.Name)
		if !desiredResourceMap[key] {
			md.stats.ResourcesWithDrift++
			result.HasDrift = true
			result.Differences = append(result.Differences, comparator.Difference{
				Path:       key,
				Type:       comparator.DiffTypeAdded,
				Expected:   nil,
				Actual:     actualResource,
				Message:    fmt.Sprintf("Unmanaged resource %s found in actual state", key),
				Importance: comparator.ImportanceMedium,
			})
		}
	}

	result.Summary = md.generateSummary()
	return result, nil
}

// deepDetection performs comprehensive attribute-level comparison
func (md *ModeDetector) deepDetection(ctx context.Context, desired, actual *models.State) (*DriftResult, error) {
	// Use the existing detector's full comparison logic
	baseResult, err := md.Detector.DetectDrift(ctx, desired, actual)
	if err != nil {
		return nil, err
	}

	// Track statistics
	for _, resource := range desired.Resources {
		md.stats.ResourcesScanned++
		md.stats.DeepScanned++
	}

	if baseResult.HasDrift {
		md.stats.ResourcesWithDrift = len(baseResult.Differences)
	}

	baseResult.Summary = md.generateSummary()
	return baseResult, nil
}

// smartDetection adapts detection depth based on resource criticality
func (md *ModeDetector) smartDetection(ctx context.Context, desired, actual *models.State) (*DriftResult, error) {
	result := &DriftResult{
		HasDrift:    false,
		Differences: []comparator.Difference{},
		Timestamp:   time.Now(),
	}

	// Build actual resource map
	actualResources := make(map[string]*models.Resource)
	for _, r := range actual.Resources {
		key := fmt.Sprintf("%s.%s", r.Type, r.Name)
		actualResources[key] = &r
	}

	// Process each resource based on its criticality
	for _, desiredResource := range desired.Resources {
		md.stats.ResourcesScanned++
		key := fmt.Sprintf("%s.%s", desiredResource.Type, desiredResource.Name)
		criticality := md.criticalityConfig.GetCriticality(desiredResource.Type)
		
		actualResource, exists := actualResources[key]
		
		// Always report missing resources
		if !exists {
			md.stats.ResourcesWithDrift++
			result.HasDrift = true
			result.Differences = append(result.Differences, comparator.Difference{
				Path:       key,
				Type:       comparator.DiffTypeRemoved,
				Expected:   desiredResource,
				Actual:     nil,
				Message:    fmt.Sprintf("Critical resource %s is missing", key),
				Importance: comparator.ImportanceCritical,
			})
			continue
		}

		// Determine detection depth based on criticality
		switch criticality {
		case CriticalPriority:
			// Deep scan for critical resources
			md.stats.DeepScanned++
			diffs := md.performDeepComparison(desiredResource, *actualResource)
			if len(diffs) > 0 {
				md.stats.ResourcesWithDrift++
				result.HasDrift = true
				result.Differences = append(result.Differences, diffs...)
			}
			
		case HighPriority:
			// Medium-depth scan for high priority resources
			md.stats.DeepScanned++
			diffs := md.performMediumComparison(desiredResource, *actualResource)
			if len(diffs) > 0 {
				md.stats.ResourcesWithDrift++
				result.HasDrift = true
				result.Differences = append(result.Differences, diffs...)
			}
			
		case MediumPriority, LowPriority:
			// Quick scan for lower priority resources
			md.stats.QuickScanned++
			// Just verify the resource exists (already done above)
		}
	}

	result.Summary = md.generateSummary()
	return result, nil
}

// performDeepComparison does a full attribute comparison
func (md *ModeDetector) performDeepComparison(desired, actual models.Resource) []comparator.Difference {
	var differences []comparator.Difference
	
	// Compare all properties
	for key, desiredValue := range desired.Properties {
		actualValue, exists := actual.Properties[key]
		if !exists {
			differences = append(differences, comparator.Difference{
				Path:       fmt.Sprintf("%s.%s.%s", desired.Type, desired.Name, key),
				Type:       comparator.DiffTypeRemoved,
				Expected:   desiredValue,
				Actual:     nil,
				Message:    fmt.Sprintf("Property %s is missing", key),
				Importance: comparator.ImportanceHigh,
			})
		} else if !md.valuesEqual(desiredValue, actualValue) {
			differences = append(differences, comparator.Difference{
				Path:       fmt.Sprintf("%s.%s.%s", desired.Type, desired.Name, key),
				Type:       comparator.DiffTypeModified,
				Expected:   desiredValue,
				Actual:     actualValue,
				Message:    fmt.Sprintf("Property %s has changed", key),
				Importance: md.getPropertyImportance(key),
			})
		}
	}
	
	// Check for unexpected properties in actual
	for key, actualValue := range actual.Properties {
		if _, exists := desired.Properties[key]; !exists {
			differences = append(differences, comparator.Difference{
				Path:       fmt.Sprintf("%s.%s.%s", actual.Type, actual.Name, key),
				Type:       comparator.DiffTypeAdded,
				Expected:   nil,
				Actual:     actualValue,
				Message:    fmt.Sprintf("Unexpected property %s found", key),
				Importance: comparator.ImportanceLow,
			})
		}
	}
	
	// Compare tags
	if !md.tagsEqual(desired.Tags, actual.Tags) {
		differences = append(differences, comparator.Difference{
			Path:       fmt.Sprintf("%s.%s.tags", desired.Type, desired.Name),
			Type:       comparator.DiffTypeModified,
			Expected:   desired.Tags,
			Actual:     actual.Tags,
			Message:    "Tags have changed",
			Importance: comparator.ImportanceLow,
		})
	}
	
	return differences
}

// performMediumComparison checks important attributes only
func (md *ModeDetector) performMediumComparison(desired, actual models.Resource) []comparator.Difference {
	var differences []comparator.Difference
	
	// Define important properties to check based on resource type
	importantProps := md.getImportantProperties(desired.Type)
	
	for _, prop := range importantProps {
		desiredValue, desiredExists := desired.Properties[prop]
		actualValue, actualExists := actual.Properties[prop]
		
		if desiredExists && !actualExists {
			differences = append(differences, comparator.Difference{
				Path:       fmt.Sprintf("%s.%s.%s", desired.Type, desired.Name, prop),
				Type:       comparator.DiffTypeRemoved,
				Expected:   desiredValue,
				Actual:     nil,
				Message:    fmt.Sprintf("Important property %s is missing", prop),
				Importance: comparator.ImportanceHigh,
			})
		} else if desiredExists && actualExists && !md.valuesEqual(desiredValue, actualValue) {
			differences = append(differences, comparator.Difference{
				Path:       fmt.Sprintf("%s.%s.%s", desired.Type, desired.Name, prop),
				Type:       comparator.DiffTypeModified,
				Expected:   desiredValue,
				Actual:     actualValue,
				Message:    fmt.Sprintf("Important property %s has changed", prop),
				Importance: comparator.ImportanceHigh,
			})
		}
	}
	
	return differences
}

// getImportantProperties returns the list of important properties for a resource type
func (md *ModeDetector) getImportantProperties(resourceType string) []string {
	// Define important properties per resource type
	importantProps := map[string][]string{
		"aws_instance":         {"instance_type", "ami", "subnet_id", "security_groups"},
		"aws_security_group":   {"ingress", "egress", "vpc_id"},
		"aws_rds_instance":     {"engine", "engine_version", "instance_class", "allocated_storage"},
		"aws_s3_bucket":        {"acl", "versioning", "encryption", "lifecycle_rule"},
		"aws_lambda_function":  {"runtime", "handler", "memory_size", "timeout", "environment"},
		"aws_iam_role":         {"assume_role_policy", "managed_policy_arns"},
		"aws_iam_policy":       {"policy"},
	}
	
	if props, exists := importantProps[resourceType]; exists {
		return props
	}
	
	// Default important properties for unknown types
	return []string{"name", "id", "status", "state"}
}

// getPropertyImportance determines how important a property change is
func (md *ModeDetector) getPropertyImportance(property string) comparator.Importance {
	criticalProps := []string{"security_groups", "ingress", "egress", "policy", "assume_role_policy", "kms_key_id", "encryption"}
	highProps := []string{"instance_type", "size", "capacity", "runtime", "engine_version"}
	
	propLower := strings.ToLower(property)
	
	for _, critical := range criticalProps {
		if strings.Contains(propLower, critical) {
			return comparator.ImportanceCritical
		}
	}
	
	for _, high := range highProps {
		if strings.Contains(propLower, high) {
			return comparator.ImportanceHigh
		}
	}
	
	if strings.Contains(propLower, "tag") || strings.Contains(propLower, "description") {
		return comparator.ImportanceLow
	}
	
	return comparator.ImportanceMedium
}

// valuesEqual compares two values for equality
func (md *ModeDetector) valuesEqual(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// tagsEqual compares two tag sets
func (md *ModeDetector) tagsEqual(a, b interface{}) bool {
	tagsA, okA := a.(map[string]string)
	tagsB, okB := b.(map[string]string)
	
	if !okA || !okB {
		return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
	}
	
	if len(tagsA) != len(tagsB) {
		return false
	}
	
	for key, valA := range tagsA {
		if valB, exists := tagsB[key]; !exists || valA != valB {
			return false
		}
	}
	
	return true
}

// generateSummary creates a summary of the detection process
func (md *ModeDetector) generateSummary() map[string]interface{} {
	duration := md.stats.EndTime.Sub(md.stats.StartTime)
	
	return map[string]interface{}{
		"mode":                md.mode,
		"duration_ms":         duration.Milliseconds(),
		"resources_scanned":   md.stats.ResourcesScanned,
		"resources_with_drift": md.stats.ResourcesWithDrift,
		"quick_scanned":       md.stats.QuickScanned,
		"deep_scanned":        md.stats.DeepScanned,
		"skipped":             md.stats.Skipped,
		"errors":              len(md.stats.Errors),
	}
}

// GetStats returns the detection statistics
func (md *ModeDetector) GetStats() *DetectionStats {
	return md.stats
}