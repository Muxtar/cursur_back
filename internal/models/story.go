package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Story represents a 24-hour story post.
// Type "media"   → MediaURL + MediaType + optional Text.
// Type "product" → ProductID (references an existing product).
type Story struct {
	ID        primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID  `json:"user_id" bson:"user_id"`
	Type      string              `json:"type" bson:"type"`                               // "media" | "product"
	MediaURL  string              `json:"media_url,omitempty" bson:"media_url,omitempty"` // for type "media"
	MediaType string              `json:"media_type,omitempty" bson:"media_type,omitempty"` // "image" | "video"
	Text      string              `json:"text,omitempty" bson:"text,omitempty"`
	ProductID *primitive.ObjectID `json:"product_id,omitempty" bson:"product_id,omitempty"` // for type "product"
	ExpiresAt time.Time           `json:"expires_at" bson:"expires_at"`                   // CreatedAt + 24 h
	CreatedAt time.Time           `json:"created_at" bson:"created_at"`
}

// StoryFeedItem is what the feed endpoint returns:
// one user's active stories bundled together.
type StoryFeedItem struct {
	UserID   primitive.ObjectID `json:"user_id"`
	UserInfo interface{}        `json:"user_info"` // basic user fields injected by handler
	Stories  []Story            `json:"stories"`
}
