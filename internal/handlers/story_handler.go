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

type StoryHandler struct {
	db *database.Database
}

func NewStoryHandler(db *database.Database) *StoryHandler {
	return &StoryHandler{db: db}
}

// ─── Request bodies ────────────────────────────────────────────────────────────

type CreateStoryRequest struct {
	Type      string  `json:"type" binding:"required"` // "media" | "product"
	MediaURL  string  `json:"media_url"`
	MediaType string  `json:"media_type"` // "image" | "video"
	Text      string  `json:"text"`
	ProductID *string `json:"product_id"`
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

// POST /stories
// Creates a new story (media or product share). Expires in 24 hours.
func (h *StoryHandler) CreateStory(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var req CreateStoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate by type
	if req.Type == "media" && req.MediaURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "media_url is required for type 'media'"})
		return
	}
	if req.Type == "product" && req.ProductID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id is required for type 'product'"})
		return
	}

	now := time.Now()
	story := models.Story{
		ID:        primitive.NewObjectID(),
		UserID:    userIDObj,
		Type:      req.Type,
		Text:      req.Text,
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	}

	if req.Type == "media" {
		story.MediaURL = req.MediaURL
		story.MediaType = req.MediaType
		if story.MediaType == "" {
			story.MediaType = "image"
		}
	}

	if req.Type == "product" && req.ProductID != nil {
		productObjID, err := primitive.ObjectIDFromHex(*req.ProductID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product_id"})
			return
		}
		// Verify product exists
		count, err := h.db.MongoDB.Collection("products").CountDocuments(
			context.Background(),
			bson.M{"_id": productObjID},
		)
		if err != nil || count == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
			return
		}
		story.ProductID = &productObjID
	}

	_, err := h.db.MongoDB.Collection("stories").InsertOne(context.Background(), story)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create story"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"story": story})
}

// GET /stories
// Returns active (non-expired) stories grouped by user.
// Each group includes the user's basic info and their stories array.
func (h *StoryHandler) GetStoryFeed(c *gin.Context) {
	ctx := context.Background()
	now := time.Now()

	// Find all non-expired stories, newest first
	findOpts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := h.db.MongoDB.Collection("stories").Find(
		ctx,
		bson.M{"expires_at": bson.M{"$gt": now}},
		findOpts,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stories"})
		return
	}
	defer cursor.Close(ctx)

	var stories []models.Story
	if err := cursor.All(ctx, &stories); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode stories"})
		return
	}

	// Group stories by user_id
	grouped := map[string][]models.Story{}
	order := []string{} // preserve insertion order (newest story first per user)
	for _, s := range stories {
		key := s.UserID.Hex()
		if _, exists := grouped[key]; !exists {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], s)
	}

	// Enrich each group with user info + product info for product-type stories
	type storyWithProduct struct {
		models.Story
		Product interface{} `json:"product,omitempty"`
	}
	type feedGroup struct {
		UserID   string      `json:"user_id"`
		UserInfo interface{} `json:"user_info"`
		Stories  interface{} `json:"stories"`
	}

	feed := []feedGroup{}
	for _, key := range order {
		userObjID, _ := primitive.ObjectIDFromHex(key)

		// Fetch user info
		var u models.User
		h.db.MongoDB.Collection("users").FindOne(ctx, bson.M{"_id": userObjID}).Decode(&u)
		userInfo := gin.H{
			"id":       u.ID,
			"username": u.Username,
			"avatar":   u.Avatar,
		}

		// For product stories, attach product details
		enriched := []storyWithProduct{}
		for _, s := range grouped[key] {
			item := storyWithProduct{Story: s}
			if s.Type == "product" && s.ProductID != nil {
				var prod models.Product
				if err := h.db.MongoDB.Collection("products").FindOne(ctx, bson.M{"_id": *s.ProductID}).Decode(&prod); err == nil {
					item.Product = prod
				}
			}
			enriched = append(enriched, item)
		}

		feed = append(feed, feedGroup{
			UserID:   key,
			UserInfo: userInfo,
			Stories:  enriched,
		})
	}

	c.JSON(http.StatusOK, gin.H{"feed": feed})
}

// GET /stories/user/:user_id
// Returns all active stories for a specific user (with product details if applicable).
func (h *StoryHandler) GetUserStories(c *gin.Context) {
	ctx := context.Background()
	now := time.Now()

	targetUserIDStr := c.Param("user_id")
	targetUserID, err := primitive.ObjectIDFromHex(targetUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	findOpts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := h.db.MongoDB.Collection("stories").Find(
		ctx,
		bson.M{
			"user_id":    targetUserID,
			"expires_at": bson.M{"$gt": now},
		},
		findOpts,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stories"})
		return
	}
	defer cursor.Close(ctx)

	var stories []models.Story
	if err := cursor.All(ctx, &stories); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode stories"})
		return
	}

	// Attach product details for product stories
	type storyWithProduct struct {
		models.Story
		Product interface{} `json:"product,omitempty"`
	}
	enriched := []storyWithProduct{}
	for _, s := range stories {
		item := storyWithProduct{Story: s}
		if s.Type == "product" && s.ProductID != nil {
			var prod models.Product
			if err := h.db.MongoDB.Collection("products").FindOne(ctx, bson.M{"_id": *s.ProductID}).Decode(&prod); err == nil {
				item.Product = prod
			}
		}
		enriched = append(enriched, item)
	}

	c.JSON(http.StatusOK, gin.H{"stories": enriched})
}

// DELETE /stories/:story_id
// Deletes a story. Only the owner can delete.
func (h *StoryHandler) DeleteStory(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	storyIDStr := c.Param("story_id")
	storyID, err := primitive.ObjectIDFromHex(storyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid story_id"})
		return
	}

	result, err := h.db.MongoDB.Collection("stories").DeleteOne(
		context.Background(),
		bson.M{
			"_id":     storyID,
			"user_id": userIDObj, // enforce ownership
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete story"})
		return
	}
	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Story not found or you are not the owner"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Story deleted"})
}
