package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationType string

const (
	NotifTypeBudget      NotificationType = "budget"
	NotifTypeTransaction NotificationType = "transaction"
	NotifTypeBill        NotificationType = "bill"
	NotifTypeDebt        NotificationType = "debt"
	NotifTypeGoal        NotificationType = "goal"
	NotifTypeSecurity    NotificationType = "security"
	NotifTypeSystem      NotificationType = "system"
	NotifTypeRecurring   NotificationType = "recurring"
)

type UserNotification struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	Title     string             `json:"title" bson:"title"`
	Message   string             `json:"message" bson:"message"`
	Type      NotificationType   `json:"type" bson:"type"`
	IsRead    bool               `json:"is_read" bson:"is_read"`
	Link      string             `json:"link,omitempty" bson:"link,omitempty"` // Optional link to redirect user
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}
