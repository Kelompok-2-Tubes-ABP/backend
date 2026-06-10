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

type DebtCommand struct {
	debtService    *services.DebtService
	accountService *services.AccountService
}

func NewDebtCommand(debtService *services.DebtService, accountService *services.AccountService) *DebtCommand {
	return &DebtCommand{
		debtService:    debtService,
		accountService: accountService,
	}
}

func (c *DebtCommand) Handle(userID string, message string) string {
	msg := strings.ToLower(message)

	// 1. Detect payment intent
	paymentKeywords := []string{"bayar", "pay", "cicil", "setor", "bayarin"}
	if utils.ContainsAny(msg, paymentKeywords) {
		return c.handleDebtPayment(userID, message)
	}

	// 2. Detect creation intent
	creationKeywords := []string{"ada", "tambah", "catat", "buat", "punya", "mempunyai", "baru"}
	amount := utils.ParseIndonesianAmount(message)
	if (utils.ContainsAny(msg, creationKeywords) && amount > 0) || (amount > 0 && utils.ContainsAny(msg, []string{"bunga", "tenor", "bulan"})) {
		return c.handleAddDebt(userID, message)
	}

	userOID, _ := primitive.ObjectIDFromHex(userID)
	if c.debtService == nil {
		return "Layanan hutang belum tersedia."
	}

	debts, err := c.debtService.GetUserDebts(userOID)
	if err != nil || len(debts) == 0 {
		return "🎯 Kamu tidak memiliki hutang aktif saat ini. Bagus sekali!"
	}

	summary := "🏦 **Daftar Hutang & Cicilan Kamu:**\n\n"
	for _, d := range debts {
		progress := (1 - (d.CurrentBalance / d.OriginalAmount)) * 100
		summary += fmt.Sprintf("- **%s**: Sisa tagihan Rp%.0f / Rp%.0f\n", d.Name, d.CurrentBalance, d.OriginalAmount)
		summary += fmt.Sprintf("  📊 Progress Pelunasan: %.1f%%\n", progress)
		summary += fmt.Sprintf("  📅 Pembayaran Berikutnya: %s (Rp%.0f)\n\n", d.NextPaymentDate.Format("02 Jan"), d.PaymentAmount)
	}

	summary += "💡 *Tips: Kamu bisa bilang: \"Bayar [nama hutang] [jumlah] pake [nama akun]\"*"
	return summary
}

func (c *DebtCommand) handleAddDebt(userID, message string) string {
	msg := strings.ToLower(message)
	userOID, _ := primitive.ObjectIDFromHex(userID)

	amount := utils.ParseIndonesianAmount(message)

	amountStr := utils.ExtractAmountString(msg)
	msgWithoutAmount := msg
	if amountStr != "" {
		msgWithoutAmount = strings.Replace(msg, amountStr, "", 1)
	}

	interest := utils.ExtractInterestRate(msg)
	tenor := utils.ExtractTenor(msg)
	name := utils.ExtractDebtNameFromMessage(msgWithoutAmount)
	if name == "" || name == "credit" || name == "kredit" || name == "tagihan" {
		name = "Kredit Baru"
	}

	debt := models.Debt{
		UserID:           userOID,
		Name:             strings.Title(name),
		OriginalAmount:   amount,
		CurrentBalance:   amount,
		InterestRate:     interest,
		TenorMonths:      tenor,
		PaymentFrequency: models.RepayMonthly,
		StartDate:        time.Now(),
	}

	if tenor > 0 {
		totalWithInterest := amount * (1 + (interest/100)*float64(tenor))
		debt.PaymentAmount = totalWithInterest / float64(tenor)
	}

	created, err := c.debtService.CreateDebt(debt)
	if err != nil {
		return fmt.Sprintf("❌ Gagal mencatat hutang: %v", err)
	}

	res := fmt.Sprintf("✅ **Hutang Berhasil Dicatat!**\n\n")
	res += fmt.Sprintf("🏦 **Nama:** %s\n", created.Name)
	res += fmt.Sprintf("💰 **Nominal:** Rp%.0f\n", created.OriginalAmount)
	if created.InterestRate > 0 {
		res += fmt.Sprintf("📈 **Bunga:** %.1f%% per bulan\n", created.InterestRate)
	}
	if created.TenorMonths > 0 {
		res += fmt.Sprintf("📅 **Tenor:** %d Bulan\n", created.TenorMonths)
		res += fmt.Sprintf("💸 **Estimasi Cicilan:** Rp%.0f/bulan\n", debt.PaymentAmount)
	}
	res += fmt.Sprintf("\nSemangat pelunasannya ya! Kamu bisa cek detailnya kapan saja dengan ketik 'cek hutang'.")

	return res
}

