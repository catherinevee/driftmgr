package remediation

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// Remediator handles drift remediation
type Remediator struct {
	// Store providers as interfaces since we'll use them through the discovery interface
}

// RemediationAction represents a remediation action
type RemediationAction struct {
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	Provider     string                 `json:"provider"`
	Action       string                 `json:"action"`
	Parameters   map[string]interface{} `json:"parameters"`
	Status       string                 `json:"status"`
	Error        string                 `json:"error,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
}

// RemediationResult represents the result of remediation
type RemediationResult struct {
	Success bool                `json:"success"`
	Actions []RemediationAction `json:"actions"`
	Summary map[string]int      `json:"summary"`
}

// NewRemediator creates a new remediator
func NewRemediator() *Remediator {
	return &Remediator{}
}

// Remediate performs remediation for drifted resources
func (r *Remediator) Remediate(ctx context.Context, drifts []models.DriftItem, dryRun bool) (*RemediationResult, error) {
	result := &RemediationResult{
		Success: true,
		Actions: []RemediationAction{},
		Summary: map[string]int{
			"total":     len(drifts),
			"succeeded": 0,
			"failed":    0,
			"skipped":   0,
		},
	}

	for _, drift := range drifts {
		action := r.createRemediationAction(drift)
		
		if dryRun {
			action.Status = "dry_run"
			result.Summary["skipped"]++
		} else {
			// Perform actual remediation based on provider
			err := r.performRemediation(ctx, drift, action)
			if err != nil {
				action.Status = "failed"
				action.Error = err.Error()
				result.Summary["failed"]++
				result.Success = false
			} else {
				action.Status = "succeeded"
				result.Summary["succeeded"]++
			}
		}
		
		action.Timestamp = time.Now()
		result.Actions = append(result.Actions, *action)
	}

	return result, nil
}

// createRemediationAction creates a remediation action from a drift item
func (r *Remediator) createRemediationAction(drift models.DriftItem) *RemediationAction {
	action := &RemediationAction{
		ResourceID:   drift.ResourceID,
		ResourceType: drift.ResourceType,
		Provider:     drift.Provider,
		Parameters:   make(map[string]interface{}),
	}

	// Determine action based on drift type
	switch drift.DriftType {
	case "created":
		action.Action = "delete"
	case "deleted":
		action.Action = "create"
	case "modified":
		action.Action = "update"
		// Add expected values as parameters
		if expected, ok := drift.Details["expected"].(map[string]interface{}); ok {
			action.Parameters = expected
		}
	default:
		action.Action = "unknown"
	}

	return action
}

// performRemediation performs the actual remediation
func (r *Remediator) performRemediation(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	switch drift.Provider {
	case "aws":
		return r.remediateAWS(ctx, drift, action)
	case "azure":
		return r.remediateAzure(ctx, drift, action)
	case "gcp":
		return r.remediateGCP(ctx, drift, action)
	default:
		return fmt.Errorf("unsupported provider: %s", drift.Provider)
	}
}

// remediateAWS handles AWS resource remediation
func (r *Remediator) remediateAWS(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	// Get region from drift details or default
	region := "us-east-1"
	if drift.Region != "" {
		region = drift.Region
	}
	
	// Create AWS remediator
	awsRemediator, err := NewAWSRemediator(ctx, region)
	if err != nil {
		return fmt.Errorf("failed to create AWS remediator: %w", err)
	}
	
	// Perform remediation using the AWS-specific remediator
	return awsRemediator.Remediate(ctx, drift, action)
}

// remediateAzure handles Azure resource remediation
func (r *Remediator) remediateAzure(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	// Extract subscription ID from resource ID or use default
	subscriptionID := ""
	if strings.Contains(drift.ResourceID, "/subscriptions/") {
		parts := strings.Split(drift.ResourceID, "/")
		for i, part := range parts {
			if part == "subscriptions" && i+1 < len(parts) {
				subscriptionID = parts[i+1]
				break
			}
		}
	}
	
	if subscriptionID == "" {
		// Try to get from environment or default
		subscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")
		if subscriptionID == "" {
			return fmt.Errorf("Azure subscription ID not found")
		}
	}
	
	// Create Azure remediator
	azureRemediator, err := NewAzureRemediator(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to create Azure remediator: %w", err)
	}
	
	// Perform remediation using the Azure-specific remediator
	return azureRemediator.Remediate(ctx, drift, action)
}

// remediateGCP handles GCP resource remediation
func (r *Remediator) remediateGCP(ctx context.Context, drift models.DriftItem, action *RemediationAction) error {
	// This would use GCP SDK to perform actual remediation
	switch action.Action {
	case "update":
		// Update GCP resource
		return nil
	case "delete":
		// Delete GCP resource
		return nil
	case "create":
		// Create GCP resource
		return nil
	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}
}