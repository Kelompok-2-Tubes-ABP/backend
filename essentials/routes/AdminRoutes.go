package routes

import (
	"financeapi/essentials/auth"
	"financeapi/essentials/handler"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

func RegisterAdminRoutes(r *gin.Engine, adminService *services.AdminService, userService *services.UserService) {
	adminHandler := handler.NewAdminHandler(adminService, userService)
	
	// Public Admin Routes
	r.POST("/admin/login", adminHandler.Login)

	// Protected Admin Routes
	adminGroup := r.Group("/admin", auth.AuthMiddleware(), auth.AdminMiddleware())
	{
		// Dashboard
		adminGroup.GET("/dashboard", adminHandler.GetDashboardStats)

		// User Management
		adminGroup.GET("/users", adminHandler.GetUsers)
		adminGroup.GET("/users/:id", adminHandler.GetUserDetail)
		adminGroup.PATCH("/users/:id/disable", adminHandler.DisableUser)
		adminGroup.DELETE("/users/:id", adminHandler.DeleteUser)

		// Transaction Management
		adminGroup.GET("/transactions/recent", adminHandler.GetRecentTransactions)
		adminGroup.GET("/transactions/all", adminHandler.GetAllTransactions)
		adminGroup.PATCH("/transactions/:id/status", adminHandler.UpdateTransactionStatus)
		adminGroup.DELETE("/transactions/:id", adminHandler.DeleteTransaction)

		// Alerts & Logs
		adminGroup.GET("/alerts/stats", adminHandler.GetAlertStats)
		adminGroup.GET("/alerts", adminHandler.GetAlerts)
		adminGroup.PATCH("/alerts/:id/read", adminHandler.MarkAlertAsRead)

		// Audit Logs
		adminGroup.GET("/audit-logs/stats", adminHandler.GetAuditLogStats)
		adminGroup.GET("/audit-logs", adminHandler.GetAuditLogs)
		adminGroup.GET("/audit-logs/export", adminHandler.ExportAuditLogs)

		// Budget & Savings Monitoring
		adminGroup.GET("/budget-savings/stats", adminHandler.GetBudgetSavingsStats)
		adminGroup.GET("/budgets", adminHandler.GetAllUserBudgets)
		adminGroup.GET("/savings-goals", adminHandler.GetAllUserSavingsGoals)

		// Investment Management
		adminGroup.GET("/investments/stats", adminHandler.GetInvestmentStats)
		adminGroup.GET("/investments", adminHandler.GetAllInvestments)
		adminGroup.GET("/investments/export", adminHandler.ExportInvestments)

		// Analytics & Reports
		adminGroup.GET("/analytics", adminHandler.GetAnalyticsAndReports)
		adminGroup.GET("/analytics/export", adminHandler.ExportAnalyticsReport)

		// Settings & Auth
		adminGroup.POST("/password", adminHandler.ChangeAdminPassword)
		adminGroup.POST("/logout", adminHandler.Logout)
	}
}
