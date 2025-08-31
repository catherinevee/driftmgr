package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/catherinevee/driftmgr/internal/logging"
)

// RedisCache provides distributed caching using Redis
type RedisCache struct {
	client      redis.UniversalClient
	prefix      string
	defaultTTL  time.Duration
	localCache  *GlobalCache // L1 cache for frequently accessed items
	logger      *logging.Logger
	metrics     *CacheMetrics
}

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	Hits       int64
	Misses     int64
	Sets       int64
	Deletes    int64
	Errors     int64
	LocalHits  int64
	RemoteHits int64
}

// RedisConfig configures Redis connection
type RedisConfig struct {
	// Standalone Redis
	Addr     string
	Password string
	DB       int

	// Redis Cluster
	ClusterAddrs []string

	// Redis Sentinel
	MasterName    string
	SentinelAddrs []string

	// Connection options
	PoolSize        int
	MinIdleConns    int
	MaxRetries      int
	ConnectTimeout  time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration

	// Cache options
	KeyPrefix   string
	DefaultTTL  time.Duration
	EnableLocal bool // Enable local L1 cache
}

// DefaultRedisConfig returns default Redis configuration
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Addr:           "localhost:6379",
		Password:       "",
		DB:             0,
		PoolSize:       10,
		MinIdleConns:   5,
		MaxRetries:     3,
		ConnectTimeout: 5 * time.Second,
		ReadTimeout:    3 * time.Second,
		WriteTimeout:   3 * time.Second,
		KeyPrefix:      "driftmgr:",
		DefaultTTL:     5 * time.Minute,
		EnableLocal:    true,
	}
}

// NewRedisCache creates a new Redis-backed cache
func NewRedisCache(config *RedisConfig) (*RedisCache, error) {
	if config == nil {
		config = DefaultRedisConfig()
	}

	// Create Redis client based on configuration
	var client redis.UniversalClient

	if len(config.ClusterAddrs) > 0 {
		// Redis Cluster mode
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:          config.ClusterAddrs,
			Password:       config.Password,
			PoolSize:       config.PoolSize,
			MinIdleConns:   config.MinIdleConns,
			MaxRetries:     config.MaxRetries,
			ReadTimeout:    config.ReadTimeout,
			WriteTimeout:   config.WriteTimeout,
			DialTimeout:    config.ConnectTimeout,
		})
	} else if len(config.SentinelAddrs) > 0 && config.MasterName != "" {
		// Redis Sentinel mode
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:     config.MasterName,
			SentinelAddrs:  config.SentinelAddrs,
			Password:       config.Password,
			DB:             config.DB,
			PoolSize:       config.PoolSize,
			MinIdleConns:   config.MinIdleConns,
			MaxRetries:     config.MaxRetries,
			ReadTimeout:    config.ReadTimeout,
			WriteTimeout:   config.WriteTimeout,
			DialTimeout:    config.ConnectTimeout,
		})
	} else {
		// Standalone Redis
		client = redis.NewClient(&redis.Options{
			Addr:           config.Addr,
			Password:       config.Password,
			DB:             config.DB,
			PoolSize:       config.PoolSize,
			MinIdleConns:   config.MinIdleConns,
			MaxRetries:     config.MaxRetries,
			ReadTimeout:    config.ReadTimeout,
			WriteTimeout:   config.WriteTimeout,
			DialTimeout:    config.ConnectTimeout,
		})
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	cache := &RedisCache{
		client:     client,
		prefix:     config.KeyPrefix,
		defaultTTL: config.DefaultTTL,
		logger:     logging.GetLogger(),
		metrics:    &CacheMetrics{},
	}

	// Initialize local L1 cache if enabled
	if config.EnableLocal {
		cache.localCache = GetGlobalCache()
	}

	cache.logger.Info("Redis cache initialized", map[string]interface{}{
		"prefix":      config.KeyPrefix,
		"default_ttl": config.DefaultTTL.String(),
		"local_cache": config.EnableLocal,
	})

	return cache, nil
}

