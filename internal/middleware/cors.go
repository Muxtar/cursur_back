package middleware

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// getAllowedOrigins returns the list of allowed origins from environment
// CRITICAL: "*" is NOT allowed when credentials are used (browser security)
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
	
	// CRITICAL: Reject "*" if credentials are used
	// Check if credentials are used (via environment variable or default true for security)
	useCredentials := os.Getenv("CORS_ALLOW_CREDENTIALS")
	if useCredentials == "" {
		// Default to true for security (explicit origins required)
		useCredentials = "true"
	}
	
	// If "*" is provided and credentials are enabled, reject it
	if strings.TrimSpace(allowedOrigins) == "*" {
		if useCredentials == "true" {
			log.Printf("⚠️ WARNING: CORS_ALLOWED_ORIGINS='*' is not compatible with credentials. Using default origins instead.")
			return defaultOrigins
		}
		// If credentials are disabled, allow "*" (but this is not recommended)
		log.Printf("⚠️ WARNING: CORS_ALLOWED_ORIGINS='*' is set. This is not recommended for production.")
		return []string{"*"}
	}
	
	// Split comma-separated origins
	origins := []string{}
	for _, origin := range strings.Split(allowedOrigins, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" && origin != "*" {
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
// 4. Only allows credentials when origin is explicitly allowed (not "*")
// 5. Sets CORS headers on ALL responses (not just OPTIONS)
func CORSMiddleware() gin.HandlerFunc {
	allowedOrigins := getAllowedOrigins()
	
	// Check if credentials should be enabled
	useCredentialsEnv := os.Getenv("CORS_ALLOW_CREDENTIALS")
	useCredentials := useCredentialsEnv == "" || useCredentialsEnv == "true" // Default to true for security
	
	// If "*" is in allowed origins, credentials cannot be used
	hasWildcard := false
	for _, orig := range allowedOrigins {
		if orig == "*" {
			hasWildcard = true
			break
		}
	}
	if hasWildcard {
		useCredentials = false
		log.Printf("⚠️ WARNING: Wildcard origin '*' detected. Credentials will be disabled.")
	}
	
	log.Printf("🔧 CORS Middleware initialized with allowed origins: %v", allowedOrigins)
	log.Printf("🔧 CORS credentials enabled: %v", useCredentials)
	log.Println("✅ CORS middleware ACTIVE - preflight requests will be handled")
	
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		method := c.Request.Method
		path := c.Request.URL.Path
		
		// DEBUG: Log all requests (especially OPTIONS)
		log.Printf("🌐 CORS Middleware: %s %s | Origin: %s", method, path, origin)
		
		// Always set Vary: Origin header for proper cache control
		c.Header("Vary", "Origin")
		
		// Handle preflight (OPTIONS) requests
		if method == http.MethodOptions {
			log.Printf("✅ OPTIONS preflight request detected: %s | Origin: %s", path, origin)
			
			// Check if origin is allowed
			originAllowed := false
			var allowedOrigin string
			
			if hasWildcard {
				// Wildcard allows all origins
				allowedOrigin = "*"
				originAllowed = true
			} else if isOriginAllowed(origin, allowedOrigins) {
				allowedOrigin = origin
				originAllowed = true
			}
			
			if originAllowed {
				log.Printf("✅ Origin allowed: %s", origin)
				c.Header("Access-Control-Allow-Origin", allowedOrigin)
				
				// Only set credentials header if not using wildcard
				if useCredentials && !hasWildcard {
					c.Header("Access-Control-Allow-Credentials", "true")
				}
				
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
		
		// Handle regular requests - ALWAYS set CORS headers if origin is allowed
		originAllowed := false
		var allowedOrigin string
		
		if hasWildcard {
			allowedOrigin = "*"
			originAllowed = true
		} else if isOriginAllowed(origin, allowedOrigins) {
			allowedOrigin = origin
			originAllowed = true
		}
		
		if originAllowed {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			// Only set credentials header if not using wildcard
			if useCredentials && !hasWildcard {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
		} else if origin != "" {
			log.Printf("⚠️ Origin not allowed for regular request: %s", origin)
		}
		
		c.Next()
	}
}
