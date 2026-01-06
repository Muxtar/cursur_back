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

type LikeHandler struct {
	db *database.Database
}

func NewLikeHandler(db *database.Database) *LikeHandler {
	return &LikeHandler{db: db}
}

// LikeProduct likes a product
func (h *LikeHandler) LikeProduct(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	productID := c.Param("product_id")
	productObjID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// Check if product exists
	var product models.Product
	err = h.db.MongoDB.Collection("products").FindOne(
		context.Background(),
		bson.M{"_id": productObjID},
	).Decode(&product)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Check if already liked
	var existingLike models.Like
	err = h.db.MongoDB.Collection("likes").FindOne(
		context.Background(),
		bson.M{"product_id": productObjID, "user_id": userIDObj},
	).Decode(&existingLike)

	if err == nil {
		// Already liked, return success
		c.JSON(http.StatusOK, gin.H{"message": "Product already liked", "liked": true})
		return
	}

	// Create like
	like := models.Like{
		ProductID: &productObjID,
		UserID:     userIDObj,
		CreatedAt:  time.Now(),
	}

	_, err = h.db.MongoDB.Collection("likes").InsertOne(context.Background(), like)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to like product"})
		return
	}

	// Increment like count
	h.db.MongoDB.Collection("products").UpdateOne(
		context.Background(),
		bson.M{"_id": productObjID},
		bson.M{"$inc": bson.M{"like_count": 1}},
	)

	c.JSON(http.StatusOK, gin.H{"message": "Product liked", "liked": true})
}

// UnlikeProduct unlikes a product
func (h *LikeHandler) UnlikeProduct(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	productID := c.Param("product_id")
	productObjID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// Delete like
	result, err := h.db.MongoDB.Collection("likes").DeleteOne(
		context.Background(),
		bson.M{"product_id": productObjID, "user_id": userIDObj},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlike product"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Like not found"})
		return
	}

	// Decrement like count
	h.db.MongoDB.Collection("products").UpdateOne(
		context.Background(),
		bson.M{"_id": productObjID},
		bson.M{"$inc": bson.M{"like_count": -1}},
	)

	c.JSON(http.StatusOK, gin.H{"message": "Product unliked", "liked": false})
}

// LikeComment likes a comment
func (h *LikeHandler) LikeComment(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	commentID := c.Param("comment_id")
	commentObjID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	// Check if comment exists
	var comment models.Comment
	err = h.db.MongoDB.Collection("comments").FindOne(
		context.Background(),
		bson.M{"_id": commentObjID},
	).Decode(&comment)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	// Check if already liked
	var existingLike models.Like
	err = h.db.MongoDB.Collection("likes").FindOne(
		context.Background(),
		bson.M{"comment_id": commentObjID, "user_id": userIDObj},
	).Decode(&existingLike)

	if err == nil {
		c.JSON(http.StatusOK, gin.H{"message": "Comment already liked", "liked": true})
		return
	}

	// Create like
	like := models.Like{
		CommentID: &commentObjID,
		UserID:     userIDObj,
		CreatedAt:  time.Now(),
	}

	_, err = h.db.MongoDB.Collection("likes").InsertOne(context.Background(), like)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to like comment"})
		return
	}

	// Increment like count
	h.db.MongoDB.Collection("comments").UpdateOne(
		context.Background(),
		bson.M{"_id": commentObjID},
		bson.M{"$inc": bson.M{"like_count": 1}},
	)

	c.JSON(http.StatusOK, gin.H{"message": "Comment liked", "liked": true})
}

// UnlikeComment unlikes a comment
func (h *LikeHandler) UnlikeComment(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	commentID := c.Param("comment_id")
	commentObjID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	// Delete like
	result, err := h.db.MongoDB.Collection("likes").DeleteOne(
		context.Background(),
		bson.M{"comment_id": commentObjID, "user_id": userIDObj},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlike comment"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Like not found"})
		return
	}

	// Decrement like count
	h.db.MongoDB.Collection("comments").UpdateOne(
		context.Background(),
		bson.M{"_id": commentObjID},
		bson.M{"$inc": bson.M{"like_count": -1}},
	)

	c.JSON(http.StatusOK, gin.H{"message": "Comment unliked", "liked": false})
}

// GetProductLikes gets all users who liked a product
func (h *LikeHandler) GetProductLikes(c *gin.Context) {
	productID := c.Param("product_id")
	productObjID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	cursor, err := h.db.MongoDB.Collection("likes").Find(
		context.Background(),
		bson.M{"product_id": productObjID},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch likes"})
		return
	}
	defer cursor.Close(context.Background())

	var likes []models.Like
	if err := cursor.All(context.Background(), &likes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode likes"})
		return
	}

	// Get user info for each like
	type LikeWithUser struct {
		Like models.Like `json:"like"`
		User gin.H       `json:"user"`
	}

	result := make([]LikeWithUser, 0, len(likes))
	for _, like := range likes {
		var user models.User
		h.db.MongoDB.Collection("users").FindOne(context.Background(), bson.M{"_id": like.UserID}).Decode(&user)

		result = append(result, LikeWithUser{
			Like: like,
			User: gin.H{
				"id":       user.ID,
				"username": user.Username,
				"avatar":   user.Avatar,
			},
		})
	}

	c.JSON(http.StatusOK, result)
}



