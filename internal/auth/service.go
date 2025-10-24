package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service handles authentication business logic
type Service struct {
	userRepo        UserRepository
	roleRepo        RoleRepository
	sessionRepo     SessionRepository
	apiKeyRepo      APIKeyRepository
	jwtService      *JWTService
	passwordService *PasswordService
}

// JWTService returns the JWT service for middleware access
func (s *Service) JWTService() *JWTService {
	return s.jwtService
}

// NewService creates a new authentication service
func NewService(
	userRepo UserRepository,
	roleRepo RoleRepository,
	sessionRepo SessionRepository,
	apiKeyRepo APIKeyRepository,
	jwtService *JWTService,
	passwordService *PasswordService,
) *Service {
	return &Service{
		userRepo:        userRepo,
		roleRepo:        roleRepo,
		sessionRepo:     sessionRepo,
		apiKeyRepo:      apiKeyRepo,
		jwtService:      jwtService,
		passwordService: passwordService,
	}
}

// Login authenticates a user and returns tokens
func (s *Service) Login(req *LoginRequest, userAgent, ipAddress string) (*AuthResponse, error) {
	// Find user by username
	user, err := s.userRepo.GetByUsername(req.Username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, errors.New("account is disabled")
	}

	// Verify password
	valid, err := s.passwordService.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("failed to verify password: %w", err)
	}
	if !valid {
		return nil, errors.New("invalid credentials")
	}

	// Get user roles
	roles, err := s.roleRepo.GetByUserID(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	// Generate tokens
	accessToken, err := s.jwtService.GenerateAccessToken(user, s.getRoleNames(roles))
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create session
	session := &Session{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		Token:        accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(s.jwtService.GetTokenExpiry()),
		CreatedAt:    time.Now(),
		LastUsedAt:   time.Now(),
		UserAgent:    userAgent,
		IPAddress:    ipAddress,
		IsActive:     true,
	}

	if err := s.sessionRepo.Create(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Update last login
	now := time.Now()
	user.LastLogin = &now
	if err := s.userRepo.Update(user); err != nil {
		// Log error but don't fail the login
		fmt.Printf("Failed to update last login: %v\n", err)
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.jwtService.GetTokenExpiry().Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// Register creates a new user account
func (s *Service) Register(req *RegisterRequest) (*UserResponse, error) {
	// Validate password strength
	if err := s.passwordService.ValidatePasswordStrength(req.Password); err != nil {
		return nil, fmt.Errorf("password validation failed: %w", err)
	}

	// Check if username already exists
	existingUser, _ := s.userRepo.GetByUsername(req.Username)
	if existingUser != nil {
		return nil, errors.New("username already exists")
	}

	// Check if email already exists
	existingUser, _ = s.userRepo.GetByEmail(req.Email)
	if existingUser != nil {
		return nil, errors.New("email already exists")
	}

	// Hash password
	passwordHash, err := s.passwordService.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &User{
		ID:           uuid.New().String(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		IsActive:     true,
		IsAdmin:      false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Assign default role (viewer)
	viewerRole, err := s.roleRepo.GetByName(RoleViewer)
	if err == nil {
		userRole := &UserRole{
			ID:        uuid.New().String(),
			UserID:    user.ID,
			RoleID:    viewerRole.ID,
			CreatedAt: time.Now(),
		}
		if err := s.roleRepo.AssignRole(userRole); err != nil {
			// Log error but don't fail registration
			fmt.Printf("Failed to assign default role: %v\n", err)
		}
	}

	return s.toUserResponse(user, []Role{}), nil
}

// RefreshToken generates new tokens from a refresh token
func (s *Service) RefreshToken(refreshToken string) (*AuthResponse, error) {
	// Validate refresh token
	userID, err := s.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Get user
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if !user.IsActive {
		return nil, errors.New("account is disabled")
	}

	// Get user roles
	roles, err := s.roleRepo.GetByUserID(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	// Generate new tokens
	accessToken, newRefreshToken, err := s.jwtService.RefreshTokenPair(refreshToken, user, s.getRoleNames(roles))
	if err != nil {
		return nil, fmt.Errorf("failed to refresh tokens: %w", err)
	}

	// Update session
	session, err := s.sessionRepo.GetByRefreshToken(refreshToken)
	if err == nil {
		session.Token = accessToken
		session.RefreshToken = newRefreshToken
		session.ExpiresAt = time.Now().Add(s.jwtService.GetTokenExpiry())
		session.LastUsedAt = time.Now()
		s.sessionRepo.Update(session)
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int(s.jwtService.GetTokenExpiry().Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// Logout invalidates a user session
func (s *Service) Logout(token string) error {
	// Validate token to get user ID
	_, err := s.jwtService.ValidateToken(token)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	// Deactivate session
	session, err := s.sessionRepo.GetByToken(token)
	if err == nil {
		session.IsActive = false
		s.sessionRepo.Update(session)
	}

	return nil
}

// ChangePassword changes a user's password
func (s *Service) ChangePassword(userID string, req *ChangePasswordRequest) error {
	// Get user
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	// Verify current password
	valid, err := s.passwordService.VerifyPassword(req.CurrentPassword, user.PasswordHash)
	if err != nil {
		return fmt.Errorf("failed to verify current password: %w", err)
	}
	if !valid {
		return errors.New("current password is incorrect")
	}

	// Validate new password strength
	if err := s.passwordService.ValidatePasswordStrength(req.NewPassword); err != nil {
		return fmt.Errorf("password validation failed: %w", err)
	}

	// Hash new password
	newPasswordHash, err := s.passwordService.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update user
	user.PasswordHash = newPasswordHash
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// CreateAPIKey creates a new API key for a user
func (s *Service) CreateAPIKey(userID string, req *CreateAPIKeyRequest) (*APIKeyResponse, error) {
	// Get user
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Generate API key
	key, keyHash, err := s.passwordService.GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Create API key record
	apiKey := &APIKey{
		ID:          uuid.New().String(),
		UserID:      user.ID,
		Name:        req.Name,
		KeyHash:     keyHash,
		KeyPrefix:   key[:8] + "...",
		Permissions: req.Permissions,
		ExpiresAt:   req.ExpiresAt,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.apiKeyRepo.Create(apiKey); err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	// Return response (without the actual key for security)
	return &APIKeyResponse{
		ID:          apiKey.ID,
		UserID:      apiKey.UserID,
		Name:        apiKey.Name,
		KeyPrefix:   apiKey.KeyPrefix,
		Permissions: apiKey.Permissions,
		ExpiresAt:   apiKey.ExpiresAt,
		IsActive:    apiKey.IsActive,
		CreatedAt:   apiKey.CreatedAt,
		UpdatedAt:   apiKey.UpdatedAt,
	}, nil
}

// ValidateAPIKey validates an API key and returns user information
func (s *Service) ValidateAPIKey(key string) (*User, *APIKey, error) {
	// Find API key by prefix
	apiKeys, err := s.apiKeyRepo.GetByUserID("") // Get all active API keys
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	var validAPIKey *APIKey
	for _, ak := range apiKeys {
		if ak.IsValid() {
			valid, err := s.passwordService.VerifyAPIKey(key, ak.KeyHash)
			if err == nil && valid {
				validAPIKey = ak
				break
			}
		}
	}

	if validAPIKey == nil {
		return nil, nil, errors.New("invalid API key")
	}

	// Get user
	user, err := s.userRepo.GetByID(validAPIKey.UserID)
	if err != nil {
		return nil, nil, errors.New("user not found")
	}

	if !user.IsActive {
		return nil, nil, errors.New("user account is disabled")
	}

	// Update last used timestamp
	now := time.Now()
	validAPIKey.LastUsedAt = &now
	s.apiKeyRepo.Update(validAPIKey)

	return user, validAPIKey, nil
}

// GetUserProfile returns a user's profile information
func (s *Service) GetUserProfile(userID string) (*UserResponse, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	roles, err := s.roleRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	return s.toUserResponse(user, roles), nil
}

// UpdateUserProfile updates a user's profile information
func (s *Service) UpdateUserProfile(userID string, updates map[string]interface{}) (*UserResponse, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Update allowed fields
	if firstName, ok := updates["first_name"].(string); ok {
		user.FirstName = firstName
	}
	if lastName, ok := updates["last_name"].(string); ok {
		user.LastName = lastName
	}
	if email, ok := updates["email"].(string); ok {
		// Check if email is already taken by another user
		existingUser, _ := s.userRepo.GetByEmail(email)
		if existingUser != nil && existingUser.ID != userID {
			return nil, errors.New("email already exists")
		}
		user.Email = email
	}

	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	roles, err := s.roleRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	return s.toUserResponse(user, roles), nil
}

// GetUserAPIKeys returns API keys for a user
func (s *Service) GetUserAPIKeys(userID string) ([]APIKeyResponse, error) {
	apiKeys, err := s.apiKeyRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user API keys: %w", err)
	}

	responses := make([]APIKeyResponse, len(apiKeys))
	for i, apiKey := range apiKeys {
		responses[i] = APIKeyResponse{
			ID:          apiKey.ID,
			UserID:      apiKey.UserID,
			Name:        apiKey.Name,
			KeyPrefix:   apiKey.KeyPrefix,
			Permissions: apiKey.Permissions,
			LastUsedAt:  apiKey.LastUsedAt,
			ExpiresAt:   apiKey.ExpiresAt,
			IsActive:    apiKey.IsActive,
			CreatedAt:   apiKey.CreatedAt,
			UpdatedAt:   apiKey.UpdatedAt,
		}
	}

	return responses, nil
}

// DeleteAPIKey deletes an API key for a user
func (s *Service) DeleteAPIKey(userID, apiKeyID string) error {
	// Verify the API key belongs to the user
	apiKey, err := s.apiKeyRepo.GetByID(apiKeyID)
	if err != nil {
		return fmt.Errorf("API key not found: %w", err)
	}

	if apiKey.UserID != userID {
		return errors.New("API key does not belong to user")
	}

	// Delete the API key
	if err := s.apiKeyRepo.Delete(apiKeyID); err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	return nil
}

// Helper methods

// getRoleNames extracts role names from role objects
func (s *Service) getRoleNames(roles []Role) []string {
	names := make([]string, len(roles))
	for i, role := range roles {
		names[i] = role.Name
	}
	return names
}

// toUserResponse converts a User to UserResponse
func (s *Service) toUserResponse(user *User, roles []Role) *UserResponse {
	return &UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		IsActive:  user.IsActive,
		IsAdmin:   user.IsAdmin,
		Roles:     roles,
		LastLogin: user.LastLogin,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
