package handlers

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"chat-backend/internal/database"
	"chat-backend/internal/models"
	"chat-backend/internal/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db      *database.Database
	twilio  *utils.TwilioService
}

func NewAuthHandler(db *database.Database, twilio *utils.TwilioService) *AuthHandler {
	return &AuthHandler{
		db:     db,
		twilio: twilio,
	}
}

type RegisterRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	UserType    string `json:"user_type"` // "normal" or "company"
	CompanyName string `json:"company_name,omitempty"`
	CompanyCategory string `json:"company_category,omitempty"`
}

type LoginRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Password    string `json:"password"`
}

type SendCodeRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
}

type VerifyCodeRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Code        string `json:"code" binding:"required"`
}

type RegisterWithCodeRequest struct {
	PhoneNumber     string `json:"phone_number" binding:"required"`
	Code            string `json:"code" binding:"required"`
	Username        string `json:"username"`
	UserType        string `json:"user_type"` // "normal" or "company"
	CompanyName     string `json:"company_name,omitempty"`
	CompanyCategory string `json:"company_category,omitempty"`
}

func (h *AuthHandler) storeVerificationCode(ctx context.Context, phone, code string, ttl time.Duration) error {
	if h.db.Postgres == nil {
		return fmt.Errorf("PostgreSQL database not available")
	}

	expiresAt := time.Now().Add(ttl)
	verificationCode := models.VerificationCode{
		PhoneNumber: phone,
		Code:        code,
		ExpiresAt:   expiresAt,
	}

	// Delete any existing codes for this phone number first
	deleteErr := h.db.Postgres.Where("phone_number = ?", phone).Delete(&models.VerificationCode{}).Error
	if deleteErr != nil {
		fmt.Printf("Warning: Failed to delete existing verification codes: %v\n", deleteErr)
		// Continue anyway - we'll try to insert the new code
	}

	// Insert new code
	err := h.db.Postgres.Create(&verificationCode).Error
	if err != nil {
		fmt.Printf("Error storing verification code: %v\n", err)
		return fmt.Errorf("failed to store verification code: %w", err)
	}

	return nil
}

func (h *AuthHandler) consumeVerificationCode(ctx context.Context, phone, code string) (bool, error) {
	if h.db.Postgres == nil {
		return false, fmt.Errorf("PostgreSQL database not available")
	}

	// Find matching code that hasn't expired
	var verificationCode models.VerificationCode
	err := h.db.Postgres.Where("phone_number = ? AND code = ? AND expires_at > ?", phone, code, time.Now()).
		First(&verificationCode).Error

	if err == gorm.ErrRecordNotFound {
		return false, nil // invalid/expired
	}
	if err != nil {
		// Log the error for debugging
		fmt.Printf("Error consuming verification code: %v\n", err)
		return false, fmt.Errorf("failed to verify code: %w", err)
	}

	// Consume: delete all codes for that phone (prevent reuse)
	deleteErr := h.db.Postgres.Where("phone_number = ?", phone).Delete(&models.VerificationCode{}).Error
	if deleteErr != nil {
		// Log but don't fail - code was already verified
		fmt.Printf("Warning: Failed to delete verification code after verification: %v\n", deleteErr)
	}

	return true, nil
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user exists
	var existingUser models.User
	err := h.db.MongoDB.Collection("users").FindOne(
		context.Background(),
		bson.M{"phone_number": req.PhoneNumber},
	).Decode(&existingUser)

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		return
	}

	// Hash password if provided (for future use)
	// Note: Password field is not yet in User model
	if req.Password != "" {
		_, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		// TODO: Add password field to User model and store hashedPassword
	}

	// Generate QR code
	userID := primitive.NewObjectID()
	qrData, qrBase64, err := utils.GenerateQRCode(userID.Hex())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate QR code"})
		return
	}

	// Create user
	userType := req.UserType
	if userType == "" {
		userType = "normal"
	}
	user := models.User{
		ID:              userID,
		PhoneNumber:     req.PhoneNumber,
		QRCode:          qrBase64,
		Username:        req.Username,
		UserType:        userType,
		CompanyName:     req.CompanyName,
		CompanyCategory: req.CompanyCategory,
		IsAnonymous:     false,
		AccountStatus:   "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		LastActive:      time.Now(),
	}

	_, err = h.db.MongoDB.Collection("users").InsertOne(context.Background(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Store QR data in PostgreSQL for quick lookup
	if h.db.Postgres != nil {
		qrCache := models.QRCodeCache{
			QRData: qrData,
			UserID: userID.Hex(),
		}
		// Delete existing entry if any
		h.db.Postgres.Where("qr_data = ?", qrData).Delete(&models.QRCodeCache{})
		// Insert new entry
		h.db.Postgres.Create(&qrCache)
	}

	// Generate token
	token, err := utils.GenerateToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"token": token,
		"user":  user,
		"qr":    qrBase64,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	err := h.db.MongoDB.Collection("users").FindOne(
		context.Background(),
		bson.M{"phone_number": req.PhoneNumber},
	).Decode(&user)

	if err == mongo.ErrNoDocuments {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// In a real app, verify password here
	// For now, we'll just check if user exists

	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
}

func (h *AuthHandler) GetQRCode(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	err = h.db.MongoDB.Collection("users").FindOne(
		context.Background(),
		bson.M{"_id": userID},
	).Decode(&user)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"qr_code": user.QRCode})
}

