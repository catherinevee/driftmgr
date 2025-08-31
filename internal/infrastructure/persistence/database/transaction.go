package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/logging"
)

// TransactionManager manages database transactions with proper isolation
type TransactionManager struct {
	db              *sql.DB
	isolationLevel  sql.IsolationLevel
	maxRetries      int
	retryDelay      time.Duration
	deadlockRetries int
	logger          *logging.Logger
	metrics         *TransactionMetrics
	mu              sync.RWMutex
}

// TransactionMetrics tracks transaction performance
type TransactionMetrics struct {
	TotalTransactions   int64
	SuccessfulCommits   int64
	Rollbacks          int64
	Deadlocks          int64
	Retries            int64
	AverageDuration    time.Duration
	mu                 sync.RWMutex
}

// TransactionOptions configures transaction behavior
type TransactionOptions struct {
	IsolationLevel  sql.IsolationLevel
	ReadOnly        bool
	Timeout         time.Duration
	MaxRetries      int
	RetryableErrors func(error) bool
}

// DefaultTransactionOptions returns default transaction options
func DefaultTransactionOptions() *TransactionOptions {
	return &TransactionOptions{
		IsolationLevel: sql.LevelReadCommitted,
		ReadOnly:       false,
		Timeout:        30 * time.Second,
		MaxRetries:     3,
		RetryableErrors: func(err error) bool {
			return IsDeadlock(err) || IsLockTimeout(err)
		},
	}
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(db *sql.DB) *TransactionManager {
	return &TransactionManager{
		db:              db,
		isolationLevel:  sql.LevelReadCommitted,
		maxRetries:      3,
		retryDelay:      100 * time.Millisecond,
		deadlockRetries: 3,
		logger:          logging.GetLogger(),
		metrics:         &TransactionMetrics{},
	}
}

// Execute executes a function within a transaction
func (tm *TransactionManager) Execute(ctx context.Context, fn func(*sql.Tx) error) error {
	return tm.ExecuteWithOptions(ctx, DefaultTransactionOptions(), fn)
}

// ExecuteWithOptions executes a function within a transaction with options
func (tm *TransactionManager) ExecuteWithOptions(ctx context.Context, opts *TransactionOptions, fn func(*sql.Tx) error) error {
	if opts == nil {
		opts = DefaultTransactionOptions()
	}

	// Apply timeout to context
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	var err error
	for attempt := 0; attempt <= opts.MaxRetries; attempt++ {
		err = tm.executeTransaction(ctx, opts, fn)
		
		if err == nil {
			// Success
			return nil
		}

		// Check if error is retryable
		if !opts.RetryableErrors(err) {
			tm.logger.Error("Transaction failed with non-retryable error", err, map[string]interface{}{
				"attempt": attempt,
			})
			return err
		}

		// Check if we've exhausted retries
		if attempt >= opts.MaxRetries {
			tm.logger.Error("Transaction failed after max retries", err, map[string]interface{}{
				"attempts": attempt + 1,
			})
			break
		}

		// Log retry
		tm.logger.Warn("Transaction failed, retrying", map[string]interface{}{
			"error":   err.Error(),
			"attempt": attempt,
			"next_attempt": attempt + 1,
		})

		// Update metrics
		tm.metrics.mu.Lock()
		tm.metrics.Retries++
		if IsDeadlock(err) {
			tm.metrics.Deadlocks++
		}
		tm.metrics.mu.Unlock()

		// Wait before retry with exponential backoff
		delay := tm.retryDelay * time.Duration(1<<uint(attempt))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to retry
		}
	}

	return fmt.Errorf("transaction failed after %d attempts: %w", opts.MaxRetries+1, err)
}

// executeTransaction executes a single transaction attempt
func (tm *TransactionManager) executeTransaction(ctx context.Context, opts *TransactionOptions, fn func(*sql.Tx) error) error {
	startTime := time.Now()
	
	// Update metrics
	tm.metrics.mu.Lock()
	tm.metrics.TotalTransactions++
	tm.metrics.mu.Unlock()

	// Begin transaction with options
	tx, err := tm.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: opts.IsolationLevel,
		ReadOnly:  opts.ReadOnly,
	})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure transaction is finalized
	committed := false
	defer func() {
		if !committed {
			if rbErr := tx.Rollback(); rbErr != nil && rbErr != sql.ErrTxDone {
				tm.logger.Error("Failed to rollback transaction", rbErr, nil)
			}
			
			tm.metrics.mu.Lock()
			tm.metrics.Rollbacks++
			tm.metrics.mu.Unlock()
		}
	}()

	// Execute the function
	if err := fn(tx); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	committed = true
	
	// Update metrics
	duration := time.Since(startTime)
	tm.metrics.mu.Lock()
	tm.metrics.SuccessfulCommits++
	tm.metrics.AverageDuration = (tm.metrics.AverageDuration + duration) / 2
	tm.metrics.mu.Unlock()

	return nil
}

// ExecuteInSavepoint executes a function within a savepoint
func (tm *TransactionManager) ExecuteInSavepoint(tx *sql.Tx, savepointName string, fn func() error) error {
	// Create savepoint
	if _, err := tx.Exec(fmt.Sprintf("SAVEPOINT %s", savepointName)); err != nil {
		return fmt.Errorf("failed to create savepoint: %w", err)
	}

	// Execute function
	if err := fn(); err != nil {
		// Rollback to savepoint
		if _, rbErr := tx.Exec(fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", savepointName)); rbErr != nil {
			tm.logger.Error("Failed to rollback to savepoint", rbErr, map[string]interface{}{
				"savepoint": savepointName,
			})
		}
		return err
	}

	// Release savepoint
	if _, err := tx.Exec(fmt.Sprintf("RELEASE SAVEPOINT %s", savepointName)); err != nil {
		return fmt.Errorf("failed to release savepoint: %w", err)
	}

	return nil
}

