package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserSettings struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	
	// Account Settings
	Account AccountSettings `json:"account" bson:"account"`
	
	// Privacy & Security
	Privacy PrivacySettings `json:"privacy" bson:"privacy"`
	
	// Chat Settings
	Chat ChatSettings `json:"chat" bson:"chat"`
	
	// Notifications
	Notifications NotificationSettings `json:"notifications" bson:"notifications"`
	
	// Appearance
	Appearance AppearanceSettings `json:"appearance" bson:"appearance"`
	
	// Data & Storage
	Data DataSettings `json:"data" bson:"data"`
	
	// Devices & Sessions
	Devices DeviceSettings `json:"devices" bson:"devices"`
	
	// Calls
	Calls CallSettings `json:"calls" bson:"calls"`
	
	// Groups & Channels
	Groups GroupSettings `json:"groups" bson:"groups"`
	
	// Advanced
	Advanced AdvancedSettings `json:"advanced" bson:"advanced"`
	
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

type AccountSettings struct {
	Username        string `json:"username" bson:"username"`
	PhoneNumber     string `json:"phone_number" bson:"phone_number"`
	Email           string `json:"email" bson:"email"`
	Bio             string `json:"bio" bson:"bio"`
	ProfilePhoto    string `json:"profile_photo" bson:"profile_photo"`
	TwoStepEnabled  bool   `json:"two_step_enabled" bson:"two_step_enabled"`
	AccountStatus   string `json:"account_status" bson:"account_status"` // active, suspended, deleted
	RecoveryEmail   string `json:"recovery_email" bson:"recovery_email"`
}

type PrivacySettings struct {
	LastSeen        string   `json:"last_seen" bson:"last_seen"` // everyone, contacts, nobody
	OnlineStatus    string   `json:"online_status" bson:"online_status"` // everyone, contacts, nobody
	ProfilePhoto    string   `json:"profile_photo" bson:"profile_photo"` // everyone, contacts, nobody
	BioVisibility   string   `json:"bio_visibility" bson:"bio_visibility"` // everyone, contacts, nobody
	FindByPhone     bool     `json:"find_by_phone" bson:"find_by_phone"`
	FindByUsername  bool     `json:"find_by_username" bson:"find_by_username"`
	BlockedUsers    []primitive.ObjectID `json:"blocked_users" bson:"blocked_users"`
	TwoStepEnabled  bool     `json:"two_step_enabled" bson:"two_step_enabled"`
	ActiveSessions  []Session `json:"active_sessions" bson:"active_sessions"`
	SecretChatTTL   int      `json:"secret_chat_ttl" bson:"secret_chat_ttl"` // seconds
	EncryptionLevel string   `json:"encryption_level" bson:"encryption_level"` // standard, end-to-end
	SpamReports     bool     `json:"spam_reports" bson:"spam_reports"`
}

type Session struct {
	ID          string    `json:"id" bson:"id"`
	DeviceName  string    `json:"device_name" bson:"device_name"`
	IPAddress   string    `json:"ip_address" bson:"ip_address"`
	LastActive  time.Time `json:"last_active" bson:"last_active"`
	IsCurrent   bool      `json:"is_current" bson:"is_current"`
}

type ChatSettings struct {
	Wallpaper      string `json:"wallpaper" bson:"wallpaper"`
	Theme          string `json:"theme" bson:"theme"` // light, dark, custom
	FontSize       string `json:"font_size" bson:"font_size"` // small, medium, large
	EmojiEnabled   bool   `json:"emoji_enabled" bson:"emoji_enabled"`
	StickersEnabled bool  `json:"stickers_enabled" bson:"stickers_enabled"`
	GIFEnabled     bool   `json:"gif_enabled" bson:"gif_enabled"`
	AutoDownload   AutoDownloadSettings `json:"auto_download" bson:"auto_download"`
	MessagePreview bool   `json:"message_preview" bson:"message_preview"`
	ReadReceipts   bool   `json:"read_receipts" bson:"read_receipts"`
	PinnedChats    []primitive.ObjectID `json:"pinned_chats" bson:"pinned_chats"`
	MuteDuration   int    `json:"mute_duration" bson:"mute_duration"` // hours, 0 = forever
	ChatFolders    []ChatFolder `json:"chat_folders" bson:"chat_folders"`
}

type AutoDownloadSettings struct {
	Photos     string `json:"photos" bson:"photos"` // wifi, mobile, never
	Videos     string `json:"videos" bson:"videos"`
	Audio      string `json:"audio" bson:"audio"`
	Documents  string `json:"documents" bson:"documents"`
}

type ChatFolder struct {
	Name    string   `json:"name" bson:"name"`
	ChatIDs []primitive.ObjectID `json:"chat_ids" bson:"chat_ids"`
	Color   string   `json:"color" bson:"color"`
}

