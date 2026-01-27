package database

import (
	"context"
	"log"
	"strings"
	"time"

	"chat-backend/internal/config"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Database struct {
	MongoDB *mongo.Database
}

func Initialize(cfg *config.Config) *Database {
	db := &Database{}

	// Initialize MongoDB with exponential backoff retry logic
	maxRetries := 5
	var mongoClient *mongo.Client
	var lastErr error
	
	// Build connection options with timeouts (10s context timeout)
	clientOptions := options.Client().ApplyURI(cfg.MongoDBURI).
		SetServerSelectionTimeout(10 * time.Second).
		SetSocketTimeout(10 * time.Second).
		SetConnectTimeout(10 * time.Second).
		SetMaxPoolSize(100).
		SetMinPoolSize(10)
	
	// Retry loop with exponential backoff
	for i := 0; i < maxRetries; i++ {
		// Create context with 10s timeout for each connection attempt
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		
		mongoClient, lastErr = mongo.Connect(ctx, clientOptions)
		cancel()
		
		if lastErr != nil {
			// Check if it's a DNS SRV error
			errStr := lastErr.Error()
			isSRVError := strings.Contains(errStr, "lookup _mongodb._tcp") || 
			             strings.Contains(errStr, "server misbehaving") ||
			             strings.Contains(cfg.MongoDBURI, "mongodb+srv://")
			
			if i < maxRetries-1 {
				// Exponential backoff: 1s, 2s, 4s, 8s
				waitTime := time.Duration(1<<uint(i)) * time.Second
				log.Printf("MongoDB connection attempt %d/%d failed, retrying in %v...", i+1, maxRetries, waitTime)
				log.Printf("Error: %v", lastErr)
				
				if isSRVError {
					log.Println("⚠️  This appears to be a DNS SRV resolution issue (common with mongodb+srv:// on Railway)")
				}
				
				time.Sleep(waitTime)
				continue
			} else {
				// Last attempt failed - prepare detailed error message
				lastErr = lastErr
			}
		} else {
			// Connection successful, verify with ping
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			pingErr := mongoClient.Ping(ctx, nil)
			cancel()
			
			if pingErr != nil {
				lastErr = pingErr
				if i < maxRetries-1 {
					waitTime := time.Duration(1<<uint(i)) * time.Second
					log.Printf("MongoDB ping attempt %d/%d failed, retrying in %v...", i+1, maxRetries, waitTime)
					log.Printf("Error: %v", pingErr)
					mongoClient.Disconnect(context.Background())
					time.Sleep(waitTime)
					continue
				}
			} else {
				log.Println("✅ MongoDB connected successfully")
				lastErr = nil
				break
			}
		}
	}
	
	// Handle connection failure with detailed error message
	if lastErr != nil {
		log.Println("")
		log.Println("❌ ERROR: Failed to connect to MongoDB after", maxRetries, "attempts")
		log.Printf("   Error: %v", lastErr)
		log.Println("")
		
		// Check if using mongodb+srv:// which often fails on Railway
		errStr := lastErr.Error()
		isSRVError := strings.Contains(errStr, "lookup _mongodb._tcp") || 
		             strings.Contains(errStr, "server misbehaving") ||
		             strings.Contains(cfg.MongoDBURI, "mongodb+srv://")
		
		if isSRVError {
			log.Println("🔍 DIAGNOSIS: DNS SRV Resolution Issue")
			log.Println("   This is a known Railway issue with mongodb+srv:// protocol")
			log.Println("")
			log.Println("✅ SOLUTION: Use Standard Connection String (mongodb://)")
			log.Println("")
			log.Println("   Steps to fix:")
			log.Println("   1. Go to MongoDB Atlas Dashboard → Your Cluster → Connect")
			log.Println("   2. Select 'Connect your application'")
			log.Println("   3. Choose 'Standard connection string' (NOT SRV)")
			log.Println("   4. Copy the connection string")
			log.Println("   5. Update MONGODB_URI in Railway Variables")
			log.Println("")
			log.Println("   Expected format:")
			log.Println("   mongodb://user:pass@cluster0-shard-00-00.xxxxx.mongodb.net:27017,")
			log.Println("                cluster0-shard-00-01.xxxxx.mongodb.net:27017,")
			log.Println("                cluster0-shard-00-02.xxxxx.mongodb.net:27017/chat_app")
			log.Println("   ?ssl=true&replicaSet=atlas-xxxxx-shard-0&authSource=admin")
			log.Println("   &retryWrites=true&w=majority")
			log.Println("")
			log.Println("   ⚠️  Important: Replace 'xxxxx' with your actual cluster name")
			log.Println("   ⚠️  Important: Include database name (/chat_app) before the '?'")
			log.Println("")
		} else {
			log.Println("🔍 Troubleshooting tips:")
			log.Println("   1. Verify MONGODB_URI environment variable is set correctly")
			log.Println("   2. Check MongoDB Atlas Network Access allows Railway IPs (0.0.0.0/0 for testing)")
			log.Println("   3. Verify MongoDB Atlas cluster is running")
			log.Println("   4. Check database name is correct (MONGODB_DB or in connection string)")
			log.Println("   5. If password has special characters, ensure it's URL-encoded")
			log.Println("")
		}
		
		log.Fatal("Cannot start application without MongoDB connection")
	}
	
	db.MongoDB = mongoClient.Database(cfg.MongoDBName)
	log.Printf("📦 Using database: %s", cfg.MongoDBName)

	return db
}

func (d *Database) Close() {
	if d.MongoDB != nil {
		d.MongoDB.Client().Disconnect(context.Background())
	}
}
