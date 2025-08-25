package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// User represents an authenticated user
type User struct {
	ID                  string       `json:"id"`
	Username            string       `json:"username"`
	Password            string       `json:"-"` // Never expose in JSON
	Role                UserRole     `json:"role"`
	Created             time.Time    `json:"created"`
	LastLogin           time.Time    `json:"last_login"`
	Email               string       `json:"email,omitempty"`
	FailedLoginAttempts int          `json:"-"`
	FailedLogins        int          `json:"failed_logins"` // Alias for compatibility
	LockedUntil         *time.Time   `json:"-"`
	MFAEnabled          bool         `json:"mfa_enabled"`
	MFASecret           string       `json:"-"`
	PasswordChangedAt   time.Time    `json:"-"`
	Permissions         []Permission `json:"permissions,omitempty"`
	APIKey              string       `json:"api_key,omitempty"`
}

// UserRole defines user permissions
type UserRole string

const (
	RoleRoot     UserRole = "root"
	RoleReadOnly UserRole = "readonly"
)

// Permission defines what actions a user can perform
type Permission string

const (
	PermissionViewDashboard      Permission = "view_dashboard"
	PermissionViewResources      Permission = "view_resources"
	PermissionViewDrift          Permission = "view_drift"
	PermissionViewCosts          Permission = "view_costs"
	PermissionViewSecurity       Permission = "view_security"
	PermissionViewCompliance     Permission = "view_compliance"
	PermissionExecuteDiscovery   Permission = "execute_discovery"
	PermissionExecuteAnalysis    Permission = "execute_analysis"
	PermissionExecuteRemediation Permission = "execute_remediation"
	PermissionManageUsers        Permission = "manage_users"
	PermissionManageConfig       Permission = "manage_config"
	PermissionViewSensitive      Permission = "view_sensitive"
)

// AuthManager manages authentication and authorization
type AuthManager struct {
	secretKey         []byte
	userDB            *UserDB
	passwordValidator *PasswordValidator
	rolePermissions   map[UserRole][]Permission
	mu                sync.RWMutex
	users             map[string]*User // In-memory user cache
}

// Claims represents JWT claims
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Role     UserRole `json:"role"`
	jwt.RegisteredClaims
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(secretKey []byte, dbPath string) (*AuthManager, error) {
	// Initialize database
	userDB, err := NewUserDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize user database: %w", err)
	}

	// Get password policy
	policy, err := userDB.GetPasswordPolicy()
	if err != nil {
		return nil, fmt.Errorf("failed to get password policy: %w", err)
	}

	am := &AuthManager{
		secretKey:         secretKey,
		userDB:            userDB,
		passwordValidator: NewPasswordValidator(policy),
		rolePermissions:   make(map[UserRole][]Permission),
		users:             make(map[string]*User),
	}

	// Initialize role permissions
	am.initializeRolePermissions()

	// Create default users if they don't exist
	if err := am.createDefaultUsers(); err != nil {
		return nil, fmt.Errorf("failed to create default users: %w", err)
	}

	return am, nil
}

// initializeRolePermissions sets up role-based permissions
func (am *AuthManager) initializeRolePermissions() {
	// Root user has all permissions
	am.rolePermissions[RoleRoot] = []Permission{
		PermissionViewDashboard, PermissionViewResources, PermissionViewDrift,
		PermissionViewCosts, PermissionViewSecurity, PermissionViewCompliance,
		PermissionExecuteDiscovery, PermissionExecuteAnalysis, PermissionExecuteRemediation,
		PermissionManageUsers, PermissionManageConfig, PermissionViewSensitive,
	}

	// Read-only user has limited permissions
	am.rolePermissions[RoleReadOnly] = []Permission{
		PermissionViewDashboard, PermissionViewResources, PermissionViewDrift,
		PermissionViewCosts, PermissionViewSecurity, PermissionViewCompliance,
	}
}