type NotificationSettings struct {
	DirectChats    bool   `json:"direct_chats" bson:"direct_chats"`
	GroupChats     bool   `json:"group_chats" bson:"group_chats"`
	Channels       bool   `json:"channels" bson:"channels"`
	Calls          bool   `json:"calls" bson:"calls"`
	Sound          string `json:"sound" bson:"sound"`
	Vibration      string `json:"vibration" bson:"vibration"` // default, short, long, off
	SilentMode     bool   `json:"silent_mode" bson:"silent_mode"`
	DoNotDisturb   bool   `json:"do_not_disturb" bson:"do_not_disturb"`
	Priority       string `json:"priority" bson:"priority"` // high, normal, low
}

type AppearanceSettings struct {
	Theme          string `json:"theme" bson:"theme"` // light, dark, system
	ColorPalette   string `json:"color_palette" bson:"color_palette"`
	FontSize       string `json:"font_size" bson:"font_size"`
	BubbleStyle    string `json:"bubble_style" bson:"bubble_style"` // default, rounded, square
	IconLayout     string `json:"icon_layout" bson:"icon_layout"`
	Animations     bool   `json:"animations" bson:"animations"`
	AutoNightMode  bool   `json:"auto_night_mode" bson:"auto_night_mode"`
}

type DataSettings struct {
	AutoDownload   AutoDownloadSettings `json:"auto_download" bson:"auto_download"`
	StorageLimit   int64  `json:"storage_limit" bson:"storage_limit"` // bytes
	MaxMediaAge    int    `json:"max_media_age" bson:"max_media_age"` // days
	CloudSync      bool   `json:"cloud_sync" bson:"cloud_sync"`
	DataUsage      DataUsageStats `json:"data_usage" bson:"data_usage"`
	MobileData     DataBehaviorSettings `json:"mobile_data" bson:"mobile_data"`
	WiFiData       DataBehaviorSettings `json:"wifi_data" bson:"wifi_data"`
	RoamingData    DataBehaviorSettings `json:"roaming_data" bson:"roaming_data"`
}

type DataUsageStats struct {
	TotalSent     int64 `json:"total_sent" bson:"total_sent"`
	TotalReceived int64 `json:"total_received" bson:"total_received"`
	Photos        int64 `json:"photos" bson:"photos"`
	Videos        int64 `json:"videos" bson:"videos"`
	Audio         int64 `json:"audio" bson:"audio"`
	Documents     int64 `json:"documents" bson:"documents"`
}

type DataBehaviorSettings struct {
	AutoDownload bool `json:"auto_download" bson:"auto_download"`
	VideoQuality string `json:"video_quality" bson:"video_quality"` // low, medium, high
}

type DeviceSettings struct {
	RequireAuthForNewDevice bool     `json:"require_auth_for_new_device" bson:"require_auth_for_new_device"`
	DeviceNotifications     bool     `json:"device_notifications" bson:"device_notifications"`
}

type CallSettings struct {
	Quality          string `json:"quality" bson:"quality"` // low, medium, high
	DataUsageMode    string `json:"data_usage_mode" bson:"data_usage_mode"` // low, medium, high
	VideoCalls       bool   `json:"video_calls" bson:"video_calls"`
	VoiceCalls       bool   `json:"voice_calls" bson:"voice_calls"`
	WhoCanCall       string `json:"who_can_call" bson:"who_can_call"` // everyone, contacts, nobody
	CallHistory      bool   `json:"call_history" bson:"call_history"`
	NoiseSuppression bool   `json:"noise_suppression" bson:"noise_suppression"`
	EchoCancellation bool   `json:"echo_cancellation" bson:"echo_cancellation"`
}

type GroupSettings struct {
	WhoCanCreate     string   `json:"who_can_create" bson:"who_can_create"` // everyone, contacts, admins
	AdminPermissions []string `json:"admin_permissions" bson:"admin_permissions"`
	JoinApproval     bool     `json:"join_approval" bson:"join_approval"`
	InviteLinks      bool     `json:"invite_links" bson:"invite_links"`
	SlowMode         bool     `json:"slow_mode" bson:"slow_mode"`
	SlowModeDelay    int      `json:"slow_mode_delay" bson:"slow_mode_delay"` // seconds
	WordFilter       bool     `json:"word_filter" bson:"word_filter"`
	FilteredWords    []string `json:"filtered_words" bson:"filtered_words"`
}

type AdvancedSettings struct {
	SelfDestructTTL  int    `json:"self_destruct_ttl" bson:"self_destruct_ttl"` // days
	ProxyEnabled     bool   `json:"proxy_enabled" bson:"proxy_enabled"`
	ProxyServer      string `json:"proxy_server" bson:"proxy_server"`
	VPNEnabled       bool   `json:"vpn_enabled" bson:"vpn_enabled"`
	SecretMode       bool   `json:"secret_mode" bson:"secret_mode"` // read receipts off
	ScreenshotBlock  bool   `json:"screenshot_block" bson:"screenshot_block"`
	AutoLogout       int    `json:"auto_logout" bson:"auto_logout"` // minutes, 0 = never
	SecureKeyboard   bool   `json:"secure_keyboard" bson:"secure_keyboard"`
}





