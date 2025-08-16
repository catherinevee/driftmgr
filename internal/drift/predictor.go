package drift

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// DriftPredictor analyzes resource patterns to predict potential drifts
type DriftPredictor struct {
	patterns   map[string]*DriftPattern
	history    []DriftEvent
	riskScorer *RiskScorer
	config     *PredictorConfig
}

// DriftPattern represents a pattern that indicates potential drift
type DriftPattern struct {
	ID          string
	Name        string
	Description string
	Conditions  []PatternCondition
	Confidence  float64
	RiskLevel   RiskLevel
	Weight      float64
}

// PatternCondition defines a condition that must be met for a pattern
type PatternCondition struct {
	Field    string
	Operator string
	Value    interface{}
}

// RiskLevel represents the severity of a predicted drift
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// DriftEvent represents a historical drift event
type DriftEvent struct {
	ID           string
	ResourceID   string
	ResourceType string
	DriftType    string
	Severity     string
	Timestamp    time.Time
	Pattern      string
	Resolved     bool
}

// PredictedDrift represents a predicted drift event
type PredictedDrift struct {
	ID              string
	ResourceID      string
	ResourceName    string
	ResourceType    string
	Pattern         *DriftPattern
	Confidence      float64
	RiskLevel       RiskLevel
	RiskScore       float64
	PredictedAt     time.Time
	Factors         []RiskFactor
	Recommendations []string
}

// RiskFactor represents a factor contributing to the risk score
type RiskFactor struct {
	Name        string
	Value       float64
	Weight      float64
	Description string
}

// RiskScorer calculates risk scores for resources
type RiskScorer struct {
	factors map[string]float64
	weights map[string]float64
}

// RiskScore represents a calculated risk score
type RiskScore struct {
	Value   float64
	Level   RiskLevel
	Factors []RiskFactor
}

// PredictorConfig holds configuration for the drift predictor
type PredictorConfig struct {
	MinConfidence  float64
	MaxHistoryAge  time.Duration
	PatternWeights map[string]float64
	RiskThresholds map[RiskLevel]float64
}

// NewDriftPredictor creates a new drift predictor instance
func NewDriftPredictor(config *PredictorConfig) *DriftPredictor {
	if config == nil {
		config = &PredictorConfig{
			MinConfidence: 0.7,
			MaxHistoryAge: 30 * 24 * time.Hour, // 30 days
			PatternWeights: map[string]float64{
				"security":     0.3,
				"cost":         0.2,
				"compliance":   0.25,
				"performance":  0.15,
				"availability": 0.1,
			},
			RiskThresholds: map[RiskLevel]float64{
				RiskLevelLow:      0.3,
				RiskLevelMedium:   0.5,
				RiskLevelHigh:     0.7,
				RiskLevelCritical: 0.9,
			},
		}
	}

	return &DriftPredictor{
		patterns:   make(map[string]*DriftPattern),
		history:    []DriftEvent{},
		riskScorer: NewRiskScorer(),
		config:     config,
	}
}

// NewRiskScorer creates a new risk scorer instance
func NewRiskScorer() *RiskScorer {
	return &RiskScorer{
		factors: map[string]float64{
			"security":     0.3,
			"cost":         0.2,
			"compliance":   0.25,
			"performance":  0.15,
			"availability": 0.1,
		},
		weights: map[string]float64{
			"security":     1.0,
			"cost":         0.8,
			"compliance":   1.2,
			"performance":  0.6,
			"availability": 0.9,
		},
	}
}

// RegisterPattern registers a new drift pattern
func (dp *DriftPredictor) RegisterPattern(pattern *DriftPattern) {
	dp.patterns[pattern.ID] = pattern
}

// AddDriftEvent adds a drift event to the history
func (dp *DriftPredictor) AddDriftEvent(event DriftEvent) {
	dp.history = append(dp.history, event)

	// Clean up old events
	dp.cleanupHistory()
}

