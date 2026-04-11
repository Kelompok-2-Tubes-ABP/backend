package handler

import (
	"financeapi/essentials/models"
	"financeapi/essentials/services"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InvestmentHandler struct {
	service *services.InvestmentService
}

func NewInvestmentHandler(service *services.InvestmentService) *InvestmentHandler {
	return &InvestmentHandler{service: service}
}

func (h *InvestmentHandler) CreateInvestment() gin.HandlerFunc {
	return func(c *gin.Context) {
		var inv models.Investment
		if err := c.ShouldBindJSON(&inv); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		inv.UserID = userID
		created, err := h.service.CreateInvestment(inv)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, created)
	}
}

func (h *InvestmentHandler) GetInvestment() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := primitive.ObjectIDFromHex(c.Param("id"))

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		inv, err := h.service.GetInvestment(id, userID)
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, inv)
	}
}

func (h *InvestmentHandler) GetUserInvestments() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		investments, err := h.service.GetUserInvestments(userID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, investments)
	}
}

func (h *InvestmentHandler) GetPortfolioSummary() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		summary, err := h.service.GetPortfolioSummary(userID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, summary)
	}
}

func (h *InvestmentHandler) UpdatePrice() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := primitive.ObjectIDFromHex(c.Param("id"))
		var req struct {
			NewPrice float64 `json:"new_price" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		err = h.service.UpdatePrice(id, userID, req.NewPrice)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "Price updated"})
	}
}

func (h *InvestmentHandler) DeleteInvestment() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := primitive.ObjectIDFromHex(c.Param("id"))

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		err = h.service.DeleteInvestment(id, userID)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "Investment deleted"})
	}
}

func (h *InvestmentHandler) RefreshAllPrices() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		currency := strings.ToLower(c.DefaultQuery("currency", "idr"))

		updatedCount, err := h.service.RefreshAllPrices(userID, currency)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error":   err.Error(),
				"message": "Failed to refresh prices",
			})
			return
		}

		c.JSON(200, gin.H{
			"success":       true,
			"updated_count": updatedCount,
			"message":       "Prices refreshed successfully",
			"currency":      currency,
		})
	}
}

func (h *InvestmentHandler) RefreshSinglePrice() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid investment ID"})
			return
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		currency := strings.ToLower(c.DefaultQuery("currency", "idr"))

		newPrice, err := h.service.RefreshSinglePrice(id, userID, currency)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error":   err.Error(),
				"message": "Failed to refresh price",
			})
			return
		}

		c.JSON(200, gin.H{
			"success":   true,
			"new_price": newPrice,
			"currency":  currency,
			"message":   "Price refreshed successfully",
		})
	}
}

func (h *InvestmentHandler) GetPortfolioWithLivePrices() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		currency := strings.ToLower(c.DefaultQuery("currency", "idr"))

		investments, err := h.service.GetPortfolioWithLivePrices(userID, currency)
		if err != nil {
			c.JSON(500, gin.H{
				"error":   err.Error(),
				"message": "Failed to fetch portfolio",
			})
			return
		}

		summary, err := h.service.GetPortfolioSummary(userID)
		if err != nil {
			summary = nil
		}

		c.JSON(200, gin.H{
			"investments": investments,
			"summary":     summary,
			"currency":    currency,
			"last_update": "auto-refreshed",
		})
	}
}

func (h *InvestmentHandler) GetPortfolioSummaryWithLivePrices() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		currency := strings.ToLower(c.DefaultQuery("currency", "idr"))

		summary, err := h.service.GetPortfolioSummaryWithLivePrices(userID, currency)
		if err != nil {
			c.JSON(500, gin.H{
				"error":   err.Error(),
				"message": "Failed to fetch summary",
			})
			return
		}

		c.JSON(200, gin.H{
			"summary":  summary,
			"currency": currency,
		})
	}
}

func (h *InvestmentHandler) AddTransaction() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tx models.InvestmentTransaction
		if err := c.ShouldBindJSON(&tx); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		tx.UserID = userID

		created, err := h.service.AddTransaction(tx)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(201, created)
	}
}

func (h *InvestmentHandler) GetInvestmentTransactions() gin.HandlerFunc {
	return func(c *gin.Context) {
		invID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid investment ID"})
			return
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		transactions, err := h.service.GetInvestmentTransactions(invID, userID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, transactions)
	}
}
