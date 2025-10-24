package api

import (
	"net/http"
	"time"

	"github.com/catherinevee/driftmgr/internal/error_handling"
	"github.com/gin-gonic/gin"
)

// ErrorHandlers handles error handling API endpoints
type ErrorHandlers struct {
	errorService        *error_handling.ErrorService
	driftErrorHandler   *error_handling.DriftErrorHandler
	circuitBreakerManager *error_handling.CircuitBreakerManager
}

// NewErrorHandlers creates a new error handlers instance
func NewErrorHandlers(
	errorService *error_handling.ErrorService,
	driftErrorHandler *error_handling.DriftErrorHandler,
	circuitBreakerManager *error_handling.CircuitBreakerManager,
) *ErrorHandlers {
	return &ErrorHandlers{
		errorService:        errorService,
		driftErrorHandler:   driftErrorHandler,
		circuitBreakerManager: circuitBreakerManager,
	}
}

// GetErrorHistory returns the error history
func (eh *ErrorHandlers) GetErrorHistory(c *gin.Context) {
	history := eh.errorService.GetErrorHistory()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"errors": history,
			"total":  len(history),
		},
	})
}

// GetErrorsBySeverity returns errors filtered by severity
func (eh *ErrorHandlers) GetErrorsBySeverity(c *gin.Context) {
	severityStr := c.Param("severity")
	severity := error_handling.ErrorSeverity(severityStr)
	
	// Validate severity
	validSeverities := []error_handling.ErrorSeverity{
		error_handling.ErrorSeverityCritical,
		error_handling.ErrorSeverityHigh,
		error_handling.ErrorSeverityMedium,
		error_handling.ErrorSeverityLow,
		error_handling.ErrorSeverityInfo,
	}
	
	valid := false
	for _, validSeverity := range validSeverities {
		if severity == validSeverity {
			valid = true
			break
		}
	}
	
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid severity level",
			"valid_severities": validSeverities,
		})
		return
	}
	
	errors := eh.errorService.GetErrorsBySeverity(severity)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"errors": errors,
			"total":  len(errors),
			"severity": string(severity),
		},
	})
}

// GetErrorsByCategory returns errors filtered by category
func (eh *ErrorHandlers) GetErrorsByCategory(c *gin.Context) {
	categoryStr := c.Param("category")
	category := error_handling.ErrorCategory(categoryStr)
	
	// Validate category
	validCategories := []error_handling.ErrorCategory{
		error_handling.ErrorCategoryAuthentication,
		error_handling.ErrorCategoryAuthorization,
		error_handling.ErrorCategoryNetwork,
		error_handling.ErrorCategoryTimeout,
		error_handling.ErrorCategoryValidation,
		error_handling.ErrorCategoryConfiguration,
		error_handling.ErrorCategoryResource,
		error_handling.ErrorCategoryProvider,
		error_handling.ErrorCategoryInternal,
		error_handling.ErrorCategoryExternal,
	}
	
	valid := false
	for _, validCategory := range validCategories {
		if category == validCategory {
			valid = true
			break
		}
	}
	
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid error category",
			"valid_categories": validCategories,
		})
		return
	}
	
	errors := eh.errorService.GetErrorsByCategory(category)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"errors": errors,
			"total":  len(errors),
			"category": string(category),
		},
	})
}

// GetErrorStatistics returns error statistics
func (eh *ErrorHandlers) GetErrorStatistics(c *gin.Context) {
	stats := eh.errorService.GetErrorStatistics()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetDriftErrorStatistics returns drift-specific error statistics
func (eh *ErrorHandlers) GetDriftErrorStatistics(c *gin.Context) {
	stats := eh.driftErrorHandler.GetDriftDetectionStatistics()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// ClearErrorHistory clears the error history
func (eh *ErrorHandlers) ClearErrorHistory(c *gin.Context) {
	eh.errorService.ClearErrorHistory()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Error history cleared",
	})
}

