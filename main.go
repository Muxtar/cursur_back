package main

import (
	"log"
	"os"
	"strings"

	"chat-backend/internal/config"
	"chat-backend/internal/database"
	"chat-backend/internal/middleware"
	"chat-backend/internal/router"
	"chat-backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Helper function to split and trim strings
func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func main() {
	// Load environment variables (only in development, Railway uses environment variables directly)
	// Check if we're in development mode (not in Railway/production)
	if os.Getenv("RAILWAY_ENVIRONMENT") == "" && os.Getenv("RAILWAY_SERVICE_NAME") == "" {
		if err := godotenv.Load(); err != nil {
			// Silently ignore in production, only log in development
			log.Println("No .env file found (this is normal in production)")
		}
	}

	// Load configuration
	cfg := config.Load()

	// Initialize databases
	db := database.Initialize(cfg)
	defer db.Close()

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Set Gin mode (release for production, debug for development)
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		// Default to release mode in production (Railway), debug in local
		if os.Getenv("RAILWAY_ENVIRONMENT") != "" || os.Getenv("RAILWAY_SERVICE_NAME") != "" {
			gin.SetMode(gin.ReleaseMode)
		} else {
			gin.SetMode(gin.DebugMode)
		}
	} else {
		gin.SetMode(ginMode)
	}

	// Setup router
	r := gin.Default()

	// Trusted proxies configuration (to avoid trusting all proxies by default)
	// This removes the Gin warning about trusting all proxies
	trustedProxiesEnv := os.Getenv("TRUSTED_PROXIES")
	if trustedProxiesEnv == "" {
		// If not provided, do not trust any proxy (safer default, removes warning)
		// Use empty slice instead of nil for better compatibility
		if err := r.SetTrustedProxies([]string{}); err != nil {
			log.Printf("Warning: Failed to set trusted proxies: %v", err)
		}
	} else {
		trusted := splitAndTrim(trustedProxiesEnv, ",")
		if err := r.SetTrustedProxies(trusted); err != nil {
			log.Printf("Warning: Failed to set trusted proxies: %v", err)
		}
	}

	// ===== CORS CONFIGURATION =====
	// CRITICAL: CORS middleware MUST be added BEFORE routes
	// This ensures preflight (OPTIONS) requests are handled correctly
	r.Use(middleware.CORSMiddleware())
	log.Println("✅ CORS middleware configured and added to router")

	// Catch-all OPTIONS so Gin never returns 404/405 for preflight.
	// CORS headers are set by the middleware above.
	r.OPTIONS("/*path", func(c *gin.Context) {
		c.Status(204)
	})
	// ===== END CORS CONFIGURATION =====

	// Setup routes (AFTER CORS middleware)
	router.SetupRoutes(r, db, hub, cfg)

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Listen on 0.0.0.0 to accept connections from all interfaces
	// This is REQUIRED for Railway and other cloud platforms
	listenAddr := "0.0.0.0:" + port
	log.Printf("🚀 Server starting on %s", listenAddr)
	
	// Log CORS configuration
	corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "https://www.fridpass.com, http://localhost:3000 (default)"
	}
	log.Printf("🔧 CORS enabled origins: %s", corsOrigins)

	// Log deploy/build identity (helps verify Railway deployed the right commit)
	if sha := os.Getenv("RAILWAY_GIT_COMMIT_SHA"); sha != "" {
		log.Printf("🔧 Build commit: %s", sha)
	} else if sha := os.Getenv("GIT_COMMIT_SHA"); sha != "" {
		log.Printf("🔧 Build commit: %s", sha)
	}
	
	if err := r.Run(listenAddr); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
