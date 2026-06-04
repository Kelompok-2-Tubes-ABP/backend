package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MonthlyBudget struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    string             `bson:"user_id" json:"user_id"`
	Month     string             `bson:"month" json:"month"`
	Limit     float64            `bson:"limit" json:"limit"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// CategoryBudget represents a budget for a specific category
type CategoryBudget struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    string             `bson:"user_id" json:"user_id"`
	Month     string             `bson:"month" json:"month"`
	Category  string             `bson:"category" json:"category"`
	Limit     float64            `bson:"limit" json:"limit"`
	Spent     float64            `bson:"spent" json:"spent"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// CategoryBudgetStatus represents the status of a category budget
type CategoryBudgetStatus string

const (
	CategoryBudgetStatusSafe     CategoryBudgetStatus = "safe"
	CategoryBudgetStatusCaution  CategoryBudgetStatus = "caution"
	CategoryBudgetStatusWarning  CategoryBudgetStatus = "warning"
	CategoryBudgetStatusExceeded CategoryBudgetStatus = "exceeded"
)
