package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

// Config represents the complete DriftMgr configuration
type Config struct {
	Provider    string                    `yaml:"provider"`
	Regions     []string                  `yaml:"regions"`
	Credentials map[string]string         `yaml:"credentials,omitempty"`
	Settings    Settings                  `yaml:"settings"`
	Providers   map[string]ProviderConfig `yaml:"providers,omitempty"`
	Discovery   DiscoveryConfig           `yaml:"discovery,omitempty"`
}

// Settings represents application settings
type Settings struct {
	AutoDiscovery   bool                 `yaml:"auto_discovery"`
	ParallelWorkers int                  `yaml:"parallel_workers"`
	CacheTTL        string               `yaml:"cache_ttl"`
	DriftDetection  DriftSettings        `yaml:"drift_detection"`
	Remediation     RemediationSettings  `yaml:"remediation"`
	Database        DatabaseSettings     `yaml:"database"`
	Logging         LoggingSettings      `yaml:"logging"`
	Notifications   NotificationSettings `yaml:"notifications"`
}

// DriftSettings represents drift detection settings
type DriftSettings struct {
	Enabled  bool   `yaml:"enabled"`
	Interval string `yaml:"interval"`
	Severity string `yaml:"severity"`
}

// RemediationSettings represents remediation settings
type RemediationSettings struct {
	Enabled          bool `yaml:"enabled"`
	DryRun           bool `yaml:"dry_run"`
	ApprovalRequired bool `yaml:"approval_required"`
	MaxRetries       int  `yaml:"max_retries"`
}

// DatabaseSettings represents database settings
type DatabaseSettings struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
	Backup  bool   `yaml:"backup"`
}

// LoggingSettings represents logging settings
type LoggingSettings struct {
	Level  string `yaml:"level"`
	File   string `yaml:"file"`
	Format string `yaml:"format"`
}

// NotificationSettings represents notification settings
type NotificationSettings struct {
	Enabled  bool              `yaml:"enabled"`
	Channels []string          `yaml:"channels"`
	Webhooks map[string]string `yaml:"webhooks,omitempty"`
	Email    EmailSettings     `yaml:"email,omitempty"`
	Slack    SlackSettings     `yaml:"slack,omitempty"`
}

// EmailSettings represents email notification settings
type EmailSettings struct {
	Enabled  bool     `yaml:"enabled"`
	SMTPHost string   `yaml:"smtp_host"`
	SMTPPort int      `yaml:"smtp_port"`
	From     string   `yaml:"from"`
	To       []string `yaml:"to"`
}

// SlackSettings represents Slack notification settings
type SlackSettings struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
	Channel    string `yaml:"channel"`
	Username   string `yaml:"username"`
}

// ProviderConfig represents provider-specific configuration
type ProviderConfig struct {
	Type          string            `yaml:"type,omitempty"`
	Region        string            `yaml:"region,omitempty"`
	Enabled       bool              `yaml:"enabled,omitempty"`
	Regions       []string          `yaml:"regions"`
	Credentials   map[string]string `yaml:"credentials,omitempty"`
	ResourceTypes []string          `yaml:"resource_types,omitempty"`
	ExcludeTags   map[string]string `yaml:"exclude_tags,omitempty"`
	IncludeTags   map[string]string `yaml:"include_tags,omitempty"`
}

// Manager manages configuration with hot reload capability
type Manager struct {
	config     *Config
	configPath string
	mu         sync.RWMutex
	watcher    *fsnotify.Watcher
	callbacks  []func(*Config)
	stopCh     chan struct{}
}

// NewManager creates a new configuration manager
func NewManager(configPath string) (*Manager, error) {
	// Expand path if needed
	configPath = expandPath(configPath)

	m := &Manager{
		configPath: configPath,
		callbacks:  []func(*Config){},
		stopCh:     make(chan struct{}),
	}

	// Load initial configuration
	if err := m.Load(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Set up file watcher for hot reload
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return m, nil // Return manager without watcher
	}

	m.watcher = watcher

	// Watch the configuration file
	if err := watcher.Add(configPath); err != nil {
		watcher.Close()
		m.watcher = nil
		return m, nil // Return manager without watcher
	}

	// Start watching for changes
	go m.watchChanges()

	return m, nil
}

// Load loads or reloads the configuration from file
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		// Use default configuration if file doesn't exist
		m.config = m.defaultConfig()
	} else {
		// Read configuration file
		data, err := ioutil.ReadFile(m.configPath)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}

		// Parse YAML
		var config Config
		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}

		m.config = &config
	}

	// Apply defaults
	m.applyDefaults(m.config)

	// Apply environment variable overrides
	m.applyEnvironmentOverrides(m.config)

	// Validate configuration
	if err := m.validate(m.config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return nil
}

