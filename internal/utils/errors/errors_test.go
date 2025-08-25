package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		errType  ErrorType
		message  string
		expected string
	}{
		{
			name:     "validation error",
			errType:  ErrorTypeValidation,
			message:  "invalid input",
			expected: "VALIDATION: invalid input",
		},
		{
			name:     "not found error",
			errType:  ErrorTypeNotFound,
			message:  "resource not found",
			expected: "NOT_FOUND: resource not found",
		},
		{
			name:     "internal error",
			errType:  ErrorTypeInternal,
			message:  "something went wrong",
			expected: "INTERNAL: something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.errType, tt.message)

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if err.Type != tt.errType {
				t.Errorf("expected type %v, got %v", tt.errType, err.Type)
			}

			if err.Message != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, err.Message)
			}

			if !strings.Contains(err.Error(), tt.expected) {
				t.Errorf("expected error string to contain %q, got %q", tt.expected, err.Error())
			}

			if err.StackTrace == "" {
				t.Error("expected stack trace to be captured")
			}
		})
	}
}

func TestNewf(t *testing.T) {
	err := Newf(ErrorTypeValidation, "field %s is required", "username")

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expectedMessage := "field username is required"
	if err.Message != expectedMessage {
		t.Errorf("expected message %q, got %q", expectedMessage, err.Message)
	}
}

func TestWrap(t *testing.T) {
	originalErr := errors.New("original error")

	tests := []struct {
		name          string
		err           error
		errType       ErrorType
		message       string
		expectNil     bool
		expectedCause error
	}{
		{
			name:          "wrap standard error",
			err:           originalErr,
			errType:       ErrorTypeInternal,
			message:       "wrapped error",
			expectNil:     false,
			expectedCause: originalErr,
		},
		{
			name:      "wrap nil error",
			err:       nil,
			errType:   ErrorTypeInternal,
			message:   "wrapped error",
			expectNil: true,
		},
		{
			name:          "wrap custom error",
			err:           New(ErrorTypeValidation, "validation failed"),
			errType:       ErrorTypeInternal,
			message:       "wrapped validation",
			expectNil:     false,
			expectedCause: nil, // Custom errors preserve their type
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := Wrap(tt.err, tt.errType, tt.message)

			if tt.expectNil {
				if wrapped != nil {
					t.Errorf("expected nil, got %v", wrapped)
				}
				return
			}

			if wrapped == nil {
				t.Fatal("expected error, got nil")
			}

			if tt.expectedCause != nil && wrapped.Cause != tt.expectedCause {
				t.Errorf("expected cause %v, got %v", tt.expectedCause, wrapped.Cause)
			}

			if !strings.Contains(wrapped.Error(), tt.message) {
				t.Errorf("expected error to contain %q, got %q", tt.message, wrapped.Error())
			}
		})
	}
}

func TestWrapf(t *testing.T) {
	originalErr := errors.New("database connection failed")
	wrapped := Wrapf(originalErr, ErrorTypeInternal, "failed to connect to %s", "PostgreSQL")

	if wrapped == nil {
		t.Fatal("expected error, got nil")
	}

	expectedMessage := "failed to connect to PostgreSQL"
	if wrapped.Message != expectedMessage {
		t.Errorf("expected message %q, got %q", expectedMessage, wrapped.Message)
	}

	if wrapped.Cause != originalErr {
		t.Errorf("expected cause to be original error")
	}
}

