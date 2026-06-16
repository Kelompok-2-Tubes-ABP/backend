package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FinanceProfile stores user's financial patterns and behaviors
type FinanceProfile struct {
	ID                primitive.ObjectID     `bson:"_id,omitempty"`
	UserID            string                  `bson:"user_id"`
	SpendingPatterns  SpendingPatterns        `bson:"spending_patterns"`
	BudgetAdherence   BudgetAdherence         `bson:"budget_adherence"`
	MonthlyHistory    []MonthlySpending       `bson:"monthly_history"`
	LastUpdated       time.Time               `bson:"last_updated"`
}

// SpendingPatterns contains analysis of user's spending behavior
type SpendingPatterns struct {
	TopCategories     []CategoryPattern `bson:"top_categories"`
	MonthlyComparison string           `bson:"monthly_comparison"` // "+8%" or "-5%"
	RecurringDetected []string          `bson:"recurring_detected"`
	WeeklyAverage     float64          `bson:"weekly_average"`
}

// CategoryPattern represents spending pattern for a category
type CategoryPattern struct {
	Category    string  `bson:"category"`
	AvgWeekly   float64 `bson:"avg_weekly"`
	Trend       string  `bson:"trend"`      // "+15%", "-5%", "stable"
	Count       int     `bson:"count"`
	LastAmount  float64 `bson:"last_amount"`
}

// BudgetAdherence tracks how well user sticks to budgets
type BudgetAdherence struct {
	OverallStatus string           `bson:"overall_status"` // safe|caution|warning|exceeded
	Categories    []CategoryBudget `bson:"categories"`
}

// CategoryBudget tracks budget status per category
type CategoryBudget struct {
	Name    string  `bson:"name"`
	Limit   float64 `bson:"limit"`
	Spent   float64 `bson:"spent"`
	Percent int     `bson:"percent"`
	Status  string  `bson:"status"` // safe|caution|warning|exceeded
}

// MonthlySpending represents spending for a specific month
type MonthlySpending struct {
	Month   string  `bson:"month"`
	Total   float64 `bson:"total"`
	Income  float64 `bson:"income"`
	Expense float64 `bson:"expense"`
}

// FinanceProfileService manages user financial profiles
type FinanceProfileService struct {
	profileCollection *mongo.Collection
	txCollection     *mongo.Collection
	budgetCollection *mongo.Collection
	catBudgetCol     *mongo.Collection
}

// NewFinanceProfileService creates a new FinanceProfileService
func NewFinanceProfileService(client *mongo.Client, dbName string) *FinanceProfileService {
	db := client.Database(dbName)
	return &FinanceProfileService{
		profileCollection: db.Collection("finance_profiles"),
		txCollection:       db.Collection("Transaction"),
		budgetCollection:   db.Collection("monthly_budget"),
		catBudgetCol:       db.Collection("category_budget"),
	}
}

// GetOrCreateProfile retrieves existing profile or creates new one
func (s *FinanceProfileService) GetOrCreateProfile(userID string) (*FinanceProfile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var profile FinanceProfile
	err := s.profileCollection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&profile)
	if err == mongo.ErrNoDocuments {
		profile = FinanceProfile{
			ID:          primitive.NewObjectID(),
			UserID:      userID,
			LastUpdated: time.Now(),
		}
		_, err = s.profileCollection.InsertOne(ctx, profile)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return &profile, nil
}

