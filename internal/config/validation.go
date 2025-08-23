package config

import (
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/constants"
)

// Validate validates the entire configuration
func (c *Config) Validate() error {
	if err := c.validateDiscovery(); err != nil {
		return fmt.Errorf("discovery configuration invalid: %w", err)
	}

	if err := c.validateSecurity(); err != nil {
		return fmt.Errorf("security configuration invalid: %w", err)
	}

	// Notifications and Remediation validation removed as these configs don't exist yet

	return nil
}

// validateDiscovery validates discovery configuration
func (c *Config) validateDiscovery() error {
	d := &c.Discovery

	// Validate timeout
	if d.Timeout < time.Second {
		return fmt.Errorf("discovery timeout too short: %v (minimum 1s)", d.Timeout)
	}
	if d.Timeout > 1*time.Hour {
		return fmt.Errorf("discovery timeout too long: %v (maximum 1h)", d.Timeout)
	}

	// Validate concurrency
	if d.ConcurrencyLimit < 1 {
		d.ConcurrencyLimit = constants.DefaultMaxConcurrency
	}
	if d.ConcurrencyLimit > 100 {
		return fmt.Errorf("concurrency limit too high: %d (maximum 100)", d.ConcurrencyLimit)
	}

	// Validate batch size
	if d.BatchSize < 1 {
		d.BatchSize = constants.DefaultBatchSize
	}
	if d.BatchSize > constants.MaxResourcesPerBatch {
		return fmt.Errorf("batch size too large: %d (maximum %d)", d.BatchSize, constants.MaxResourcesPerBatch)
	}

	// Validate retry attempts
	if d.RetryAttempts < 0 {
		d.RetryAttempts = constants.DefaultMaxRetries
	}
	if d.RetryAttempts > 10 {
		return fmt.Errorf("too many retry attempts: %d (maximum 10)", d.RetryAttempts)
	}

	// Validate retry delay
	if d.RetryDelay < 0 {
		d.RetryDelay = constants.DefaultRetryDelay
	}
	if d.RetryDelay > 1*time.Minute {
		return fmt.Errorf("retry delay too long: %v (maximum 1m)", d.RetryDelay)
	}

	// Validate cache settings
	if d.EnableCaching {
		if d.CacheTTL < time.Second {
			d.CacheTTL = constants.DefaultCacheTTL
		}
		if d.CacheTTL > 1*time.Hour {
			return fmt.Errorf("cache TTL too long: %v (maximum 1h)", d.CacheTTL)
		}
		if d.CacheMaxSize < 1 {
			d.CacheMaxSize = constants.DefaultCacheMaxSize
		}
		if d.CacheMaxSize > 10000 {
			return fmt.Errorf("cache size too large: %d (maximum 10000)", d.CacheMaxSize)
		}
	}

	// Validate API timeout
	if d.APITimeout < 0 {
		d.APITimeout = constants.DefaultAPITimeout
	}
	if d.APITimeout > 5*time.Minute {
		return fmt.Errorf("API timeout too long: %v (maximum 5m)", d.APITimeout)
	}

	return nil
}

// validateSecurity validates security configuration
func (c *Config) validateSecurity() error {
	// Basic security validation
	// Most security fields are handled by the existing security package
	return nil
}


// ValidateProviderConfig validates provider-specific configuration
func ValidateProviderConfig(provider string, config ProviderConfig) error {
	// Validate common fields
	if len(config.Regions) == 0 {
		return fmt.Errorf("no regions specified for provider %s", provider)
	}

	// Validate credentials based on provider
	switch provider {
	case constants.ProviderAWS:
		if config.Credentials["access_key_id"] == "" && config.Credentials["profile"] == "" {
			return fmt.Errorf("AWS credentials missing: need access_key_id or profile")
		}
	case constants.ProviderAzure:
		if config.Credentials["subscription_id"] == "" {
			return fmt.Errorf("Azure subscription ID missing")
		}
	case constants.ProviderGCP:
		if config.Credentials["project_id"] == "" {
			return fmt.Errorf("GCP project ID missing")
		}
	}

	// ResourceTypes and Tags validation removed as these fields don't exist in ProviderConfig

	return nil
}