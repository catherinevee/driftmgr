package analytics

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// AnomalyDetector detects anomalies in data
type AnomalyDetector struct {
	mu     sync.RWMutex
	config *AnomalyConfig
}

// AnomalyConfig represents configuration for anomaly detection
type AnomalyConfig struct {
	Sensitivity     float64       `json:"sensitivity"` // 0-1, lower = more sensitive
	WindowSize      int           `json:"window_size"`
	MinDataPoints   int           `json:"min_data_points"`
	RetentionPeriod time.Duration `json:"retention_period"`
	AutoThreshold   bool          `json:"auto_threshold"`
	Threshold       float64       `json:"threshold"`
}

// AnomalyReport represents the result of anomaly detection
type AnomalyReport struct {
	ID          string                 `json:"id"`
	Anomalies   []Anomaly              `json:"anomalies"`
	TotalCount  int                    `json:"total_count"`
	Severity    string                 `json:"severity"` // low, medium, high, critical
	GeneratedAt time.Time              `json:"generated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Anomaly represents a detected anomaly
type Anomaly struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	Value      float64                `json:"value"`
	Expected   float64                `json:"expected"`
	Deviation  float64                `json:"deviation"`
	Severity   string                 `json:"severity"`   // low, medium, high, critical
	Type       string                 `json:"type"`       // statistical, trend, seasonal, etc.
	Confidence float64                `json:"confidence"` // 0-1 confidence score
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector() *AnomalyDetector {
	config := &AnomalyConfig{
		Sensitivity:     0.1,
		WindowSize:      30,
		MinDataPoints:   10,
		RetentionPeriod: 30 * 24 * time.Hour,
		AutoThreshold:   true,
		Threshold:       3.0, // 3 standard deviations
	}

	return &AnomalyDetector{
		config: config,
	}
}

// DetectAnomalies detects anomalies in data
func (ad *AnomalyDetector) DetectAnomalies(ctx context.Context, data []DataPoint) (*AnomalyReport, error) {
	if len(data) < ad.config.MinDataPoints {
		return nil, fmt.Errorf("insufficient data points: need at least %d, got %d", ad.config.MinDataPoints, len(data))
	}

	report := &AnomalyReport{
		ID:          fmt.Sprintf("anomaly_report_%d", time.Now().Unix()),
		Anomalies:   []Anomaly{},
		GeneratedAt: time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Detect different types of anomalies
	statisticalAnomalies := ad.detectStatisticalAnomalies(data)
	trendAnomalies := ad.detectTrendAnomalies(data)
	seasonalAnomalies := ad.detectSeasonalAnomalies(data)

	// Combine all anomalies
	report.Anomalies = append(report.Anomalies, statisticalAnomalies...)
	report.Anomalies = append(report.Anomalies, trendAnomalies...)
	report.Anomalies = append(report.Anomalies, seasonalAnomalies...)

	report.TotalCount = len(report.Anomalies)
	report.Severity = ad.calculateOverallSeverity(report.Anomalies)

	return report, nil
}

// DetectStatisticalAnomalies detects statistical anomalies using Z-score
func (ad *AnomalyDetector) detectStatisticalAnomalies(data []DataPoint) []Anomaly {
	var anomalies []Anomaly

	// Calculate mean and standard deviation
	mean, stdDev := ad.calculateMeanAndStdDev(data)

	if stdDev == 0 {
		return anomalies // No variation, no anomalies
	}

	threshold := ad.config.Threshold
	if ad.config.AutoThreshold {
		threshold = 3.0 - (ad.config.Sensitivity * 2.0) // Adjust threshold based on sensitivity
	}

	for _, point := range data {
		zScore := math.Abs(point.Value-mean) / stdDev

		if zScore > threshold {
			severity := ad.calculateSeverity(zScore, threshold)
			confidence := math.Min(zScore/threshold, 1.0)

			anomaly := Anomaly{
				ID:         fmt.Sprintf("anomaly_%d", time.Now().UnixNano()),
				Timestamp:  point.Timestamp,
				Value:      point.Value,
				Expected:   mean,
				Deviation:  zScore,
				Severity:   severity,
				Type:       "statistical",
				Confidence: confidence,
				Metadata:   make(map[string]interface{}),
			}
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

// DetectTrendAnomalies detects anomalies in trends
func (ad *AnomalyDetector) detectTrendAnomalies(data []DataPoint) []Anomaly {
	var anomalies []Anomaly

	if len(data) < ad.config.WindowSize*2 {
		return anomalies
	}

	// Calculate rolling trend
	for i := ad.config.WindowSize; i < len(data); i++ {
		window := data[i-ad.config.WindowSize : i]
		trend, _, _ := ad.calculateTrend(window)

		// Check if current point deviates significantly from trend
		expected := ad.predictFromTrend(window, trend)
		actual := data[i].Value

		deviation := math.Abs(actual - expected)
		threshold := ad.calculateTrendThreshold(window)

		if deviation > threshold {
			severity := ad.calculateSeverity(deviation/threshold, 1.0)
			confidence := math.Min(deviation/threshold, 1.0)

			anomaly := Anomaly{
				ID:         fmt.Sprintf("anomaly_%d", time.Now().UnixNano()),
				Timestamp:  data[i].Timestamp,
				Value:      actual,
				Expected:   expected,
				Deviation:  deviation,
				Severity:   severity,
				Type:       "trend",
				Confidence: confidence,
				Metadata:   make(map[string]interface{}),
			}
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

// DetectSeasonalAnomalies detects seasonal anomalies
func (ad *AnomalyDetector) detectSeasonalAnomalies(data []DataPoint) []Anomaly {
	var anomalies []Anomaly

	// This is a simplified implementation
	// In a real system, you would use more sophisticated seasonal analysis

	// Check for daily seasonality
	dailyAnomalies := ad.detectDailySeasonalAnomalies(data)
	anomalies = append(anomalies, dailyAnomalies...)

	// Check for weekly seasonality
	weeklyAnomalies := ad.detectWeeklySeasonalAnomalies(data)
	anomalies = append(anomalies, weeklyAnomalies...)

	return anomalies
}

// Helper methods

// calculateMeanAndStdDev calculates mean and standard deviation
func (ad *AnomalyDetector) calculateMeanAndStdDev(data []DataPoint) (float64, float64) {
	if len(data) == 0 {
		return 0, 0
	}

	var sum float64
	for _, point := range data {
		sum += point.Value
	}
	mean := sum / float64(len(data))

	var variance float64
	for _, point := range data {
		variance += math.Pow(point.Value-mean, 2)
	}
	variance /= float64(len(data))

	return mean, math.Sqrt(variance)
}

// calculateTrend calculates the trend of data
func (ad *AnomalyDetector) calculateTrend(data []DataPoint) (string, float64, float64) {
	if len(data) < 2 {
		return "stable", 0, 0
	}

	// Calculate linear regression
	n := len(data)
	var sumX, sumY, sumXY, sumXX float64

	for i, point := range data {
		x := float64(i)
		y := point.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	slope := (float64(n)*sumXY - sumX*sumY) / (float64(n)*sumXX - sumX*sumX)

	var ssRes, ssTot float64
	meanY := sumY / float64(n)

	for i, point := range data {
		x := float64(i)
		predicted := slope*x + (sumY-slope*sumX)/float64(n)
		ssRes += math.Pow(point.Value-predicted, 2)
		ssTot += math.Pow(point.Value-meanY, 2)
	}

	rSquared := 1 - (ssRes / ssTot)
	if rSquared < 0 {
		rSquared = 0
	}

	var trend string
	if math.Abs(slope) < 0.01 {
		trend = "stable"
	} else if slope > 0 {
		trend = "increasing"
	} else {
		trend = "decreasing"
	}

	return trend, rSquared, slope
}

// predictFromTrend predicts value from trend
func (ad *AnomalyDetector) predictFromTrend(data []DataPoint, trend string) float64 {
	if len(data) == 0 {
		return 0
	}

	// Simple prediction based on trend
	lastValue := data[len(data)-1].Value

	switch trend {
	case "increasing":
		// Predict slight increase
		return lastValue * 1.01
	case "decreasing":
		// Predict slight decrease
		return lastValue * 0.99
	default:
		// Predict stable
		return lastValue
	}
}

// calculateTrendThreshold calculates threshold for trend anomalies
func (ad *AnomalyDetector) calculateTrendThreshold(data []DataPoint) float64 {
	_, stdDev := ad.calculateMeanAndStdDev(data)
	return stdDev * (1.0 + ad.config.Sensitivity)
}

// calculateSeverity calculates severity based on deviation
func (ad *AnomalyDetector) calculateSeverity(deviation, threshold float64) string {
	ratio := deviation / threshold

	if ratio >= 3.0 {
		return "critical"
	} else if ratio >= 2.0 {
		return "high"
	} else if ratio >= 1.5 {
		return "medium"
	} else {
		return "low"
	}
}

// calculateOverallSeverity calculates overall severity of anomalies
func (ad *AnomalyDetector) calculateOverallSeverity(anomalies []Anomaly) string {
	if len(anomalies) == 0 {
		return "low"
	}

	criticalCount := 0
	highCount := 0
	mediumCount := 0

	for _, anomaly := range anomalies {
		switch anomaly.Severity {
		case "critical":
			criticalCount++
		case "high":
			highCount++
		case "medium":
			mediumCount++
		}
	}

	if criticalCount > 0 {
		return "critical"
	} else if highCount > 0 {
		return "high"
	} else if mediumCount > 0 {
		return "medium"
	} else {
		return "low"
	}
}

// detectDailySeasonalAnomalies detects daily seasonal anomalies
func (ad *AnomalyDetector) detectDailySeasonalAnomalies(data []DataPoint) []Anomaly {
	// Simplified implementation
	// In reality, you would analyze hourly patterns
	return []Anomaly{}
}

// detectWeeklySeasonalAnomalies detects weekly seasonal anomalies
func (ad *AnomalyDetector) detectWeeklySeasonalAnomalies(data []DataPoint) []Anomaly {
	// Simplified implementation
	// In reality, you would analyze daily patterns
	return []Anomaly{}
}

// SetConfig updates the anomaly detector configuration
func (ad *AnomalyDetector) SetConfig(config *AnomalyConfig) {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	ad.config = config
}

// GetConfig returns the current anomaly detector configuration
func (ad *AnomalyDetector) GetConfig() *AnomalyConfig {
	ad.mu.RLock()
	defer ad.mu.RUnlock()
	return ad.config
}
