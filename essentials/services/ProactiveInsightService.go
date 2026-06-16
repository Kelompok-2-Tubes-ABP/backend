package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// ProactiveInsightService generates proactive insights for users
type ProactiveInsightService struct {
	txCollection     *mongo.Collection
	budgetCollection *mongo.Collection
	catBudgetCol     *mongo.Collection
	billCollection   *mongo.Collection
	savingsCol       *mongo.Collection
}

// NewProactiveInsightService creates a new ProactiveInsightService
func NewProactiveInsightService(client *mongo.Client, dbName string) *ProactiveInsightService {
	db := client.Database(dbName)
	return &ProactiveInsightService{
		txCollection:     db.Collection("Transaction"),
		budgetCollection: db.Collection("monthly_budget"),
		catBudgetCol:     db.Collection("category_budget"),
		billCollection:   db.Collection("bill_reminders"),
		savingsCol:       db.Collection("savings_goals"),
	}
}

// Insight represents a single proactive insight
type Insight struct {
	Type    string  // "budget_pace", "unusual_spending", "trend_alert", "savings_progress", "bill_reminder"
	Title   string  `json:"title"`
	Message string  `json:"message"`
	Priority int    `json:"priority"` // 1=high, 2=medium, 3=low
}

// GenerateInsights creates proactive insights for a user
func (s *ProactiveInsightService) GenerateInsights(userID string) []string {
	var insights []string

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Budget pace insights
	budgetInsights := s.analyzeBudgetPace(ctx, userID)
	insights = append(insights, budgetInsights...)

	// 2. Unusual spending detection
	unusualInsights := s.detectUnusualSpending(ctx, userID)
	insights = append(insights, unusualInsights...)

	// 3. Spending trend insights
	trendInsights := s.analyzeSpendingTrends(ctx, userID)
	insights = append(insights, trendInsights...)

	// 4. Savings progress insights
	savingsInsights := s.analyzeSavingsProgress(ctx, userID)
	insights = append(insights, savingsInsights...)

	// 5. Bill reminder insights
	billInsights := s.analyzeUpcomingBills(ctx, userID)
	insights = append(insights, billInsights...)

	// Limit to top 5 insights
	if len(insights) > 5 {
		insights = insights[:5]
	}

	return insights
}

// analyzeBudgetPace checks if user is on track with their budget
func (s *ProactiveInsightService) analyzeBudgetPace(ctx context.Context, userID string) []string {
	var insights []string

	month := time.Now().Format("2006-01")
	now := time.Now()
	dayOfMonth := now.Day()
	daysInMonth := 30 // Approximate
	monthProgress := float64(dayOfMonth) / float64(daysInMonth) * 100

	// Get category budgets
	cursor, err := s.catBudgetCol.Find(ctx, bson.M{
		"user_id": userID,
		"month":   month,
	})
	if err != nil {
		return insights
	}
	defer cursor.Close(ctx)

	var budgets []struct {
		Category string  `bson:"category"`
		Limit    float64 `bson:"limit"`
		Spent    float64 `bson:"spent"`
	}
	if err := cursor.All(ctx, &budgets); err != nil {
		return insights
	}

	for _, b := range budgets {
		if b.Limit <= 0 {
			continue
		}

		percentUsed := (b.Spent / b.Limit) * 100
		expectedPercent := monthProgress

		// Check if overspending early in month
		if percentUsed > 100 {
			daysLeft := daysInMonth - dayOfMonth
			remainingBudget := b.Limit - b.Spent
			if remainingBudget < 0 {
				insights = append(insights, fmt.Sprintf(
					"⚠️ Peringatan: Kamu sudah melebihi budget %s sebesar Rp%.0f. Sisa hari: %d hari",
					b.Category, math.Abs(remainingBudget), daysLeft))
			}
		} else if percentUsed > expectedPercent+20 {
			// Spending faster than expected
			daysLeft := daysInMonth - dayOfMonth
			projectedOverspend := (percentUsed - expectedPercent) / 100 * b.Limit
			insights = append(insights, fmt.Sprintf(
				"📊 Perhatian: Pengeluaran %s (%d%%) sudah melewati ekspektasi (%d%%). "+
					"Potensi melebihi budget Rp%.0f dalam %d hari ke depan",
				b.Category, int(percentUsed), int(expectedPercent), projectedOverspend, daysLeft))
		} else if percentUsed < expectedPercent-15 && percentUsed < 60 {
			// Spending less than expected - positive insight
			insights = append(insights, fmt.Sprintf(
				"🎉 Selamat! Pengeluaran %s (%d%%) masih di bawah ekspektasi (%d%%). "+
					"Pertahankan!",
				b.Category, int(percentUsed), int(expectedPercent)))
		}
	}

	return insights
}

