package handler

import (
	"financeapi/essentials/models"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SpendingInsightHandler struct {
	service *services.SpendingInsightService
}

func NewSpendingInsightHandler(service *services.SpendingInsightService) *SpendingInsightHandler {
	return &SpendingInsightHandler{service: service}
}

func (h *SpendingInsightHandler) CreateInsight() gin.HandlerFunc {
	return func(c *gin.Context) {
		var insight models.SpendingInsight
		if err := c.ShouldBindJSON(&insight); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}
		insight.UserID = userID

		created, err := h.service.CreateInsight(insight)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(201, created)
	}
}

func (h *SpendingInsightHandler) GetInsights() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}
		limit := int64(10)
		insights, err := h.service.GetUserInsights(userID, limit)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, insights)
	}
}

func (h *SpendingInsightHandler) MarkAsRead() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := primitive.ObjectIDFromHex(c.Param("id"))
		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}
		err = h.service.MarkAsRead(id, userID)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "Insight marked as read"})
	}
}

func (h *SpendingInsightHandler) MarkAsActioned() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := primitive.ObjectIDFromHex(c.Param("id"))
		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}
		err = h.service.MarkAsActioned(id, userID)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "Insight marked as actioned"})
	}
}

func (h *SpendingInsightHandler) GetHealthScore() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, _ := c.Get("user_id")
		userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}
		score, err := h.service.GetFinancialHealthScore(userID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, score)
	}
}
