package routes

import (
	authh "financeapi/essentials/auth"
	"financeapi/essentials/handler"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

func RegisterDebtRoutes(r *gin.Engine, debtService *services.DebtService) {
	debtHandler := handler.NewDebtHandler(debtService)
	debtProtected := r.Group("/debt", authh.AuthMiddleware())
	{
		debtProtected.POST("/", debtHandler.CreateDebt())
		debtProtected.GET("/", debtHandler.GetUserDebts())
		debtProtected.GET("/summary", debtHandler.GetDebtSummary())
		debtProtected.GET("/:id", debtHandler.GetDebt())
		debtProtected.GET("/:id/history", debtHandler.GetPaymentHistory())
		debtProtected.POST("/:id/pay", debtHandler.MakePayment())
		debtProtected.DELETE("/:id", debtHandler.DeleteDebt())
	}
}
