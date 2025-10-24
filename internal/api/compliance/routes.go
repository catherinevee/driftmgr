package api

import (
	"github.com/gin-gonic/gin"
)

// SetupComplianceRoutes sets up compliance-related routes
func SetupComplianceRoutes(router *gin.Engine, complianceHandlers *ComplianceHandlers) {
	// Compliance API group
	compliance := router.Group("/api/v1/compliance")
	{
		// Policy management routes
		policies := compliance.Group("/policies")
		{
			policies.GET("", complianceHandlers.ListPolicies)
			policies.GET("/:id", complianceHandlers.GetPolicy)
			policies.POST("", complianceHandlers.CreatePolicy)
			policies.PUT("/:id", complianceHandlers.UpdatePolicy)
			policies.DELETE("/:id", complianceHandlers.DeletePolicy)
			policies.POST("/:id/evaluate", complianceHandlers.EvaluatePolicy)
		}

		// Report management routes
		reports := compliance.Group("/reports")
		{
			reports.POST("", complianceHandlers.GenerateReport)
			reports.GET("/:id/export", complianceHandlers.ExportReport)
		}
	}
}
