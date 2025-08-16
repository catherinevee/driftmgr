package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/monitoring"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

const (
	serviceName = "api-gateway"
	servicePort = "8080"
)

var (
	logger   = monitoring.GetGlobalLogger()
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}
)

// ServiceConfig holds configuration for each microservice
type ServiceConfig struct {
	Name string
	URL  string
	Port string
}

// GatewayConfig holds the gateway configuration
type GatewayConfig struct {
	Services map[string]ServiceConfig
}

var config GatewayConfig

func main() {
	// Initialize service configuration
	config = GatewayConfig{
		Services: map[string]ServiceConfig{
			"discovery": {
				Name: "discovery-service",
				URL:  getEnv("DISCOVERY_SERVICE_URL", "http://discovery-service:8081"),
				Port: "8081",
			},
			"analysis": {
				Name: "analysis-service",
				URL:  getEnv("ANALYSIS_SERVICE_URL", "http://analysis-service:8082"),
				Port: "8082",
			},
			"remediation": {
				Name: "remediation-service",
				URL:  getEnv("REMEDIATION_SERVICE_URL", "http://remediation-service:8083"),
				Port: "8083",
			},
			"workflow": {
				Name: "workflow-service",
				URL:  getEnv("WORKFLOW_SERVICE_URL", "http://workflow-service:8084"),
				Port: "8084",
			},
			"notification": {
				Name: "notification-service",
				URL:  getEnv("NOTIFICATION_SERVICE_URL", "http://notification-service:8085"),
				Port: "8085",
			},
			"web": {
				Name: "web-service",
				URL:  getEnv("WEB_SERVICE_URL", "http://web-service:8086"),
				Port: "8086",
			},
			"cli": {
				Name: "cli-service",
				URL:  getEnv("CLI_SERVICE_URL", "http://cli-service:8087"),
				Port: "8087",
			},
			"database": {
				Name: "database-service",
				URL:  getEnv("DATABASE_SERVICE_URL", "http://database-service:8088"),
				Port: "8088",
			},
			"cache": {
				Name: "cache-service",
				URL:  getEnv("CACHE_SERVICE_URL", "http://cache-service:8089"),
				Port: "8089",
			},
			"ml": {
				Name: "ml-service",
				URL:  getEnv("ML_SERVICE_URL", "http://ml-service:8090"),
				Port: "8090",
			},
			"monitoring": {
				Name: "monitoring-service",
				URL:  getEnv("MONITORING_SERVICE_URL", "http://monitoring-service:8091"),
				Port: "8091",
			},
			"security": {
				Name: "security-service",
				URL:  getEnv("SECURITY_SERVICE_URL", "http://security-service:8092"),
				Port: "8092",
			},
		},
	}

	// Set up router
	router := mux.NewRouter()

	// Middleware
	router.Use(loggingMiddleware)
	router.Use(corsMiddleware)
	router.Use(rateLimitMiddleware)
	router.Use(authMiddleware)

	// Health check
	router.HandleFunc("/health", handleHealth).Methods("GET")

	// API routes with service routing
	api := router.PathPrefix("/api/v1").Subrouter()

	// Discovery service routes
	api.PathPrefix("/discover").HandlerFunc(createReverseProxy("discovery"))
	api.PathPrefix("/resources").HandlerFunc(createReverseProxy("discovery"))
	api.PathPrefix("/monitor").HandlerFunc(createReverseProxy("discovery"))
	api.PathPrefix("/cache").HandlerFunc(createReverseProxy("discovery"))

	// Analysis service routes
	api.PathPrefix("/analyze").HandlerFunc(createReverseProxy("analysis"))
	api.PathPrefix("/analysis").HandlerFunc(createReverseProxy("analysis"))
	api.PathPrefix("/predict").HandlerFunc(createReverseProxy("analysis"))
	api.PathPrefix("/patterns").HandlerFunc(createReverseProxy("analysis"))
	api.PathPrefix("/risks").HandlerFunc(createReverseProxy("analysis"))
	api.PathPrefix("/history").HandlerFunc(createReverseProxy("analysis"))
	api.PathPrefix("/trends").HandlerFunc(createReverseProxy("analysis"))

	// Remediation service routes
	api.PathPrefix("/remediate").HandlerFunc(createReverseProxy("remediation"))
	api.PathPrefix("/strategies").HandlerFunc(createReverseProxy("remediation"))
	api.PathPrefix("/approval").HandlerFunc(createReverseProxy("remediation"))

	// Workflow service routes
	api.PathPrefix("/workflows").HandlerFunc(createReverseProxy("workflow"))
	api.PathPrefix("/templates").HandlerFunc(createReverseProxy("workflow"))

	// Notification service routes
	api.PathPrefix("/notify").HandlerFunc(createReverseProxy("notification"))
	api.PathPrefix("/notifications").HandlerFunc(createReverseProxy("notification"))
	api.PathPrefix("/webhooks").HandlerFunc(createReverseProxy("notification"))

	// Database service routes
	api.PathPrefix("/db").HandlerFunc(createReverseProxy("database"))

	// Cache service routes
	api.PathPrefix("/cache").HandlerFunc(createReverseProxy("cache"))

	// ML service routes
	api.PathPrefix("/ml").HandlerFunc(createReverseProxy("ml"))

	// Monitoring service routes
	api.PathPrefix("/monitoring").HandlerFunc(createReverseProxy("monitoring"))

	// Security service routes
	api.PathPrefix("/auth").HandlerFunc(createReverseProxy("security"))
	api.PathPrefix("/audit").HandlerFunc(createReverseProxy("security"))

	// WebSocket support
	router.HandleFunc("/ws", handleWebSocket)

	// Default route to web service
	router.PathPrefix("/").HandlerFunc(createReverseProxy("web"))

	// Start server
	logger.Info("Starting API Gateway on port " + servicePort)
	log.Fatal(http.ListenAndServe(":"+servicePort, router))
}

