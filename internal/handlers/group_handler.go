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

type GroupHandler struct {
	db *database.Database
}

func NewGroupHandler(db *database.Database) *GroupHandler {
	return &GroupHandler{db: db}
}

type CreateGroupRequest struct {
	GroupName string   `json:"group_name" binding:"required"`
	GroupIcon string   `json:"group_icon,omitempty"`
	MemberIDs []string `json:"member_ids"`
}

func (h *GroupHandler) CreateGroup(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	members := []primitive.ObjectID{userIDObj}
	for _, memberIDStr := range req.MemberIDs {
		memberID, err := primitive.ObjectIDFromHex(memberIDStr)
		if err != nil {
			continue
		}
		members = append(members, memberID)
	}

	chat := models.Chat{
		ID:        primitive.NewObjectID(),
		Type:      "group",
		Members:   members,
		Admins: []models.AdminRole{
			{
				UserID:      userIDObj,
				Role:        "owner",
				Permissions: []string{"all"},
				GrantedAt:   time.Now(),
				GrantedBy:   userIDObj,
			},
		},
		GroupName: req.GroupName,
		GroupIcon: req.GroupIcon,
		MaxMembers: 200000, // Telegram limit
		Statistics: models.ChatStatistics{
			LastCalculated: time.Now(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := h.db.MongoDB.Collection("chats").InsertOne(context.Background(), chat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group"})
		return
	}

	c.JSON(http.StatusCreated, chat)
}

func (h *GroupHandler) GetGroups(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	cursor, err := h.db.MongoDB.Collection("chats").Find(
		context.Background(),
		bson.M{
			"type":    "group",
			"members": userIDObj,
		},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
		return
	}
	defer cursor.Close(context.Background())

	var groups []models.Chat
	if err := cursor.All(context.Background(), &groups); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode groups"})
		return
	}

	c.JSON(http.StatusOK, groups)
}

func (h *GroupHandler) GetGroup(c *gin.Context) {
	groupIDStr := c.Param("group_id")
	groupID, err := primitive.ObjectIDFromHex(groupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	var group models.Chat
	err = h.db.MongoDB.Collection("chats").FindOne(
		context.Background(),
		bson.M{"_id": groupID, "type": "group"},
	).Decode(&group)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (h *GroupHandler) UpdateGroup(c *gin.Context) {
	groupIDStr := c.Param("group_id")
	groupID, err := primitive.ObjectIDFromHex(groupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updateData["updated_at"] = time.Now()
	_, err = h.db.MongoDB.Collection("chats").UpdateOne(
		context.Background(),
		bson.M{"_id": groupID, "type": "group"},
		bson.M{"$set": updateData},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group updated successfully"})
}

func (h *GroupHandler) DeleteGroup(c *gin.Context) {
	groupIDStr := c.Param("group_id")
	groupID, err := primitive.ObjectIDFromHex(groupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	_, err = h.db.MongoDB.Collection("chats").DeleteOne(
		context.Background(),
		bson.M{"_id": groupID, "type": "group"},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group deleted successfully"})
}

func (h *GroupHandler) AddMember(c *gin.Context) {
	groupIDStr := c.Param("group_id")
	groupID, err := primitive.ObjectIDFromHex(groupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	var req struct {
		MemberID string `json:"member_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	memberID, err := primitive.ObjectIDFromHex(req.MemberID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}

	_, err = h.db.MongoDB.Collection("chats").UpdateOne(
		context.Background(),
		bson.M{"_id": groupID},
		bson.M{"$addToSet": bson.M{"members": memberID}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member added successfully"})
}

func (h *GroupHandler) RemoveMember(c *gin.Context) {
	groupIDStr := c.Param("group_id")
	groupID, err := primitive.ObjectIDFromHex(groupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	memberIDStr := c.Param("member_id")
	memberID, err := primitive.ObjectIDFromHex(memberIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}

	_, err = h.db.MongoDB.Collection("chats").UpdateOne(
		context.Background(),
		bson.M{"_id": groupID},
		bson.M{"$pull": bson.M{"members": memberID}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed successfully"})
}

func (h *GroupHandler) GetStatistics(c *gin.Context) {
	groupIDStr := c.Param("group_id")
	groupID, err := primitive.ObjectIDFromHex(groupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	var group models.Chat
	err = h.db.MongoDB.Collection("chats").FindOne(
		context.Background(),
		bson.M{"_id": groupID, "type": "group"},
	).Decode(&group)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	// Calculate statistics
	messageCount, _ := h.db.MongoDB.Collection("messages").CountDocuments(
		context.Background(),
		bson.M{"chat_id": groupID},
	)

	mediaCount, _ := h.db.MongoDB.Collection("messages").CountDocuments(
		context.Background(),
		bson.M{
			"chat_id": groupID,
			"message_type": bson.M{"$in": []string{"image", "video", "audio", "file"}},
		},
	)

	// Update statistics
	stats := models.ChatStatistics{
		TotalMessages:  messageCount,
		TotalMedia:     mediaCount,
		ActiveMembers:  len(group.Members),
		LastCalculated: time.Now(),
	}

	_, err = h.db.MongoDB.Collection("chats").UpdateOne(
		context.Background(),
		bson.M{"_id": groupID},
		bson.M{"$set": bson.M{"statistics": stats}},
	)

	c.JSON(http.StatusOK, stats)
}

