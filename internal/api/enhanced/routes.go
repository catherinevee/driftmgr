package enhanced

import (
	"github.com/catherinevee/driftmgr/internal/api"
	"github.com/gin-gonic/gin"
)

// RegisterEnhancedRoutes registers enhanced features API routes
func RegisterEnhancedRoutes(router *gin.RouterGroup, enhancedHandlers *api.EnhancedHandlers) {
	// Enhanced features routes
	enhanced := router.Group("/enhanced")
	{
		// Resource filtering routes
		filtering := enhanced.Group("/filtering")
		{
			// Filter resources
			filtering.POST("/resources/:provider", enhancedHandlers.FilterResources)
		}

		// Drift summary routes
		summary := enhanced.Group("/summary")
		{
			// Get drift summary for a provider
			summary.GET("/drift/:provider", enhancedHandlers.GetDriftSummary)

			// Get multi-provider drift summary
			summary.POST("/drift/multi-provider", enhancedHandlers.GetMultiProviderDriftSummary)

			// Get drift trends
			summary.GET("/drift/:provider/trends", enhancedHandlers.GetDriftTrends)

			// Get drift statistics
			summary.GET("/drift/:provider/statistics", enhancedHandlers.GetDriftStatistics)
		}

		// Importance filtering routes
		importance := enhanced.Group("/importance")
		{
			// Calculate importance scores
			importance.POST("/scores", enhancedHandlers.CalculateImportanceScores)

			// Filter by importance level
			importance.POST("/filter", enhancedHandlers.FilterByImportance)

			// Get top important resources
			importance.GET("/top/:provider", enhancedHandlers.GetTopImportantResources)

			// Get importance statistics
			importance.GET("/statistics/:provider", enhancedHandlers.GetImportanceStatistics)

			// Get importance distribution
			importance.GET("/distribution/:provider", enhancedHandlers.GetImportanceDistribution)
		}
	}
}
