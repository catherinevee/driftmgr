package rules

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/google/uuid"
)

// NotificationService handles notification delivery
type NotificationService struct {
	channelRepo ChannelRepository
	config      NotificationConfig
}

// ChannelRepository defines the interface for notification channel persistence
type ChannelRepository interface {
	CreateChannel(ctx context.Context, channel *models.NotificationChannel) error
	GetChannel(ctx context.Context, id uuid.UUID) (*models.NotificationChannel, error)
	UpdateChannel(ctx context.Context, channel *models.NotificationChannel) error
	DeleteChannel(ctx context.Context, id uuid.UUID) error
	ListChannels(ctx context.Context, filter ChannelFilter) ([]*models.NotificationChannel, error)
}

// NotificationConfig holds configuration for the notification service
type NotificationConfig struct {
	MaxChannelsPerUser      int           `json:"max_channels_per_user"`
	MaxNotificationsPerHour int           `json:"max_notifications_per_hour"`
	NotificationTimeout     time.Duration `json:"notification_timeout"`
	EnableRetry             bool          `json:"enable_retry"`
	MaxRetryAttempts        int           `json:"max_retry_attempts"`
	RetryDelay              time.Duration `json:"retry_delay"`
	EnableLogging           bool          `json:"enable_logging"`
	EnableMetrics           bool          `json:"enable_metrics"`
}

// ChannelFilter defines filters for channel queries
type ChannelFilter struct {
	UserID *uuid.UUID            `json:"user_id,omitempty"`
	Type   *models.ChannelType   `json:"type,omitempty"`
	Status *models.ChannelStatus `json:"status,omitempty"`
	Tags   []string              `json:"tags,omitempty"`
	Search string                `json:"search,omitempty"`
	Limit  int                   `json:"limit,omitempty"`
	Offset int                   `json:"offset,omitempty"`
}

// NotificationRequest represents a notification request
type NotificationRequest struct {
	ChannelID  uuid.UUID              `json:"channel_id"`
	AlertID    uuid.UUID              `json:"alert_id"`
	RuleID     uuid.UUID              `json:"rule_id"`
	UserID     uuid.UUID              `json:"user_id"`
	Type       models.ChannelType     `json:"type"`
	Recipients []string               `json:"recipients"`
	Subject    string                 `json:"subject"`
	Message    string                 `json:"message"`
	Data       map[string]interface{} `json:"data"`
	Priority   string                 `json:"priority"`
	Template   string                 `json:"template"`
	Variables  map[string]interface{} `json:"variables"`
}

// NotificationResult represents the result of a notification
type NotificationResult struct {
	NotificationID uuid.UUID                 `json:"notification_id"`
	ChannelID      uuid.UUID                 `json:"channel_id"`
	Status         models.NotificationStatus `json:"status"`
	SentAt         time.Time                 `json:"sent_at"`
	ErrorMessage   string                    `json:"error_message,omitempty"`
	RetryCount     int                       `json:"retry_count"`
	Metadata       map[string]interface{}    `json:"metadata"`
}

// NewNotificationService creates a new notification service
func NewNotificationService(
	channelRepo ChannelRepository,
	config NotificationConfig,
) *NotificationService {
	return &NotificationService{
		channelRepo: channelRepo,
		config:      config,
	}
}

