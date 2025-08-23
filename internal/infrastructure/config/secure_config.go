package config

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/catherinevee/driftmgr/internal/utils/errors"
	"github.com/catherinevee/driftmgr/internal/observability/logging"
	"github.com/catherinevee/driftmgr/internal/infrastructure/secrets"
	"github.com/rs/zerolog"
)

// SecureConfig manages all application configuration with security best practices
type SecureConfig struct {
	mu sync.RWMutex
	
	// Configuration sources (in priority order)
	envVars      map[string]string
	vaultManager *secrets.VaultManager
	configFile   map[string]interface{}
	
	// Encryption for sensitive data
	encryptionKey []byte
	
	// Validation rules
	validators map[string]Validator
	
	// Audit logger
	auditLogger *zerolog.Logger
	
	// Cache for decrypted values
	cache      map[string]interface{}
	cacheMutex sync.RWMutex
}

// Validator validates configuration values
type Validator func(value interface{}) error

// SensitiveFields that should never be logged or exposed
var SensitiveFields = map[string]bool{
	"password":           true,
	"secret":            true,
	"token":             true,
	"key":               true,
	"credential":        true,
	"api_key":           true,
	"access_key":        true,
	"secret_key":        true,
	"private_key":       true,
	"client_secret":     true,
	"database_password": true,
	"jwt_secret":        true,
}

// NewSecureConfig creates a new secure configuration manager
func NewSecureConfig() (*SecureConfig, error) {
	// Generate or load encryption key
	encKey, err := loadOrGenerateEncryptionKey()
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrorTypeConfig, "failed to initialize encryption key")
	}
	
	sc := &SecureConfig{
		envVars:       make(map[string]string),
		configFile:    make(map[string]interface{}),
		validators:    make(map[string]Validator),
		cache:         make(map[string]interface{}),
		encryptionKey: encKey,
		auditLogger:   func() *zerolog.Logger { l := logging.WithComponent("config-audit"); return &l }(),
	}
	
	// Load environment variables
	sc.loadEnvironmentVariables()
	
	// Initialize Vault if configured
	if vaultAddr := os.Getenv("VAULT_ADDR"); vaultAddr != "" {
		vaultConfig := &secrets.Config{
			Address: vaultAddr,
			Token:   os.Getenv("VAULT_TOKEN"),
		}
		sc.vaultManager, err = secrets.NewVaultManager(vaultConfig)
		if err != nil {
			sc.auditLogger.Warn().Err(err).Msg("Vault initialization failed, using local config only")
		}
	}
	
	// Register default validators
	sc.registerDefaultValidators()
	
	return sc, nil
}

// GetString retrieves a string configuration value
func (sc *SecureConfig) GetString(key string) (string, error) {
	value, err := sc.get(key)
	if err != nil {
		return "", err
	}
	
	str, ok := value.(string)
	if !ok {
		return "", errors.Newf(errors.ErrorTypeConfig, "value for key %s is not a string", key)
	}
	
	// Audit access to sensitive fields
	if isSensitive(key) {
		sc.auditLogger.Info().
			Str("key", key).
			Str("action", "access").
			Msg("sensitive configuration accessed")
	}
	
	return str, nil
}

// GetSecureString retrieves a sensitive string (never logged)
func (sc *SecureConfig) GetSecureString(key string) (string, error) {
	// Try Vault first for sensitive data
	if sc.vaultManager != nil {
		ctx := context.Background()
		data, err := sc.vaultManager.GetSecret(ctx, "config/"+key)
		if err == nil {
			if val, ok := data["value"].(string); ok {
				return val, nil
			}
		}
	}
	
	// Fall back to encrypted local storage
	return sc.GetString(key)
}

// get retrieves a configuration value from available sources
func (sc *SecureConfig) get(key string) (interface{}, error) {
	sc.cacheMutex.RLock()
	if cached, exists := sc.cache[key]; exists {
		sc.cacheMutex.RUnlock()
		return cached, nil
	}
	sc.cacheMutex.RUnlock()
	
	// Check environment variables first (highest priority)
	if envVal, exists := sc.envVars[strings.ToUpper(key)]; exists {
		return sc.decryptIfNeeded(key, envVal)
	}
	
	// Check Vault
	if sc.vaultManager != nil {
		ctx := context.Background()
		data, err := sc.vaultManager.GetSecret(ctx, "config/"+key)
		if err == nil {
			if val, exists := data["value"]; exists {
				sc.cacheValue(key, val)
				return val, nil
			}
		}
	}
	
	// Check config file
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	
	if val, exists := sc.configFile[key]; exists {
		return sc.decryptIfNeeded(key, val)
	}
	
	return nil, errors.NotFoundError(fmt.Sprintf("configuration key %s", key))
}

