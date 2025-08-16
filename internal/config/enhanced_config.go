package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// EnhancedConfigManager provides advanced configuration management
type EnhancedConfigManager struct {
	mu           sync.RWMutex
	configPath   string
	config       *EnhancedConfig
	watchers     []ConfigWatcher
	lastModified time.Time
	ctx          context.Context
	cancel       context.CancelFunc
}

// ConfigWatcher interface for configuration change notifications
type ConfigWatcher interface {
	OnConfigChanged(config *EnhancedConfig) error
}

// EnhancedConfig represents the enhanced configuration structure
type EnhancedConfig struct {
	Environment string                 `yaml:"environment" validate:"required,oneof=development staging production"`
	Server      EnhancedServerConfig   `yaml:"server"`
	Database    EnhancedDatabaseConfig `yaml:"database"`
	Cloud       CloudConfig            `yaml:"cloud"`
	Security    EnhancedSecurityConfig `yaml:"security"`
	Monitoring  MonitoringConfig       `yaml:"monitoring"`
	Features    FeaturesConfig         `yaml:"features"`
	Custom      map[string]interface{} `yaml:"custom"`
}

// EnhancedServerConfig represents server configuration
type EnhancedServerConfig struct {
	Host           string        `yaml:"host" validate:"required"`
	Port           int           `yaml:"port" validate:"required,min=1,max=65535"`
	ReadTimeout    time.Duration `yaml:"read_timeout" validate:"required"`
	WriteTimeout   time.Duration `yaml:"write_timeout" validate:"required"`
	IdleTimeout    time.Duration `yaml:"idle_timeout" validate:"required"`
	MaxConnections int           `yaml:"max_connections" validate:"required,min=1"`
}

// EnhancedDatabaseConfig represents database configuration
type EnhancedDatabaseConfig struct {
	Type           string `yaml:"type" validate:"required,oneof=sqlite postgres mysql"`
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	Database       string `yaml:"database"`
	Username       string `yaml:"username"`
	Password       string `yaml:"password"`
	SSLMode        string `yaml:"ssl_mode"`
	MaxConnections int    `yaml:"max_connections" validate:"min=1"`
}

// CloudConfig represents cloud provider configuration
type CloudConfig struct {
	Providers map[string]ProviderConfig `yaml:"providers" validate:"required"`
	Default   string                    `yaml:"default" validate:"required"`
}

// ProviderConfig represents individual cloud provider configuration
type ProviderConfig struct {
	Enabled     bool                   `yaml:"enabled"`
	Credentials map[string]string      `yaml:"credentials"`
	Regions     []string               `yaml:"regions"`
	Settings    map[string]interface{} `yaml:"settings"`
}

// EnhancedSecurityConfig represents security configuration
type EnhancedSecurityConfig struct {
	JWTSecret        string         `yaml:"jwt_secret" validate:"required,min=32"`
	SessionSecret    string         `yaml:"session_secret" validate:"required,min=32"`
	TokenExpiry      time.Duration  `yaml:"token_expiry" validate:"required"`
	MaxLoginAttempts int            `yaml:"max_login_attempts" validate:"min=1"`
	PasswordPolicy   PasswordPolicy `yaml:"password_policy"`
}

// PasswordPolicy represents password policy configuration
type PasswordPolicy struct {
	MinLength      int  `yaml:"min_length" validate:"min=8"`
	RequireUpper   bool `yaml:"require_upper"`
	RequireLower   bool `yaml:"require_lower"`
	RequireNumber  bool `yaml:"require_number"`
	RequireSpecial bool `yaml:"require_special"`
}

// MonitoringConfig represents monitoring configuration
type MonitoringConfig struct {
	Enabled     bool              `yaml:"enabled"`
	LogLevel    string            `yaml:"log_level" validate:"oneof=debug info warn error"`
	MetricsPort int               `yaml:"metrics_port" validate:"min=1,max=65535"`
	HealthCheck HealthCheckConfig `yaml:"health_check"`
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Enabled          bool          `yaml:"enabled"`
	Interval         time.Duration `yaml:"interval" validate:"required"`
	Timeout          time.Duration `yaml:"timeout" validate:"required"`
	FailureThreshold int           `yaml:"failure_threshold" validate:"min=1"`
}

