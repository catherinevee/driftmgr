package performance

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewParallelProcessor(t *testing.T) {
	config := &ProcessorConfig{
		WorkerCount:       4,
		QueueSize:         100,
		BatchSize:         10,
		MaxRetries:        3,
		BatchTimeout:      5 * time.Second,
		StealThreshold:    5,
		WorkerIdleTimeout: 10 * time.Second,
	}

	processor := NewParallelProcessor(config)

	assert.NotNil(t, processor)
	assert.Equal(t, config, processor.config)
	assert.NotNil(t, processor.workers)
	assert.NotNil(t, processor.workQueue)
	assert.NotNil(t, processor.resultQueue)
	assert.NotNil(t, processor.metrics)
	assert.False(t, processor.running)
}

func TestProcessorStartStop(t *testing.T) {
	processor := NewParallelProcessor(&ProcessorConfig{
		WorkerCount: 2,
		QueueSize:   10,
	})

	// Start processor
	err := processor.Start()
	require.NoError(t, err)
	assert.True(t, processor.running)

	// Try to start again - should return error
	err = processor.Start()
	assert.Error(t, err)

	// Stop processor
	ctx := context.Background()
	err = processor.Stop(ctx)
	require.NoError(t, err)
	assert.False(t, processor.running)
}

func TestProcessBatch(t *testing.T) {
	processor := NewParallelProcessor(&ProcessorConfig{
		WorkerCount: 4,
		QueueSize:   100,
		MaxRetries:  2,
	})

	// Test data
	items := []interface{}{1, 2, 3, 4, 5}

	// Processor function that doubles the input
	processorFunc := func(ctx context.Context, item interface{}) (interface{}, error) {
		num, ok := item.(int)
		if !ok {
			return nil, errors.New("invalid type")
		}
		return num * 2, nil
	}

	ctx := context.Background()
	results, err := processor.ProcessBatch(ctx, items, processorFunc)

	require.NoError(t, err)
	assert.Len(t, results, len(items))

	// Verify results
	for i, result := range results {
		expected := (i + 1) * 2
		assert.Equal(t, expected, result)
	}
}

func TestProcessBatchWithErrors(t *testing.T) {
	processor := NewParallelProcessor(&ProcessorConfig{
		WorkerCount: 2,
		QueueSize:   10,
		MaxRetries:  1,
	})

	items := []interface{}{"valid", "error", "valid2"}

	processorFunc := func(ctx context.Context, item interface{}) (interface{}, error) {
		str, ok := item.(string)
		if !ok {
			return nil, errors.New("invalid type")
		}
		if str == "error" {
			return nil, errors.New("processing error")
		}
		return str + "_processed", nil
	}

	ctx := context.Background()
	results, err := processor.ProcessBatch(ctx, items, processorFunc)

	require.NoError(t, err)
	assert.Len(t, results, 3)

	// Check first result
	assert.Equal(t, "valid_processed", results[0])

	// Check error result
	errResult, isError := results[1].(error)
	assert.True(t, isError)
	assert.Contains(t, errResult.Error(), "processing error")

	// Check third result
	assert.Equal(t, "valid2_processed", results[2])
}

