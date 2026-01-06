package database

import (
	"context"
	"fmt"
	"log"

	"chat-backend/internal/config"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Database struct {
	MongoDB  *mongo.Database
	Redis    *redis.Client
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

	// Initialize Redis
	db.Redis = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       0,
	})
	if err := db.Redis.Ping(context.Background()).Err(); err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}

	// Initialize PostgreSQL
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.PostgresHost, cfg.PostgresUser, cfg.PostgresPass, cfg.PostgresDB, cfg.PostgresPort)
	postgresDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
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
	if d.Redis != nil {
		d.Redis.Close()
	}
}

func (d *Database) migrate() {
	// Auto-migrate will be handled by GORM
	// Add your models here for auto-migration
}
