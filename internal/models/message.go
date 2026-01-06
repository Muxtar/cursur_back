package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Message struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ChatID      primitive.ObjectID `json:"chat_id" bson:"chat_id"`
	SenderID    primitive.ObjectID `json:"sender_id" bson:"sender_id"`
	Content     string            `json:"content" bson:"content"`
	MessageType string            `json:"message_type" bson:"message_type"` // text, image, audio, video, voice_message, video_message, file, location, contact, poll, sticker, gif, music
	FileURL     string            `json:"file_url,omitempty" bson:"file_url,omitempty"`
	ThumbnailURL string           `json:"thumbnail_url,omitempty" bson:"thumbnail_url,omitempty"`
	FileName    string            `json:"file_name,omitempty" bson:"file_name,omitempty"`
	FileSize    int64             `json:"file_size,omitempty" bson:"file_size,omitempty"`
	Duration    int               `json:"duration,omitempty" bson:"duration,omitempty"` // for audio/video in seconds
	
	// Message Status
	Status      string            `json:"status" bson:"status"` // sending, sent, delivered, read
	ReadBy      []ReadReceipt    `json:"read_by,omitempty" bson:"read_by,omitempty"`
	
	// Message Features
	IsEdited    bool              `json:"is_edited" bson:"is_edited"`
	EditedAt    *time.Time        `json:"edited_at,omitempty" bson:"edited_at,omitempty"`
	IsDeleted   bool              `json:"is_deleted" bson:"is_deleted"`
	DeletedAt   *time.Time        `json:"deleted_at,omitempty" bson:"deleted_at,omitempty"`
	DeletedFor  []primitive.ObjectID `json:"deleted_for,omitempty" bson:"deleted_for,omitempty"` // users who deleted it
	
	// Reply and Forward
	ReplyToID   *primitive.ObjectID `json:"reply_to_id,omitempty" bson:"reply_to_id,omitempty"`
	ForwardedFrom *primitive.ObjectID `json:"forwarded_from,omitempty" bson:"forwarded_from,omitempty"`
	ForwardedFromChat *primitive.ObjectID `json:"forwarded_from_chat,omitempty" bson:"forwarded_from_chat,omitempty"`
	
	// Reactions
	Reactions   []Reaction        `json:"reactions,omitempty" bson:"reactions,omitempty"`
	
	// Formatting
	Formatting  MessageFormatting  `json:"formatting,omitempty" bson:"formatting,omitempty"`
	
	// Media Specific
	Location    *MessageLocation         `json:"location,omitempty" bson:"location,omitempty"`
	Contact     *ContactInfo     `json:"contact,omitempty" bson:"contact,omitempty"`
	Poll        *Poll            `json:"poll,omitempty" bson:"poll,omitempty"`
	
	// Privacy
	IsAnonymous bool             `json:"is_anonymous" bson:"is_anonymous"`
	IsSecret    bool             `json:"is_secret" bson:"is_secret"`
	SelfDestructTTL int          `json:"self_destruct_ttl,omitempty" bson:"self_destruct_ttl,omitempty"` // seconds
	
	// Group Features
	IsPinned    bool             `json:"is_pinned" bson:"is_pinned"`
	PinnedAt    *time.Time       `json:"pinned_at,omitempty" bson:"pinned_at,omitempty"`
	ThreadID    *primitive.ObjectID `json:"thread_id,omitempty" bson:"thread_id,omitempty"` // for threaded replies
	
	// Link Preview
	LinkPreview *LinkPreview    `json:"link_preview,omitempty" bson:"link_preview,omitempty"`
	
	// Mentions
	Mentions    []primitive.ObjectID `json:"mentions,omitempty" bson:"mentions,omitempty"`
	
	// Translation
	TranslatedText string         `json:"translated_text,omitempty" bson:"translated_text,omitempty"`
	TranslatedTo   string         `json:"translated_to,omitempty" bson:"translated_to,omitempty"`
	
	// Scheduling
	ScheduledFor *time.Time      `json:"scheduled_for,omitempty" bson:"scheduled_for,omitempty"`
	IsDraft      bool            `json:"is_draft" bson:"is_draft"`
	
	// Bot Integration
	BotCommand  string           `json:"bot_command,omitempty" bson:"bot_command,omitempty"`
	InlineBot   string           `json:"inline_bot,omitempty" bson:"inline_bot,omitempty"`
	
	CreatedAt   time.Time        `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at" bson:"updated_at"`
}

