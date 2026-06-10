package chatbot_commands

import (
	"fmt"
	"strings"
	"time"

	"financeapi/essentials/constants"
	"financeapi/essentials/models"
	"financeapi/essentials/services"
	"financeapi/essentials/utils"
)

type TransactionCommand struct {
	txService *services.TransactionService
}

func NewTransactionCommand(txService *services.TransactionService) *TransactionCommand {
	return &TransactionCommand{
		txService: txService,
	}
}

func (c *TransactionCommand) Handle(userID string, message string) string {
	msg := strings.ToLower(message)
	return c.handleTransaction(userID, msg, message)
}

func (c *TransactionCommand) handleTransaction(userID, msgLower, message string) string {
	// Check for delete keywords first
	deleteKeywords := []string{"hapus", "delete", "remove", "batal", "cancel"}
	if utils.ContainsAny(msgLower, deleteKeywords) {
		return c.handleDeleteTransaction(userID, message)
	}

	// Check for edit keywords
	editKeywords := []string{"edit", "ubah", "update", "ganti", "change", "modify"}
	if utils.ContainsAny(msgLower, editKeywords) {
		return c.handleEditTransaction(userID, message)
	}

	// Check for expense/spending keywords
	expenseKeywords := []string{"spent", "beli", "buy", "purchase", "makan", "food", "lunch", "dinner", "breakfast", "belanja", "keluar", "bayar", "pay", "pengeluaran", "expense", "transaction"}
	if utils.ContainsAny(msgLower, expenseKeywords) {
		return c.handleAddTransaction(userID, message)
	}

	// Check for income/earning keywords
	incomeKeywords := []string{"income", "pemasukan", "gaji", "pendapatan", "uang masuk", "gajian", "dapet", "dapat", "salary", "earned"}
	if utils.ContainsAny(msgLower, incomeKeywords) {
		return c.handleAddTransaction(userID, message)
	}

	// Check for explicit add keywords
	if utils.ContainsAny(msgLower, []string{"tambah", "add", "input", "catat", "insert"}) {
		return c.handleAddTransaction(userID, message)
	}

	transactions, err := c.txService.ShowTransaction(userID)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if len(transactions) == 0 {
		return "Belum ada transaksi. Mau tambahkan transaksi pertama?"
	}

	summary := fmt.Sprintf("Kamu punya %d transaksi:\n\n", len(transactions))
	for i, tx := range transactions {
		if i >= 5 {
			break
		}
		summary += fmt.Sprintf("- %s: Rp%.0f (%s)\n", tx.Date.Format("02 Jan"), tx.Amount, tx.Category)
	}
	return summary
}

func (c *TransactionCommand) handleAddTransaction(userID, message string) string {
	msg := strings.ToLower(message)

	amount := utils.ParseIndonesianAmount(message)
	if amount <= 0 {
		return "Maaf, saya tidak dapat menentukan jumlah transaksi. Contoh: 'tambah pengeluaran 50000 untuk makan'"
	}

	isIncome := utils.ContainsAny(msg, []string{"income", "pemasukan", "gaji", "pendapatan", "uang masuk", "gajian", "dapet", "dapat", "salary", "earned", "terima", "duit masuk"})
	isExpense := utils.ContainsAny(msg, []string{"spent", "beli", "buy", "purchase", "makan", "food", "lunch", "dinner", "belanja", "keluar", "bayar", "pay", "pengeluaran", "expense", "transaction", "untuk", "buying"})

	category := constants.TrxOutcome
	txType := constants.TrxOutcome
	if isIncome && !isExpense {
		category = constants.TrxIncome
		txType = constants.TrxIncome
	} else {
		specificCategory := utils.ExtractExpenseCategory(msg)
		if specificCategory != "" {
			category = specificCategory
		}
	}

	description := utils.ExtractTransactionDescription(message)

	transaction := models.Transaction{
		User_id:     userID,
		Amount:      amount,
		Category:    category,
		Type:        txType,
		Date:        time.Now(),
		Description: description,
	}

	_, err := c.txService.CreateTransaction(transaction)
	if err != nil {
		return fmt.Sprintf("Error menambahkan transaksi: %v", err)
	}

	categoryLabel := "Pengeluaran"
	if txType == constants.TrxIncome {
		categoryLabel = "Pemasukan"
	}

	return fmt.Sprintf("✅ Transaksi berhasil dicatat!\n\n💰 %s: Rp%.0f\n📝 Note: %s",
		categoryLabel, amount, description)
}

