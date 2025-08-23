package drift

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// Detector provides unified drift detection capabilities
type Detector struct {
	analyzer  *Analyzer
	predictor *DriftPredictor
	policy    *PolicyEngine
	mu        sync.RWMutex
}

// DetectionOptions configures drift detection
type DetectionOptions struct {
	CompareWith    string                 `json:"compare_with"` // "terraform", "baseline", "snapshot"
	StateFile      string                 `json:"state_file,omitempty"`
	BaselineID     string                 `json:"baseline_id,omitempty"`
	IgnorePatterns []string               `json:"ignore_patterns,omitempty"`
	SmartDefaults  bool                   `json:"smart_defaults"`
	Environment    string                 `json:"environment"`
	Thresholds     map[string]float64     `json:"thresholds"`
	DeepComparison bool                   `json:"deep_comparison"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// DetectionResult contains drift detection results
type DetectionResult struct {
	DriftItems      []models.DriftItem     `json:"drift_items"`
	Summary         *Summary               `json:"summary"`
	Analysis        *Analysis              `json:"analysis"`
	Predictions     *Predictions           `json:"predictions,omitempty"`
	Recommendations []Recommendation       `json:"recommendations"`
	Timestamp       time.Time              `json:"timestamp"`
	Duration        time.Duration          `json:"duration"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// Summary provides drift statistics
type Summary struct {
	TotalResources   int            `json:"total_resources"`
	DriftedResources int            `json:"drifted_resources"`
	DriftPercentage  float64        `json:"drift_percentage"`
	BySeverity       map[string]int `json:"by_severity"`
	ByProvider       map[string]int `json:"by_provider"`
	ByResourceType   map[string]int `json:"by_resource_type"`
	ByDriftType      map[string]int `json:"by_drift_type"`
}

// Analysis provides drift analysis
type Analysis struct {
	Patterns         []Pattern              `json:"patterns"`
	Trends           []Trend                `json:"trends"`
	ImpactScore      float64                `json:"impact_score"`
	RiskLevel        string                 `json:"risk_level"`
	CostImpact       float64                `json:"cost_impact"`
	SecurityImpact   string                 `json:"security_impact"`
	ComplianceImpact string                 `json:"compliance_impact"`
	Details          map[string]interface{} `json:"details"`
}

// Pattern represents a drift pattern
type Pattern struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Frequency   int       `json:"frequency"`
	Resources   []string  `json:"resources"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
}

// Trend represents a drift trend
type Trend struct {
	Metric    string  `json:"metric"`
	Direction string  `json:"direction"` // "increasing", "decreasing", "stable"
	Rate      float64 `json:"rate"`
	Period    string  `json:"period"`
}

// Predictions contains drift predictions
type Predictions struct {
	FutureDrift       []FutureDrift `json:"future_drift"`
	Likelihood        float64       `json:"likelihood"`
	TimeFrame         string        `json:"time_frame"`
	PreventiveActions []string      `json:"preventive_actions"`
}

// FutureDrift represents predicted future drift
type FutureDrift struct {
	ResourceType string  `json:"resource_type"`
	Probability  float64 `json:"probability"`
	TimeFrame    string  `json:"time_frame"`
	Reason       string  `json:"reason"`
}

// Recommendation represents a drift remediation recommendation
type Recommendation struct {
	Type        string   `json:"type"`
	Priority    string   `json:"priority"`
	Description string   `json:"description"`
	Actions     []string `json:"actions"`
	Impact      string   `json:"impact"`
}

// NewDetector creates a new drift detector
func NewDetector() *Detector {
	return &Detector{
		analyzer:  NewAnalyzer(),
		predictor: NewDriftPredictor(),
		policy:    NewPolicyEngine(),
	}
}

// Detect performs drift detection
func (d *Detector) Detect(ctx context.Context, currentResources []models.Resource, options DetectionOptions) (*DetectionResult, error) {
	startTime := time.Now()

	// Get baseline resources based on comparison type
	baselineResources, err := d.getBaselineResources(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get baseline resources: %w", err)
	}

	// Perform drift detection
	driftItems := d.detectDrift(currentResources, baselineResources, options)

	// Apply smart filters if enabled
	if options.SmartDefaults {
		driftItems = d.applySmartFilters(driftItems, options.Environment)
	}

	// Apply ignore patterns
	if len(options.IgnorePatterns) > 0 {
		driftItems = d.applyIgnorePatterns(driftItems, options.IgnorePatterns)
	}

	// Apply policy rules
	if d.policy != nil {
		driftItems = d.policy.EvaluateDrifts(driftItems, options.Environment)
	}

	// Generate summary
	summary := d.generateSummary(driftItems, currentResources)

	// Perform analysis
	analysis := d.analyzer.Analyze(driftItems, currentResources)

	// Generate predictions if predictor is available
	var predictions *Predictions
	if d.predictor != nil {
		predictions = d.predictor.Predict(driftItems, analysis)
	}

	// Generate recommendations
	recommendations := d.generateRecommendations(driftItems, analysis)

	result := &DetectionResult{
		DriftItems:      driftItems,
		Summary:         summary,
		Analysis:        analysis,
		Predictions:     predictions,
		Recommendations: recommendations,
		Timestamp:       time.Now(),
		Duration:        time.Since(startTime),
		Metadata:        options.Metadata,
	}

	return result, nil
}

// detectDrift compares current and baseline resources to detect drift
func (d *Detector) detectDrift(current, baseline []models.Resource, options DetectionOptions) []models.DriftItem {
	drifts := make([]models.DriftItem, 0)

	// Create maps for efficient lookup
	currentMap := make(map[string]models.Resource)
	for _, r := range current {
		currentMap[r.ID] = r
	}

	baselineMap := make(map[string]models.Resource)
	for _, r := range baseline {
		baselineMap[r.ID] = r
	}

	// Check for modified and deleted resources
	for id, baselineResource := range baselineMap {
		if currentResource, exists := currentMap[id]; exists {
			// Resource exists in both - check for modifications
			if drift := d.compareResources(baselineResource, currentResource, options); drift != nil {
				drifts = append(drifts, *drift)
			}
		} else {
			// Resource deleted
			drifts = append(drifts, models.DriftItem{
				ResourceID:   id,
				ResourceName: baselineResource.Name,
				ResourceType: baselineResource.Type,
				Provider:     baselineResource.Provider,
				Region:       baselineResource.Region,
				DriftType:    "deleted",
				Severity:     "high",
				Description:  fmt.Sprintf("Resource %s has been deleted", baselineResource.Name),
				Details: map[string]interface{}{
					"baseline_state": baselineResource.State,
				},
			})
		}
	}

	// Check for added resources
	for id, currentResource := range currentMap {
		if _, exists := baselineMap[id]; !exists {
			drifts = append(drifts, models.DriftItem{
				ResourceID:   id,
				ResourceName: currentResource.Name,
				ResourceType: currentResource.Type,
				Provider:     currentResource.Provider,
				Region:       currentResource.Region,
				DriftType:    "added",
				Severity:     "medium",
				Description:  fmt.Sprintf("New resource %s has been added", currentResource.Name),
				Details: map[string]interface{}{
					"current_state": currentResource.State,
				},
			})
		}
	}

	return drifts
}

// compareResources compares two resources for drift
func (d *Detector) compareResources(baseline, current models.Resource, options DetectionOptions) *models.DriftItem {
	if !options.DeepComparison {
		// Simple comparison
		if baseline.State != current.State {
			return &models.DriftItem{
				ResourceID:   current.ID,
				ResourceName: current.Name,
				ResourceType: current.Type,
				Provider:     current.Provider,
				Region:       current.Region,
				DriftType:    "modified",
				Severity:     d.calculateSeverity(baseline, current),
				Description:  fmt.Sprintf("Resource state changed from %s to %s", baseline.State, current.State),
				Details: map[string]interface{}{
					"baseline_state": baseline.State,
					"current_state":  current.State,
				},
			}
		}
	} else {
		// Deep comparison including properties and tags
		changes := make(map[string]interface{})

		// Compare properties
		for key, baselineValue := range baseline.Properties {
			if currentValue, exists := current.Properties[key]; !exists {
				changes[fmt.Sprintf("property.%s", key)] = map[string]interface{}{
					"baseline": baselineValue,
					"current":  nil,
				}
			} else if baselineValue != currentValue {
				changes[fmt.Sprintf("property.%s", key)] = map[string]interface{}{
					"baseline": baselineValue,
					"current":  currentValue,
				}
			}
		}

		// Check for new properties
		for key, currentValue := range current.Properties {
			if _, exists := baseline.Properties[key]; !exists {
				changes[fmt.Sprintf("property.%s", key)] = map[string]interface{}{
					"baseline": nil,
					"current":  currentValue,
				}
			}
		}

		// Compare tags
		baselineTags := baseline.GetTagsAsMap()
		currentTags := current.GetTagsAsMap()

		for key, baselineValue := range baselineTags {
			if currentValue, exists := currentTags[key]; !exists {
				changes[fmt.Sprintf("tag.%s", key)] = map[string]interface{}{
					"baseline": baselineValue,
					"current":  nil,
				}
			} else if baselineValue != currentValue {
				changes[fmt.Sprintf("tag.%s", key)] = map[string]interface{}{
					"baseline": baselineValue,
					"current":  currentValue,
				}
			}
		}

		// Check for new tags
		for key, currentValue := range currentTags {
			if _, exists := baselineTags[key]; !exists {
				changes[fmt.Sprintf("tag.%s", key)] = map[string]interface{}{
					"baseline": nil,
					"current":  currentValue,
				}
			}
		}

		if len(changes) > 0 {
			return &models.DriftItem{
				ResourceID:   current.ID,
				ResourceName: current.Name,
				ResourceType: current.Type,
				Provider:     current.Provider,
				Region:       current.Region,
				DriftType:    "modified",
				Severity:     d.calculateSeverity(baseline, current),
				Description:  fmt.Sprintf("Resource has %d configuration changes", len(changes)),
				Details:      changes,
			}
		}
	}

	return nil
}

// calculateSeverity calculates drift severity
func (d *Detector) calculateSeverity(baseline, current models.Resource) string {
	// Simplified severity calculation - can be enhanced
	if baseline.State != current.State {
		if current.State == "terminated" || current.State == "deleted" {
			return "critical"
		}
		return "high"
	}

	// Check for security-related changes
	if d.hasSecurityChanges(baseline, current) {
		return "high"
	}

	// Check for cost-related changes
	if d.hasCostChanges(baseline, current) {
		return "medium"
	}

	return "low"
}

// hasSecurityChanges checks for security-related changes
func (d *Detector) hasSecurityChanges(baseline, current models.Resource) bool {
	// Check for security group changes, encryption changes, etc.
	securityKeys := []string{"security_group", "encryption", "public_access", "ssl", "tls"}

	for _, key := range securityKeys {
		if baseline.Properties[key] != current.Properties[key] {
			return true
		}
	}

	return false
}

// hasCostChanges checks for cost-related changes
func (d *Detector) hasCostChanges(baseline, current models.Resource) bool {
	// Check for instance size changes, storage changes, etc.
	costKeys := []string{"instance_type", "size", "storage", "capacity"}

	for _, key := range costKeys {
		if baseline.Properties[key] != current.Properties[key] {
			return true
		}
	}

	return false
}

// applySmartFilters applies environment-aware filtering
func (d *Detector) applySmartFilters(drifts []models.DriftItem, environment string) []models.DriftItem {
	thresholds := d.getSmartThresholds(environment)
	filtered := make([]models.DriftItem, 0)

	for _, drift := range drifts {
		threshold := thresholds[drift.Severity]
		// Keep drift if it's above the threshold for its severity
		if threshold < 1.0 {
			filtered = append(filtered, drift)
		}
	}

	return filtered
}

// getSmartThresholds returns environment-specific thresholds
func (d *Detector) getSmartThresholds(environment string) map[string]float64 {
	switch environment {
	case "production":
		return map[string]float64{
			"critical": 0.0,  // No tolerance
			"high":     0.05, // 5% tolerance
			"medium":   0.15, // 15% tolerance
			"low":      0.75, // 75% filter (noise reduction)
		}
	case "staging":
		return map[string]float64{
			"critical": 0.10,
			"high":     0.25,
			"medium":   0.50,
			"low":      0.85,
		}
	default: // development
		return map[string]float64{
			"critical": 0.25,
			"high":     0.50,
			"medium":   0.75,
			"low":      0.90,
		}
	}
}

// applyIgnorePatterns applies ignore patterns to filter drifts
func (d *Detector) applyIgnorePatterns(drifts []models.DriftItem, patterns []string) []models.DriftItem {
	filtered := make([]models.DriftItem, 0)

	for _, drift := range drifts {
		ignore := false
		for _, pattern := range patterns {
			if d.matchesPattern(drift, pattern) {
				ignore = true
				break
			}
		}
		if !ignore {
			filtered = append(filtered, drift)
		}
	}

	return filtered
}

// matchesPattern checks if a drift matches an ignore pattern
func (d *Detector) matchesPattern(drift models.DriftItem, pattern string) bool {
	// Simple pattern matching - can be enhanced with regex
	return drift.ResourceType == pattern || drift.DriftType == pattern
}

// generateSummary generates drift summary statistics
func (d *Detector) generateSummary(drifts []models.DriftItem, resources []models.Resource) *Summary {
	summary := &Summary{
		TotalResources:   len(resources),
		DriftedResources: len(drifts),
		BySeverity:       make(map[string]int),
		ByProvider:       make(map[string]int),
		ByResourceType:   make(map[string]int),
		ByDriftType:      make(map[string]int),
	}

	if summary.TotalResources > 0 {
		summary.DriftPercentage = float64(summary.DriftedResources) / float64(summary.TotalResources) * 100
	}

	for _, drift := range drifts {
		summary.BySeverity[drift.Severity]++
		summary.ByProvider[drift.Provider]++
		summary.ByResourceType[drift.ResourceType]++
		summary.ByDriftType[drift.DriftType]++
	}

	return summary
}

// generateRecommendations generates drift remediation recommendations
func (d *Detector) generateRecommendations(drifts []models.DriftItem, analysis *Analysis) []Recommendation {
	recommendations := make([]Recommendation, 0)

	// High-level recommendations based on analysis
	if analysis.RiskLevel == "high" || analysis.RiskLevel == "critical" {
		recommendations = append(recommendations, Recommendation{
			Type:        "immediate_action",
			Priority:    "critical",
			Description: "Critical drift detected requiring immediate attention",
			Actions: []string{
				"Review and approve remediation plan",
				"Execute remediation in maintenance window",
				"Verify resource state post-remediation",
			},
			Impact: "high",
		})
	}

	// Pattern-based recommendations
	for _, pattern := range analysis.Patterns {
		if pattern.Frequency > 5 {
			recommendations = append(recommendations, Recommendation{
				Type:        "pattern_remediation",
				Priority:    "high",
				Description: fmt.Sprintf("Recurring drift pattern detected: %s", pattern.Description),
				Actions: []string{
					"Investigate root cause of pattern",
					"Update IaC templates to prevent recurrence",
					"Implement automated remediation",
				},
				Impact: "medium",
			})
		}
	}

	// Environment-specific recommendations
	if len(drifts) > 10 {
		recommendations = append(recommendations, Recommendation{
			Type:        "automation",
			Priority:    "medium",
			Description: "High drift volume suggests need for automation",
			Actions: []string{
				"Enable auto-remediation for low-risk drifts",
				"Implement drift detection automation",
				"Set up continuous compliance monitoring",
			},
			Impact: "low",
		})
	}

	return recommendations
}

// getBaselineResources retrieves baseline resources based on comparison type
func (d *Detector) getBaselineResources(ctx context.Context, options DetectionOptions) ([]models.Resource, error) {
	switch options.CompareWith {
	case "terraform":
		if options.StateFile == "" {
			return nil, fmt.Errorf("state file required for terraform comparison")
		}
		return d.loadTerraformState(options.StateFile)
	case "baseline":
		if options.BaselineID == "" {
			return nil, fmt.Errorf("baseline ID required for baseline comparison")
		}
		return d.loadBaseline(options.BaselineID)
	case "snapshot":
		return d.loadLatestSnapshot()
	default:
		return nil, fmt.Errorf("invalid comparison type: %s", options.CompareWith)
	}
}

// Placeholder methods for loading baseline resources
func (d *Detector) loadTerraformState(stateFile string) ([]models.Resource, error) {
	// Implementation would load and parse Terraform state
	return []models.Resource{}, nil
}

func (d *Detector) loadBaseline(baselineID string) ([]models.Resource, error) {
	// Implementation would load baseline from storage
	return []models.Resource{}, nil
}

func (d *Detector) loadLatestSnapshot() ([]models.Resource, error) {
	// Implementation would load latest snapshot
	return []models.Resource{}, nil
}
