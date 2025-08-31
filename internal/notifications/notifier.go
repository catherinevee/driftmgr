package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"sync"
	"text/template"
	"time"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	TypeInfo     NotificationType = "info"
	TypeWarning  NotificationType = "warning"
	TypeError    NotificationType = "error"
	TypeSuccess  NotificationType = "success"
	TypeAlert    NotificationType = "alert"
)

// Channel represents a notification channel
type Channel string

const (
	ChannelEmail   Channel = "email"
	ChannelSlack   Channel = "slack"
	ChannelWebhook Channel = "webhook"
	ChannelConsole Channel = "console"
)

// Notification represents a notification to be sent
type Notification struct {
	ID        string                 `json:"id"`
	Type      NotificationType       `json:"type"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Severity  string                 `json:"severity"`
	Metadata  map[string]interface{} `json:"metadata"`
	Channels  []Channel              `json:"channels"`
	CreatedAt time.Time              `json:"created_at"`
	SentAt    *time.Time             `json:"sent_at"`
	Status    string                 `json:"status"` // pending, sent, failed
	Error     string                 `json:"error,omitempty"`
}

// EmailConfig represents email notification configuration
type EmailConfig struct {
	Enabled  bool     `json:"enabled"`
	SMTPHost string   `json:"smtp_host"`
	SMTPPort int      `json:"smtp_port"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	From     string   `json:"from"`
	To       []string `json:"to"`
	TLS      bool     `json:"tls"`
}

// SlackConfig represents Slack notification configuration
type SlackConfig struct {
	Enabled    bool   `json:"enabled"`
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel"`
	Username   string `json:"username"`
	IconEmoji  string `json:"icon_emoji"`
}

// WebhookConfig represents webhook notification configuration
type WebhookConfig struct {
	Enabled bool              `json:"enabled"`
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
}

// Config represents the notification system configuration
type Config struct {
	Enabled  bool                     `json:"enabled"`
	Email    EmailConfig              `json:"email"`
	Slack    SlackConfig              `json:"slack"`
	Webhooks map[string]WebhookConfig `json:"webhooks"`
	Rules    []NotificationRule       `json:"rules"`
}

// NotificationRule represents a rule for sending notifications
type NotificationRule struct {
	Name       string           `json:"name"`
	Enabled    bool             `json:"enabled"`
	EventTypes []string         `json:"event_types"`
	Severity   []string         `json:"severity"`
	Channels   []Channel        `json:"channels"`
	Filters    map[string]string `json:"filters"`
}

// Notifier handles sending notifications through various channels
type Notifier struct {
	config       *Config
	queue        chan *Notification
	history      []*Notification
	historyMu    sync.RWMutex
	templates    map[string]*template.Template
	httpClient   *http.Client
	persistence  interface{} // Can be *api.PersistenceManager
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

// NewNotifier creates a new notification manager
func NewNotifier(config *Config) *Notifier {
	n := &Notifier{
		config:     config,
		queue:      make(chan *Notification, 100),
		history:    make([]*Notification, 0),
		templates:  make(map[string]*template.Template),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		stopCh:     make(chan struct{}),
	}
	
	// Load notification templates
	n.loadTemplates()
	
	// Start notification processor
	n.wg.Add(1)
	go n.processNotifications()
	
	return n
}

// Send sends a notification
func (n *Notifier) Send(ctx context.Context, notification *Notification) error {
	if !n.config.Enabled {
		return nil
	}
	
	// Set default values
	if notification.ID == "" {
		notification.ID = fmt.Sprintf("notif-%d", time.Now().UnixNano())
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now()
	}
	notification.Status = "pending"
	
	// Apply rules to determine channels
	if len(notification.Channels) == 0 {
		notification.Channels = n.getChannelsForNotification(notification)
	}
	
	// Queue the notification
	select {
	case n.queue <- notification:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("notification queue is full")
	}
}

// processNotifications processes queued notifications
func (n *Notifier) processNotifications() {
	defer n.wg.Done()
	
	for {
		select {
		case notification := <-n.queue:
			n.sendNotification(notification)
			
		case <-n.stopCh:
			// Process remaining notifications
			for len(n.queue) > 0 {
				notification := <-n.queue
				n.sendNotification(notification)
			}
			return
		}
	}
}

// sendNotification sends a notification through configured channels
func (n *Notifier) sendNotification(notification *Notification) {
	var errors []string
	success := false
	
	for _, channel := range notification.Channels {
		var err error
		
		switch channel {
		case ChannelEmail:
			err = n.sendEmail(notification)
		case ChannelSlack:
			err = n.sendSlack(notification)
		case ChannelWebhook:
			err = n.sendWebhook(notification, "default")
		case ChannelConsole:
			err = n.sendConsole(notification)
		}
		
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", channel, err))
		} else {
			success = true
		}
	}
	
	// Update notification status
	now := time.Now()
	notification.SentAt = &now
	
	if success {
		notification.Status = "sent"
	} else {
		notification.Status = "failed"
		notification.Error = strings.Join(errors, "; ")
	}
	
	// Store in history
	n.historyMu.Lock()
	n.history = append(n.history, notification)
	if len(n.history) > 1000 {
		n.history = n.history[len(n.history)-1000:]
	}
	n.historyMu.Unlock()
	
	// Persist if persistence is available
	if n.persistence != nil {
		// Save to database
	}
}

