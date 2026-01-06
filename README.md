# Chat Backend

Backend API for the chat application built with Go.

## Features

- User authentication with phone numbers
- QR code generation for contact sharing
- Real-time messaging via WebSocket
- File uploads (images, audio, icons)
- Group chats
- Video/voice calling
- Location-based user discovery
- Proposal/offer system
- Contact management

## Tech Stack

- Go 1.21+
- Gin (HTTP framework)
- MongoDB (primary database)
- Redis (caching)
- PostgreSQL (relational data)
- WebSocket (real-time communication)

## Setup

1. Install dependencies:
```bash
go mod download
```

2. Copy `.env.example` to `.env` and configure:
```bash
cp .env.example .env
```

3. Make sure MongoDB, Redis, and PostgreSQL are running

4. Run the server:
```bash
go run main.go
```

The server will start on port 8080 by default.

## Railway Deployment (Ayrı Proje Olarak)

Bu back-end projesi Railway'de ayrı bir servis olarak deploy edilmelidir.

### Önemli Kurulum Adımları:

1. **Git Repository Oluştur:**
   - Back-end klasörünü ayrı bir Git repository'sine yükleyin
   - Railway'de yeni bir proje oluşturun ve bu repository'yi bağlayın

2. **Database Servisleri Ekleme (ÖNEMLİ):**
   
   Railway'de back-end servisinize **MUTLAKA** şu database servislerini eklemeniz gerekiyor:
   
   **a) PostgreSQL Ekleme:**
   - Railway dashboard'da projenize gidin
   - "New" butonuna tıklayın
   - "Database" > "Add PostgreSQL" seçin
   - PostgreSQL servisi oluşturulacak
   - PostgreSQL servisinin ayarlarına gidin
   - "Variables" sekmesinde otomatik olarak şu değişkenler oluşturulur:
     - `PGHOST` (veya `POSTGRES_HOST`)
     - `PGPORT` (veya `POSTGRES_PORT`)
     - `PGUSER` (veya `POSTGRES_USER`)
     - `PGPASSWORD` (veya `POSTGRES_PASSWORD`)
     - `PGDATABASE` (veya `POSTGRES_DB`)
   - PostgreSQL servisini back-end servisinize bağlayın (Connect butonu ile)
   
   **b) MongoDB Ekleme:**
   - Railway'de "New" > "Database" > "Add MongoDB" seçin
   - MongoDB servisi oluşturulacak
   - MongoDB servisinin "Variables" sekmesinde `MONGO_URL` veya `MONGODB_URI` değişkenini bulun
   - MongoDB servisini back-end servisinize bağlayın
   
   **c) Redis Ekleme (Opsiyonel):**
   - Redis kullanmak isterseniz: "New" > "Database" > "Add Redis"
   - Redis servisini back-end servisinize bağlayın
   - Veya `REDIS_ENABLED=false` olarak ayarlayın

