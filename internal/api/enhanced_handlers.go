package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/catherinevee/driftmgr/internal/drift"
	"github.com/catherinevee/driftmgr/internal/filtering"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/gin-gonic/gin"
)

// EnhancedHandlers handles enhanced features API endpoints
type EnhancedHandlers struct {
	resourceFilterService   *filtering.ResourceFilterService
	driftSummaryService     *drift.DriftSummaryService
	importanceFilterService *filtering.ImportanceFilterService
	providerFactory         *providers.ProviderFactory
}

// NewEnhancedHandlers creates a new enhanced handlers instance
func NewEnhancedHandlers(
	resourceFilterService *filtering.ResourceFilterService,
	driftSummaryService *drift.DriftSummaryService,
	importanceFilterService *filtering.ImportanceFilterService,
	providerFactory *providers.ProviderFactory,
) *EnhancedHandlers {
	return &EnhancedHandlers{
		resourceFilterService:   resourceFilterService,
		driftSummaryService:     driftSummaryService,
		importanceFilterService: importanceFilterService,
		providerFactory:         providerFactory,
	}
}

// FilterResources filters resources based on provided criteria
func (eh *EnhancedHandlers) FilterResources(c *gin.Context) {
	var filter filtering.ResourceFilter
	if err := c.ShouldBindJSON(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid filter format",
			"details": err.Error(),
		})
		return
	}

	// Get resources from provider
	provider := c.Param("provider")
	region := c.Query("region")
	if region == "" {
		region = "us-east-1" // Default region
	}

	// Create provider and get resources
	providerInstance, err := eh.providerFactory.CreateProvider(provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create provider",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	resources, err := providerInstance.DiscoverResources(ctx, region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to discover resources",
			"details": err.Error(),
		})
		return
	}

	// Apply filters
	filteredResources, err := eh.resourceFilterService.FilterResources(resources, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to filter resources",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"resources": filteredResources,
			"total":     len(filteredResources),
			"filtered":  len(filteredResources),
			"original":  len(resources),
		},
	})
}

// GetDriftSummary generates and returns a drift summary
func (eh *EnhancedHandlers) GetDriftSummary(c *gin.Context) {
	provider := c.Param("provider")
	region := c.Query("region")
	if region == "" {
		region = "us-east-1" // Default region
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	summary, err := eh.driftSummaryService.GenerateSummary(ctx, provider, region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate drift summary",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}

// GetMultiProviderDriftSummary generates and returns a multi-provider drift summary
func (eh *EnhancedHandlers) GetMultiProviderDriftSummary(c *gin.Context) {
	var request struct {
		Providers []string `json:"providers" binding:"required"`
		Region    string   `json:"region"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if request.Region == "" {
		request.Region = "us-east-1" // Default region
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	summary, err := eh.driftSummaryService.GenerateMultiProviderSummary(ctx, request.Providers, request.Region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate multi-provider drift summary",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}

// GetDriftTrends returns drift trends for a specific time period
func (eh *EnhancedHandlers) GetDriftTrends(c *gin.Context) {
	provider := c.Param("provider")
	region := c.Query("region")
	if region == "" {
		region = "us-east-1" // Default region
	}

	days := 7 // Default to 7 days
	if daysStr := c.Query("days"); daysStr != "" {
		if parsedDays, err := strconv.Atoi(daysStr); err == nil {
			days = parsedDays
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	trends, err := eh.driftSummaryService.GetDriftTrends(ctx, provider, region, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get drift trends",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    trends,
	})
}

// GetDriftStatistics returns detailed drift statistics
func (eh *EnhancedHandlers) GetDriftStatistics(c *gin.Context) {
	provider := c.Param("provider")
	region := c.Query("region")
	if region == "" {
		region = "us-east-1" // Default region
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	stats, err := eh.driftSummaryService.GetDriftStatistics(ctx, provider, region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get drift statistics",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// CalculateImportanceScores calculates importance scores for resources
func (eh *EnhancedHandlers) CalculateImportanceScores(c *gin.Context) {
	var request struct {
		Provider string `json:"provider" binding:"required"`
		Region   string `json:"region"`
		Limit    int    `json:"limit"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if request.Region == "" {
		request.Region = "us-east-1" // Default region
	}

	// Create provider and get resources
	providerInstance, err := eh.providerFactory.CreateProvider(request.Provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create provider",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	resources, err := providerInstance.DiscoverResources(ctx, request.Region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to discover resources",
			"details": err.Error(),
		})
		return
	}

	// For now, we'll use empty drifts since we don't have a drift repository
	// In a real implementation, you would get actual drift records
	var drifts []models.DriftRecord

	// Calculate importance scores
	scores, err := eh.importanceFilterService.CalculateImportanceScores(ctx, resources, drifts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to calculate importance scores",
			"details": err.Error(),
		})
		return
	}

	// Apply limit if specified
	if request.Limit > 0 && request.Limit < len(scores) {
		scores = scores[:request.Limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"scores": scores,
			"total":  len(scores),
		},
	})
}

