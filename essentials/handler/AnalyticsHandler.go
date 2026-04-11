package handler

import (
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AnalyticsHandler struct {
	analyticsService *services.AnalyticsService
}

func NewAnalyticsHandler(as *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsService: as}
}

func (h *AnalyticsHandler) GetFullAnalytics() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userIDStr := userID.(string)

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
		userIDStr := userID.(string)

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
		userIDStr := userID.(string)

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

func (h *AnalyticsHandler) GetNetWorthDetail() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userIDStr := userID.(string)

		userOID, err := primitive.ObjectIDFromHex(userIDStr)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid user ID"})
			return
		}

		detail, err := h.analyticsService.GetNetWorthDetail(userOID)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    detail,
		})
	}
}
