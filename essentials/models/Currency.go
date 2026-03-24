package models

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Currency represents a currency with its properties
type Currency struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Code        string             `json:"code" bson:"code"`                 // ISO 4217 code (IDR, USD, EUR, etc.)
	Name        string             `json:"name" bson:"name"`                 // Currency name
	Symbol      string             `json:"symbol" bson:"symbol"`             // Currency symbol (Rp, $, €, etc.)
	Decimal     int                `json:"decimal" bson:"decimal"`           // Decimal places (usually 0 or 2)
	ThousandSep string             `json:"thousand_sep" bson:"thousand_sep"` // Thousand separator
	DecimalSep  string             `json:"decimal_sep" bson:"decimal_sep"`   // Decimal separator
	IsDefault   bool               `json:"is_default" bson:"is_default"`     // Default currency for user
	IsActive    bool               `json:"is_active" bson:"is_active"`       // Currency is available
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

// ExchangeRate represents the exchange rate between two currencies
type ExchangeRate struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	FromCurrency   string             `json:"from_currency" bson:"from_currency"`     // Source currency code
	ToCurrency     string             `json:"to_currency" bson:"to_currency"`         // Target currency code
	Rate           float64            `json:"rate" bson:"rate"`                       // Exchange rate (1 From = Rate To)
	Source         string             `json:"source" bson:"source"`                   // Where we got the rate (e.g., "bni", "bi", "manual")
	LastUpdated    time.Time          `json:"last_updated" bson:"last_updated"`       // When rate was last updated
	ExpirationTime time.Time          `json:"expiration_time" bson:"expiration_time"` // When rate expires
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
}

// CurrencyConversion represents a currency conversion request/result
type CurrencyConversion struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID          primitive.ObjectID `json:"user_id" bson:"user_id"`
	FromCurrency    string             `json:"from_currency" bson:"from_currency"`
	ToCurrency      string             `json:"to_currency" bson:"to_currency"`
	OriginalAmount  float64            `json:"original_amount" bson:"original_amount"`
	ConvertedAmount float64            `json:"converted_amount" bson:"converted_amount"`
	Rate            float64            `json:"rate" bson:"rate"`
	RateID          primitive.ObjectID `json:"rate_id" bson:"rate_id"` // Reference to ExchangeRate used
	CreatedAt       time.Time          `json:"created_at" bson:"created_at"`
}

// UserCurrency represents a user's currency preferences
type UserCurrency struct {
	UserID      primitive.ObjectID `json:"user_id" bson:"user_id"`
	DefaultCode string             `json:"default_code" bson:"default_code"` // Default currency code (e.g., "IDR")
	Currencies  []string           `json:"currencies" bson:"currencies"`     // List of currency codes user uses
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

// SupportedCurrencies returns a list of commonly used currencies
func SupportedCurrencies() []Currency {
	return []Currency{
		{Code: "IDR", Name: "Indonesian Rupiah", Symbol: "Rp", Decimal: 0, ThousandSep: ".", DecimalSep: ",", IsDefault: false, IsActive: true},
		{Code: "USD", Name: "US Dollar", Symbol: "$", Decimal: 2, ThousandSep: ",", DecimalSep: ".", IsDefault: false, IsActive: true},
		{Code: "EUR", Name: "Euro", Symbol: "€", Decimal: 2, ThousandSep: ".", DecimalSep: ",", IsDefault: false, IsActive: true},
		{Code: "GBP", Name: "British Pound", Symbol: "£", Decimal: 2, ThousandSep: ",", DecimalSep: ".", IsDefault: false, IsActive: true},
		{Code: "JPY", Name: "Japanese Yen", Symbol: "¥", Decimal: 0, ThousandSep: ",", DecimalSep: ".", IsDefault: false, IsActive: true},
		{Code: "SGD", Name: "Singapore Dollar", Symbol: "S$", Decimal: 2, ThousandSep: ",", DecimalSep: ".", IsDefault: false, IsActive: true},
		{Code: "MYR", Name: "Malaysian Ringgit", Symbol: "RM", Decimal: 2, ThousandSep: ",", DecimalSep: ".", IsDefault: false, IsActive: true},
		{Code: "AUD", Name: "Australian Dollar", Symbol: "A$", Decimal: 2, ThousandSep: ",", DecimalSep: ".", IsDefault: false, IsActive: true},
	}
}

// GetDefaultIDR returns the Indonesian Rupiah configuration
func GetDefaultIDR() Currency {
	return Currency{
		Code:        "IDR",
		Name:        "Indonesian Rupiah",
		Symbol:      "Rp",
		Decimal:     0,
		ThousandSep: ".",
		DecimalSep:  ",",
	}
}

// FormatAmount formats an amount according to currency configuration
func (c *Currency) FormatAmount(amount float64) string {
	// Format with decimal places
	format := "%s%.0f"
	if c.Decimal == 2 {
		format = "%s%.2f"
	}

	return fmt.Sprintf(format, c.Symbol, amount)
}

// ConvertIDRToUSD converts IDR to USD (placeholder - use actual API)
func ConvertIDRToUSD(amountIDR float64, rate float64) float64 {
	return amountIDR / rate
}

// ConvertUSDToIDR converts USD to IDR (placeholder - use actual API)
func ConvertUSDToIDR(amountUSD float64, rate float64) float64 {
	return amountUSD * rate
}
