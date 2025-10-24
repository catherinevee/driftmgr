package models

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// ChartType represents the type of a chart
type ChartType string

const (
	ChartTypeLine    ChartType = "line"
	ChartTypeBar     ChartType = "bar"
	ChartTypePie     ChartType = "pie"
	ChartTypeArea    ChartType = "area"
	ChartTypeScatter ChartType = "scatter"
	ChartTypeHeatmap ChartType = "heatmap"
	ChartTypeGauge   ChartType = "gauge"
)

// String returns the string representation of ChartType
func (ct ChartType) String() string {
	return string(ct)
}

// Chart represents a chart
type Chart struct {
	ID            string                   `json:"id" db:"id" validate:"required,uuid"`
	Type          ChartType                `json:"type" db:"type" validate:"required"`
	Title         string                   `json:"title" db:"title" validate:"required"`
	Description   string                   `json:"description" db:"description"`
	Data          []map[string]interface{} `json:"data" db:"data"`
	Configuration map[string]interface{}   `json:"configuration" db:"configuration"`
	CreatedAt     time.Time                `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time                `json:"updated_at" db:"updated_at"`
}

// CostPrediction represents a cost prediction
type CostPrediction struct {
	ID                 string                `json:"id" db:"id" validate:"required,uuid"`
	PredictionPeriod   time.Duration         `json:"prediction_period" db:"prediction_period"`
	Predictions        []CostPredictionPoint `json:"predictions" db:"predictions"`
	ConfidenceInterval ConfidenceInterval    `json:"confidence_interval" db:"confidence_interval"`
	Insights           []string              `json:"insights" db:"insights"`
	GeneratedAt        time.Time             `json:"generated_at" db:"generated_at"`
}

// CostPredictionPoint represents a single cost prediction point
type CostPredictionPoint struct {
	Date      time.Time `json:"date" db:"date"`
	Predicted float64   `json:"predicted" db:"predicted"`
	Lower     float64   `json:"lower" db:"lower"`
	Upper     float64   `json:"upper" db:"upper"`
}

// ResourceUsagePrediction represents a resource usage prediction
type ResourceUsagePrediction struct {
	ID                 string                 `json:"id" db:"id" validate:"required,uuid"`
	PredictionPeriod   time.Duration          `json:"prediction_period" db:"prediction_period"`
	Predictions        []UsagePredictionPoint `json:"predictions" db:"predictions"`
	ConfidenceInterval ConfidenceInterval     `json:"confidence_interval" db:"confidence_interval"`
	Insights           []string               `json:"insights" db:"insights"`
	GeneratedAt        time.Time              `json:"generated_at" db:"generated_at"`
}

// UsagePredictionPoint represents a single usage prediction point
type UsagePredictionPoint struct {
	Date      time.Time `json:"date" db:"date"`
	Predicted float64   `json:"predicted" db:"predicted"`
	Lower     float64   `json:"lower" db:"lower"`
	Upper     float64   `json:"upper" db:"upper"`
}

// DriftPrediction represents a drift prediction
type DriftPrediction struct {
	ID                 string                 `json:"id" db:"id" validate:"required,uuid"`
	PredictionPeriod   time.Duration          `json:"prediction_period" db:"prediction_period"`
	Predictions        []DriftPredictionPoint `json:"predictions" db:"predictions"`
	ConfidenceInterval ConfidenceInterval     `json:"confidence_interval" db:"confidence_interval"`
	Insights           []string               `json:"insights" db:"insights"`
	GeneratedAt        time.Time              `json:"generated_at" db:"generated_at"`
}

// DriftPredictionPoint represents a single drift prediction point
type DriftPredictionPoint struct {
	Date      time.Time `json:"date" db:"date"`
	Predicted float64   `json:"predicted" db:"predicted"`
	Lower     float64   `json:"lower" db:"lower"`
	Upper     float64   `json:"upper" db:"upper"`
}

// PerformancePrediction represents a performance prediction
type PerformancePrediction struct {
	ID                 string                       `json:"id" db:"id" validate:"required,uuid"`
	PredictionPeriod   time.Duration                `json:"prediction_period" db:"prediction_period"`
	Predictions        []PerformancePredictionPoint `json:"predictions" db:"predictions"`
	ConfidenceInterval ConfidenceInterval           `json:"confidence_interval" db:"confidence_interval"`
	Insights           []string                     `json:"insights" db:"insights"`
	GeneratedAt        time.Time                    `json:"generated_at" db:"generated_at"`
}

// PerformancePredictionPoint represents a single performance prediction point
type PerformancePredictionPoint struct {
	Date      time.Time `json:"date" db:"date"`
	Predicted float64   `json:"predicted" db:"predicted"`
	Lower     float64   `json:"lower" db:"lower"`
	Upper     float64   `json:"upper" db:"upper"`
}

// ConfidenceInterval represents a confidence interval
type ConfidenceInterval struct {
	Level float64 `json:"level" db:"level"`
	Lower float64 `json:"lower" db:"lower"`
	Upper float64 `json:"upper" db:"upper"`
}

// Validation methods

// Validate validates the Chart struct
func (c *Chart) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

// Validate validates the CostPrediction struct
func (cp *CostPrediction) Validate() error {
	validate := validator.New()
	return validate.Struct(cp)
}

// Validate validates the ResourceUsagePrediction struct
func (rup *ResourceUsagePrediction) Validate() error {
	validate := validator.New()
	return validate.Struct(rup)
}

// Validate validates the DriftPrediction struct
func (dp *DriftPrediction) Validate() error {
	validate := validator.New()
	return validate.Struct(dp)
}

// Validate validates the PerformancePrediction struct
func (pp *PerformancePrediction) Validate() error {
	validate := validator.New()
	return validate.Struct(pp)
}
