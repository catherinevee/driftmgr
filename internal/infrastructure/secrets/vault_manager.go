package secrets

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"sync"
	"time"

	vault "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/kubernetes"

	"github.com/catherinevee/driftmgr/internal/observability/logging"
	"github.com/catherinevee/driftmgr/internal/utils/circuit"
	"github.com/catherinevee/driftmgr/internal/utils/errors"
	"github.com/rs/zerolog"
)

// VaultManager handles all interactions with HashiCorp Vault
type VaultManager struct {
	client          *vault.Client
	mountPath       string
	cacheTTL        time.Duration
	cache           map[string]*cachedSecret
	cacheMutex      sync.RWMutex
	renewalStopChan chan struct{}
	logger          *zerolog.Logger
	circuitBreaker  *resilience.CircuitBreaker
}

// cachedSecret represents a cached secret with expiration
type cachedSecret struct {
	data      map[string]interface{}
	expiresAt time.Time
}

// Config represents Vault configuration
type Config struct {
	Address          string        `json:"address" yaml:"address"`
	Token            string        `json:"token" yaml:"token"`
	MountPath        string        `json:"mount_path" yaml:"mount_path"`
	CacheTTL         time.Duration `json:"cache_ttl" yaml:"cache_ttl"`
	Namespace        string        `json:"namespace" yaml:"namespace"`
	RoleID           string        `json:"role_id" yaml:"role_id"`
	SecretID         string        `json:"secret_id" yaml:"secret_id"`
	KubernetesRole   string        `json:"kubernetes_role" yaml:"kubernetes_role"`
	KubernetesSAPath string        `json:"kubernetes_sa_path" yaml:"kubernetes_sa_path"`
	TLSConfig        *TLSConfig    `json:"tls_config" yaml:"tls_config"`
}

// TLSConfig represents TLS configuration for Vault
type TLSConfig struct {
	CACert     string `json:"ca_cert" yaml:"ca_cert"`
	ClientCert string `json:"client_cert" yaml:"client_cert"`
	ClientKey  string `json:"client_key" yaml:"client_key"`
	Insecure   bool   `json:"insecure" yaml:"insecure"`
}

// DefaultConfig returns default Vault configuration
func DefaultConfig() *Config {
	return &Config{
		Address:   getEnvOrDefault("VAULT_ADDR", "https://vault.service.consul:8200"),
		MountPath: getEnvOrDefault("VAULT_MOUNT", "secret"),
		CacheTTL:  5 * time.Minute,
	}
}

// NewVaultManager creates a new Vault manager
func NewVaultManager(cfg *Config) (*VaultManager, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Create Vault client configuration
	vaultConfig := vault.DefaultConfig()
	vaultConfig.Address = cfg.Address

	// Configure TLS if provided
	if cfg.TLSConfig != nil {
		tlsConfig := &vault.TLSConfig{
			CACert:     cfg.TLSConfig.CACert,
			ClientCert: cfg.TLSConfig.ClientCert,
			ClientKey:  cfg.TLSConfig.ClientKey,
			Insecure:   cfg.TLSConfig.Insecure,
		}
		if err := vaultConfig.ConfigureTLS(tlsConfig); err != nil {
			return nil, errors.Wrapf(err, errors.ErrorTypeConfig, "failed to configure TLS")
		}
	}

	// Create Vault client
	client, err := vault.NewClient(vaultConfig)
	if err != nil {
		return nil, errors.Wrapf(err, errors.ErrorTypeConfig, "failed to create Vault client")
	}

	// Set namespace if provided
	if cfg.Namespace != "" {
		client.SetNamespace(cfg.Namespace)
	}

	vm := &VaultManager{
		client:          client,
		mountPath:       cfg.MountPath,
		cacheTTL:        cfg.CacheTTL,
		cache:           make(map[string]*cachedSecret),
		renewalStopChan: make(chan struct{}),
		logger:          func() *zerolog.Logger { l := logging.WithComponent("vault"); return &l }(),
		circuitBreaker: resilience.NewCircuitBreaker(resilience.Config{
			Name:             "vault",
			MaxFailures:      5,
			ResetTimeout:     30 * time.Second,
			HalfOpenMaxCalls: 3,
		}),
	}

	// Authenticate based on available credentials
	if err := vm.authenticate(cfg); err != nil {
		return nil, errors.Wrapf(err, errors.ErrorTypeUnauthorized, "failed to authenticate with Vault")
	}

	// Start token renewal if using auth methods that provide renewable tokens
	go vm.renewToken()

	// Start cache cleanup
	go vm.cleanupCache()

	return vm, nil
}

