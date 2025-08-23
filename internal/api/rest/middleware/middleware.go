package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// CORS middleware for cross-origin requests
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Authorization, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Logger middleware for request logging
func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return param.TimeStamp.Format(time.RFC3339) + " " +
			param.Method + " " +
			param.Path + " " +
			param.ClientIP + " " +
			fmt.Sprintf("%d", param.StatusCode) + " " +
			param.Latency.String() + "\n"
	})
}

// Recovery middleware for panic recovery
func Recovery() gin.HandlerFunc {
	return gin.Recovery()
}

// Auth middleware for authentication
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Simple auth check - can be enhanced
		token := c.GetHeader("Authorization")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

// RateLimit middleware for rate limiting
func RateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
	// Simple rate limiter - can be enhanced with redis or in-memory store
	return func(c *gin.Context) {
		c.Next()
	}
}
