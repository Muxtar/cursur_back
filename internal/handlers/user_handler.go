package handlers

import (
	"context"
	"net/http"
	"time"

	"chat-backend/internal/database"
	"chat-backend/internal/models"
	"chat-backend/internal/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserHandler struct {
	db *database.Database
}

func NewUserHandler(db *database.Database) *UserHandler {
	return &UserHandler{db: db}
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var user models.User
	err := h.db.MongoDB.Collection("users").FindOne(
		context.Background(),
		bson.M{"_id": userIDObj},
	).Decode(&user)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Hide phone number if user wants it hidden
	if user.HidePhoneNumber {
		user.PhoneNumber = ""
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) SearchByUsername(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username required"})
		return
	}

	var user models.User
	err := h.db.MongoDB.Collection("users").FindOne(
		context.Background(),
		bson.M{"username": username},
	).Decode(&user)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Hide phone number if user wants it hidden
	if user.HidePhoneNumber {
		user.PhoneNumber = ""
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) GetDevices(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var user models.User
	err := h.db.MongoDB.Collection("users").FindOne(
		context.Background(),
		bson.M{"_id": userIDObj},
	).Decode(&user)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user.ActiveDevices)
}

func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updateData["updated_at"] = time.Now()
	_, err := h.db.MongoDB.Collection("users").UpdateOne(
		context.Background(),
		bson.M{"_id": userIDObj},
		bson.M{"$set": updateData},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

type LocationUpdate struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
	Address   string  `json:"address"`
}

func (h *UserHandler) UpdateLocation(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var loc LocationUpdate
	if err := c.ShouldBindJSON(&loc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	location := models.Location{
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
		Address:   loc.Address,
	}

	_, err := h.db.MongoDB.Collection("users").UpdateOne(
		context.Background(),
		bson.M{"_id": userIDObj},
		bson.M{"$set": bson.M{
			"location":  location,
			"updated_at": time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update location"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Location updated successfully"})
}

func (h *UserHandler) GetNearbyUsers(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	// Get current user's location
	var user models.User
	err := h.db.MongoDB.Collection("users").FindOne(
		context.Background(),
		bson.M{"_id": userIDObj},
	).Decode(&user)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Get radius from query (default 10km)
	radius := 10.0
	if r := c.Query("radius"); r != "" {
		// Parse radius if provided
	}

	// Get all users with locations
	cursor, err := h.db.MongoDB.Collection("users").Find(
		context.Background(),
		bson.M{
			"_id": bson.M{"$ne": userIDObj},
			"location": bson.M{"$exists": true},
		},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	defer cursor.Close(context.Background())

	var nearbyUsers []models.User
	for cursor.Next(context.Background()) {
		var u models.User
		if err := cursor.Decode(&u); err != nil {
			continue
		}

		distance := utils.CalculateDistance(
			user.Location.Latitude, user.Location.Longitude,
			u.Location.Latitude, u.Location.Longitude,
		)

		if distance <= radius {
			nearbyUsers = append(nearbyUsers, u)
		}
	}

	c.JSON(http.StatusOK, nearbyUsers)
}

