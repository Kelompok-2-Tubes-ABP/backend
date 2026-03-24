package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SpendingInsight represents an AI-generated spending insight
type SpendingInsight struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID      primitive.ObjectID `json:"user_id" bson:"user_id"`
	Type        string             `json:"type" bson:"type"` // "trend", "anomaly", "recommendation", "warning"
	Category    string             `json:"category" bson:"category"`
	Title       string             `json:"title" bson:"title"`
	Description string             `json:"description" bson:"description"`
	Metric      float64            `json:"metric" bson:"metric"`
	Percentage  float64            `json:"percentage" bson:"percentage"`
	Period      string             `json:"period" bson:"period"`           // "week", "month", "year"
	ComparedTo  string             `json:"compared_to" bson:"compared_to"` // vs previous period
	Priority    int                `json:"priority" bson:"priority"`       // 1=high, 2=medium, 3=low
	IsRead      bool               `json:"is_read" bson:"is_read"`
	IsActioned  bool               `json:"is_actioned" bson:"is_actioned"`

	// Related data
	RelatedCategories   []string             `json:"related_categories" bson:"related_categories"`
	RelatedTransactions []primitive.ObjectID `json:"related_transactions" bson:"related_transactions"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	ExpiresAt time.Time `json:"expires_at" bson:"expires_at"`
}

// InsightType constants
const (
	InsightTrend          = "trend"
	InsightAnomaly        = "anomaly"
	InsightRecommendation = "recommendation"
	InsightWarning        = "warning"
	InsightMilestone      = "milestone"
)

// SpendingPattern represents analyzed spending patterns
type SpendingPattern struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	Period    string             `json:"period" bson:"period"` // daily, weekly, monthly
	StartDate time.Time          `json:"start_date" bson:"start_date"`
	EndDate   time.Time          `json:"end_date" bson:"end_date"`

	// By category
	CategoryBreakdown map[string]float64 `json:"category_breakdown" bson:"category_breakdown"`

	// Statistics
	TotalSpent   float64 `json:"total_spent" bson:"total_spent"`
	TotalIncome  float64 `json:"total_income" bson:"total_income"`
	NetCashFlow  float64 `json:"net_cash_flow" bson:"net_cash_flow"`
	AverageDaily float64 `json:"average_daily" bson:"average_daily"`

	// Comparisons
	VsPrevious float64 `json:"vs_previous" bson:"vs_previous"` // Percentage change

	// Top categories
	TopExpenseCategories []CategoryStat `json:"top_expense_categories" bson:"top_expense_categories"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

// CategoryStat represents spending statistics for a category
type CategoryStat struct {
	Category         string  `json:"category" bson:"category"`
	Amount           float64 `json:"amount" bson:"amount"`
	Percentage       float64 `json:"percentage" bson:"percentage"`
	TransactionCount int     `json:"transaction_count" bson:"transaction_count"`
	AverageAmount    float64 `json:"average_amount" bson:"average_amount"`
}

// FinancialHealthScore represents overall financial health
type FinancialHealthScore struct {
	ID     primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID primitive.ObjectID `json:"user_id" bson:"user_id"`
	Period string             `json:"period" bson:"period"`

	// Scores (0-100)
	OverallScore    int `json:"overall_score" bson:"overall_score"`
	SavingsRate     int `json:"savings_rate" bson:"savings_rate"`
	DebtLevel       int `json:"debt_level" bson:"debt_level"`
	BudgetAdherence int `json:"budget_adherence" bson:"budget_adherence"`
	ExpenseControl  int `json:"expense_control" bson:"expense_control"`

	// Details
	SavingsRatePercent  float64 `json:"savings_rate_percent" bson:"savings_rate_percent"`
	DebtToIncomeRatio   float64 `json:"debt_to_income_ratio" bson:"debt_to_income_ratio"`
	EmergencyFundMonths float64 `json:"emergency_fund_months" bson:"emergency_fund_months"`

	// Recommendations
	Recommendations []string `json:"recommendations" bson:"recommendations"`

	CalculatedAt time.Time `json:"calculated_at" bson:"calculated_at"`
}
