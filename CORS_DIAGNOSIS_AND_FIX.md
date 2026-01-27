# CORS Preflight Sorunu - Teşhis ve Kesin Çözüm

## 🔍 TEŞHİS PLANI

### 1. Preflight OPTIONS İsteğine Backend Ne Dönüyor?
**Kontrol:** `curl -i -X OPTIONS` ile test et
- Status code ne? (204, 200, 405, 404, 403?)
- CORS header'ları var mı?
- Vary: Origin var mı?

### 2. Header'lar Middleware'den Önce mi Sonra mı Set Ediliyor?
**Sorun:** Gin.Default() Logger ve Recovery middleware'lerini otomatik ekler ve bunlar CORS'dan ÖNCE çalışır.
**Kontrol:** CORS middleware'ine log ekle ve Railway logs'da görünüyor mu?

### 3. Router OPTIONS'u Düşürüyor mu? (405/404)
**Sorun:** Gin router'ı OPTIONS route'u tanımlı değilse 405 Method Not Allowed dönebilir.
**Kontrol:** Router'da OPTIONS handler var mı? Yoksa Gin 405 döner ve middleware çalışmaz.

### 4. /api/v1 Group'a Middleware Gerçekten Uygulanıyor mu?
**Sorun:** CORS middleware sadece root router'a eklenmiş, group'lara eklenmemiş olabilir.
**Kontrol:** Middleware tüm route'lara uygulanıyor mu?

### 5. CORS Sadece POST'a mı Ekli, OPTIONS'a Ekli Değil mi?
**Sorun:** Middleware OPTIONS'u handle ediyor ama router OPTIONS'u yakalıyor olabilir.

### 6. Credentials/Cookie Var mı? Varsa "*" Kullanımı Yasak
**Kontrol:** Access-Control-Allow-Origin "*" kullanılıyor mu? (Yasak!)

---

## ✅ KESİN ÇÖZÜM

### ÇÖZÜM A: Gin Framework (Önerilen - Log ile Debug)

**Sorun:** Gin.Default() Logger ve Recovery ekler, ama CORS middleware'i çalışıyor olmalı. 
**Gerçek Sorun:** Middleware çalışıyor ama log yok, ya da router OPTIONS'u yakalıyor.

**Dosya:** `back-end/internal/middleware/cors.go`

```go
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
```

**Dosya:** `back-end/main.go`

```go
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
	if os.Getenv("RAILWAY_ENVIRONMENT") == "" && os.Getenv("RAILWAY_SERVICE_NAME") == "" {
		if err := godotenv.Load(); err != nil {
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

	// Set Gin mode
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
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

	// Trusted proxies configuration
	trustedProxiesEnv := os.Getenv("TRUSTED_PROXIES")
	if trustedProxiesEnv == "" {
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
	log.Printf("🔧 CORS enabled for: https://www.fridpass.com, http://localhost:3000")
	
	if err := r.Run(listenAddr); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
```

---

### ÇÖZÜM B: net/http (Framework Yoksa)

```go
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

func main() {
	// Your handlers
	mux := http.NewServeMux()
	
	// Apply CORS middleware to all routes
	handler := corsMiddleware(mux)
	
	// Setup routes
	mux.HandleFunc("/api/v1/auth/send-code", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Your handler logic here
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Code sent", "success": true}`))
	})
	
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
```

---

## 🧪 DOĞRULAMA KOMUTLARI

### 1. Preflight (OPTIONS) Testi

```bash
curl -i -X OPTIONS 'https://cursurback-production.up.railway.app/api/v1/auth/send-code' \
  -H 'Origin: https://www.fridpass.com' \
  -H 'Access-Control-Request-Method: POST' \
  -H 'Access-Control-Request-Headers: content-type'
