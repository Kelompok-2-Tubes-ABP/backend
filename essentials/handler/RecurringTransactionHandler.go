package handler

import (
	"financeapi/essentials/models"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RecurringTransactionHandler struct {
	service *services.RecurringTransactionService
}

func NewRecurringTransactionHandler(service *services.RecurringTransactionService) *RecurringTransactionHandler {
	return &RecurringTransactionHandler{service: service}
}

func (h *RecurringTransactionHandler) CreateRecurringTransaction() gin.HandlerFunc {
	return func(c *gin.Context) {
		var recurring models.RecurringTransaction
		if err := c.ShouldBindJSON(&recurring); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		userID, _ := c.Get("user_id")
		recurring.UserID = userID.(primitive.ObjectID)

		created, err := h.service.CreateRecurringTransaction(recurring)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(201, created)
	}
}

func (h *RecurringTransactionHandler) GetRecurringTransaction() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid ID"})
			return
		}

		userID, _ := c.Get("user_id")
		recurring, err := h.service.GetRecurringTransaction(id, userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, recurring)
	}
}

func (h *RecurringTransactionHandler) GetUserRecurringTransactions() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		recurring, err := h.service.GetUserRecurringTransactions(userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, recurring)
	}
}

func (h *RecurringTransactionHandler) GetActiveRecurringTransactions() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		recurring, err := h.service.GetActiveRecurringTransactions(userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, recurring)
	}
}

func (h *RecurringTransactionHandler) GetDueRecurringTransactions() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		recurring, err := h.service.GetDueRecurringTransactions(userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, recurring)
	}
}

func (h *RecurringTransactionHandler) UpdateRecurringTransaction() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid ID"})
			return
		}

		var updates map[string]interface{}
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		userID, _ := c.Get("user_id")
		recurring, err := h.service.UpdateRecurringTransaction(id, userID.(primitive.ObjectID), updates)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, recurring)
	}
}

func (h *RecurringTransactionHandler) DeleteRecurringTransaction() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid ID"})
			return
		}

		userID, _ := c.Get("user_id")
		err = h.service.DeleteRecurringTransaction(id, userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Recurring transaction deleted"})
	}
}

func (h *RecurringTransactionHandler) ProcessDueTransactions() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		generated, err := h.service.ProcessDueTransactions(userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message":   "Processed due transactions",
			"generated": len(generated),
		})
	}
}

func (h *RecurringTransactionHandler) SkipNextRun() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid ID"})
			return
		}

		userID, _ := c.Get("user_id")
		err = h.service.SkipNextRun(id, userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Next run skipped"})
	}
}

func (h *RecurringTransactionHandler) PauseRecurringTransaction() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid ID"})
			return
		}

		userID, _ := c.Get("user_id")
		err = h.service.PauseRecurringTransaction(id, userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Recurring transaction paused"})
	}
}

func (h *RecurringTransactionHandler) ResumeRecurringTransaction() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid ID"})
			return
		}

		userID, _ := c.Get("user_id")
		err = h.service.ResumeRecurringTransaction(id, userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Recurring transaction resumed"})
	}
}

func (h *RecurringTransactionHandler) GetGeneratedTransactions() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid ID"})
			return
		}

		userID, _ := c.Get("user_id")
		generated, err := h.service.GetGeneratedTransactions(id, userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, generated)
	}
}

func (h *RecurringTransactionHandler) GetRecurringSummary() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		summary, err := h.service.GetRecurringSummary(userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, summary)
	}
}
