package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"financeapi/essentials/models"
	"financeapi/essentials/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	mongoOpts "go.mongodb.org/mongo-driver/mongo/options"
)

type SpendingInsightService struct {
	collection          *mongo.Collection
	transactionService  *TransactionService
	debtService         *DebtService
	accountService      *AccountService
	budgetService       *BudgetService
	savingsGoalService  *SavingsGoalService
	billReminderService *BillReminderService
}

func NewSpendingInsightService(client *mongo.Client, dbName string) *SpendingInsightService {
	return &SpendingInsightService{
		collection: client.Database(dbName).Collection("spending_insights"),
	}
}

func (s *SpendingInsightService) SetTransactionService(ts *TransactionService) {
	s.transactionService = ts
}

func (s *SpendingInsightService) SetDebtService(ds *DebtService) {
	s.debtService = ds
}

func (s *SpendingInsightService) SetAccountService(as *AccountService) {
	s.accountService = as
}

func (s *SpendingInsightService) SetBudgetService(bs *BudgetService) {
	s.budgetService = bs
}

func (s *SpendingInsightService) SetSavingsGoalService(sgs *SavingsGoalService) {
	s.savingsGoalService = sgs
}

func (s *SpendingInsightService) SetBillReminderService(brs *BillReminderService) {
	s.billReminderService = brs
}

func (s *SpendingInsightService) GetUserInsights(userID primitive.ObjectID, limit int64) ([]models.SpendingInsight, error) {
	opts := mongoOpts.Find().SetSort(bson.D{{Key: "priority", Value: 1}, {Key: "created_at", Value: -1}}).SetLimit(limit)

	cursor, err := s.collection.Find(context.TODO(), bson.M{
		"user_id":    userID,
		"expires_at": bson.M{"$gt": time.Now()},
	}, opts)

	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var insights []models.SpendingInsight
	if err := cursor.All(context.TODO(), &insights); err != nil {
		return nil, err
	}

	return insights, nil
}

func (s *SpendingInsightService) CreateInsight(insight models.SpendingInsight) (models.SpendingInsight, error) {
	insight.ID = primitive.NewObjectID()
	insight.CreatedAt = time.Now()
	insight.ExpiresAt = time.Now().AddDate(0, 1, 0)
	insight.IsRead = false
	insight.IsActioned = false

	_, err := s.collection.InsertOne(context.TODO(), insight)
	if err != nil {
		return models.SpendingInsight{}, err
	}

	return insight, nil
}

func (s *SpendingInsightService) MarkAsRead(insightID primitive.ObjectID, userID primitive.ObjectID) error {
	_, err := s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": insightID, "user_id": userID},
		bson.M{"$set": bson.M{"is_read": true}},
	)
	return err
}

func (s *SpendingInsightService) MarkAsActioned(insightID primitive.ObjectID, userID primitive.ObjectID) error {
	_, err := s.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": insightID, "user_id": userID},
		bson.M{"$set": bson.M{"is_actioned": true, "is_read": true}},
	)
	return err
}

func (s *SpendingInsightService) GetFinancialHealthScore(userID primitive.ObjectID) (*models.FinancialHealthScore, error) {
	score := &models.FinancialHealthScore{
		ID:              primitive.NewObjectID(),
		UserID:          userID,
		Period:          "monthly",
		CalculatedAt:    time.Now(),
		Recommendations: []string{},
	}

	savingsRate, savingsRec := s.calculateSavingsRate(userID)
	debtLevel, debtRec := s.calculateDebtLevel(userID)
	budgetAdherence, budgetRec := s.calculateBudgetAdherence(userID)
	expenseControl, expenseRec := s.calculateExpenseControl(userID)

	// Set individual scores (0-100)
	score.SavingsRate = savingsRate
	score.DebtLevel = debtLevel
	score.BudgetAdherence = budgetAdherence
	score.ExpenseControl = expenseControl

	score.OverallScore = int(float64(savingsRate)*0.30 + float64(debtLevel)*0.25 + float64(budgetAdherence)*0.25 + float64(expenseControl)*0.20)

	score.Recommendations = append(score.Recommendations, savingsRec...)
	score.Recommendations = append(score.Recommendations, debtRec...)
	score.Recommendations = append(score.Recommendations, budgetRec...)
	score.Recommendations = append(score.Recommendations, expenseRec...)

	score.SavingsRatePercent = s.calculateSavingsRatePercent(userID)
	score.DebtToIncomeRatio = s.calculateDebtToIncomeRatio(userID)
	score.EmergencyFundMonths = s.calculateEmergencyFundMonths(userID)

	return score, nil
}

