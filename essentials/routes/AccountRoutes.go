package routes

import (
	authh "financeapi/essentials/auth"
	"financeapi/essentials/handler"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

func RegisterAccountRoutes(r *gin.Engine, accountService *services.AccountService) {
	accountHandler := handler.NewAccountHandler(accountService)
	accountProtected := r.Group("/account", authh.AuthMiddleware())
	{
		accountProtected.POST("/", accountHandler.CreateAccount())
		accountProtected.GET("/", accountHandler.GetUserAccounts())
		accountProtected.GET("/summary", accountHandler.GetAccountSummary())
		accountProtected.GET("/type/:type", accountHandler.GetAccountsByType())
		accountProtected.GET("/groups", accountHandler.GetAccountGroups())
		accountProtected.POST("/groups", accountHandler.CreateAccountGroup())
		accountProtected.GET("/:id", accountHandler.GetAccount())
		accountProtected.PATCH("/:id", accountHandler.UpdateAccount())
		accountProtected.PATCH("/:id/balance", accountHandler.UpdateBalance())
		accountProtected.POST("/:id/sync", accountHandler.SyncAccount())
		accountProtected.DELETE("/:id", accountHandler.DeleteAccount())
		accountProtected.POST("/transfer", accountHandler.Transfer())
		accountProtected.GET("/transfers", accountHandler.GetTransferHistory())
	}
}
