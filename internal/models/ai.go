package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SmartReply struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"user_id" bson:"user_id"`
	ChatID      primitive.ObjectID `json:"chat_id" bson:"chat_id"`
	MessageID   primitive.ObjectID `json:"message_id" bson:"message_id"`
	Suggestions []string          `json:"suggestions" bson:"suggestions"`
	CreatedAt   time.Time         `json:"created_at" bson:"created_at"`
}

type MediaRecognition struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	MessageID   primitive.ObjectID `json:"message_id" bson:"message_id"`
	MediaType   string            `json:"media_type" bson:"media_type"` // image, video
	MediaURL    string            `json:"media_url" bson:"media_url"`
	Objects     []DetectedObject  `json:"objects" bson:"objects"`
	Text        string            `json:"text,omitempty" bson:"text,omitempty"` // OCR text
	Labels      []string          `json:"labels,omitempty" bson:"labels,omitempty"`
	CreatedAt   time.Time         `json:"created_at" bson:"created_at"`
}

type DetectedObject struct {
	Label      string  `json:"label" bson:"label"`
	Confidence float64 `json:"confidence" bson:"confidence"`
	Bounds     Bounds  `json:"bounds,omitempty" bson:"bounds,omitempty"`
}

type Bounds struct {
	X      int `json:"x" bson:"x"`
	Y      int `json:"y" bson:"y"`
	Width  int `json:"width" bson:"width"`
	Height int `json:"height" bson:"height"`
}

type VoiceToText struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	MessageID   primitive.ObjectID `json:"message_id" bson:"message_id"`
	AudioURL    string            `json:"audio_url" bson:"audio_url"`
	Transcript  string            `json:"transcript" bson:"transcript"`
	Language    string            `json:"language" bson:"language"`
	Confidence  float64           `json:"confidence" bson:"confidence"`
	CreatedAt   time.Time         `json:"created_at" bson:"created_at"`
}

type SpamDetection struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"user_id" bson:"user_id"`
	MessageID   primitive.ObjectID `json:"message_id" bson:"message_id"`
	IsSpam      bool              `json:"is_spam" bson:"is_spam"`
	Confidence  float64           `json:"confidence" bson:"confidence"`
	Reason      string            `json:"reason,omitempty" bson:"reason,omitempty"`
	CreatedAt   time.Time         `json:"created_at" bson:"created_at"`
}





