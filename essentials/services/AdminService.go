package services

import (
	"context"
	"errors"
	models "financeapi/essentials/models"
	"financeapi/essentials/utils"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AdminService struct {
	userCollection           *mongo.Collection
	transactionCollection    *mongo.Collection
	alertCollection          *mongo.Collection
	auditLogCollection       *mongo.Collection
	budgetCollection         *mongo.Collection
	categoryBudgetCollection *mongo.Collection
	savingsGoalCollection    *mongo.Collection
	investmentCollection     *mongo.Collection
	accountCollection        *mongo.Collection
	adminCollection          *mongo.Collection
}

func NewAdminService(client *mongo.Client, dbName string) *AdminService {
	db := client.Database(dbName)
	return &AdminService{
		userCollection:           db.Collection("users"),
		transactionCollection:    db.Collection("Transaction"),
		alertCollection:          db.Collection("system_alerts"),
		auditLogCollection:       db.Collection("audit_logs"),
		budgetCollection:         db.Collection("monthly_budget"),
		categoryBudgetCollection: db.Collection("category_budget"),
		savingsGoalCollection:    db.Collection("savings_goals"),
		investmentCollection:     db.Collection("investments"),
		accountCollection:        db.Collection("accounts"),
		adminCollection:          db.Collection("admins"),
	}
}

// GetDashboardStats returns data for the main admin dashboard
func (s *AdminService) GetDashboardStats() (map[string]interface{}, error) {
	now := time.Now()
	firstDayOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	firstDayOfLastMonth := firstDayOfThisMonth.AddDate(0, -1, 0)

	// 1. QUICK STATS (Total Users, Transactions, Revenue, Active Sessions)
	totalUsers, _ := s.userCollection.CountDocuments(context.TODO(), bson.M{})
	usersLastMonth, _ := s.userCollection.CountDocuments(context.TODO(), bson.M{"created_at": bson.M{"$lt": firstDayOfThisMonth}})

	// Mocking active sessions based on is_active for now
	activeSessions, _ := s.userCollection.CountDocuments(context.TODO(), bson.M{"is_active": true})

	// Transactions & Revenue (This month vs Last Month)
	pipelineThisMonth := mongo.Pipeline{
		{{"$match", bson.D{{"date", bson.D{{"$gte", firstDayOfThisMonth}}}}}},
		{{"$group", bson.D{
			{"_id", nil},
			{"total_revenue", bson.D{{"$sum", "$amount"}}},
			{"count", bson.D{{"$sum", 1}}},
		}}},
	}

	pipelineLastMonth := mongo.Pipeline{
		{{"$match", bson.D{
			{"date", bson.D{{"$gte", firstDayOfLastMonth}, {"$lt", firstDayOfThisMonth}}},
		}}},
		{{"$group", bson.D{
			{"_id", nil},
			{"total_revenue", bson.D{{"$sum", "$amount"}}},
			{"count", bson.D{{"$sum", 1}}},
		}}},
	}

	type AggResult struct {
		TotalRevenue float64 `bson:"total_revenue"`
		Count        int64   `bson:"count"`
	}

	var thisMonthStats, lastMonthStats AggResult

	cursorThis, _ := s.transactionCollection.Aggregate(context.TODO(), pipelineThisMonth)
	if cursorThis.Next(context.TODO()) {
		cursorThis.Decode(&thisMonthStats)
	}

	cursorLast, _ := s.transactionCollection.Aggregate(context.TODO(), pipelineLastMonth)
	if cursorLast.Next(context.TODO()) {
		cursorLast.Decode(&lastMonthStats)
	}

	calcPercentChange := func(current, previous float64) float64 {
		if previous == 0 {
			if current > 0 {
				return 100.0
			}
			return 0.0
		}
		return ((current - previous) / previous) * 100.0
	}

	userGrowth := calcPercentChange(float64(totalUsers), float64(usersLastMonth))
	txGrowth := calcPercentChange(float64(thisMonthStats.Count), float64(lastMonthStats.Count))
	revGrowth := calcPercentChange(thisMonthStats.TotalRevenue, lastMonthStats.TotalRevenue)

	// Just mock session growth for UI purposes since we don't track historical sessions right now
	sessionGrowth := 2.5

	// Generate dummy history for mini charts
	generateDummyHistory := func(baseValue float64) map[string]interface{} {
		return map[string]interface{}{
			"daily": []map[string]interface{}{
				{"date": now.AddDate(0, 0, -6).Format("2006-01-02"), "value": baseValue * 0.8},
				{"date": now.AddDate(0, 0, -5).Format("2006-01-02"), "value": baseValue * 0.9},
				{"date": now.AddDate(0, 0, -4).Format("2006-01-02"), "value": baseValue * 0.85},
				{"date": now.AddDate(0, 0, -3).Format("2006-01-02"), "value": baseValue * 1.1},
				{"date": now.AddDate(0, 0, -2).Format("2006-01-02"), "value": baseValue * 1.05},
				{"date": now.AddDate(0, 0, -1).Format("2006-01-02"), "value": baseValue * 1.2},
				{"date": now.Format("2006-01-02"), "value": baseValue},
			},
			"weekly": []map[string]interface{}{
				{"date": "Week 1", "value": baseValue * 4},
				{"date": "Week 2", "value": baseValue * 4.2},
				{"date": "Week 3", "value": baseValue * 3.8},
				{"date": "Week 4", "value": baseValue * 4.5},
			},
			"monthly": []map[string]interface{}{
				{"date": "Jan", "value": baseValue * 15},
				{"date": "Feb", "value": baseValue * 16},
				{"date": "Mar", "value": baseValue * 18},
				{"date": "Apr", "value": baseValue * 17},
			},
			"yearly": []map[string]interface{}{
				{"date": "2023", "value": baseValue * 180},
				{"date": "2024", "value": baseValue * 210},
			},
		}
	}

	quickStats := map[string]interface{}{
		"total_users":        map[string]interface{}{"value": totalUsers, "change_percent": userGrowth, "history": generateDummyHistory(float64(totalUsers))},
		"total_transactions": map[string]interface{}{"value": thisMonthStats.Count, "change_percent": txGrowth, "history": generateDummyHistory(float64(thisMonthStats.Count))},
		"total_revenue":      map[string]interface{}{"value": thisMonthStats.TotalRevenue, "change_percent": revGrowth, "history": generateDummyHistory(thisMonthStats.TotalRevenue)},
		"active_sessions":    map[string]interface{}{"value": activeSessions, "change_percent": sessionGrowth, "history": generateDummyHistory(float64(activeSessions))},
	}

	// 2. REVENUE & TRANSACTIONS OVERVIEW (Chart Data - Daily for current week)
	startOfWeek := now.AddDate(0, 0, -int(now.Weekday())) // Assuming Sunday start
	chartPipeline := mongo.Pipeline{
		{{"$match", bson.D{{"date", bson.D{{"$gte", startOfWeek}}}}}},
		{{"$group", bson.D{
			{"_id", bson.D{{"$dayOfWeek", "$date"}}},
			{"revenue", bson.D{{"$sum", "$amount"}}},
			{"transactions", bson.D{{"$sum", 1}}},
		}}},
		{{"$sort", bson.D{{"_id", 1}}}},
	}

	chartCursor, _ := s.transactionCollection.Aggregate(context.TODO(), chartPipeline)
	var chartData []bson.M
	chartCursor.All(context.TODO(), &chartData)

	if len(chartData) == 0 {
		chartData = []bson.M{
			{"_id": 1, "revenue": 1200, "transactions": 15},
			{"_id": 2, "revenue": 1800, "transactions": 22},
			{"_id": 3, "revenue": 900, "transactions": 10},
			{"_id": 4, "revenue": 2400, "transactions": 30},
			{"_id": 5, "revenue": 3100, "transactions": 45},
			{"_id": 6, "revenue": 2800, "transactions": 40},
			{"_id": 7, "revenue": 1500, "transactions": 18},
		}
	}

	// 3. TOP CATEGORIES (Pie Chart)
	catPipeline := mongo.Pipeline{
		{{"$group", bson.D{
			{"_id", "$category"},
			{"count", bson.D{{"$sum", 1}}},
		}}},
		{{"$sort", bson.D{{"count", -1}}}},
		{{"$limit", 4}},
	}

	catCursor, _ := s.transactionCollection.Aggregate(context.TODO(), catPipeline)
	var topCategories []bson.M
	catCursor.All(context.TODO(), &topCategories)

	// Calculate percentages for categories
	totalCatCount := 0
	for _, cat := range topCategories {
		if count, ok := cat["count"].(int32); ok {
			totalCatCount += int(count)
		}
	}

	for i, cat := range topCategories {
		if count, ok := cat["count"].(int32); ok && totalCatCount > 0 {
			topCategories[i]["percentage"] = (float64(count) / float64(totalCatCount)) * 100
		}
	}

	if len(topCategories) == 0 {
		topCategories = []bson.M{
			{"_id": "Food", "count": 120, "percentage": 40.0},
			{"_id": "Transport", "count": 90, "percentage": 30.0},
			{"_id": "Entertainment", "count": 60, "percentage": 20.0},
			{"_id": "Shopping", "count": 30, "percentage": 10.0},
		}
	}

	// 4. RECENT ACTIVITIES (Last 5 Audit Logs)
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}).SetLimit(5)
	auditCursor, _ := s.auditLogCollection.Find(context.TODO(), bson.M{}, opts)
	var recentActivities []models.AuditLog
	auditCursor.All(context.TODO(), &recentActivities)

	if len(recentActivities) == 0 {
		recentActivities = []models.AuditLog{
			{ActionType: "Login", ActorName: "Admin 1", TargetData: "System", Details: "Logged in successfully", Timestamp: now.Add(-1 * time.Hour)},
			{ActionType: "Update", ActorName: "Admin 2", TargetData: "User #123", Details: "Updated user profile", Timestamp: now.Add(-2 * time.Hour)},
			{ActionType: "Delete", ActorName: "Admin 1", TargetData: "Transaction #456", Details: "Deleted spam transaction", Timestamp: now.Add(-3 * time.Hour)},
		}
	}

	return map[string]interface{}{
		"quick_stats":       quickStats,
		"chart_data":        chartData,
		"top_categories":    topCategories,
		"recent_activities": recentActivities,
	}, nil
}

