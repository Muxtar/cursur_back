# MongoDB Connection Fix for Railway

## Sorun
Railway'de MongoDB bağlantısı DNS lookup hatası veriyor:
```
error parsing uri: lookup _mongodb._tcp.cluster0.g2e8hv9.mongodb.net on [fd12::10]:53: server misbehaving
```

## Çözüm

### Seçenek 1: Standard MongoDB Connection String Kullanın (Önerilen)

MongoDB Atlas'tan `mongodb+srv://` yerine **standard connection string** kullanın:

1. MongoDB Atlas Dashboard'a gidin
2. **Connect** butonuna tıklayın
3. **Connect your application** seçeneğini seçin
4. **Driver:** Node.js (veya herhangi biri - connection string aynı)
5. **Version:** En son sürüm
6. Connection string'i kopyalayın

**Örnek `mongodb+srv://` formatı:**
```
mongodb+srv://muxtarbayramov92:ZcbRm9j6ISIwTmIg@cluster0.g2e8hv9.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0
```

**Standard format'a çevirin:**
1. MongoDB Atlas Dashboard → **Database Access** → Kullanıcınızın **Connect** butonuna tıklayın
2. Veya cluster'ınızın **Connect** → **Connect your application** → **Standard connection string** seçeneğini kullanın
3. Format şöyle olmalı:
```
mongodb://muxtarbayramov92:ZcbRm9j6ISIwTmIg@cluster0-shard-00-00.g2e8hv9.mongodb.net:27017,cluster0-shard-00-01.g2e8hv9.mongodb.net:27017,cluster0-shard-00-02.g2e8hv9.mongodb.net:27017/chat_app?ssl=true&replicaSet=atlas-xxxxx-shard-0&authSource=admin&retryWrites=true&w=majority
```

**Railway'de ayarlayın:**
- Variable: `MONGODB_URI`
- Value: Yukarıdaki standard connection string (database adını `/chat_app` olarak ekleyin)

### Seçenek 2: MongoDB Atlas Network Access Ayarları

1. MongoDB Atlas Dashboard → **Network Access**
2. **Add IP Address** → **Allow Access from Anywhere** (development için)
   - Veya Railway'in IP adreslerini ekleyin
3. **Confirm**

### Seçenek 3: Connection String'e Database Adı Ekleyin

Eğer `mongodb+srv://` kullanmaya devam edecekseniz, connection string'e database adını ekleyin:

**Önceki:**
```
mongodb+srv://muxtarbayramov92:ZcbRm9j6ISIwTmIg@cluster0.g2e8hv9.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0
```

**Sonraki (database adı eklendi):**
```
mongodb+srv://muxtarbayramov92:ZcbRm9j6ISIwTmIg@cluster0.g2e8hv9.mongodb.net/chat_app?retryWrites=true&w=majority&appName=Cluster0
```

**Önemli:** `/chat_app` kısmını ekleyin (cluster0.g2e8hv9.mongodb.net ile `?` arasına)

## Railway Environment Variables

Railway'de şu environment variable'ları ayarlayın:

1. **MONGODB_URI** - MongoDB connection string (standard format önerilir)
2. **MONGODB_DB** (opsiyonel) - Database adı (default: `chat_app`)

## Test

Deploy sonrası logları kontrol edin:
```
MongoDB URI: mongodb://muxtarbayramov92:***@cluster0-shard-00-00.g2e8hv9.mongodb.net:27017,...
MongoDB Database: chat_app
MongoDB connected successfully
```

## Notlar

- `mongodb+srv://` protokolü SRV DNS kayıtlarını kullanır ve Railway'de bazen DNS sorunlarına yol açabilir
- Standard `mongodb://` formatı daha güvenilirdir ve Railway'de daha iyi çalışır
- Connection string'de şifre URL-encoded olmalı (özel karakterler için)
