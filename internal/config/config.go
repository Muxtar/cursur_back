package config

import (
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Port         string
	MongoDBURI   string
	MongoDBName  string
	PostgresHost string
	PostgresPort string
	PostgresUser string
	PostgresPass string
	PostgresDB   string
	JWTSecret    string
	JWTExpiration string
	UploadDir    string
	MaxFileSize  int64
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioPhoneNumber string
	TwilioEnabled     bool
}

func Load() *Config {
	// Railway uses different variable names, so we check both
	// Railway sometimes provides DATABASE_URL as a single connection string
	// Format: postgresql://user:password@host:port/database
	
	var postgresHost, postgresPort, postgresUser, postgresPass, postgresDB string
	
	// First, check if DATABASE_URL is provided (Railway sometimes uses this)
	databaseURL := getEnv("DATABASE_URL", "")
	if databaseURL != "" {
		// Parse DATABASE_URL
		parsed, err := url.Parse(databaseURL)
		if err == nil && parsed.Scheme == "postgres" || parsed.Scheme == "postgresql" {
			postgresHost = parsed.Hostname()
			postgresPort = parsed.Port()
			if postgresPort == "" {
				postgresPort = "5432"
			}
			postgresUser = parsed.User.Username()
			postgresPass, _ = parsed.User.Password()
			postgresDB = strings.TrimPrefix(parsed.Path, "/")
		}
	}
	
	// If DATABASE_URL didn't provide values, check individual variables
	if postgresHost == "" {
		postgresHost = getEnv("POSTGRES_HOST", "")
		if postgresHost == "" {
			postgresHost = getEnv("PGHOST", "localhost")
		}
	}
	
	if postgresPort == "" {
		postgresPort = getEnv("POSTGRES_PORT", "")
		if postgresPort == "" {
			postgresPort = getEnv("PGPORT", "5432")
		}
	}
	
	if postgresUser == "" {
		postgresUser = getEnv("POSTGRES_USER", "")
		if postgresUser == "" {
			postgresUser = getEnv("PGUSER", "postgres")
		}
	}
	
	if postgresPass == "" {
		postgresPass = getEnv("POSTGRES_PASSWORD", "")
		if postgresPass == "" {
			postgresPass = getEnv("PGPASSWORD", "postgres")
		}
	}
	
	if postgresDB == "" {
		postgresDB = getEnv("POSTGRES_DB", "")
		if postgresDB == "" {
			postgresDB = getEnv("PGDATABASE", "chat_app")
		}
	}
	
	// For MongoDB: Railway uses MONGO_URL or MONGODB_URI
	mongoURI := getEnv("MONGODB_URI", "")
	if mongoURI == "" {
		mongoURI = getEnv("MONGO_URL", "mongodb://localhost:27017")
	}
	
	// Twilio configuration
	twilioEnabled := getEnv("TWILIO_ENABLED", "false")
	twilioEnabledBool := twilioEnabled == "true" || twilioEnabled == "1"
	
	return &Config{
		Port:          getEnv("PORT", "8080"),
		MongoDBURI:    mongoURI,
		MongoDBName:   getEnv("MONGODB_DB", getEnv("MONGO_DATABASE", "chat_app")),
		PostgresHost:  postgresHost,
		PostgresPort:  postgresPort,
		PostgresUser:  postgresUser,
		PostgresPass:  postgresPass,
		PostgresDB:    postgresDB,
		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key"),
		JWTExpiration: getEnv("JWT_EXPIRATION", "24h"),
		UploadDir:     getEnv("UPLOAD_DIR", "./uploads"),
		MaxFileSize:   10485760, // 10MB
		TwilioAccountSID:  getEnv("TWILIO_ACCOUNT_SID", ""),
		TwilioAuthToken:   getEnv("TWILIO_AUTH_TOKEN", ""),
		TwilioPhoneNumber: getEnv("TWILIO_PHONE_NUMBER", ""),
		TwilioEnabled:      twilioEnabledBool,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}