// sendEmail sends an email notification
func (n *Notifier) sendEmail(notification *Notification) error {
	if !n.config.Email.Enabled {
		return fmt.Errorf("email notifications disabled")
	}
	
	// Prepare email content
	subject := notification.Title
	body := n.formatEmailBody(notification)
	
	// Prepare message
	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: [DriftMgr] %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		n.config.Email.From,
		strings.Join(n.config.Email.To, ", "),
		subject,
		body,
	))
	
	// Send email
	auth := smtp.PlainAuth("", n.config.Email.Username, n.config.Email.Password, n.config.Email.SMTPHost)
	addr := fmt.Sprintf("%s:%d", n.config.Email.SMTPHost, n.config.Email.SMTPPort)
	
	err := smtp.SendMail(addr, auth, n.config.Email.From, n.config.Email.To, msg)
	return err
}

// sendSlack sends a Slack notification
func (n *Notifier) sendSlack(notification *Notification) error {
	if !n.config.Slack.Enabled {
		return fmt.Errorf("slack notifications disabled")
	}
	
	// Prepare Slack message
	color := n.getSlackColor(notification.Type)
	
	payload := map[string]interface{}{
		"channel":  n.config.Slack.Channel,
		"username": n.config.Slack.Username,
		"attachments": []map[string]interface{}{
			{
				"color":     color,
				"title":     notification.Title,
				"text":      notification.Message,
				"footer":    "DriftMgr",
				"ts":        notification.CreatedAt.Unix(),
				"fields":    n.formatSlackFields(notification),
			},
		},
	}
	
	if n.config.Slack.IconEmoji != "" {
		payload["icon_emoji"] = n.config.Slack.IconEmoji
	}
	
	// Send to Slack
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	
	resp, err := n.httpClient.Post(n.config.Slack.WebhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}
	
	return nil
}

// sendWebhook sends a webhook notification
func (n *Notifier) sendWebhook(notification *Notification, webhookName string) error {
	webhook, ok := n.config.Webhooks[webhookName]
	if !ok || !webhook.Enabled {
		return fmt.Errorf("webhook %s not configured or disabled", webhookName)
	}
	
	// Prepare webhook payload
	payload := map[string]interface{}{
		"notification": notification,
		"timestamp":    time.Now().Unix(),
		"source":       "driftmgr",
	}
	
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	
	// Create request
	req, err := http.NewRequest(webhook.Method, webhook.URL, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	
	// Add headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range webhook.Headers {
		req.Header.Set(key, value)
	}
	
	// Send request
	resp, err := n.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	
	return nil
}

// sendConsole sends a console notification (for debugging)
func (n *Notifier) sendConsole(notification *Notification) error {
	fmt.Printf("[%s] %s: %s - %s\n",
		notification.Type,
		notification.CreatedAt.Format(time.RFC3339),
		notification.Title,
		notification.Message,
	)
	return nil
}

// formatEmailBody formats the email body using templates
func (n *Notifier) formatEmailBody(notification *Notification) string {
	tmpl := n.templates["email"]
	if tmpl == nil {
		// Use default template
		return fmt.Sprintf(`
			<html>
			<body>
				<h2>%s</h2>
				<p>%s</p>
				<hr>
				<small>DriftMgr Notification - %s</small>
			</body>
			</html>
		`, notification.Title, notification.Message, notification.CreatedAt.Format(time.RFC3339))
	}
	
	var buf bytes.Buffer
	tmpl.Execute(&buf, notification)
	return buf.String()
}