// createDefaultUsers creates the default root and readonly users
func (am *AuthManager) createDefaultUsers() error {
	// Check if root user exists
	if _, err := am.userDB.GetUserByUsername("root"); err != nil {
		// Create root user
		rootPassword, err := HashPassword("admin")
		if err != nil {
			return fmt.Errorf("failed to hash root password: %w", err)
		}

		rootUser := &User{
			ID:       "root",
			Username: "root",
			Password: rootPassword,
			Role:     RoleRoot,
			Created:  time.Now(),
			Email:    "admin@driftmgr.local",
		}

		if err := am.userDB.CreateUser(rootUser); err != nil {
			return fmt.Errorf("failed to create root user: %w", err)
		}
	}

	// Check if readonly user exists
	if _, err := am.userDB.GetUserByUsername("readonly"); err != nil {
		// Create readonly user
		readonlyPassword, err := HashPassword("readonly")
		if err != nil {
			return fmt.Errorf("failed to hash readonly password: %w", err)
		}

		readonlyUser := &User{
			ID:       "readonly",
			Username: "readonly",
			Password: readonlyPassword,
			Role:     RoleReadOnly,
			Created:  time.Now(),
			Email:    "readonly@driftmgr.local",
		}

		if err := am.userDB.CreateUser(readonlyUser); err != nil {
			return fmt.Errorf("failed to create readonly user: %w", err)
		}
	}

	return nil
}

// AuthenticateUser authenticates a user with username and password
func (am *AuthManager) AuthenticateUser(username, password string) (*User, error) {
	user, err := am.userDB.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if account is locked
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		return nil, fmt.Errorf("account is locked until %s", user.LockedUntil.Format("2006-01-02 15:04:05"))
	}

	// Verify password
	if err := ComparePassword(user.Password, password); err != nil {
		// Increment failed login attempts
		user.FailedLoginAttempts++

		// Check if account should be locked
		policy, _ := am.userDB.GetPasswordPolicy()
		if policy != nil && user.FailedLoginAttempts >= policy.LockoutThreshold {
			lockoutDuration := time.Duration(policy.LockoutDurationMinutes) * time.Minute
			lockedUntil := time.Now().Add(lockoutDuration)
			user.LockedUntil = &lockedUntil
		}

		am.userDB.UpdateUser(user)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Reset failed login attempts on successful login
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	user.LastLogin = time.Now()

	// Update user in database
	am.userDB.UpdateUser(user)

	return user, nil
}

// GenerateJWTToken generates a JWT token for a user
func (am *AuthManager) GenerateJWTToken(user *User) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(am.secretKey)
}

