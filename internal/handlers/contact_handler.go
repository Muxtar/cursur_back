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

type ContactHandler struct {
	db *database.Database
}

func NewContactHandler(db *database.Database) *ContactHandler {
	return &ContactHandler{db: db}
}

type ScanQRRequest struct {
	QRData string `json:"qr_data" binding:"required"`
}

func (h *ContactHandler) GetContacts(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	cursor, err := h.db.MongoDB.Collection("contacts").Find(
		context.Background(),
		bson.M{"user_id": userIDObj},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contacts"})
		return
	}
	defer cursor.Close(context.Background())

	var contacts []models.Contact
	if err := cursor.All(context.Background(), &contacts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode contacts"})
		return
	}

	// Fetch user details for each contact
	var contactDetails []map[string]interface{}
	for _, contact := range contacts {
		var user models.User
		err := h.db.MongoDB.Collection("users").FindOne(
			context.Background(),
			bson.M{"_id": contact.ContactID},
		).Decode(&user)

		if err == nil {
			contactDetails = append(contactDetails, map[string]interface{}{
				"contact": contact,
				"user":    user,
			})
		}
	}

	c.JSON(http.StatusOK, contactDetails)
}

func (h *ContactHandler) ScanQRCode(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var req ScanQRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse QR code
	contactUserIDStr, err := utils.ParseQRCode(req.QRData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid QR code"})
		return
	}

	contactUserID, err := primitive.ObjectIDFromHex(contactUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID in QR code"})
		return
	}

	// Check if contact already exists
	var existingContact models.Contact
	err = h.db.MongoDB.Collection("contacts").FindOne(
		context.Background(),
		bson.M{
			"user_id":    userIDObj,
			"contact_id": contactUserID,
		},
	).Decode(&existingContact)

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Contact already exists"})
		return
	}

	// Create contact
	contact := models.Contact{
		ID:          primitive.NewObjectID(),
		UserID:      userIDObj,
		ContactID:   contactUserID,
		ContactQR:   req.QRData,
		IsAnonymous: false,
		CreatedAt:   time.Now(),
	}

	_, err = h.db.MongoDB.Collection("contacts").InsertOne(context.Background(), contact)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add contact"})
		return
	}

	// Also create reverse contact (bidirectional)
	reverseContact := models.Contact{
		ID:          primitive.NewObjectID(),
		UserID:      contactUserID,
		ContactID:   userIDObj,
		ContactQR:   "", // Will be generated if needed
		IsAnonymous: false,
		CreatedAt:   time.Now(),
	}
	h.db.MongoDB.Collection("contacts").InsertOne(context.Background(), reverseContact)

	c.JSON(http.StatusCreated, contact)
}

func (h *ContactHandler) DeleteContact(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	contactIDStr := c.Param("contact_id")
	contactID, err := primitive.ObjectIDFromHex(contactIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contact ID"})
		return
	}

	result, err := h.db.MongoDB.Collection("contacts").DeleteOne(
		context.Background(),
		bson.M{
			"user_id":    userIDObj,
			"contact_id": contactID,
		},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete contact"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contact deleted successfully"})
}





