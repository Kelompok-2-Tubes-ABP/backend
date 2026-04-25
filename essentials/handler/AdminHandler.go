package handler

import (
	"context"
	"encoding/csv"
	"financeapi/essentials/auth"
	"financeapi/essentials/config"
	"financeapi/essentials/models"
	"financeapi/essentials/services"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

type AdminHandler struct {
	adminService *services.AdminService
	userService  *services.UserService
}

func NewAdminHandler(adminService *services.AdminService, userService *services.UserService) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
		userService:  userService,
	}
}

// helper to log admin actions
func (h *AdminHandler) logAdminAction(c *gin.Context, actionType, target, details string) {
	adminID, _ := c.Get("user_id")
	adminName, _ := c.Get("username")
	
	h.adminService.CreateAuditLog(models.AuditLog{
		Timestamp:  time.Now(),
		ActorID:    adminID.(string),
		ActorName:  adminName.(string),
		ActorRole:  "admin",
		ActionType: actionType,
		TargetData: target,
		IPAddress:  c.ClientIP(),
		Details:    details,
	})
}

// GetDashboardStats returns high-level metrics
func (h *AdminHandler) GetDashboardStats(c *gin.Context) {
	stats, err := h.adminService.GetDashboardStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load dashboard stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": stats})
}

// GetUsers lists users with pagination, search, and status filter
func (h *AdminHandler) GetUsers(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	search := c.Query("search")
	status := c.Query("status")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}

	users, total, err := h.adminService.GetUsers(page, limit, search, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   users,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// GetUserDetail fetches user info and recent transactions
func (h *AdminHandler) GetUserDetail(c *gin.Context) {
	userID := c.Param("id")

	detail, err := h.adminService.GetUserDetail(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user details"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": detail})
}

// DisableUser marks user as inactive
func (h *AdminHandler) DisableUser(c *gin.Context) {
	userID := c.Param("id")

	err := h.adminService.DisableUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disable user"})
		return
	}

	h.logAdminAction(c, "Update", "User Account #"+userID, "Disabled user account access")

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "User disabled successfully"})
}

// DeleteUser permanently removes a user
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	err := h.adminService.DeleteUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	h.logAdminAction(c, "Delete", "User Account #"+userID, "Permanently deleted user account")

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "User deleted successfully"})
}

// GetRecentTransactions lists transactions with optional date filters
func (h *AdminHandler) GetRecentTransactions(c *gin.Context) {
	fromDate := c.Query("from_date")
	toDate := c.Query("to_date")
	limitStr := c.DefaultQuery("limit", "20")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 20
	}

	transactions, err := h.adminService.GetRecentTransactions(fromDate, toDate, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch recent transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": transactions})
}

// DeleteTransaction removes a problematic transaction
func (h *AdminHandler) DeleteTransaction(c *gin.Context) {
	txID := c.Param("id")

	err := h.adminService.DeleteTransaction(txID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete transaction"})
		return
	}

	h.logAdminAction(c, "Delete", "Transaction #"+txID, "Deleted problematic transaction")

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Transaction deleted successfully"})
}

// ChangeAdminPassword allows admin to change their own password
func (h *AdminHandler) ChangeAdminPassword(c *gin.Context) {
	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	err := h.userService.ChangePasswordByString(adminID.(string), req.CurrentPassword, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Admin password changed successfully"})
}

// GetAlertStats returns the counts for alerts, failed actions, unread, etc.
func (h *AdminHandler) GetAlertStats(c *gin.Context) {
	stats, err := h.adminService.GetAlertStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load alert stats"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": stats})
}

// GetAlerts returns lists of system alerts or failed actions
func (h *AdminHandler) GetAlerts(c *gin.Context) {
	alertType := c.Query("type") // "system_alert" or "failed_action"
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 {
		limit = 10
	}

	alerts, total, err := h.adminService.GetAlerts(alertType, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch alerts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   alerts,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// MarkAlertAsRead sets an alert to read
func (h *AdminHandler) MarkAlertAsRead(c *gin.Context) {
	alertID := c.Param("id")
	err := h.adminService.MarkAlertAsRead(alertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark alert as read"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Alert marked as read"})
}

// GetAuditLogStats returns stats for the audit logs
func (h *AdminHandler) GetAuditLogStats(c *gin.Context) {
	stats, err := h.adminService.GetAuditLogStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load audit log stats"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": stats})
}

