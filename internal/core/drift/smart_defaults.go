package drift

import (
	"github.com/catherinevee/driftmgr/internal/models"
)

// SmartDefaults provides intelligent drift filtering
type SmartDefaults struct {
	Environment string
	Thresholds  map[string]float64
}

// NewSmartDefaults creates new smart defaults
func NewSmartDefaults(environment string) *SmartDefaults {
	thresholds := make(map[string]float64)

	switch environment {
	case "production":
		thresholds["critical"] = 0.0
		thresholds["high"] = 0.05
		thresholds["medium"] = 0.15
		thresholds["low"] = 0.75
	case "staging":
		thresholds["critical"] = 0.0
		thresholds["high"] = 0.1
		thresholds["medium"] = 0.3
		thresholds["low"] = 0.8
	default:
		thresholds["critical"] = 0.0
		thresholds["high"] = 0.2
		thresholds["medium"] = 0.5
		thresholds["low"] = 0.9
	}

	return &SmartDefaults{
		Environment: environment,
		Thresholds:  thresholds,
	}
}

// ShouldIgnore determines if a drift item should be ignored
func (s *SmartDefaults) ShouldIgnore(item models.DriftItem) bool {
	threshold, exists := s.Thresholds[item.Severity]
	if !exists {
		return false
	}

	// Simple logic - in reality this would be more sophisticated
	return threshold > 0.5
}

// GetNotificationChannels returns notification channels for a drift item
func (s *SmartDefaults) GetNotificationChannels(item models.DriftItem, environment string) []string {
	if item.Severity == "critical" {
		return []string{"email", "slack", "pagerduty"}
	} else if item.Severity == "high" && environment == "production" {
		return []string{"email", "slack"}
	}
	return []string{}
}
