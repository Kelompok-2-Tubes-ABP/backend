package services

import (
	"context"
	"financeapi/essentials/config"
	models "financeapi/essentials/models"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type TransactionService struct {
	collection *mongo.Collection
}

func NewTransactionService(client *mongo.Client, dbName string) *TransactionService {
	collection := client.Database(dbName).Collection("Transaction")
	return &TransactionService{collection: collection}
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
		if t.Category == "outcome" {
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
		if totalOutcome+transaction.Amount > budget.Limit {
			return models.Transaction{}, fmt.Errorf(
				"❌ Budget exceeded: total %.2f / limit %.2f",
				totalOutcome+transaction.Amount, budget.Limit,
			)
		} else if totalOutcome+transaction.Amount > 0.9*budget.Limit {
			fmt.Printf("⚠️ Warning: You've reached 90%% of your monthly budget (%.2f / %.2f)\n",
				totalOutcome+transaction.Amount, budget.Limit)
		}
	}

	result, err := s.collection.InsertOne(context.TODO(), transaction)
	if err != nil {
		return models.Transaction{}, err
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

	if filter.FromDate != "" || filter.ToDate != "" {
		dateFilter := bson.M{}

		if filter.FromDate != "" {
			fromTime, err := time.Parse("2006-01-02", filter.FromDate)
			if err != nil {
				fromTime, _ = time.Parse(time.RFC3339, filter.FromDate)
			}
			if err == nil {
				dateFilter["$gte"] = fromTime
			}
		}

		if filter.ToDate != "" {
			toTime, err := time.Parse("2006-01-02", filter.ToDate)
			if err != nil {
				toTime, _ = time.Parse(time.RFC3339, filter.ToDate)
			}
			if err == nil {
				toTime = toTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
				dateFilter["$lte"] = toTime
			}
		}

		filters["date"] = dateFilter
	}

	if filter.SortBy != "" {
		filters["category"] = filter.SortBy
	}

	result, err := s.collection.Find(context.TODO(), filters)

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

	incomeCategories := map[string]bool{
		"income":     true,
		"gaji":       true,
		"salary":     true,
		"pendapatan": true,
		"revenue":    true,
	}

	totalIncome := 0.0
	totalOutcome := 0.0
	categoryBreakdown := make(map[string]float64)

	for _, tr := range transactions {
		category := tr.Category
		categoryLower := strings.ToLower(category)

		if incomeCategories[categoryLower] || incomeCategories[category] {
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
	filters := bson.M{
		"user_id": userID,
	}

	if filter.FromDate != "" || filter.ToDate != "" {
		dateFilter := bson.M{}
		if filter.FromDate != "" {
			fromTime, _ := time.Parse("2006-01-02", filter.FromDate)
			dateFilter["$gte"] = fromTime
		}
		if filter.ToDate != "" {
			toTime, _ := time.Parse("2006-01-02", filter.ToDate)
			dateFilter["$lte"] = toTime
		}
		filters["date"] = dateFilter
	}

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

func BudgetingTransaction(budgett models.MonthlyBudget, userID string) (models.MonthlyBudget, error) {
	collections := config.GetCollection(config.DB, "monthly_budget")
	budget, err := collections.InsertOne(context.TODO(), budgett)

	if err != nil {
		return models.MonthlyBudget{}, err
	}
	budgett.ID = budget.InsertedID.(primitive.ObjectID)
	budgett.UserID = userID
	if budgett.Month == "" {
		budgett.Month = time.Now().Format("2006-01")
	}

	return budgett, nil
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
	update := bson.M{"$set": tr}

	_, err := t.collection.UpdateOne(context.TODO(), filter, update)

	return err
}
