package errors

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDriftError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *DriftError
		expected string
	}{
		{
			name: "basic error",
			err: &DriftError{
				Message: "Test error",
			},
			expected: "Test error",
		},
		{
			name: "error with code",
			err: &DriftError{
				Code:    "TEST001",
				Message: "Test error",
			},
			expected: "[TEST001] Test error",
		},
		{
			name: "error with resource",
			err: &DriftError{
				Message:  "Test error",
				Resource: "test-resource",
			},
			expected: "Test error (resource: test-resource)",
		},
		{
			name: "error with wrapped error",
			err: &DriftError{
				Message: "Test error",
				Wrapped: errors.New("wrapped error"),
			},
			expected: "Test error caused by: wrapped error",
		},
		{
			name: "complete error",
			err: &DriftError{
				Code:     "TEST001",
				Message:  "Test error",
				Resource: "test-resource",
				Wrapped:  errors.New("wrapped error"),
			},
			expected: "[TEST001] Test error (resource: test-resource) caused by: wrapped error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDriftError_Unwrap(t *testing.T) {
	wrappedErr := errors.New("wrapped error")
	driftErr := &DriftError{
		Message: "Test error",
		Wrapped: wrappedErr,
	}

	assert.Equal(t, wrappedErr, driftErr.Unwrap())
}

func TestDriftError_Is(t *testing.T) {
	err1 := &DriftError{
		Type: ErrorTypeTransient,
		Code: "TEST001",
	}
	err2 := &DriftError{
		Type: ErrorTypeTransient,
		Code: "TEST001",
	}
	err3 := &DriftError{
		Type: ErrorTypePermanent,
		Code: "TEST001",
	}
	err4 := &DriftError{
		Type: ErrorTypeTransient,
		Code: "TEST002",
	}

	assert.True(t, err1.Is(err2))
	assert.False(t, err1.Is(err3))
	assert.False(t, err1.Is(err4))
	assert.False(t, err1.Is(errors.New("different error")))
}

func TestDriftError_WithUserHelp(t *testing.T) {
	err := &DriftError{
		Message: "Test error",
	}

	result := err.WithUserHelp("User help text")

	assert.Equal(t, "User help text", result.UserHelp)
	assert.Equal(t, err, result) // Should return same instance
}

func TestDriftError_WithRecovery(t *testing.T) {
	err := &DriftError{
		Message: "Test error",
	}

	recovery := RecoveryStrategy{
		Strategy:    "retry",
		Description: "Retry the operation",
	}

	result := err.WithRecovery(recovery)

	assert.Equal(t, recovery, result.Recovery)
	assert.Equal(t, err, result) // Should return same instance
}

func TestDriftError_WithDetails(t *testing.T) {
	err := &DriftError{
		Message: "Test error",
	}

	result := err.WithDetails("key1", "value1")
	assert.Equal(t, "value1", result.Details["key1"])
	assert.Equal(t, err, result) // Should return same instance

	result = err.WithDetails("key2", 123)
	assert.Equal(t, 123, result.Details["key2"])
	assert.Equal(t, "value1", result.Details["key1"]) // Previous details should remain
}

func TestDriftError_ToJSON(t *testing.T) {
	err := &DriftError{
		Type:      ErrorTypeTransient,
		Severity:  SeverityHigh,
		Code:      "TEST001",
		Message:   "Test error",
		UserHelp:  "User help",
		Resource:  "test-resource",
		Provider:  "aws",
		Operation: "test-operation",
		Details: map[string]interface{}{
			"key": "value",
		},
		Timestamp:  time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		TraceID:    "trace-123",
		Retryable:  true,
		RetryAfter: 5 * time.Second,
	}

	jsonStr := err.ToJSON()
	assert.Contains(t, jsonStr, "TEST001")
	assert.Contains(t, jsonStr, "Test error")
	assert.Contains(t, jsonStr, "transient")
	assert.Contains(t, jsonStr, "high")
}

func TestErrorBuilder_NewError(t *testing.T) {
	builder := NewError(ErrorTypeTransient, "Test error")

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.err)
	assert.Equal(t, ErrorTypeTransient, builder.err.Type)
	assert.Equal(t, "Test error", builder.err.Message)
	assert.Equal(t, SeverityMedium, builder.err.Severity)
	assert.False(t, builder.err.Timestamp.IsZero())
	assert.NotEmpty(t, builder.err.StackTrace)
	assert.NotNil(t, builder.err.Details)
}

