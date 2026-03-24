package handler

import (
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CurrencyHandler struct {
	service *services.CurrencyService
}

func NewCurrencyHandler(service *services.CurrencyService) *CurrencyHandler {
	return &CurrencyHandler{service: service}
}

func (h *CurrencyHandler) GetSupportedCurrencies() gin.HandlerFunc {
	return func(c *gin.Context) {
		currencies, err := h.service.GetSupportedCurrencies()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, currencies)
	}
}

func (h *CurrencyHandler) GetExchangeRate() gin.HandlerFunc {
	return func(c *gin.Context) {
		fromCurrency := c.Param("from")
		toCurrency := c.Param("to")

		rate, err := h.service.GetExchangeRate(fromCurrency, toCurrency)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"from_currency": rate.FromCurrency,
			"to_currency":   rate.ToCurrency,
			"rate":          rate.Rate,
			"source":        rate.Source,
			"last_updated":  rate.LastUpdated,
		})
	}
}

func (h *CurrencyHandler) ConvertCurrency() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			FromCurrency string  `json:"from_currency" binding:"required"`
			ToCurrency   string  `json:"to_currency" binding:"required"`
			Amount       float64 `json:"amount" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		userID, _ := c.Get("user_id")
		conversion, err := h.service.ConvertCurrency(userID.(primitive.ObjectID), req.FromCurrency, req.ToCurrency, req.Amount)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, conversion)
	}
}

func (h *CurrencyHandler) GetUserCurrency() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userCurrency, err := h.service.GetUserCurrency(userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, userCurrency)
	}
}

func (h *CurrencyHandler) SetUserDefaultCurrency() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			CurrencyCode string `json:"currency_code" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		userID, _ := c.Get("user_id")
		err := h.service.SetUserDefaultCurrency(userID.(primitive.ObjectID), req.CurrencyCode)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Default currency updated successfully"})
	}
}

func (h *CurrencyHandler) AddUserCurrency() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			CurrencyCode string `json:"currency_code" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		userID, _ := c.Get("user_id")
		err := h.service.AddUserCurrency(userID.(primitive.ObjectID), req.CurrencyCode)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Currency added successfully"})
	}
}

func (h *CurrencyHandler) GetAllRates() gin.HandlerFunc {
	return func(c *gin.Context) {
		currencyCode := c.Param("code")

		rates, err := h.service.GetAllRatesForCurrency(currencyCode)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, rates)
	}
}

func (h *CurrencyHandler) InitializeCurrencies() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := h.service.InitializeSupportedCurrencies()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Currencies initialized successfully"})
	}
}