// GetMetrics returns transaction metrics
func (tm *TransactionManager) GetMetrics() TransactionMetrics {
	tm.metrics.mu.RLock()
	defer tm.metrics.mu.RUnlock()
	
	return TransactionMetrics{
		TotalTransactions: tm.metrics.TotalTransactions,
		SuccessfulCommits: tm.metrics.SuccessfulCommits,
		Rollbacks:        tm.metrics.Rollbacks,
		Deadlocks:        tm.metrics.Deadlocks,
		Retries:          tm.metrics.Retries,
		AverageDuration:  tm.metrics.AverageDuration,
	}
}

// BatchExecutor executes multiple operations in batches within transactions
type BatchExecutor struct {
	tm        *TransactionManager
	batchSize int
	logger    *logging.Logger
}

// NewBatchExecutor creates a new batch executor
func NewBatchExecutor(tm *TransactionManager, batchSize int) *BatchExecutor {
	return &BatchExecutor{
		tm:        tm,
		batchSize: batchSize,
		logger:    logging.GetLogger(),
	}
}

// ExecuteBatch executes operations in batches
func (be *BatchExecutor) ExecuteBatch(ctx context.Context, items []interface{}, processor func(*sql.Tx, []interface{}) error) error {
	totalItems := len(items)
	
	for i := 0; i < totalItems; i += be.batchSize {
		end := i + be.batchSize
		if end > totalItems {
			end = totalItems
		}
		
		batch := items[i:end]
		
		// Execute batch in transaction
		err := be.tm.Execute(ctx, func(tx *sql.Tx) error {
			return processor(tx, batch)
		})
		
		if err != nil {
			be.logger.Error("Batch execution failed", err, map[string]interface{}{
				"batch_start": i,
				"batch_end":   end,
			})
			return fmt.Errorf("batch %d-%d failed: %w", i, end, err)
		}
		
		be.logger.Debug("Batch executed successfully", map[string]interface{}{
			"batch_start": i,
			"batch_end":   end,
			"items":       len(batch),
		})
	}
	
	return nil
}

// UnitOfWork represents a unit of work pattern
type UnitOfWork struct {
	tx         *sql.Tx
	tm         *TransactionManager
	operations []Operation
	committed  bool
	rolledback bool
	mu         sync.Mutex
}

// Operation represents a database operation
type Operation struct {
	Name     string
	Execute  func(*sql.Tx) error
	Rollback func(*sql.Tx) error // Optional compensating action
}

// NewUnitOfWork creates a new unit of work
func (tm *TransactionManager) NewUnitOfWork(ctx context.Context) (*UnitOfWork, error) {
	tx, err := tm.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: tm.isolationLevel,
	})
	if err != nil {
		return nil, err
	}

	return &UnitOfWork{
		tx:         tx,
		tm:         tm,
		operations: make([]Operation, 0),
	}, nil
}

// AddOperation adds an operation to the unit of work
func (uow *UnitOfWork) AddOperation(op Operation) {
	uow.mu.Lock()
	defer uow.mu.Unlock()
	
	uow.operations = append(uow.operations, op)
}

// Execute executes all operations in the unit of work
func (uow *UnitOfWork) Execute() error {
	uow.mu.Lock()
	defer uow.mu.Unlock()

	if uow.committed || uow.rolledback {
		return errors.New("unit of work already finalized")
	}

	// Execute all operations
	executedOps := make([]Operation, 0, len(uow.operations))
	
	for _, op := range uow.operations {
		if err := op.Execute(uow.tx); err != nil {
			// Rollback executed operations if they have compensating actions
			for i := len(executedOps) - 1; i >= 0; i-- {
				if executedOps[i].Rollback != nil {
					if rbErr := executedOps[i].Rollback(uow.tx); rbErr != nil {
						uow.tm.logger.Error("Failed to execute compensating action", rbErr, map[string]interface{}{
							"operation": executedOps[i].Name,
						})
					}
				}
			}
			
			return fmt.Errorf("operation %s failed: %w", op.Name, err)
		}
		
		executedOps = append(executedOps, op)
	}

	return nil
}

// Commit commits the unit of work
func (uow *UnitOfWork) Commit() error {
	uow.mu.Lock()
	defer uow.mu.Unlock()

	if uow.committed || uow.rolledback {
		return errors.New("unit of work already finalized")
	}

	if err := uow.tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit unit of work: %w", err)
	}

	uow.committed = true
	return nil
}

// Rollback rolls back the unit of work
func (uow *UnitOfWork) Rollback() error {
	uow.mu.Lock()
	defer uow.mu.Unlock()

	if uow.committed || uow.rolledback {
		return errors.New("unit of work already finalized")
	}

	if err := uow.tx.Rollback(); err != nil && err != sql.ErrTxDone {
		return fmt.Errorf("failed to rollback unit of work: %w", err)
	}

	uow.rolledback = true
	return nil
}

// Error detection helpers

// IsDeadlock checks if an error is a deadlock error
func IsDeadlock(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	// MySQL/MariaDB
	if contains(errStr, "Deadlock found") || contains(errStr, "Lock wait timeout") {
		return true
	}
	// PostgreSQL
	if contains(errStr, "deadlock detected") {
		return true
	}
	// SQL Server
	if contains(errStr, "Transaction (Process ID") && contains(errStr, "was deadlocked") {
		return true
	}
	
	return false
}

// IsLockTimeout checks if an error is a lock timeout
func IsLockTimeout(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	return contains(errStr, "lock timeout") || 
		   contains(errStr, "Lock wait timeout exceeded") ||
		   contains(errStr, "could not obtain lock")
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}