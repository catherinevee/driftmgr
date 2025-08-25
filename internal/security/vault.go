package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/logging"
	"golang.org/x/crypto/pbkdf2"
)

// SecureVault provides encrypted storage for sensitive data
type SecureVault struct {
	mu        sync.RWMutex
	filePath  string
	masterKey []byte
	data      map[string]*SecureEntry
	logger    *logging.Logger
}

// SecureEntry represents an encrypted entry in the vault
type SecureEntry struct {
	Value       string            `json:"value"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	AccessCount int               `json:"access_count"`
	LastAccess  time.Time         `json:"last_access"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// VaultConfig holds configuration for the vault
type VaultConfig struct {
	FilePath  string
	MasterKey string
	AutoLock  bool
	LockAfter time.Duration
}

// NewSecureVault creates a new secure vault
func NewSecureVault(config *VaultConfig) (*SecureVault, error) {
	// Derive key from master password
	salt := []byte("driftmgr-vault-salt-v1") // In production, use random salt stored separately
	key := pbkdf2.Key([]byte(config.MasterKey), salt, 100000, 32, sha256.New)

	vault := &SecureVault{
		filePath:  config.FilePath,
		masterKey: key,
		data:      make(map[string]*SecureEntry),
		logger:    logging.GetLogger(),
	}

	// Create vault directory if needed
	if err := os.MkdirAll(filepath.Dir(config.FilePath), 0700); err != nil {
		return nil, fmt.Errorf("failed to create vault directory: %w", err)
	}

	// Load existing vault if it exists
	if _, err := os.Stat(config.FilePath); err == nil {
		if err := vault.load(); err != nil {
			return nil, fmt.Errorf("failed to load vault: %w", err)
		}
	}

	// Start auto-lock if configured
	if config.AutoLock {
		go vault.autoLockRoutine(config.LockAfter)
	}

	vault.logger.Info("Secure vault initialized", map[string]interface{}{
		"path": config.FilePath,
	})

	return vault, nil
}

// Store securely stores a value in the vault
func (v *SecureVault) Store(key string, value string, metadata ...map[string]string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Encrypt the value
	encrypted, err := v.encrypt(value)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	entry := &SecureEntry{
		Value:       encrypted,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		AccessCount: 0,
		LastAccess:  time.Now(),
	}

	if len(metadata) > 0 {
		entry.Metadata = metadata[0]
	}

	v.data[key] = entry

	// Persist to disk
	if err := v.save(); err != nil {
		return fmt.Errorf("failed to save vault: %w", err)
	}

	// Audit log
	logging.Audit("vault_store", "system", "success", map[string]interface{}{
		"key": key,
	})

	return nil
}

// Retrieve securely retrieves a value from the vault
func (v *SecureVault) Retrieve(key string) (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	entry, exists := v.data[key]
	if !exists {
		return "", errors.New("key not found")
	}

	// Decrypt the value
	decrypted, err := v.decrypt(entry.Value)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	// Update access metadata
	entry.AccessCount++
	entry.LastAccess = time.Now()

	// Save updated metadata
	v.save()

	// Audit log
	logging.Audit("vault_retrieve", "system", "success", map[string]interface{}{
		"key": key,
	})

	return decrypted, nil
}

// Delete removes an entry from the vault
func (v *SecureVault) Delete(key string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if _, exists := v.data[key]; !exists {
		return errors.New("key not found")
	}

	delete(v.data, key)

	// Persist changes
	if err := v.save(); err != nil {
		return fmt.Errorf("failed to save vault: %w", err)
	}

	// Audit log
	logging.Audit("vault_delete", "system", "success", map[string]interface{}{
		"key": key,
	})

	return nil
}

// List returns all keys in the vault (not values)
func (v *SecureVault) List() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	keys := make([]string, 0, len(v.data))
	for key := range v.data {
		keys = append(keys, key)
	}

	return keys
}

// encrypt encrypts data using AES-GCM
func (v *SecureVault) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(v.masterKey)
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

// decrypt decrypts data using AES-GCM
func (v *SecureVault) decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(v.masterKey)
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

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// save persists the vault to disk
func (v *SecureVault) save() error {
	// Marshal data
	data, err := json.Marshal(v.data)
	if err != nil {
		return err
	}

	// Encrypt entire vault file
	encrypted, err := v.encrypt(string(data))
	if err != nil {
		return err
	}

	// Write to temporary file first
	tempFile := v.filePath + ".tmp"
	if err := os.WriteFile(tempFile, []byte(encrypted), 0600); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tempFile, v.filePath)
}

