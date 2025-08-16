package notification

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"gopkg.in/gomail.v2"
)

// EmailConfig represents email notification configuration
type EmailConfig struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	FromEmail    string `json:"from_email"`
	FromName     string `json:"from_name"`
	UseTLS       bool   `json:"use_tls"`
	UseSSL       bool   `json:"use_ssl"`
}

// EmailProvider implements the notification provider interface for email
type EmailProvider struct {
	config EmailConfig
	dialer *gomail.Dialer
}

// NewEmailProvider creates a new email notification provider
func NewEmailProvider(config EmailConfig) *EmailProvider {
	dialer := gomail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPUsername, config.SMTPPassword)

	if config.UseSSL {
		dialer.SSL = true
	} else if config.UseTLS {
		dialer.TLSConfig = nil // Will use default TLS config
	}

	return &EmailProvider{
		config: config,
		dialer: dialer,
	}
}

// NewEmailProviderFromEnv creates an email provider from environment variables
func NewEmailProviderFromEnv() *EmailProvider {
	config := EmailConfig{
		SMTPHost:     getEnvOrDefault("DRIFT_SMTP_HOST", "localhost"),
		SMTPPort:     getEnvAsIntOrDefault("DRIFT_SMTP_PORT", 587),
		SMTPUsername: getEnvOrDefault("DRIFT_SMTP_USERNAME", ""),
		SMTPPassword: getEnvOrDefault("DRIFT_SMTP_PASSWORD", ""),
		FromEmail:    getEnvOrDefault("DRIFT_FROM_EMAIL", "driftmgr@example.com"),
		FromName:     getEnvOrDefault("DRIFT_FROM_NAME", "DriftMgr"),
		UseTLS:       getEnvAsBoolOrDefault("DRIFT_SMTP_TLS", true),
		UseSSL:       getEnvAsBoolOrDefault("DRIFT_SMTP_SSL", false),
	}

	return NewEmailProvider(config)
}

// SendNotification sends an email notification
func (e *EmailProvider) SendNotification(req models.NotificationRequest) (models.NotificationResponse, error) {
	// Validate request
	if err := e.validateRequest(req); err != nil {
		return models.NotificationResponse{}, fmt.Errorf("invalid notification request: %w", err)
	}

	// Create email message
	msg := gomail.NewMessage()
	msg.SetHeader("From", fmt.Sprintf("%s <%s>", e.config.FromName, e.config.FromEmail))
	msg.SetHeader("To", strings.Join(req.Recipients, ","))
	msg.SetHeader("Subject", req.Subject)

	// Set priority header
	if req.Priority != "" {
		msg.SetHeader("X-Priority", e.getPriorityHeader(req.Priority))
	}

	// Generate email content
	htmlContent, textContent, err := e.generateEmailContent(req)
	if err != nil {
		return models.NotificationResponse{}, fmt.Errorf("failed to generate email content: %w", err)
	}

	msg.SetBody("text/html", htmlContent)
	msg.AddAlternative("text/plain", textContent)

	// Send email
	if err := e.dialer.DialAndSend(msg); err != nil {
		return models.NotificationResponse{}, fmt.Errorf("failed to send email: %w", err)
	}

	return models.NotificationResponse{
		Success:   true,
		MessageID: generateMessageID(),
		SentAt:    time.Now(),
	}, nil
}

// validateRequest validates the notification request
func (e *EmailProvider) validateRequest(req models.NotificationRequest) error {
	if req.Type != "email" {
		return fmt.Errorf("unsupported notification type: %s", req.Type)
	}

	if len(req.Recipients) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	for _, recipient := range req.Recipients {
		if !isValidEmail(recipient) {
			return fmt.Errorf("invalid email address: %s", recipient)
		}
	}

	if req.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if req.Message == "" {
		return fmt.Errorf("message is required")
	}

	return nil
}

// generateEmailContent generates HTML and text email content
func (e *EmailProvider) generateEmailContent(req models.NotificationRequest) (string, string, error) {
	// Parse message as template if it contains template syntax
	if strings.Contains(req.Message, "{{") {
		return e.generateTemplatedContent(req)
	}

	// Generate simple HTML content
	htmlContent := e.generateSimpleHTML(req)
	textContent := e.generateSimpleText(req)

	return htmlContent, textContent, nil
}

