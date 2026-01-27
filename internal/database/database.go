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

	// Initialize MongoDB with retry logic and better connection options
	maxRetries := 5
	var mongoClient *mongo.Client
	var lastErr error
	
	// Build connection options with timeouts
	clientOptions := options.Client().ApplyURI(cfg.MongoDBURI).
		SetServerSelectionTimeout(10 * time.Second).
		SetSocketTimeout(10 * time.Second).
		SetConnectTimeout(10 * time.Second).
		SetMaxPoolSize(100).
		SetMinPoolSize(10)
	
	for i := 0; i < maxRetries; i++ {
		mongoClient, lastErr = mongo.Connect(context.Background(), clientOptions)
		if lastErr != nil {
			if i < maxRetries-1 {
				waitTime := time.Duration(i+1) * time.Second
				log.Printf("MongoDB connection attempt %d/%d failed, retrying in %v...", i+1, maxRetries, waitTime)
				log.Printf("Error: %v", lastErr)
				time.Sleep(waitTime)
				continue
			}
		} else {
			// Ping MongoDB to verify connection
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			pingErr := mongoClient.Ping(ctx, nil)
			cancel()
			
			if pingErr != nil {
				lastErr = pingErr
				if i < maxRetries-1 {
					waitTime := time.Duration(i+1) * time.Second
					log.Printf("MongoDB ping attempt %d/%d failed, retrying in %v...", i+1, maxRetries, waitTime)
					log.Printf("Error: %v", pingErr)
					mongoClient.Disconnect(context.Background())
					time.Sleep(waitTime)
					continue
				}
			} else {
				log.Println("MongoDB connected successfully")
				lastErr = nil
				break
			}
		}
	}
	
	if lastErr != nil {
		log.Printf("ERROR: Failed to connect to MongoDB after %d attempts: %v", maxRetries, lastErr)
		log.Println("")
		
		// Check if using mongodb+srv:// which often fails on Railway
		if strings.Contains(cfg.MongoDBURI, "mongodb+srv://") {
			log.Println("⚠️  DETECTED: You are using 'mongodb+srv://' connection string")
			log.Println("⚠️  Railway often has DNS issues with mongodb+srv:// protocol")
			log.Println("")
			log.Println("✅ SOLUTION: Use standard 'mongodb://' connection string instead")
			log.Println("")
			log.Println("How to get standard connection string:")
			log.Println("1. Go to MongoDB Atlas Dashboard")
			log.Println("2. Click 'Connect' on your cluster")
			log.Println("3. Select 'Connect your application'")
			log.Println("4. Choose 'Standard connection string' (NOT SRV)")
			log.Println("5. Copy the connection string")
			log.Println("6. Format should be: mongodb://username:password@host1:port1,host2:port2/database?options")
			log.Println("7. Update MONGODB_URI in Railway with this standard connection string")
			log.Println("")
			log.Println("Example standard format:")
			log.Println("mongodb://user:pass@cluster0-shard-00-00.xxxxx.mongodb.net:27017,cluster0-shard-00-01.xxxxx.mongodb.net:27017/chat_app?ssl=true&replicaSet=atlas-xxxxx-shard-0&authSource=admin&retryWrites=true&w=majority")
			log.Println("")
		} else {
			log.Println("Troubleshooting tips:")
			log.Println("1. Check if MongoDB URI is correct (MONGODB_URI environment variable)")
			log.Println("2. For Railway: Ensure MongoDB Atlas Network Access allows Railway IPs")
			log.Println("3. Check MongoDB Atlas cluster status")
			log.Println("4. Verify database name is correct")
		}
		log.Fatal("Cannot start application without MongoDB connection")
	}
	
	db.MongoDB = mongoClient.Database(cfg.MongoDBName)

	return db
}

func (d *Database) Close() {
	if d.MongoDB != nil {
		d.MongoDB.Client().Disconnect(context.Background())
	}
}