func TestErrorBuilder_WithCode(t *testing.T) {
	builder := NewError(ErrorTypeTransient, "Test error")
	result := builder.WithCode("TEST001")

	assert.Equal(t, "TEST001", builder.err.Code)
	assert.Equal(t, builder, result) // Should return same instance
}

func TestErrorBuilder_WithSeverity(t *testing.T) {
	builder := NewError(ErrorTypeTransient, "Test error")
	result := builder.WithSeverity(SeverityCritical)

	assert.Equal(t, SeverityCritical, builder.err.Severity)
	assert.Equal(t, builder, result) // Should return same instance
}

func TestErrorBuilder_WithResource(t *testing.T) {
	builder := NewError(ErrorTypeTransient, "Test error")
	result := builder.WithResource("test-resource")

	assert.Equal(t, "test-resource", builder.err.Resource)
	assert.Equal(t, builder, result) // Should return same instance
}

func TestErrorBuilder_WithProvider(t *testing.T) {
	builder := NewError(ErrorTypeTransient, "Test error")
	result := builder.WithProvider("aws")

	assert.Equal(t, "aws", builder.err.Provider)
	assert.Equal(t, builder, result) // Should return same instance
}

func TestErrorBuilder_WithOperation(t *testing.T) {
	builder := NewError(ErrorTypeTransient, "Test error")
	result := builder.WithOperation("test-operation")

	assert.Equal(t, "test-operation", builder.err.Operation)
	assert.Equal(t, builder, result) // Should return same instance
}

func TestErrorBuilder_WithUserHelp(t *testing.T) {
	builder := NewError(ErrorTypeTransient, "Test error")
	result := builder.WithUserHelp("User help text")

	assert.Equal(t, "User help text", builder.err.UserHelp)
	assert.Equal(t, builder, result) // Should return same instance
}

func TestErrorBuilder_WithDetails(t *testing.T) {
	builder := NewError(ErrorTypeTransient, "Test error")
	result := builder.WithDetails("key1", "value1")

	assert.Equal(t, "value1", builder.err.Details["key1"])
	assert.Equal(t, builder, result) // Should return same instance

	result = builder.WithDetails("key2", 123)
	assert.Equal(t, 123, builder.err.Details["key2"])
	assert.Equal(t, "value1", builder.err.Details["key1"]) // Previous details should remain
}

func TestErrorBuilder_WithWrapped(t *testing.T) {
	builder := NewError(ErrorTypeTransient, "Test error")
	wrappedErr := errors.New("wrapped error")
	result := builder.WithWrapped(wrappedErr)

	assert.Equal(t, wrappedErr, builder.err.Wrapped)
	assert.Equal(t, builder, result) // Should return same instance
}

func TestErrorBuilder_WithRetry(t *testing.T) {
	builder := NewError(ErrorTypeTransient, "Test error")
	result := builder.WithRetry(true, 5*time.Second)

	assert.True(t, builder.err.Retryable)
	assert.Equal(t, 5*time.Second, builder.err.RetryAfter)
	assert.Equal(t, builder, result) // Should return same instance
}

func TestErrorBuilder_WithRecovery(t *testing.T) {
	builder := NewError(ErrorTypeTransient, "Test error")
	recovery := RecoveryStrategy{
		Strategy:    "retry",
		Description: "Retry the operation",
	}
	result := builder.WithRecovery(recovery)

	assert.Equal(t, recovery, builder.err.Recovery)
	assert.Equal(t, builder, result) // Should return same instance
}

