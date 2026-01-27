# CORS Preflight Problemi - Tam Çözüm Dokümantasyonu

## 🔍 Sorunun Kök Nedenleri

1. **Preflight (OPTIONS) İstekleri Doğru Handle Edilmiyor**
   - Browser, POST/PUT gibi "complex" request'lerden önce OPTIONS gönderir
   - Backend OPTIONS'a 204/200 dönmeli ve CORS header'larını set etmeli
   - Origin kontrolü yapılmadan header'lar set edilmemeli (güvenlik açığı)

2. **Vary: Origin Header Eksik**
   - Cache kontrolü için kritik
   - Browser ve proxy'lerin doğru cache davranışı için gerekli
   - CORS response'larının cache'lenmesini önler

3. **Origin Kontrolü Eksik veya Yanlış**
   - `AllowCredentials: true` kullanıldığında `AllowAllOrigins: true` kullanılamaz
   - Origin "*" ile credentials birlikte çalışmaz (browser güvenlik kuralı)
   - Spesifik origin'ler kullanılmalı: `https://www.fridpass.com`
   - Origin kontrolü yapılmadan header'lar set edilmemeli

4. **Railway PORT ve Listen Adresi**
   - Server mutlaka `os.Getenv("PORT")` ile port alsın
   - Listen adresi `0.0.0.0:port` olmalı (tüm interface'lerden bağlantı kabul etmek için)

5. **CORS Middleware Sırası**
   - CORS middleware route handler'lardan ÖNCE olmalı
   - OPTIONS handler'ı en başta olmalı (preflight'ları yakalamak için)

---

## ✅ ÇÖZÜM A: Gin Framework (Mevcut Proje)

### 1. CORS Middleware (Güncellenmiş)

**Dosya:** `back-end/internal/middleware/cors.go`

```go
package middleware

import (
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
	
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		
		// Always set Vary: Origin header for proper cache control
		// This tells caches that the response varies based on the Origin header
		c.Header("Vary", "Origin")
		
		// Handle preflight (OPTIONS) requests
		if c.Request.Method == http.MethodOptions {
			// Only set CORS headers if origin is allowed
			if isOriginAllowed(origin, allowedOrigins) {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
				c.Header("Access-Control-Max-Age", "86400") // 24 hours
				c.AbortWithStatus(http.StatusNoContent) // 204
			} else {
				// Origin not allowed - return 403 Forbidden
				c.AbortWithStatus(http.StatusForbidden)
			}
			return
		}
		
		// Handle regular requests
		if isOriginAllowed(origin, allowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		// If origin is not allowed, don't set CORS headers
		// Browser will block the request automatically
		
		c.Next()
	}
}
```

### 2. main.go Güncellemesi

**Dosya:** `back-end/main.go`

```go
// ... existing code ...

// ===== CORS CONFIGURATION =====
// Apply CORS middleware FIRST (before routes)
// This ensures preflight (OPTIONS) requests are handled correctly
r.Use(middleware.CORSMiddleware())
log.Println("✅ CORS middleware configured")
// ===== END CORS CONFIGURATION =====

// Setup routes (AFTER CORS middleware)
router.SetupRoutes(r, db, hub, cfg)

port := os.Getenv("PORT")
if port == "" {
	port = "8080"
}

// Listen on 0.0.0.0 to accept connections from all interfaces
// This is required for Railway and other cloud platforms
listenAddr := "0.0.0.0:" + port
log.Printf("Server starting on %s", listenAddr)
if err := r.Run(listenAddr); err != nil {
	log.Fatal("Failed to start server:", err)
}
```

---

## ✅ ÇÖZÜM B: net/http (Framework Yoksa)

Eğer Gin kullanmıyorsanız, `net/http` için CORS middleware:

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
	
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		
		// Always set Vary: Origin header
		w.Header().Set("Vary", "Origin")
		
		// Handle preflight (OPTIONS) requests
		if r.Method == http.MethodOptions {
			if isOriginAllowed(origin, allowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
				w.Header().Set("Access-Control-Max-Age", "86400")
				w.WriteHeader(http.StatusNoContent) // 204
			} else {
				w.WriteHeader(http.StatusForbidden) // 403
			}
			return
		}
		
		// Handle regular requests
		if isOriginAllowed(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Your handlers
	mux := http.NewServeMux()
	
	// Apply CORS middleware
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
		w.Write([]byte(`{"message": "Code sent"}`))
	})
	
	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	// Listen on 0.0.0.0:port
	listenAddr := "0.0.0.0:" + port
	log.Printf("Server starting on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, handler); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
```

---

## 🌐 Next.js Frontend Tarafı

### 1. Environment Variable

**Dosya:** `front-end/.env.local` (veya Railway environment variables)

```bash
NEXT_PUBLIC_API_URL=https://cursurback-production.up.railway.app/api/v1
```

### 2. API Client Örneği

**Dosya:** `front-end/src/lib/api.ts` (zaten mevcut ve doğru)

```typescript
// API client with credentials support
class ApiClient {
  private baseURL: string;
  private token: string | null = null;

  constructor(baseURL: string) {
    this.baseURL = baseURL;
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('token');
    }
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string> || {}),
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    try {
      const response = await fetch(url, {
        ...options,
        headers,
        credentials: 'include', // ✅ Include cookies if using credentials
      });

      // ... rest of the code
    } catch (error) {
      // ... error handling
    }
  }

  async post<T>(endpoint: string, data?: any): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }
}

