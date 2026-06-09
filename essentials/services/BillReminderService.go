package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"financeapi/essentials/models"
	"financeapi/essentials/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	mongoOptions "go.mongodb.org/mongo-driver/mongo/options"
)

type BillReminderService struct {
	client              *mongo.Client
	collection          *mongo.Collection
	paymentCol          *mongo.Collection
	notificationService *NotificationService
}

func NewBillReminderService(client *mongo.Client, dbName string) *BillReminderService {
	db := client.Database(dbName)
	return &BillReminderService{
		client:     client,
		collection: db.Collection("bill_reminders"),
		paymentCol: db.Collection("bill_payments"),
	}
}

func (s *BillReminderService) SetNotificationService(ns *NotificationService) {
	s.notificationService = ns
}

func (s *BillReminderService) GetClient() *mongo.Client {
	return s.client
}

func (s *BillReminderService) CreateBillReminder(bill models.BillReminder) (models.BillReminder, error) {
	bill.Name = utils.SanitizeMongoValue(bill.Name)
	bill.Description = utils.SanitizeMongoValue(bill.Description)
	bill.PayTo = utils.SanitizeMongoValue(bill.PayTo)

	if bill.Name == "" {
		return models.BillReminder{}, errors.New("name is required")
	}
	if bill.Amount <= 0 {
		return models.BillReminder{}, errors.New("amount must be greater than 0")
	}

	if bill.Currency == "" {
		bill.Currency = "IDR"
	}
	if bill.RemindDaysBefore == 0 {
		bill.RemindDaysBefore = 3
	}
	if bill.BillingCycle == "" {
		bill.BillingCycle = "monthly"
	}
	if bill.DayOfMonth == 0 {
		bill.DayOfMonth = 1
	}

	bill.IsActive = true
	bill.IsPaid = false
	bill.CreatedAt = time.Now()
	bill.UpdatedAt = time.Now()

	// Automatically calculate next_due_date if not provided
	if bill.NextDueDate.IsZero() {
		now := time.Now()
		day := bill.DayOfMonth
		if day == 0 {
			day = 1
		}

		// Target date for current month
		dueDate := time.Date(now.Year(), now.Month(), day, 0, 0, 0, 0, time.Local)

		// If the date has passed or is today, go to the next cycle
		if dueDate.Before(now) || dueDate.Equal(now) {
			switch bill.BillingCycle {
			case "daily":
				dueDate = dueDate.AddDate(0, 0, 1)
			case "weekly":
				dueDate = dueDate.AddDate(0, 0, 7)
			case "monthly":
				dueDate = dueDate.AddDate(0, 1, 0)
			case "quarterly":
				dueDate = dueDate.AddDate(0, 3, 0)
			case "yearly":
				dueDate = dueDate.AddDate(1, 0, 0)
			default:
				dueDate = dueDate.AddDate(0, 1, 0)
			}
		}
		bill.NextDueDate = dueDate
	}

	result, err := s.collection.InsertOne(context.TODO(), bill)
	if err != nil {
		return models.BillReminder{}, err
	}

	bill.ID = result.InsertedID.(primitive.ObjectID)
	return bill, nil
}

func (s *BillReminderService) GetBillReminder(id primitive.ObjectID, userID primitive.ObjectID) (models.BillReminder, error) {
	var bill models.BillReminder
	err := s.collection.FindOne(context.TODO(), bson.M{
		"_id":     id,
		"user_id": userID,
	}).Decode(&bill)
	if err != nil {
		return models.BillReminder{}, errors.New("bill reminder not found")
	}
	return bill, nil
}

