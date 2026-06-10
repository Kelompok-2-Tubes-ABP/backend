package chatbot_commands

import (
	"fmt"
	"strings"
	"time"

	"financeapi/essentials/constants"
	"financeapi/essentials/models"
	"financeapi/essentials/services"
	"financeapi/essentials/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InvestCommand struct {
	investmentService *services.InvestmentService
	priceService      *services.PriceService
}

func NewInvestCommand(invService *services.InvestmentService, priceService *services.PriceService) *InvestCommand {
	return &InvestCommand{
		investmentService: invService,
		priceService:      priceService,
	}
}

func (c *InvestCommand) Handle(userID string, message string) string {
	msgLower := strings.ToLower(message)

	// Priority 1: Price check (most specific)
	priceKeywords := []string{"harga", "price", "nilai", "berapa"}
	if utils.ContainsAny(msgLower, priceKeywords) && len(strings.Fields(msgLower)) <= 8 {
		return c.handleAssetPriceQuery(userID, message)
	}

	// Priority 2: Suggestions/recommendations
	suggestKeywords := []string{"saran", "recommend", "suggest", "ide", "tips", "bagus", "good", "stocks", "saham", "crypto", "aapl", "googl", "msft", "tsla", "tesla", "apple", "google", "microsoft"}
	if utils.ContainsAny(msgLower, suggestKeywords) {
		return c.handleInvestmentRecommendation(userID, message)
	}

	// Priority 3: Create/add investment (only if explicit add words)
	createKeywords := []string{"tambah investasi", "add investasi", "beli investasi", "buy investasi", "investasikan", "masukan investasi"}
	if utils.ContainsAny(msgLower, createKeywords) {
		return c.handleAddInvestment(userID, message)
	}

	// Default: Show portfolio
	return c.handleInvestment(userID, message)
}

func (c *InvestCommand) handleInvestment(userID, message string) string {
	userOID, _ := primitive.ObjectIDFromHex(userID)
	if c.investmentService == nil {
		return "Layanan investasi belum tersedia."
	}

	investments, err := c.investmentService.GetUserInvestments(userOID)
	if err != nil {
		return fmt.Sprintf("Error mengambil data investasi: %v", err)
	}

	if len(investments) == 0 {
		return c.handleInvestmentRecommendation(userID, message)
	}

	summary := "📈 Portfolio Investasi:\n\n"
	totalValue := 0.0
	totalCost := 0.0

	for _, inv := range investments {
		icon := "📊"
		switch string(inv.Type) {
		case constants.InvCrypto:
			icon = "🪙"
		case constants.InvStock:
			icon = "📈"
		case constants.InvBond:
			icon = "📜"
		case constants.InvRealEstate:
			icon = "🏠"
		}

		totalValue += inv.TotalValue
		totalCost += inv.TotalCost

		gainLossIcon := "📊"
		if inv.GainLoss >= 0 {
			gainLossIcon = "📈"
		} else {
			gainLossIcon = "📉"
		}

		summary += fmt.Sprintf("%s %s (%s)\n", icon, inv.Name, inv.Symbol)
		summary += fmt.Sprintf("   Jumlah: %.4f\n", inv.Quantity)
		summary += fmt.Sprintf("   Harga Rata-rata: Rp%.0f\n", inv.AverageCost)
		summary += fmt.Sprintf("   Harga Saat Ini: Rp%.0f\n", inv.CurrentPrice)
		summary += fmt.Sprintf("%s Gain/Loss: Rp%.0f (%.2f%%)\n\n", gainLossIcon, inv.GainLoss, inv.GainLossPercent)
	}

	if totalCost > 0 {
		totalGainLoss := totalValue - totalCost
		percent := (totalGainLoss / totalCost) * 100
		summary += fmt.Sprintf("💰 Total Nilai: Rp%.0f\n", totalValue)
		summary += fmt.Sprintf("💵 Total Biaya: Rp%.0f\n", totalCost)
		if totalGainLoss >= 0 {
			summary += fmt.Sprintf("📈 Total Gain: Rp%.0f (%.2f%%)", totalGainLoss, percent)
		} else {
			summary += fmt.Sprintf("📉 Total Loss: Rp%.0f (%.2f%%)", totalGainLoss, percent)
		}
	}

	return summary
}