func TestProcessBatchCancellation(t *testing.T) {
	processor := NewParallelProcessor(&ProcessorConfig{
		WorkerCount: 2,
		QueueSize:   10,
	})

	items := make([]interface{}, 100)
	for i := range items {
		items[i] = i
	}

	// Slow processor function
	processorFunc := func(ctx context.Context, item interface{}) (interface{}, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			return item, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	results, err := processor.ProcessBatch(ctx, items, processorFunc)

	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Nil(t, results)
}

func TestConcurrentProcessing(t *testing.T) {
	processor := NewParallelProcessor(&ProcessorConfig{
		WorkerCount: 4,
		QueueSize:   100,
	})

	var counter int32
	items := make([]interface{}, 20)
	for i := range items {
		items[i] = i
	}

	processorFunc := func(ctx context.Context, item interface{}) (interface{}, error) {
		atomic.AddInt32(&counter, 1)
		time.Sleep(10 * time.Millisecond)
		return item, nil
	}

	start := time.Now()
	ctx := context.Background()
	results, err := processor.ProcessBatch(ctx, items, processorFunc)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Len(t, results, 20)
	assert.Equal(t, int32(20), atomic.LoadInt32(&counter))

	// With 4 workers processing 20 items that each take 10ms,
	// it should take roughly 50ms (20/4 * 10ms), not 200ms (serial)
	assert.Less(t, duration, 100*time.Millisecond)
}

func TestWorkStealing(t *testing.T) {
	config := &ProcessorConfig{
		WorkerCount:    2,
		QueueSize:      100,
		StealThreshold: 2,
	}

	processor := NewParallelProcessor(config)
	processor.stealer = &WorkStealer{
		processor: processor,
		enabled:   true,
	}

	err := processor.Start()
	require.NoError(t, err)
	defer processor.Stop(context.Background())

	// Submit work items with different processing times
	var wg sync.WaitGroup
	results := make([]int, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		idx := i
		workItem := WorkItem{
			ID: fmt.Sprintf("work-%d", idx),
			Task: func(ctx context.Context) (interface{}, error) {
				// Some items take longer
				if idx < 5 {
					time.Sleep(50 * time.Millisecond)
				} else {
					time.Sleep(10 * time.Millisecond)
				}
				return idx, nil
			},
			Context: context.Background(),
			Metadata: map[string]interface{}{
				"resultChan": make(chan WorkResult, 1),
			},
		}

		processor.workQueue <- workItem

		go func(idx int) {
			defer wg.Done()
			resultChan := workItem.Metadata["resultChan"].(chan WorkResult)
			result := <-resultChan
			if result.Error == nil {
				results[idx] = result.Result.(int)
			}
		}(idx)
	}

	wg.Wait()

	// Verify all work was completed
	for i := 0; i < 10; i++ {
		assert.Equal(t, i, results[i])
	}
}

func TestRetryMechanism(t *testing.T) {
	processor := NewParallelProcessor(&ProcessorConfig{
		WorkerCount: 1,
		QueueSize:   10,
		MaxRetries:  3,
	})

	err := processor.Start()
	require.NoError(t, err)
	defer processor.Stop(context.Background())

	attemptCount := 0
	var mu sync.Mutex

	resultChan := make(chan WorkResult, 1)
	workItem := WorkItem{
		ID: "retry-test",
		Task: func(ctx context.Context) (interface{}, error) {
			mu.Lock()
			attemptCount++
			count := attemptCount
			mu.Unlock()

			if count < 3 {
				return nil, errors.New("temporary error")
			}
			return "success", nil
		},
		Context:    context.Background(),
		MaxRetries: 3,
		Metadata: map[string]interface{}{
			"resultChan": resultChan,
		},
	}

	processor.workQueue <- workItem

	// Wait for result with timeout
	select {
	case result := <-resultChan:
		assert.NoError(t, result.Error)
		assert.Equal(t, "success", result.Result)
		assert.Equal(t, 3, attemptCount)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for result")
	}
}

func TestProcessorMetrics(t *testing.T) {
	processor := NewParallelProcessor(&ProcessorConfig{
		WorkerCount: 2,
		QueueSize:   10,
	})

	items := []interface{}{1, 2, 3}
	processorFunc := func(ctx context.Context, item interface{}) (interface{}, error) {
		return item, nil
	}

	ctx := context.Background()
	_, err := processor.ProcessBatch(ctx, items, processorFunc)
	require.NoError(t, err)

	stats := processor.GetStats()
	assert.Greater(t, stats.TotalQueued, int64(0))
	assert.Greater(t, stats.TotalProcessed, int64(0))
	assert.Greater(t, stats.TotalCompleted, int64(0))
}

func BenchmarkProcessBatch(b *testing.B) {
	processor := NewParallelProcessor(&ProcessorConfig{
		WorkerCount: 4,
		QueueSize:   1000,
	})

	items := make([]interface{}, 100)
	for i := range items {
		items[i] = i
	}

	processorFunc := func(ctx context.Context, item interface{}) (interface{}, error) {
		// Simulate some work
		sum := 0
		for i := 0; i < 1000; i++ {
			sum += i
		}
		return sum, nil
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = processor.ProcessBatch(ctx, items, processorFunc)
	}
}

func BenchmarkConcurrentProcessing(b *testing.B) {
	testCases := []struct {
		name        string
		workerCount int
		itemCount   int
	}{
		{"2workers-100items", 2, 100},
		{"4workers-100items", 4, 100},
		{"8workers-100items", 8, 100},
		{"4workers-1000items", 4, 1000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			processor := NewParallelProcessor(&ProcessorConfig{
				WorkerCount: tc.workerCount,
				QueueSize:   tc.itemCount * 2,
			})

			items := make([]interface{}, tc.itemCount)
			for i := range items {
				items[i] = i
			}

			processorFunc := func(ctx context.Context, item interface{}) (interface{}, error) {
				return item.(int) * 2, nil
			}

			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = processor.ProcessBatch(ctx, items, processorFunc)
			}
		})
	}
}