// SendCode sends a verification code to the phone number
func (h *AuthHandler) SendCode(c *gin.Context) {
	var req SendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate phone number
	if req.PhoneNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phone number is required"})
		return
	}

	// Basic phone number validation (should start with + and have at least 10 digits)
	if len(req.PhoneNumber) < 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phone number format"})
		return
	}

	// Generate 6-digit code
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate code"})
		return
	}
	code := fmt.Sprintf("%06d", n.Int64())

	// Store code with 5 minute expiration in PostgreSQL
	if err := h.storeVerificationCode(context.Background(), req.PhoneNumber, code, 5*time.Minute); err != nil {
		// If storage fails, we cannot safely verify later
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store verification code"})
		return
	}

	// Send SMS via Twilio
	twilioSent := false
	if h.twilio != nil && h.twilio.IsEnabled() {
		err = h.twilio.SendVerificationCode(req.PhoneNumber, code)
		if err != nil {
		// Log error but don't fail the request - code is still stored in PostgreSQL
		fmt.Printf("Failed to send SMS via Twilio: %v\n", err)
		} else {
			twilioSent = true
		}
	}

	// Response
	response := gin.H{
		"message": "Verification code sent",
		"success": true,
	}

	// Only return code in development mode (when Twilio is disabled or failed)
	// In production with Twilio enabled, don't return the code
	if h.twilio == nil || !h.twilio.IsEnabled() || !twilioSent {
		response["code"] = code // Development mode or Twilio failed
	}

	c.JSON(http.StatusOK, response)
}

// VerifyCode verifies the code for login
func (h *AuthHandler) VerifyCode(c *gin.Context) {
	var req VerifyCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ok, err := h.consumeVerificationCode(context.Background(), req.PhoneNumber, req.Code)
	if err != nil {
		fmt.Printf("Error in consumeVerificationCode: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Verification lookup failed: " + err.Error()})
		return
	}
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired code"})
		return
	}

	// Find user
	var user models.User
	err = h.db.MongoDB.Collection("users").FindOne(
		context.Background(),
		bson.M{"phone_number": req.PhoneNumber},
	).Decode(&user)

	if err == mongo.ErrNoDocuments {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found. Please register first."})
		return
	}

	if err != nil {
		fmt.Printf("Error finding user in MongoDB: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}

	// Generate token
	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
}

// RegisterWithCode registers a new user after code verification
func (h *AuthHandler) RegisterWithCode(c *gin.Context) {
	var req RegisterWithCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate user type
	if req.UserType != "normal" && req.UserType != "company" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_type must be 'normal' or 'company'"})
		return
	}

	// Validate company fields if user type is company
	if req.UserType == "company" {
		if req.CompanyName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "company_name is required for company users"})
			return
		}
		if req.CompanyCategory == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "company_category is required for company users"})
			return
		}
	}

	// Verify code
	ok, err := h.consumeVerificationCode(context.Background(), req.PhoneNumber, req.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Verification lookup failed"})
		return
	}
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired code"})
		return
	}

	// Check if user already exists
	var existingUser models.User
	err = h.db.MongoDB.Collection("users").FindOne(
		context.Background(),
		bson.M{"phone_number": req.PhoneNumber},
	).Decode(&existingUser)

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		return
	}

	// Generate QR code
	userID := primitive.NewObjectID()
	qrData, qrBase64, err := utils.GenerateQRCode(userID.Hex())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate QR code"})
		return
	}

	// Create user
	user := models.User{
		ID:              userID,
		PhoneNumber:     req.PhoneNumber,
		QRCode:          qrBase64,
		Username:        req.Username,
		UserType:        req.UserType,
		CompanyName:     req.CompanyName,
		CompanyCategory: req.CompanyCategory,
		IsAnonymous:     false,
		AccountStatus:   "active",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		LastActive:      time.Now(),
	}

	_, err = h.db.MongoDB.Collection("users").InsertOne(context.Background(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Store QR data in PostgreSQL
	if h.db.Postgres != nil {
		qrCache := models.QRCodeCache{
			QRData: qrData,
			UserID: userID.Hex(),
		}
		// Delete existing entry if any
		h.db.Postgres.Where("qr_data = ?", qrData).Delete(&models.QRCodeCache{})
		// Insert new entry
		h.db.Postgres.Create(&qrCache)
	}

	// Generate token
	token, err := utils.GenerateToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"token": token,
		"user":  user,
		"qr":    qrBase64,
	})
}

func (h *AuthHandler) VerifyPhone(c *gin.Context) {
	// Legacy endpoint - redirects to SendCode
	h.SendCode(c)
}





