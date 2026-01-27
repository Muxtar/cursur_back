package config

import (
	"log"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Port         string
	MongoDBURI   string
	MongoDBName  string
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
	// For MongoDB: Railway uses MONGO_URL or MONGODB_URI
	mongoURI := getEnv("MONGODB_URI", "")
	if mongoURI == "" {
		mongoURI = getEnv("MONGO_URL", "")
		if mongoURI == "" {
			mongoURI = "mongodb://localhost:27017"
			log.Println("WARNING: MongoDB URI not set, using default localhost:27017")
			log.Println("WARNING: Set MONGODB_URI or MONGO_URL environment variable in Railway")
		}
	}
	
	// ⚠️ CRITICAL: Detect mongodb+srv:// and warn about Railway DNS issues
	if strings.HasPrefix(mongoURI, "mongodb+srv://") {
		log.Println("")
		log.Println("⚠️  WARNING: Detected 'mongodb+srv://' connection string")
		log.Println("⚠️  Railway has known DNS issues with SRV DNS resolution")
		log.Println("⚠️  If you encounter 'lookup _mongodb._tcp... server misbehaving' errors:")
		log.Println("    → Use 'mongodb://' standard connection string instead")
		log.Println("    → Get it from MongoDB Atlas: Connect → Connect your application → Standard connection string")
		log.Println("")
		log.Println("   Expected format:")
		log.Println("   mongodb://user:pass@cluster0-shard-00-00.xxxxx.mongodb.net:27017,cluster0-shard-00-01.xxxxx.mongodb.net:27017,cluster0-shard-00-02.xxxxx.mongodb.net:27017/chat_app?ssl=true&replicaSet=atlas-xxxxx-shard-0&authSource=admin&retryWrites=true&w=majority")
		log.Println("")
	}
	
	// Log MongoDB URI (mask password for security) - only for logging, don't modify original
	mongoURILog := mongoURI
	if strings.Contains(mongoURILog, "@") {
		// Mask password in connection string for logging
		if u, err := url.Parse(mongoURILog); err == nil && u.User != nil {
			if _, hasPass := u.User.Password(); hasPass {
				maskedUser := url.UserPassword(u.User.Username(), "***")
				u.User = maskedUser
				// Rebuild URL without encoding issues
				mongoURILog = u.Scheme + "://" + maskedUser.String() + "@" + u.Host + u.Path
				if u.RawQuery != "" {
					mongoURILog += "?" + u.RawQuery
				}
			}
		}
	}
	log.Printf("MongoDB URI: %s", mongoURILog)
	
	// Twilio configuration
	twilioEnabled := getEnv("TWILIO_ENABLED", "false")
	twilioEnabledBool := twilioEnabled == "true" || twilioEnabled == "1"
	
	mongoDBName := getEnv("MONGODB_DB", getEnv("MONGO_DATABASE", "chat_app"))
	log.Printf("MongoDB Database: %s", mongoDBName)
	
	return &Config{
		Port:          getEnv("PORT", "8080"),
		MongoDBURI:    mongoURI,
		MongoDBName:   mongoDBName,
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





