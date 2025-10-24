package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// Predictor handles predictive analytics and forecasting
type Predictor struct {
	// In a real implementation, this would have ML models and statistical libraries
}

// NewPredictor creates a new predictor
func NewPredictor() *Predictor {
	return &Predictor{}
}

// PredictCosts predicts future costs based on historical data
func (p *Predictor) PredictCosts(ctx context.Context, historicalData []map[string]interface{}, predictionPeriod time.Duration) (*models.CostPrediction, error) {
	if len(historicalData) == 0 {
		return nil, fmt.Errorf("no historical data provided")
	}

	// Extract cost data
	costs := p.extractCostData(historicalData)
	if len(costs) < 2 {
		return nil, fmt.Errorf("insufficient data for prediction")
	}

	// Perform prediction
	prediction := &models.CostPrediction{
		PredictionPeriod: predictionPeriod,
		GeneratedAt:      time.Now(),
	}

	// Simple linear regression prediction
	slope, intercept := p.linearRegression(costs)

	// Calculate predictions for the next period
	predictions := p.calculateCostPredictions(costs, slope, intercept, predictionPeriod)
	prediction.Predictions = predictions

	// Calculate confidence interval
	prediction.ConfidenceInterval = p.calculateConfidenceInterval(costs, slope, intercept)

	// Generate insights
	prediction.Insights = p.generateCostPredictionInsights(predictions, costs)

	return prediction, nil
}

// PredictResourceUsage predicts future resource usage
func (p *Predictor) PredictResourceUsage(ctx context.Context, historicalData []map[string]interface{}, predictionPeriod time.Duration) (*models.ResourceUsagePrediction, error) {
	if len(historicalData) == 0 {
		return nil, fmt.Errorf("no historical data provided")
	}

	// Extract usage data
	usage := p.extractUsageData(historicalData)
	if len(usage) < 2 {
		return nil, fmt.Errorf("insufficient data for prediction")
	}

	// Perform prediction
	prediction := &models.ResourceUsagePrediction{
		PredictionPeriod: predictionPeriod,
		GeneratedAt:      time.Now(),
	}

	// Simple linear regression prediction
	slope, intercept := p.linearRegression(usage)

	// Calculate predictions for the next period
	predictions := p.calculateUsagePredictions(usage, slope, intercept, predictionPeriod)
	prediction.Predictions = predictions

	// Calculate confidence interval
	prediction.ConfidenceInterval = p.calculateConfidenceInterval(usage, slope, intercept)

	// Generate insights
	prediction.Insights = p.generateUsagePredictionInsights(predictions, usage)

	return prediction, nil
}

// PredictDrift predicts potential drift scenarios
func (p *Predictor) PredictDrift(ctx context.Context, historicalData []map[string]interface{}, predictionPeriod time.Duration) (*models.DriftPrediction, error) {
	if len(historicalData) == 0 {
		return nil, fmt.Errorf("no historical data provided")
	}

	// Extract drift data
	drift := p.extractDriftData(historicalData)
	if len(drift) < 2 {
		return nil, fmt.Errorf("insufficient data for prediction")
	}

	// Perform prediction
	prediction := &models.DriftPrediction{
		PredictionPeriod: predictionPeriod,
		GeneratedAt:      time.Now(),
	}

	// Simple linear regression prediction
	slope, intercept := p.linearRegression(drift)

	// Calculate predictions for the next period
	predictions := p.calculateDriftPredictions(drift, slope, intercept, predictionPeriod)
	prediction.Predictions = predictions

	// Calculate confidence interval
	prediction.ConfidenceInterval = p.calculateConfidenceInterval(drift, slope, intercept)

	// Generate insights
	prediction.Insights = p.generateDriftPredictionInsights(predictions, drift)

	return prediction, nil
}

