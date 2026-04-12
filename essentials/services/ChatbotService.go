package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"financeapi/essentials/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ChatbotService handles chatbot interactions
type ChatbotService struct {
	collection                  *mongo.Collection
	transactionService          *TransactionService
	accountService              *AccountService
	investmentService           *InvestmentService
	priceService                *PriceService
	savingsGoalService          *SavingsGoalService
	spendingInsightService      *SpendingInsightService
	billReminderService         *BillReminderService
	debtService                 *DebtService
	recurringTransactionService *RecurringTransactionService
	budgetService               *BudgetService
	budgetCollection            *mongo.Collection
	openAIAPIKey                string
	ollamaURL                   string
	modelName                   string
	maxMemoryMessages           int
}

func NewChatService(client *mongo.Client, dbName string, transactionService *TransactionService) *ChatbotService {
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	modelName := os.Getenv("OLLAMA_MODEL")
	if modelName == "" {
		modelName = "gemma3:4b"
	}

	return &ChatbotService{
		collection:         client.Database(dbName).Collection("chat_history"),
		budgetCollection:   client.Database(dbName).Collection("budgets"),
		transactionService: transactionService,
		openAIAPIKey:       os.Getenv("OPENAI_API_KEY"),
		ollamaURL:          ollamaURL,
		modelName:          modelName,
		maxMemoryMessages:  10,
	}
}

// SetInvestmentService - Inject InvestmentService
func (s *ChatbotService) SetInvestmentService(is *InvestmentService) {
	s.investmentService = is
}

// SetPriceService - Inject PriceService
func (s *ChatbotService) SetPriceService(ps *PriceService) {
	s.priceService = ps
}

// SetSavingsGoalService - Inject SavingsGoalService
func (s *ChatbotService) SetSavingsGoalService(sgs *SavingsGoalService) {
	s.savingsGoalService = sgs
}

// SetSpendingInsightService - Inject SpendingInsightService
func (s *ChatbotService) SetSpendingInsightService(sis *SpendingInsightService) {
	s.spendingInsightService = sis
}

// SetBillReminderService - Inject BillReminderService
func (s *ChatbotService) SetBillReminderService(brs *BillReminderService) {
	s.billReminderService = brs
}

// SetDebtService - Inject DebtService
func (s *ChatbotService) SetDebtService(ds *DebtService) {
	s.debtService = ds
}

// SetRecurringTransactionService - Inject RecurringTransactionService
func (s *ChatbotService) SetRecurringTransactionService(rts *RecurringTransactionService) {
	s.recurringTransactionService = rts
}

// SetAccountService - Inject AccountService
func (s *ChatbotService) SetAccountService(as *AccountService) {
	s.accountService = as
}

// SetBudgetService - Inject BudgetService
func (s *ChatbotService) SetBudgetService(bs *BudgetService) {
	s.budgetService = bs
}

// ProcessMessage - Main entry point
func (s *ChatbotService) ProcessMessage(userID string, message string, sessionID string) (string, error) {
	// Save user message
	userMsg := models.ChatMessage{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		SessionID:   sessionID,
		Message:     message,
		MessageType: "user",
		Timestamp:   time.Now(),
	}
	s.saveMessage(userMsg)

	// Get context
	context := s.GetFinancialContext(userID)

	// Get conversation memory for context
	conversationHistory := s.GetConversationMemory(userID, sessionID)
	context["conversation_history"] = conversationHistory

	// Analyze intent
	needsTool, toolName, toolArgs := s.analyzeIntent(message, context)

	var response string
	if needsTool {
		response = s.executeTool(toolName, toolArgs, userID, context)
	} else {
		// Use Ollama with History for better follow-up capability
		resp, err := s.callOllamaWithHistory(message, context, conversationHistory)
		if err != nil {
			// Try OpenAI as fallback
			resp, err = s.callOpenAI(message, context)
			if err != nil {
				return "", err
			}
		}
		response = resp
	}

	// Save bot response
	botMsg := models.ChatMessage{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		SessionID:   sessionID,
		Message:     message,
		Response:    response,
		MessageType: "assistant",
		Timestamp:   time.Now(),
	}
	s.saveMessage(botMsg)

	return response, nil
}

// analyzeIntent - Detect what user wants with improved specificity
func (s *ChatbotService) analyzeIntent(message string, context map[string]interface{}) (bool, string, map[string]interface{}) {
	msg := strings.ToLower(message)

	// Account - saldo, bank, e-wallet, akun, rekening
	if containsAny(msg, []string{"saldo", "uang di", "bank", "e-wallet", "kartu debit", "kartu kredit", "akun", "rekening", "cash", "tunai"}) {
		return true, "account", map[string]interface{}{}
	}

	// Investment - crypto, bitcoin, portfolio, stock, saham, harga (specific requests only)
	if containsAny(msg, []string{"crypto", "bitcoin", "ethereum", "invest", "portfolio", "investasi", "trading", "saham", "stock", "stocks", "aapl", "googl", "msft", "tsla", "tesla", "apple", "google", "microsoft", "harga", "price"}) {
		return true, "investment", map[string]interface{}{}
	}

	// Investment suggestions/recommendations
	if containsAny(msg, []string{"saran", "recommend", "suggest", "tips", "bagus", "good", "ide"}) {
		return true, "investment", map[string]interface{}{}
	}

	// Debt - hutang, cicilan, pinjaman, credit
	if containsAny(msg, []string{"hutang", "debt", "pinjaman", "kredit", "credit", "cicilan", "loan"}) {
		return true, "debt", map[string]interface{}{"message": message}
	}

	// Transaction - transaksi, pengeluaran, income
	if containsAny(msg, []string{"pengeluaran", "income", "pemasukan", "transaction", "beli", "jual", "belanja", "uang keluar", "uang masuk", "transaksi", "gajian", "gaji", "dapet", "dapat", "earned", "salary", "spent", "beli", "buy", "purchase", "makan", "food", "lunch", "dinner", "breakfast"}) {
		return true, "transaction", map[string]interface{}{"message": message}
	}

	// Savings - tabungan, target, save (after transaction to prioritize adding income)
	if containsAny(msg, []string{"tabungan", "savings", "goal", "target", "menabung", "nabung", "save", "saved", "vacation", "holiday"}) {
		return true, "savings", map[string]interface{}{"message": message}
	}

	// Budget
	if containsAny(msg, []string{"budget", "anggaran", "limit"}) {
		return true, "budget", map[string]interface{}{}
	}

	// Spending Analysis
	if containsAny(msg, []string{"analisa", "analysis", "spending", "pola", "pengeluaran", "total", "cek", "lihat", "berapa", "bulanan", "bulan ini", "bulan lalu"}) {
		return true, "spending", map[string]interface{}{}
	}

	// Bills & Recurring
	if containsAny(msg, []string{"bill", "tagihan", "reminder", "jatuh tempo", "pembayaran", "bulanan"}) {
		return true, "bills", map[string]interface{}{"message": message}
	}

	// Recurring - transaksi berulang
	if containsAny(msg, []string{"recurring", "berulang", "auto", "otomatis", "langganan", "subscription"}) {
		return true, "recurring", map[string]interface{}{}
	}

	// Financial Health
	if containsAny(msg, []string{"health", "kesehatan", "keuangan", "summary", "ringkasan"}) {
		return true, "health", map[string]interface{}{}
	}

	return false, "", nil
}