// SetSecure sets a sensitive configuration value (encrypted)
func (sc *SecureConfig) SetSecure(key string, value string) error {
	// Validate the value
	if validator, exists := sc.validators[key]; exists {
		if err := validator(value); err != nil {
			return errors.Wrap(err, errors.ErrorTypeValidation, "configuration validation failed")
		}
	}
	
	// Encrypt the value
	encrypted, err := sc.encrypt(value)
	if err != nil {
		return err
	}
	
	// Store in Vault if available
	if sc.vaultManager != nil {
		ctx := context.Background()
		data := map[string]interface{}{
			"value": value,
			"encrypted": false,
		}
		if err := sc.vaultManager.PutSecret(ctx, "config/"+key, data); err == nil {
			sc.invalidateCache(key)
			sc.auditLogger.Info().
				Str("key", key).
				Str("action", "set").
				Str("storage", "vault").
				Msg("secure configuration updated")
			return nil
		}
	}
	
	// Store locally (encrypted)
	sc.mu.Lock()
	sc.configFile[key] = encrypted
	sc.mu.Unlock()
	
	sc.invalidateCache(key)
	
	sc.auditLogger.Info().
		Str("key", key).
		Str("action", "set").
		Str("storage", "local-encrypted").
		Msg("secure configuration updated")
	
	return nil
}

// Validate validates all configuration
func (sc *SecureConfig) Validate() error {
	var validationErrors []string
	
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	
	for key, validator := range sc.validators {
		value, err := sc.get(key)
		if err != nil {
			// Skip if not found (might be optional)
			continue
		}
		
		if err := validator(value); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("%s: %v", key, err))
		}
	}
	
	if len(validationErrors) > 0 {
		return errors.ValidationError(fmt.Sprintf("configuration validation failed: %s", 
			strings.Join(validationErrors, "; ")))
	}
	
	return nil
}

// RegisterValidator registers a validator for a configuration key
func (sc *SecureConfig) RegisterValidator(key string, validator Validator) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.validators[key] = validator
}

// loadEnvironmentVariables loads all environment variables
func (sc *SecureConfig) loadEnvironmentVariables() {
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			// Only load DRIFTMGR_ prefixed variables
			if strings.HasPrefix(parts[0], "DRIFTMGR_") {
				key := strings.TrimPrefix(parts[0], "DRIFTMGR_")
				sc.envVars[key] = parts[1]
			}
		}
	}
}

// registerDefaultValidators registers built-in validators
func (sc *SecureConfig) registerDefaultValidators() {
	// Port validator
	sc.RegisterValidator("port", func(value interface{}) error {
		port, ok := value.(string)
		if !ok {
			return fmt.Errorf("port must be a string")
		}
		// Validate port range
		if port < "1" || port > "65535" {
			return fmt.Errorf("invalid port number")
		}
		return nil
	})
	
	// Database URL validator
	sc.RegisterValidator("database_url", func(value interface{}) error {
		url, ok := value.(string)
		if !ok {
			return fmt.Errorf("database URL must be a string")
		}
		if !strings.Contains(url, "://") {
			return fmt.Errorf("invalid database URL format")
		}
		return nil
	})
	
	// API key validator
	sc.RegisterValidator("api_key", func(value interface{}) error {
		key, ok := value.(string)
		if !ok {
			return fmt.Errorf("API key must be a string")
		}
		if len(key) < 32 {
			return fmt.Errorf("API key too short (minimum 32 characters)")
		}
		return nil
	})
}

// Encryption methods

func (sc *SecureConfig) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(sc.encryptionKey)
	if err != nil {
		return "", err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (sc *SecureConfig) decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	
	block, err := aes.NewCipher(sc.encryptionKey)
	if err != nil {
		return "", err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}
	
	return string(plaintext), nil
}

func (sc *SecureConfig) decryptIfNeeded(key string, value interface{}) (interface{}, error) {
	str, ok := value.(string)
	if !ok {
		return value, nil
	}
	
	// Check if it's encrypted (base64 with prefix)
	if strings.HasPrefix(str, "ENC:") {
		decrypted, err := sc.decrypt(strings.TrimPrefix(str, "ENC:"))
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrorTypeConfig, "failed to decrypt configuration")
		}
		return decrypted, nil
	}
	
	return value, nil
}

func (sc *SecureConfig) cacheValue(key string, value interface{}) {
	sc.cacheMutex.Lock()
	defer sc.cacheMutex.Unlock()
	sc.cache[key] = value
}

