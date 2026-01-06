package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Call struct {
	ID        primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	Type      string              `json:"type" bson:"type"` // video, voice, group
	CallerID  primitive.ObjectID   `json:"caller_id" bson:"caller_id"`
	ChatID    primitive.ObjectID   `json:"chat_id" bson:"chat_id"`
	Members   []primitive.ObjectID `json:"members" bson:"members"`
	Status    string              `json:"status" bson:"status"` // ringing, active, ended
	StartedAt time.Time           `json:"started_at" bson:"started_at"`
	EndedAt   *time.Time          `json:"ended_at,omitempty" bson:"ended_at,omitempty"`
}