type ReadReceipt struct {
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	ReadAt    time.Time          `json:"read_at" bson:"read_at"`
}

type Reaction struct {
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	Emoji     string            `json:"emoji" bson:"emoji"`
	CreatedAt time.Time         `json:"created_at" bson:"created_at"`
}

type MessageFormatting struct {
	Bold      []TextRange `json:"bold,omitempty" bson:"bold,omitempty"`
	Italic    []TextRange `json:"italic,omitempty" bson:"italic,omitempty"`
	Strikethrough []TextRange `json:"strikethrough,omitempty" bson:"strikethrough,omitempty"`
	Code      []TextRange `json:"code,omitempty" bson:"code,omitempty"`
	Links     []Link      `json:"links,omitempty" bson:"links,omitempty"`
}

type TextRange struct {
	Start int `json:"start" bson:"start"`
	End   int `json:"end" bson:"end"`
}

type Link struct {
	URL   string `json:"url" bson:"url"`
	Start int    `json:"start" bson:"start"`
	End   int    `json:"end" bson:"end"`
}

type MessageLocation struct {
	Latitude  float64 `json:"latitude" bson:"latitude"`
	Longitude float64 `json:"longitude" bson:"longitude"`
	Address   string  `json:"address,omitempty" bson:"address,omitempty"`
	IsLive    bool    `json:"is_live" bson:"is_live"` // for live location
	LiveUntil *time.Time `json:"live_until,omitempty" bson:"live_until,omitempty"`
}

type ContactInfo struct {
	Name        string `json:"name" bson:"name"`
	PhoneNumber string `json:"phone_number" bson:"phone_number"`
	UserID      *primitive.ObjectID `json:"user_id,omitempty" bson:"user_id,omitempty"`
}

type Poll struct {
	Question    string   `json:"question" bson:"question"`
	Options     []PollOption `json:"options" bson:"options"`
	IsMultiple  bool     `json:"is_multiple" bson:"is_multiple"`
	IsAnonymous bool     `json:"is_anonymous" bson:"is_anonymous"`
	EndsAt      *time.Time `json:"ends_at,omitempty" bson:"ends_at,omitempty"`
}

type PollOption struct {
	ID      string   `json:"id" bson:"id"`
	Text    string   `json:"text" bson:"text"`
	Votes   []primitive.ObjectID `json:"votes" bson:"votes"`
}

type LinkPreview struct {
	URL         string `json:"url" bson:"url"`
	Title       string `json:"title,omitempty" bson:"title,omitempty"`
	Description string `json:"description,omitempty" bson:"description,omitempty"`
	ImageURL    string `json:"image_url,omitempty" bson:"image_url,omitempty"`
	SiteName    string `json:"site_name,omitempty" bson:"site_name,omitempty"`
}

