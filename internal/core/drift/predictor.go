package drift

import (
	"github.com/catherinevee/driftmgr/internal/models"
)

// Predictor provides drift prediction capabilities
type Predictor struct{}

// NewPredictor creates a new drift predictor
func NewPredictor() *Predictor {
	return &Predictor{}
}

// Predict generates drift predictions
func (p *Predictor) Predict(drifts []models.DriftItem, analysis *Analysis) *Predictions {
	return &Predictions{
		FutureDrift: p.predictFutureDrift(drifts),
		Likelihood:  p.calculateLikelihood(analysis),
		TimeFrame:   "7 days",
		PreventiveActions: []string{
			"Enable drift detection automation",
			"Review and update IaC templates",
			"Implement stricter change controls",
		},
	}
}

func (p *Predictor) predictFutureDrift(drifts []models.DriftItem) []FutureDrift {
	predictions := []FutureDrift{}

	// Analyze patterns to predict future drift
	resourceTypes := make(map[string]int)
	for _, drift := range drifts {
		resourceTypes[drift.ResourceType]++
	}

	for resourceType, count := range resourceTypes {
		if count > 2 {
			predictions = append(predictions, FutureDrift{
				ResourceType: resourceType,
				Probability:  float64(count) / float64(len(drifts)),
				TimeFrame:    "within 7 days",
				Reason:       "Historical pattern detected",
			})
		}
	}

	return predictions
}

func (p *Predictor) calculateLikelihood(analysis *Analysis) float64 {
	if analysis.RiskLevel == "critical" {
		return 0.9
	} else if analysis.RiskLevel == "high" {
		return 0.7
	} else if analysis.RiskLevel == "medium" {
		return 0.5
	}
	return 0.3
}
