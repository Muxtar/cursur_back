# Railway'de MongoDB Atlas Bağlantısı Kurulum Kılavuzu

## 🎯 Amaç
Railway'de çalışan Go backend'inizi MongoDB Atlas'a bağlamak.

## ⚠️ ÖNEMLİ: mongodb+srv:// KULLANMAYIN!

Railway'de `mongodb+srv://` protokolü DNS sorunlarına yol açar. **Mutlaka standard `mongodb://` connection string kullanın.**

---

## 📋 Adım Adım Kurulum

### 1️⃣ MongoDB Atlas'tan Standard Connection String Alın

1. **MongoDB Atlas Dashboard'a gidin:**
   - https://cloud.mongodb.com adresine gidin
   - Giriş yapın

2. **Cluster'ınızı seçin:**
   - Sol menüden **"Database"** sekmesine tıklayın
   - Bağlanmak istediğiniz cluster'ınıza tıklayın

3. **Connect butonuna tıklayın:**
   - Cluster sayfasında **"Connect"** butonuna tıklayın

4. **"Connect your application" seçeneğini seçin:**
   - Açılan pencerede **"Connect your application"** seçeneğine tıklayın

5. **"Standard connection string" seçin:**
   - ⚠️ **ÖNEMLİ:** "Standard connection string" seçeneğini seçin
   - ❌ "SRV connection string" seçmeyin!

6. **Connection string'i kopyalayın:**
   - Connection string şu formatta olmalı:
   ```
   mongodb://username:password@cluster0-shard-00-00.xxxxx.mongodb.net:27017,cluster0-shard-00-01.xxxxx.mongodb.net:27017,cluster0-shard-00-02.xxxxx.mongodb.net:27017/?ssl=true&replicaSet=atlas-xxxxx-shard-0&authSource=admin&retryWrites=true&w=majority
   ```

7. **Database adını ekleyin:**
   - Connection string'in sonunda `?` öncesine database adını ekleyin
   - Örnek: `/chat_app` ekleyin
   - **Sonuç:**
   ```
   mongodb://username:password@cluster0-shard-00-00.xxxxx.mongodb.net:27017,cluster0-shard-00-01.xxxxx.mongodb.net:27017,cluster0-shard-00-02.xxxxx.mongodb.net:27017/chat_app?ssl=true&replicaSet=atlas-xxxxx-shard-0&authSource=admin&retryWrites=true&w=majority
   ```

---

### 2️⃣ MongoDB Atlas Network Access Ayarları

1. **MongoDB Atlas Dashboard → Network Access:**
   - Sol menüden **"Network Access"** sekmesine tıklayın

