package models

import "errors"

// Common error definitions for the application
var (
	// Validation errors
	ErrInvalidProvider  = errors.New("invalid provider")
	ErrInvalidLimit     = errors.New("invalid limit")
	ErrInvalidOffset    = errors.New("invalid offset")
	ErrLimitTooHigh     = errors.New("limit too high")
	ErrInvalidDateRange = errors.New("invalid date range")
	ErrInvalidSortField = errors.New("invalid sort field")
	ErrInvalidSortOrder = errors.New("invalid sort order")

	// Drift result errors
	ErrDriftResultNotFound = errors.New("drift result not found")
	ErrDriftResultExists   = errors.New("drift result already exists")
	ErrInvalidDriftStatus  = errors.New("invalid drift status")
	ErrDriftResultInUse    = errors.New("drift result is in use")

	// Database errors
	ErrDatabaseConnection  = errors.New("database connection error")
	ErrDatabaseQuery       = errors.New("database query error")
	ErrDatabaseTransaction = errors.New("database transaction error")

	// Authentication and authorization errors
	ErrInvalidCredentials = errors.New("invalid credentials")

	// Resource errors
	ErrResourceNotFound = errors.New("resource not found")
	ErrResourceExists   = errors.New("resource already exists")
	ErrResourceInUse    = errors.New("resource is in use")

	// Configuration errors
	ErrInvalidConfiguration = errors.New("invalid configuration")
	ErrMissingConfiguration = errors.New("missing configuration")

	// Provider errors
	ErrProviderNotSupported = errors.New("provider not supported")
	ErrProviderConnection   = errors.New("provider connection error")
	ErrProviderAuth         = errors.New("provider authentication error")

	// Remediation errors
	ErrRemediationJobNotFound      = errors.New("remediation job not found")
	ErrRemediationJobExists        = errors.New("remediation job already exists")
	ErrRemediationJobInUse         = errors.New("remediation job is in use")
	ErrRemediationStrategyNotFound = errors.New("remediation strategy not found")
	ErrRemediationStrategyExists   = errors.New("remediation strategy already exists")
	ErrInvalidJobStatus            = errors.New("invalid job status")
	ErrInvalidJobPriority          = errors.New("invalid job priority")
	ErrInvalidStrategyType         = errors.New("invalid strategy type")
	ErrJobCannotBeCancelled        = errors.New("job cannot be cancelled")
	ErrJobRequiresApproval         = errors.New("job requires approval")
	ErrJobAlreadyApproved          = errors.New("job already approved")
	ErrJobNotApproved              = errors.New("job not approved")

	// State management errors
	ErrStateFileNotFound      = errors.New("state file not found")
	ErrStateFileExists        = errors.New("state file already exists")
	ErrStateFileLocked        = errors.New("state file is locked")
	ErrStateFileCorrupted     = errors.New("state file is corrupted")
	ErrBackendNotFound        = errors.New("backend not found")
	ErrBackendExists          = errors.New("backend already exists")
	ErrBackendNotSupported    = errors.New("backend type not supported")
	ErrBackendConfiguration   = errors.New("invalid backend configuration")
	ErrResourceImportFailed   = errors.New("resource import failed")
	ErrResourceRemoveFailed   = errors.New("resource removal failed")
	ErrResourceMoveFailed     = errors.New("resource move failed")
	ErrResourceExportFailed   = errors.New("resource export failed")
	ErrStateOperationNotFound = errors.New("state operation not found")
	ErrStateOperationExists   = errors.New("state operation already exists")
	ErrStateOperationFailed   = errors.New("state operation failed")
	ErrStateLockNotFound      = errors.New("state lock not found")
	ErrStateLockExists        = errors.New("state lock already exists")
	ErrStateLockFailed        = errors.New("state lock failed")
	ErrStateUnlockFailed      = errors.New("state unlock failed")
	ErrInvalidStateFormat     = errors.New("invalid state file format")
	ErrInvalidResourceAddress = errors.New("invalid resource address")
	ErrInvalidBackendType     = errors.New("invalid backend type")
	ErrInvalidOperationType   = errors.New("invalid operation type")

	// General errors
	ErrInternalServer     = errors.New("internal server error")
	ErrBadRequest         = errors.New("bad request")
	ErrConflict           = errors.New("conflict")
	ErrTooManyRequests    = errors.New("too many requests")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrTimeout            = errors.New("timeout")
)
