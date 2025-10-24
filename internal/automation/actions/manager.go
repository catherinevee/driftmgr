package actions

import (
	"context"
	"fmt"

	"github.com/catherinevee/driftmgr/internal/events"
	"github.com/catherinevee/driftmgr/internal/models"
)

// ActionManager manages and coordinates all automation actions
type ActionManager struct {
	notificationAction     *NotificationAction
	loggingAction          *LoggingAction
	resourceUpdateAction   *ResourceUpdateAction
	alertCreationAction    *AlertCreationAction
	scriptExecutionAction  *ScriptExecutionAction
	commandExecutionAction *CommandExecutionAction
	eventBus               *events.EventBus
}

// NewActionManager creates a new action manager
func NewActionManager(eventBus *events.EventBus, notificationService *events.NotificationService, logDir string) *ActionManager {
	return &ActionManager{
		notificationAction:     NewNotificationAction(notificationService, eventBus),
		loggingAction:          NewLoggingAction(eventBus, logDir),
		resourceUpdateAction:   NewResourceUpdateAction(eventBus),
		alertCreationAction:    NewAlertCreationAction(eventBus),
		scriptExecutionAction:  NewScriptExecutionAction(eventBus),
		commandExecutionAction: NewCommandExecutionAction(eventBus),
		eventBus:               eventBus,
	}
}

// ExecuteAction executes an automation action based on its type
func (am *ActionManager) ExecuteAction(ctx context.Context, action *models.AutomationAction, context map[string]interface{}) (*models.ActionResult, error) {
	// Determine action type and delegate to appropriate handler
	switch action.Type {
	case models.ActionTypeNotification:
		return am.notificationAction.Execute(ctx, action, context)
	case models.ActionTypeScript:
		return am.scriptExecutionAction.Execute(ctx, action, context)
	case models.ActionTypeCustom:
		// For custom actions, determine the subtype from the configuration
		return am.executeCustomAction(ctx, action, context)
	default:
		return nil, fmt.Errorf("unsupported action type: %s", action.Type)
	}
}

// ValidateAction validates an automation action
func (am *ActionManager) ValidateAction(action *models.AutomationAction) error {
	switch action.Type {
	case models.ActionTypeNotification:
		return am.notificationAction.Validate(action)
	case models.ActionTypeScript:
		return am.scriptExecutionAction.Validate(action)
	case models.ActionTypeCustom:
		// For custom actions, determine the subtype from the configuration
		return am.validateCustomAction(action)
	default:
		return fmt.Errorf("unsupported action type: %s", action.Type)
	}
}

// GetSupportedActionTypes returns the list of supported action types
func (am *ActionManager) GetSupportedActionTypes() []models.ActionType {
	return []models.ActionType{
		models.ActionTypeNotification,
		models.ActionTypeScript,
		models.ActionTypeCustom,
	}
}

// executeCustomAction executes a custom action based on its subtype
func (am *ActionManager) executeCustomAction(ctx context.Context, action *models.AutomationAction, context map[string]interface{}) (*models.ActionResult, error) {
	// Determine the custom action subtype from the configuration
	subtype, err := am.getCustomActionSubtype(action)
	if err != nil {
		return nil, fmt.Errorf("failed to determine custom action subtype: %w", err)
	}

	switch subtype {
	case "logging":
		return am.loggingAction.Execute(ctx, action, context)
	case "resource_update":
		return am.resourceUpdateAction.Execute(ctx, action, context)
	case "alert_creation":
		return am.alertCreationAction.Execute(ctx, action, context)
	case "command_execution":
		return am.commandExecutionAction.Execute(ctx, action, context)
	default:
		return nil, fmt.Errorf("unsupported custom action subtype: %s", subtype)
	}
}

// validateCustomAction validates a custom action based on its subtype
func (am *ActionManager) validateCustomAction(action *models.AutomationAction) error {
	// Determine the custom action subtype from the configuration
	subtype, err := am.getCustomActionSubtype(action)
	if err != nil {
		return fmt.Errorf("failed to determine custom action subtype: %w", err)
	}

	switch subtype {
	case "logging":
		return am.loggingAction.Validate(action)
	case "resource_update":
		return am.resourceUpdateAction.Validate(action)
	case "alert_creation":
		return am.alertCreationAction.Validate(action)
	case "command_execution":
		return am.commandExecutionAction.Validate(action)
	default:
		return fmt.Errorf("unsupported custom action subtype: %s", subtype)
	}
}

// getCustomActionSubtype determines the subtype of a custom action from its configuration
func (am *ActionManager) getCustomActionSubtype(action *models.AutomationAction) (string, error) {
	// JSONB is already a map[string]interface{}
	configMap := action.Configuration

	// Look for subtype in configuration
	if subtypeVal, ok := configMap["subtype"].(string); ok {
		return subtypeVal, nil
	}

	// Look for action_type in configuration (alternative field name)
	if actionTypeVal, ok := configMap["action_type"].(string); ok {
		return actionTypeVal, nil
	}

	// Look for type in configuration (another alternative field name)
	if typeVal, ok := configMap["type"].(string); ok {
		return typeVal, nil
	}

	return "", fmt.Errorf("custom action subtype not found in configuration")
}

// GetActionHandlers returns all available action handlers
func (am *ActionManager) GetActionHandlers() map[string]interface{} {
	return map[string]interface{}{
		"notification":      am.notificationAction,
		"logging":           am.loggingAction,
		"resource_update":   am.resourceUpdateAction,
		"alert_creation":    am.alertCreationAction,
		"script_execution":  am.scriptExecutionAction,
		"command_execution": am.commandExecutionAction,
	}
}

// GetActionCapabilities returns the capabilities of each action type
func (am *ActionManager) GetActionCapabilities() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		"notification": {
			"description": "Send notifications through various channels",
			"channels":    []string{"websocket", "email", "slack", "webhook", "sms", "push"},
			"features":    []string{"templating", "filtering", "priority", "retry"},
		},
		"logging": {
			"description": "Structured logging for automation events",
			"formats":     []string{"json", "text", "structured"},
			"outputs":     []string{"stdout", "stderr", "file"},
			"features":    []string{"templating", "retention", "rotation"},
		},
		"resource_update": {
			"description": "Update cloud resources",
			"providers":   []string{"aws", "azure", "gcp", "digitalocean"},
			"operations":  []string{"update", "tag", "scale", "configure", "restart", "stop", "start"},
			"features":    []string{"backup", "dry_run", "validation"},
		},
		"alert_creation": {
			"description": "Create alerts and notifications",
			"types":       []string{"incident", "warning", "info", "maintenance"},
			"severities":  []string{"low", "medium", "high", "critical"},
			"features":    []string{"escalation", "suppression", "correlation", "templating"},
		},
		"script_execution": {
			"description": "Execute scripts and commands",
			"types":       []string{"shell", "bash", "python", "powershell", "cmd", "custom"},
			"features":    []string{"retry", "timeout", "output_capture", "validation"},
		},
		"command_execution": {
			"description": "Execute system commands",
			"shells":      []string{"bash", "sh", "cmd", "powershell", "zsh", "fish"},
			"features":    []string{"retry", "timeout", "output_capture", "validation"},
		},
	}
}
