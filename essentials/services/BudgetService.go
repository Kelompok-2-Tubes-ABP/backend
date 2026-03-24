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

type BudgetService struct {
	collection        *mongo.Collection
	transactionCol    *mongo.Collection
	categoryBudgetCol *mongo.Collection
}

func NewBudgetService(client *mongo.Client, dbName string) *BudgetService {
	db := client.Database(dbName)
	return &BudgetService{
		collection:        db.Collection("monthly_budget"),
		transactionCol:    db.Collection("Transaction"),
		categoryBudgetCol: db.Collection("category_budget"),
	}
}

func (b *BudgetService) SetTransactionService(txService *TransactionService) {
}

func (b *BudgetService) GetCollection() *mongo.Collection {
	return b.transactionCol
}

func (b *BudgetService) CreateBudget(budget models.MonthlyBudget) (models.MonthlyBudget, error) {
	if budget.Limit <= 0 {
		return models.MonthlyBudget{}, errors.New("budget limit must be greater than 0")
	}

	if budget.UserID == "" {
		return models.MonthlyBudget{}, errors.New("user ID is required")
	}

	if budget.Month == "" {
		budget.Month = time.Now().Format("2006-01")
	}

	existingBudget := models.MonthlyBudget{}
	err := b.collection.FindOne(context.TODO(), bson.M{
		"user_id": budget.UserID,
		"month":   budget.Month,
	}).Decode(&existingBudget)

	if err == nil {
		return models.MonthlyBudget{}, errors.New("budget for this month already exists")
	}

	if err != mongo.ErrNoDocuments {
		return models.MonthlyBudget{}, err
	}

	result, err := b.collection.InsertOne(context.TODO(), budget)
	if err != nil {
		return models.MonthlyBudget{}, err
	}

	budget.ID = result.InsertedID.(primitive.ObjectID)
	return budget, nil
}

func (b *BudgetService) GetBudget(budgetID primitive.ObjectID, userID string) (models.MonthlyBudget, error) {
	var budget models.MonthlyBudget
	err := b.collection.FindOne(context.TODO(), bson.M{
		"_id":     budgetID,
		"user_id": userID,
	}).Decode(&budget)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.MonthlyBudget{}, errors.New("budget not found")
		}
		return models.MonthlyBudget{}, err
	}

	return budget, nil
}

func (b *BudgetService) GetUserBudgets(userID string) ([]models.MonthlyBudget, error) {
	cursor, err := b.collection.Find(context.TODO(), bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var budgets []models.MonthlyBudget
	if err := cursor.All(context.TODO(), &budgets); err != nil {
		return nil, err
	}

	return budgets, nil
}

func (b *BudgetService) GetBudgetByMonth(userID, month string) (models.MonthlyBudget, error) {
	var budget models.MonthlyBudget
	err := b.collection.FindOne(context.TODO(), bson.M{
		"user_id": userID,
		"month":   month,
	}).Decode(&budget)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.MonthlyBudget{}, errors.New("budget not found for this month")
		}
		return models.MonthlyBudget{}, err
	}

	return budget, nil
}

func (b *BudgetService) UpdateBudget(budgetID primitive.ObjectID, userID string, updates bson.M) (models.MonthlyBudget, error) {
	if limit, ok := updates["limit"].(float64); ok {
		if limit <= 0 {
			return models.MonthlyBudget{}, errors.New("budget limit must be greater than 0")
		}
	}

	if month, ok := updates["month"].(string); ok {
		updates["month"] = utils.SanitizeMongoValue(month)
	}

	updates["updated_at"] = time.Now()

	opts := mongoOptions.FindOneAndUpdate().SetReturnDocument(mongoOptions.After)
	result := b.collection.FindOneAndUpdate(
		context.TODO(),
		bson.M{"_id": budgetID, "user_id": userID},
		bson.M{"$set": updates},
		opts,
	)

	var budget models.MonthlyBudget
	if err := result.Decode(&budget); err != nil {
		return models.MonthlyBudget{}, errors.New("budget not found")
	}

	return budget, nil
}

