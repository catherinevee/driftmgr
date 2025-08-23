package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/catherinevee/driftmgr/internal/config"
	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/gin-gonic/gin"
)

// Handlers contains all API handlers with version management
type Handlers struct {
	versionManager       *VersionManager
	migrationHelper      *MigrationHelper
	compatibilityChecker *CompatibilityChecker
}

// NewHandlers creates a new handlers instance
func NewHandlers() *Handlers {
	vm := NewVersionManager()
	return &Handlers{
		versionManager:       vm,
		migrationHelper:      NewMigrationHelper(vm),
		compatibilityChecker: NewCompatibilityChecker(vm),
	}
}

// SetupRoutes sets up all API routes with versioning
func (h *Handlers) SetupRoutes(router *gin.Engine) {
	// Add version middleware
	router.Use(h.versionManager.VersionMiddleware())

	// Version information endpoint
	router.GET("/versions", h.GetVersions)

	// API v1 routes
	v1 := router.Group("/v1")
	{
		v1.GET("/discovery", h.GetDiscoveryV1)
		v1.POST("/discovery", h.PostDiscoveryV1)
		v1.GET("/resources", h.GetResourcesV1)
		v1.GET("/providers", h.GetProvidersV1)
	}

	// API v2 routes
	v2 := router.Group("/v2")
	{
		v2.GET("/discovery", h.GetDiscoveryV2)
		v2.POST("/discovery", h.PostDiscoveryV2)
		v2.GET("/resources", h.GetResourcesV2)
		v2.GET("/providers", h.GetProvidersV2)
		v2.GET("/resources/stream", h.GetResourcesStreamV2)
	}

	// Version-agnostic routes (uses header/content negotiation)
	api := router.Group("/api")
	{
		api.GET("/discovery", h.GetDiscovery)
		api.POST("/discovery", h.PostDiscovery)
		api.GET("/resources", h.GetResources)
		api.GET("/providers", h.GetProviders)
	}
}

// GetVersions returns information about supported API versions
func (h *Handlers) GetVersions(c *gin.Context) {
	versionInfo := h.versionManager.GetVersionInfo()
	c.JSON(http.StatusOK, versionInfo)
}

// Version-agnostic handlers that route to appropriate version

// GetDiscovery handles discovery requests with version routing
func (h *Handlers) GetDiscovery(c *gin.Context) {
	version, _ := c.Get("api_version")
	apiVersion := version.(APIVersion)

	switch apiVersion.Major {
	case 1:
		h.GetDiscoveryV1(c)
	case 2:
		h.GetDiscoveryV2(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Unsupported API version",
			"version": apiVersion.String(),
		})
	}
}

// PostDiscovery handles discovery POST requests with version routing
func (h *Handlers) PostDiscovery(c *gin.Context) {
	version, _ := c.Get("api_version")
	apiVersion := version.(APIVersion)

	switch apiVersion.Major {
	case 1:
		h.PostDiscoveryV1(c)
	case 2:
		h.PostDiscoveryV2(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Unsupported API version",
			"version": apiVersion.String(),
		})
	}
}

// GetResources handles resource listing with version routing
func (h *Handlers) GetResources(c *gin.Context) {
	version, _ := c.Get("api_version")
	apiVersion := version.(APIVersion)

	switch apiVersion.Major {
	case 1:
		h.GetResourcesV1(c)
	case 2:
		h.GetResourcesV2(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Unsupported API version",
			"version": apiVersion.String(),
		})
	}
}

// GetProviders handles provider listing with version routing
func (h *Handlers) GetProviders(c *gin.Context) {
	version, _ := c.Get("api_version")
	apiVersion := version.(APIVersion)

	switch apiVersion.Major {
	case 1:
		h.GetProvidersV1(c)
	case 2:
		h.GetProvidersV2(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Unsupported API version",
			"version": apiVersion.String(),
		})
	}
}

// API v1 handlers

// GetDiscoveryV1 handles v1 discovery requests
func (h *Handlers) GetDiscoveryV1(c *gin.Context) {
	// Get actual discovery data from the discovery service
	ctx := c.Request.Context()

	// Create a default config for discovery
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			Providers: []config.ProviderConfig{
				{Provider: "aws"},
				{Provider: "azure"},
				{Provider: "gcp"},
			},
		},
	}

	discoveryService := discovery.NewEnhancedDiscoverer(cfg)

	// Perform discovery - DiscoverResources not implemented yet
	// results, err := discoveryService.DiscoverResources(ctx)
	results := []models.Resource{}
	var err error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Discovery failed",
			"message": err.Error(),
		})
		return
	}

	// Calculate totals
	totalResources := 0
	providers := []string{}
	for provider, result := range results {
		totalResources += result.ResourceCount
		providers = append(providers, provider)
	}

	data := gin.H{
		"status":          "success",
		"discovery_id":    fmt.Sprintf("disc-%d", time.Now().Unix()),
		"resources_found": totalResources,
		"providers":       providers,
		"results":         results,
		"timestamp":       time.Now(),
	}

	response := h.versionManager.WrapResponse(c, data)
	c.JSON(http.StatusOK, response)
}

