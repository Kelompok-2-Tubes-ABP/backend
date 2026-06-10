package chatbot_commands

import (
	"fmt"
	"strings"
	"time"

	"financeapi/essentials/models"
	"financeapi/essentials/services"
	"financeapi/essentials/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	// Check for delete keyword
	deleteKeywords := []string{"hapus", "delete", "remove", "batal"}
	if utils.ContainsAny(msgLower, deleteKeywords) {
		return c.handleDeleteBudget(userID, message)
	}

	// Check for edit keyword
	editKeywords := []string{"edit", "ubah", "update", "ganti", "change"}
	if utils.ContainsAny(msgLower, editKeywords) {
		return c.handleEditBudget(userID, message)
	}

	// Check for create keyword
	createKeywords := []string{"buat", "create", "tambah", "add", "new"}
	if utils.ContainsAny(msgLower, createKeywords) {
		return c.handleCreateBudget(userID, message)
	}

	month := time.Now().Format("2006-01")

	// Determine if user is asking for category budget specifically
	isCategoryQuery := utils.ContainsAny(msgLower, []string{"kategori", "category", "per group", "per bagian"})

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
			// Access nested budget object
			if budgetObj, ok := cb["budget"].(models.CategoryBudget); ok {
				limit := budgetObj.Limit
				spent := budgetObj.Spent
				catName := budgetObj.Category
				percent := 0.0
				if limit > 0 {
					percent = (spent / limit) * 100
				}

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

func (c *BudgetCommand) handleCreateBudget(userID, message string) string {
	msg := strings.ToLower(message)
	amount := utils.ParseIndonesianAmount(message)

	if amount <= 0 {
		return "Maaf, saya tidak dapat menentukan jumlah budget. Contoh: 'buat budget 5 juta'"
	}

	// Determine if category budget or monthly budget
	isCategory := utils.ContainsAny(msg, []string{"kategori", "category", "makanan", "transport", "belanja", "hiburan"})
	category := utils.ExtractExpenseCategory(msg)

	if isCategory || category != "" {
		// Create category budget
		if category == "" {
			category = "other"
		}
		month := time.Now().Format("2006-01")
		budget := models.CategoryBudget{
			UserID:    userID,
			Category:  category,
			Month:     month,
			Limit:     amount,
			Spent:     0,
			CreatedAt: time.Now(),
		}

		created, err := c.budgetService.CreateCategoryBudget(budget)
		if err != nil {
			return fmt.Sprintf("❌ Gagal membuat budget kategori: %v", err)
		}

		return fmt.Sprintf("✅ **Budget Kategori Berhasil Dibuat!**\n\n📂 Kategori: %s\n💰 Limit: Rp%.0f\n📅 Bulan: %s", created.Category, created.Limit, created.Month)
	}

	// Create monthly budget
	month := time.Now().Format("2006-01")
	budget := models.MonthlyBudget{
		UserID:    userID,
		Month:     month,
		Limit:     amount,
		CreatedAt: time.Now(),
	}

	created, err := c.budgetService.CreateBudget(budget)
	if err != nil {
		return fmt.Sprintf("❌ Gagal membuat budget: %v", err)
	}

	return fmt.Sprintf("✅ **Budget Bulanan Berhasil Dibuat!**\n\n💰 Total Limit: Rp%.0f\n📅 Bulan: %s", created.Limit, created.Month)
}

func (c *BudgetCommand) handleEditBudget(userID, message string) string {
	msg := strings.ToLower(message)
	newAmount := utils.ParseIndonesianAmount(message)

	if newAmount <= 0 {
		return "Maaf, saya tidak dapat menentukan jumlah budget. Contoh: 'edit budget menjadi 6 juta'"
	}

	month := time.Now().Format("2006-01")

	// Try to find budget to edit
	budgets, err := c.budgetService.GetAllBudgetsWithSpending(userID)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	// Find current month budget
	var targetBudget map[string]interface{}
	for _, b := range budgets {
		if b["month"] == month {
			targetBudget = b
			break
		}
	}

	// Check if it's a category budget
	category := utils.ExtractExpenseCategory(msg)
	if category != "" {
		catBudgets, _ := c.budgetService.GetAllCategoryBudgetsWithSpending(userID, month)
		for _, cb := range catBudgets {
			// Access nested budget object
			if budgetObj, ok := cb["budget"].(models.CategoryBudget); ok {
				if budgetObj.Category == category {
					_, err = c.budgetService.UpdateCategoryBudget(budgetObj.ID, userID, bson.M{"limit": newAmount})
					if err != nil {
						return fmt.Sprintf("❌ Gagal edit budget kategori: %v", err)
					}
					return fmt.Sprintf("✅ Budget kategori '%s' berhasil diupdate ke Rp%.0f", category, newAmount)
				}
			}
		}
	}

	if targetBudget == nil {
		return "Tidak ada budget untuk diedit bulan ini."
	}

	budgetID, ok := targetBudget["_id"].(primitive.ObjectID)
	if !ok {
		return "Tidak dapat menemukan ID budget."
	}

	_, err = c.budgetService.UpdateBudget(budgetID, userID, bson.M{"limit": newAmount})
	if err != nil {
		return fmt.Sprintf("❌ Gagal edit budget: %v", err)
	}

	return fmt.Sprintf("✅ Budget berhasil diupdate ke Rp%.0f", newAmount)
}

func (c *BudgetCommand) handleDeleteBudget(userID, message string) string {
	msg := strings.ToLower(message)
	month := time.Now().Format("2006-01")

	// Check if it's a category budget
	category := utils.ExtractExpenseCategory(msg)
	if category != "" {
		catBudgets, _ := c.budgetService.GetAllCategoryBudgetsWithSpending(userID, month)
		for _, cb := range catBudgets {
			// Access nested budget object
			if budgetObj, ok := cb["budget"].(models.CategoryBudget); ok {
				if budgetObj.Category == category {
					err := c.budgetService.DeleteCategoryBudget(budgetObj.ID, userID)
					if err != nil {
						return fmt.Sprintf("❌ Gagal hapus budget kategori: %v", err)
					}
					return fmt.Sprintf("🗑️ Budget kategori '%s' berhasil dihapus", category)
				}
			}
		}
		return fmt.Sprintf("Budget kategori '%s' tidak ditemukan", category)
	}

	// Delete monthly budget
	budgets, err := c.budgetService.GetAllBudgetsWithSpending(userID)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	var targetBudget map[string]interface{}
	for _, b := range budgets {
		if b["month"] == month {
			targetBudget = b
			break
		}
	}

	if targetBudget == nil {
		return "Tidak ada budget untuk dihapus bulan ini."
	}

	budgetID, ok := targetBudget["_id"].(primitive.ObjectID)
	if !ok {
		return "Tidak dapat menemukan ID budget."
	}

	err = c.budgetService.DeleteBudget(budgetID, userID)
	if err != nil {
		return fmt.Sprintf("❌ Gagal hapus budget: %v", err)
	}

	return "🗑️ Budget berhasil dihapus"
}
