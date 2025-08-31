package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrTokenExpired       = errors.New("token expired")
	ErrInvalidToken       = errors.New("invalid token")
	ErrInsufficientRole   = errors.New("insufficient role permissions")
)

// Role represents user roles in the system
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleViewer   Role = "viewer"
	RoleApprover Role = "approver"
)

// User represents an authenticated user
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Role         Role      `json:"role"`
	Permissions  []string  `json:"permissions"`
	LastLogin    time.Time `json:"last_login"`
	MFAEnabled   bool      `json:"mfa_enabled"`
	SessionToken string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Claims represents JWT claims
type Claims struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	Role        Role     `json:"role"`
	Permissions []string `json:"permissions"`
	SessionID   string   `json:"session_id"`
	jwt.RegisteredClaims
}

// AuthService handles authentication and authorization
type AuthService struct {
	jwtSecret     []byte
	tokenDuration time.Duration
	users         map[string]*User
	sessions      map[string]*Session
	passwordStore map[string]string // username -> hashed password
	apiKeys       map[string]*APIKey
	mu            sync.RWMutex
}

// Session represents an active user session
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
}

// APIKey represents an API key for service authentication
type APIKey struct {
	ID          string    `json:"id"`
	Key         string    `json:"-"`
	HashedKey   string    `json:"hashed_key"`
	Name        string    `json:"name"`
	Role        Role      `json:"role"`
	Permissions []string  `json:"permissions"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsed    *time.Time `json:"last_used,omitempty"`
}

// NewAuthService creates a new authentication service
func NewAuthService(jwtSecret string) *AuthService {
	if jwtSecret == "" {
		// Generate a random secret if none provided
		secret := make([]byte, 32)
		rand.Read(secret)
		jwtSecret = base64.StdEncoding.EncodeToString(secret)
	}

	service := &AuthService{
		jwtSecret:     []byte(jwtSecret),
		tokenDuration: 24 * time.Hour,
		users:         make(map[string]*User),
		sessions:      make(map[string]*Session),
		passwordStore: make(map[string]string),
		apiKeys:       make(map[string]*APIKey),
	}

	// Initialize with default admin user (should be changed on first login)
	service.createDefaultUsers()

	return service
}

// createDefaultUsers creates default users for initial setup
func (s *AuthService) createDefaultUsers() {
	// Default admin user - password should be changed immediately
	adminPassword, _ := bcrypt.GenerateFromPassword([]byte("ChangeMeNow!"), bcrypt.DefaultCost)
	s.passwordStore["admin"] = string(adminPassword)
	s.users["admin"] = &User{
		ID:       "user-admin",
		Username: "admin",
		Email:    "admin@driftmgr.local",
		Role:     RoleAdmin,
		Permissions: []string{
			"discovery:*",
			"drift:*",
			"remediation:*",
			"state:*",
			"config:*",
			"users:*",
		},
		CreatedAt: time.Now(),
	}

	// Read-only viewer user
	viewerPassword, _ := bcrypt.GenerateFromPassword([]byte("ViewerPass123!"), bcrypt.DefaultCost)
	s.passwordStore["viewer"] = string(viewerPassword)
	s.users["viewer"] = &User{
		ID:       "user-viewer",
		Username: "viewer",
		Email:    "viewer@driftmgr.local",
		Role:     RoleViewer,
		Permissions: []string{
			"discovery:read",
			"drift:read",
			"state:read",
		},
		CreatedAt: time.Now(),
	}
}

// Authenticate validates credentials and returns a token
func (s *AuthService) Authenticate(ctx context.Context, username, password string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate credentials
	hashedPassword, exists := s.passwordStore[username]
	if !exists {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	user, exists := s.users[username]
	if !exists {
		return nil, ErrInvalidCredentials
	}

	// Update last login
	user.LastLogin = time.Now()

	// Create session
	session, err := s.createSession(user)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// createSession creates a new session for a user
func (s *AuthService) createSession(user *User) (*Session, error) {
	sessionID := generateSessionID()
	
	// Create JWT token
	claims := &Claims{
		UserID:      user.ID,
		Username:    user.Username,
		Role:        user.Role,
		Permissions: user.Permissions,
		SessionID:   sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "driftmgr",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:        sessionID,
		UserID:    user.ID,
		Token:     tokenString,
		ExpiresAt: time.Now().Add(s.tokenDuration),
		CreatedAt: time.Now(),
	}

	s.sessions[sessionID] = session
	return session, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Check if session still exists
		s.mu.RLock()
		session, exists := s.sessions[claims.SessionID]
		s.mu.RUnlock()

		if !exists || time.Now().After(session.ExpiresAt) {
			return nil, ErrTokenExpired
		}

		return claims, nil
	}

	return nil, ErrInvalidToken
}

// Authorize checks if a user has permission for an action
func (s *AuthService) Authorize(claims *Claims, resource string, action string) error {
	permission := fmt.Sprintf("%s:%s", resource, action)
	
	// Admin has all permissions
	if claims.Role == RoleAdmin {
		return nil
	}

	// Check specific permissions
	for _, p := range claims.Permissions {
		if p == permission || p == fmt.Sprintf("%s:*", resource) || p == "*" {
			return nil
		}
	}

	// Check role-based permissions
	if s.checkRolePermission(claims.Role, resource, action) {
		return nil
	}

	return ErrInsufficientRole
}

// checkRolePermission checks if a role has implicit permission
func (s *AuthService) checkRolePermission(role Role, resource string, action string) bool {
	switch role {
	case RoleOperator:
		// Operators can read and execute, but not delete or modify config
		if action == "read" || action == "execute" || action == "discover" {
			return true
		}
	case RoleViewer:
		// Viewers can only read
		if action == "read" {
			return true
		}
	case RoleApprover:
		// Approvers can read and approve
		if action == "read" || action == "approve" {
			return true
		}
	}
	return false
}

// CreateAPIKey creates a new API key
func (s *AuthService) CreateAPIKey(name string, role Role, permissions []string, expiresIn *time.Duration) (*APIKey, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate API key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, "", err
	}
	
	key := base64.URLEncoding.EncodeToString(keyBytes)
	hashedKey, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	apiKey := &APIKey{
		ID:          fmt.Sprintf("apikey-%d", time.Now().Unix()),
		HashedKey:   string(hashedKey),
		Name:        name,
		Role:        role,
		Permissions: permissions,
		CreatedAt:   time.Now(),
	}

	if expiresIn != nil {
		expiresAt := time.Now().Add(*expiresIn)
		apiKey.ExpiresAt = &expiresAt
	}

	s.apiKeys[apiKey.ID] = apiKey

	// Return the API key only once (it can't be retrieved later)
	return apiKey, key, nil
}

// ValidateAPIKey validates an API key and returns the associated permissions
func (s *AuthService) ValidateAPIKey(key string) (*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, apiKey := range s.apiKeys {
		if err := bcrypt.CompareHashAndPassword([]byte(apiKey.HashedKey), []byte(key)); err == nil {
			// Check expiration
			if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
				return nil, ErrTokenExpired
			}

			// Update last used
			now := time.Now()
			apiKey.LastUsed = &now

			return apiKey, nil
		}
	}

	return nil, ErrInvalidToken
}

// RevokeSession revokes a user session
func (s *AuthService) RevokeSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(username, oldPassword, newPassword string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate old password
	hashedPassword, exists := s.passwordStore[username]
	if !exists {
		return ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	// Validate new password strength
	if err := validatePasswordStrength(newPassword); err != nil {
		return err
	}

	// Hash and store new password
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	s.passwordStore[username] = string(newHashedPassword)

	// Revoke all existing sessions for this user
	user := s.users[username]
	for id, session := range s.sessions {
		if session.UserID == user.ID {
			delete(s.sessions, id)
		}
	}

	return nil
}

// validatePasswordStrength checks if password meets security requirements
func validatePasswordStrength(password string) error {
	if len(password) < 12 {
		return errors.New("password must be at least 12 characters long")
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return errors.New("password must contain uppercase, lowercase, digit, and special character")
	}

	return nil
}

// generateSessionID generates a random session ID
func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// GetUser returns user information by ID
func (s *AuthService) GetUser(userID string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.ID == userID {
			return user, nil
		}
	}

	return nil, errors.New("user not found")
}

// ListSessions lists active sessions for a user
func (s *AuthService) ListSessions(userID string) []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sessions []*Session
	for _, session := range s.sessions {
		if session.UserID == userID && time.Now().Before(session.ExpiresAt) {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// CleanupExpiredSessions removes expired sessions
func (s *AuthService) CleanupExpiredSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, id)
		}
	}
}

// ExtractBearerToken extracts token from Authorization header
func ExtractBearerToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", ErrUnauthorized
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", ErrInvalidToken
	}

	return parts[1], nil
}

// CompareSecure performs constant-time comparison
func CompareSecure(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}