package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Product represents a product/item that users can share
type Product struct {
	ID          primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	Name        string               `json:"name" bson:"name"`
	Description string               `json:"description" bson:"description"`
	Category    string               `json:"category" bson:"category"` // vehicle, house, design, collection, nft, etc.
	MediaURLs   []string             `json:"media_urls" bson:"media_urls"` // Array of image/video URLs
	ProductID   string               `json:"product_id" bson:"product_id"` // Unique product identifier
	OwnerID     primitive.ObjectID   `json:"owner_id" bson:"owner_id"` // User who owns this product
	Price       *float64             `json:"price,omitempty" bson:"price,omitempty"` // Optional price
	IsVerified  bool                 `json:"is_verified" bson:"is_verified"` // Ownership verification status
	Privacy     string               `json:"privacy" bson:"privacy"` // public, followers, private
	LikeCount   int                  `json:"like_count" bson:"like_count"`
	CommentCount int                 `json:"comment_count" bson:"comment_count"`
	ViewCount   int                  `json:"view_count" bson:"view_count"`
	CreatedAt   time.Time            `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at" bson:"updated_at"`
}

// Comment represents a comment on a product
type Comment struct {
	ID        primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	ProductID primitive.ObjectID   `json:"product_id" bson:"product_id"`
	UserID    primitive.ObjectID   `json:"user_id" bson:"user_id"`
	Content   string               `json:"content" bson:"content"`
	ParentID  *primitive.ObjectID  `json:"parent_id,omitempty" bson:"parent_id,omitempty"` // For reply comments
	LikeCount int                  `json:"like_count" bson:"like_count"`
	IsSpam    bool                 `json:"is_spam" bson:"is_spam"`
	CreatedAt time.Time            `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time            `json:"updated_at" bson:"updated_at"`
}

// Like represents a like on a product or comment
type Like struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ProductID *primitive.ObjectID `json:"product_id,omitempty" bson:"product_id,omitempty"` // For product likes
	CommentID *primitive.ObjectID `json:"comment_id,omitempty" bson:"comment_id,omitempty"` // For comment likes
	UserID    primitive.ObjectID  `json:"user_id" bson:"user_id"`
	CreatedAt time.Time           `json:"created_at" bson:"created_at"`
}



