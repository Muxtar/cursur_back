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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SettingsHandler struct {
	db *database.Database
}

func NewSettingsHandler(db *database.Database) *SettingsHandler {
	return &SettingsHandler{db: db}
}

func (h *SettingsHandler) GetSettings(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var settings models.UserSettings
	err := h.db.MongoDB.Collection("user_settings").FindOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
	).Decode(&settings)

	if err == mongo.ErrNoDocuments {
		// Create default settings
		settings = models.UserSettings{
			ID:     primitive.NewObjectID(),
			UserID: userIDObj,
			Account: models.AccountSettings{
				AccountStatus: "active",
			},
			Privacy: models.PrivacySettings{
				LastSeen:      "everyone",
				OnlineStatus:  "everyone",
				ProfilePhoto:  "everyone",
				BioVisibility: "everyone",
				FindByPhone:   true,
				FindByUsername: true,
				SecretChatTTL: 0,
				EncryptionLevel: "standard",
			},
			Chat: models.ChatSettings{
				Theme:          "light",
				FontSize:       "medium",
				EmojiEnabled:   true,
				StickersEnabled: true,
				GIFEnabled:     true,
				MessagePreview: true,
				ReadReceipts:   true,
				AutoDownload: models.AutoDownloadSettings{
					Photos:    "wifi",
					Videos:    "wifi",
					Audio:     "wifi",
					Documents: "wifi",
				},
			},
			Notifications: models.NotificationSettings{
				DirectChats: true,
				GroupChats:  true,
				Calls:       true,
				Sound:       "default",
				Vibration:   "default",
			},
			Appearance: models.AppearanceSettings{
				Theme:    "system",
				FontSize: "medium",
				Animations: true,
			},
			Data: models.DataSettings{
				CloudSync: true,
			},
			Calls: models.CallSettings{
				Quality:       "medium",
				DataUsageMode: "medium",
				VideoCalls:    true,
				VoiceCalls:    true,
				WhoCanCall:    "everyone",
				CallHistory:   true,
			},
			Groups: models.GroupSettings{
				WhoCanCreate: "everyone",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_, err = h.db.MongoDB.Collection("user_settings").InsertOne(context.Background(), settings)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create settings"})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updateData["updated_at"] = time.Now()
	
	// Use upsert to create if doesn't exist
	opts := options.Update().SetUpsert(true)
	_, err := h.db.MongoDB.Collection("user_settings").UpdateOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
		bson.M{"$set": updateData},
		opts,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Settings updated successfully"})
}

func (h *SettingsHandler) UpdateAccountSettings(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var accountSettings models.AccountSettings
	if err := c.ShouldBindJSON(&accountSettings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.db.MongoDB.Collection("user_settings").UpdateOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
		bson.M{"$set": bson.M{
			"account":    accountSettings,
			"updated_at": time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update account settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account settings updated"})
}

func (h *SettingsHandler) UpdatePrivacySettings(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var privacySettings models.PrivacySettings
	if err := c.ShouldBindJSON(&privacySettings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.db.MongoDB.Collection("user_settings").UpdateOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
		bson.M{"$set": bson.M{
			"privacy":    privacySettings,
			"updated_at": time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update privacy settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Privacy settings updated"})
}

func (h *SettingsHandler) UpdateChatSettings(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var chatSettings models.ChatSettings
	if err := c.ShouldBindJSON(&chatSettings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.db.MongoDB.Collection("user_settings").UpdateOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
		bson.M{"$set": bson.M{
			"chat":       chatSettings,
			"updated_at": time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update chat settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chat settings updated"})
}

func (h *SettingsHandler) UpdateNotificationSettings(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var notificationSettings models.NotificationSettings
	if err := c.ShouldBindJSON(&notificationSettings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.db.MongoDB.Collection("user_settings").UpdateOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
		bson.M{"$set": bson.M{
			"notifications": notificationSettings,
			"updated_at":    time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notification settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification settings updated"})
}

func (h *SettingsHandler) UpdateAppearanceSettings(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var appearanceSettings models.AppearanceSettings
	if err := c.ShouldBindJSON(&appearanceSettings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.db.MongoDB.Collection("user_settings").UpdateOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
		bson.M{"$set": bson.M{
			"appearance": appearanceSettings,
			"updated_at": time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update appearance settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Appearance settings updated"})
}

func (h *SettingsHandler) UpdateDataSettings(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var dataSettings models.DataSettings
	if err := c.ShouldBindJSON(&dataSettings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.db.MongoDB.Collection("user_settings").UpdateOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
		bson.M{"$set": bson.M{
			"data":       dataSettings,
			"updated_at": time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update data settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data settings updated"})
}

func (h *SettingsHandler) GetSessions(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var settings models.UserSettings
	err := h.db.MongoDB.Collection("user_settings").FindOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
	).Decode(&settings)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Settings not found"})
		return
	}

	c.JSON(http.StatusOK, settings.Privacy.ActiveSessions)
}

func (h *SettingsHandler) TerminateSession(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)
	sessionID := c.Param("session_id")

	var settings models.UserSettings
	err := h.db.MongoDB.Collection("user_settings").FindOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
	).Decode(&settings)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Settings not found"})
		return
	}

	// Remove session from list
	var updatedSessions []models.Session
	for _, session := range settings.Privacy.ActiveSessions {
		if session.ID != sessionID {
			updatedSessions = append(updatedSessions, session)
		}
	}

	_, err = h.db.MongoDB.Collection("user_settings").UpdateOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
		bson.M{"$set": bson.M{
			"privacy.active_sessions": updatedSessions,
			"updated_at":              time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to terminate session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session terminated"})
}

func (h *SettingsHandler) BlockUser(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	blockedUserID, err := primitive.ObjectIDFromHex(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var settings models.UserSettings
	err = h.db.MongoDB.Collection("user_settings").FindOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
	).Decode(&settings)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Settings not found"})
		return
	}

	// Add to blocked list if not already blocked
	alreadyBlocked := false
	for _, id := range settings.Privacy.BlockedUsers {
		if id == blockedUserID {
			alreadyBlocked = true
			break
		}
	}

	if !alreadyBlocked {
		settings.Privacy.BlockedUsers = append(settings.Privacy.BlockedUsers, blockedUserID)
		_, err = h.db.MongoDB.Collection("user_settings").UpdateOne(
			context.Background(),
			bson.M{"user_id": userIDObj},
			bson.M{"$set": bson.M{
				"privacy.blocked_users": settings.Privacy.BlockedUsers,
				"updated_at":            time.Now(),
			}},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to block user"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "User blocked"})
}

func (h *SettingsHandler) UnblockUser(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)
	blockedUserIDStr := c.Param("user_id")

	blockedUserID, err := primitive.ObjectIDFromHex(blockedUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var settings models.UserSettings
	err = h.db.MongoDB.Collection("user_settings").FindOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
	).Decode(&settings)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Settings not found"})
		return
	}

	// Remove from blocked list
	var updatedBlocked []primitive.ObjectID
	for _, id := range settings.Privacy.BlockedUsers {
		if id != blockedUserID {
			updatedBlocked = append(updatedBlocked, id)
		}
	}

	_, err = h.db.MongoDB.Collection("user_settings").UpdateOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
		bson.M{"$set": bson.M{
			"privacy.blocked_users": updatedBlocked,
			"updated_at":            time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unblock user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User unblocked"})
}

func (h *SettingsHandler) GetBlockedUsers(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var settings models.UserSettings
	err := h.db.MongoDB.Collection("user_settings").FindOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
	).Decode(&settings)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Settings not found"})
		return
	}

	// Fetch user details for blocked users
	var blockedUsers []map[string]interface{}
	for _, blockedID := range settings.Privacy.BlockedUsers {
		var user models.User
		err := h.db.MongoDB.Collection("users").FindOne(
			context.Background(),
			bson.M{"_id": blockedID},
		).Decode(&user)

		if err == nil {
			blockedUsers = append(blockedUsers, map[string]interface{}{
				"id":          user.ID,
				"username":    user.Username,
				"phone_number": user.PhoneNumber,
			})
		}
	}

	c.JSON(http.StatusOK, blockedUsers)
}

func (h *SettingsHandler) UpdateCallSettings(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var callSettings models.CallSettings
	if err := c.ShouldBindJSON(&callSettings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.db.MongoDB.Collection("user_settings").UpdateOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
		bson.M{"$set": bson.M{
			"calls":      callSettings,
			"updated_at": time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update call settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Call settings updated"})
}

func (h *SettingsHandler) UpdateGroupSettings(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var groupSettings models.GroupSettings
	if err := c.ShouldBindJSON(&groupSettings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.db.MongoDB.Collection("user_settings").UpdateOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
		bson.M{"$set": bson.M{
			"groups":     groupSettings,
			"updated_at": time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group settings updated"})
}

func (h *SettingsHandler) UpdateAdvancedSettings(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var advancedSettings models.AdvancedSettings
	if err := c.ShouldBindJSON(&advancedSettings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.db.MongoDB.Collection("user_settings").UpdateOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
		bson.M{"$set": bson.M{
			"advanced":   advancedSettings,
			"updated_at": time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update advanced settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Advanced settings updated"})
}

func (h *SettingsHandler) SuspendAccount(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	_, err := h.db.MongoDB.Collection("user_settings").UpdateOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
		bson.M{"$set": bson.M{
			"account.account_status": "suspended",
			"updated_at":             time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to suspend account"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account suspended"})
}

func (h *SettingsHandler) DeleteAccount(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	_, err := h.db.MongoDB.Collection("user_settings").UpdateOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
		bson.M{"$set": bson.M{
			"account.account_status": "deleted",
			"updated_at":             time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete account"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account deleted"})
}

func (h *SettingsHandler) ClearCache(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	// Clear MongoDB cache collections for user
	ctx := context.Background()
	
	// Delete QR code cache entries
	_, _ = h.db.MongoDB.Collection("qr_code_cache").DeleteMany(ctx, bson.M{"user_id": userIDObj.Hex()})
	
	// Delete typing indicators
	_, _ = h.db.MongoDB.Collection("typing_indicators").DeleteMany(ctx, bson.M{"user_id": userIDObj})
	
	// Note: Verification codes are already auto-expired based on expires_at field

	c.JSON(http.StatusOK, gin.H{"message": "Cache cleared"})
}

func (h *SettingsHandler) GetDataUsage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var settings models.UserSettings
	err := h.db.MongoDB.Collection("user_settings").FindOne(
		context.Background(),
		bson.M{"user_id": userIDObj},
	).Decode(&settings)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Settings not found"})
		return
	}

	c.JSON(http.StatusOK, settings.Data.DataUsage)
}





