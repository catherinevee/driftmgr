package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ParallelProcessor provides a parallel execution framework with work stealing
type ParallelProcessor struct {
	workers     []*Worker
	workQueue   chan WorkItem
	resultQueue chan WorkResult
	scheduler   *WorkScheduler
	stealer     *WorkStealer
	batcher     *BatchProcessor
	config      *ProcessorConfig
	metrics     *ProcessorMetrics

	// State management
	mu       sync.RWMutex
	running  bool
	shutdown chan struct{}
	wg       sync.WaitGroup

	// Statistics
	stats ProcessorStats
}

// ProcessorConfig holds configuration for the parallel processor
type ProcessorConfig struct {
	WorkerCount       int
	QueueSize         int
	ResultBufferSize  int
	BatchSize         int
	BatchTimeout      time.Duration
	StealThreshold    int
	WorkerIdleTimeout time.Duration
	MaxRetries        int
	EnableProfiling   bool
}

// ProcessorMetrics holds Prometheus metrics
type ProcessorMetrics struct {
	tasksQueued         prometheus.Counter
	tasksProcessed      prometheus.Counter
	tasksCompleted      prometheus.Counter
	tasksFailed         prometheus.Counter
	tasksRetried        prometheus.Counter
	batchesProcessed    prometheus.Counter
	workSteals          prometheus.Counter
	workerUtilization   prometheus.GaugeVec
	queueDepth          prometheus.Gauge
	processingTime      prometheus.Histogram
	batchProcessingTime prometheus.Histogram
	stealLatency        prometheus.Histogram
}

// Worker represents a worker goroutine
type Worker struct {
	id         int
	processor  *ParallelProcessor
	localQueue chan WorkItem
	busy       int32
	processed  int64
	failed     int64
	steals     int64
	lastActive time.Time

	// Work stealing
	stealTarget *Worker
	stealCount  int32
}

// Task represents a function to be executed
type Task func(context.Context) (interface{}, error)

// WorkItem represents a unit of work
type WorkItem struct {
	ID          string
	Task        Task
	Priority    int
	Deadline    time.Time
	Retries     int
	MaxRetries  int
	Context     context.Context
	Metadata    map[string]interface{}
	SubmittedAt time.Time
	StartedAt   time.Time
}

// WorkResult represents the result of processing a work item
type WorkResult struct {
	WorkItem    *WorkItem
	Result      interface{}
	Error       error
	Duration    time.Duration
	WorkerID    int
	CompletedAt time.Time
	Retries     int
}

// WorkScheduler manages work distribution and prioritization
type WorkScheduler struct {
	priorityQueues map[int]chan WorkItem
	defaultQueue   chan WorkItem
	mu             sync.RWMutex
	metrics        *SchedulerMetrics
}

// SchedulerMetrics holds scheduler metrics
type SchedulerMetrics struct {
	queuedByPriority     *prometheus.CounterVec
	schedulingLatency    prometheus.Histogram
	queueDepthByPriority *prometheus.GaugeVec
}

// WorkStealer implements work stealing algorithm
type WorkStealer struct {
	processor     *ParallelProcessor
	stealInterval time.Duration
	enabled       bool
	mu            sync.RWMutex
	metrics       *StealerMetrics
}

// StealerMetrics holds work stealer metrics
type StealerMetrics struct {
	stealAttempts    prometheus.Counter
	successfulSteals prometheus.Counter
	failedSteals     prometheus.Counter
	stealLatency     prometheus.Histogram
}

// BatchProcessor handles batch processing optimizations
type BatchProcessor struct {
	batchSize int
	timeout   time.Duration
	pending   []WorkItem
	lastFlush time.Time
	mu        sync.Mutex
	metrics   *BatchMetrics
}

// BatchMetrics holds batch processing metrics
type BatchMetrics struct {
	batchesCreated prometheus.Counter
	batchSizeHist  prometheus.Histogram
	batchLatency   prometheus.Histogram
	itemsPerBatch  prometheus.Histogram
}

// ProcessorStats holds processor statistics
type ProcessorStats struct {
	TotalQueued       int64
	TotalProcessed    int64
	TotalCompleted    int64
	TotalFailed       int64
	TotalRetried      int64
	AvgProcessingTime time.Duration
	WorkerUtilization float64
	QueueDepth        int
}

