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
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MessageHandler struct {
	db  *database.Database
	hub *websocket.Hub
}

func NewMessageHandler(db *database.Database, hub *websocket.Hub) *MessageHandler {
	return &MessageHandler{db: db, hub: hub}
}

type SendMessageRequest struct {
	Content         string    `json:"content"`
	MessageType     string    `json:"message_type" binding:"required"`
	FileURL         string    `json:"file_url,omitempty"`
	ThumbnailURL    string    `json:"thumbnail_url,omitempty"`
	FileName        string    `json:"file_name,omitempty"`
	FileSize        int64     `json:"file_size,omitempty"`
	Duration        int       `json:"duration,omitempty"`
	IsAnonymous     bool      `json:"is_anonymous"`
	IsSecret        bool      `json:"is_secret"`
	SelfDestructTTL int       `json:"self_destruct_ttl,omitempty"`
	ReplyToID       string    `json:"reply_to_id,omitempty"`
	Location        *models.MessageLocation `json:"location,omitempty"`
	Contact         *models.ContactInfo `json:"contact,omitempty"`
	Poll            *models.Poll `json:"poll,omitempty"`
	Mentions        []string  `json:"mentions,omitempty"`
	Formatting      *models.MessageFormatting `json:"formatting,omitempty"`
	LinkPreview     *models.LinkPreview `json:"link_preview,omitempty"`
	ScheduledFor    *time.Time `json:"scheduled_for,omitempty"`
	IsDraft         bool      `json:"is_draft"`
	BotCommand      string    `json:"bot_command,omitempty"`
}