// FeaturesConfig represents feature flags configuration
type FeaturesConfig struct {
	EnhancedDiscovery     bool `yaml:"enhanced_discovery"`
	AIEnabled             bool `yaml:"ai_enabled"`
	AutoRemediation       bool `yaml:"auto_remediation"`
	RealTimeMonitoring    bool `yaml:"real_time_monitoring"`
	AdvancedVisualization bool `yaml:"advanced_visualization"`
}

// NewEnhancedConfigManager creates a new enhanced configuration manager
func NewEnhancedConfigManager(configPath string) (*EnhancedConfigManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	ecm := &EnhancedConfigManager{
		configPath: configPath,
		watchers:   make([]ConfigWatcher, 0),
		ctx:        ctx,
		cancel:     cancel,
	}

	// Load initial configuration
	if err := ecm.loadConfig(); err != nil {
		return nil, fmt.Errorf("failed to load initial config: %w", err)
	}

	// Start file watcher for hot reload
	go ecm.watchConfigFile()

	return ecm, nil
}

// GetConfig returns the current configuration
func (ecm *EnhancedConfigManager) GetConfig() *EnhancedConfig {
	ecm.mu.RLock()
	defer ecm.mu.RUnlock()
	return ecm.config
}

// UpdateConfig updates the configuration
func (ecm *EnhancedConfigManager) UpdateConfig(config *EnhancedConfig) error {
	ecm.mu.Lock()
	defer ecm.mu.Unlock()

	// Validate the new configuration
	if err := ecm.validateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Save to file
	if err := ecm.saveConfig(config); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Update in memory
	ecm.config = config

	// Notify watchers
	ecm.notifyWatchers(config)

	return nil
}

// AddWatcher adds a configuration watcher
func (ecm *EnhancedConfigManager) AddWatcher(watcher ConfigWatcher) {
	ecm.mu.Lock()
	defer ecm.mu.Unlock()
	ecm.watchers = append(ecm.watchers, watcher)
}

// RemoveWatcher removes a configuration watcher
func (ecm *EnhancedConfigManager) RemoveWatcher(watcher ConfigWatcher) {
	ecm.mu.Lock()
	defer ecm.mu.Unlock()

	for i, w := range ecm.watchers {
		if w == watcher {
			ecm.watchers = append(ecm.watchers[:i], ecm.watchers[i+1:]...)
			break
		}
	}
}

// GetEnvironment returns the current environment
func (ecm *EnhancedConfigManager) GetEnvironment() string {
	ecm.mu.RLock()
	defer ecm.mu.RUnlock()
	return ecm.config.Environment
}

// IsDevelopment returns true if running in development environment
func (ecm *EnhancedConfigManager) IsDevelopment() bool {
	return ecm.GetEnvironment() == "development"
}

// IsProduction returns true if running in production environment
func (ecm *EnhancedConfigManager) IsProduction() bool {
	return ecm.GetEnvironment() == "production"
}

// GetFeatureFlag returns a feature flag value
func (ecm *EnhancedConfigManager) GetFeatureFlag(name string) bool {
	ecm.mu.RLock()
	defer ecm.mu.RUnlock()

	switch name {
	case "enhanced_discovery":
		return ecm.config.Features.EnhancedDiscovery
	case "ai_enabled":
		return ecm.config.Features.AIEnabled
	case "auto_remediation":
		return ecm.config.Features.AutoRemediation
	case "real_time_monitoring":
		return ecm.config.Features.RealTimeMonitoring
	case "advanced_visualization":
		return ecm.config.Features.AdvancedVisualization
	default:
		return false
	}
}

// GetCloudProvider returns configuration for a specific cloud provider
func (ecm *EnhancedConfigManager) GetCloudProvider(name string) (*ProviderConfig, bool) {
	ecm.mu.RLock()
	defer ecm.mu.RUnlock()

	provider, exists := ecm.config.Cloud.Providers[name]
	return &provider, exists
}

// GetDefaultCloudProvider returns the default cloud provider configuration
func (ecm *EnhancedConfigManager) GetDefaultCloudProvider() (*ProviderConfig, bool) {
	return ecm.GetCloudProvider(ecm.config.Cloud.Default)
}