func (c *TransactionCommand) handleEditTransaction(userID, message string) string {
	msg := strings.ToLower(message)

	// First, get all transactions to find the one to edit
	transactions, err := c.txService.ShowTransaction(userID)
	if err != nil || len(transactions) == 0 {
		return "Tidak ada transaksi untuk diedit."
	}

	// Try to extract amount to identify the transaction
	amount := utils.ParseIndonesianAmount(message)

	// Try to extract category/description
	category := utils.ExtractExpenseCategory(msg)
	description := utils.ExtractTransactionDescription(message)

	// Try to find matching transaction by amount or description
	var targetTx *models.Transaction
	for i := range transactions {
		if amount > 0 && transactions[i].Amount == amount {
			targetTx = &transactions[i]
			break
		}
		if description != "" && description != "Via Chatbot" && strings.Contains(strings.ToLower(transactions[i].Description), description) {
			targetTx = &transactions[i]
			break
		}
	}

	// If no match by amount/description, use the most recent one
	if targetTx == nil && len(transactions) > 0 {
		targetTx = &transactions[0] // Most recent
	}

	if targetTx == nil {
		return "Tidak dapat menemukan transaksi yang ingin diedit. Coba sebutkan jumlah atau deskripsinya."
	}

	// Build update model
	update := models.Transaction{}
	if amount > 0 {
		update.Amount = amount
	}
	if category != "" {
		update.Category = category
	}
	if description != "" && description != "Via Chatbot" {
		update.Description = description
	}

	// Check if there's anything to update
	if update.Amount == 0 && update.Category == "" && update.Description == "" {
		return "Tidak ada perubahan yang diberikan. Contoh: 'edit transaksi 50000 menjadi 60000'"
	}

	err = c.txService.UpdateTransaction(targetTx.ID, update)
	if err != nil {
		return fmt.Sprintf("❌ Gagal mengedit transaksi: %v", err)
	}

	return fmt.Sprintf("✅ Transaksi berhasil diupdate!\n\n📝 ID: %s\n💰 Jumlah: Rp%.0f\n📂 Kategori: %s\n📝 Note: %s",
		targetTx.ID.Hex(), targetTx.Amount, targetTx.Category, targetTx.Description)
}

func (c *TransactionCommand) handleDeleteTransaction(userID, message string) string {
	// Get transactions to find one to delete
	transactions, err := c.txService.ShowTransaction(userID)
	if err != nil || len(transactions) == 0 {
		return "Tidak ada transaksi untuk dihapus."
	}

	// Try to extract amount to identify the transaction
	amount := utils.ParseIndonesianAmount(message)
	description := utils.ExtractTransactionDescription(message)

	var targetTx *models.Transaction
	for i := range transactions {
		if amount > 0 && transactions[i].Amount == amount {
			targetTx = &transactions[i]
			break
		}
		if description != "" && description != "Via Chatbot" && strings.Contains(strings.ToLower(transactions[i].Description), description) {
			targetTx = &transactions[i]
			break
		}
	}

	// Use most recent if no match
	if targetTx == nil && len(transactions) > 0 {
		targetTx = &transactions[0]
	}

	if targetTx == nil {
		return "Tidak dapat menemukan transaksi yang ingin dihapus."
	}

	err = c.txService.DeleteTransaction(targetTx.ID)
	if err != nil {
		return fmt.Sprintf("❌ Gagal menghapus transaksi: %v", err)
	}

	return fmt.Sprintf("🗑️ Transaksi berhasil dihapus!\n\n📝 ID: %s\n💰 Rp%.0f (%s)",
		targetTx.ID.Hex(), targetTx.Amount, targetTx.Category)
}
