package analysis

import (
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// AnalysisResult represents the result of a drift analysis
type AnalysisResult struct {
	ResourceID   string                 `json:"resource_id"`
	ResourceName string                 `json:"resource_name"`
	ResourceType string                 `json:"resource_type"`
	DriftType    string                 `json:"drift_type"`
	Severity     string                 `json:"severity"`
	Confidence   float64                `json:"confidence"`
	Description  string                 `json:"description"`
	DetectedAt   time.Time              `json:"detected_at"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// EnhancedConfig represents enhanced configuration for drift analysis
type EnhancedConfig struct {
	EnableMLPrediction  bool                   `json:"enable_ml_prediction"`
	ConfidenceThreshold float64                `json:"confidence_threshold"`
	RiskScoringEnabled  bool                   `json:"risk_scoring_enabled"`
	CustomRules         map[string]interface{} `json:"custom_rules"`
	AnalysisTimeout     time.Duration          `json:"analysis_timeout"`
	SecurityGroupWeight float64                `json:"security_group_weight"`
	EnvironmentWeight   float64                `json:"environment_weight"`
}

// DriftAnalyzer represents an enhanced drift analyzer
type DriftAnalyzer struct {
	config   *EnhancedConfig
	patterns []DriftPattern
}

// NewDriftAnalyzer creates a new drift analyzer
func NewDriftAnalyzer(config ...*EnhancedConfig) *DriftAnalyzer {
	var cfg *EnhancedConfig
	if len(config) > 0 {
		cfg = config[0]
	}
	if cfg == nil {
		cfg = &EnhancedConfig{
			EnableMLPrediction:  true,
			ConfidenceThreshold: 0.7,
			RiskScoringEnabled:  true,
			AnalysisTimeout:     30 * time.Second,
		}
	}

	return &DriftAnalyzer{
		config:   cfg,
		patterns: make([]DriftPattern, 0),
	}
}

// NewEnhancedDriftAnalyzer creates a new enhanced drift analyzer
func NewEnhancedDriftAnalyzer(config *EnhancedConfig) *DriftAnalyzer {
	return NewDriftAnalyzer(config)
}

// Analyze performs drift analysis on resources
func (da *DriftAnalyzer) Analyze(resources []models.Resource) []AnalysisResult {
	var results []AnalysisResult

	for _, resource := range resources {
		// Basic drift detection
		if da.config.EnableMLPrediction {
			// Simulate ML-based prediction
			resourceTags := resource.GetTagsAsMap()
			if publicTag, exists := resourceTags["public"]; exists && publicTag == "true" {
				results = append(results, AnalysisResult{
					ResourceID:   resource.ID,
					ResourceName: resource.Name,
					ResourceType: resource.Type,
					DriftType:    "security_risk",
					Severity:     "high",
					Confidence:   0.85,
					Description:  "Resource has public access",
					DetectedAt:   time.Now(),
					Metadata:     map[string]interface{}{"risk_score": 0.85},
				})
			}
		}

		// Check for unused resources
		if resource.Created.Before(time.Now().AddDate(0, -1, 0)) {
			results = append(results, AnalysisResult{
				ResourceID:   resource.ID,
				ResourceName: resource.Name,
				ResourceType: resource.Type,
				DriftType:    "unused_resource",
				Severity:     "medium",
				Confidence:   0.75,
				Description:  "Resource may be unused",
				DetectedAt:   time.Now(),
				Metadata:     map[string]interface{}{"age_days": time.Since(resource.Created).Hours() / 24},
			})
		}
	}

	return results
}

// DriftPattern represents a drift detection pattern
type DriftPattern struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// AnalysisResultWithDrifts represents analysis result with drifts
type AnalysisResultWithDrifts struct {
	Drifts    []AnalysisResult `json:"drifts"`
	RiskScore float64          `json:"risk_score"`
	Total     int              `json:"total"`
}

// AnalyzeWithDrifts performs drift analysis and returns result with drifts
func (da *DriftAnalyzer) AnalyzeWithDrifts(resources []models.Resource) *AnalysisResultWithDrifts {
	results := da.Analyze(resources)

	riskScore := 0.0
	for _, result := range results {
		if result.Severity == "high" {
			riskScore += 0.3
		} else if result.Severity == "medium" {
			riskScore += 0.2
		} else {
			riskScore += 0.1
		}
	}

	return &AnalysisResultWithDrifts{
		Drifts:    results,
		RiskScore: riskScore,
		Total:     len(results),
	}
}