// PredictPerformance predicts future performance metrics
func (p *Predictor) PredictPerformance(ctx context.Context, historicalData []map[string]interface{}, predictionPeriod time.Duration) (*models.PerformancePrediction, error) {
	if len(historicalData) == 0 {
		return nil, fmt.Errorf("no historical data provided")
	}

	// Extract performance data
	performance := p.extractPerformanceData(historicalData)
	if len(performance) < 2 {
		return nil, fmt.Errorf("insufficient data for prediction")
	}

	// Perform prediction
	prediction := &models.PerformancePrediction{
		PredictionPeriod: predictionPeriod,
		GeneratedAt:      time.Now(),
	}

	// Simple linear regression prediction
	slope, intercept := p.linearRegression(performance)

	// Calculate predictions for the next period
	predictions := p.calculatePerformancePredictions(performance, slope, intercept, predictionPeriod)
	prediction.Predictions = predictions

	// Calculate confidence interval
	prediction.ConfidenceInterval = p.calculateConfidenceInterval(performance, slope, intercept)

	// Generate insights
	prediction.Insights = p.generatePerformancePredictionInsights(predictions, performance)

	return prediction, nil
}

// Helper methods

// extractCostData extracts cost data from historical data
func (p *Predictor) extractCostData(data []map[string]interface{}) []float64 {
	var costs []float64
	for _, item := range data {
		if cost, ok := item["cost"].(float64); ok {
			costs = append(costs, cost)
		}
	}
	return costs
}

// extractUsageData extracts usage data from historical data
func (p *Predictor) extractUsageData(data []map[string]interface{}) []float64 {
	var usage []float64
	for _, item := range data {
		if count, ok := item["count"].(float64); ok {
			usage = append(usage, count)
		}
	}
	return usage
}

// extractDriftData extracts drift data from historical data
func (p *Predictor) extractDriftData(data []map[string]interface{}) []float64 {
	var drift []float64
	for _, item := range data {
		if rate, ok := item["drift_rate"].(float64); ok {
			drift = append(drift, rate)
		}
	}
	return drift
}

// extractPerformanceData extracts performance data from historical data
func (p *Predictor) extractPerformanceData(data []map[string]interface{}) []float64 {
	var performance []float64
	for _, item := range data {
		if avg, ok := item["average"].(float64); ok {
			performance = append(performance, avg)
		}
	}
	return performance
}

// linearRegression performs simple linear regression
func (p *Predictor) linearRegression(values []float64) (slope, intercept float64) {
	n := float64(len(values))
	if n < 2 {
		return 0.0, 0.0
	}

	// Calculate means
	sumX, sumY := 0.0, 0.0
	for i, v := range values {
		sumX += float64(i)
		sumY += v
	}
	meanX := sumX / n
	meanY := sumY / n

	// Calculate slope and intercept
	var numerator, denominator float64
	for i, v := range values {
		x := float64(i)
		numerator += (x - meanX) * (v - meanY)
		denominator += (x - meanX) * (x - meanX)
	}

	if denominator == 0 {
		return 0.0, meanY
	}

	slope = numerator / denominator
	intercept = meanY - slope*meanX

	return slope, intercept
}

// calculateCostPredictions calculates cost predictions
func (p *Predictor) calculateCostPredictions(historical []float64, slope, intercept float64, period time.Duration) []models.CostPredictionPoint {
	var predictions []models.CostPredictionPoint

	// Calculate number of prediction points based on period
	points := int(period.Hours() / 24) // Daily predictions
	if points < 1 {
		points = 1
	}

	startIndex := len(historical)
	for i := 0; i < points; i++ {
		x := float64(startIndex + i)
		predicted := slope*x + intercept

		// Ensure prediction is not negative
		if predicted < 0 {
			predicted = 0
		}

		predictions = append(predictions, models.CostPredictionPoint{
			Date:      time.Now().AddDate(0, 0, i+1),
			Predicted: predicted,
			Lower:     predicted * 0.9, // 10% lower bound
			Upper:     predicted * 1.1, // 10% upper bound
		})
	}

	return predictions
}