// authenticate handles various authentication methods
func (vm *VaultManager) authenticate(cfg *Config) error {
	// Try token authentication first
	if cfg.Token != "" {
		vm.client.SetToken(cfg.Token)
		return vm.verifyConnection()
	}

	// Try AppRole authentication
	if cfg.RoleID != "" && cfg.SecretID != "" {
		return vm.authenticateAppRole(cfg.RoleID, cfg.SecretID)
	}

	// Try Kubernetes authentication
	if cfg.KubernetesRole != "" {
		return vm.authenticateKubernetes(cfg.KubernetesRole, cfg.KubernetesSAPath)
	}

	// Try environment variable token
	if token := os.Getenv("VAULT_TOKEN"); token != "" {
		vm.client.SetToken(token)
		return vm.verifyConnection()
	}

	return errors.New(errors.ErrorTypeConfig, "no valid authentication method configured")
}

// authenticateAppRole authenticates using AppRole
func (vm *VaultManager) authenticateAppRole(roleID, secretID string) error {
	data := map[string]interface{}{
		"role_id":   roleID,
		"secret_id": secretID,
	}

	resp, err := vm.client.Logical().Write("auth/approle/login", data)
	if err != nil {
		return err
	}

	if resp.Auth == nil {
		return errors.New(errors.ErrorTypeUnauthorized, "no auth info returned from AppRole login")
	}

	vm.client.SetToken(resp.Auth.ClientToken)
	return nil
}

// authenticateKubernetes authenticates using Kubernetes service account
func (vm *VaultManager) authenticateKubernetes(role, saPath string) error {
	if saPath == "" {
		saPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	}

	k8sAuth, err := kubernetes.NewKubernetesAuth(role)
	if err != nil {
		return err
	}

	authInfo, err := vm.client.Auth().Login(context.Background(), k8sAuth)
	if err != nil {
		return err
	}

	if authInfo == nil {
		return errors.New(errors.ErrorTypeUnauthorized, "no auth info returned from Kubernetes login")
	}

	return nil
}

// verifyConnection verifies the connection to Vault
func (vm *VaultManager) verifyConnection() error {
	_, err := vm.client.Sys().Health()
	return err
}

// GetSecret retrieves a secret from Vault with caching
func (vm *VaultManager) GetSecret(ctx context.Context, path string) (map[string]interface{}, error) {
	// Check cache first
	if cached := vm.getFromCache(path); cached != nil {
		return cached, nil
	}

	// Use circuit breaker for Vault calls
	result, err := vm.circuitBreaker.Call(ctx, func() (interface{}, error) {
		return vm.fetchSecret(path)
	})

	if err != nil {
		return nil, err
	}

	data := result.(map[string]interface{})
	vm.addToCache(path, data)

	return data, nil
}

// fetchSecret fetches a secret from Vault
func (vm *VaultManager) fetchSecret(path string) (map[string]interface{}, error) {
	fullPath := fmt.Sprintf("%s/data/%s", vm.mountPath, path)

	secret, err := vm.client.Logical().Read(fullPath)
	if err != nil {
		return nil, errors.Wrapf(err, errors.ErrorTypeProvider, "failed to read secret from path %s", path)
	}

	if secret == nil || secret.Data == nil {
		return nil, errors.NotFoundError(fmt.Sprintf("secret at path %s", path))
	}

	// For KV v2, the actual data is nested under "data"
	if data, ok := secret.Data["data"].(map[string]interface{}); ok {
		return data, nil
	}

	return secret.Data, nil
}

