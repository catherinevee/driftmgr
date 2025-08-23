package errors

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// ErrorType represents the type of error
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "VALIDATION"
	ErrorTypeNotFound     ErrorType = "NOT_FOUND"
	ErrorTypeUnauthorized ErrorType = "UNAUTHORIZED"
	ErrorTypeForbidden    ErrorType = "FORBIDDEN"
	ErrorTypeInternal     ErrorType = "INTERNAL"
	ErrorTypeTimeout      ErrorType = "TIMEOUT"
	ErrorTypeRateLimit    ErrorType = "RATE_LIMIT"
	ErrorTypeProvider     ErrorType = "PROVIDER"
	ErrorTypeNetwork      ErrorType = "NETWORK"
	ErrorTypeConfig       ErrorType = "CONFIG"
	ErrorTypeState        ErrorType = "STATE"
	ErrorTypeDrift        ErrorType = "DRIFT"
)

// Error represents a structured error with context
type Error struct {
	Type       ErrorType              `json:"type"`
	Message    string                 `json:"message"`
	Code       string                 `json:"code,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Cause      error                  `json:"-"`
	StackTrace string                 `json:"stack_trace,omitempty"`
	Retryable  bool                   `json:"retryable"`
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Cause
}

// WithDetails adds details to the error
func (e *Error) WithDetails(key string, value interface{}) *Error {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// New creates a new error with the given type and message
func New(errType ErrorType, message string) *Error {
	return &Error{
		Type:       errType,
		Message:    message,
		StackTrace: getStackTrace(),
		Retryable:  isRetryable(errType),
	}
}

// Newf creates a new error with formatted message
func Newf(errType ErrorType, format string, args ...interface{}) *Error {
	return New(errType, fmt.Sprintf(format, args...))
}

// Wrap wraps an existing error with additional context
func Wrap(err error, errType ErrorType, message string) *Error {
	if err == nil {
		return nil
	}
	
	// If it's already our error type, preserve the original
	if e, ok := err.(*Error); ok {
		e.Message = fmt.Sprintf("%s: %s", message, e.Message)
		return e
	}
	
	return &Error{
		Type:       errType,
		Message:    message,
		Cause:      err,
		StackTrace: getStackTrace(),
		Retryable:  isRetryable(errType),
	}
}

// Wrapf wraps an error with formatted message
func Wrapf(err error, errType ErrorType, format string, args ...interface{}) *Error {
	return Wrap(err, errType, fmt.Sprintf(format, args...))
}

// Is checks if an error is of a specific type
func Is(err error, errType ErrorType) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Type == errType
	}
	return false
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Retryable
	}
	return false
}

// GetType returns the error type if it's a structured error
func GetType(err error) ErrorType {
	var e *Error
	if errors.As(err, &e) {
		return e.Type
	}
	return ErrorTypeInternal
}

// isRetryable determines if an error type is retryable
func isRetryable(errType ErrorType) bool {
	switch errType {
	case ErrorTypeTimeout, ErrorTypeRateLimit, ErrorTypeNetwork:
		return true
	default:
		return false
	}
}

// getStackTrace captures the current stack trace
func getStackTrace() string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	
	var sb strings.Builder
	frames := runtime.CallersFrames(pcs[:n])
	
	for {
		frame, more := frames.Next()
		if !strings.Contains(frame.File, "runtime/") {
			sb.WriteString(fmt.Sprintf("%s:%d %s\n", frame.File, frame.Line, frame.Function))
		}
		if !more {
			break
		}
	}
	
	return sb.String()
}

// ValidationError creates a validation error
func ValidationError(message string) *Error {
	return New(ErrorTypeValidation, message)
}

// NotFoundError creates a not found error
func NotFoundError(resource string) *Error {
	return Newf(ErrorTypeNotFound, "%s not found", resource)
}

// UnauthorizedError creates an unauthorized error
func UnauthorizedError(message string) *Error {
	return New(ErrorTypeUnauthorized, message)
}

// InternalError creates an internal error
func InternalError(message string) *Error {
	return New(ErrorTypeInternal, message)
}

// ConfigError creates a configuration error
func ConfigError(message string) *Error {
	return New(ErrorTypeConfig, message)
}

// ProviderError creates a provider error
func ProviderError(provider, message string) *Error {
	return Newf(ErrorTypeProvider, "provider %s: %s", provider, message)
}