package middleware

import (
	"log"
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
// This middleware:
// 1. Handles preflight (OPTIONS) requests with proper CORS headers
// 2. Validates origin against allowed list
// 3. Sets Vary: Origin header for proper caching
// 4. Only allows credentials when origin is explicitly allowed
func CORSMiddleware() gin.HandlerFunc {
	allowedOrigins := getAllowedOrigins()
	log.Printf("🔧 CORS Middleware initialized with allowed origins: %v", allowedOrigins)
	
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		method := c.Request.Method
		path := c.Request.URL.Path
		
		// DEBUG: Log all requests (especially OPTIONS)
		log.Printf("🌐 CORS Middleware: %s %s | Origin: %s", method, path, origin)
		
		// Always set Vary: Origin header for proper cache control
		// This tells caches that the response varies based on the Origin header
		c.Header("Vary", "Origin")
		
		// Handle preflight (OPTIONS) requests
		if method == http.MethodOptions {
			log.Printf("✅ OPTIONS preflight request detected: %s | Origin: %s", path, origin)
			
			if isOriginAllowed(origin, allowedOrigins) {
				log.Printf("✅ Origin allowed: %s", origin)
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
				c.Header("Access-Control-Max-Age", "86400") // 24 hours
				c.AbortWithStatus(http.StatusNoContent) // 204
				log.Printf("✅ OPTIONS response sent with CORS headers")
				return
			} else {
				log.Printf("❌ Origin NOT allowed: %s", origin)
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
		}
		
		// Handle regular requests
		if isOriginAllowed(origin, allowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		} else if origin != "" {
			log.Printf("⚠️ Origin not allowed for regular request: %s", origin)
		}
		
		c.Next()
	}
}
