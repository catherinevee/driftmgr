package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// TokenManager manages authentication tokens
type TokenManager struct {
	secretKey []byte
	tokens    map[string]*TokenInfo
}

// TokenInfo represents token information
type TokenInfo struct {
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// NewTokenManager creates a new token manager
func NewTokenManager(secretKey []byte) *TokenManager {
	return &TokenManager{
		secretKey: secretKey,
		tokens:    make(map[string]*TokenInfo),
	}
}

// GenerateToken generates a new authentication token
func (tm *TokenManager) GenerateToken(userID string, duration time.Duration) (string, error) {
	// Generate random bytes for token
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Create token hash
	tokenData := fmt.Sprintf("%s:%s:%d", userID, hex.EncodeToString(randomBytes), time.Now().Unix())
	hash := sha256.Sum256([]byte(tokenData + string(tm.secretKey)))
	token := hex.EncodeToString(hash[:])

	// Store token info
	tm.tokens[token] = &TokenInfo{
		UserID:    userID,
		ExpiresAt: time.Now().Add(duration),
		CreatedAt: time.Now(),
	}

	return token, nil
}

// ValidateToken validates an authentication token
func (tm *TokenManager) ValidateToken(token string) (string, bool) {
	info, exists := tm.tokens[token]
	if !exists {
		return "", false
	}

	if time.Now().After(info.ExpiresAt) {
		// Token has expired, remove it
		delete(tm.tokens, token)
		return "", false
	}

	return info.UserID, true
}

// RevokeToken revokes an authentication token
func (tm *TokenManager) RevokeToken(token string) bool {
	if _, exists := tm.tokens[token]; exists {
		delete(tm.tokens, token)
		return true
	}
	return false
}

// CleanupExpiredTokens removes expired tokens
func (tm *TokenManager) CleanupExpiredTokens() {
	now := time.Now()
	for token, info := range tm.tokens {
		if now.After(info.ExpiresAt) {
			delete(tm.tokens, token)
		}
	}
}

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Get existing requests for this key
	requests, exists := rl.requests[key]
	if !exists {
		requests = []time.Time{}
	}

	// Remove old requests outside the window
	var validRequests []time.Time
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Check if we're under the limit
	if len(validRequests) >= rl.limit {
		return false
	}

	// Add current request
	validRequests = append(validRequests, now)
	rl.requests[key] = validRequests

	return true
}

// Cleanup removes old entries to prevent memory leaks
func (rl *RateLimiter) Cleanup() {
	cutoff := time.Now().Add(-rl.window)
	for key, requests := range rl.requests {
		var validRequests []time.Time
		for _, reqTime := range requests {
			if reqTime.After(cutoff) {
				validRequests = append(validRequests, reqTime)
			}
		}
		if len(validRequests) == 0 {
			delete(rl.requests, key)
		} else {
			rl.requests[key] = validRequests
		}
	}
}