// load loads the vault from disk
func (v *SecureVault) load() error {
	// Read file
	data, err := os.ReadFile(v.filePath)
	if err != nil {
		return err
	}

	// Decrypt vault file
	decrypted, err := v.decrypt(string(data))
	if err != nil {
		return err
	}

	// Unmarshal data
	return json.Unmarshal([]byte(decrypted), &v.data)
}

// autoLockRoutine automatically locks the vault after inactivity
func (v *SecureVault) autoLockRoutine(lockAfter time.Duration) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		v.mu.RLock()
		shouldLock := true
		for _, entry := range v.data {
			if time.Since(entry.LastAccess) < lockAfter {
				shouldLock = false
				break
			}
		}
		v.mu.RUnlock()

		if shouldLock {
			v.Lock()
		}
	}
}

// Lock clears the master key from memory
func (v *SecureVault) Lock() {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Clear master key
	for i := range v.masterKey {
		v.masterKey[i] = 0
	}
	v.masterKey = nil

	// Clear cached data
	v.data = make(map[string]*SecureEntry)

	v.logger.Info("Vault locked")
}

// Unlock re-establishes the master key
func (v *SecureVault) Unlock(masterKey string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Re-derive key
	salt := []byte("driftmgr-vault-salt-v1")
	v.masterKey = pbkdf2.Key([]byte(masterKey), salt, 100000, 32, sha256.New)

	// Reload vault
	if err := v.load(); err != nil {
		// Clear key on failure
		for i := range v.masterKey {
			v.masterKey[i] = 0
		}
		v.masterKey = nil
		return fmt.Errorf("failed to unlock vault: %w", err)
	}

	v.logger.Info("Vault unlocked")
	return nil
}

// IsLocked checks if the vault is locked
func (v *SecureVault) IsLocked() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.masterKey == nil
}

// SecureCredentialStore provides secure storage for cloud credentials
type SecureCredentialStore struct {
	vault  *SecureVault
	logger *logging.Logger
}

// NewSecureCredentialStore creates a new secure credential store
func NewSecureCredentialStore(vaultPath string) (*SecureCredentialStore, error) {
	// Get master key from environment or prompt user
	masterKey := os.Getenv("DRIFTMGR_VAULT_KEY")
	if masterKey == "" {
		// In production, prompt user for password
		masterKey = "default-dev-key" // NEVER use this in production
	}

	vault, err := NewSecureVault(&VaultConfig{
		FilePath:  vaultPath,
		MasterKey: masterKey,
		AutoLock:  true,
		LockAfter: 30 * time.Minute,
	})
	if err != nil {
		return nil, err
	}

	return &SecureCredentialStore{
		vault:  vault,
		logger: logging.GetLogger(),
	}, nil
}

// StoreCredential securely stores a cloud credential
func (s *SecureCredentialStore) StoreCredential(provider string, credential map[string]string) error {
	// Serialize credential
	data, err := json.Marshal(credential)
	if err != nil {
		return err
	}

	// Store in vault
	key := fmt.Sprintf("credential:%s", provider)
	return s.vault.Store(key, string(data), map[string]string{
		"provider": provider,
		"type":     "credential",
	})
}

// GetCredential retrieves a cloud credential
func (s *SecureCredentialStore) GetCredential(provider string) (map[string]string, error) {
	key := fmt.Sprintf("credential:%s", provider)

	// Retrieve from vault
	data, err := s.vault.Retrieve(key)
	if err != nil {
		return nil, err
	}

	// Deserialize credential
	var credential map[string]string
	if err := json.Unmarshal([]byte(data), &credential); err != nil {
		return nil, err
	}

	return credential, nil
}

// DeleteCredential removes a cloud credential
func (s *SecureCredentialStore) DeleteCredential(provider string) error {
	key := fmt.Sprintf("credential:%s", provider)
	return s.vault.Delete(key)
}

// ListProviders returns all providers with stored credentials
func (s *SecureCredentialStore) ListProviders() []string {
	keys := s.vault.List()
	providers := make([]string, 0)

	for _, key := range keys {
		if len(key) > 11 && key[:11] == "credential:" {
			providers = append(providers, key[11:])
		}
	}

	return providers
}