2. **IP Address ekleyin:**
   - **"Add IP Address"** butonuna tıklayın
   - **"Allow Access from Anywhere"** seçeneğini seçin
   - Bu `0.0.0.0/0` anlamına gelir (tüm IP'lere izin verir)
   - ⚠️ **Development için geçici olarak bu ayarı kullanabilirsiniz**
   - ⚠️ **Production'da sadece Railway IP'lerini ekleyin** (daha güvenli)

3. **Confirm butonuna tıklayın**

---

### 3️⃣ Railway'de Environment Variables Ayarlayın

1. **Railway Dashboard'a gidin:**
   - https://railway.app adresine gidin
   - Giriş yapın

2. **Backend servisinizi seçin:**
   - Projenizdeki backend servisine tıklayın

3. **Variables sekmesine gidin:**
   - Üst menüden **"Variables"** sekmesine tıklayın

4. **MONGODB_URI variable'ını ekleyin/güncelleyin:**
   - **"New Variable"** butonuna tıklayın
   - **Variable Name:** `MONGODB_URI`
   - **Value:** Yukarıda kopyaladığınız standard connection string'i yapıştırın
   - **Örnek:**
   ```
   mongodb://muxtarbayramov92:ZcbRm9j6ISIwTmIg@cluster0-shard-00-00.g2e8hv9.mongodb.net:27017,cluster0-shard-00-01.g2e8hv9.mongodb.net:27017,cluster0-shard-00-02.g2e8hv9.mongodb.net:27017/chat_app?ssl=true&replicaSet=atlas-xxxxx-shard-0&authSource=admin&retryWrites=true&w=majority
   ```
   - **"Add"** butonuna tıklayın

5. **MONGODB_DB variable'ını ekleyin (opsiyonel):**
   - **"New Variable"** butonuna tıklayın
   - **Variable Name:** `MONGODB_DB`
   - **Value:** `chat_app` (veya istediğiniz database adı)
   - **"Add"** butonuna tıklayın
   - ⚠️ **Not:** Eğer connection string'de database adı varsa (`/chat_app`), bu opsiyoneldir

---

### 4️⃣ Şifre Özel Karakter İçeriyorsa

Eğer MongoDB şifreniz özel karakterler içeriyorsa (`@`, `#`, `%`, vb.):

**Seçenek 1: URL-Encode edin**
- `@` → `%40`
- `#` → `%23`
- `%` → `%25`
- `&` → `%26`
- `=` → `%3D`

**Seçenek 2: Railway'de Raw String olarak ayarlayın**
- Railway Variables'da value'yu tırnak içine alın (genellikle gerekmez)

---

### 5️⃣ Deploy ve Test

1. **Railway'de deploy edin:**
   - Railway otomatik olarak yeni değişiklikleri deploy edecektir
   - Veya manuel olarak **"Deploy"** butonuna tıklayın

2. **Logları kontrol edin:**
   - Railway Dashboard → Backend servisi → **"Deployments"** sekmesi
   - En son deployment'a tıklayın
   - **"View Logs"** butonuna tıklayın
   - Şu mesajları görmelisiniz:
   ```
   MongoDB URI: mongodb://muxtarbayramov92:***@cluster0-shard-00-00.g2e8hv9.mongodb.net:27017,...
   MongoDB Database: chat_app
   ✅ MongoDB connected successfully
   ```

3. **Hata alırsanız:**
   - Logları kontrol edin
   - Hata mesajında DNS SRV hatası görüyorsanız, `mongodb+srv://` kullanıyorsunuz demektir
   - Standard `mongodb://` connection string kullandığınızdan emin olun

---

## ✅ Kontrol Listesi

- [ ] MongoDB Atlas'tan **Standard connection string** aldım (SRV değil!)
- [ ] Connection string `mongodb://` ile başlıyor (srv değil!)
- [ ] Connection string'de database adı var (`/chat_app`)
- [ ] Connection string'de port numaraları var (`:27017`)
- [ ] MongoDB Atlas Network Access'te `0.0.0.0/0` ekledim
- [ ] Railway'de `MONGODB_URI` variable'ını ekledim
- [ ] Railway'de `MONGODB_DB` variable'ını ekledim (opsiyonel)
- [ ] Şifre özel karakter içeriyorsa URL-encoded ettim
- [ ] Railway'de deploy ettim
- [ ] Loglarda "MongoDB connected successfully" mesajını görüyorum

---

## 🔍 Troubleshooting

### Hata: "lookup _mongodb._tcp... server misbehaving"
**Çözüm:** `mongodb+srv://` kullanıyorsunuz. Standard `mongodb://` connection string kullanın.

### Hata: "Failed to connect to MongoDB"
**Kontrol edin:**
1. MongoDB Atlas Network Access'te `0.0.0.0/0` ekli mi?
2. Connection string doğru mu? (`mongodb://` ile başlıyor mu?)
3. Şifre doğru mu? (Özel karakterler URL-encoded mi?)
4. Database adı connection string'de var mı? (`/chat_app`)

### Hata: "Authentication failed"
**Kontrol edin:**
1. MongoDB Atlas Database Access'te kullanıcı adı ve şifre doğru mu?
2. Kullanıcının database'e erişim izni var mı?

---

## 📝 Örnek Connection String Formatı

**✅ DOĞRU (Standard - Railway'de çalışır):**
```
mongodb://username:password@cluster0-shard-00-00.xxxxx.mongodb.net:27017,cluster0-shard-00-01.xxxxx.mongodb.net:27017,cluster0-shard-00-02.xxxxx.mongodb.net:27017/chat_app?ssl=true&replicaSet=atlas-xxxxx-shard-0&authSource=admin&retryWrites=true&w=majority
```

**❌ YANLIŞ (SRV - Railway'de çalışmaz):**
```
mongodb+srv://username:password@cluster0.xxxxx.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0
```

---

## 🎉 Başarılı!

Eğer loglarda "✅ MongoDB connected successfully" mesajını görüyorsanız, bağlantı başarılı demektir!
