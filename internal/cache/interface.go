package cache

import (
	"time"
)

// Cache defines the interface for caching implementations
type Cache interface {
	// Get retrieves a value from the cache
	Get(key string) (interface{}, bool)
	
	// Set stores a value in the cache with a TTL
	Set(key string, value interface{}, ttl time.Duration)
	
	// Delete removes a value from the cache
	Delete(key string) error
	
	// Clear removes all values from the cache
	Clear() error
	
	// GetWithAge retrieves a value with its age
	GetWithAge(key string) (interface{}, bool, time.Duration)
	
	// InvalidatePattern removes all keys matching a pattern
	InvalidatePattern(pattern string) error
	
	// GetStats returns cache statistics
	GetStats() map[string]interface{}
	
	// Keys returns all cache keys matching a pattern
	Keys(pattern string) []string
}