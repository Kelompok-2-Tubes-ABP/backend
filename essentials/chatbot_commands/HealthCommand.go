package chatbot_commands

import (
	"fmt"

	"financeapi/essentials/services"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type HealthCommand struct {
	insightService *services.SpendingInsightService
	txService      *services.TransactionService
}

func NewHealthCommand(insightService *services.SpendingInsightService, txService *services.TransactionService) *HealthCommand {
	return &HealthCommand{
		insightService: insightService,
		txService:      txService,
	}
}

func (c *HealthCommand) Handle(userID string, message string) string {
	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return "Error: Invalid user ID"
	}

	summary := "🩺 **Laporan Kesehatan Keuangan Kamu**\n\n"

	// 1. Get Real Health Score
	if c.insightService != nil {
		healthScore, err := c.insightService.GetFinancialHealthScore(userOID)
		if err == nil {
			statusColor := "🟢" // Sehat
			if healthScore.OverallScore < 50 {
				statusColor = "🔴" // Bahaya
			} else if healthScore.OverallScore < 75 {
				statusColor = "🟡" // Waspada
			}

			summary += fmt.Sprintf("%s **Skor Keseluruhan: %d/100**\n", statusColor, healthScore.OverallScore)
			summary += fmt.Sprintf("📊 Tabungan: %d%% | Hutang: %d%% | Kontrol: %d%%\n\n",
				healthScore.SavingsRate, healthScore.DebtLevel, healthScore.ExpenseControl)

			if len(healthScore.Recommendations) > 0 {
				summary += "**Saran Utama:**\n"
				for i, rec := range healthScore.Recommendations {
					if i >= 3 {
						break
					}
					summary += fmt.Sprintf("💡 %s\n", rec)
				}
				summary += "\n"
			}
		}
	}

	// 2. Add Transaction Stats
	transactions, _ := c.txService.ShowTransaction(userID)
	var totalIncome, totalExpense float64
	for _, tx := range transactions {
		if tx.Category == "income" || tx.Category == "pemasukan" {
			totalIncome += tx.Amount
		} else {
			totalExpense += tx.Amount
		}
	}
	summary += fmt.Sprintf("💰 **Ringkasan Bulan Ini:**\n- Pemasukan: Rp%.0f\n- Pengeluaran: Rp%.0f\n- Saldo: Rp%.0f\n\n",
		totalIncome, totalExpense, totalIncome-totalExpense)

	// 3. Show Recent Unread Insights & Mark as Read
	if c.insightService != nil {
		insights, err := c.insightService.GetUserInsights(userOID, 3)
		if err == nil && len(insights) > 0 {
			hasInsights := false
			for _, insight := range insights {
				if !insight.IsRead {
					if !hasInsights {
						summary += "🔔 **Insight Terbaru:**\n"
						hasInsights = true
					}
					summary += fmt.Sprintf("- **%s**: %s\n", insight.Title, insight.Description)
					go c.insightService.MarkAsRead(insight.ID, userOID)
				}
			}
		}
	}

	return summary
}
