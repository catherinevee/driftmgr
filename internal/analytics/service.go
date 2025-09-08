package analytics

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AnalyticsService provides a unified interface for analytics operations
type AnalyticsService struct {
	predictiveEngine *PredictiveEngine
	trendAnalyzer    *TrendAnalyzer
	anomalyDetector  *AnomalyDetector
	mu               sync.RWMutex
	config           *AnalyticsConfig
}

// AnalyticsConfig represents configuration for the analytics service
type AnalyticsConfig struct {
	PredictiveEnabled       bool          `json:"predictive_enabled"`
	TrendAnalysisEnabled    bool          `json:"trend_analysis_enabled"`
	AnomalyDetectionEnabled bool          `json:"anomaly_detection_enabled"`
	DataRetention           time.Duration `json:"data_retention"`
	AnalysisInterval        time.Duration `json:"analysis_interval"`
	AutoAnalysis            bool          `json:"auto_analysis"`
	NotificationEnabled     bool          `json:"notification_enabled"`
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService() *AnalyticsService {
	config := &AnalyticsConfig{
		PredictiveEnabled:       true,
		TrendAnalysisEnabled:    true,
		AnomalyDetectionEnabled: true,
		DataRetention:           365 * 24 * time.Hour,
		AnalysisInterval:        1 * time.Hour,
		AutoAnalysis:            true,
		NotificationEnabled:     true,
	}

	// Create components
	predictiveEngine := NewPredictiveEngine()
	trendAnalyzer := NewTrendAnalyzer()
	anomalyDetector := NewAnomalyDetector()

	return &AnalyticsService{
		predictiveEngine: predictiveEngine,
		trendAnalyzer:    trendAnalyzer,
		anomalyDetector:  anomalyDetector,
		config:           config,
	}
}

// GetPredictiveEngine returns the predictive engine
func (s *AnalyticsService) GetPredictiveEngine() *PredictiveEngine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.predictiveEngine
}

// GetTrendAnalyzer returns the trend analyzer
func (s *AnalyticsService) GetTrendAnalyzer() *TrendAnalyzer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.trendAnalyzer
}

// GetAnomalyDetector returns the anomaly detector
func (s *AnalyticsService) GetAnomalyDetector() *AnomalyDetector {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.anomalyDetector
}

// Start starts the analytics service
func (as *AnalyticsService) Start(ctx context.Context) error {
	// Create default models and forecasters
	if err := as.createDefaultModels(ctx); err != nil {
		return fmt.Errorf("failed to create default models: %w", err)
	}

	// Start background analysis if enabled
	if as.config.AutoAnalysis {
		go as.backgroundAnalysis(ctx)
	}

	return nil
}

// Stop stops the analytics service
func (as *AnalyticsService) Stop(ctx context.Context) error {
	// Stop background processes
	return nil
}

// CreateModel creates a new predictive model
func (as *AnalyticsService) CreateModel(ctx context.Context, model *PredictiveModel) error {
	return as.predictiveEngine.CreateModel(ctx, model)
}

// TrainModel trains a predictive model
func (as *AnalyticsService) TrainModel(ctx context.Context, modelID string, data []DataPoint) error {
	return as.predictiveEngine.TrainModel(ctx, modelID, data)
}

// CreateForecaster creates a new forecaster
func (as *AnalyticsService) CreateForecaster(ctx context.Context, forecaster *Forecaster) error {
	return as.predictiveEngine.CreateForecaster(ctx, forecaster)
}

// GenerateForecast generates a forecast
func (as *AnalyticsService) GenerateForecast(ctx context.Context, forecasterID string, data []DataPoint) (*Forecast, error) {
	return as.predictiveEngine.GenerateForecast(ctx, forecasterID, data)
}

// AnalyzeTrends analyzes trends in data
func (as *AnalyticsService) AnalyzeTrends(ctx context.Context, data []DataPoint) (*TrendAnalysis, error) {
	return as.trendAnalyzer.AnalyzeTrends(ctx, data)
}

// DetectAnomalies detects anomalies in data
func (as *AnalyticsService) DetectAnomalies(ctx context.Context, data []DataPoint) (*AnomalyReport, error) {
	return as.anomalyDetector.DetectAnomalies(ctx, data)
}