// CreateChannel creates a new notification channel
func (ns *NotificationService) CreateChannel(ctx context.Context, userID uuid.UUID, req *models.NotificationChannelRequest) (*models.NotificationChannel, error) {
	// Check channel limit
	if err := ns.checkChannelLimit(ctx, userID); err != nil {
		return nil, fmt.Errorf("channel limit exceeded: %w", err)
	}

	// Validate the channel
	if err := ns.validateChannel(req); err != nil {
		return nil, fmt.Errorf("channel validation failed: %w", err)
	}

	// Create the channel
	channel := &models.NotificationChannel{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Config:      req.Config,
		Status:      models.ChannelStatusActive,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save the channel
	if err := ns.channelRepo.CreateChannel(ctx, channel); err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	// Test the channel
	if err := ns.testChannel(ctx, channel); err != nil {
		log.Printf("Channel test failed for %s: %v", channel.ID, err)
		// Don't fail the creation, just log the test failure
	}

	return channel, nil
}

// GetChannel retrieves a channel by ID
func (ns *NotificationService) GetChannel(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*models.NotificationChannel, error) {
	channel, err := ns.channelRepo.GetChannel(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	// Check ownership
	if channel.UserID != userID {
		return nil, fmt.Errorf("channel not found or access denied")
	}

	return channel, nil
}

// UpdateChannel updates an existing channel
func (ns *NotificationService) UpdateChannel(ctx context.Context, userID uuid.UUID, id uuid.UUID, req *models.NotificationChannelRequest) (*models.NotificationChannel, error) {
	// Get existing channel
	channel, err := ns.channelRepo.GetChannel(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	// Check ownership
	if channel.UserID != userID {
		return nil, fmt.Errorf("channel not found or access denied")
	}

	// Validate the channel
	if err := ns.validateChannel(req); err != nil {
		return nil, fmt.Errorf("channel validation failed: %w", err)
	}

	// Update channel fields
	channel.Name = req.Name
	channel.Description = req.Description
	channel.Type = req.Type
	channel.Config = req.Config
	channel.Tags = req.Tags
	channel.UpdatedAt = time.Now()

	// Save the updated channel
	if err := ns.channelRepo.UpdateChannel(ctx, channel); err != nil {
		return nil, fmt.Errorf("failed to update channel: %w", err)
	}

	// Test the updated channel
	if err := ns.testChannel(ctx, channel); err != nil {
		log.Printf("Channel test failed for %s: %v", channel.ID, err)
		// Don't fail the update, just log the test failure
	}

	return channel, nil
}

// DeleteChannel deletes a channel
func (ns *NotificationService) DeleteChannel(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	// Get channel to check ownership
	channel, err := ns.channelRepo.GetChannel(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}

	// Check ownership
	if channel.UserID != userID {
		return fmt.Errorf("channel not found or access denied")
	}

	// Delete the channel
	if err := ns.channelRepo.DeleteChannel(ctx, id); err != nil {
		return fmt.Errorf("failed to delete channel: %w", err)
	}

	return nil
}

// ListChannels lists channels with optional filtering
func (ns *NotificationService) ListChannels(ctx context.Context, userID uuid.UUID, filter ChannelFilter) ([]*models.NotificationChannel, error) {
	// Set user ID filter
	filter.UserID = &userID

	channels, err := ns.channelRepo.ListChannels(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list channels: %w", err)
	}
	return channels, nil
}

// TestChannel tests a notification channel
func (ns *NotificationService) TestChannel(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	channel, err := ns.channelRepo.GetChannel(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}

	// Check ownership
	if channel.UserID != userID {
		return fmt.Errorf("channel not found or access denied")
	}

	// Test the channel
	if err := ns.testChannel(ctx, channel); err != nil {
		return fmt.Errorf("channel test failed: %w", err)
	}

	return nil
}

// SendNotification sends a notification via a channel
func (ns *NotificationService) SendNotification(ctx context.Context, req *NotificationRequest) (*NotificationResult, error) {
	// Get the channel
	channel, err := ns.channelRepo.GetChannel(ctx, req.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	// Check if channel is active
	if channel.Status != models.ChannelStatusActive {
		return nil, fmt.Errorf("channel is not active")
	}

	// Create notification result
	result := &NotificationResult{
		NotificationID: uuid.New(),
		ChannelID:      req.ChannelID,
		Status:         models.NotificationStatusPending,
		RetryCount:     0,
		Metadata:       make(map[string]interface{}),
	}

	// Send notification based on channel type
	switch channel.Type {
	case models.ChannelTypeEmail:
		result, err = ns.sendEmailNotification(ctx, channel, req)
	case models.ChannelTypeSlack:
		result, err = ns.sendSlackNotification(ctx, channel, req)
	case models.ChannelTypeWebhook:
		result, err = ns.sendWebhookNotification(ctx, channel, req)
	case models.ChannelTypeSMS:
		result, err = ns.sendSMSNotification(ctx, channel, req)
	case models.ChannelTypeDashboard:
		result, err = ns.sendDashboardNotification(ctx, channel, req)
	default:
		return nil, fmt.Errorf("unsupported channel type: %s", channel.Type)
	}

	if err != nil {
		result.Status = models.NotificationStatusFailed
		result.ErrorMessage = err.Error()
		return result, err
	}

	result.Status = models.NotificationStatusSent
	result.SentAt = time.Now()

	// Log notification
	if ns.config.EnableLogging {
		log.Printf("Notification sent via %s channel %s for alert %s", channel.Type, channel.ID, req.AlertID)
	}

	return result, nil
}

// sendEmailNotification sends an email notification
func (ns *NotificationService) sendEmailNotification(ctx context.Context, channel *models.NotificationChannel, req *NotificationRequest) (*NotificationResult, error) {
	// Parse email configuration
	config, err := ns.parseEmailConfig(channel.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse email config: %w", err)
	}

	// Simulate email sending
	// In a real implementation, this would use an email service like SendGrid, SES, etc.
	time.Sleep(100 * time.Millisecond) // Simulate sending time

	result := &NotificationResult{
		NotificationID: uuid.New(),
		ChannelID:      req.ChannelID,
		Status:         models.NotificationStatusSent,
		SentAt:         time.Now(),
		RetryCount:     0,
		Metadata: map[string]interface{}{
			"email_provider":  config.Provider,
			"recipient_count": len(req.Recipients),
			"subject":         req.Subject,
		},
	}

	// Log email details
	if ns.config.EnableLogging {
		log.Printf("Email sent to %v with subject: %s", req.Recipients, req.Subject)
	}

	return result, nil
}

// sendSlackNotification sends a Slack notification
func (ns *NotificationService) sendSlackNotification(ctx context.Context, channel *models.NotificationChannel, req *NotificationRequest) error {
	// Parse Slack configuration
	config, err := ns.parseSlackConfig(channel.Config)
	if err != nil {
		return fmt.Errorf("failed to parse Slack config: %w", err)
	}

	// Simulate Slack message sending
	// In a real implementation, this would use the Slack API
	time.Sleep(50 * time.Millisecond) // Simulate sending time

	// Log Slack details
	if ns.config.EnableLogging {
		log.Printf("Slack message sent to channel %s: %s", config.Channel, req.Message)
	}

	return nil
}

// sendWebhookNotification sends a webhook notification
func (ns *NotificationService) sendWebhookNotification(ctx context.Context, channel *models.NotificationChannel, req *NotificationRequest) error {
	// Parse webhook configuration
	config, err := ns.parseWebhookConfig(channel.Config)
	if err != nil {
		return fmt.Errorf("failed to parse webhook config: %w", err)
	}

	// Simulate webhook call
	// In a real implementation, this would make an HTTP POST request
	time.Sleep(75 * time.Millisecond) // Simulate sending time

	// Log webhook details
	if ns.config.EnableLogging {
		log.Printf("Webhook called at %s with payload size: %d", config.URL, len(req.Message))
	}

	return nil
}

// sendSMSNotification sends an SMS notification
func (ns *NotificationService) sendSMSNotification(ctx context.Context, channel *models.NotificationChannel, req *NotificationRequest) error {
	// Parse SMS configuration
	config, err := ns.parseSMSConfig(channel.Config)
	if err != nil {
		return fmt.Errorf("failed to parse SMS config: %w", err)
	}

	// Simulate SMS sending
	// In a real implementation, this would use an SMS service like Twilio, SNS, etc.
	time.Sleep(200 * time.Millisecond) // Simulate sending time

	// Log SMS details
	if ns.config.EnableLogging {
		log.Printf("SMS sent to %v: %s", req.Recipients, req.Message)
	}

	return nil
}

// sendDashboardNotification sends a dashboard notification
func (ns *NotificationService) sendDashboardNotification(ctx context.Context, channel *models.NotificationChannel, req *NotificationRequest) error {
	// Parse dashboard configuration
	config, err := ns.parseDashboardConfig(channel.Config)
	if err != nil {
		return fmt.Errorf("failed to parse dashboard config: %w", err)
	}

	// Simulate dashboard notification
	// In a real implementation, this would update the dashboard or send via WebSocket
	time.Sleep(25 * time.Millisecond) // Simulate sending time

	// Log dashboard details
	if ns.config.EnableLogging {
		log.Printf("Dashboard notification sent to %s: %s", config.DashboardID, req.Message)
	}

	return nil
}

// testChannel tests a notification channel
func (ns *NotificationService) testChannel(ctx context.Context, channel *models.NotificationChannel) error {
	// Create test notification request
	testReq := &NotificationRequest{
		ChannelID:  channel.ID,
		Type:       channel.Type,
		Subject:    "Test Notification",
		Message:    "This is a test notification to verify the channel configuration.",
		Priority:   "low",
		Recipients: []string{"test@example.com"},
		Data:       map[string]interface{}{"test": true},
	}

	// Send test notification
	_, err := ns.SendNotification(ctx, testReq)
	if err != nil {
		return fmt.Errorf("test notification failed: %w", err)
	}

	return nil
}

// checkChannelLimit checks if the user has reached the channel limit
func (ns *NotificationService) checkChannelLimit(ctx context.Context, userID uuid.UUID) error {
	if ns.config.MaxChannelsPerUser <= 0 {
		return nil // No limit
	}

	filter := ChannelFilter{
		UserID: &userID,
		Limit:  1,
	}

	channels, err := ns.channelRepo.ListChannels(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to check channel limit: %w", err)
	}

	if len(channels) >= ns.config.MaxChannelsPerUser {
		return fmt.Errorf("channel limit exceeded: %d channels", ns.config.MaxChannelsPerUser)
	}

	return nil
}

// validateChannel validates a channel request
func (ns *NotificationService) validateChannel(req *models.NotificationChannelRequest) error {
	if req.Name == "" {
		return fmt.Errorf("channel name is required")
	}

	if req.Type == "" {
		return fmt.Errorf("channel type is required")
	}

	if req.Config == nil {
		return fmt.Errorf("channel configuration is required")
	}

	return nil
}

// Configuration parsing structures
type EmailConfig struct {
	Provider     string `json:"provider"` // sendgrid, ses, smtp
	APIKey       string `json:"api_key"`
	FromEmail    string `json:"from_email"`
	FromName     string `json:"from_name"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
}

type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel"`
	Username   string `json:"username"`
	IconEmoji  string `json:"icon_emoji"`
}

type WebhookConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Secret  string            `json:"secret"`
}

type SMSConfig struct {
	Provider string `json:"provider"` // twilio, sns, etc.
	APIKey   string `json:"api_key"`
	From     string `json:"from"`
}

type DashboardConfig struct {
	DashboardID string `json:"dashboard_id"`
	WidgetID    string `json:"widget_id"`
}

// parseEmailConfig parses email configuration from JSONB
func (ns *NotificationService) parseEmailConfig(config models.JSONB) (*EmailConfig, error) {
	var emailConfig EmailConfig
	if err := config.Unmarshal(&emailConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal email config: %w", err)
	}
	return &emailConfig, nil
}

// parseSlackConfig parses Slack configuration from JSONB
func (ns *NotificationService) parseSlackConfig(config models.JSONB) (*SlackConfig, error) {
	var slackConfig SlackConfig
	if err := config.Unmarshal(&slackConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Slack config: %w", err)
	}
	return &slackConfig, nil
}

// parseWebhookConfig parses webhook configuration from JSONB
func (ns *NotificationService) parseWebhookConfig(config models.JSONB) (*WebhookConfig, error) {
	var webhookConfig WebhookConfig
	if err := config.Unmarshal(&webhookConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook config: %w", err)
	}
	return &webhookConfig, nil
}

// parseSMSConfig parses SMS configuration from JSONB
func (ns *NotificationService) parseSMSConfig(config models.JSONB) (*SMSConfig, error) {
	var smsConfig SMSConfig
	if err := config.Unmarshal(&smsConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SMS config: %w", err)
	}
	return &smsConfig, nil
}

// parseDashboardConfig parses dashboard configuration from JSONB
func (ns *NotificationService) parseDashboardConfig(config models.JSONB) (*DashboardConfig, error) {
	var dashboardConfig DashboardConfig
	if err := config.Unmarshal(&dashboardConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dashboard config: %w", err)
	}
	return &dashboardConfig, nil
}
