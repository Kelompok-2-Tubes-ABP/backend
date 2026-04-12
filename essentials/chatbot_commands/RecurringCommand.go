package chatbot_commands

import (
	"fmt"
	"time"

	"financeapi/essentials/services"

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
	if c.recurringService == nil {
		return "Layanan transaksi rutin belum tersedia."
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
