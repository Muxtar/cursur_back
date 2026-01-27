# Railway MongoDB Servisi Kullanım Kılavuzu

## 🎯 Railway'in Kendi MongoDB Servisini Kullanma

Railway'de MongoDB Atlas yerine Railway'in kendi MongoDB servisini kullanabilirsiniz.

---

## 📋 Adım Adım Kurulum

### 1️⃣ Railway'de MongoDB Servisi Ekleme

1. **Railway Dashboard'a gidin:**
   - https://railway.app adresine gidin
   - Projenize tıklayın

2. **MongoDB Servisi Ekleyin:**
   - **"New"** butonuna tıklayın
   - **"Database"** → **"Add MongoDB"** seçin
   - MongoDB servisi oluşturulacak

3. **MongoDB Servisini Backend'e Bağlayın:**
   - MongoDB servisine tıklayın
   - **"Connect"** butonuna tıklayın
   - Backend servisinizi seçin
   - Railway otomatik olarak environment variable'ları ekleyecek

---

### 2️⃣ Railway'in Sağladığı Environment Variables

Railway MongoDB servisi şu environment variable'ları otomatik olarak sağlar:

```
MONGO_URL=mongodb://mongo:password@mongodb.railway.internal:27017
MONGO_PUBLIC_URL=mongodb://mongo:password@gondola.proxy.rlwy.net:16955
MONGOUSER=mongo
MONGOPASSWORD=password
MONGOHOST=mongodb.railway.internal
MONGOPORT=27017
```

**Önemli:** Railway'in `MONGO_URL` değişkeni database adı içermez. Kod otomatik olarak ekler.

---

### 3️⃣ Backend Servisinde Ayarlar

Backend servisinizde şu environment variable'ları ayarlayın:

#### Zorunlu:
- `MONGO_URL` - Railway otomatik olarak ekler (MongoDB servisini bağladığınızda)
- `MONGODB_DB` - Database adı (opsiyonel, default: `chat_app`)

#### Nasıl Ayarlanır:

1. **Backend servisinize gidin:**
   - Railway Dashboard → Backend servisiniz

2. **Variables sekmesine gidin:**
   - Üst menüden **"Variables"** sekmesine tıklayın

3. **MONGODB_DB ekleyin (opsiyonel):**
   - **"New Variable"** butonuna tıklayın
   - **Variable Name:** `MONGODB_DB`
   - **Value:** `chat_app` (veya istediğiniz database adı)
   - **"Add"** butonuna tıklayın

   ⚠️ **Not:** Eğer `MONGODB_DB` ayarlamazsanız, default olarak `chat_app` kullanılır.

---

### 4️⃣ Kod Otomatik Olarak Yapılandırır

Kod şu şekilde çalışır:

1. `MONGO_URL` environment variable'ını kontrol eder
2. Eğer Railway MongoDB servisi kullanılıyorsa (`railway.internal` veya `proxy.rlwy.net` içeriyorsa):
   - Connection string'e database adını otomatik ekler
   - `MONGODB_DB` variable'ından database adını alır (yoksa `chat_app`)

**Örnek:**
- Railway'in sağladığı: `mongodb://mongo:password@mongodb.railway.internal:27017`
- Kod otomatik olarak ekler: `mongodb://mongo:password@mongodb.railway.internal:27017/chat_app`

---

### 5️⃣ Deploy ve Test

1. **Railway otomatik deploy eder:**
   - Environment variable değişiklikleri otomatik olarak deploy'u tetikler

2. **Logları kontrol edin:**
   - Railway Dashboard → Backend servisi → **"Deployments"** sekmesi
   - En son deployment'a tıklayın → **"View Logs"**
   - Şu mesajları görmelisiniz:
   ```
   MongoDB URI: mongodb://mongo:***@mongodb.railway.internal:27017/chat_app
   Added database name to Railway MongoDB connection string: chat_app
   MongoDB Database: chat_app
   ✅ MongoDB connected successfully
   ```

---

## ✅ Kontrol Listesi

- [ ] Railway'de MongoDB servisi ekledim
- [ ] MongoDB servisini backend servisine bağladım (Connect butonu ile)
- [ ] Backend servisinde `MONGO_URL` variable'ının otomatik eklendiğini kontrol ettim
- [ ] (Opsiyonel) `MONGODB_DB` variable'ını ekledim (default: `chat_app`)
- [ ] Railway'de deploy ettim
- [ ] Loglarda "MongoDB connected successfully" mesajını görüyorum

---

## 🔍 Troubleshooting

### Hata: "MongoDB URI not set"
**Çözüm:** MongoDB servisini backend servisine bağladığınızdan emin olun (Connect butonu ile).

### Hata: "Failed to connect to MongoDB"
**Kontrol edin:**
1. MongoDB servisinin çalıştığını kontrol edin
2. MongoDB servisini backend servisine bağladığınızdan emin olun
3. `MONGO_URL` variable'ının backend servisinde olduğunu kontrol edin

### Database adı eklenmiyor
**Çözüm:** `MONGODB_DB` variable'ını backend servisinde ayarlayın veya kod otomatik olarak `chat_app` kullanacaktır.

---

## 📝 Örnek Environment Variables

Backend servisinizde şu variable'lar olmalı:

```
MONGO_URL=mongodb://mongo:TnQgsVJBqKAmPXyiiKvcoGKRqrCNVykk@mongodb.railway.internal:27017
MONGODB_DB=chat_app
```

**Not:** `MONGO_URL` Railway tarafından otomatik eklenir (MongoDB servisini bağladığınızda).

---

## 🎉 Başarılı!

Eğer loglarda "✅ MongoDB connected successfully" mesajını görüyorsanız, bağlantı başarılı demektir!

---

## 💡 Railway MongoDB vs MongoDB Atlas

**Railway MongoDB:**
- ✅ Railway içinde, daha hızlı bağlantı
- ✅ Otomatik yedekleme
- ✅ Kolay kurulum
- ✅ Railway'in kendi network'ünde çalışır

**MongoDB Atlas:**
- ✅ Daha fazla özellik
- ✅ Global cluster desteği
- ✅ Daha fazla storage seçeneği
- ⚠️ Railway'de DNS sorunları olabilir (`mongodb+srv://` kullanmayın!)

Her ikisi de çalışır, Railway MongoDB daha kolay kurulum sağlar.
