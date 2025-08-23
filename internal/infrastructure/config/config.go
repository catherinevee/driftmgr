package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the main application configuration
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Discovery DiscoveryConfig `yaml:"discovery"`
	Cache     CacheConfig     `yaml:"cache"`
	Security  SecurityConfig  `yaml:"security"`
	Logging   LoggingConfig   `yaml:"logging"`
	Database  DatabaseConfig  `yaml:"database"`
}

// ServerConfig contains server-related configuration
type ServerConfig struct {
	Port           int           `yaml:"port" env:"DRIFT_SERVER_PORT"`
	Host           string        `yaml:"host" env:"DRIFT_SERVER_HOST"`
	ReadTimeout    time.Duration `yaml:"read_timeout" env:"DRIFT_SERVER_READ_TIMEOUT"`
	WriteTimeout   time.Duration `yaml:"write_timeout" env:"DRIFT_SERVER_WRITE_TIMEOUT"`
	IdleTimeout    time.Duration `yaml:"idle_timeout" env:"DRIFT_SERVER_IDLE_TIMEOUT"`
	MaxRequestSize int64         `yaml:"max_request_size" env:"DRIFT_SERVER_MAX_REQUEST_SIZE"`
	EnableCORS     bool          `yaml:"enable_cors" env:"DRIFT_SERVER_ENABLE_CORS"`
	AllowedOrigins []string      `yaml:"allowed_origins" env:"DRIFT_SERVER_ALLOWED_ORIGINS"`
}

// DiscoveryConfig contains discovery-related configuration
type DiscoveryConfig struct {
	ConcurrencyLimit     int                   `yaml:"concurrency_limit" env:"DRIFT_DISCOVERY_CONCURRENCY_LIMIT"`
	Timeout              time.Duration         `yaml:"timeout" env:"DRIFT_DISCOVERY_TIMEOUT"`
	RetryAttempts        int                   `yaml:"retry_attempts" env:"DRIFT_DISCOVERY_RETRY_ATTEMPTS"`
	RetryDelay           time.Duration         `yaml:"retry_delay" env:"DRIFT_DISCOVERY_RETRY_DELAY"`
	BatchSize            int                   `yaml:"batch_size" env:"DRIFT_DISCOVERY_BATCH_SIZE"`
	EnableCaching        bool                  `yaml:"enable_caching" env:"DRIFT_DISCOVERY_ENABLE_CACHING"`
	CacheTTL             time.Duration         `yaml:"cache_ttl" env:"DRIFT_DISCOVERY_CACHE_TTL"`
	CacheMaxSize         int                   `yaml:"cache_max_size" env:"DRIFT_DISCOVERY_CACHE_MAX_SIZE"`
	MaxConcurrentRegions int                   `yaml:"max_concurrent_regions" env:"DRIFT_DISCOVERY_MAX_CONCURRENT_REGIONS"`
	APITimeout           time.Duration         `yaml:"api_timeout" env:"DRIFT_DISCOVERY_API_TIMEOUT"`
	ShieldTimeout        time.Duration         `yaml:"shield_timeout" env:"DRIFT_DISCOVERY_SHIELD_TIMEOUT"`
	SkipShield           bool                  `yaml:"skip_shield" env:"DRIFT_DISCOVERY_SKIP_SHIELD"`
	Regions              []string              `yaml:"regions" env:"DRIFT_DISCOVERY_REGIONS"`
	AWSProfile           string                `yaml:"aws_profile" env:"DRIFT_DISCOVERY_AWS_PROFILE"`
	AzureProfile         string                `yaml:"azure_profile" env:"DRIFT_DISCOVERY_AZURE_PROFILE"`
	GCPProject           string                `yaml:"gcp_project" env:"DRIFT_DISCOVERY_GCP_PROJECT"`
	DigitalOceanToken    string                `yaml:"digitalocean_token" env:"DRIFT_DISCOVERY_DIGITALOCEAN_TOKEN"`
	QualityThresholds    QualityThresholds     `yaml:"quality_thresholds"`
	DefaultFilters       DiscoveryFilters      `yaml:"default_filters"`
	CLIVerification      CLIVerificationConfig `yaml:"cli_verification"`
}

