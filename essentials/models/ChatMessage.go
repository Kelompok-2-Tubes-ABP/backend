package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatMessage struct {
	ID          primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	UserID      string                 `bson:"user_id" json:"user_id"`
	SessionID   string                 `bson:"session_id" json:"session_id"`
	Message     string                 `bson:"message" json:"message"`
	Response    string                 `bson:"response" json:"response"`
	MessageType string                 `bson:"message_type" json:"message_type"` // "user" / "assistant"
	Timestamp   time.Time              `bson:"timestamp" json:"timestamp"`
	Metadata    map[string]interface{} `bson:"metadata" json:"metadata"`
}