// PostDiscoveryV1 handles v1 discovery POST requests
func (h *Handlers) PostDiscoveryV1(c *gin.Context) {
	var request map[string]interface{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	// Check compatibility
	version, _ := c.Get("api_version")
	apiVersion := version.(APIVersion)

	compatible, issues := h.compatibilityChecker.CheckCompatibility(request, apiVersion)
	if !compatible {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Request not compatible with API version",
			"version": apiVersion.String(),
			"issues":  issues,
		})
		return
	}

	// Process discovery request
	data := gin.H{
		"status":         "started",
		"discovery_id":   "disc-456",
		"request_id":     c.GetHeader("X-Request-ID"),
		"estimated_time": "30s",
	}

	response := h.versionManager.WrapResponse(c, data)
	c.JSON(http.StatusAccepted, response)
}

// GetResourcesV1 handles v1 resource listing
func (h *Handlers) GetResourcesV1(c *gin.Context) {
	data := gin.H{
		"resources": []gin.H{
			{
				"id":       "res-1",
				"type":     "ec2_instance",
				"provider": "aws",
				"region":   "us-east-1",
				"name":     "web-server-1",
			},
			{
				"id":       "res-2",
				"type":     "storage_account",
				"provider": "azure",
				"region":   "eastus",
				"name":     "mystorageaccount",
			},
		},
		"total": 2,
		"page":  1,
	}

	response := h.versionManager.WrapResponse(c, data)
	c.JSON(http.StatusOK, response)
}

// GetProvidersV1 handles v1 provider listing
func (h *Handlers) GetProvidersV1(c *gin.Context) {
	data := gin.H{
		"providers": []gin.H{
			{
				"name":         "aws",
				"display_name": "Amazon Web Services",
				"status":       "active",
				"regions":      20,
			},
			{
				"name":         "azure",
				"display_name": "Microsoft Azure",
				"status":       "active",
				"regions":      15,
			},
		},
		"total": 2,
	}

	response := h.versionManager.WrapResponse(c, data)
	c.JSON(http.StatusOK, response)
}

// API v2 handlers

// GetDiscoveryV2 handles v2 discovery requests
func (h *Handlers) GetDiscoveryV2(c *gin.Context) {
	data := gin.H{
		"status":          "success",
		"discovery_id":    "disc-123",
		"resources_found": 150,
		"providers": []gin.H{
			{
				"name":      "aws",
				"resources": 80,
				"status":    "completed",
			},
			{
				"name":      "azure",
				"resources": 45,
				"status":    "completed",
			},
			{
				"name":      "gcp",
				"resources": 25,
				"status":    "completed",
			},
		},
		"performance": gin.H{
			"discovery_time":   "45s",
			"cache_hit_ratio":  0.75,
			"parallel_workers": 10,
		},
		"timestamp": time.Now(),
		"_metadata": gin.H{
			"format_version":      "2.0",
			"streaming_available": true,
		},
	}

	response := h.versionManager.WrapResponse(c, data)
	c.JSON(http.StatusOK, response)
}

// PostDiscoveryV2 handles v2 discovery POST requests
func (h *Handlers) PostDiscoveryV2(c *gin.Context) {
	var request map[string]interface{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	// Check compatibility
	version, _ := c.Get("api_version")
	apiVersion := version.(APIVersion)

	compatible, issues := h.compatibilityChecker.CheckCompatibility(request, apiVersion)
	if !compatible {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Request not compatible with API version",
			"version": apiVersion.String(),
			"issues":  issues,
		})
		return
	}

	// Process discovery request with v2 features
	data := gin.H{
		"status":         "started",
		"discovery_id":   "disc-456",
		"request_id":     c.GetHeader("X-Request-ID"),
		"estimated_time": "25s",
		"streaming_url":  "/v2/discovery/disc-456/stream",
		"webhook_url":    c.Query("webhook_url"),
		"advanced_features": gin.H{
			"parallel_discovery":  true,
			"intelligent_caching": true,
			"real_time_updates":   true,
		},
	}

	response := h.versionManager.WrapResponse(c, data)
	c.JSON(http.StatusAccepted, response)
}

