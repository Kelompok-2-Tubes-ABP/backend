package handler

import (
	"financeapi/essentials/models"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DebtHandler struct {
	service *services.DebtService
}

func NewDebtHandler(service *services.DebtService) *DebtHandler {
	return &DebtHandler{service: service}
}

func (h *DebtHandler) CreateDebt() gin.HandlerFunc {
	return func(c *gin.Context) {
		var debt models.Debt
		if err := c.ShouldBindJSON(&debt); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		userID, _ := c.Get("user_id")
		debt.UserID = userID.(primitive.ObjectID)
		created, err := h.service.CreateDebt(debt)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, created)
	}
}

func (h *DebtHandler) GetDebt() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := primitive.ObjectIDFromHex(c.Param("id"))
		userID, _ := c.Get("user_id")
		debt, err := h.service.GetDebt(id, userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, debt)
	}
}

func (h *DebtHandler) GetUserDebts() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		debts, err := h.service.GetUserDebts(userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, debts)
	}
}

func (h *DebtHandler) MakePayment() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := primitive.ObjectIDFromHex(c.Param("id"))
		var req struct {
			Amount float64 `json:"amount" binding:"required"`
		}
		c.ShouldBindJSON(&req)
		userID, _ := c.Get("user_id")
		err := h.service.MakePayment(id, userID.(primitive.ObjectID), req.Amount)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "Payment recorded"})
	}
}

func (h *DebtHandler) GetDebtSummary() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		summary, err := h.service.GetDebtSummary(userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, summary)
	}
}
