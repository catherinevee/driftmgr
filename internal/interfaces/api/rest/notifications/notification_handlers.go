package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/catherinevee/driftmgr/internal/notifications"
)

// handleSendNotification sends a notification
func (s *Server) handleSendNotification(w http.ResponseWriter, r *http.Request) {
	if s.notifier == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Notification system not configured")
		return
	}
	
	var req struct {
		Type     string                 `json:"type"`
		Title    string                 `json:"title"`
		Message  string                 `json:"message"`
		Severity string                 `json:"severity"`
		Metadata map[string]interface{} `json:"metadata"`
		Channels []string               `json:"channels"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Validate required fields
	if req.Title == "" || req.Message == "" {
		s.respondError(w, http.StatusBadRequest, "Title and message are required")
		return
	}
	
	// Set defaults
	if req.Type == "" {
		req.Type = "info"
	}
	if req.Severity == "" {
		req.Severity = "medium"
	}
	
	// Convert channels
	channels := []notifications.Channel{}
	for _, ch := range req.Channels {
		channels = append(channels, notifications.Channel(ch))
	}
	
	// Create notification
	notification := &notifications.Notification{
		Type:     notifications.NotificationType(req.Type),
		Title:    req.Title,
		Message:  req.Message,
		Severity: req.Severity,
		Metadata: req.Metadata,
		Channels: channels,
	}
	
	// Send notification
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	
	if err := s.notifier.Send(ctx, notification); err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to send notification: %v", err))
		return
	}
	
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Notification queued successfully",
		"id":      notification.ID,
	})
}

// handleNotificationHistory returns notification history
func (s *Server) handleNotificationHistory(w http.ResponseWriter, r *http.Request) {
	if s.notifier == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Notification system not configured")
		return
	}
	
	// Get limit from query params
	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	// Get history
	history := s.notifier.GetHistory(limit)
	
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"notifications": history,
		"count":         len(history),
		"limit":         limit,
	})
}

// handleTestNotification sends a test notification
func (s *Server) handleTestNotification(w http.ResponseWriter, r *http.Request) {
	if s.notifier == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Notification system not configured")
		return
	}
	
	var req struct {
		Channel string `json:"channel"`
	}
	
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}
	
	// Create test notification
	notification := &notifications.Notification{
		Type:     notifications.TypeInfo,
		Title:    "Test Notification",
		Message:  fmt.Sprintf("This is a test notification from DriftMgr sent at %s", time.Now().Format(time.RFC3339)),
		Severity: "low",
		Metadata: map[string]interface{}{
			"test":      true,
			"timestamp": time.Now().Unix(),
			"source":    "api",
		},
	}
	
	// Set channel if specified
	if req.Channel != "" {
		notification.Channels = []notifications.Channel{notifications.Channel(req.Channel)}
	}
	
	// Send notification
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	
	if err := s.notifier.Send(ctx, notification); err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to send test notification: %v", err))
		return
	}
	
	// Wait a bit for the notification to be processed
	time.Sleep(2 * time.Second)
	
	// Check if it was sent successfully
	history := s.notifier.GetHistory(1)
	if len(history) > 0 && history[0].ID == notification.ID {
		if history[0].Status == "sent" {
			s.respondJSON(w, http.StatusOK, map[string]interface{}{
				"success": true,
				"message": "Test notification sent successfully",
				"channel": req.Channel,
			})
		} else {
			s.respondJSON(w, http.StatusOK, map[string]interface{}{
				"success": false,
				"message": "Test notification failed",
				"error":   history[0].Error,
			})
		}
	} else {
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"message": "Test notification status unknown",
		})
	}
}

// handleNotificationConfig manages notification configuration
func (s *Server) handleNotificationConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Get current notification configuration
		if s.configManager != nil {
			cfg := s.configManager.Get()
			if cfg != nil {
				s.respondJSON(w, http.StatusOK, cfg.Settings.Notifications)
				return
			}
		}
		
		// Return default config
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"enabled":  false,
			"channels": []string{},
		})
		return
	}
	
	// POST - Update notification configuration
	var req struct {
		Enabled  bool                          `json:"enabled"`
		Channels []string                      `json:"channels"`
		Email    notifications.EmailConfig     `json:"email"`
		Slack    notifications.SlackConfig     `json:"slack"`
		Webhooks map[string]interface{}        `json:"webhooks"`
		Rules    []notifications.NotificationRule `json:"rules"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Update configuration
	if s.configManager != nil {
		// Update notification settings in config
		s.configManager.Set("settings.notifications.enabled", req.Enabled)
		s.configManager.Set("settings.notifications.channels", req.Channels)
		
		// Save configuration
		if err := s.configManager.Save(); err != nil {
			s.respondError(w, http.StatusInternalServerError, "Failed to save configuration")
			return
		}
		
		// Reinitialize notifier with new config
		if req.Enabled && s.notifier == nil {
			notifConfig := &notifications.Config{
				Enabled: req.Enabled,
				Email:   req.Email,
				Slack:   req.Slack,
				Rules:   req.Rules,
			}
			s.notifier = notifications.NewNotifier(notifConfig)
		} else if !req.Enabled && s.notifier != nil {
			// Stop notifier if disabled
			s.notifier.Stop()
			s.notifier = nil
		}
	}
	
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Notification configuration updated",
	})
}

// sendDriftNotification sends a notification about drift detection
func (s *Server) sendDriftNotification(driftCount int, criticalCount int) {
	if s.notifier == nil {
		return
	}
	
	severity := "low"
	notifType := notifications.TypeInfo
	
	if criticalCount > 0 {
		severity = "high"
		notifType = notifications.TypeAlert
	} else if driftCount > 10 {
		severity = "medium"
		notifType = notifications.TypeWarning
	}
	
	notification := &notifications.Notification{
		Type:     notifType,
		Title:    fmt.Sprintf("Drift Detected: %d resources", driftCount),
		Message:  fmt.Sprintf("DriftMgr has detected drift in %d resources. Critical drift: %d", driftCount, criticalCount),
		Severity: severity,
		Metadata: map[string]interface{}{
			"drift_count":    driftCount,
			"critical_count": criticalCount,
			"timestamp":      time.Now().Unix(),
			"source":         "drift_detector",
		},
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	s.notifier.Send(ctx, notification)
}

// sendRemediationNotification sends a notification about remediation
func (s *Server) sendRemediationNotification(action string, resourceCount int, success bool) {
	if s.notifier == nil {
		return
	}
	
	notifType := notifications.TypeSuccess
	if !success {
		notifType = notifications.TypeError
	}
	
	notification := &notifications.Notification{
		Type:     notifType,
		Title:    fmt.Sprintf("Remediation %s: %d resources", action, resourceCount),
		Message:  fmt.Sprintf("Remediation action '%s' %s for %d resources", action, map[bool]string{true: "succeeded", false: "failed"}[success], resourceCount),
		Severity: map[bool]string{true: "low", false: "high"}[success],
		Metadata: map[string]interface{}{
			"action":         action,
			"resource_count": resourceCount,
			"success":        success,
			"timestamp":      time.Now().Unix(),
			"source":         "remediator",
		},
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	s.notifier.Send(ctx, notification)
}