// generateTemplatedContent generates content using Go templates
func (e *EmailProvider) generateTemplatedContent(req models.NotificationRequest) (string, string, error) {
	// Parse HTML template
	htmlTmpl, err := template.New("email").Parse(e.getHTMLTemplate())
	if err != nil {
		return "", "", fmt.Errorf("failed to parse HTML template: %w", err)
	}

	// Parse text template
	textTmpl, err := template.New("email").Parse(e.getTextTemplate())
	if err != nil {
		return "", "", fmt.Errorf("failed to parse text template: %w", err)
	}

	// Template data
	data := map[string]interface{}{
		"Subject":   req.Subject,
		"Message":   req.Message,
		"Priority":  req.Priority,
		"Timestamp": time.Now().Format("2006-01-02 15:04:05 UTC"),
		"FromName":  e.config.FromName,
	}

	// Execute HTML template
	var htmlBuf bytes.Buffer
	if err := htmlTmpl.Execute(&htmlBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to execute HTML template: %w", err)
	}

	// Execute text template
	var textBuf bytes.Buffer
	if err := textTmpl.Execute(&textBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to execute text template: %w", err)
	}

	return htmlBuf.String(), textBuf.String(), nil
}

// generateSimpleHTML generates simple HTML email content
func (e *EmailProvider) generateSimpleHTML(req models.NotificationRequest) string {
	priorityClass := "normal"
	if req.Priority == "high" {
		priorityClass = "high"
	} else if req.Priority == "critical" {
		priorityClass = "critical"
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>%s</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #f8f9fa; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .content { background: white; padding: 20px; border-radius: 5px; }
        .priority-high { border-left: 4px solid #ffc107; }
        .priority-critical { border-left: 4px solid #dc3545; }
        .priority-normal { border-left: 4px solid #28a745; }
        .footer { margin-top: 20px; padding: 20px; background: #f8f9fa; border-radius: 5px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>%s</h2>
            <p><strong>From:</strong> %s</p>
            <p><strong>Priority:</strong> %s</p>
            <p><strong>Time:</strong> %s</p>
        </div>
        <div class="content priority-%s">
            %s
        </div>
        <div class="footer">
            <p>This message was sent by DriftMgr - Terraform Drift Detection & Remediation Tool</p>
        </div>
    </div>
</body>
</html>`,
		req.Subject,
		req.Subject,
		e.config.FromName,
		req.Priority,
		time.Now().Format("2006-01-02 15:04:05 UTC"),
		priorityClass,
		strings.ReplaceAll(req.Message, "\n", "<br>"),
	)
}

// generateSimpleText generates simple text email content
func (e *EmailProvider) generateSimpleText(req models.NotificationRequest) string {
	return fmt.Sprintf(`Subject: %s
From: %s
Priority: %s
Time: %s

%s

---
This message was sent by DriftMgr - Terraform Drift Detection & Remediation Tool`,
		req.Subject,
		e.config.FromName,
		req.Priority,
		time.Now().Format("2006-01-02 15:04:05 UTC"),
		req.Message,
	)
}

// getHTMLTemplate returns the HTML email template
func (e *EmailProvider) getHTMLTemplate() string {
	return `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.Subject}}</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #f8f9fa; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .content { background: white; padding: 20px; border-radius: 5px; }
        .footer { margin-top: 20px; padding: 20px; background: #f8f9fa; border-radius: 5px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>{{.Subject}}</h2>
            <p><strong>From:</strong> {{.FromName}}</p>
            <p><strong>Priority:</strong> {{.Priority}}</p>
            <p><strong>Time:</strong> {{.Timestamp}}</p>
        </div>
        <div class="content">
            {{.Message}}
        </div>
        <div class="footer">
            <p>This message was sent by DriftMgr - Terraform Drift Detection & Remediation Tool</p>
        </div>
    </div>
</body>
</html>`
}

// getTextTemplate returns the text email template
func (e *EmailProvider) getTextTemplate() string {
	return `Subject: {{.Subject}}
From: {{.FromName}}
Priority: {{.Priority}}
Time: {{.Timestamp}}

{{.Message}}

---
This message was sent by DriftMgr - Terraform Drift Detection & Remediation Tool`
}

// getPriorityHeader returns the email priority header value
func (e *EmailProvider) getPriorityHeader(priority string) string {
	switch priority {
	case "critical":
		return "1"
	case "high":
		return "2"
	case "normal":
		return "3"
	case "low":
		return "4"
	default:
		return "3"
	}
}

// isValidEmail validates email address format
func isValidEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// Helper functions for environment variables
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := fmt.Sscanf(value, "%d", &defaultValue); err == nil && intValue == 1 {
			return defaultValue
		}
	}
	return defaultValue
}

func getEnvAsBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}
