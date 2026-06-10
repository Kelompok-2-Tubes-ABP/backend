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

type BillsCommand struct {
	billService *services.BillReminderService
}

func NewBillsCommand(billReminderService *services.BillReminderService) *BillsCommand {
	return &BillsCommand{
		billService: billReminderService,
	}
}

func (c *BillsCommand) Handle(userID string, message string) string {
	msg := strings.ToLower(message)
	userOID, _ := primitive.ObjectIDFromHex(userID)

	// Check for payment keyword first
	paymentKeywords := []string{"bayar", "pay", "lunas", "settle"}
	if utils.ContainsAny(msg, paymentKeywords) {
		return c.handlePayBill(userID, message)
	}

	// Check for delete keyword
	deleteKeywords := []string{"hapus", "delete", "remove", "batal"}
	if utils.ContainsAny(msg, deleteKeywords) {
		return c.handleDeleteBill(userID, message)
	}

	// Check for edit/update keyword
	editKeywords := []string{"edit", "ubah", "update", "ganti"}
	if utils.ContainsAny(msg, editKeywords) {
		return c.handleEditBill(userID, message)
	}

	// Check for creation keyword
	creationKeywords := []string{"ada", "tambah", "catat", "buat", "punya", "baru", "jatuh tempo"}
	amount := utils.ParseIndonesianAmount(message)
	if utils.ContainsAny(msg, creationKeywords) && amount > 0 {
		return c.handleAddBill(userID, message)
	}

	if c.billService == nil {
		return "Layanan tagihan belum tersedia."
	}

	bills, err := c.billService.GetUserBillReminders(userOID)
	if err != nil || len(bills) == 0 {
		return "📋 Kamu tidak memiliki daftar tagihan saat ini. Mau saya bantu catat tagihan baru?"
	}

	var overdue []models.BillReminder
	var dueSoon []models.BillReminder
	var upcoming []models.BillReminder

	now := time.Now()
	oneWeekLater := now.AddDate(0, 0, 7)

	for _, bill := range bills {
		if bill.IsPaid {
			continue
		}
		status := bill.GetDueStatus()
		if status == "overdue" {
			overdue = append(overdue, bill)
		} else if bill.NextDueDate.Before(oneWeekLater) {
			dueSoon = append(dueSoon, bill)
		} else {
			upcoming = append(upcoming, bill)
		}
	}

	if len(overdue) == 0 && len(dueSoon) == 0 && len(upcoming) == 0 {
		return "✅ Mantap! Semua tagihan kamu bulan ini sudah lunas."
	}

	summary := ""

	if len(overdue) > 0 {
		summary += "🚨 **GAWAT! Tagihan ini sudah lewat tempo:**\n"
		for _, b := range overdue {
			summary += fmt.Sprintf("- %s (Rp%.0f) - Segera bayar ya!\n", b.Name, b.Amount)
		}
		summary += "\n"
	}

	if len(dueSoon) > 0 {
		summary += "📅 **Minggu ini ada tagihan yang mau jatuh tempo lho:**\n"
		for _, b := range dueSoon {
			days := int(b.NextDueDate.Sub(now).Hours() / 24)
			dateStr := b.NextDueDate.Format("02 Jan")
			dayLabel := fmt.Sprintf("%d hari lagi", days)
			if days <= 0 {
				dayLabel = "HARI INI"
			} else if days == 1 {
				dayLabel = "Besok"
			}

			summary += fmt.Sprintf("- **%s** (Rp%.0f) - Jatuh tempo %s (%s)\n",
				b.Name, b.Amount, dateStr, dayLabel)
		}
		summary += "\n"
	}

	if len(upcoming) > 0 && len(summary) < 500 {
		summary += "🗒️ **Tagihan lainnya:**\n"
		for i, b := range upcoming {
			if i >= 3 {
				break
			}
			summary += fmt.Sprintf("- %s (Rp%.0f) - %s\n", b.Name, b.Amount, b.NextDueDate.Format("02 Jan"))
		}
	}

	return summary
}

func (c *BillsCommand) handleAddBill(userID, message string) string {
	msg := strings.ToLower(message)
	userOID, _ := primitive.ObjectIDFromHex(userID)

	amount := utils.ParseIndonesianAmount(message)
	name := utils.ExtractBillNameFromMessage(msg)
	if name == "" || name == "tagihan" || name == "bill" {
		name = "Tagihan Baru"
	}

	category := utils.ExtractExpenseCategory(msg)
	if category == "" {
		category = "other"
	}

	bill := models.BillReminder{
		UserID:      userOID,
		Name:        strings.Title(name),
		Amount:      amount,
		Category:    category,
		NextDueDate: time.Now().AddDate(0, 1, 0), // Default to next month
		IsPaid:      false,
	}

	created, err := c.billService.CreateBillReminder(bill)
	if err != nil {
		return fmt.Sprintf("❌ Gagal mencatat tagihan: %v", err)
	}

	res := fmt.Sprintf("✅ **Tagihan Berhasil Dicatat!**\n\n")
	res += fmt.Sprintf("📋 **Nama:** %s\n", created.Name)
	res += fmt.Sprintf("💰 **Nominal:** Rp%.0f\n", created.Amount)
	res += fmt.Sprintf("📂 **Kategori:** %s\n", created.Category)
	res += fmt.Sprintf("📅 **Tempo:** %s (Estimasi)\n", created.NextDueDate.Format("02 Jan 2006"))
	res += fmt.Sprintf("\nSaya akan ingatkan kamu sebelum jatuh tempo ya!")

	return res
}

