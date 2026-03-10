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

type CompanyHandler struct {
	db *database.Database
}

func NewCompanyHandler(db *database.Database) *CompanyHandler {
	return &CompanyHandler{db: db}
}

// isValidCategory checks whether the given category is in the allowed list
func isValidCategory(cat string) bool {
	for _, c := range models.CompanyCategories {
		if c == cat {
			return true
		}
	}
	return false
}

// CreateCompany — POST /companies
func (h *CompanyHandler) CreateCompany(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var req struct {
		Name        string `json:"name" binding:"required"`
		Category    string `json:"category" binding:"required"`
		Description string `json:"description"`
		Website     string `json:"website"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !isValidCategory(req.Category) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category"})
		return
	}

	now := time.Now()
	company := models.Company{
		ID:          primitive.NewObjectID(),
		UserID:      userIDObj,
		Name:        req.Name,
		Category:    req.Category,
		Description: req.Description,
		Website:     req.Website,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if _, err := h.db.MongoDB.Collection("companies").InsertOne(context.Background(), company); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create company"})
		return
	}

	c.JSON(http.StatusCreated, company)
}

// GetMyCompanies — GET /companies/me
func (h *CompanyHandler) GetMyCompanies(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := h.db.MongoDB.Collection("companies").Find(
		context.Background(),
		bson.M{"user_id": userIDObj},
		opts,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch companies"})
		return
	}
	defer cursor.Close(context.Background())

	var companies []models.Company
	if err := cursor.All(context.Background(), &companies); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode companies"})
		return
	}
	if companies == nil {
		companies = []models.Company{}
	}

	c.JSON(http.StatusOK, companies)
}

// GetUserCompanies — GET /companies/user/:user_id  (public — any authenticated user can view)
func (h *CompanyHandler) GetUserCompanies(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userIDObj, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := h.db.MongoDB.Collection("companies").Find(
		context.Background(),
		bson.M{"user_id": userIDObj},
		opts,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch companies"})
		return
	}
	defer cursor.Close(context.Background())

	var companies []models.Company
	if err := cursor.All(context.Background(), &companies); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode companies"})
		return
	}
	if companies == nil {
		companies = []models.Company{}
	}

	c.JSON(http.StatusOK, companies)
}

// UpdateCompany — PUT /companies/:company_id
func (h *CompanyHandler) UpdateCompany(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	companyIDStr := c.Param("company_id")
	companyIDObj, err := primitive.ObjectIDFromHex(companyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid company ID"})
		return
	}

	var req struct {
		Name        string `json:"name"`
		Category    string `json:"category"`
		Description string `json:"description"`
		Website     string `json:"website"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Category != "" && !isValidCategory(req.Category) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category"})
		return
	}

	// Only owner can update
	existing := models.Company{}
	if err := h.db.MongoDB.Collection("companies").FindOne(
		context.Background(),
		bson.M{"_id": companyIDObj},
	).Decode(&existing); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Company not found"})
		return
	}
	if existing.UserID != userIDObj {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed"})
		return
	}

	update := bson.M{"updated_at": time.Now()}
	if req.Name != "" {
		update["name"] = req.Name
	}
	if req.Category != "" {
		update["category"] = req.Category
	}
	update["description"] = req.Description
	update["website"] = req.Website

	if _, err := h.db.MongoDB.Collection("companies").UpdateOne(
		context.Background(),
		bson.M{"_id": companyIDObj},
		bson.M{"$set": update},
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update company"})
		return
	}

	// Return updated document
	updated := models.Company{}
	h.db.MongoDB.Collection("companies").FindOne(context.Background(), bson.M{"_id": companyIDObj}).Decode(&updated)
	c.JSON(http.StatusOK, updated)
}

// DeleteCompany — DELETE /companies/:company_id
func (h *CompanyHandler) DeleteCompany(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	companyIDStr := c.Param("company_id")
	companyIDObj, err := primitive.ObjectIDFromHex(companyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid company ID"})
		return
	}

	// Only owner can delete
	existing := models.Company{}
	if err := h.db.MongoDB.Collection("companies").FindOne(
		context.Background(),
		bson.M{"_id": companyIDObj},
	).Decode(&existing); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Company not found"})
		return
	}
	if existing.UserID != userIDObj {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed"})
		return
	}

	if _, err := h.db.MongoDB.Collection("companies").DeleteOne(
		context.Background(),
		bson.M{"_id": companyIDObj},
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete company"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Company deleted"})
}

// GetCategories — GET /companies/categories  (public — returns allowed category list)
func (h *CompanyHandler) GetCategories(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"categories": models.CompanyCategories})
}