func TestIs(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		errType  ErrorType
		expected bool
	}{
		{
			name:     "matching error type",
			err:      New(ErrorTypeValidation, "test"),
			errType:  ErrorTypeValidation,
			expected: true,
		},
		{
			name:     "non-matching error type",
			err:      New(ErrorTypeValidation, "test"),
			errType:  ErrorTypeInternal,
			expected: false,
		},
		{
			name:     "standard error",
			err:      errors.New("standard error"),
			errType:  ErrorTypeInternal,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			errType:  ErrorTypeInternal,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Is(tt.err, tt.errType)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "timeout error",
			err:      New(ErrorTypeTimeout, "request timeout"),
			expected: true,
		},
		{
			name:     "rate limit error",
			err:      New(ErrorTypeRateLimit, "rate limit exceeded"),
			expected: true,
		},
		{
			name:     "network error",
			err:      New(ErrorTypeNetwork, "connection refused"),
			expected: true,
		},
		{
			name:     "validation error",
			err:      New(ErrorTypeValidation, "invalid input"),
			expected: false,
		},
		{
			name:     "standard error",
			err:      errors.New("standard error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetType(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorType
	}{
		{
			name:     "custom error",
			err:      New(ErrorTypeValidation, "test"),
			expected: ErrorTypeValidation,
		},
		{
			name:     "standard error",
			err:      errors.New("standard error"),
			expected: ErrorTypeInternal,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: ErrorTypeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetType(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestWithDetails(t *testing.T) {
	err := New(ErrorTypeValidation, "validation failed")

	err.WithDetails("field", "username").
		WithDetails("value", "invalid@user").
		WithDetails("reason", "contains invalid characters")

	if err.Details == nil {
		t.Fatal("expected details to be set")
	}

	if err.Details["field"] != "username" {
		t.Errorf("expected field to be 'username', got %v", err.Details["field"])
	}

	if err.Details["value"] != "invalid@user" {
		t.Errorf("expected value to be 'invalid@user', got %v", err.Details["value"])
	}

	if err.Details["reason"] != "contains invalid characters" {
		t.Errorf("expected reason to be 'contains invalid characters', got %v", err.Details["reason"])
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("ValidationError", func(t *testing.T) {
		err := ValidationError("invalid input")
		if err.Type != ErrorTypeValidation {
			t.Errorf("expected validation error type, got %v", err.Type)
		}
		if err.Message != "invalid input" {
			t.Errorf("expected message 'invalid input', got %q", err.Message)
		}
	})

	t.Run("NotFoundError", func(t *testing.T) {
		err := NotFoundError("user")
		if err.Type != ErrorTypeNotFound {
			t.Errorf("expected not found error type, got %v", err.Type)
		}
		if err.Message != "user not found" {
			t.Errorf("expected message 'user not found', got %q", err.Message)
		}
	})

	t.Run("UnauthorizedError", func(t *testing.T) {
		err := UnauthorizedError("invalid token")
		if err.Type != ErrorTypeUnauthorized {
			t.Errorf("expected unauthorized error type, got %v", err.Type)
		}
		if err.Message != "invalid token" {
			t.Errorf("expected message 'invalid token', got %q", err.Message)
		}
	})

	t.Run("InternalError", func(t *testing.T) {
		err := InternalError("server error")
		if err.Type != ErrorTypeInternal {
			t.Errorf("expected internal error type, got %v", err.Type)
		}
		if err.Message != "server error" {
			t.Errorf("expected message 'server error', got %q", err.Message)
		}
	})

	t.Run("ConfigError", func(t *testing.T) {
		err := ConfigError("missing configuration")
		if err.Type != ErrorTypeConfig {
			t.Errorf("expected config error type, got %v", err.Type)
		}
		if err.Message != "missing configuration" {
			t.Errorf("expected message 'missing configuration', got %q", err.Message)
		}
	})

	t.Run("ProviderError", func(t *testing.T) {
		err := ProviderError("AWS", "authentication failed")
		if err.Type != ErrorTypeProvider {
			t.Errorf("expected provider error type, got %v", err.Type)
		}
		expectedMessage := "provider AWS: authentication failed"
		if err.Message != expectedMessage {
			t.Errorf("expected message %q, got %q", expectedMessage, err.Message)
		}
	})
}

func TestUnwrap(t *testing.T) {
	originalErr := errors.New("original error")
	wrapped := Wrap(originalErr, ErrorTypeInternal, "wrapped")

	unwrapped := wrapped.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("expected unwrapped error to be original error")
	}

	// Test unwrap on error without cause
	err := New(ErrorTypeValidation, "test")
	if err.Unwrap() != nil {
		t.Error("expected nil when unwrapping error without cause")
	}
}

func TestStackTrace(t *testing.T) {
	err := New(ErrorTypeInternal, "test error")

	if err.StackTrace == "" {
		t.Error("expected stack trace to be captured")
	}

	// Stack trace should contain this test function
	if !strings.Contains(err.StackTrace, "TestStackTrace") {
		t.Error("expected stack trace to contain test function name")
	}
}

// Benchmarks

func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = New(ErrorTypeInternal, "benchmark error")
	}
}

func BenchmarkWrap(b *testing.B) {
	originalErr := errors.New("original error")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Wrap(originalErr, ErrorTypeInternal, "wrapped error")
	}
}

func BenchmarkIs(b *testing.B) {
	err := New(ErrorTypeValidation, "test")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Is(err, ErrorTypeValidation)
	}
}