// formatSlackFields formats fields for Slack attachment
func (n *Notifier) formatSlackFields(notification *Notification) []map[string]interface{} {
	fields := []map[string]interface{}{}
	
	if notification.Severity != "" {
		fields = append(fields, map[string]interface{}{
			"title": "Severity",
			"value": notification.Severity,
			"short": true,
		})
	}
	
	if notification.Type != "" {
		fields = append(fields, map[string]interface{}{
			"title": "Type",
			"value": string(notification.Type),
			"short": true,
		})
	}
	
	// Add metadata fields
	for key, value := range notification.Metadata {
		fields = append(fields, map[string]interface{}{
			"title": key,
			"value": fmt.Sprintf("%v", value),
			"short": true,
		})
	}
	
	return fields
}

// getSlackColor returns the color for Slack attachment based on notification type
func (n *Notifier) getSlackColor(notifType NotificationType) string {
	switch notifType {
	case TypeSuccess:
		return "good"
	case TypeWarning:
		return "warning"
	case TypeError, TypeAlert:
		return "danger"
	default:
		return "#36a64f"
	}
}

// getChannelsForNotification determines channels based on rules
func (n *Notifier) getChannelsForNotification(notification *Notification) []Channel {
	channels := []Channel{}
	
	for _, rule := range n.config.Rules {
		if !rule.Enabled {
			continue
		}
		
		// Check if rule matches
		if n.ruleMatches(rule, notification) {
			channels = append(channels, rule.Channels...)
		}
	}
	
	// Remove duplicates
	seen := make(map[Channel]bool)
	unique := []Channel{}
	for _, ch := range channels {
		if !seen[ch] {
			seen[ch] = true
			unique = append(unique, ch)
		}
	}
	
	// Default to console if no channels
	if len(unique) == 0 {
		unique = []Channel{ChannelConsole}
	}
	
	return unique
}

// ruleMatches checks if a rule matches a notification
func (n *Notifier) ruleMatches(rule NotificationRule, notification *Notification) bool {
	// Check event type
	if len(rule.EventTypes) > 0 {
		found := false
		for _, eventType := range rule.EventTypes {
			if eventType == string(notification.Type) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check severity
	if len(rule.Severity) > 0 {
		found := false
		for _, severity := range rule.Severity {
			if severity == notification.Severity {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check filters
	for key, value := range rule.Filters {
		if metaValue, ok := notification.Metadata[key]; ok {
			if fmt.Sprintf("%v", metaValue) != value {
				return false
			}
		} else {
			return false
		}
	}
	
	return true
}

// loadTemplates loads notification templates
func (n *Notifier) loadTemplates() {
	// Load default email template
	emailTmpl := `
	<html>
	<head>
		<style>
			body { font-family: Arial, sans-serif; }
			.header { background-color: #f0f0f0; padding: 20px; }
			.content { padding: 20px; }
			.footer { background-color: #f0f0f0; padding: 10px; font-size: 12px; }
		</style>
	</head>
	<body>
		<div class="header">
			<h2>{{.Title}}</h2>
		</div>
		<div class="content">
			<p>{{.Message}}</p>
			{{if .Metadata}}
			<h3>Details:</h3>
			<ul>
				{{range $key, $value := .Metadata}}
				<li><strong>{{$key}}:</strong> {{$value}}</li>
				{{end}}
			</ul>
			{{end}}
		</div>
		<div class="footer">
			DriftMgr Notification - {{.CreatedAt.Format "2006-01-02 15:04:05"}}
		</div>
	</body>
	</html>
	`
	
	n.templates["email"], _ = template.New("email").Parse(emailTmpl)
}

// GetHistory returns notification history
func (n *Notifier) GetHistory(limit int) []*Notification {
	n.historyMu.RLock()
	defer n.historyMu.RUnlock()
	
	if limit <= 0 || limit > len(n.history) {
		limit = len(n.history)
	}
	
	result := make([]*Notification, limit)
	copy(result, n.history[len(n.history)-limit:])
	return result
}

// UpdateConfig updates the notifier configuration
func (n *Notifier) UpdateConfig(config *Config) {
	n.config = config
}

// Stop stops the notifier
func (n *Notifier) Stop() {
	close(n.stopCh)
	n.wg.Wait()
}