```

**Beklenen Çıktı:**
```
HTTP/1.1 204 No Content
Access-Control-Allow-Origin: https://www.fridpass.com
Access-Control-Allow-Credentials: true
Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization, X-Requested-With
Access-Control-Max-Age: 86400
Vary: Origin
```

**Railway Logs'da Görmeli:**
```
🌐 CORS Middleware: OPTIONS /api/v1/auth/send-code | Origin: https://www.fridpass.com
✅ OPTIONS preflight request detected: /api/v1/auth/send-code | Origin: https://www.fridpass.com
✅ Origin allowed: https://www.fridpass.com
✅ OPTIONS response sent with CORS headers
```

### 2. POST İsteği Testi

```bash
curl -i -X POST 'https://cursurback-production.up.railway.app/api/v1/auth/send-code' \
  -H 'Origin: https://www.fridpass.com' \
  -H 'Content-Type: application/json' \
  --data '{"phone_number":"+994516480030"}'
```

**Beklenen Çıktı:**
```
HTTP/1.1 200 OK
Access-Control-Allow-Origin: https://www.fridpass.com
Access-Control-Allow-Credentials: true
Vary: Origin
Content-Type: application/json

{"message":"Verification code sent","success":true}
```

### 3. Geçersiz Origin Testi (403 Beklenir)

```bash
curl -i -X OPTIONS 'https://cursurback-production.up.railway.app/api/v1/auth/send-code' \
  -H 'Origin: https://evil.com' \
  -H 'Access-Control-Request-Method: POST'
```

**Beklenen Çıktı:**
```
HTTP/1.1 403 Forbidden
Vary: Origin
```

---

## 🌐 Next.js Frontend Tarafı

### 1. Environment Variable

**Dosya:** `front-end/.env.local` veya Railway environment variables

```bash
NEXT_PUBLIC_API_URL=https://cursurback-production.up.railway.app/api/v1
```

### 2. API Client (Credentials Kullanılmıyorsa)

**Eğer cookie/credentials kullanmıyorsanız:**

```typescript
// front-end/src/lib/api.ts
const response = await fetch(url, {
  ...options,
  headers,
  // credentials: 'include' KULLANMAYIN eğer cookie yoksa
});
```

**Eğer cookie/credentials kullanıyorsanız:**

```typescript
// front-end/src/lib/api.ts
const response = await fetch(url, {
  ...options,
  headers,
  credentials: 'include', // ✅ Cookie göndermek için gerekli
});
```

**Backend'de de credentials kullanıyorsanız:**
- `Access-Control-Allow-Credentials: true` ✅ (zaten var)
- `Access-Control-Allow-Origin` ASLA "*" olmamalı ✅ (zaten spesifik origin)

### 3. Direct Fetch Örneği

```typescript
// Credentials YOKSA
const sendCode = async (phoneNumber: string) => {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'https://cursurback-production.up.railway.app/api/v1';
  
  const response = await fetch(`${apiUrl}/auth/send-code`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    // credentials: 'include' YOK
    body: JSON.stringify({ phone_number: phoneNumber }),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Request failed');
  }

  return response.json();
};

