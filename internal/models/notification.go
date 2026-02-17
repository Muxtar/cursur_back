package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Notification struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"` // Receiver (target user)
	Type      string             `json:"type" bson:"type"`       // e.g., "profile_comment"
	Title     string             `json:"title" bson:"title"`
	Body      string             `json:"body" bson:"body"`
	Data      map[string]interface{} `json:"data,omitempty" bson:"data,omitempty"` // e.g., {"comment_id": "...", "phone": "..."}
	ReadAt    *time.Time         `json:"read_at,omitempty" bson:"read_at,omitempty"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}
