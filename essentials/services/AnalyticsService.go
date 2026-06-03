package services

import (
	"fmt"
	"time"

	"financeapi/essentials/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AnalyticsService struct {
	transactionService  *TransactionService
	investmentService   *InvestmentService
	debtService         *DebtService
	savingsGoalService  *SavingsGoalService
	accountService      *AccountService
	billReminderService *BillReminderService
	priceService        *PriceService
}

func NewAnalyticsService() *AnalyticsService {
	return &AnalyticsService{}
}

func (s *AnalyticsService) SetTransactionService(ts *TransactionService) {
	s.transactionService = ts
}

func (s *AnalyticsService) SetInvestmentService(is *InvestmentService) {
	s.investmentService = is
}

func (s *AnalyticsService) SetDebtService(ds *DebtService) {
	s.debtService = ds
}

func (s *AnalyticsService) SetSavingsGoalService(sgs *SavingsGoalService) {
	s.savingsGoalService = sgs
}

func (s *AnalyticsService) SetAccountService(as *AccountService) {
	s.accountService = as
}

func (s *AnalyticsService) SetBillReminderService(brs *BillReminderService) {
	s.billReminderService = brs
}

func (s *AnalyticsService) SetPriceService(ps *PriceService) {
	s.priceService = ps
}

func (s *AnalyticsService) GetFullAnalytics(userID string, period string) (*models.AnalyticsReport, error) {
	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	startDate, endDate := s.getDateRange(period)

	report := &models.AnalyticsReport{
		ID:        primitive.NewObjectID(),
		UserID:    userOID,
		Period:    period,
		StartDate: startDate,
		EndDate:   endDate,
	}

	cashFlow, err := s.getCashFlow(userID, startDate, endDate)
	if err == nil {
		report.CashFlow = *cashFlow
	}

	netWorth, assets, liabilities, err := s.calculateNetWorth(userOID)
	if err == nil {
		report.NetWorth = netWorth
		report.TotalAssets = assets
		report.TotalLiabilities = liabilities
	}

	allocations, err := s.getAssetAllocation(userOID)
	if err == nil {
		report.AssetAllocation = allocations
	}

	expenses, err := s.getExpenseBreakdown(userID, startDate, endDate)
	if err == nil {
		report.ExpenseBreakdown = expenses
	}

	income, err := s.getIncomeBreakdown(userID, startDate, endDate)
	if err == nil {
		report.IncomeSources = income
	}

	debts, err := s.getDebtBreakdown(userOID)
	if err == nil {
		report.DebtBreakdown = debts
	}

	if report.CashFlow.TotalIncome > 0 {
		report.SavingsRate = ((report.CashFlow.TotalIncome - report.CashFlow.TotalExpenses) / report.CashFlow.TotalIncome) * 100
		report.ExpenseToIncome = (report.CashFlow.TotalExpenses / report.CashFlow.TotalIncome) * 100
	}

	report.CreatedAt = time.Now()

	return report, nil
}

