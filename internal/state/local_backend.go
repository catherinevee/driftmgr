package state

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LocalBackend implements Backend interface for local file storage
type LocalBackend struct {
	basePath string
	mu       sync.RWMutex
	locks    map[string]bool
}

// NewLocalBackend creates a new local file backend
func NewLocalBackend(basePath string) Backend {
	os.MkdirAll(basePath, 0755)
	return &LocalBackend{
		basePath: basePath,
		locks:    make(map[string]bool),
	}
}

// Get retrieves data for a key
func (lb *LocalBackend) Get(ctx context.Context, key string) ([]byte, error) {
	path := filepath.Join(lb.basePath, key)
	data, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return data, err
}

// Put stores data for a key
func (lb *LocalBackend) Put(ctx context.Context, key string, data []byte) error {
	path := filepath.Join(lb.basePath, key)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
}

// Delete removes data for a key
func (lb *LocalBackend) Delete(ctx context.Context, key string) error {
	path := filepath.Join(lb.basePath, key)
	return os.Remove(path)
}

// List returns keys with a given prefix
func (lb *LocalBackend) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	
	err := filepath.Walk(lb.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(lb.basePath, path)
			if strings.HasPrefix(relPath, prefix) {
				keys = append(keys, relPath)
			}
		}
		return nil
	})
	
	return keys, err
}

// Lock acquires a lock for a key
func (lb *LocalBackend) Lock(ctx context.Context, key string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	
	if lb.locks[key] {
		return fmt.Errorf("key already locked: %s", key)
	}
	lb.locks[key] = true
	return nil
}

// Unlock releases a lock for a key
func (lb *LocalBackend) Unlock(ctx context.Context, key string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	
	delete(lb.locks, key)
	return nil
}

// ListStates returns all state keys
func (lb *LocalBackend) ListStates(ctx context.Context) ([]string, error) {
	return lb.List(ctx, "")
}

// ListStateVersions returns versions of a state
func (lb *LocalBackend) ListStateVersions(ctx context.Context, key string) ([]StateVersion, error) {
	// Simple implementation - just return current version
	return []StateVersion{
		{
			Version:   1,
			Timestamp: time.Now(),
		},
	}, nil
}

// GetStateVersion retrieves a specific version of a state
func (lb *LocalBackend) GetStateVersion(ctx context.Context, key string, version int) ([]byte, error) {
	// Simple implementation - just return current state
	return lb.Get(ctx, key)
}