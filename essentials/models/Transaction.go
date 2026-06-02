package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Transaction struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	User_id     string             `bson:"user_id" json:"user_id"`
	Amount      float64            `bson:"amount" json:"amount"`
	Category    string             `bson:"category" json:"category"`
	Description string             `bson:"description" json:"description"`
	Date        time.Time          `bson:"date" json:"date"`
	Month       string             `bson:"month" json:"month"`
	Status      string             `bson:"status" json:"status"`
	Type        string             `bson:"type" json:"type"` // "income" or "outcome"
}
