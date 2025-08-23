package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// Service represents the unified discovery service
type Service struct {
	providers   map[string]Provider
	cache       *Cache
	filters     *FilterManager
	parallelism int
	mu          sync.RWMutex
}

// Provider defines the interface for cloud providers
type Provider interface {
	// Discovery operations
	Discover(ctx context.Context, options DiscoveryOptions) (*Result, error)
	DiscoverRegion(ctx context.Context, region string) ([]models.Resource, error)

	// Provider information
	Name() string
	Regions() []string
	Services() []string

	// Credential operations
	ValidateCredentials(ctx context.Context) error
	GetAccountInfo(ctx context.Context) (*AccountInfo, error)
}

// DiscoveryOptions configures discovery behavior
type DiscoveryOptions struct {
	Regions       []string               `json:"regions,omitempty"`
	Services      []string               `json:"services,omitempty"`
	ResourceTypes []string               `json:"resource_types,omitempty"`
	Filters       map[string]interface{} `json:"filters,omitempty"`
	Parallel      bool                   `json:"parallel"`
	MaxWorkers    int                    `json:"max_workers"`
	Timeout       time.Duration          `json:"timeout"`
	UseCache      bool                   `json:"use_cache"`
}

// Result contains discovery results
type Result struct {
	Provider      string                 `json:"provider"`
	AccountInfo   *AccountInfo           `json:"account_info"`
	Resources     []models.Resource      `json:"resources"`
	ResourceCount int                    `json:"resource_count"`
	Regions       []string               `json:"regions"`
	Services      []string               `json:"services"`
	Duration      time.Duration          `json:"duration"`
	Timestamp     time.Time              `json:"timestamp"`
	Errors        []DiscoveryError       `json:"errors,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	FromCache     bool                   `json:"from_cache"`
}

// AccountInfo represents cloud account information
type AccountInfo struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Provider string                 `json:"provider"`
	Regions  []string               `json:"regions,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// DiscoveryError represents an error during discovery
type DiscoveryError struct {
	Region    string    `json:"region,omitempty"`
	Service   string    `json:"service,omitempty"`
	Resource  string    `json:"resource,omitempty"`
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
	Retryable bool      `json:"retryable"`
}

// NewService creates a new discovery service
func NewService() *Service {
	return &Service{
		providers:   make(map[string]Provider),
		cache:       NewCache(15 * time.Minute),
		filters:     NewFilterManager(),
		parallelism: 10,
	}
}

// RegisterProvider registers a cloud provider
func (s *Service) RegisterProvider(name string, provider Provider) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}

	s.providers[name] = provider
	return nil
}

// DiscoverAll discovers resources across all providers
func (s *Service) DiscoverAll(ctx context.Context) map[string]*Result {
	s.mu.RLock()
	providers := make(map[string]Provider, len(s.providers))
	for k, v := range s.providers {
		providers[k] = v
	}
	s.mu.RUnlock()

	if len(providers) == 0 {
		return make(map[string]*Result)
	}

	results := make(map[string]*Result)
	var mu sync.Mutex
	options := DiscoveryOptions{Parallel: true}

	if options.Parallel {
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, s.parallelism)

		for name, provider := range providers {
			wg.Add(1)
			go func(n string, p Provider) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				result, err := s.discoverProvider(ctx, n, p, options)
				if err != nil {
					result = &Result{
						Provider:  n,
						Timestamp: time.Now(),
						Errors:    []DiscoveryError{{Error: err.Error()}},
					}
				}

				mu.Lock()
				results[n] = result
				mu.Unlock()
			}(name, provider)
		}
		wg.Wait()
	} else {
		for name, provider := range providers {
			result, err := s.discoverProvider(ctx, name, provider, options)
			if err != nil {
				result = &Result{
					Provider:  name,
					Timestamp: time.Now(),
					Errors:    []DiscoveryError{{Error: err.Error()}},
				}
			}
			results[name] = result
		}
	}

	return results
}

// DiscoverProvider discovers resources for a specific provider
func (s *Service) DiscoverProvider(ctx context.Context, providerName string, options DiscoveryOptions) (*Result, error) {
	s.mu.RLock()
	provider, exists := s.providers[providerName]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("provider %s not registered", providerName)
	}

	return s.discoverProvider(ctx, providerName, provider, options)
}

