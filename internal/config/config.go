package config

import (
	"os"
)

type Config struct {
	Port         string
	MongoDBURI   string
	MongoDBName  string
	RedisHost    string
	RedisPort    string
	RedisPassword string
	PostgresHost string
	PostgresPort string
	PostgresUser string
	PostgresPass string
	PostgresDB   string
	JWTSecret    string
	JWTExpiration string
	UploadDir    string
	MaxFileSize  int64
}

func Load() *Config {
	return &Config{
		Port:          getEnv("PORT", "8080"),
		MongoDBURI:    getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		MongoDBName:   getEnv("MONGODB_DB", "chat_app"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		PostgresHost:  getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:  getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:  getEnv("POSTGRES_USER", "postgres"),
		PostgresPass:  getEnv("POSTGRES_PASSWORD", "postgres"),
		PostgresDB:    getEnv("POSTGRES_DB", "chat_app"),
		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key"),
		JWTExpiration: getEnv("JWT_EXPIRATION", "24h"),
		UploadDir:     getEnv("UPLOAD_DIR", "./uploads"),
		MaxFileSize:   10485760, // 10MB
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}





