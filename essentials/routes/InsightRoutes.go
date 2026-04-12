package routes

import (
	authh "financeapi/essentials/auth"
	"financeapi/essentials/handler"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

func RegisterInsightRoutes(r *gin.Engine, spendingInsightService *services.SpendingInsightService) {
	insightHandler := handler.NewSpendingInsightHandler(spendingInsightService)
	insightProtected := r.Group("/insights", authh.AuthMiddleware())
	{
		insightProtected.POST("/create", insightHandler.CreateInsight())
		insightProtected.GET("/", insightHandler.GetInsights())
		insightProtected.GET("/health", insightHandler.GetHealthScore())
		insightProtected.POST("/:id/read", insightHandler.MarkAsRead())
		insightProtected.POST("/:id/action", insightHandler.MarkAsActioned())
	}
}