// QualityThresholds defines quality metrics for discovery
type QualityThresholds struct {
	Completeness float64       `yaml:"completeness"`
	Accuracy     float64       `yaml:"accuracy"`
	Freshness    time.Duration `yaml:"freshness"`
}

// DiscoveryFilters defines default filtering criteria
type DiscoveryFilters struct {
	IncludeTags   map[string]string `yaml:"include_tags"`
	ExcludeTags   map[string]string `yaml:"exclude_tags"`
	ResourceTypes []string          `yaml:"resource_types"`
	AgeThreshold  time.Duration     `yaml:"age_threshold"`
	UsagePatterns []string          `yaml:"usage_patterns"`
	CostThreshold float64           `yaml:"cost_threshold"`
	SecurityScore int               `yaml:"security_score"`
	Environment   string            `yaml:"environment"`
}

// CLIVerificationConfig defines CLI verification settings
type CLIVerificationConfig struct {
	Enabled        bool `yaml:"enabled" env:"DRIFT_CLI_VERIFICATION_ENABLED"`
	TimeoutSeconds int  `yaml:"timeout_seconds" env:"DRIFT_CLI_VERIFICATION_TIMEOUT_SECONDS"`
	MaxRetries     int  `yaml:"max_retries" env:"DRIFT_CLI_VERIFICATION_MAX_RETRIES"`
	Verbose        bool `yaml:"verbose" env:"DRIFT_CLI_VERIFICATION_VERBOSE"`
}

// CacheConfig contains cache-related configuration
type CacheConfig struct {
	Type          string        `yaml:"type" env:"DRIFT_CACHE_TYPE"`
	TTL           time.Duration `yaml:"ttl" env:"DRIFT_CACHE_TTL"`
	MaxSize       int64         `yaml:"max_size" env:"DRIFT_CACHE_MAX_SIZE"`
	MaxEntries    int           `yaml:"max_entries" env:"DRIFT_CACHE_MAX_ENTRIES"`
	CleanupPeriod time.Duration `yaml:"cleanup_period" env:"DRIFT_CACHE_CLEANUP_PERIOD"`
	RedisURL      string        `yaml:"redis_url" env:"DRIFT_CACHE_REDIS_URL"`
	RedisPassword string        `yaml:"redis_password" env:"DRIFT_CACHE_REDIS_PASSWORD"`
	RedisDB       int           `yaml:"redis_db" env:"DRIFT_CACHE_REDIS_DB"`
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	EnableAuth      bool          `yaml:"enable_auth" env:"DRIFT_SECURITY_ENABLE_AUTH"`
	JWTSecret       string        `yaml:"jwt_secret" env:"DRIFT_SECURITY_JWT_SECRET"`
	JWTExpiration   time.Duration `yaml:"jwt_expiration" env:"DRIFT_SECURITY_JWT_EXPIRATION"`
	RateLimit       int           `yaml:"rate_limit" env:"DRIFT_SECURITY_RATE_LIMIT"`
	RateLimitWindow time.Duration `yaml:"rate_limit_window" env:"DRIFT_SECURITY_RATE_LIMIT_WINDOW"`
	AllowedIPs      []string      `yaml:"allowed_ips" env:"DRIFT_SECURITY_ALLOWED_IPS"`
	EnableTLS       bool          `yaml:"enable_tls" env:"DRIFT_SECURITY_ENABLE_TLS"`
	TLSCertFile     string        `yaml:"tls_cert_file" env:"DRIFT_SECURITY_TLS_CERT_FILE"`
	TLSKeyFile      string        `yaml:"tls_key_file" env:"DRIFT_SECURITY_TLS_KEY_FILE"`
}

