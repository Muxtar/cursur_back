package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Bot struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	OwnerID     primitive.ObjectID `json:"owner_id" bson:"owner_id"`
	Username    string            `json:"username" bson:"username"` // @botname
	Token       string            `json:"token" bson:"token"` // API token
	Name        string            `json:"name" bson:"name"`
	Description string            `json:"description" bson:"description"`
	Avatar      string            `json:"avatar,omitempty" bson:"avatar,omitempty"`
	
	// Bot Settings
	IsActive    bool              `json:"is_active" bson:"is_active"`
	IsInline    bool              `json:"is_inline" bson:"is_inline"` // inline bot support
	CanJoinGroups bool            `json:"can_join_groups" bson:"can_join_groups"`
	CanReadAllGroupMessages bool `json:"can_read_all_group_messages" bson:"can_read_all_group_messages"`
	
	// Commands
	Commands    []BotCommand     `json:"commands" bson:"commands"`
	
	// Web App
	WebAppURL   string            `json:"web_app_url,omitempty" bson:"web_app_url,omitempty"`
	
	// Permissions
	Permissions BotPermissions    `json:"permissions" bson:"permissions"`
	
	CreatedAt   time.Time        `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at" bson:"updated_at"`
}

type BotCommand struct {
	Command     string `json:"command" bson:"command"` // /start, /help, etc.
	Description string `json:"description" bson:"description"`
	Handler     string `json:"handler,omitempty" bson:"handler,omitempty"` // webhook URL or function
}

type BotPermissions struct {
	CanReadMessages   bool `json:"can_read_messages" bson:"can_read_messages"`
	CanSendMessages   bool `json:"can_send_messages" bson:"can_send_messages"`
	CanDeleteMessages bool `json:"can_delete_messages" bson:"can_delete_messages"`
	CanManageChat     bool `json:"can_manage_chat" bson:"can_manage_chat"`
	CanInviteUsers     bool `json:"can_invite_users" bson:"can_invite_users"`
	CanRestrictMembers bool `json:"can_restrict_members" bson:"can_restrict_members"`
	CanPromoteMembers  bool `json:"can_promote_members" bson:"can_promote_members"`
}

type BotMessage struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	BotID     primitive.ObjectID `json:"bot_id" bson:"bot_id"`
	ChatID    primitive.ObjectID `json:"chat_id" bson:"chat_id"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	Command   string            `json:"command,omitempty" bson:"command,omitempty"`
	Query     string            `json:"query,omitempty" bson:"query,omitempty"` // for inline queries
	Response  string            `json:"response" bson:"response"`
	CreatedAt time.Time         `json:"created_at" bson:"created_at"`
}

type InlineQuery struct {
	ID        string            `json:"id" bson:"id"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	BotID     primitive.ObjectID `json:"bot_id" bson:"bot_id"`
	Query     string            `json:"query" bson:"query"`
	Offset    string            `json:"offset" bson:"offset"`
	Results   []InlineResult    `json:"results" bson:"results"`
	CreatedAt time.Time         `json:"created_at" bson:"created_at"`
}

type InlineResult struct {
	Type        string            `json:"type" bson:"type"` // article, photo, gif, video, etc.
	ID          string            `json:"id" bson:"id"`
	Title       string            `json:"title" bson:"title"`
	Description string            `json:"description,omitempty" bson:"description,omitempty"`
	ThumbURL    string            `json:"thumb_url,omitempty" bson:"thumb_url,omitempty"`
	ContentURL  string            `json:"content_url,omitempty" bson:"content_url,omitempty"`
	MessageText string            `json:"message_text,omitempty" bson:"message_text,omitempty"`
}





