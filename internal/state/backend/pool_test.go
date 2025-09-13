package backend

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockConnection implements io.Closer for testing
type MockConnection struct {
	id     int
	closed bool
	mu     sync.Mutex
}

func NewMockConnection(id int) *MockConnection {
	return &MockConnection{
		id: id,
	}
}

func (m *MockConnection) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *MockConnection) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func (m *MockConnection) ID() int {
	return m.id
}

// MockConnectionPool implements ConnectionPool interface for testing
type MockConnectionPool struct {
	maxOpen         int
	maxIdle         int
	idleTimeout     time.Duration
	connections     []poolConn
	nextID          int
	stats           PoolStats
	mu              sync.Mutex
	createCount     int
	getCount        int
	putCount        int
	cleanupInterval time.Duration
	closed          bool
}

func NewMockConnectionPool(maxOpen, maxIdle int, idleTimeout time.Duration) *MockConnectionPool {
	pool := &MockConnectionPool{
		maxOpen:         maxOpen,
		maxIdle:         maxIdle,
		idleTimeout:     idleTimeout,
		connections:     make([]poolConn, 0, maxOpen),
		cleanupInterval: time.Second,
		stats: PoolStats{
			MaxOpen:     maxOpen,
			MaxIdle:     maxIdle,
			IdleTimeout: idleTimeout,
		},
	}

	// Start cleanup goroutine
	go pool.cleanupLoop()

	return pool
}

func (p *MockConnectionPool) Get(ctx context.Context) (io.Closer, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, io.ErrClosedPipe
	}

	p.getCount++

	// Try to get an idle connection
	for i, conn := range p.connections {
		if !conn.inUse {
			p.connections[i].inUse = true
			p.connections[i].lastUsed = time.Now()
			p.stats.Active++
			p.stats.Idle--
			return conn.client.(io.Closer), nil
		}
	}

	// Create new connection if under limit
	if len(p.connections) < p.maxOpen {
		p.nextID++
		newConn := NewMockConnection(p.nextID)
		pc := poolConn{
			client:   newConn,
			lastUsed: time.Now(),
			inUse:    true,
		}
		p.connections = append(p.connections, pc)
		p.stats.Active++
		p.stats.Created++
		p.createCount++
		return newConn, nil
	}

	// Wait for available connection or timeout
	p.stats.WaitCount++
	waitStart := time.Now()

	// Simple implementation: return error if no connections available
	p.stats.WaitDuration += time.Since(waitStart)
	return nil, context.DeadlineExceeded
}

func (p *MockConnectionPool) Put(conn io.Closer) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		conn.Close()
		return
	}

	p.putCount++

	// Find the connection and mark as not in use
	for i, pc := range p.connections {
		if pc.client == conn {
			p.connections[i].inUse = false
			p.connections[i].lastUsed = time.Now()
			p.stats.Active--
			p.stats.Idle++
			return
		}
	}

	// If not found, close it
	conn.Close()
}

func (p *MockConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	// Close all connections
	for _, conn := range p.connections {
		conn.client.(io.Closer).Close()
		p.stats.Closed++
	}

	p.connections = nil
	return nil
}

func (p *MockConnectionPool) Stats() *PoolStats {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Return copy of stats
	statsCopy := p.stats
	return &statsCopy
}

func (p *MockConnectionPool) cleanupLoop() {
	ticker := time.NewTicker(p.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.cleanup()
		}

		p.mu.Lock()
		closed := p.closed
		p.mu.Unlock()

		if closed {
			break
		}
	}
}

func (p *MockConnectionPool) cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	now := time.Now()
	var keepConnections []poolConn

	// Keep only non-idle connections or recent connections
	for _, conn := range p.connections {
		if conn.inUse || now.Sub(conn.lastUsed) < p.idleTimeout {
			keepConnections = append(keepConnections, conn)
		} else {
			// Close idle connection
			conn.client.(io.Closer).Close()
			p.stats.Closed++
			if !conn.inUse {
				p.stats.Idle--
			}
		}
	}

	p.connections = keepConnections
}

