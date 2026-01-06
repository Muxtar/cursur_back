package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"chat-backend/internal/config"
	"chat-backend/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type FileHandler struct {
	db *database.Database
}

func NewFileHandler(db *database.Database) *FileHandler {
	return &FileHandler{db: db}
}

func (h *FileHandler) UploadFile(c *gin.Context) {
	cfg := config.Load()

	// Parse multipart form
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	// Check file size
	if file.Size > cfg.MaxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large"})
		return
	}

	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(cfg.UploadDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	filePath := filepath.Join(cfg.UploadDir, filename)

	// Save file
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// In production, upload to cloud storage (S3, etc.)
	fileURL := fmt.Sprintf("/uploads/%s", filename)

	c.JSON(http.StatusOK, gin.H{
		"file_url": fileURL,
		"filename": filename,
		"size":     file.Size,
		"type":     file.Header.Get("Content-Type"),
	})
}

func (h *FileHandler) ServeFile(c *gin.Context) {
	cfg := config.Load()
	filename := c.Param("filename")
	filePath := filepath.Join(cfg.UploadDir, filename)

	// Security: prevent directory traversal
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(cfg.UploadDir, filepath.Base(filename))
	}

	file, err := os.Open(filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	defer file.Close()

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.File(filePath)
}

