package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BillReminder represents a bill reminder
type BillReminder struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"user_id" bson:"user_id"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description" bson:"description"`
	Amount      float64            `json:"amount" bson:"amount"`
	Currency    string             `json:"currency" bson:"currency"`
	Category    string             `json:"category" bson:"category"`
	PayTo       string             `json:"pay_to" bson:"pay_to"` // Company/service name

	// Billing cycle
	BillingCycle string `json:"billing_cycle" bson:"billing_cycle"` // monthly, quarterly, yearly
	DayOfMonth   int    `json:"day_of_month" bson:"day_of_month"`   // Due day (1-31)

	// Reminder settings
	RemindDaysBefore int   `json:"remind_days_before" bson:"remind_days_before"` // Days before to remind
	ReminderDays     []int `json:"reminder_days" bson:"reminder_days"`           // Specific days to remind

	// Status
	IsPaid       bool       `json:"is_paid" bson:"is_paid"`
	LastPaidDate *time.Time `json:"last_paid_date,omitempty" bson:"last_paid_date,omitempty"`
	NextDueDate  time.Time  `json:"next_due_date" bson:"next_due_date"`
	IsActive     bool       `json:"is_active" bson:"is_active"`

	// Auto-pay
	AutoPayEnabled bool               `json:"auto_pay_enabled" bson:"auto_pay_enabled"`
	AccountID      primitive.ObjectID `json:"account_id,omitempty" bson:"account_id,omitempty"`

	// Metadata
	Icon  string   `json:"icon" bson:"icon"`
	Color string   `json:"color" bson:"color"`
	Tags  []string `json:"tags" bson:"tags"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// GetDueStatus returns the due status of the bill
func (b *BillReminder) GetDueStatus() string {
	now := time.Now()
	daysUntilDue := int(b.NextDueDate.Sub(now).Hours() / 24)

	if b.IsPaid {
		return "paid"
	}

	if daysUntilDue < 0 {
		return "overdue"
	}

	if daysUntilDue <= b.RemindDaysBefore {
		return "due_soon"
	}

	return "upcoming"
}

// CalculateNextDueDate calculates the next due date
func (b *BillReminder) CalculateNextDueDate() time.Time {
	now := time.Now()
	next := b.NextDueDate

	for next.Before(now) || next.Equal(now) {
		switch b.BillingCycle {
		case "daily":
			next = next.AddDate(0, 0, 1)
		case "weekly":
			next = next.AddDate(0, 0, 7)
		case "monthly":
			next = next.AddDate(0, 1, 0)
		case "quarterly":
			next = next.AddDate(0, 3, 0)
		case "yearly":
			next = next.AddDate(1, 0, 0)
		default:
			next = next.AddDate(0, 1, 0)
		}

		// Handle months with fewer days
		day := b.DayOfMonth
		if day == 0 {
			day = 1
		}

		lastDay := time.Date(next.Year(), next.Month()+1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1).Day()
		if day > lastDay {
			day = lastDay
		}

		next = time.Date(next.Year(), next.Month(), day, 0, 0, 0, 0, time.UTC)
	}

	return next
}

// MarkAsPaid marks the bill as paid
func (b *BillReminder) MarkAsPaid() {
	now := time.Now()
	b.IsPaid = true
	b.LastPaidDate = &now
	b.NextDueDate = b.CalculateNextDueDate()
	b.UpdatedAt = now
}

// BillPaymentLog represents a log of bill payments
type BillPaymentLog struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	BillReminderID primitive.ObjectID `json:"bill_reminder_id" bson:"bill_reminder_id"`
	UserID         primitive.ObjectID `json:"user_id" bson:"user_id"`
	Amount         float64            `json:"amount" bson:"amount"`
	Currency       string             `json:"currency" bson:"currency"`
	PaidDate       time.Time          `json:"paid_date" bson:"paid_date"`
	Notes          string             `json:"notes" bson:"notes"`
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
}
