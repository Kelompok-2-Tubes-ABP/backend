package handler

import (
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AnalyticsHandler struct {
	analyticsService *services.AnalyticsService
}

func NewAnalyticsHandler(analyticsService *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsService: analyticsService}
}

func (h *AnalyticsHandler) GetFullAnalytics() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userIDStr := userID.(primitive.ObjectID).Hex()

		period := c.DefaultQuery("period", "monthly")
		if period != "daily" && period != "weekly" && period != "monthly" && period != "yearly" {
			period = "monthly"
		}

		report, err := h.analyticsService.GetFullAnalytics(userIDStr, period)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    report,
		})
	}
}

func (h *AnalyticsHandler) GetQuickStats() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userIDStr := userID.(primitive.ObjectID).Hex()

		stats, err := h.analyticsService.GetQuickStats(userIDStr)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    stats,
		})
	}
}

func (h *AnalyticsHandler) GetGoalProgress() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userIDStr := userID.(primitive.ObjectID).Hex()

		progress, err := h.analyticsService.GetGoalProgress(userIDStr)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    progress,
		})
	}
}
