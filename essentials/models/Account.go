package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AccountType represents the type of account
type AccountType string

const (
	AccountTypeBank       AccountType = "bank"
	AccountTypeWallet     AccountType = "wallet"
	AccountTypeCash       AccountType = "cash"
	AccountTypeCredit     AccountType = "credit"
	AccountTypeSavings    AccountType = "savings"
	AccountTypeInvestment AccountType = "investment"
)

// Account represents a bank account or wallet
type Account struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID          primitive.ObjectID `json:"user_id" bson:"user_id"`
	Name            string             `json:"name" bson:"name"`
	Type            AccountType        `json:"type" bson:"type"`
	Institution     string             `json:"institution" bson:"institution"` // Bank name or wallet provider
	AccountNumber   string             `json:"account_number" bson:"account_number"`
	Currency        string             `json:"currency" bson:"currency"` // Currency code (IDR, USD, etc.)
	CurrentBalance  float64            `json:"current_balance" bson:"current_balance"`
	InitialBalance  float64            `json:"initial_balance" bson:"initial_balance"`
	AvailableCredit float64            `json:"available_credit" bson:"available_credit"` // For credit cards
	InterestRate    float64            `json:"interest_rate" bson:"interest_rate"`       // Annual interest rate
	IsActive        bool               `json:"is_active" bson:"is_active"`
	IsDefault       bool               `json:"is_default" bson:"is_default"` // Default account for transactions
	LastSynced      time.Time          `json:"last_synced" bson:"last_synced"`
	Notes           string             `json:"notes" bson:"notes"`
	Icon            string             `json:"icon" bson:"icon"`   // Icon identifier
	Color           string             `json:"color" bson:"color"` // Account color
	CreatedAt       time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" bson:"updated_at"`
}

// CalculateTotalAssets returns the sum of all positive balances
func (a *Account) CalculateTotalAssets() float64 {
	if a.Type == AccountTypeCredit {
		// For credit cards, the debt is the negative asset
		return -a.CurrentBalance
	}
	return a.CurrentBalance
}

// CalculateDebt returns the total debt for credit accounts
func (a *Account) CalculateDebt() float64 {
	if a.Type == AccountTypeCredit {
		return a.CurrentBalance
	}
	return 0
}

// AccountTransaction represents a transfer between accounts
type AccountTransaction struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID          primitive.ObjectID `json:"user_id" bson:"user_id"`
	FromAccountID   primitive.ObjectID `json:"from_account_id" bson:"from_account_id"`
	ToAccountID     primitive.ObjectID `json:"to_account_id" bson:"to_account_id"`
	Amount          float64            `json:"amount" bson:"amount"`
	Currency        string             `json:"currency" bson:"currency"`
	Note            string             `json:"note" bson:"note"`
	TransactionDate time.Time          `json:"transaction_date" bson:"transaction_date"`
	CreatedAt       time.Time          `json:"created_at" bson:"created_at"`
}

// AccountSyncLog represents a log of account synchronization
type AccountSyncLog struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	AccountID primitive.ObjectID `json:"account_id" bson:"account_id"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	SyncType  string             `json:"sync_type" bson:"sync_type"` // "manual", "auto", "scheduled"
	Status    string             `json:"status" bson:"status"`       // "success", "failed", "partial"
	Message   string             `json:"message" bson:"message"`
	Changes   int                `json:"changes" bson:"changes"` // Number of transactions synced
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}

// AccountGroup represents a group of accounts (e.g., "My Banks")
type AccountGroup struct {
	ID         primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	UserID     primitive.ObjectID   `json:"user_id" bson:"user_id"`
	Name       string               `json:"name" bson:"name"`
	AccountIDs []primitive.ObjectID `json:"account_ids" bson:"account_ids"`
	Icon       string               `json:"icon" bson:"icon"`
	Color      string               `json:"color" bson:"color"`
	CreatedAt  time.Time            `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at" bson:"updated_at"`
}