// GetCircuitBreakerStatus returns the status of all circuit breakers
func (eh *ErrorHandlers) GetCircuitBreakerStatus(c *gin.Context) {
	stats := eh.circuitBreakerManager.GetStatistics()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetCircuitBreakerStatusByName returns the status of a specific circuit breaker
func (eh *ErrorHandlers) GetCircuitBreakerStatusByName(c *gin.Context) {
	name := c.Param("name")
	
	breaker, exists := eh.circuitBreakerManager.Get(name)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Circuit breaker not found",
			"name":  name,
		})
		return
	}
	
	status := map[string]interface{}{
		"name":          name,
		"state":         string(breaker.GetState()),
		"failure_count": breaker.GetFailureCount(),
		"available":     breaker.IsAvailable(),
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}

// ResetCircuitBreaker resets a specific circuit breaker
func (eh *ErrorHandlers) ResetCircuitBreaker(c *gin.Context) {
	name := c.Param("name")
	
	breaker, exists := eh.circuitBreakerManager.Get(name)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Circuit breaker not found",
			"name":  name,
		})
		return
	}
	
	breaker.Reset()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Circuit breaker reset",
		"name":    name,
	})
}

// ResetAllCircuitBreakers resets all circuit breakers
func (eh *ErrorHandlers) ResetAllCircuitBreakers(c *gin.Context) {
	eh.circuitBreakerManager.ResetAll()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "All circuit breakers reset",
	})
}

// CreateCircuitBreaker creates a new circuit breaker
func (eh *ErrorHandlers) CreateCircuitBreaker(c *gin.Context) {
	var request struct {
		Name             string `json:"name" binding:"required"`
		FailureThreshold int    `json:"failure_threshold" binding:"required"`
		Timeout          int    `json:"timeout" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}
	
	// Validate parameters
	if request.FailureThreshold <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failure threshold must be greater than 0",
		})
		return
	}
	
	if request.Timeout <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Timeout must be greater than 0",
		})
		return
	}
	
	// Check if circuit breaker already exists
	if _, exists := eh.circuitBreakerManager.Get(request.Name); exists {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Circuit breaker already exists",
			"name":  request.Name,
		})
		return
	}
	
	// Create circuit breaker
	config := error_handling.CircuitBreakerConfig{
		Name:             request.Name,
		FailureThreshold: request.FailureThreshold,
		Timeout:          time.Duration(request.Timeout) * time.Second,
	}
	
	breaker := eh.circuitBreakerManager.GetOrCreate(request.Name, config)
	
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Circuit breaker created",
		"data": gin.H{
			"name":              breaker.GetState(),
			"state":             string(breaker.GetState()),
			"failure_count":     breaker.GetFailureCount(),
			"available":         breaker.IsAvailable(),
		},
	})
}

// GetErrorDetails returns detailed information about a specific error
func (eh *ErrorHandlers) GetErrorDetails(c *gin.Context) {
	errorID := c.Param("id")
	
	history := eh.errorService.GetErrorHistory()
	
	for _, err := range history {
		if err.ID == errorID {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    err,
			})
			return
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{
		"error": "Error not found",
		"id":    errorID,
	})
}

// GetErrorsByProvider returns errors filtered by provider
func (eh *ErrorHandlers) GetErrorsByProvider(c *gin.Context) {
	provider := c.Param("provider")
	
	history := eh.errorService.GetErrorHistory()
	var filtered []error_handling.EnhancedError
	
	for _, err := range history {
		if err.Context.Provider == provider {
			filtered = append(filtered, err)
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"errors": filtered,
			"total":  len(filtered),
			"provider": provider,
		},
	})
}

// GetErrorsByTimeRange returns errors within a specific time range
func (eh *ErrorHandlers) GetErrorsByTimeRange(c *gin.Context) {
	// Parse time range parameters
	startStr := c.Query("start")
	endStr := c.Query("end")
	
	if startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Start and end time parameters are required",
		})
		return
	}
	
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid start time format",
			"details": err.Error(),
		})
		return
	}
	
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid end time format",
			"details": err.Error(),
		})
		return
	}
	
	history := eh.errorService.GetErrorHistory()
	var filtered []error_handling.EnhancedError
	
	for _, err := range history {
		if err.Timestamp.After(start) && err.Timestamp.Before(end) {
			filtered = append(filtered, err)
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"errors": filtered,
			"total":  len(filtered),
			"start":  start,
			"end":    end,
		},
	})
}