// UpdateProfile recalculates and stores spending patterns
func (s *FinanceProfileService) UpdateProfile(userID string) (*FinanceProfile, error) {
	profile, err := s.GetOrCreateProfile(userID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get transactions from last 90 days
	since := time.Now().AddDate(0, 0, -90)
	cursor, err := s.txCollection.Find(ctx, bson.M{
		"user_id": userID,
		"date":    bson.M{"$gte": since},
	}, options.Find().SetSort(bson.D{{Key: "date", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var transactions []struct {
		Amount   float64   `bson:"amount"`
		Category string    `bson:"category"`
		Type     string    `bson:"type"`
		Date     time.Time `bson:"date"`
	}
	if err := cursor.All(ctx, &transactions); err != nil {
		return nil, err
	}

	// Calculate spending patterns
	profile.SpendingPatterns = s.calculateSpendingPatterns(transactions)

	// Calculate budget adherence
	profile.BudgetAdherence = s.calculateBudgetAdherence(ctx, userID)

	// Update monthly history
	profile.MonthlyHistory = s.calculateMonthlyHistory(transactions)

	profile.LastUpdated = time.Now()

	// Save updated profile
	_, err = s.profileCollection.UpdateOne(ctx,
		bson.M{"user_id": userID},
		bson.M{"$set": profile},
	)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

// calculateSpendingPatterns analyzes transaction data
func (s *FinanceProfileService) calculateSpendingPatterns(transactions []struct {
	Amount   float64   `bson:"amount"`
	Category string    `bson:"category"`
	Type     string    `bson:"type"`
	Date     time.Time `bson:"date"`
}) SpendingPatterns {

	// Group by category
	categoryTotals := make(map[string]float64)
	categoryCounts := make(map[string]int)
	categoryLastAmount := make(map[string]float64)

	var totalSpending float64
	var totalExpenses int

	for _, tx := range transactions {
		if tx.Type == "outcome" {
			categoryTotals[tx.Category] += tx.Amount
			categoryCounts[tx.Category]++
			categoryLastAmount[tx.Category] = tx.Amount
			totalSpending += tx.Amount
			totalExpenses++
		}
	}

	// Calculate weekly averages and trends
	var patterns []CategoryPattern
	for cat, total := range categoryTotals {
		weekCount := float64(len(transactions)) / 12.0 // 90 days ≈ 12 weeks
		avgWeekly := total / weekCount
		trend := s.calculateTrend(cat, transactions)
		patterns = append(patterns, CategoryPattern{
			Category:   cat,
			AvgWeekly:  math.Round(avgWeekly),
			Trend:      trend,
			Count:      categoryCounts[cat],
			LastAmount: categoryLastAmount[cat],
		})
	}

	// Sort by total spending (highest first)
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].AvgWeekly > patterns[j].AvgWeekly
	})

	// Keep top 5
	if len(patterns) > 5 {
		patterns = patterns[:5]
	}

	// Calculate monthly comparison
	monthlyComparison := s.calculateMonthlyComparison(transactions)

	// Detect recurring expenses
	recurring := s.detectRecurringExpenses(transactions)

	// Calculate overall weekly average
	weekCount := float64(len(transactions)) / 12.0
	weeklyAverage := totalSpending / weekCount

	return SpendingPatterns{
		TopCategories:     patterns,
		MonthlyComparison: monthlyComparison,
		RecurringDetected: recurring,
		WeeklyAverage:     math.Round(weeklyAverage),
	}
}

// calculateTrend determines spending trend for a category
func (s *FinanceProfileService) calculateTrend(category string, transactions []struct {
	Amount   float64   `bson:"amount"`
	Category string    `bson:"category"`
	Type     string    `bson:"type"`
	Date     time.Time `bson:"date"`
}) string {
	var recentTotal, olderTotal float64
	var recentCount, olderCount int
	cutoff := time.Now().AddDate(0, 0, -45) // Split at 45 days ago

	for _, tx := range transactions {
		if tx.Type == "outcome" && tx.Category == category {
			if tx.Date.After(cutoff) {
				recentTotal += tx.Amount
				recentCount++
			} else {
				olderTotal += tx.Amount
				olderCount++
			}
		}
	}

	if recentCount == 0 || olderCount == 0 {
		return "stable"
	}

	recentAvg := recentTotal / float64(recentCount)
	olderAvg := olderTotal / float64(olderCount)

	if olderAvg == 0 {
		return "stable"
	}

	change := ((recentAvg - olderAvg) / olderAvg) * 100
	if math.Abs(change) < 10 {
		return "stable"
	}

	sign := "+"
	if change < 0 {
		sign = ""
	}
	return fmt.Sprintf("%s%.0f%%", sign, change)
}

// calculateMonthlyComparison compares this month to last month
func (s *FinanceProfileService) calculateMonthlyComparison(transactions []struct {
	Amount   float64   `bson:"amount"`
	Category string    `bson:"category"`
	Type     string    `bson:"type"`
	Date     time.Time `bson:"date"`
}) string {
	now := time.Now()
	thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastMonth := thisMonth.AddDate(0, -1, 0)

	var thisMonthTotal, lastMonthTotal float64

	for _, tx := range transactions {
		if tx.Type == "outcome" {
			if tx.Date.After(thisMonth) || tx.Date.Equal(thisMonth) {
				thisMonthTotal += tx.Amount
			} else if tx.Date.After(lastMonth) && tx.Date.Before(thisMonth) {
				lastMonthTotal += tx.Amount
			}
		}
	}

	if lastMonthTotal == 0 {
		return "stable"
	}

	change := ((thisMonthTotal - lastMonthTotal) / lastMonthTotal) * 100
	if math.Abs(change) < 5 {
		return "stable"
	}

	sign := "+"
	if change < 0 {
		sign = ""
	}
	return fmt.Sprintf("%s%.0f%%", sign, change)
}

// detectRecurringExpenses finds subscriptions and regular expenses
func (s *FinanceProfileService) detectRecurringExpenses(transactions []struct {
	Amount   float64   `bson:"amount"`
	Category string    `bson:"category"`
	Type     string    `bson:"type"`
	Date     time.Time `bson:"date"`
}) []string {
	// Group by similar amounts (within 10% tolerance)
	amountGroups := make(map[float64][]time.Time)

	for _, tx := range transactions {
		if tx.Type == "outcome" {
			// Round to nearest 1000 for grouping
			rounded := math.Round(tx.Amount/1000) * 1000
			amountGroups[rounded] = append(amountGroups[rounded], tx.Date)
		}
	}

	var recurring []string
	for amount, dates := range amountGroups {
		if len(dates) >= 2 {
			// Check if dates are roughly monthly (25-35 days apart)
			if len(dates) >= 2 {
				sort.Slice(dates, func(i, j int) bool {
					return dates[i].Before(dates[j])
				})
				first := dates[0]
				last := dates[len(dates)-1]
				daysBetween := last.Sub(first).Hours() / 24
				

				if len(dates) >= 2 {
					interval := daysBetween / float64(len(dates)-1)
					if interval >= 25 && interval <= 35 {
						recurring = append(recurring, fmt.Sprintf("Rp%.0f (monthly)", amount))
					}
				}
			}
		}
	}

	// Keep top 3
	if len(recurring) > 3 {
		recurring = recurring[:3]
	}

	return recurring
}

// calculateBudgetAdherence gets budget status per category
func (s *FinanceProfileService) calculateBudgetAdherence(ctx context.Context, userID string) BudgetAdherence {
	month := time.Now().Format("2006-01")

	// Get category budgets
	cursor, err := s.catBudgetCol.Find(ctx, bson.M{
		"user_id": userID,
		"month":   month,
	})
	if err != nil {
		return BudgetAdherence{OverallStatus: "safe"}
	}
	defer cursor.Close(ctx)

	var catBudgets []struct {
		Category string  `bson:"category"`
		Limit    float64 `bson:"limit"`
		Spent    float64 `bson:"spent"`
	}
	if err := cursor.All(ctx, &catBudgets); err != nil {
		return BudgetAdherence{OverallStatus: "safe"}
	}

	var categories []CategoryBudget
	worstStatus := 0 // 0=safe, 1=caution, 2=warning, 3=exceeded

	for _, cb := range catBudgets {
		percent := 0
		if cb.Limit > 0 {
			percent = int((cb.Spent / cb.Limit) * 100)
		}

		status := "safe"
		if percent >= 100 {
			status = "exceeded"
			worstStatus = max(worstStatus, 3)
		} else if percent >= 90 {
			status = "warning"
			worstStatus = max(worstStatus, 2)
		} else if percent >= 75 {
			status = "caution"
			worstStatus = max(worstStatus, 1)
		}

		categories = append(categories, CategoryBudget{
			Name:    cb.Category,
			Limit:   cb.Limit,
			Spent:   cb.Spent,
			Percent: percent,
			Status:  status,
		})
	}

	overallStatus := "safe"
	switch worstStatus {
	case 1:
		overallStatus = "caution"
	case 2:
		overallStatus = "warning"
	case 3:
		overallStatus = "exceeded"
	}

	return BudgetAdherence{
		OverallStatus: overallStatus,
		Categories:    categories,
	}
}

// calculateMonthlyHistory builds monthly spending history
func (s *FinanceProfileService) calculateMonthlyHistory(transactions []struct {
	Amount   float64   `bson:"amount"`
	Category string    `bson:"category"`
	Type     string    `bson:"type"`
	Date     time.Time `bson:"date"`
}) []MonthlySpending {
	monthly := make(map[string]MonthlySpending)

	for _, tx := range transactions {
		monthKey := tx.Date.Format("2006-01")
		entry := monthly[monthKey]
		entry.Month = monthKey

		if tx.Type == "income" {
			entry.Income += tx.Amount
		} else {
			entry.Expense += tx.Amount
		}
		entry.Total += tx.Amount
		monthly[monthKey] = entry
	}

	// Convert to slice and sort by month
	var history []MonthlySpending
	for _, m := range monthly {
		history = append(history, m)
	}
	sort.Slice(history, func(i, j int) bool {
		return history[i].Month < history[j].Month
	})

	// Keep last 6 months
	if len(history) > 6 {
		history = history[len(history)-6:]
	}

	return history
}

// GetProfileContext returns enriched context for chatbot
func (s *FinanceProfileService) GetProfileContext(userID string) map[string]interface{} {
	profile, err := s.UpdateProfile(userID)
	if err != nil {
		return map[string]interface{}{}
	}

	context := map[string]interface{}{}

	// Spending patterns
	if len(profile.SpendingPatterns.TopCategories) > 0 {
		var topCats []map[string]interface{}
		for _, p := range profile.SpendingPatterns.TopCategories {
			topCats = append(topCats, map[string]interface{}{
				"category":    p.Category,
				"avg_weekly":  fmt.Sprintf("Rp%.0f", p.AvgWeekly),
				"trend":       p.Trend,
				"count":       p.Count,
				"last_amount": fmt.Sprintf("Rp%.0f", p.LastAmount),
			})
		}
		context["spending_patterns"] = map[string]interface{}{
			"top_categories":      topCats,
			"monthly_comparison":  profile.SpendingPatterns.MonthlyComparison,
			"recurring_detected":  profile.SpendingPatterns.RecurringDetected,
			"weekly_average":      fmt.Sprintf("Rp%.0f", profile.SpendingPatterns.WeeklyAverage),
		}
	}

	// Budget adherence
	if len(profile.BudgetAdherence.Categories) > 0 {
		var cats []map[string]interface{}
		for _, c := range profile.BudgetAdherence.Categories {
			cats = append(cats, map[string]interface{}{
				"name":    c.Name,
				"limit":   fmt.Sprintf("Rp%.0f", c.Limit),
				"spent":   fmt.Sprintf("Rp%.0f", c.Spent),
				"percent": c.Percent,
				"status":  c.Status,
			})
		}
		context["budget_adherence"] = map[string]interface{}{
			"overall_status": profile.BudgetAdherence.OverallStatus,
			"categories":     cats,
		}
	}

	return context
}
