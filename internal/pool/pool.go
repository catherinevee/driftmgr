package pool

import (
	"context"
	"sync"
)

// ConnectionPool manages a pool of connections
type ConnectionPool struct {
	connections chan interface{}
	factory     func() (interface{}, error)
	close       func(interface{}) error
	maxSize     int
	mu          sync.RWMutex
	closed      bool
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(factory func() (interface{}, error), close func(interface{}) error, maxSize int) *ConnectionPool {
	return &ConnectionPool{
		connections: make(chan interface{}, maxSize),
		factory:     factory,
		close:       close,
		maxSize:     maxSize,
	}
}

// Get retrieves a connection from the pool
func (cp *ConnectionPool) Get(ctx context.Context) (interface{}, error) {
	select {
	case conn := <-cp.connections:
		if conn != nil {
			return conn, nil
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Create new connection if pool is empty
	return cp.factory()
}

// Put returns a connection to the pool
func (cp *ConnectionPool) Put(conn interface{}) error {
	cp.mu.RLock()
	if cp.closed {
		cp.mu.RUnlock()
		return cp.close(conn)
	}
	cp.mu.RUnlock()

	select {
	case cp.connections <- conn:
		return nil
	default:
		// Pool is full, close the connection
		return cp.close(conn)
	}
}

// Close closes all connections in the pool
func (cp *ConnectionPool) Close() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.closed {
		return nil
	}

	cp.closed = true
	close(cp.connections)

	var lastErr error
	for conn := range cp.connections {
		if err := cp.close(conn); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// Size returns the current number of connections in the pool
func (cp *ConnectionPool) Size() int {
	return len(cp.connections)
}

// ObjectPool manages a pool of reusable objects
type ObjectPool struct {
	objects chan interface{}
	factory func() interface{}
	reset   func(interface{})
	maxSize int
	mu      sync.RWMutex
	closed  bool
}

// NewObjectPool creates a new object pool
func NewObjectPool(factory func() interface{}, reset func(interface{}), maxSize int) *ObjectPool {
	return &ObjectPool{
		objects: make(chan interface{}, maxSize),
		factory: factory,
		reset:   reset,
		maxSize: maxSize,
	}
}

// Get retrieves an object from the pool
func (op *ObjectPool) Get() interface{} {
	select {
	case obj := <-op.objects:
		if obj != nil {
			return obj
		}
	default:
	}

	// Create new object if pool is empty
	return op.factory()
}

// Put returns an object to the pool
func (op *ObjectPool) Put(obj interface{}) {
	if obj == nil {
		return
	}

	op.mu.RLock()
	if op.closed {
		op.mu.RUnlock()
		return
	}
	op.mu.RUnlock()

	// Reset the object before returning it to the pool
	if op.reset != nil {
		op.reset(obj)
	}

	select {
	case op.objects <- obj:
	default:
		// Pool is full, discard the object
	}
}

// Close closes the object pool
func (op *ObjectPool) Close() {
	op.mu.Lock()
	defer op.mu.Unlock()

	if op.closed {
		return
	}

	op.closed = true
	close(op.objects)

	// Clear all objects
	for range op.objects {
		// Objects are discarded
	}
}

// Size returns the current number of objects in the pool
func (op *ObjectPool) Size() int {
	return len(op.objects)
}
