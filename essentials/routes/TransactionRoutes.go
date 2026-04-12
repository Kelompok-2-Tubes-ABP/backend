package routes

import (
	authh "financeapi/essentials/auth"
	"financeapi/essentials/handler"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

func RegisterTransactionRoutes(r *gin.Engine, txService *services.TransactionService) {
	transactionProtected := r.Group("/transaction", authh.AuthMiddleware())
	{
		transactionProtected.POST("/new", handler.CreateTransactionHandler(txService))
		transactionProtected.DELETE("/delete/:id", handler.DeleteTransactionHandler(txService))
		transactionProtected.GET("/", handler.GetAllTransaction(txService))
		transactionProtected.PATCH("/update/:id", handler.UpdateTransactionHandler(txService))
		transactionProtected.GET("/filter", handler.FilterByCategoryHandler(txService))
		transactionProtected.GET("/getMonthly", handler.GetMonthlyHandler(txService))
		transactionProtected.GET("/getReport", handler.GetReportHandler(txService))
	}
}
