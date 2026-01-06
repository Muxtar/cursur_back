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
	// Railway uses different variable names, so we check both
	// For PostgreSQL: Railway uses PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE
	// But we also support POSTGRES_HOST, POSTGRES_PORT, etc.
	postgresHost := getEnv("POSTGRES_HOST", "")
	if postgresHost == "" {
		postgresHost = getEnv("PGHOST", "localhost")
	}
	
	postgresPort := getEnv("POSTGRES_PORT", "")
	if postgresPort == "" {
		postgresPort = getEnv("PGPORT", "5432")
	}
	
	postgresUser := getEnv("POSTGRES_USER", "")
	if postgresUser == "" {
		postgresUser = getEnv("PGUSER", "postgres")
	}
	
	postgresPass := getEnv("POSTGRES_PASSWORD", "")
	if postgresPass == "" {
		postgresPass = getEnv("PGPASSWORD", "postgres")
	}
	
	postgresDB := getEnv("POSTGRES_DB", "")
	if postgresDB == "" {
		postgresDB = getEnv("PGDATABASE", "chat_app")
	}
	
	// For MongoDB: Railway uses MONGO_URL or MONGODB_URI
	mongoURI := getEnv("MONGODB_URI", "")
	if mongoURI == "" {
		mongoURI = getEnv("MONGO_URL", "mongodb://localhost:27017")
	}
	
	return &Config{
		Port:          getEnv("PORT", "8080"),
		MongoDBURI:    mongoURI,
		MongoDBName:   getEnv("MONGODB_DB", getEnv("MONGO_DATABASE", "chat_app")),
		RedisHost:     getEnv("REDIS_HOST", getEnv("REDIS_URL", "localhost")),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		PostgresHost:  postgresHost,
		PostgresPort:  postgresPort,
		PostgresUser:  postgresUser,
		PostgresPass:  postgresPass,
		PostgresDB:    postgresDB,
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





