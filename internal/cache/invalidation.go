package cache

import (
	"fmt"
	"strings"
	"sync"

	"github.com/catherinevee/driftmgr/internal/events"
)

// InvalidationHandler handles cache invalidation based on events
type InvalidationHandler struct {
	cache    *GlobalCache
	eventBus *events.EventBus
	subscriptions []*events.Subscription
	rules    []InvalidationRule
	mu       sync.RWMutex
}

// InvalidationRule defines when and how to invalidate cache
type InvalidationRule struct {
	EventType    events.EventType
	CachePattern string
	Handler      func(event events.Event) []string
}

// NewInvalidationHandler creates a new cache invalidation handler
func NewInvalidationHandler(cache *GlobalCache, eventBus *events.EventBus) *InvalidationHandler {
	handler := &InvalidationHandler{
		cache:    cache,
		eventBus: eventBus,
		subscriptions: []*events.Subscription{},
	}
	
	// Define invalidation rules
	handler.rules = []InvalidationRule{
		// Resource changes invalidate discovery cache
		{
			EventType: events.ResourceCreated,
			Handler: func(e events.Event) []string {
				return handler.getResourceCacheKeys(e)
			},
		},
		{
			EventType: events.ResourceUpdated,
			Handler: func(e events.Event) []string {
				return handler.getResourceCacheKeys(e)
			},
		},
		{
			EventType: events.ResourceDeleted,
			Handler: func(e events.Event) []string {
				return handler.getResourceCacheKeys(e)
			},
		},
		// Remediation invalidates related resources
		{
			EventType: events.RemediationCompleted,
			Handler: func(e events.Event) []string {
				return handler.getRemediationCacheKeys(e)
			},
		},
		// State changes invalidate state cache
		{
			EventType: events.StateImported,
			Handler: func(e events.Event) []string {
				return []string{"state:*", "drift:*"}
			},
		},
		{
			EventType: events.StateDeleted,
			Handler: func(e events.Event) []string {
				return []string{"state:*", "drift:*"}
			},
		},
		// Manual cache clear
		{
			EventType: events.CacheCleared,
			Handler: func(e events.Event) []string {
				return []string{"*"} // Clear all
			},
		},
	}
	
	return handler
}

// Start begins listening for invalidation events
func (h *InvalidationHandler) Start() {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	// Subscribe to each event type
	for _, rule := range h.rules {
		sub := h.eventBus.SubscribeToType(rule.EventType, h.createHandler(rule))
		h.subscriptions = append(h.subscriptions, sub)
	}
}

// Stop stops listening for invalidation events
func (h *InvalidationHandler) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	for _, sub := range h.subscriptions {
		h.eventBus.Unsubscribe(sub)
	}
	h.subscriptions = []*events.Subscription{}
}

// createHandler creates an event handler for a rule
func (h *InvalidationHandler) createHandler(rule InvalidationRule) events.Handler {
	return func(event events.Event) {
		// Get cache keys to invalidate
		keys := rule.Handler(event)
		
		// Invalidate each key pattern
		for _, pattern := range keys {
			h.invalidatePattern(pattern)
		}
		
		// Publish cache invalidation event
		h.eventBus.Publish(events.Event{
			Type:   events.CacheInvalidated,
			Source: "InvalidationHandler",
			Data: map[string]interface{}{
				"trigger_event": event.Type,
				"patterns":      keys,
			},
		})
	}
}

// invalidatePattern invalidates cache entries matching a pattern
func (h *InvalidationHandler) invalidatePattern(pattern string) {
	if pattern == "*" {
		// Clear entire cache
		h.cache.Clear()
	} else if strings.Contains(pattern, "*") {
		// Invalidate pattern
		prefix := strings.Split(pattern, "*")[0]
		h.cache.InvalidatePattern(prefix)
	} else {
		// Delete specific key
		h.cache.Delete(pattern)
	}
}

// getResourceCacheKeys returns cache keys for a resource event
func (h *InvalidationHandler) getResourceCacheKeys(event events.Event) []string {
	keys := []string{}
	
	// Extract resource info from event data
	if provider, ok := event.Data["provider"].(string); ok {
		keys = append(keys, fmt.Sprintf("discovery:%s:*", provider))
		
		if region, ok := event.Data["region"].(string); ok {
			keys = append(keys, fmt.Sprintf("discovery:%s:%s", provider, region))
		}
	}
	
	// Invalidate resource-specific cache
	if resourceID, ok := event.Data["resource_id"].(string); ok {
		keys = append(keys, fmt.Sprintf("resource:%s", resourceID))
	}
	
	// Invalidate drift detection cache
	keys = append(keys, "drift:*")
	
	return keys
}

// getRemediationCacheKeys returns cache keys for remediation events
func (h *InvalidationHandler) getRemediationCacheKeys(event events.Event) []string {
	keys := []string{}
	
	// Invalidate affected resources
	if resources, ok := event.Data["affected_resources"].([]interface{}); ok {
		for _, r := range resources {
			if resourceID, ok := r.(string); ok {
				keys = append(keys, fmt.Sprintf("resource:%s", resourceID))
			}
		}
	}
	
	// Invalidate provider cache
	if provider, ok := event.Data["provider"].(string); ok {
		keys = append(keys, fmt.Sprintf("discovery:%s:*", provider))
	}
	
	// Always invalidate drift cache after remediation
	keys = append(keys, "drift:*")
	
	return keys
}

// InvalidateProvider invalidates all cache for a provider
func (h *InvalidationHandler) InvalidateProvider(provider string) {
	pattern := fmt.Sprintf("discovery:%s:*", provider)
	h.invalidatePattern(pattern)
	
	// Publish event
	h.eventBus.Publish(events.Event{
		Type:   events.CacheInvalidated,
		Source: "InvalidationHandler",
		Data: map[string]interface{}{
			"provider": provider,
			"pattern":  pattern,
		},
	})
}

// InvalidateRegion invalidates cache for a specific region
func (h *InvalidationHandler) InvalidateRegion(provider, region string) {
	pattern := fmt.Sprintf("discovery:%s:%s", provider, region)
	h.cache.Delete(pattern)
	
	// Also invalidate incremental baseline
	baselineKey := fmt.Sprintf("discovery:baseline:%s", provider)
	h.cache.Delete(baselineKey)
	
	// Publish event
	h.eventBus.Publish(events.Event{
		Type:   events.CacheInvalidated,
		Source: "InvalidationHandler",
		Data: map[string]interface{}{
			"provider": provider,
			"region":   region,
		},
	})
}

// InvalidateResource invalidates cache for a specific resource
func (h *InvalidationHandler) InvalidateResource(resourceID string, provider string, region string) {
	// Invalidate specific resource
	h.cache.Delete(fmt.Sprintf("resource:%s", resourceID))
	
	// Invalidate discovery cache for the region
	h.InvalidateRegion(provider, region)
}