// GetCreateCount returns number of connections created (for testing)
func (p *MockConnectionPool) GetCreateCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.createCount
}

// GetGetCount returns number of Get calls (for testing)
func (p *MockConnectionPool) GetGetCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.getCount
}

// GetPutCount returns number of Put calls (for testing)
func (p *MockConnectionPool) GetPutCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.putCount
}

// Test Connection Pool Creation
func TestConnectionPool_Creation(t *testing.T) {
	pool := NewMockConnectionPool(10, 5, 30*time.Second)
	defer pool.Close()

	require.NotNil(t, pool)
	assert.Equal(t, 10, pool.maxOpen)
	assert.Equal(t, 5, pool.maxIdle)
	assert.Equal(t, 30*time.Second, pool.idleTimeout)

	stats := pool.Stats()
	assert.Equal(t, 10, stats.MaxOpen)
	assert.Equal(t, 5, stats.MaxIdle)
	assert.Equal(t, 30*time.Second, stats.IdleTimeout)
}

// Test Connection Pool Basic Operations
func TestConnectionPool_BasicOperations(t *testing.T) {
	pool := NewMockConnectionPool(5, 3, 10*time.Second)
	defer pool.Close()

	ctx := context.Background()

	t.Run("Get and Put connection", func(t *testing.T) {
		// Get a connection
		conn, err := pool.Get(ctx)
		require.NoError(t, err)
		require.NotNil(t, conn)

		// Verify it's a mock connection
		mockConn, ok := conn.(*MockConnection)
		require.True(t, ok)
		assert.False(t, mockConn.IsClosed())

		// Check stats
		stats := pool.Stats()
		assert.Equal(t, int64(1), stats.Active)
		assert.Equal(t, int64(0), stats.Idle)
		assert.Equal(t, int64(1), stats.Created)

		// Put connection back
		pool.Put(conn)

		// Check stats after put
		stats = pool.Stats()
		assert.Equal(t, int64(0), stats.Active)
		assert.Equal(t, int64(1), stats.Idle)
	})

	t.Run("Reuse idle connection", func(t *testing.T) {
		// Get a connection
		conn1, err := pool.Get(ctx)
		require.NoError(t, err)

		// Put it back
		pool.Put(conn1)

		// Get another connection (should reuse)
		conn2, err := pool.Get(ctx)
		require.NoError(t, err)

		// Should be the same connection
		assert.Equal(t, conn1, conn2)

		// Should not have created new connection
		assert.Equal(t, 1, pool.GetCreateCount())

		pool.Put(conn2)
	})

	t.Run("Pool limit enforcement", func(t *testing.T) {
		// Get all available connections
		var connections []io.Closer
		for i := 0; i < 5; i++ {
			conn, err := pool.Get(ctx)
			require.NoError(t, err)
			connections = append(connections, conn)
		}

		// Try to get one more (should fail)
		_, err := pool.Get(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)

		// Put connections back
		for _, conn := range connections {
			pool.Put(conn)
		}
	})
}

// Test Connection Pool Concurrency
func TestConnectionPool_Concurrency(t *testing.T) {
	pool := NewMockConnectionPool(10, 5, 5*time.Second)
	defer pool.Close()

	ctx := context.Background()
	var wg sync.WaitGroup
	successCount := int64(0)
	errorCount := int64(0)
	var mu sync.Mutex

	// Launch multiple goroutines to get/put connections
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			conn, err := pool.Get(ctx)
			mu.Lock()
			if err != nil {
				errorCount++
			} else {
				successCount++
			}
			mu.Unlock()

			if err == nil {
				// Simulate some work
				time.Sleep(10 * time.Millisecond)
				pool.Put(conn)
			}
		}(i)
	}

	wg.Wait()

	// Verify some operations succeeded (up to pool limit)
	mu.Lock()
	assert.LessOrEqual(t, successCount, int64(10))
	assert.Equal(t, int64(20), successCount+errorCount)
	mu.Unlock()

	// Check final stats
	stats := pool.Stats()
	assert.GreaterOrEqual(t, stats.WaitCount, int64(10)) // At least 10 requests had to wait
}