func (b *BudgetService) DeleteBudget(budgetID primitive.ObjectID, userID string) error {
	result, err := b.collection.DeleteOne(context.TODO(), bson.M{
		"_id":     budgetID,
		"user_id": userID,
	})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("budget not found")
	}

	return nil
}

func (b *BudgetService) GetBudgetWithSpending(userID, month string) (map[string]interface{}, error) {
	budget, err := b.GetBudgetByMonth(userID, month)
	if err != nil {
		return nil, err
	}

	spending, err := b.CalculateSpending(userID, month)
	if err != nil {
		return nil, err
	}

	remaining := budget.Limit - spending

	percentageUsed := 0.0
	if budget.Limit > 0 {
		percentageUsed = (spending / budget.Limit) * 100
	}

	status := "safe"
	if percentageUsed >= 100 {
		status = "exceeded"
	} else if percentageUsed >= 90 {
		status = "warning"
	} else if percentageUsed >= 75 {
		status = "caution"
	}

	return map[string]interface{}{
		"budget":          budget,
		"spending":        spending,
		"remaining":       remaining,
		"percentage_used": percentageUsed,
		"status":          status,
	}, nil
}

func (b *BudgetService) CalculateSpending(userID, month string) (float64, error) {
	if b.transactionCol == nil {
		return 0, errors.New("transaction collection not initialized")
	}

	cursor, err := b.transactionCol.Find(context.TODO(), bson.M{
		"user_id": userID,
		"month":   month,
	})
	if err != nil {
		return 0, err
	}
	defer cursor.Close(context.TODO())

	var transactions []models.Transaction
	if err := cursor.All(context.TODO(), &transactions); err != nil {
		return 0, err
	}

	totalSpending := 0.0
	for _, t := range transactions {
		if t.Category == "outcome" {
			totalSpending += t.Amount
		}
	}

	return totalSpending, nil
}

func (b *BudgetService) GetAllBudgetsWithSpending(userID string) ([]map[string]interface{}, error) {
	budgets, err := b.GetUserBudgets(userID)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, budget := range budgets {
		spending, err := b.CalculateSpending(userID, budget.Month)
		if err != nil {
			spending = 0
		}

		remaining := budget.Limit - spending
		percentageUsed := 0.0
		if budget.Limit > 0 {
			percentageUsed = (spending / budget.Limit) * 100
		}

		status := "safe"
		if percentageUsed >= 100 {
			status = "exceeded"
		} else if percentageUsed >= 90 {
			status = "warning"
		} else if percentageUsed >= 75 {
			status = "caution"
		}

		result = append(result, map[string]interface{}{
			"budget":          budget,
			"spending":        spending,
			"remaining":       remaining,
			"percentage_used": percentageUsed,
			"status":          status,
		})
	}

	return result, nil
}

func (b *BudgetService) GetBudgetSummary(userID string) (map[string]interface{}, error) {
	budgets, err := b.GetUserBudgets(userID)
	if err != nil {
		return nil, err
	}

	if len(budgets) == 0 {
		return map[string]interface{}{
			"total_budgets":   0,
			"total_budgeted":  0,
			"total_spent":     0,
			"total_remaining": 0,
			"average_budget":  0,
			"average_spent":   0,
		}, nil
	}

	totalBudgeted := 0.0
	totalSpent := 0.0

	for _, budget := range budgets {
		spending, err := b.CalculateSpending(userID, budget.Month)
		if err != nil {
			spending = 0
		}

		totalBudgeted += budget.Limit
		totalSpent += spending
	}

	totalRemaining := totalBudgeted - totalSpent

	return map[string]interface{}{
		"total_budgets":   len(budgets),
		"total_budgeted":  totalBudgeted,
		"total_spent":     totalSpent,
		"total_remaining": totalRemaining,
		"average_budget":  totalBudgeted / float64(len(budgets)),
		"average_spent":   totalSpent / float64(len(budgets)),
	}, nil
}