// calculateUsagePredictions calculates usage predictions
func (p *Predictor) calculateUsagePredictions(historical []float64, slope, intercept float64, period time.Duration) []models.UsagePredictionPoint {
	var predictions []models.UsagePredictionPoint

	// Calculate number of prediction points based on period
	points := int(period.Hours() / 24) // Daily predictions
	if points < 1 {
		points = 1
	}

	startIndex := len(historical)
	for i := 0; i < points; i++ {
		x := float64(startIndex + i)
		predicted := slope*x + intercept

		// Ensure prediction is not negative
		if predicted < 0 {
			predicted = 0
		}

		predictions = append(predictions, models.UsagePredictionPoint{
			Date:      time.Now().AddDate(0, 0, i+1),
			Predicted: predicted,
			Lower:     predicted * 0.9, // 10% lower bound
			Upper:     predicted * 1.1, // 10% upper bound
		})
	}

	return predictions
}

// calculateDriftPredictions calculates drift predictions
func (p *Predictor) calculateDriftPredictions(historical []float64, slope, intercept float64, period time.Duration) []models.DriftPredictionPoint {
	var predictions []models.DriftPredictionPoint

	// Calculate number of prediction points based on period
	points := int(period.Hours() / 24) // Daily predictions
	if points < 1 {
		points = 1
	}

	startIndex := len(historical)
	for i := 0; i < points; i++ {
		x := float64(startIndex + i)
		predicted := slope*x + intercept

		// Ensure prediction is not negative
		if predicted < 0 {
			predicted = 0
		}

		predictions = append(predictions, models.DriftPredictionPoint{
			Date:      time.Now().AddDate(0, 0, i+1),
			Predicted: predicted,
			Lower:     predicted * 0.9, // 10% lower bound
			Upper:     predicted * 1.1, // 10% upper bound
		})
	}

	return predictions
}

// calculatePerformancePredictions calculates performance predictions
func (p *Predictor) calculatePerformancePredictions(historical []float64, slope, intercept float64, period time.Duration) []models.PerformancePredictionPoint {
	var predictions []models.PerformancePredictionPoint

	// Calculate number of prediction points based on period
	points := int(period.Hours() / 24) // Daily predictions
	if points < 1 {
		points = 1
	}

	startIndex := len(historical)
	for i := 0; i < points; i++ {
		x := float64(startIndex + i)
		predicted := slope*x + intercept

		// Ensure prediction is not negative
		if predicted < 0 {
			predicted = 0
		}

		predictions = append(predictions, models.PerformancePredictionPoint{
			Date:      time.Now().AddDate(0, 0, i+1),
			Predicted: predicted,
			Lower:     predicted * 0.9, // 10% lower bound
			Upper:     predicted * 1.1, // 10% upper bound
		})
	}

	return predictions
}

// calculateConfidenceInterval calculates confidence interval for predictions
func (p *Predictor) calculateConfidenceInterval(historical []float64, slope, intercept float64) models.ConfidenceInterval {
	// Simple confidence interval calculation
	// In production, this would use proper statistical methods

	// Calculate standard error
	var sumSquaredErrors float64
	for i, v := range historical {
		x := float64(i)
		predicted := slope*x + intercept
		error := v - predicted
		sumSquaredErrors += error * error
	}

	meanSquaredError := sumSquaredErrors / float64(len(historical))
	standardError := meanSquaredError // Simplified

	return models.ConfidenceInterval{
		Level: 0.95, // 95% confidence
		Lower: -standardError,
		Upper: standardError,
	}
}

