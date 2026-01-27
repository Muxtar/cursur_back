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

	// CORS configuration - Read allowed origins from environment variable
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	
	// Default origins if not set (include common front-end URLs)
	defaultOrigins := []string{
		"https://www.fridpass.com",
		"https://fridpass.com",
		"http://localhost:3000",
		"http://localhost:3001",
	}
	
	var corsConfig cors.Config
	var origins []string
	
	if allowedOrigins == "" || allowedOrigins == "*" {
		// Use default origins + allow all as fallback
		origins = defaultOrigins
		log.Println("⚠️ CORS_ALLOWED_ORIGINS not set, using default origins")
		log.Printf("✅ CORS default allowed origins: %v", origins)
		
		corsConfig = cors.Config{
			AllowOrigins:     origins,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH", "HEAD"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Requested-With", "Accept", "Accept-Language", "Content-Language"},
			ExposeHeaders:    []string{"Content-Length", "Content-Type", "Authorization"},
			AllowCredentials: true,
			MaxAge:           12 * 3600, // 12 hours
			AllowWildcard:    false,
		}
	} else {
		// Split comma-separated origins
		origins = []string{}
		for _, origin := range splitAndTrim(allowedOrigins, ",") {
			if origin != "" {
				origins = append(origins, origin)
				log.Printf("✅ CORS allowed origin: %s", origin)
			}
		}
		
		corsConfig = cors.Config{
			AllowOrigins:     origins,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH", "HEAD"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Requested-With", "Accept", "Accept-Language", "Content-Language"},
			ExposeHeaders:    []string{"Content-Length", "Content-Type", "Authorization"},
			AllowCredentials: true,
			MaxAge:           12 * 3600, // 12 hours
		}
	}

	// Apply CORS middleware BEFORE routes
	r.Use(cors.New(corsConfig))
	
	// Add manual OPTIONS handler for all routes to ensure preflight works
	r.OPTIONS("/*path", func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			// Check if origin is in allowed list
			allowed := false
			for _, allowedOrigin := range origins {
				if origin == allowedOrigin {
					allowed = true
					break
				}
			}
			if allowed {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
			} else {
				// If not in list but CORS_ALLOWED_ORIGINS is "*", allow it
				if allowedOrigins == "" || allowedOrigins == "*" {
					c.Header("Access-Control-Allow-Origin", origin)
					c.Header("Access-Control-Allow-Credentials", "true")
				}
			}
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH, HEAD")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Requested-With, Accept, Accept-Language, Content-Language")
		c.Header("Access-Control-Max-Age", "43200")
		c.Status(204)
	})
	
	log.Println("✅ CORS middleware configured")

	// Setup routes
	router.SetupRoutes(r, db, hub, cfg)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
