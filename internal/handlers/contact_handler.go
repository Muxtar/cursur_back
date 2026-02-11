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

type AddContactRequest struct {
	UserID      string `json:"user_id"`      // optional when phone_number+display_name given
	PhoneNumber string `json:"phone_number"` // add by number
	DisplayName string `json:"display_name"` // name for the contact
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

	// Fetch user details for each contact (or use phone_number/display_name for external contacts)
	var contactDetails []map[string]interface{}
	for _, contact := range contacts {
		if contact.ContactID != nil {
			var user models.User
			err := h.db.MongoDB.Collection("users").FindOne(
				context.Background(),
				bson.M{"_id": *contact.ContactID},
			).Decode(&user)
			if err == nil {
				contactDetails = append(contactDetails, map[string]interface{}{
					"contact": contact,
					"user":    user,
				})
			}
		} else {
			// External contact (added by phone + name only)
			contactDetails = append(contactDetails, map[string]interface{}{
				"contact": contact,
				"user": map[string]interface{}{
					"id":           nil,
					"phone_number": contact.PhoneNumber,
					"username":     contact.DisplayName,
					"display_name": contact.DisplayName,
				},
			})
		}
	}

	c.JSON(http.StatusOK, contactDetails)
}

// AddContact adds a contact by user_id OR by phone_number + display_name.
func (h *ContactHandler) AddContact(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var req AddContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Add by phone_number + display_name
	if req.PhoneNumber != "" {
		phone := normalizePhone(req.PhoneNumber)
		if phone == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phone number"})
			return
		}
		// Check if already in contacts (by phone or by user)
		var existing models.Contact
		err := h.db.MongoDB.Collection("contacts").FindOne(
			context.Background(),
			bson.M{"user_id": userIDObj, "phone_number": phone},
		).Decode(&existing)
		if err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Contact already exists"})
			return
		}
		var contactUser models.User
		err = h.db.MongoDB.Collection("users").FindOne(
			context.Background(),
			bson.M{"phone_number": phone},
		).Decode(&contactUser)
		displayName := req.DisplayName
		if displayName == "" {
			displayName = phone
		}
		if err != nil {
			// User not registered: add as external contact (no ContactID)
			contact := models.Contact{
				ID:          primitive.NewObjectID(),
				UserID:      userIDObj,
				ContactID:   nil,
				PhoneNumber: phone,
				DisplayName: displayName,
				IsAnonymous: false,
				CreatedAt:   time.Now(),
			}
			if _, err := h.db.MongoDB.Collection("contacts").InsertOne(context.Background(), contact); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add contact"})
				return
			}
			c.JSON(http.StatusCreated, contact)
			return
		}
		// User registered: add with ContactID
		if contactUser.ID == userIDObj {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot add yourself as contact"})
			return
		}
		err = h.db.MongoDB.Collection("contacts").FindOne(
			context.Background(),
			bson.M{"user_id": userIDObj, "contact_id": contactUser.ID},
		).Decode(&existing)
		if err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Contact already exists"})
			return
		}
		contact := models.Contact{
			ID:          primitive.NewObjectID(),
			UserID:      userIDObj,
			ContactID:   &contactUser.ID,
			ContactQR:   contactUser.QRCode,
			DisplayName: displayName,
			IsAnonymous: false,
			CreatedAt:   time.Now(),
		}
		if _, err := h.db.MongoDB.Collection("contacts").InsertOne(context.Background(), contact); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add contact"})
			return
		}
		reverseContact := models.Contact{
			ID:          primitive.NewObjectID(),
			UserID:      contactUser.ID,
			ContactID:   &userIDObj,
			IsAnonymous: false,
			CreatedAt:   time.Now(),
		}
		h.db.MongoDB.Collection("contacts").InsertOne(context.Background(), reverseContact)
		c.JSON(http.StatusCreated, contact)
		return
	}

	// Add by user_id (existing flow)
	if req.UserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provide user_id or phone_number + display_name"})
		return
	}
	contactUserID, err := primitive.ObjectIDFromHex(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	if contactUserID == userIDObj {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot add yourself as contact"})
		return
	}
	var contactUser models.User
	if err := h.db.MongoDB.Collection("users").FindOne(
		context.Background(),
		bson.M{"_id": contactUserID},
	).Decode(&contactUser); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	var existingContact models.Contact
	err = h.db.MongoDB.Collection("contacts").FindOne(
		context.Background(),
		bson.M{"user_id": userIDObj, "contact_id": contactUserID},
	).Decode(&existingContact)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Contact already exists"})
		return
	}
	contact := models.Contact{
		ID:          primitive.NewObjectID(),
		UserID:      userIDObj,
		ContactID:   &contactUserID,
		ContactQR:   contactUser.QRCode,
		IsAnonymous: false,
		CreatedAt:   time.Now(),
	}
	if _, err := h.db.MongoDB.Collection("contacts").InsertOne(context.Background(), contact); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add contact"})
		return
	}
	reverseContact := models.Contact{
		ID:          primitive.NewObjectID(),
		UserID:      contactUserID,
		ContactID:   &userIDObj,
		IsAnonymous: false,
		CreatedAt:   time.Now(),
	}
	h.db.MongoDB.Collection("contacts").InsertOne(context.Background(), reverseContact)
	c.JSON(http.StatusCreated, contact)
}

func normalizePhone(s string) string {
	var out []rune
	for _, r := range s {
		if r >= '0' && r <= '9' || r == '+' {
			out = append(out, r)
		}
	}
	return string(out)
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

	contactUserIDPtr := &contactUserID
	contact := models.Contact{
		ID:          primitive.NewObjectID(),
		UserID:      userIDObj,
		ContactID:   contactUserIDPtr,
		ContactQR:   req.QRData,
		IsAnonymous: false,
		CreatedAt:   time.Now(),
	}

	_, err = h.db.MongoDB.Collection("contacts").InsertOne(context.Background(), contact)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add contact"})
		return
	}

	userIDObjPtr := &userIDObj
	reverseContact := models.Contact{
		ID:          primitive.NewObjectID(),
		UserID:      contactUserID,
		ContactID:   userIDObjPtr,
		ContactQR:   "",
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

	// Try delete by contact record _id first (for external contacts or when frontend sends contact.id)
	result, err := h.db.MongoDB.Collection("contacts").DeleteOne(
		context.Background(),
		bson.M{"_id": contactID, "user_id": userIDObj},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete contact"})
		return
	}
	if result.DeletedCount == 0 {
		// Else treat as "other user" id (contact_id)
		result, err = h.db.MongoDB.Collection("contacts").DeleteOne(
			context.Background(),
			bson.M{"user_id": userIDObj, "contact_id": contactID},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete contact"})
			return
		}
	}
	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Contact deleted successfully"})
}