type Chat struct {
	ID        primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	Type      string              `json:"type" bson:"type"` // direct, group, channel
	Members   []primitive.ObjectID `json:"members" bson:"members"`
	Admins    []AdminRole         `json:"admins,omitempty" bson:"admins,omitempty"`
	GroupName string              `json:"group_name,omitempty" bson:"group_name,omitempty"`
	GroupIcon string              `json:"group_icon,omitempty" bson:"group_icon,omitempty"`
	Description string            `json:"description,omitempty" bson:"description,omitempty"`
	PublicLink string             `json:"public_link,omitempty" bson:"public_link,omitempty"` // t.me/username
	InviteLink string             `json:"invite_link,omitempty" bson:"invite_link,omitempty"`
	MaxMembers int                `json:"max_members,omitempty" bson:"max_members,omitempty"` // 200000 for groups
	PinnedMessages []primitive.ObjectID `json:"pinned_messages,omitempty" bson:"pinned_messages,omitempty"`
	IsSecret  bool                `json:"is_secret" bson:"is_secret"`
	SlowMode  int                 `json:"slow_mode,omitempty" bson:"slow_mode,omitempty"` // seconds between messages
	LastSlowModeMessage map[string]time.Time `json:"last_slow_mode_message,omitempty" bson:"last_slow_mode_message,omitempty"`
	UnreadCount map[string]int    `json:"unread_count,omitempty" bson:"unread_count,omitempty"`
	LastMessageID *primitive.ObjectID `json:"last_message_id,omitempty" bson:"last_message_id,omitempty"`
	LastMessageAt *time.Time      `json:"last_message_at,omitempty" bson:"last_message_at,omitempty"`
	Wallpaper  string             `json:"wallpaper,omitempty" bson:"wallpaper,omitempty"`
	MutedUntil *time.Time         `json:"muted_until,omitempty" bson:"muted_until,omitempty"`
	
	// Channel specific
	SubscriberCount int            `json:"subscriber_count,omitempty" bson:"subscriber_count,omitempty"`
	ViewCount       map[string]int  `json:"view_count,omitempty" bson:"view_count,omitempty"` // message_id -> count
	CommentGroupID  *primitive.ObjectID `json:"comment_group_id,omitempty" bson:"comment_group_id,omitempty"`
	IsBroadcast     bool           `json:"is_broadcast" bson:"is_broadcast"` // for channels
	
	// Group restrictions
	Restrictions GroupRestrictions `json:"restrictions,omitempty" bson:"restrictions,omitempty"`
	
	// Statistics
	Statistics ChatStatistics     `json:"statistics,omitempty" bson:"statistics,omitempty"`
	
	CreatedAt time.Time           `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time           `json:"updated_at" bson:"updated_at"`
}

type AdminRole struct {
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	Role      string            `json:"role" bson:"role"` // owner, admin, moderator
	Permissions []string        `json:"permissions" bson:"permissions"` // delete, ban, pin, invite, etc.
	GrantedAt time.Time         `json:"granted_at" bson:"granted_at"`
	GrantedBy primitive.ObjectID `json:"granted_by" bson:"granted_by"`
}

type GroupRestrictions struct {
	CanSendMessages bool `json:"can_send_messages" bson:"can_send_messages"`
	CanSendMedia    bool `json:"can_send_media" bson:"can_send_media"`
	CanSendLinks    bool `json:"can_send_links" bson:"can_send_links"`
	CanSendPolls    bool `json:"can_send_polls" bson:"can_send_polls"`
	CanChangeInfo   bool `json:"can_change_info" bson:"can_change_info"`
	CanInviteUsers  bool `json:"can_invite_users" bson:"can_invite_users"`
}

type ChatStatistics struct {
	TotalMessages    int64     `json:"total_messages" bson:"total_messages"`
	TotalMedia       int64     `json:"total_media" bson:"total_media"`
	TotalViews       int64     `json:"total_views" bson:"total_views"`
	ActiveMembers    int       `json:"active_members" bson:"active_members"`
	GrowthRate       float64   `json:"growth_rate" bson:"growth_rate"`
	LastCalculated   time.Time `json:"last_calculated" bson:"last_calculated"`
}

type TypingIndicator struct {
	ChatID    primitive.ObjectID `json:"chat_id" bson:"chat_id"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	Type      string            `json:"type" bson:"type"` // typing, recording_voice, recording_video
	StartedAt time.Time         `json:"started_at" bson:"started_at"`
}

type MessageSearch struct {
	Query     string            `json:"query" bson:"query"`
	ChatID    *primitive.ObjectID `json:"chat_id,omitempty" bson:"chat_id,omitempty"`
	SenderID  *primitive.ObjectID `json:"sender_id,omitempty" bson:"sender_id,omitempty"`
	DateFrom  *time.Time        `json:"date_from,omitempty" bson:"date_from,omitempty"`
	DateTo    *time.Time        `json:"date_to,omitempty" bson:"date_to,omitempty"`
	MessageType string          `json:"message_type,omitempty" bson:"message_type,omitempty"`
}
