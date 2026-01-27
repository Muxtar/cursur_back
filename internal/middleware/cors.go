package middleware

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware provides production-ready CORS using gin-contrib/cors.
//
// Requirements:
// - AllowOrigins: https://www.fridpass.com + (dev) http://localhost:3000
// - AllowMethods: GET,POST,PUT,PATCH,DELETE,OPTIONS
// - AllowHeaders: Content-Type, Authorization, X-Requested-With
// - Vary: Origin
// - Preflight returns 204
func CORSMiddleware() gin.HandlerFunc {
	allowedOrigins := []string{
		"https://www.fridpass.com",
		"http://localhost:3000",
	}

	cfg := cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type", "Authorization", "X-Requested-With"},
		// IMPORTANT:
		// - Keep false unless you actually use cookies/sessions cross-site.
		// - If you later enable it, NEVER use AllowAllOrigins / "*" with it.
		// Required when frontend uses fetch(..., { credentials: "include" }) or axios withCredentials.
		// Safe here because AllowOrigins is an explicit whitelist (NOT "*").
		AllowCredentials: true,
		MaxAge:           24 * time.Hour,
	}

	// Log once on startup (proof in Railway logs)
	log.Printf("✅ CORS enabled origins: %v", allowedOrigins)
	log.Printf("✅ CORS allow credentials: %v", cfg.AllowCredentials)

	return cors.New(cfg)
}
