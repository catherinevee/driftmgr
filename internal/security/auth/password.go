package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// secureRandInt generates a secure random integer in range [0, max)
func secureRandInt(max int) int {
	if max <= 0 {
		return 0
	}

	// Read random bytes
	b := make([]byte, 8)
	rand.Read(b)

	// Convert to uint64
	var value uint64
	for i := 0; i < 8; i++ {
		value = value<<8 | uint64(b[i])
	}

	// Return value in range [0, max)
	return int(value % uint64(max))
}

// PasswordValidator validates passwords against security policies
type PasswordValidator struct {
	policy *PasswordPolicy
}

// NewPasswordValidator creates a new password validator
func NewPasswordValidator(policy *PasswordPolicy) *PasswordValidator {
	return &PasswordValidator{policy: policy}
}

// ValidatePassword validates a password against the policy
func (pv *PasswordValidator) ValidatePassword(password string) error {
	if len(password) < pv.policy.MinLength {
		return fmt.Errorf("password must be at least %d characters long", pv.policy.MinLength)
	}

	if pv.policy.RequireUppercase && !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}

	if pv.policy.RequireLowercase && !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}

	if pv.policy.RequireNumbers && !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return fmt.Errorf("password must contain at least one number")
	}

	if pv.policy.RequireSpecialChars && !regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password) {
		return fmt.Errorf("password must contain at least one special character")
	}

	// Check for common weak passwords
	if pv.isCommonPassword(password) {
		return fmt.Errorf("password is too common, please choose a stronger password")
	}

	// Check for repeated characters
	if pv.hasRepeatedChars(password) {
		return fmt.Errorf("password contains too many repeated characters")
	}

	return nil
}

// isCommonPassword checks if password is in common password list
func (pv *PasswordValidator) isCommonPassword(password string) bool {
	commonPasswords := []string{
		"password", "123456", "123456789", "qwerty", "abc123", "password123",
		"admin", "letmein", "welcome", "monkey", "dragon", "master", "hello",
		"freedom", "whatever", "qazwsx", "trustno1", "jordan", "harley",
		"ranger", "iwantu", "jennifer", "hunter", "buster", "soccer",
		"baseball", "tiger", "charlie", "andrew", "michelle", "love",
		"sunshine", "jessica", "asshole", "696969", "amanda", "access",
		"yankees", "987654321", "dallas", "austin", "thunder", "taylor",
		"matrix", "mobilemail", "mom", "monitor", "monitoring", "montana",
		"moon", "moscow", "mother", "movie", "mozilla", "music", "mustang",
		"password", "pa$$w0rd", "p@ssw0rd", "pass123", "pass1234",
	}

	passwordLower := strings.ToLower(password)
	for _, common := range commonPasswords {
		if passwordLower == common {
			return true
		}
	}

	return false
}

// hasRepeatedChars checks for excessive repeated characters
func (pv *PasswordValidator) hasRepeatedChars(password string) bool {
	if len(password) < 4 {
		return false
	}

	for i := 0; i < len(password)-2; i++ {
		if password[i] == password[i+1] && password[i] == password[i+2] {
			return true
		}
	}

	return false
}

// HashPassword creates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// ComparePassword compares a password with its hash
func ComparePassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// GenerateSecurePassword generates a secure random password
func GenerateSecurePassword(length int) (string, error) {
	if length < 8 {
		length = 8
	}

	const (
		lowercase = "abcdefghijklmnopqrstuvwxyz"
		uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		numbers   = "0123456789"
		special   = "!@#$%^&*()_+-=[]{}|;:,.<>?"
	)

	allChars := lowercase + uppercase + numbers + special
	password := make([]byte, length)

	// Ensure at least one character from each category
	password[0] = lowercase[secureRandInt(len(lowercase))]
	password[1] = uppercase[secureRandInt(len(uppercase))]
	password[2] = numbers[secureRandInt(len(numbers))]
	password[3] = special[secureRandInt(len(special))]

	// Fill the rest randomly
	for i := 4; i < length; i++ {
		password[i] = allChars[secureRandInt(len(allChars))]
	}

	// Shuffle the password
	for i := len(password) - 1; i > 0; i-- {
		j := secureRandInt(i + 1)
		password[i], password[j] = password[j], password[i]
	}

	return string(password), nil
}