func (c *DebtCommand) handleDebtPayment(userID, message string) string {
	userOID, _ := primitive.ObjectIDFromHex(userID)
	msg := strings.ToLower(message)

	amount := utils.ParseIndonesianAmount(message)
	if amount <= 0 {
		return "⚠️ **Jumlah Tidak Valid.** Sebutkan nominalnya ya, contoh: 'Bayar KPR 2 juta pake BCA'."
	}

	amountStr := utils.ExtractAmountString(msg)
	msgWithoutAmount := msg
	if amountStr != "" {
		msgWithoutAmount = strings.Replace(msg, amountStr, "", 1)
	}

	debtNameQuery := utils.ExtractDebtNameFromMessage(msgWithoutAmount)
	accountNameQuery := utils.ExtractAccountNameFromMessage(msg)

	if debtNameQuery == "" {
		return "🤔 **Hutang yang mana?** Sebutkan nama hutangnya, misal: 'Bayar **Laptop** 500rb'."
	}

	debts, _ := c.debtService.GetUserDebts(userOID)
	var targetDebt *models.Debt
	for _, d := range debts {
		dName := strings.ToLower(d.Name)
		if dName == debtNameQuery || strings.Contains(dName, debtNameQuery) || strings.Contains(debtNameQuery, dName) {
			targetDebt = &d
			break
		}
	}

	if targetDebt == nil {
		return fmt.Sprintf("❌ **Hutang '%s' tidak ditemukan.**\nCoba ketik 'cek hutang' untuk melihat daftar hutangmu.", debtNameQuery)
	}

	var accountID primitive.ObjectID
	var targetAccount *models.Account
	if accountNameQuery != "" && c.accountService != nil {
		accounts, _ := c.accountService.GetUserAccounts(userOID)
		for _, acc := range accounts {
			accName := strings.ToLower(acc.Name)
			if accName == accountNameQuery || strings.Contains(accName, accountNameQuery) {
				targetAccount = &acc
				accountID = acc.ID
				break
			}
		}

		if targetAccount != nil {
			if targetAccount.CurrentBalance < amount {
				return fmt.Sprintf("🚫 **Saldo Tidak Cukup.**\nSaldo di **%s** cuma Rp%.0f, sedangkan kamu mau bayar Rp%.0f.",
					targetAccount.Name, targetAccount.CurrentBalance, amount)
			}
		}
	}

	_, err := c.debtService.MakePayment(targetDebt.ID, userOID, accountID, amount)
	if err != nil {
		return fmt.Sprintf("💥 **Gagal memproses pembayaran:** %v", err)
	}

	res := "🎉 **Pembayaran Berhasil Dicatat!**\n\n"
	res += fmt.Sprintf("🔹 **Tujuan:** %s\n", targetDebt.Name)
	res += fmt.Sprintf("💰 **Nominal:** Rp%.0f\n", amount)
	if targetAccount != nil {
		res += fmt.Sprintf("💳 **Sumber:** %s\n", targetAccount.Name)
	}

	remaining := targetDebt.CurrentBalance - amount
	if remaining <= 0 {
		res += "\n🎊 **LUNAS!** Selamat, hutang ini sudah lunas sepenuhnya!"
	} else {
		res += fmt.Sprintf("\n📉 **Sisa Hutang:** Rp%.0f", remaining)
	}

	return res
}
