package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuditLog struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Timestamp  time.Time          `bson:"timestamp" json:"timestamp"`
	ActorID    string             `bson:"actor_id" json:"actor_id"`
	ActorName  string             `bson:"actor_name" json:"actor_name"`
	ActorRole  string             `bson:"actor_role" json:"actor_role"`   // "admin" or "user"
	ActionType string             `bson:"action_type" json:"action_type"` // "Create", "Update", "Delete"
	TargetData string             `bson:"target_data" json:"target_data"`
	IPAddress  string             `bson:"ip_address" json:"ip_address"`
	Details    string             `bson:"details" json:"details"`
}
