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
- MongoDB (primary database) - **MongoDB Atlas recommended**
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

3. Make sure MongoDB is running (local MongoDB or MongoDB Atlas)

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
   
   **b) MongoDB Atlas Bağlantısı (ÖNERİLEN):**
   
   ⚠️ **ÖNEMLİ: Railway'de `mongodb+srv://` kullanmayın!**
   
   Railway'de MongoDB Atlas'a bağlanırken **Standard Connection String** kullanmalısınız:
   
   1. **MongoDB Atlas'tan Standard Connection String Alın:**
      - MongoDB Atlas Dashboard → Cluster'ınıza tıklayın → **Connect**
      - **"Connect your application"** seçeneğini seçin
      - **"Standard connection string"** seçeneğini seçin (SRV değil!)
      - Connection string'i kopyalayın
   
   2. **Connection String Formatı:**
      ```
      mongodb://username:password@cluster0-shard-00-00.xxxxx.mongodb.net:27017,cluster0-shard-00-01.xxxxx.mongodb.net:27017,cluster0-shard-00-02.xxxxx.mongodb.net:27017/chat_app?ssl=true&replicaSet=atlas-xxxxx-shard-0&authSource=admin&retryWrites=true&w=majority
      ```
      
      **Örnek tam connection string:**
      ```
      mongodb://muxtarbayramov92:ZcbRm9j6ISIwTmIg@cluster0-shard-00-00.g2e8hv9.mongodb.net:27017,cluster0-shard-00-01.g2e8hv9.mongodb.net:27017,cluster0-shard-00-02.g2e8hv9.mongodb.net:27017/chat_app?ssl=true&replicaSet=atlas-xxxxx-shard-0&authSource=admin&retryWrites=true&w=majority
      ```
   
   3. **Railway'de Ayarlayın:**
      - Back-end servisinizin **Variables** sekmesine gidin
      - `MONGODB_URI` variable'ını ekleyin veya güncelleyin
      - Value olarak yukarıdaki standard connection string'i yapıştırın
      - `MONGODB_DB` variable'ını da ekleyin (opsiyonel, default: `chat_app`)
   
   4. **MongoDB Atlas Network Access:**
      - MongoDB Atlas Dashboard → **Network Access**
      - **Add IP Address** → **Allow Access from Anywhere** (`0.0.0.0/0`)
      - ⚠️ Development için geçici olarak tüm IP'lere izin verin
      - Production'da sadece Railway IP'lerini ekleyin
   
   **❌ KULLANMAYIN (Railway'de DNS sorunlarına yol açar):**
   ```
   mongodb+srv://user:pass@cluster0.xxxxx.mongodb.net/?options
   ```
   
   **✅ KULLANIN (Railway'de çalışır):**
   ```
   mongodb://user:pass@cluster0-shard-00-00.xxxxx.mongodb.net:27017,cluster0-shard-00-01.xxxxx.mongodb.net:27017,cluster0-shard-00-02.xxxxx.mongodb.net:27017/chat_app?ssl=true&replicaSet=atlas-xxxxx-shard-0&authSource=admin&retryWrites=true&w=majority
   ```
   

3. **Root Directory Ayarları:**
   - Railway service ayarlarına gidin
   - "Source" bölümünde **Root Directory** boş bırakın (zaten root'ta olduğu için)
   - Veya `.` olarak ayarlayın

4. **Environment Variables (Railway Dashboard'da Ayarlayın):**
   ```env
   PORT=8080
   
   # MongoDB (ZORUNLU) - Standard connection string kullanın (mongodb://, NOT mongodb+srv://)
   MONGODB_URI=mongodb://user:pass@cluster0-shard-00-00.xxxxx.mongodb.net:27017,cluster0-shard-00-01.xxxxx.mongodb.net:27017,cluster0-shard-00-02.xxxxx.mongodb.net:27017/chat_app?ssl=true&replicaSet=atlas-xxxxx-shard-0&authSource=admin&retryWrites=true&w=majority
   MONGODB_DB=chat_app
   
   JWT_SECRET=your-very-secure-secret-key
   JWT_EXPIRATION=24h
   UPLOAD_DIR=./uploads
   MAX_FILE_SIZE=10485760
   CORS_ALLOWED_ORIGINS=https://your-frontend-domain.com,https://www.your-frontend-domain.com
   ```
   
   **Railway MongoDB Setup Checklist:**
   
   ✅ **Variables Sekmesi:**
   - [ ] `MONGODB_URI` - Standard connection string (mongodb:// formatında)
   - [ ] `MONGODB_DB` - Database adı (opsiyonel, default: `chat_app`)
   
   ✅ **MongoDB Atlas Network Access:**
   - [ ] MongoDB Atlas Dashboard → Network Access
   - [ ] Add IP Address → Allow Access from Anywhere (`0.0.0.0/0`)
   - [ ] ⚠️ Development için geçici olarak tüm IP'lere izin verin
   - [ ] Production'da sadece Railway IP'lerini ekleyin (least privilege)
   
   ✅ **Connection String Özellikleri:**
   - [ ] `mongodb://` ile başlamalı (SRV değil!)
   - [ ] Host adresleri `cluster0-shard-00-00.xxxxx.mongodb.net:27017` formatında olmalı
   - [ ] Port numaraları (`:27017`) belirtilmiş olmalı
   - [ ] Database adı (`/chat_app`) connection string'de olmalı
   - [ ] Şifre özel karakter içeriyorsa URL-encoded olmalı
   
   ✅ **Özel Karakterler İçin:**
   - Railway Variables'da şifre özel karakter içeriyorsa:
     - URL-encode edin (örn: `@` → `%40`, `#` → `%23`)
     - Veya Railway'de variable'ı raw string olarak ayarlayın
   
   **ÖNEMLİ NOTLAR:**
   - **MongoDB ZORUNLUDUR** - Uygulama bu olmadan çalışmaz
   - `mongodb+srv://` kullanmayın - Railway'de DNS sorunlarına yol açar
   - Standard connection string kullanın (`mongodb://`)
   - `CORS_ALLOWED_ORIGINS` değişkenine front-end domain'inizi ekleyin
   - Birden fazla domain varsa virgülle ayırın

5. **Build ve Deploy:**
   - Railway otomatik olarak `Dockerfile`'ı algılayacak
   - Build işlemi otomatik başlayacak
   - Deploy tamamlandıktan sonra Railway size bir URL verecek (örn: `https://your-backend-app.railway.app`)
   - Bu URL'yi front-end projesindeki environment variable'lara ekleyeceksiniz

6. **Troubleshooting:**
   
   **MongoDB Bağlantı Hatası:**
   - `Failed to connect to MongoDB` veya `lookup _mongodb._tcp... server misbehaving` hatası alırsanız:
     - ⚠️ **DNS SRV Hatası:** `mongodb+srv://` kullanıyorsanız, standard `mongodb://` connection string kullanın
     - `MONGODB_URI` değişkeninin standard format (`mongodb://`) olduğunu kontrol edin
     - MongoDB Atlas Network Access'te `0.0.0.0/0` (tüm IP'ler) ekli olduğunu kontrol edin
     - Connection string'de database adının (`/chat_app`) olduğunu kontrol edin
     - Şifre özel karakter içeriyorsa URL-encoded olduğunu kontrol edin
     - MongoDB Atlas cluster'ının çalıştığını kontrol edin
   
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