// Test Connection Pool Cleanup
func TestConnectionPool_Cleanup(t *testing.T) {
	pool := NewMockConnectionPool(5, 3, 100*time.Millisecond) // Very short idle timeout
	defer pool.Close()

	ctx := context.Background()

	// Get and put several connections
	var connections []io.Closer
	for i := 0; i < 3; i++ {
		conn, err := pool.Get(ctx)
		require.NoError(t, err)
		connections = append(connections, conn)
	}

	for _, conn := range connections {
		pool.Put(conn)
	}

	// Verify all connections are idle
	stats := pool.Stats()
	assert.Equal(t, int64(3), stats.Idle)
	assert.Equal(t, int64(0), stats.Active)

	// Wait for cleanup to happen
	time.Sleep(200 * time.Millisecond)

	// Connections should be cleaned up due to idle timeout
	stats = pool.Stats()
	assert.Equal(t, int64(0), stats.Idle)
	assert.Equal(t, int64(3), stats.Closed)
}

// Test Connection Pool Statistics
func TestConnectionPool_Statistics(t *testing.T) {
	pool := NewMockConnectionPool(3, 2, 10*time.Second)
	defer pool.Close()

	ctx := context.Background()

	t.Run("Initial stats", func(t *testing.T) {
		stats := pool.Stats()
		assert.Equal(t, 3, stats.MaxOpen)
		assert.Equal(t, 2, stats.MaxIdle)
		assert.Equal(t, 10*time.Second, stats.IdleTimeout)
		assert.Equal(t, int64(0), stats.Active)
		assert.Equal(t, int64(0), stats.Idle)
		assert.Equal(t, int64(0), stats.Created)
		assert.Equal(t, int64(0), stats.Closed)
		assert.Equal(t, int64(0), stats.WaitCount)
	})

	t.Run("Stats after operations", func(t *testing.T) {
		// Get connections
		conn1, err := pool.Get(ctx)
		require.NoError(t, err)
		conn2, err := pool.Get(ctx)
		require.NoError(t, err)

		stats := pool.Stats()
		assert.Equal(t, int64(2), stats.Active)
		assert.Equal(t, int64(0), stats.Idle)
		assert.Equal(t, int64(2), stats.Created)

		// Put one back
		pool.Put(conn1)

		stats = pool.Stats()
		assert.Equal(t, int64(1), stats.Active)
		assert.Equal(t, int64(1), stats.Idle)

		// Close the other
		conn2.Close()
		pool.Put(conn2) // Put closed connection

		stats = pool.Stats()
		assert.Equal(t, int64(0), stats.Active)
		assert.Equal(t, int64(1), stats.Idle)
	})
}

// Test S3 Connection Pool (from s3.go)
func TestS3ConnectionPool(t *testing.T) {
	pool := NewS3ConnectionPool(5, 3, 10*time.Minute)

	require.NotNil(t, pool)
	assert.Equal(t, 5, pool.maxOpen)
	assert.Equal(t, 3, pool.maxIdle)
	assert.Equal(t, 10*time.Minute, pool.idleTimeout)

	// Check stats
	assert.Equal(t, 5, pool.stats.MaxOpen)
	assert.Equal(t, 3, pool.stats.MaxIdle)
	assert.Equal(t, 10*time.Minute, pool.stats.IdleTimeout)
	assert.Equal(t, 0, pool.stats.Active)
	assert.Equal(t, 0, pool.stats.Idle)
}

