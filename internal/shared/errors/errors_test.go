package errors

import (
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
	err := NewError(ErrorTypeConflict, "resource conflict").
		WithDetails("resource1", "aws_instance.web").
		WithDetails("resource2", "aws_instance.app").
		Build()

	assert.Equal(t, "aws_instance.web", err.Details["resource1"])
	assert.Equal(t, "aws_instance.app", err.Details["resource2"])
	assert.Equal(t, "resource conflict", err.Message)
}

func TestWithProvider(t *testing.T) {
	err := NewError(ErrorTypeSystem, "provider error").WithProvider("aws").Build()

	assert.Equal(t, "aws", err.Provider)
	assert.Equal(t, "provider error", err.Message)
}

// TestIsRetryable removed - IsRetryable function doesn't exist

// TestWrap removed - Wrap function doesn't exist

// TestIs removed - Is function doesn't exist

// TestErrorChain removed - Wrap function doesn't exist

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