func (s *AnalyticsService) GetQuickStats(userID string) (*models.QuickStats, error) {
	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	weekStart := today.AddDate(0, 0, -int(now.Weekday()))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	stats := &models.QuickStats{}

	if s.transactionService != nil {
		todayFilter := models.FilterTransaction{
			FromDate: today.Format("2006-01-02"),
			ToDate:   today.Format("2006-01-02"),
		}
		report, _ := s.transactionService.GetReport(userID, todayFilter)
		stats.TodaySpending = report["outcome"]
		stats.TodayIncome = report["income"]

		weekFilter := models.FilterTransaction{
			FromDate: weekStart.Format("2006-01-02"),
			ToDate:   now.Format("2006-01-02"),
		}
		report, _ = s.transactionService.GetReport(userID, weekFilter)
		stats.WeekSpending = report["outcome"]
		stats.WeekIncome = report["income"]

		monthFilter := models.FilterTransaction{
			FromDate: monthStart.Format("2006-01-02"),
			ToDate:   now.Format("2006-01-02"),
		}
		report, _ = s.transactionService.GetReport(userID, monthFilter)
		stats.MonthSpending = report["outcome"]
		stats.MonthIncome = report["income"]
		stats.MonthSavings = report["income"] - report["outcome"]

		// Get top expenses for the month
		expenseFilter := models.FilterTransaction{
			FromDate: monthStart.Format("2006-01-02"),
			ToDate:   now.Format("2006-01-02"),
			Type:     "outcome",
		}
		topExpenses, _ := s.transactionService.GetCategoryStats(userID, expenseFilter)
		// Limit to top 3 expenses
		if len(topExpenses) > 3 {
			stats.TopExpenses = topExpenses[:3]
		} else {
			stats.TopExpenses = topExpenses
		}
	}

	if s.investmentService != nil && s.priceService != nil {
		investments, _ := s.investmentService.GetUserInvestments(userOID)
		totalValue := 0.0
		for _, inv := range investments {
			var price float64
			var err error
			if string(inv.Type) == "crypto" {
				price, err = s.priceService.GetCryptoPrice(inv.Symbol, "idr")
			} else {
				price, err = s.priceService.GetStockPrice(inv.Symbol, true)
			}
			if err == nil {
				totalValue += price * inv.Quantity
			}
		}
		stats.InvestmentValue = totalValue
	}

	if s.investmentService != nil {
		netWorth, _, _, _ := s.calculateNetWorth(userOID)
		stats.NetWorth = netWorth
	}

	if s.billReminderService != nil {
		bills, _ := s.billReminderService.GetUserBillReminders(userOID)
		stats.ActiveBills = len(bills)
		upcoming := 0.0
		for _, bill := range bills {
			if !bill.IsPaid && bill.NextDueDate.After(now) {
				upcoming += bill.Amount
			}
		}
		stats.UpcomingBills = upcoming
	}

	return stats, nil
}

func (s *AnalyticsService) GetGoalProgress(userID string) ([]models.GoalProgress, error) {
	if s.savingsGoalService == nil {
		return nil, fmt.Errorf("savings goal service not set")
	}

	goals, err := s.savingsGoalService.GetUserSavingsGoals(userID)
	if err != nil {
		return nil, err
	}

	var progress []models.GoalProgress
	for _, goal := range goals {
		currentAmount := goal.CurrentAmount
		remaining := goal.TargetAmount - currentAmount
		percentage := 0.0
		if goal.TargetAmount > 0 {
			percentage = (currentAmount / goal.TargetAmount) * 100
		}

		p := models.GoalProgress{
			GoalID:          goal.ID,
			GoalName:        goal.Name,
			TargetAmount:    goal.TargetAmount,
			CurrentAmount:   currentAmount,
			RemainingAmount: remaining,
			Percentage:      percentage,
			TargetDate:      goal.TargetDate,
		}

		if !goal.TargetDate.IsZero() && remaining > 0 {
			daysRemaining := goal.TargetDate.Sub(time.Now()).Hours() / 24 / 30
			if daysRemaining > 0 {
				p.MonthlyNeeded = remaining / daysRemaining
				p.EstimatedCompletion = time.Now().AddDate(0, int(daysRemaining), 0)
			}
		}

		progress = append(progress, p)
	}

	return progress, nil
}

// ============================================================
// HELPER METHODS
// ============================================================

func (s *AnalyticsService) getDateRange(period string) (time.Time, time.Time) {
	now := time.Now()
	var startDate, endDate time.Time

	switch period {
	case "daily":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		endDate = now
	case "weekly":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startDate = now.AddDate(0, 0, -(weekday - 1))
		startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
		endDate = now
	case "monthly":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		endDate = now
	case "yearly":
		startDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		endDate = now
	default:
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		endDate = now
	}

	return startDate, endDate
}

