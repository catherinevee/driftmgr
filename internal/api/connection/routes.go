package connection

import (
	"github.com/catherinevee/driftmgr/internal/api"
	"github.com/gin-gonic/gin"
)

// RegisterConnectionRoutes registers connection testing API routes
func RegisterConnectionRoutes(router *gin.RouterGroup, connectionHandlers *api.ConnectionHandlers) {
	// Connection testing routes
	connection := router.Group("/connection")
	{
		// Test specific provider connection
		connection.GET("/test/:provider", connectionHandlers.TestProviderConnection)

		// Test specific service connection
		connection.GET("/test/:provider/service/:service", connectionHandlers.TestProviderService)

		// Test all providers
		connection.GET("/test/all", connectionHandlers.TestAllProviders)

		// Test all regions for a provider
		connection.GET("/test/:provider/regions", connectionHandlers.TestProviderAllRegions)

		// Test all services for a provider
		connection.GET("/test/:provider/services", connectionHandlers.TestProviderAllServices)

		// Get connection test results
		connection.GET("/results", connectionHandlers.GetConnectionResults)
		connection.GET("/results/:provider", connectionHandlers.GetConnectionResults)

		// Get connection summary
		connection.GET("/summary", connectionHandlers.GetConnectionSummary)

		// Run health check
		connection.POST("/health-check", connectionHandlers.RunHealthCheck)

		// Clear connection results
		connection.DELETE("/results", connectionHandlers.ClearConnectionResults)
		connection.DELETE("/results/:provider", connectionHandlers.ClearConnectionResults)
	}
}