// Get returns the current configuration
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// GetSetting returns a specific setting value
func (m *Manager) GetSetting(path string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Simple path resolution (could be enhanced)
	switch path {
	case "provider":
		return m.config.Provider
	case "regions":
		return m.config.Regions
	case "settings.auto_discovery":
		return m.config.Settings.AutoDiscovery
	case "settings.parallel_workers":
		return m.config.Settings.ParallelWorkers
	case "settings.drift_detection.enabled":
		return m.config.Settings.DriftDetection.Enabled
	case "settings.remediation.enabled":
		return m.config.Settings.Remediation.Enabled
	case "settings.database.enabled":
		return m.config.Settings.Database.Enabled
	default:
		return nil
	}
}

// Set updates a configuration value
func (m *Manager) Set(path string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update the configuration in memory
	switch path {
	case "provider":
		if v, ok := value.(string); ok {
			m.config.Provider = v
		}
	case "settings.auto_discovery":
		if v, ok := value.(bool); ok {
			m.config.Settings.AutoDiscovery = v
		}
	case "settings.parallel_workers":
		if v, ok := value.(int); ok {
			m.config.Settings.ParallelWorkers = v
		}
	// Add more cases as needed
	default:
		return fmt.Errorf("unknown configuration path: %s", path)
	}

	// Save to file
	return m.save()
}

// Save saves the current configuration to file
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.save()
}

func (m *Manager) save() error {
	// Marshal configuration to YAML
	data, err := yaml.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write to file
	if err := ioutil.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// OnChange registers a callback for configuration changes
func (m *Manager) OnChange(callback func(*Config)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, callback)
}

// watchChanges watches for configuration file changes
func (m *Manager) watchChanges() {
	if m.watcher == nil {
		return
	}

	defer m.watcher.Close()

	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				// Configuration file was modified
				fmt.Println("Configuration file changed, reloading...")

				// Reload configuration
				if err := m.Load(); err != nil {
					fmt.Printf("Failed to reload configuration: %v\n", err)
					continue
				}

				// Notify callbacks
				m.mu.RLock()
				config := m.config
				callbacks := m.callbacks
				m.mu.RUnlock()

				for _, callback := range callbacks {
					callback(config)
				}
			}

		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Configuration watcher error: %v\n", err)

		case <-m.stopCh:
			return
		}
	}
}

// Stop stops the configuration manager
func (m *Manager) Stop() {
	close(m.stopCh)
	if m.watcher != nil {
		m.watcher.Close()
	}
}

// defaultConfig returns the default configuration
func (m *Manager) defaultConfig() *Config {
	return &Config{
		Provider: "aws",
		Regions:  []string{"us-east-1"},
		Settings: Settings{
			AutoDiscovery:   true,
			ParallelWorkers: 10,
			CacheTTL:        "1h",
			DriftDetection: DriftSettings{
				Enabled:  true,
				Interval: "15m",
				Severity: "medium",
			},
			Remediation: RemediationSettings{
				Enabled:          false,
				DryRun:           true,
				ApprovalRequired: true,
				MaxRetries:       3,
			},
			Database: DatabaseSettings{
				Enabled: true,
				Path:    "~/.driftmgr/driftmgr.db",
				Backup:  true,
			},
			Logging: LoggingSettings{
				Level:  "info",
				File:   "~/.driftmgr/driftmgr.log",
				Format: "json",
			},
			Notifications: NotificationSettings{
				Enabled:  false,
				Channels: []string{},
			},
		},
	}
}

// applyDefaults applies default values to missing configuration fields
func (m *Manager) applyDefaults(config *Config) {
	defaults := m.defaultConfig()

	if config.Provider == "" {
		config.Provider = defaults.Provider
	}

	if len(config.Regions) == 0 {
		config.Regions = defaults.Regions
	}

	if config.Settings.ParallelWorkers == 0 {
		config.Settings.ParallelWorkers = defaults.Settings.ParallelWorkers
	}

	if config.Settings.CacheTTL == "" {
		config.Settings.CacheTTL = defaults.Settings.CacheTTL
	}

	if config.Settings.DriftDetection.Interval == "" {
		config.Settings.DriftDetection.Interval = defaults.Settings.DriftDetection.Interval
	}

	if config.Settings.Database.Path == "" {
		config.Settings.Database.Path = defaults.Settings.Database.Path
	}

	if config.Settings.Logging.Level == "" {
		config.Settings.Logging.Level = defaults.Settings.Logging.Level
	}

	if config.Settings.Logging.Format == "" {
		config.Settings.Logging.Format = defaults.Settings.Logging.Format
	}
}

