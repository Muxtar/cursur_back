package database

import (
	"context"
	"log"
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

	// Initialize MongoDB
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.MongoDBURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	
	// Ping MongoDB to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}
	log.Println("MongoDB connected successfully")
	
	db.MongoDB = mongoClient.Database(cfg.MongoDBName)

	return db
}

func (d *Database) Close() {
	if d.MongoDB != nil {
		d.MongoDB.Client().Disconnect(context.Background())
	}
}
