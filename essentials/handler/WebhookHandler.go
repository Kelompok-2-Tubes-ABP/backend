package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"financeapi/essentials/auth"
	"financeapi/essentials/models"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type WebhookHandler struct {
	webhookService *services.WebhookService
	emailService   *services.EmailService
}

func NewWebhookHandler(db *mongo.Database) *WebhookHandler {
	return &WebhookHandler{
		webhookService: services.NewWebhookService(db),
		emailService:   services.NewEmailService(db),
	}
}

func (h *WebhookHandler) SetupRoutes(r *gin.Engine) {
	webhooks := r.Group("/api/webhooks")
	webhooks.Use(auth.AuthMiddleware())
	{
		webhooks.POST("", h.CreateWebhook)
		webhooks.GET("", h.GetWebhooks)
		webhooks.GET("/:id", h.GetWebhook)
		webhooks.PUT("/:id", h.UpdateWebhook)
		webhooks.DELETE("/:id", h.DeleteWebhook)
		webhooks.POST("/:id/test", h.TestWebhook)
	}

	notifications := r.Group("/api/notifications")
	notifications.Use(auth.AuthMiddleware())
	{
		notifications.GET("/preferences", h.GetNotificationPreferences)
		notifications.PUT("/preferences", h.UpdateNotificationPreferences)
		notifications.GET("/emails", h.GetEmailLogs)
	}
}

func (h *WebhookHandler) CreateWebhook(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Name   string   `json:"name" binding:"required"`
		URL    string   `json:"url" binding:"required,url"`
		Events []string `json:"events" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validEvents := map[string]bool{
		"bill_reminder":  true,
		"debt_due":       true,
		"transaction":    true,
		"budget_warning": true,
		"goal_reached":   true,
		"weekly_report":  true,
		"monthly_report": true,
		"security_alert": true,
	}

	for _, event := range req.Events {
		if !validEvents[event] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event: " + event})
			return
		}
	}

	webhook := &models.Webhook{
		UserID:    userID.(primitive.ObjectID),
		Name:      req.Name,
		URL:       req.URL,
		Secret:    services.GenerateSecret(),
		Events:    req.Events,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.webhookService.CreateWebhook(c.Request.Context(), webhook); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"webhook": webhook,
		"secret":  webhook.Secret,
	})
}

func (h *WebhookHandler) GetWebhooks(c *gin.Context) {
	userID, _ := c.Get("user_id")

	webhooks, err := h.webhookService.GetWebhooksByUser(c.Request.Context(), userID.(primitive.ObjectID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"webhooks": webhooks})
}

func (h *WebhookHandler) GetWebhook(c *gin.Context) {
	userID, _ := c.Get("user_id")

	webhookID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook ID"})
		return
	}

	webhook, err := h.webhookService.GetWebhook(c.Request.Context(), webhookID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Webhook not found"})
		return
	}

	if webhook.UserID != userID.(primitive.ObjectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"webhook": webhook})
}

func (h *WebhookHandler) UpdateWebhook(c *gin.Context) {
	userID, _ := c.Get("user_id")

	webhookID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook ID"})
		return
	}

	webhook, err := h.webhookService.GetWebhook(c.Request.Context(), webhookID)
	if err != nil || webhook.UserID != userID.(primitive.ObjectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		Name     string   `json:"name"`
		URL      string   `json:"url"`
		Events   []string `json:"events"`
		IsActive bool     `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		webhook.Name = req.Name
	}
	if req.URL != "" {
		webhook.URL = req.URL
	}
	if req.Events != nil {
		webhook.Events = req.Events
	}
	webhook.IsActive = req.IsActive
	webhook.UpdatedAt = time.Now()

	if err := h.webhookService.UpdateWebhook(c.Request.Context(), webhook); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"webhook": webhook})
}

func (h *WebhookHandler) DeleteWebhook(c *gin.Context) {
	userID, _ := c.Get("user_id")

	webhookID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook ID"})
		return
	}

	if err := h.webhookService.DeleteWebhook(c.Request.Context(), webhookID, userID.(primitive.ObjectID)); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhook deleted successfully"})
}

func (h *WebhookHandler) TestWebhook(c *gin.Context) {
	userID, _ := c.Get("user_id")

	webhookID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook ID"})
		return
	}

	webhook, err := h.webhookService.GetWebhook(c.Request.Context(), webhookID)
	if err != nil || webhook.UserID != userID.(primitive.ObjectID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	testPayload := map[string]interface{}{
		"event":     "test",
		"message":   "This is a test webhook from FinanceAPI",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	payload, _ := json.Marshal(testPayload)

	statusCode, response, err := h.webhookService.TriggerWebhook(webhook.URL, webhook.Secret, "test", payload)

	h.webhookService.LogWebhookTrigger(c.Request.Context(), &models.WebhookLog{
		WebhookID:   webhookID,
		Event:       "test",
		Payload:     string(payload),
		StatusCode:  statusCode,
		Success:     err == nil,
		Response:    response,
		TriggeredAt: time.Now(),
	})

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success":     false,
			"status_code": statusCode,
			"error":       err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"status_code": statusCode,
		"response":    response,
	})
}

func (h *WebhookHandler) GetNotificationPreferences(c *gin.Context) {
	userID, _ := c.Get("user_id")

	pref, err := h.webhookService.GetNotificationPreferences(c.Request.Context(), userID.(primitive.ObjectID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"preferences": pref})
}

func (h *WebhookHandler) UpdateNotificationPreferences(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var pref models.UserNotificationPreference
	if err := c.ShouldBindJSON(&pref); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pref.UserID = userID.(primitive.ObjectID)
	pref.UpdatedAt = time.Now()

	if err := h.webhookService.UpdateNotificationPreferences(c.Request.Context(), &pref); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"preferences": pref})
}

func (h *WebhookHandler) GetEmailLogs(c *gin.Context) {
	userID, _ := c.Get("user_id")

	limit := int64(50)
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.ParseInt(l, 10, 64); err == nil {
			limit = parsed
		}
	}

	logs, err := h.emailService.GetEmailLogs(c.Request.Context(), userID.(primitive.ObjectID), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}