// PutSecret stores a secret in Vault
func (vm *VaultManager) PutSecret(ctx context.Context, path string, data map[string]interface{}) error {
	_, err := vm.circuitBreaker.Call(ctx, func() (interface{}, error) {
		fullPath := fmt.Sprintf("%s/data/%s", vm.mountPath, path)

		// For KV v2, wrap the data
		wrappedData := map[string]interface{}{
			"data": data,
		}

		_, err := vm.client.Logical().Write(fullPath, wrappedData)
		if err != nil {
			return nil, errors.Wrapf(err, errors.ErrorTypeProvider, "failed to write secret to path %s", path)
		}

		// Invalidate cache for this path
		vm.invalidateCache(path)

		return nil, nil
	})

	return err
}

// DeleteSecret deletes a secret from Vault
func (vm *VaultManager) DeleteSecret(ctx context.Context, path string) error {
	_, err := vm.circuitBreaker.Call(ctx, func() (interface{}, error) {
		fullPath := fmt.Sprintf("%s/metadata/%s", vm.mountPath, path)

		_, err := vm.client.Logical().Delete(fullPath)
		if err != nil {
			return nil, errors.Wrapf(err, errors.ErrorTypeProvider, "failed to delete secret at path %s", path)
		}

		// Invalidate cache for this path
		vm.invalidateCache(path)

		return nil, nil
	})

	return err
}

// GetCloudCredentials retrieves cloud provider credentials from Vault
func (vm *VaultManager) GetCloudCredentials(ctx context.Context, provider string) (*CloudCredentials, error) {
	path := fmt.Sprintf("cloud/%s/credentials", provider)
	data, err := vm.GetSecret(ctx, path)
	if err != nil {
		return nil, err
	}

	creds := &CloudCredentials{
		Provider: provider,
	}

	switch provider {
	case "aws":
		creds.AWS = &AWSCredentials{
			AccessKeyID:     getString(data, "access_key_id"),
			SecretAccessKey: getString(data, "secret_access_key"),
			SessionToken:    getString(data, "session_token"),
			Region:          getString(data, "region"),
		}
	case "azure":
		creds.Azure = &AzureCredentials{
			TenantID:       getString(data, "tenant_id"),
			ClientID:       getString(data, "client_id"),
			ClientSecret:   getString(data, "client_secret"),
			SubscriptionID: getString(data, "subscription_id"),
		}
	case "gcp":
		// GCP credentials are usually stored as a JSON key file
		if keyJSON, ok := data["key_json"].(string); ok {
			creds.GCP = &GCPCredentials{
				KeyJSON: keyJSON,
			}
		}
	case "digitalocean":
		creds.DigitalOcean = &DigitalOceanCredentials{
			Token: getString(data, "token"),
		}
	default:
		return nil, errors.Newf(errors.ErrorTypeValidation, "unsupported provider: %s", provider)
	}

	return creds, nil
}

// RotateCredentials rotates credentials for a specific provider
func (vm *VaultManager) RotateCredentials(ctx context.Context, provider string) error {
	// This would typically integrate with cloud provider APIs to generate new credentials
	// For now, we'll just log the rotation request
	vm.logger.Info().
		Str("provider", provider).
		Msg("credential rotation requested")

	// Invalidate cached credentials
	path := fmt.Sprintf("cloud/%s/credentials", provider)
	vm.invalidateCache(path)

	return nil
}

// EncryptData encrypts data using Vault's transit engine
func (vm *VaultManager) EncryptData(ctx context.Context, keyName string, plaintext []byte) (string, error) {
	result, err := vm.circuitBreaker.Call(ctx, func() (interface{}, error) {
		data := map[string]interface{}{
			"plaintext": base64.StdEncoding.EncodeToString(plaintext),
		}

		path := fmt.Sprintf("transit/encrypt/%s", keyName)
		secret, err := vm.client.Logical().Write(path, data)
		if err != nil {
			return nil, errors.Wrapf(err, errors.ErrorTypeProvider, "failed to encrypt data")
		}

		ciphertext, ok := secret.Data["ciphertext"].(string)
		if !ok {
			return nil, errors.New(errors.ErrorTypeInternal, "invalid response from Vault transit engine")
		}

		return ciphertext, nil
	})

	if err != nil {
		return "", err
	}

	return result.(string), nil
}

