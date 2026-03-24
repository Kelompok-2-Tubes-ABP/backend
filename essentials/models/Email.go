package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EmailTemplate represents an email template
type EmailTemplate struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Subject     string             `json:"subject" bson:"subject"`
	Body        string             `json:"body" bson:"body"`
	Type        EmailType          `json:"type" bson:"type"`
	HTMLContent string             `json:"html_content" bson:"html_content"`
	IsActive    bool               `json:"is_active" bson:"is_active"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

// EmailType represents the type of email
type EmailType string

const (
	EmailTypeBillReminder  EmailType = "bill_reminder"
	EmailTypeDebtDue       EmailType = "debt_due"
	EmailTypeTransaction   EmailType = "transaction_alert"
	EmailTypeBudgetWarning EmailType = "budget_warning"
	EmailTypeGoalReached   EmailType = "goal_reached"
	EmailTypeWeeklyReport  EmailType = "weekly_report"
	EmailTypeMonthlyReport EmailType = "monthly_report"
	EmailTypeSecurityAlert EmailType = "security_alert"
	EmailTypeVerification  EmailType = "email_verification"
	EmailTypePasswordReset EmailType = "password_reset"
	EmailTypeCustom        EmailType = "custom"
)

// EmailLog represents a sent email log
type EmailLog struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID       primitive.ObjectID `json:"user_id" bson:"user_id"`
	Recipient    string             `json:"recipient" bson:"recipient"`
	Subject      string             `json:"subject" bson:"subject"`
	Type         EmailType          `json:"type" bson:"type"`
	Status       string             `json:"status" bson:"status"` // "sent", "failed", "pending"
	ErrorMessage string             `json:"error_message,omitempty" bson:"error_message,omitempty"`
	SentAt       time.Time          `json:"sent_at" bson:"sent_at"`
	OpenedAt     *time.Time         `json:"opened_at,omitempty" bson:"opened_at,omitempty"`
}

// UserNotificationPreference represents user's notification settings
type UserNotificationPreference struct {
	UserID        primitive.ObjectID `json:"user_id" bson:"user_id"`
	EmailEnabled  bool               `json:"email_enabled" bson:"email_enabled"`
	Email         string             `json:"email" bson:"email"`
	EmailVerified bool               `json:"email_verified" bson:"email_verified"`

	// Notification types
	BillReminders     bool `json:"bill_reminders" bson:"bill_reminders"`
	DebtAlerts        bool `json:"debt_alerts" bson:"debt_alerts"`
	TransactionAlerts bool `json:"transaction_alerts" bson:"transaction_alerts"`
	BudgetWarnings    bool `json:"budget_warnings" bson:"budget_warnings"`
	GoalUpdates       bool `json:"goal_updates" bson:"goal_updates"`
	WeeklyReports     bool `json:"weekly_reports" bson:"weekly_reports"`
	MonthlyReports    bool `json:"monthly_reports" bson:"monthly_reports"`
	SecurityAlerts    bool `json:"security_alerts" bson:"security_alerts"`

	// Timing
	DailyDigest bool   `json:"daily_digest" bson:"daily_digest"`
	DigestTime  string `json:"digest_time" bson:"digest_time"` // "08:00"

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// EmailQueueItem represents an email in the queue
type EmailQueueItem struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"user_id" bson:"user_id"`
	Recipient   string             `json:"recipient" bson:"recipient"`
	Subject     string             `json:"subject" bson:"subject"`
	Body        string             `json:"body" bson:"body"`
	HTMLBody    string             `json:"html_body" bson:"html_body"`
	Type        EmailType          `json:"type" bson:"type"`
	Priority    int                `json:"priority" bson:"priority"` // 1=high, 2=normal, 3=low
	ScheduledAt time.Time          `json:"scheduled_at" bson:"scheduled_at"`
	Retries     int                `json:"retries" bson:"retries"`
	MaxRetries  int                `json:"max_retries" bson:"max_retries"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
}

// Webhook represents an external webhook URL
type Webhook struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID        primitive.ObjectID `json:"user_id" bson:"user_id"`
	Name          string             `json:"name" bson:"name"`
	URL           string             `json:"url" bson:"url"`
	Secret        string             `json:"secret" bson:"secret"`
	Events        []string           `json:"events" bson:"events"` // "bill_reminder", "transaction", etc.
	IsActive      bool               `json:"is_active" bson:"is_active"`
	LastTriggered *time.Time         `json:"last_triggered,omitempty" bson:"last_triggered,omitempty"`
	SuccessCount  int                `json:"success_count" bson:"success_count"`
	FailCount     int                `json:"fail_count" bson:"fail_count"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at" bson:"updated_at"`
}

// WebhookLog represents a webhook trigger log
type WebhookLog struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	WebhookID   primitive.ObjectID `json:"webhook_id" bson:"webhook_id"`
	Event       string             `json:"event" bson:"event"`
	Payload     string             `json:"payload" bson:"payload"`
	StatusCode  int                `json:"status_code" bson:"status_code"`
	Success     bool               `json:"success" bson:"success"`
	Response    string             `json:"response,omitempty" bson:"response,omitempty"`
	TriggeredAt time.Time          `json:"triggered_at" bson:"triggered_at"`
}

// NotificationEvent represents events that can trigger notifications
type NotificationEvent struct {
	Type      string                 `json:"type" bson:"type"`
	UserID    primitive.ObjectID     `json:"user_id" bson:"user_id"`
	Data      map[string]interface{} `json:"data" bson:"data"`
	Timestamp time.Time              `json:"timestamp" bson:"timestamp"`
}
