package chatbot_commands

import (
	"fmt"
	"strings"

	"financeapi/essentials/models"
	"financeapi/essentials/services"
	"financeapi/essentials/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SavingsCommand struct {
	savingsGoalService *services.SavingsGoalService
}

func NewSavingsCommand(savingsGoalService *services.SavingsGoalService) *SavingsCommand {
	return &SavingsCommand{
		savingsGoalService: savingsGoalService,
	}
}

func (c *SavingsCommand) Handle(userID string, message string) string {
	msg := strings.ToLower(message)
	return c.handleSavings(userID, msg, message)
}

func (c *SavingsCommand) handleSavings(userID, msgLower, message string) string {
	if containsAny(msgLower, []string{"ada", "exist", "cek", "lihat", "status", "progress", "how", "apa"}) {
		goals, err := c.savingsGoalService.GetUserSavingsGoals(userID)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}

		if len(goals) == 0 {
			return "Belum ada tabungan yang kamu buat."
		}

		for _, goal := range goals {
			goalNameLower := strings.ToLower(goal.Name)
			if strings.Contains(goalNameLower, "milan") && strings.Contains(msgLower, "milan") {
				progress := (goal.CurrentAmount / goal.TargetAmount) * 100
				return fmt.Sprintf("✅ Ya! Kamu punya target '%s':\n\n💰 Terkumpul: Rp%.0f\n🎯 Target: Rp%.0f\n📊 Progress: %.0f%%",
					goal.Name, goal.CurrentAmount, goal.TargetAmount, progress)
			}
			if strings.Contains(goalNameLower, "jepang") && strings.Contains(msgLower, "jepang") {
				progress := (goal.CurrentAmount / goal.TargetAmount) * 100
				return fmt.Sprintf("✅ Ya! Kamu punya target '%s':\n\n💰 Terkumpul: Rp%.0f\n🎯 Target: Rp%.0f\n📊 Progress: %.0f%%",
					goal.Name, goal.CurrentAmount, goal.TargetAmount, progress)
			}
		}

		summary := "🎯 Semua Target Tabungan:\n\n"
		for _, goal := range goals {
			progress := (goal.CurrentAmount / goal.TargetAmount) * 100
			summary += fmt.Sprintf("%s: Rp%.0f / Rp%.0f (%.0f%%)\n", goal.Name, goal.CurrentAmount, goal.TargetAmount, progress)
		}
		return summary
	}

	if containsAny(msgLower, []string{"tambah", "add", "setor", "nabung", "simpan", "saved", "menabung"}) {
		return c.handleAddSavingsContribution(userID, message)
	}

	if containsAny(msgLower, []string{"buat", "create", "target baru", "goal baru"}) {
		return "Untuk membuat tabungan baru, sebutkan: nama tabungan dan target jumlah. Contoh: 'buat tabungan mobil 10 juta'"
	}

	goals, err := c.savingsGoalService.GetUserSavingsGoals(userID)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if len(goals) == 0 {
		return "Belum ada tabungan. Mau buat target tabungan?"
	}

	summary := "🎯 Target Tabungan:\n\n"
	for _, goal := range goals {
		progress := (goal.CurrentAmount / goal.TargetAmount) * 100
		summary += fmt.Sprintf("%s: Rp%.0f / Rp%.0f (%.0f%%)\n", goal.Name, goal.CurrentAmount, goal.TargetAmount, progress)
	}
	return summary
}

func (c *SavingsCommand) handleAddSavingsContribution(userID, message string) string {
	amount := utils.ParseIndonesianAmount(message)
	if amount <= 0 {
		return "Maaf, saya tidak dapat menentukan jumlah yang ingin ditabung. Contoh: 'tambah tabungan jepang 1 juta'"
	}

	goalName := utils.ExtractSavingsGoalName(message)
	msgLower := strings.ToLower(message)

	goals, err := c.savingsGoalService.GetUserSavingsGoals(userID)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if len(goals) == 0 {
		return "Belum ada tabungan yang kamu buat. Mau buat target tabungan baru?"
	}

	var targetGoal models.SavingsGoal
	found := false

	if goalName != "" {
		for _, goal := range goals {
			goalNameLower := strings.ToLower(goal.Name)
			if strings.Contains(goalNameLower, goalName) || strings.Contains(goalName, strings.Split(goalNameLower, " ")[0]) {
				targetGoal = goal
				found = true
				break
			}
		}
	}

	if !found {
		for _, goal := range goals {
			goalNameLower := strings.ToLower(goal.Name)
			if strings.Contains(msgLower, "milan") && strings.Contains(goalNameLower, "milan") {
				targetGoal = goal
				found = true
				break
			}
			if strings.Contains(msgLower, "jepang") && strings.Contains(goalNameLower, "jepang") {
				targetGoal = goal
				found = true
				break
			}
			if strings.Contains(msgLower, "liburan") && strings.Contains(goalNameLower, "liburan") {
				targetGoal = goal
				found = true
				break
			}
		}
	}

	if !found {
		return fmt.Sprintf("Tabungan '%s' tidak ditemukan. Berikut tabungan kamu:\n%s", goalName, c.listSavingsGoals(goals))
	}

	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return "Error: Invalid user ID"
	}

	contribution := models.SavingsContribution{
		SavingsGoalID: targetGoal.ID,
		UserID:        userOID,
		Amount:        amount,
		Note:          "Added via chatbot",
	}

	_, err = c.savingsGoalService.AddContribution(contribution)
	if err != nil {
		return fmt.Sprintf("Error menambahkan tabungan: %v", err)
	}

	newAmount := targetGoal.CurrentAmount + amount
	var newProgress float64 = 0
	if targetGoal.TargetAmount > 0 {
		newProgress = (newAmount / targetGoal.TargetAmount) * 100
	}

	return fmt.Sprintf("✅ Berhasil menambahkan Rp%.0f ke tabungan '%s'!\n\n💰 Total: Rp%.0f / Rp%.0f (%.0f%%)",
		amount, targetGoal.Name, newAmount, targetGoal.TargetAmount, newProgress)
}

func (c *SavingsCommand) listSavingsGoals(goals []models.SavingsGoal) string {
	if len(goals) == 0 {
		return "Belum ada tabungan."
	}

	list := ""
	for _, goal := range goals {
		var progress float64 = 0
		if goal.TargetAmount > 0 {
			progress = (goal.CurrentAmount / goal.TargetAmount) * 100
		}
		list += fmt.Sprintf("- %s: Rp%.0f / Rp%.0f (%.0f%%)\n", goal.Name, goal.CurrentAmount, goal.TargetAmount, progress)
	}
	return list
}