// handleHealth handles health check requests
func handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check health of all services
	healthStatus := make(map[string]interface{})

	for serviceName, serviceConfig := range config.Services {
		status := checkServiceHealth(serviceConfig.URL)
		healthStatus[serviceName] = status
	}

	response := map[string]interface{}{
		"service":   serviceName,
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
		"services":  healthStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// createReverseProxy creates a reverse proxy for a specific service
func createReverseProxy(serviceName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serviceConfig, exists := config.Services[serviceName]
		if !exists {
			http.Error(w, "Service not found", http.StatusNotFound)
			return
		}

		// Parse the target URL
		targetURL, err := url.Parse(serviceConfig.URL)
		if err != nil {
			http.Error(w, "Invalid target URL", http.StatusInternalServerError)
			return
		}

		// Create reverse proxy
		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		// Modify response
		proxy.ModifyResponse = func(resp *http.Response) error {
			// Add CORS headers
			resp.Header.Set("Access-Control-Allow-Origin", "*")
			resp.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			resp.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			return nil
		}

		// Update request headers
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.Header.Set("X-Forwarded-Host", req.Host)
			req.Header.Set("X-Forwarded-Proto", "https")
			req.Header.Set("X-Forwarded-For", req.RemoteAddr)
			req.Header.Set("X-Service-Name", serviceConfig.Name)
		}

		// Handle errors
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Error("Proxy error for " + serviceName + ": " + err.Error())
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		}

		// Serve the request
		proxy.ServeHTTP(w, r)
	}
}

// handleWebSocket handles WebSocket connections
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket upgrade failed: " + err.Error())
		return
	}
	defer conn.Close()

	// Handle WebSocket messages
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			logger.Error("WebSocket read error: " + err.Error())
			break
		}

		// Echo the message back (for now)
		err = conn.WriteMessage(messageType, message)
		if err != nil {
			logger.Error("WebSocket write error: " + err.Error())
			break
		}
	}
}

// checkServiceHealth checks the health of a service
func checkServiceHealth(serviceURL string) map[string]interface{} {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(serviceURL + "/health")
	if err != nil {
		return map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
			"url":    serviceURL,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return map[string]interface{}{
			"status": "healthy",
			"url":    serviceURL,
		}
	}

	return map[string]interface{}{
		"status": "unhealthy",
		"code":   resp.StatusCode,
		"url":    serviceURL,
	}
}

// Middleware functions

// loggingMiddleware logs all requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		logger.Info(fmt.Sprintf(
			"%s %s %d %v %s",
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration,
			r.RemoteAddr,
		))
	})
}

// corsMiddleware handles CORS
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// rateLimitMiddleware implements basic rate limiting
func rateLimitMiddleware(next http.Handler) http.Handler {
	// Simple in-memory rate limiter (in production, use Redis)
	clients := make(map[string]*rateLimiter)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr

		limiter, exists := clients[clientIP]
		if !exists {
			limiter = &rateLimiter{
				requests: make([]time.Time, 0),
				limit:    100, // requests per minute
			}
			clients[clientIP] = limiter
		}

		if !limiter.allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// authMiddleware handles authentication
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health checks and public endpoints
		if r.URL.Path == "/health" || strings.HasPrefix(r.URL.Path, "/api/v1/auth") {
			next.ServeHTTP(w, r)
			return
		}

		// Check for API key or JWT token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// For now, allow requests without auth (development mode)
			// In production, return 401 Unauthorized
			next.ServeHTTP(w, r)
			return
		}

		// Validate token (implement JWT validation here)
		// For now, just pass through
		next.ServeHTTP(w, r)
	})
}

// Helper types and functions

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// rateLimiter implements basic rate limiting
type rateLimiter struct {
	requests []time.Time
	limit    int
}

func (rl *rateLimiter) allow() bool {
	now := time.Now()

	// Remove old requests
	var validRequests []time.Time
	for _, req := range rl.requests {
		if now.Sub(req) < time.Minute {
			validRequests = append(validRequests, req)
		}
	}
	rl.requests = validRequests

	// Check if under limit
	if len(rl.requests) >= rl.limit {
		return false
	}

	// Add current request
	rl.requests = append(rl.requests, now)
	return true
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
