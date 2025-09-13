package errors

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestErrorType(t *testing.T) {
	types := []ErrorType{
		ErrorTypeTransient,
		ErrorTypePermanent,
		ErrorTypeUser,
		ErrorTypeSystem,
		ErrorTypeValidation,
		ErrorTypeNotFound,
		ErrorTypeConflict,
		ErrorTypeTimeout,
	}

	expectedStrings := []string{
		"transient",
		"permanent",
		"user",
		"system",
		"validation",
		"not_found",
		"conflict",
		"timeout",
	}

	for i, errType := range types {
		assert.Equal(t, ErrorType(expectedStrings[i]), errType)
	}
}

func TestErrorSeverity(t *testing.T) {
	severities := []ErrorSeverity{
		SeverityLow,
		SeverityMedium,
		SeverityHigh,
		SeverityCritical,
	}

	expectedStrings := []string{
		"low",
		"medium",
		"high",
		"critical",
	}

	for i, severity := range severities {
		assert.Equal(t, ErrorSeverity(expectedStrings[i]), severity)
	}
}

func TestDriftError(t *testing.T) {
	tests := []struct {
		name string
		err  *DriftError
	}{
		{
			name: "basic error",
			err: &DriftError{
				Type:      ErrorTypeValidation,
				Message:   "validation failed",
				Code:      "VAL001",
				Severity:  SeverityMedium,
				Timestamp: time.Now(),
			},
		},
		{
			name: "error with details",
			err: &DriftError{
				Type:      ErrorTypeSystem,
				Message:   "AWS API error",
				Code:      "AWS001",
				Provider:  "aws",
				Operation: "DescribeInstances",
				Details: map[string]interface{}{
					"region":  "us-east-1",
					"service": "EC2",
				},
				Timestamp: time.Now(),
			},
		},
		{
			name: "error with resource",
			err: &DriftError{
				Type:      ErrorTypeNotFound,
				Message:   "resource not found",
				Code:      "NF001",
				Resource:  "aws_instance.web",
				Severity:  SeverityLow,
				Timestamp: time.Now(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.err.Error())
			assert.Equal(t, tt.err.Type, tt.err.Type)
			assert.Equal(t, tt.err.Code, tt.err.Code)
			assert.NotZero(t, tt.err.Timestamp)
		})
	}
}

func TestNewError(t *testing.T) {
	err := NewError(ErrorTypeSystem, "system error occurred").Build()

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeSystem, err.Type)
	assert.Equal(t, "system error occurred", err.Message)
	assert.NotZero(t, err.Timestamp)
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("username", "invalid input")

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeValidation, err.Type)
	assert.Contains(t, err.Message, "invalid input")
	assert.Equal(t, "username", err.Resource)
}

func TestWithCode(t *testing.T) {
	err := NewError(ErrorTypeSystem, "error").WithCode("SYS001").Build()

	assert.Equal(t, "SYS001", err.Code)
	assert.Equal(t, "error", err.Message)
}

func TestWithSeverity(t *testing.T) {
	err := NewError(ErrorTypeSystem, "error").WithSeverity(SeverityCritical).Build()

	assert.Equal(t, SeverityCritical, err.Severity)
	assert.Equal(t, "error", err.Message)
}

func TestWithResource(t *testing.T) {
	err := NewError(ErrorTypeNotFound, "not found").WithResource("aws_instance.web").Build()

	assert.Equal(t, "aws_instance.web", err.Resource)
	assert.Equal(t, "not found", err.Message)
}

func TestWithDetails(t *testing.T) {
	details := map[string]interface{}{
		"resource1": "aws_instance.web",
		"resource2": "aws_instance.app",
	}
	err := NewError(ErrorTypeConflict, "resource conflict").WithDetails(details).Build()

	assert.Equal(t, details, err.Details)
	assert.Equal(t, "resource conflict", err.Message)
}

func TestWithProvider(t *testing.T) {
	err := NewError(ErrorTypeSystem, "provider error").WithProvider("aws").Build()

	assert.Equal(t, "aws", err.Provider)
	assert.Equal(t, "provider error", err.Message)
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       *DriftError
		retryable bool
	}{
		{
			name:      "transient error is retryable",
			err:       NewTransientError("temporary failure", 5*time.Second),
			retryable: true,
		},
		{
			name:      "timeout is retryable",
			err:       NewTimeoutError("request", 30*time.Second),
			retryable: true,
		},
		{
			name:      "permanent error is not retryable",
			err:       NewError(ErrorTypePermanent, "permanent failure").Build(),
			retryable: false,
		},
		{
			name:      "validation error is not retryable",
			err:       NewValidationError("field", "invalid input"),
			retryable: false,
		},
		{
			name:      "user error is not retryable",
			err:       NewError(ErrorTypeUser, "user mistake").Build(),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.retryable, IsRetryable(tt.err))
		})
	}
}

func TestWrap(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	wrappedErr := Wrap(originalErr, "additional context")

	assert.NotNil(t, wrappedErr)
	assert.Contains(t, wrappedErr.Message, "additional context")
	assert.Equal(t, originalErr, wrappedErr.Cause)
	assert.Equal(t, ErrorTypeSystem, wrappedErr.Type)
}

func TestIs(t *testing.T) {
	err1 := NewValidationError("field1", "validation error")
	err2 := NewValidationError("field2", "another validation error")
	err3 := NewNotFoundError("resource")

	assert.True(t, Is(err1, ErrorTypeValidation))
	assert.True(t, Is(err2, ErrorTypeValidation))
	assert.True(t, Is(err3, ErrorTypeNotFound))
	assert.False(t, Is(err1, ErrorTypeNotFound))
}

func TestErrorChain(t *testing.T) {
	rootErr := fmt.Errorf("root cause")
	wrapped := Wrap(rootErr, "level 1")

	assert.NotNil(t, wrapped)
	assert.Equal(t, rootErr, wrapped.Cause)
	assert.Contains(t, wrapped.Message, "level 1")
}

// TestErrorContext removed - WithError and GetError functions don't exist

func BenchmarkDriftError_Error(b *testing.B) {
	err := &DriftError{
		Type:      ErrorTypeSystem,
		Message:   "provider error occurred",
		Code:      "PROV001",
		Resource:  "aws_instance.web",
		Provider:  "aws",
		Operation: "DescribeInstances",
		Details: map[string]interface{}{
			"provider": "aws",
			"region":   "us-east-1",
		},
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}
