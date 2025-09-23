package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `json:"server"`
	Backend  BackendConfig  `json:"backend"`
	Security SecurityConfig `json:"security"`
	Logging  LoggingConfig  `json:"logging"`
	Features FeatureConfig  `json:"features"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
	AuthEnabled  bool          `json:"auth_enabled"`
	TLSCert      string        `json:"tls_cert,omitempty"`
	TLSKey       string        `json:"tls_key,omitempty"`
}

// BackendConfig represents backend configuration
type BackendConfig struct {
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"config"`
	MaxRetries  int                    `json:"max_retries"`
	RetryDelay  time.Duration          `json:"retry_delay"`
	Timeout     time.Duration          `json:"timeout"`
	PoolSize    int                    `json:"pool_size"`
	IdleTimeout time.Duration          `json:"idle_timeout"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	EnableHTTPS     bool          `json:"enable_https"`
	EnableCORS      bool          `json:"enable_cors"`
	CORSOrigins     []string      `json:"cors_origins"`
	EnableRateLimit bool          `json:"enable_rate_limit"`
	RateLimitRPS    int           `json:"rate_limit_rps"`
	SessionTimeout  time.Duration `json:"session_timeout"`
	JWTSecret       string        `json:"jwt_secret,omitempty"`
	APIKeyRequired  bool          `json:"api_key_required"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	Output     string `json:"output"`
	FilePath   string `json:"file_path,omitempty"`
	MaxSize    int    `json:"max_size"`
	MaxBackups int    `json:"max_backups"`
	MaxAge     int    `json:"max_age"`
	Compress   bool   `json:"compress"`
}

// FeatureConfig represents feature flags
type FeatureConfig struct {
	EnableDiscovery    bool `json:"enable_discovery"`
	EnableRemediation  bool `json:"enable_remediation"`
	EnableCostAnalysis bool `json:"enable_cost_analysis"`
	EnableSecurityScan bool `json:"enable_security_scan"`
	EnableTerragrunt   bool `json:"enable_terragrunt"`
	EnableWebUI        bool `json:"enable_web_ui"`
	EnableAPI          bool `json:"enable_api"`
	EnableWebSocket    bool `json:"enable_websocket"`
	EnableMetrics      bool `json:"enable_metrics"`
	EnableHealthCheck  bool `json:"enable_health_check"`
}

// ConfigManager handles configuration loading and management
type ConfigManager struct {
	config     *Config
	configPath string
	watchers   []ConfigWatcher
}

// ConfigWatcher defines an interface for configuration change watchers
type ConfigWatcher interface {
	OnConfigChange(config *Config) error
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		config:   getDefaultConfig(),
		watchers: make([]ConfigWatcher, 0),
	}
}

// LoadConfig loads configuration from a file
func (cm *ConfigManager) LoadConfig(configPath string) error {
	cm.configPath = configPath

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config file if it doesn't exist
		return cm.SaveConfig()
	}

	// Read configuration file
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON configuration
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := cm.validateConfig(&config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	cm.config = &config
	return nil
}

