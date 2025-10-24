package models

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// AlertCondition represents a condition for an alert
type AlertCondition struct {
	ID          string                 `json:"id" db:"id" validate:"required,uuid"`
	Name        string                 `json:"name" db:"name" validate:"required"`
	Type        AlertConditionType     `json:"type" db:"type" validate:"required"`
	Expression  string                 `json:"expression" db:"expression" validate:"required"`
	Parameters  map[string]interface{} `json:"parameters" db:"parameters"`
	Threshold   float64                `json:"threshold" db:"threshold"`
	Operator    ComparisonOperator     `json:"operator" db:"operator" validate:"required"`
	Duration    time.Duration          `json:"duration" db:"duration"`
	Aggregation AggregationFunction    `json:"aggregation" db:"aggregation"`
	GroupBy     []string               `json:"group_by" db:"group_by"`
	Filters     []AnalyticsFilter      `json:"filters" db:"filters"`
	Description string                 `json:"description" db:"description"`
}

// AlertConditionType represents the type of an alert condition
type AlertConditionType string

const (
	AlertConditionTypeMetric     AlertConditionType = "metric"
	AlertConditionTypeExpression AlertConditionType = "expression"
	AlertConditionTypeResource   AlertConditionType = "resource"
	AlertConditionTypeCompliance AlertConditionType = "compliance"
	AlertConditionTypeCost       AlertConditionType = "cost"
	AlertConditionTypeSecurity   AlertConditionType = "security"
	AlertConditionTypeCustom     AlertConditionType = "custom"
)

// String returns the string representation of AlertConditionType
func (act AlertConditionType) String() string {
	return string(act)
}

// ComparisonOperator represents a comparison operator
type ComparisonOperator string

const (
	ComparisonOperatorEquals       ComparisonOperator = "equals"
	ComparisonOperatorNotEquals    ComparisonOperator = "not_equals"
	ComparisonOperatorGreaterThan  ComparisonOperator = "greater_than"
	ComparisonOperatorLessThan     ComparisonOperator = "less_than"
	ComparisonOperatorGreaterEqual ComparisonOperator = "greater_equal"
	ComparisonOperatorLessEqual    ComparisonOperator = "less_equal"
	ComparisonOperatorContains     ComparisonOperator = "contains"
	ComparisonOperatorNotContains  ComparisonOperator = "not_contains"
	ComparisonOperatorRegex        ComparisonOperator = "regex"
	ComparisonOperatorNotRegex     ComparisonOperator = "not_regex"
)

// String returns the string representation of ComparisonOperator
func (co ComparisonOperator) String() string {
	return string(co)
}

