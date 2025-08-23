package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/logger"
	"github.com/catherinevee/driftmgr/internal/telemetry"
	"github.com/hashicorp/vault/api"
)

var (
	ErrNotInitialized = errors.New("vault client not initialized")
	ErrSecretNotFound = errors.New("secret not found")
	ErrInvalidPath    = errors.New("invalid vault path")
)

// Client provides secure secret management using HashiCorp Vault
type Client struct {
	client      *api.Client
	mountPath   string
	cacheTTL    time.Duration
	cache       map[string]*cachedSecret
	cacheMu     sync.RWMutex
	log         logger.Logger
	initialized bool
}

// Config represents Vault client configuration
type Config struct {
	Address      string        `json:"address"`
	Token        string        `json:"-"` // Never log tokens
	Namespace    string        `json:"namespace"`
	MountPath    string        `json:"mount_path"`
	CacheTTL     time.Duration `json:"cache_ttl"`
	MaxRetries   int           `json:"max_retries"`
	Timeout      time.Duration `json:"timeout"`
	TLSConfig    *TLSConfig    `json:"tls_config"`
}

// TLSConfig represents TLS configuration for Vault
type TLSConfig struct {
	CACert     string `json:"ca_cert"`
	ClientCert string `json:"client_cert"`
	ClientKey  string `json:"client_key"`
	Insecure   bool   `json:"insecure"`
}

// cachedSecret represents a cached secret with TTL
type cachedSecret struct {
	data      map[string]interface{}
	expiresAt time.Time
}

// NewClient creates a new Vault client
func NewClient(config Config) (*Client, error) {
	vaultConfig := api.DefaultConfig()
	vaultConfig.Address = config.Address
	
	if config.Timeout > 0 {
		vaultConfig.Timeout = config.Timeout
	}
	
	if config.MaxRetries > 0 {
		vaultConfig.MaxRetries = config.MaxRetries
	}
	
	// Configure TLS
	if config.TLSConfig != nil {
		tlsConfig := &api.TLSConfig{
			CACert:        config.TLSConfig.CACert,
			ClientCert:    config.TLSConfig.ClientCert,
			ClientKey:     config.TLSConfig.ClientKey,
			Insecure:      config.TLSConfig.Insecure,
		}
		
		if err := vaultConfig.ConfigureTLS(tlsConfig); err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
	}
	
	client, err := api.NewClient(vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}
	
	// Set token if provided
	if config.Token != "" {
		client.SetToken(config.Token)
	}
	
	// Set namespace if provided
	if config.Namespace != "" {
		client.SetNamespace(config.Namespace)
	}
	
	if config.MountPath == "" {
		config.MountPath = "secret"
	}
	
	if config.CacheTTL <= 0 {
		config.CacheTTL = 5 * time.Minute
	}
	
	vc := &Client{
		client:      client,
		mountPath:   config.MountPath,
		cacheTTL:    config.CacheTTL,
		cache:       make(map[string]*cachedSecret),
		log:         logger.New("vault_client"),
		initialized: true,
	}
	
	// Test connectivity
	if err := vc.healthCheck(); err != nil {
		return nil, fmt.Errorf("vault health check failed: %w", err)
	}
	
	vc.log.Info("Vault client initialized",
		logger.String("address", config.Address),
		logger.String("mount_path", config.MountPath),
		logger.Duration("cache_ttl", config.CacheTTL),
	)
	
	return vc, nil
}