// SaveConfig saves the current configuration to file
func (cm *ConfigManager) SaveConfig() error {
	if cm.configPath == "" {
		return fmt.Errorf("no config path specified")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal configuration to JSON
	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := ioutil.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *Config {
	return cm.config
}

// UpdateConfig updates the configuration
func (cm *ConfigManager) UpdateConfig(updates *Config) error {
	// Validate the updated configuration
	if err := cm.validateConfig(updates); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Update configuration
	cm.config = updates

	// Notify watchers
	for _, watcher := range cm.watchers {
		if err := watcher.OnConfigChange(cm.config); err != nil {
			// Log error but don't fail the update
			fmt.Printf("Warning: config watcher failed: %v\n", err)
		}
	}

	return nil
}

// AddWatcher adds a configuration change watcher
func (cm *ConfigManager) AddWatcher(watcher ConfigWatcher) {
	cm.watchers = append(cm.watchers, watcher)
}

// RemoveWatcher removes a configuration change watcher
func (cm *ConfigManager) RemoveWatcher(watcher ConfigWatcher) {
	for i, w := range cm.watchers {
		if w == watcher {
			cm.watchers = append(cm.watchers[:i], cm.watchers[i+1:]...)
			break
		}
	}
}

// ReloadConfig reloads configuration from file
func (cm *ConfigManager) ReloadConfig() error {
	if cm.configPath == "" {
		return fmt.Errorf("no config path specified")
	}

	return cm.LoadConfig(cm.configPath)
}

// WatchConfigFile watches the configuration file for changes
func (cm *ConfigManager) WatchConfigFile() error {
	if cm.configPath == "" {
		return fmt.Errorf("no config path specified")
	}

	// Simple file watching implementation
	// In production, you might want to use fsnotify or similar
	go func() {
		var lastModTime time.Time

		for {
			stat, err := os.Stat(cm.configPath)
			if err != nil {
				time.Sleep(5 * time.Second)
				continue
			}

			if stat.ModTime().After(lastModTime) {
				lastModTime = stat.ModTime()
				if err := cm.ReloadConfig(); err != nil {
					fmt.Printf("Failed to reload config: %v\n", err)
				}
			}

			time.Sleep(1 * time.Second)
		}
	}()

	return nil
}

// validateConfig validates the configuration
func (cm *ConfigManager) validateConfig(config *Config) error {
	// Validate server configuration
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.Server.ReadTimeout < 0 {
		return fmt.Errorf("invalid read timeout: %v", config.Server.ReadTimeout)
	}

	if config.Server.WriteTimeout < 0 {
		return fmt.Errorf("invalid write timeout: %v", config.Server.WriteTimeout)
	}

	// Validate backend configuration
	if config.Backend.Type == "" {
		return fmt.Errorf("backend type is required")
	}

	if config.Backend.MaxRetries < 0 {
		return fmt.Errorf("invalid max retries: %d", config.Backend.MaxRetries)
	}

	if config.Backend.RetryDelay < 0 {
		return fmt.Errorf("invalid retry delay: %v", config.Backend.RetryDelay)
	}

	// Validate security configuration
	if config.Security.RateLimitRPS < 0 {
		return fmt.Errorf("invalid rate limit RPS: %d", config.Security.RateLimitRPS)
	}

	if config.Security.SessionTimeout < 0 {
		return fmt.Errorf("invalid session timeout: %v", config.Security.SessionTimeout)
	}

	// Validate logging configuration
	validLevels := []string{"debug", "info", "warn", "error", "fatal"}
	validLevel := false
	for _, level := range validLevels {
		if config.Logging.Level == level {
			validLevel = true
			break
		}
	}
	if !validLevel {
		return fmt.Errorf("invalid log level: %s", config.Logging.Level)
	}

	validFormats := []string{"json", "text"}
	validFormat := false
	for _, format := range validFormats {
		if config.Logging.Format == format {
			validFormat = true
			break
		}
	}
	if !validFormat {
		return fmt.Errorf("invalid log format: %s", config.Logging.Format)
	}

	return nil
}

// getDefaultConfig returns the default configuration
func getDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
			AuthEnabled:  false,
		},
		Backend: BackendConfig{
			Type:        "local",
			Config:      make(map[string]interface{}),
			MaxRetries:  3,
			RetryDelay:  1 * time.Second,
			Timeout:     30 * time.Second,
			PoolSize:    10,
			IdleTimeout: 5 * time.Minute,
		},
		Security: SecurityConfig{
			EnableHTTPS:     false,
			EnableCORS:      true,
			CORSOrigins:     []string{"*"},
			EnableRateLimit: true,
			RateLimitRPS:    100,
			SessionTimeout:  24 * time.Hour,
			APIKeyRequired:  false,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		},
		Features: FeatureConfig{
			EnableDiscovery:    true,
			EnableRemediation:  true,
			EnableCostAnalysis: true,
			EnableSecurityScan: true,
			EnableTerragrunt:   true,
			EnableWebUI:        true,
			EnableAPI:          true,
			EnableWebSocket:    true,
			EnableMetrics:      true,
			EnableHealthCheck:  true,
		},
	}
}

// GetConfigPath returns the default configuration file path
func GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".driftmgr", "config.json")
}

// LoadConfigFromFile loads configuration from a file path
func LoadConfigFromFile(configPath string) (*Config, error) {
	manager := NewConfigManager()
	if err := manager.LoadConfig(configPath); err != nil {
		return nil, err
	}
	return manager.GetConfig(), nil
}

// SaveConfigToFile saves configuration to a file path
func SaveConfigToFile(config *Config, configPath string) error {
	manager := NewConfigManager()
	manager.config = config
	manager.configPath = configPath
	return manager.SaveConfig()
}