// AlertRule represents an alert rule
type AlertRule struct {
	ID                 string                  `json:"id" db:"id" validate:"required,uuid"`
	Name               string                  `json:"name" db:"name" validate:"required"`
	Description        string                  `json:"description" db:"description"`
	Condition          AlertCondition          `json:"condition" db:"condition" validate:"required"`
	Severity           AlertSeverity           `json:"severity" db:"severity" validate:"required"`
	Channels           []NotificationChannel   `json:"channels" db:"channels" validate:"required"`
	Tags               map[string]string       `json:"tags" db:"tags"`
	IsActive           bool                    `json:"is_active" db:"is_active"`
	IsEnabled          bool                    `json:"is_enabled" db:"is_enabled"`
	EvaluationInterval time.Duration           `json:"evaluation_interval" db:"evaluation_interval"`
	SuppressionConfig  *AlertSuppressionConfig `json:"suppression_config" db:"suppression_config"`
	EscalationConfig   *AlertEscalationConfig  `json:"escalation_config" db:"escalation_config"`
	RunbookURL         string                  `json:"runbook_url" db:"runbook_url"`
	LastEvaluated      *time.Time              `json:"last_evaluated" db:"last_evaluated"`
	LastTriggered      *time.Time              `json:"last_triggered" db:"last_triggered"`
	TriggerCount       int                     `json:"trigger_count" db:"trigger_count"`
	CreatedBy          string                  `json:"created_by" db:"created_by" validate:"required"`
	CreatedAt          time.Time               `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time               `json:"updated_at" db:"updated_at"`
}

// AlertSuppressionConfig represents alert suppression configuration
type AlertSuppressionConfig struct {
	Enabled        bool             `json:"enabled" db:"enabled"`
	Duration       time.Duration    `json:"duration" db:"duration"`
	MaxAlerts      int              `json:"max_alerts" db:"max_alerts"`
	SuppressionKey string           `json:"suppression_key" db:"suppression_key"`
	Conditions     []AlertCondition `json:"conditions" db:"conditions"`
}

// AlertEscalationConfig represents alert escalation configuration
type AlertEscalationConfig struct {
	Enabled          bool              `json:"enabled" db:"enabled"`
	EscalationDelay  time.Duration     `json:"escalation_delay" db:"escalation_delay"`
	EscalationLevels []EscalationLevel `json:"escalation_levels" db:"escalation_levels"`
	MaxEscalations   int               `json:"max_escalations" db:"max_escalations"`
}

// EscalationLevel represents an escalation level
type EscalationLevel struct {
	Level      int                   `json:"level" db:"level"`
	Delay      time.Duration         `json:"delay" db:"delay"`
	Channels   []NotificationChannel `json:"channels" db:"channels"`
	Recipients []string              `json:"recipients" db:"recipients"`
	Message    string                `json:"message" db:"message"`
	Priority   NotificationPriority  `json:"priority" db:"priority"`
}

// Alert represents an alert instance
type Alert struct {
	ID              string            `json:"id" db:"id" validate:"required,uuid"`
	RuleID          string            `json:"rule_id" db:"rule_id" validate:"required,uuid"`
	Status          AlertStatus       `json:"status" db:"status" validate:"required"`
	Severity        AlertSeverity     `json:"severity" db:"severity" validate:"required"`
	Title           string            `json:"title" db:"title" validate:"required"`
	Message         string            `json:"message" db:"message" validate:"required"`
	Source          string            `json:"source" db:"source"`
	ResourceID      string            `json:"resource_id" db:"resource_id"`
	ResourceType    string            `json:"resource_type" db:"resource_type"`
	Provider        CloudProvider     `json:"provider" db:"provider"`
	Region          string            `json:"region" db:"region"`
	AccountID       string            `json:"account_id" db:"account_id"`
	Labels          map[string]string `json:"labels" db:"labels"`
	Annotations     map[string]string `json:"annotations" db:"annotations"`
	Value           float64           `json:"value" db:"value"`
	Threshold       float64           `json:"threshold" db:"threshold"`
	Unit            string            `json:"unit" db:"unit"`
	Fingerprint     string            `json:"fingerprint" db:"fingerprint"`
	GroupKey        string            `json:"group_key" db:"group_key"`
	Suppressed      bool              `json:"suppressed" db:"suppressed"`
	SuppressedUntil *time.Time        `json:"suppressed_until" db:"suppressed_until"`
	EscalationLevel int               `json:"escalation_level" db:"escalation_level"`
	EscalatedAt     *time.Time        `json:"escalated_at" db:"escalated_at"`
	ResolvedAt      *time.Time        `json:"resolved_at" db:"resolved_at"`
	ResolvedBy      string            `json:"resolved_by" db:"resolved_by"`
	Resolution      string            `json:"resolution" db:"resolution"`
	RunbookURL      string            `json:"runbook_url" db:"runbook_url"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
}

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusFiring     AlertStatus = "firing"
	AlertStatusResolved   AlertStatus = "resolved"
	AlertStatusSuppressed AlertStatus = "suppressed"
	AlertStatusSilenced   AlertStatus = "silenced"
)

// String returns the string representation of AlertStatus
func (as AlertStatus) String() string {
	return string(as)
}

// AlertSeverity is defined in common.go

// AlertGroup represents a group of related alerts
type AlertGroup struct {
	ID              string            `json:"id" db:"id" validate:"required,uuid"`
	GroupKey        string            `json:"group_key" db:"group_key" validate:"required"`
	RuleID          string            `json:"rule_id" db:"rule_id" validate:"required,uuid"`
	Status          AlertStatus       `json:"status" db:"status" validate:"required"`
	Severity        AlertSeverity     `json:"severity" db:"severity" validate:"required"`
	Title           string            `json:"title" db:"title" validate:"required"`
	Message         string            `json:"message" db:"message" validate:"required"`
	AlertCount      int               `json:"alert_count" db:"alert_count"`
	Alerts          []Alert           `json:"alerts" db:"alerts"`
	Labels          map[string]string `json:"labels" db:"labels"`
	Annotations     map[string]string `json:"annotations" db:"annotations"`
	Suppressed      bool              `json:"suppressed" db:"suppressed"`
	SuppressedUntil *time.Time        `json:"suppressed_until" db:"suppressed_until"`
	ResolvedAt      *time.Time        `json:"resolved_at" db:"resolved_at"`
	ResolvedBy      string            `json:"resolved_by" db:"resolved_by"`
	Resolution      string            `json:"resolution" db:"resolution"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
}

// AlertSilence represents a silence for alerts
type AlertSilence struct {
	ID        string         `json:"id" db:"id" validate:"required,uuid"`
	RuleID    string         `json:"rule_id" db:"rule_id" validate:"required,uuid"`
	Matchers  []AlertMatcher `json:"matchers" db:"matchers" validate:"required"`
	StartsAt  time.Time      `json:"starts_at" db:"starts_at" validate:"required"`
	EndsAt    time.Time      `json:"ends_at" db:"ends_at" validate:"required"`
	Comment   string         `json:"comment" db:"comment" validate:"required"`
	CreatedBy string         `json:"created_by" db:"created_by" validate:"required"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
}