// GetUsers returns list of users with search, filter and pagination, including balance
func (s *AdminService) GetUsers(page, limit int, search, status string) ([]map[string]interface{}, int64, error) {
	filter := bson.M{}

	if search != "" {
		filter["$or"] = []bson.M{
			{"username": bson.M{"$regex": search, "$options": "i"}},
			{"email": bson.M{"$regex": search, "$options": "i"}},
		}
	}

	if status != "" && status != "All Status" {
		if status == "Active" {
			filter["is_active"] = true
			filter["is_email_verified"] = true
		} else if status == "Disabled" {
			filter["is_active"] = false
		} else if status == "Pending" {
			filter["is_active"] = true
			filter["is_email_verified"] = false
		}
	}

	total, err := s.userCollection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find()
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := s.userCollection.Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, 0, err
	}

	var users []models.User
	if err = cursor.All(context.TODO(), &users); err != nil {
		return nil, 0, err
	}

	var result []map[string]interface{}
	for _, u := range users {
		balance := 0.0
		accountCursor, err := s.accountCollection.Find(context.TODO(), bson.M{"user_id": u.ID})
		if err == nil {
			var accounts []models.Account
			accountCursor.All(context.TODO(), &accounts)
			for _, acc := range accounts {
				if acc.Type == models.AccountTypeCredit {
					balance -= acc.CurrentBalance
				} else {
					balance += acc.CurrentBalance
				}
			}
		}

		userStatus := "Active"
		if !u.IsActive {
			userStatus = "Disabled"
		} else if !u.IsEmailVerified {
			userStatus = "Pending"
		}

		result = append(result, map[string]interface{}{
			"id":        u.ID.Hex(),
			"name":      u.Username,
			"email":     u.Email,
			"status":    userStatus,
			"join_date": u.CreatedAt.Format("2006-01-02"),
			"balance":   balance,
		})
	}

	return result, total, nil
}

