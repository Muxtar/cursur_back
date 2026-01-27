# Railway MongoDB Atlas Setup Guide

## Sorun
MongoDB bağlantısı `localhost:27017` olarak görünüyor. Bu, Railway'de MongoDB URI environment variable'ının ayarlanmadığı anlamına gelir.

## Çözüm

### 1. MongoDB Atlas Connection String'i Hazırlayın

Mevcut connection string'iniz:
```
mongodb+srv://muxtarbayramov92:ZcbRm9j6ISIwTmIg@cluster0.g2e8hv9.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0
```

**Database adını ekleyin** (örn: `chat_app`):
```
mongodb+srv://muxtarbayramov92:ZcbRm9j6ISIwTmIg@cluster0.g2e8hv9.mongodb.net/chat_app?retryWrites=true&w=majority&appName=Cluster0
```

### 2. Railway'de Environment Variable Ayarlayın

1. Railway dashboard'a gidin: https://railway.app
2. Backend servisinizi seçin
3. **Variables** sekmesine tıklayın
4. **New Variable** butonuna tıklayın
5. Şu değişkenleri ekleyin:

#### Zorunlu:
- **Variable Name:** `MONGODB_URI`
- **Value:** `mongodb+srv://muxtarbayramov92:ZcbRm9j6ISIwTmIg@cluster0.g2e8hv9.mongodb.net/chat_app?retryWrites=true&w=majority&appName=Cluster0`

#### Opsiyonel (eğer farklı bir database adı kullanmak istiyorsanız):
- **Variable Name:** `MONGODB_DB`
- **Value:** `chat_app` (veya istediğiniz database adı)

### 3. MongoDB Atlas Network Access Ayarları

MongoDB Atlas'ta Railway'den bağlantıya izin vermek için:

1. MongoDB Atlas dashboard'a gidin
2. **Network Access** sekmesine tıklayın
3. **Add IP Address** butonuna tıklayın
4. **Allow Access from Anywhere** seçeneğini seçin (development için) veya Railway'in IP adreslerini ekleyin
5. **Confirm** butonuna tıklayın

**Not:** Production için sadece Railway IP'lerini eklemek daha güvenlidir.

### 4. Deploy ve Test

1. Railway'de backend servisinizi yeniden deploy edin
2. Logları kontrol edin - şu mesajları görmelisiniz:
   ```
   MongoDB URI: mongodb+srv://muxtarbayramov92:***@cluster0.g2e8hv9.mongodb.net/chat_app?...
   MongoDB Database: chat_app
   MongoDB connected successfully
   ```

### 5. Alternatif Environment Variable İsimleri

Kod şu environment variable'ları kontrol eder (sırayla):
1. `MONGODB_URI` (öncelikli)
2. `MONGO_URL` (alternatif)
3. `mongodb://localhost:27017` (default, sadece development için)

## Troubleshooting

### Hala `localhost:27017` görüyorsanız:
- Railway'de environment variable'ın doğru yazıldığından emin olun
- Variable'ın **saved** olduğundan emin olun
- Backend servisini yeniden deploy edin

### Connection timeout hatası alıyorsanız:
- MongoDB Atlas Network Access'te IP'nizin whitelist'te olduğundan emin olun
- Connection string'in doğru olduğundan emin olun
- MongoDB Atlas cluster'ının çalıştığından emin olun

### Authentication hatası alıyorsanız:
- MongoDB Atlas'ta kullanıcı adı ve şifrenin doğru olduğundan emin olun
- Kullanıcının database'e erişim izni olduğundan emin olun
