package auth

import (
	"testing"
	"time"
)

func TestBasicAuthService(t *testing.T) {
	// Create repositories
	userRepo := NewMemoryUserRepository()
	roleRepo := NewMemoryRoleRepository()
	sessionRepo := NewMemorySessionRepository()
	apiKeyRepo := NewMemoryAPIKeyRepository()

	// Create JWT service
	jwtService := NewJWTService(
		"test-secret",
		"test-issuer",
		"test-audience",
		15*time.Minute,
		7*24*time.Hour,
	)

	// Create password service
	passwordService := NewPasswordService()

	// Create auth service
	authService := NewService(
		userRepo,
		roleRepo,
		sessionRepo,
		apiKeyRepo,
		jwtService,
		passwordService,
	)

	// Test service creation
	if authService == nil {
		t.Fatal("Auth service should not be nil")
	}
}

func TestJWTServiceBasic(t *testing.T) {
	// Create JWT service
	jwtService := NewJWTService(
		"test-secret",
		"test-issuer",
		"test-audience",
		15*time.Minute,
		7*24*time.Hour,
	)

	// Create a test user
	user := &User{
		ID:       "test-user-id",
		Username: "testuser",
		Email:    "test@example.com",
		IsAdmin:  false,
	}

	// Test token generation
	accessToken, err := jwtService.GenerateAccessToken(user, []string{"user"})
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	if accessToken == "" {
		t.Error("Generated access token should not be empty")
	}

	// Test token validation
	validatedClaims, err := jwtService.ValidateToken(accessToken)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if validatedClaims.UserID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, validatedClaims.UserID)
	}

	if validatedClaims.Username != user.Username {
		t.Errorf("Expected username %s, got %s", user.Username, validatedClaims.Username)
	}
}

func TestMemoryRepositories(t *testing.T) {
	// Test user repository
	userRepo := NewMemoryUserRepository()
	if userRepo == nil {
		t.Fatal("User repository should not be nil")
	}

	// Test role repository
	roleRepo := NewMemoryRoleRepository()
	if roleRepo == nil {
		t.Fatal("Role repository should not be nil")
	}

	// Test session repository
	sessionRepo := NewMemorySessionRepository()
	if sessionRepo == nil {
		t.Fatal("Session repository should not be nil")
	}

	// Test API key repository
	apiKeyRepo := NewMemoryAPIKeyRepository()
	if apiKeyRepo == nil {
		t.Fatal("API key repository should not be nil")
	}
}
