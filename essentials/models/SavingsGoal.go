package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SavingsGoal represents a savings goal
type SavingsGoal struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID        primitive.ObjectID `json:"user_id" bson:"user_id"`
	Name          string             `json:"name" bson:"name"`
	Description   string             `json:"description" bson:"description"`
	TargetAmount  float64            `json:"target_amount" bson:"target_amount"`
	CurrentAmount float64            `json:"current_amount" bson:"current_amount"`
	StartDate     time.Time          `json:"start_date" bson:"start_date"`
	TargetDate    time.Time          `json:"target_date" bson:"target_date"`
	Category      string             `json:"category" bson:"category"`
	Priority      int                `json:"priority" bson:"priority"` // 1 = high, 2 = medium, 3 = low
	Status        string             `json:"status" bson:"status"`     // active, completed, paused, cancelled
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at" bson:"updated_at"`
}

// CalculateProgress returns the percentage of the goal achieved
func (sg *SavingsGoal) CalculateProgress() float64 {
	if sg.TargetAmount <= 0 {
		return 0
	}
	return (sg.CurrentAmount / sg.TargetAmount) * 100
}

// CalculateMonthlyNeeded returns how much needs to be saved monthly to reach the goal
func (sg *SavingsGoal) CalculateMonthlyNeeded() float64 {
	now := time.Now()
	monthsRemaining := int(sg.TargetDate.Sub(now).Hours() / 24 / 30)

	if monthsRemaining <= 0 {
		return 0
	}

	remaining := sg.TargetAmount - sg.CurrentAmount
	if remaining <= 0 {
		return 0
	}

	return remaining / float64(monthsRemaining)
}

// IsOnTrack returns true if the user is on track to reach the goal
func (sg *SavingsGoal) IsOnTrack() bool {
	now := time.Now()
	totalDays := sg.TargetDate.Sub(sg.StartDate).Hours() / 24
	elapsedDays := now.Sub(sg.StartDate).Hours() / 24

	if totalDays <= 0 || elapsedDays < 0 {
		return false
	}

	expectedProgress := (elapsedDays / totalDays) * 100
	actualProgress := sg.CalculateProgress()

	// Allow 10% tolerance
	return actualProgress >= expectedProgress-10
}

// SavingsContribution represents a contribution to a savings goal
type SavingsContribution struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	SavingsGoalID primitive.ObjectID `json:"savings_goal_id" bson:"savings_goal_id"`
	UserID        primitive.ObjectID `json:"user_id" bson:"user_id"`
	Amount        float64            `json:"amount" bson:"amount"`
	Date          time.Time          `json:"date" bson:"date"`
	Note          string             `json:"note" bson:"note"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
}
