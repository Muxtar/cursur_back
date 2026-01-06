# Docker ile Back-end Kurulumu

Bu kılavuz, back-end'i tüm servislerle birlikte (PostgreSQL, MongoDB, Redis) Docker Compose ile çalıştırmak için hazırlanmıştır.

## Gereksinimler

- Docker Desktop (veya Docker Engine + Docker Compose)
- Git

## Hızlı Başlangıç

### 1. Projeyi Klonlayın (veya mevcut dizine gidin)

```bash
cd back-end
```

### 2. Docker Compose ile Tüm Servisleri Başlatın

```bash
docker-compose up -d
```

Bu komut şunları yapacak:
- PostgreSQL container'ını başlatacak
- MongoDB container'ını başlatacak
- Redis container'ını başlatacak
- Back-end container'ını başlatacak (tüm servisler hazır olduktan sonra)

### 3. Logları Kontrol Edin

```bash
docker-compose logs -f backend
```

Back-end'in başarıyla başladığını görmelisiniz:
```
Server starting on port 8080
PostgreSQL connected successfully
MongoDB connected successfully
Redis connected successfully
```

### 4. Servisleri Durdurma

```bash
docker-compose down
```

Verileri de silmek isterseniz:
```bash
docker-compose down -v
```

## Servisler

### PostgreSQL
- **Port**: 5432
- **Host**: localhost (dışarıdan) veya `postgres` (container içinden)
- **User**: postgres
- **Password**: postgres
- **Database**: chat_app

### MongoDB
- **Port**: 27017
- **Host**: localhost (dışarıdan) veya `mongodb` (container içinden)
- **Database**: chat_app

### Redis
- **Port**: 6379
- **Host**: localhost (dışarıdan) veya `redis` (container içinden)
- **Password**: (yok)

### Back-end API
- **Port**: 8080
- **URL**: http://localhost:8080
- **Health Check**: http://localhost:8080/api/v1/health

## Komutlar

### Tüm servisleri başlat
```bash
docker-compose up -d
```

### Tüm servisleri durdur
```bash
docker-compose down
```

### Logları görüntüle
```bash
# Tüm servislerin logları
docker-compose logs -f

# Sadece back-end logları
docker-compose logs -f backend

# Sadece PostgreSQL logları
docker-compose logs -f postgres
```

### Servisleri yeniden başlat
```bash
docker-compose restart
```

### Sadece back-end'i yeniden build et
```bash
docker-compose build backend
docker-compose up -d backend
```

### Container'lara bağlanma
```bash
# Back-end container'ına bağlan
docker-compose exec backend sh

# PostgreSQL'e bağlan
docker-compose exec postgres psql -U postgres -d chat_app

# MongoDB'ye bağlan
docker-compose exec mongodb mongosh chat_app

# Redis'e bağlan
docker-compose exec redis redis-cli
```

## Veri Kalıcılığı

Tüm veriler Docker volume'lerinde saklanır:
- `postgres_data`: PostgreSQL verileri
- `mongodb_data`: MongoDB verileri
- `redis_data`: Redis verileri
- `uploads_data`: Yüklenen dosyalar

Volume'leri silmek için:
```bash
docker-compose down -v
```

## Environment Variables

Environment variable'ları değiştirmek için `docker-compose.yml` dosyasındaki `backend` servisinin `environment` bölümünü düzenleyin.

Örnek: JWT Secret değiştirmek için:
```yaml
environment:
  JWT_SECRET: your-new-secret-key
```

Değişiklikleri uygulamak için:
```bash
docker-compose up -d --force-recreate backend
```

## Sorun Giderme

### Port zaten kullanılıyor hatası

Eğer 5432, 27017, 6379 veya 8080 portları zaten kullanılıyorsa, `docker-compose.yml` dosyasındaki port mapping'leri değiştirin:

```yaml
ports:
  - "5433:5432"  # PostgreSQL için farklı port
```

### Container'lar başlamıyor

1. Logları kontrol edin:
```bash
docker-compose logs
```

2. Container'ların durumunu kontrol edin:
```bash
docker-compose ps
```

3. Tüm container'ları durdurup yeniden başlatın:
```bash
docker-compose down
docker-compose up -d
```

### Back-end PostgreSQL'e bağlanamıyor

1. PostgreSQL container'ının çalıştığını kontrol edin:
```bash
docker-compose ps postgres
```

2. PostgreSQL loglarını kontrol edin:
```bash
docker-compose logs postgres
```

3. Health check'i kontrol edin:
```bash
docker-compose exec postgres pg_isready -U postgres
```

### Veriler kayboldu

Volume'ler silinmiş olabilir. `docker-compose down -v` komutunu çalıştırmadığınızdan emin olun.

## Railway'de Kullanım

Railway'de Docker Compose kullanmak için:

1. Railway'de yeni bir proje oluşturun
2. GitHub repository'nizi bağlayın
3. Root directory'yi `back-end` olarak ayarlayın
4. Railway otomatik olarak `docker-compose.yml` dosyasını algılayacak

**NOT**: Railway'de production için managed database servisleri kullanmanız önerilir, ancak Docker Compose ile de çalışabilir.

## Production İçin Öneriler

1. **JWT Secret**: Production'da güçlü bir secret kullanın
2. **Database Passwords**: Production'da güçlü şifreler kullanın
3. **CORS**: `CORS_ALLOWED_ORIGINS` değerini front-end domain'inize göre ayarlayın
4. **SSL**: Production'da SSL sertifikaları kullanın
5. **Backup**: Düzenli olarak database backup'ları alın

## Özet

- ✅ Tek komutla tüm servisleri başlatın: `docker-compose up -d`
- ✅ Tüm veriler Docker volume'lerinde saklanır
- ✅ Local development için hazır
- ✅ Railway'de de kullanılabilir
- ✅ Kolay yönetim ve sorun giderme