func (s *BillReminderService) GetUserBillReminders(userID primitive.ObjectID) ([]models.BillReminder, error) {
	cursor, err := s.collection.Find(context.TODO(), bson.M{
		"user_id":   userID,
		"is_active": true,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var bills []models.BillReminder
	if err := cursor.All(context.TODO(), &bills); err != nil {
		return nil, err
	}

	return bills, nil
}

func (s *BillReminderService) GetDueBillReminders(userID primitive.ObjectID) ([]models.BillReminder, error) {
	now := time.Now()
	daysAhead := now.AddDate(0, 0, 7)

	cursor, err := s.collection.Find(context.TODO(), bson.M{
		"user_id":       userID,
		"is_active":     true,
		"is_paid":       false,
		"next_due_date": bson.M{"$lte": daysAhead},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var bills []models.BillReminder
	if err := cursor.All(context.TODO(), &bills); err != nil {
		return nil, err
	}

	return bills, nil
}

func (s *BillReminderService) GetOverdueBillReminders(userID primitive.ObjectID) ([]models.BillReminder, error) {
	now := time.Now()

	cursor, err := s.collection.Find(context.TODO(), bson.M{
		"user_id":       userID,
		"is_active":     true,
		"is_paid":       false,
		"next_due_date": bson.M{"$lt": now},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var bills []models.BillReminder
	if err := cursor.All(context.TODO(), &bills); err != nil {
		return nil, err
	}

	return bills, nil
}

func (s *BillReminderService) UpdateBillReminder(id primitive.ObjectID, userID primitive.ObjectID, updates bson.M) (models.BillReminder, error) {
	if name, ok := updates["name"].(string); ok {
		updates["name"] = utils.SanitizeMongoValue(name)
	}

	updates["updated_at"] = time.Now()

	opts := mongoOptions.FindOneAndUpdate().SetReturnDocument(mongoOptions.After)
	result := s.collection.FindOneAndUpdate(
		context.TODO(),
		bson.M{"_id": id, "user_id": userID},
		bson.M{"$set": updates},
		opts,
	)

	var bill models.BillReminder
	if err := result.Decode(&bill); err != nil {
		return models.BillReminder{}, errors.New("bill reminder not found")
	}

	return bill, nil
}

func (s *BillReminderService) MarkAsPaid(id primitive.ObjectID, userID primitive.ObjectID, amount float64, notes string) error {
	bill, err := s.GetBillReminder(id, userID)
	if err != nil {
		return err
	}

	now := time.Now()

	_, err = s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id, "user_id": userID},
		bson.M{
			"$set": bson.M{
				"is_paid":        true,
				"last_paid_date": now,
				"next_due_date":  bill.CalculateNextDueDate(),
				"updated_at":     now,
			},
		},
	)
	if err != nil {
		return err
	}

	payment := models.BillPaymentLog{
		BillReminderID: id,
		UserID:         userID,
		Amount:         amount,
		Currency:       bill.Currency,
		PaidDate:       now,
		Notes:          notes,
		CreatedAt:      now,
	}

	_, err = s.paymentCol.InsertOne(context.TODO(), payment)
	return err
}

func (s *BillReminderService) DeleteBillReminder(id primitive.ObjectID, userID primitive.ObjectID) error {
	result, err := s.collection.DeleteOne(context.TODO(), bson.M{
		"_id":     id,
		"user_id": userID,
	})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("bill reminder not found")
	}

	return nil
}

func (s *BillReminderService) GetBillSummary(userID primitive.ObjectID) (map[string]interface{}, error) {
	bills, err := s.GetUserBillReminders(userID)
	if err != nil {
		return nil, err
	}

	totalMonthly := 0.0
	upcomingCount := 0
	overdueCount := 0
	paidCount := 0

	for _, bill := range bills {
		var monthlyAmount float64
		switch bill.BillingCycle {
		case "monthly":
			monthlyAmount = bill.Amount
		case "quarterly":
			monthlyAmount = bill.Amount / 3
		case "yearly":
			monthlyAmount = bill.Amount / 12
		}
		totalMonthly += monthlyAmount

		status := bill.GetDueStatus()
		switch status {
		case "paid":
			paidCount++
		case "overdue":
			overdueCount++
		case "due_soon":
			upcomingCount++
		}
	}

	return map[string]interface{}{
		"total_bills":   len(bills),
		"total_monthly": totalMonthly,
		"upcoming":      upcomingCount,
		"overdue":       overdueCount,
		"paid":          paidCount,
		"upcoming_amount": func() float64 {
			var t float64
			for _, b := range bills {
				if b.GetDueStatus() == "due_soon" {
					t += b.Amount
				}
			}
			return t
		}(),
		"overdue_amount": func() float64 {
			var t float64
			for _, b := range bills {
				if b.GetDueStatus() == "overdue" {
					t += b.Amount
				}
			}
			return t
		}(),
	}, nil
}

func (s *BillReminderService) GetPaymentHistory(billID primitive.ObjectID, userID primitive.ObjectID) ([]models.BillPaymentLog, error) {
	cursor, err := s.paymentCol.Find(context.TODO(), bson.M{
		"bill_reminder_id": billID,
		"user_id":          userID,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var payments []models.BillPaymentLog
	if err := cursor.All(context.TODO(), &payments); err != nil {
		return nil, err
	}

	return payments, nil
}

// CheckAndNotifyDueBills checks for upcoming/overdue bills and creates notifications
func (s *BillReminderService) CheckAndNotifyDueBills(userID primitive.ObjectID) error {
	if s.notificationService == nil {
		fmt.Println("[DEBUG] BillReminderService.notificationService is nil")
		return nil
	}

	fmt.Printf("[DEBUG] CheckAndNotifyDueBills called for user: %s\n", userID.Hex())

	// Get bills due in the next 7 days (includes overdue)
	dueBills, err := s.GetDueBillReminders(userID)
	if err != nil {
		fmt.Printf("[DEBUG] GetDueBillReminders error: %v\n", err)
		return err
	}

	fmt.Printf("[DEBUG] GetDueBillReminders found %d bills\n", len(dueBills))

	for _, bill := range dueBills {
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

		// Use 24 hour window to avoid duplicate notifications
		s.notificationService.CheckAndCreateNotification(
			context.TODO(),
			userID,
			title,
			message,
			models.NotifTypeBill,
			"/bills",
			24*time.Hour,
		)
	}

	// Also check for overdue bills
	overdueBills, err := s.GetOverdueBillReminders(userID)
	if err != nil {
		return err
	}

	for _, bill := range overdueBills {
		daysOverdue := int(-time.Until(bill.NextDueDate).Hours() / 24)
		title := fmt.Sprintf("Overdue: %s", bill.Name)
		message := fmt.Sprintf("Bill payment of %.2f is %d days overdue!", bill.Amount, daysOverdue)

		// Use 24 hour window to avoid duplicate notifications
		s.notificationService.CheckAndCreateNotification(
			context.TODO(),
			userID,
			title,
			message,
			models.NotifTypeBill,
			"/bills",
			24*time.Hour,
		)
	}

	return nil
}
