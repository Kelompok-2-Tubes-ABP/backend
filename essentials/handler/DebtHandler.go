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
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}
		debt.UserID = userID

		created, err := h.service.CreateDebt(debt)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Get advice based on income
		income, _ := h.service.GetUserMonthlyIncome(userID)
		_, advice := created.CheckDebtHealth(income)

		c.JSON(201, gin.H{
			"debt":   created,
			"advice": advice,
		})
	}
}

func (h *DebtHandler) GetDebt() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid debt ID"})
			return
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		debt, err := h.service.GetDebt(id, userID)
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, debt)
	}
}

func (h *DebtHandler) GetUserDebts() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		debts, err := h.service.GetUserDebts(userID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		if debts == nil {
			debts = []models.Debt{}
		}

		c.JSON(200, debts)
	}
}

func (h *DebtHandler) MakePayment() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid debt ID"})
			return
		}

		var req struct {
			Amount    float64 `json:"amount" binding:"required"`
			AccountID string  `json:"account_id"` // Sekarang opsional
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Amount is required"})
			return
		}

		var accID primitive.ObjectID
		if req.AccountID != "" {
			accID, err = primitive.ObjectIDFromHex(req.AccountID)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid account ID format"})
				return
			}
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		payment, err := h.service.MakePayment(id, userID, accID, req.Amount)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message": "Payment recorded successfully",
			"payment": payment,
		})
	}
}

func (h *DebtHandler) GetPaymentHistory() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid debt ID"})
			return
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		history, err := h.service.GetPaymentHistory(id, userID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		if history == nil {
			history = []models.DebtPayment{}
		}

		c.JSON(200, history)
	}
}

func (h *DebtHandler) GetDebtSummary() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		summary, err := h.service.GetDebtSummary(userID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, summary)
	}
}

func (h *DebtHandler) DeleteDebt() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid debt ID"})
			return
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		err = h.service.DeleteDebt(id, userID)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "Debt record deleted"})
	}
}
