package services

import (
	"context"
	"errors"
	"time"

	"financeapi/essentials/models"
	"financeapi/essentials/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	mongoOptions "go.mongodb.org/mongo-driver/mongo/options"
)

type RecurringTransactionService struct {
	collection   *mongo.Collection
	generatedCol *mongo.Collection
}

func NewRecurringTransactionService(client *mongo.Client, dbName string) *RecurringTransactionService {
	db := client.Database(dbName)
	return &RecurringTransactionService{
		collection:   db.Collection("recurring_transactions"),
		generatedCol: db.Collection("generated_transactions"),
	}
}

func (s *RecurringTransactionService) CreateRecurringTransaction(recurring models.RecurringTransaction) (models.RecurringTransaction, error) {
	recurring.Name = utils.SanitizeMongoValue(recurring.Name)
	recurring.Description = utils.SanitizeMongoValue(recurring.Description)
	recurring.Category = utils.SanitizeMongoValue(recurring.Category)

	if recurring.Name == "" {
		return models.RecurringTransaction{}, errors.New("name is required")
	}
	if recurring.Amount <= 0 {
		return models.RecurringTransaction{}, errors.New("amount must be greater than 0")
	}
	if recurring.Frequency == "" {
		return models.RecurringTransaction{}, errors.New("frequency is required")
	}

	recurring.IsActive = true
	recurring.CreatedAt = time.Now()
	recurring.UpdatedAt = time.Now()

	if recurring.StartDate.IsZero() {
		recurring.StartDate = time.Now()
	}

	recurring.NextRunDate = recurring.StartDate

	result, err := s.collection.InsertOne(context.TODO(), recurring)
	if err != nil {
		return models.RecurringTransaction{}, err
	}

	recurring.ID = result.InsertedID.(primitive.ObjectID)
	return recurring, nil
}

func (s *RecurringTransactionService) GetRecurringTransaction(id primitive.ObjectID, userID primitive.ObjectID) (models.RecurringTransaction, error) {
	var recurring models.RecurringTransaction
	err := s.collection.FindOne(context.TODO(), bson.M{
		"_id":     id,
		"user_id": userID,
	}).Decode(&recurring)
	if err != nil {
		return models.RecurringTransaction{}, errors.New("recurring transaction not found")
	}
	return recurring, nil
}

