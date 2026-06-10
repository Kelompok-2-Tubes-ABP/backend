package chatbot_commands

import (
	"fmt"
	"strings"

	"financeapi/essentials/models"
	"financeapi/essentials/services"
	"financeapi/essentials/utils"

	"go.mongodb.org/mongo-driver/bson"
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

	msg := strings.ToLower(message)

	if c.accountService == nil {
		return "Account service belum tersedia."
	}

	// Check for delete keyword
	deleteKeywords := []string{"hapus", "delete", "remove"}
	if utils.ContainsAny(msg, deleteKeywords) {
		return c.handleDeleteAccount(userID, message)
	}

	// Check for edit keyword
	editKeywords := []string{"edit", "ubah", "update", "ganti", "topup", "tarik"}
	if utils.ContainsAny(msg, editKeywords) {
		return c.handleEditAccount(userID, message)
	}

	// Check for create keyword
	createKeywords := []string{"tambah", "add", "buat", "register", "buka"}
	if utils.ContainsAny(msg, createKeywords) {
		return c.handleCreateAccount(userID, message)
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
		if account.Type == models.AccountTypeCredit {
			balance = -account.CurrentBalance
		}
		totalBalance += balance

		icon := "🏦"
		switch account.Type {
		case models.AccountTypeWallet:
			icon = "👛"
		case models.AccountTypeCash:
			icon = "💵"
		case models.AccountTypeCredit:
			icon = "💳"
		case models.AccountTypeSavings:
			icon = "🎯"
		case models.AccountTypeInvestment:
			icon = "📈"
		}

		balanceStr := fmt.Sprintf("Rp%.0f", account.CurrentBalance)
		if account.Type == models.AccountTypeCredit {
			balanceStr = fmt.Sprintf("Rp%.0f (hutang)", account.CurrentBalance)
		}

		summary += fmt.Sprintf("%s %s\n   %s - %s\n\n", icon, account.Name, account.Institution, balanceStr)
	}

	summary += fmt.Sprintf("💰 Total: Rp%.0f", totalBalance)

	return summary
}

func (c *AccountCommand) handleCreateAccount(userID, message string) string {
	userOID, _ := primitive.ObjectIDFromHex(userID)
	msg := strings.ToLower(message)

	name := utils.ExtractAccountNameFromMessage(msg)
	if name == "" {
		return "Maaf, saya tidak dapat menentukan nama akun. Contoh: 'tambah akun bank bca'"
	}

	// Determine account type
	accountType := models.AccountTypeBank
	if utils.ContainsAny(msg, []string{"e-wallet", "ewallet", "gojek", "grab", "dana", "ovo", "linkaja"}) {
		accountType = models.AccountTypeWallet
	} else if utils.ContainsAny(msg, []string{"cash", "tunai"}) {
		accountType = models.AccountTypeCash
	} else if utils.ContainsAny(msg, []string{"kredit", "credit", "card"}) {
		accountType = models.AccountTypeCredit
	} else if utils.ContainsAny(msg, []string{"savings", "tabungan"}) {
		accountType = models.AccountTypeSavings
	}

	initialBalance := utils.ParseIndonesianAmount(msg)

	account := models.Account{
		UserID:          userOID,
		Name:            strings.Title(name),
		Type:            accountType,
		Institution:     strings.Title(name),
		CurrentBalance:  initialBalance,
	}

	created, err := c.accountService.CreateAccount(account)
	if err != nil {
		return fmt.Sprintf("❌ Gagal membuat akun: %v", err)
	}

	return fmt.Sprintf("✅ **Akun Berhasil Dibuat!**\n\n🏦 Nama: %s\n💰 Tipe: %s\n💵 Saldo: Rp%.0f", created.Name, created.Type, created.CurrentBalance)
}

func (c *AccountCommand) handleEditAccount(userID, message string) string {
	userOID, _ := primitive.ObjectIDFromHex(userID)
	msg := strings.ToLower(message)

	accounts, err := c.accountService.GetUserAccounts(userOID)
	if err != nil || len(accounts) == 0 {
		return "Tidak ada akun untuk diedit."
	}

	// Try to find account by name
	accountName := utils.ExtractAccountNameFromMessage(msg)
	var targetAccount *models.Account

	for i := range accounts {
		if accountName != "" && strings.Contains(strings.ToLower(accounts[i].Name), accountName) {
			targetAccount = &accounts[i]
			break
		}
	}

	if targetAccount == nil && len(accounts) > 0 {
		targetAccount = &accounts[0]
	}

	if targetAccount == nil {
		return "Tidak dapat menemukan akun."
	}

	updates := bson.M{}

	// Check for balance update (topup/withdraw)
	newBalance := utils.ParseIndonesianAmount(msg)
	if newBalance > 0 {
		if utils.ContainsAny(msg, []string{"topup", "tambah"}) {
			newBalance = targetAccount.CurrentBalance + newBalance
		} else if utils.ContainsAny(msg, []string{"tarik", "withdraw", "kurangi"}) {
			newBalance = targetAccount.CurrentBalance - newBalance
		}
		updates["current_balance"] = newBalance
	}

	if len(updates) == 0 {
		return "Tidak ada perubahan yang diberikan."
	}

	_, err = c.accountService.UpdateAccount(targetAccount.ID, userOID, updates)
	if err != nil {
		return fmt.Sprintf("❌ Gagal edit akun: %v", err)
	}

	return fmt.Sprintf("✅ Akun berhasil diupdate! %s - Rp%.0f", targetAccount.Name, newBalance)
}

func (c *AccountCommand) handleDeleteAccount(userID, message string) string {
	userOID, _ := primitive.ObjectIDFromHex(userID)
	msg := strings.ToLower(message)

	accounts, err := c.accountService.GetUserAccounts(userOID)
	if err != nil || len(accounts) == 0 {
		return "Tidak ada akun untuk dihapus."
	}

	accountName := utils.ExtractAccountNameFromMessage(msg)
	var targetAccount *models.Account

	for i := range accounts {
		if accountName != "" && strings.Contains(strings.ToLower(accounts[i].Name), accountName) {
			targetAccount = &accounts[i]
			break
		}
	}

	if targetAccount == nil && len(accounts) > 0 {
		targetAccount = &accounts[0]
	}

	if targetAccount == nil {
		return "Tidak dapat menemukan akun."
	}

	err = c.accountService.DeleteAccount(targetAccount.ID, userOID)
	if err != nil {
		return fmt.Sprintf("❌ Gagal hapus akun: %v", err)
	}

	return fmt.Sprintf("🗑️ Akun berhasil dihapus: %s", targetAccount.Name)
}
