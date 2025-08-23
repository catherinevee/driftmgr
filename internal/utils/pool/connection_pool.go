package pool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/catherinevee/driftmgr/internal/logger"
	"github.com/catherinevee/driftmgr/internal/telemetry"
)

var (
	ErrPoolClosed   = errors.New("connection pool is closed")
	ErrPoolExhausted = errors.New("connection pool exhausted")
	ErrInvalidConfig = errors.New("invalid pool configuration")
)

// Connection represents a pooled connection
type Connection interface {
	// IsAlive checks if the connection is still valid
	IsAlive() bool
	// Close closes the connection
	Close() error
	// GetID returns the connection ID
	GetID() string
}

// Factory creates new connections
type Factory func(ctx context.Context) (Connection, error)

// ConnectionPool manages a pool of connections
type ConnectionPool struct {
	// Configuration
	minSize      int
	maxSize      int
	maxIdleTime  time.Duration
	factory      Factory
	
	// State
	connections  chan *pooledConnection
	waiters      chan chan *pooledConnection
	closed       int32
	
	// Metrics
	created      int64
	destroyed    int64
	active       int64
	idle         int64
	waitCount    int64
	
	// Lifecycle
	mu           sync.RWMutex
	wg           sync.WaitGroup
	shutdownCh   chan struct{}
	
	// Logging
	log          logger.Logger
}

// pooledConnection wraps a connection with metadata
type pooledConnection struct {
	conn        Connection
	pool        *ConnectionPool
	createdAt   time.Time
	lastUsedAt  time.Time
	usageCount  int64
	id          string
}

// PoolConfig contains pool configuration
type PoolConfig struct {
	MinSize      int           `json:"min_size"`
	MaxSize      int           `json:"max_size"`
	MaxIdleTime  time.Duration `json:"max_idle_time"`
	Factory      Factory       `json:"-"`
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config PoolConfig) (*ConnectionPool, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	
	pool := &ConnectionPool{
		minSize:      config.MinSize,
		maxSize:      config.MaxSize,
		maxIdleTime:  config.MaxIdleTime,
		factory:      config.Factory,
		connections:  make(chan *pooledConnection, config.MaxSize),
		waiters:      make(chan chan *pooledConnection, config.MaxSize),
		shutdownCh:   make(chan struct{}),
		log:          logger.New("connection_pool"),
	}
	
	// Initialize minimum connections
	ctx := context.Background()
	for i := 0; i < config.MinSize; i++ {
		conn, err := pool.createConnection(ctx)
		if err != nil {
			pool.log.Error("Failed to create initial connection",
				logger.Int("index", i),
				logger.Error(err),
			)
			continue
		}
		pool.connections <- conn
		atomic.AddInt64(&pool.idle, 1)
	}
	
	// Start maintenance routine
	pool.wg.Add(1)
	go pool.maintain()
	
	pool.log.Info("Connection pool initialized",
		logger.Int("min_size", config.MinSize),
		logger.Int("max_size", config.MaxSize),
		logger.Duration("max_idle_time", config.MaxIdleTime),
	)
	
	return pool, nil
}

// Get retrieves a connection from the pool
func (p *ConnectionPool) Get(ctx context.Context) (Connection, error) {
	if atomic.LoadInt32(&p.closed) == 1 {
		return nil, ErrPoolClosed
	}
	
	// Record telemetry
	if telemetry.Get() != nil {
		defer func() {
			telemetry.Get().IncrementActiveConnections(ctx, "pool")
		}()
	}
	
	// Try to get an existing connection
	select {
	case conn := <-p.connections:
		atomic.AddInt64(&p.idle, -1)
		atomic.AddInt64(&p.active, 1)
		
		// Check if connection is still valid
		if !conn.conn.IsAlive() || time.Since(conn.lastUsedAt) > p.maxIdleTime {
			p.destroyConnection(conn)
			return p.Get(ctx) // Recursive call to get another connection
		}
		
		conn.lastUsedAt = time.Now()
		conn.usageCount++
		
		p.log.Debug("Connection retrieved from pool",
			logger.String("conn_id", conn.id),
			logger.Int64("usage_count", conn.usageCount),
		)
		
		return conn.conn, nil
		
	default:
		// No connections available, try to create a new one
		if atomic.LoadInt64(&p.created) < int64(p.maxSize) {
			conn, err := p.createConnection(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to create connection: %w", err)
			}
			
			atomic.AddInt64(&p.active, 1)
			conn.lastUsedAt = time.Now()
			conn.usageCount++
			
			return conn.conn, nil
		}
		
		// Pool is at maximum capacity, wait for a connection
		return p.waitForConnection(ctx)
	}
}

