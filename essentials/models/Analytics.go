package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ============================================================
// ANALYTICS MODELS
// ============================================================

// AnalyticsReport represents a comprehensive financial analytics report
type AnalyticsReport struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	Period    string             `json:"period"` // "daily", "weekly", "monthly", "yearly"
	StartDate time.Time          `json:"start_date" bson:"start_date"`
	EndDate   time.Time          `json:"end_date" bson:"end_date"`

	// Summary
	NetWorth         float64         `json:"net_worth" bson:"net_worth"`
	TotalAssets      float64         `json:"total_assets" bson:"total_assets"`
	TotalLiabilities float64         `json:"total_liabilities" bson:"total_liabilities"`
	CashFlow         CashFlowSummary `json:"cash_flow" bson:"cash_flow"`

	// Breakdown
	AssetAllocation  []AllocationItem `json:"asset_allocation" bson:"asset_allocation"`
	ExpenseBreakdown []CategoryStat   `json:"expense_breakdown" bson:"expense_breakdown"`
	IncomeSources    []CategoryStat   `json:"income_sources" bson:"income_sources"`
	DebtBreakdown    []DebtSummary    `json:"debt_breakdown" bson:"debt_breakdown"`

	// Metrics
	FinancialHealth *FinancialHealthScore `json:"financial_health" bson:"financial_health"`
	SavingsRate     float64               `json:"savings_rate" bson:"savings_rate"`
	ExpenseToIncome float64               `json:"expense_to_income" bson:"expense_to_income"`

	// Trends (optional - for longer periods)
	TrendComparison *TrendComparison `json:"trend_comparison,omitempty" bson:"trend_comparison,omitempty"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

// CashFlowSummary represents cash flow summary
type CashFlowSummary struct {
	TotalIncome    float64 `json:"total_income" bson:"total_income"`
	TotalExpenses  float64 `json:"total_expenses" bson:"total_expenses"`
	NetCashFlow    float64 `json:"net_cash_flow" bson:"net_cash_flow"`
	AverageDaily   float64 `json:"average_daily" bson:"average_daily"`
	AverageMonthly float64 `json:"average_monthly" bson:"average_monthly"`
}

// AllocationItem represents an asset allocation breakdown
type AllocationItem struct {
	Category    string  `json:"category" bson:"category"` // "crypto", "stocks", "cash", "savings", "property"
	SubCategory string  `json:"sub_category,omitempty" bson:"sub_category,omitempty"`
	Amount      float64 `json:"amount" bson:"amount"`
	Percentage  float64 `json:"percentage" bson:"percentage"`
	Count       int     `json:"count,omitempty" bson:"count,omitempty"` // number of items
}

// DebtSummary represents debt breakdown
type DebtSummary struct {
	Type            string  `json:"type" bson:"type"` // "loan", "credit_card", "mortgage", etc
	Name            string  `json:"name" bson:"name"`
	Original        float64 `json:"original" bson:"original"`
	Current         float64 `json:"current" bson:"current"`
	MonthlyPayment  float64 `json:"monthly_payment" bson:"monthly_payment"`
	InterestRate    float64 `json:"interest_rate" bson:"interest_rate"`
	RemainingMonths int     `json:"remaining_months,omitempty" bson:"remaining_months,omitempty"`
}

// TrendComparison represents period-over-period comparison
type TrendComparison struct {
	Period            string             `json:"period" bson:"period"` // "month_over_month", "year_over_year"
	IncomeChange      float64            `json:"income_change" bson:"income_change"`
	ExpenseChange     float64            `json:"expense_change" bson:"expense_change"`
	NetWorthChange    float64            `json:"net_worth_change" bson:"net_worth_change"`
	ExpenseCategories map[string]float64 `json:"expense_categories" bson:"expense_categories"`
}

// MonthlySnapshot represents a monthly financial snapshot
type MonthlySnapshot struct {
	ID     primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID primitive.ObjectID `json:"user_id" bson:"user_id"`
	Year   int                `json:"year" bson:"year"`
	Month  int                `json:"month" bson:"month"` // 1-12

	// Financial position at end of month
	NetWorth         float64 `json:"net_worth" bson:"net_worth"`
	TotalAssets      float64 `json:"total_assets" bson:"total_assets"`
	TotalLiabilities float64 `json:"total_liabilities" bson:"total_liabilities"`

	// Flow for the month
	Income   float64 `json:"income" bson:"income"`
	Expenses float64 `json:"expenses" bson:"expenses"`
	Savings  float64 `json:"savings" bson:"savings"`

	// Metrics
	SavingsRate     float64 `json:"savings_rate" bson:"savings_rate"`
	FinancialHealth int     `json:"financial_health" bson:"financial_health"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

// GoalProgress represents progress toward a financial goal
type GoalProgress struct {
	GoalID              primitive.ObjectID `json:"goal_id" bson:"goal_id"`
	GoalName            string             `json:"goal_name" bson:"goal_name"`
	TargetAmount        float64            `json:"target_amount" bson:"target_amount"`
	CurrentAmount       float64            `json:"current_amount" bson:"current_amount"`
	RemainingAmount     float64            `json:"remaining_amount" bson:"remaining_amount"`
	Percentage          float64            `json:"percentage" bson:"percentage"`
	TargetDate          time.Time          `json:"target_date,omitempty" bson:"target_date,omitempty"`
	EstimatedCompletion time.Time          `json:"estimated_completion,omitempty" bson:"estimated_completion,omitempty"`
	MonthlyNeeded       float64            `json:"monthly_needed" bson:"monthly_needed"`
}

// QuickStats represents quick summary stats for dashboard
type QuickStats struct {
	TodaySpending   float64          `json:"today_spending" bson:"today_spending"`
	WeekSpending    float64          `json:"week_spending" bson:"week_spending"`
	MonthSpending   float64          `json:"month_spending" bson:"month_spending"`
	TodayIncome     float64          `json:"today_income" bson:"today_income"`
	WeekIncome      float64          `json:"week_income" bson:"week_income"`
	MonthIncome     float64          `json:"month_income" bson:"month_income"`
	MonthSavings    float64          `json:"month_savings" bson:"month_savings"`
	ActiveBills     int              `json:"active_bills" bson:"active_bills"`
	UpcomingBills   float64          `json:"upcoming_bills" bson:"upcoming_bills"`
	InvestmentValue float64          `json:"investment_value" bson:"investment_value"`
	NetWorth        float64          `json:"net_worth" bson:"net_worth"`
	TopExpenses     []CategoryStat   `json:"top_expenses" bson:"top_expenses"` // Top expense categories
}
