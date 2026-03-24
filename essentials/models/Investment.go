package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InvestmentType represents the type of investment
type InvestmentType string

const (
	InvStock      InvestmentType = "stock"
	InvCrypto     InvestmentType = "crypto"
	InvBond       InvestmentType = "bond"
	InvMutualFund InvestmentType = "mutual_fund"
	InvETF        InvestmentType = "etf"
	InvRealEstate InvestmentType = "real_estate"
	InvCommodity  InvestmentType = "commodity"
	InvOther      InvestmentType = "other"
)

// Investment represents an investment holding
type Investment struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID       primitive.ObjectID `json:"user_id" bson:"user_id"`
	Name         string             `json:"name" bson:"name"`
	Symbol       string             `json:"symbol" bson:"symbol"` // Stock symbol (AAPL), Crypto (BTC)
	Type         InvestmentType     `json:"type" bson:"type"`
	Quantity     float64            `json:"quantity" bson:"quantity"`
	AverageCost  float64            `json:"average_cost" bson:"average_cost"`
	CurrentPrice float64            `json:"current_price" bson:"current_price"`
	Currency     string             `json:"currency" bson:"currency"`
	Exchange     string             `json:"exchange" bson:"exchange"` // NYSE, NASDAQ, etc.

	// Calculated fields
	TotalValue      float64 `json:"total_value" bson:"total_value"`
	TotalCost       float64 `json:"total_cost" bson:"total_cost"`
	GainLoss        float64 `json:"gain_loss" bson:"gain_loss"`
	GainLossPercent float64 `json:"gain_loss_percent" bson:"gain_loss_percent"`

	// Dates
	PurchaseDate    time.Time `json:"purchase_date" bson:"purchase_date"`
	LastPriceUpdate time.Time `json:"last_price_update" bson:"last_price_update"`

	// Status
	IsActive bool     `json:"is_active" bson:"is_active"`
	Notes    string   `json:"notes" bson:"notes"`
	Tags     []string `json:"tags" bson:"tags"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// CalculateValues calculates total value, gain/loss
func (i *Investment) CalculateValues() {
	i.TotalCost = i.Quantity * i.AverageCost
	i.TotalValue = i.Quantity * i.CurrentPrice
	i.GainLoss = i.TotalValue - i.TotalCost
	if i.TotalCost > 0 {
		i.GainLossPercent = (i.GainLoss / i.TotalCost) * 100
	}
}

// InvestmentTransaction represents a buy/sell transaction
type InvestmentTransaction struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	InvestmentID primitive.ObjectID `json:"investment_id" bson:"investment_id"`
	UserID       primitive.ObjectID `json:"user_id" bson:"user_id"`
	Type         string             `json:"type" bson:"type"` // "buy", "sell", "dividend"
	Quantity     float64            `json:"quantity" bson:"quantity"`
	Price        float64            `json:"price" bson:"price"`
	TotalAmount  float64            `json:"total_amount" bson:"total_amount"`
	Fee          float64            `json:"fee" bson:"fee"`
	Currency     string             `json:"currency" bson:"currency"`
	Date         time.Time          `json:"date" bson:"date"`
	Notes        string             `json:"notes" bson:"notes"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
}

// InvestmentPortfolio represents a collection of investments
type InvestmentPortfolio struct {
	ID            primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	UserID        primitive.ObjectID   `json:"user_id" bson:"user_id"`
	Name          string               `json:"name" bson:"name"`
	Description   string               `json:"description" bson:"description"`
	InvestmentIDs []primitive.ObjectID `json:"investment_ids" bson:"investment_ids"`

	// Summary
	TotalValue    float64 `json:"total_value" bson:"total_value"`
	TotalCost     float64 `json:"total_cost" bson:"total_cost"`
	TotalGainLoss float64 `json:"total_gain_loss" bson:"total_gain_loss"`

	// Allocation
	Allocation map[string]float64 `json:"allocation" bson:"allocation"` // By type

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// PriceHistory represents historical price data
type PriceHistory struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	InvestmentID primitive.ObjectID `json:"investment_id" bson:"investment_id"`
	Symbol       string             `json:"symbol" bson:"symbol"`
	Price        float64            `json:"price" bson:"price"`
	Date         time.Time          `json:"date" bson:"date"`
	Currency     string             `json:"currency" bson:"currency"`
}
