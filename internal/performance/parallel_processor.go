package performance

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ParallelProcessor provides parallel processing capabilities with worker pools
type ParallelProcessor struct {
	workerPool   *WorkerPool
	cacheManager *CacheManager
	queueManager *QueueManager
	config       *ProcessingConfig
}

// ProcessingConfig defines processing behavior
type ProcessingConfig struct {
	MaxConcurrency int
	BatchSize      int
	Timeout        time.Duration
	RetryPolicy    *RetryPolicy
	CacheEnabled   bool
	CacheTTL       time.Duration
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries    int
	RetryDelay    time.Duration
	BackoffFactor float64
	MaxDelay      time.Duration
}

// NewParallelProcessor creates a new parallel processor
func NewParallelProcessor(config *ProcessingConfig) *ParallelProcessor {
	return &ParallelProcessor{
		workerPool:   NewWorkerPool(config.MaxConcurrency),
		cacheManager: NewCacheManager(config.CacheTTL),
		queueManager: NewQueueManager(),
		config:       config,
	}
}

// ProcessBatch processes a batch of items in parallel
func (pp *ParallelProcessor) ProcessBatch(
	ctx context.Context,
	items []interface{},
	processor func(context.Context, interface{}) (interface{}, error),
) ([]interface{}, error) {
	if len(items) == 0 {
		return []interface{}{}, nil
	}

	// Check cache first if enabled
	if pp.config.CacheEnabled {
		cachedResults := pp.cacheManager.GetBatch(items)
		if len(cachedResults) == len(items) {
			return cachedResults, nil
		}
	}

	// Create channels for coordination
	resultChan := make(chan interface{}, len(items))
	errorChan := make(chan error, len(items))
	doneChan := make(chan bool)

	// Process items in batches
	batchSize := pp.config.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}

	var wg sync.WaitGroup
	var results []interface{}
	var errors []error

	// Start result collector
	go func() {
		for {
			select {
			case result := <-resultChan:
				results = append(results, result)
			case err := <-errorChan:
				errors = append(errors, err)
			case <-doneChan:
				return
			}
		}
	}()

	// Process batches
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		wg.Add(1)

		go func(batchItems []interface{}) {
			defer wg.Done()
			pp.processBatchWithRetry(ctx, batchItems, processor, resultChan, errorChan)
		}(batch)
	}

	// Wait for all batches to complete
	wg.Wait()
	close(doneChan)

	// Cache results if enabled
	if pp.config.CacheEnabled {
		pp.cacheManager.SetBatch(items, results)
	}

	// Return results and errors
	if len(errors) > 0 {
		return results, fmt.Errorf("batch processing completed with %d errors", len(errors))
	}

	return results, nil
}

// processBatchWithRetry processes a batch with retry logic
func (pp *ParallelProcessor) processBatchWithRetry(
	ctx context.Context,
	items []interface{},
	processor func(context.Context, interface{}) (interface{}, error),
	resultChan chan<- interface{},
	errorChan chan<- error,
) {
	for _, item := range items {
		result, err := pp.processWithRetry(ctx, item, processor)
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- result
		}
	}
}

// processWithRetry processes a single item with retry logic
func (pp *ParallelProcessor) processWithRetry(
	ctx context.Context,
	item interface{},
	processor func(context.Context, interface{}) (interface{}, error),
) (interface{}, error) {
	var lastErr error
	delay := pp.config.RetryPolicy.RetryDelay

	for attempt := 0; attempt <= pp.config.RetryPolicy.MaxRetries; attempt++ {
		// Create timeout context
		timeoutCtx, cancel := context.WithTimeout(ctx, pp.config.Timeout)
		defer cancel()

		result, err := processor(timeoutCtx, item)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't retry on last attempt
		if attempt == pp.config.RetryPolicy.MaxRetries {
			break
		}

		// Wait before retry
		select {
		case <-timeoutCtx.Done():
			return nil, timeoutCtx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}

		// Calculate next delay with backoff
		delay = time.Duration(float64(delay) * pp.config.RetryPolicy.BackoffFactor)
		if delay > pp.config.RetryPolicy.MaxDelay {
			delay = pp.config.RetryPolicy.MaxDelay
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", pp.config.RetryPolicy.MaxRetries+1, lastErr)
}

// WorkerPool manages a pool of workers for parallel processing
type WorkerPool struct {
	workers    int
	jobQueue   chan Job
	workerPool chan chan Job
	quit       chan bool
	wg         sync.WaitGroup
}

// Job represents a job to be processed
type Job struct {
	ID        string
	Data      interface{}
	Processor func(context.Context, interface{}) (interface{}, error)
	Result    chan JobResult
}

// JobResult represents the result of a job
type JobResult struct {
	JobID  string
	Result interface{}
	Error  error
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers int) *WorkerPool {
	if workers <= 0 {
		workers = 10
	}

	pool := &WorkerPool{
		workers:    workers,
		jobQueue:   make(chan Job, workers*2),
		workerPool: make(chan chan Job, workers),
		quit:       make(chan bool),
	}

	pool.start()
	return pool
}

// start starts the worker pool
func (wp *WorkerPool) start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}

	go wp.dispatcher()
}