// AlertMatcher represents a matcher for alert silence
type AlertMatcher struct {
	Name    string `json:"name" db:"name" validate:"required"`
	Value   string `json:"value" db:"value" validate:"required"`
	IsRegex bool   `json:"is_regex" db:"is_regex"`
	IsEqual bool   `json:"is_equal" db:"is_equal"`
}

// AlertTemplate represents a template for alert notifications
type AlertTemplate struct {
	ID        string              `json:"id" db:"id" validate:"required,uuid"`
	Name      string              `json:"name" db:"name" validate:"required"`
	Type      NotificationChannel `json:"type" db:"type" validate:"required"`
	Subject   string              `json:"subject" db:"subject"`
	Body      string              `json:"body" db:"body" validate:"required"`
	Variables []string            `json:"variables" db:"variables"`
	IsDefault bool                `json:"is_default" db:"is_default"`
	CreatedBy string              `json:"created_by" db:"created_by" validate:"required"`
	CreatedAt time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt time.Time           `json:"updated_at" db:"updated_at"`
}

// Request/Response Models

// AlertRuleCreateRequest represents a request to create an alert rule
type AlertRuleCreateRequest struct {
	Name               string                  `json:"name" validate:"required"`
	Description        string                  `json:"description"`
	Condition          AlertCondition          `json:"condition" validate:"required"`
	Severity           AlertSeverity           `json:"severity" validate:"required"`
	Channels           []NotificationChannel   `json:"channels" validate:"required"`
	Tags               map[string]string       `json:"tags"`
	IsActive           bool                    `json:"is_active"`
	IsEnabled          bool                    `json:"is_enabled"`
	EvaluationInterval time.Duration           `json:"evaluation_interval"`
	SuppressionConfig  *AlertSuppressionConfig `json:"suppression_config"`
	EscalationConfig   *AlertEscalationConfig  `json:"escalation_config"`
	RunbookURL         string                  `json:"runbook_url"`
}