func (c *InvestCommand) handleInvestmentRecommendation(userID, message string) string {
	suggestion := "💰 Harga Investasi Terkini:\n\n"
	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return "Error: Invalid user ID"
	}

	if c.investmentService != nil {
		investments, err := c.investmentService.GetUserInvestments(userOID)
		if err == nil && len(investments) > 0 {
			suggestion += "📊 Portfolio Kamu:\n"
			totalValue := 0.0
			for _, inv := range investments {
				var price float64
				var priceErr error
				invType := string(inv.Type)
				if strings.ToLower(invType) == constants.InvCrypto {
					price, priceErr = c.priceService.GetCryptoPrice(inv.Symbol, "idr")
				} else {
					price, priceErr = c.priceService.GetStockPrice(inv.Symbol, true)
				}
				if priceErr == nil {
					currentValue := price * inv.Quantity
					totalValue += currentValue
					icon := "📊"
					if strings.ToLower(invType) == constants.InvCrypto {
						icon = "🪙"
					}
					suggestion += fmt.Sprintf("  %s %s: %.4f @ Rp%.0f = Rp%.0f\n",
						icon, strings.ToUpper(inv.Symbol), inv.Quantity, price, currentValue)
				}
			}
			if totalValue > 0 {
				suggestion += fmt.Sprintf("  💵 Total: Rp%.0f\n\n", totalValue)
			}
		}
	}

	page := 1
	msgLower := strings.ToLower(message)
	if strings.Contains(msgLower, "page 2") || strings.Contains(msgLower, "halaman 2") {
		page = 2
	} else if strings.Contains(msgLower, "page 3") || strings.Contains(msgLower, "halaman 3") {
		page = 3
	}

	if c.priceService != nil {
		coins, err := c.priceService.GetTrendingCoins("idr", 20, page)
		if err == nil && len(coins) > 0 {
			suggestion += fmt.Sprintf("🪙 Crypto (Halaman %d):\n", page)
			for _, coin := range coins {
				changeIcon := ""
				if coin.PriceChangePct > 0 {
					changeIcon = "📈"
				} else if coin.PriceChangePct < 0 {
					changeIcon = "📉"
				}
				suggestion += fmt.Sprintf("  %s %s (%.2f%%): Rp%.0f\n",
					changeIcon, strings.ToUpper(coin.Symbol), coin.PriceChangePct, coin.CurrentPrice)
			}
			suggestion += "\n💡 Ketik 'page 2' atau 'halaman 2' untuk lebih banyak\n\n"
		}
	}

	if c.priceService != nil {
		suggestion += "📈 Saham:\n"
		stocks := []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN", "NVDA", "META", "NFLX"}
		stockNames := map[string]string{
			"AAPL":  "Apple",
			"GOOGL": "Google",
			"MSFT":  "Microsoft",
			"TSLA":  "Tesla",
			"AMZN":  "Amazon",
			"NVDA":  "Nvidia",
			"META":  "Meta",
			"NFLX":  "Netflix",
		}

		for _, symbol := range stocks {
			price, err := c.priceService.GetStockPrice(symbol, true)
			if err == nil {
				suggestion += fmt.Sprintf("  %s (%s): Rp%.0f\n", stockNames[symbol], symbol, price)
			}
		}
		suggestion += "\n"
	}

	suggestion += "💰 Disclaimer:\n"
	suggestion += "- Investasi mengandung risiko\n"
	suggestion += "- Lakukan riset sebelum investasi\n"
	suggestion += "- Investasi sesuai kemampuan finansial\n\n"
	suggestion += "Mau tambah investasi? Ketik 'tambah investasi [symbol] [jumlah]'"

	return suggestion
}