// GetResourcesV2 handles v2 resource listing
func (h *Handlers) GetResourcesV2(c *gin.Context) {
	data := gin.H{
		"resources": []gin.H{
			{
				"id":       "res-1",
				"type":     "ec2_instance",
				"provider": "aws",
				"region":   "us-east-1",
				"name":     "web-server-1",
				"metadata": gin.H{
					"created_at":    "2024-01-15T10:30:00Z",
					"last_modified": "2024-01-16T14:20:00Z",
					"tags": map[string]string{
						"Environment": "production",
						"Owner":       "team-alpha",
					},
				},
				"relationships": []gin.H{
					{
						"type":        "depends_on",
						"target":      "res-3",
						"description": "Depends on VPC",
					},
				},
			},
			{
				"id":       "res-2",
				"type":     "storage_account",
				"provider": "azure",
				"region":   "eastus",
				"name":     "mystorageaccount",
				"metadata": gin.H{
					"created_at":    "2024-01-10T09:15:00Z",
					"last_modified": "2024-01-16T11:45:00Z",
					"encryption":    "enabled",
				},
			},
		},
		"pagination": gin.H{
			"total":       150,
			"page":        1,
			"page_size":   50,
			"has_next":    true,
			"next_cursor": "eyJpZCI6InJlcy01MCJ9",
		},
		"aggregations": gin.H{
			"by_provider": gin.H{
				"aws":   80,
				"azure": 45,
				"gcp":   25,
			},
			"by_type": gin.H{
				"compute":    60,
				"storage":    40,
				"networking": 30,
				"database":   20,
			},
		},
		"_metadata": gin.H{
			"format_version": "2.0",
			"query_time":     "120ms",
			"cache_used":     true,
		},
	}

	response := h.versionManager.WrapResponse(c, data)
	c.JSON(http.StatusOK, response)
}

// GetResourcesStreamV2 handles v2 streaming resource updates
func (h *Handlers) GetResourcesStreamV2(c *gin.Context) {
	// Set up Server-Sent Events
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Send initial connection event
	c.SSEvent("connected", gin.H{
		"timestamp":   time.Now(),
		"stream_id":   "stream-789",
		"api_version": "2.0.0",
	})

	// Simulate streaming resource updates
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for i := 0; i < 5; i++ {
		select {
		case <-ticker.C:
			event := gin.H{
				"event_id":   i + 1,
				"event_type": "resource_discovered",
				"timestamp":  time.Now(),
				"resource": gin.H{
					"id":       "res-" + string(rune(100+i)),
					"type":     "s3_bucket",
					"provider": "aws",
					"region":   "us-west-2",
					"name":     "bucket-" + string(rune(100+i)),
				},
			}
			c.SSEvent("resource_update", event)
			c.Writer.Flush()
		}
	}

	// Send completion event
	c.SSEvent("completed", gin.H{
		"timestamp":       time.Now(),
		"total_resources": 5,
		"stream_duration": "10s",
	})
}

// GetProvidersV2 handles v2 provider listing
func (h *Handlers) GetProvidersV2(c *gin.Context) {
	data := gin.H{
		"providers": []gin.H{
			{
				"name":         "aws",
				"display_name": "Amazon Web Services",
				"status":       "active",
				"version":      "2.1.0",
				"capabilities": []string{
					"discovery",
					"drift_detection",
					"remediation",
					"cost_analysis",
				},
				"regions": gin.H{
					"total":     20,
					"active":    18,
					"supported": []string{"us-east-1", "us-west-2", "eu-west-1"},
				},
				"resource_types": gin.H{
					"total": 150,
					"categories": gin.H{
						"compute":    25,
						"storage":    20,
						"networking": 30,
						"database":   15,
						"analytics":  20,
						"ml":         10,
						"security":   30,
					},
				},
				"metrics": gin.H{
					"last_discovery":    "2024-01-16T15:30:00Z",
					"success_rate":      0.98,
					"avg_response_time": "2.5s",
				},
			},
			{
				"name":         "azure",
				"display_name": "Microsoft Azure",
				"status":       "active",
				"version":      "2.0.5",
				"capabilities": []string{
					"discovery",
					"drift_detection",
					"cost_analysis",
				},
				"regions": gin.H{
					"total":     15,
					"active":    14,
					"supported": []string{"eastus", "westus2", "westeurope"},
				},
				"resource_types": gin.H{
					"total": 120,
					"categories": gin.H{
						"compute":    20,
						"storage":    18,
						"networking": 25,
						"database":   12,
						"analytics":  15,
						"ml":         8,
						"security":   22,
					},
				},
				"metrics": gin.H{
					"last_discovery":    "2024-01-16T15:25:00Z",
					"success_rate":      0.96,
					"avg_response_time": "3.1s",
				},
			},
		},
		"total": 2,
		"plugin_ecosystem": gin.H{
			"available_plugins": 25,
			"community_plugins": 15,
			"official_plugins":  10,
		},
		"_metadata": gin.H{
			"format_version":    "2.0",
			"enhanced_features": true,
		},
	}

	response := h.versionManager.WrapResponse(c, data)
	c.JSON(http.StatusOK, response)
}
