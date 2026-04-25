package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Admin struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username          string             `bson:"username" json:"username"`
	Email             string             `bson:"email" json:"email"`
	Password          string             `bson:"password" json:"password"`
	IsActive          bool               `bson:"is_active" json:"is_active"`
	PasswordChangedAt time.Time          `bson:"password_changed_at" json:"password_changed_at"`
	CreatedAt         time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt         time.Time          `bson:"updated_at" json:"updated_at"`
}