func (b *BudgetService) CreateCategoryBudget(budget models.CategoryBudget) (models.CategoryBudget, error) {
	if budget.Limit <= 0 {
		return models.CategoryBudget{}, errors.New("budget limit must be greater than 0")
	}

	if budget.UserID == "" {
		return models.CategoryBudget{}, errors.New("user ID is required")
	}

	if budget.Category == "" {
		return models.CategoryBudget{}, errors.New("category is required")
	}

	budget.Category = utils.SanitizeMongoValue(budget.Category)

	if budget.Month == "" {
		budget.Month = time.Now().Format("2006-01")
	}

	existingBudget := models.CategoryBudget{}
	err := b.categoryBudgetCol.FindOne(context.TODO(), bson.M{
		"user_id":  budget.UserID,
		"month":    budget.Month,
		"category": budget.Category,
	}).Decode(&existingBudget)

	if err == nil {
		return models.CategoryBudget{}, errors.New("budget for this category and month already exists")
	}

	if err != mongo.ErrNoDocuments {
		return models.CategoryBudget{}, err
	}

	budget.Spent = 0
	budget.CreatedAt = time.Now()
	budget.UpdatedAt = time.Now()

	result, err := b.categoryBudgetCol.InsertOne(context.TODO(), budget)
	if err != nil {
		return models.CategoryBudget{}, err
	}

	budget.ID = result.InsertedID.(primitive.ObjectID)
	return budget, nil
}

func (b *BudgetService) GetCategoryBudget(budgetID primitive.ObjectID, userID string) (models.CategoryBudget, error) {
	var budget models.CategoryBudget
	err := b.categoryBudgetCol.FindOne(context.TODO(), bson.M{
		"_id":     budgetID,
		"user_id": userID,
	}).Decode(&budget)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.CategoryBudget{}, errors.New("category budget not found")
		}
		return models.CategoryBudget{}, err
	}

	b.updateCategoryBudgetSpent(&budget)

	return budget, nil
}

func (b *BudgetService) GetUserCategoryBudgets(userID string) ([]models.CategoryBudget, error) {
	cursor, err := b.categoryBudgetCol.Find(context.TODO(), bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var budgets []models.CategoryBudget
	if err := cursor.All(context.TODO(), &budgets); err != nil {
		return nil, err
	}

	// Update spent amounts
	for i := range budgets {
		b.updateCategoryBudgetSpent(&budgets[i])
	}

	return budgets, nil
}

func (b *BudgetService) GetCategoryBudgetByMonth(userID, month string) ([]models.CategoryBudget, error) {
	cursor, err := b.categoryBudgetCol.Find(context.TODO(), bson.M{
		"user_id": userID,
		"month":   month,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var budgets []models.CategoryBudget
	if err := cursor.All(context.TODO(), &budgets); err != nil {
		return nil, err
	}

	for i := range budgets {
		b.updateCategoryBudgetSpent(&budgets[i])
	}

	return budgets, nil
}

func (b *BudgetService) GetCategoryBudgetByCategory(userID, month, category string) (models.CategoryBudget, error) {
	var budget models.CategoryBudget
	err := b.categoryBudgetCol.FindOne(context.TODO(), bson.M{
		"user_id":  userID,
		"month":    month,
		"category": category,
	}).Decode(&budget)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.CategoryBudget{}, errors.New("category budget not found")
		}
		return models.CategoryBudget{}, err
	}

	b.updateCategoryBudgetSpent(&budget)

	return budget, nil
}

func (b *BudgetService) UpdateCategoryBudget(budgetID primitive.ObjectID, userID string, updates bson.M) (models.CategoryBudget, error) {
	if limit, ok := updates["limit"].(float64); ok {
		if limit <= 0 {
			return models.CategoryBudget{}, errors.New("budget limit must be greater than 0")
		}
	}

	if category, ok := updates["category"].(string); ok {
		updates["category"] = utils.SanitizeMongoValue(category)
	}

	updates["updated_at"] = time.Now()

	opts := mongoOptions.FindOneAndUpdate().SetReturnDocument(mongoOptions.After)
	result := b.categoryBudgetCol.FindOneAndUpdate(
		context.TODO(),
		bson.M{"_id": budgetID, "user_id": userID},
		bson.M{"$set": updates},
		opts,
	)

	var budget models.CategoryBudget
	if err := result.Decode(&budget); err != nil {
		return models.CategoryBudget{}, errors.New("category budget not found")
	}

	return budget, nil
}

func (b *BudgetService) DeleteCategoryBudget(budgetID primitive.ObjectID, userID string) error {
	result, err := b.categoryBudgetCol.DeleteOne(context.TODO(), bson.M{
		"_id":     budgetID,
		"user_id": userID,
	})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("category budget not found")
	}

	return nil
}