// GetUserDetail returns user basic info + stats
func (s *AdminService) GetUserDetail(userIDStr string) (map[string]interface{}, error) {
	id, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return nil, err
	}

	var user models.User
	err = s.userCollection.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&user)
	if err != nil {
		return nil, err
	}

	// Recent transactions
	opts := options.Find()
	opts.SetLimit(5)
	opts.SetSort(bson.D{{Key: "date", Value: -1}})

	cursor, err := s.transactionCollection.Find(context.TODO(), bson.M{"user_id": userIDStr}, opts)
	if err != nil {
		return nil, err
	}

	var recentTransactions []models.Transaction
	if err = cursor.All(context.TODO(), &recentTransactions); err != nil {
		return nil, err
	}

	totalTransactions, _ := s.transactionCollection.CountDocuments(context.TODO(), bson.M{"user_id": userIDStr})

	return map[string]interface{}{
		"user":                user,
		"total_transactions":  totalTransactions,
		"recent_transactions": recentTransactions,
	}, nil
}

// DisableUser disables a user (cannot login)
func (s *AdminService) DisableUser(userIDStr string) error {
	id, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return err
	}

	_, err = s.userCollection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id},
		bson.M{"$set": bson.M{
			"is_active":  false,
			"updated_at": time.Now(),
		}},
	)
	return err
}

