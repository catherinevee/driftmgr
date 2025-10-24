package auth

import (
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_Register(t *testing.T) {
	service := createMockAuthService(t)

	tests := []struct {
		name        string
		request     auth.RegisterRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid registration",
			request: auth.RegisterRequest{
				Username:  "testuser",
				Email:     "test@example.com",
				Password:  "password123",
				FirstName: "Test",
				LastName:  "User",
			},
			expectError: false,
		},
		{
			name: "duplicate username",
			request: auth.RegisterRequest{
				Username:  "testuser", // Same as above
				Email:     "test2@example.com",
				Password:  "password123",
				FirstName: "Test",
				LastName:  "User 2",
			},
			expectError: true,
			errorMsg:    "username already exists",
		},
		{
			name: "duplicate email",
			request: auth.RegisterRequest{
				Username:  "testuser2",
				Email:     "test@example.com", // Same as above
				Password:  "password123",
				FirstName: "Test",
				LastName:  "User 2",
			},
			expectError: true,
			errorMsg:    "email already exists",
		},
		{
			name: "invalid email format",
			request: auth.RegisterRequest{
				Username:  "testuser3",
				Email:     "invalid-email",
				Password:  "password123",
				FirstName: "Test",
				LastName:  "User 3",
			},
			expectError: true,
			errorMsg:    "invalid email format",
		},
		{
			name: "weak password",
			request: auth.RegisterRequest{
				Username:  "testuser4",
				Email:     "test4@example.com",
				Password:  "123", // Too short
				FirstName: "Test",
				LastName:  "User 4",
			},
			expectError: true,
			errorMsg:    "password too weak",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := service.Register(&tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotEmpty(t, response.AccessToken)
				assert.NotEmpty(t, response.RefreshToken)
				assert.Equal(t, tt.request.Username, response.User.Username)
				assert.Equal(t, tt.request.Email, response.User.Email)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	service := createMockAuthService(t)

	// Register a user first
	registerRequest := auth.RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	_, err := service.Register(&registerRequest)
	require.NoError(t, err)

	tests := []struct {
		name        string
		request     auth.LoginRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid login with username",
			request: auth.LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			expectError: false,
		},
		{
			name: "valid login with email",
			request: auth.LoginRequest{
				Username: "test@example.com",
				Password: "password123",
			},
			expectError: false,
		},
		{
			name: "invalid username",
			request: auth.LoginRequest{
				Username: "nonexistent",
				Password: "password123",
			},
			expectError: true,
			errorMsg:    "invalid credentials",
		},
		{
			name: "invalid password",
			request: auth.LoginRequest{
				Username: "testuser",
				Password: "wrongpassword",
			},
			expectError: true,
			errorMsg:    "invalid credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := service.Login(&tt.request, "test-agent", "127.0.0.1")

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotEmpty(t, response.AccessToken)
				assert.NotEmpty(t, response.RefreshToken)
				assert.Equal(t, "testuser", response.User.Username)
			}
		})
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	service := createMockAuthService(t)

	// Register and login a user
	registerRequest := auth.RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	loginResponse, err := service.Register(&registerRequest)
	require.NoError(t, err)

	tests := []struct {
		name         string
		refreshToken string
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "valid refresh token",
			refreshToken: loginResponse.RefreshToken,
			expectError:  false,
		},
		{
			name:         "invalid refresh token",
			refreshToken: "invalid-token",
			expectError:  true,
			errorMsg:     "invalid refresh token",
		},
		{
			name:         "empty refresh token",
			refreshToken: "",
			expectError:  true,
			errorMsg:     "refresh token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := service.RefreshToken(tt.refreshToken)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotEmpty(t, response.AccessToken)
				assert.NotEmpty(t, response.RefreshToken)
				assert.Equal(t, "testuser", response.User.Username)
			}
		})
	}
}

func TestAuthService_CreateAPIKey(t *testing.T) {
	service := createMockAuthService(t)

	// Register a user first
	registerRequest := auth.RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	_, err := service.Register(&registerRequest)
	require.NoError(t, err)

	// Get user ID (in a real implementation, this would be from the login response)
	userID := "testuser" // Simplified for testing

	tests := []struct {
		name        string
		request     auth.CreateAPIKeyRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid API key creation",
			request: auth.CreateAPIKeyRequest{
				Name:        "Test API Key",
				Permissions: []string{"read", "write"},
				ExpiresAt:   time.Now().Add(30 * 24 * time.Hour),
			},
			expectError: false,
		},
		{
			name: "API key with no permissions",
			request: auth.CreateAPIKeyRequest{
				Name:        "No Permissions Key",
				Permissions: []string{},
				ExpiresAt:   time.Now().Add(30 * 24 * time.Hour),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := service.CreateAPIKey(userID, &tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotEmpty(t, response.ID)
				assert.NotEmpty(t, response.KeyPrefix)
				assert.Equal(t, tt.request.Name, response.Name)
				assert.Equal(t, tt.request.Permissions, response.Permissions)
			}
		})
	}
}

// Helper functions

func createMockAuthService(t *testing.T) *auth.Service {
	// Create mock repositories
	userRepo := &mockUserRepository{}
	apiKeyRepo := &mockAPIKeyRepository{}
	roleRepo := &mockRoleRepository{}
	sessionRepo := &mockSessionRepository{}

	// Create password service
	passwordService := auth.NewPasswordService()

	// Create JWT service
	jwtService := auth.NewJWTService("test-secret-key", "test-issuer", "test-audience", 24*time.Hour, 7*24*time.Hour)

	// Create auth service
	service := auth.NewService(userRepo, roleRepo, sessionRepo, apiKeyRepo, jwtService, passwordService)

	return service
}

