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

type ProposalHandler struct {
	db *database.Database
}

func NewProposalHandler(db *database.Database) *ProposalHandler {
	return &ProposalHandler{db: db}
}

type CreateProposalRequest struct {
	ReceiverID string `json:"receiver_id" binding:"required"`
	Title      string `json:"title" binding:"required"`
	Content    string `json:"content" binding:"required"`
}

func (h *ProposalHandler) CreateProposal(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var req CreateProposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	receiverID, err := primitive.ObjectIDFromHex(req.ReceiverID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid receiver ID"})
		return
	}

	proposal := models.Proposal{
		ID:         primitive.NewObjectID(),
		SenderID:   userIDObj,
		ReceiverID: receiverID,
		Title:      req.Title,
		Content:    req.Content,
		Status:     "pending",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	_, err = h.db.MongoDB.Collection("proposals").InsertOne(context.Background(), proposal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create proposal"})
		return
	}

	c.JSON(http.StatusCreated, proposal)
}

func (h *ProposalHandler) GetProposals(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	// Get proposals where user is sender or receiver
	cursor, err := h.db.MongoDB.Collection("proposals").Find(
		context.Background(),
		bson.M{
			"$or": []bson.M{
				{"sender_id": userIDObj},
				{"receiver_id": userIDObj},
			},
		},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch proposals"})
		return
	}
	defer cursor.Close(context.Background())

	var proposals []models.Proposal
	if err := cursor.All(context.Background(), &proposals); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode proposals"})
		return
	}

	c.JSON(http.StatusOK, proposals)
}

func (h *ProposalHandler) AcceptProposal(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	proposalIDStr := c.Param("proposal_id")
	proposalID, err := primitive.ObjectIDFromHex(proposalIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid proposal ID"})
		return
	}

	// Verify user is the receiver
	var proposal models.Proposal
	err = h.db.MongoDB.Collection("proposals").FindOne(
		context.Background(),
		bson.M{"_id": proposalID, "receiver_id": userIDObj},
	).Decode(&proposal)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Proposal not found"})
		return
	}

	_, err = h.db.MongoDB.Collection("proposals").UpdateOne(
		context.Background(),
		bson.M{"_id": proposalID},
		bson.M{"$set": bson.M{
			"status":     "accepted",
			"updated_at": time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to accept proposal"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Proposal accepted"})
}

func (h *ProposalHandler) RejectProposal(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	proposalIDStr := c.Param("proposal_id")
	proposalID, err := primitive.ObjectIDFromHex(proposalIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid proposal ID"})
		return
	}

	// Verify user is the receiver
	var proposal models.Proposal
	err = h.db.MongoDB.Collection("proposals").FindOne(
		context.Background(),
		bson.M{"_id": proposalID, "receiver_id": userIDObj},
	).Decode(&proposal)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Proposal not found"})
		return
	}

	_, err = h.db.MongoDB.Collection("proposals").UpdateOne(
		context.Background(),
		bson.M{"_id": proposalID},
		bson.M{"$set": bson.M{
			"status":     "rejected",
			"updated_at": time.Now(),
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject proposal"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Proposal rejected"})
}





