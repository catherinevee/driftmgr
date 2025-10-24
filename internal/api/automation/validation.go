package automation

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
		"error":   "validation_failed",
		"message": "Request validation failed",
		"details": err.Error(),
	}

	json.NewEncoder(w).Encode(response)
}

// WriteErrorResponse writes an error response
func WriteErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error":   http.StatusText(statusCode),
		"message": message,
	}

	json.NewEncoder(w).Encode(response)
}

// WriteJSONResponse writes a JSON response
func WriteJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// ValidateStruct validates a struct using go-playground/validator
func ValidateStruct(s interface{}) error {
	validate := validator.New()
	return validate.Struct(s)
}
