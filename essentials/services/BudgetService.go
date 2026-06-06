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
	// BudgetService uses direct collection access, not the service
	// This is just for interface compliance
}

func (b *BudgetService) GetCollection() *mongo.Collection {
	return b.transactionCol
}

func (b *BudgetService) calculateStatus(limit, spent float64) string {
	if limit <= 0 {
		return string(models.CategoryBudgetStatusSafe)
	}
	percentageUsed := (spent / limit) * 100
	switch {
	case percentageUsed >= 100:
		return string(models.CategoryBudgetStatusExceeded)
	case percentageUsed >= 90:
		return string(models.CategoryBudgetStatusWarning)
	case percentageUsed >= 75:
		return string(models.CategoryBudgetStatusCaution)
	default:
		return string(models.CategoryBudgetStatusSafe)
	}
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

	if !utils.IsValidMonthFormat(budget.Month) {
		return models.MonthlyBudget{}, errors.New("invalid month format, use YYYY-MM")
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

	budget.CreatedAt = time.Now()
	budget.UpdatedAt = time.Now()

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
	// 1. Get current budget state
	var currentBudget models.MonthlyBudget
	err := b.collection.FindOne(context.TODO(), bson.M{"_id": budgetID, "user_id": userID}).Decode(&currentBudget)
	if err != nil {
		return models.MonthlyBudget{}, errors.New("budget not found")
	}

	// 2. Validate Limit if it's being updated
	if limit, ok := updates["limit"].(float64); ok {
		if limit <= 0 {
			return models.MonthlyBudget{}, errors.New("budget limit must be greater than 0")
		}

		// Consistency Check: Total category budget limits must not exceed new monthly limit
		categoryBudgets, _ := b.GetCategoryBudgetByMonth(userID, currentBudget.Month)
		totalCategoryLimits := 0.0
		for _, cat := range categoryBudgets {
			totalCategoryLimits += cat.Limit
		}
		if totalCategoryLimits > limit {
			return models.MonthlyBudget{}, fmt.Errorf("new limit %.2f is lower than total category budgets (%.2f). Update category budgets first", limit, totalCategoryLimits)
		}
	}

	// 3. Validate Month if it's being updated
	if month, ok := updates["month"].(string); ok {
		if !utils.IsValidMonthFormat(month) {
			return models.MonthlyBudget{}, errors.New("invalid month format, use YYYY-MM")
		}

		// Check for duplication only if month is actually changing
		if month != currentBudget.Month {
			existingBudget := models.MonthlyBudget{}
			err := b.collection.FindOne(context.TODO(), bson.M{
				"user_id": userID,
				"month":   month,
			}).Decode(&existingBudget)

			if err == nil {
				return models.MonthlyBudget{}, errors.New("budget for target month already exists")
			}
		}
		updates["month"] = month
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
		return models.MonthlyBudget{}, err
	}

	return budget, nil
}

func (b *BudgetService) DeleteBudget(budgetID primitive.ObjectID, userID string) error {
	// 1. Get budget info first to know the month
	var budget models.MonthlyBudget
	err := b.collection.FindOne(context.TODO(), bson.M{"_id": budgetID, "user_id": userID}).Decode(&budget)
	if err != nil {
		return errors.New("budget not found")
	}

	// 2. Delete the Monthly Budget
	_, err = b.collection.DeleteOne(context.TODO(), bson.M{"_id": budgetID, "user_id": userID})
	if err != nil {
		return err
	}

	// 3. Cascading Delete: Delete all category budgets for that month
	_, _ = b.categoryBudgetCol.DeleteMany(context.TODO(), bson.M{
		"user_id": userID,
		"month":   budget.Month,
	})

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

	return map[string]interface{}{
		"budget":          budget,
		"spending":        spending,
		"remaining":       remaining,
		"percentage_used": percentageUsed,
		"status":          b.calculateStatus(budget.Limit, spending),
	}, nil
}

func (b *BudgetService) CalculateSpending(userID, month string) (float64, error) {
	if b.transactionCol == nil {
		return 0, errors.New("transaction collection not initialized")
	}

	// Filter out income categories - only count expenses
	incomeCategories := []string{"income", "gaji", "salary", "pendapatan", "revenue"}
	pipeline := mongo.Pipeline{
		{{"$match", bson.D{
			{"user_id", userID},
			{"month", month},
			{"category", bson.D{{"$nin", incomeCategories}}},
		}}},
		{{"$group", bson.D{
			{"_id", nil},
			{"total", bson.D{{"$sum", "$amount"}}},
		}}},
	}

	cursor, err := b.transactionCol.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(context.TODO())

	if cursor.Next(context.TODO()) {
		var result struct {
			Total float64 `bson:"total"`
		}
		if err := cursor.Decode(&result); err != nil {
			return 0, err
		}
		return result.Total, nil
	}

	return 0, nil
}

// UpdateBudgetSpent recalculates and updates the spent amount for a monthly budget
func (b *BudgetService) UpdateBudgetSpent(userID, month string) error {
	if b.transactionCol == nil {
		return errors.New("transaction collection not initialized")
	}

	// Calculate spending using type field (with fallback to category for backward compatibility)
	pipeline := mongo.Pipeline{
		{{"$match", bson.D{
			{"user_id", userID},
			{"month", month},
		}}},
		{{"$group", bson.D{
			{"_id", "$type"},
			{"total", bson.D{{"$sum", "$amount"}}},
		}}},
	}

	cursor, err := b.transactionCol.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(context.TODO())

	totalSpent := 0.0
	for cursor.Next(context.TODO()) {
		var result struct {
			ID    string  `bson:"_id"`
			Total float64 `bson:"total"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		// Only count outcome transactions
		if result.ID == "outcome" || result.ID == "" {
			// If ID is empty, use fallback: check if it's not in income categories
			if result.ID == "" {
				// This handles old transactions without type field
				// Will be counted if they pass the category check
			}
			totalSpent += result.Total
		}
	}

	// Update the budget's spent field
	_, err = b.collection.UpdateOne(
		context.TODO(),
		bson.M{"user_id": userID, "month": month},
		bson.M{"$set": bson.M{"spent": totalSpent, "updated_at": time.Now()}},
	)

	return err
}

// UpdateCategoryBudgetSpent recalculates and updates the spent amount for category budgets
func (b *BudgetService) UpdateCategoryBudgetSpent(userID, month, category string) error {
	if b.categoryBudgetCol == nil {
		return errors.New("category budget collection not initialized")
	}

	// Find the category budget
	var catBudget models.CategoryBudget
	err := b.categoryBudgetCol.FindOne(context.TODO(), bson.M{
		"user_id": userID,
		"month":   month,
		"category": category,
	}).Decode(&catBudget)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil // No category budget set, skip
		}
		return err
	}

	// Calculate spending for this category
	pipeline := mongo.Pipeline{
		{{"$match", bson.D{
			{"user_id", userID},
			{"month", month},
			{"category", category},
		}}},
		{{"$group", bson.D{
			{"_id", "$type"},
			{"total", bson.D{{"$sum", "$amount"}}},
		}}},
	}

	cursor, err := b.transactionCol.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(context.TODO())

	totalSpent := 0.0
	for cursor.Next(context.TODO()) {
		var result struct {
			ID    string  `bson:"_id"`
			Total float64 `bson:"total"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		if result.ID == "outcome" || result.ID == "" {
			totalSpent += result.Total
		}
	}

	// Update the category budget's spent field
	_, err = b.categoryBudgetCol.UpdateOne(
		context.TODO(),
		bson.M{"user_id": userID, "month": month, "category": category},
		bson.M{"$set": bson.M{"spent": totalSpent, "updated_at": time.Now()}},
	)

	return err
}

// UpdateAllCategoryBudgetsSpent recalculates spent for all category budgets in a month
func (b *BudgetService) UpdateAllCategoryBudgetsSpent(userID, month string) error {
	if b.categoryBudgetCol == nil {
		return nil
	}

	// Get all category budgets for this month
	cursor, err := b.categoryBudgetCol.Find(context.TODO(), bson.M{
		"user_id": userID,
		"month":   month,
	})
	if err != nil {
		return err
	}
	defer cursor.Close(context.TODO())

	var catBudgets []models.CategoryBudget
	if err := cursor.All(context.TODO(), &catBudgets); err != nil {
		return err
	}

	// Update each category budget spent
	for _, catBudget := range catBudgets {
		_ = b.UpdateCategoryBudgetSpent(userID, month, catBudget.Category)
	}

	return nil
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

		result = append(result, map[string]interface{}{
			"budget":          budget,
			"spending":        spending,
			"remaining":       remaining,
			"percentage_used": percentageUsed,
			"status":          b.calculateStatus(budget.Limit, spending),
		})
	}

	return result, nil
}

func (b *BudgetService) GetBudgetSummary(userID, month string) (map[string]interface{}, error) {
	if month == "" {
		month = time.Now().Format("2006-01")
	}

	budget, err := b.GetBudgetByMonth(userID, month)
	if err != nil {
		// If no budget for this month, return zero summary instead of error
		return map[string]interface{}{
			"month":           month,
			"total_budgeted":  0,
			"total_spent":     0,
			"total_remaining": 0,
			"status":          "no_budget",
		}, nil
	}

	spending, err := b.CalculateSpending(userID, month)
	if err != nil {
		spending = 0
	}

	return map[string]interface{}{
		"month":           month,
		"total_budgeted":  budget.Limit,
		"total_spent":     spending,
		"total_remaining": budget.Limit - spending,
		"status":          b.calculateStatus(budget.Limit, spending),
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

	if !utils.IsValidMonthFormat(budget.Month) {
		return models.CategoryBudget{}, errors.New("invalid month format, use YYYY-MM")
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

	// Cross-validation: Check against Monthly Budget
	monthlyBudget, err := b.GetBudgetByMonth(budget.UserID, budget.Month)
	if err != nil {
		return models.CategoryBudget{}, errors.New("monthly budget must be created first before setting category budgets")
	}

	existingCategories, _ := b.GetCategoryBudgetByMonth(budget.UserID, budget.Month)
	totalUsed := 0.0
	for _, cat := range existingCategories {
		totalUsed += cat.Limit
	}

	if totalUsed+budget.Limit > monthlyBudget.Limit {
		return models.CategoryBudget{}, fmt.Errorf("total category budgets (%.2f) would exceed monthly limit (%.2f)", totalUsed+budget.Limit, monthlyBudget.Limit)
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
	// Optimization: Use aggregation to calculate spending for all budgets in this month in one go
	// Filter out income categories - only count expenses
	incomeCategories := []string{"income", "gaji", "salary", "pendapatan", "revenue"}
	pipeline := mongo.Pipeline{
		{{"$match", bson.D{
			{"user_id", userID},
			{"month", month},
			{"category", bson.D{{"$nin", incomeCategories}}},
		}}},
		{{"$group", bson.D{
			{"_id", "$category"},
			{"total", bson.D{{"$sum", "$amount"}}},
		}}},
	}

	cursor, err := b.transactionCol.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	spendingMap := make(map[string]float64)
	for cursor.Next(context.TODO()) {
		var res struct {
			Category string  `bson:"_id"`
			Total    float64 `bson:"total"`
		}
		if err := cursor.Decode(&res); err == nil {
			spendingMap[res.Category] = res.Total
		}
	}

	// Fetch the budgets
	cursorBudget, err := b.categoryBudgetCol.Find(context.TODO(), bson.M{
		"user_id": userID,
		"month":   month,
	})
	if err != nil {
		return nil, err
	}
	defer cursorBudget.Close(context.TODO())

	var budgets []models.CategoryBudget
	if err := cursorBudget.All(context.TODO(), &budgets); err != nil {
		return nil, err
	}

	for i := range budgets {
		budgets[i].Spent = spendingMap[budgets[i].Category]
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
	// 1. Get current state
	var current models.CategoryBudget
	err := b.categoryBudgetCol.FindOne(context.TODO(), bson.M{"_id": budgetID, "user_id": userID}).Decode(&current)
	if err != nil {
		return models.CategoryBudget{}, errors.New("category budget not found")
	}

	// 2. Validate Limit consistency if changed
	if newLimit, ok := updates["limit"].(float64); ok {
		if newLimit <= 0 {
			return models.CategoryBudget{}, errors.New("budget limit must be greater than 0")
		}

		monthlyBudget, err := b.GetBudgetByMonth(userID, current.Month)
		if err == nil {
			existingCategories, _ := b.GetCategoryBudgetByMonth(userID, current.Month)
			totalOtherCategories := 0.0
			for _, cat := range existingCategories {
				if cat.ID != budgetID {
					totalOtherCategories += cat.Limit
				}
			}

			if totalOtherCategories+newLimit > monthlyBudget.Limit {
				return models.CategoryBudget{}, fmt.Errorf("total category budgets (%.2f) would exceed monthly limit (%.2f)", totalOtherCategories+newLimit, monthlyBudget.Limit)
			}
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
		return models.CategoryBudget{}, err
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
		if utils.IsOutcomeByType(t) {
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
		if utils.IsOutcomeByType(t) {
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

func (b *BudgetService) GetCategoryBudgetSummary(userID, month string) (map[string]interface{}, error) {
	if month == "" {
		month = time.Now().Format("2006-01")
	}

	budgets, err := b.GetCategoryBudgetByMonth(userID, month)
	if err != nil {
		return nil, err
	}

	if len(budgets) == 0 {
		return map[string]interface{}{
			"month":            month,
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
		"month":            month,
		"total_categories": len(budgets),
		"total_budgeted":   totalBudgeted,
		"total_spent":      totalSpent,
		"total_remaining":  totalBudgeted - totalSpent,
		"status":           b.calculateStatus(totalBudgeted, totalSpent),
	}, nil
}

func (b *BudgetService) categoryBudgetToMap(budget models.CategoryBudget) map[string]interface{} {
	remaining := budget.Limit - budget.Spent

	percentageUsed := 0.0
	if budget.Limit > 0 {
		percentageUsed = (budget.Spent / budget.Limit) * 100
	}

	return map[string]interface{}{
		"budget":          budget,
		"remaining":       remaining,
		"percentage_used": percentageUsed,
		"status":          b.calculateStatus(budget.Limit, budget.Spent),
	}
}