func (s *SpendingInsightService) calculateSavingsRate(userID primitive.ObjectID) (int, []string) {
	if s.transactionService == nil {
		return 50, []string{"Connect transaction service for accurate savings rate"}
	}

	// Use local timezone for consistent date handling
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	totalIncome := 0.0
	totalExpense := 0.0

	transactions, err := s.transactionService.ShowTransaction(userID.Hex())
	if err != nil {
		return 50, []string{}
	}

	for _, tx := range transactions {
		if (tx.Date.After(startOfMonth) || tx.Date.Equal(startOfMonth)) && tx.Date.Year() == now.Year() && tx.Date.Month() == now.Month() {
			if utils.IsIncomeByType(tx) {
				totalIncome += tx.Amount
			} else {
				totalExpense += tx.Amount
			}
		}
	}

	if totalIncome == 0 {
		return 30, []string{"No income recorded this month"}
	}

	savingsRate := ((totalIncome - totalExpense) / totalIncome) * 100

	var score int
	if savingsRate >= 20 {
		score = 100
	} else if savingsRate >= 15 {
		score = 85
	} else if savingsRate >= 10 {
		score = 70
	} else if savingsRate >= 5 {
		score = 55
	} else if savingsRate > 0 {
		score = 40
	} else {
		score = 20
	}

	var recommendations []string
	if savingsRate < 10 {
		recommendations = append(recommendations, "Try to save at least 10% of your income")
	}
	if savingsRate < 20 {
		recommendations = append(recommendations, "Consider reducing discretionary expenses to increase savings")
	}

	return score, recommendations
}

func (s *SpendingInsightService) calculateDebtLevel(userID primitive.ObjectID) (int, []string) {
	if s.debtService == nil {
		return 70, []string{"Connect debt service for accurate debt analysis"}
	}

	summary, err := s.debtService.GetDebtSummary(userID)
	if err != nil {
		return 70, []string{}
	}

	totalDebt := summary["total_debt"].(float64)
	totalDebtCount := summary["total_debts"].(int)

	if totalDebtCount == 0 {
		return 100, []string{"Great job! You're debt-free"}
	}

	monthlyIncome := s.getMonthlyIncome(userID)
	if monthlyIncome == 0 {
		return 50, []string{"Unable to calculate debt-to-income ratio - no income recorded"}
	}

	// Use monthly payment for DTI calculation (standard financial practice)
	monthlyPayment := summary["total_monthly_payment"].(float64)
	debtToIncome := (monthlyPayment / monthlyIncome) * 100

	var score int
	if debtToIncome <= 10 {
		score = 100
	} else if debtToIncome <= 20 {
		score = 85
	} else if debtToIncome <= 36 {
		score = 70
	} else if debtToIncome <= 50 {
		score = 50
	} else {
		score = 25
	}

	var recommendations []string
	if debtToIncome > 36 {
		recommendations = append(recommendations, fmt.Sprintf("Your debt-to-income ratio is high (%.0f%%). Consider a debt payoff plan", debtToIncome))
	}
	if totalDebtCount > 3 {
		recommendations = append(recommendations, "Consider consolidating your debts to simplify payments")
	}
	if monthlyPayment > monthlyIncome*0.5 {
		recommendations = append(recommendations, "More than 50% of income goes to debt payments - prioritize reducing this")
	}
	if totalDebt > monthlyIncome*6 {
		recommendations = append(recommendations, fmt.Sprintf("Total debt (%.0f) exceeds 6 months of income - focus on aggressive debt payoff", totalDebt))
	}

	return score, recommendations
}

