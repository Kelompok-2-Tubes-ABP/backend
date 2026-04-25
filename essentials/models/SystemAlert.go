package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SystemAlert struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	Severity    string             `bson:"severity" json:"severity"` // "Warning", "Error", "Info", "Success"
	Type        string             `bson:"type" json:"type"`         // "system_alert" or "failed_action"
	IsRead      bool               `bson:"is_read" json:"is_read"`
	CanRetry    bool               `bson:"can_retry" json:"can_retry"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}
