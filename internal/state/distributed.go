package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/logging"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// DistributedStateManager manages state across multiple instances
type DistributedStateManager struct {
	client     *clientv3.Client
	session    *concurrency.Session
	lockTTL    time.Duration
	watchers   map[string][]chan StateChange
	watchersMu sync.RWMutex
	logger     *logging.Logger
	localCache map[string]*StateEntry
	cacheMu    sync.RWMutex
}

// StateEntry represents a state entry with metadata
type StateEntry struct {
	Key       string                 `json:"key"`
	Value     interface{}            `json:"value"`
	Version   int64                  `json:"version"`
	Timestamp time.Time              `json:"timestamp"`
	Owner     string                 `json:"owner"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// StateChange represents a change in state
type StateChange struct {
	Type     string // "created", "updated", "deleted"
	Key      string
	OldValue *StateEntry
	NewValue *StateEntry
}

// Lock represents a distributed lock
type Lock struct {
	mutex   *concurrency.Mutex
	session *concurrency.Session
	key     string
	ctx     context.Context
	cancel  context.CancelFunc
}

// DistributedConfig holds configuration for distributed state
type DistributedConfig struct {
	Endpoints   []string
	DialTimeout time.Duration
	Username    string
	Password    string
	LockTTL     time.Duration
	Namespace   string
}

// NewDistributedStateManager creates a new distributed state manager
func NewDistributedStateManager(config *DistributedConfig) (*DistributedStateManager, error) {
	// Create etcd client
	etcdConfig := clientv3.Config{
		Endpoints:   config.Endpoints,
		DialTimeout: config.DialTimeout,
	}

	if config.Username != "" {
		etcdConfig.Username = config.Username
		etcdConfig.Password = config.Password
	}

	client, err := clientv3.New(etcdConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	// Create session for locks
	session, err := concurrency.NewSession(client,
		concurrency.WithTTL(int(config.LockTTL.Seconds())))
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	manager := &DistributedStateManager{
		client:     client,
		session:    session,
		lockTTL:    config.LockTTL,
		watchers:   make(map[string][]chan StateChange),
		logger:     logging.GetLogger(),
		localCache: make(map[string]*StateEntry),
	}

	// Start watcher for state changes
	go manager.watchStateChanges(context.Background(), config.Namespace)

	manager.logger.Info("Distributed state manager initialized", map[string]interface{}{
		"endpoints": config.Endpoints,
		"namespace": config.Namespace,
	})

	return manager, nil
}

// Get retrieves a value from distributed state
func (d *DistributedStateManager) Get(ctx context.Context, key string) (*StateEntry, error) {
	// Check local cache first
	d.cacheMu.RLock()
	if cached, exists := d.localCache[key]; exists {
		d.cacheMu.RUnlock()
		return cached, nil
	}
	d.cacheMu.RUnlock()

	// Fetch from etcd
	resp, err := d.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get key %s: %w", key, err)
	}

	if len(resp.Kvs) == 0 {
		return nil, errors.New("key not found")
	}

	var entry StateEntry
	if err := json.Unmarshal(resp.Kvs[0].Value, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Update local cache
	d.cacheMu.Lock()
	d.localCache[key] = &entry
	d.cacheMu.Unlock()

	return &entry, nil
}

// Set stores a value in distributed state with optimistic concurrency control
func (d *DistributedStateManager) Set(ctx context.Context, key string, value interface{}, metadata map[string]interface{}) error {
	// Get current version for CAS
	current, _ := d.Get(ctx, key)

	entry := &StateEntry{
		Key:       key,
		Value:     value,
		Version:   1,
		Timestamp: time.Now(),
		Owner:     d.getInstanceID(),
		Metadata:  metadata,
	}

	if current != nil {
		entry.Version = current.Version + 1

		// Check for conflicts
		if conflict := d.detectConflict(current, entry); conflict {
			// Resolve conflict
			resolved := d.resolveConflict(current, entry)
			entry = resolved
		}
	}

	// Marshal entry
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Compare-and-swap operation
	txn := d.client.Txn(ctx)

	if current != nil {
		// Update existing key with version check
		txn = txn.If(
			clientv3.Compare(clientv3.ModRevision(key), "=", current.Version),
		).Then(
			clientv3.OpPut(key, string(data)),
		).Else(
			clientv3.OpGet(key),
		)
	} else {
		// Create new key
		txn = txn.If(
			clientv3.Compare(clientv3.CreateRevision(key), "=", 0),
		).Then(
			clientv3.OpPut(key, string(data)),
		)
	}

	resp, err := txn.Commit()
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	if !resp.Succeeded {
		// Retry with backoff
		return d.retrySet(ctx, key, value, metadata, 3)
	}

	// Update local cache
	d.cacheMu.Lock()
	d.localCache[key] = entry
	d.cacheMu.Unlock()

	// Notify watchers
	d.notifyWatchers(key, &StateChange{
		Type:     "updated",
		Key:      key,
		OldValue: current,
		NewValue: entry,
	})

	d.logger.Debug("State updated", map[string]interface{}{
		"key":     key,
		"version": entry.Version,
	})

	return nil
}

// Delete removes a key from distributed state
func (d *DistributedStateManager) Delete(ctx context.Context, key string) error {
	// Get current value for notification
	current, _ := d.Get(ctx, key)

	_, err := d.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}

	// Remove from local cache
	d.cacheMu.Lock()
	delete(d.localCache, key)
	d.cacheMu.Unlock()

	// Notify watchers
	if current != nil {
		d.notifyWatchers(key, &StateChange{
			Type:     "deleted",
			Key:      key,
			OldValue: current,
			NewValue: nil,
		})
	}

	return nil
}

// AcquireLock acquires a distributed lock for a resource
func (d *DistributedStateManager) AcquireLock(ctx context.Context, resource string, ttl time.Duration) (*Lock, error) {
	// Create new session for this lock
	session, err := concurrency.NewSession(d.client,
		concurrency.WithTTL(int(ttl.Seconds())))
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Create mutex
	lockKey := path.Join("/locks", resource)
	mutex := concurrency.NewMutex(session, lockKey)

	// Try to acquire lock with timeout
	lockCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := mutex.Lock(lockCtx); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	// Create lock context for automatic release
	lockCtx, lockCancel := context.WithCancel(context.Background())

	lock := &Lock{
		mutex:   mutex,
		session: session,
		key:     resource,
		ctx:     lockCtx,
		cancel:  lockCancel,
	}

	// Start lock refresh goroutine
	go d.refreshLock(lock, ttl)

	d.logger.Debug("Lock acquired", map[string]interface{}{
		"resource": resource,
		"ttl":      ttl.String(),
	})

	logging.Audit("lock_acquired", d.getInstanceID(), "success", map[string]interface{}{
		"resource": resource,
	})

	return lock, nil
}

// ReleaseLock releases a distributed lock
func (d *DistributedStateManager) ReleaseLock(lock *Lock) error {
	if lock == nil {
		return errors.New("lock is nil")
	}

	// Cancel refresh goroutine
	lock.cancel()

	// Unlock mutex
	if err := lock.mutex.Unlock(context.Background()); err != nil {
		return fmt.Errorf("failed to unlock: %w", err)
	}

	// Close session
	lock.session.Close()

	d.logger.Debug("Lock released", map[string]interface{}{
		"resource": lock.key,
	})

	logging.Audit("lock_released", d.getInstanceID(), "success", map[string]interface{}{
		"resource": lock.key,
	})

	return nil
}

// Watch registers a watcher for state changes
func (d *DistributedStateManager) Watch(key string) <-chan StateChange {
	d.watchersMu.Lock()
	defer d.watchersMu.Unlock()

	ch := make(chan StateChange, 10)
	d.watchers[key] = append(d.watchers[key], ch)

	return ch
}

// Unwatch removes a watcher
func (d *DistributedStateManager) Unwatch(key string, ch <-chan StateChange) {
	d.watchersMu.Lock()
	defer d.watchersMu.Unlock()

	watchers := d.watchers[key]
	for i, watcher := range watchers {
		if watcher == ch {
			d.watchers[key] = append(watchers[:i], watchers[i+1:]...)
			close(watcher)
			break
		}
	}
}

// ListKeys lists all keys with a prefix
func (d *DistributedStateManager) ListKeys(ctx context.Context, prefix string) ([]string, error) {
	resp, err := d.client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithKeysOnly())
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	keys := make([]string, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		keys = append(keys, string(kv.Key))
	}

	return keys, nil
}

// Transaction executes a distributed transaction
func (d *DistributedStateManager) Transaction(ctx context.Context, ops []TransactionOp) error {
	txn := d.client.Txn(ctx)

	// Build transaction
	var conditions []clientv3.Cmp
	var thenOps []clientv3.Op

	for _, op := range ops {
		switch op.Type {
		case "check":
			conditions = append(conditions,
				clientv3.Compare(clientv3.Value(op.Key), "=", op.ExpectedValue))
		case "set":
			data, _ := json.Marshal(op.Value)
			thenOps = append(thenOps, clientv3.OpPut(op.Key, string(data)))
		case "delete":
			thenOps = append(thenOps, clientv3.OpDelete(op.Key))
		}
	}

	// Execute transaction
	txn = txn.If(conditions...).Then(thenOps...)
	resp, err := txn.Commit()
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	if !resp.Succeeded {
		return errors.New("transaction conditions not met")
	}

	return nil
}

// TransactionOp represents a transaction operation
type TransactionOp struct {
	Type          string // "check", "set", "delete"
	Key           string
	Value         interface{}
	ExpectedValue string
}

// Helper methods

func (d *DistributedStateManager) detectConflict(current, proposed *StateEntry) bool {
	// Check if there's a concurrent modification
	timeDiff := proposed.Timestamp.Sub(current.Timestamp)
	if timeDiff < 1*time.Second && proposed.Owner != current.Owner {
		return true
	}

	// Check for semantic conflicts
	if current.Metadata != nil && current.Metadata["locked"] == true {
		return true
	}

	return false
}

func (d *DistributedStateManager) resolveConflict(current, proposed *StateEntry) *StateEntry {
	resolved := &StateEntry{
		Key:       proposed.Key,
		Version:   current.Version + 1,
		Timestamp: time.Now(),
		Owner:     proposed.Owner,
		Metadata:  make(map[string]interface{}),
	}

	// Merge values based on type
	switch v := proposed.Value.(type) {
	case map[string]interface{}:
		// Deep merge for maps
		if currentMap, ok := current.Value.(map[string]interface{}); ok {
			resolved.Value = d.deepMerge(currentMap, v)
		} else {
			resolved.Value = v
		}
	case []interface{}:
		// Append for arrays
		if currentArray, ok := current.Value.([]interface{}); ok {
			resolved.Value = append(currentArray, v...)
		} else {
			resolved.Value = v
		}
	default:
		// Last-write-wins for primitive types
		resolved.Value = proposed.Value
	}

	// Merge metadata
	for k, v := range current.Metadata {
		resolved.Metadata[k] = v
	}
	for k, v := range proposed.Metadata {
		resolved.Metadata[k] = v
	}

	d.logger.Debug("Conflict resolved", map[string]interface{}{
		"key":           resolved.Key,
		"resolution":    "merge",
		"final_version": resolved.Version,
	})

	return resolved
}

func (d *DistributedStateManager) deepMerge(dst, src map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy dst
	for k, v := range dst {
		result[k] = v
	}

	// Merge src
	for k, v := range src {
		if existing, exists := result[k]; exists {
			// Recursive merge for nested maps
			if existingMap, ok := existing.(map[string]interface{}); ok {
				if srcMap, ok := v.(map[string]interface{}); ok {
					result[k] = d.deepMerge(existingMap, srcMap)
					continue
				}
			}
		}
		result[k] = v
	}

	return result
}

func (d *DistributedStateManager) retrySet(ctx context.Context, key string, value interface{}, metadata map[string]interface{}, attempts int) error {
	for i := 0; i < attempts; i++ {
		// Exponential backoff
		time.Sleep(time.Duration(1<<i) * 100 * time.Millisecond)

		if err := d.Set(ctx, key, value, metadata); err == nil {
			return nil
		}
	}

	return errors.New("max retries exceeded")
}

func (d *DistributedStateManager) watchStateChanges(ctx context.Context, namespace string) {
	watchChan := d.client.Watch(ctx, namespace, clientv3.WithPrefix())

	for response := range watchChan {
		for _, event := range response.Events {
			key := string(event.Kv.Key)

			var change StateChange
			change.Key = key

			switch event.Type {
			case clientv3.EventTypePut:
				if event.IsCreate() {
					change.Type = "created"
				} else {
					change.Type = "updated"
				}

				var entry StateEntry
				if err := json.Unmarshal(event.Kv.Value, &entry); err == nil {
					change.NewValue = &entry

					// Update local cache
					d.cacheMu.Lock()
					d.localCache[key] = &entry
					d.cacheMu.Unlock()
				}

			case clientv3.EventTypeDelete:
				change.Type = "deleted"

				// Remove from local cache
				d.cacheMu.Lock()
				delete(d.localCache, key)
				d.cacheMu.Unlock()
			}

			// Notify watchers
			d.notifyWatchers(key, &change)
		}
	}
}

func (d *DistributedStateManager) notifyWatchers(key string, change *StateChange) {
	d.watchersMu.RLock()
	watchers := d.watchers[key]
	d.watchersMu.RUnlock()

	for _, ch := range watchers {
		select {
		case ch <- *change:
		default:
			// Channel full, skip
		}
	}
}

func (d *DistributedStateManager) refreshLock(lock *Lock, ttl time.Duration) {
	ticker := time.NewTicker(ttl / 3)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Refresh session lease
			_, err := lock.session.Client().KeepAliveOnce(context.Background(), lock.session.Lease())
			if err != nil {
				d.logger.Error("Failed to refresh lock", err, map[string]interface{}{
					"resource": lock.key,
				})
				return
			}
		case <-lock.ctx.Done():
			return
		}
	}
}

func (d *DistributedStateManager) getInstanceID() string {
	hostname, _ := os.Hostname()
	return fmt.Sprintf("%s-%d", hostname, os.Getpid())
}

// Close closes the distributed state manager
func (d *DistributedStateManager) Close() error {
	// Close all watchers
	d.watchersMu.Lock()
	for _, watchers := range d.watchers {
		for _, ch := range watchers {
			close(ch)
		}
	}
	d.watchers = make(map[string][]chan StateChange)
	d.watchersMu.Unlock()

	// Close session
	if d.session != nil {
		d.session.Close()
	}

	// Close client
	if d.client != nil {
		return d.client.Close()
	}

	return nil
}
