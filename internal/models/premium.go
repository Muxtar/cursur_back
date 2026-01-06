package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PremiumSubscription struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"user_id" bson:"user_id"`
	Plan        string            `json:"plan" bson:"plan"` // monthly, yearly, lifetime
	Status      string            `json:"status" bson:"status"` // active, expired, cancelled
	StartedAt   time.Time         `json:"started_at" bson:"started_at"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty" bson:"expires_at,omitempty"`
	AutoRenew   bool              `json:"auto_renew" bson:"auto_renew"`
	PaymentID   string            `json:"payment_id,omitempty" bson:"payment_id,omitempty"`
	CreatedAt   time.Time         `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" bson:"updated_at"`
}

type PremiumFeature struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"user_id" bson:"user_id"`
	Feature     string            `json:"feature" bson:"feature"` // 4gb_files, faster_download, badge, animated_emoji, voice_translation, ad_free, premium_stickers, voice_to_text
	IsActive    bool              `json:"is_active" bson:"is_active"`
	ActivatedAt time.Time         `json:"activated_at" bson:"activated_at"`
}





