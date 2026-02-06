package middleware

import (
	"log"
	"time"

	"chat-backend/internal/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware provides production-ready CORS using gin-contrib/cors.
// Origins are read from config (CORS_ALLOWED_ORIGINS env). Railway'da front-end
// URL'inizi ekleyin, örn: https://yourfrontend.up.railway.app,http://localhost:3000
func CORSMiddleware(cfg *config.Config) gin.HandlerFunc {
	allowedOrigins := cfg.CORSAllowedOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"http://localhost:3000"}
	}

	corsCfg := cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Requested-With"},
		AllowCredentials: true,
		MaxAge:           24 * time.Hour,
	}

	log.Printf("✅ CORS enabled origins: %v", allowedOrigins)
	return cors.New(corsCfg)
}