// loadConfig loads configuration from file
func (ecm *EnhancedConfigManager) loadConfig() error {
	data, err := os.ReadFile(ecm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config EnhancedConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := ecm.validateConfig(&config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	ecm.config = &config
	return nil
}

// saveConfig saves configuration to file
func (ecm *EnhancedConfigManager) saveConfig(config *EnhancedConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(ecm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// validateConfig validates configuration
func (ecm *EnhancedConfigManager) validateConfig(config *EnhancedConfig) error {
	// Basic validation
	if config.Environment == "" {
		return fmt.Errorf("environment is required")
	}

	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.Security.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	if len(config.Security.JWTSecret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters")
	}

	// Cloud provider validation
	if config.Cloud.Default == "" {
		return fmt.Errorf("default cloud provider is required")
	}

	if _, exists := config.Cloud.Providers[config.Cloud.Default]; !exists {
		return fmt.Errorf("default cloud provider '%s' not found in providers", config.Cloud.Default)
	}

	return nil
}

// watchConfigFile watches for configuration file changes
func (ecm *EnhancedConfigManager) watchConfigFile() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ecm.ctx.Done():
			return
		case <-ticker.C:
			ecm.checkConfigFileChanges()
		}
	}
}

// checkConfigFileChanges checks if the configuration file has changed
func (ecm *EnhancedConfigManager) checkConfigFileChanges() {
	info, err := os.Stat(ecm.configPath)
	if err != nil {
		return
	}

	ecm.mu.Lock()
	if info.ModTime().After(ecm.lastModified) {
		ecm.lastModified = info.ModTime()
		ecm.mu.Unlock()

		// Reload configuration
		if err := ecm.loadConfig(); err != nil {
			fmt.Printf("Failed to reload configuration: %v\n", err)
			return
		}

		// Notify watchers
		ecm.notifyWatchers(ecm.config)
	} else {
		ecm.mu.Unlock()
	}
}

// notifyWatchers notifies all configuration watchers
func (ecm *EnhancedConfigManager) notifyWatchers(config *EnhancedConfig) {
	for _, watcher := range ecm.watchers {
		if err := watcher.OnConfigChanged(config); err != nil {
			fmt.Printf("Config watcher error: %v\n", err)
		}
	}
}

// Close closes the configuration manager
func (ecm *EnhancedConfigManager) Close() {
	ecm.cancel()
}

// CreateDefaultEnhancedConfig creates a default enhanced configuration
func CreateDefaultEnhancedConfig() *EnhancedConfig {
	return &EnhancedConfig{
		Environment: "development",
		Server: EnhancedServerConfig{
			Host:           "localhost",
			Port:           8080,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			IdleTimeout:    60 * time.Second,
			MaxConnections: 100,
		},
		Database: EnhancedDatabaseConfig{
			Type:           "sqlite",
			Database:       "driftmgr.db",
			MaxConnections: 10,
		},
		Cloud: CloudConfig{
			Providers: map[string]ProviderConfig{
				"aws": {
					Enabled: true,
					Regions: []string{"us-east-1", "us-west-2"},
				},
				"azure": {
					Enabled: true,
					Regions: []string{"eastus", "westus2"},
				},
				"gcp": {
					Enabled: true,
					Regions: []string{"us-central1", "us-east1"},
				},
			},
			Default: "aws",
		},
		Security: EnhancedSecurityConfig{
			JWTSecret:        "your-super-secret-jwt-key-here-make-it-long",
			SessionSecret:    "your-super-secret-session-key-here-make-it-long",
			TokenExpiry:      24 * time.Hour,
			MaxLoginAttempts: 5,
			PasswordPolicy: PasswordPolicy{
				MinLength:      8,
				RequireUpper:   true,
				RequireLower:   true,
				RequireNumber:  true,
				RequireSpecial: true,
			},
		},
		Monitoring: MonitoringConfig{
			Enabled:     true,
			LogLevel:    "info",
			MetricsPort: 9090,
			HealthCheck: HealthCheckConfig{
				Enabled:          true,
				Interval:         30 * time.Second,
				Timeout:          5 * time.Second,
				FailureThreshold: 3,
			},
		},
		Features: FeaturesConfig{
			EnhancedDiscovery:     true,
			AIEnabled:             false,
			AutoRemediation:       false,
			RealTimeMonitoring:    true,
			AdvancedVisualization: true,
		},
		Custom: make(map[string]interface{}),
	}
}

// SaveDefaultEnhancedConfig saves a default enhanced configuration to file
func SaveDefaultEnhancedConfig(configPath string) error {
	config := CreateDefaultEnhancedConfig()

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write default config file: %w", err)
	}

	return nil
}