func (s *AnalyticsService) getCashFlow(userID string, startDate, endDate time.Time) (*models.CashFlowSummary, error) {
	if s.transactionService == nil {
		return &models.CashFlowSummary{}, nil
	}

	filter := models.FilterTransaction{
		FromDate: startDate.Format("2006-01-02"),
		ToDate:   endDate.Format("2006-01-02"),
	}

	report, err := s.transactionService.GetReport(userID, filter)
	if err != nil {
		return nil, err
	}

	income := report["income"]
	expenses := report["outcome"]
	netCashFlow := income - expenses

	days := endDate.Sub(startDate).Hours() / 24
	if days < 1 {
		days = 1
	}

	return &models.CashFlowSummary{
		TotalIncome:    income,
		TotalExpenses:  expenses,
		NetCashFlow:    netCashFlow,
		AverageDaily:   netCashFlow / days,
		AverageMonthly: netCashFlow * 30 / days,
	}, nil
}

func (s *AnalyticsService) calculateNetWorth(userOID primitive.ObjectID) (float64, float64, float64, error) {
	var totalAssets float64
	var totalLiabilities float64

	if s.investmentService != nil && s.priceService != nil {
		investments, _ := s.investmentService.GetUserInvestments(userOID)
		for _, inv := range investments {
			var price float64
			var err error
			if string(inv.Type) == "crypto" {
				price, err = s.priceService.GetCryptoPrice(inv.Symbol, "idr")
			} else {
				price, err = s.priceService.GetStockPrice(inv.Symbol, true)
			}
			if err == nil {
				totalAssets += price * inv.Quantity
			}
		}
	}

	if s.accountService != nil {
		accounts, _ := s.accountService.GetUserAccounts(userOID)
		for _, acc := range accounts {
			totalAssets += acc.CurrentBalance
		}
	}

	if s.debtService != nil {
		debts, _ := s.debtService.GetUserDebts(userOID)
		for _, debt := range debts {
			totalLiabilities += debt.CurrentBalance
		}
	}

	netWorth := totalAssets - totalLiabilities

	return netWorth, totalAssets, totalLiabilities, nil
}

func (s *AnalyticsService) getAssetAllocation(userOID primitive.ObjectID) ([]models.AllocationItem, error) {
	var allocations []models.AllocationItem

	if s.investmentService != nil && s.priceService != nil {
		investments, _ := s.investmentService.GetUserInvestments(userOID)

		typeAmounts := make(map[string]float64)
		typeCounts := make(map[string]int)

		for _, inv := range investments {
			var price float64
			var err error
			if string(inv.Type) == "crypto" {
				price, err = s.priceService.GetCryptoPrice(inv.Symbol, "idr")
			} else {
				price, err = s.priceService.GetStockPrice(inv.Symbol, true)
			}
			if err == nil {
				value := price * inv.Quantity
				typeName := string(inv.Type)
				typeAmounts[typeName] += value
				typeCounts[typeName]++
			}
		}

		for typeName, amount := range typeAmounts {
			allocations = append(allocations, models.AllocationItem{
				Category: typeName,
				Amount:   amount,
				Count:    typeCounts[typeName],
			})
		}
	}

	if s.savingsGoalService != nil {
		goals, _ := s.savingsGoalService.GetUserSavingsGoals(userOID.Hex())
		savingsAmount := 0.0
		for _, goal := range goals {
			savingsAmount += goal.CurrentAmount
		}
		if savingsAmount > 0 {
			allocations = append(allocations, models.AllocationItem{
				Category: "savings",
				Amount:   savingsAmount,
			})
		}
	}

	if s.accountService != nil {
		accounts, _ := s.accountService.GetUserAccounts(userOID)
		cashAmount := 0.0
		for _, acc := range accounts {
			cashAmount += acc.CurrentBalance
		}
		if cashAmount > 0 {
			allocations = append(allocations, models.AllocationItem{
				Category: "cash",
				Amount:   cashAmount,
			})
		}
	}

	var total float64
	for _, a := range allocations {
		total += a.Amount
	}
	if total > 0 {
		for i := range allocations {
			allocations[i].Percentage = (allocations[i].Amount / total) * 100
		}
	}

	return allocations, nil
}

