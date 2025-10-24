package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"
)

// generateDiscoveryJobID generates a unique discovery job ID
func generateDiscoveryJobID() string {
	return fmt.Sprintf("discovery-%d-%s", time.Now().Unix(), generateShortID())
}

// getCurrentUserID returns a mock user ID for testing purposes
// In a real implementation, this would get the user ID from the context
func getCurrentUserID(ctx context.Context) string {
	// In a real implementation, you would extract the user ID from the context
	// For now, return a mock user ID
	return "user-123"
}

// generateShortID generates a short random ID
func generateShortID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// generateDriftID generates a unique drift ID
func generateDriftID() string {
	return fmt.Sprintf("drift-%d-%s", time.Now().Unix(), generateShortID())
}
