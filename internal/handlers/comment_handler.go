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
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CommentHandler struct {
	db *database.Database
}

func NewCommentHandler(db *database.Database) *CommentHandler {
	return &CommentHandler{db: db}
}

// CreateComment creates a new comment on a product
func (h *CommentHandler) CreateComment(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	productID := c.Param("product_id")
	productObjID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var comment models.Comment
	if err := c.ShouldBindJSON(&comment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify product exists
	var product models.Product
	err = h.db.MongoDB.Collection("products").FindOne(
		context.Background(),
		bson.M{"_id": productObjID},
	).Decode(&product)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Set comment data
	comment.ProductID = productObjID
	comment.UserID = userIDObj
	comment.CreatedAt = time.Now()
	comment.UpdatedAt = time.Now()
	comment.LikeCount = 0
	comment.IsSpam = false

	// Handle reply (parent comment)
	if comment.ParentID != nil {
		var parentComment models.Comment
		err := h.db.MongoDB.Collection("comments").FindOne(
			context.Background(),
			bson.M{"_id": *comment.ParentID, "product_id": productObjID},
		).Decode(&parentComment)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parent comment not found"})
			return
		}
	}

	result, err := h.db.MongoDB.Collection("comments").InsertOne(context.Background(), comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
		return
	}

	comment.ID = result.InsertedID.(primitive.ObjectID)

	// Increment comment count on product
	h.db.MongoDB.Collection("products").UpdateOne(
		context.Background(),
		bson.M{"_id": productObjID},
		bson.M{"$inc": bson.M{"comment_count": 1}},
	)

	// Get user info
	var user models.User
	h.db.MongoDB.Collection("users").FindOne(context.Background(), bson.M{"_id": userIDObj}).Decode(&user)

	response := gin.H{
		"comment": comment,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"avatar":   user.Avatar,
		},
	}

	c.JSON(http.StatusCreated, response)
}

// GetComments gets all comments for a product
func (h *CommentHandler) GetComments(c *gin.Context) {
	productID := c.Param("product_id")
	productObjID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// Pagination
	page := 1
	limit := 50
	skip := (page - 1) * limit

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}).SetSkip(int64(skip)).SetLimit(int64(limit))
	cursor, err := h.db.MongoDB.Collection("comments").Find(
		context.Background(),
		bson.M{"product_id": productObjID, "parent_id": nil, "is_spam": false}, // Only top-level comments
		opts,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments"})
		return
	}
	defer cursor.Close(context.Background())

	var comments []models.Comment
	if err := cursor.All(context.Background(), &comments); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode comments"})
		return
	}

	// Populate user info and replies for each comment
	type CommentWithUser struct {
		models.Comment
		User    gin.H                `json:"user"`
		Replies []CommentWithUser    `json:"replies,omitempty"`
		IsLiked bool                 `json:"is_liked"`
	}

	result := make([]CommentWithUser, 0, len(comments))
	userID, exists := c.Get("user_id")
	var userIDObj primitive.ObjectID
	if exists {
		userIDObj = userID.(primitive.ObjectID)
	}

	for _, comment := range comments {
		var user models.User
		h.db.MongoDB.Collection("users").FindOne(context.Background(), bson.M{"_id": comment.UserID}).Decode(&user)

		// Check if user liked this comment
		isLiked := false
		if exists {
			var like models.Like
			err := h.db.MongoDB.Collection("likes").FindOne(
				context.Background(),
				bson.M{"comment_id": comment.ID, "user_id": userIDObj},
			).Decode(&like)
			isLiked = err == nil
		}

		// Get replies
		replyCursor, _ := h.db.MongoDB.Collection("comments").Find(
			context.Background(),
			bson.M{"parent_id": comment.ID, "is_spam": false},
			options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}).SetLimit(10),
		)
		var replies []models.Comment
		replyCursor.All(context.Background(), &replies)
		replyCursor.Close(context.Background())

		repliesWithUser := make([]CommentWithUser, 0, len(replies))
		for _, reply := range replies {
			var replyUser models.User
			h.db.MongoDB.Collection("users").FindOne(context.Background(), bson.M{"_id": reply.UserID}).Decode(&replyUser)

			replyIsLiked := false
			if exists {
				var like models.Like
				err := h.db.MongoDB.Collection("likes").FindOne(
					context.Background(),
					bson.M{"comment_id": reply.ID, "user_id": userIDObj},
				).Decode(&like)
				replyIsLiked = err == nil
			}

			repliesWithUser = append(repliesWithUser, CommentWithUser{
				Comment: reply,
				User: gin.H{
					"id":       replyUser.ID,
					"username": replyUser.Username,
					"avatar":   replyUser.Avatar,
				},
				IsLiked: replyIsLiked,
			})
		}

		result = append(result, CommentWithUser{
			Comment: comment,
			User: gin.H{
				"id":       user.ID,
				"username": user.Username,
				"avatar":   user.Avatar,
			},
			Replies: repliesWithUser,
			IsLiked: isLiked,
		})
	}

	c.JSON(http.StatusOK, result)
}

// DeleteComment deletes a comment (only owner or admin can delete)
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	commentID := c.Param("comment_id")
	commentObjID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	// Check ownership
	var comment models.Comment
	err = h.db.MongoDB.Collection("comments").FindOne(
		context.Background(),
		bson.M{"_id": commentObjID},
	).Decode(&comment)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	if comment.UserID != userIDObj {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own comments"})
		return
	}

	// Delete comment
	_, err = h.db.MongoDB.Collection("comments").DeleteOne(
		context.Background(),
		bson.M{"_id": commentObjID},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete comment"})
		return
	}

	// Decrement comment count on product
	h.db.MongoDB.Collection("products").UpdateOne(
		context.Background(),
		bson.M{"_id": comment.ProductID},
		bson.M{"$inc": bson.M{"comment_count": -1}},
	)

	// Delete related likes
	h.db.MongoDB.Collection("likes").DeleteMany(context.Background(), bson.M{"comment_id": commentObjID})

	// Delete replies if any
	h.db.MongoDB.Collection("comments").DeleteMany(context.Background(), bson.M{"parent_id": commentObjID})

	c.JSON(http.StatusOK, gin.H{"message": "Comment deleted successfully"})
}

// ReportSpam reports a comment as spam
func (h *CommentHandler) ReportSpam(c *gin.Context) {
	commentID := c.Param("comment_id")
	commentObjID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	_, err = h.db.MongoDB.Collection("comments").UpdateOne(
		context.Background(),
		bson.M{"_id": commentObjID},
		bson.M{"$set": bson.M{"is_spam": true}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to report spam"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment reported as spam"})
}



