package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RecurrenceFrequency represents how often a transaction recurs
type RecurrenceFrequency string

const (
	FreqDaily     RecurrenceFrequency = "daily"
	FreqWeekly    RecurrenceFrequency = "weekly"
	FreqBiweekly  RecurrenceFrequency = "biweekly"
	FreqMonthly   RecurrenceFrequency = "monthly"
	FreqQuarterly RecurrenceFrequency = "quarterly"
	FreqYearly    RecurrenceFrequency = "yearly"
)

// RecurringTransaction represents a recurring transaction template
type RecurringTransaction struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"user_id" bson:"user_id"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description" bson:"description"`
	Amount      float64            `json:"amount" bson:"amount"`
	Currency    string             `json:"currency" bson:"currency"`
	Category    string             `json:"category" bson:"category"`
	Type        string             `json:"type" bson:"type"`             // "income" or "expense"
	AccountID   primitive.ObjectID `json:"account_id" bson:"account_id"` // Account to use

	// Recurrence settings
	Frequency   RecurrenceFrequency `json:"frequency" bson:"frequency"`
	Interval    int                 `json:"interval" bson:"interval"` // For custom intervals (e.g., every 2 weeks)
	StartDate   time.Time           `json:"start_date" bson:"start_date"`
	EndDate     *time.Time          `json:"end_date,omitempty" bson:"end_date,omitempty"`
	NextRunDate time.Time           `json:"next_run_date" bson:"next_run_date"`
	DayOfMonth  int                 `json:"day_of_month" bson:"day_of_month"` // For monthly (1-31)
	DayOfWeek   int                 `json:"day_of_week" bson:"day_of_week"`   // 0=Sunday, 6=Saturday

	// Status
	IsActive     bool `json:"is_active" bson:"is_active"`
	AutoCreate   bool `json:"auto_create" bson:"auto_create"`     // Automatically create transactions
	SkipWeekends bool `json:"skip_weekends" bson:"skip_weekends"` // Skip weekends for monthly

	// Metadata
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// GeneratedTransaction represents a transaction generated from a recurring template
type GeneratedTransaction struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	RecurringID   primitive.ObjectID `json:"recurring_id" bson:"recurring_id"`
	UserID        primitive.ObjectID `json:"user_id" bson:"user_id"`
	GeneratedFrom string             `json:"generated_from" bson:"generated_from"` // Name of recurring template

	// Transaction details
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description" bson:"description"`
	Amount      float64            `json:"amount" bson:"amount"`
	Currency    string             `json:"currency" bson:"currency"`
	Category    string             `json:"category" bson:"category"`
	Type        string             `json:"type" bson:"type"`
	AccountID   primitive.ObjectID `json:"account_id" bson:"account_id"`

	// Dates
	TransactionDate time.Time `json:"transaction_date" bson:"transaction_date"`
	CreatedDate     time.Time `json:"created_date" bson:"created_date"`

	Status string `json:"status" bson:"status"` // "pending", "posted", "cancelled"
}

// CalculateNextRunDate calculates the next run date based on frequency
func (r *RecurringTransaction) CalculateNextRunDate() time.Time {
	now := time.Now()
	next := r.NextRunDate

	// If next run is in the past, calculate from now
	if next.Before(now) {
		next = now
	}

	switch r.Frequency {
	case FreqDaily:
		next = next.AddDate(0, 0, 1)
	case FreqWeekly:
		next = next.AddDate(0, 0, 7)
	case FreqBiweekly:
		next = next.AddDate(0, 0, 14)
	case FreqMonthly:
		// Move to next month
		nextMonth := next.Month() + 1
		year := next.Year()
		if nextMonth > 12 {
			nextMonth = 1
			year++
		}

		// Handle months with fewer days
		day := r.DayOfMonth
		if day == 0 {
			day = next.Day()
		}

		// Get the last day of the target month
		lastDay := time.Date(year, nextMonth+1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1).Day()
		if day > lastDay {
			day = lastDay
		}

		next = time.Date(year, nextMonth, day, next.Hour(), next.Minute(), next.Second(), 0, time.UTC)
	case FreqQuarterly:
		next = next.AddDate(0, 3, 0)
	case FreqYearly:
		next = next.AddDate(1, 0, 0)
	}

	// Handle skip weekends
	if r.SkipWeekends && next.Weekday() == time.Saturday {
		next = next.AddDate(0, 0, 2)
	} else if r.SkipWeekends && next.Weekday() == time.Sunday {
		next = next.AddDate(0, 0, 1)
	}

	return next
}

// IsDue checks if the recurring transaction is due
func (r *RecurringTransaction) IsDue() bool {
	if !r.IsActive {
		return false
	}

	now := time.Now()
	return !r.NextRunDate.After(now)
}

// GetOccurrences returns the number of times this has occurred
func (r *RecurringTransaction) GetOccurrences() int {
	if r.StartDate.IsZero() {
		return 0
	}

	totalDays := time.Since(r.StartDate).Hours() / 24
	var interval int

	switch r.Frequency {
	case FreqDaily:
		interval = 1
	case FreqWeekly:
		interval = 7
	case FreqBiweekly:
		interval = 14
	case FreqMonthly:
		interval = 30
	case FreqQuarterly:
		interval = 90
	case FreqYearly:
		interval = 365
	default:
		return 0
	}

	return int(totalDays / float64(interval))
}
