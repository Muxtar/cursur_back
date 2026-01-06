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

type CallHandler struct {
	db  *database.Database
	hub *websocket.Hub
}

func NewCallHandler(db *database.Database, hub *websocket.Hub) *CallHandler {
	return &CallHandler{db: db, hub: hub}
}

type InitiateCallRequest struct {
	Type    string   `json:"type" binding:"required"` // video, voice, group
	ChatID  string   `json:"chat_id" binding:"required"`
	Members []string `json:"members,omitempty"`
}

func (h *CallHandler) InitiateCall(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var req InitiateCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chatID, err := primitive.ObjectIDFromHex(req.ChatID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	members := []primitive.ObjectID{userIDObj}
	for _, memberIDStr := range req.Members {
		memberID, err := primitive.ObjectIDFromHex(memberIDStr)
		if err != nil {
			continue
		}
		members = append(members, memberID)
	}

	call := models.Call{
		ID:        primitive.NewObjectID(),
		Type:      req.Type,
		CallerID:  userIDObj,
		ChatID:    chatID,
		Members:   members,
		Status:    "ringing",
		StartedAt: time.Now(),
	}

	_, err = h.db.MongoDB.Collection("calls").InsertOne(context.Background(), call)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate call"})
		return
	}

	// Broadcast call via WebSocket
	// This would notify all members in the chat

	c.JSON(http.StatusCreated, call)
}

func (h *CallHandler) AnswerCall(c *gin.Context) {
	callIDStr := c.Param("call_id")
	callID, err := primitive.ObjectIDFromHex(callIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid call ID"})
		return
	}

	_, err = h.db.MongoDB.Collection("calls").UpdateOne(
		context.Background(),
		bson.M{"_id": callID},
		bson.M{"$set": bson.M{"status": "active"}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to answer call"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Call answered"})
}

func (h *CallHandler) EndCall(c *gin.Context) {
	callIDStr := c.Param("call_id")
	callID, err := primitive.ObjectIDFromHex(callIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid call ID"})
		return
	}

	now := time.Now()
	_, err = h.db.MongoDB.Collection("calls").UpdateOne(
		context.Background(),
		bson.M{"_id": callID},
		bson.M{"$set": bson.M{
			"status":   "ended",
			"ended_at": now,
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to end call"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Call ended"})
}