// PredictDrifts analyzes resources and predicts potential drifts
func (dp *DriftPredictor) PredictDrifts(ctx context.Context, resources []models.Resource) []PredictedDrift {
	var predictions []PredictedDrift

	for _, resource := range resources {
		// Check for patterns that match this resource
		patterns := dp.findMatchingPatterns(resource)

		for _, pattern := range patterns {
			confidence := dp.calculateConfidence(pattern, resource)

			if confidence >= dp.config.MinConfidence {
				riskScore := dp.riskScorer.CalculateRisk(resource)

				prediction := PredictedDrift{
					ID:              fmt.Sprintf("pred_%s_%s", resource.ID, pattern.ID),
					ResourceID:      resource.ID,
					ResourceName:    resource.Name,
					ResourceType:    resource.Type,
					Pattern:         pattern,
					Confidence:      confidence,
					RiskLevel:       riskScore.Level,
					RiskScore:       riskScore.Value,
					PredictedAt:     time.Now(),
					Factors:         riskScore.Factors,
					Recommendations: dp.generateRecommendations(pattern, resource),
				}

				predictions = append(predictions, prediction)
			}
		}
	}

	// Sort by risk score (highest first)
	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i].RiskScore > predictions[j].RiskScore
	})

	return predictions
}

// findMatchingPatterns finds patterns that match a resource
func (dp *DriftPredictor) findMatchingPatterns(resource models.Resource) []*DriftPattern {
	var matching []*DriftPattern

	for _, pattern := range dp.patterns {
		if dp.matchesPattern(resource, pattern) {
			matching = append(matching, pattern)
		}
	}

	return matching
}

// matchesPattern checks if a resource matches a pattern
func (dp *DriftPredictor) matchesPattern(resource models.Resource, pattern *DriftPattern) bool {
	for _, condition := range pattern.Conditions {
		if !dp.evaluateCondition(resource, condition) {
			return false
		}
	}
	return true
}

// evaluateCondition evaluates a single pattern condition
func (dp *DriftPredictor) evaluateCondition(resource models.Resource, condition PatternCondition) bool {
	switch condition.Field {
	case "type":
		return resource.Type == condition.Value.(string)
	case "provider":
		return resource.Provider == condition.Value.(string)
	case "region":
		return resource.Region == condition.Value.(string)
	case "state":
		return resource.State == condition.Value.(string)
	case "tags":
		if tags, ok := condition.Value.(map[string]string); ok {
			for key, value := range tags {
				if resource.Tags[key] != value {
					return false
				}
			}
			return true
		}
	}
	return false
}

// calculateConfidence calculates the confidence level for a prediction
func (dp *DriftPredictor) calculateConfidence(pattern *DriftPattern, resource models.Resource) float64 {
	baseConfidence := pattern.Confidence

	// Adjust based on historical data
	historicalConfidence := dp.getHistoricalConfidence(pattern.ID, resource.Type)

	// Adjust based on resource characteristics
	resourceConfidence := dp.getResourceConfidence(resource)

	// Weighted average
	confidence := (baseConfidence*0.4 + historicalConfidence*0.4 + resourceConfidence*0.2)

	return math.Min(confidence, 1.0)
}

// getHistoricalConfidence gets confidence based on historical patterns
func (dp *DriftPredictor) getHistoricalConfidence(patternID, resourceType string) float64 {
	var matchingEvents int
	var totalEvents int

	for _, event := range dp.history {
		if event.ResourceType == resourceType {
			totalEvents++
			if event.Pattern == patternID {
				matchingEvents++
			}
		}
	}

	if totalEvents == 0 {
		return 0.5 // Default confidence
	}

	return float64(matchingEvents) / float64(totalEvents)
}

// getResourceConfidence calculates confidence based on resource characteristics
func (dp *DriftPredictor) getResourceConfidence(resource models.Resource) float64 {
	confidence := 0.5 // Base confidence

	// Higher confidence for resources with more tags (more context)
	if len(resource.Tags) > 5 {
		confidence += 0.1
	}

	// Higher confidence for resources that have been around longer
	if !resource.Created.IsZero() {
		age := time.Since(resource.Created)
		if age > 7*24*time.Hour { // More than a week old
			confidence += 0.1
		}
	}

	return math.Min(confidence, 1.0)
}

