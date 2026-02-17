package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"chat-backend/internal/database"
	"chat-backend/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type NotificationHandler struct {
	db *database.Database
}

func NewNotificationHandler(db *database.Database) *NotificationHandler {
	return &NotificationHandler{db: db}
}

// GetNotifications returns all notifications for the current user
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := h.db.MongoDB.Collection("notifications").Find(
		context.Background(),
		bson.M{"user_id": userIDObj},
		opts,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}
	defer cursor.Close(context.Background())

	var notifications []models.Notification
	if err := cursor.All(context.Background(), &notifications); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"notifications": notifications})
}

// MarkNotificationsRead marks notifications as read
func (h *NotificationHandler) MarkNotificationsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	filter := bson.M{"user_id": userIDObj}
	if len(req.IDs) > 0 {
		ids := make([]primitive.ObjectID, 0, len(req.IDs))
		for _, idStr := range req.IDs {
			if id, err := primitive.ObjectIDFromHex(idStr); err == nil {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			filter["_id"] = bson.M{"$in": ids}
		}
	}

	_, err := h.db.MongoDB.Collection("notifications").UpdateMany(
		context.Background(),
		filter,
		bson.M{"$set": bson.M{"read_at": now}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notifications as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notifications marked as read"})
}

// GetUnreadCount returns the count of unread notifications
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	count, err := h.db.MongoDB.Collection("notifications").CountDocuments(
		context.Background(),
		bson.M{
			"user_id": userIDObj,
			"read_at": nil,
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count unread notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}
