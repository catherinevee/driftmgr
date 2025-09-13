package config

import "time"

// DiscoveryConfig represents discovery configuration
type DiscoveryConfig struct {
	MaxConcurrency int           `json:"max_concurrency"`
	Timeout        int           `json:"timeout"` // seconds
	RetryCount     int           `json:"retry_count"`
	RetryDelay     time.Duration `json:"retry_delay"`
	CacheTTL       int           `json:"cache_ttl"` // seconds
}