// GetSecret retrieves a secret from Vault
func (c *Client) GetSecret(ctx context.Context, path string) (map[string]interface{}, error) {
	if !c.initialized {
		return nil, ErrNotInitialized
	}
	
	// Record telemetry
	if telemetry.Get() != nil {
		_, span := telemetry.Get().StartSpan(ctx, "vault.get_secret")
		defer span.End()
	}
	
	// Check cache first
	if cached := c.getCached(path); cached != nil {
		c.log.Debug("Secret retrieved from cache",
			logger.String("path", path),
		)
		return cached, nil
	}
	
	// Fetch from Vault
	fullPath := fmt.Sprintf("%s/data/%s", c.mountPath, path)
	secret, err := c.client.Logical().ReadWithContext(ctx, fullPath)
	if err != nil {
		c.log.Error("Failed to read secret",
			logger.String("path", path),
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to read secret: %w", err)
	}
	
	if secret == nil || secret.Data == nil {
		return nil, ErrSecretNotFound
	}
	
	// Extract data from KV v2 format
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		// Try KV v1 format
		data = secret.Data
	}
	
	// Cache the secret
	c.setCached(path, data)
	
	c.log.Debug("Secret retrieved from Vault",
		logger.String("path", path),
	)
	
	return data, nil
}

// WriteSecret writes a secret to Vault
func (c *Client) WriteSecret(ctx context.Context, path string, data map[string]interface{}) error {
	if !c.initialized {
		return ErrNotInitialized
	}
	
	// Record telemetry
	if telemetry.Get() != nil {
		_, span := telemetry.Get().StartSpan(ctx, "vault.write_secret")
		defer span.End()
	}
	
	fullPath := fmt.Sprintf("%s/data/%s", c.mountPath, path)
	
	// Wrap data for KV v2
	wrappedData := map[string]interface{}{
		"data": data,
	}
	
	_, err := c.client.Logical().WriteWithContext(ctx, fullPath, wrappedData)
	if err != nil {
		c.log.Error("Failed to write secret",
			logger.String("path", path),
			logger.Error(err),
		)
		return fmt.Errorf("failed to write secret: %w", err)
	}
	
	// Invalidate cache
	c.invalidateCache(path)
	
	c.log.Info("Secret written to Vault",
		logger.String("path", path),
	)
	
	return nil
}

// DeleteSecret deletes a secret from Vault
func (c *Client) DeleteSecret(ctx context.Context, path string) error {
	if !c.initialized {
		return ErrNotInitialized
	}
	
	// Record telemetry
	if telemetry.Get() != nil {
		_, span := telemetry.Get().StartSpan(ctx, "vault.delete_secret")
		defer span.End()
	}
	
	fullPath := fmt.Sprintf("%s/metadata/%s", c.mountPath, path)
	
	_, err := c.client.Logical().DeleteWithContext(ctx, fullPath)
	if err != nil {
		c.log.Error("Failed to delete secret",
			logger.String("path", path),
			logger.Error(err),
		)
		return fmt.Errorf("failed to delete secret: %w", err)
	}
	
	// Invalidate cache
	c.invalidateCache(path)
	
	c.log.Info("Secret deleted from Vault",
		logger.String("path", path),
	)
	
	return nil
}

// ListSecrets lists secrets at a given path
func (c *Client) ListSecrets(ctx context.Context, path string) ([]string, error) {
	if !c.initialized {
		return nil, ErrNotInitialized
	}
	
	// Record telemetry
	if telemetry.Get() != nil {
		_, span := telemetry.Get().StartSpan(ctx, "vault.list_secrets")
		defer span.End()
	}
	
	fullPath := fmt.Sprintf("%s/metadata/%s", c.mountPath, path)
	
	secret, err := c.client.Logical().ListWithContext(ctx, fullPath)
	if err != nil {
		c.log.Error("Failed to list secrets",
			logger.String("path", path),
			logger.Error(err),
		)
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}
	
	if secret == nil || secret.Data == nil {
		return []string{}, nil
	}
	
	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return []string{}, nil
	}
	
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		if keyStr, ok := key.(string); ok {
			result = append(result, keyStr)
		}
	}
	
	return result, nil
}

// GetCloudCredentials retrieves cloud provider credentials from Vault
func (c *Client) GetCloudCredentials(ctx context.Context, provider string) (map[string]string, error) {
	path := fmt.Sprintf("cloud/%s/credentials", provider)
	
	data, err := c.GetSecret(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s credentials: %w", provider, err)
	}
	
	credentials := make(map[string]string)
	for k, v := range data {
		if str, ok := v.(string); ok {
			credentials[k] = str
		}
	}
	
	return credentials, nil
}