// NewParallelProcessor creates a new parallel processor
func NewParallelProcessor(config *ProcessorConfig) *ParallelProcessor {
	if config.WorkerCount <= 0 {
		config.WorkerCount = runtime.NumCPU()
	}

	metrics := &ProcessorMetrics{
		tasksQueued: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_tasks_queued_total",
			Help: "Total number of tasks queued",
		}),
		tasksProcessed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_tasks_processed_total",
			Help: "Total number of tasks processed",
		}),
		tasksCompleted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_tasks_completed_total",
			Help: "Total number of tasks completed successfully",
		}),
		tasksFailed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_tasks_failed_total",
			Help: "Total number of tasks that failed",
		}),
		tasksRetried: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_tasks_retried_total",
			Help: "Total number of task retries",
		}),
		batchesProcessed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_batches_processed_total",
			Help: "Total number of batches processed",
		}),
		workSteals: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_work_steals_total",
			Help: "Total number of work steals",
		}),
		workerUtilization: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "driftmgr_worker_utilization",
			Help: "Worker utilization percentage",
		}, []string{"worker_id"}),
		queueDepth: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "driftmgr_queue_depth",
			Help: "Current queue depth",
		}),
		processingTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_processing_time_seconds",
			Help:    "Task processing time",
			Buckets: prometheus.DefBuckets,
		}),
		batchProcessingTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_batch_processing_time_seconds",
			Help:    "Batch processing time",
			Buckets: prometheus.DefBuckets,
		}),
		stealLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_steal_latency_seconds",
			Help:    "Work steal latency",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1},
		}),
	}

	schedulerMetrics := &SchedulerMetrics{
		queuedByPriority: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "driftmgr_queued_by_priority_total",
			Help: "Total tasks queued by priority",
		}, []string{"priority"}),
		schedulingLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_scheduling_latency_seconds",
			Help:    "Task scheduling latency",
			Buckets: prometheus.DefBuckets,
		}),
		queueDepthByPriority: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "driftmgr_queue_depth_by_priority",
			Help: "Queue depth by priority",
		}, []string{"priority"}),
	}

	stealerMetrics := &StealerMetrics{
		stealAttempts: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_steal_attempts_total",
			Help: "Total steal attempts",
		}),
		successfulSteals: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_successful_steals_total",
			Help: "Total successful steals",
		}),
		failedSteals: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_failed_steals_total",
			Help: "Total failed steals",
		}),
		stealLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_steal_operation_latency_seconds",
			Help:    "Steal operation latency",
			Buckets: prometheus.DefBuckets,
		}),
	}

	batchMetrics := &BatchMetrics{
		batchesCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_batches_created_total",
			Help: "Total batches created",
		}),
		batchSizeHist: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_batch_size",
			Help:    "Batch size distribution",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		}),
		batchLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_batch_latency_seconds",
			Help:    "Batch creation latency",
			Buckets: prometheus.DefBuckets,
		}),
		itemsPerBatch: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_items_per_batch",
			Help:    "Number of items per batch",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		}),
	}

	processor := &ParallelProcessor{
		workQueue:   make(chan WorkItem, config.QueueSize),
		resultQueue: make(chan WorkResult, config.ResultBufferSize),
		config:      config,
		metrics:     metrics,
		shutdown:    make(chan struct{}),
	}

	// Initialize scheduler
	processor.scheduler = &WorkScheduler{
		priorityQueues: make(map[int]chan WorkItem),
		defaultQueue:   processor.workQueue,
		metrics:        schedulerMetrics,
	}

	// Initialize work stealer
	processor.stealer = &WorkStealer{
		processor:     processor,
		stealInterval: time.Millisecond * 100,
		enabled:       true,
		metrics:       stealerMetrics,
	}

	// Initialize batch processor
	processor.batcher = &BatchProcessor{
		batchSize: config.BatchSize,
		timeout:   config.BatchTimeout,
		metrics:   batchMetrics,
	}

	// Create workers
	processor.workers = make([]*Worker, config.WorkerCount)
	for i := 0; i < config.WorkerCount; i++ {
		processor.workers[i] = &Worker{
			id:         i,
			processor:  processor,
			localQueue: make(chan WorkItem, 100),
			lastActive: time.Now(),
		}
	}

	return processor
}

// Start starts the parallel processor
func (pp *ParallelProcessor) Start() error {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	if pp.running {
		return fmt.Errorf("processor is already running")
	}

	pp.running = true

	// Start workers
	for _, worker := range pp.workers {
		pp.wg.Add(1)
		go worker.run()
	}

	// Start work stealer
	if pp.stealer.enabled {
		pp.wg.Add(1)
		go pp.stealer.run()
	}

	// Start batch processor
	pp.wg.Add(1)
	go pp.batcher.run(pp.shutdown)

	// Start metrics updater
	pp.wg.Add(1)
	go pp.updateMetrics()

	return nil
}