// generateCostPredictionInsights generates insights for cost predictions
func (p *Predictor) generateCostPredictionInsights(predictions []models.CostPredictionPoint, historical []float64) []string {
	var insights []string

	if len(predictions) == 0 {
		return insights
	}

	// Calculate trend
	lastHistorical := historical[len(historical)-1]
	lastPrediction := predictions[len(predictions)-1].Predicted

	if lastPrediction > lastHistorical {
		increase := ((lastPrediction - lastHistorical) / lastHistorical) * 100
		insights = append(insights, fmt.Sprintf("Costs predicted to increase by %.1f%% over the prediction period", increase))
	} else if lastPrediction < lastHistorical {
		decrease := ((lastHistorical - lastPrediction) / lastHistorical) * 100
		insights = append(insights, fmt.Sprintf("Costs predicted to decrease by %.1f%% over the prediction period", decrease))
	} else {
		insights = append(insights, "Costs predicted to remain stable")
	}

	// Check for significant changes
	if len(predictions) > 1 {
		first := predictions[0].Predicted
		last := predictions[len(predictions)-1].Predicted
		if first > 0 {
			change := ((last - first) / first) * 100
			if change > 10 {
				insights = append(insights, "Significant cost increase predicted within the period")
			} else if change < -10 {
				insights = append(insights, "Significant cost decrease predicted within the period")
			}
		}
	}

	return insights
}

// generateUsagePredictionInsights generates insights for usage predictions
func (p *Predictor) generateUsagePredictionInsights(predictions []models.UsagePredictionPoint, historical []float64) []string {
	var insights []string

	if len(predictions) == 0 {
		return insights
	}

	// Calculate trend
	lastHistorical := historical[len(historical)-1]
	lastPrediction := predictions[len(predictions)-1].Predicted

	if lastPrediction > lastHistorical {
		increase := ((lastPrediction - lastHistorical) / lastHistorical) * 100
		insights = append(insights, fmt.Sprintf("Resource usage predicted to increase by %.1f%% over the prediction period", increase))
	} else if lastPrediction < lastHistorical {
		decrease := ((lastHistorical - lastPrediction) / lastHistorical) * 100
		insights = append(insights, fmt.Sprintf("Resource usage predicted to decrease by %.1f%% over the prediction period", decrease))
	} else {
		insights = append(insights, "Resource usage predicted to remain stable")
	}

	return insights
}

// generateDriftPredictionInsights generates insights for drift predictions
func (p *Predictor) generateDriftPredictionInsights(predictions []models.DriftPredictionPoint, historical []float64) []string {
	var insights []string

	if len(predictions) == 0 {
		return insights
	}

	// Calculate trend
	lastHistorical := historical[len(historical)-1]
	lastPrediction := predictions[len(predictions)-1].Predicted

	if lastPrediction > lastHistorical {
		insights = append(insights, "Drift rate predicted to increase, consider preventive measures")
	} else if lastPrediction < lastHistorical {
		insights = append(insights, "Drift rate predicted to decrease, current measures are effective")
	} else {
		insights = append(insights, "Drift rate predicted to remain stable")
	}

	// Check for high drift predictions
	if lastPrediction > 10 {
		insights = append(insights, "High drift rate predicted, immediate attention required")
	}

	return insights
}

// generatePerformancePredictionInsights generates insights for performance predictions
func (p *Predictor) generatePerformancePredictionInsights(predictions []models.PerformancePredictionPoint, historical []float64) []string {
	var insights []string

	if len(predictions) == 0 {
		return insights
	}

	// Calculate trend
	lastHistorical := historical[len(historical)-1]
	lastPrediction := predictions[len(predictions)-1].Predicted

	if lastPrediction > lastHistorical {
		insights = append(insights, "Performance predicted to improve over the prediction period")
	} else if lastPrediction < lastHistorical {
		insights = append(insights, "Performance predicted to degrade, consider optimization")
	} else {
		insights = append(insights, "Performance predicted to remain stable")
	}

	// Check for performance thresholds
	if lastPrediction > 80 {
		insights = append(insights, "High performance predicted, monitor for potential bottlenecks")
	} else if lastPrediction < 20 {
		insights = append(insights, "Low performance predicted, consider scaling or optimization")
	}

	return insights
}
