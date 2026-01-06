package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Proposal struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	SenderID    primitive.ObjectID `json:"sender_id" bson:"sender_id"`
	ReceiverID  primitive.ObjectID `json:"receiver_id" bson:"receiver_id"`
	Title       string            `json:"title" bson:"title"`
	Content     string            `json:"content" bson:"content"`
	Status      string            `json:"status" bson:"status"` // pending, accepted, rejected
	CreatedAt   time.Time         `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" bson:"updated_at"`
}