func (sc *SecureConfig) invalidateCache(key string) {
	sc.cacheMutex.Lock()
	defer sc.cacheMutex.Unlock()
	delete(sc.cache, key)
}

// Helper functions

func loadOrGenerateEncryptionKey() ([]byte, error) {
	keyFile := ".encryption.key"
	
	// Try to load existing key
	if data, err := os.ReadFile(keyFile); err == nil {
		hash := sha256.Sum256(data)
		return hash[:], nil
	}
	
	// Generate new key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	
	// Save for future use (with restricted permissions)
	if err := os.WriteFile(keyFile, key, 0600); err != nil {
		return nil, err
	}
	
	return key, nil
}

func isSensitive(key string) bool {
	lowerKey := strings.ToLower(key)
	for sensitive := range SensitiveFields {
		if strings.Contains(lowerKey, sensitive) {
			return true
		}
	}
	return false
}

// GetCloudCredentials retrieves cloud provider credentials securely
func (sc *SecureConfig) GetCloudCredentials(provider string) (*CloudCredentials, error) {
	switch provider {
	case "aws":
		return sc.getAWSCredentials()
	case "azure":
		return sc.getAzureCredentials()
	case "gcp":
		return sc.getGCPCredentials()
	case "digitalocean":
		return sc.getDigitalOceanCredentials()
	default:
		return nil, errors.Newf(errors.ErrorTypeValidation, "unsupported provider: %s", provider)
	}
}

func (sc *SecureConfig) getAWSCredentials() (*CloudCredentials, error) {
	accessKey, _ := sc.GetSecureString("aws_access_key_id")
	secretKey, _ := sc.GetSecureString("aws_secret_access_key")
	sessionToken, _ := sc.GetSecureString("aws_session_token")
	region, _ := sc.GetString("aws_region")
	
	if accessKey == "" || secretKey == "" {
		return nil, errors.ValidationError("AWS credentials not configured")
	}
	
	return &CloudCredentials{
		Provider: "aws",
		AWS: &AWSCredentials{
			AccessKeyID:     accessKey,
			SecretAccessKey: secretKey,
			SessionToken:    sessionToken,
			Region:          region,
		},
	}, nil
}

func (sc *SecureConfig) getAzureCredentials() (*CloudCredentials, error) {
	tenantID, _ := sc.GetSecureString("azure_tenant_id")
	clientID, _ := sc.GetSecureString("azure_client_id")
	clientSecret, _ := sc.GetSecureString("azure_client_secret")
	subscriptionID, _ := sc.GetString("azure_subscription_id")
	
	if tenantID == "" || clientID == "" || clientSecret == "" {
		return nil, errors.ValidationError("Azure credentials not configured")
	}
	
	return &CloudCredentials{
		Provider: "azure",
		Azure: &AzureCredentials{
			TenantID:       tenantID,
			ClientID:       clientID,
			ClientSecret:   clientSecret,
			SubscriptionID: subscriptionID,
		},
	}, nil
}

func (sc *SecureConfig) getGCPCredentials() (*CloudCredentials, error) {
	keyJSON, _ := sc.GetSecureString("gcp_key_json")
	
	if keyJSON == "" {
		// Try to load from file
		keyFile, _ := sc.GetString("gcp_key_file")
		if keyFile != "" {
			data, err := os.ReadFile(keyFile)
			if err != nil {
				return nil, errors.Wrap(err, errors.ErrorTypeConfig, "failed to read GCP key file")
			}
			keyJSON = string(data)
		}
	}
	
	if keyJSON == "" {
		return nil, errors.ValidationError("GCP credentials not configured")
	}
	
	return &CloudCredentials{
		Provider: "gcp",
		GCP: &GCPCredentials{
			KeyJSON: keyJSON,
		},
	}, nil
}

func (sc *SecureConfig) getDigitalOceanCredentials() (*CloudCredentials, error) {
	token, _ := sc.GetSecureString("digitalocean_token")
	
	if token == "" {
		return nil, errors.ValidationError("DigitalOcean token not configured")
	}
	
	return &CloudCredentials{
		Provider: "digitalocean",
		DigitalOcean: &DigitalOceanCredentials{
			Token: token,
		},
	}, nil
}

// Cloud credential types
type CloudCredentials struct {
	Provider     string
	AWS          *AWSCredentials
	Azure        *AzureCredentials
	GCP          *GCPCredentials
	DigitalOcean *DigitalOceanCredentials
}

type AWSCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Region          string
}

type AzureCredentials struct {
	TenantID       string
	ClientID       string
	ClientSecret   string
	SubscriptionID string
}

type GCPCredentials struct {
	KeyJSON string
}

type DigitalOceanCredentials struct {
	Token string
}