# CORS Preflight Sorunu - Kök Neden ve Kesin Çözüm

## 🔍 KÖK NEDENLER (Repo Koduna Göre)

### 1. **docker-compose.yml'de `CORS_ALLOWED_ORIGINS: "*"` Kullanımı**
   - **Dosya:** `back-end/docker-compose.yml` (satır 94)
   - **Sorun:** `"*"` wildcard kullanımı credentials (cookie) ile çalışmaz
   - **Kanıt:** Frontend'de `credentials: 'include'` kullanılıyor (`front-end/src/lib/api.ts:81`)
   - **Browser Güvenlik Kuralı:** `Access-Control-Allow-Origin: "*"` ile `Access-Control-Allow-Credentials: true` birlikte kullanılamaz

### 2. **CORS Middleware'de "*" Kontrolü Eksikti**
   - **Dosya:** `back-end/internal/middleware/cors.go`
   - **Sorun:** Eğer `CORS_ALLOWED_ORIGINS="*"` gelirse, middleware bunu kabul ediyordu ama credentials ile çalışmıyordu
   - **Sonuç:** Browser preflight isteğini reddediyordu

### 3. **Credentials Her Zaman Açık**
   - **Dosya:** `back-end/internal/middleware/cors.go` (satır 86, 103)
   - **Sorun:** `Access-Control-Allow-Credentials: true` her zaman set ediliyordu, "*" kullanımında bu geçersiz

### 4. **Railway Environment Variable Eksik**
   - Railway dashboard'da `CORS_ALLOWED_ORIGINS` set edilmemiş olabilir
   - docker-compose.yml'deki "*" değeri production'a taşınmış olabilir

---

## ✅ DÜZELTİLMİŞ GO/GIN CORS MIDDLEWARE KODU

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
```

---

## 🔧 RAILWAY ENV ÖNERİSİ

**Railway Dashboard → Backend Project → Variables:**

```bash
# CRITICAL: "*" is NOT compatible with credentials
# Use specific origins separated by commas
CORS_ALLOWED_ORIGINS=https://www.fridpass.com

# Optional: Add localhost for local development
# CORS_ALLOWED_ORIGINS=https://www.fridpass.com,http://localhost:3000

# Credentials enabled (default: true)
CORS_ALLOW_CREDENTIALS=true

# Port (Railway automatically sets this, but you can override)
PORT=8080
```

**ÖNEMLİ:**
- `CORS_ALLOWED_ORIGINS` ASLA `"*"` olmamalı (credentials kullanılıyorsa)
- Spesifik origin kullan: `https://www.fridpass.com`
- Dev için opsiyonel: `https://www.fridpass.com,http://localhost:3000`

---

## 🧪 CURL İLE KANIT TESTLERİ

### A) Preflight (OPTIONS) Testi

```bash
curl -i -X OPTIONS 'https://cursurback-production.up.railway.app/api/v1/auth/send-code' \
  -H 'Origin: https://www.fridpass.com' \
  -H 'Access-Control-Request-Method: POST' \
  -H 'Access-Control-Request-Headers: content-type,authorization'
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

### B) POST İsteği Testi

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

**Railway Logs'da Görmeli:**
```
🌐 CORS Middleware: POST /api/v1/auth/send-code | Origin: https://www.fridpass.com
```

---

## 🌐 NEXT.JS (Ayrı Proje) Örnek İstek

### Environment Variable

**Railway Frontend Project → Variables:**

```bash
NEXT_PUBLIC_API_URL=https://cursurback-production.up.railway.app/api/v1
```

### Credentials KULLANILIYORSA (Mevcut Durum)

**Frontend'de zaten doğru:** `front-end/src/lib/api.ts:81`

```typescript
// front-end/src/lib/api.ts
const response = await fetch(url, {
  ...options,
  headers,
  credentials: 'include', // ✅ Cookie göndermek için gerekli
});
```

**Backend'de:**
- `Access-Control-Allow-Credentials: true` ✅ (zaten var)
- `Access-Control-Allow-Origin` spesifik origin ✅ (ASLA "*" değil)

**Örnek Kullanım:**
```typescript
const sendCode = async (phoneNumber: string) => {
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

### Credentials KULLANILMIYORSA (Alternatif)

**Eğer cookie kullanmıyorsanız:**

```typescript
const sendCode = async (phoneNumber: string) => {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'https://cursurback-production.up.railway.app/api/v1';
  
  const response = await fetch(`${apiUrl}/auth/send-code`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    // credentials: 'include' KALDIRIN
    body: JSON.stringify({ phone_number: phoneNumber }),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Request failed');
  }

  return response.json();
};
```

**Backend'de:**
- `CORS_ALLOW_CREDENTIALS=false` set edin
- Veya `CORS_ALLOW_CREDENTIALS` env var'ını kaldırın (default true ama "*" kullanabilirsiniz - önerilmez)

---

## 📝 ÖZET DEĞİŞİKLİKLER

### 1. CORS Middleware Güncellendi
- ✅ "*" kontrolü eklendi
- ✅ Credentials kontrolü eklendi
- ✅ "*" ile credentials birlikte kullanılamaz kontrolü eklendi
- ✅ Log'lar eklendi (kanıt için)

### 2. docker-compose.yml Güncellendi
- ✅ `CORS_ALLOWED_ORIGINS: "*"` → `CORS_ALLOWED_ORIGINS: "https://www.fridpass.com,http://localhost:3000"`
- ✅ `CORS_ALLOW_CREDENTIALS: "true"` eklendi

### 3. main.go Güncellendi
- ✅ Startup log'u güncellendi (CORS origins gösteriliyor)

---

## ✅ SONUÇ

**Sorun:** docker-compose.yml'de `"*"` kullanımı + credentials kullanımı = Browser CORS hatası

**Çözüm:**
1. ✅ CORS middleware "*" kontrolü yapıyor
2. ✅ Credentials sadece spesifik origin'lerle çalışıyor
3. ✅ docker-compose.yml düzeltildi
4. ✅ Railway env önerisi verildi
5. ✅ Test komutları ve beklenen çıktılar verildi

**Sonraki Adımlar:**
1. Railway dashboard'da `CORS_ALLOWED_ORIGINS=https://www.fridpass.com` set et
2. Backend'i deploy et
3. Railway logs'u kontrol et (CORS log'larını görmelisiniz)
4. curl ile OPTIONS testini yap
5. curl ile POST testini yap
6. Browser console'da Network tab'i kontrol et
