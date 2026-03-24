package handler

import (
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

func (h *SpendingInsightHandler) GetInsights() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		limit := int64(10)
		insights, err := h.service.GetUserInsights(userID.(primitive.ObjectID), limit)
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
		userID, _ := c.Get("user_id")
		err := h.service.MarkAsRead(id, userID.(primitive.ObjectID))
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
		userID, _ := c.Get("user_id")
		err := h.service.MarkAsActioned(id, userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "Insight marked as actioned"})
	}
}

func (h *SpendingInsightHandler) GetHealthScore() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		score, err := h.service.GetFinancialHealthScore(userID.(primitive.ObjectID))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, score)
	}
}
