package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"chat-backend/internal/database"
	"chat-backend/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProductHandler struct {
	db *database.Database
}

func NewProductHandler(db *database.Database) *ProductHandler {
	return &ProductHandler{db: db}
}

// CreateProduct creates a new product
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	var product models.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set owner and timestamps
	product.OwnerID = userIDObj
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()
	product.LikeCount = 0
	product.CommentCount = 0
	product.ViewCount = 0
	product.IsVerified = false
	if product.Privacy == "" {
		product.Privacy = "public"
	}

	// Generate product ID if not provided
	if product.ProductID == "" {
		product.ProductID = primitive.NewObjectID().Hex()
	}

	result, err := h.db.MongoDB.Collection("products").InsertOne(context.Background(), product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
		return
	}

	product.ID = result.InsertedID.(primitive.ObjectID)

	// Populate owner info
	var owner models.User
	h.db.MongoDB.Collection("users").FindOne(context.Background(), bson.M{"_id": userIDObj}).Decode(&owner)
	
	response := gin.H{
		"product": product,
		"owner": gin.H{
			"id":       owner.ID,
			"username": owner.Username,
			"avatar":   owner.Avatar,
		},
	}

	c.JSON(http.StatusCreated, response)
}

// GetProducts fetches products feed (paginated)
func (h *ProductHandler) GetProducts(c *gin.Context) {
	userID, exists := c.Get("user_id")
	var userIDObj primitive.ObjectID
	if exists {
		userIDObj = userID.(primitive.ObjectID)
	}

	// Pagination
	page := 1
	limit := 20
	if p := c.Query("page"); p != "" {
		if parsedPage, err := strconv.Atoi(p); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	skip := (page - 1) * limit

	// Filter options
	filter := bson.M{"privacy": "public"}
	if category := c.Query("category"); category != "" {
		filter["category"] = category
	}
	if ownerID := c.Query("owner_id"); ownerID != "" {
		ownerObjID, _ := primitive.ObjectIDFromHex(ownerID)
		filter["owner_id"] = ownerObjID
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetSkip(int64(skip)).SetLimit(int64(limit))
	cursor, err := h.db.MongoDB.Collection("products").Find(context.Background(), filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}
	defer cursor.Close(context.Background())

	var products []models.Product
	if err := cursor.All(context.Background(), &products); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode products"})
		return
	}

	// Populate owner info and check if user liked each product
	type ProductWithOwner struct {
		models.Product
		Owner     gin.H `json:"owner"`
		IsLiked   bool  `json:"is_liked"`
	}

	result := make([]ProductWithOwner, 0, len(products))
	for _, product := range products {
		var owner models.User
		h.db.MongoDB.Collection("users").FindOne(context.Background(), bson.M{"_id": product.OwnerID}).Decode(&owner)

		isLiked := false
		if exists {
			var like models.Like
			err := h.db.MongoDB.Collection("likes").FindOne(
				context.Background(),
				bson.M{"product_id": product.ID, "user_id": userIDObj},
			).Decode(&like)
			isLiked = err == nil
		}

		result = append(result, ProductWithOwner{
			Product: product,
			Owner: gin.H{
				"id":       owner.ID,
				"username": owner.Username,
				"avatar":   owner.Avatar,
			},
			IsLiked: isLiked,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"products": result,
		"page":     page,
		"limit":    limit,
	})
}

// GetProduct gets a single product by ID
func (h *ProductHandler) GetProduct(c *gin.Context) {
	productID := c.Param("product_id")
	productObjID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var product models.Product
	err = h.db.MongoDB.Collection("products").FindOne(
		context.Background(),
		bson.M{"_id": productObjID},
	).Decode(&product)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Increment view count
	h.db.MongoDB.Collection("products").UpdateOne(
		context.Background(),
		bson.M{"_id": productObjID},
		bson.M{"$inc": bson.M{"view_count": 1}},
	)

	// Get owner
	var owner models.User
	h.db.MongoDB.Collection("users").FindOne(context.Background(), bson.M{"_id": product.OwnerID}).Decode(&owner)

	// Check if user liked
	userID, exists := c.Get("user_id")
	isLiked := false
	if exists {
		userIDObj := userID.(primitive.ObjectID)
		var like models.Like
		err := h.db.MongoDB.Collection("likes").FindOne(
			context.Background(),
			bson.M{"product_id": productObjID, "user_id": userIDObj},
		).Decode(&like)
		isLiked = err == nil
	}

	c.JSON(http.StatusOK, gin.H{
		"product": product,
		"owner": gin.H{
			"id":       owner.ID,
			"username": owner.Username,
			"avatar":   owner.Avatar,
			"bio":      owner.Bio,
		},
		"is_liked": isLiked,
	})
}

// UpdateProduct updates a product (only owner can update)
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	productID := c.Param("product_id")
	productObjID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// Check ownership
	var product models.Product
	err = h.db.MongoDB.Collection("products").FindOne(
		context.Background(),
		bson.M{"_id": productObjID, "owner_id": userIDObj},
	).Decode(&product)

	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't own this product"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updateData["updated_at"] = time.Now()
	_, err = h.db.MongoDB.Collection("products").UpdateOne(
		context.Background(),
		bson.M{"_id": productObjID},
		bson.M{"$set": updateData},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product updated successfully"})
}

// DeleteProduct deletes a product (only owner can delete)
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDObj := userID.(primitive.ObjectID)

	productID := c.Param("product_id")
	productObjID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// Check ownership
	var product models.Product
	err = h.db.MongoDB.Collection("products").FindOne(
		context.Background(),
		bson.M{"_id": productObjID, "owner_id": userIDObj},
	).Decode(&product)

	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't own this product"})
		return
	}

	// Delete product and related data
	_, err = h.db.MongoDB.Collection("products").DeleteOne(
		context.Background(),
		bson.M{"_id": productObjID},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product"})
		return
	}

	// Delete related comments and likes
	h.db.MongoDB.Collection("comments").DeleteMany(context.Background(), bson.M{"product_id": productObjID})
	h.db.MongoDB.Collection("likes").DeleteMany(context.Background(), bson.M{"product_id": productObjID})

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}

// GetUserProducts gets all products owned by a user
func (h *ProductHandler) GetUserProducts(c *gin.Context) {
	userIDParam := c.Param("user_id")
	userObjID, err := primitive.ObjectIDFromHex(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	cursor, err := h.db.MongoDB.Collection("products").Find(
		context.Background(),
		bson.M{"owner_id": userObjID},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}
	defer cursor.Close(context.Background())

	var products []models.Product
	if err := cursor.All(context.Background(), &products); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode products"})
		return
	}

	c.JSON(http.StatusOK, products)
}

