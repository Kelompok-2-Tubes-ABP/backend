package routes

import (
	authh "financeapi/essentials/auth"
	"financeapi/essentials/handler"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

func RegisterBudgetRoutes(r *gin.Engine, budgetService *services.BudgetService) {
	budgetHandler := handler.NewBudgetHandler(budgetService)
	budgetProtected := r.Group("/budget", authh.AuthMiddleware())
	{
		// Basic CRUD - root level
		budgetProtected.POST("/", budgetHandler.CreateBudget())
		budgetProtected.GET("/", budgetHandler.GetUserBudgets())

		// Summary & reports
		budgetProtected.GET("/summary", budgetHandler.GetBudgetSummary())
		budgetProtected.GET("/all-spending", budgetHandler.GetAllBudgetsWithSpending())

		// Monthly routes
		budgetProtected.GET("/by-month/:month", budgetHandler.GetBudgetByMonth())
		budgetProtected.GET("/by-month/:month/spending", budgetHandler.GetBudgetWithSpending())

		// ID-based routes
		budgetProtected.GET("/:id", budgetHandler.GetBudget())
		budgetProtected.PATCH("/:id", budgetHandler.UpdateBudget())
		budgetProtected.DELETE("/:id", budgetHandler.DeleteBudget())

		// Category Budget routes
		budgetProtected.POST("/category", budgetHandler.CreateCategoryBudget())
		budgetProtected.GET("/category", budgetHandler.GetUserCategoryBudgets())
		budgetProtected.GET("/category/summary", budgetHandler.GetCategoryBudgetSummary())

		// Category monthly routes
		budgetProtected.GET("/category/by-month/:month", budgetHandler.GetCategoryBudgetByMonth())
		budgetProtected.GET("/category/by-month/:month/all", budgetHandler.GetAllCategoryBudgetsWithSpending())
		budgetProtected.GET("/category/by-month/:month/:category/spending", budgetHandler.GetCategoryBudgetWithSpending())

		// Category ID routes
		budgetProtected.GET("/category/:id", budgetHandler.GetCategoryBudget())
		budgetProtected.PATCH("/category/:id", budgetHandler.UpdateCategoryBudget())
		budgetProtected.DELETE("/category/:id", budgetHandler.DeleteCategoryBudget())
	}
}
