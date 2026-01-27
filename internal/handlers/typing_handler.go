package handlers

import (
	"context"
	"net/http"
	"time"

	"chat-backend/internal/database"
	"chat-backend/internal/models"
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

	// Store typing indicator in PostgreSQL (expires in 5 seconds)
	if h.db.Postgres != nil {
		expiresAt := time.Now().Add(5 * time.Second)
		
		// Delete existing indicator for this chat/user combination
		h.db.Postgres.Where("chat_id = ? AND user_id = ?", chatID.Hex(), userIDObj.Hex()).
			Delete(&models.TypingIndicator{})
		
		// Create new indicator
		indicator := models.TypingIndicator{
			ChatID:    chatID.Hex(),
			UserID:    userIDObj.Hex(),
			Type:      req.Type,
			ExpiresAt: expiresAt,
		}
		h.db.Postgres.Create(&indicator)
	}

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

	// Get all typing indicators for this chat that haven't expired
	if h.db.Postgres == nil {
		c.JSON(http.StatusOK, gin.H{"typing": []interface{}{}})
		return
	}
	
	var indicators []models.TypingIndicator
	err := h.db.Postgres.Where("chat_id = ? AND expires_at > ?", chatID.Hex(), time.Now()).
		Find(&indicators).Error
	
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"typing": []interface{}{}})
		return
	}

	var typingUsers []map[string]interface{}
	for _, indicator := range indicators {
		userID, err := primitive.ObjectIDFromHex(indicator.UserID)
		if err != nil {
			continue
		}

		typingUsers = append(typingUsers, map[string]interface{}{
			"user_id": userID,
			"type":    indicator.Type,
		})
	}

	c.JSON(http.StatusOK, gin.H{"typing": typingUsers})
}





