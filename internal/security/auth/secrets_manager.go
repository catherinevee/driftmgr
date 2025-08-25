package security

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/hashicorp/vault/api"
	"golang.org/x/crypto/pbkdf2"
)

// SecretsManager provides secure secrets management
type SecretsManager struct {
	provider     SecretsProvider
	cache        *SecretCache
	encryptor    *Encryptor
	audit        *AuditLogger
	mu           sync.RWMutex
	rotationJobs map[string]*RotationJob
}

// SecretsProvider interface for different secret storage backends
type SecretsProvider interface {
	GetSecret(ctx context.Context, key string) (*Secret, error)
	SetSecret(ctx context.Context, key string, secret *Secret) error
	DeleteSecret(ctx context.Context, key string) error
	ListSecrets(ctx context.Context) ([]string, error)
	RotateSecret(ctx context.Context, key string) (*Secret, error)
}

// Secret represents a secret value with metadata
type Secret struct {
	Key       string                 `json:"key"`
	Value     string                 `json:"value"`
	Type      SecretType             `json:"type"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
	Version   int                    `json:"version"`
	Encrypted bool                   `json:"encrypted"`
	Tags      map[string]string      `json:"tags"`
}

// SecretType represents the type of secret
type SecretType string

const (
	SecretTypePassword    SecretType = "password"
	SecretTypeAPIKey      SecretType = "api_key"
	SecretTypeCertificate SecretType = "certificate"
	SecretTypeSSHKey      SecretType = "ssh_key"
	SecretTypeToken       SecretType = "token"
	SecretTypeGeneric     SecretType = "generic"
)

// SecretCache provides in-memory caching of secrets
type SecretCache struct {
	secrets map[string]*CachedSecret
	ttl     time.Duration
	mu      sync.RWMutex
}

// CachedSecret represents a cached secret
type CachedSecret struct {
	Secret    *Secret
	CachedAt  time.Time
	ExpiresAt time.Time
}

// Encryptor provides encryption/decryption capabilities
type Encryptor struct {
	key       []byte
	algorithm string
}

// AuditLogger logs secret access
type AuditLogger struct {
	logFile *os.File
	mu      sync.Mutex
}

// RotationJob represents a secret rotation job
type RotationJob struct {
	Key      string
	Interval time.Duration
	LastRun  time.Time
	NextRun  time.Time
	Active   bool
	cancel   context.CancelFunc
}

// NewSecretsManager creates a new secrets manager
func NewSecretsManager(providerType string, config map[string]interface{}) (*SecretsManager, error) {
	var provider SecretsProvider
	var err error

	switch providerType {
	case "vault":
		provider, err = NewVaultProvider(config)
	case "aws":
		provider, err = NewAWSSecretsProvider(config)
	case "local":
		provider, err = NewLocalSecretsProvider(config)
	default:
		return nil, fmt.Errorf("unsupported secrets provider: %s", providerType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create secrets provider: %w", err)
	}

	// Create encryptor with default key
	encryptor, err := NewEncryptor()
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	// Create audit logger
	auditLogger, err := NewAuditLogger("secrets_audit.log")
	if err != nil {
		return nil, fmt.Errorf("failed to create audit logger: %w", err)
	}

	return &SecretsManager{
		provider: provider,
		cache: &SecretCache{
			secrets: make(map[string]*CachedSecret),
			ttl:     5 * time.Minute,
		},
		encryptor:    encryptor,
		audit:        auditLogger,
		rotationJobs: make(map[string]*RotationJob),
	}, nil
}

// GetSecret retrieves a secret
func (sm *SecretsManager) GetSecret(ctx context.Context, key string) (*Secret, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Check cache first
	if cached := sm.cache.Get(key); cached != nil {
		sm.audit.Log("secret_accessed", key, "cache_hit")
		return cached, nil
	}

	// Retrieve from provider
	secret, err := sm.provider.GetSecret(ctx, key)
	if err != nil {
		sm.audit.Log("secret_access_failed", key, err.Error())
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	// Decrypt if encrypted
	if secret.Encrypted {
		decrypted, err := sm.encryptor.Decrypt(secret.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt secret: %w", err)
		}
		secret.Value = decrypted
	}

	// Cache the secret
	sm.cache.Set(key, secret)
	sm.audit.Log("secret_accessed", key, "provider")

	return secret, nil
}

// SetSecret stores a secret
func (sm *SecretsManager) SetSecret(ctx context.Context, key string, value string, secretType SecretType) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Encrypt the value
	encrypted, err := sm.encryptor.Encrypt(value)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	secret := &Secret{
		Key:       key,
		Value:     encrypted,
		Type:      secretType,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
		Encrypted: true,
		Tags:      make(map[string]string),
	}

	// Store in provider
	if err := sm.provider.SetSecret(ctx, key, secret); err != nil {
		sm.audit.Log("secret_set_failed", key, err.Error())
		return fmt.Errorf("failed to set secret: %w", err)
	}

	// Invalidate cache
	sm.cache.Delete(key)
	sm.audit.Log("secret_set", key, string(secretType))

	return nil
}

// DeleteSecret removes a secret
func (sm *SecretsManager) DeleteSecret(ctx context.Context, key string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if err := sm.provider.DeleteSecret(ctx, key); err != nil {
		sm.audit.Log("secret_delete_failed", key, err.Error())
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	sm.cache.Delete(key)
	sm.audit.Log("secret_deleted", key, "")

	return nil
}

// RotateSecret rotates a secret
func (sm *SecretsManager) RotateSecret(ctx context.Context, key string) (*Secret, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	newSecret, err := sm.provider.RotateSecret(ctx, key)
	if err != nil {
		sm.audit.Log("secret_rotation_failed", key, err.Error())
		return nil, fmt.Errorf("failed to rotate secret: %w", err)
	}

	sm.cache.Delete(key)
	sm.audit.Log("secret_rotated", key, fmt.Sprintf("version:%d", newSecret.Version))

	return newSecret, nil
}

// EnableAutoRotation enables automatic secret rotation
func (sm *SecretsManager) EnableAutoRotation(key string, interval time.Duration) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.rotationJobs[key]; exists {
		return fmt.Errorf("rotation already enabled for key: %s", key)
	}

	ctx, cancel := context.WithCancel(context.Background())
	job := &RotationJob{
		Key:      key,
		Interval: interval,
		LastRun:  time.Now(),
		NextRun:  time.Now().Add(interval),
		Active:   true,
		cancel:   cancel,
	}

	sm.rotationJobs[key] = job

	// Start rotation goroutine
	go sm.runRotationJob(ctx, job)

	sm.audit.Log("rotation_enabled", key, fmt.Sprintf("interval:%v", interval))
	return nil
}

// runRotationJob runs a rotation job
func (sm *SecretsManager) runRotationJob(ctx context.Context, job *RotationJob) {
	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if _, err := sm.RotateSecret(ctx, job.Key); err != nil {
				sm.audit.Log("auto_rotation_failed", job.Key, err.Error())
			} else {
				job.LastRun = time.Now()
				job.NextRun = time.Now().Add(job.Interval)
			}
		case <-ctx.Done():
			return
		}
	}
}

// VaultProvider implements SecretsProvider for HashiCorp Vault
type VaultProvider struct {
	client *api.Client
	path   string
}

// NewVaultProvider creates a new Vault provider
func NewVaultProvider(config map[string]interface{}) (*VaultProvider, error) {
	vaultConfig := api.DefaultConfig()

	if addr, ok := config["address"].(string); ok {
		vaultConfig.Address = addr
	}

	client, err := api.NewClient(vaultConfig)
	if err != nil {
		return nil, err
	}

	if token, ok := config["token"].(string); ok {
		client.SetToken(token)
	}

	path := "secret/data"
	if p, ok := config["path"].(string); ok {
		path = p
	}

	return &VaultProvider{
		client: client,
		path:   path,
	}, nil
}

// GetSecret retrieves a secret from Vault
func (vp *VaultProvider) GetSecret(ctx context.Context, key string) (*Secret, error) {
	logical := vp.client.Logical()
	secret, err := logical.ReadWithContext(ctx, fmt.Sprintf("%s/%s", vp.path, key))
	if err != nil {
		return nil, err
	}

	if secret == nil || secret.Data == nil {
		return nil, errors.New("secret not found")
	}

	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid secret format")
	}

	value, ok := data["value"].(string)
	if !ok {
		return nil, errors.New("secret value not found")
	}

	return &Secret{
		Key:       key,
		Value:     value,
		UpdatedAt: time.Now(),
		Version:   1,
	}, nil
}

// SetSecret stores a secret in Vault
func (vp *VaultProvider) SetSecret(ctx context.Context, key string, secret *Secret) error {
	logical := vp.client.Logical()

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"value":     secret.Value,
			"type":      string(secret.Type),
			"encrypted": secret.Encrypted,
			"metadata":  secret.Metadata,
		},
	}

	_, err := logical.WriteWithContext(ctx, fmt.Sprintf("%s/%s", vp.path, key), data)
	return err
}

// DeleteSecret deletes a secret from Vault
func (vp *VaultProvider) DeleteSecret(ctx context.Context, key string) error {
	logical := vp.client.Logical()
	_, err := logical.DeleteWithContext(ctx, fmt.Sprintf("%s/%s", vp.path, key))
	return err
}

// ListSecrets lists all secrets in Vault
func (vp *VaultProvider) ListSecrets(ctx context.Context) ([]string, error) {
	logical := vp.client.Logical()
	secret, err := logical.ListWithContext(ctx, vp.path)
	if err != nil {
		return nil, err
	}

	if secret == nil || secret.Data == nil {
		return []string{}, nil
	}

	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return []string{}, nil
	}

	result := make([]string, len(keys))
	for i, k := range keys {
		result[i] = k.(string)
	}

	return result, nil
}

// RotateSecret rotates a secret in Vault
func (vp *VaultProvider) RotateSecret(ctx context.Context, key string) (*Secret, error) {
	// Get current secret
	current, err := vp.GetSecret(ctx, key)
	if err != nil {
		return nil, err
	}

	// Generate new value (example for API key)
	newValue := generateSecureToken(32)

	current.Value = newValue
	current.Version++
	current.UpdatedAt = time.Now()

	// Store new version
	if err := vp.SetSecret(ctx, key, current); err != nil {
		return nil, err
	}

	return current, nil
}

// AWSSecretsProvider implements SecretsProvider for AWS Secrets Manager
type AWSSecretsProvider struct {
	client *secretsmanager.Client
}

// NewAWSSecretsProvider creates a new AWS Secrets Manager provider
func NewAWSSecretsProvider(config map[string]interface{}) (*AWSSecretsProvider, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}

	return &AWSSecretsProvider{
		client: secretsmanager.NewFromConfig(cfg),
	}, nil
}

// GetSecret retrieves a secret from AWS Secrets Manager
func (asp *AWSSecretsProvider) GetSecret(ctx context.Context, key string) (*Secret, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: &key,
	}

	result, err := asp.client.GetSecretValue(ctx, input)
	if err != nil {
		return nil, err
	}

	return &Secret{
		Key:       key,
		Value:     *result.SecretString,
		UpdatedAt: *result.CreatedDate,
		Version:   1,
	}, nil
}

// SetSecret stores a secret in AWS Secrets Manager
func (asp *AWSSecretsProvider) SetSecret(ctx context.Context, key string, secret *Secret) error {
	value, err := json.Marshal(secret)
	if err != nil {
		return err
	}

	valueStr := string(value)
	input := &secretsmanager.CreateSecretInput{
		Name:         &key,
		SecretString: &valueStr,
	}

	_, err = asp.client.CreateSecret(ctx, input)
	return err
}

// DeleteSecret deletes a secret from AWS Secrets Manager
func (asp *AWSSecretsProvider) DeleteSecret(ctx context.Context, key string) error {
	forceDelete := true
	input := &secretsmanager.DeleteSecretInput{
		SecretId:                   &key,
		ForceDeleteWithoutRecovery: &forceDelete,
	}

	_, err := asp.client.DeleteSecret(ctx, input)
	return err
}

// ListSecrets lists all secrets in AWS Secrets Manager
func (asp *AWSSecretsProvider) ListSecrets(ctx context.Context) ([]string, error) {
	input := &secretsmanager.ListSecretsInput{}
	result, err := asp.client.ListSecrets(ctx, input)
	if err != nil {
		return nil, err
	}

	keys := make([]string, len(result.SecretList))
	for i, secret := range result.SecretList {
		keys[i] = *secret.Name
	}

	return keys, nil
}

// RotateSecret rotates a secret in AWS Secrets Manager
func (asp *AWSSecretsProvider) RotateSecret(ctx context.Context, key string) (*Secret, error) {
	// AWS Secrets Manager has built-in rotation support
	// This is a simplified implementation
	return asp.GetSecret(ctx, key)
}

// LocalSecretsProvider implements SecretsProvider for local file storage
type LocalSecretsProvider struct {
	directory string
	mu        sync.RWMutex
}

// NewLocalSecretsProvider creates a new local file provider
func NewLocalSecretsProvider(config map[string]interface{}) (*LocalSecretsProvider, error) {
	dir := "./secrets"
	if d, ok := config["directory"].(string); ok {
		dir = d
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	return &LocalSecretsProvider{
		directory: dir,
	}, nil
}

// GetSecret retrieves a secret from local storage
func (lsp *LocalSecretsProvider) GetSecret(ctx context.Context, key string) (*Secret, error) {
	lsp.mu.RLock()
	defer lsp.mu.RUnlock()

	path := fmt.Sprintf("%s/%s.json", lsp.directory, key)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var secret Secret
	if err := json.Unmarshal(data, &secret); err != nil {
		return nil, err
	}

	return &secret, nil
}

// SetSecret stores a secret in local storage
func (lsp *LocalSecretsProvider) SetSecret(ctx context.Context, key string, secret *Secret) error {
	lsp.mu.Lock()
	defer lsp.mu.Unlock()

	data, err := json.Marshal(secret)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s/%s.json", lsp.directory, key)
	return os.WriteFile(path, data, 0600)
}

// DeleteSecret deletes a secret from local storage
func (lsp *LocalSecretsProvider) DeleteSecret(ctx context.Context, key string) error {
	lsp.mu.Lock()
	defer lsp.mu.Unlock()

	path := fmt.Sprintf("%s/%s.json", lsp.directory, key)
	return os.Remove(path)
}

// ListSecrets lists all secrets in local storage
func (lsp *LocalSecretsProvider) ListSecrets(ctx context.Context) ([]string, error) {
	lsp.mu.RLock()
	defer lsp.mu.RUnlock()

	entries, err := os.ReadDir(lsp.directory)
	if err != nil {
		return nil, err
	}

	var keys []string
	for _, entry := range entries {
		if !entry.IsDir() && len(entry.Name()) > 5 {
			keys = append(keys, entry.Name()[:len(entry.Name())-5])
		}
	}

	return keys, nil
}

// RotateSecret rotates a secret in local storage
func (lsp *LocalSecretsProvider) RotateSecret(ctx context.Context, key string) (*Secret, error) {
	secret, err := lsp.GetSecret(ctx, key)
	if err != nil {
		return nil, err
	}

	// Generate new value
	secret.Value = generateSecureToken(32)
	secret.Version++
	secret.UpdatedAt = time.Now()

	if err := lsp.SetSecret(ctx, key, secret); err != nil {
		return nil, err
	}

	return secret, nil
}

// Cache methods

// Get retrieves a secret from cache
func (sc *SecretCache) Get(key string) *Secret {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	cached, exists := sc.secrets[key]
	if !exists {
		return nil
	}

	if time.Now().After(cached.ExpiresAt) {
		delete(sc.secrets, key)
		return nil
	}

	return cached.Secret
}

// Set stores a secret in cache
func (sc *SecretCache) Set(key string, secret *Secret) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.secrets[key] = &CachedSecret{
		Secret:    secret,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(sc.ttl),
	}
}

// Delete removes a secret from cache
func (sc *SecretCache) Delete(key string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.secrets, key)
}

// Encryptor methods

// NewEncryptor creates a new encryptor
func NewEncryptor() (*Encryptor, error) {
	// Generate or load encryption key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	return &Encryptor{
		key:       key,
		algorithm: "AES-256-GCM",
	}, nil
}

// Encrypt encrypts a value
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(e.key)
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

// Decrypt decrypts a value
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// AuditLogger methods

// NewAuditLogger creates a new audit logger
func NewAuditLogger(filename string) (*AuditLogger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}

	return &AuditLogger{
		logFile: file,
	}, nil
}

// Log writes an audit log entry
func (al *AuditLogger) Log(action, key, details string) {
	al.mu.Lock()
	defer al.mu.Unlock()

	entry := map[string]interface{}{
		"timestamp": time.Now().UTC(),
		"action":    action,
		"key":       key,
		"details":   details,
	}

	data, _ := json.Marshal(entry)
	al.logFile.Write(append(data, '\n'))
}

// Close closes the audit logger
func (al *AuditLogger) Close() error {
	return al.logFile.Close()
}

// Utility functions

// generateSecureToken generates a secure random token
func generateSecureToken(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// deriveKey derives an encryption key from a password
func deriveKey(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)
}

// generateRSAKeyPair generates an RSA key pair
func generateRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}

// encodePrivateKey encodes an RSA private key to PEM format
func encodePrivateKey(privateKey *rsa.PrivateKey) string {
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}
	return string(pem.EncodeToMemory(privateKeyPEM))
}

// encodePublicKey encodes an RSA public key to PEM format
func encodePublicKey(publicKey *rsa.PublicKey) (string, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}
	publicKeyPEM := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	return string(pem.EncodeToMemory(publicKeyPEM)), nil
}
