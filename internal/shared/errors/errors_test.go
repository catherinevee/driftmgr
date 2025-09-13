package errors

import (
	"fmt"
	"testing"

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

func TestNewDriftError(t *testing.T) {
	err := NewDriftError(ErrorTypeSystem, "system error occurred")

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeSystem, err.Type)
	assert.Equal(t, "system error occurred", err.Message)
	assert.NotZero(t, err.Timestamp)
	assert.NotEmpty(t, err.TraceID)
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("invalid input", map[string]interface{}{
		"field": "username",
		"value": "admin123",
	})

	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeValidation, err.Type)
	assert.Contains(t, err.Message, "invalid input")
	assert.Equal(t, "username", err.Details["field"])
}

func TestWithCode(t *testing.T) {
	err := NewDriftError(ErrorTypeSystem, "error")
	errWithCode := err.WithCode("SYS001")

	assert.Equal(t, "SYS001", errWithCode.Code)
	assert.Equal(t, err.Message, errWithCode.Message)
}

func TestWithSeverity(t *testing.T) {
	err := NewDriftError(ErrorTypeSystem, "error")
	errWithSeverity := err.WithSeverity(SeverityCritical)

	assert.Equal(t, SeverityCritical, errWithSeverity.Severity)
	assert.Equal(t, err.Message, errWithSeverity.Message)
}

func TestWithResource(t *testing.T) {
	err := NewDriftError(ErrorTypeNotFound, "not found")
	errWithResource := err.WithResource("aws_instance.web")

	assert.Equal(t, "aws_instance.web", errWithResource.Resource)
	assert.Equal(t, err.Message, errWithResource.Message)
}

func TestWithDetails(t *testing.T) {
	err := NewDriftError(ErrorTypeConflict, "resource conflict")
	details := map[string]interface{}{
		"resource1": "aws_instance.web",
		"resource2": "aws_instance.app",
	}
	errWithDetails := err.WithDetails(details)

	assert.Equal(t, details, errWithDetails.Details)
	assert.Equal(t, err.Message, errWithDetails.Message)
}

func TestWithProvider(t *testing.T) {
	err := NewDriftError(ErrorTypeSystem, "provider error")
	errWithProvider := err.WithProvider("aws")

	assert.Equal(t, "aws", errWithProvider.Provider)
	assert.Equal(t, err.Message, errWithProvider.Message)
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       *DriftError
		retryable bool
	}{
		{
			name:      "transient error is retryable",
			err:       NewDriftError(ErrorTypeTransient, "temporary failure"),
			retryable: true,
		},
		{
			name:      "timeout is retryable",
			err:       NewDriftError(ErrorTypeTimeout, "request timeout"),
			retryable: true,
		},
		{
			name:      "permanent error is not retryable",
			err:       NewDriftError(ErrorTypePermanent, "permanent failure"),
			retryable: false,
		},
		{
			name:      "validation error is not retryable",
			err:       NewDriftError(ErrorTypeValidation, "invalid input"),
			retryable: false,
		},
		{
			name:      "user error is not retryable",
			err:       NewDriftError(ErrorTypeUser, "user mistake"),
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
	err1 := NewDriftError(ErrorTypeValidation, "validation error")
	err2 := NewDriftError(ErrorTypeValidation, "another validation error")
	err3 := NewDriftError(ErrorTypeNotFound, "not found")

	assert.True(t, Is(err1, ErrorTypeValidation))
	assert.True(t, Is(err2, ErrorTypeValidation))
	assert.True(t, Is(err3, ErrorTypeNotFound))
	assert.False(t, Is(err1, ErrorTypeNotFound))
}

func TestErrorChain(t *testing.T) {
	rootErr := fmt.Errorf("root cause")
	level1 := Wrap(rootErr, "level 1")
	level2 := level1.WithOperation("DescribeInstances")
	level3 := level2.WithDetails(map[string]interface{}{"key": "value"})

	assert.Equal(t, rootErr, level3.Cause)
	assert.Contains(t, level3.Message, "level 1")
	assert.Equal(t, "DescribeInstances", level3.Operation)
	assert.Equal(t, "value", level3.Details["key"])
}

func TestErrorContext(t *testing.T) {
	ctx := context.Background()
	err := NewDriftError(ErrorTypeSystem, "test error")

	// Add error to context
	ctxWithErr := WithError(ctx, err)

	// Retrieve error from context
	retrieved := GetError(ctxWithErr)
	assert.NotNil(t, retrieved)
	assert.Equal(t, err.Message, retrieved.Message)

	// Empty context should return nil
	emptyErr := GetError(context.Background())
	assert.Nil(t, emptyErr)
}

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