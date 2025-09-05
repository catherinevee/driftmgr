package remediation

import (
	"time"
)

// RemediationConfig holds configuration for remediation operations
type RemediationConfig struct {
	AutoApprove     bool          `json:"auto_approve"`
	DryRun          bool          `json:"dry_run"`
	OutputFormat    string        `json:"output_format"`
	Timeout         time.Duration `json:"timeout"`
	MaxRetries      int           `json:"max_retries"`
	BackupState     bool          `json:"backup_state"`
	ParallelOps     int           `json:"parallel_ops"`
	RequireApproval bool          `json:"require_approval"`
}

// DefaultRemediationConfig returns default remediation configuration
func DefaultRemediationConfig() *RemediationConfig {
	return &RemediationConfig{
		AutoApprove:     false,
		DryRun:          false,
		OutputFormat:    "json",
		Timeout:         30 * time.Minute,
		MaxRetries:      3,
		BackupState:     true,
		ParallelOps:     5,
		RequireApproval: true,
	}
}