func (s *RecurringTransactionService) GetUserRecurringTransactions(userID primitive.ObjectID) ([]models.RecurringTransaction, error) {
	cursor, err := s.collection.Find(context.TODO(), bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var recurring []models.RecurringTransaction
	if err := cursor.All(context.TODO(), &recurring); err != nil {
		return nil, err
	}

	return recurring, nil
}

func (s *RecurringTransactionService) GetActiveRecurringTransactions(userID primitive.ObjectID) ([]models.RecurringTransaction, error) {
	cursor, err := s.collection.Find(context.TODO(), bson.M{
		"user_id":   userID,
		"is_active": true,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var recurring []models.RecurringTransaction
	if err := cursor.All(context.TODO(), &recurring); err != nil {
		return nil, err
	}

	return recurring, nil
}

func (s *RecurringTransactionService) GetDueRecurringTransactions(userID primitive.ObjectID) ([]models.RecurringTransaction, error) {
	now := time.Now()
	cursor, err := s.collection.Find(context.TODO(), bson.M{
		"user_id":       userID,
		"is_active":     true,
		"next_run_date": bson.M{"$lte": now},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var recurring []models.RecurringTransaction
	if err := cursor.All(context.TODO(), &recurring); err != nil {
		return nil, err
	}

	return recurring, nil
}

func (s *RecurringTransactionService) UpdateRecurringTransaction(id primitive.ObjectID, userID primitive.ObjectID, updates bson.M) (models.RecurringTransaction, error) {
	if name, ok := updates["name"].(string); ok {
		updates["name"] = utils.SanitizeMongoValue(name)
	}
	if desc, ok := updates["description"].(string); ok {
		updates["description"] = utils.SanitizeMongoValue(desc)
	}

	updates["updated_at"] = time.Now()

	opts := mongoOptions.FindOneAndUpdate().SetReturnDocument(mongoOptions.After)
	result := s.collection.FindOneAndUpdate(
		context.TODO(),
		bson.M{"_id": id, "user_id": userID},
		bson.M{"$set": updates},
		opts,
	)

	var recurring models.RecurringTransaction
	if err := result.Decode(&recurring); err != nil {
		return models.RecurringTransaction{}, errors.New("recurring transaction not found")
	}

	return recurring, nil
}

func (s *RecurringTransactionService) DeleteRecurringTransaction(id primitive.ObjectID, userID primitive.ObjectID) error {
	result, err := s.collection.DeleteOne(context.TODO(), bson.M{
		"_id":     id,
		"user_id": userID,
	})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("recurring transaction not found")
	}

	return nil
}

func (s *RecurringTransactionService) GenerateTransaction(recurring models.RecurringTransaction) (models.GeneratedTransaction, error) {
	generated := models.GeneratedTransaction{
		RecurringID:     recurring.ID,
		UserID:          recurring.UserID,
		GeneratedFrom:   recurring.Name,
		Name:            recurring.Name,
		Description:     recurring.Description,
		Amount:          recurring.Amount,
		Currency:        recurring.Currency,
		Category:        recurring.Category,
		Type:            recurring.Type,
		AccountID:       recurring.AccountID,
		TransactionDate: recurring.NextRunDate,
		CreatedDate:     time.Now(),
		Status:          "pending",
	}

	result, err := s.generatedCol.InsertOne(context.TODO(), generated)
	if err != nil {
		return generated, err
	}

	generated.ID = result.InsertedID.(primitive.ObjectID)

	nextDate := recurring.CalculateNextRunDate()
	s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": recurring.ID},
		bson.M{"$set": bson.M{
			"next_run_date": nextDate,
			"updated_at":    time.Now(),
		}},
	)

	return generated, nil
}

func (s *RecurringTransactionService) ProcessDueTransactions(userID primitive.ObjectID) ([]models.GeneratedTransaction, error) {
	due, err := s.GetDueRecurringTransactions(userID)
	if err != nil {
		return nil, err
	}

	var generated []models.GeneratedTransaction
	for _, recurring := range due {
		g, err := s.GenerateTransaction(recurring)
		if err != nil {
			continue
		}
		generated = append(generated, g)
	}

	return generated, nil
}

func (s *RecurringTransactionService) GetGeneratedTransactions(recurringID primitive.ObjectID, userID primitive.ObjectID) ([]models.GeneratedTransaction, error) {
	cursor, err := s.generatedCol.Find(context.TODO(), bson.M{
		"recurring_id": recurringID,
		"user_id":      userID,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var generated []models.GeneratedTransaction
	if err := cursor.All(context.TODO(), &generated); err != nil {
		return nil, err
	}

	return generated, nil
}

func (s *RecurringTransactionService) SkipNextRun(id primitive.ObjectID, userID primitive.ObjectID) error {
	recurring, err := s.GetRecurringTransaction(id, userID)
	if err != nil {
		return err
	}

	nextDate := recurring.CalculateNextRunDate()
	_, err = s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id, "user_id": userID},
		bson.M{
			"$set": bson.M{
				"next_run_date": nextDate,
				"updated_at":    time.Now(),
			},
		},
	)

	return err
}

func (s *RecurringTransactionService) PauseRecurringTransaction(id primitive.ObjectID, userID primitive.ObjectID) error {
	_, err := s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id, "user_id": userID},
		bson.M{
			"$set": bson.M{
				"is_active":  false,
				"updated_at": time.Now(),
			},
		},
	)
	return err
}

func (s *RecurringTransactionService) ResumeRecurringTransaction(id primitive.ObjectID, userID primitive.ObjectID) error {
	_, err := s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id, "user_id": userID},
		bson.M{
			"$set": bson.M{
				"is_active":  true,
				"updated_at": time.Now(),
			},
		},
	)
	return err
}

func (s *RecurringTransactionService) GetRecurringSummary(userID primitive.ObjectID) (map[string]interface{}, error) {
	recurring, err := s.GetUserRecurringTransactions(userID)
	if err != nil {
		return nil, err
	}

	totalMonthlyIncome := 0.0
	totalMonthlyExpenses := 0.0
	activeCount := 0
	pausedCount := 0

	for _, r := range recurring {
		if !r.IsActive {
			pausedCount++
			continue
		}
		activeCount++

		var monthlyAmount float64
		switch r.Frequency {
		case models.FreqDaily:
			monthlyAmount = r.Amount * 30
		case models.FreqWeekly:
			monthlyAmount = r.Amount * 4
		case models.FreqBiweekly:
			monthlyAmount = r.Amount * 2
		case models.FreqMonthly:
			monthlyAmount = r.Amount
		case models.FreqQuarterly:
			monthlyAmount = r.Amount / 3
		case models.FreqYearly:
			monthlyAmount = r.Amount / 12
		}

		if r.Type == "income" {
			totalMonthlyIncome += monthlyAmount
		} else {
			totalMonthlyExpenses += monthlyAmount
		}
	}

	return map[string]interface{}{
		"total_recurring":      len(recurring),
		"active_count":         activeCount,
		"paused_count":         pausedCount,
		"monthly_income":       totalMonthlyIncome,
		"monthly_expenses":     totalMonthlyExpenses,
		"net_monthly_cashflow": totalMonthlyIncome - totalMonthlyExpenses,
	}, nil
}
