package handlers

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"chat-backend/internal/database"
	"chat-backend/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	TargetTypePhone      = "phone"
	TargetTypeCarNumber  = "car_number"
	TargetTypePersonName = "person_name"
)

func normalizeTargetValue(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, " ", ""))
}

type ProfileCommentHandler struct {
	db *database.Database
}

func NewProfileCommentHandler(db *database.Database) *ProfileCommentHandler {
	return &ProfileCommentHandler{db: db}
}

type CreateProfileCommentRequest struct {
	TargetUserID string `json:"target_user_id"` // optional when target_type+target_value used
	TargetType   string `json:"target_type"`    // "phone" | "car_number" | "person_name"
	TargetValue  string `json:"target_value"`   // phone, car plate, or person name
	Text         string `json:"text" binding:"required"`
}

// CreateProfileCommentByPhoneRequest: guest/public comment. Supports phone (notify+deletable by owner) or car_number/person_name (search-only).
type CreateProfileCommentByPhoneRequest struct {
	PhoneNumber string `json:"phone_number"` // legacy: when target_type is empty, treated as phone
	TargetType  string `json:"target_type"` // "phone" | "car_number" | "person_name"
	TargetValue string `json:"target_value"`
	Text        string `json:"text" binding:"required"`
}

// CreateProfileCommentByPhone creates a comment (guest). For "phone": user must exist, notification sent, owner can delete. For "car_number"/"person_name": only searchable.
func (h *ProfileCommentHandler) CreateProfileCommentByPhone(c *gin.Context) {
	var req CreateProfileCommentByPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.TargetType = strings.TrimSpace(strings.ToLower(req.TargetType))
	if req.TargetType == "" {
		req.TargetType = TargetTypePhone
	}
	if req.TargetType != TargetTypePhone && req.TargetType != TargetTypeCarNumber && req.TargetType != TargetTypePersonName {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_type must be phone, car_number, or person_name"})
		return
	}

	var targetValue string
	if req.TargetType == TargetTypePhone {
		targetValue = normalizeTargetValue(req.PhoneNumber)
		if targetValue == "" {
			targetValue = normalizeTargetValue(req.TargetValue)
		}
		if targetValue == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phone number"})
			return
		}
	} else {
		targetValue = strings.TrimSpace(req.TargetValue)
		if targetValue == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "target_value required for car_number and person_name"})
			return
		}
	}

	comment := models.ProfileComment{
		ID:          primitive.NewObjectID(),
		TargetType:  req.TargetType,
		TargetValue: targetValue,
		CommenterID: primitive.NilObjectID,
		Text:        req.Text,
		CreatedAt:   time.Now(),
	}

	if req.TargetType == TargetTypePhone {
		var targetUser models.User
		err := h.db.MongoDB.Collection("users").FindOne(
			context.Background(),
			bson.M{"phone_number": targetValue},
		).Decode(&targetUser)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found with this phone number"})
			return
		}
		comment.TargetUserID = &targetUser.ID
		if _, err := h.db.MongoDB.Collection("profile_comments").InsertOne(context.Background(), comment); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
			return
		}
		notification := models.Notification{
			ID:        primitive.NewObjectID(),
			UserID:    targetUser.ID,
			Type:      "profile_comment",
			Title:     "Yeni şərh",
			Body:      "Nömrənizə yeni şərh yazıldı",
			Data: map[string]interface{}{
				"comment_id": comment.ID.Hex(),
				"phone":      targetUser.PhoneNumber,
			},
			CreatedAt: time.Now(),
		}
		h.db.MongoDB.Collection("notifications").InsertOne(context.Background(), notification)
	} else {
		comment.TargetUserID = nil
		if _, err := h.db.MongoDB.Collection("profile_comments").InsertOne(context.Background(), comment); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           comment.ID,
		"target_type":  comment.TargetType,
		"target_value": comment.TargetValue,
		"text":         comment.Text,
		"created_at":   comment.CreatedAt,
	})
}

