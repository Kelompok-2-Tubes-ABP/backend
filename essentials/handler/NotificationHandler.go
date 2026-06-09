package handler

import (
	"net/http"
	"strconv"

	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type NotificationHandler struct {
	notificationService    *services.NotificationService
	billReminderService    *services.BillReminderService
	debtService            *services.DebtService
	recurringService       *services.RecurringTransactionService
}

func NewNotificationHandler(db *mongo.Database) *NotificationHandler {
	notificationService := services.NewNotificationService(db)
	billReminderService := services.NewBillReminderService(db.Client(), "mydb")
	debtService := services.NewDebtService(db.Client(), "mydb")
	recurringService := services.NewRecurringTransactionService(db.Client(), "mydb")

	// Wire notification service to other services
	billReminderService.SetNotificationService(notificationService)
	debtService.SetNotificationService(notificationService)
	recurringService.SetNotificationService(notificationService)

	return &NotificationHandler{
		notificationService:   notificationService,
		billReminderService:    billReminderService,
		debtService:             debtService,
		recurringService:       recurringService,
	}
}

// GetMyNotifications fetches notifications for the authenticated user
// It also triggers checks for Bill, Debt, and Recurring notifications
func (h *NotificationHandler) GetMyNotifications(c *gin.Context) {
	userIDStr, _ := c.Get("user_id")
	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Trigger notification checks for Bill, Debt, and Recurring
	// These will create notifications if there are due/overdue items
	go func() {
		h.billReminderService.CheckAndNotifyDueBills(userID)
		h.debtService.CheckAndNotifyDueDebtPayments(userID)
		h.recurringService.CheckAndNotifyDueRecurring(userID)
	}()

	unreadOnlyStr := c.Query("unread_only")
	unreadOnly := unreadOnlyStr == "true"

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		limit = 50
	}

	notifications, err := h.notificationService.GetUserNotifications(c.Request.Context(), userID, unreadOnly, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	unreadCount, _ := h.notificationService.GetUnreadCount(c.Request.Context(), userID)

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"unread_count":  unreadCount,
	})
}

// MarkAsRead marks a specific notification as read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userIDStr, _ := c.Get("user_id")
	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	notifIDStr := c.Param("id")
	notifID, err := primitive.ObjectIDFromHex(notifIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	err = h.notificationService.MarkAsRead(c.Request.Context(), notifID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notification as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

// MarkAllAsRead marks all user notifications as read
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userIDStr, _ := c.Get("user_id")
	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	err = h.notificationService.MarkAllAsRead(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark all as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}