func (h *MessageHandler) SendMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	// Support both /chats/:chat_id/messages and /messages endpoints
	chatIDStr := c.Param("chat_id")
	if chatIDStr == "" {
		// Try to get from body
		var body struct {
			ChatID string `json:"chat_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err == nil {
			chatIDStr = body.ChatID
		}
	}
	
	chatID, err := primitive.ObjectIDFromHex(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check slow mode
	var chat models.Chat
	err = h.db.MongoDB.Collection("chats").FindOne(
		context.Background(),
		bson.M{"_id": chatID},
	).Decode(&chat)

	if err == nil && chat.SlowMode > 0 {
		lastMessageKey := userIDObj.Hex()
		if lastTime, exists := chat.LastSlowModeMessage[lastMessageKey]; exists {
			timeSinceLastMessage := time.Since(lastTime)
			if timeSinceLastMessage < time.Duration(chat.SlowMode)*time.Second {
				remaining := chat.SlowMode - int(timeSinceLastMessage.Seconds())
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": "Slow mode active",
					"remaining_seconds": remaining,
				})
				return
			}
		}
	}

	// Parse reply to
	var replyToID *primitive.ObjectID
	if req.ReplyToID != "" {
		id, err := primitive.ObjectIDFromHex(req.ReplyToID)
		if err == nil {
			replyToID = &id
		}
	}

	// Parse mentions
	var mentions []primitive.ObjectID
	for _, mentionIDStr := range req.Mentions {
		id, err := primitive.ObjectIDFromHex(mentionIDStr)
		if err == nil {
			mentions = append(mentions, id)
		}
	}

	message := models.Message{
		ID:          primitive.NewObjectID(),
		ChatID:     chatID,
		SenderID:   userIDObj,
		Content:    req.Content,
		MessageType: req.MessageType,
		FileURL:    req.FileURL,
		ThumbnailURL: req.ThumbnailURL,
		FileName:   req.FileName,
		FileSize:   req.FileSize,
		Duration:   req.Duration,
		Status:     "sent",
		IsAnonymous: req.IsAnonymous,
		IsSecret:   req.IsSecret,
		SelfDestructTTL: req.SelfDestructTTL,
		ReplyToID:  replyToID,
		Location:    req.Location,
		Contact:     req.Contact,
		Poll:        req.Poll,
		Mentions:    mentions,
		Formatting:  *req.Formatting,
		LinkPreview: req.LinkPreview,
		ScheduledFor: req.ScheduledFor,
		IsDraft:     req.IsDraft,
		BotCommand:  req.BotCommand,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// If scheduled, don't send immediately
	if req.ScheduledFor != nil && req.ScheduledFor.After(time.Now()) {
		message.Status = "scheduled"
	} else if !req.IsDraft {
		message.Status = "sent"
	}

	_, err = h.db.MongoDB.Collection("messages").InsertOne(context.Background(), message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	// Update chat last message
	_, err = h.db.MongoDB.Collection("chats").UpdateOne(
		context.Background(),
		bson.M{"_id": chatID},
		bson.M{"$set": bson.M{
			"last_message_id": message.ID,
			"last_message_at": time.Now(),
			"updated_at": time.Now(),
		}},
	)

	// Update slow mode timestamp
	if chat.SlowMode > 0 {
		if chat.LastSlowModeMessage == nil {
			chat.LastSlowModeMessage = make(map[string]time.Time)
		}
		chat.LastSlowModeMessage[userIDObj.Hex()] = time.Now()
		_, err = h.db.MongoDB.Collection("chats").UpdateOne(
			context.Background(),
			bson.M{"_id": chatID},
			bson.M{"$set": bson.M{"last_slow_mode_message": chat.LastSlowModeMessage}},
		)
	}

	// Broadcast message via WebSocket
	if !req.IsDraft && (req.ScheduledFor == nil || req.ScheduledFor.Before(time.Now())) {
		h.hub.BroadcastToRoom(chatID, message)
	}

	c.JSON(http.StatusCreated, message)
}

func (h *MessageHandler) EditMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	messageIDStr := c.Param("message_id")
	messageID, err := primitive.ObjectIDFromHex(messageIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	now := time.Now()
	_, err = h.db.MongoDB.Collection("messages").UpdateOne(
		context.Background(),
		bson.M{"_id": messageID},
		bson.M{"$set": bson.M{
			"content":    req.Content,
			"is_edited":  true,
			"edited_at":  now,
			"updated_at": now,
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to edit message"})
		return
	}

	// Broadcast update
	h.hub.BroadcastToRoom(message.ChatID, message)

	c.JSON(http.StatusOK, gin.H{"message": "Message edited successfully"})
}

func (h *MessageHandler) DeleteMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	messageIDStr := c.Param("message_id")
	messageID, err := primitive.ObjectIDFromHex(messageIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	var req struct {
		DeleteForEveryone bool `json:"delete_for_everyone"`
	}
	c.ShouldBindJSON(&req)

	var message models.Message
	err = h.db.MongoDB.Collection("messages").FindOne(
		context.Background(),
		bson.M{"_id": messageID},
	).Decode(&message)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}

	now := time.Now()
	if req.DeleteForEveryone && message.SenderID == userIDObj {
		// Delete for everyone
		_, err = h.db.MongoDB.Collection("messages").UpdateOne(
			context.Background(),
			bson.M{"_id": messageID},
			bson.M{"$set": bson.M{
				"is_deleted": true,
				"deleted_at": now,
				"updated_at": now,
			}},
		)
	} else {
		// Delete for me only
		var deletedFor []primitive.ObjectID
		if message.DeletedFor != nil {
			deletedFor = message.DeletedFor
		}
		deletedFor = append(deletedFor, userIDObj)
		
		_, err = h.db.MongoDB.Collection("messages").UpdateOne(
			context.Background(),
			bson.M{"_id": messageID},
			bson.M{"$set": bson.M{
				"deleted_for": deletedFor,
				"updated_at":  now,
			}},
		)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message deleted successfully"})
}

func (h *MessageHandler) ForwardMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	messageIDStr := c.Param("message_id")
	messageID, err := primitive.ObjectIDFromHex(messageIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	var req struct {
		ChatIDs []string `json:"chat_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get original message
	var originalMessage models.Message
	err = h.db.MongoDB.Collection("messages").FindOne(
		context.Background(),
		bson.M{"_id": messageID},
	).Decode(&originalMessage)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}

	// Forward to each chat
	var forwardedMessages []models.Message
	for _, chatIDStr := range req.ChatIDs {
		chatID, err := primitive.ObjectIDFromHex(chatIDStr)
		if err != nil {
			continue
		}

		forwardedMessage := models.Message{
			ID:              primitive.NewObjectID(),
			ChatID:         chatID,
			SenderID:       userIDObj,
			Content:        originalMessage.Content,
			MessageType:    originalMessage.MessageType,
			FileURL:        originalMessage.FileURL,
			ThumbnailURL:   originalMessage.ThumbnailURL,
			FileName:       originalMessage.FileName,
			FileSize:       originalMessage.FileSize,
			Duration:       originalMessage.Duration,
			Status:         "sent",
			ForwardedFrom:   &originalMessage.ID,
			ForwardedFromChat: &originalMessage.ChatID,
			Location:        originalMessage.Location,
			Contact:         originalMessage.Contact,
			Poll:            originalMessage.Poll,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		_, err = h.db.MongoDB.Collection("messages").InsertOne(context.Background(), forwardedMessage)
		if err == nil {
			forwardedMessages = append(forwardedMessages, forwardedMessage)
			h.hub.BroadcastToRoom(chatID, forwardedMessage)
		}
	}

	c.JSON(http.StatusOK, gin.H{"messages": forwardedMessages})
}

func (h *MessageHandler) AddReaction(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	messageIDStr := c.Param("message_id")
	messageID, err := primitive.ObjectIDFromHex(messageIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	var req struct {
		Emoji string `json:"emoji" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var message models.Message
	err = h.db.MongoDB.Collection("messages").FindOne(
		context.Background(),
		bson.M{"_id": messageID},
	).Decode(&message)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}

	// Remove existing reaction from user
	var reactions []models.Reaction
	for _, reaction := range message.Reactions {
		if reaction.UserID != userIDObj {
			reactions = append(reactions, reaction)
		}
	}

	// Add new reaction
	reactions = append(reactions, models.Reaction{
		UserID:    userIDObj,
		Emoji:     req.Emoji,
		CreatedAt: time.Now(),
	})

	_, err = h.db.MongoDB.Collection("messages").UpdateOne(
		context.Background(),
		bson.M{"_id": messageID},
		bson.M{"$set": bson.M{
			"reactions": reactions,
			"updated_at": time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add reaction"})
		return
	}

	h.hub.BroadcastToRoom(message.ChatID, message)
	c.JSON(http.StatusOK, gin.H{"message": "Reaction added"})
}

func (h *MessageHandler) RemoveReaction(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	messageIDStr := c.Param("message_id")
	messageID, err := primitive.ObjectIDFromHex(messageIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	var message models.Message
	err = h.db.MongoDB.Collection("messages").FindOne(
		context.Background(),
		bson.M{"_id": messageID},
	).Decode(&message)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}

	// Remove user's reactions
	var reactions []models.Reaction
	for _, reaction := range message.Reactions {
		if reaction.UserID != userIDObj {
			reactions = append(reactions, reaction)
		}
	}

	_, err = h.db.MongoDB.Collection("messages").UpdateOne(
		context.Background(),
		bson.M{"_id": messageID},
		bson.M{"$set": bson.M{
			"reactions": reactions,
			"updated_at": time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove reaction"})
		return
	}

	h.hub.BroadcastToRoom(message.ChatID, message)
	c.JSON(http.StatusOK, gin.H{"message": "Reaction removed"})
}

func (h *MessageHandler) MarkAsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var req struct {
		ChatID     string   `json:"chat_id" binding:"required"`
		MessageIDs []string `json:"message_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chatID, err := primitive.ObjectIDFromHex(req.ChatID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	if len(req.MessageIDs) > 0 {
		// Mark specific messages as read
		for _, messageIDStr := range req.MessageIDs {
			messageID, err := primitive.ObjectIDFromHex(messageIDStr)
			if err != nil {
				continue
			}

			var message models.Message
			err = h.db.MongoDB.Collection("messages").FindOne(
				context.Background(),
				bson.M{"_id": messageID, "chat_id": chatID},
			).Decode(&message)

			if err != nil {
				continue
			}

			// Check if already read
			alreadyRead := false
			for _, receipt := range message.ReadBy {
				if receipt.UserID == userIDObj {
					alreadyRead = true
					break
				}
			}

			if !alreadyRead {
				readBy := message.ReadBy
				readBy = append(readBy, models.ReadReceipt{
					UserID: userIDObj,
					ReadAt: time.Now(),
				})

				status := "read"
				if len(readBy) < len(message.Mentions) {
					status = "delivered"
				}

				_, err = h.db.MongoDB.Collection("messages").UpdateOne(
					context.Background(),
					bson.M{"_id": messageID},
					bson.M{"$set": bson.M{
						"read_by": readBy,
						"status":  status,
						"updated_at": time.Now(),
					}},
				)
			}
		}
	} else {
		// Mark all unread messages in chat as read
		cursor, err := h.db.MongoDB.Collection("messages").Find(
			context.Background(),
			bson.M{
				"chat_id": chatID,
				"sender_id": bson.M{"$ne": userIDObj},
				"read_by.user_id": bson.M{"$ne": userIDObj},
			},
		)

		if err == nil {
			defer cursor.Close(context.Background())
			for cursor.Next(context.Background()) {
				var message models.Message
				if err := cursor.Decode(&message); err != nil {
					continue
				}

				readBy := message.ReadBy
				readBy = append(readBy, models.ReadReceipt{
					UserID: userIDObj,
					ReadAt: time.Now(),
				})

				_, err = h.db.MongoDB.Collection("messages").UpdateOne(
					context.Background(),
					bson.M{"_id": message.ID},
					bson.M{"$set": bson.M{
						"read_by": readBy,
						"status":  "read",
						"updated_at": time.Now(),
					}},
				)
			}
		}
	}

	// Reset unread count
	_, err = h.db.MongoDB.Collection("chats").UpdateOne(
		context.Background(),
		bson.M{"_id": chatID},
		bson.M{"$set": bson.M{
			"unread_count." + userIDObj.Hex(): 0,
		}},
	)

	c.JSON(http.StatusOK, gin.H{"message": "Messages marked as read"})
}

func (h *MessageHandler) PinMessage(c *gin.Context) {
	chatIDStr := c.Param("chat_id")
	chatID, err := primitive.ObjectIDFromHex(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	messageIDStr := c.Param("message_id")
	messageID, err := primitive.ObjectIDFromHex(messageIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	// Update message
	_, err = h.db.MongoDB.Collection("messages").UpdateOne(
		context.Background(),
		bson.M{"_id": messageID},
		bson.M{"$set": bson.M{
			"is_pinned": true,
			"pinned_at": time.Now(),
			"updated_at": time.Now(),
		}},
	)

	// Add to chat pinned messages
	var chat models.Chat
	err = h.db.MongoDB.Collection("chats").FindOne(
		context.Background(),
		bson.M{"_id": chatID},
	).Decode(&chat)

	if err == nil {
		pinnedMessages := chat.PinnedMessages
		pinnedMessages = append(pinnedMessages, messageID)
		
		_, err = h.db.MongoDB.Collection("chats").UpdateOne(
			context.Background(),
			bson.M{"_id": chatID},
			bson.M{"$set": bson.M{
				"pinned_messages": pinnedMessages,
			}},
		)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message pinned"})
}

func (h *MessageHandler) UnpinMessage(c *gin.Context) {
	chatIDStr := c.Param("chat_id")
	chatID, err := primitive.ObjectIDFromHex(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	messageIDStr := c.Param("message_id")
	messageID, err := primitive.ObjectIDFromHex(messageIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	// Update message
	_, err = h.db.MongoDB.Collection("messages").UpdateOne(
		context.Background(),
		bson.M{"_id": messageID},
		bson.M{"$set": bson.M{
			"is_pinned": false,
			"pinned_at": nil,
			"updated_at": time.Now(),
		}},
	)

	// Remove from chat pinned messages
	var chat models.Chat
	err = h.db.MongoDB.Collection("chats").FindOne(
		context.Background(),
		bson.M{"_id": chatID},
	).Decode(&chat)

	if err == nil {
		var pinnedMessages []primitive.ObjectID
		for _, id := range chat.PinnedMessages {
			if id != messageID {
				pinnedMessages = append(pinnedMessages, id)
			}
		}
		
		_, err = h.db.MongoDB.Collection("chats").UpdateOne(
			context.Background(),
			bson.M{"_id": chatID},
			bson.M{"$set": bson.M{
				"pinned_messages": pinnedMessages,
			}},
		)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message unpinned"})
}

func (h *MessageHandler) VotePoll(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	messageIDStr := c.Param("message_id")
	messageID, err := primitive.ObjectIDFromHex(messageIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	var req struct {
		OptionID string `json:"option_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var message models.Message
	err = h.db.MongoDB.Collection("messages").FindOne(
		context.Background(),
		bson.M{"_id": messageID},
	).Decode(&message)

	if err != nil || message.Poll == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Poll not found"})
		return
	}

	// Remove user's vote from all options
	for i := range message.Poll.Options {
		var votes []primitive.ObjectID
		for _, voteID := range message.Poll.Options[i].Votes {
			if voteID != userIDObj {
				votes = append(votes, voteID)
			}
		}
		message.Poll.Options[i].Votes = votes
	}

	// Add vote to selected option
	for i := range message.Poll.Options {
		if message.Poll.Options[i].ID == req.OptionID {
			message.Poll.Options[i].Votes = append(message.Poll.Options[i].Votes, userIDObj)
			break
		}
	}

	_, err = h.db.MongoDB.Collection("messages").UpdateOne(
		context.Background(),
		bson.M{"_id": messageID},
		bson.M{"$set": bson.M{
			"poll": message.Poll,
			"updated_at": time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to vote"})
		return
	}

	h.hub.BroadcastToRoom(message.ChatID, message)
	c.JSON(http.StatusOK, gin.H{"message": "Vote recorded"})
}

func (h *MessageHandler) SearchMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query required"})
		return
	}

	chatIDStr := c.Query("chat_id")
	var chatID *primitive.ObjectID
	if chatIDStr != "" {
		id, err := primitive.ObjectIDFromHex(chatIDStr)
		if err == nil {
			chatID = &id
		}
	}

	filter := bson.M{
		"$or": []bson.M{
			{"content": bson.M{"$regex": query, "$options": "i"}},
			{"file_name": bson.M{"$regex": query, "$options": "i"}},
		},
		"is_deleted": false,
		"deleted_for": bson.M{"$ne": userIDObj},
	}

	if chatID != nil {
		filter["chat_id"] = *chatID
	}

	cursor, err := h.db.MongoDB.Collection("messages").Find(
		context.Background(),
		filter,
		options.Find().SetSort(bson.M{"created_at": -1}).SetLimit(50),
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search messages"})
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

func (h *MessageHandler) TranslateMessage(c *gin.Context) {
	messageIDStr := c.Param("message_id")
	messageID, err := primitive.ObjectIDFromHex(messageIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	targetLang := c.Query("lang")
	if targetLang == "" {
		targetLang = "en"
	}

	var message models.Message
	err = h.db.MongoDB.Collection("messages").FindOne(
		context.Background(),
		bson.M{"_id": messageID},
	).Decode(&message)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}

	// In a real implementation, you would call a translation API here
	// For now, we'll just return a placeholder
	translatedText := "[Translation to " + targetLang + ": " + message.Content + "]"

	_, err = h.db.MongoDB.Collection("messages").UpdateOne(
		context.Background(),
		bson.M{"_id": messageID},
		bson.M{"$set": bson.M{
			"translated_text": translatedText,
			"translated_to": targetLang,
			"updated_at": time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to translate message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"translated_text": translatedText})
}

