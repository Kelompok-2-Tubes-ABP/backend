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
	notificationService *services.NotificationService
}

func NewNotificationHandler(db *mongo.Database) *NotificationHandler {
	return &NotificationHandler{
		notificationService: services.NewNotificationService(db),
	}
}

// GetMyNotifications fetches notifications for the authenticated user
func (h *NotificationHandler) GetMyNotifications(c *gin.Context) {
	userID, _ := c.Get("user_id")
	
	unreadOnlyStr := c.Query("unread_only")
	unreadOnly := unreadOnlyStr == "true"

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		limit = 50
	}

	notifications, err := h.notificationService.GetUserNotifications(c.Request.Context(), userID.(primitive.ObjectID), unreadOnly, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	unreadCount, _ := h.notificationService.GetUnreadCount(c.Request.Context(), userID.(primitive.ObjectID))

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"unread_count":  unreadCount,
	})
}

// MarkAsRead marks a specific notification as read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	notifIDStr := c.Param("id")

	notifID, err := primitive.ObjectIDFromHex(notifIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	err = h.notificationService.MarkAsRead(c.Request.Context(), notifID, userID.(primitive.ObjectID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notification as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

// MarkAllAsRead marks all user notifications as read
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")

	err := h.notificationService.MarkAllAsRead(c.Request.Context(), userID.(primitive.ObjectID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark all as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}
