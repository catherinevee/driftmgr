package api

import (
	"context"
	"net/http"
	"time"

	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/gin-gonic/gin"
)

// ConnectionHandlers handles connection testing API endpoints
type ConnectionHandlers struct {
	connectionService *providers.ConnectionService
}

// NewConnectionHandlers creates a new connection handlers instance
func NewConnectionHandlers(connectionService *providers.ConnectionService) *ConnectionHandlers {
	return &ConnectionHandlers{
		connectionService: connectionService,
	}
}

// TestProviderConnection tests connection to a specific provider
func (ch *ConnectionHandlers) TestProviderConnection(c *gin.Context) {
	provider := c.Param("provider")
	region := c.Query("region")

	if region == "" {
		region = "us-east-1" // Default region
	}

	// Set timeout from query parameter
	timeout := 30 * time.Second
	if timeoutStr := c.Query("timeout"); timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	result, err := ch.connectionService.TestProviderConnection(ctx, provider, region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Connection test failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

// TestProviderService tests connection to a specific service
func (ch *ConnectionHandlers) TestProviderService(c *gin.Context) {
	provider := c.Param("provider")
	service := c.Param("service")
	region := c.Query("region")

	if region == "" {
		region = "us-east-1" // Default region
	}

	// Set timeout from query parameter
	timeout := 30 * time.Second
	if timeoutStr := c.Query("timeout"); timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	result, err := ch.connectionService.TestProviderService(ctx, provider, region, service)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Service connection test failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

// TestAllProviders tests connection to all providers
func (ch *ConnectionHandlers) TestAllProviders(c *gin.Context) {
	region := c.Query("region")

	if region == "" {
		region = "us-east-1" // Default region
	}

	// Set timeout from query parameter
	timeout := 60 * time.Second
	if timeoutStr := c.Query("timeout"); timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	results, err := ch.connectionService.TestAllProviders(ctx, region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "All providers test failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"results": results,
	})
}

// TestProviderAllRegions tests connection to all regions of a provider
func (ch *ConnectionHandlers) TestProviderAllRegions(c *gin.Context) {
	provider := c.Param("provider")

	// Set timeout from query parameter
	timeout := 120 * time.Second
	if timeoutStr := c.Query("timeout"); timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	results, err := ch.connectionService.TestProviderAllRegions(ctx, provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "All regions test failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"results": results,
	})
}

// TestProviderAllServices tests connection to all services of a provider
func (ch *ConnectionHandlers) TestProviderAllServices(c *gin.Context) {
	provider := c.Param("provider")
	region := c.Query("region")

	if region == "" {
		region = "us-east-1" // Default region
	}

	// Set timeout from query parameter
	timeout := 60 * time.Second
	if timeoutStr := c.Query("timeout"); timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	results, err := ch.connectionService.TestProviderAllServices(ctx, provider, region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "All services test failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"results": results,
	})
}

// GetConnectionResults returns stored connection test results
func (ch *ConnectionHandlers) GetConnectionResults(c *gin.Context) {
	provider := c.Param("provider")

	var results interface{}
	if provider == "" {
		// Return all results
		results = ch.connectionService.GetAllConnectionResults()
	} else {
		// Return specific provider results
		results = ch.connectionService.GetConnectionResults(provider)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"results": results,
	})
}

// GetConnectionSummary returns a summary of connection test results
func (ch *ConnectionHandlers) GetConnectionSummary(c *gin.Context) {
	summary := ch.connectionService.GetConnectionSummary()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"summary": summary,
	})
}

// RunHealthCheck runs a comprehensive health check
func (ch *ConnectionHandlers) RunHealthCheck(c *gin.Context) {
	region := c.Query("region")

	if region == "" {
		region = "us-east-1" // Default region
	}

	// Set timeout from query parameter
	timeout := 120 * time.Second
	if timeoutStr := c.Query("timeout"); timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	result, err := ch.connectionService.RunHealthCheck(ctx, region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Health check failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

// ClearConnectionResults clears stored connection test results
func (ch *ConnectionHandlers) ClearConnectionResults(c *gin.Context) {
	provider := c.Param("provider")

	ch.connectionService.ClearConnectionResults(provider)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Connection results cleared",
	})
}
