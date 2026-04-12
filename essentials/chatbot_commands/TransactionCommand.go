package chatbot_commands

import (
	"fmt"
	"strings"
	"time"

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
	// Check for expense/spending keywords
	expenseKeywords := []string{"spent", "beli", "buy", "purchase", "makan", "food", "lunch", "dinner", "breakfast", "belanja", "keluar", "bayar", "pay", "pengeluaran", "expense", "transaction"}
	if containsAny(msgLower, expenseKeywords) {
		return c.handleAddTransaction(userID, message)
	}

	// Check for income/earning keywords
	incomeKeywords := []string{"income", "pemasukan", "gaji", "pendapatan", "uang masuk", "gajian", "dapet", "dapat", "salary", "earned"}
	if containsAny(msgLower, incomeKeywords) {
		return c.handleAddTransaction(userID, message)
	}

	// Check for explicit add keywords
	if containsAny(msgLower, []string{"tambah", "add", "input", "catat", "insert"}) {
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

	isIncome := containsAny(msg, []string{"income", "pemasukan", "gaji", "pendapatan", "uang masuk", "gajian", "dapet", "dapat", "salary", "earned", "terima", "duit masuk"})
	isExpense := containsAny(msg, []string{"spent", "beli", "buy", "purchase", "makan", "food", "lunch", "dinner", "belanja", "keluar", "bayar", "pay", "pengeluaran", "expense", "transaction", "untuk", "buying"})

	category := "outcome"
	if isIncome && !isExpense {
		category = "income"
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
		Date:        time.Now(),
		Description: description,
	}

	_, err := c.txService.CreateTransaction(transaction)
	if err != nil {
		return fmt.Sprintf("Error menambahkan transaksi: %v", err)
	}

	categoryLabel := "Pengeluaran"
	if category == "income" {
		categoryLabel = "Pemasukan"
	}

	return fmt.Sprintf("✅ Transaksi berhasil dicatat!\n\n💰 %s: Rp%.0f\n📝 Note: %s",
		categoryLabel, amount, description)
}