// detectUnusualSpending finds unusually large or small transactions
func (s *ProactiveInsightService) detectUnusualSpending(ctx context.Context, userID string) []string {
	var insights []string

	// Get last 30 days of transactions
	since := time.Now().AddDate(0, 0, -30)
	cursor, err := s.txCollection.Find(ctx, bson.M{
		"user_id": userID,
		"date":    bson.M{"$gte": since},
		"type":    "outcome",
	}, nil)
	if err != nil {
		return insights
	}
	defer cursor.Close(ctx)

	var transactions []struct {
		Amount   float64   `bson:"amount"`
		Category string    `bson:"category"`
		Date     time.Time `bson:"date"`
	}
	if err := cursor.All(ctx, &transactions); err != nil {
		return insights
	}

	if len(transactions) < 5 {
		return insights
	}

	// Calculate category averages
	categoryTotals := make(map[string]float64)
	categoryCounts := make(map[string]int)
	for _, tx := range transactions {
		categoryTotals[tx.Category] += tx.Amount
		categoryCounts[tx.Category]++
	}

	// Check recent transactions (last 7 days) for unusual amounts
	recentSince := time.Now().AddDate(0, 0, -7)
	for _, tx := range transactions {
		if tx.Date.After(recentSince) {
			avg := categoryTotals[tx.Category] / float64(categoryCounts[tx.Category])
			if tx.Amount > avg*2 && tx.Amount > 100000 {
				// Unusually large purchase
				percentAbove := int(((tx.Amount - avg) / avg) * 100)
				insights = append(insights, fmt.Sprintf(
					"💰 Pengeluaran tidak biasa: Rp%.0f untuk %s (%d%% di atas rata-rata Rp%.0f). Apakah ini sesuai rencana?",
					tx.Amount, tx.Category, percentAbove, avg))
			}
		}
	}

	return insights
}

// analyzeSpendingTrends compares spending patterns over time
func (s *ProactiveInsightService) analyzeSpendingTrends(ctx context.Context, userID string) []string {
	var insights []string

	// Get this month and last month transactions
	now := time.Now()
	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastMonthStart := thisMonthStart.AddDate(0, -1, 0)

	// This month
	thisMonthTxs, _ := s.txCollection.Find(ctx, bson.M{
		"user_id": userID,
		"date":    bson.M{"$gte": thisMonthStart},
		"type":    "outcome",
	}, nil)

	// Last month
	lastMonthTxs, _ := s.txCollection.Find(ctx, bson.M{
		"user_id": userID,
		"date":    bson.M{"$gte": lastMonthStart, "$lt": thisMonthStart},
		"type":    "outcome",
	}, nil)

	var thisMonthTotal, lastMonthTotal float64

	var txs []struct{ Amount float64 `bson:"amount"` }
	if thisMonthTxs != nil {
		thisMonthTxs.All(ctx, &txs)
		for _, tx := range txs {
			thisMonthTotal += tx.Amount
		}
		thisMonthTxs.Close(ctx)
	}

	txs = nil
	if lastMonthTxs != nil {
		lastMonthTxs.All(ctx, &txs)
		for _, tx := range txs {
			lastMonthTotal += tx.Amount
		}
		lastMonthTxs.Close(ctx)
	}

	if lastMonthTotal > 0 {
		change := ((thisMonthTotal - lastMonthTotal) / lastMonthTotal) * 100
		if math.Abs(change) > 15 {
			if change > 0 {
				insights = append(insights, fmt.Sprintf(
					"📈 Tren: Pengeluaran bulan ini %d%% lebih tinggi dari bulan lalu. "+
						"Bulan lalu: Rp%.0f, Bulan ini: Rp%.0f",
					int(change), lastMonthTotal, thisMonthTotal))
			} else {
				insights = append(insights, fmt.Sprintf(
					"📉 Kabar baik: Pengeluaran bulan ini %d%% lebih hemat dari bulan lalu! "+
						"Bulan lalu: Rp%.0f, Bulan ini: Rp%.0f",
					int(math.Abs(change)), lastMonthTotal, thisMonthTotal))
			}
		}
	}

	return insights
}

