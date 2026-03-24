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
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		userID, _ := c.Get("user_id")
		bill.UserID = userID.(primitive.ObjectID)
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
		id, _ := primitive.ObjectIDFromHex(c.Param("id"))
		userID, _ := c.Get("user_id")
		bill, err := h.service.GetBillReminder(id, userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"bill": bill, "status": bill.GetDueStatus()})
	}
}

func (h *BillReminderHandler) GetUserBillReminders() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		bills, err := h.service.GetUserBillReminders(userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, bills)
	}
}

func (h *BillReminderHandler) GetDueBillReminders() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		bills, err := h.service.GetDueBillReminders(userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, bills)
	}
}

func (h *BillReminderHandler) GetOverdueBillReminders() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		bills, err := h.service.GetOverdueBillReminders(userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, bills)
	}
}

func (h *BillReminderHandler) UpdateBillReminder() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := primitive.ObjectIDFromHex(c.Param("id"))
		var updates map[string]interface{}
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		userID, _ := c.Get("user_id")
		bill, err := h.service.UpdateBillReminder(id, userID.(primitive.ObjectID), updates)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, bill)
	}
}

func (h *BillReminderHandler) MarkAsPaid() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := primitive.ObjectIDFromHex(c.Param("id"))
		var req struct {
			Amount float64 `json:"amount"`
			Notes  string  `json:"notes"`
		}
		c.ShouldBindJSON(&req)
		userID, _ := c.Get("user_id")
		err := h.service.MarkAsPaid(id, userID.(primitive.ObjectID), req.Amount, req.Notes)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "Bill marked as paid"})
	}
}

func (h *BillReminderHandler) DeleteBillReminder() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := primitive.ObjectIDFromHex(c.Param("id"))
		userID, _ := c.Get("user_id")
		err := h.service.DeleteBillReminder(id, userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "Bill reminder deleted"})
	}
}

func (h *BillReminderHandler) GetBillSummary() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		summary, err := h.service.GetBillSummary(userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, summary)
	}
}

func (h *BillReminderHandler) GetPaymentHistory() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := primitive.ObjectIDFromHex(c.Param("id"))
		userID, _ := c.Get("user_id")
		payments, err := h.service.GetPaymentHistory(id, userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, payments)
	}
}
