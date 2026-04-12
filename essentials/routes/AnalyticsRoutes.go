package routes

import (
	authh "financeapi/essentials/auth"
	"financeapi/essentials/handler"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

func RegisterAnalyticsRoutes(r *gin.Engine, analyticsService *services.AnalyticsService) {
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)
	analyticsProtected := r.Group("/analytics", authh.AuthMiddleware())
	{
		analyticsProtected.GET("/", analyticsHandler.GetFullAnalytics())
		analyticsProtected.GET("/quick", analyticsHandler.GetQuickStats())
		analyticsProtected.GET("/goals", analyticsHandler.GetGoalProgress())
		analyticsProtected.GET("/net-worth", analyticsHandler.GetNetWorthDetail())
	}
}
