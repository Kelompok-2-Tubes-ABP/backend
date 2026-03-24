package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DebtType represents the type of debt
type DebtType string

const (
	DebtCreditCard DebtType = "credit_card"
	DebtLoan       DebtType = "loan"
	DebtMortgage   DebtType = "mortgage"
	DebtPersonal   DebtType = "personal"
	DebtStudent    DebtType = "student"
	DebtOther      DebtType = "other"
)

// RepaymentFrequency represents how often payments are made
type RepaymentFrequency string

const (
	RepayWeekly   RepaymentFrequency = "weekly"
	RepayBiweekly RepaymentFrequency = "biweekly"
	RepayMonthly  RepaymentFrequency = "monthly"
	RepayAnnually RepaymentFrequency = "annually"
)

// Debt represents a debt/loan
type Debt struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID   primitive.ObjectID `json:"user_id" bson:"user_id"`
	Name     string             `json:"name" bson:"name"`
	Type     DebtType           `json:"type" bson:"type"`
	Creditor string             `json:"creditor" bson:"creditor"` // Bank or creditor name
	Currency string             `json:"currency" bson:"currency"`

	// Original terms
	OriginalAmount float64 `json:"original_amount" bson:"original_amount"`
	CurrentBalance float64 `json:"current_balance" bson:"current_balance"`
	InterestRate   float64 `json:"interest_rate" bson:"interest_rate"` // Annual percentage rate
	APRType        string  `json:"apr_type" bson:"apr_type"`           // "fixed", "variable"

	// Repayment terms
	PaymentAmount    float64            `json:"payment_amount" bson:"payment_amount"`
	PaymentFrequency RepaymentFrequency `json:"payment_frequency" bson:"payment_frequency"`
	MinimumPayment   float64            `json:"minimum_payment" bson:"minimum_payment"`
	NextPaymentDate  time.Time          `json:"next_payment_date" bson:"next_payment_date"`

	// Dates
	StartDate time.Time  `json:"start_date" bson:"start_date"`
	EndDate   *time.Time `json:"end_date,omitempty" bson:"end_date,omitempty"`

	// Status
	IsActive  bool `json:"is_active" bson:"is_active"`
	IsPaidOff bool `json:"is_paid_off" bson:"is_paid_off"`

	// Calculated
	TotalPaid         float64 `json:"total_paid" bson:"total_paid"`
	TotalInterest     float64 `json:"total_interest" bson:"total_interest"`
	RemainingPayments int     `json:"remaining_payments" bson:"remaining_payments"`

	// Metadata
	AccountID primitive.ObjectID `json:"account_id,omitempty" bson:"account_id,omitempty"` // Linked account
	Notes     string             `json:"notes" bson:"notes"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// CalculatePayoffDate calculates estimated payoff date
func (d *Debt) CalculatePayoffDate() time.Time {
	if d.PaymentAmount <= d.InterestRate*d.CurrentBalance/12 {
		return time.Time{} // Will never be paid off
	}

	monthlyRate := d.InterestRate / 12 / 100
	remaining := d.CurrentBalance
	payment := d.PaymentAmount

	months := 0
	for remaining > 0 && months < 1200 { // Cap at 100 years
		interest := remaining * monthlyRate
		principal := payment - interest
		if principal > remaining {
			principal = remaining
		}
		remaining -= principal
		months++
	}

	return d.StartDate.AddDate(0, months, 0)
}

// CalculateTotalInterest calculates total interest to be paid
func (d *Debt) CalculateTotalInterest() float64 {
	monthlyRate := d.InterestRate / 12 / 100
	payment := d.PaymentAmount
	remaining := d.CurrentBalance

	totalInterest := 0.0
	for remaining > 0 && remaining < 10000000 { // Safety limit
		interest := remaining * monthlyRate
		totalInterest += interest
		principal := payment - interest
		if principal <= 0 {
			break
		}
		remaining -= principal
	}

	return totalInterest
}

// DebtPayment represents a debt payment
type DebtPayment struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	DebtID    primitive.ObjectID `json:"debt_id" bson:"debt_id"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	Amount    float64            `json:"amount" bson:"amount"`
	Principal float64            `json:"principal" bson:"principal"`
	Interest  float64            `json:"interest" bson:"interest"`
	Date      time.Time          `json:"date" bson:"date"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}

// DebtSnowballPlan represents a debt snowball repayment plan
type DebtSnowballPlan struct {
	ID           primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	UserID       primitive.ObjectID   `json:"user_id" bson:"user_id"`
	DebtIDs      []primitive.ObjectID `json:"debt_ids" bson:"debt_ids"`
	ExtraPayment float64              `json:"extra_payment" bson:"extra_payment"`
	OrderType    string               `json:"order_type" bson:"order_type"` // "snowball" (smallest first) or "avalanche" (highest interest first)

	// Summary
	TotalDebt      float64   `json:"total_debt" bson:"total_debt"`
	MonthlyPayment float64   `json:"monthly_payment" bson:"monthly_payment"`
	PayoffDate     time.Time `json:"payoff_date" bson:"payoff_date"`
	TotalInterest  float64   `json:"total_interest" bson:"total_interest"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}
