package handlers

import (
	"context"
	"net/http"
	"time"

	"chat-backend/internal/database"
	"chat-backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	// Store typing indicator in Redis (expires in 5 seconds)
	key := "typing:" + chatID.Hex() + ":" + userIDObj.Hex()
	h.db.Redis.Set(context.Background(), key, req.Type, 5*time.Second)

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

	// Get all typing indicators for this chat
	pattern := "typing:" + chatID.Hex() + ":*"
	keys, err := h.db.Redis.Keys(context.Background(), pattern).Result()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"typing": []interface{}{}})
		return
	}

	var typingUsers []map[string]interface{}
	for _, key := range keys {
		typ, err := h.db.Redis.Get(context.Background(), key).Result()
		if err != nil {
			continue
		}

		// Extract user ID from key
		// Format: typing:chatID:userID
		userIDStr := key[len("typing:"+chatID.Hex()+":"):]
		userID, err := primitive.ObjectIDFromHex(userIDStr)
		if err != nil {
			continue
		}

		typingUsers = append(typingUsers, map[string]interface{}{
			"user_id": userID,
			"type":    typ,
		})
	}

	c.JSON(http.StatusOK, gin.H{"typing": typingUsers})
}





