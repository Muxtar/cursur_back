package utils

import (
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
)

func GenerateQRCode(userID string) (string, string, error) {
	// Generate unique QR code data
	qrData := fmt.Sprintf("CHATAPP:%s:%s", userID, uuid.New().String())
	
	// Generate QR code image
	qr, err := qrcode.New(qrData, qrcode.Medium)
	if err != nil {
		return "", "", err
	}

	// Convert to PNG bytes
	png, err := qr.PNG(256)
	if err != nil {
		return "", "", err
	}

	// Convert to base64 string for storage
	qrBase64 := base64.StdEncoding.EncodeToString(png)
	
	return qrData, qrBase64, nil
}

func ParseQRCode(qrData string) (string, error) {
	// Parse QR code format: CHATAPP:user_id:uuid
	var userID string
	_, err := fmt.Sscanf(qrData, "CHATAPP:%s", &userID)
	if err != nil {
		return "", err
	}
	return userID, nil
}





