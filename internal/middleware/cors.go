package middleware

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// getAllowedOrigins returns the list of allowed origins from environment.
//
// Production rule: DO NOT use "*" (especially if credentials/cookies are used).
// If "*" is provided, we ignore it and fall back to safe defaults.
func getAllowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))

	// Default (safe) origins
	defaultOrigins := []string{
		"https://www.fridpass.com",
		"http://localhost:3000", // dev only
	}

	if raw == "" {
		return defaultOrigins
	}

	// Reject wildcard in this service (production-ready, strict)
	if raw == "*" {
		log.Printf("⚠️ CORS_ALLOWED_ORIGINS='*' is not allowed for this service. Falling back to defaults: %v", defaultOrigins)
		return defaultOrigins
	}

	var origins []string
	for _, part := range strings.Split(raw, ",") {
		o := strings.TrimSpace(part)
		if o == "" || o == "*" {
			continue
		}
		origins = append(origins, o)
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

func shouldAllowCredentials() bool {
	// Default OFF (safer, avoids accidental cookie CORS).
	// Enable explicitly in Railway if you really need cookies:
	// CORS_ALLOW_CREDENTIALS=true
	v := strings.TrimSpace(strings.ToLower(os.Getenv("CORS_ALLOW_CREDENTIALS")))
	return v == "true" || v == "1" || v == "yes"
}

func setCORSHeaders(c *gin.Context, origin string, allowCredentials bool) {
	// Always vary on Origin for correct caching/proxies.
	c.Header("Vary", "Origin")

	// IMPORTANT: never "*" here for credentialed flows.
	c.Header("Access-Control-Allow-Origin", origin)

	if allowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}

	c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
	c.Header("Access-Control-Max-Age", "86400")
}

// CORSMiddleware handles CORS for all requests
// This middleware:
// 1. Handles preflight (OPTIONS) requests with proper CORS headers
// 2. Validates origin against allowed list
// 3. Sets Vary: Origin header for proper caching
// 4. Optionally allows credentials (explicit opt-in)
// 5. Ensures CORS headers are set on success/error responses too
func CORSMiddleware() gin.HandlerFunc {
	allowedOrigins := getAllowedOrigins()
	allowCredentials := shouldAllowCredentials()

	log.Printf("🔧 CORS enabled origins: %v", allowedOrigins)
	log.Printf("🔧 CORS allow credentials: %v", allowCredentials)
	log.Println("✅ CORS middleware ACTIVE")

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		method := c.Request.Method
		path := c.Request.URL.Path

		// Only care about CORS when browser sends Origin.
		if origin != "" {
			allowed := isOriginAllowed(origin, allowedOrigins)

			if method == http.MethodOptions {
				log.Printf("✅ OPTIONS preflight hit: path=%s origin=%s allowed=%v", path, origin, allowed)
				if !allowed {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}

				setCORSHeaders(c, origin, allowCredentials)
				// let the catch-all OPTIONS route return 204
				c.Next()
				return
			}

			if allowed {
				setCORSHeaders(c, origin, allowCredentials)
			}
		}

		c.Next()
	}
}
