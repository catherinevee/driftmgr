package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// PasswordService handles password hashing and verification
type PasswordService struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

// NewPasswordService creates a new password service with default parameters
func NewPasswordService() *PasswordService {
	return &PasswordService{
		memory:      64 * 1024, // 64 MB
		iterations:  3,
		parallelism: 2,
		saltLength:  16,
		keyLength:   32,
	}
}

// HashPassword hashes a password using Argon2id
func (p *PasswordService) HashPassword(password string) (string, error) {
	// Generate a random salt
	salt, err := p.generateRandomBytes(p.saltLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash the password
	hash := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, p.parallelism, p.keyLength)

	// Encode the hash and salt
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Format: argon2id$v=19$m=65536,t=3,p=2$salt$hash
	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.memory, p.iterations, p.parallelism, b64Salt, b64Hash)

	return encodedHash, nil
}

// VerifyPassword verifies a password against a hash
func (p *PasswordService) VerifyPassword(password, encodedHash string) (bool, error) {
	// Parse the encoded hash
	salt, hash, params, err := p.decodeHash(encodedHash)
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// Hash the provided password with the same parameters
	otherHash := argon2.IDKey([]byte(password), salt, params.iterations, params.memory, params.parallelism, params.keyLength)

	// Compare the hashes using constant time comparison
	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}

	return false, nil
}

// generateRandomBytes generates random bytes of the specified length
func (p *PasswordService) generateRandomBytes(n uint32) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// decodeHash decodes an Argon2 hash string
func (p *PasswordService) decodeHash(encodedHash string) (salt, hash []byte, params *argon2Params, err error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, nil, nil, fmt.Errorf("invalid hash format")
	}

	// Check algorithm
	if parts[1] != "argon2id" {
		return nil, nil, nil, fmt.Errorf("unsupported algorithm: %s", parts[1])
	}

	// Parse version
	var version int
	_, err = fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid version: %w", err)
	}
	if version != argon2.Version {
		return nil, nil, nil, fmt.Errorf("incompatible version: %d", version)
	}

	// Parse parameters
	params = &argon2Params{}
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.memory, &params.iterations, &params.parallelism)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Decode salt
	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid salt: %w", err)
	}

	// Decode hash
	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid hash: %w", err)
	}

	return salt, hash, params, nil
}

// argon2Params represents Argon2 parameters
type argon2Params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	keyLength   uint32
}

// ValidatePasswordStrength validates password strength
func (p *PasswordService) ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	if len(password) > 128 {
		return fmt.Errorf("password must be no more than 128 characters long")
	}

	// Check for at least one lowercase letter
	hasLower := false
	// Check for at least one uppercase letter
	hasUpper := false
	// Check for at least one digit
	hasDigit := false
	// Check for at least one special character
	hasSpecial := false

	for _, char := range password {
		switch {
		case 'a' <= char && char <= 'z':
			hasLower = true
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case '0' <= char && char <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}

	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}

// GenerateRandomPassword generates a random password
func (p *PasswordService) GenerateRandomPassword(length int) (string, error) {
	if length < 8 {
		length = 8
	}
	if length > 128 {
		length = 128
	}

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{}|;:,.<>?"
	charsetLen := len(charset)

	password := make([]byte, length)
	for i := range password {
		randomByte := make([]byte, 1)
		_, err := rand.Read(randomByte)
		if err != nil {
			return "", fmt.Errorf("failed to generate random byte: %w", err)
		}
		password[i] = charset[randomByte[0]%byte(charsetLen)]
	}

	return string(password), nil
}

// GenerateAPIKey generates a random API key
func (p *PasswordService) GenerateAPIKey() (string, string, error) {
	// Generate 32 random bytes
	keyBytes := make([]byte, 32)
	_, err := rand.Read(keyBytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Encode as base64
	key := base64.URLEncoding.EncodeToString(keyBytes)

	// Create prefix for display (first 8 characters)
	_ = key[:8] + "..."

	// Hash the full key for storage
	hashedKey, err := p.HashPassword(key)
	if err != nil {
		return "", "", fmt.Errorf("failed to hash API key: %w", err)
	}

	return key, hashedKey, nil
}

// VerifyAPIKey verifies an API key against its hash
func (p *PasswordService) VerifyAPIKey(key, hashedKey string) (bool, error) {
	return p.VerifyPassword(key, hashedKey)
}