// analyzeSavingsProgress checks savings goal progress
func (s *ProactiveInsightService) analyzeSavingsProgress(ctx context.Context, userID string) []string {
	var insights []string

	cursor, err := s.savingsCol.Find(ctx, bson.M{
		"user_id": userID,
	})
	if err != nil {
		return insights
	}
	defer cursor.Close(ctx)

	var goals []struct {
		Name         string  `bson:"name"`
		TargetAmount float64 `bson:"target_amount"`
		CurrentAmount float64 `bson:"current_amount"`
	}
	if err := cursor.All(ctx, &goals); err != nil {
		return insights
	}

	for _, g := range goals {
		if g.TargetAmount <= 0 {
			continue
		}

		progress := (g.CurrentAmount / g.TargetAmount) * 100

		if progress >= 100 {
			insights = append(insights, fmt.Sprintf(
				"🎊 Selamat! Target tabungan \"%s\" sudah tercapai! "+
					"Kamu berhasil menabung Rp%.0f dari target Rp%.0f",
				g.Name, g.CurrentAmount, g.TargetAmount))
		} else if progress >= 75 {
			insights = append(insights, fmt.Sprintf(
				"🚀 Hampir selesai! Tabungan \"%s\" sudah %d%% complete "+
					"(Rp%.0f dari Rp%.0f). Hampir there!",
				g.Name, int(progress), g.CurrentAmount, g.TargetAmount))
		} else if progress >= 50 {
			insights = append(insights, fmt.Sprintf(
				"💪 Bagus! Tabungan \"%s\" sudah %d%% jalan. "+
					"Rp%.0f lagi untuk mencapai target Rp%.0f",
				g.Name, int(progress), g.TargetAmount-g.CurrentAmount, g.TargetAmount))
		}
	}

	return insights
}

// analyzeUpcomingBills checks for upcoming bill payments
func (s *ProactiveInsightService) analyzeUpcomingBills(ctx context.Context, userID string) []string {
	var insights []string

	userOID, _ := primitive.ObjectIDFromHex(userID)
	now := time.Now()
	threeDays := now.AddDate(0, 0, 3)

	cursor, err := s.billCollection.Find(ctx, bson.M{
		"user_id":    userOID,
		"is_paid":    false,
		"next_due_date": bson.M{"$lte": threeDays, "$gte": now},
	})
	if err != nil {
		return insights
	}
	defer cursor.Close(ctx)

	var bills []struct {
		Name        string    `bson:"name"`
		Amount      float64   `bson:"amount"`
		NextDueDate time.Time `bson:"next_due_date"`
	}
	if err := cursor.All(ctx, &bills); err != nil {
		return insights
	}

	for _, b := range bills {
		daysUntil := int(b.NextDueDate.Sub(now).Hours() / 24)
		if daysUntil <= 1 {
			insights = append(insights, fmt.Sprintf(
				"⏰ Segera: Tagihan \"%s\" Rp%.0f jatuh tempo %s!",
				b.Name, b.Amount, b.NextDueDate.Format("02 Jan 2006")))
		} else {
			insights = append(insights, fmt.Sprintf(
				"📅 Pengingat: Tagihan \"%s\" Rp%.0f jatuh tempo dalam %d hari (%s)",
				b.Name, b.Amount, daysUntil, b.NextDueDate.Format("02 Jan 2006")))
		}
	}

	return insights
}

// GetInsightsForChatbot returns formatted insights for chatbot context
func (s *ProactiveInsightService) GetInsightsForChatbot(userID string) []string {
	return s.GenerateInsights(userID)
}
