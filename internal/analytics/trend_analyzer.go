package analytics

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// TrendAnalyzer analyzes trends in data
type TrendAnalyzer struct {
	mu     sync.RWMutex
	config *TrendConfig
}

// TrendConfig represents configuration for trend analysis
type TrendConfig struct {
	MinDataPoints   int           `json:"min_data_points"`
	WindowSize      int           `json:"window_size"`
	Sensitivity     float64       `json:"sensitivity"`
	RetentionPeriod time.Duration `json:"retention_period"`
}

// TrendAnalysis represents the result of trend analysis
type TrendAnalysis struct {
	ID          string                 `json:"id"`
	Trend       string                 `json:"trend"`      // increasing, decreasing, stable, volatile
	Strength    float64                `json:"strength"`   // 0-1 trend strength
	Direction   float64                `json:"direction"`  // slope of trend
	Volatility  float64                `json:"volatility"` // measure of volatility
	Seasonality bool                   `json:"seasonality"`
	Period      time.Duration          `json:"period,omitempty"`
	Confidence  float64                `json:"confidence"` // 0-1 confidence score
	GeneratedAt time.Time              `json:"generated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TrendPoint represents a point in a trend
type TrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Trend     string    `json:"trend"`
	Strength  float64   `json:"strength"`
}

// NewTrendAnalyzer creates a new trend analyzer
func NewTrendAnalyzer() *TrendAnalyzer {
	config := &TrendConfig{
		MinDataPoints:   10,
		WindowSize:      30,
		Sensitivity:     0.1,
		RetentionPeriod: 30 * 24 * time.Hour,
	}

	return &TrendAnalyzer{
		config: config,
	}
}

// AnalyzeTrends analyzes trends in data
func (ta *TrendAnalyzer) AnalyzeTrends(ctx context.Context, data []DataPoint) (*TrendAnalysis, error) {
	if len(data) < ta.config.MinDataPoints {
		return nil, fmt.Errorf("insufficient data points: need at least %d, got %d", ta.config.MinDataPoints, len(data))
	}

	analysis := &TrendAnalysis{
		ID:          fmt.Sprintf("trend_%d", time.Now().Unix()),
		GeneratedAt: time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Calculate basic trend metrics
	trend, strength, direction := ta.calculateTrend(data)
	volatility := ta.calculateVolatility(data)
	seasonality, period := ta.detectSeasonality(data)

	analysis.Trend = trend
	analysis.Strength = strength
	analysis.Direction = direction
	analysis.Volatility = volatility
	analysis.Seasonality = seasonality
	analysis.Period = period
	analysis.Confidence = ta.calculateConfidence(data, trend, strength)

	return analysis, nil
}

// CalculateTrendPoints calculates trend points for visualization
func (ta *TrendAnalyzer) CalculateTrendPoints(ctx context.Context, data []DataPoint, windowSize int) ([]TrendPoint, error) {
	if len(data) < windowSize {
		return nil, fmt.Errorf("insufficient data for window size %d", windowSize)
	}

	var trendPoints []TrendPoint

	for i := windowSize; i <= len(data); i++ {
		window := data[i-windowSize : i]
		trend, strength, _ := ta.calculateTrend(window)

		point := TrendPoint{
			Timestamp: window[len(window)-1].Timestamp,
			Value:     window[len(window)-1].Value,
			Trend:     trend,
			Strength:  strength,
		}
		trendPoints = append(trendPoints, point)
	}

	return trendPoints, nil
}

// DetectTrendChanges detects significant trend changes
func (ta *TrendAnalyzer) DetectTrendChanges(ctx context.Context, data []DataPoint) ([]TrendChange, error) {
	if len(data) < ta.config.WindowSize*2 {
		return nil, fmt.Errorf("insufficient data for trend change detection")
	}

	var changes []TrendChange
	windowSize := ta.config.WindowSize

	for i := windowSize; i < len(data)-windowSize; i++ {
		window1 := data[i-windowSize : i]
		window2 := data[i : i+windowSize]

		trend1, strength1, _ := ta.calculateTrend(window1)
		trend2, strength2, _ := ta.calculateTrend(window2)

		// Check if trend has changed significantly
		if trend1 != trend2 && (strength1 > ta.config.Sensitivity || strength2 > ta.config.Sensitivity) {
			change := TrendChange{
				ID:           fmt.Sprintf("change_%d", time.Now().UnixNano()),
				Timestamp:    data[i].Timestamp,
				FromTrend:    trend1,
				ToTrend:      trend2,
				FromStrength: strength1,
				ToStrength:   strength2,
				Significance: math.Abs(strength1 - strength2),
				Metadata:     make(map[string]interface{}),
			}
			changes = append(changes, change)
		}
	}

	return changes, nil
}

// TrendChange represents a significant trend change
type TrendChange struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	FromTrend    string                 `json:"from_trend"`
	ToTrend      string                 `json:"to_trend"`
	FromStrength float64                `json:"from_strength"`
	ToStrength   float64                `json:"to_strength"`
	Significance float64                `json:"significance"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Helper methods

// calculateTrend calculates the trend of data
func (ta *TrendAnalyzer) calculateTrend(data []DataPoint) (string, float64, float64) {
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

	// Calculate slope (direction)
	slope := (float64(n)*sumXY - sumX*sumY) / (float64(n)*sumXX - sumX*sumX)

	// Calculate R-squared (strength)
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

	// Determine trend type
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

// calculateVolatility calculates the volatility of data
func (ta *TrendAnalyzer) calculateVolatility(data []DataPoint) float64 {
	if len(data) < 2 {
		return 0
	}

	// Calculate standard deviation
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

	return math.Sqrt(variance)
}

// detectSeasonality detects seasonal patterns in data
func (ta *TrendAnalyzer) detectSeasonality(data []DataPoint) (bool, time.Duration) {
	if len(data) < 24 { // Need at least 24 data points
		return false, 0
	}

	// Simple seasonality detection based on autocorrelation
	// In a real system, you would use more sophisticated methods

	// Check for daily seasonality (24-hour pattern)
	if ta.hasDailySeasonality(data) {
		return true, 24 * time.Hour
	}

	// Check for weekly seasonality (7-day pattern)
	if ta.hasWeeklySeasonality(data) {
		return true, 7 * 24 * time.Hour
	}

	return false, 0
}

// hasDailySeasonality checks for daily seasonality
func (ta *TrendAnalyzer) hasDailySeasonality(data []DataPoint) bool {
	// Simplified check - in reality, you'd use autocorrelation
	// This is a placeholder implementation
	return false
}

// hasWeeklySeasonality checks for weekly seasonality
func (ta *TrendAnalyzer) hasWeeklySeasonality(data []DataPoint) bool {
	// Simplified check - in reality, you'd use autocorrelation
	// This is a placeholder implementation
	return false
}

// calculateConfidence calculates confidence in the trend analysis
func (ta *TrendAnalyzer) calculateConfidence(data []DataPoint, trend string, strength float64) float64 {
	// Base confidence on data quality and trend strength
	confidence := strength

	// Adjust for data quality
	if len(data) < 20 {
		confidence *= 0.8
	} else if len(data) < 50 {
		confidence *= 0.9
	}

	// Adjust for trend consistency
	if trend == "stable" {
		confidence *= 0.7 // Lower confidence for stable trends
	}

	return math.Min(confidence, 1.0)
}

// SetConfig updates the trend analyzer configuration
func (ta *TrendAnalyzer) SetConfig(config *TrendConfig) {
	ta.mu.Lock()
	defer ta.mu.Unlock()
	ta.config = config
}

// GetConfig returns the current trend analyzer configuration
func (ta *TrendAnalyzer) GetConfig() *TrendConfig {
	ta.mu.RLock()
	defer ta.mu.RUnlock()
	return ta.config
}
