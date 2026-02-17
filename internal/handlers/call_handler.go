package handlers

import (
	"context"
	"encoding/json"
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

	// Load chat to verify membership and get members when not provided.
	var chat models.Chat
	if err := h.db.MongoDB.Collection("chats").FindOne(
		context.Background(),
		bson.M{"_id": chatID, "members": userIDObj},
	).Decode(&chat); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found or you are not a member"})
		return
	}

	// Determine call members:
	// - If client passes explicit members (e.g., group call selection), use them.
	// - Otherwise default to chat members (so direct calls include the other party).
	memberSet := make(map[primitive.ObjectID]bool)
	memberSet[userIDObj] = true
	if len(req.Members) > 0 {
		for _, memberIDStr := range req.Members {
			memberID, err := primitive.ObjectIDFromHex(memberIDStr)
			if err != nil {
				continue
			}
			memberSet[memberID] = true
		}
	} else {
		for _, m := range chat.Members {
			memberSet[m] = true
		}
	}

	members := make([]primitive.ObjectID, 0, len(memberSet))
	for id := range memberSet {
		members = append(members, id)
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

	// Broadcast call via WebSocket to all chat members
	callNotification := map[string]interface{}{
		"type":      "call",
		"call_id":   call.ID.Hex(),
		"chat_id":   chatID.Hex(),
		"call_type": call.Type,
		"caller_id": call.CallerID.Hex(),
		"status":    call.Status,
	}
	callJSON, _ := json.Marshal(callNotification)

	// 1) Broadcast to the room (when users are actively inside the chat)
	h.hub.BroadcastJSONToRoom(chatID, callJSON)
	// 2) Also send to users directly so they can receive even if they haven't joined the room yet
	for _, memberID := range members {
		h.hub.SendJSONToUser(memberID, callJSON)
	}

	c.JSON(http.StatusCreated, call)
}

func (h *CallHandler) AnswerCall(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	callIDStr := c.Param("call_id")
	callID, err := primitive.ObjectIDFromHex(callIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid call ID"})
		return
	}

	// Load call to get chat ID and caller ID
	var call models.Call
	err = h.db.MongoDB.Collection("calls").FindOne(
		context.Background(),
		bson.M{"_id": callID},
	).Decode(&call)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Call not found"})
		return
	}

	// Verify user is a member of the call
	isMember := false
	for _, memberID := range call.Members {
		if memberID == userIDObj {
			isMember = true
			break
		}
	}
	if !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this call"})
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

	// Notify caller that call was answered
	answerNotification := map[string]interface{}{
		"type":      "call_answered",
		"call_id":   callID.Hex(),
		"chat_id":   call.ChatID.Hex(),
		"call_type": call.Type,
		"status":    "active",
	}
	answerJSON, _ := json.Marshal(answerNotification)
	
	// Send to caller
	h.hub.SendJSONToUser(call.CallerID, answerJSON)
	// Also broadcast to room
	h.hub.BroadcastJSONToRoom(call.ChatID, answerJSON)

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





