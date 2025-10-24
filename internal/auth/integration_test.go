package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuthServiceIntegration(t *testing.T) {
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

func TestUserRegistrationIntegration(t *testing.T) {
	// Create auth service
	authService := createTestAuthService()

	// Test user registration
	registerReq := &RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "Password123!",
		FirstName: "Test",
		LastName:  "User",
	}

	registeredUser, err := authService.Register(registerReq)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	if registeredUser.ID == "" {
		t.Error("User ID should not be empty")
	}

	if registeredUser.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", registeredUser.Username)
	}
}

func TestUserLoginIntegration(t *testing.T) {
	// Create auth service
	authService := createTestAuthService()

	// Register a user first
	registerReq := &RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "Password123!",
		FirstName: "Test",
		LastName:  "User",
	}

	_, err := authService.Register(registerReq)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	// Test login
	loginReq := &LoginRequest{
		Username: "testuser",
		Password: "Password123!",
	}

	loginResp, err := authService.Login(loginReq, "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	if loginResp.AccessToken == "" {
		t.Error("Access token should not be empty")
	}

	if loginResp.RefreshToken == "" {
		t.Error("Refresh token should not be empty")
	}
}

func TestTokenValidationIntegration(t *testing.T) {
	// Create auth service
	authService := createTestAuthService()

	// Register and login user
	registerReq := &RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "Password123!",
		FirstName: "Test",
		LastName:  "User",
	}

	_, err := authService.Register(registerReq)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	loginReq := &LoginRequest{
		Username: "testuser",
		Password: "Password123!",
	}

	loginResp, err := authService.Login(loginReq, "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	// Test token validation
	claims, err := authService.JWTService().ValidateToken(loginResp.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.UserID == "" {
		t.Error("User ID should not be empty in claims")
	}

	if claims.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", claims.Username)
	}
}

func TestTokenRefreshIntegration(t *testing.T) {
	// Create auth service
	authService := createTestAuthService()

	// Register and login user
	registerReq := &RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "Password123!",
		FirstName: "Test",
		LastName:  "User",
	}

	_, err := authService.Register(registerReq)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	loginReq := &LoginRequest{
		Username: "testuser",
		Password: "Password123!",
	}

	loginResp, err := authService.Login(loginReq, "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	// Test token refresh
	refreshResp, err := authService.RefreshToken(loginResp.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}

	if refreshResp.AccessToken == "" {
		t.Error("New access token should not be empty")
	}

	if refreshResp.AccessToken == loginResp.AccessToken {
		t.Error("New access token should be different from old one")
	}
}

func TestUserProfileIntegration(t *testing.T) {
	// Create auth service
	authService := createTestAuthService()

	// Register user
	registerReq := &RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "Password123!",
		FirstName: "Test",
		LastName:  "User",
	}

	registeredUser, err := authService.Register(registerReq)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	// Test get profile
	profile, err := authService.GetUserProfile(registeredUser.ID)
	if err != nil {
		t.Fatalf("Failed to get user profile: %v", err)
	}

	if profile.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", profile.Username)
	}

	if profile.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", profile.Email)
	}
}

func TestAuthHandlersIntegration(t *testing.T) {
	// Create auth service
	authService := createTestAuthService()

	// Create auth handlers
	handlers := NewAuthHandlers(authService)

	// Test register handler
	registerReq := RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "Password123!",
		FirstName: "Test",
		LastName:  "User",
	}

	reqBody, _ := json.Marshal(registerReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	// Test login handler
	loginReq := LoginRequest{
		Username: "testuser",
		Password: "Password123!",
	}

	reqBody, _ = json.Marshal(loginReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	handlers.Login(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestPasswordServiceIntegration(t *testing.T) {
	// Create password service
	passwordService := NewPasswordService()

	// Test password hashing
	password := "TestPassword123!"
	hashedPassword, err := passwordService.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hashedPassword == password {
		t.Error("Hashed password should not be the same as original")
	}

	// Test password verification
	isValid, err := passwordService.VerifyPassword(password, hashedPassword)
	if err != nil {
		t.Fatalf("Failed to verify password: %v", err)
	}

	if !isValid {
		t.Error("Password verification should succeed")
	}

	// Test wrong password
	isValid, err = passwordService.VerifyPassword("wrongpassword", hashedPassword)
	if err != nil {
		t.Fatalf("Failed to verify wrong password: %v", err)
	}

	if isValid {
		t.Error("Wrong password verification should fail")
	}
}

func TestJWTServiceIntegration(t *testing.T) {
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

// Helper function to create test auth service
func createTestAuthService() *Service {
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
	return NewService(
		userRepo,
		roleRepo,
		sessionRepo,
		apiKeyRepo,
		jwtService,
		passwordService,
	)
}