// Put returns a connection to the pool
func (p *ConnectionPool) Put(conn Connection) error {
	if atomic.LoadInt32(&p.closed) == 1 {
		conn.Close()
		return ErrPoolClosed
	}
	
	// Record telemetry
	if telemetry.Get() != nil {
		ctx := context.Background()
		telemetry.Get().DecrementActiveConnections(ctx, "pool")
	}
	
	// Find the pooled connection wrapper
	pc := p.findPooledConnection(conn)
	if pc == nil {
		// Connection not from this pool
		conn.Close()
		return errors.New("connection not from this pool")
	}
	
	atomic.AddInt64(&p.active, -1)
	
	// Check if connection is still valid
	if !conn.IsAlive() {
		p.destroyConnection(pc)
		return nil
	}
	
	// Check if there are waiters
	select {
	case waiter := <-p.waiters:
		pc.lastUsedAt = time.Now()
		waiter <- pc
		return nil
	default:
		// No waiters, return to pool
		atomic.AddInt64(&p.idle, 1)
		pc.lastUsedAt = time.Now()
		
		select {
		case p.connections <- pc:
			p.log.Debug("Connection returned to pool",
				logger.String("conn_id", pc.id),
				logger.Int64("usage_count", pc.usageCount),
			)
			return nil
		default:
			// Pool is full, close the connection
			p.destroyConnection(pc)
			return nil
		}
	}
}

// Close closes the pool and all connections
func (p *ConnectionPool) Close() error {
	if !atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		return ErrPoolClosed
	}
	
	p.log.Info("Closing connection pool")
	
	// Signal shutdown
	close(p.shutdownCh)
	
	// Close all idle connections
	close(p.connections)
	for conn := range p.connections {
		p.destroyConnection(conn)
	}
	
	// Wait for maintenance routine to finish
	p.wg.Wait()
	
	p.log.Info("Connection pool closed",
		logger.Int64("created", atomic.LoadInt64(&p.created)),
		logger.Int64("destroyed", atomic.LoadInt64(&p.destroyed)),
	)
	
	return nil
}

// Stats returns pool statistics
func (p *ConnectionPool) Stats() PoolStats {
	return PoolStats{
		Created:   atomic.LoadInt64(&p.created),
		Destroyed: atomic.LoadInt64(&p.destroyed),
		Active:    atomic.LoadInt64(&p.active),
		Idle:      atomic.LoadInt64(&p.idle),
		WaitCount: atomic.LoadInt64(&p.waitCount),
		MinSize:   p.minSize,
		MaxSize:   p.maxSize,
	}
}

// createConnection creates a new connection
func (p *ConnectionPool) createConnection(ctx context.Context) (*pooledConnection, error) {
	conn, err := p.factory(ctx)
	if err != nil {
		return nil, err
	}
	
	atomic.AddInt64(&p.created, 1)
	
	pc := &pooledConnection{
		conn:       conn,
		pool:       p,
		createdAt:  time.Now(),
		lastUsedAt: time.Now(),
		id:         conn.GetID(),
	}
	
	p.log.Debug("Created new connection",
		logger.String("conn_id", pc.id),
	)
	
	return pc, nil
}

// destroyConnection destroys a connection
func (p *ConnectionPool) destroyConnection(conn *pooledConnection) {
	if err := conn.conn.Close(); err != nil {
		p.log.Error("Failed to close connection",
			logger.String("conn_id", conn.id),
			logger.Error(err),
		)
	}
	
	atomic.AddInt64(&p.destroyed, 1)
	
	p.log.Debug("Destroyed connection",
		logger.String("conn_id", conn.id),
		logger.Duration("lifetime", time.Since(conn.createdAt)),
		logger.Int64("usage_count", conn.usageCount),
	)
}

