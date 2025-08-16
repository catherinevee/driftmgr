package concurrency

import (
	"context"
	"sync"
	"time"
)

// WorkerPool manages a pool of workers
type WorkerPool struct {
	workers int
	tasks   chan func()
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		workers: workers,
		tasks:   make(chan func(), workers*2),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Start workers
	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

// worker runs in a goroutine and processes tasks
func (wp *WorkerPool) worker() {
	defer wp.wg.Done()

	for {
		select {
		case task := <-wp.tasks:
			if task != nil {
				task()
			}
		case <-wp.ctx.Done():
			return
		}
	}
}

// Submit adds a task to the pool
func (wp *WorkerPool) Submit(task func()) error {
	select {
	case wp.tasks <- task:
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	}
}

// Shutdown gracefully shuts down the worker pool
func (wp *WorkerPool) Shutdown(timeout time.Duration) error {
	wp.cancel()

	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return context.DeadlineExceeded
	}
}

// Semaphore provides a counting semaphore
type Semaphore struct {
	sem chan struct{}
}

// NewSemaphore creates a new semaphore with the given capacity
func NewSemaphore(capacity int) *Semaphore {
	return &Semaphore{
		sem: make(chan struct{}, capacity),
	}
}

// Acquire acquires a permit from the semaphore
func (s *Semaphore) Acquire() {
	s.sem <- struct{}{}
}

// Release releases a permit back to the semaphore
func (s *Semaphore) Release() {
	<-s.sem
}

// TryAcquire attempts to acquire a permit without blocking
func (s *Semaphore) TryAcquire() bool {
	select {
	case s.sem <- struct{}{}:
		return true
	default:
		return false
	}
}

// AcquireWithTimeout attempts to acquire a permit with a timeout
func (s *Semaphore) AcquireWithTimeout(timeout time.Duration) bool {
	select {
	case s.sem <- struct{}{}:
		return true
	case <-time.After(timeout):
		return false
	}
}