// StoreCloudCredentials stores cloud provider credentials in Vault
func (c *Client) StoreCloudCredentials(ctx context.Context, provider string, credentials map[string]string) error {
	path := fmt.Sprintf("cloud/%s/credentials", provider)
	
	data := make(map[string]interface{})
	for k, v := range credentials {
		data[k] = v
	}
	
	// Add metadata
	data["provider"] = provider
	data["updated_at"] = time.Now().UTC().Format(time.RFC3339)
	
	return c.WriteSecret(ctx, path, data)
}

// GetDatabaseCredentials retrieves database credentials from Vault
func (c *Client) GetDatabaseCredentials(ctx context.Context, database string) (*DatabaseCredentials, error) {
	path := fmt.Sprintf("database/%s/credentials", database)
	
	data, err := c.GetSecret(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get database credentials: %w", err)
	}
	
	creds := &DatabaseCredentials{}
	
	if username, ok := data["username"].(string); ok {
		creds.Username = username
	}
	if password, ok := data["password"].(string); ok {
		creds.Password = password
	}
	if host, ok := data["host"].(string); ok {
		creds.Host = host
	}
	if port, ok := data["port"].(json.Number); ok {
		creds.Port = port.String()
	} else if port, ok := data["port"].(float64); ok {
		creds.Port = fmt.Sprintf("%d", int(port))
	}
	if database, ok := data["database"].(string); ok {
		creds.Database = database
	}
	if sslMode, ok := data["ssl_mode"].(string); ok {
		creds.SSLMode = sslMode
	}
	
	return creds, nil
}

// RenewToken renews the Vault token
func (c *Client) RenewToken(ctx context.Context) error {
	if !c.initialized {
		return ErrNotInitialized
	}
	
	// Record telemetry
	if telemetry.Get() != nil {
		_, span := telemetry.Get().StartSpan(ctx, "vault.renew_token")
		defer span.End()
	}
	
	_, err := c.client.Auth().Token().RenewSelfWithContext(ctx, 0)
	if err != nil {
		c.log.Error("Failed to renew token", logger.Error(err))
		return fmt.Errorf("failed to renew token: %w", err)
	}
	
	c.log.Info("Vault token renewed successfully")
	return nil
}

// healthCheck performs a health check on the Vault server
func (c *Client) healthCheck() error {
	health, err := c.client.Sys().Health()
	if err != nil {
		return err
	}
	
	if !health.Initialized {
		return errors.New("vault is not initialized")
	}
	
	if health.Sealed {
		return errors.New("vault is sealed")
	}
	
	return nil
}

// getCached retrieves a secret from cache if valid
func (c *Client) getCached(path string) map[string]interface{} {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	
	cached, exists := c.cache[path]
	if !exists {
		return nil
	}
	
	if time.Now().After(cached.expiresAt) {
		return nil
	}
	
	return cached.data
}

// setCached stores a secret in cache
func (c *Client) setCached(path string, data map[string]interface{}) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	
	c.cache[path] = &cachedSecret{
		data:      data,
		expiresAt: time.Now().Add(c.cacheTTL),
	}
}

// invalidateCache removes a secret from cache
func (c *Client) invalidateCache(path string) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	
	delete(c.cache, path)
}

// ClearCache clears all cached secrets
func (c *Client) ClearCache() {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	
	c.cache = make(map[string]*cachedSecret)
	c.log.Info("Cache cleared")
}

// DatabaseCredentials represents database connection credentials
type DatabaseCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Database string `json:"database"`
	SSLMode  string `json:"ssl_mode"`
}

// ConnectionString returns a database connection string
func (d *DatabaseCredentials) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.Username, d.Password, d.Host, d.Port, d.Database, d.SSLMode)
}