// Set stores a value in the cache
func (rc *RedisCache) Set(key string, value interface{}, ttl time.Duration) error {
	ctx := context.Background()
	
	// Use default TTL if not specified
	if ttl == 0 {
		ttl = rc.defaultTTL
	}

	// Serialize value
	data, err := json.Marshal(value)
	if err != nil {
		rc.metrics.Errors++
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// Store in Redis
	fullKey := rc.prefix + key
	err = rc.client.Set(ctx, fullKey, data, ttl).Err()
	if err != nil {
		rc.metrics.Errors++
		return fmt.Errorf("failed to set value in Redis: %w", err)
	}

	rc.metrics.Sets++

	// Also store in local cache if enabled
	if rc.localCache != nil {
		rc.localCache.Set(key, value, ttl)
	}

	return nil
}

// Get retrieves a value from the cache
func (rc *RedisCache) Get(key string) (interface{}, bool) {
	ctx := context.Background()

	// Check local cache first if enabled
	if rc.localCache != nil {
		if value, found := rc.localCache.Get(key); found {
			rc.metrics.LocalHits++
			rc.metrics.Hits++
			return value, true
		}
	}

	// Get from Redis
	fullKey := rc.prefix + key
	data, err := rc.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			rc.metrics.Misses++
		} else {
			rc.metrics.Errors++
			rc.logger.Error("Failed to get value from Redis", err, map[string]interface{}{
				"key": key,
			})
		}
		return nil, false
	}

	// Deserialize value
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		rc.metrics.Errors++
		rc.logger.Error("Failed to unmarshal value", err, map[string]interface{}{
			"key": key,
		})
		return nil, false
	}

	rc.metrics.RemoteHits++
	rc.metrics.Hits++

	// Store in local cache if enabled
	if rc.localCache != nil {
		// Get TTL from Redis
		ttl, _ := rc.client.TTL(ctx, fullKey).Result()
		if ttl > 0 {
			rc.localCache.Set(key, value, ttl)
		}
	}

	return value, true
}

// GetWithAge retrieves a value with its age
func (rc *RedisCache) GetWithAge(key string) (interface{}, bool, time.Duration) {
	ctx := context.Background()

	// Check local cache first
	if rc.localCache != nil {
		if value, found, age := rc.localCache.GetWithAge(key); found {
			rc.metrics.LocalHits++
			rc.metrics.Hits++
			return value, true, age
		}
	}

	fullKey := rc.prefix + key
	
	// Get value and TTL in a pipeline for efficiency
	pipe := rc.client.Pipeline()
	getCmd := pipe.Get(ctx, fullKey)
	ttlCmd := pipe.TTL(ctx, fullKey)
	_, err := pipe.Exec(ctx)

	if err != nil {
		if errors.Is(err, redis.Nil) {
			rc.metrics.Misses++
		} else {
			rc.metrics.Errors++
		}
		return nil, false, 0
	}

	data, err := getCmd.Bytes()
	if err != nil {
		return nil, false, 0
	}

	ttl, err := ttlCmd.Result()
	if err != nil || ttl <= 0 {
		return nil, false, 0
	}

	// Deserialize value
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		rc.metrics.Errors++
		return nil, false, 0
	}

	// Calculate age (original TTL - remaining TTL)
	age := rc.defaultTTL - ttl
	if age < 0 {
		age = 0
	}

	rc.metrics.RemoteHits++
	rc.metrics.Hits++

	// Update local cache
	if rc.localCache != nil {
		rc.localCache.Set(key, value, ttl)
	}

	return value, true, age
}

// Delete removes a value from the cache
func (rc *RedisCache) Delete(key string) error {
	ctx := context.Background()
	
	fullKey := rc.prefix + key
	err := rc.client.Del(ctx, fullKey).Err()
	if err != nil {
		rc.metrics.Errors++
		return err
	}

	rc.metrics.Deletes++

	// Also delete from local cache
	if rc.localCache != nil {
		rc.localCache.Delete(key)
	}

	return nil
}

// InvalidatePattern removes all keys matching a pattern
func (rc *RedisCache) InvalidatePattern(pattern string) error {
	ctx := context.Background()
	
	fullPattern := rc.prefix + pattern + "*"
	
	// Use SCAN to avoid blocking Redis
	iter := rc.client.Scan(ctx, 0, fullPattern, 100).Iterator()
	
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
		
		// Delete in batches
		if len(keys) >= 100 {
			if err := rc.client.Del(ctx, keys...).Err(); err != nil {
				rc.metrics.Errors++
				rc.logger.Error("Failed to delete keys", err, map[string]interface{}{
					"pattern": pattern,
				})
			}
			keys = keys[:0]
		}
	}
	
	// Delete remaining keys
	if len(keys) > 0 {
		if err := rc.client.Del(ctx, keys...).Err(); err != nil {
			rc.metrics.Errors++
			return err
		}
	}
	
	if err := iter.Err(); err != nil {
		rc.metrics.Errors++
		return err
	}

	// Invalidate local cache pattern
	if rc.localCache != nil {
		rc.localCache.InvalidatePattern(pattern)
	}

	return nil
}

