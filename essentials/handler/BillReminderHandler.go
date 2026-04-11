package handler

import (
	"financeapi/essentials/models"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BillReminderHandler struct {
	service *services.BillReminderService
}

func NewBillReminderHandler(service *services.BillReminderService) *BillReminderHandler {
	return &BillReminderHandler{service: service}
}

func (h *BillReminderHandler) CreateBillReminder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var bill models.BillReminder
		if err := c.ShouldBindJSON(&bill); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}
		bill.UserID = userID

		created, err := h.service.CreateBillReminder(bill)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, created)
	}
}

func (h *BillReminderHandler) GetBillReminder() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid bill ID"})
			return
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		bill, err := h.service.GetBillReminder(id, userID)
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"bill": bill, "status": bill.GetDueStatus()})
	}
}

func (h *BillReminderHandler) GetUserBillReminders() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		bills, err := h.service.GetUserBillReminders(userID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		if bills == nil {
			bills = []models.BillReminder{}
		}

		c.JSON(200, bills)
	}
}

func (h *BillReminderHandler) GetDueBillReminders() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		bills, err := h.service.GetDueBillReminders(userID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		if bills == nil {
			bills = []models.BillReminder{}
		}

		c.JSON(200, bills)
	}
}

func (h *BillReminderHandler) GetOverdueBillReminders() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		bills, err := h.service.GetOverdueBillReminders(userID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		if bills == nil {
			bills = []models.BillReminder{}
		}

		c.JSON(200, bills)
	}
}

func (h *BillReminderHandler) UpdateBillReminder() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid bill ID"})
			return
		}

		var updates map[string]interface{}
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		bill, err := h.service.UpdateBillReminder(id, userID, updates)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, bill)
	}
}

func (h *BillReminderHandler) MarkAsPaid() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid bill ID"})
			return
		}

		var req struct {
			Amount float64 `json:"amount" binding:"required"`
			Notes  string  `json:"notes"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "amount is required"})
			return
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		err = h.service.MarkAsPaid(id, userID, req.Amount, req.Notes)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "Bill marked as paid"})
	}
}

func (h *BillReminderHandler) DeleteBillReminder() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid bill ID"})
			return
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		err = h.service.DeleteBillReminder(id, userID)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "Bill reminder deleted"})
	}
}

func (h *BillReminderHandler) GetBillSummary() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		summary, err := h.service.GetBillSummary(userID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, summary)
	}
}

func (h *BillReminderHandler) GetPaymentHistory() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid bill ID"})
			return
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		payments, err := h.service.GetPaymentHistory(id, userID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		if payments == nil {
			payments = []models.BillPaymentLog{}
		}

		c.JSON(200, payments)
	}
}
