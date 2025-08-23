package analysis

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// DriftAnalysis represents the result of drift analysis
type DriftAnalysis struct {
	ResourceID    string                 `json:"resource_id"`
	ResourceType  string                 `json:"resource_type"`
	Provider      string                 `json:"provider"`
	Region        string                 `json:"region"`
	DriftDetected bool                   `json:"drift_detected"`
	DriftType     string                 `json:"drift_type,omitempty"`
	Changes       []DriftChange          `json:"changes,omitempty"`
	Severity      string                 `json:"severity"`
	Timestamp     time.Time              `json:"timestamp"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// DriftChange represents a specific change detected in drift analysis
type DriftChange struct {
	Field      string      `json:"field"`
	OldValue   interface{} `json:"old_value"`
	NewValue   interface{} `json:"new_value"`
	ChangeType string      `json:"change_type"`
}

// Analyzer provides drift analysis capabilities
type Analyzer struct {
	config map[string]interface{}
}

// NewAnalyzer creates a new drift analyzer
func NewAnalyzer(config map[string]interface{}) *Analyzer {
	return &Analyzer{
		config: config,
	}
}

// AnalyzeResource performs drift analysis on a single resource
func (a *Analyzer) AnalyzeResource(ctx context.Context, resource models.Resource, expectedState map[string]interface{}) (*DriftAnalysis, error) {
	analysis := &DriftAnalysis{
		ResourceID:   resource.ID,
		ResourceType: resource.Type,
		Provider:     resource.Provider,
		Region:       resource.Region,
		Timestamp:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	// Compare actual vs expected state
	changes := a.detectChanges(resource, expectedState)

	if len(changes) > 0 {
		analysis.DriftDetected = true
		analysis.Changes = changes
		analysis.DriftType = a.determineDriftType(changes)
		analysis.Severity = a.calculateSeverity(changes)
	}

	return analysis, nil
}

// detectChanges compares actual resource state with expected state
func (a *Analyzer) detectChanges(resource models.Resource, expectedState map[string]interface{}) []DriftChange {
	var changes []DriftChange

	// Compare properties
	for key, expectedValue := range expectedState {
		if actualValue, exists := resource.Properties[key]; exists {
			if !a.valuesEqual(actualValue, expectedValue) {
				changes = append(changes, DriftChange{
					Field:      key,
					OldValue:   expectedValue,
					NewValue:   actualValue,
					ChangeType: "property_change",
				})
			}
		} else {
			// Property missing in actual state
			changes = append(changes, DriftChange{
				Field:      key,
				OldValue:   expectedValue,
				NewValue:   nil,
				ChangeType: "property_missing",
			})
		}
	}

	// Check for extra properties in actual state
	for key, actualValue := range resource.Properties {
		if _, exists := expectedState[key]; !exists {
			changes = append(changes, DriftChange{
				Field:      key,
				OldValue:   nil,
				NewValue:   actualValue,
				ChangeType: "property_added",
			})
		}
	}

	return changes
}

// valuesEqual compares two values for equality
func (a *Analyzer) valuesEqual(val1, val2 interface{}) bool {
	// Simple equality check - in production, you'd want more sophisticated comparison
	return fmt.Sprintf("%v", val1) == fmt.Sprintf("%v", val2)
}

// determineDriftType determines the type of drift based on changes
func (a *Analyzer) determineDriftType(changes []DriftChange) string {
	if len(changes) == 0 {
		return "none"
	}

	// Simple logic - can be enhanced based on business rules
	for _, change := range changes {
		if change.ChangeType == "property_missing" {
			return "deletion"
		}
		if change.ChangeType == "property_added" {
			return "addition"
		}
	}

	return "modification"
}

// calculateSeverity calculates the severity of drift
func (a *Analyzer) calculateSeverity(changes []DriftChange) string {
	if len(changes) == 0 {
		return "none"
	}

	// Simple severity calculation - can be enhanced
	criticalFields := map[string]bool{
		"security_group": true,
		"encryption":     true,
		"access_control": true,
	}

	for _, change := range changes {
		if criticalFields[change.Field] {
			return "critical"
		}
	}

	if len(changes) > 5 {
		return "high"
	}

	return "medium"
}

// AnalyzeBatch performs drift analysis on multiple resources
func (a *Analyzer) AnalyzeBatch(ctx context.Context, resources []models.Resource, expectedStates map[string]map[string]interface{}) ([]*DriftAnalysis, error) {
	var analyses []*DriftAnalysis

	for _, resource := range resources {
		expectedState, exists := expectedStates[resource.ID]
		if !exists {
			// No expected state found - treat as potential drift
			unexpectedAnalysis := &DriftAnalysis{
				ResourceID:    resource.ID,
				ResourceType:  resource.Type,
				Provider:      resource.Provider,
				Region:        resource.Region,
				DriftDetected: true,
				DriftType:     "unexpected_resource",
				Severity:      "high",
				Timestamp:     time.Now(),
			}
			analyses = append(analyses, unexpectedAnalysis)
			continue
		}

		analysis, err := a.AnalyzeResource(ctx, resource, expectedState)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze resource %s: %w", resource.ID, err)
		}

		analyses = append(analyses, analysis)
	}

	return analyses, nil
}
