package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// getAllowedOrigins returns the list of allowed origins from environment
func getAllowedOrigins() []string {
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	
	// Default origins
	defaultOrigins := []string{
		"https://www.fridpass.com",
		"http://localhost:3000", // Dev only
	}
	
	if allowedOrigins == "" {
		return defaultOrigins
	}
	
	// Split comma-separated origins
	origins := []string{}
	for _, origin := range strings.Split(allowedOrigins, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	
	if len(origins) == 0 {
		return defaultOrigins
	}
	
	return origins
}

// isOriginAllowed checks if the given origin is in the allowed list
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

// CORSMiddleware handles CORS for all requests
func CORSMiddleware() gin.HandlerFunc {
	allowedOrigins := getAllowedOrigins()
	
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		
		// Handle preflight (OPTIONS) requests
		if c.Request.Method == http.MethodOptions {
			if isOriginAllowed(origin, allowedOrigins) {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
			}
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			c.Header("Access-Control-Max-Age", "86400") // 24 hours
			c.AbortWithStatus(http.StatusNoContent) // 204
			return
		}
		
		// Handle regular requests
		if isOriginAllowed(origin, allowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		
		c.Next()
	}
}
