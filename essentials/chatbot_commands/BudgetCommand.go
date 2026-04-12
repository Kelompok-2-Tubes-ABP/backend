package chatbot_commands

import (
	"fmt"
	"strings"
	"time"

	"financeapi/essentials/services"
)

type BudgetCommand struct {
	budgetService *services.BudgetService
}

func NewBudgetCommand(budgetService *services.BudgetService) *BudgetCommand {
	return &BudgetCommand{
		budgetService: budgetService,
	}
}

func (c *BudgetCommand) Handle(userID string, message string) string {
	msg := strings.ToLower(message)
	return c.handleBudget(userID, msg, message)
}

func (c *BudgetCommand) handleBudget(userID, msgLower, message string) string {
	if c.budgetService == nil {
		return "Layanan anggaran belum siap. Hubungi admin."
	}

	month := time.Now().Format("2006-01")

	// Determine if user is asking for category budget specifically
	isCategoryQuery := containsAny(msgLower, []string{"kategori", "category", "per group", "per bagian"})

	summary := fmt.Sprintf("📊 **Analisa Anggaran Kamu (%s)**\n\n", month)

	// 1. Get Monthly Budget Summary
	budgets, err := c.budgetService.GetAllBudgetsWithSpending(userID)
	if err != nil || len(budgets) == 0 {
		return "Kamu belum buat budget nih bulan ini. Mau dibantu buat anggaran pertama?"
	}

	// Find current month's budget
	var currentBudget map[string]interface{}
	for _, b := range budgets {
		if b["month"] == month {
			currentBudget = b
			break
		}
	}

	if currentBudget != nil {
		limit := currentBudget["limit"].(float64)
		spent := currentBudget["spent"].(float64)
		remaining := limit - spent
		percent := (spent / limit) * 100

		statusLabel := "✅ Aman"
		if percent >= 100 {
			statusColor := "🔴"
			statusLabel = "BAHAYA (Over-budget!)"
			summary += fmt.Sprintf("%s **Status: %s**\n", statusColor, statusLabel)
		} else if percent >= 80 {
			statusColor := "🟡"
			statusLabel = "Waspada (Sudah jalan 80%+)"
			summary += fmt.Sprintf("%s **Status: %s**\n", statusColor, statusLabel)
		} else {
			summary += fmt.Sprintf("✅ **Status: %s**\n", statusLabel)
		}

		summary += fmt.Sprintf("💰 Total Limit: Rp%.0f\n", limit)
		summary += fmt.Sprintf("💸 Sudah Terpakai: Rp%.0f (%.1f%%)\n", spent, percent)
		summary += fmt.Sprintf("📥 Sisa Saldo Budget: Rp%.0f\n\n", remaining)
	}

	// 2. Get Detailed Category Budgets if requested or if total is high
	catBudgets, err := c.budgetService.GetAllCategoryBudgetsWithSpending(userID, month)
	if err == nil && len(catBudgets) > 0 {
		summary += "📂 **Detail per Kategori:**\n"
		count := 0
		for _, cb := range catBudgets {
			// Only show current month
			if cb["month"] == month {
				limit := cb["budget_amount"].(float64)
				spent := cb["spent"].(float64)
				catName := cb["category_name"].(string)
				percent := (spent / limit) * 100

				emoji := "🔹"
				if percent >= 100 {
					emoji = "❌"
				} else if percent >= 80 {
					emoji = "⚠️"
				}

				summary += fmt.Sprintf("%s %s: Rp%.0f / Rp%.0f (%.0f%%)\n",
					emoji, catName, spent, limit, percent)
				count++
			}
		}
		if count == 0 {
			summary += "_Belum ada budget kategori yang dibuat._\n"
		}
	}

	if !isCategoryQuery {
		summary += "\n💡 *Tips: Kamu bisa tanya \"Budget kategori\" untuk melihat detail per pos pengeluaran.*"
	}

	return summary
}
