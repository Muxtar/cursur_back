package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Proxy struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"user_id" bson:"user_id"`
	Type        string            `json:"type" bson:"type"` // socks5, mtproto
	Server      string            `json:"server" bson:"server"`
	Port        int               `json:"port" bson:"port"`
	Username    string            `json:"username,omitempty" bson:"username,omitempty"`
	Password    string            `json:"password,omitempty" bson:"password,omitempty"`
	Secret      string            `json:"secret,omitempty" bson:"secret,omitempty"` // for MTProto
	IsActive    bool              `json:"is_active" bson:"is_active"`
	IsDefault   bool              `json:"is_default" bson:"is_default"`
	CreatedAt   time.Time         `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" bson:"updated_at"`
}

type VPN struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"user_id" bson:"user_id"`
	Provider    string            `json:"provider" bson:"provider"`
	Config      string            `json:"config" bson:"config"` // VPN configuration
	IsActive    bool              `json:"is_active" bson:"is_active"`
	CreatedAt   time.Time         `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" bson:"updated_at"`
}





