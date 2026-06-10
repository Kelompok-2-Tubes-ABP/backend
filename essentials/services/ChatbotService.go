package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"financeapi/essentials/models"
	"financeapi/essentials/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ChatbotCommand defines the interface for separate chatbot intent handlers
type ChatbotCommand interface {
	Handle(userID string, message string) string
}

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

	commands map[string]ChatbotCommand
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

// RegisterCommand attaches a new command handler to the chatbot
func (s *ChatbotService) RegisterCommand(intentName string, cmd ChatbotCommand) {
	if s.commands == nil {
		s.commands = make(map[string]ChatbotCommand)
	}
	s.commands[intentName] = cmd
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

	// === PRIORITY 1: Explicit action words (buat, tambah, hapus, edit) ===
	// Check for create/intent FIRST to avoid misclassification

	// Budget creation - "buat budget", "buatkan budget", "mau budget"
	if utils.ContainsAny(msg, []string{"buat budget", "buatkan budget", "mau budget", "ingin budget", "butuh budget"}) {
		return true, "budget", map[string]interface{}{"message": message}
	}

	// Budget edit/update - "edit budget", "ubah budget", "update budget", "ganti budget"
	if utils.ContainsAny(msg, []string{"edit budget", "ubah budget", "update budget", "ganti budget", "rubah budget"}) {
		return true, "budget", map[string]interface{}{"message": message}
	}

	// Budget delete - "hapus budget", "delete budget"
	if utils.ContainsAny(msg, []string{"hapus budget", "delete budget", "hapus anggaran"}) {
		return true, "budget", map[string]interface{}{"message": message}
	}

	// Account creation - "tambah akun", "buka akun", "daftar akun", "buat akun"
	if utils.ContainsAny(msg, []string{"tambah akun", "buka akun", "daftar akun", "buat akun", "register akun", "bikin akun", "buat rekening", "tambah rekening", "buka rekening"}) {
		return true, "account", map[string]interface{}{"message": message}
	}

	// Account edit/update - "edit akun", "ubah akun", "update saldo"
	if utils.ContainsAny(msg, []string{"edit akun", "ubah akun", "update akun", "ganti akun", "edit rekening", "topup", "tarik"}) {
		return true, "account", map[string]interface{}{"message": message}
	}

	// Account delete - "hapus akun", "delete akun"
	if utils.ContainsAny(msg, []string{"hapus akun", "delete akun", "hapus rekening"}) {
		return true, "account", map[string]interface{}{"message": message}
	}

	// Recurring/subscription creation - "tambah langganan", "buatkan langganan", "langganan baru"
	if utils.ContainsAny(msg, []string{"tambah langganan", "buatkan langganan", "langganan baru", "subscription baru", "buatkan subscription", "daftarin langganan"}) {
		return true, "recurring", map[string]interface{}{"message": message}
	}

	// Recurring edit/delete - "edit langganan", "hapus langganan", "pause subscription"
	if utils.ContainsAny(msg, []string{"edit langganan", "ubah langganan", "hapus langganan", "delete langganan", "pause langganan", "batal langganan"}) {
		return true, "recurring", map[string]interface{}{"message": message}
	}

	// Savings creation - "buat tabungan", "buatkan tabungan", "target baru", "goal baru"
	if utils.ContainsAny(msg, []string{"buat tabungan", "buatkan tabungan", "target baru", "goal baru", "tabungan baru", "target tabungan"}) {
		return true, "savings", map[string]interface{}{"message": message}
	}

	// === PRIORITY 2: Standalone action words with context ===

	// Budget - standalone keywords (check if not part of transaction context)
	if utils.ContainsAny(msg, []string{"budget", "anggaran", "limit budget", "planning budget"}) &&
		!utils.ContainsAny(msg, []string{"pengeluaran", "transaction", "transaksi", "beli", "makan"}) {
		return true, "budget", map[string]interface{}{}
	}

	// Recurring - transaksi berulang (check standalone)
	if utils.ContainsAny(msg, []string{"recurring", "berulang", "auto debit", "otomatis", "langganan", "subscription", "member"}) {
		return true, "recurring", map[string]interface{}{}
	}

	// Account - saldo, bank, e-wallet (check standalone)
	if utils.ContainsAny(msg, []string{"saldo", "uang di", "bank", "e-wallet", "kartu debit", "kartu kredit", "akun", "rekening", "cash", "tunai"}) {
		return true, "account", map[string]interface{}{}
	}

	// === PRIORITY 3: Transaction (only if explicit expense/income words) ===
	// Only trigger transaction if there's clear expense/income context
	expenseKeywords := []string{"pengeluaran", "income", "pemasukan", "gajian", "gaji", "dapet", "dapat", "earned", "salary", "spent", "uang keluar", "uang masuk"}
	if utils.ContainsAny(msg, expenseKeywords) {
		return true, "transaction", map[string]interface{}{"message": message}
	}

	// Transaction with explicit spending words
	if utils.ContainsAny(msg, []string{"beli ", "buy ", "purchase", "bayar ", "pay ", "transaction", "transaksi"}) {
		return true, "transaction", map[string]interface{}{"message": message}
	}

	// === PRIORITY 4: Other intents ===

	// Investment - crypto, bitcoin, portfolio, stock, saham, harga
	if utils.ContainsAny(msg, []string{"crypto", "bitcoin", "ethereum", "invest", "portfolio", "investasi", "trading", "saham", "stock", "stocks", "aapl", "googl", "msft", "tsla", "tesla", "apple", "google", "microsoft"}) {
		return true, "investment", map[string]interface{}{}
	}

	// Investment suggestions/recommendations
	if utils.ContainsAny(msg, []string{"saran", "recommend", "suggest", "tips", "bagus", "good", "ide"}) {
		return true, "investment", map[string]interface{}{}
	}

	// Debt - hutang, cicilan, pinjaman
	if utils.ContainsAny(msg, []string{"hutang", "debt", "pinjaman", "kredit", "cicilan", "loan"}) {
		return true, "debt", map[string]interface{}{"message": message}
	}

	// Savings - tabungan, target, save
	if utils.ContainsAny(msg, []string{"tabungan", "savings", "goal", "target", "menabung", "nabung", "save", "saved"}) {
		return true, "savings", map[string]interface{}{"message": message}
	}

	// Spending Analysis
	if utils.ContainsAny(msg, []string{"analisa", "analysis", "spending", "pola", "total", "cek", "lihat", "berapa", "bulanan", "bulan ini", "bulan lalu"}) {
		return true, "spending", map[string]interface{}{}
	}

	// Bills & Recurring
	if utils.ContainsAny(msg, []string{"bill", "tagihan", "reminder", "jatuh tempo", "pembayaran"}) {
		return true, "bills", map[string]interface{}{"message": message}
	}

	// Financial Health
	if utils.ContainsAny(msg, []string{"health", "kesehatan", "keuangan", "summary", "ringkasan"}) {
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

	// Route to command pattern
	if cmd, exists := s.commands[toolName]; exists {
		return cmd.Handle(userID, msg)
	}

	return "Maaf, saya tidak mengerti. Bisa jelaskan lagi?"
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
			if utils.IsIncomeByType(tx) {
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
