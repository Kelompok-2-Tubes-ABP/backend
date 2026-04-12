package chatbot_commands

import (
	"fmt"

	"financeapi/essentials/services"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AccountCommand struct {
	accountService *services.AccountService
}

func NewAccountCommand(accountService *services.AccountService) *AccountCommand {
	return &AccountCommand{
		accountService: accountService,
	}
}

func (c *AccountCommand) Handle(userID string, message string) string {
	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return "Error: Invalid user ID"
	}

	if c.accountService == nil {
		return "Account service belum tersedia."
	}

	accounts, err := c.accountService.GetUserAccounts(userOID)
	if err != nil {
		return fmt.Sprintf("Error mengambil data akun: %v", err)
	}

	if len(accounts) == 0 {
		return "Belum ada akun yang terdaftar. Tambahkan akun di menu utama."
	}

	summary := "🏦 Saldo Akun:\n\n"
	totalBalance := 0.0

	for _, account := range accounts {
		balance := account.CurrentBalance
		if account.Type == "credit" {
			balance = -account.CurrentBalance
		}
		totalBalance += balance

		icon := "🏦"
		switch account.Type {
		case "wallet":
			icon = "👛"
		case "cash":
			icon = "💵"
		case "credit":
			icon = "💳"
		case "savings":
			icon = "🎯"
		case "investment":
			icon = "📈"
		}

		balanceStr := fmt.Sprintf("Rp%.0f", account.CurrentBalance)
		if account.Type == "credit" {
			balanceStr = fmt.Sprintf("Rp%.0f (hutang)", account.CurrentBalance)
		}

		summary += fmt.Sprintf("%s %s\n   %s - %s\n\n", icon, account.Name, account.Institution, balanceStr)
	}

	summary += fmt.Sprintf("💰 Total: Rp%.0f", totalBalance)

	return summary
}
