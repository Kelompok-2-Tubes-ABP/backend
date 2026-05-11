package services

import (
	"context"
	"financeapi/essentials/config"
	models "financeapi/essentials/models"
	"financeapi/essentials/utils"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TransactionService struct {
	collection *mongo.Collection
	notificationService *NotificationService
}

func NewTransactionService(client *mongo.Client, dbName string) *TransactionService {
	collection := client.Database(dbName).Collection("Transaction")
	return &TransactionService{collection: collection}
}

func (s *TransactionService) SetNotificationService(ns *NotificationService) {
	s.notificationService = ns
}

func buildFilters(userID string, filter models.FilterTransaction) bson.M {
	filters := bson.M{
		"user_id": userID,
	}

	if filter.MinAmount > 0 || filter.MaxAmount > 0 {
		amountFilter := bson.M{}
		if filter.MinAmount > 0 {
			amountFilter["$gte"] = filter.MinAmount
		}
		if filter.MaxAmount > 0 {
			amountFilter["$lte"] = filter.MaxAmount
		}
		filters["amount"] = amountFilter
	}

	if filter.Category != "" {
		filters["category"] = filter.Category
	} else if filter.Type != "" {
		incomeCategories := []string{"income", "gaji", "salary", "pendapatan", "revenue"}
		if filter.Type == "income" {
			filters["category"] = bson.M{"$in": incomeCategories}
		} else if filter.Type == "outcome" {
			filters["category"] = bson.M{"$nin": incomeCategories}
		}
	}

	if filter.FromDate != "" || filter.ToDate != "" {
		dateFilter := bson.M{}
		if filter.FromDate != "" {
			fromTime, err := time.Parse("2006-01-02", filter.FromDate)
			if err != nil {
				fromTime, err = time.Parse(time.RFC3339, filter.FromDate)
			}
			if err == nil {
				dateFilter["$gte"] = fromTime
			}
		}

		if filter.ToDate != "" {
			toTime, err := time.Parse("2006-01-02", filter.ToDate)
			if err != nil {
				toTime, err = time.Parse(time.RFC3339, filter.ToDate)
			}
			if err == nil {
				toTime = toTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
				dateFilter["$lte"] = toTime
			}
		}

		filters["date"] = dateFilter
	}
	return filters
}
func (s *TransactionService) CreateTransaction(transaction models.Transaction) (models.Transaction, error) {
	filters := bson.M{
		"user_id": transaction.User_id,
		"month":   transaction.Month,
	}

	//
	cursor, err := s.collection.Find(context.TODO(), filters)
	if err != nil {
		return models.Transaction{}, err
	}
	defer cursor.Close(context.TODO())

	var transactions []models.Transaction
	if err := cursor.All(context.TODO(), &transactions); err != nil {
		return models.Transaction{}, err
	}

	totalOutcome := 0.0
	for _, t := range transactions {
		if utils.IsOutcome(t.Category) {
			totalOutcome += t.Amount
		}
	}

	//
	budgetColl := config.GetCollection(config.DB, "monthly_budget")
	filterBudget := bson.M{
		"user_id": transaction.User_id,
		"month":   transaction.Month,
	}

	var budget models.MonthlyBudget
	err = budgetColl.FindOne(context.TODO(), filterBudget).Decode(&budget)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("⚠️ No monthly budget found, skipping budget check")
		} else {
			return models.Transaction{}, err
		}
	} else {
		if utils.IsOutcome(transaction.Category) {
			if totalOutcome+transaction.Amount > budget.Limit {
				if s.notificationService != nil {
					objID, _ := primitive.ObjectIDFromHex(transaction.User_id)
					s.notificationService.CreateNotification(context.TODO(), objID, "Over Budget Alert", fmt.Sprintf("Your transaction exceeds your monthly budget limit of %.2f", budget.Limit), models.NotifTypeBudget, "")
				}
				return models.Transaction{}, fmt.Errorf(
					"❌ Budget exceeded: total %.2f / limit %.2f",
					totalOutcome+transaction.Amount, budget.Limit,
				)
			} else if totalOutcome+transaction.Amount > 0.9*budget.Limit {
				fmt.Printf("⚠️ Warning: You've reached 90%% of your monthly budget (%.2f / %.2f)\n",
					totalOutcome+transaction.Amount, budget.Limit)
				if s.notificationService != nil {
					objID, _ := primitive.ObjectIDFromHex(transaction.User_id)
					s.notificationService.CreateNotification(context.TODO(), objID, "Near Budget Limit", fmt.Sprintf("You have reached 90%% of your monthly budget (%.2f / %.2f)", totalOutcome+transaction.Amount, budget.Limit), models.NotifTypeBudget, "")
				}
			}
		}
	}

	// Set default status if empty
	if transaction.Status == "" {
		transaction.Status = "completed" // default to completed for regular transactions
	}

	result, err := s.collection.InsertOne(context.TODO(), transaction)
	if err != nil {
		return models.Transaction{}, err
	}

	// Alert for large transactions
	if transaction.Amount >= 5000000 && utils.IsOutcome(transaction.Category) {
		if s.notificationService != nil {
			objID, _ := primitive.ObjectIDFromHex(transaction.User_id)
			s.notificationService.CreateNotification(context.TODO(), objID, "Large Transaction Alert", fmt.Sprintf("A large transaction of %.2f was detected in category %s.", transaction.Amount, transaction.Category), models.NotifTypeTransaction, "")
		}
	}

	transaction.ID = result.InsertedID.(primitive.ObjectID)
	return transaction, nil
}