// DeleteUser deletes a user from the DB
func (s *AdminService) DeleteUser(userIDStr string) error {
	id, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return err
	}

	_, err = s.userCollection.DeleteOne(context.TODO(), bson.M{"_id": id})
	return err
}

// GetRecentTransactions gets all recent transactions with optional date filter
func (s *AdminService) GetRecentTransactions(fromDate, toDate string, limit int) ([]models.Transaction, error) {
	filter := bson.M{}

	if fromDate != "" || toDate != "" {
		dateFilter := bson.M{}
		if fromDate != "" {
			fromTime, err := time.Parse("2006-01-02", fromDate)
			if err == nil {
				dateFilter["$gte"] = fromTime
			}
		}
		if toDate != "" {
			toTime, err := time.Parse("2006-01-02", toDate)
			if err == nil {
				toTime = toTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
				dateFilter["$lte"] = toTime
			}
		}
		filter["date"] = dateFilter
	}

	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	opts.SetSort(bson.D{{Key: "date", Value: -1}})

	cursor, err := s.transactionCollection.Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	var transactions []models.Transaction
	if err = cursor.All(context.TODO(), &transactions); err != nil {
		return nil, err
	}

	if len(transactions) == 0 {
		return []models.Transaction{}, nil
	}
	return transactions, nil
}

// DeleteTransaction deletes a problematic transaction
func (s *AdminService) DeleteTransaction(txIDStr string) error {
	id, err := primitive.ObjectIDFromHex(txIDStr)
	if err != nil {
		return err
	}

	_, err = s.transactionCollection.DeleteOne(context.TODO(), bson.M{"_id": id})
	return err
}

// GetAlertStats returns counts for alerts and logs
func (s *AdminService) GetAlertStats() (map[string]interface{}, error) {
	totalAlerts, err := s.alertCollection.CountDocuments(context.TODO(), bson.M{"type": "system_alert"})
	if err != nil {
		return nil, err
	}

	failedActions, err := s.alertCollection.CountDocuments(context.TODO(), bson.M{"type": "failed_action"})
	if err != nil {
		return nil, err
	}

	unread, err := s.alertCollection.CountDocuments(context.TODO(), bson.M{"is_read": false})
	if err != nil {
		return nil, err
	}

	canRetry, err := s.alertCollection.CountDocuments(context.TODO(), bson.M{"can_retry": true})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_alerts":   totalAlerts,
		"failed_actions": failedActions,
		"unread":         unread,
		"can_retry":      canRetry,
	}, nil
}

// GetAlerts returns a list of alerts based on type (system_alert or failed_action) with pagination
func (s *AdminService) GetAlerts(alertType string, page, limit int) ([]models.SystemAlert, int64, error) {
	filter := bson.M{}
	if alertType != "" {
		filter["type"] = alertType
	}

	total, err := s.alertCollection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find()
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := s.alertCollection.Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, 0, err
	}

	var alerts []models.SystemAlert
	if err = cursor.All(context.TODO(), &alerts); err != nil {
		return nil, 0, err
	}

	return alerts, total, nil
}

// MarkAlertAsRead marks an alert as read
func (s *AdminService) MarkAlertAsRead(alertIDStr string) error {
	id, err := primitive.ObjectIDFromHex(alertIDStr)
	if err != nil {
		return err
	}

	_, err = s.alertCollection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"is_read": true, "updated_at": time.Now()}},
	)
	return err
}

// CreateAlert internally creates an alert (can be used by other services)
func (s *AdminService) CreateAlert(alert models.SystemAlert) error {
	alert.CreatedAt = time.Now()
	alert.UpdatedAt = time.Now()
	_, err := s.alertCollection.InsertOne(context.TODO(), alert)
	return err
}