// Submit submits a work item for processing
func (pp *ParallelProcessor) Submit(ctx context.Context, task Task) (<-chan WorkResult, error) {
	if !pp.running {
		return nil, fmt.Errorf("processor is not running")
	}

	workItem := &WorkItem{
		ID:          generateWorkID(),
		Task:        task,
		Priority:    task.Priority(),
		Context:     ctx,
		MaxRetries:  pp.config.MaxRetries,
		SubmittedAt: time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Create result channel
	resultChan := make(chan WorkResult, 1)

	// Store result channel in metadata
	workItem.Metadata["resultChan"] = resultChan

	// Submit to scheduler
	if err := pp.scheduler.Schedule(workItem); err != nil {
		close(resultChan)
		return nil, err
	}

	pp.metrics.tasksQueued.Inc()
	atomic.AddInt64(&pp.stats.TotalQueued, 1)

	return resultChan, nil
}

// SubmitBatch submits multiple work items as a batch
func (pp *ParallelProcessor) SubmitBatch(ctx context.Context, tasks []Task) (<-chan []WorkResult, error) {
	if !pp.running {
		return nil, fmt.Errorf("processor is not running")
	}

	start := time.Now()
	resultChan := make(chan []WorkResult, 1)
	results := make([]WorkResult, len(tasks))
	var wg sync.WaitGroup

	wg.Add(len(tasks))

	for i, task := range tasks {
		go func(index int, t Task) {
			defer wg.Done()

			taskResultChan, err := pp.Submit(ctx, t)
			if err != nil {
				results[index] = WorkResult{
					Error:       err,
					CompletedAt: time.Now(),
				}
				return
			}

			select {
			case result := <-taskResultChan:
				results[index] = result
			case <-ctx.Done():
				results[index] = WorkResult{
					Error:       ctx.Err(),
					CompletedAt: time.Now(),
				}
			}
		}(i, task)
	}

	go func() {
		wg.Wait()
		resultChan <- results
		close(resultChan)

		pp.metrics.batchProcessingTime.Observe(time.Since(start).Seconds())
		pp.metrics.batchesProcessed.Inc()
		pp.batcher.metrics.batchesCreated.Inc()
		pp.batcher.metrics.itemsPerBatch.Observe(float64(len(tasks)))
	}()

	return resultChan, nil
}

// Worker run loop
func (w *Worker) run() {
	defer w.processor.wg.Done()

	for {
		select {
		case <-w.processor.shutdown:
			return
		case workItem := <-w.localQueue:
			w.processWorkItem(&workItem)
		case workItem := <-w.processor.workQueue:
			w.processWorkItem(&workItem)
		default:
			// Try to steal work if idle
			if w.tryStealWork() {
				continue
			}

			// Short sleep to prevent busy waiting
			time.Sleep(time.Millisecond)
		}
	}
}

// processWorkItem processes a single work item
func (w *Worker) processWorkItem(workItem *WorkItem) {
	atomic.StoreInt32(&w.busy, 1)
	defer atomic.StoreInt32(&w.busy, 0)

	w.lastActive = time.Now()
	workItem.StartedAt = time.Now()

	start := time.Now()
	result := WorkResult{
		WorkItem:    workItem,
		WorkerID:    w.id,
		CompletedAt: time.Now(),
	}

	// Execute task
	err := workItem.Task.Execute(workItem.Context)
	duration := time.Since(start)

	result.Error = err
	result.Duration = duration
	result.CompletedAt = time.Now()
	result.Retries = workItem.Retries

	// Update metrics
	w.processor.metrics.tasksProcessed.Inc()
	w.processor.metrics.processingTime.Observe(duration.Seconds())
	atomic.AddInt64(&w.processed, 1)

	if err != nil {
		atomic.AddInt64(&w.failed, 1)
		w.processor.metrics.tasksFailed.Inc()

		// Retry if possible
		if workItem.Retries < workItem.MaxRetries {
			workItem.Retries++
			w.processor.metrics.tasksRetried.Inc()

			// Resubmit after delay
			go func() {
				time.Sleep(time.Duration(workItem.Retries) * time.Second)
				select {
				case w.processor.workQueue <- *workItem:
				case <-w.processor.shutdown:
				}
			}()
			return
		}
	} else {
		w.processor.metrics.tasksCompleted.Inc()
		atomic.AddInt64(&w.processor.stats.TotalCompleted, 1)
	}

	// Send result
	if resultChan, ok := workItem.Metadata["resultChan"].(chan WorkResult); ok {
		select {
		case resultChan <- result:
		case <-workItem.Context.Done():
		}
		close(resultChan)
	}
}

// tryStealWork attempts to steal work from other workers
func (w *Worker) tryStealWork() bool {
	if !w.processor.stealer.enabled {
		return false
	}

	// Find a busy worker to steal from
	for _, otherWorker := range w.processor.workers {
		if otherWorker.id == w.id {
			continue
		}

		if atomic.LoadInt32(&otherWorker.busy) == 1 && len(otherWorker.localQueue) > 0 {
			select {
			case workItem := <-otherWorker.localQueue:
				atomic.AddInt64(&w.steals, 1)
				w.processor.metrics.workSteals.Inc()
				w.processWorkItem(&workItem)
				return true
			default:
				// Queue empty or contended
			}
		}
	}

	return false
}

// Schedule schedules a work item
func (ws *WorkScheduler) Schedule(workItem *WorkItem) error {
	start := time.Now()
	defer func() {
		ws.metrics.schedulingLatency.Observe(time.Since(start).Seconds())
	}()

	ws.mu.RLock()
	defer ws.mu.RUnlock()

	// Route by priority
	if priorityQueue, exists := ws.priorityQueues[workItem.Priority]; exists {
		select {
		case priorityQueue <- *workItem:
			ws.metrics.queuedByPriority.WithLabelValues(fmt.Sprintf("%d", workItem.Priority)).Inc()
			return nil
		default:
			return fmt.Errorf("priority queue %d is full", workItem.Priority)
		}
	}

	// Use default queue
	select {
	case ws.defaultQueue <- *workItem:
		ws.metrics.queuedByPriority.WithLabelValues("default").Inc()
		return nil
	default:
		return fmt.Errorf("work queue is full")
	}
}

// Work stealer run loop
func (ws *WorkStealer) run() {
	defer ws.processor.wg.Done()

	ticker := time.NewTicker(ws.stealInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ws.processor.shutdown:
			return
		case <-ticker.C:
			ws.attemptStealing()
		}
	}
}

