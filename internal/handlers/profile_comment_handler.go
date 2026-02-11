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

type ProfileCommentHandler struct {
	db *database.Database
}

func NewProfileCommentHandler(db *database.Database) *ProfileCommentHandler {
	return &ProfileCommentHandler{db: db}
}

type CreateProfileCommentRequest struct {
	TargetUserID string `json:"target_user_id" binding:"required"`
	Text         string `json:"text" binding:"required"`
}

// CreateProfileComment creates an anonymous comment about a user. Commenter is stored but not exposed.
func (h *ProfileCommentHandler) CreateProfileComment(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var req CreateProfileCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	targetID, err := primitive.ObjectIDFromHex(req.TargetUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target_user_id"})
		return
	}
	if targetID == userIDObj {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot comment on yourself"})
		return
	}

	comment := models.ProfileComment{
		ID:           primitive.NewObjectID(),
		TargetUserID: targetID,
		CommenterID:  userIDObj,
		Text:         req.Text,
		CreatedAt:    time.Now(),
	}
	if _, err := h.db.MongoDB.Collection("profile_comments").InsertOne(context.Background(), comment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":             comment.ID,
		"target_user_id": req.TargetUserID,
		"text":           comment.Text,
		"created_at":     comment.CreatedAt,
	})
}

// GetProfileComments returns comments about a user. Author is never exposed.
func (h *ProfileCommentHandler) GetProfileComments(c *gin.Context) {
	targetIDStr := c.Query("target_user_id")
	if targetIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_user_id required"})
		return
	}
	targetID, err := primitive.ObjectIDFromHex(targetIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target_user_id"})
		return
	}

	cursor, err := h.db.MongoDB.Collection("profile_comments").Find(
		context.Background(),
		bson.M{"target_user_id": targetID},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments"})
		return
	}
	defer cursor.Close(context.Background())

	var comments []models.ProfileComment
	if err := cursor.All(context.Background(), &comments); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode comments"})
		return
	}

	// Return without commenter_id (anonymous)
	out := make([]map[string]interface{}, 0, len(comments))
	for _, c := range comments {
		out = append(out, map[string]interface{}{
			"id":         c.ID,
			"text":       c.Text,
			"created_at": c.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, out)
}

// ReplyToProfileComment lets the profile owner start a chat with the commenter (commenter stays anonymous).
func (h *ProfileCommentHandler) ReplyToProfileComment(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	commentIDStr := c.Param("comment_id")
	commentID, err := primitive.ObjectIDFromHex(commentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment_id"})
		return
	}

	var comment models.ProfileComment
	err = h.db.MongoDB.Collection("profile_comments").FindOne(
		context.Background(),
		bson.M{"_id": commentID},
	).Decode(&comment)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	// Only the target user (profile owner) can reply
	if comment.TargetUserID != userIDObj {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the profile owner can reply to this comment"})
		return
	}

	commenterID := comment.CommenterID
	if commenterID == userIDObj {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot reply to your own comment"})
		return
	}

	// Find or create direct chat between profile owner and commenter; make chat anonymous FROM commenter so owner sees "Anonymous"
	members := []primitive.ObjectID{userIDObj, commenterID}
	var chat models.Chat
	err = h.db.MongoDB.Collection("chats").FindOne(
		context.Background(),
		bson.M{
			"type": "direct",
			"$and": []bson.M{
				{"members": bson.M{"$all": members}},
				{"members": bson.M{"$size": 2}},
			},
		},
	).Decode(&chat)
	if err != nil {
		// Create new chat; commenter is anonymous to profile owner
		chat = models.Chat{
			ID:                  primitive.NewObjectID(),
			Type:                "direct",
			Members:             members,
			AnonymousFromUserID: &commenterID,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		_, err = h.db.MongoDB.Collection("chats").InsertOne(context.Background(), chat)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"chat_id": chat.ID.Hex(),
		"message": "Chat created; you can message the commenter (they appear as Anonymous)",
	})
}