// AlertRuleListRequest represents a request to list alert rules
type AlertRuleListRequest struct {
	Severity  *AlertSeverity `json:"severity,omitempty"`
	IsActive  *bool          `json:"is_active,omitempty"`
	IsEnabled *bool          `json:"is_enabled,omitempty"`
	CreatedBy *string        `json:"created_by,omitempty"`
	Limit     int            `json:"limit" validate:"min=1,max=1000"`
	Offset    int            `json:"offset" validate:"min=0"`
	SortBy    string         `json:"sort_by" validate:"omitempty,oneof=name created_at updated_at last_triggered trigger_count"`
	SortOrder string         `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// AlertRuleListResponse represents the response for listing alert rules
type AlertRuleListResponse struct {
	Rules  []AlertRule `json:"rules"`
	Total  int         `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

// AlertListRequest represents a request to list alerts
type AlertListRequest struct {
	RuleID       *string        `json:"rule_id,omitempty"`
	Status       *AlertStatus   `json:"status,omitempty"`
	Severity     *AlertSeverity `json:"severity,omitempty"`
	Provider     *CloudProvider `json:"provider,omitempty"`
	ResourceID   *string        `json:"resource_id,omitempty"`
	ResourceType *string        `json:"resource_type,omitempty"`
	StartTime    *time.Time     `json:"start_time,omitempty"`
	EndTime      *time.Time     `json:"end_time,omitempty"`
	Limit        int            `json:"limit" validate:"min=1,max=1000"`
	Offset       int            `json:"offset" validate:"min=0"`
	SortBy       string         `json:"sort_by" validate:"omitempty,oneof=created_at severity status"`
	SortOrder    string         `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// AlertListResponse represents the response for listing alerts
type AlertListResponse struct {
	Alerts []Alert `json:"alerts"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

// AlertResolveRequest represents a request to resolve an alert
type AlertResolveRequest struct {
	Resolution string `json:"resolution" validate:"required"`
	ResolvedBy string `json:"resolved_by" validate:"required"`
}

// AlertSilenceCreateRequest represents a request to create an alert silence
type AlertSilenceCreateRequest struct {
	RuleID   string         `json:"rule_id" validate:"required,uuid"`
	Matchers []AlertMatcher `json:"matchers" validate:"required"`
	StartsAt time.Time      `json:"starts_at" validate:"required"`
	EndsAt   time.Time      `json:"ends_at" validate:"required"`
	Comment  string         `json:"comment" validate:"required"`
}

// Validation methods

// Validate validates the AlertCondition struct
func (ac *AlertCondition) Validate() error {
	validate := validator.New()
	return validate.Struct(ac)
}

// Validate validates the AlertRule struct
func (ar *AlertRule) Validate() error {
	validate := validator.New()
	return validate.Struct(ar)
}

// Validate validates the Alert struct
func (a *Alert) Validate() error {
	validate := validator.New()
	return validate.Struct(a)
}

// Validate validates the AlertGroup struct
func (ag *AlertGroup) Validate() error {
	validate := validator.New()
	return validate.Struct(ag)
}

// Validate validates the AlertSilence struct
func (as *AlertSilence) Validate() error {
	validate := validator.New()
	return validate.Struct(as)
}

// Validate validates the AlertTemplate struct
func (at *AlertTemplate) Validate() error {
	validate := validator.New()
	return validate.Struct(at)
}

// Validate validates the AlertRuleCreateRequest struct
func (arcr *AlertRuleCreateRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(arcr)
}

// Validate validates the AlertRuleListRequest struct
func (arlr *AlertRuleListRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(arlr)
}

// Validate validates the AlertListRequest struct
func (alr *AlertListRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(alr)
}

// Validate validates the AlertResolveRequest struct
func (arr *AlertResolveRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(arr)
}

// Validate validates the AlertSilenceCreateRequest struct
func (ascr *AlertSilenceCreateRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(ascr)
}

// Helper methods

// IsRuleActive returns true if the alert rule is active
func (ar *AlertRule) IsRuleActive() bool {
	return ar.IsActive && ar.IsEnabled
}

// IsFiring returns true if the alert is firing
func (a *Alert) IsFiring() bool {
	return a.Status == AlertStatusFiring
}

// IsResolved returns true if the alert is resolved
func (a *Alert) IsResolved() bool {
	return a.Status == AlertStatusResolved
}

// IsSuppressed returns true if the alert is suppressed
func (a *Alert) IsSuppressed() bool {
	return a.Status == AlertStatusSuppressed || a.Suppressed
}

// IsSilenced returns true if the alert is silenced
func (a *Alert) IsSilenced() bool {
	return a.Status == AlertStatusSilenced
}

// GetDuration returns the duration since the alert was created
func (a *Alert) GetDuration() time.Duration {
	if a.IsResolved() && a.ResolvedAt != nil {
		return a.ResolvedAt.Sub(a.CreatedAt)
	}
	return time.Since(a.CreatedAt)
}

// Resolve resolves the alert
func (a *Alert) Resolve(resolution, resolvedBy string) {
	a.Status = AlertStatusResolved
	a.Resolution = resolution
	a.ResolvedBy = resolvedBy
	now := time.Now()
	a.ResolvedAt = &now
	a.UpdatedAt = now
}

// Suppress suppresses the alert
func (a *Alert) Suppress(until time.Time) {
	a.Status = AlertStatusSuppressed
	a.Suppressed = true
	a.SuppressedUntil = &until
	a.UpdatedAt = time.Now()
}

// Escalate escalates the alert
func (a *Alert) Escalate(level int) {
	a.EscalationLevel = level
	now := time.Now()
	a.EscalatedAt = &now
	a.UpdatedAt = now
}

// IsActive returns true if the alert group is active
func (ag *AlertGroup) IsActive() bool {
	return ag.Status == AlertStatusFiring
}

// IsResolved returns true if the alert group is resolved
func (ag *AlertGroup) IsResolved() bool {
	return ag.Status == AlertStatusResolved
}

// GetDuration returns the duration since the alert group was created
func (ag *AlertGroup) GetDuration() time.Duration {
	if ag.IsResolved() && ag.ResolvedAt != nil {
		return ag.ResolvedAt.Sub(ag.CreatedAt)
	}
	return time.Since(ag.CreatedAt)
}

// Resolve resolves the alert group
func (ag *AlertGroup) Resolve(resolution, resolvedBy string) {
	ag.Status = AlertStatusResolved
	ag.Resolution = resolution
	ag.ResolvedBy = resolvedBy
	now := time.Now()
	ag.ResolvedAt = &now
	ag.UpdatedAt = now
}

// IsActive returns true if the alert silence is active
func (as *AlertSilence) IsActive() bool {
	now := time.Now()
	return now.After(as.StartsAt) && now.Before(as.EndsAt)
}

// IsExpired returns true if the alert silence is expired
func (as *AlertSilence) IsExpired() bool {
	return time.Now().After(as.EndsAt)
}

// GetRemainingDuration returns the remaining duration of the silence
func (as *AlertSilence) GetRemainingDuration() time.Duration {
	if as.IsExpired() {
		return 0
	}
	return as.EndsAt.Sub(time.Now())
}