// DecryptData decrypts data using Vault's transit engine
func (vm *VaultManager) DecryptData(ctx context.Context, keyName string, ciphertext string) ([]byte, error) {
	result, err := vm.circuitBreaker.Call(ctx, func() (interface{}, error) {
		data := map[string]interface{}{
			"ciphertext": ciphertext,
		}

		path := fmt.Sprintf("transit/decrypt/%s", keyName)
		secret, err := vm.client.Logical().Write(path, data)
		if err != nil {
			return nil, errors.Wrapf(err, errors.ErrorTypeProvider, "failed to decrypt data")
		}

		plaintextB64, ok := secret.Data["plaintext"].(string)
		if !ok {
			return nil, errors.New(errors.ErrorTypeInternal, "invalid response from Vault transit engine")
		}

		plaintext, err := base64.StdEncoding.DecodeString(plaintextB64)
		if err != nil {
			return nil, errors.Wrapf(err, errors.ErrorTypeInternal, "failed to decode plaintext")
		}

		return plaintext, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]byte), nil
}

// renewToken periodically renews the Vault token
func (vm *VaultManager) renewToken() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if _, err := vm.client.Auth().Token().RenewSelf(0); err != nil {
				vm.logger.Error().Err(err).Msg("failed to renew Vault token")
			} else {
				vm.logger.Debug().Msg("successfully renewed Vault token")
			}
		case <-vm.renewalStopChan:
			return
		}
	}
}

// Cache management functions

func (vm *VaultManager) getFromCache(path string) map[string]interface{} {
	vm.cacheMutex.RLock()
	defer vm.cacheMutex.RUnlock()

	if cached, ok := vm.cache[path]; ok {
		if time.Now().Before(cached.expiresAt) {
			return cached.data
		}
	}

	return nil
}

func (vm *VaultManager) addToCache(path string, data map[string]interface{}) {
	vm.cacheMutex.Lock()
	defer vm.cacheMutex.Unlock()

	vm.cache[path] = &cachedSecret{
		data:      data,
		expiresAt: time.Now().Add(vm.cacheTTL),
	}
}

func (vm *VaultManager) invalidateCache(path string) {
	vm.cacheMutex.Lock()
	defer vm.cacheMutex.Unlock()

	delete(vm.cache, path)
}

func (vm *VaultManager) cleanupCache() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		vm.cacheMutex.Lock()
		now := time.Now()
		for path, cached := range vm.cache {
			if now.After(cached.expiresAt) {
				delete(vm.cache, path)
			}
		}
		vm.cacheMutex.Unlock()
	}
}

// Close gracefully shuts down the Vault manager
func (vm *VaultManager) Close() {
	close(vm.renewalStopChan)
}

// Helper functions

func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// CloudCredentials represents cloud provider credentials
type CloudCredentials struct {
	Provider     string                   `json:"provider"`
	AWS          *AWSCredentials          `json:"aws,omitempty"`
	Azure        *AzureCredentials        `json:"azure,omitempty"`
	GCP          *GCPCredentials          `json:"gcp,omitempty"`
	DigitalOcean *DigitalOceanCredentials `json:"digitalocean,omitempty"`
}

// AWSCredentials represents AWS credentials
type AWSCredentials struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	SessionToken    string `json:"session_token,omitempty"`
	Region          string `json:"region"`
}

// AzureCredentials represents Azure credentials
type AzureCredentials struct {
	TenantID       string `json:"tenant_id"`
	ClientID       string `json:"client_id"`
	ClientSecret   string `json:"client_secret"`
	SubscriptionID string `json:"subscription_id"`
}

// GCPCredentials represents GCP credentials
type GCPCredentials struct {
	KeyJSON string `json:"key_json"`
}

// DigitalOceanCredentials represents DigitalOcean credentials
type DigitalOceanCredentials struct {
	Token string `json:"token"`
}
