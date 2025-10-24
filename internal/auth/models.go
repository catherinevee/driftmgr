package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// User represents a system user
type User struct {
	ID           string     `json:"id" db:"id" validate:"required,uuid"`
	Username     string     `json:"username" db:"username" validate:"required,min=3,max=50"`
	Email        string     `json:"email" db:"email" validate:"required,email"`
	PasswordHash string     `json:"-" db:"password_hash" validate:"required"`
	FirstName    string     `json:"first_name" db:"first_name" validate:"required,min=1,max=50"`
	LastName     string     `json:"last_name" db:"last_name" validate:"required,min=1,max=50"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	IsAdmin      bool       `json:"is_admin" db:"is_admin"`
	LastLogin    *time.Time `json:"last_login" db:"last_login"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// Role represents a user role
type Role struct {
	ID          string    `json:"id" db:"id" validate:"required,uuid"`
	Name        string    `json:"name" db:"name" validate:"required,min=1,max=50"`
	Description string    `json:"description" db:"description"`
	Permissions []string  `json:"permissions" db:"permissions"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// UserRole represents the relationship between users and roles
type UserRole struct {
	ID        string    `json:"id" db:"id" validate:"required,uuid"`
	UserID    string    `json:"user_id" db:"user_id" validate:"required,uuid"`
	RoleID    string    `json:"role_id" db:"role_id" validate:"required,uuid"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Permission represents a system permission
type Permission struct {
	ID          string    `json:"id" db:"id" validate:"required,uuid"`
	Name        string    `json:"name" db:"name" validate:"required,min=1,max=100"`
	Description string    `json:"description" db:"description"`
	Resource    string    `json:"resource" db:"resource" validate:"required"`
	Action      string    `json:"action" db:"action" validate:"required"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Session represents a user session
type Session struct {
	ID           string    `json:"id" db:"id" validate:"required,uuid"`
	UserID       string    `json:"user_id" db:"user_id" validate:"required,uuid"`
	Token        string    `json:"token" db:"token" validate:"required"`
	RefreshToken string    `json:"refresh_token" db:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	LastUsedAt   time.Time `json:"last_used_at" db:"last_used_at"`
	UserAgent    string    `json:"user_agent" db:"user_agent"`
	IPAddress    string    `json:"ip_address" db:"ip_address"`
	IsActive     bool      `json:"is_active" db:"is_active"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6"`
	Remember bool   `json:"remember"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Username  string `json:"username" validate:"required,min=3,max=50"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required,min=1,max=50"`
	LastName  string `json:"last_name" validate:"required,min=1,max=50"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

// ResetPasswordRequest represents a password reset request
type ResetPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ConfirmResetPasswordRequest represents a password reset confirmation
type ConfirmResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// UserResponse represents a user response (without sensitive data)
type UserResponse struct {
	ID        string     `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	FirstName string     `json:"first_name"`
	LastName  string     `json:"last_name"`
	IsActive  bool       `json:"is_active"`
	IsAdmin   bool       `json:"is_admin"`
	Roles     []Role     `json:"roles"`
	LastLogin *time.Time `json:"last_login"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	IsAdmin  bool     `json:"is_admin"`
	Exp      int64    `json:"exp"`
	Iat      int64    `json:"iat"`
	Iss      string   `json:"iss"`
	Aud      string   `json:"aud"`
}

// GetAudience returns the audience claim
func (c JWTClaims) GetAudience() (jwt.ClaimStrings, error) {
	return jwt.ClaimStrings{c.Aud}, nil
}

// GetExpirationTime returns the expiration time claim
func (c JWTClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(time.Unix(c.Exp, 0)), nil
}

// GetIssuedAt returns the issued at claim
func (c JWTClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(time.Unix(c.Iat, 0)), nil
}

// GetNotBefore returns the not before claim
func (c JWTClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(time.Unix(c.Iat, 0)), nil
}

// GetIssuer returns the issuer claim
func (c JWTClaims) GetIssuer() (string, error) {
	return c.Iss, nil
}

// GetSubject returns the subject claim
func (c JWTClaims) GetSubject() (string, error) {
	return c.UserID, nil
}

// OAuth2Provider represents an OAuth2 provider configuration
type OAuth2Provider struct {
	ID           string    `json:"id" db:"id" validate:"required,uuid"`
	Name         string    `json:"name" db:"name" validate:"required"`
	ClientID     string    `json:"client_id" db:"client_id" validate:"required"`
	ClientSecret string    `json:"client_secret" db:"client_secret" validate:"required"`
	AuthURL      string    `json:"auth_url" db:"auth_url" validate:"required,url"`
	TokenURL     string    `json:"token_url" db:"token_url" validate:"required,url"`
	UserInfoURL  string    `json:"user_info_url" db:"user_info_url" validate:"required,url"`
	RedirectURL  string    `json:"redirect_url" db:"redirect_url" validate:"required,url"`
	Scopes       string    `json:"scopes" db:"scopes"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// OAuth2CallbackRequest represents an OAuth2 callback request
type OAuth2CallbackRequest struct {
	Code  string `json:"code" validate:"required"`
	State string `json:"state" validate:"required"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details"`
}

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID          string     `json:"id" db:"id" validate:"required,uuid"`
	UserID      string     `json:"user_id" db:"user_id" validate:"required,uuid"`
	Name        string     `json:"name" db:"name" validate:"required,min=1,max=100"`
	KeyHash     string     `json:"-" db:"key_hash" validate:"required"`
	KeyPrefix   string     `json:"key_prefix" db:"key_prefix" validate:"required"`
	Permissions []string   `json:"permissions" db:"permissions"`
	LastUsedAt  *time.Time `json:"last_used_at" db:"last_used_at"`
	ExpiresAt   *time.Time `json:"expires_at" db:"expires_at"`
	IsActive    bool       `json:"is_active" db:"is_active"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateAPIKeyRequest represents a request to create an API key
type CreateAPIKeyRequest struct {
	Name        string     `json:"name" validate:"required,min=1,max=100"`
	Permissions []string   `json:"permissions" validate:"required"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

// APIKeyResponse represents an API key response
type APIKeyResponse struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	KeyPrefix   string     `json:"key_prefix"`
	Permissions []string   `json:"permissions"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Helper methods

// GetFullName returns the user's full name
func (u *User) GetFullName() string {
	return u.FirstName + " " + u.LastName
}

// HasPermission checks if the user has a specific permission
func (u *User) HasPermission(permission string) bool {
	// This would be implemented based on the user's roles
	// For now, return true for admin users
	return u.IsAdmin
}

// IsExpired checks if the session is expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsValid checks if the session is valid and active
func (s *Session) IsValid() bool {
	return s.IsActive && !s.IsExpired()
}

// GetDisplayName returns the role's display name
func (r *Role) GetDisplayName() string {
	if r.Description != "" {
		return r.Description
	}
	return r.Name
}

// HasPermission checks if the role has a specific permission
func (r *Role) HasPermission(permission string) bool {
	for _, p := range r.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// IsExpired checks if the API key is expired
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsValid checks if the API key is valid and active
func (k *APIKey) IsValid() bool {
	return k.IsActive && !k.IsExpired()
}

// Constants for permissions
const (
	// User permissions
	PermissionUserRead   = "user:read"
	PermissionUserWrite  = "user:write"
	PermissionUserDelete = "user:delete"
	PermissionUserAdmin  = "user:admin"

	// Backend permissions
	PermissionBackendRead   = "backend:read"
	PermissionBackendWrite  = "backend:write"
	PermissionBackendDelete = "backend:delete"
	PermissionBackendAdmin  = "backend:admin"

	// State permissions
	PermissionStateRead   = "state:read"
	PermissionStateWrite  = "state:write"
	PermissionStateDelete = "state:delete"
	PermissionStateAdmin  = "state:admin"

	// Resource permissions
	PermissionResourceRead   = "resource:read"
	PermissionResourceWrite  = "resource:write"
	PermissionResourceDelete = "resource:delete"
	PermissionResourceAdmin  = "resource:admin"

	// Drift permissions
	PermissionDriftRead   = "drift:read"
	PermissionDriftWrite  = "drift:write"
	PermissionDriftDelete = "drift:delete"
	PermissionDriftAdmin  = "drift:admin"

	// Remediation permissions
	PermissionRemediationRead   = "remediation:read"
	PermissionRemediationWrite  = "remediation:write"
	PermissionRemediationDelete = "remediation:delete"
	PermissionRemediationAdmin  = "remediation:admin"

	// System permissions
	PermissionSystemRead   = "system:read"
	PermissionSystemWrite  = "system:write"
	PermissionSystemDelete = "system:delete"
	PermissionSystemAdmin  = "system:admin"
)

// Constants for roles
const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
	RoleAuditor  = "auditor"
)

// Default roles and their permissions
var DefaultRoles = map[string][]string{
	RoleAdmin: {
		PermissionUserRead, PermissionUserWrite, PermissionUserDelete, PermissionUserAdmin,
		PermissionBackendRead, PermissionBackendWrite, PermissionBackendDelete, PermissionBackendAdmin,
		PermissionStateRead, PermissionStateWrite, PermissionStateDelete, PermissionStateAdmin,
		PermissionResourceRead, PermissionResourceWrite, PermissionResourceDelete, PermissionResourceAdmin,
		PermissionDriftRead, PermissionDriftWrite, PermissionDriftDelete, PermissionDriftAdmin,
		PermissionRemediationRead, PermissionRemediationWrite, PermissionRemediationDelete, PermissionRemediationAdmin,
		PermissionSystemRead, PermissionSystemWrite, PermissionSystemDelete, PermissionSystemAdmin,
	},
	RoleOperator: {
		PermissionBackendRead, PermissionBackendWrite,
		PermissionStateRead, PermissionStateWrite,
		PermissionResourceRead, PermissionResourceWrite,
		PermissionDriftRead, PermissionDriftWrite,
		PermissionRemediationRead, PermissionRemediationWrite,
	},
	RoleViewer: {
		PermissionBackendRead,
		PermissionStateRead,
		PermissionResourceRead,
		PermissionDriftRead,
		PermissionRemediationRead,
	},
	RoleAuditor: {
		PermissionBackendRead,
		PermissionStateRead,
		PermissionResourceRead,
		PermissionDriftRead,
		PermissionRemediationRead,
		PermissionUserRead,
		PermissionSystemRead,
	},
}