// Clear removes all cache entries
func (rc *RedisCache) Clear() error {
	ctx := context.Background()
	
	// Use SCAN to find all keys with our prefix
	pattern := rc.prefix + "*"
	iter := rc.client.Scan(ctx, 0, pattern, 100).Iterator()
	
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
		
		// Delete in batches
		if len(keys) >= 1000 {
			if err := rc.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
			keys = keys[:0]
		}
	}
	
	// Delete remaining keys
	if len(keys) > 0 {
		if err := rc.client.Del(ctx, keys...).Err(); err != nil {
			return err
		}
	}

	// Clear local cache
	if rc.localCache != nil {
		rc.localCache.Clear()
	}

	return iter.Err()
}

// GetStats returns cache statistics
func (rc *RedisCache) GetStats() map[string]interface{} {
	ctx := context.Background()
	
	// Get Redis info
	info := rc.client.Info(ctx, "stats").Val()
	
	stats := map[string]interface{}{
		"hits":        rc.metrics.Hits,
		"misses":      rc.metrics.Misses,
		"sets":        rc.metrics.Sets,
		"deletes":     rc.metrics.Deletes,
		"errors":      rc.metrics.Errors,
		"local_hits":  rc.metrics.LocalHits,
		"remote_hits": rc.metrics.RemoteHits,
		"hit_rate":    float64(rc.metrics.Hits) / float64(rc.metrics.Hits+rc.metrics.Misses) * 100,
		"redis_info":  info,
	}
	
	// Add local cache stats if enabled
	if rc.localCache != nil {
		stats["local_cache"] = rc.localCache.GetStats()
	}
	
	return stats
}

// Close closes the Redis connection
func (rc *RedisCache) Close() error {
	return rc.client.Close()
}

// Publish publishes a cache invalidation message
func (rc *RedisCache) Publish(ctx context.Context, channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	
	return rc.client.Publish(ctx, rc.prefix+channel, data).Err()
}

// Subscribe subscribes to cache invalidation messages
func (rc *RedisCache) Subscribe(ctx context.Context, channels ...string) <-chan CacheMessage {
	// Prefix channels
	prefixedChannels := make([]string, len(channels))
	for i, ch := range channels {
		prefixedChannels[i] = rc.prefix + ch
	}
	
	pubsub := rc.client.Subscribe(ctx, prefixedChannels...)
	ch := make(chan CacheMessage, 100)
	
	go func() {
		defer close(ch)
		defer pubsub.Close()
		
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-pubsub.Channel():
				var data interface{}
				if err := json.Unmarshal([]byte(msg.Payload), &data); err != nil {
					rc.logger.Error("Failed to unmarshal pub/sub message", err, map[string]interface{}{
						"channel": msg.Channel,
					})
					continue
				}
				
				ch <- CacheMessage{
					Channel: strings.TrimPrefix(msg.Channel, rc.prefix),
					Data:    data,
				}
			}
		}
	}()
	
	return ch
}

// CacheMessage represents a cache pub/sub message
type CacheMessage struct {
	Channel string
	Data    interface{}
}

// SetIfNotExists sets a value only if it doesn't exist
func (rc *RedisCache) SetIfNotExists(key string, value interface{}, ttl time.Duration) (bool, error) {
	ctx := context.Background()
	
	if ttl == 0 {
		ttl = rc.defaultTTL
	}
	
	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}
	
	fullKey := rc.prefix + key
	result := rc.client.SetNX(ctx, fullKey, data, ttl)
	
	if result.Err() != nil {
		return false, result.Err()
	}
	
	success := result.Val()
	
	// Update local cache if successful
	if success && rc.localCache != nil {
		rc.localCache.Set(key, value, ttl)
	}
	
	return success, nil
}

// Increment increments a counter
func (rc *RedisCache) Increment(key string) (int64, error) {
	ctx := context.Background()
	fullKey := rc.prefix + key
	return rc.client.Incr(ctx, fullKey).Result()
}

// Decrement decrements a counter
func (rc *RedisCache) Decrement(key string) (int64, error) {
	ctx := context.Background()
	fullKey := rc.prefix + key
	return rc.client.Decr(ctx, fullKey).Result()
}

// Keys returns all cache keys matching a pattern
func (rc *RedisCache) Keys(pattern string) []string {
	ctx := context.Background()
	fullPattern := rc.prefix + pattern
	
	// Use SCAN for non-blocking key iteration
	iter := rc.client.Scan(ctx, 0, fullPattern, 100).Iterator()
	
	var keys []string
	for iter.Next(ctx) {
		key := iter.Val()
		// Remove prefix from key
		if strings.HasPrefix(key, rc.prefix) {
			keys = append(keys, strings.TrimPrefix(key, rc.prefix))
		}
	}
	
	if err := iter.Err(); err != nil {
		rc.logger.Error("Failed to scan keys", err, map[string]interface{}{
			"pattern": pattern,
		})
	}
	
	return keys
}