// FilterByImportance filters resources by importance level
func (eh *EnhancedHandlers) FilterByImportance(c *gin.Context) {
	var request struct {
		Provider string                    `json:"provider" binding:"required"`
		Region   string                    `json:"region"`
		Level    filtering.ImportanceLevel `json:"level" binding:"required"`
		Limit    int                       `json:"limit"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if request.Region == "" {
		request.Region = "us-east-1" // Default region
	}

	// Create provider and get resources
	providerInstance, err := eh.providerFactory.CreateProvider(request.Provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create provider",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	resources, err := providerInstance.DiscoverResources(ctx, request.Region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to discover resources",
			"details": err.Error(),
		})
		return
	}

	// Calculate importance scores
	var drifts []models.DriftRecord
	scores, err := eh.importanceFilterService.CalculateImportanceScores(ctx, resources, drifts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to calculate importance scores",
			"details": err.Error(),
		})
		return
	}

	// Filter by importance level
	filteredScores := eh.importanceFilterService.FilterByImportance(scores, request.Level)

	// Apply limit if specified
	if request.Limit > 0 && request.Limit < len(filteredScores) {
		filteredScores = filteredScores[:request.Limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"scores": filteredScores,
			"total":  len(filteredScores),
			"level":  string(request.Level),
		},
	})
}

// GetTopImportantResources returns the top N most important resources
func (eh *EnhancedHandlers) GetTopImportantResources(c *gin.Context) {
	provider := c.Param("provider")
	region := c.Query("region")
	if region == "" {
		region = "us-east-1" // Default region
	}

	limit := 10 // Default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	// Create provider and get resources
	providerInstance, err := eh.providerFactory.CreateProvider(provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create provider",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	resources, err := providerInstance.DiscoverResources(ctx, region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to discover resources",
			"details": err.Error(),
		})
		return
	}

	// Calculate importance scores
	var drifts []models.DriftRecord
	scores, err := eh.importanceFilterService.CalculateImportanceScores(ctx, resources, drifts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to calculate importance scores",
			"details": err.Error(),
		})
		return
	}

	// Get top important resources
	topResources := eh.importanceFilterService.GetTopImportantResources(scores, limit)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"resources": topResources,
			"total":     len(topResources),
			"limit":     limit,
		},
	})
}

// GetImportanceStatistics returns statistics about importance scores
func (eh *EnhancedHandlers) GetImportanceStatistics(c *gin.Context) {
	provider := c.Param("provider")
	region := c.Query("region")
	if region == "" {
		region = "us-east-1" // Default region
	}

	// Create provider and get resources
	providerInstance, err := eh.providerFactory.CreateProvider(provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create provider",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	resources, err := providerInstance.DiscoverResources(ctx, region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to discover resources",
			"details": err.Error(),
		})
		return
	}

	// Calculate importance scores
	var drifts []models.DriftRecord
	scores, err := eh.importanceFilterService.CalculateImportanceScores(ctx, resources, drifts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to calculate importance scores",
			"details": err.Error(),
		})
		return
	}

	// Get statistics
	stats := eh.importanceFilterService.GetImportanceStatistics(scores)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetImportanceDistribution returns the distribution of importance levels
func (eh *EnhancedHandlers) GetImportanceDistribution(c *gin.Context) {
	provider := c.Param("provider")
	region := c.Query("region")
	if region == "" {
		region = "us-east-1" // Default region
	}

	// Create provider and get resources
	providerInstance, err := eh.providerFactory.CreateProvider(provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create provider",
			"details": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	resources, err := providerInstance.DiscoverResources(ctx, region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to discover resources",
			"details": err.Error(),
		})
		return
	}

	// Calculate importance scores
	var drifts []models.DriftRecord
	scores, err := eh.importanceFilterService.CalculateImportanceScores(ctx, resources, drifts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to calculate importance scores",
			"details": err.Error(),
		})
		return
	}

	// Get distribution
	distribution := eh.importanceFilterService.GetImportanceDistribution(scores)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    distribution,
	})
}
