package handlers

import (
	"context"
	"net/http"
	"time"

	"chat-backend/internal/database"
	"chat-backend/internal/models"
	"chat-backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatHandler struct {
	db  *database.Database
	hub *websocket.Hub
}

func NewChatHandler(db *database.Database, hub *websocket.Hub) *ChatHandler {
	return &ChatHandler{db: db, hub: hub}
}

type CreateChatRequest struct {
	Type      string   `json:"type" binding:"required"` // direct, group
	MemberIDs []string `json:"member_ids"`
	GroupName string   `json:"group_name,omitempty"`
}

// SendMessage is now handled by MessageHandler

func (h *ChatHandler) GetChats(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	cursor, err := h.db.MongoDB.Collection("chats").Find(
		context.Background(),
		bson.M{"members": userIDObj},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chats"})
		return
	}
	defer cursor.Close(context.Background())

	var chats []models.Chat
	if err := cursor.All(context.Background(), &chats); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode chats"})
		return
	}

	c.JSON(http.StatusOK, chats)
}

func (h *ChatHandler) CreateChat(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var req CreateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	members := []primitive.ObjectID{userIDObj}
	for _, memberIDStr := range req.MemberIDs {
		memberID, err := primitive.ObjectIDFromHex(memberIDStr)
		if err != nil {
			continue
		}
		members = append(members, memberID)
	}

	chat := models.Chat{
		ID:        primitive.NewObjectID(),
		Type:      req.Type,
		Members:   members,
		GroupName: req.GroupName,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := h.db.MongoDB.Collection("chats").InsertOne(context.Background(), chat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat"})
		return
	}

	c.JSON(http.StatusCreated, chat)
}

func (h *ChatHandler) GetChat(c *gin.Context) {
	chatIDStr := c.Param("chat_id")
	chatID, err := primitive.ObjectIDFromHex(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	var chat models.Chat
	err = h.db.MongoDB.Collection("chats").FindOne(
		context.Background(),
		bson.M{"_id": chatID},
	).Decode(&chat)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found"})
		return
	}

	c.JSON(http.StatusOK, chat)
}

func (h *ChatHandler) GetMessages(c *gin.Context) {
	chatIDStr := c.Param("chat_id")
	chatID, err := primitive.ObjectIDFromHex(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	cursor, err := h.db.MongoDB.Collection("messages").Find(
		context.Background(),
		bson.M{
			"chat_id":   chatID,
			"is_deleted": false,
		},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}
	defer cursor.Close(context.Background())

	var messages []models.Message
	if err := cursor.All(context.Background(), &messages); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// SendMessage is now handled by MessageHandler.SendMessage
// This endpoint redirects to /messages endpoint
func (h *ChatHandler) SendMessage(c *gin.Context) {
	// Redirect to message handler
	messageHandler := NewMessageHandler(h.db, h.hub)
	messageHandler.SendMessage(c)
}

func (h *ChatHandler) DeleteMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	messageIDStr := c.Param("message_id")
	messageID, err := primitive.ObjectIDFromHex(messageIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	// Verify message belongs to user
	var message models.Message
	err = h.db.MongoDB.Collection("messages").FindOne(
		context.Background(),
		bson.M{"_id": messageID, "sender_id": userIDObj},
	).Decode(&message)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}

	// Soft delete
	_, err = h.db.MongoDB.Collection("messages").UpdateOne(
		context.Background(),
		bson.M{"_id": messageID},
		bson.M{"$set": bson.M{
			"is_deleted": true,
			"updated_at": time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message deleted successfully"})
}