// GetAuditLogStats returns counts for audit logs
func (s *AdminService) GetAuditLogStats() (map[string]interface{}, error) {
	totalLogs, err := s.auditLogCollection.CountDocuments(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}

	adminActions, err := s.auditLogCollection.CountDocuments(context.TODO(), bson.M{"actor_role": "admin"})
	if err != nil {
		return nil, err
	}

	userActions, err := s.auditLogCollection.CountDocuments(context.TODO(), bson.M{"actor_role": "user"})
	if err != nil {
		return nil, err
	}

	deleteActions, err := s.auditLogCollection.CountDocuments(context.TODO(), bson.M{"action_type": "Delete"})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_logs":     totalLogs,
		"admin_actions":  adminActions,
		"user_actions":   userActions,
		"delete_actions": deleteActions,
	}, nil
}

// GetAuditLogs returns a list of audit logs with search and filters
func (s *AdminService) GetAuditLogs(page, limit int, search, actionType, actorRole string) ([]models.AuditLog, int64, error) {
	filter := bson.M{}

	if search != "" {
		filter["$or"] = []bson.M{
			{"actor_name": bson.M{"$regex": search, "$options": "i"}},
			{"target_data": bson.M{"$regex": search, "$options": "i"}},
			{"ip_address": bson.M{"$regex": search, "$options": "i"}},
		}
	}

	if actionType != "" && actionType != "All Actions" {
		filter["action_type"] = actionType
	}

	if actorRole != "" && actorRole != "All Actors" {
		filter["actor_role"] = actorRole
	}

	total, err := s.auditLogCollection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find()
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))
	opts.SetSort(bson.D{{Key: "timestamp", Value: -1}})

	cursor, err := s.auditLogCollection.Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, 0, err
	}

	var logs []models.AuditLog
	if err = cursor.All(context.TODO(), &logs); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetAllAuditLogsForExport returns all audit logs for CSV export
func (s *AdminService) GetAllAuditLogsForExport() ([]models.AuditLog, error) {
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	cursor, err := s.auditLogCollection.Find(context.TODO(), bson.M{}, opts)
	if err != nil {
		return nil, err
	}

	var logs []models.AuditLog
	if err = cursor.All(context.TODO(), &logs); err != nil {
		return nil, err
	}

	return logs, nil
}

// CreateAuditLog saves an action to the audit log
func (s *AdminService) CreateAuditLog(log models.AuditLog) error {
	log.Timestamp = time.Now()
	_, err := s.auditLogCollection.InsertOne(context.TODO(), log)
	return err
}

// GetBudgetSavingsStats returns global stats for budgets and savings
func (s *AdminService) GetBudgetSavingsStats() (map[string]interface{}, error) {
	totalBudgets, err := s.budgetCollection.CountDocuments(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}

	totalSavingsGoals, err := s.savingsGoalCollection.CountDocuments(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}

	// Calculate over budget count (global)
	// This is a bit complex as we need to compare spent with limit across all users
	// For simplicity in this demo, let's use a simpler count or aggregation
	overBudgetCount := 0
	cursor, err := s.categoryBudgetCollection.Find(context.TODO(), bson.M{})
	if err == nil {
		var budgets []models.CategoryBudget
		cursor.All(context.TODO(), &budgets)
		for _, b := range budgets {
			if b.Spent > b.Limit {
				overBudgetCount++
			}
		}
	}

	return map[string]interface{}{
		"total_budgets": totalBudgets,
		"over_budget":   overBudgetCount,
		"savings_goals": totalSavingsGoals,
	}, nil
}

// GetAllUserBudgets returns a list of all budgets with user details
func (s *AdminService) GetAllUserBudgets(page, limit int) ([]map[string]interface{}, int64, error) {
	total, err := s.categoryBudgetCollection.CountDocuments(context.TODO(), bson.M{})
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find()
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))
	opts.SetSort(bson.D{{Key: "month", Value: -1}})

	cursor, err := s.categoryBudgetCollection.Find(context.TODO(), bson.M{}, opts)
	if err != nil {
		return nil, 0, err
	}

	var budgets []models.CategoryBudget
	cursor.All(context.TODO(), &budgets)

	var result []map[string]interface{}
	for _, b := range budgets {
		// Get username
		var user models.User
		objID, _ := primitive.ObjectIDFromHex(b.UserID)
		s.userCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&user)

		remaining := b.Limit - b.Spent
		percentage := 0.0
		if b.Limit > 0 {
			percentage = (b.Spent / b.Limit) * 100
		}

		status := "Safe"
		if percentage >= 100 {
			status = "Over Budget"
		} else if percentage >= 90 {
			status = "Near Limit"
		}

		result = append(result, map[string]interface{}{
			"user_name":       user.Username,
			"category":        b.Category,
			"month":           b.Month,
			"spent":           b.Spent,
			"limit":           b.Limit,
			"remaining":       remaining,
			"percentage_used": percentage,
			"status":          status,
		})
	}

	return result, total, nil
}