func (s *AnalyticsService) getExpenseBreakdown(userID string, startDate, endDate time.Time) ([]models.CategoryStat, error) {
	if s.transactionService == nil {
		return []models.CategoryStat{}, nil
	}

	filter := models.FilterTransaction{
		FromDate: startDate.Format("2006-01-02"),
		ToDate:   endDate.Format("2006-01-02"),
		Type:     "outcome",
	}

	return s.transactionService.GetCategoryStats(userID, filter)
}

func (s *AnalyticsService) getIncomeBreakdown(userID string, startDate, endDate time.Time) ([]models.CategoryStat, error) {
	if s.transactionService == nil {
		return []models.CategoryStat{}, nil
	}

	filter := models.FilterTransaction{
		FromDate: startDate.Format("2006-01-02"),
		ToDate:   endDate.Format("2006-01-02"),
		Type:     "income",
	}

	return s.transactionService.GetCategoryStats(userID, filter)
}

func (s *AnalyticsService) getDebtBreakdown(userOID primitive.ObjectID) ([]models.DebtSummary, error) {
	if s.debtService == nil {
		return []models.DebtSummary{}, nil
	}

	debts, err := s.debtService.GetUserDebts(userOID)
	if err != nil {
		return nil, err
	}

	var summaries []models.DebtSummary
	for _, debt := range debts {
		summaries = append(summaries, models.DebtSummary{
			Type:           string(debt.Type),
			Name:           debt.Name,
			Original:       debt.OriginalAmount,
			Current:        debt.CurrentBalance,
			MonthlyPayment: debt.PaymentAmount,
			InterestRate:   debt.InterestRate,
		})
	}

	return summaries, nil
}

func (s *AnalyticsService) GetNetWorthDetail(userOID primitive.ObjectID) (map[string]interface{}, error) {
	// 1. Assets
	var accountDetails []map[string]interface{}
	var totalAssets float64
	if s.accountService != nil {
		accounts, _ := s.accountService.GetUserAccounts(userOID)
		for _, acc := range accounts {
			totalAssets += acc.CurrentBalance
			accountDetails = append(accountDetails, map[string]interface{}{
				"name":    acc.Name,
				"balance": acc.CurrentBalance,
				"type":    acc.Type,
			})
		}
	}

	var investmentDetails []map[string]interface{}
	if s.investmentService != nil {
		investments, _ := s.investmentService.GetUserInvestments(userOID)
		for _, inv := range investments {
			totalAssets += inv.TotalValue
			investmentDetails = append(investmentDetails, map[string]interface{}{
				"name":   inv.Name,
				"symbol": inv.Symbol,
				"value":  inv.TotalValue,
				"type":   inv.Type,
			})
		}
	}

	// 2. Liabilities
	var debtDetails []map[string]interface{}
	var totalLiabilities float64
	if s.debtService != nil {
		debts, _ := s.debtService.GetUserDebts(userOID)
		for _, d := range debts {
			totalLiabilities += d.CurrentBalance
			debtDetails = append(debtDetails, map[string]interface{}{
				"name":      d.Name,
				"remaining": d.CurrentBalance,
				"type":      d.Type,
			})
		}
	}

	return map[string]interface{}{
		"total_assets":      totalAssets,
		"total_liabilities": totalLiabilities,
		"net_worth":         totalAssets - totalLiabilities,
		"assets_breakdown": map[string]interface{}{
			"accounts":    accountDetails,
			"investments": investmentDetails,
		},
		"liabilities_breakdown": debtDetails,
		"last_updated":          time.Now(),
	}, nil
}