// GetAnalyticsStatus returns the overall analytics status
func (as *AnalyticsService) GetAnalyticsStatus(ctx context.Context) (*AnalyticsStatus, error) {
	status := &AnalyticsStatus{
		OverallStatus: "Unknown",
		Models:        make(map[string]int),
		Forecasters:   make(map[string]int),
		LastAnalysis:  time.Time{},
		Metadata:      make(map[string]interface{}),
	}

	// Get model counts
	models, err := as.predictiveEngine.ListModels(ctx)
	if err == nil {
		for _, model := range models {
			status.Models[model.Category]++
		}
	}

	// Get forecaster counts
	forecasters, err := as.predictiveEngine.ListForecasters(ctx)
	if err == nil {
		for _, forecaster := range forecasters {
			status.Forecasters[forecaster.Target]++
		}
	}

	// Determine overall status
	totalComponents := len(models) + len(forecasters)
	if totalComponents > 0 {
		status.OverallStatus = "Active"
	} else {
		status.OverallStatus = "Inactive"
	}

	return status, nil
}

// AnalyticsStatus represents the overall analytics status
type AnalyticsStatus struct {
	OverallStatus string                 `json:"overall_status"`
	Models        map[string]int         `json:"models"`
	Forecasters   map[string]int         `json:"forecasters"`
	LastAnalysis  time.Time              `json:"last_analysis"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Helper methods

// createDefaultModels creates default predictive models and forecasters
func (as *AnalyticsService) createDefaultModels(ctx context.Context) error {
	// Create default models
	models := []*PredictiveModel{
		{
			Name:     "Cost Trend Model",
			Type:     "linear",
			Category: "cost",
			Parameters: map[string]interface{}{
				"window_size": 30,
				"seasonality": false,
			},
			Status: "inactive",
		},
		{
			Name:     "Performance Model",
			Type:     "exponential",
			Category: "performance",
			Parameters: map[string]interface{}{
				"alpha": 0.3,
				"beta":  0.1,
			},
			Status: "inactive",
		},
		{
			Name:     "Capacity Model",
			Type:     "arima",
			Category: "capacity",
			Parameters: map[string]interface{}{
				"p": 2,
				"d": 1,
				"q": 1,
			},
			Status: "inactive",
		},
		{
			Name:     "Security Risk Model",
			Type:     "lstm",
			Category: "security",
			Parameters: map[string]interface{}{
				"layers":  3,
				"neurons": 64,
				"dropout": 0.2,
			},
			Status: "inactive",
		},
	}

	// Create models
	for _, model := range models {
		if err := as.predictiveEngine.CreateModel(ctx, model); err != nil {
			return fmt.Errorf("failed to create model %s: %w", model.Name, err)
		}
	}

	// Create default forecasters
	forecasters := []*Forecaster{
		{
			Name:      "Cost Forecaster",
			ModelID:   "model_1", // Will be updated with actual model ID
			Target:    "cost",
			Horizon:   30 * 24 * time.Hour, // 30 days
			Frequency: 24 * time.Hour,      // Daily
			Enabled:   true,
		},
		{
			Name:      "Performance Forecaster",
			ModelID:   "model_2", // Will be updated with actual model ID
			Target:    "performance",
			Horizon:   7 * 24 * time.Hour, // 7 days
			Frequency: 6 * time.Hour,      // Every 6 hours
			Enabled:   true,
		},
		{
			Name:      "Capacity Forecaster",
			ModelID:   "model_3", // Will be updated with actual model ID
			Target:    "capacity",
			Horizon:   14 * 24 * time.Hour, // 14 days
			Frequency: 12 * time.Hour,      // Every 12 hours
			Enabled:   true,
		},
		{
			Name:      "Security Risk Forecaster",
			ModelID:   "model_4", // Will be updated with actual model ID
			Target:    "security_risk",
			Horizon:   3 * 24 * time.Hour, // 3 days
			Frequency: 1 * time.Hour,      // Hourly
			Enabled:   true,
		},
	}

	// Create forecasters
	for _, forecaster := range forecasters {
		if err := as.predictiveEngine.CreateForecaster(ctx, forecaster); err != nil {
			return fmt.Errorf("failed to create forecaster %s: %w", forecaster.Name, err)
		}
	}

	return nil
}

// backgroundAnalysis runs background analysis
func (as *AnalyticsService) backgroundAnalysis(ctx context.Context) {
	ticker := time.NewTicker(as.config.AnalysisInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			as.performBackgroundAnalysis(ctx)
		}
	}
}

// performBackgroundAnalysis performs background analysis
func (as *AnalyticsService) performBackgroundAnalysis(ctx context.Context) {
	// This would perform regular analysis tasks
	// For now, it's a placeholder
	fmt.Println("Performing background analysis...")
}

// SetConfig updates the analytics service configuration
func (as *AnalyticsService) SetConfig(config *AnalyticsConfig) {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.config = config
}

// GetConfig returns the current analytics service configuration
func (as *AnalyticsService) GetConfig() *AnalyticsConfig {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.config
}