func TestErrorBuilder_WithContext(t *testing.T) {
	builder := NewError(ErrorTypeTransient, "Test error")
	ctx := context.WithValue(context.Background(), "trace_id", "trace-123")
	ctx = context.WithValue(ctx, "user_id", "user-456")
	ctx = context.WithValue(ctx, "request_id", "req-789")
	result := builder.WithContext(ctx)

	assert.Equal(t, "trace-123", builder.err.TraceID)
	assert.Equal(t, "user-456", builder.err.Details["user_id"])
	assert.Equal(t, "req-789", builder.err.Details["request_id"])
	assert.Equal(t, builder, result) // Should return same instance
}

func TestErrorBuilder_Build(t *testing.T) {
	builder := NewError(ErrorTypeTransient, "Test error")
	err := builder.Build()

	assert.Equal(t, builder.err, err)
}

func TestErrorBuilder_Error(t *testing.T) {
	builder := NewError(ErrorTypeTransient, "Test error")
	err := builder.Error()

	assert.Equal(t, builder.err, err)
}

func TestNewTransientError(t *testing.T) {
	err := NewTransientError("Test transient error", 5*time.Second)

	assert.Equal(t, ErrorTypeTransient, err.Type)
	assert.Equal(t, "Test transient error", err.Message)
	assert.True(t, err.Retryable)
	assert.Equal(t, 5*time.Second, err.RetryAfter)
	assert.Equal(t, "retry", err.Recovery.Strategy)
	assert.Contains(t, err.Recovery.Description, "Retry the operation")
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("test-resource", "Validation failed")

	assert.Equal(t, ErrorTypeValidation, err.Type)
	assert.Equal(t, "Validation failed", err.Message)
	assert.Equal(t, "test-resource", err.Resource)
	assert.Equal(t, SeverityLow, err.Severity)
	assert.Contains(t, err.UserHelp, "Please check your input")
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("test-resource")

	assert.Equal(t, ErrorTypeNotFound, err.Type)
	assert.Contains(t, err.Message, "Resource not found")
	assert.Contains(t, err.Message, "test-resource")
	assert.Equal(t, "test-resource", err.Resource)
	assert.Equal(t, SeverityLow, err.Severity)
	assert.Contains(t, err.UserHelp, "Ensure the resource exists")
}

func TestNewTimeoutError(t *testing.T) {
	err := NewTimeoutError("test-operation", 30*time.Second)

	assert.Equal(t, ErrorTypeTimeout, err.Type)
	assert.Contains(t, err.Message, "Operation timed out")
	assert.Contains(t, err.Message, "30s")
	assert.Equal(t, "test-operation", err.Operation)
	assert.Equal(t, SeverityMedium, err.Severity)
	assert.True(t, err.Retryable)
	assert.Equal(t, 5*time.Second, err.RetryAfter)
	assert.Equal(t, "retry_with_backoff", err.Recovery.Strategy)
	assert.Contains(t, err.Recovery.Description, "Retry with exponential backoff")
	assert.Equal(t, 3, err.Recovery.Params["max_retries"])
	assert.Equal(t, "5s", err.Recovery.Params["base_delay"])
}

func TestErrorHandler_NewErrorHandler(t *testing.T) {
	handler := NewErrorHandler()

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.handlers)
	assert.NotNil(t, handler.fallback)
}

func TestErrorHandler_RegisterHandler(t *testing.T) {
	handler := NewErrorHandler()
	customHandler := func(err *DriftError) error {
		return err
	}

	handler.RegisterHandler(ErrorTypeTransient, customHandler)

	// Can't compare functions directly, just verify handler was registered
	assert.NotNil(t, handler.handlers[ErrorTypeTransient])
}

func TestErrorHandler_SetFallback(t *testing.T) {
	handler := NewErrorHandler()
	customFallback := func(err *DriftError) error {
		return err
	}

	handler.SetFallback(customFallback)

	// Can't compare functions directly, just verify fallback was set
	assert.NotNil(t, handler.fallback)
}