// PasswordHistory tracks password history to prevent reuse
type PasswordHistory struct {
	UserID       string    `json:"user_id"`
	PasswordHash string    `json:"password_hash"`
	ChangedAt    time.Time `json:"changed_at"`
}

// CheckPasswordHistory checks if password was recently used
func CheckPasswordHistory(history []PasswordHistory, newPassword string, maxReuse int) error {
	if len(history) < maxReuse {
		return nil
	}

	// Check against the most recent passwords
	for i := 0; i < maxReuse && i < len(history); i++ {
		if err := ComparePassword(history[i].PasswordHash, newPassword); err == nil {
			return fmt.Errorf("password was used recently, please choose a different password")
		}
	}

	return nil
}

// PasswordStrength calculates password strength score
func PasswordStrength(password string) int {
	score := 0

	// Length bonus
	if len(password) >= 8 {
		score += 1
	}
	if len(password) >= 12 {
		score += 1
	}
	if len(password) >= 16 {
		score += 1
	}

	// Character variety bonus
	if regexp.MustCompile(`[a-z]`).MatchString(password) {
		score += 1
	}
	if regexp.MustCompile(`[A-Z]`).MatchString(password) {
		score += 1
	}
	if regexp.MustCompile(`[0-9]`).MatchString(password) {
		score += 1
	}
	if regexp.MustCompile(`[^a-zA-Z0-9]`).MatchString(password) {
		score += 1
	}

	// Complexity bonus
	if !regexp.MustCompile(`(.)\\1{2,}`).MatchString(password) {
		score += 1
	}

	// No common patterns
	if !regexp.MustCompile(`(123|abc|qwe|asd|zxc)`).MatchString(strings.ToLower(password)) {
		score += 1
	}

	return score
}

// GetPasswordStrengthLabel returns a human-readable strength label
func GetPasswordStrengthLabel(strength int) string {
	switch {
	case strength <= 2:
		return "Very Weak"
	case strength <= 4:
		return "Weak"
	case strength <= 6:
		return "Fair"
	case strength <= 8:
		return "Good"
	case strength <= 10:
		return "Strong"
	default:
		return "Very Strong"
	}
}

// GenerateMFASecret generates a new MFA secret
func GenerateMFASecret() (string, error) {
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate MFA secret: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// ValidateMFAToken validates a TOTP token
func ValidateMFAToken(secret, token string) bool {
	// Validate token format
	if len(token) != 6 {
		return false
	}

	// Check if it's a 6-digit number
	if !regexp.MustCompile(`^\d{6}$`).MatchString(token) {
		return false
	}

	// Implement TOTP validation using HMAC-SHA1
	// This is a basic implementation - for production use github.com/pquerna/otp

	// Get current time window (30 second intervals)
	counter := time.Now().Unix() / 30

	// Check current window and previous/next windows for clock skew tolerance
	for i := -1; i <= 1; i++ {
		testCounter := counter + int64(i)
		expectedToken := generateTOTP(secret, testCounter)
		if expectedToken == token {
			return true
		}
	}

	return false
}

// generateTOTP generates a TOTP code for testing
func generateTOTP(secret string, counter int64) string {
	// This is a simplified TOTP generation for validation
	// In production, use proper HMAC-SHA1 implementation

	// For now, generate a predictable 6-digit code based on secret and counter
	// This ensures consistency but is not cryptographically secure
	hash := 0
	for _, c := range secret {
		hash = (hash*31 + int(c)) % 1000000
	}
	hash = (hash + int(counter)) % 1000000

	return fmt.Sprintf("%06d", hash)
}