// CreateProfileComment creates a comment (authenticated). Supports target_user_id (phone) or target_type+target_value (phone/car_number/person_name).
func (h *ProfileCommentHandler) CreateProfileComment(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var req CreateProfileCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.TargetType = strings.TrimSpace(strings.ToLower(req.TargetType))
	if req.TargetType == "" && req.TargetUserID != "" {
		req.TargetType = TargetTypePhone
	}
	if req.TargetType != "" && req.TargetType != TargetTypePhone && req.TargetType != TargetTypeCarNumber && req.TargetType != TargetTypePersonName {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_type must be phone, car_number, or person_name"})
		return
	}

	var targetID primitive.ObjectID
	usePhone := req.TargetType == TargetTypePhone || req.TargetUserID != ""
	if req.TargetUserID != "" {
		var err error
		targetID, err = primitive.ObjectIDFromHex(req.TargetUserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target_user_id"})
			return
		}
	} else if req.TargetType == TargetTypePhone && req.TargetValue != "" {
		phone := normalizeTargetValue(req.TargetValue)
		var targetUser models.User
		err := h.db.MongoDB.Collection("users").FindOne(
			context.Background(),
			bson.M{"phone_number": phone},
		).Decode(&targetUser)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found with this phone number"})
			return
		}
		targetID = targetUser.ID
	}

	targetValue := strings.TrimSpace(req.TargetValue)
	if usePhone {
		if targetID == userIDObj {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot comment on yourself"})
			return
		}
		comment := models.ProfileComment{
			ID:           primitive.NewObjectID(),
			TargetType:   TargetTypePhone,
			TargetValue:  targetValue,
			TargetUserID: &targetID,
			CommenterID:  userIDObj,
			Text:         req.Text,
			CreatedAt:    time.Now(),
		}
		if targetValue == "" {
			var u models.User
			if h.db.MongoDB.Collection("users").FindOne(context.Background(), bson.M{"_id": targetID}).Decode(&u) == nil {
				comment.TargetValue = u.PhoneNumber
			}
		}
		if _, err := h.db.MongoDB.Collection("profile_comments").InsertOne(context.Background(), comment); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
			return
		}
		var targetUser models.User
		if h.db.MongoDB.Collection("users").FindOne(context.Background(), bson.M{"_id": targetID}).Decode(&targetUser) == nil {
			notification := models.Notification{
				ID:        primitive.NewObjectID(),
				UserID:    targetID,
				Type:      "profile_comment",
				Title:     "Yeni şərh",
				Body:      "Nömrənizə yeni şərh yazıldı",
				Data: map[string]interface{}{
					"comment_id": comment.ID.Hex(),
					"phone":      targetUser.PhoneNumber,
				},
				CreatedAt: time.Now(),
			}
			h.db.MongoDB.Collection("notifications").InsertOne(context.Background(), notification)
		}
		c.JSON(http.StatusCreated, gin.H{
			"id":             comment.ID,
			"target_user_id": targetID.Hex(),
			"target_type":    comment.TargetType,
			"text":           comment.Text,
			"created_at":     comment.CreatedAt,
		})
		return
	}

	// car_number or person_name
	if req.TargetType == "" || (req.TargetType != TargetTypeCarNumber && req.TargetType != TargetTypePersonName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_type and target_value required (car_number or person_name)"})
		return
	}
	if targetValue == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_value required"})
		return
	}
	comment := models.ProfileComment{
		ID:           primitive.NewObjectID(),
		TargetType:   req.TargetType,
		TargetValue:  targetValue,
		TargetUserID: nil,
		CommenterID:  userIDObj,
		Text:         req.Text,
		CreatedAt:    time.Now(),
	}
	if _, err := h.db.MongoDB.Collection("profile_comments").InsertOne(context.Background(), comment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":           comment.ID,
		"target_type":  comment.TargetType,
		"target_value": comment.TargetValue,
		"text":         comment.Text,
		"created_at":   comment.CreatedAt,
	})
}

// GetProfileComments returns comments about a user. Author is never exposed.
// Can query by target_user_id or phone_number
func (h *ProfileCommentHandler) GetProfileComments(c *gin.Context) {
	targetIDStr := c.Query("target_user_id")
	phoneNumber := c.Query("phone_number")
	
	var targetID primitive.ObjectID
	var err error
	
	if targetIDStr != "" {
		targetID, err = primitive.ObjectIDFromHex(targetIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target_user_id"})
			return
		}
	} else if phoneNumber != "" {
		// Find user by phone number
		var targetUser models.User
		err = h.db.MongoDB.Collection("users").FindOne(
			context.Background(),
			bson.M{"phone_number": phoneNumber},
		).Decode(&targetUser)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found with this phone number"})
			return
		}
		targetID = targetUser.ID
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_user_id or phone_number required"})
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
	for _, pc := range comments {
		out = append(out, map[string]interface{}{
			"id":            pc.ID,
			"text":          pc.Text,
			"like_count":    pc.LikeCount,
			"dislike_count": pc.DislikeCount,
			"created_at":    pc.CreatedAt,
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

	// Only the target user (profile owner) can reply; only phone-type comments have target_user_id
	if comment.TargetUserID == nil || *comment.TargetUserID != userIDObj {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the profile owner can reply to this comment"})
		return
	}

	commenterID := comment.CommenterID
	if commenterID == primitive.NilObjectID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot reply to a comment from a guest"})
		return
	}
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

// SearchProfileComments returns comments whose target_value matches the query (any type). Used to find car_number/person_name comments.
func (h *ProfileCommentHandler) SearchProfileComments(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q required"})
		return
	}
	// Also find by phone: if q is a registered phone, include comments for that user
	var targetUserID *primitive.ObjectID
	var targetUser models.User
	if err := h.db.MongoDB.Collection("users").FindOne(
		context.Background(),
		bson.M{"phone_number": normalizeTargetValue(q)},
	).Decode(&targetUser); err == nil {
		targetUserID = &targetUser.ID
	}
	escaped := regexp.QuoteMeta(q)
	pattern := ".*" + escaped + ".*"
	orConditions := []bson.M{
		{"target_value": bson.M{"$regex": pattern, "$options": "i"}},
	}
	if targetUserID != nil {
		orConditions = append(orConditions, bson.M{"target_user_id": targetUserID})
	}
	filter := bson.M{"$or": orConditions}
	cursor, err := h.db.MongoDB.Collection("profile_comments").Find(
		context.Background(),
		filter,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search comments"})
		return
	}
	defer cursor.Close(context.Background())
	var comments []models.ProfileComment
	if err := cursor.All(context.Background(), &comments); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode comments"})
		return
	}
	out := make([]map[string]interface{}, 0, len(comments))
	for _, c := range comments {
		out = append(out, map[string]interface{}{
			"id":           c.ID,
			"target_type":  c.TargetType,
			"target_value": c.TargetValue,
			"text":         c.Text,
			"created_at":   c.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, out)
}

// DeleteProfileComment allows the target user (number owner) to delete only phone-type comments about them.
func (h *ProfileCommentHandler) DeleteProfileComment(c *gin.Context) {
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
	if comment.TargetUserID == nil || *comment.TargetUserID != userIDObj {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the number owner can delete this comment"})
		return
	}
	_, err = h.db.MongoDB.Collection("profile_comments").DeleteOne(
		context.Background(),
		bson.M{"_id": commentID},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete comment"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Comment deleted"})
}