func TestErrorHandler_Handle_DriftError(t *testing.T) {
	handler := NewErrorHandler()
	driftErr := &DriftError{
		Type:    ErrorTypeTransient,
		Message: "Test error",
	}

	result := handler.Handle(driftErr)

	assert.Equal(t, driftErr, result)
}

func TestErrorHandler_Handle_NonDriftError(t *testing.T) {
	handler := NewErrorHandler()
	regularErr := errors.New("regular error")

	result := handler.Handle(regularErr)

	driftErr, ok := result.(*DriftError)
	require.True(t, ok)
	assert.Equal(t, ErrorTypeSystem, driftErr.Type)
	assert.Equal(t, "regular error", driftErr.Message)
	assert.Equal(t, regularErr, driftErr.Wrapped)
}

func TestErrorHandler_Handle_WithCustomHandler(t *testing.T) {
	handler := NewErrorHandler()
	customHandler := func(err *DriftError) error {
		return errors.New("custom handled")
	}
	handler.RegisterHandler(ErrorTypeTransient, customHandler)

	driftErr := &DriftError{
		Type:    ErrorTypeTransient,
		Message: "Test error",
	}

	result := handler.Handle(driftErr)

	assert.Equal(t, "custom handled", result.Error())
}

func TestErrorContext_WithErrorContext(t *testing.T) {
	ctx := context.Background()
	errorCtx := WithErrorContext(ctx)

	assert.NotNil(t, errorCtx)
	assert.NotNil(t, errorCtx.Context)
	assert.NotNil(t, errorCtx.errors)
	assert.Len(t, errorCtx.errors, 0)
}

func TestErrorContext_AddError(t *testing.T) {
	ctx := context.Background()
	errorCtx := WithErrorContext(ctx)

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	errorCtx.AddError(err1)
	errorCtx.AddError(err2)

	assert.Len(t, errorCtx.errors, 2)
	assert.Equal(t, err1, errorCtx.errors[0])
	assert.Equal(t, err2, errorCtx.errors[1])
}

func TestErrorContext_HasErrors(t *testing.T) {
	ctx := context.Background()
	errorCtx := WithErrorContext(ctx)

	assert.False(t, errorCtx.HasErrors())

	errorCtx.AddError(errors.New("test error"))

	assert.True(t, errorCtx.HasErrors())
}

func TestErrorContext_GetErrors(t *testing.T) {
	ctx := context.Background()
	errorCtx := WithErrorContext(ctx)

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	errorCtx.AddError(err1)
	errorCtx.AddError(err2)

	errors := errorCtx.GetErrors()
	assert.Len(t, errors, 2)
	assert.Equal(t, err1, errors[0])
	assert.Equal(t, err2, errors[1])
}

func TestErrorContext_GetFirstError(t *testing.T) {
	ctx := context.Background()
	errorCtx := WithErrorContext(ctx)

	assert.Nil(t, errorCtx.GetFirstError())

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	errorCtx.AddError(err1)
	errorCtx.AddError(err2)

	assert.Equal(t, err1, errorCtx.GetFirstError())
}

func TestErrorType_Constants(t *testing.T) {
	assert.Equal(t, string(ErrorTypeTransient), "transient")
	assert.Equal(t, string(ErrorTypePermanent), "permanent")
	assert.Equal(t, string(ErrorTypeUser), "user")
	assert.Equal(t, string(ErrorTypeSystem), "system")
	assert.Equal(t, string(ErrorTypeValidation), "validation")
	assert.Equal(t, string(ErrorTypeNotFound), "not_found")
	assert.Equal(t, string(ErrorTypeConflict), "conflict")
	assert.Equal(t, string(ErrorTypeTimeout), "timeout")
}