// CalculateRisk calculates the risk score for a resource
func (rs *RiskScorer) CalculateRisk(resource models.Resource) RiskScore {
	score := 0.0
	var factors []RiskFactor

	// Security risk factors
	if resource.Type == "aws_security_group" || resource.Type == "azure_network_security_group" {
		securityScore := rs.factors["security"] * rs.weights["security"]
		score += securityScore
		factors = append(factors, RiskFactor{
			Name:        "security",
			Value:       securityScore,
			Weight:      rs.weights["security"],
			Description: "Security-related resource",
		})
	}

	// Cost risk factors
	if resource.Type == "aws_instance" || resource.Type == "azure_virtual_machine" {
		costScore := rs.factors["cost"] * rs.weights["cost"]
		score += costScore
		factors = append(factors, RiskFactor{
			Name:        "cost",
			Value:       costScore,
			Weight:      rs.weights["cost"],
			Description: "Compute resource with potential cost impact",
		})
	}

	// Compliance risk factors
	if resource.Tags != nil {
		if _, hasCompliance := resource.Tags["compliance"]; hasCompliance {
			complianceScore := rs.factors["compliance"] * rs.weights["compliance"]
			score += complianceScore
			factors = append(factors, RiskFactor{
				Name:        "compliance",
				Value:       complianceScore,
				Weight:      rs.weights["compliance"],
				Description: "Compliance-related resource",
			})
		}
	}

	// Performance risk factors
	if resource.Type == "aws_rds_instance" || resource.Type == "azure_sql_database" {
		performanceScore := rs.factors["performance"] * rs.weights["performance"]
		score += performanceScore
		factors = append(factors, RiskFactor{
			Name:        "performance",
			Value:       performanceScore,
			Weight:      rs.weights["performance"],
			Description: "Database resource with performance impact",
		})
	}

	// Availability risk factors
	if resource.Type == "aws_load_balancer" || resource.Type == "azure_lb" {
		availabilityScore := rs.factors["availability"] * rs.weights["availability"]
		score += availabilityScore
		factors = append(factors, RiskFactor{
			Name:        "availability",
			Value:       availabilityScore,
			Weight:      rs.weights["availability"],
			Description: "Load balancer with availability impact",
		})
	}

	return RiskScore{
		Value:   score,
		Level:   rs.determineLevel(score),
		Factors: factors,
	}
}

// determineLevel determines the risk level based on score
func (rs *RiskScorer) determineLevel(score float64) RiskLevel {
	switch {
	case score <= 0.3:
		return RiskLevelLow
	case score <= 0.5:
		return RiskLevelMedium
	case score <= 0.7:
		return RiskLevelHigh
	default:
		return RiskLevelCritical
	}
}

// generateRecommendations generates recommendations for a predicted drift
func (dp *DriftPredictor) generateRecommendations(pattern *DriftPattern, resource models.Resource) []string {
	var recommendations []string

	switch pattern.ID {
	case "security_group_open":
		recommendations = append(recommendations,
			"Review and restrict security group rules",
			"Implement least-privilege access principles",
			"Consider using security group references instead of CIDR blocks",
		)
	case "unused_resource":
		recommendations = append(recommendations,
			"Review resource usage and consider termination if unused",
			"Implement resource tagging for better tracking",
			"Set up automated cleanup for unused resources",
		)
	case "cost_optimization":
		recommendations = append(recommendations,
			"Review instance sizing and consider downsizing",
			"Implement auto-scaling for variable workloads",
			"Consider reserved instances for predictable workloads",
		)
	case "compliance_violation":
		recommendations = append(recommendations,
			"Review compliance requirements and update resource configuration",
			"Implement compliance monitoring and alerting",
			"Consider using compliance-focused resource templates",
		)
	default:
		recommendations = append(recommendations,
			"Review resource configuration and ensure it aligns with best practices",
			"Monitor resource usage and performance metrics",
			"Implement automated drift detection and remediation",
		)
	}

	return recommendations
}

// cleanupHistory removes old drift events from history
func (dp *DriftPredictor) cleanupHistory() {
	cutoff := time.Now().Add(-dp.config.MaxHistoryAge)
	var filtered []DriftEvent

	for _, event := range dp.history {
		if event.Timestamp.After(cutoff) {
			filtered = append(filtered, event)
		}
	}

	dp.history = filtered
}

// GetPredictionStats returns statistics about predictions
func (dp *DriftPredictor) GetPredictionStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_patterns": len(dp.patterns),
		"total_events":   len(dp.history),
		"patterns":       make(map[string]interface{}),
	}

	for id, pattern := range dp.patterns {
		stats["patterns"].(map[string]interface{})[id] = map[string]interface{}{
			"name":       pattern.Name,
			"confidence": pattern.Confidence,
			"risk_level": pattern.RiskLevel,
			"weight":     pattern.Weight,
		}
	}

	return stats
}