// ValidateJWTToken validates a JWT token and returns user info
func (am *AuthManager) ValidateJWTToken(tokenString string) (*User, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return am.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		user, err := am.userDB.GetUserByUsername(claims.Username)
		if err != nil {
			return nil, fmt.Errorf("user not found")
		}
		return user, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// HasPermission checks if a user has a specific permission
func (am *AuthManager) HasPermission(user *User, permission Permission) bool {
	permissions, exists := am.rolePermissions[user.Role]
	if !exists {
		return false
	}

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// GetUserPermissions returns all permissions for a user
func (am *AuthManager) GetUserPermissions(user *User) []Permission {
	return am.rolePermissions[user.Role]
}

// SanitizeUserData removes sensitive information based on user permissions
func (am *AuthManager) SanitizeUserData(user *User, data map[string]interface{}) map[string]interface{} {
	if am.HasPermission(user, PermissionViewSensitive) {
		return data // Return full data for root users
	}

	// For readonly users, remove sensitive fields
	sanitized := make(map[string]interface{})
	for key, value := range data {
		if !isSensitiveField(key) {
			sanitized[key] = value
		} else {
			sanitized[key] = "[REDACTED]"
		}
	}

	return sanitized
}

// isSensitiveField checks if a field contains sensitive information
func isSensitiveField(fieldName string) bool {
	sensitiveFields := []string{
		"password", "secret", "key", "token", "credential",
		"private_key", "access_key", "secret_key", "api_key",
		"connection_string", "endpoint", "url", "uri",
	}

	fieldLower := strings.ToLower(fieldName)
	for _, sensitive := range sensitiveFields {
		if strings.Contains(fieldLower, sensitive) {
			return true
		}
	}
	return false
}

// ExtractTokenFromRequest extracts JWT token from HTTP request
func ExtractTokenFromRequest(r *http.Request) string {
	// Check Authorization header
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	// Check cookie
	if cookie, err := r.Cookie("auth_token"); err == nil {
		return cookie.Value
	}

	// Check query parameter
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}

	return ""
}

// ListUsers returns a list of all users
func (a *AuthManager) ListUsers() ([]User, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	users := make([]User, 0, len(a.users))
	for _, user := range a.users {
		// Don't include password hashes in the response
		sanitizedUser := User{
			Username:     user.Username,
			Role:         user.Role,
			Permissions:  user.Permissions,
			LastLogin:    user.LastLogin,
			FailedLogins: user.FailedLogins,
			LockedUntil:  user.LockedUntil,
			APIKey:       user.APIKey,
		}
		users = append(users, sanitizedUser)
	}

	return users, nil
}

// GetAuditLogs returns audit logs with optional limit
func (a *AuthManager) GetAuditLogs(limitStr string) ([]map[string]interface{}, error) {
	limit := 100 // default limit
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// In production, this would query a database
	// For now, return a placeholder response
	logs := []map[string]interface{}{
		{
			"timestamp": time.Now().Format(time.RFC3339),
			"event":     "system_initialized",
			"user":      "system",
			"details":   "Audit logging system initialized",
		},
	}

	if limit < len(logs) {
		logs = logs[:limit]
	}

	return logs, nil
}

// Authenticate validates user credentials (alias for compatibility)
func (am *AuthManager) Authenticate(username, password string) (*User, error) {
	return am.AuthenticateUser(username, password)
}

// GenerateToken generates a JWT token (alias for compatibility)
func (am *AuthManager) GenerateToken(user *User) (string, error) {
	return am.GenerateJWTToken(user)
}

// ValidateToken validates a JWT token (alias for compatibility)
func (am *AuthManager) ValidateToken(token string) (*User, error) {
	return am.ValidateJWTToken(token)
}

// ValidateAPIKey validates an API key and returns the associated user
func (am *AuthManager) ValidateAPIKey(apiKey string) (*User, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	// Check cache first
	for _, user := range am.users {
		if user.APIKey == apiKey {
			return user, nil
		}
	}

	// Check database
	users, err := am.userDB.GetAllUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	for _, user := range users {
		if user.APIKey == apiKey {
			// Cache the user
			am.users[user.ID] = user
			return user, nil
		}
	}

	return nil, fmt.Errorf("invalid API key")
}

// GenerateAPIKey generates a new API key for a user
func (am *AuthManager) GenerateAPIKey(userID string) (string, error) {
	// Generate a secure random API key
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}
	apiKey := fmt.Sprintf("dk_%s", hex.EncodeToString(b))

	// Get user from database
	user, err := am.userDB.GetUserByID(userID)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	// Update user with new API key
	user.APIKey = apiKey
	if err := am.userDB.UpdateUser(user); err != nil {
		return "", fmt.Errorf("failed to update user: %w", err)
	}

	// Update cache
	am.mu.Lock()
	am.users[userID] = user
	am.mu.Unlock()

	return apiKey, nil
}

// RevokeAPIKey revokes a user's API key
func (am *AuthManager) RevokeAPIKey(userID string) error {
	// Get user from database
	user, err := am.userDB.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Clear API key
	user.APIKey = ""
	if err := am.userDB.UpdateUser(user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Update cache
	am.mu.Lock()
	if cachedUser, exists := am.users[userID]; exists {
		cachedUser.APIKey = ""
	}
	am.mu.Unlock()

	return nil
}