// GetAllUserSavingsGoals returns all savings goals with user info
func (s *AdminService) GetAllUserSavingsGoals(page, limit int) ([]map[string]interface{}, int64, error) {
	total, err := s.savingsGoalCollection.CountDocuments(context.TODO(), bson.M{})
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find()
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := s.savingsGoalCollection.Find(context.TODO(), bson.M{}, opts)
	if err != nil {
		return nil, 0, err
	}

	var goals []models.SavingsGoal
	cursor.All(context.TODO(), &goals)

	var result []map[string]interface{}
	for _, g := range goals {
		var user models.User
		s.userCollection.FindOne(context.TODO(), bson.M{"_id": g.UserID}).Decode(&user)

		percentage := 0.0
		if g.TargetAmount > 0 {
			percentage = (g.CurrentAmount / g.TargetAmount) * 100
		}

		result = append(result, map[string]interface{}{
			"user_name":       user.Username,
			"goal_name":       g.Name,
			"target_date":     g.TargetDate.Format("2006-01-02"),
			"current_amount":  g.CurrentAmount,
			"target_amount":   g.TargetAmount,
			"percentage_used": percentage,
			"status":          "On track", // Simple default
		})
	}

	return result, total, nil
}

// GetInvestmentStats returns global stats for investments
func (s *AdminService) GetInvestmentStats() (map[string]interface{}, error) {
	cursor, err := s.investmentCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}

	var investments []models.Investment
	if err = cursor.All(context.TODO(), &investments); err != nil {
		return nil, err
	}

	totalInvested := 0.0
	currentValue := 0.0

	for _, inv := range investments {
		totalInvested += inv.TotalCost
		currentValue += inv.TotalValue
	}

	totalGainLoss := currentValue - totalInvested
	performance := 0.0
	if totalInvested > 0 {
		performance = (totalGainLoss / totalInvested) * 100
	}

	return map[string]interface{}{
		"total_invested":  totalInvested,
		"current_value":   currentValue,
		"total_gain_loss": totalGainLoss,
		"performance":     performance,
	}, nil
}

// GetAllInvestments returns a paginated list of all investments with search and filters
func (s *AdminService) GetAllInvestments(page, limit int, search, status string) ([]map[string]interface{}, int64, error) {
	filter := bson.M{}

	if search != "" {
		filter["$or"] = []bson.M{
			{"name": bson.M{"$regex": search, "$options": "i"}},
			{"symbol": bson.M{"$regex": search, "$options": "i"}},
		}
	}

	if status != "" && status != "All Status" {
		isActive := status == "Active"
		filter["is_active"] = isActive
	}

	total, err := s.investmentCollection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find()
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := s.investmentCollection.Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, 0, err
	}

	var investments []models.Investment
	if err = cursor.All(context.TODO(), &investments); err != nil {
		return nil, 0, err
	}

	var result []map[string]interface{}
	for _, inv := range investments {
		var user models.User
		s.userCollection.FindOne(context.TODO(), bson.M{"_id": inv.UserID}).Decode(&user)

		userName := user.Username
		if userName == "" {
			userName = "Unknown User"
		}

		statusStr := "Active"
		if !inv.IsActive {
			statusStr = "Closed"
		}

		result = append(result, map[string]interface{}{
			"id":             inv.ID.Hex(),
			"user_name":      userName,
			"asset_name":     inv.Name,
			"asset_symbol":   inv.Symbol,
			"invested":       inv.TotalCost,
			"current_value":  inv.TotalValue,
			"change_percent": inv.GainLossPercent,
			"status":         statusStr,
			"date":           inv.CreatedAt.Format("2006-01-02"),
		})
	}

	return result, total, nil
}