// Credentials VARSA (cookie göndermek için)
const sendCodeWithCredentials = async (phoneNumber: string) => {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'https://cursurback-production.up.railway.app/api/v1';
  
  const response = await fetch(`${apiUrl}/auth/send-code`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include', // ✅ Cookie göndermek için
    body: JSON.stringify({ phone_number: phoneNumber }),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Request failed');
  }

  return response.json();
};
```

---

## ❌ HALA AYNI HATAYI GÖRÜRSEM - KONTROL LİSTESİ

### 1. Middleware'in Gerçekten Çalıştığını Log ile Kanıtla

**Railway logs'da şunları görmelisiniz:**
```
🔧 CORS Middleware initialized with allowed origins: [https://www.fridpass.com http://localhost:3000]
✅ CORS middleware configured and added to router
🚀 Server starting on 0.0.0.0:8080
```

**OPTIONS isteği geldiğinde:**
```
🌐 CORS Middleware: OPTIONS /api/v1/auth/send-code | Origin: https://www.fridpass.com
✅ OPTIONS preflight request detected: /api/v1/auth/send-code | Origin: https://www.fridpass.com
✅ Origin allowed: https://www.fridpass.com
✅ OPTIONS response sent with CORS headers
```

**Eğer bu log'ları görmüyorsanız:**
- Middleware çalışmıyor demektir
- Router yapılandırmasını kontrol edin
- Deploy'un doğru commit'i aldığını kontrol edin

### 2. Railway Deploy'un Doğru Commit'i Aldığını Kontrol Et

```bash
# Railway dashboard'da:
# 1. Deployments sekmesine git
# 2. Son deploy'un commit hash'ini kontrol et
# 3. GitHub'da aynı commit'te CORS middleware'in olduğunu doğrula
```

### 3. Cloudflare / Custom Domain Proxy Header Kırpıyor mu Kontrol Et

**Eğer Cloudflare kullanıyorsanız:**
- Cloudflare'de "Always Use HTTPS" kapalı olmalı (backend kendi HTTPS'i handle ediyorsa)
- Cloudflare'de CORS header'ları kırpılıyor olabilir
- Test için direkt Railway URL'i kullanın: `https://cursurback-production.up.railway.app`

### 4. Aynı Endpoint'e Postman Çalışıp Tarayıcı Çalışmıyorsa Kesin CORS'tur

**Postman testi:**
```bash
# Postman'de OPTIONS isteği gönder
# CORS header'ları görünüyor mu?
# Eğer görünüyorsa ama tarayıcıda çalışmıyorsa:
# - Browser cache'i temizle
# - Incognito mode'da test et
# - Browser console'da Network tab'de OPTIONS isteğini kontrol et
```

### 5. Browser Console'da Network Tab Kontrolü

1. Browser DevTools açın (F12)
2. Network tab'e gidin
3. OPTIONS isteğini bulun
4. Response Headers'ı kontrol edin:
   - `Access-Control-Allow-Origin` var mı?
   - `Access-Control-Allow-Methods` var mı?
   - `Access-Control-Allow-Headers` var mı?
   - `Vary: Origin` var mı?

**Eğer header'lar yoksa:**
- Backend'de middleware çalışmıyor demektir
- Railway logs'u kontrol edin

### 6. Environment Variables Kontrolü

**Railway dashboard'da kontrol edin:**
```bash
PORT=8080  # Railway otomatik set eder
CORS_ALLOWED_ORIGINS=https://www.fridpass.com,http://localhost:3000  # Opsiyonel
```

**Eğer CORS_ALLOWED_ORIGINS set edilmemişse:**
- Default değerler kullanılacak: `https://www.fridpass.com`, `http://localhost:3000`
- Bu normal ve çalışmalı

### 7. Gin Router'da OPTIONS Route'u Var mı?

**Kontrol:** Router'da `/api/v1/auth/send-code` için OPTIONS handler var mı?
- **Yoksa:** Gin 405 Method Not Allowed döner ve middleware çalışmaz
- **Çözüm:** CORS middleware'i OPTIONS'u handle ediyor, router'da OPTIONS handler'a gerek yok

### 8. Middleware Sırası Doğru mu?

**Kontrol:** `main.go`'da CORS middleware route'lardan ÖNCE mi?
```go
// ✅ DOĞRU:
r.Use(middleware.CORSMiddleware())  // Önce
router.SetupRoutes(r, db, hub, cfg)  // Sonra

// ❌ YANLIŞ:
router.SetupRoutes(r, db, hub, cfg)  // Önce
r.Use(middleware.CORSMiddleware())   // Sonra (çalışmaz!)
```

---

## 📝 ÖZET

1. ✅ CORS middleware log eklenmiş (debug için)
2. ✅ OPTIONS preflight handle ediliyor
3. ✅ Origin kontrolü yapılıyor
4. ✅ Vary: Origin header'ı set ediliyor
5. ✅ Railway port ve listen adresi doğru
6. ✅ Frontend credentials kullanımı açıklandı

**Deploy sonrası:**
1. Railway logs'u kontrol edin (CORS log'larını görmelisiniz)
2. curl ile OPTIONS testini yapın
3. Browser console'da Network tab'i kontrol edin
4. Hala çalışmıyorsa yukarıdaki kontrol listesini takip edin
