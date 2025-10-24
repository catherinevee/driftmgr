package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTService handles JWT token operations
type JWTService struct {
	secretKey     []byte
	issuer        string
	audience      string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// NewJWTService creates a new JWT service
func NewJWTService(secretKey, issuer, audience string, accessExpiry, refreshExpiry time.Duration) *JWTService {
	return &JWTService{
		secretKey:     []byte(secretKey),
		issuer:        issuer,
		audience:      audience,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// GenerateAccessToken generates a new access token
func (j *JWTService) GenerateAccessToken(user *User, roles []string) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Roles:    roles,
		IsAdmin:  user.IsAdmin,
		Exp:      now.Add(j.accessExpiry).Unix(),
		Iat:      now.Unix(),
		Iss:      j.issuer,
		Aud:      j.audience,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// GenerateRefreshToken generates a new refresh token
func (j *JWTService) GenerateRefreshToken(user *User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"type":    "refresh",
		"exp":     now.Add(j.refreshExpiry).Unix(),
		"iat":     now.Unix(),
		"iss":     j.issuer,
		"aud":     j.audience,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// ValidateToken validates a JWT token and returns the claims
func (j *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		// Validate issuer and audience
		if claims.Iss != j.issuer {
			return nil, errors.New("invalid issuer")
		}
		if claims.Aud != j.audience {
			return nil, errors.New("invalid audience")
		}
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ValidateRefreshToken validates a refresh token
func (j *JWTService) ValidateRefreshToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Check if it's a refresh token
		if tokenType, ok := claims["type"].(string); !ok || tokenType != "refresh" {
			return "", errors.New("invalid token type")
		}

		// Validate issuer and audience
		if iss, ok := claims["iss"].(string); !ok || iss != j.issuer {
			return "", errors.New("invalid issuer")
		}
		if aud, ok := claims["aud"].(string); !ok || aud != j.audience {
			return "", errors.New("invalid audience")
		}

		// Extract user ID
		userID, ok := claims["user_id"].(string)
		if !ok {
			return "", errors.New("invalid user ID in token")
		}

		return userID, nil
	}

	return "", errors.New("invalid token")
}

// ExtractTokenFromHeader extracts the token from the Authorization header
func (j *JWTService) ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.New("authorization header is required")
	}

	// Check for Bearer token format
	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", errors.New("authorization header must start with 'Bearer '")
	}

	token := authHeader[len(bearerPrefix):]
	if token == "" {
		return "", errors.New("token is required")
	}

	return token, nil
}

// GetTokenExpiry returns the token expiry time
func (j *JWTService) GetTokenExpiry() time.Duration {
	return j.accessExpiry
}

// GetRefreshTokenExpiry returns the refresh token expiry time
func (j *JWTService) GetRefreshTokenExpiry() time.Duration {
	return j.refreshExpiry
}

// IsTokenExpired checks if a token is expired
func (j *JWTService) IsTokenExpired(tokenString string) bool {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return true
	}

	return time.Now().Unix() > claims.Exp
}

// RefreshTokenPair generates a new token pair from a refresh token
func (j *JWTService) RefreshTokenPair(refreshToken string, user *User, roles []string) (string, string, error) {
	// Validate the refresh token
	userID, err := j.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Verify the user ID matches
	if userID != user.ID {
		return "", "", errors.New("refresh token user mismatch")
	}

	// Generate new tokens
	accessToken, err := j.GenerateAccessToken(user, roles)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := j.GenerateRefreshToken(user)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, newRefreshToken, nil
}

// RevokeToken marks a token as revoked (in a real implementation, this would be stored in a blacklist)
func (j *JWTService) RevokeToken(tokenString string) error {
	// In a production system, you would store revoked tokens in a database
	// or use a token blacklist. For now, we'll just validate the token format.
	_, err := j.ValidateToken(tokenString)
	return err
}

// GetTokenInfo extracts information from a token without validating it
func (j *JWTService) GetTokenInfo(tokenString string) (*JWTClaims, error) {
	// Parse token without validation to extract claims
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &JWTClaims{})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok {
		return claims, nil
	}

	return nil, errors.New("invalid token format")
}

// GenerateAPIKeyToken generates a token for API key authentication
func (j *JWTService) GenerateAPIKeyToken(apiKey *APIKey, user *User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"api_key_id":  apiKey.ID,
		"user_id":     user.ID,
		"username":    user.Username,
		"email":       user.Email,
		"type":        "api_key",
		"permissions": apiKey.Permissions,
		"exp":         now.Add(j.accessExpiry).Unix(),
		"iat":         now.Unix(),
		"iss":         j.issuer,
		"aud":         j.audience,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// ValidateAPIKeyToken validates an API key token
func (j *JWTService) ValidateAPIKeyToken(tokenString string) (*APIKeyClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &APIKeyClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*APIKeyClaims); ok && token.Valid {
		// Validate issuer and audience
		if claims.Iss != j.issuer {
			return nil, errors.New("invalid issuer")
		}
		if claims.Aud != j.audience {
			return nil, errors.New("invalid audience")
		}
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// APIKeyClaims represents claims for API key tokens
type APIKeyClaims struct {
	APIKeyID    string   `json:"api_key_id"`
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Type        string   `json:"type"`
	Permissions []string `json:"permissions"`
	Exp         int64    `json:"exp"`
	Iat         int64    `json:"iat"`
	Iss         string   `json:"iss"`
	Aud         string   `json:"aud"`
	jwt.RegisteredClaims
}
