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
	Type        string   `json:"type" binding:"required"` // direct, group
	MemberIDs   []string `json:"member_ids"`
	GroupName   string   `json:"group_name,omitempty"`
	IsAnonymous bool     `json:"is_anonymous"` // current user's identity hidden from others
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

	// Enrich: for direct chats, set other_party_anonymous when the other member is anonymous to current user
	out := make([]map[string]interface{}, 0, len(chats))
	for _, ch := range chats {
		m := map[string]interface{}{
			"id":         ch.ID,
			"type":       ch.Type,
			"members":    ch.Members,
			"group_name": ch.GroupName,
			"created_at": ch.CreatedAt,
			"updated_at": ch.UpdatedAt,
		}
		if ch.Type == "direct" && ch.AnonymousFromUserID != nil && len(ch.Members) == 2 {
			// The other party is anonymous to me if AnonymousFromUserID is the other user (not me)
			otherIsAnonymous := *ch.AnonymousFromUserID != userIDObj
			m["other_party_anonymous"] = otherIsAnonymous
		}
		out = append(out, m)
	}
	c.JSON(http.StatusOK, out)
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

	var anonymousFrom *primitive.ObjectID
	if req.IsAnonymous && req.Type == "direct" {
		anonymousFrom = &userIDObj
	}

	chat := models.Chat{
		ID:                  primitive.NewObjectID(),
		Type:                req.Type,
		Members:             members,
		GroupName:           req.GroupName,
		AnonymousFromUserID: anonymousFrom,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	_, err := h.db.MongoDB.Collection("chats").InsertOne(context.Background(), chat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat"})
		return
	}

	c.JSON(http.StatusCreated, chat)
}

func (h *ChatHandler) GetChat(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

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

	out := map[string]interface{}{
		"id":         chat.ID,
		"type":       chat.Type,
		"members":    chat.Members,
		"group_name": chat.GroupName,
		"created_at": chat.CreatedAt,
		"updated_at": chat.UpdatedAt,
	}
	if chat.Type == "direct" && chat.AnonymousFromUserID != nil && len(chat.Members) == 2 {
		out["other_party_anonymous"] = *chat.AnonymousFromUserID != userIDObj
	}
	c.JSON(http.StatusOK, out)
}

func (h *ChatHandler) GetMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	chatIDStr := c.Param("chat_id")
	chatID, err := primitive.ObjectIDFromHex(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	var chat models.Chat
	if err := h.db.MongoDB.Collection("chats").FindOne(
		context.Background(),
		bson.M{"_id": chatID},
	).Decode(&chat); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found"})
		return
	}

	cursor, err := h.db.MongoDB.Collection("messages").Find(
		context.Background(),
		bson.M{
			"chat_id":     chatID,
			"is_deleted":  false,
			"deleted_for": bson.M{"$ne": userIDObj},
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

	// Mask anonymous sender for the recipient (do not expose sender_id when chat has AnonymousFromUserID)
	var out []map[string]interface{}
	for _, m := range messages {
		raw := map[string]interface{}{
			"id":           m.ID,
			"chat_id":      m.ChatID,
			"sender_id":    m.SenderID,
			"content":      m.Content,
			"message_type": m.MessageType,
			"is_anonymous": m.IsAnonymous,
			"status":       m.Status,
			"created_at":   m.CreatedAt,
			"updated_at":   m.UpdatedAt,
		}
		if chat.AnonymousFromUserID != nil && m.SenderID == *chat.AnonymousFromUserID && m.SenderID != userIDObj {
			raw["sender_id"] = nil
			raw["is_anonymous"] = true
		}
		out = append(out, raw)
	}
	c.JSON(http.StatusOK, out)
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

func (h *ChatHandler) DeleteChat(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	chatIDStr := c.Param("chat_id")
	chatID, err := primitive.ObjectIDFromHex(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	// Verify user is a member of the chat
	var chat models.Chat
	err = h.db.MongoDB.Collection("chats").FindOne(
		context.Background(),
		bson.M{"_id": chatID, "members": userIDObj},
	).Decode(&chat)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found or you are not a member"})
		return
	}

	// Remove user from members list (soft delete - chat will be hidden from user)
	// This works for both direct and group chats
	_, err = h.db.MongoDB.Collection("chats").UpdateOne(
		context.Background(),
		bson.M{"_id": chatID},
		bson.M{"$pull": bson.M{"members": userIDObj}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete chat"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chat deleted successfully"})
}