func (s *TransactionService) ShowTransaction(userID string) ([]models.Transaction, error) {
	result, err := s.collection.Find(context.TODO(), bson.M{"user_id": userID})

	if err != nil {
		return nil, err
	}
	var transaction []models.Transaction

	if err = result.All(context.TODO(), &transaction); err != nil {
		return nil, err
	}
	if len(transaction) == 0 {
		return []models.Transaction{}, nil
	}
	return transaction, nil
}

func (s *TransactionService) ShowTransactionByFilter(userID string, filter models.FilterTransaction) ([]models.Transaction, error) {
	filters := buildFilters(userID, filter)

	opts := options.Find()
	if filter.SortBy != "" {
		sortOrder := 1
		sortField := filter.SortBy
		if strings.HasPrefix(filter.SortBy, "-") {
			sortOrder = -1
			sortField = strings.TrimPrefix(filter.SortBy, "-")
		}
		opts.SetSort(bson.D{{Key: sortField, Value: sortOrder}})
	}

	result, err := s.collection.Find(context.TODO(), filters, opts)

	if err != nil {
		return nil, err
	}
	var transaction []models.Transaction
	if err = result.All(context.TODO(), &transaction); err != nil {
		return nil, err
	}
	if len(transaction) == 0 {
		return []models.Transaction{}, nil
	}
	return transaction, nil
}

func (t *TransactionService) GetMonthly(userID string, filter models.Report) (map[string]float64, error) {
	filters := bson.M{
		"user_id": userID,
	}
	if filter.Month != "" {
		filters["month"] = filter.Month
	}
	cursor, err := t.collection.Find(context.TODO(), filters)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var transactions []models.Transaction
	if err := cursor.All(context.TODO(), &transactions); err != nil {
		return nil, err
	}

	totalIncome := 0.0
	totalOutcome := 0.0
	categoryBreakdown := make(map[string]float64)

	for _, tr := range transactions {
		category := tr.Category

		if utils.IsIncome(category) {
			totalIncome += tr.Amount
		} else {
			totalOutcome += tr.Amount
		}

		categoryBreakdown[category] += tr.Amount
	}

	report := map[string]float64{
		"income":  totalIncome,
		"outcome": totalOutcome,
		"net":     totalIncome - totalOutcome,
	}

	for cat, amount := range categoryBreakdown {
		report[cat] = amount
	}

	return report, nil
}
func (t *TransactionService) GetReport(userID string, filter models.FilterTransaction) (map[string]float64, error) {
	filters := buildFilters(userID, filter)

	pipeline := mongo.Pipeline{
		{{"$match", filters}},
		{{"$group", bson.D{
			{"_id", "$category"},
			{"total", bson.D{{"$sum", "$amount"}}},
		}}},
	}

	cursor, err := t.collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	report := map[string]float64{"income": 0, "outcome": 0, "net": 0}
	for cursor.Next(context.TODO()) {
		var result struct {
			ID    string  `bson:"_id"`
			Total float64 `bson:"total"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		report[result.ID] = result.Total
	}

	report["net"] = report["income"] - report["outcome"]

	return report, nil
}

func (s *TransactionService) GetCategoryStats(userID string, filter models.FilterTransaction) ([]models.CategoryStat, error) {
	filters := buildFilters(userID, filter)

	pipeline := mongo.Pipeline{
		{{"$match", filters}},
		{{"$group", bson.D{
			{"_id", "$category"},
			{"amount", bson.D{{"$sum", "$amount"}}},
			{"transaction_count", bson.D{{"$sum", 1}}},
			{"average_amount", bson.D{{"$avg", "$amount"}}},
		}}},
		{{"$sort", bson.D{{"amount", -1}}}},
	}

	cursor, err := s.collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var results []struct {
		Category         string  `bson:"_id"`
		Amount           float64 `bson:"amount"`
		TransactionCount int     `bson:"transaction_count"`
		AverageAmount    float64 `bson:"average_amount"`
	}

	if err := cursor.All(context.TODO(), &results); err != nil {
		return nil, err
	}

	var totalAmount float64
	for _, r := range results {
		totalAmount += r.Amount
	}

	var stats []models.CategoryStat
	for _, r := range results {
		percentage := 0.0
		if totalAmount > 0 {
			percentage = (r.Amount / totalAmount) * 100
		}

		stats = append(stats, models.CategoryStat{
			Category:         r.Category,
			Amount:           r.Amount,
			Percentage:       percentage,
			TransactionCount: r.TransactionCount,
			AverageAmount:    r.AverageAmount,
		})
	}

	return stats, nil
}

func (t *TransactionService) DeleteTransaction(id primitive.ObjectID) error {
	filter := bson.M{
		"_id": id,
	}
	_, err := t.collection.DeleteOne(context.TODO(), filter)
	return err
}

func (t *TransactionService) UpdateTransaction(id primitive.ObjectID, tr models.Transaction) error {
	filter := bson.M{
		"_id": id,
	}

	updateFields := bson.M{}

	if tr.Amount != 0 {
		updateFields["amount"] = tr.Amount
	}
	if tr.Category != "" {
		updateFields["category"] = tr.Category
	}
	if tr.Description != "" {
		updateFields["description"] = tr.Description
	}
	if tr.Status != "" {
		updateFields["status"] = tr.Status
	}

	if len(updateFields) == 0 {
		return fmt.Errorf("no valid fields to update")
	}

	update := bson.M{"$set": updateFields}

	_, err := t.collection.UpdateOne(context.TODO(), filter, update)

	return err
}