// GetAllInvestmentsForExport returns all investments for CSV export
func (s *AdminService) GetAllInvestmentsForExport() ([]map[string]interface{}, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := s.investmentCollection.Find(context.TODO(), bson.M{}, opts)
	if err != nil {
		return nil, err
	}

	var investments []models.Investment
	if err = cursor.All(context.TODO(), &investments); err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, inv := range investments {
		var user models.User
		s.userCollection.FindOne(context.TODO(), bson.M{"_id": inv.UserID}).Decode(&user)

		userName := user.Username
		if userName == "" {
			userName = "Unknown User"
		}

		statusStr := "Active"
		if !inv.IsActive {
			statusStr = "Closed"
		}

		result = append(result, map[string]interface{}{
			"id":             inv.ID.Hex(),
			"user_name":      userName,
			"asset_name":     inv.Name,
			"asset_symbol":   inv.Symbol,
			"invested":       inv.TotalCost,
			"current_value":  inv.TotalValue,
			"change_percent": inv.GainLossPercent,
			"status":         statusStr,
			"date":           inv.CreatedAt.Format("2006-01-02"),
		})
	}

	return result, nil
}

// GetAllTransactions returns a paginated list of all transactions with search and filter
func (s *AdminService) GetAllTransactions(page, limit int, search string) ([]map[string]interface{}, int64, error) {
	filter := bson.M{}

	if search != "" {
		filter["$or"] = []bson.M{
			{"category": bson.M{"$regex": search, "$options": "i"}},
			{"type": bson.M{"$regex": search, "$options": "i"}},
			{"description": bson.M{"$regex": search, "$options": "i"}},
		}
	}

	total, err := s.transactionCollection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find()
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))
	opts.SetSort(bson.D{{Key: "date", Value: -1}})

	cursor, err := s.transactionCollection.Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, 0, err
	}

	var transactions []models.Transaction
	if err = cursor.All(context.TODO(), &transactions); err != nil {
		return nil, 0, err
	}

	var result []map[string]interface{}
	for _, tx := range transactions {
		var user models.User
		if objID, err := primitive.ObjectIDFromHex(tx.User_id); err == nil {
			s.userCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&user)
		}
		
		userName := user.Username
		if userName == "" {
			userName = "Unknown User"
		}

		result = append(result, map[string]interface{}{
			"id":          tx.ID.Hex(),
			"user_name":   userName,
			"amount":      tx.Amount,
			"category":    tx.Category,
			"description": tx.Description,
			"date":        tx.Date.Format("2006-01-02"),
		})
	}

	return result, total, nil
}

// LoginAdmin handles admin authentication
func (s *AdminService) LoginAdmin(email, password string) (models.Admin, error) {
	var admin models.Admin
	err := s.adminCollection.FindOne(context.TODO(), bson.M{"email": email}).Decode(&admin)
	if err != nil {
		return models.Admin{}, err
	}

	if !utils.CheckPassword(password, admin.Password) {
		return models.Admin{}, errors.New("invalid password")
	}

	if !admin.IsActive {
		return models.Admin{}, errors.New("account is disabled")
	}

	return admin, nil
}