// Test Connection Pool Error Handling
func TestConnectionPool_ErrorHandling(t *testing.T) {
	pool := NewMockConnectionPool(2, 1, 5*time.Second)
	defer pool.Close()

	ctx := context.Background()

	t.Run("Get after close", func(t *testing.T) {
		// Close the pool
		err := pool.Close()
		require.NoError(t, err)

		// Try to get connection from closed pool
		_, err = pool.Get(ctx)
		assert.Error(t, err)
		assert.Equal(t, io.ErrClosedPipe, err)
	})

	t.Run("Put to closed pool", func(t *testing.T) {
		// Create new pool
		newPool := NewMockConnectionPool(2, 1, 5*time.Second)

		// Get connection before closing
		conn, err := newPool.Get(ctx)
		require.NoError(t, err)

		// Close pool
		err = newPool.Close()
		require.NoError(t, err)

		// Put connection back to closed pool (should close connection)
		mockConn := conn.(*MockConnection)
		assert.False(t, mockConn.IsClosed())

		newPool.Put(conn)

		// Connection should be closed
		assert.True(t, mockConn.IsClosed())
	})

	t.Run("Multiple close calls", func(t *testing.T) {
		newPool := NewMockConnectionPool(2, 1, 5*time.Second)

		// First close should succeed
		err := newPool.Close()
		require.NoError(t, err)

		// Second close should not error
		err = newPool.Close()
		require.NoError(t, err)
	})
}

// Benchmark Connection Pool Operations
func BenchmarkConnectionPool_Get(b *testing.B) {
	pool := NewMockConnectionPool(10, 5, 30*time.Second)
	defer pool.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn, err := pool.Get(ctx)
			if err != nil {
				b.Fatal(err)
			}
			pool.Put(conn)
		}
	})
}

func BenchmarkConnectionPool_GetPut(b *testing.B) {
	pool := NewMockConnectionPool(100, 50, 30*time.Second)
	defer pool.Close()

	ctx := context.Background()
	var connections [100]io.Closer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Get batch of connections
		for j := 0; j < 10; j++ {
			conn, err := pool.Get(ctx)
			if err != nil {
				b.Fatal(err)
			}
			connections[j] = conn
		}

		// Put them back
		for j := 0; j < 10; j++ {
			pool.Put(connections[j])
		}
	}
}

func BenchmarkConnectionPool_Contention(b *testing.B) {
	pool := NewMockConnectionPool(5, 3, 30*time.Second)
	defer pool.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn, err := pool.Get(ctx)
			if err != nil {
				continue // Skip contended gets
			}

			// Simulate very brief work
			time.Sleep(time.Microsecond)

			pool.Put(conn)
		}
	})
}

// Test Pool Statistics Accuracy
func TestConnectionPool_StatisticsAccuracy(t *testing.T) {
	pool := NewMockConnectionPool(3, 2, 5*time.Second)
	defer pool.Close()

	ctx := context.Background()

	// Perform various operations and check stats
	connections := make([]io.Closer, 0, 3)

	// Get 3 connections
	for i := 0; i < 3; i++ {
		conn, err := pool.Get(ctx)
		require.NoError(t, err)
		connections = append(connections, conn)
	}

	stats := pool.Stats()
	assert.Equal(t, int64(3), stats.Active)
	assert.Equal(t, int64(0), stats.Idle)
	assert.Equal(t, int64(3), stats.Created)

	// Put 2 back
	for i := 0; i < 2; i++ {
		pool.Put(connections[i])
	}

	stats = pool.Stats()
	assert.Equal(t, int64(1), stats.Active)
	assert.Equal(t, int64(2), stats.Idle)

	// Close one connection manually
	connections[2].Close()
	pool.Put(connections[2])

	stats = pool.Stats()
	assert.Equal(t, int64(0), stats.Active)
	assert.Equal(t, int64(2), stats.Idle)

	// Try to get connection beyond limit to increment wait count
	conn, err := pool.Get(ctx)
	require.NoError(t, err)
	pool.Put(conn)

	conn, err = pool.Get(ctx)
	require.NoError(t, err)
	pool.Put(conn)

	// Now all connections are used, next get should increment wait count
	conn1, err := pool.Get(ctx)
	require.NoError(t, err)
	conn2, err := pool.Get(ctx)
	require.NoError(t, err)

	// This should trigger wait (will fail due to our mock implementation)
	_, err = pool.Get(ctx)
	assert.Error(t, err)

	stats = pool.Stats()
	assert.GreaterOrEqual(t, stats.WaitCount, int64(1))

	// Clean up
	pool.Put(conn1)
	pool.Put(conn2)
}