// validate validates the configuration
func (m *Manager) validate(config *Config) error {
	// Validate provider
	validProviders := []string{"aws", "azure", "gcp", "digitalocean", "multi"}
	valid := false
	for _, p := range validProviders {
		if config.Provider == p {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid provider: %s", config.Provider)
	}

	// Validate parallel workers
	if config.Settings.ParallelWorkers < 1 || config.Settings.ParallelWorkers > 100 {
		return fmt.Errorf("parallel_workers must be between 1 and 100")
	}

	// Validate cache TTL
	if _, err := time.ParseDuration(config.Settings.CacheTTL); err != nil {
		return fmt.Errorf("invalid cache_ttl: %v", err)
	}

	// Validate drift detection interval
	if _, err := time.ParseDuration(config.Settings.DriftDetection.Interval); err != nil {
		return fmt.Errorf("invalid drift_detection.interval: %v", err)
	}

	return nil
}

// applyEnvironmentOverrides applies environment variable overrides to configuration
func (m *Manager) applyEnvironmentOverrides(config *Config) {
	// Provider override
	if provider := os.Getenv("DRIFTMGR_PROVIDER"); provider != "" {
		config.Provider = provider
	}

	// Auto-discovery override
	if autoDiscover := os.Getenv("DRIFTMGR_AUTO_DISCOVERY"); autoDiscover != "" {
		config.Settings.AutoDiscovery = autoDiscover == "true" || autoDiscover == "1"
	}

	// Parallel workers override
	if workers := os.Getenv("DRIFTMGR_PARALLEL_WORKERS"); workers != "" {
		if w, err := strconv.Atoi(workers); err == nil {
			config.Settings.ParallelWorkers = w
		}
	}

	// Cache TTL override
	if ttl := os.Getenv("DRIFTMGR_CACHE_TTL"); ttl != "" {
		config.Settings.CacheTTL = ttl
	}

	// Drift detection enabled override
	if driftEnabled := os.Getenv("DRIFTMGR_DRIFT_ENABLED"); driftEnabled != "" {
		config.Settings.DriftDetection.Enabled = driftEnabled == "true" || driftEnabled == "1"
	}

	// Drift detection interval override
	if interval := os.Getenv("DRIFTMGR_DRIFT_INTERVAL"); interval != "" {
		config.Settings.DriftDetection.Interval = interval
	}

	// Remediation enabled override
	if remEnabled := os.Getenv("DRIFTMGR_REMEDIATION_ENABLED"); remEnabled != "" {
		config.Settings.Remediation.Enabled = remEnabled == "true" || remEnabled == "1"
	}

	// Remediation dry run override
	if dryRun := os.Getenv("DRIFTMGR_REMEDIATION_DRY_RUN"); dryRun != "" {
		config.Settings.Remediation.DryRun = dryRun == "true" || dryRun == "1"
	}

	// Database enabled override
	if dbEnabled := os.Getenv("DRIFTMGR_DATABASE_ENABLED"); dbEnabled != "" {
		config.Settings.Database.Enabled = dbEnabled == "true" || dbEnabled == "1"
	}

	// Database path override
	if dbPath := os.Getenv("DRIFTMGR_DATABASE_PATH"); dbPath != "" {
		config.Settings.Database.Path = dbPath
	}

	// Logging level override
	if logLevel := os.Getenv("DRIFTMGR_LOG_LEVEL"); logLevel != "" {
		config.Settings.Logging.Level = logLevel
	}

	// Logging file override
	if logFile := os.Getenv("DRIFTMGR_LOG_FILE"); logFile != "" {
		config.Settings.Logging.File = logFile
	}

	// Notification enabled override
	if notifEnabled := os.Getenv("DRIFTMGR_NOTIFICATIONS_ENABLED"); notifEnabled != "" {
		config.Settings.Notifications.Enabled = notifEnabled == "true" || notifEnabled == "1"
	}

	// Slack webhook override
	if slackWebhook := os.Getenv("DRIFTMGR_SLACK_WEBHOOK"); slackWebhook != "" {
		config.Settings.Notifications.Slack.WebhookURL = slackWebhook
		config.Settings.Notifications.Slack.Enabled = true
	}

	// Email settings overrides
	if smtpHost := os.Getenv("DRIFTMGR_SMTP_HOST"); smtpHost != "" {
		config.Settings.Notifications.Email.SMTPHost = smtpHost
		config.Settings.Notifications.Email.Enabled = true
	}

	if smtpPort := os.Getenv("DRIFTMGR_SMTP_PORT"); smtpPort != "" {
		if port, err := strconv.Atoi(smtpPort); err == nil {
			config.Settings.Notifications.Email.SMTPPort = port
		}
	}

	// AWS credentials override
	if awsProfile := os.Getenv("AWS_PROFILE"); awsProfile != "" {
		if config.Credentials == nil {
			config.Credentials = make(map[string]string)
		}
		config.Credentials["aws_profile"] = awsProfile
	}

	// Azure credentials override
	if azureSubID := os.Getenv("AZURE_SUBSCRIPTION_ID"); azureSubID != "" {
		if config.Credentials == nil {
			config.Credentials = make(map[string]string)
		}
		config.Credentials["azure_subscription_id"] = azureSubID
	}

	// GCP project override
	if gcpProject := os.Getenv("GCP_PROJECT"); gcpProject != "" {
		if config.Credentials == nil {
			config.Credentials = make(map[string]string)
		}
		config.Credentials["gcp_project"] = gcpProject
	}

	// DigitalOcean token override
	if doToken := os.Getenv("DIGITALOCEAN_TOKEN"); doToken != "" {
		if config.Credentials == nil {
			config.Credentials = make(map[string]string)
		}
		config.Credentials["digitalocean_token"] = doToken
	}
}

func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}