// GetAnalyticsAndReports returns data for the Analytics & Reports page
func (s *AdminService) GetAnalyticsAndReports() (map[string]interface{}, error) {
	// 1. Top Spenders (Top 6 users by transaction volume)
	topSpendersPipeline := mongo.Pipeline{
		{{"$group", bson.D{
			{"_id", "$user_id"},
			{"total_spent", bson.D{{"$sum", "$amount"}}},
		}}},
		{{"$sort", bson.D{{"total_spent", -1}}}},
		{{"$limit", 6}},
	}

	spendersCursor, _ := s.transactionCollection.Aggregate(context.TODO(), topSpendersPipeline)
	var topSpendersRaw []bson.M
	spendersCursor.All(context.TODO(), &topSpendersRaw)

	var topSpenders []map[string]interface{}
	for _, ts := range topSpendersRaw {
		var user models.User
		if idStr, ok := ts["_id"].(string); ok {
			objID, _ := primitive.ObjectIDFromHex(idStr)
			s.userCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&user)
		}

		name := "Unknown User"
		if user.Username != "" {
			name = user.Username
		}

		topSpenders = append(topSpenders, map[string]interface{}{
			"name":        name,
			"total_spent": ts["total_spent"],
		})
	}

	// 2. Category Distribution
	catPipeline := mongo.Pipeline{
		{{"$group", bson.D{
			{"_id", "$category"},
			{"amount", bson.D{{"$sum", "$amount"}}},
		}}},
		{{"$sort", bson.D{{"amount", -1}}}},
		{{"$limit", 4}},
	}

	catCursor, _ := s.transactionCollection.Aggregate(context.TODO(), catPipeline)
	var categoryDistribution []bson.M
	catCursor.All(context.TODO(), &categoryDistribution)

	totalCatAmount := 0.0
	for _, cat := range categoryDistribution {
		if amt, ok := cat["amount"].(float64); ok {
			totalCatAmount += amt
		}
	}

	for i, cat := range categoryDistribution {
		if amt, ok := cat["amount"].(float64); ok && totalCatAmount > 0 {
			categoryDistribution[i]["percentage"] = (amt / totalCatAmount) * 100
		}
	}

	// 3. User Growth Trend (Last 6 Months)
	now := time.Now()
	sixMonthsAgo := time.Date(now.Year(), now.Month()-5, 1, 0, 0, 0, 0, now.Location())

	userGrowthPipeline := mongo.Pipeline{
		{{"$match", bson.D{{"created_at", bson.D{{"$gte", sixMonthsAgo}}}}}},
		{{"$group", bson.D{
			{"_id", bson.D{{"$month", "$created_at"}}},
			{"count", bson.D{{"$sum", 1}}},
		}}},
		{{"$sort", bson.D{{"_id", 1}}}},
	}

	userCursor, _ := s.userCollection.Aggregate(context.TODO(), userGrowthPipeline)
	var userGrowth []bson.M
	userCursor.All(context.TODO(), &userGrowth)

	// Cumulative sum for user growth
	cumulativeUsers := 0
	// Base users before 6 months ago
	baseUsers, _ := s.userCollection.CountDocuments(context.TODO(), bson.M{"created_at": bson.M{"$lt": sixMonthsAgo}})
	cumulativeUsers = int(baseUsers)

	for i, ug := range userGrowth {
		if count, ok := ug["count"].(int32); ok {
			cumulativeUsers += int(count)
			userGrowth[i]["total_users"] = cumulativeUsers
		}
	}

	// 4. Bottom Stats
	totalVolumePipeline := mongo.Pipeline{
		{{"$group", bson.D{
			{"_id", nil},
			{"total_volume", bson.D{{"$sum", "$amount"}}},
			{"transaction_count", bson.D{{"$sum", 1}}},
		}}},
	}

	volCursor, _ := s.transactionCollection.Aggregate(context.TODO(), totalVolumePipeline)
	var volResult []bson.M
	volCursor.All(context.TODO(), &volResult)

	totalVolume := 0.0
	avgTransaction := 0.0
	if len(volResult) > 0 {
		if vol, ok := volResult[0]["total_volume"].(float64); ok {
			totalVolume = vol
		}
		if count, ok := volResult[0]["transaction_count"].(int32); ok && count > 0 {
			avgTransaction = totalVolume / float64(count)
		}
	}

	totalUsers, _ := s.userCollection.CountDocuments(context.TODO(), bson.M{})

	// 5. Mocked Data for UI completeness (Price Trends & Savings Progress which require historical snapshots)
	priceTrends := []map[string]interface{}{
		{"month": "Jan", "Stocks": 120, "Cryptocurrency": 50, "Commodities": 80},
		{"month": "Feb", "Stocks": 135, "Cryptocurrency": 45, "Commodities": 85},
		{"month": "Mar", "Stocks": 140, "Cryptocurrency": 60, "Commodities": 82},
		{"month": "Apr", "Stocks": 155, "Cryptocurrency": 80, "Commodities": 90},
		{"month": "May", "Stocks": 165, "Cryptocurrency": 75, "Commodities": 95},
		{"month": "Jun", "Stocks": 170, "Cryptocurrency": 90, "Commodities": 100},
	}

	savingsProgress := []map[string]interface{}{
		{"week": "Week 1", "target": 5000, "actual": 4800},
		{"week": "Week 2", "target": 10000, "actual": 9800},
		{"week": "Week 3", "target": 15000, "actual": 14900},
		{"week": "Week 4", "target": 20000, "actual": 20200},
	}

	return map[string]interface{}{
		"top_spenders":          topSpenders,
		"category_distribution": categoryDistribution,
		"user_growth_trend":     userGrowth,
		"price_trends":          priceTrends,
		"savings_progress":      savingsProgress,
		"bottom_stats": map[string]interface{}{
			"total_volume":    totalVolume,
			"avg_transaction": avgTransaction,
			"total_users":     totalUsers,
			"success_rate":    97.3, // Mocked for UI
		},
	}, nil
}