// attemptStealing attempts to balance work across workers
func (ws *WorkStealer) attemptStealing() {
	start := time.Now()
	defer func() {
		ws.metrics.stealLatency.Observe(time.Since(start).Seconds())
	}()

	ws.metrics.stealAttempts.Inc()

	// Find overloaded and underloaded workers
	var overloaded, underloaded []*Worker

	for _, worker := range ws.processor.workers {
		queueLen := len(worker.localQueue)

		if queueLen > ws.processor.config.StealThreshold {
			overloaded = append(overloaded, worker)
		} else if queueLen == 0 && atomic.LoadInt32(&worker.busy) == 0 {
			underloaded = append(underloaded, worker)
		}
	}

	// Perform stealing
	stolen := false
	for i := 0; i < len(overloaded) && i < len(underloaded); i++ {
		source := overloaded[i]
		target := underloaded[i]

		// Try to steal half the work
		stealCount := len(source.localQueue) / 2
		if stealCount == 0 {
			stealCount = 1
		}

		for j := 0; j < stealCount; j++ {
			select {
			case workItem := <-source.localQueue:
				select {
				case target.localQueue <- workItem:
					stolen = true
				default:
					// Target queue full, put back
					select {
					case source.localQueue <- workItem:
					default:
						// Lost work item - should not happen in production
					}
				}
			default:
				break
			}
		}
	}

	if stolen {
		ws.metrics.successfulSteals.Inc()
	} else {
		ws.metrics.failedSteals.Inc()
	}
}

// Batch processor run loop
func (bp *BatchProcessor) run(shutdown <-chan struct{}) {
	ticker := time.NewTicker(bp.timeout)
	defer ticker.Stop()

	for {
		select {
		case <-shutdown:
			bp.flushPending()
			return
		case <-ticker.C:
			bp.checkFlush()
		}
	}
}

// checkFlush checks if pending batch should be flushed
func (bp *BatchProcessor) checkFlush() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if len(bp.pending) > 0 && time.Since(bp.lastFlush) > bp.timeout {
		bp.flushPendingLocked()
	}
}

// flushPending flushes all pending work items
func (bp *BatchProcessor) flushPending() {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.flushPendingLocked()
}