// Usage
export const authApi = {
  sendCode: (phoneNumber: string) =>
    api.post('/auth/send-code', { phone_number: phoneNumber }),
};
```

### 3. Direct Fetch Örneği (Alternatif)

```typescript
// Direct fetch example
const sendCode = async (phoneNumber: string) => {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'https://cursurback-production.up.railway.app/api/v1';
  
  const response = await fetch(`${apiUrl}/auth/send-code`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      // Authorization header if needed
      // 'Authorization': `Bearer ${token}`,
    },
    credentials: 'include', // ✅ Important: include cookies
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

## 🧪 Test Komutları

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

## 🔧 Railway Environment Variables

Railway dashboard'da şu environment variable'ları ayarlayın:

```bash
PORT=8080  # Railway otomatik set eder, ama kontrol için
CORS_ALLOWED_ORIGINS=https://www.fridpass.com,http://localhost:3000
```

**Not:** Eğer `CORS_ALLOWED_ORIGINS` set edilmezse, default olarak:
- `https://www.fridpass.com`
- `http://localhost:3000`

kullanılacaktır.

---

## ✅ Kontrol Listesi

- [x] CORS middleware route handler'lardan ÖNCE eklendi
- [x] OPTIONS (preflight) istekleri handle ediliyor
- [x] Origin kontrolü yapılıyor
- [x] `Vary: Origin` header'ı set ediliyor
- [x] `Access-Control-Allow-Credentials: true` sadece allowed origin'ler için
- [x] `Access-Control-Allow-Origin` ASLA "*" değil, spesifik origin
- [x] Railway PORT environment variable'dan alınıyor
- [x] Listen adresi `0.0.0.0:port` olarak ayarlandı
- [x] Frontend `credentials: 'include'` kullanıyor
- [x] Frontend API URL environment variable'dan geliyor

---

## 🐛 Debug Adımları

1. **Browser Console'da CORS Hatası Görüyorsanız:**
   - Network tab'de OPTIONS isteğini kontrol edin
   - Response headers'da CORS header'larının olup olmadığını kontrol edin
   - Origin header'ının doğru gönderildiğini kontrol edin

2. **Backend Log'larını Kontrol Edin:**
   - Railway logs'da OPTIONS isteklerinin geldiğini görün
   - CORS middleware'in çalıştığını doğrulayın

3. **curl Testleri:**
   - Önce OPTIONS testini yapın
   - Sonra POST testini yapın
   - Her ikisinde de CORS header'larının geldiğini doğrulayın

4. **Environment Variables:**
   - Railway'de `CORS_ALLOWED_ORIGINS` doğru set edilmiş mi?
   - Frontend'de `NEXT_PUBLIC_API_URL` doğru set edilmiş mi?

---

## 📝 Özet

Bu çözüm:
1. ✅ Preflight (OPTIONS) isteklerini doğru handle ediyor
2. ✅ Origin kontrolü yapıyor ve güvenli
3. ✅ `Vary: Origin` header'ı set ediyor
4. ✅ Credentials ile çalışıyor
5. ✅ Railway'de çalışacak şekilde yapılandırılmış
6. ✅ Production-ready ve güvenli

**Deploy sonrası test edin:**
```bash
# Preflight test
curl -i -X OPTIONS 'https://cursurback-production.up.railway.app/api/v1/auth/send-code' \
  -H 'Origin: https://www.fridpass.com' \
  -H 'Access-Control-Request-Method: POST'

# POST test
curl -i -X POST 'https://cursurback-production.up.railway.app/api/v1/auth/send-code' \
  -H 'Origin: https://www.fridpass.com' \
  -H 'Content-Type: application/json' \
  --data '{"phone_number":"+994516480030"}'
```