// discoverProvider performs discovery for a single provider
func (s *Service) discoverProvider(ctx context.Context, name string, provider Provider, options DiscoveryOptions) (*Result, error) {
	// Check cache if enabled
	if options.UseCache {
		if cached, ok := s.cache.Get(name); ok {
			if result, ok := cached.(*Result); ok {
				result.FromCache = true
				return result, nil
			}
		}
	}

	// Set timeout if specified
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	// Perform discovery
	startTime := time.Now()
	result, err := provider.Discover(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("discovery failed for %s: %w", name, err)
	}

	// Enrich result
	result.Provider = name
	result.Duration = time.Since(startTime)
	result.Timestamp = time.Now()
	result.ResourceCount = len(result.Resources)

	// Apply filters
	if s.filters != nil && len(options.Filters) > 0 {
		result.Resources = s.filters.ApplyFilters(result.Resources, options.Filters)
		result.ResourceCount = len(result.Resources)
	}

	// Update cache
	if options.UseCache {
		s.cache.Set(name, result)
	}

	return result, nil
}

// GetProviders returns list of registered providers
func (s *Service) GetProviders() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	providers := make([]string, 0, len(s.providers))
	for name := range s.providers {
		providers = append(providers, name)
	}
	return providers
}

// GetProviderStatus returns the status of all providers
func (s *Service) GetProviderStatus(ctx context.Context) map[string]ProviderStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := make(map[string]ProviderStatus)

	for name, provider := range s.providers {
		ps := ProviderStatus{
			Name:      name,
			Available: true,
		}

		// Check credentials
		if err := provider.ValidateCredentials(ctx); err != nil {
			ps.Available = false
			ps.Error = err.Error()
		} else {
			ps.Regions = provider.Regions()
			ps.Services = provider.Services()

			if info, err := provider.GetAccountInfo(ctx); err == nil {
				ps.AccountInfo = info
			}
		}

		status[name] = ps
	}

	return status
}

// ProviderStatus represents the status of a provider
type ProviderStatus struct {
	Name        string       `json:"name"`
	Available   bool         `json:"available"`
	Error       string       `json:"error,omitempty"`
	Regions     []string     `json:"regions,omitempty"`
	Services    []string     `json:"services,omitempty"`
	AccountInfo *AccountInfo `json:"account_info,omitempty"`
}

// Statistics generates discovery statistics
func (s *Service) Statistics(results map[string]*Result) *Stats {
	stats := &Stats{
		TotalResources: 0,
		ByProvider:     make(map[string]int),
		ByRegion:       make(map[string]int),
		ByType:         make(map[string]int),
		Errors:         0,
	}

	for provider, result := range results {
		stats.TotalResources += result.ResourceCount
		stats.ByProvider[provider] = result.ResourceCount
		stats.Errors += len(result.Errors)

		for _, resource := range result.Resources {
			stats.ByRegion[resource.Region]++
			stats.ByType[resource.Type]++
		}

		if result.Duration > stats.TotalDuration {
			stats.TotalDuration = result.Duration
		}
	}

	return stats
}

// Stats represents discovery statistics
type Stats struct {
	TotalResources int            `json:"total_resources"`
	ByProvider     map[string]int `json:"by_provider"`
	ByRegion       map[string]int `json:"by_region"`
	ByType         map[string]int `json:"by_type"`
	TotalDuration  time.Duration  `json:"total_duration"`
	Errors         int            `json:"errors"`
}

// GetAccounts returns all available accounts across providers
func (s *Service) GetAccounts(ctx context.Context) ([]AccountInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var accounts []AccountInfo
	for name, provider := range s.providers {
		if info, err := provider.GetAccountInfo(ctx); err == nil {
			info.Provider = name
			accounts = append(accounts, *info)
		}
	}
	return accounts, nil
}

// DiscoverAccountResources discovers resources for a specific account
func (s *Service) DiscoverAccountResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var allResources []models.Resource
	for _, provider := range s.providers {
		result, err := provider.Discover(ctx, DiscoveryOptions{})
		if err != nil {
			continue
		}
		// Filter by account if needed
		for _, resource := range result.Resources {
			if resource.AccountID == accountID || accountID == "" {
				allResources = append(allResources, resource)
			}
		}
	}
	return allResources, nil
}

// ConvertToModelsResources converts resources to models.Resource format
func (s *Service) ConvertToModelsResources(resources []models.Resource) []models.Resource {
	// Already in the correct format
	return resources
}