// Mock repositories for testing

type mockUserRepository struct {
	users map[string]*auth.User
}

func (m *mockUserRepository) Create(user *auth.User) error {
	if m.users == nil {
		m.users = make(map[string]*auth.User)
	}

	// Check for duplicates
	for _, existingUser := range m.users {
		if existingUser.Username == user.Username {
			return &mockError{Code: "USERNAME_EXISTS", Message: "username already exists"}
		}
		if existingUser.Email == user.Email {
			return &mockError{Code: "EMAIL_EXISTS", Message: "email already exists"}
		}
	}

	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) GetByID(id string) (*auth.User, error) {
	if user, exists := m.users[id]; exists {
		return user, nil
	}
	return nil, &mockError{Code: "USER_NOT_FOUND", Message: "user not found"}
}

func (m *mockUserRepository) GetByUsername(username string) (*auth.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, &mockError{Code: "USER_NOT_FOUND", Message: "user not found"}
}

func (m *mockUserRepository) GetByEmail(email string) (*auth.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, &mockError{Code: "USER_NOT_FOUND", Message: "user not found"}
}

func (m *mockUserRepository) Update(user *auth.User) error {
	if _, exists := m.users[user.ID]; !exists {
		return &mockError{Code: "USER_NOT_FOUND", Message: "user not found"}
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) Delete(id string) error {
	if _, exists := m.users[id]; !exists {
		return &mockError{Code: "USER_NOT_FOUND", Message: "user not found"}
	}
	delete(m.users, id)
	return nil
}

func (m *mockUserRepository) List(limit, offset int) ([]*auth.User, error) {
	users := make([]*auth.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, nil
}

type mockAPIKeyRepository struct {
	apiKeys map[string]*auth.APIKey
}

func (m *mockAPIKeyRepository) Create(apiKey *auth.APIKey) error {
	if m.apiKeys == nil {
		m.apiKeys = make(map[string]*auth.APIKey)
	}
	m.apiKeys[apiKey.ID] = apiKey
	return nil
}

func (m *mockAPIKeyRepository) GetByID(id string) (*auth.APIKey, error) {
	if apiKey, exists := m.apiKeys[id]; exists {
		return apiKey, nil
	}
	return nil, &mockError{Code: "API_KEY_NOT_FOUND", Message: "API key not found"}
}

func (m *mockAPIKeyRepository) GetByUserID(userID string) ([]*auth.APIKey, error) {
	var userAPIKeys []*auth.APIKey
	for _, apiKey := range m.apiKeys {
		if apiKey.UserID == userID {
			userAPIKeys = append(userAPIKeys, apiKey)
		}
	}
	return userAPIKeys, nil
}

func (m *mockAPIKeyRepository) Update(apiKey *auth.APIKey) error {
	if _, exists := m.apiKeys[apiKey.ID]; !exists {
		return &mockError{Code: "API_KEY_NOT_FOUND", Message: "API key not found"}
	}
	m.apiKeys[apiKey.ID] = apiKey
	return nil
}

func (m *mockAPIKeyRepository) Delete(id string) error {
	if _, exists := m.apiKeys[id]; !exists {
		return &mockError{Code: "API_KEY_NOT_FOUND", Message: "API key not found"}
	}
	delete(m.apiKeys, id)
	return nil
}

func (m *mockAPIKeyRepository) Validate(key string) (*auth.APIKey, error) {
	// Simplified validation for testing
	if key == "valid-api-key" {
		return &auth.APIKey{
			ID:          "test-key-id",
			UserID:      "testuser",
			Name:        "Test API Key",
			KeyPrefix:   "test_",
			Permissions: []string{"read", "write"},
			IsActive:    true,
			CreatedAt:   time.Now(),
		}, nil
	}
	return nil, &mockError{Code: "INVALID_API_KEY", Message: "invalid API key"}
}

type mockRoleRepository struct{}

func (m *mockRoleRepository) GetByUserID(userID string) ([]*auth.Role, error) {
	// Return default roles for testing
	return []*auth.Role{
		{
			ID:   "viewer",
			Name: "Viewer",
		},
	}, nil
}

func (m *mockRoleRepository) AssignToUser(userID string, roleID string) error {
	return nil
}

func (m *mockRoleRepository) RemoveFromUser(userID string, roleID string) error {
	return nil
}

type mockSessionRepository struct{}

func (m *mockSessionRepository) Create(session *auth.Session) error {
	return nil
}

func (m *mockSessionRepository) GetByToken(token string) (*auth.Session, error) {
	return nil, &mockError{Code: "SESSION_NOT_FOUND", Message: "session not found"}
}

func (m *mockSessionRepository) GetByUserID(userID string) ([]*auth.Session, error) {
	return []*auth.Session{}, nil
}

func (m *mockSessionRepository) Update(session *auth.Session) error {
	return nil
}

func (m *mockSessionRepository) Delete(id string) error {
	return nil
}

func (m *mockSessionRepository) DeleteByUserID(userID string) error {
	return nil
}

func (m *mockSessionRepository) DeleteExpired() error {
	return nil
}

// Mock error type
type mockError struct {
	Code    string
	Message string
}

func (e *mockError) Error() string {
	return e.Message
}
