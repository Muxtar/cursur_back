package handlers

import (
	"context"
	"net/http"
	"time"

	"chat-backend/internal/database"
	"chat-backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TypingHandler struct {
	db  *database.Database
	hub *websocket.Hub
}

func NewTypingHandler(db *database.Database, hub *websocket.Hub) *TypingHandler {
	return &TypingHandler{db: db, hub: hub}
}

func (h *TypingHandler) SetTyping(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	chatIDStr := c.Param("chat_id")
	chatID, err := primitive.ObjectIDFromHex(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	var req struct {
		Type string `json:"type"` // typing, recording_voice, recording_video
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Store typing indicator in MongoDB (expires in 5 seconds)
	expiresAt := time.Now().Add(5 * time.Second)
	doc := bson.M{
		"chat_id":   chatID,
		"user_id":   userIDObj,
		"type":      req.Type,
		"expires_at": expiresAt,
		"created_at": time.Now(),
	}
	
	// Use upsert to update if exists
	filter := bson.M{
		"chat_id": chatID,
		"user_id": userIDObj,
	}
	update := bson.M{"$set": doc}
	_, _ = h.db.MongoDB.Collection("typing_indicators").UpdateOne(context.Background(), filter, update, options.Update().SetUpsert(true))

	// Broadcast via WebSocket
	// This would be handled by WebSocket hub

	c.JSON(http.StatusOK, gin.H{"message": "Typing indicator set"})
}

func (h *TypingHandler) GetTyping(c *gin.Context) {
	chatIDStr := c.Param("chat_id")
	chatID, err := primitive.ObjectIDFromHex(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	// Get all typing indicators for this chat from MongoDB
	now := time.Now()
	filter := bson.M{
		"chat_id":   chatID,
		"expires_at": bson.M{"$gt": now}, // Only get non-expired indicators
	}
	
	cursor, err := h.db.MongoDB.Collection("typing_indicators").Find(context.Background(), filter)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"typing": []interface{}{}})
		return
	}
	defer cursor.Close(context.Background())

	var typingUsers []map[string]interface{}
	for cursor.Next(context.Background()) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		userID, ok := doc["user_id"].(primitive.ObjectID)
		if !ok {
			continue
		}

		typ, _ := doc["type"].(string)
		typingUsers = append(typingUsers, map[string]interface{}{
			"user_id": userID,
			"type":    typ,
		})
	}

	c.JSON(http.StatusOK, gin.H{"typing": typingUsers})
}