func (c *BillsCommand) handlePayBill(userID, message string) string {
	userOID, _ := primitive.ObjectIDFromHex(userID)
	msg := strings.ToLower(message)

	bills, err := c.billService.GetUserBillReminders(userOID)
	if err != nil || len(bills) == 0 {
		return "Tidak ada tagihan untuk dibayar."
	}

	// Try to find bill by name
	billName := utils.ExtractBillNameFromMessage(msg)
	var targetBill *models.BillReminder

	for i := range bills {
		if billName != "" && strings.Contains(strings.ToLower(bills[i].Name), billName) {
			targetBill = &bills[i]
			break
		}
	}

	// If no name match, use first unpaid bill
	if targetBill == nil {
		for i := range bills {
			if !bills[i].IsPaid {
				targetBill = &bills[i]
				break
			}
		}
	}

	if targetBill == nil {
		return "Tidak ada tagihan yang perlu dibayar."
	}

	err = c.billService.MarkAsPaid(targetBill.ID, userOID, targetBill.Amount, "Via Chatbot")
	if err != nil {
		return fmt.Sprintf("❌ Gagal membayar tagihan: %v", err)
	}

	return fmt.Sprintf("✅ **Tagihan Lunas!**\n\n📋 %s\n💰 Rp%.0f\n\nSelamat! Kamu sudah tidak punya tagihan ini.", targetBill.Name, targetBill.Amount)
}

func (c *BillsCommand) handleEditBill(userID, message string) string {
	userOID, _ := primitive.ObjectIDFromHex(userID)
	msg := strings.ToLower(message)

	bills, err := c.billService.GetUserBillReminders(userOID)
	if err != nil || len(bills) == 0 {
		return "Tidak ada tagihan untuk diedit."
	}

	billName := utils.ExtractBillNameFromMessage(msg)
	var targetBill *models.BillReminder

	for i := range bills {
		if billName != "" && strings.Contains(strings.ToLower(bills[i].Name), billName) {
			targetBill = &bills[i]
			break
		}
	}

	if targetBill == nil && len(bills) > 0 {
		targetBill = &bills[0]
	}

	if targetBill == nil {
		return "Tidak dapat menemukan tagihan yang ingin diedit."
	}

	// Build updates
	updates := bson.M{}
	newAmount := utils.ParseIndonesianAmount(msg)
	if newAmount > 0 {
		updates["amount"] = newAmount
	}
	newName := utils.ExtractBillNameFromMessage(msg)
	if newName != "" && newName != billName {
		updates["name"] = strings.Title(newName)
	}

	if len(updates) == 0 {
		return "Tidak ada perubahan yang diberikan. Contoh: 'edit tagihan internet menjadi 150000'"
	}

	_, err = c.billService.UpdateBillReminder(targetBill.ID, userOID, updates)
	if err != nil {
		return fmt.Sprintf("❌ Gagal mengedit tagihan: %v", err)
	}

	return fmt.Sprintf("✅ Tagihan berhasil diedit!\n\n📋 %s\n💰 Rp%.0f", targetBill.Name, targetBill.Amount)
}

func (c *BillsCommand) handleDeleteBill(userID, message string) string {
	userOID, _ := primitive.ObjectIDFromHex(userID)
	msg := strings.ToLower(message)

	bills, err := c.billService.GetUserBillReminders(userOID)
	if err != nil || len(bills) == 0 {
		return "Tidak ada tagihan untuk dihapus."
	}

	billName := utils.ExtractBillNameFromMessage(msg)
	var targetBill *models.BillReminder

	for i := range bills {
		if billName != "" && strings.Contains(strings.ToLower(bills[i].Name), billName) {
			targetBill = &bills[i]
			break
		}
	}

	if targetBill == nil && len(bills) > 0 {
		targetBill = &bills[0]
	}

	if targetBill == nil {
		return "Tidak dapat menemukan tagihan yang ingin dihapus."
	}

	err = c.billService.DeleteBillReminder(targetBill.ID, userOID)
	if err != nil {
		return fmt.Sprintf("❌ Gagal menghapus tagihan: %v", err)
	}

	return fmt.Sprintf("🗑️ Tagihan berhasil dihapus!\n\n📋 %s\n💰 Rp%.0f", targetBill.Name, targetBill.Amount)
}