func TestErrorSeverity_Constants(t *testing.T) {
	assert.Equal(t, string(SeverityCritical), "critical")
	assert.Equal(t, string(SeverityHigh), "high")
	assert.Equal(t, string(SeverityMedium), "medium")
	assert.Equal(t, string(SeverityLow), "low")
}

func TestRecoveryStrategy_Struct(t *testing.T) {
	strategy := RecoveryStrategy{
		Strategy:    "retry",
		Description: "Retry the operation",
		Steps:       []string{"step1", "step2"},
		Params: map[string]interface{}{
			"max_retries": 3,
			"delay":       "5s",
		},
	}

	assert.Equal(t, "retry", strategy.Strategy)
	assert.Equal(t, "Retry the operation", strategy.Description)
	assert.Len(t, strategy.Steps, 2)
	assert.Equal(t, "step1", strategy.Steps[0])
	assert.Equal(t, "step2", strategy.Steps[1])
	assert.Equal(t, 3, strategy.Params["max_retries"])
	assert.Equal(t, "5s", strategy.Params["delay"])
}

func TestDriftError_JSONSerialization(t *testing.T) {
	err := &DriftError{
		Type:      ErrorTypeTransient,
		Severity:  SeverityHigh,
		Code:      "TEST001",
		Message:   "Test error",
		UserHelp:  "User help",
		Resource:  "test-resource",
		Provider:  "aws",
		Operation: "test-operation",
		Details: map[string]interface{}{
			"key": "value",
		},
		Timestamp:  time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		TraceID:    "trace-123",
		Retryable:  true,
		RetryAfter: 5 * time.Second,
		Recovery: RecoveryStrategy{
			Strategy:    "retry",
			Description: "Retry the operation",
		},
	}

	jsonStr := err.ToJSON()
	assert.Contains(t, jsonStr, `"type": "transient"`)
	assert.Contains(t, jsonStr, `"severity": "high"`)
	assert.Contains(t, jsonStr, `"code": "TEST001"`)
	assert.Contains(t, jsonStr, `"message": "Test error"`)
	assert.Contains(t, jsonStr, `"user_help": "User help"`)
	assert.Contains(t, jsonStr, `"resource": "test-resource"`)
	assert.Contains(t, jsonStr, `"provider": "aws"`)
	assert.Contains(t, jsonStr, `"operation": "test-operation"`)
	assert.Contains(t, jsonStr, `"trace_id": "trace-123"`)
	assert.Contains(t, jsonStr, `"retryable": true`)
	assert.Contains(t, jsonStr, `"retry_after": 5000000000`)
}

func TestErrorBuilder_Chaining(t *testing.T) {
	wrappedErr := errors.New("wrapped error")
	recovery := RecoveryStrategy{
		Strategy:    "retry",
		Description: "Retry the operation",
	}

	err := NewError(ErrorTypeTransient, "Test error").
		WithCode("TEST001").
		WithSeverity(SeverityHigh).
		WithResource("test-resource").
		WithProvider("aws").
		WithOperation("test-operation").
		WithUserHelp("User help text").
		WithDetails("key1", "value1").
		WithDetails("key2", 123).
		WithWrapped(wrappedErr).
		WithRetry(true, 5*time.Second).
		WithRecovery(recovery).
		Build()

	assert.Equal(t, ErrorTypeTransient, err.Type)
	assert.Equal(t, "TEST001", err.Code)
	assert.Equal(t, SeverityHigh, err.Severity)
	assert.Equal(t, "test-resource", err.Resource)
	assert.Equal(t, "aws", err.Provider)
	assert.Equal(t, "test-operation", err.Operation)
	assert.Equal(t, "User help text", err.UserHelp)
	assert.Equal(t, "value1", err.Details["key1"])
	assert.Equal(t, 123, err.Details["key2"])
	assert.Equal(t, wrappedErr, err.Wrapped)
	assert.True(t, err.Retryable)
	assert.Equal(t, 5*time.Second, err.RetryAfter)
	assert.Equal(t, recovery, err.Recovery)
}
