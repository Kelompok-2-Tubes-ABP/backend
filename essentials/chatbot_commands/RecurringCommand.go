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

type RecurringCommand struct {
	recurringService *services.RecurringTransactionService
}

func NewRecurringCommand(recurringService *services.RecurringTransactionService) *RecurringCommand {
	return &RecurringCommand{
		recurringService: recurringService,
	}
}

func (c *RecurringCommand) Handle(userID string, message string) string {
	userOID, _ := primitive.ObjectIDFromHex(userID)
	msg := strings.ToLower(message)

	if c.recurringService == nil {
		return "Layanan transaksi rutin belum tersedia."
	}

	// Check for delete keyword
	deleteKeywords := []string{"hapus", "delete", "remove", "batal", "cancel"}
	if utils.ContainsAny(msg, deleteKeywords) {
		return c.handleDeleteRecurring(userID, message)
	}

	// Check for edit keyword
	editKeywords := []string{"edit", "ubah", "update", "pause", "jeda", "resume", "lanjutkan"}
	if utils.ContainsAny(msg, editKeywords) {
		return c.handleEditRecurring(userID, message)
	}

	// Check for create keyword
	createKeywords := []string{"tambah", "add", "buat", "create", "langganan baru"}
	if utils.ContainsAny(msg, createKeywords) {
		return c.handleCreateRecurring(userID, message)
	}

	recurrings, err := c.recurringService.GetActiveRecurringTransactions(userOID)
	if err != nil || len(recurrings) == 0 {
		return "🔄 Kamu belum punya langganan atau pengeluaran rutin yang aktif."
	}

	summary := "🔄 **Info Pengeluaran Rutin Kamu:**\n\n"

	now := time.Now()
	oneWeekLater := now.AddDate(0, 0, 7)

	hasSoon := false
	for _, rt := range recurrings {
		if !rt.NextRunDate.IsZero() && rt.NextRunDate.Before(oneWeekLater) {
			if !hasSoon {
				summary += "⏳ **Akan didebet dalam 7 hari ke depan:**\n"
				hasSoon = true
			}
			summary += fmt.Sprintf("- **%s**: Rp%.0f (Tanggal %s)\n",
				rt.Name, rt.Amount, rt.NextRunDate.Format("02 Jan"))
		}
	}

	if hasSoon {
		summary += "\n"
	}

	summary += "📝 **Daftar Langganan Aktif:**\n"
	for _, rt := range recurrings {
		summary += fmt.Sprintf("- %s (Rp%.0f) - %s\n", rt.Name, rt.Amount, rt.Frequency)
	}

	return summary
}

func (c *RecurringCommand) handleCreateRecurring(userID, message string) string {
	userOID, _ := primitive.ObjectIDFromHex(userID)
	msg := strings.ToLower(message)

	amount := utils.ParseIndonesianAmount(message)
	if amount <= 0 {
		return "Maaf, saya tidak dapat menentukan jumlah. Contoh: 'tambah langganan netflix 100rb'"
	}

	name := utils.ExtractBillNameFromMessage(msg)
	if name == "" {
		return "Maaf, saya tidak dapat menentukan nama langganan."
	}

	// Determine frequency
	frequency := models.FreqMonthly
	if utils.ContainsAny(msg, []string{"mingguan", "weekly"}) {
		frequency = models.FreqWeekly
	} else if utils.ContainsAny(msg, []string{"tahunan", "yearly"}) {
		frequency = models.FreqYearly
	}

	txType := "expense"
	if utils.ContainsAny(msg, []string{"income", "gaji", "pemasukan"}) {
		txType = "income"
	}

	recurring := models.RecurringTransaction{
		UserID:      userOID,
		Name:        strings.Title(name),
		Amount:      amount,
		Type:        txType,
		Frequency:   frequency,
		NextRunDate: time.Now().AddDate(0, 1, 0),
		IsActive:    true,
		AutoCreate:  false,
	}

	created, err := c.recurringService.CreateRecurringTransaction(recurring)
	if err != nil {
		return fmt.Sprintf("❌ Gagal membuat langganan: %v", err)
	}

	return fmt.Sprintf("✅ **Langganan Berhasil Ditambahkan!**\n\n📋 Nama: %s\n💰 Jumlah: Rp%.0f\n🔄 Frekuensi: %s", created.Name, created.Amount, created.Frequency)
}

func (c *RecurringCommand) handleEditRecurring(userID, message string) string {
	userOID, _ := primitive.ObjectIDFromHex(userID)
	msg := strings.ToLower(message)

	recurrings, err := c.recurringService.GetActiveRecurringTransactions(userOID)
	if err != nil || len(recurrings) == 0 {
		return "Tidak ada langganan untuk diedit."
	}

	// Try to find by name
	name := utils.ExtractBillNameFromMessage(msg)
	var target *models.RecurringTransaction
	for i := range recurrings {
		if name != "" && strings.Contains(strings.ToLower(recurrings[i].Name), name) {
			target = &recurrings[i]
			break
		}
	}

	if target == nil && len(recurrings) > 0 {
		target = &recurrings[0]
	}

	if target == nil {
		return "Tidak dapat menemukan langganan."
	}

	updates := bson.M{}
	newAmount := utils.ParseIndonesianAmount(msg)
	if newAmount > 0 {
		updates["amount"] = newAmount
	}

	// Handle pause/resume
	if utils.ContainsAny(msg, []string{"pause", "jeda"}) {
		updates["status"] = "paused"
	} else if utils.ContainsAny(msg, []string{"resume", "lanjutkan"}) {
		updates["status"] = "active"
	}

	if len(updates) == 0 {
		return "Tidak ada perubahan yang diberikan."
	}

	_, err = c.recurringService.UpdateRecurringTransaction(target.ID, userOID, updates)
	if err != nil {
		return fmt.Sprintf("❌ Gagal edit langganan: %v", err)
	}

	return fmt.Sprintf("✅ Langganan berhasil diupdate: %s", target.Name)
}

func (c *RecurringCommand) handleDeleteRecurring(userID, message string) string {
	userOID, _ := primitive.ObjectIDFromHex(userID)
	msg := strings.ToLower(message)

	recurrings, err := c.recurringService.GetActiveRecurringTransactions(userOID)
	if err != nil || len(recurrings) == 0 {
		return "Tidak ada langganan untuk dihapus."
	}

	name := utils.ExtractBillNameFromMessage(msg)
	var target *models.RecurringTransaction
	for i := range recurrings {
		if name != "" && strings.Contains(strings.ToLower(recurrings[i].Name), name) {
			target = &recurrings[i]
			break
		}
	}

	if target == nil && len(recurrings) > 0 {
		target = &recurrings[0]
	}

	if target == nil {
		return "Tidak dapat menemukan langganan."
	}

	err = c.recurringService.DeleteRecurringTransaction(target.ID, userOID)
	if err != nil {
		return fmt.Sprintf("❌ Gagal hapus langganan: %v", err)
	}

	return fmt.Sprintf("🗑️ Langganan berhasil dihapus: %s", target.Name)
}