// GetAuditLogs returns the list of audit logs
func (h *AdminHandler) GetAuditLogs(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	search := c.Query("search")
	actionType := c.Query("actionType")
	actorRole := c.Query("actorRole")

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 {
		limit = 10
	}

	logs, total, err := h.adminService.GetAuditLogs(page, limit, search, actionType, actorRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   logs,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// ExportAuditLogs generates and downloads a CSV of the audit logs
func (h *AdminHandler) ExportAuditLogs(c *gin.Context) {
	logs, err := h.adminService.GetAllAuditLogsForExport()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load logs for export"})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment;filename=audit_logs.csv")

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"Timestamp", "Actor Name", "Actor Role", "Action Type", "Target Data", "IP Address", "Details"})

	for _, log := range logs {
		writer.Write([]string{
			log.Timestamp.Format("2006-01-02 15:04:05"),
			log.ActorName,
			log.ActorRole,
			log.ActionType,
			log.TargetData,
			log.IPAddress,
			log.Details,
		})
	}
}

// GetBudgetSavingsStats returns global stats for budgets and savings
func (h *AdminHandler) GetBudgetSavingsStats(c *gin.Context) {
	stats, err := h.adminService.GetBudgetSavingsStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load budget/savings stats"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": stats})
}

// GetAllUserBudgets returns a list of all user budgets
func (h *AdminHandler) GetAllUserBudgets(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 {
		limit = 10
	}

	budgets, total, err := h.adminService.GetAllUserBudgets(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user budgets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   budgets,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// GetAllUserSavingsGoals returns all user savings goals
func (h *AdminHandler) GetAllUserSavingsGoals(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 {
		limit = 10
	}

	goals, total, err := h.adminService.GetAllUserSavingsGoals(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user savings goals"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   goals,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// GetInvestmentStats returns stats for investments
func (h *AdminHandler) GetInvestmentStats(c *gin.Context) {
	stats, err := h.adminService.GetInvestmentStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load investment stats"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": stats})
}

// GetAllInvestments returns the list of investments
func (h *AdminHandler) GetAllInvestments(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	search := c.Query("search")
	status := c.Query("status")

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 {
		limit = 10
	}

	investments, total, err := h.adminService.GetAllInvestments(page, limit, search, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch investments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   investments,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// ExportInvestments generates and downloads a CSV of the investments
func (h *AdminHandler) ExportInvestments(c *gin.Context) {
	investments, err := h.adminService.GetAllInvestmentsForExport()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load investments for export"})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment;filename=investments.csv")

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"ID", "User", "Asset Name", "Asset Symbol", "Invested", "Current Value", "Change %", "Status", "Date"})

	for _, inv := range investments {
		writer.Write([]string{
			fmt.Sprintf("%v", inv["id"]),
			fmt.Sprintf("%v", inv["user_name"]),
			fmt.Sprintf("%v", inv["asset_name"]),
			fmt.Sprintf("%v", inv["asset_symbol"]),
			fmt.Sprintf("%.2f", inv["invested"]),
			fmt.Sprintf("%.2f", inv["current_value"]),
			fmt.Sprintf("%.2f%%", inv["change_percent"]),
			fmt.Sprintf("%v", inv["status"]),
			fmt.Sprintf("%v", inv["date"]),
		})
	}
}

// GetAnalyticsAndReports returns data for the Analytics page
func (h *AdminHandler) GetAnalyticsAndReports(c *gin.Context) {
	data, err := h.adminService.GetAnalyticsAndReports()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load analytics"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": data})
}

// ExportAnalyticsReport generates a CSV report for analytics
func (h *AdminHandler) ExportAnalyticsReport(c *gin.Context) {
	data, err := h.adminService.GetAnalyticsAndReports()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report"})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment;filename=analytics_report.csv")

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// In a real scenario, this would generate a complex PDF or multi-sheet CSV.
	// For now, we'll export the top spenders as a representative CSV report.
	writer.Write([]string{"Top Spenders Report"})
	writer.Write([]string{"Name", "Total Spent"})

	if topSpenders, ok := data["top_spenders"].([]map[string]interface{}); ok {
		for _, ts := range topSpenders {
			writer.Write([]string{
				fmt.Sprintf("%v", ts["name"]),
				fmt.Sprintf("%.2f", ts["total_spent"]),
			})
		}
	}
}

// Login handles admin login
func (h *AdminHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	admin, err := h.adminService.LoginAdmin(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Create JWT token for admin
	tokenString, jti, err := auth.CreateToken(admin.ID.Hex(), admin.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Save token session (similar to LoginHandler)
	collection := config.GetCollection(config.DB, "active_tokens")
	_, _ = collection.InsertOne(context.TODO(), bson.M{
		"jti":       jti,
		"user_id":   admin.ID.Hex(),
		"username":  admin.Username,
		"role":      "admin",
		"expiresAt": time.Now().Add(2 * time.Hour),
	})

	c.JSON(http.StatusOK, gin.H{
		"status":     "success",
		"token":      tokenString,
		"expires_in": 7200,
		"token_type": "Bearer",
		"admin": gin.H{
			"id":       admin.ID.Hex(),
			"username": admin.Username,
			"email":    admin.Email,
		},
	})
}
