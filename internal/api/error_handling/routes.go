package error_handling

import (
	"github.com/catherinevee/driftmgr/internal/api"
	"github.com/gin-gonic/gin"
)

// RegisterErrorHandlingRoutes registers error handling API routes
func RegisterErrorHandlingRoutes(router *gin.RouterGroup, errorHandlers *api.ErrorHandlers) {
	// Error handling routes
	errors := router.Group("/errors")
	{
		// Error history and statistics
		errors.GET("/history", errorHandlers.GetErrorHistory)
		errors.GET("/statistics", errorHandlers.GetErrorStatistics)
		errors.DELETE("/history", errorHandlers.ClearErrorHistory)
		
		// Error filtering
		errors.GET("/severity/:severity", errorHandlers.GetErrorsBySeverity)
		errors.GET("/category/:category", errorHandlers.GetErrorsByCategory)
		errors.GET("/provider/:provider", errorHandlers.GetErrorsByProvider)
		errors.GET("/range", errorHandlers.GetErrorsByTimeRange)
		
		// Error details
		errors.GET("/:id", errorHandlers.GetErrorDetails)
	}
	
	// Drift error handling routes
	drift := router.Group("/drift-errors")
	{
		// Drift error statistics
		drift.GET("/statistics", errorHandlers.GetDriftErrorStatistics)
	}
	
	// Circuit breaker routes
	circuit := router.Group("/circuit-breakers")
	{
		// Circuit breaker management
		circuit.GET("/", errorHandlers.GetCircuitBreakerStatus)
		circuit.POST("/", errorHandlers.CreateCircuitBreaker)
		circuit.DELETE("/reset", errorHandlers.ResetAllCircuitBreakers)
		
		// Individual circuit breaker operations
		circuit.GET("/:name", errorHandlers.GetCircuitBreakerStatusByName)
		circuit.POST("/:name/reset", errorHandlers.ResetCircuitBreaker)
	}
}
