package services

import (
	"context"
	"fmt"
	"financeapi/essentials/models"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type NotificationService struct {
	client          *mongo.Client
	notificationCol *mongo.Collection
	userCol        *mongo.Collection
	// Service dependencies for notification checks
	billService    *BillReminderService
	debtService    *DebtService
	recurringService *RecurringTransactionService
	budgetService  *BudgetService
	txService      *TransactionService
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

func NewNotificationService(db *mongo.Database) *NotificationService {
	return &NotificationService{
		client:          db.Client(),
		notificationCol: db.Collection("user_notifications"),
		userCol:         db.Collection("users"),
		stopChan:        make(chan struct{}),
	}
}

// SetDependencies sets the service dependencies for notification checks
func (s *NotificationService) SetDependencies(
	billService *BillReminderService,
	debtService *DebtService,
	recurringService *RecurringTransactionService,
	budgetService *BudgetService,
	txService *TransactionService,
) {
	s.billService = billService
	s.debtService = debtService
	s.recurringService = recurringService
	s.budgetService = budgetService
	s.txService = txService
}

// CreateNotification creates a new in-app notification for a user
func (s *NotificationService) CreateNotification(ctx context.Context, userID primitive.ObjectID, title, message string, notifType models.NotificationType, link string) error {
	notification := models.UserNotification{
		UserID:    userID,
		Title:     title,
		Message:   message,
		Type:      notifType,
		IsRead:    false,
		Link:      link,
		CreatedAt: time.Now(),
	}

	_, err := s.notificationCol.InsertOne(ctx, notification)
	return err
}

// GetUserNotifications gets all notifications for a specific user
func (s *NotificationService) GetUserNotifications(ctx context.Context, userID primitive.ObjectID, unreadOnly bool, limit int64) ([]models.UserNotification, error) {
	filter := bson.M{"user_id": userID}
	if unreadOnly {
		filter["is_read"] = false
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(limit)
	}

	cursor, err := s.notificationCol.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []models.UserNotification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, err
	}

	return notifications, nil
}

// MarkAsRead marks a specific notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, notificationID primitive.ObjectID, userID primitive.ObjectID) error {
	filter := bson.M{"_id": notificationID, "user_id": userID}
	update := bson.M{"$set": bson.M{"is_read": true}}

	_, err := s.notificationCol.UpdateOne(ctx, filter, update)
	return err
}

// MarkAllAsRead marks all notifications for a user as read
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID primitive.ObjectID) error {
	filter := bson.M{"user_id": userID, "is_read": false}
	update := bson.M{"$set": bson.M{"is_read": true}}

	_, err := s.notificationCol.UpdateMany(ctx, filter, update)
	return err
}

// GetUnreadCount gets the count of unread notifications for a user
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID primitive.ObjectID) (int64, error) {
	filter := bson.M{"user_id": userID, "is_read": false}
	return s.notificationCol.CountDocuments(ctx, filter)
}

// CheckAndCreateNotification checks if a similar notification exists (within timeWindow) before creating
// This prevents duplicate notifications from being spammed
func (s *NotificationService) CheckAndCreateNotification(ctx context.Context, userID primitive.ObjectID, title, message string, notifType models.NotificationType, link string, timeWindow time.Duration) error {
	// Check if similar notification exists within time window
	filter := bson.M{
		"user_id":   userID,
		"title":     title,
		"created_at": bson.M{"$gte": time.Now().Add(-timeWindow)},
	}

	count, _ := s.notificationCol.CountDocuments(ctx, filter)
	if count > 0 {
		return nil // Already exists, skip
	}

	return s.CreateNotification(ctx, userID, title, message, notifType, link)
}

