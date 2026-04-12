package routes

import (
	authh "financeapi/essentials/auth"
	"financeapi/essentials/handler"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

func RegisterRecurringRoutes(r *gin.Engine, recurringService *services.RecurringTransactionService) {
	recurringHandler := handler.NewRecurringTransactionHandler(recurringService)
	recurringProtected := r.Group("/recurring", authh.AuthMiddleware())
	{
		recurringProtected.POST("/", recurringHandler.CreateRecurringTransaction())
		recurringProtected.GET("/", recurringHandler.GetUserRecurringTransactions())
		recurringProtected.GET("/active", recurringHandler.GetActiveRecurringTransactions())
		recurringProtected.GET("/due", recurringHandler.GetDueRecurringTransactions())
		recurringProtected.GET("/summary", recurringHandler.GetRecurringSummary())
		recurringProtected.GET("/:id", recurringHandler.GetRecurringTransaction())
		recurringProtected.GET("/:id/generated", recurringHandler.GetGeneratedTransactions())
		recurringProtected.PATCH("/:id", recurringHandler.UpdateRecurringTransaction())
		recurringProtected.DELETE("/:id", recurringHandler.DeleteRecurringTransaction())
		recurringProtected.POST("/:id/skip", recurringHandler.SkipNextRun())
		recurringProtected.POST("/:id/pause", recurringHandler.PauseRecurringTransaction())
		recurringProtected.POST("/:id/resume", recurringHandler.ResumeRecurringTransaction())
		recurringProtected.POST("/process", recurringHandler.ProcessDueTransactions())
	}
}