func (c *InvestCommand) handleAssetPriceQuery(userID, message string) string {
	if c.priceService == nil {
		return "Layanan pengecekan harga belum aktif."
	}

	msg := strings.ToLower(message)
	symbol := utils.ExtractSymbolFromMessage(msg)

	if symbol == "" {
		return "Tentu! Kamu mau cek harga apa? Sebutkan nama asetnya, misal: 'Harga Bitcoin' atau 'Harga AAPL'."
	}

	invType := constants.InvStock
	if _, ok := services.CryptoSymbolMapping[symbol]; ok {
		invType = constants.InvCrypto
	} else if len(symbol) <= 3 && !strings.ContainsAny(symbol, "0123456789") {
		invType = constants.InvStock
	}

	if strings.Contains(msg, "bitcoin") || strings.Contains(msg, "btc") {
		symbol = "btc"
		invType = constants.InvCrypto
	} else if strings.Contains(msg, "eth") || strings.Contains(msg, "ethereum") {
		symbol = "eth"
		invType = constants.InvCrypto
	}

	price, err := c.priceService.GetPrice(symbol, invType, "idr")
	if err != nil {
		if invType == constants.InvCrypto {
			price, err = c.priceService.GetPrice(symbol, constants.InvStock, "idr")
		} else {
			price, err = c.priceService.GetPrice(symbol, constants.InvCrypto, "idr")
		}

		if err != nil {
			return fmt.Sprintf("Maaf, saya tidak bisa menemukan harga untuk '%s'. Pastikan simbol/namanya benar ya.", symbol)
		}
	}

	assetName := strings.ToUpper(symbol)
	icon := "📈"
	if invType == constants.InvCrypto {
		icon = "🪙"
	}

	summary := fmt.Sprintf("%s **Harga Real-time %s**\n\n", icon, assetName)
	summary += fmt.Sprintf("💰 **Rp%s**\n", utils.FormatNumber(price))
	summary += fmt.Sprintf("🕒 *Update: %s*\n\n", time.Now().Format("15:04:05 WIB"))

	summary += "💡 *Disclaimer: Harga di atas adalah indikasi real-time dari market global. Tetap lakukan riset sebelum berinvestasi.*"

	return summary
}

func (c *InvestCommand) handleAddInvestment(userID, message string) string {
	userOID, _ := primitive.ObjectIDFromHex(userID)
	msg := strings.ToLower(message)

	amount := utils.ParseIndonesianAmount(message)
	if amount <= 0 {
		return "Maaf, saya tidak dapat menentukan jumlah investasi. Contoh: 'tambah investasi BTC 1 juta'"
	}

	symbol := utils.ExtractSymbolFromMessage(msg)
	if symbol == "" {
		// Try to extract common crypto/stock names
		if strings.Contains(msg, "bitcoin") || strings.Contains(msg, "btc") {
			symbol = "BTC"
		} else if strings.Contains(msg, "ethereum") || strings.Contains(msg, "eth") {
			symbol = "ETH"
		} else if strings.Contains(msg, "solana") || strings.Contains(msg, "sol") {
			symbol = "SOL"
		} else {
			return "Maaf, saya tidak dapat menentukan aset investasi. Contoh: 'tambah investasi BTC 1 juta'"
		}
	}

	// Determine investment type
	invType := models.InvStock
	if _, ok := services.CryptoSymbolMapping[symbol]; ok {
		invType = models.InvCrypto
	}

	// Get current price if available
	currentPrice := amount // Default to amount as unit price
	if c.priceService != nil {
		if invType == models.InvCrypto {
			price, err := c.priceService.GetCryptoPrice(symbol, "IDR")
			if err == nil && price > 0 {
				currentPrice = price
			}
		} else {
			price, err := c.priceService.GetStockPrice(symbol, true)
			if err == nil && price > 0 {
				currentPrice = price
			}
		}
	}

	// Calculate quantity based on amount
	quantity := 1.0
	if currentPrice > 0 {
		quantity = amount / currentPrice
	}

	investment := models.Investment{
		UserID:        userOID,
		Name:          symbol,
		Symbol:        strings.ToUpper(symbol),
		Type:          invType,
		Quantity:      quantity,
		AverageCost:   currentPrice,
		CurrentPrice:  currentPrice,
		PurchaseDate:  time.Now(),
		IsActive:      true,
	}

	created, err := c.investmentService.CreateInvestment(investment)
	if err != nil {
		return fmt.Sprintf("❌ Gagal menambahkan investasi: %v", err)
	}

	return fmt.Sprintf("✅ **Investasi Berhasil Ditambahkan!**\n\n📊 Aset: %s (%s)\n💰 Nilai: Rp%.0f\n📈 Jumlah: %.4f unit", created.Name, created.Symbol, amount, created.Quantity)
}
