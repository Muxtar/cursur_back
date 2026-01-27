//go:build ignore
// +build ignore

// This file is an example only and is excluded from builds.
// (It previously conflicted with the real `main.go` by defining another main package.)
package main

import (
	"log"
	"net/http"
	"os"
	"strings"
)

// getAllowedOrigins returns the list of allowed origins
func getAllowedOrigins() []string {
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	
	defaultOrigins := []string{
		"https://www.fridpass.com",
		"http://localhost:3000",
	}
	
	if allowedOrigins == "" {
		return defaultOrigins
	}
	
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

// corsMiddleware handles CORS for net/http
func corsMiddleware(next http.Handler) http.Handler {
	allowedOrigins := getAllowedOrigins()
	log.Printf("🔧 CORS Middleware initialized with allowed origins: %v", allowedOrigins)
	
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		method := r.Method
		path := r.URL.Path
		
		// DEBUG: Log all requests (especially OPTIONS)
		log.Printf("🌐 CORS Middleware: %s %s | Origin: %s", method, path, origin)
		
		// Always set Vary: Origin header
		w.Header().Set("Vary", "Origin")
		
		// Handle preflight (OPTIONS) requests
		if method == http.MethodOptions {
			log.Printf("✅ OPTIONS preflight request detected: %s | Origin: %s", path, origin)
			
			if isOriginAllowed(origin, allowedOrigins) {
				log.Printf("✅ Origin allowed: %s", origin)
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
				w.Header().Set("Access-Control-Max-Age", "86400")
				w.WriteHeader(http.StatusNoContent) // 204
				log.Printf("✅ OPTIONS response sent with CORS headers")
				return
			} else {
				log.Printf("❌ Origin NOT allowed: %s", origin)
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}
		
		// Handle regular requests
		if isOriginAllowed(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else if origin != "" {
			log.Printf("⚠️ Origin not allowed for regular request: %s", origin)
		}
		
		next.ServeHTTP(w, r)
	})
}

func sendCodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Your handler logic here
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Verification code sent", "success": true}`))
}

func main() {
	// Create router
	mux := http.NewServeMux()
	
	// Setup routes
	mux.HandleFunc("/api/v1/auth/send-code", sendCodeHandler)
	
	// Apply CORS middleware to all routes
	handler := corsMiddleware(mux)
	
	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	// Listen on 0.0.0.0:port
	listenAddr := "0.0.0.0:" + port
	log.Printf("🚀 Server starting on %s", listenAddr)
	log.Printf("🔧 CORS enabled for: https://www.fridpass.com, http://localhost:3000")
	
	if err := http.ListenAndServe(listenAddr, handler); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