// executeTool - Execute the appropriate tool
func (s *ChatbotService) executeTool(toolName string, args map[string]interface{}, userID string, context map[string]interface{}) string {
	// Get message from args if available
	msg := ""
	if m, ok := args["message"].(string); ok {
		msg = m
	}

	switch toolName {
	case "account":
		return s.handleAccount(userID, msg)
	case "transaction":
		msgLower := strings.ToLower(msg)
		suggestKeywords := []string{"bulan", "tahun", "hari"}
		if containsAny(msgLower, suggestKeywords) {
			return s.handleSpendingAnalysis(userID, msg)
		}
		return s.handleTransaction(userID, msg)
	case "savings":
		return s.handleSavings(userID, msg)
	case "investment":
		// Check for suggestion/recommendation keywords OR stock price queries
		msgLower := strings.ToLower(msg)
		suggestKeywords := []string{"saran", "recommend", "suggest", "ide", "tips", "bagus", "good", "beli", "buy", "invest", "stocks", "saham", "crypto", "price", "harga", "aapl", "googl", "msft", "tsla", "tesla", "apple", "google", "microsoft"}
		if containsAny(msgLower, suggestKeywords) {
			return s.handleInvestmentRecommendation(userID, msg)
		}
		return s.handleInvestment(userID, msg)
	case "debt":
		return s.handleDebt(userID, msg)
	case "budget":
		return s.handleBudget(userID, msg)
	case "spending":
		return s.handleSpendingAnalysis(userID, msg)
	case "bills":
		return s.handleBills(userID, msg)
	case "recurring":
		return s.handleRecurring(userID, msg)
	case "health":
		return s.handleFinancialHealth(userID, msg)
	default:
		return "Maaf, saya tidak mengerti. Bisa jelaskan lagi?"
	}
}

// Tool handlers
func (s *ChatbotService) handleTransaction(userID, message string) string {
	msg := strings.ToLower(message)

	// Check for expense/spending keywords first - these should trigger adding transaction
	expenseKeywords := []string{"spent", "beli", "buy", "purchase", "makan", "food", "lunch", "dinner", "breakfast", "belanja", "keluar", "bayar", "pay", "pengeluaran", "expense", "transaction"}
	if containsAny(msg, expenseKeywords) {
		return s.handleAddTransaction(userID, message)
	}

	// Check for income/earning keywords
	incomeKeywords := []string{"income", "pemasukan", "gaji", "pendapatan", "uang masuk", "gajian", "dapet", "dapat", "salary", "earned"}
	if containsAny(msg, incomeKeywords) {
		return s.handleAddTransaction(userID, message)
	}

	// Check for explicit add keywords
	if containsAny(msg, []string{"tambah", "add", "input", "catat", "insert"}) {
		return s.handleAddTransaction(userID, message)
	}

	transactions, err := s.transactionService.ShowTransaction(userID)
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

// handleAddTransaction handles adding a new transaction via chatbot
func (s *ChatbotService) handleAddTransaction(userID, message string) string {
	msg := strings.ToLower(message)

	// Parse amount
	amount := parseIndonesianAmount(message)
	if amount <= 0 {
		return "Maaf, saya tidak dapat menentukan jumlah transaksi. Contoh: 'tambah pengeluaran 50000 untuk makan'"
	}

	// Determine transaction type (income or expense)
	// Check for income keywords first
	isIncome := containsAny(msg, []string{"income", "pemasukan", "gaji", "pendapatan", "uang masuk", "gajian", "dapet", "dapat", "salary", "earned", "terima", "duit masuk"})
	// Check for expense keywords
	isExpense := containsAny(msg, []string{"spent", "beli", "buy", "purchase", "makan", "food", "lunch", "dinner", "belanja", "keluar", "bayar", "pay", "pengeluaran", "expense", "transaction", "untuk", "buying"})

	category := "outcome"
	if isIncome && !isExpense {
		category = "income"
	} else {
		// Determine specific expense category
		specificCategory := extractExpenseCategory(msg)
		if specificCategory != "" {
			category = specificCategory
		}
	}

	// Extract description/note from message
	description := extractTransactionDescription(message)

	// Create transaction
	transaction := models.Transaction{
		User_id:     userID,
		Amount:      amount,
		Category:    category,
		Date:        time.Now(),
		Description: description,
	}

	// Call the transaction service to create the transaction
	_, err := s.transactionService.CreateTransaction(transaction)
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

func (s *ChatbotService) handleSavings(userID, message string) string {
	msg := strings.ToLower(message)

	// Check if asking about a specific goal (ada, exist, cek)
	if containsAny(msg, []string{"ada", "exist", "cek", "lihat", "status", "progress", "how", "apa"}) {
		// Try to find specific goal mentioned in message
		goals, err := s.savingsGoalService.GetUserSavingsGoals(userID)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}

		if len(goals) == 0 {
			return "Belum ada tabungan yang kamu buat."
		}

		// Look for specific goal names in the message
		for _, goal := range goals {
			goalNameLower := strings.ToLower(goal.Name)
			// Check if any known goal is mentioned
			if strings.Contains(goalNameLower, "milan") && strings.Contains(msg, "milan") {
				progress := (goal.CurrentAmount / goal.TargetAmount) * 100
				return fmt.Sprintf("✅ Ya! Kamu punya target '%s':\n\n💰 Terkumpul: Rp%.0f\n🎯 Target: Rp%.0f\n📊 Progress: %.0f%%",
					goal.Name, goal.CurrentAmount, goal.TargetAmount, progress)
			}
			if strings.Contains(goalNameLower, "jepang") && strings.Contains(msg, "jepang") {
				progress := (goal.CurrentAmount / goal.TargetAmount) * 100
				return fmt.Sprintf("✅ Ya! Kamu punya target '%s':\n\n💰 Terkumpul: Rp%.0f\n🎯 Target: Rp%.0f\n📊 Progress: %.0f%%",
					goal.Name, goal.CurrentAmount, goal.TargetAmount, progress)
			}
		}

		// If no specific goal found, show all goals
		summary := "🎯 Semua Target Tabungan:\n\n"
		for _, goal := range goals {
			progress := (goal.CurrentAmount / goal.TargetAmount) * 100
			summary += fmt.Sprintf("%s: Rp%.0f / Rp%.0f (%.0f%%)\n", goal.Name, goal.CurrentAmount, goal.TargetAmount, progress)
		}
		return summary
	}

	// Handle adding contribution to savings goal - check BEFORE "create" to prioritize adding
	if containsAny(msg, []string{"tambah", "add", "setor", "nabung", "simpan", "saved", "menabung"}) {
		return s.handleAddSavingsContribution(userID, message)
	}

	if containsAny(msg, []string{"buat", "create", "target baru", "goal baru"}) {
		return "Untuk membuat tabungan baru, sebutkan: nama tabungan dan target jumlah. Contoh: 'buat tabungan mobil 10 juta'"
	}

	goals, err := s.savingsGoalService.GetUserSavingsGoals(userID)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if len(goals) == 0 {
		return "Belum ada tabungan. Mau buat target tabungan?"
	}

	summary := "🎯 Target Tabungan:\n\n"
	for _, goal := range goals {
		progress := (goal.CurrentAmount / goal.TargetAmount) * 100
		summary += fmt.Sprintf("%s: Rp%.0f / Rp%.0f (%.0f%%)\n", goal.Name, goal.CurrentAmount, goal.TargetAmount, progress)
	}
	return summary
}

// handleAddSavingsContribution handles adding money to a savings goal
func (s *ChatbotService) handleAddSavingsContribution(userID, message string) string {
	// Parse amount from message (e.g., "1 juta" -> 1000000, "500 ribu" -> 500000)
	amount := parseIndonesianAmount(message)
	if amount <= 0 {
		return "Maaf, saya tidak dapat menentukan jumlah yang ingin ditabung. Contoh: 'tambah tabungan jepang 1 juta'"
	}

	// Extract goal name from message
	goalName := extractSavingsGoalName(message)
	msgLower := strings.ToLower(message)

	// Get user's savings goals
	goals, err := s.savingsGoalService.GetUserSavingsGoals(userID)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if len(goals) == 0 {
		return "Belum ada tabungan yang kamu buat. Mau buat target tabungan baru?"
	}

	// Find matching goal - improved case-insensitive matching
	var targetGoal models.SavingsGoal
	found := false

	// First, try to find by extracted goal name
	if goalName != "" {
		for _, goal := range goals {
			goalNameLower := strings.ToLower(goal.Name)
			// Check if goal name contains the extracted name OR if extracted name contains goal name
			if strings.Contains(goalNameLower, goalName) || strings.Contains(goalName, strings.Split(goalNameLower, " ")[0]) {
				targetGoal = goal
				found = true
				break
			}
		}
	}

	// If not found, try to match by specific keywords in the message
	if !found {
		for _, goal := range goals {
			goalNameLower := strings.ToLower(goal.Name)
			// Check for specific destination keywords
			if strings.Contains(msgLower, "milan") && strings.Contains(goalNameLower, "milan") {
				targetGoal = goal
				found = true
				break
			}
			if strings.Contains(msgLower, "jepang") && strings.Contains(goalNameLower, "jepang") {
				targetGoal = goal
				found = true
				break
			}
			if strings.Contains(msgLower, "liburan") && strings.Contains(goalNameLower, "liburan") {
				targetGoal = goal
				found = true
				break
			}
		}
	}

	if !found {
		return fmt.Sprintf("Tabungan '%s' tidak ditemukan. Berikut tabungan kamu:\n%s", goalName, s.listSavingsGoals(goals))
	}

	// Convert userID string to ObjectID
	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return "Error: Invalid user ID"
	}

	// Create contribution
	contribution := models.SavingsContribution{
		SavingsGoalID: targetGoal.ID,
		UserID:        userOID,
		Amount:        amount,
		Note:          "Added via chatbot",
	}

	_, err = s.savingsGoalService.AddContribution(contribution)
	if err != nil {
		return fmt.Sprintf("Error menambahkan tabungan: %v", err)
	}

	// Calculate new progress
	newAmount := targetGoal.CurrentAmount + amount
	newProgress := (newAmount / targetGoal.TargetAmount) * 100

	return fmt.Sprintf("✅ Berhasil menambahkan Rp%.0f ke tabungan '%s'!\n\n💰 Total: Rp%.0f / Rp%.0f (%.0f%%)",
		amount, targetGoal.Name, newAmount, targetGoal.TargetAmount, newProgress)
}

