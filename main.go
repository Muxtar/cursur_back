package main

import (
	"log"
	"os"
	"strings"

	"chat-backend/internal/config"
	"chat-backend/internal/database"
	"chat-backend/internal/router"
	"chat-backend/internal/websocket"

	"github.com/gin-contrib/cors"
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

	// Setup router
	r := gin.Default()

	// Trusted proxies configuration (to avoid trusting all proxies by default)
	trustedProxiesEnv := os.Getenv("TRUSTED_PROXIES")
	if trustedProxiesEnv == "" {
		// If not provided, do not trust any proxy (safer default, removes warning)
		if err := r.SetTrustedProxies(nil); err != nil {
			log.Fatalf("Failed to set trusted proxies: %v", err)
		}
	} else {
		trusted := splitAndTrim(trustedProxiesEnv, ",")
		if err := r.SetTrustedProxies(trusted); err != nil {
			log.Fatalf("Failed to set trusted proxies: %v", err)
		}
	}

	// CORS configuration - Read allowed origins from environment variable
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "*" // Default to allow all for development
	}

	// Split comma-separated origins
	origins := []string{}
	if allowedOrigins != "*" {
		// Split by comma and trim spaces
		for _, origin := range splitAndTrim(allowedOrigins, ",") {
			if origin != "" {
				origins = append(origins, origin)
			}
		}
	} else {
		origins = []string{"*"}
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * 3600, // 12 hours
	}))

	// Setup routes
	router.SetupRoutes(r, db, hub)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