// dispatcher dispatches jobs to available workers
func (wp *WorkerPool) dispatcher() {
	for {
		select {
		case job := <-wp.jobQueue:
			worker := <-wp.workerPool
			worker <- job
		case <-wp.quit:
			return
		}
	}
}

// worker represents a worker in the pool
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	jobChannel := make(chan Job)

	for {
		wp.workerPool <- jobChannel

		select {
		case job := <-jobChannel:
			// Process the job
			result, err := job.Processor(context.Background(), job.Data)
			job.Result <- JobResult{
				JobID:  job.ID,
				Result: result,
				Error:  err,
			}
		case <-wp.quit:
			return
		}
	}
}

// SubmitJob submits a job to the worker pool
func (wp *WorkerPool) SubmitJob(job Job) {
	wp.jobQueue <- job
}

// Stop stops the worker pool
func (wp *WorkerPool) Stop() {
	close(wp.quit)
	wp.wg.Wait()
}

// CacheManager provides caching capabilities
type CacheManager struct {
	cache map[string]CacheEntry
	ttl   time.Duration
	mu    sync.RWMutex
}

// CacheEntry represents a cache entry
type CacheEntry struct {
	Data      interface{}
	Timestamp time.Time
}

// NewCacheManager creates a new cache manager
func NewCacheManager(ttl time.Duration) *CacheManager {
	cm := &CacheManager{
		cache: make(map[string]CacheEntry),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go cm.cleanup()

	return cm
}

// Get retrieves a value from cache
func (cm *CacheManager) Get(key string) (interface{}, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	entry, exists := cm.cache[key]
	if !exists {
		return nil, false
	}

	// Check if entry has expired
	if time.Since(entry.Timestamp) > cm.ttl {
		delete(cm.cache, key)
		return nil, false
	}

	return entry.Data, true
}

// Set stores a value in cache
func (cm *CacheManager) Set(key string, value interface{}) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.cache[key] = CacheEntry{
		Data:      value,
		Timestamp: time.Now(),
	}
}

// GetBatch retrieves multiple values from cache
func (cm *CacheManager) GetBatch(keys []interface{}) []interface{} {
	results := make([]interface{}, len(keys))
	allFound := true

	for i, key := range keys {
		if result, found := cm.Get(fmt.Sprintf("%v", key)); found {
			results[i] = result
		} else {
			allFound = false
			break
		}
	}

	if allFound {
		return results
	}
	return nil
}

// SetBatch stores multiple values in cache
func (cm *CacheManager) SetBatch(keys []interface{}, values []interface{}) {
	for i, key := range keys {
		cm.Set(fmt.Sprintf("%v", key), values[i])
	}
}

// cleanup removes expired entries from cache
func (cm *CacheManager) cleanup() {
	ticker := time.NewTicker(cm.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		cm.mu.Lock()
		now := time.Now()
		for key, entry := range cm.cache {
			if now.Sub(entry.Timestamp) > cm.ttl {
				delete(cm.cache, key)
			}
		}
		cm.mu.Unlock()
	}
}

// QueueManager provides queue-based processing
type QueueManager struct {
	queues map[string]*Queue
	mu     sync.RWMutex
}

// Queue represents a processing queue
type Queue struct {
	name      string
	jobs      chan QueueJob
	workers   int
	processor func(context.Context, interface{}) error
	quit      chan bool
	wg        sync.WaitGroup
}

// QueueJob represents a job in a queue
type QueueJob struct {
	ID   string
	Data interface{}
}

// NewQueueManager creates a new queue manager
func NewQueueManager() *QueueManager {
	return &QueueManager{
		queues: make(map[string]*Queue),
	}
}

// CreateQueue creates a new processing queue
func (qm *QueueManager) CreateQueue(name string, workers int, processor func(context.Context, interface{}) error) *Queue {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	queue := &Queue{
		name:      name,
		jobs:      make(chan QueueJob, workers*2),
		workers:   workers,
		processor: processor,
		quit:      make(chan bool),
	}

	queue.start()
	qm.queues[name] = queue
	return queue
}

// GetQueue returns a queue by name
func (qm *QueueManager) GetQueue(name string) (*Queue, bool) {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	queue, exists := qm.queues[name]
	return queue, exists
}

// start starts the queue workers
func (q *Queue) start() {
	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}
}

// worker represents a worker in the queue
func (q *Queue) worker(id int) {
	defer q.wg.Done()

	for {
		select {
		case job := <-q.jobs:
			ctx := context.Background()
			err := q.processor(ctx, job.Data)
			if err != nil {
				// Log error but continue processing
				fmt.Printf("Worker %d failed to process job %s: %v\n", id, job.ID, err)
			}
		case <-q.quit:
			return
		}
	}
}

// Enqueue adds a job to the queue
func (q *Queue) Enqueue(job QueueJob) {
	q.jobs <- job
}

// Stop stops the queue
func (q *Queue) Stop() {
	close(q.quit)
	q.wg.Wait()
}

// GetStatistics returns parallel processor statistics
func (pp *ParallelProcessor) GetStatistics() map[string]interface{} {
	return map[string]interface{}{
		"max_concurrency": pp.config.MaxConcurrency,
		"batch_size":      pp.config.BatchSize,
		"timeout":         pp.config.Timeout,
		"cache_enabled":   pp.config.CacheEnabled,
		"cache_ttl":       pp.config.CacheTTL,
	}
}