// GetAllActiveUsers returns all active users
func (s *NotificationService) GetAllActiveUsers() ([]primitive.ObjectID, error) {
	cursor, err := s.userCol.Find(context.Background(), bson.M{
		"is_active": true,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var users []models.User
	if err := cursor.All(context.Background(), &users); err != nil {
		return nil, err
	}

	ids := make([]primitive.ObjectID, len(users))
	for i, u := range users {
		ids[i] = u.ID
	}
	return ids, nil
}

// CheckAllNotificationsForUser checks and creates all notifications for a single user
func (s *NotificationService) CheckAllNotificationsForUser(userID primitive.ObjectID) {
	ctx := context.Background()

	// 1. Check Bill Reminders
	if s.billService != nil {
		bills, _ := s.billService.GetDueBillReminders(userID)
		for _, bill := range bills {
			daysUntilDue := int(time.Until(bill.NextDueDate).Hours() / 24)
			var title, message string
			if daysUntilDue < 0 {
				title = fmt.Sprintf("Overdue: %s", bill.Name)
				message = fmt.Sprintf("Bill payment of %.2f is %d days overdue!", bill.Amount, -daysUntilDue)
			} else if daysUntilDue == 0 {
				title = fmt.Sprintf("Due Today: %s", bill.Name)
				message = fmt.Sprintf("Bill payment of %.2f is due today!", bill.Amount)
			} else if daysUntilDue == 1 {
				title = fmt.Sprintf("Due Tomorrow: %s", bill.Name)
				message = fmt.Sprintf("Bill payment of %.2f is due tomorrow!", bill.Amount)
			} else {
				title = fmt.Sprintf("Due in %d days: %s", daysUntilDue, bill.Name)
				message = fmt.Sprintf("Bill payment of %.2f is due in %d days", bill.Amount, daysUntilDue)
			}
			s.CheckAndCreateNotification(ctx, userID, title, message, models.NotifTypeBill, "/bills", 24*time.Hour)
		}

		// Also check overdue bills
		overdue, _ := s.billService.GetOverdueBillReminders(userID)
		for _, bill := range overdue {
			daysOverdue := int(-time.Until(bill.NextDueDate).Hours() / 24)
			title := fmt.Sprintf("Overdue: %s", bill.Name)
			message := fmt.Sprintf("Bill payment of %.2f is %d days overdue!", bill.Amount, daysOverdue)
			s.CheckAndCreateNotification(ctx, userID, title, message, models.NotifTypeBill, "/bills", 24*time.Hour)
		}
	}

	// 2. Check Debt Payments
	if s.debtService != nil {
		debts, _ := s.debtService.GetUserDebts(userID)
		now := time.Now()
		for _, debt := range debts {
			if debt.IsPaidOff {
				continue
			}
			daysUntilDue := int(time.Until(debt.NextPaymentDate).Hours() / 24)

			// Notify if payment is due today, tomorrow, or overdue (within 3 days)
			if daysUntilDue <= 0 && daysUntilDue >= -3 {
				var title, message string
				if daysUntilDue < 0 {
					title = fmt.Sprintf("Overdue: %s Payment", debt.Name)
					message = fmt.Sprintf("Your debt payment of %.2f to %s is %d days overdue!", debt.PaymentAmount, debt.Creditor, -daysUntilDue)
				} else if daysUntilDue == 0 {
					title = fmt.Sprintf("Due Today: %s Payment", debt.Name)
					message = fmt.Sprintf("Your debt payment of %.2f to %s is due today!", debt.PaymentAmount, debt.Creditor)
				} else {
					title = fmt.Sprintf("Due Tomorrow: %s Payment", debt.Name)
					message = fmt.Sprintf("Your debt payment of %.2f to %s is due tomorrow", debt.PaymentAmount, debt.Creditor)
				}
				s.CheckAndCreateNotification(ctx, userID, title, message, models.NotifTypeDebt, "/debt", 24*time.Hour)
			}

			// Early warning: 7 days before payment due
			if daysUntilDue == 7 {
				title := fmt.Sprintf("Reminder: %s Payment in 7 days", debt.Name)
				message := fmt.Sprintf("Your debt payment of %.2f to %s is due in 7 days. Remaining balance: %.2f", debt.PaymentAmount, debt.Creditor, debt.CurrentBalance)
				s.CheckAndCreateNotification(ctx, userID, title, message, models.NotifTypeDebt, "/debt", 24*time.Hour)
			}
		}
		_ = now // suppress unused variable warning
	}

	// 3. Check Recurring Transactions
	if s.recurringService != nil {
		// Due recurring
		dueRecurring, _ := s.recurringService.GetDueRecurringTransactions(userID)
		for _, recurring := range dueRecurring {
			var title, message string
			if recurring.Type == "income" {
				title = fmt.Sprintf("Due: %s Income", recurring.Name)
				message = fmt.Sprintf("Your recurring income of %.2f (%s) is ready to be recorded!", recurring.Amount, recurring.Frequency)
			} else {
				title = fmt.Sprintf("Due: %s Payment", recurring.Name)
				message = fmt.Sprintf("Your recurring payment of %.2f (%s) is due. Make sure you have enough balance!", recurring.Amount, recurring.Frequency)
			}
			s.CheckAndCreateNotification(ctx, userID, title, message, models.NotifTypeRecurring, "/recurring", 24*time.Hour)
		}

		// Recurring due tomorrow
		now := time.Now()
		tomorrow := now.AddDate(0, 0, 1)
		recurringList, _ := s.recurringService.GetActiveRecurringTransactions(userID)
		for _, recurring := range recurringList {
			nextRun := recurring.NextRunDate
			if nextRun.Year() == tomorrow.Year() && nextRun.Month() == tomorrow.Month() && nextRun.Day() == tomorrow.Day() {
				var title, message string
				if recurring.Type == "income" {
					title = fmt.Sprintf("Tomorrow: %s Income", recurring.Name)
					message = fmt.Sprintf("Your recurring income of %.2f (%s) will be processed tomorrow!", recurring.Amount, recurring.Frequency)
				} else {
					title = fmt.Sprintf("Tomorrow: %s Payment", recurring.Name)
					message = fmt.Sprintf("Your recurring payment of %.2f (%s) will be due tomorrow. Prepare your balance!", recurring.Amount, recurring.Frequency)
				}
				s.CheckAndCreateNotification(ctx, userID, title, message, models.NotifTypeRecurring, "/recurring", 24*time.Hour)
			}
		}
		_ = tomorrow // suppress unused variable warning
	}

	// 4. Check Budget Thresholds
	if s.budgetService != nil {
		// Check monthly budget
		currentMonth := time.Now().Format("2006-01")
		monthlyBudgetWithSpending, err := s.budgetService.GetBudgetWithSpending(userID.Hex(), currentMonth)
		if err == nil {
			percentageUsed := monthlyBudgetWithSpending["percentage_used"].(float64)
			s.checkBudgetThreshold(ctx, userID, "Monthly Budget", percentageUsed, currentMonth)
		}

		// Check category budgets
		categoryBudgets, err := s.budgetService.GetAllCategoryBudgetsWithSpending(userID.Hex(), currentMonth)
		if err == nil {
			for _, catBudgetData := range categoryBudgets {
				budget := catBudgetData["budget"].(models.CategoryBudget)
				percentageUsed := catBudgetData["percentage_used"].(float64)
				s.checkBudgetThreshold(ctx, userID, budget.Category, percentageUsed, currentMonth)
			}
		}
	}
}

// checkBudgetThreshold checks and creates notification for budget threshold milestones
func (s *NotificationService) checkBudgetThreshold(ctx context.Context, userID primitive.ObjectID, budgetName string, percentageUsed float64, _ string) {
	thresholds := []struct {
		percent     float64
		titlePrefix string
		messageTmpl string
	}{
		{100, "Budget Exceeded", "Your %s has exceeded its limit! (%.0f%%)"},
		{90, "Budget Warning", "Your %s is at 90%% capacity (%.0f%% used)"},
		{75, "Budget Caution", "Your %s is at 75%% capacity (%.0f%% used)"},
	}

	for _, threshold := range thresholds {
		if percentageUsed >= threshold.percent {
			title := fmt.Sprintf("%s: %s", threshold.titlePrefix, budgetName)
			message := fmt.Sprintf(threshold.messageTmpl, budgetName, percentageUsed)
			s.CheckAndCreateNotification(ctx, userID, title, message, models.NotifTypeBudget, "/budget", 12*time.Hour)
			break // Only notify for the highest threshold reached
		}
	}
}

// StartNotificationWorker starts a background goroutine that checks notifications every interval
func (s *NotificationService) StartNotificationWorker(interval time.Duration) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		fmt.Printf("[NotificationWorker] Started - checking every %v\n", interval)

		// Run immediately on start
		s.runNotificationCheck()

		for {
			select {
			case <-ticker.C:
				s.runNotificationCheck()
			case <-s.stopChan:
				fmt.Println("[NotificationWorker] Stopping...")
				return
			}
		}
	}()
}

// StopNotificationWorker stops the notification worker
func (s *NotificationService) StopNotificationWorker() {
	close(s.stopChan)
	s.wg.Wait()
}

// runNotificationCheck checks notifications for all users
func (s *NotificationService) runNotificationCheck() {
	fmt.Println("[NotificationWorker] Running notification check...")
	users, err := s.GetAllActiveUsers()
	if err != nil {
		fmt.Printf("[NotificationWorker] Error getting users: %v\n", err)
		return
	}

	checked := 0
	for _, userID := range users {
		s.CheckAllNotificationsForUser(userID)
		checked++
	}
	fmt.Printf("[NotificationWorker] Checked notifications for %d users\n", checked)
}