func (s *SpendingInsightService) calculateBudgetAdherence(userID primitive.ObjectID) (int, []string) {
	if s.budgetService == nil {
		return 60, []string{"Connect budget service for accurate budget tracking"}
	}

	userIDStr := userID.Hex()
	currentMonth := time.Now().Format("2006-01")

	budgetsWithSpending, err := s.budgetService.GetAllBudgetsWithSpending(userIDStr)
	if err != nil {
		return 60, []string{}
	}

	// If no budgets, check current month specifically
	if len(budgetsWithSpending) == 0 {
		currentMonthBudget, err := s.budgetService.GetBudgetWithSpending(userIDStr, currentMonth)
		if err != nil {
			return 50, []string{"Create a budget to track spending"}
		}
		budgetsWithSpending = append(budgetsWithSpending, currentMonthBudget)
	}

	overBudgetCount := 0
	totalBudgets := len(budgetsWithSpending)
	exceedPercent := 0.0

	for _, b := range budgetsWithSpending {
		spent := b["spending"].(float64)
		budget := b["budget"].(models.MonthlyBudget)
		budgetAmount := budget.Limit

		if budgetAmount > 0 && spent > budgetAmount {
			overBudgetCount++
			exceedPercent += (spent - budgetAmount) / budgetAmount * 100
		}
	}

	if totalBudgets == 0 {
		return 50, []string{"No active budgets to track"}
	}

	adherenceRate := float64(totalBudgets-overBudgetCount) / float64(totalBudgets) * 100

	var score int
	if adherenceRate >= 90 {
		score = 100
	} else if adherenceRate >= 75 {
		score = 85
	} else if adherenceRate >= 60 {
		score = 70
	} else if adherenceRate >= 40 {
		score = 55
	} else {
		score = 35
	}

	var recommendations []string
	if adherenceRate < 75 {
		recommendations = append(recommendations, "You're over budget frequently. Consider adjusting your budget limits")
	}
	if exceedPercent > 0 {
		recommendations = append(recommendations, "Review categories where you consistently overspend")
	}

	return score, recommendations
}

func (s *SpendingInsightService) calculateExpenseControl(userID primitive.ObjectID) (int, []string) {
	if s.transactionService == nil {
		return 60, []string{}
	}

	now := time.Now()
	threeMonthsAgo := now.AddDate(0, -3, 0)

	transactions, err := s.transactionService.ShowTransaction(userID.Hex())
	if err != nil {
		return 60, []string{}
	}

	monthlyExpenses := make(map[string]float64)
	var recentExpenses []float64

	for _, tx := range transactions {
		if tx.Date.After(threeMonthsAgo) && utils.IsOutcomeByType(tx) {
			monthKey := tx.Date.Format("2006-01")
			monthlyExpenses[monthKey] += tx.Amount
			recentExpenses = append(recentExpenses, tx.Amount)
		}
	}

	if len(monthlyExpenses) < 2 {
		return 60, []string{"Record more transactions for accurate expense analysis"}
	}

	// Calculate expense variance
	var sum float64
	for _, exp := range recentExpenses {
		sum += exp
	}
	avgExpense := sum / float64(len(recentExpenses))

	// Dynamic anomaly detection: flag transactions > 3x average expense
	anomalyCount := 0
	anomalyThreshold := avgExpense * 3
	if anomalyThreshold < 50000 { // Minimum threshold of 50k to avoid false positives on small budgets
		anomalyThreshold = 50000
	}

	for _, exp := range recentExpenses {
		if exp > anomalyThreshold {
			anomalyCount++
		}
	}

	// Get last 3 months of data
	var monthTotals []float64
	for i := 2; i >= 0; i-- {
		month := now.AddDate(0, -i, 1).Format("2006-01")
		monthTotals = append(monthTotals, monthlyExpenses[month])
	}

	var trend float64
	// Division by zero protection
	if len(monthTotals) >= 2 && monthTotals[0] > 0 {
		trend = (monthTotals[len(monthTotals)-1] - monthTotals[0]) / monthTotals[0] * 100
	}

	var score int
	if anomalyCount == 0 && trend < 5 {
		score = 100
	} else if anomalyCount <= 1 && trend < 10 {
		score = 80
	} else if anomalyCount <= 2 && trend < 20 {
		score = 65
	} else if trend < 30 {
		score = 50
	} else {
		score = 35
	}

	var recommendations []string
	if anomalyCount > 2 {
		recommendations = append(recommendations, fmt.Sprintf("You have %d large unusual expenses. Review them for potential savings", anomalyCount))
	}
	if trend > 15 {
		recommendations = append(recommendations, "Your expenses are increasing month-over-month. Consider cutting back")
	}
	if avgExpense > 0 && anomalyThreshold > avgExpense*5 {
		recommendations = append(recommendations, "Your spending is relatively stable with few large purchases")
	}

	return score, recommendations
}