func (b *BudgetService) updateCategoryBudgetSpent(budget *models.CategoryBudget) {
	if b.transactionCol == nil {
		return
	}

	cursor, err := b.transactionCol.Find(context.TODO(), bson.M{
		"user_id":  budget.UserID,
		"month":    budget.Month,
		"category": budget.Category,
	})
	if err != nil {
		return
	}
	defer cursor.Close(context.TODO())

	var transactions []models.Transaction
	if err := cursor.All(context.TODO(), &transactions); err != nil {
		return
	}

	totalSpent := 0.0
	for _, t := range transactions {
		if t.Category == "outcome" {
			totalSpent += t.Amount
		}
	}

	budget.Spent = totalSpent
}

func (b *BudgetService) CalculateCategorySpending(userID, month, category string) (float64, error) {
	if b.transactionCol == nil {
		return 0, errors.New("transaction collection not initialized")
	}

	cursor, err := b.transactionCol.Find(context.TODO(), bson.M{
		"user_id":  userID,
		"month":    month,
		"category": category,
	})
	if err != nil {
		return 0, err
	}
	defer cursor.Close(context.TODO())

	var transactions []models.Transaction
	if err := cursor.All(context.TODO(), &transactions); err != nil {
		return 0, err
	}

	totalSpent := 0.0
	for _, t := range transactions {
		if t.Category == "outcome" {
			totalSpent += t.Amount
		}
	}

	return totalSpent, nil
}

func (b *BudgetService) GetCategoryBudgetWithSpending(userID, month, category string) (map[string]interface{}, error) {
	budget, err := b.GetCategoryBudgetByCategory(userID, month, category)
	if err != nil {
		return nil, err
	}

	return b.categoryBudgetToMap(budget), nil
}

func (b *BudgetService) GetAllCategoryBudgetsWithSpending(userID, month string) ([]map[string]interface{}, error) {
	budgets, err := b.GetCategoryBudgetByMonth(userID, month)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, budget := range budgets {
		result = append(result, b.categoryBudgetToMap(budget))
	}

	return result, nil
}

func (b *BudgetService) GetCategoryBudgetSummary(userID string) (map[string]interface{}, error) {
	budgets, err := b.GetUserCategoryBudgets(userID)
	if err != nil {
		return nil, err
	}

	if len(budgets) == 0 {
		return map[string]interface{}{
			"total_categories": 0,
			"total_budgeted":   0,
			"total_spent":      0,
			"total_remaining":  0,
		}, nil
	}

	totalBudgeted := 0.0
	totalSpent := 0.0

	for _, budget := range budgets {
		totalBudgeted += budget.Limit
		totalSpent += budget.Spent
	}

	return map[string]interface{}{
		"total_categories": len(budgets),
		"total_budgeted":   totalBudgeted,
		"total_spent":      totalSpent,
		"total_remaining":  totalBudgeted - totalSpent,
	}, nil
}

func (b *BudgetService) categoryBudgetToMap(budget models.CategoryBudget) map[string]interface{} {
	remaining := budget.Limit - budget.Spent

	percentageUsed := 0.0
	if budget.Limit > 0 {
		percentageUsed = (budget.Spent / budget.Limit) * 100
	}

	var status string
	switch {
	case percentageUsed >= 100:
		status = string(models.CategoryBudgetStatusExceeded)
	case percentageUsed >= 90:
		status = string(models.CategoryBudgetStatusWarning)
	case percentageUsed >= 75:
		status = string(models.CategoryBudgetStatusCaution)
	default:
		status = string(models.CategoryBudgetStatusSafe)
	}

	return map[string]interface{}{
		"budget":          budget,
		"remaining":       remaining,
		"percentage_used": percentageUsed,
		"status":          status,
	}
}
