package chatbot_commands

import (
	"fmt"
	"strings"
	"time"

	"financeapi/essentials/models"
	"financeapi/essentials/services"
	"financeapi/essentials/utils"

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

	creationKeywords := []string{"ada", "tambah", "catat", "buat", "punya", "baru", "jatuh tempo"}
	amount := utils.ParseIndonesianAmount(message)
	if containsAny(msg, creationKeywords) && amount > 0 {
		return c.handleAddBill(userID, message)
	}

	userOID, _ := primitive.ObjectIDFromHex(userID)
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