// LoggingConfig contains logging-related configuration
type LoggingConfig struct {
	Level      string `yaml:"level" env:"DRIFT_LOGGING_LEVEL"`
	Format     string `yaml:"format" env:"DRIFT_LOGGING_FORMAT"`
	Output     string `yaml:"output" env:"DRIFT_LOGGING_OUTPUT"`
	File       string `yaml:"file" env:"DRIFT_LOGGING_FILE"`
	MaxSize    int    `yaml:"max_size" env:"DRIFT_LOGGING_MAX_SIZE"`
	MaxBackups int    `yaml:"max_backups" env:"DRIFT_LOGGING_MAX_BACKUPS"`
	MaxAge     int    `yaml:"max_age" env:"DRIFT_LOGGING_MAX_AGE"`
	Compress   bool   `yaml:"compress" env:"DRIFT_LOGGING_COMPRESS"`
}

// DatabaseConfig contains database-related configuration
type DatabaseConfig struct {
	Type           string `yaml:"type" env:"DRIFT_DATABASE_TYPE"`
	Host           string `yaml:"host" env:"DRIFT_DATABASE_HOST"`
	Port           int    `yaml:"port" env:"DRIFT_DATABASE_PORT"`
	Username       string `yaml:"username" env:"DRIFT_DATABASE_USERNAME"`
	Password       string `yaml:"password" env:"DRIFT_DATABASE_PASSWORD"`
	Database       string `yaml:"database" env:"DRIFT_DATABASE_NAME"`
	SSLMode        string `yaml:"ssl_mode" env:"DRIFT_DATABASE_SSL_MODE"`
	MaxConnections int    `yaml:"max_connections" env:"DRIFT_DATABASE_MAX_CONNECTIONS"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	config := &Config{}

	// Load from file if provided
	if configPath != "" {
		if err := loadFromFile(configPath, config); err != nil {
			return nil, fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	// Override with environment variables
	if err := loadFromEnvironment(config); err != nil {
		return nil, fmt.Errorf("failed to load config from environment: %w", err)
	}

	// Set defaults for missing values
	setDefaults(config)

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// loadFromFile loads configuration from a YAML file
func loadFromFile(configPath string, config *Config) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// loadFromEnvironment loads configuration from environment variables
func loadFromEnvironment(config *Config) error {
	// Server config
	if port := os.Getenv("DRIFT_SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}
	if host := os.Getenv("DRIFT_SERVER_HOST"); host != "" {
		config.Server.Host = host
	}

	// Discovery config
	if concurrency := os.Getenv("DRIFT_DISCOVERY_CONCURRENCY_LIMIT"); concurrency != "" {
		if c, err := strconv.Atoi(concurrency); err == nil {
			config.Discovery.ConcurrencyLimit = c
		}
	}

	// Cache config
	if cacheType := os.Getenv("DRIFT_CACHE_TYPE"); cacheType != "" {
		config.Cache.Type = cacheType
	}

	// Security config
	if jwtSecret := os.Getenv("DRIFT_SECURITY_JWT_SECRET"); jwtSecret != "" {
		config.Security.JWTSecret = jwtSecret
	}

	// Logging config
	if logLevel := os.Getenv("DRIFT_LOGGING_LEVEL"); logLevel != "" {
		config.Logging.Level = logLevel
	}

	// Database config
	if dbHost := os.Getenv("DRIFT_DATABASE_HOST"); dbHost != "" {
		config.Database.Host = dbHost
	}

	return nil
}

// setDefaults sets default values for configuration
func setDefaults(config *Config) {
	// Server defaults
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Server.Host == "" {
		config.Server.Host = "localhost"
	}
	if config.Server.ReadTimeout == 0 {
		config.Server.ReadTimeout = 30 * time.Second
	}
	if config.Server.WriteTimeout == 0 {
		config.Server.WriteTimeout = 30 * time.Second
	}
	if config.Server.IdleTimeout == 0 {
		config.Server.IdleTimeout = 60 * time.Second
	}
	if config.Server.MaxRequestSize == 0 {
		config.Server.MaxRequestSize = 10 * 1024 * 1024 // 10MB
	}

	// Discovery defaults
	if config.Discovery.ConcurrencyLimit == 0 {
		config.Discovery.ConcurrencyLimit = 10
	}
	if config.Discovery.Timeout == 0 {
		config.Discovery.Timeout = 5 * time.Minute
	}
	if config.Discovery.RetryAttempts == 0 {
		config.Discovery.RetryAttempts = 3
	}
	if config.Discovery.RetryDelay == 0 {
		config.Discovery.RetryDelay = 1 * time.Second
	}
	if config.Discovery.BatchSize == 0 {
		config.Discovery.BatchSize = 100
	}

	// Cache defaults
	if config.Cache.Type == "" {
		config.Cache.Type = "memory"
	}
	if config.Cache.TTL == 0 {
		config.Cache.TTL = 15 * time.Minute
	}
	if config.Cache.MaxSize == 0 {
		config.Cache.MaxSize = 100 * 1024 * 1024 // 100MB
	}
	if config.Cache.MaxEntries == 0 {
		config.Cache.MaxEntries = 10000
	}
	if config.Cache.CleanupPeriod == 0 {
		config.Cache.CleanupPeriod = 5 * time.Minute
	}

	// Security defaults
	if config.Security.JWTExpiration == 0 {
		config.Security.JWTExpiration = 24 * time.Hour
	}
	if config.Security.RateLimit == 0 {
		config.Security.RateLimit = 1000
	}
	if config.Security.RateLimitWindow == 0 {
		config.Security.RateLimitWindow = 1 * time.Minute
	}

	// Logging defaults
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Format == "" {
		config.Logging.Format = "json"
	}
	if config.Logging.Output == "" {
		config.Logging.Output = "stdout"
	}
	if config.Logging.MaxSize == 0 {
		config.Logging.MaxSize = 100
	}
	if config.Logging.MaxBackups == 0 {
		config.Logging.MaxBackups = 3
	}
	if config.Logging.MaxAge == 0 {
		config.Logging.MaxAge = 28
	}

	// Database defaults
	if config.Database.Type == "" {
		config.Database.Type = "postgres"
	}
	if config.Database.Port == 0 {
		config.Database.Port = 5432
	}
	if config.Database.SSLMode == "" {
		config.Database.SSLMode = "disable"
	}
	if config.Database.MaxConnections == 0 {
		config.Database.MaxConnections = 10
	}
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate server config
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	// Validate discovery config
	if config.Discovery.ConcurrencyLimit < 1 {
		return fmt.Errorf("invalid concurrency limit: %d", config.Discovery.ConcurrencyLimit)
	}
	if config.Discovery.BatchSize < 1 {
		return fmt.Errorf("invalid batch size: %d", config.Discovery.BatchSize)
	}

	// Validate cache config
	validCacheTypes := []string{"memory", "redis"}
	if !contains(validCacheTypes, config.Cache.Type) {
		return fmt.Errorf("invalid cache type: %s", config.Cache.Type)
	}

	// Validate logging config
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, strings.ToLower(config.Logging.Level)) {
		return fmt.Errorf("invalid log level: %s", config.Logging.Level)
	}

	// Validate database config
	validDBTypes := []string{"postgres", "mysql", "sqlite"}
	if !contains(validDBTypes, config.Database.Type) {
		return fmt.Errorf("invalid database type: %s", config.Database.Type)
	}

	return nil
}

// contains checks if a slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// GetDatabaseDSN returns the database connection string
func (c *Config) GetDatabaseDSN() string {
	switch c.Database.Type {
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Database.Host, c.Database.Port, c.Database.Username, c.Database.Password,
			c.Database.Database, c.Database.SSLMode)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			c.Database.Username, c.Database.Password, c.Database.Host,
			c.Database.Port, c.Database.Database)
	case "sqlite":
		return c.Database.Database
	default:
		return ""
	}
}