3. **Root Directory Ayarları:**
   - Railway service ayarlarına gidin
   - "Source" bölümünde **Root Directory** boş bırakın (zaten root'ta olduğu için)
   - Veya `.` olarak ayarlayın

4. **Environment Variables (Railway Dashboard'da Ayarlayın):**
   ```env
   PORT=8080
   MONGODB_URI=your-mongodb-connection-string
   MONGODB_DB=chat_app
   
   # Redis (Optional - set REDIS_ENABLED=false to disable)
   REDIS_ENABLED=true
   REDIS_HOST=your-redis-host
   REDIS_PORT=6379
   REDIS_PASSWORD=your-redis-password
   
   POSTGRES_HOST=your-postgres-host
   POSTGRES_PORT=5432
   POSTGRES_USER=your-postgres-user
   POSTGRES_PASSWORD=your-postgres-password
   POSTGRES_DB=chat_app
   JWT_SECRET=your-very-secure-secret-key
   JWT_EXPIRATION=24h
   UPLOAD_DIR=./uploads
   MAX_FILE_SIZE=10485760
   CORS_ALLOWED_ORIGINS=https://your-frontend-domain.com,https://www.your-frontend-domain.com
   ```
   
   **ÖNEMLİ NOTLAR:**
   - **PostgreSQL ve MongoDB ZORUNLUDUR** - Uygulama bu olmadan çalışmaz
   - Railway'de database servislerini ekledikten sonra, Railway otomatik olarak environment variable'ları ayarlar
   - Eğer Railway otomatik olarak ayarlamazsa, database servisinin "Variables" sekmesinden değerleri kopyalayıp back-end servisine ekleyin
   - `CORS_ALLOWED_ORIGINS` değişkenine front-end domain'inizi ekleyin
   - **Redis opsiyoneldir**. Eğer Redis servisi kurulu değilse, `REDIS_ENABLED=false` olarak ayarlayın
   - Redis olmadan da uygulama çalışır, ancak bazı özellikler sınırlı olur (QR kod cache, verification code, typing indicators)
   - Redis bağlantısı başarısız olursa, uygulama 5 kez deneyecek ve sonra Redis olmadan devam edecek
   - GoDaddy domain'inizi front-end'e bağladıktan sonra `CORS_ALLOWED_ORIGINS`'e ekleyin
   - Örnek: `https://yourdomain.com,https://www.yourdomain.com`
   - Birden fazla domain varsa virgülle ayırın

5. **Build ve Deploy:**
   - Railway otomatik olarak `Dockerfile`'ı algılayacak
   - Build işlemi otomatik başlayacak
   - Deploy tamamlandıktan sonra Railway size bir URL verecek (örn: `https://your-backend-app.railway.app`)
   - Bu URL'yi front-end projesindeki environment variable'lara ekleyeceksiniz

6. **Troubleshooting:**
   
   **PostgreSQL Bağlantı Hatası:**
   - `Failed to connect to PostgreSQL` hatası alırsanız:
     - Railway'de PostgreSQL servisi eklediğinizden emin olun
     - PostgreSQL servisini back-end servisinize bağladığınızdan emin olun
     - Back-end servisinin "Variables" sekmesinde PostgreSQL environment variable'larının olduğunu kontrol edin
     - Railway bazen `PGHOST`, `PGPORT` gibi değişkenler kullanır, bunları `POSTGRES_HOST`, `POSTGRES_PORT` olarak manuel eklemeniz gerekebilir
   
   **MongoDB Bağlantı Hatası:**
   - `Failed to connect to MongoDB` hatası alırsanız:
     - Railway'de MongoDB servisi eklediğinizden emin olun
     - MongoDB servisini back-end servisinize bağladığınızdan emin olun
     - `MONGODB_URI` değişkeninin doğru ayarlandığını kontrol edin
   
   **Build Hatası:**
   - Eğer `"/go.mod": not found` hatası alırsanız:
     - Root Directory'nin boş veya `.` olduğundan emin olun
     - `go.mod` ve `go.sum` dosyalarının Git'e commit edildiğinden emin olun
     - Railway build cache'ini temizleyin
     - Service'in doğru branch/commit'e işaret ettiğini kontrol edin

## API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login
- `GET /api/v1/auth/qr/:user_id` - Get user QR code

### Users
- `GET /api/v1/users/me` - Get current user
- `PUT /api/v1/users/me` - Update user
- `PUT /api/v1/users/location` - Update location
- `GET /api/v1/users/nearby` - Get nearby users

### Contacts
- `GET /api/v1/contacts` - Get contacts
- `POST /api/v1/contacts/scan` - Scan QR code to add contact
- `DELETE /api/v1/contacts/:contact_id` - Delete contact

### Chats
- `GET /api/v1/chats` - Get all chats
- `POST /api/v1/chats` - Create chat
- `GET /api/v1/chats/:chat_id` - Get chat
- `GET /api/v1/chats/:chat_id/messages` - Get messages
- `POST /api/v1/chats/:chat_id/messages` - Send message
- `DELETE /api/v1/chats/messages/:message_id` - Delete message

### Groups
- `POST /api/v1/groups` - Create group
- `GET /api/v1/groups` - Get groups
- `GET /api/v1/groups/:group_id` - Get group
- `PUT /api/v1/groups/:group_id` - Update group
- `DELETE /api/v1/groups/:group_id` - Delete group
- `POST /api/v1/groups/:group_id/members` - Add member
- `DELETE /api/v1/groups/:group_id/members/:member_id` - Remove member

### Proposals
- `POST /api/v1/proposals` - Create proposal
- `GET /api/v1/proposals` - Get proposals
- `PUT /api/v1/proposals/:proposal_id/accept` - Accept proposal
- `PUT /api/v1/proposals/:proposal_id/reject` - Reject proposal

### Calls
- `POST /api/v1/calls` - Initiate call
- `POST /api/v1/calls/:call_id/answer` - Answer call
- `POST /api/v1/calls/:call_id/end` - End call

### Files
- `POST /api/v1/files/upload` - Upload file
- `GET /api/v1/files/:filename` - Get file

### WebSocket
- `GET /ws?token=<token>` - WebSocket connection





