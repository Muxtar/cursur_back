package handlers

import (
	"context"
	"net/http"
	"time"

	"chat-backend/internal/database"
	"chat-backend/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChannelHandler struct {
	db *database.Database
}

func NewChannelHandler(db *database.Database) *ChannelHandler {
	return &ChannelHandler{db: db}
}

func (h *ChannelHandler) CreateChannel(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var req struct {
		ChannelName string `json:"channel_name" binding:"required"`
		Description string `json:"description,omitempty"`
		IsPublic    bool   `json:"is_public"`
		PublicLink  string `json:"public_link,omitempty"` // t.me/channelname
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	channel := models.Chat{
		ID:          primitive.NewObjectID(),
		Type:        "channel",
		Members:     []primitive.ObjectID{userIDObj},
		Admins: []models.AdminRole{
			{
				UserID:      userIDObj,
				Role:        "owner",
				Permissions: []string{"all"},
				GrantedAt:   time.Now(),
				GrantedBy:   userIDObj,
			},
		},
		GroupName:      req.ChannelName,
		Description:    req.Description,
		PublicLink:     req.PublicLink,
		IsBroadcast:    true,
		SubscriberCount: 1,
		ViewCount:      make(map[string]int),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err := h.db.MongoDB.Collection("chats").InsertOne(context.Background(), channel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create channel"})
		return
	}

	c.JSON(http.StatusCreated, channel)
}

func (h *ChannelHandler) Subscribe(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	channelIDStr := c.Param("channel_id")
	channelID, err := primitive.ObjectIDFromHex(channelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	var channel models.Chat
	err = h.db.MongoDB.Collection("chats").FindOne(
		context.Background(),
		bson.M{"_id": channelID, "type": "channel"},
	).Decode(&channel)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	// Add user to members
	_, err = h.db.MongoDB.Collection("chats").UpdateOne(
		context.Background(),
		bson.M{"_id": channelID},
		bson.M{"$addToSet": bson.M{"members": userIDObj}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to subscribe"})
		return
	}

	// Increment subscriber count
	_, err = h.db.MongoDB.Collection("chats").UpdateOne(
		context.Background(),
		bson.M{"_id": channelID},
		bson.M{"$inc": bson.M{"subscriber_count": 1}},
	)

	c.JSON(http.StatusOK, gin.H{"message": "Subscribed to channel"})
}

func (h *ChannelHandler) Unsubscribe(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	channelIDStr := c.Param("channel_id")
	channelID, err := primitive.ObjectIDFromHex(channelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	// Remove user from members
	_, err = h.db.MongoDB.Collection("chats").UpdateOne(
		context.Background(),
		bson.M{"_id": channelID},
		bson.M{"$pull": bson.M{"members": userIDObj}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unsubscribe"})
		return
	}

	// Decrement subscriber count
	_, err = h.db.MongoDB.Collection("chats").UpdateOne(
		context.Background(),
		bson.M{"_id": channelID},
		bson.M{"$inc": bson.M{"subscriber_count": -1}},
	)

	c.JSON(http.StatusOK, gin.H{"message": "Unsubscribed from channel"})
}

func (h *ChannelHandler) RecordView(c *gin.Context) {
	channelIDStr := c.Param("channel_id")
	channelID, err := primitive.ObjectIDFromHex(channelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	messageIDStr := c.Param("message_id")
	_, err = primitive.ObjectIDFromHex(messageIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	var channel models.Chat
	err = h.db.MongoDB.Collection("chats").FindOne(
		context.Background(),
		bson.M{"_id": channelID},
	).Decode(&channel)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	if channel.ViewCount == nil {
		channel.ViewCount = make(map[string]int)
	}
	channel.ViewCount[messageIDStr]++

	_, err = h.db.MongoDB.Collection("chats").UpdateOne(
		context.Background(),
		bson.M{"_id": channelID},
		bson.M{"$set": bson.M{"view_count": channel.ViewCount}},
	)

	c.JSON(http.StatusOK, gin.H{"view_count": channel.ViewCount[messageIDStr]})
}

func (h *ChannelHandler) GetStatistics(c *gin.Context) {
	channelIDStr := c.Param("channel_id")
	channelID, err := primitive.ObjectIDFromHex(channelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	var channel models.Chat
	err = h.db.MongoDB.Collection("chats").FindOne(
		context.Background(),
		bson.M{"_id": channelID, "type": "channel"},
	).Decode(&channel)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	c.JSON(http.StatusOK, channel.Statistics)
}





