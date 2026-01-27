package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"chat-backend/internal/config"
	"chat-backend/internal/models"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Database struct {
	MongoDB  *mongo.Database
	Postgres *gorm.DB
}

func Initialize(cfg *config.Config) *Database {
	db := &Database{}

	// Initialize MongoDB
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.MongoDBURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	db.MongoDB = mongoClient.Database(cfg.MongoDBName)

	// Initialize PostgreSQL (with retries for Railway deployment)
	maxRetries := 5
	var postgresDB *gorm.DB
	var lastErr error
	
	// Log current PostgreSQL configuration (without password)
	log.Printf("PostgreSQL Configuration:")
	log.Printf("  Host: %s", cfg.PostgresHost)
	log.Printf("  Port: %s", cfg.PostgresPort)
	log.Printf("  User: %s", cfg.PostgresUser)
	log.Printf("  Database: %s", cfg.PostgresDB)
	passwordStatus := "(empty)"
	if cfg.PostgresPass != "" {
		passwordStatus = "*** (set)"
	}
	log.Printf("  Password: %s", passwordStatus)
	
	// Check if we're using default/localhost values (which won't work in Railway)
	if cfg.PostgresHost == "localhost" || cfg.PostgresHost == "127.0.0.1" {
		log.Println("")
		log.Println("WARNING: PostgreSQL host is set to 'localhost' - this won't work in Railway!")
		log.Println("Railway environment variables may not be set correctly.")
		log.Println("")
		log.Println("Please check your Railway backend service Variables tab:")
		log.Println("  - Look for: PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE")
		log.Println("  - Or: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		log.Println("")
		log.Println("If these are missing:")
		log.Println("1. Go to your PostgreSQL service in Railway")
		log.Println("2. Click on 'Variables' tab")
		log.Println("3. Copy the values (PGHOST, PGPORT, etc.)")
		log.Println("4. Go to your backend service")
		log.Println("5. Add these as environment variables")
		log.Println("")
	}
	
	for i := 0; i < maxRetries; i++ {
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			cfg.PostgresHost, cfg.PostgresUser, cfg.PostgresPass, cfg.PostgresDB, cfg.PostgresPort)
		
		postgresDB, lastErr = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if lastErr != nil {
			if i < maxRetries-1 {
				waitTime := time.Duration(i+1) * time.Second
				log.Printf("PostgreSQL connection attempt %d/%d failed, retrying in %v...", i+1, maxRetries, waitTime)
				time.Sleep(waitTime)
				continue
			}
		} else {
			log.Println("PostgreSQL connected successfully")
			lastErr = nil
			break
		}
	}
	
	if lastErr != nil {
		log.Printf("ERROR: Failed to connect to PostgreSQL after %d attempts: %v", maxRetries, lastErr)
		log.Println("ERROR: PostgreSQL is required for this application to function properly.")
		log.Println("")
		log.Println("Current PostgreSQL configuration:")
		log.Printf("  Host: %s", cfg.PostgresHost)
		log.Printf("  Port: %s", cfg.PostgresPort)
		log.Printf("  User: %s", cfg.PostgresUser)
		log.Printf("  Database: %s", cfg.PostgresDB)
		log.Println("")
		log.Println("To fix this issue:")
		log.Println("1. Add PostgreSQL service in Railway: 'New' > 'Database' > 'Add PostgreSQL'")
		log.Println("2. Connect PostgreSQL service to your backend service (click 'Connect' button)")
		log.Println("3. Railway will automatically set PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE")
		log.Println("4. If not automatic, manually copy these from PostgreSQL service Variables tab")
		log.Println("5. Add them to your backend service Variables tab")
		log.Println("")
		log.Fatal("Cannot start application without PostgreSQL connection")
	}
	
	db.Postgres = postgresDB

	// Auto-migrate tables
	db.migrate()

	return db
}

func (d *Database) Close() {
	if d.MongoDB != nil {
		d.MongoDB.Client().Disconnect(context.Background())
	}
}

func (d *Database) migrate() {
	// Auto-migrate PostgreSQL models
	if d.Postgres != nil {
		log.Println("Starting PostgreSQL migration...")
		err := d.Postgres.AutoMigrate(
			&models.VerificationCode{},
			&models.QRCodeCache{},
			&models.TypingIndicator{},
			&models.CacheEntry{},
		)
		if err != nil {
			log.Printf("ERROR: Failed to auto-migrate PostgreSQL models: %v", err)
			log.Println("This may cause issues with verification codes and other features.")
		} else {
			log.Println("PostgreSQL models migrated successfully:")
			log.Println("  - verification_codes")
			log.Println("  - qr_code_cache")
			log.Println("  - typing_indicators")
			log.Println("  - cache_entries")
		}
	} else {
		log.Println("WARNING: PostgreSQL is not available, skipping migration")
	}
}
