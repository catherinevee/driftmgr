package remediation

import (
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/go-playground/validator/v10"
)

// Validator holds the validator instance
type Validator struct {
	validator *validator.Validate
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	v := validator.New()

	// Register custom validators
	v.RegisterValidation("job_status", validateJobStatus)
	v.RegisterValidation("job_priority", validateJobPriority)
	v.RegisterValidation("strategy_type", validateStrategyType)
	v.RegisterValidation("log_level", validateLogLevel)

	return &Validator{
		validator: v,
	}
}

// ValidateJobRequest validates a remediation job request
func (v *Validator) ValidateJobRequest(req *models.RemediationJobRequest) error {
	return v.validator.Struct(req)
}

// ValidateJobListRequest validates a job list request
func (v *Validator) ValidateJobListRequest(req *models.RemediationJobListRequest) error {
	return v.validator.Struct(req)
}

// ValidateStrategyRequest validates a strategy request
func (v *Validator) ValidateStrategyRequest(req *models.RemediationStrategyRequest) error {
	return v.validator.Struct(req)
}

// ValidateHistoryRequest validates a history request
func (v *Validator) ValidateHistoryRequest(req *models.RemediationHistoryRequest) error {
	return v.validator.Struct(req)
}

// ValidateProgressUpdate validates a progress update
func (v *Validator) ValidateProgressUpdate(update *models.JobProgressUpdate) error {
	return v.validator.Struct(update)
}

// ValidateCancelRequest validates a cancel request
func (v *Validator) ValidateCancelRequest(req *models.JobCancelRequest) error {
	return v.validator.Struct(req)
}

// ValidateApprovalRequest validates an approval request
func (v *Validator) ValidateApprovalRequest(req *models.ApprovalRequest) error {
	return v.validator.Struct(req)
}

// Custom validation functions

// validateJobStatus validates job status
func validateJobStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	validStatuses := []string{
		string(models.JobStatusPending),
		string(models.JobStatusQueued),
		string(models.JobStatusRunning),
		string(models.JobStatusCompleted),
		string(models.JobStatusFailed),
		string(models.JobStatusCancelled),
	}

	for _, validStatus := range validStatuses {
		if status == validStatus {
			return true
		}
	}
	return false
}

// validateJobPriority validates job priority
func validateJobPriority(fl validator.FieldLevel) bool {
	priority := fl.Field().String()
	validPriorities := []string{
		string(models.JobPriorityLow),
		string(models.JobPriorityMedium),
		string(models.JobPriorityHigh),
		string(models.JobPriorityCritical),
	}

	for _, validPriority := range validPriorities {
		if priority == validPriority {
			return true
		}
	}
	return false
}

// validateStrategyType validates strategy type
func validateStrategyType(fl validator.FieldLevel) bool {
	strategyType := fl.Field().String()
	validTypes := []string{
		string(models.StrategyTypeTerraformApply),
		string(models.StrategyTypeTerraformDestroy),
		string(models.StrategyTypeTerraformImport),
		string(models.StrategyTypeStateManipulation),
		string(models.StrategyTypeResourceCreation),
		string(models.StrategyTypeResourceDeletion),
	}

	for _, validType := range validTypes {
		if strategyType == validType {
			return true
		}
	}
	return false
}

// validateLogLevel validates log level
func validateLogLevel(fl validator.FieldLevel) bool {
	level := fl.Field().String()
	validLevels := []string{
		string(models.LogLevelDebug),
		string(models.LogLevelInfo),
		string(models.LogLevelWarn),
		string(models.LogLevelError),
	}

	for _, validLevel := range validLevels {
		if level == validLevel {
			return true
		}
	}
	return false
}

// Validation helpers

// ValidateJobID validates a job ID
func ValidateJobID(jobID string) error {
	if jobID == "" {
		return models.ErrBadRequest
	}

	// Add more specific validation if needed
	// For example, check UUID format
	return nil
}

// ValidateStrategyID validates a strategy ID
func ValidateStrategyID(strategyID string) error {
	if strategyID == "" {
		return models.ErrBadRequest
	}

	// Add more specific validation if needed
	return nil
}

// ValidateDateRange validates a date range
func ValidateDateRange(startDate, endDate *time.Time) error {
	if startDate != nil && endDate != nil {
		if startDate.After(*endDate) {
			return models.ErrInvalidDateRange
		}
	}
	return nil
}

// ValidatePagination validates pagination parameters
func ValidatePagination(limit, offset int) error {
	if limit < 1 {
		return models.ErrInvalidLimit
	}
	if limit > 1000 {
		return models.ErrLimitTooHigh
	}
	if offset < 0 {
		return models.ErrInvalidOffset
	}
	return nil
}

// ValidateSorting validates sorting parameters
func ValidateSorting(sortBy, sortOrder string) error {
	validSortFields := []string{"created_at", "updated_at", "status", "priority"}
	validSortOrders := []string{"asc", "desc"}

	if sortBy != "" {
		valid := false
		for _, field := range validSortFields {
			if sortBy == field {
				valid = true
				break
			}
		}
		if !valid {
			return models.ErrInvalidSortField
		}
	}

	if sortOrder != "" {
		valid := false
		for _, order := range validSortOrders {
			if sortOrder == order {
				valid = true
				break
			}
		}
		if !valid {
			return models.ErrInvalidSortOrder
		}
	}

	return nil
}