func (s *SpendingInsightService) getMonthlyIncome(userID primitive.ObjectID) float64 {
	if s.transactionService == nil {
		return 0
	}

	now := time.Now()
	totalIncome := 0.0

	transactions, err := s.transactionService.ShowTransaction(userID.Hex())
	if err != nil {
		return 0
	}

	for _, tx := range transactions {
		// Filter to current month only
		if tx.Date.Year() == now.Year() && tx.Date.Month() == now.Month() && utils.IsIncomeByType(tx) {
			totalIncome += tx.Amount
		}
	}

	return totalIncome
}

func (s *SpendingInsightService) calculateSavingsRatePercent(userID primitive.ObjectID) float64 {
	now := time.Now()

	totalIncome := 0.0
	totalExpense := 0.0

	if s.transactionService == nil {
		return 0
	}

	transactions, _ := s.transactionService.ShowTransaction(userID.Hex())
	for _, tx := range transactions {
		// Filter to current month only
		if tx.Date.Year() == now.Year() && tx.Date.Month() == now.Month() {
			if utils.IsIncomeByType(tx) {
				totalIncome += tx.Amount
			} else {
				totalExpense += tx.Amount
			}
		}
	}

	if totalIncome == 0 {
		return 0
	}

	return math.Round(((totalIncome - totalExpense) / totalIncome) * 100)
}

func (s *SpendingInsightService) calculateDebtToIncomeRatio(userID primitive.ObjectID) float64 {
	if s.debtService == nil {
		return 0
	}

	summary, err := s.debtService.GetDebtSummary(userID)
	if err != nil {
		return 0
	}

	// Use monthly payment for DTI (standard financial practice)
	monthlyPayment := summary["total_monthly_payment"].(float64)
	monthlyIncome := s.getMonthlyIncome(userID)

	if monthlyIncome == 0 {
		return 0
	}

	return math.Round((monthlyPayment / monthlyIncome) * 100)
}

func (s *SpendingInsightService) calculateEmergencyFundMonths(userID primitive.ObjectID) float64 {
	now := time.Now()
	totalSavings := 0.0

	// 1. Include bank and cash accounts
	if s.accountService != nil {
		accounts, err := s.accountService.GetUserAccounts(userID)
		if err == nil {
			for _, acc := range accounts {
				if acc.Type == models.AccountTypeBank || acc.Type == models.AccountTypeCash {
					totalSavings += acc.CurrentBalance
				}
			}
		}
	}

	// 2. Include savings goals (liquid savings)
	if s.savingsGoalService != nil {
		goals, err := s.savingsGoalService.GetUserSavingsGoals(userID.Hex())
		if err == nil {
			for _, goal := range goals {
				// Only count active goals with progress
				if goal.Status == "active" && goal.CurrentAmount > 0 {
					totalSavings += goal.CurrentAmount
				}
			}
		}
	}

	// Calculate monthly expenses for current month
	monthlyExpenses := 0.0
	if s.transactionService != nil {
		transactions, _ := s.transactionService.ShowTransaction(userID.Hex())
		for _, tx := range transactions {
			if tx.Date.Year() == now.Year() && tx.Date.Month() == now.Month() && utils.IsOutcomeByType(tx) {
				monthlyExpenses += tx.Amount
			}
		}
	}

	if monthlyExpenses == 0 {
		return 0
	}

	emergencyMonths := math.Round(totalSavings / monthlyExpenses)

	// Cap at reasonable maximum (e.g., 24 months)
	if emergencyMonths > 24 {
		emergencyMonths = 24
	}

	return emergencyMonths
}
