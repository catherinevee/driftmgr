package detector

import (
	"context"
	"errors"
	"testing"

	errorspkg "github.com/catherinevee/driftmgr/internal/shared/errors"
	"github.com/catherinevee/driftmgr/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnhancedDetector_ErrorHandling tests various error scenarios in the EnhancedDetector
func TestEnhancedDetector_ErrorHandling(t *testing.T) {
	t.Run("ProviderNotFound", func(t *testing.T) {
		detector := NewEnhancedDetector()
		resource := state.Resource{
			ID:       "test-resource",
			Type:     "test_type",
			Provider: "nonexistent-provider",
		}

		err := detector.processResource(context.Background(), resource, &DriftReport{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider not found")
	})

	t.Run("EnrichError", func(t *testing.T) {
		detector := NewEnhancedDetector()
		ctx := context.WithValue(context.Background(), "trace_id", "test-trace-123")
		resource := state.Resource{
			ID:       "test-resource",
			Type:     "test_type",
			Provider: "test",
		}

		err := errors.New("test error")
		enrichedErr := detector.enrichError(ctx, err, resource)
		require.NotNil(t, enrichedErr)
		assert.Equal(t, resource.ID, enrichedErr.Resource)
	})

	t.Run("ErrorHandlers", func(t *testing.T) {
		detector := NewEnhancedDetector()

		tests := []struct {
			name          string
			err           *errorspkg.DriftError
			handler       func(*errorspkg.DriftError) error
			expectedError string
			expectedType  errorspkg.ErrorType
		}{
			{
				name: "TransientError",
				err: errorspkg.NewError(errorspkg.ErrorTypeTransient, "connection timeout").
					WithSeverity(errorspkg.SeverityHigh).
					Build(),
				handler:       detector.handleTransientError,
				expectedError: "connection timeout",
				expectedType:  errorspkg.ErrorTypeTransient,
			},
			{
				name: "ValidationError",
				err: errorspkg.NewError(errorspkg.ErrorTypeValidation, "invalid configuration").
					WithSeverity(errorspkg.SeverityMedium).
					Build(),
				handler:       detector.handleValidationError,
				expectedError: "invalid configuration",
				expectedType:  errorspkg.ErrorTypeValidation,
			},
			{
				name: "NotFoundError",
				err: errorspkg.NewError(errorspkg.ErrorTypeNotFound, "resource not found").
					WithSeverity(errorspkg.SeverityLow).
					Build(),
				handler:       detector.handleNotFoundError,
				expectedError: "resource not found",
				expectedType:  errorspkg.ErrorTypeNotFound,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				handledErr := tc.handler(tc.err)
				require.Error(t, handledErr)

				// The handlers return different error types, but they should all implement error
				assert.Contains(t, handledErr.Error(), tc.expectedError)
			})
		}
	})
}
