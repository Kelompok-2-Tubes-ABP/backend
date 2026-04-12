package routes

import (
	authh "financeapi/essentials/auth"
	"financeapi/essentials/handler"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

func RegisterBillRoutes(r *gin.Engine, billReminderService *services.BillReminderService) {
	billHandler := handler.NewBillReminderHandler(billReminderService)
	billProtected := r.Group("/bill", authh.AuthMiddleware())
	{
		billProtected.POST("/", billHandler.CreateBillReminder())
		billProtected.GET("/", billHandler.GetUserBillReminders())
		billProtected.GET("/due", billHandler.GetDueBillReminders())
		billProtected.GET("/overdue", billHandler.GetOverdueBillReminders())
		billProtected.GET("/summary", billHandler.GetBillSummary())
		billProtected.GET("/:id", billHandler.GetBillReminder())
		billProtected.GET("/:id/history", billHandler.GetPaymentHistory())
		billProtected.PATCH("/:id", billHandler.UpdateBillReminder())
		billProtected.POST("/:id/pay", billHandler.MarkAsPaid())
		billProtected.DELETE("/:id", billHandler.DeleteBillReminder())
	}
}
