package detector

import (
	"time"
)

// DetectionMode defines how thorough the drift detection should be
type DetectionMode string

const (
	// QuickMode performs basic existence checks (< 30 seconds)
	QuickMode DetectionMode = "quick"
	
	// DeepMode performs comprehensive analysis of all attributes
	DeepMode DetectionMode = "deep"
	
	// SmartMode adapts based on resource criticality
	SmartMode DetectionMode = "smart"
)

// CriticalityLevel defines how critical a resource type is
type CriticalityLevel int

const (
	CriticalLevel CriticalityLevel = iota
	HighLevel
	MediumLevel
	LowLevel
)

// ResourceCriticality maps resource types to their criticality levels
var ResourceCriticality = map[string]CriticalityLevel{
	// Critical resources - always deep scan
	"aws_db_instance":               CriticalLevel,
	"aws_security_group":            CriticalLevel,
	"aws_iam_role":                  CriticalLevel,
	"aws_iam_policy":                CriticalLevel,
	"azure_sql_database":            CriticalLevel,
	"google_sql_database_instance":  CriticalLevel,
	
	// High priority resources
	"aws_lb":                        HighLevel,
	"aws_alb":                       HighLevel,
	"aws_network_acl":               HighLevel,
	"aws_kms_key":                   HighLevel,
	"azure_lb":                      HighLevel,
	"google_compute_firewall":       HighLevel,
	
	// Medium priority resources
	"aws_instance":                  MediumLevel,
	"aws_s3_bucket":                 MediumLevel,
	"azure_virtual_machine":         MediumLevel,
	"google_compute_instance":       MediumLevel,
	
	// Low priority resources - quick scan in smart mode
	"aws_s3_bucket_object":          LowLevel,
	"aws_route53_record":            LowLevel,
}

// DetectionStats tracks statistics about the detection process
type DetectionStats struct {
	StartTime          time.Time
	EndTime            time.Time
	Mode               DetectionMode
	ResourcesScanned   int
	ResourcesWithDrift int
	Duration           time.Duration
}

// GetResourceCriticality returns the criticality level for a resource type
func GetResourceCriticality(resourceType string) CriticalityLevel {
	if level, exists := ResourceCriticality[resourceType]; exists {
		return level
	}
	return MediumLevel // Default to medium if not specified
}

// ShouldDeepScan determines if a resource should be deep scanned based on mode and criticality
func ShouldDeepScan(mode DetectionMode, resourceType string) bool {
	switch mode {
	case QuickMode:
		return false // Quick mode never deep scans
	case DeepMode:
		return true // Deep mode always deep scans
	case SmartMode:
		// Smart mode deep scans based on criticality
		criticality := GetResourceCriticality(resourceType)
		return criticality == CriticalLevel || criticality == HighLevel
	default:
		return true
	}
}

// ApplyModeSettings applies detection mode settings to the detector configuration
func ApplyModeSettings(config *DetectorConfig, mode DetectionMode) {
	switch mode {
	case QuickMode:
		config.MaxWorkers = 20
		config.Timeout = 30 * time.Second
		config.DeepComparison = false
		config.ParallelDiscovery = true
	case DeepMode:
		config.MaxWorkers = 10
		config.Timeout = 5 * time.Minute
		config.DeepComparison = true
		config.ParallelDiscovery = true
	case SmartMode:
		config.MaxWorkers = 15
		config.Timeout = 2 * time.Minute
		config.DeepComparison = false // Will be set per resource
		config.ParallelDiscovery = true
	}
}