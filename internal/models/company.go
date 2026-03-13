package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Company categories
var CompanyCategories = []string{
	"technology",
	"finance",
	"healthcare",
	"education",
	"retail",
	"manufacturing",
	"construction",
	"real_estate",
	"transportation",
	"food_beverage",
	"entertainment",
	"marketing",
	"consulting",
	"agriculture",
	"energy",
	"other",
}

type Company struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"user_id" bson:"user_id"`
	Name        string             `json:"name" bson:"name"`
	Category    string             `json:"category" bson:"category"`
	Position    string             `json:"position,omitempty" bson:"position,omitempty"` // job title, e.g. "Software Engineer", "CEO"
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	Website     string             `json:"website,omitempty" bson:"website,omitempty"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}