// flushPendingLocked flushes pending work items (must hold lock)
func (bp *BatchProcessor) flushPendingLocked() {
	if len(bp.pending) == 0 {
		return
	}

	start := time.Now()

	// Create batch work item
	batch := BatchWorkItem{
		Items:     bp.pending,
		CreatedAt: time.Now(),
	}

	// Submit batch for processing
	// This would integrate with the main work queue

	// Clear pending
	bp.pending = bp.pending[:0]
	bp.lastFlush = time.Now()

	// Update metrics
	bp.metrics.batchLatency.Observe(time.Since(start).Seconds())
	bp.metrics.batchSizeHist.Observe(float64(len(batch.Items)))
}

// BatchWorkItem represents a batch of work items
type BatchWorkItem struct {
	Items     []WorkItem
	CreatedAt time.Time
}

// updateMetrics updates processor metrics
func (pp *ParallelProcessor) updateMetrics() {
	defer pp.wg.Done()

	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-pp.shutdown:
			return
		case <-ticker.C:
			pp.updateWorkerMetrics()
			pp.updateQueueMetrics()
		}
	}
}

// updateWorkerMetrics updates worker utilization metrics
func (pp *ParallelProcessor) updateWorkerMetrics() {
	for _, worker := range pp.workers {
		utilization := float64(0)
		if atomic.LoadInt32(&worker.busy) == 1 {
			utilization = 100.0
		}

		pp.metrics.workerUtilization.WithLabelValues(fmt.Sprintf("%d", worker.id)).Set(utilization)
	}
}

// updateQueueMetrics updates queue depth metrics
func (pp *ParallelProcessor) updateQueueMetrics() {
	queueDepth := len(pp.workQueue)
	pp.metrics.queueDepth.Set(float64(queueDepth))
	pp.stats.QueueDepth = queueDepth
}

// GetStats returns processor statistics
func (pp *ParallelProcessor) GetStats() ProcessorStats {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	stats := pp.stats

	// Calculate worker utilization
	busyWorkers := 0
	for _, worker := range pp.workers {
		if atomic.LoadInt32(&worker.busy) == 1 {
			busyWorkers++
		}
	}
	stats.WorkerUtilization = float64(busyWorkers) / float64(len(pp.workers)) * 100.0

	return stats
}

// Stop stops the parallel processor
func (pp *ParallelProcessor) Stop(ctx context.Context) error {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	if !pp.running {
		return nil
	}

	close(pp.shutdown)
	pp.running = false

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		pp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout: %w", ctx.Err())
	}
}

// ProcessBatch processes a batch of items in parallel
func (pp *ParallelProcessor) ProcessBatch(ctx context.Context, items []interface{}, processor func(context.Context, interface{}) (interface{}, error)) ([]interface{}, error) {
	if !pp.running {
		if err := pp.Start(); err != nil {
			return nil, fmt.Errorf("failed to start processor: %w", err)
		}
		defer pp.Stop(ctx)
	}

	results := make([]interface{}, len(items))
	resultChans := make([]chan WorkResult, len(items))
	var wg sync.WaitGroup

	// Submit all work items
	for i, item := range items {
		resultChan := make(chan WorkResult, 1)
		resultChans[i] = resultChan
		wg.Add(1)

		workItem := WorkItem{
			ID:         generateWorkID(),
			Task:       func(ctx context.Context) (interface{}, error) { return processor(ctx, item) },
			Priority:   0,
			Context:    ctx,
			Metadata:   map[string]interface{}{"resultChan": resultChan, "index": i},
			SubmittedAt: time.Now(),
			MaxRetries: pp.config.MaxRetries,
		}

		select {
		case pp.workQueue <- workItem:
			pp.metrics.tasksQueued.Inc()
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-pp.shutdown:
			return nil, fmt.Errorf("processor shutting down")
		}
	}

	// Collect results
	go func() {
		for i, ch := range resultChans {
			go func(idx int, resultChan chan WorkResult) {
				defer wg.Done()
				select {
				case result := <-resultChan:
					if result.Error != nil {
						// Store error in results
						results[idx] = fmt.Errorf("item %d failed: %w", idx, result.Error)
					} else {
						results[idx] = result.Result
					}
				case <-ctx.Done():
					results[idx] = fmt.Errorf("item %d cancelled: %w", idx, ctx.Err())
				}
			}(i, ch)
		}
	}()

	// Wait for all items to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		pp.metrics.batchesProcessed.Inc()
		return results, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Utility functions
func generateWorkID() string {
	return fmt.Sprintf("work_%d", time.Now().UnixNano())
}