// waitForConnection waits for an available connection
func (p *ConnectionPool) waitForConnection(ctx context.Context) (Connection, error) {
	atomic.AddInt64(&p.waitCount, 1)
	defer atomic.AddInt64(&p.waitCount, -1)
	
	waiter := make(chan *pooledConnection, 1)
	
	select {
	case p.waiters <- waiter:
		// Successfully registered as waiter
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.shutdownCh:
		return nil, ErrPoolClosed
	}
	
	select {
	case conn := <-waiter:
		atomic.AddInt64(&p.active, 1)
		conn.usageCount++
		return conn.conn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.shutdownCh:
		return nil, ErrPoolClosed
	}
}

// findPooledConnection finds the pooled connection wrapper
func (p *ConnectionPool) findPooledConnection(conn Connection) *pooledConnection {
	// In a real implementation, this would maintain a map
	// For now, create a wrapper
	return &pooledConnection{
		conn:       conn,
		pool:       p,
		lastUsedAt: time.Now(),
	}
}

// maintain performs periodic maintenance on the pool
func (p *ConnectionPool) maintain() {
	defer p.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			p.performMaintenance()
		case <-p.shutdownCh:
			return
		}
	}
}

// performMaintenance performs maintenance tasks
func (p *ConnectionPool) performMaintenance() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Remove idle connections that have exceeded max idle time
	var validConnections []*pooledConnection
	
	for {
		select {
		case conn := <-p.connections:
			if time.Since(conn.lastUsedAt) > p.maxIdleTime || !conn.conn.IsAlive() {
				p.destroyConnection(conn)
				atomic.AddInt64(&p.idle, -1)
			} else {
				validConnections = append(validConnections, conn)
			}
		default:
			// No more connections to check
			goto done
		}
	}
	
done:
	// Return valid connections to the pool
	for _, conn := range validConnections {
		select {
		case p.connections <- conn:
		default:
			// Pool is full, destroy excess connection
			p.destroyConnection(conn)
			atomic.AddInt64(&p.idle, -1)
		}
	}
	
	// Ensure minimum connections
	currentSize := atomic.LoadInt64(&p.created) - atomic.LoadInt64(&p.destroyed)
	if currentSize < int64(p.minSize) {
		ctx := context.Background()
		for i := currentSize; i < int64(p.minSize); i++ {
			conn, err := p.createConnection(ctx)
			if err != nil {
				p.log.Error("Failed to create connection during maintenance",
					logger.Error(err),
				)
				break
			}
			select {
			case p.connections <- conn:
				atomic.AddInt64(&p.idle, 1)
			default:
				p.destroyConnection(conn)
			}
		}
	}
	
	p.log.Debug("Pool maintenance completed",
		logger.Int64("active", atomic.LoadInt64(&p.active)),
		logger.Int64("idle", atomic.LoadInt64(&p.idle)),
		logger.Int64("total", currentSize),
	)
}

// validateConfig validates pool configuration
func validateConfig(config PoolConfig) error {
	if config.MinSize < 0 {
		return fmt.Errorf("%w: min_size must be >= 0", ErrInvalidConfig)
	}
	if config.MaxSize <= 0 {
		return fmt.Errorf("%w: max_size must be > 0", ErrInvalidConfig)
	}
	if config.MinSize > config.MaxSize {
		return fmt.Errorf("%w: min_size must be <= max_size", ErrInvalidConfig)
	}
	if config.Factory == nil {
		return fmt.Errorf("%w: factory is required", ErrInvalidConfig)
	}
	if config.MaxIdleTime <= 0 {
		config.MaxIdleTime = 5 * time.Minute
	}
	return nil
}

// PoolStats contains pool statistics
type PoolStats struct {
	Created   int64 `json:"created"`
	Destroyed int64 `json:"destroyed"`
	Active    int64 `json:"active"`
	Idle      int64 `json:"idle"`
	WaitCount int64 `json:"wait_count"`
	MinSize   int   `json:"min_size"`
	MaxSize   int   `json:"max_size"`
}