// parseIndonesianAmount converts Indonesian number format to float
// Examples: "1 juta" -> 1000000, "500 ribu" -> 500000, "100rb" -> 100000, "1000000" -> 1000000
func parseIndonesianAmount(message string) float64 {
	msg := strings.ToLower(message)

	// Pattern for "X juta" or "X million" - CHECK THIS FIRST
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:juta|jt|million|m)`)
	matches := re.FindStringSubmatch(msg)
	if matches != nil {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val * 1000000
		}
	}

	// Pattern for "X ribu" or "X rb" or "X thousand"
	re = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:ribu|rb|thousand|k)`)
	matches = re.FindStringSubmatch(msg)
	if matches != nil {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val * 1000
		}
	}

	// Pattern for plain numbers last (e.g., "1000000")
	re = regexp.MustCompile(`(\d+)`)
	matches = re.FindStringSubmatch(msg)
	if matches != nil {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val
		}
	}

	return 0
}

// extractSavingsGoalName extracts the savings goal name from the message
func extractSavingsGoalName(message string) string {
	msg := strings.ToLower(message)

	// Common patterns: look for savings goal names in the message
	// Try to find existing goal names first
	patterns := []string{
		`tabungan\s+(.+?)(?:\s+\d+|$)`,
		`nabung\s+(.+?)(?:\s+\d+|$)`,
		`target\s+(.+?)(?:\s+\d+|$)`,
		`untuk\s+(.+?)(?:\s+\d+|$)`,
		`buat\s+(.+?)(?:\s+\d+|$)`,
		// For "liburan ke milan" pattern - capture destination
		`liburan\s+ke\s+(\w+)`,
		// Also try to find multi-word names
		`(?:ke|jepang|milan|liburan|mobil|bayar|hp|laptop)\s*(\w+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(msg)
		if matches != nil && len(matches) > 1 {
			name := strings.TrimSpace(matches[1])
			// Remove common suffixes
			name = strings.TrimSuffix(name, "nya")
			name = strings.TrimSuffix(name, "p") // remove 'p' from "jepang p"
			if len(name) > 1 {
				return name
			}
		}
	}

	// Fallback: look for known goal keywords
	knownGoals := []string{"jepang", "milan", "liburan", "mobil", "hp", "laptop", "rumah"}
	for _, goal := range knownGoals {
		if strings.Contains(msg, goal) {
			return goal
		}
	}

	// If no pattern matches, return empty
	return ""
}

// extractExpenseCategory extracts the expense category from the message
func extractExpenseCategory(message string) string {
	msg := strings.ToLower(message)

	// Map of keywords to categories
	categoryMap := map[string]string{
		// Food
		"makan": "food", "food": "food", "lunch": "food", "dinner": "food", "breakfast": "food", "s breakfast": "food", "s lunch": "food", "s dinner": "food",
		// Transport
		"ojek": "transport", "taxi": "transport", "grab": "transport", "gojek": "transport", "bensin": "transport", "bbm": "transport", "parkir": "transport", "tol": "transport", "transport": "transport", "angkot": "transport", "bus": "transport", "kereta": "transport", "b起飞": "transport",
		// Shopping
		"belanja": "shopping", "beli": "shopping", "shopping": "shopping", "buy": "shopping", "pakaian": "shopping", "baju": "shopping", "sepatu": "shopping", "tas": "shopping",
		// Entertainment
		"nonton": "entertainment", "film": "entertainment", "movie": "entertainment", "bioskop": "entertainment", "konser": "entertainment", "game": "entertainment", "streaming": "entertainment", "netflix": "entertainment",
		// Bills
		"listrik": "bills", "air": "bills", "internet": "bills", "wifi": "bills", "pulsa": "bills", "token": "bills", "tagihan": "bills", "bill": "bills",
		// Health
		"obat": "health", "dokter": "health", "rumah sakit": "health", "rs": "health", "apotek": "health", "medical": "health",
		// Education
		"buku": "education", "kursus": "education", "sekolah": "education", "kuliah": "education", "les": "education", "study": "education", "pelajaran": "education",
		// Other
		"other": "other", "lain": "other", "lainnya": "other",
	}

	for keyword, category := range categoryMap {
		if strings.Contains(msg, keyword) {
			return category
		}
	}

	return ""
}

// extractTransactionDescription extracts description from transaction message
func extractTransactionDescription(message string) string {
	msg := strings.ToLower(message)

	// Common patterns: "untuk [desc]" or "ke [desc]" or "desc: [desc]"
	patterns := []string{
		`untuk\s+(.+)`,
		`ke\s+(.+)`,
		`desc\s*:\s*(.+)`,
		`catatan\s*:\s*(.+)`,
		`note\s*:\s*(.+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(msg)
		if matches != nil && len(matches) > 1 {
			desc := matches[1]
			// Clean up - take first few words if too long
			words := strings.Fields(desc)
			if len(words) > 5 {
				desc = strings.Join(words[:5], " ") + "..."
			}
			return desc
		}
	}

	return "Via Chatbot"
}

// listSavingsGoals returns a formatted list of savings goals
func (s *ChatbotService) listSavingsGoals(goals []models.SavingsGoal) string {
	if len(goals) == 0 {
		return "Belum ada tabungan."
	}

	list := ""
	for _, goal := range goals {
		progress := (goal.CurrentAmount / goal.TargetAmount) * 100
		list += fmt.Sprintf("- %s: Rp%.0f / Rp%.0f (%.0f%%)\n", goal.Name, goal.CurrentAmount, goal.TargetAmount, progress)
	}
	return list
}

func (s *ChatbotService) handleBudget(userID, message string) string {
	if s.budgetService == nil {
		return "Layanan anggaran belum siap. Hubungi admin."
	}

	month := time.Now().Format("2006-01")
	msg := strings.ToLower(message)

	// Determine if user is asking for category budget specifically
	isCategoryQuery := containsAny(msg, []string{"kategori", "category", "per group", "per bagian"})

	summary := fmt.Sprintf("📊 **Analisa Anggaran Kamu (%s)**\n\n", month)

	// 1. Get Monthly Budget Summary
	budgets, err := s.budgetService.GetAllBudgetsWithSpending(userID)
	if err != nil || len(budgets) == 0 {
		return "Kamu belum buat budget nih bulan ini. Mau dibantu buat anggaran pertama?"
	}

	// Find current month's budget
	var currentBudget map[string]interface{}
	for _, b := range budgets {
		if b["month"] == month {
			currentBudget = b
			break
		}
	}

	if currentBudget != nil {
		limit := currentBudget["limit"].(float64)
		spent := currentBudget["spent"].(float64)
		remaining := limit - spent
		percent := (spent / limit) * 100

		statusLabel := "✅ Aman"
		if percent >= 100 {
			statusColor := "🔴"
			statusLabel = "BAHAYA (Over-budget!)"
			summary += fmt.Sprintf("%s **Status: %s**\n", statusColor, statusLabel)
		} else if percent >= 80 {
			statusColor := "🟡"
			statusLabel = "Waspada (Sudah jalan 80%+)"
			summary += fmt.Sprintf("%s **Status: %s**\n", statusColor, statusLabel)
		} else {
			summary += fmt.Sprintf("✅ **Status: %s**\n", statusLabel)
		}

		summary += fmt.Sprintf("💰 Total Limit: Rp%.0f\n", limit)
		summary += fmt.Sprintf("💸 Sudah Terpakai: Rp%.0f (%.1f%%)\n", spent, percent)
		summary += fmt.Sprintf("📥 Sisa Saldo Budget: Rp%.0f\n\n", remaining)
	}

	// 2. Get Detailed Category Budgets if requested or if total is high
	catBudgets, err := s.budgetService.GetAllCategoryBudgetsWithSpending(userID, month)
	if err == nil && len(catBudgets) > 0 {
		summary += "📂 **Detail per Kategori:**\n"
		count := 0
		for _, cb := range catBudgets {
			// Only show current month
			if cb["month"] == month {
				limit := cb["budget_amount"].(float64)
				spent := cb["spent"].(float64)
				catName := cb["category_name"].(string)
				percent := (spent / limit) * 100

				emoji := "🔹"
				if percent >= 100 {
					emoji = "❌"
				} else if percent >= 80 {
					emoji = "⚠️"
				}

				summary += fmt.Sprintf("%s %s: Rp%.0f / Rp%.0f (%.0f%%)\n",
					emoji, catName, spent, limit, percent)
				count++
			}
		}
		if count == 0 {
			summary += "_Belum ada budget kategori yang dibuat._\n"
		}
	}

	if !isCategoryQuery {
		summary += "\n💡 *Tips: Kamu bisa tanya \"Budget kategori\" untuk melihat detail per pos pengeluaran.*"
	}

	return summary
}

func (s *ChatbotService) handleInvestment(userID, message string) string {
	msg := strings.ToLower(message)

	// 1. Detect Price Query (e.g., "Harga Bitcoin", "Price AAPL")
	priceKeywords := []string{"harga", "price", "nilai", "berapa", "asuransi", "saham", "crypto"}
	if containsAny(msg, priceKeywords) && len(strings.Fields(msg)) <= 6 {
		return s.handleAssetPriceQuery(userID, message)
	}

	userOID, _ := primitive.ObjectIDFromHex(userID)
	if s.investmentService == nil {
		return "Layanan investasi belum tersedia."
	}

	investments, err := s.investmentService.GetUserInvestments(userOID)
	if err != nil {
		return fmt.Sprintf("Error mengambil data investasi: %v", err)
	}

	if len(investments) == 0 {
		return s.handleInvestmentRecommendation(userID, message)
	}

	summary := "📈 Portfolio Investasi:\n\n"
	totalValue := 0.0
	totalCost := 0.0

	for _, inv := range investments {
		icon := "📊"
		switch string(inv.Type) {
		case "crypto":
			icon = "🪙"
		case "stock":
			icon = "📈"
		case "bond":
			icon = "📜"
		case "real_estate":
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

func (s *ChatbotService) handleSpendingAnalysis(userID, message string) string {
	// Parse date range from message
	now := time.Now()
	var startDate, endDate time.Time
	var periodName string

	msgLower := strings.ToLower(message)

	// Try to extract relative periods first (e.g., "3 bulan lalu", "2 tahun lalu")
	// Pattern: number + time unit + lalu/kemarin
	numberMonthAgo := regexp.MustCompile(`(\d+)\s*(bulan|month)\s*(lalu|yang lalu|kemarin|ago)`)
	numberYearAgo := regexp.MustCompile(`(\d+)\s*(tahun|year)\s*(lalu|yang lalu|kemarin|ago)`)
	numberWeekAgo := regexp.MustCompile(`(\d+)\s*(minggu|week)\s*(lalu|yang lalu|kemarin|ago)`)

	if match := numberMonthAgo.FindStringSubmatch(msgLower); match != nil {
		if num, err := strconv.Atoi(match[1]); err == nil {
			targetMonth := now.AddDate(0, -num, 0)
			startDate = time.Date(targetMonth.Year(), targetMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
			endDate = startDate.AddDate(0, 1, 0).Add(-time.Second)
			periodName = fmt.Sprintf("%d Bulan Lalu", num)
		}
	} else if match := numberYearAgo.FindStringSubmatch(msgLower); match != nil {
		if num, err := strconv.Atoi(match[1]); err == nil {
			targetYear := now.Year() - num
			startDate = time.Date(targetYear, 1, 1, 0, 0, 0, 0, time.UTC)
			endDate = time.Date(targetYear, 12, 31, 23, 59, 59, 0, time.UTC)
			periodName = fmt.Sprintf("%d Tahun Lalu", num)
		}
	} else if match := numberWeekAgo.FindStringSubmatch(msgLower); match != nil {
		if num, err := strconv.Atoi(match[1]); err == nil {
			targetWeek := now.AddDate(0, 0, -num*7)
			weekday := int(targetWeek.Weekday())
			if weekday == 0 {
				weekday = 7
			}
			startDate = targetWeek.AddDate(0, 0, -(weekday - 1))
			startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
			endDate = startDate.AddDate(0, 0, 6).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			periodName = fmt.Sprintf("%d Minggu Lalu", num)
		}
	} else {
		// Try to parse specific month names
		monthNames := map[string]time.Month{
			"januari":   time.January,
			"februari":  time.February,
			"maret":     time.March,
			"april":     time.April,
			"mei":       time.May,
			"juni":      time.June,
			"juli":      time.July,
			"agustus":   time.August,
			"september": time.September,
			"oktober":   time.October,
			"november":  time.November,
			"desember":  time.December,
		}

		// Check for "bulan [nama bulan]" or "bulan [angka]"
		for monthName, monthVal := range monthNames {
			if strings.Contains(msgLower, "bulan "+monthName) || strings.Contains(msgLower, "bulan "+strconv.Itoa(int(monthVal))) {
				startDate = time.Date(now.Year(), monthVal, 1, 0, 0, 0, 0, time.UTC)
				endDate = startDate.AddDate(0, 1, 0).Add(-time.Second)
				periodName = startDate.Format("January 2006")
				break
			}
		}

		// If no specific month found, check for relative periods
		if startDate.IsZero() {
			switch {
			case containsAny(msgLower, []string{"bulan lalu", "last month"}):
				lastMonth := now.AddDate(0, -1, 0)
				startDate = time.Date(lastMonth.Year(), lastMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
				endDate = startDate.AddDate(0, 1, 0).Add(-time.Second)
				periodName = lastMonth.Format("January 2006")
			case containsAny(msgLower, []string{"bulan ini", "this month", "bulan sekarang"}):
				startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
				endDate = startDate.AddDate(0, 1, 0).Add(-time.Second)
				periodName = now.Format("January 2006")
			case containsAny(msgLower, []string{"minggu ini", "this week"}):
				weekday := int(now.Weekday())
				if weekday == 0 {
					weekday = 7
				}
				startDate = now.AddDate(0, 0, -(weekday - 1))
				startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
				endDate = startDate.AddDate(0, 0, 6).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
				periodName = "Minggu Ini"
			case containsAny(msgLower, []string{"minggu lalu", "last week"}):
				weekday := int(now.Weekday())
				if weekday == 0 {
					weekday = 7
				}
				lastWeekStart := now.AddDate(0, 0, -(weekday-1)-7)
				startDate = time.Date(lastWeekStart.Year(), lastWeekStart.Month(), lastWeekStart.Day(), 0, 0, 0, 0, time.UTC)
				endDate = startDate.AddDate(0, 0, 6).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
				periodName = "Minggu Lalu"
			case containsAny(msgLower, []string{"tahun ini", "this year"}):
				startDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
				endDate = time.Date(now.Year(), 12, 31, 23, 59, 59, 0, time.UTC)
				periodName = fmt.Sprintf("Tahun %d", now.Year())
			case containsAny(msgLower, []string{"tahun lalu", "last year"}):
				lastYear := now.Year() - 1
				startDate = time.Date(lastYear, 1, 1, 0, 0, 0, 0, time.UTC)
				endDate = time.Date(lastYear, 12, 31, 23, 59, 59, 0, time.UTC)
				periodName = fmt.Sprintf("Tahun %d", lastYear)
			case containsAny(msgLower, []string{"kemarin", "semalam", "yesterday"}):
				yesterday := now.AddDate(0, 0, -1)
				startDate = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.UTC)
				endDate = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 0, time.UTC)
				periodName = "Kemarin"
			case containsAny(msgLower, []string{"hari ini", "today"}):
				startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
				endDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC)
				periodName = "Hari Ini"
			default:
				// Default to current month
				startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
				endDate = startDate.AddDate(0, 1, 0).Add(-time.Second)
				periodName = now.Format("January 2006")
			}
		}
	}

	filter := models.FilterTransaction{
		FromDate: startDate.Format("2006-01-02"),
		ToDate:   endDate.Format("2006-01-02"),
	}

	report, err := s.transactionService.GetReport(userID, filter)
	if err != nil {
		return "❌ Gagal mengambil data pengeluaran: " + err.Error()
	}

	// Build response
	outcome := report["outcome"]
	income := report["income"]
	net := report["net"]

	response := fmt.Sprintf("📊 Laporan Pengeluaran %s\n\n", periodName)
	response += fmt.Sprintf("💰 Pemasukan: Rp %.0f\n", income)
	response += fmt.Sprintf("💸 Pengeluaran: Rp %.0f\n", outcome)
	response += fmt.Sprintf("📈 Sisa: Rp %.0f\n\n", net)

	if outcome > 0 && income > 0 {
		savingsRate := ((income - outcome) / income) * 100
		if savingsRate > 20 {
			response += "✅ Kondisi keuangan baik! Tabungan > 20%%"
		} else if savingsRate > 0 {
			response += "⚠️ Coba lebih hemat! Tabungan < 20%%"
		} else {
			response += "❌ Pengeluaran melebihi pemasukan!"
		}
	} else if outcome == 0 {
		response += "💡 Belum ada pengeluaran"
	}

	return response
}

// handleInvestmentRecommendation provides investment suggestions based on current prices
func (s *ChatbotService) handleInvestmentRecommendation(userID, message string) string {
	suggestion := "💰 Harga Investasi Terkini:\n\n"

	// Convert userID to ObjectID
	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return "Error: Invalid user ID"
	}

	// Try to get user's portfolio first
	if s.investmentService != nil {
		investments, err := s.investmentService.GetUserInvestments(userOID)
		if err == nil && len(investments) > 0 {
			suggestion += "📊 Portfolio Kamu:\n"
			totalValue := 0.0
			for _, inv := range investments {
				var price float64
				var priceErr error
				invType := string(inv.Type)
				if strings.ToLower(invType) == "crypto" {
					price, priceErr = s.priceService.GetCryptoPrice(inv.Symbol, "idr")
				} else {
					price, priceErr = s.priceService.GetStockPrice(inv.Symbol, true)
				}
				if priceErr == nil {
					currentValue := price * inv.Quantity
					totalValue += currentValue
					icon := "📊"
					invType := string(inv.Type)
					if strings.ToLower(invType) == "crypto" {
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

	// Get page from message if user wants specific page
	page := 1
	msgLower := strings.ToLower(message)
	if strings.Contains(msgLower, "page 2") || strings.Contains(msgLower, "halaman 2") {
		page = 2
	} else if strings.Contains(msgLower, "page 3") || strings.Contains(msgLower, "halaman 3") {
		page = 3
	}

	// Get crypto prices with pagination from CoinGecko
	if s.priceService != nil {
		coins, err := s.priceService.GetTrendingCoins("idr", 20, page)
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

	// Get stock prices if available
	if s.priceService != nil {
		suggestion += "📈 Saham:\n"

		// Popular stocks
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
			price, err := s.priceService.GetStockPrice(symbol, true)
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

func (s *ChatbotService) handleBills(userID, message string) string {
	msg := strings.ToLower(message)

	// 1. Detect creation intent
	creationKeywords := []string{"ada", "tambah", "catat", "buat", "punya", "baru", "jatuh tempo"}
	amount := parseIndonesianAmount(message)
	if containsAny(msg, creationKeywords) && amount > 0 {
		return s.handleAddBill(userID, message)
	}

	userOID, _ := primitive.ObjectIDFromHex(userID)
	if s.billReminderService == nil {
		return "Layanan tagihan belum tersedia."
	}

	bills, err := s.billReminderService.GetUserBillReminders(userOID)
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

	// Handle Overdue
	if len(overdue) > 0 {
		summary += "🚨 **GAWAT! Tagihan ini sudah lewat tempo:**\n"
		for _, b := range overdue {
			summary += fmt.Sprintf("- %s (Rp%.0f) - Segera bayar ya!\n", b.Name, b.Amount)
		}
		summary += "\n"
	}

	// Handle Due Soon
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

	// Handle Upcoming
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

func (s *ChatbotService) handleAddBill(userID, message string) string {
	msg := strings.ToLower(message)
	userOID, _ := primitive.ObjectIDFromHex(userID)

	amount := parseIndonesianAmount(message)
	name := extractBillNameFromMessage(msg)
	if name == "" || name == "tagihan" || name == "bill" {
		name = "Tagihan Baru"
	}

	category := extractExpenseCategory(msg)
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

	created, err := s.billReminderService.CreateBillReminder(bill)
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

func extractBillNameFromMessage(msg string) string {
	removables := []string{"tagihan", "bill", "ada", "tambah", "catat", "buat", "punya", "baru", "jatuh", "tempo", "dalam", "bulan", "minggu", "hari", "sebesar", "nominalny", "nominal", "rp", "juta", "ribu", "rb", "jt"}
	cleaned := msg
	for _, r := range removables {
		cleaned = strings.ReplaceAll(cleaned, r, "")
	}
	re := regexp.MustCompile(`\d+`)
	cleaned = re.ReplaceAllString(cleaned, "")
	words := strings.Fields(cleaned)
	if len(words) > 0 {
		return strings.Join(words, " ")
	}
	return ""
}

// handleRecurring handles recurring transactions
func (s *ChatbotService) handleRecurring(userID, message string) string {
	userOID, _ := primitive.ObjectIDFromHex(userID)
	if s.recurringTransactionService == nil {
		return "Layanan transaksi rutin belum tersedia."
	}

	recurrings, err := s.recurringTransactionService.GetActiveRecurringTransactions(userOID)
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

func (s *ChatbotService) handleDebt(userID, message string) string {
	msg := strings.ToLower(message)

	// 1. Detect payment intent
	paymentKeywords := []string{"bayar", "pay", "cicil", "setor", "bayarin"}
	if containsAny(msg, paymentKeywords) {
		return s.handleDebtPayment(userID, message)
	}

	// 2. Detect creation intent (e.g., "aku ada hutang...", "tambah cicilan...")
	creationKeywords := []string{"ada", "tambah", "catat", "buat", "punya", "mempunyai", "baru"}
	amount := parseIndonesianAmount(message)
	if (containsAny(msg, creationKeywords) && amount > 0) || (amount > 0 && containsAny(msg, []string{"bunga", "tenor", "bulan"})) {
		return s.handleAddDebt(userID, message)
	}

	userOID, _ := primitive.ObjectIDFromHex(userID)
	if s.debtService == nil {
		return "Layanan hutang belum tersedia."
	}

	debts, err := s.debtService.GetUserDebts(userOID)
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

func (s *ChatbotService) handleAddDebt(userID, message string) string {
	msg := strings.ToLower(message)
	userOID, _ := primitive.ObjectIDFromHex(userID)

	amount := parseIndonesianAmount(message)

	// Create a "name-only" string by removing the amount part
	amountStr := extractAmountString(msg)
	msgWithoutAmount := msg
	if amountStr != "" {
		msgWithoutAmount = strings.Replace(msg, amountStr, "", 1)
	}

	interest := extractInterestRate(msg)
	tenor := extractTenor(msg)
	name := extractDebtNameFromMessage(msgWithoutAmount)
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

	// Calculate payment amount if tenor and interest are present
	if tenor > 0 {
		// Simple interest calculation for display / basic tracking
		totalWithInterest := amount * (1 + (interest/100)*float64(tenor))
		debt.PaymentAmount = totalWithInterest / float64(tenor)
	}

	created, err := s.debtService.CreateDebt(debt)
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

func extractInterestRate(msg string) float64 {
	// Support: "1.2%", "1.2 persen", "1.2 percent"
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:%|persen|percent)`)
	matches := re.FindStringSubmatch(msg)
	if len(matches) > 1 {
		val, _ := strconv.ParseFloat(matches[1], 64)
		return val
	}
	return 0
}

func extractTenor(msg string) int {
	// Support: "12 bulan", "1 tahun" (auto x12), "tenor 24"

	// Check for years first
	reYear := regexp.MustCompile(`(\d+)\s*(?:tahun|year|thn|yr)`)
	matchesYear := reYear.FindStringSubmatch(msg)
	if len(matchesYear) > 1 {
		val, _ := strconv.Atoi(matchesYear[1])
		return val * 12
	}

	// Check for months
	reMonth := regexp.MustCompile(`(\d+)\s*(?:bulan|month|bln|mo|tenor)`)
	matchesMonth := reMonth.FindStringSubmatch(msg)
	if len(matchesMonth) > 1 {
		val, _ := strconv.Atoi(matchesMonth[1])
		return val
	}
	return 0
}

func (s *ChatbotService) handleDebtPayment(userID, message string) string {
	userOID, _ := primitive.ObjectIDFromHex(userID)
	msg := strings.ToLower(message)

	// 1. Parse amount and create a version of message without that amount
	amount := parseIndonesianAmount(message)
	if amount <= 0 {
		return "⚠️ **Jumlah Tidak Valid.** Sebutkan nominalnya ya, contoh: 'Bayar KPR 2 juta pake BCA'."
	}

	// Create a "name-only" string by removing the amount part to avoid digits-stripping issues
	// e.g., "iphone 15 pro 3 juta" -> "iphone 15 pro"
	amountStr := extractAmountString(msg)
	msgWithoutAmount := msg
	if amountStr != "" {
		msgWithoutAmount = strings.Replace(msg, amountStr, "", 1)
	}

	// 2. Extract Names
	debtNameQuery := extractDebtNameFromMessage(msgWithoutAmount)
	accountNameQuery := extractAccountNameFromMessage(msg)

	if debtNameQuery == "" {
		return "🤔 **Hutang yang mana?** Sebutkan nama hutangnya, misal: 'Bayar **Laptop** 500rb'."
	}

	// 3. Find the Best Matching Debt
	debts, _ := s.debtService.GetUserDebts(userOID)
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

	// 4. Find the Account & Check Balance
	var accountID primitive.ObjectID
	var targetAccount *models.Account
	if accountNameQuery != "" && s.accountService != nil {
		accounts, _ := s.accountService.GetUserAccounts(userOID)
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

	// 5. Atomic Payment
	_, err := s.debtService.MakePayment(targetDebt.ID, userOID, accountID, amount)
	if err != nil {
		return fmt.Sprintf("💥 **Gagal memproses pembayaran:** %v", err)
	}

	// 6. Response
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

// handleAccount handles account/saldo queries
func (s *ChatbotService) handleAccount(userID, message string) string {
	// Convert userID string to ObjectID
	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return "Error: Invalid user ID"
	}

	if s.accountService == nil {
		return "Account service belum tersedia."
	}

	accounts, err := s.accountService.GetUserAccounts(userOID)
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
			balance = -account.CurrentBalance // Credit cards show as negative
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

func (s *ChatbotService) handleFinancialHealth(userID, message string) string {
	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return "Error: Invalid user ID"
	}

	summary := "🩺 **Laporan Kesehatan Keuangan Kamu**\n\n"

	// 1. Get Real Health Score
	if s.spendingInsightService != nil {
		healthScore, err := s.spendingInsightService.GetFinancialHealthScore(userOID)
		if err == nil {
			statusColor := "🟢" // Sehat
			if healthScore.OverallScore < 50 {
				statusColor = "🔴" // Bahaya
			} else if healthScore.OverallScore < 75 {
				statusColor = "🟡" // Waspada
			}

			summary += fmt.Sprintf("%s **Skor Keseluruhan: %d/100**\n", statusColor, healthScore.OverallScore)
			summary += fmt.Sprintf("📊 Tabungan: %d%% | Hutang: %d%% | Kontrol: %d%%\n\n",
				healthScore.SavingsRate, healthScore.DebtLevel, healthScore.ExpenseControl)

			if len(healthScore.Recommendations) > 0 {
				summary += "**Saran Utama:**\n"
				for i, rec := range healthScore.Recommendations {
					if i >= 3 {
						break
					}
					summary += fmt.Sprintf("💡 %s\n", rec)
				}
				summary += "\n"
			}
		}
	}

	// 2. Add Transaction Stats
	transactions, _ := s.transactionService.ShowTransaction(userID)
	var totalIncome, totalExpense float64
	for _, tx := range transactions {
		if tx.Category == "income" || tx.Category == "pemasukan" {
			totalIncome += tx.Amount
		} else {
			totalExpense += tx.Amount
		}
	}
	summary += fmt.Sprintf("💰 **Ringkasan Bulan Ini:**\n- Pemasukan: Rp%.0f\n- Pengeluaran: Rp%.0f\n- Saldo: Rp%.0f\n\n",
		totalIncome, totalExpense, totalIncome-totalExpense)

	// 3. Show Recent Unread Insights & Mark as Read
	if s.spendingInsightService != nil {
		insights, err := s.spendingInsightService.GetUserInsights(userOID, 3)
		if err == nil && len(insights) > 0 {
			hasInsights := false
			for _, insight := range insights {
				if !insight.IsRead {
					if !hasInsights {
						summary += "🔔 **Insight Terbaru:**\n"
						hasInsights = true
					}
					summary += fmt.Sprintf("- **%s**: %s\n", insight.Title, insight.Description)
					// Mark as read in background
					go s.spendingInsightService.MarkAsRead(insight.ID, userOID)
				}
			}
		}
	}

	return summary
}

// callOllama - Call Ollama local AI
func (s *ChatbotService) callOllama(message string, context map[string]interface{}) (string, error) {
	// Build system prompt with context - FORCE English language
	systemPrompt := `You are a helpful personal finance assistant. 
IMPORTANT: Always respond in the SAME LANGUAGE as the user uses. If user writes in Indonesian, respond in Indonesian. If user writes in English, respond in English.

You help users manage their personal finances including:
- Tracking income and expenses
- Managing savings goals
- Monitoring investments
- Tracking bills and debts
- Financial planning

Be concise, friendly, and practical in your responses.`

	// Add financial context if available
	if income, ok := context["totalIncome"].(float64); ok {
		systemPrompt += fmt.Sprintf("\nUser's total income: Rp%.0f", income)
	}
	if expense, ok := context["totalExpense"].(float64); ok {
		systemPrompt += fmt.Sprintf("\nUser's total expenses: Rp%.0f", expense)
	}

	url := s.ollamaURL + "/api/chat"

	reqBody := map[string]interface{}{
		"model": s.modelName,
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role":    "user",
				"content": message,
			},
		},
		"stream": false,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Ollama connection error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Ollama error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if msg, ok := result["message"].(map[string]interface{}); ok {
		if content, ok := msg["content"].(string); ok {
			return content, nil
		}
	}

	return "Maaf, ada masalah dengan respons AI.", nil
}

// callOllamaWithHistory - Call Ollama with conversation history
func (s *ChatbotService) callOllamaWithHistory(message string, context map[string]interface{}, history []models.ChatMessage) (string, error) {
	// Build system prompt
	systemPrompt := `You are a helpful personal finance assistant. 
IMPORTANT: Always respond in the SAME LANGUAGE as the user uses. If user writes in Indonesian, respond in Indonesian. If user writes in English, respond in English.

You help users manage their personal finances including:
- Tracking income and expenses
- Managing savings goals
- Monitoring investments
- Tracking bills and debts
- Financial planning

IMPORTANT: You have access to conversation history. When user asks follow-up questions like "which one?", "what about...?", "cheaper?", use the conversation history to understand what they're referring to.

Be concise, friendly, and practical in your responses.`

	// Add financial context
	if income, ok := context["totalIncome"].(float64); ok {
		systemPrompt += fmt.Sprintf("\nUser's total income: Rp%.0f", income)
	}
	if expense, ok := context["totalExpense"].(float64); ok {
		systemPrompt += fmt.Sprintf("\nUser's total expenses: Rp%.0f", expense)
	}

	// Build messages array with history
	var messages []map[string]interface{}

	// Add system prompt
	messages = append(messages, map[string]interface{}{
		"role":    "system",
		"content": systemPrompt,
	})

	// Add conversation history (chronological order)
	for _, h := range history {
		if h.MessageType == "user" {
			messages = append(messages, map[string]interface{}{
				"role":    "user",
				"content": h.Message,
			})
		} else if h.MessageType == "assistant" && h.Response != "" {
			messages = append(messages, map[string]interface{}{
				"role":    "assistant",
				"content": h.Response,
			})
		}
	}

	// Add current message
	messages = append(messages, map[string]interface{}{
		"role":    "user",
		"content": message,
	})

	url := s.ollamaURL + "/api/chat"

	reqBody := map[string]interface{}{
		"model":    s.modelName,
		"messages": messages,
		"stream":   false,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Ollama connection error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Ollama error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if msg, ok := result["message"].(map[string]interface{}); ok {
		if content, ok := msg["content"].(string); ok {
			return content, nil
		}
	}

	return "Maaf, ada masalah dengan respons AI.", nil
}

// callOpenAI - Call OpenAI API (fallback)
func (s *ChatbotService) callOpenAI(message string, context map[string]interface{}) (string, error) {
	if s.openAIAPIKey == "" {
		return "", fmt.Errorf("OpenAI API key not configured")
	}

	systemPrompt := "You are a helpful personal finance assistant. Respond in the same language as the user."
	if income, ok := context["totalIncome"].(float64); ok {
		systemPrompt += fmt.Sprintf("\nUser's total income: Rp%.0f", income)
	}
	if expense, ok := context["totalExpense"].(float64); ok {
		systemPrompt += fmt.Sprintf("\nUser's total expenses: Rp%.0f", expense)
	}

	url := "https://api.openai.com/v1/chat/completions"

	reqBody := map[string]interface{}{
		"model":      "gpt-4o-mini",
		"max_tokens": 500,
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role":    "user",
				"content": message,
			},
		},
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.openAIAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if c, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := c["message"].(map[string]interface{}); ok {
				if content, ok := msg["content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	return "Maaf, ada masalah dengan respons AI.", nil
}

// GetFinancialContext - Get user's financial data enriched with budget, bills, and savings
func (s *ChatbotService) GetFinancialContext(userID string) map[string]interface{} {
	context := make(map[string]interface{})
	userOID, _ := primitive.ObjectIDFromHex(userID)

	// 1. Transaction Summary
	if s.transactionService != nil {
		transactions, _ := s.transactionService.ShowTransaction(userID)
		var income, expense float64
		for _, tx := range transactions {
			if tx.Category == "income" || tx.Category == "pemasukan" {
				income += tx.Amount
			} else {
				expense += tx.Amount
			}
		}
		context["totalIncome"] = income
		context["totalExpense"] = expense
		context["balance"] = income - expense
		context["transactionCount"] = len(transactions)
	}

	// 2. Budget Context
	if s.budgetService != nil {
		month := time.Now().Format("2006-01")
		budgets, _ := s.budgetService.GetAllBudgetsWithSpending(userID)
		for _, b := range budgets {
			if b["month"] == month {
				context["currentBudgetLimit"] = b["limit"]
				context["currentBudgetSpent"] = b["spent"]
				context["budgetOverLimit"] = b["spent"].(float64) > b["limit"].(float64)
				break
			}
		}
	}

	// 3. Upcoming Bills Context
	if s.billReminderService != nil {
		bills, _ := s.billReminderService.GetUserBillReminders(userOID)
		var upcomingBills []string
		now := time.Now()
		for _, b := range bills {
			if !b.IsPaid && b.NextDueDate.Before(now.AddDate(0, 0, 7)) {
				upcomingBills = append(upcomingBills, fmt.Sprintf("%s (Rp%.0f on %s)", b.Name, b.Amount, b.NextDueDate.Format("02 Jan")))
			}
		}
		context["nearTermBills"] = upcomingBills
	}

	// 4. Savings Context
	if s.savingsGoalService != nil {
		goals, _ := s.savingsGoalService.GetUserSavingsGoals(userID)
		var savingsSummary []string
		for _, g := range goals {
			progress := (g.CurrentAmount / g.TargetAmount) * 100
			savingsSummary = append(savingsSummary, fmt.Sprintf("%s: %.1f%% complete", g.Name, progress))
		}
		context["savingsProgress"] = savingsSummary
	}

	return context
}

// saveMessage - Save chat message to database
func (s *ChatbotService) saveMessage(msg models.ChatMessage) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.collection.InsertOne(ctx, msg)
	}()
}

// GetConversationMemory - Get conversation history
func (s *ChatbotService) GetConversationMemory(userID, sessionID string) []models.ChatMessage {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, _ := s.collection.Find(ctx, bson.M{
		"user_id":    userID,
		"session_id": sessionID,
	}, options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}).SetLimit(int64(s.maxMemoryMessages)))

	var messages []models.ChatMessage
	cursor.All(ctx, &messages)
	return messages
}

// ClearConversationMemory - Clear conversation history
func (s *ChatbotService) ClearConversationMemory(userID, sessionID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.collection.DeleteMany(ctx, bson.M{
		"user_id":    userID,
		"session_id": sessionID,
	})
	return err
}

// GetChatHistory - Get full chat history
func (s *ChatbotService) GetChatHistory(userID, sessionID string) ([]models.ChatMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := s.collection.Find(ctx, bson.M{
		"user_id":    userID,
		"session_id": sessionID,
	}, options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}}))

	if err != nil {
		return nil, err
	}

	var messages []models.ChatMessage
	err = cursor.All(ctx, &messages)
	return messages, err
}

// Helper functions
func containsAny(s string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

// Helper extraction functions for Debt
func extractDebtNameFromMessage(msg string) string {
	// Clean typical command and filler words
	removables := []string{
		"bayar", "bayarin", "cicil", "cicilan", "pake", "pakai", "pakek", "menggunakan",
		"jumlah", "untuk", "sebesar", "rp", "juta", "ribu", "rb", "jt", "nominalnya",
		"bayarkan", "ada", "baru", "tambah", "catat", "buat", "punya", "bunga",
		"persen", "percent", "tenor", "bulan", "tahun", "aku", "mau", "saya", "ingin",
		"hutang", "hutangku", "tagihan", "dong", "nih", "ya", "sip", "oke", "tolong",
	}

	cleaned := msg
	for _, r := range removables {
		// Use regex to replace whole words only to avoid stripping parts of names (e.g., "bank")
		re := regexp.MustCompile(`(?i)\b` + r + `\b`)
		cleaned = re.ReplaceAllString(cleaned, "")
	}

	// Remove any remaining percentage signs but keep digits (they might be part of an asset name like "iPhone 15")
	rePct := regexp.MustCompile(`\d+(?:\.\d+)?\s*%`)
	cleaned = rePct.ReplaceAllString(cleaned, "")

	words := strings.Fields(cleaned)
	if len(words) > 0 {
		// Join up to 3 words for richer entity names (e.g., "Bank Mandiri KPR")
		limit := 3
		if len(words) < limit {
			limit = len(words)
		}
		return strings.Join(words[:limit], " ")
	}
	return ""
}

func extractAccountNameFromMessage(msg string) string {
	// Pattern like "pake [account]" or "pakai [account]" or "dari [account]"
	keywords := []string{"pake", "pakai", "pakek", "dari", "rekening", "akun"}
	for _, kw := range keywords {
		if strings.Contains(msg, kw) {
			parts := strings.Split(msg, kw)
			if len(parts) >= 2 {
				words := strings.Fields(parts[1])
				if len(words) > 0 {
					return words[0]
				}
			}
		}
	}
	return ""
}

func (s *ChatbotService) handleAssetPriceQuery(userID, message string) string {
	if s.priceService == nil {
		return "Layanan pengecekan harga belum aktif."
	}

	msg := strings.ToLower(message)
	symbol := extractSymbolFromMessage(msg)

	if symbol == "" {
		return "Tentu! Kamu mau cek harga apa? Sebutkan nama asetnya, misal: 'Harga Bitcoin' atau 'Harga AAPL'."
	}

	// Try to determine type (crypto or stock)
	// Simple heuristic: check crypto mapping first
	invType := "stock"
	if _, ok := CryptoSymbolMapping[symbol]; ok {
		invType = "crypto"
	} else if len(symbol) <= 3 && !strings.ContainsAny(symbol, "0123456789") {
		// Common stocks are 3-4 letters
		invType = "stock"
	}

	// Special cases for common names
	if strings.Contains(msg, "bitcoin") || strings.Contains(msg, "btc") {
		symbol = "btc"
		invType = "crypto"
	} else if strings.Contains(msg, "eth") || strings.Contains(msg, "ethereum") {
		symbol = "eth"
		invType = "crypto"
	}

	price, err := s.priceService.GetPrice(symbol, invType, "idr")
	if err != nil {
		// If failed as crypto, try as stock
		if invType == "crypto" {
			price, err = s.priceService.GetPrice(symbol, "stock", "idr")
		} else {
			price, err = s.priceService.GetPrice(symbol, "crypto", "idr")
		}

		if err != nil {
			return fmt.Sprintf("Maaf, saya tidak bisa menemukan harga untuk '%s'. Pastikan simbol/namanya benar ya.", symbol)
		}
	}

	assetName := strings.ToUpper(symbol)
	icon := "📈"
	if invType == "crypto" {
		icon = "🪙"
	}

	summary := fmt.Sprintf("%s **Harga Real-time %s**\n\n", icon, assetName)
	summary += fmt.Sprintf("💰 **Rp%s**\n", formatNumber(price))
	summary += fmt.Sprintf("🕒 *Update: %s*\n\n", time.Now().Format("15:04:05 WIB"))

	summary += "💡 *Disclaimer: Harga di atas adalah indikasi real-time dari market global. Tetap lakukan riset sebelum berinvestasi.*"

	return summary
}

func extractSymbolFromMessage(msg string) string {
	removables := []string{"harga", "berapa", "price", "nilai", "saat", "ini", "sekarang", "cek", "dong", "saham", "crypto"}
	cleaned := msg
	for _, r := range removables {
		cleaned = strings.ReplaceAll(cleaned, r, "")
	}
	words := strings.Fields(cleaned)
	if len(words) > 0 {
		return words[len(words)-1] // Often the symbol is the last word
	}
	return ""
}

func formatNumber(val float64) string {
	if val >= 1000 {
		return fmt.Sprintf("%.0f", val)
	}
	return fmt.Sprintf("%.2f", val)
}

func extractAmountString(msg string) string {
	// Pattern for "X juta" or "X million"
	re := regexp.MustCompile(`(\d+(?:\.\d+)?\s*(?:juta|jt|million|m|ribu|rb|thousand|k))`)
	if match := re.FindString(msg); match != "" {
		return match
	}

	// Pattern for plain numbers
	reNumeric := regexp.MustCompile(`(\d{4,})`) // 4 digits or more usually an amount
	return reNumeric.FindString(msg)
}
