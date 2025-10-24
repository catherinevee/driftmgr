package discovery

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// WriteValidationError writes a validation error response
func WriteValidationError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	response := map[string]interface{}{
		"error":   "validation_error",
		"message": "Request validation failed",
		"details": err.Error(),
	}

	json.NewEncoder(w).Encode(response)
}

// WriteErrorResponse writes an error response
func WriteErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error":   http.StatusText(statusCode),
		"message": message,
	}

	if err != nil {
		response["details"] = err.Error()
	}

	json.NewEncoder(w).Encode(response)
}

// WriteJSONResponse writes a JSON response
func WriteJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// ValidateRequest validates a request using the validator
func ValidateRequest(v *validator.Validate, req interface{}) error {
	return v.Struct(req)
}

// NewValidator creates a new validator instance
func NewValidator() *validator.Validate {
	return validator.New()
}
