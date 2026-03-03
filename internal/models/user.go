package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	PhoneNumber     string             `json:"phone_number" bson:"phone_number"`
	QRCode          string             `json:"qr_code" bson:"qr_code"`
	Username        string             `json:"username" bson:"username"`
	FirstName       string             `json:"first_name,omitempty" bson:"first_name,omitempty"`
	LastName        string             `json:"last_name,omitempty" bson:"last_name,omitempty"`
	Avatar          string             `json:"avatar" bson:"avatar"`
	Bio             string             `json:"bio,omitempty" bson:"bio,omitempty"`
	Profession      string             `json:"profession,omitempty" bson:"profession,omitempty"` // meslek / peşə
	IsAnonymous     bool               `json:"is_anonymous" bson:"is_anonymous"`
	HidePhoneNumber bool               `json:"hide_phone_number" bson:"hide_phone_number"` // Gizli numara
	IsPremium       bool               `json:"is_premium" bson:"is_premium"`
	PremiumUntil    *time.Time         `json:"premium_until,omitempty" bson:"premium_until,omitempty"`
	Location        Location           `json:"location" bson:"location"`
	ActiveDevices   []DeviceInfo       `json:"active_devices,omitempty" bson:"active_devices,omitempty"`
	AccountStatus   string             `json:"account_status" bson:"account_status"`                           // active, suspended, deleted
	SelfDestructTTL int                `json:"self_destruct_ttl,omitempty" bson:"self_destruct_ttl,omitempty"` // days
	UserType        string             `json:"user_type" bson:"user_type"`                                     // "normal" or "company"
	CompanyName     string             `json:"company_name,omitempty" bson:"company_name,omitempty"`
	CompanyCategory string             `json:"company_category,omitempty" bson:"company_category,omitempty"`
	LastActive      time.Time          `json:"last_active" bson:"last_active"`
	CreatedAt       time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" bson:"updated_at"`
}

type DeviceInfo struct {
	DeviceID   string    `json:"device_id" bson:"device_id"`
	DeviceName string    `json:"device_name" bson:"device_name"`
	DeviceType string    `json:"device_type" bson:"device_type"` // mobile, tablet, desktop, web
	IPAddress  string    `json:"ip_address" bson:"ip_address"`
	LastActive time.Time `json:"last_active" bson:"last_active"`
	IsCurrent  bool      `json:"is_current" bson:"is_current"`
}

type Location struct {
	Latitude  float64 `json:"latitude" bson:"latitude"`
	Longitude float64 `json:"longitude" bson:"longitude"`
	Address   string  `json:"address" bson:"address"`
}

type Contact struct {
	ID          primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID  `json:"user_id" bson:"user_id"`
	ContactID   *primitive.ObjectID `json:"contact_id,omitempty" bson:"contact_id,omitempty"` // nil when added by phone only
	ContactQR   string              `json:"contact_qr" bson:"contact_qr"`
	PhoneNumber string              `json:"phone_number,omitempty" bson:"phone_number,omitempty"` // when adding by number
	DisplayName string              `json:"display_name,omitempty" bson:"display_name,omitempty"` // name given by user
	IsAnonymous bool                `json:"is_anonymous" bson:"is_anonymous"`
	CreatedAt   time.Time           `json:"created_at" bson:"created_at"`
}

// ProfileComment: anonymous comment about a phone number, car number, or person name.
// target_type: "phone" -> target_user_id set, owner gets notification and can delete.
// target_type: "car_number" | "person_name" -> only target_value set, searchable only, no delete/notification.
type ProfileComment struct {
	ID           primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	TargetType   string              `json:"target_type" bson:"target_type"`                       // "phone" | "car_number" | "person_name"
	TargetValue  string              `json:"target_value" bson:"target_value"`                     // phone, plate, or name (normalized)
	TargetUserID *primitive.ObjectID `json:"target_user_id,omitempty" bson:"target_user_id,omitempty"` // only when target_type == "phone"
	CommenterID  primitive.ObjectID  `json:"-" bson:"commenter_id"`                                // not exposed in API
	Text         string              `json:"text" bson:"text"`
	CreatedAt    time.Time           `json:"created_at" bson:"created_at"`
}