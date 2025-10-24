package filtering

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/catherinevee/driftmgr/internal/events"
	internalModels "github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/pkg/models"
)

// ImportanceLevel represents the importance level of a resource or drift
type ImportanceLevel string

const (
	ImportanceLevelCritical ImportanceLevel = "critical"
	ImportanceLevelHigh     ImportanceLevel = "high"
	ImportanceLevelMedium   ImportanceLevel = "medium"
	ImportanceLevelLow      ImportanceLevel = "low"
	ImportanceLevelMinimal  ImportanceLevel = "minimal"
)

// ImportanceScore represents an importance score for a resource or drift
type ImportanceScore struct {
	ResourceID      string                 `json:"resource_id"`
	ResourceName    string                 `json:"resource_name"`
	ResourceType    string                 `json:"resource_type"`
	Provider        string                 `json:"provider"`
	Region          string                 `json:"region"`
	ImportanceLevel ImportanceLevel        `json:"importance_level"`
	Score           float64                `json:"score"`
	Factors         []ImportanceFactor     `json:"factors"`
	CalculatedAt    time.Time              `json:"calculated_at"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// ImportanceFactor represents a factor that contributes to the importance score
type ImportanceFactor struct {
	Name         string  `json:"name"`
	Weight       float64 `json:"weight"`
	Value        float64 `json:"value"`
	Contribution float64 `json:"contribution"`
	Description  string  `json:"description"`
}

// ImportanceFilterService provides importance filtering functionality
type ImportanceFilterService struct {
	config       ImportanceConfig
	eventBus     *events.EventBus
	scoringRules []ImportanceScoringRule
}

// ImportanceConfig represents configuration for the importance filter service
type ImportanceConfig struct {
	DefaultWeights      map[string]float64            `json:"default_weights"`
	ResourceTypeWeights map[string]map[string]float64 `json:"resource_type_weights"`
	SeverityWeights     map[string]float64            `json:"severity_weights"`
	ImpactWeights       map[string]float64            `json:"impact_weights"`
	TimeDecayFactor     float64                       `json:"time_decay_factor"`
	MaxScore            float64                       `json:"max_score"`
	MinScore            float64                       `json:"min_score"`
}

// ImportanceScoringRule represents a rule for calculating importance scores
type ImportanceScoringRule struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Weight      float64               `json:"weight"`
	Conditions  []ImportanceCondition `json:"conditions"`
	Calculator  ImportanceCalculator  `json:"-"`
}

// ImportanceCondition represents a condition for importance scoring
type ImportanceCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// ImportanceCalculator is a function that calculates importance contribution
type ImportanceCalculator func(resource models.Resource, drift *internalModels.DriftRecord) float64

// NewImportanceFilterService creates a new importance filter service
func NewImportanceFilterService(config ImportanceConfig, eventBus *events.EventBus) *ImportanceFilterService {
	service := &ImportanceFilterService{
		config:   config,
		eventBus: eventBus,
	}

	// Initialize default scoring rules
	service.initializeDefaultRules()

	return service
}

// CalculateImportanceScore calculates the importance score for a resource
func (ifs *ImportanceFilterService) CalculateImportanceScore(ctx context.Context, resource models.Resource, drift *internalModels.DriftRecord) (*ImportanceScore, error) {
	score := &ImportanceScore{
		ResourceID:   resource.ID,
		ResourceName: resource.Name,
		ResourceType: resource.Type,
		Provider:     resource.Provider,
		Region:       resource.Region,
		CalculatedAt: time.Now(),
		Factors:      make([]ImportanceFactor, 0),
		Metadata:     make(map[string]interface{}),
	}

	totalScore := 0.0
	totalWeight := 0.0

	// Apply each scoring rule
	for _, rule := range ifs.scoringRules {
		if ifs.matchesConditions(resource, drift, rule.Conditions) {
			contribution := rule.Calculator(resource, drift)
			factor := ImportanceFactor{
				Name:         rule.Name,
				Weight:       rule.Weight,
				Value:        contribution,
				Contribution: contribution * rule.Weight,
				Description:  rule.Description,
			}
			score.Factors = append(score.Factors, factor)
			totalScore += factor.Contribution
			totalWeight += rule.Weight
		}
	}

	// Normalize score
	if totalWeight > 0 {
		score.Score = totalScore / totalWeight
	}

	// Apply bounds
	if score.Score > ifs.config.MaxScore {
		score.Score = ifs.config.MaxScore
	}
	if score.Score < ifs.config.MinScore {
		score.Score = ifs.config.MinScore
	}

	// Determine importance level
	score.ImportanceLevel = ifs.determineImportanceLevel(score.Score)

	// Publish importance calculated event
	ifs.publishImportanceEvent(score)

	return score, nil
}

// CalculateImportanceScores calculates importance scores for multiple resources
func (ifs *ImportanceFilterService) CalculateImportanceScores(ctx context.Context, resources []models.Resource, drifts []internalModels.DriftRecord) ([]ImportanceScore, error) {
	var scores []ImportanceScore

	// Create drift map for quick lookup
	driftMap := make(map[string][]internalModels.DriftRecord)
	for _, drift := range drifts {
		driftMap[drift.ResourceID] = append(driftMap[drift.ResourceID], drift)
	}

	// Calculate scores for each resource
	for _, resource := range resources {
		var resourceDrifts []internalModels.DriftRecord
		if drifts, exists := driftMap[resource.ID]; exists {
			resourceDrifts = drifts
		}

		// Use the most recent drift for scoring
		var mostRecentDrift *internalModels.DriftRecord
		if len(resourceDrifts) > 0 {
			mostRecentDrift = &resourceDrifts[0]
			for i := 1; i < len(resourceDrifts); i++ {
				if resourceDrifts[i].DetectedAt.After(mostRecentDrift.DetectedAt) {
					mostRecentDrift = &resourceDrifts[i]
				}
			}
		}

		score, err := ifs.CalculateImportanceScore(ctx, resource, mostRecentDrift)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate importance score for resource %s: %w", resource.ID, err)
		}

		scores = append(scores, *score)
	}

	// Sort by importance score (descending)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	return scores, nil
}

// FilterByImportance filters resources by importance level
func (ifs *ImportanceFilterService) FilterByImportance(scores []ImportanceScore, level ImportanceLevel) []ImportanceScore {
	var filtered []ImportanceScore
	for _, score := range scores {
		if score.ImportanceLevel == level {
			filtered = append(filtered, score)
		}
	}
	return filtered
}

// FilterByImportanceRange filters resources by importance score range
func (ifs *ImportanceFilterService) FilterByImportanceRange(scores []ImportanceScore, minScore, maxScore float64) []ImportanceScore {
	var filtered []ImportanceScore
	for _, score := range scores {
		if score.Score >= minScore && score.Score <= maxScore {
			filtered = append(filtered, score)
		}
	}
	return filtered
}

// GetTopImportantResources returns the top N most important resources
func (ifs *ImportanceFilterService) GetTopImportantResources(scores []ImportanceScore, limit int) []ImportanceScore {
	if limit <= 0 || limit >= len(scores) {
		return scores
	}
	return scores[:limit]
}

// GetImportanceDistribution returns the distribution of importance levels
func (ifs *ImportanceFilterService) GetImportanceDistribution(scores []ImportanceScore) map[ImportanceLevel]int {
	distribution := make(map[ImportanceLevel]int)
	for _, score := range scores {
		distribution[score.ImportanceLevel]++
	}
	return distribution
}

// GetImportanceStatistics returns statistics about importance scores
func (ifs *ImportanceFilterService) GetImportanceStatistics(scores []ImportanceScore) map[string]interface{} {
	if len(scores) == 0 {
		return map[string]interface{}{
			"count": 0,
		}
	}

	stats := make(map[string]interface{})
	stats["count"] = len(scores)

	// Calculate average score
	totalScore := 0.0
	for _, score := range scores {
		totalScore += score.Score
	}
	stats["average_score"] = totalScore / float64(len(scores))

	// Find min and max scores
	minScore := scores[0].Score
	maxScore := scores[0].Score
	for _, score := range scores {
		if score.Score < minScore {
			minScore = score.Score
		}
		if score.Score > maxScore {
			maxScore = score.Score
		}
	}
	stats["min_score"] = minScore
	stats["max_score"] = maxScore

	// Calculate distribution
	stats["distribution"] = ifs.GetImportanceDistribution(scores)

	return stats
}

// Helper methods

func (ifs *ImportanceFilterService) initializeDefaultRules() {
	ifs.scoringRules = []ImportanceScoringRule{
		{
			Name:        "resource_type_importance",
			Description: "Importance based on resource type",
			Weight:      0.3,
			Calculator:  ifs.calculateResourceTypeImportance,
		},
		{
			Name:        "drift_severity",
			Description: "Importance based on drift severity",
			Weight:      0.4,
			Calculator:  ifs.calculateDriftSeverityImportance,
		},
		{
			Name:        "drift_frequency",
			Description: "Importance based on drift frequency",
			Weight:      0.2,
			Calculator:  ifs.calculateDriftFrequencyImportance,
		},
		{
			Name:        "resource_age",
			Description: "Importance based on resource age",
			Weight:      0.1,
			Calculator:  ifs.calculateResourceAgeImportance,
		},
	}
}

func (ifs *ImportanceFilterService) matchesConditions(resource models.Resource, drift *internalModels.DriftRecord, conditions []ImportanceCondition) bool {
	// This is a simplified implementation
	// In a real implementation, you would evaluate each condition
	return true
}

func (ifs *ImportanceFilterService) calculateResourceTypeImportance(resource models.Resource, drift *internalModels.DriftRecord) float64 {
	// Define importance weights for different resource types
	resourceTypeWeights := map[string]float64{
		"aws_s3_bucket":           0.9,
		"aws_ec2_instance":        0.8,
		"aws_rds_instance":        0.9,
		"aws_lambda_function":     0.7,
		"aws_iam_role":            0.8,
		"aws_iam_policy":          0.8,
		"aws_vpc":                 0.7,
		"aws_security_group":      0.6,
		"aws_subnet":              0.5,
		"azurerm_storage_account": 0.9,
		"azurerm_virtual_machine": 0.8,
		"azurerm_sql_database":    0.9,
		"google_storage_bucket":   0.9,
		"google_compute_instance": 0.8,
		"google_sql_database":     0.9,
		"digitalocean_droplet":    0.7,
		"digitalocean_volume":     0.6,
	}

	if weight, exists := resourceTypeWeights[resource.Type]; exists {
		return weight
	}
	return 0.5 // Default weight
}

func (ifs *ImportanceFilterService) calculateDriftSeverityImportance(resource models.Resource, drift *internalModels.DriftRecord) float64 {
	if drift == nil {
		return 0.0
	}

	severityWeights := map[string]float64{
		"critical": 1.0,
		"high":     0.8,
		"medium":   0.6,
		"low":      0.4,
		"minimal":  0.2,
	}

	if weight, exists := severityWeights[drift.Severity]; exists {
		return weight
	}
	return 0.5 // Default weight
}

func (ifs *ImportanceFilterService) calculateDriftFrequencyImportance(resource models.Resource, drift *internalModels.DriftRecord) float64 {
	if drift == nil {
		return 0.0
	}

	// This is a simplified implementation
	// In a real implementation, you would count the number of drifts for this resource
	return 0.5
}

func (ifs *ImportanceFilterService) calculateResourceAgeImportance(resource models.Resource, drift *internalModels.DriftRecord) float64 {
	// Calculate resource age
	var resourceAge time.Duration
	if !resource.CreatedAt.IsZero() {
		resourceAge = time.Since(resource.CreatedAt)
	} else if !resource.Created.IsZero() {
		resourceAge = time.Since(resource.Created)
	} else {
		return 0.5 // Default weight if no creation time
	}

	// Older resources are generally more important
	days := resourceAge.Hours() / 24
	if days > 365 {
		return 0.9 // Very old resources
	} else if days > 180 {
		return 0.7 // Old resources
	} else if days > 90 {
		return 0.5 // Medium age resources
	} else if days > 30 {
		return 0.3 // New resources
	} else {
		return 0.1 // Very new resources
	}
}

func (ifs *ImportanceFilterService) determineImportanceLevel(score float64) ImportanceLevel {
	if score >= 0.9 {
		return ImportanceLevelCritical
	} else if score >= 0.7 {
		return ImportanceLevelHigh
	} else if score >= 0.5 {
		return ImportanceLevelMedium
	} else if score >= 0.3 {
		return ImportanceLevelLow
	} else {
		return ImportanceLevelMinimal
	}
}

func (ifs *ImportanceFilterService) publishImportanceEvent(score *ImportanceScore) {
	if ifs.eventBus == nil {
		return
	}

	event := events.Event{
		Type:      events.EventType("importance.calculated"),
		Timestamp: time.Now(),
		Source:    "importance_filter_service",
		Data: map[string]interface{}{
			"resource_id":      score.ResourceID,
			"resource_name":    score.ResourceName,
			"resource_type":    score.ResourceType,
			"provider":         score.Provider,
			"region":           score.Region,
			"importance_level": string(score.ImportanceLevel),
			"score":            score.Score,
			"factor_count":     len(score.Factors),
		},
	}

	ifs.eventBus.Publish(event)
}
