package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"chat-backend/internal/database"
	"chat-backend/internal/models"
	"chat-backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	// CallTimeoutDuration is the maximum time a call can stay in "ringing" status
	CallTimeoutDuration = 60 * time.Second // 60 seconds timeout
	// CallTimeoutCheckInterval is how often we check for timed-out calls
	CallTimeoutCheckInterval = 30 * time.Second // Check every 30 seconds
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

	// Broadcast call via WebSocket to call members ONLY
	// CRITICAL: Use SINGLE delivery path to prevent duplicates
	// Prefer direct send to call members; room broadcast is fallback only
	callNotification := map[string]interface{}{
		"type":       "call",
		"call_id":    call.ID.Hex(),
		"chat_id":    chatID.Hex(),
		"call_type":  call.Type,
		"caller_id":  call.CallerID.Hex(),
		"status":     call.Status,
		"message_id": primitive.NewObjectID().Hex(), // Unique message ID for deduplication
		"timestamp":  time.Now().Unix(),
	}
	callJSON, _ := json.Marshal(callNotification)

	// PRIMARY PATH: Send directly to call members (excludes caller automatically)
	log.Printf("📞 Call initiated call_id=%s caller=%s: sending to %d call members",
		call.ID.Hex(), call.CallerID.Hex(), len(members))
	for _, memberID := range members {
		if memberID == call.CallerID {
			continue // Skip caller
		}
		h.hub.SendJSONToUser(memberID, callJSON)
	}
	
	// FALLBACK PATH: Broadcast to room EXCLUDING caller to prevent caller receiving their own call event
	// This ensures users actively in chat receive the call even if not in call members list
	// CRITICAL: Exclude sender (caller) to prevent duplicate/self-call events
	h.hub.BroadcastJSONToRoomExcludingSender(chatID, call.CallerID, callJSON)

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

	now := time.Now()
	updateData := bson.M{
		"status":     "active",
		"answered_at": now,
		"answered_by": userIDObj,
	}
	
	_, err = h.db.MongoDB.Collection("calls").UpdateOne(
		context.Background(),
		bson.M{"_id": callID},
		bson.M{"$set": updateData},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to answer call"})
		return
	}

	// Notify caller that call was answered
	// CRITICAL: Use SINGLE delivery path to prevent duplicates
	answerNotification := map[string]interface{}{
		"type":        "call_answered",
		"call_id":     callID.Hex(),
		"chat_id":     call.ChatID.Hex(),
		"call_type":   call.Type,
		"status":      "active",
		"answered_by": userIDObj.Hex(),
		"answered_at": now,
		"message_id":  primitive.NewObjectID().Hex(), // Unique message ID for deduplication
		"timestamp":   now.Unix(),
	}
	answerJSON, _ := json.Marshal(answerNotification)
	
	// PRIMARY PATH: Send directly to caller
	log.Printf("📞 Call answered call_id=%s answered_by=%s caller=%s: sending call_answered to caller",
		callID.Hex(), userIDObj.Hex(), call.CallerID.Hex())
	h.hub.SendJSONToUser(call.CallerID, answerJSON)
	
	// FALLBACK PATH: Also broadcast to room (safe because SendJSONToUser already delivered to caller)
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

	// Load call to get chat ID and members
	var call models.Call
	err = h.db.MongoDB.Collection("calls").FindOne(
		context.Background(),
		bson.M{"_id": callID},
	).Decode(&call)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Call not found"})
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

	// Notify all call members that the call has ended
	// CRITICAL: Use SINGLE delivery path to prevent duplicates
	endNotification := map[string]interface{}{
		"type":       "call_ended",
		"call_id":    callID.Hex(),
		"chat_id":    call.ChatID.Hex(),
		"call_type":  call.Type,
		"status":     "ended",
		"message_id": primitive.NewObjectID().Hex(),
		"timestamp":  now.Unix(),
	}
	endJSON, _ := json.Marshal(endNotification)
	
	// PRIMARY PATH: Send directly to call members
	log.Printf("📞 Call ended call_id=%s: sending to %d call members", callID.Hex(), len(call.Members))
	for _, memberID := range call.Members {
		h.hub.SendJSONToUser(memberID, endJSON)
	}
	
	// FALLBACK PATH: Also broadcast to room (safe because SendJSONToUser already delivered)
	h.hub.BroadcastJSONToRoom(call.ChatID, endJSON)

	c.JSON(http.StatusOK, gin.H{"message": "Call ended"})
}

// StartCallTimeoutChecker starts a background goroutine that periodically checks for timed-out calls
func (h *CallHandler) StartCallTimeoutChecker() {
	go func() {
		ticker := time.NewTicker(CallTimeoutCheckInterval)
		defer ticker.Stop()

		for range ticker.C {
			h.checkTimedOutCalls()
		}
	}()
	log.Println("✅ Call timeout checker started")
}

// checkTimedOutCalls finds and ends calls that have been ringing for too long
func (h *CallHandler) checkTimedOutCalls() {
	ctx := context.Background()
	timeoutThreshold := time.Now().Add(-CallTimeoutDuration)

	// Find calls that are still ringing and started before the threshold
	filter := bson.M{
		"status":    "ringing",
		"started_at": bson.M{"$lt": timeoutThreshold},
	}

	cursor, err := h.db.MongoDB.Collection("calls").Find(ctx, filter)
	if err != nil {
		log.Printf("Error finding timed-out calls: %v", err)
		return
	}
	defer cursor.Close(ctx)

	var timedOutCalls []models.Call
	if err := cursor.All(ctx, &timedOutCalls); err != nil {
		log.Printf("Error reading timed-out calls: %v", err)
		return
	}

	for _, call := range timedOutCalls {
		now := time.Now()
		_, err := h.db.MongoDB.Collection("calls").UpdateOne(
			ctx,
			bson.M{"_id": call.ID},
			bson.M{"$set": bson.M{
				"status":   "ended",
				"ended_at": now,
			}},
		)

		if err != nil {
			log.Printf("Error ending timed-out call %s: %v", call.ID.Hex(), err)
			continue
		}

		log.Printf("⏱️ Call %s timed out after %v", call.ID.Hex(), CallTimeoutDuration)

		// Notify all call members that the call timed out
		endNotification := map[string]interface{}{
			"type":       "call_ended",
			"call_id":    call.ID.Hex(),
			"chat_id":    call.ChatID.Hex(),
			"call_type":  call.Type,
			"status":     "ended",
			"reason":     "timeout",
			"message_id": primitive.NewObjectID().Hex(),
			"timestamp":  time.Now().Unix(),
		}
		endJSON, _ := json.Marshal(endNotification)

		// PRIMARY PATH: Send directly to call members
		for _, memberID := range call.Members {
			h.hub.SendJSONToUser(memberID, endJSON)
		}
		// FALLBACK PATH: Also broadcast to room
		h.hub.BroadcastJSONToRoom(call.ChatID, endJSON)
	}

	if len(timedOutCalls) > 0 {
		log.Printf("⏱️ Ended %d timed-out call(s)", len(timedOutCalls))
	}
}





