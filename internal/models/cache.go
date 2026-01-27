package models

import (
	"time"

	"gorm.io/gorm"
)

// VerificationCode stores phone verification codes in PostgreSQL
type VerificationCode struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PhoneNumber string    `gorm:"index;not null" json:"phone_number"`
	Code        string    `gorm:"not null" json:"code"`
	ExpiresAt   time.Time `gorm:"index;not null" json:"expires_at"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName specifies the table name for VerificationCode
func (VerificationCode) TableName() string {
	return "verification_codes"
}

// QRCodeCache stores QR code data for quick lookup
type QRCodeCache struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	QRData    string    `gorm:"uniqueIndex;not null" json:"qr_data"`
	UserID    string    `gorm:"not null" json:"user_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName specifies the table name for QRCodeCache
func (QRCodeCache) TableName() string {
	return "qr_code_cache"
}

// TypingIndicator stores typing indicators for chats
type TypingIndicator struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ChatID    string    `gorm:"index;not null" json:"chat_id"`
	UserID    string    `gorm:"index;not null" json:"user_id"`
	Type      string    `gorm:"not null" json:"type"` // typing, recording_voice, recording_video
	ExpiresAt time.Time `gorm:"index;not null" json:"expires_at"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName specifies the table name for TypingIndicator
func (TypingIndicator) TableName() string {
	return "typing_indicators"
}

// CacheEntry stores general cache entries
type CacheEntry struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"uniqueIndex;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	ExpiresAt *time.Time `gorm:"index" json:"expires_at,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName specifies the table name for CacheEntry
func (CacheEntry) TableName() string {
	return "cache_entries"
}

// BeforeCreate hook to set default expiration if not set
func (c *CacheEntry) BeforeCreate(tx *gorm.DB) error {
	if c.ExpiresAt == nil {
		// Default expiration: 1 hour
		expires := time.Now().Add(1 * time.Hour)
		c.ExpiresAt = &expires
	}
	return nil
}
