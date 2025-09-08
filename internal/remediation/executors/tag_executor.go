package executors

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/remediation"
)

// TagExecutor handles tag-related remediation actions
type TagExecutor struct {
	executorType string
}

// NewTagExecutor creates a new tag executor
func NewTagExecutor() *TagExecutor {
	return &TagExecutor{
		executorType: "tag",
	}
}

// Execute executes a tag remediation action
func (te *TagExecutor) Execute(ctx context.Context, action *remediation.RemediationAction) (*remediation.ActionResult, error) {
	result := &remediation.ActionResult{
		Success:    true,
		Message:    "Tag remediation completed successfully",
		Changes:    []remediation.ResourceChange{},
		Metrics:    make(map[string]interface{}),
		CostImpact: 0.0,
		RiskLevel:  remediation.RiskLevelLow,
	}

	// Get the operation type from parameters
	operation, ok := action.Parameters["operation"].(string)
	if !ok {
		return &remediation.ActionResult{
			Success: false,
			Error:   "operation parameter is required",
		}, fmt.Errorf("operation parameter is required")
	}

	switch operation {
	case "add_tag":
		return te.addTag(ctx, action, result)
	case "remove_tag":
		return te.removeTag(ctx, action, result)
	case "update_tag":
		return te.updateTag(ctx, action, result)
	case "add_required_tags":
		return te.addRequiredTags(ctx, action, result)
	default:
		return &remediation.ActionResult{
			Success: false,
			Error:   fmt.Sprintf("unsupported operation: %s", operation),
		}, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// GetType returns the executor type
func (te *TagExecutor) GetType() string {
	return te.executorType
}

// GetDescription returns the executor description
func (te *TagExecutor) GetDescription() string {
	return "Tag remediation executor for adding, removing, and updating resource tags"
}

// Validate validates a tag action
func (te *TagExecutor) Validate(action *remediation.RemediationAction) error {
	operation, ok := action.Parameters["operation"].(string)
	if !ok {
		return fmt.Errorf("operation parameter is required")
	}

	switch operation {
	case "add_tag", "update_tag":
		if _, ok := action.Parameters["key"].(string); !ok {
			return fmt.Errorf("key parameter is required for %s operation", operation)
		}
		if _, ok := action.Parameters["value"].(string); !ok {
			return fmt.Errorf("value parameter is required for %s operation", operation)
		}
	case "remove_tag":
		if _, ok := action.Parameters["key"].(string); !ok {
			return fmt.Errorf("key parameter is required for %s operation", operation)
		}
	case "add_required_tags":
		if _, ok := action.Parameters["required_tags"].(map[string]interface{}); !ok {
			return fmt.Errorf("required_tags parameter is required for add_required_tags operation")
		}
	default:
		return fmt.Errorf("unsupported operation: %s", operation)
	}

	return nil
}

// addTag adds a tag to a resource
func (te *TagExecutor) addTag(ctx context.Context, action *remediation.RemediationAction, result *remediation.ActionResult) (*remediation.ActionResult, error) {
	key := action.Parameters["key"].(string)
	value := action.Parameters["value"].(string)

	// Simulate adding tag (in real implementation, this would call the cloud provider API)
	time.Sleep(100 * time.Millisecond) // Simulate API call

	// Record the change
	change := remediation.ResourceChange{
		ResourceID: action.ResourceID,
		Field:      fmt.Sprintf("tags.%s", key),
		OldValue:   nil,
		NewValue:   value,
		ChangeType: remediation.ChangeTypeCreate,
		Metadata: map[string]interface{}{
			"tag_key":   key,
			"tag_value": value,
		},
	}
	result.Changes = append(result.Changes, change)

	result.Message = fmt.Sprintf("Added tag %s=%s to resource %s", key, value, action.ResourceID)
	result.Metrics["tags_added"] = 1

	return result, nil
}

// removeTag removes a tag from a resource
func (te *TagExecutor) removeTag(ctx context.Context, action *remediation.RemediationAction, result *remediation.ActionResult) (*remediation.ActionResult, error) {
	key := action.Parameters["key"].(string)

	// Simulate removing tag (in real implementation, this would call the cloud provider API)
	time.Sleep(100 * time.Millisecond) // Simulate API call

	// Record the change
	change := remediation.ResourceChange{
		ResourceID: action.ResourceID,
		Field:      fmt.Sprintf("tags.%s", key),
		OldValue:   "existing_value", // In real implementation, this would be the actual old value
		NewValue:   nil,
		ChangeType: remediation.ChangeTypeDelete,
		Metadata: map[string]interface{}{
			"tag_key": key,
		},
	}
	result.Changes = append(result.Changes, change)

	result.Message = fmt.Sprintf("Removed tag %s from resource %s", key, action.ResourceID)
	result.Metrics["tags_removed"] = 1

	return result, nil
}

// updateTag updates a tag value on a resource
func (te *TagExecutor) updateTag(ctx context.Context, action *remediation.RemediationAction, result *remediation.ActionResult) (*remediation.ActionResult, error) {
	key := action.Parameters["key"].(string)
	value := action.Parameters["value"].(string)

	// Simulate updating tag (in real implementation, this would call the cloud provider API)
	time.Sleep(100 * time.Millisecond) // Simulate API call

	// Record the change
	change := remediation.ResourceChange{
		ResourceID: action.ResourceID,
		Field:      fmt.Sprintf("tags.%s", key),
		OldValue:   "old_value", // In real implementation, this would be the actual old value
		NewValue:   value,
		ChangeType: remediation.ChangeTypeUpdate,
		Metadata: map[string]interface{}{
			"tag_key":   key,
			"tag_value": value,
		},
	}
	result.Changes = append(result.Changes, change)

	result.Message = fmt.Sprintf("Updated tag %s=%s on resource %s", key, value, action.ResourceID)
	result.Metrics["tags_updated"] = 1

	return result, nil
}

// addRequiredTags adds required tags to a resource
func (te *TagExecutor) addRequiredTags(ctx context.Context, action *remediation.RemediationAction, result *remediation.ActionResult) (*remediation.ActionResult, error) {
	requiredTags := action.Parameters["required_tags"].(map[string]interface{})

	// Simulate adding multiple tags (in real implementation, this would call the cloud provider API)
	time.Sleep(200 * time.Millisecond) // Simulate API call

	tagsAdded := 0
	for key, value := range requiredTags {
		change := remediation.ResourceChange{
			ResourceID: action.ResourceID,
			Field:      fmt.Sprintf("tags.%s", key),
			OldValue:   nil,
			NewValue:   value,
			ChangeType: remediation.ChangeTypeCreate,
			Metadata: map[string]interface{}{
				"tag_key":   key,
				"tag_value": value,
			},
		}
		result.Changes = append(result.Changes, change)
		tagsAdded++
	}

	result.Message = fmt.Sprintf("Added %d required tags to resource %s", tagsAdded, action.ResourceID)
	result.Metrics["tags_added"] = tagsAdded

	return result, nil
}
