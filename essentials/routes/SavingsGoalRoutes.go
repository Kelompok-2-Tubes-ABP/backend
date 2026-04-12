package routes

import (
	authh "financeapi/essentials/auth"
	"financeapi/essentials/handler"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

func RegisterSavingsGoalRoutes(r *gin.Engine, savingsGoalService *services.SavingsGoalService) {
	savingsGoalHandler := handler.NewSavingsGoalHandler(savingsGoalService)
	savingsProtected := r.Group("/savings_goal", authh.AuthMiddleware())
	{
		savingsProtected.POST("/", savingsGoalHandler.CreateSavingsGoal())
		savingsProtected.GET("/get", savingsGoalHandler.GetUserSavingsGoals())
		savingsProtected.GET("/summary", savingsGoalHandler.GetSavingsSummary())
		savingsProtected.GET("/get/:id", savingsGoalHandler.GetSavingsGoal())
		savingsProtected.PATCH("/update/:id", savingsGoalHandler.UpdateSavingsGoal())
		savingsProtected.POST("/:id/contribute", savingsGoalHandler.AddContribution())
		savingsProtected.GET("/:id/contributions", savingsGoalHandler.GetContributions())
		savingsProtected.DELETE("/:id", savingsGoalHandler.DeleteSavingsGoal())
	}
}
