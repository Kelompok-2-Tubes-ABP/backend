package services

import (
	"bytes"
	"errors"
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
	collection *mongo.Collection
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

	// External AI config
	aiAPIKey  string
	aiAPIURL  string
	aiModel   string

	// RAG for knowledge base
	ragService *RAGService

	// Rate limiting
	rateLimitManager *ChatbotRateLimitManager

	// Enhanced context services
	financeProfileService      *FinanceProfileService
	proactiveInsightService   *ProactiveInsightService
	conversationContextService *ConversationContextService
	feedbackLearningService   *FeedbackLearningService

	commands map[string]ChatbotCommand
}

func NewChatService(client *mongo.Client, dbName string, transactionService *TransactionService) *ChatbotService {
	// External AI config (OpenAI-compatible APIs)
	aiAPIKey := os.Getenv("AI_API_KEY")
	aiAPIURL := os.Getenv("AI_API_URL")
	aiModel := os.Getenv("AI_MODEL")

	// Defaults for external AI
	if aiAPIURL == "" {
		aiAPIURL = "https://api.openai.com/v1/chat/completions"
	}
	if aiModel == "" {
		aiModel = "gpt-4o-mini"
	}

	// Ollama config (local fallback)
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
		openAIAPIKey:       aiAPIKey,
		ollamaURL:          ollamaURL,
		modelName:          modelName,
		maxMemoryMessages:  10,
		aiAPIKey:           aiAPIKey,
		aiAPIURL:           aiAPIURL,
		aiModel:            aiModel,
		rateLimitManager:    NewChatbotRateLimitManager(),
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

// SetRAGService - Inject RAG Service for knowledge base
func (s *ChatbotService) SetRAGService(rag *RAGService) {
	s.ragService = rag
}

// SetFinanceProfileService - Inject FinanceProfileService
func (s *ChatbotService) SetFinanceProfileService(fps *FinanceProfileService) {
	s.financeProfileService = fps
}

// SetProactiveInsightService - Inject ProactiveInsightService
func (s *ChatbotService) SetProactiveInsightService(pis *ProactiveInsightService) {
	s.proactiveInsightService = pis
}

// SetConversationContextService - Inject ConversationContextService
func (s *ChatbotService) SetConversationContextService(ccs *ConversationContextService) {
	s.conversationContextService = ccs
}

// SetFeedbackLearningService - Inject FeedbackLearningService
func (s *ChatbotService) SetFeedbackLearningService(fls *FeedbackLearningService) {
	s.feedbackLearningService = fls
}

// RegisterCommand attaches a new command handler to the chatbot
func (s *ChatbotService) RegisterCommand(intentName string, cmd ChatbotCommand) {
	if s.commands == nil {
		s.commands = make(map[string]ChatbotCommand)
	}
	s.commands[intentName] = cmd
}

// ProcessMessage - Main entry point
// ProcessMessage - Main entry point
func (s *ChatbotService) ProcessMessage(userID string, message string, sessionID string) (string, error) {
	// Input validation
	if userID == "" {
		return "", errors.New("userID is required")
	}
	if message == "" {
		return "", errors.New("message is required")
	}

	// Rate limiting check
	// Rate limiting check
	if s.rateLimitManager != nil {
		// Check request rate limit
		allowed, msg := s.rateLimitManager.AllowRequest(userID)
		if !allowed {
			return msg, nil
		}

		// Check token limit and cache
		allowed, errMsg, cachedResp := s.rateLimitManager.AllowTokens(userID, message)
		if !allowed {
			return errMsg, nil
		}
		if cachedResp != "" {
			// Return cached response
			botMsg := models.ChatMessage{
				ID:          primitive.NewObjectID(),
				UserID:      userID,
				SessionID:   sessionID,
				Message:     message,
				Response:    cachedResp,
				MessageType: "assistant",
				Timestamp:   time.Now(),
			}
			s.saveMessage(botMsg)
			return cachedResp, nil
		}
	}

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
	needsTool, toolName, toolArgs := s.analyzeIntent(userID, message, context)

	var response string
	if needsTool {
		response = s.executeTool(toolName, toolArgs, userID, context)
	} else {
		// Use AI with retry logic
		resp, err := s.callWithRetry(message, context, conversationHistory)
		if err != nil {
			return "", fmt.Errorf("AI services unavailable after retries: %w", err)
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

	// Cache AI response for rate limiting
	if s.rateLimitManager != nil && !needsTool {
		s.rateLimitManager.CacheResponse(message, response)
	}

	return response, nil
}

// analyzeIntent - Detect what user wants using NLU
func (s *ChatbotService) analyzeIntent(userID string, message string, context map[string]interface{}) (bool, string, map[string]interface{}) {
	// === Step 1: Use NEW NLU for intent detection ===
	intent := utils.DetectIntent(message)
	_, confidence := utils.DetectIntentWithConfidence(message)
	
	// === Step 1.5: Detect sentiment for empathetic responses ===
	sentiment := utils.DetectSentiment(message)
	sentimentCtx := utils.GetSentimentContext(sentiment)

	// === Step 1.6: Check learned feedback ===
	if s.feedbackLearningService != nil {
		if correctedIntent, found := s.feedbackLearningService.GetCorrectedIntent(userID, message); found {
			intent = correctedIntent
		}
	}

	// === Step 2: Check for conversation context references ("itu", "yang ini") ===
	if s.conversationContextService != nil {
		lastTopic, _, _ := s.conversationContextService.ResolveReference("", "", message)
		if lastTopic != "" && lastTopic != "ai" {
			// User is referring to previous topic
			intent = lastTopic
		}
	}

	// === Step 3: Extract entities for tool arguments ===
	entities := utils.ExtractEntities(message)
	toolArgs := map[string]interface{}{
		"message":   message,
		"action":    entities.Action,
		"amount":    entities.Amount,
		"category":  entities.Category,
		"note":      entities.Note,
		"time":      entities.Time,
		"confidence": confidence,
		"sentiment":  sentimentCtx["sentiment"],
		"needs_empathy": sentimentCtx["needs_empathy"],
		"needs_urgency": sentimentCtx["needs_urgency"],
	}

	// === Step 4: Route based on intent ===
	// If AI intent with low confidence, send to AI
	if intent == "ai" {
		return true, "ai", toolArgs
	}

	// Check if this is a command we support
	supportedIntents := map[string]bool{
		"transaction": true,
		"budget":      true,
		"bills":       true,
		"account":     true,
		"savings":     true,
		"investment":  true,
		"debt":        true,
		"recurring":   true,
		"spending":    true,
		"health":      true,
	}

	if supportedIntents[intent] {
		// Update conversation context
		if s.conversationContextService != nil {
			parsedEntities := ParsedEntitiesFromUtils(entities)
			s.conversationContextService.UpdateContext("", "", intent, parsedEntities, message)
		}
		return true, intent, toolArgs
	}

	// Unknown intent → send to AI for processing
	return true, "ai", toolArgs
}

// ParsedEntitiesFromUtils converts utils.ParsedIntent to ConversationContext ParsedEntities
func ParsedEntitiesFromUtils(e utils.ParsedIntent) ParsedEntities {
	return ParsedEntities{
		Action:   e.Action,
		Amount:   e.Amount,
		Category: e.Category,
		Note:     e.Note,
		Time:     e.Time,
	}
}

// executeTool - Execute the appropriate tool
func (s *ChatbotService) executeTool(toolName string, args map[string]interface{}, userID string, context map[string]interface{}) string {
	// Get message from args if available
	msg := ""
	if m, ok := args["message"].(string); ok {
		msg = m
	}

	// Handle AI fallback
	if toolName == "ai" {
		if s.aiAPIKey != "" {
			resp, err := s.callExternalAIWithHistory(msg, context, nil)
			if err != nil {
				return fmt.Sprintf("Maaf, AI tidak tersedia saat ini: %v", err)
			}
			return resp
		}
		return "Maaf, AI belum dikonfigurasi."
	}

	// Route to command pattern
	if cmd, exists := s.commands[toolName]; exists {
		return cmd.Handle(userID, msg)
	}

	// Fallback: route unknown commands to AI
	resp, err := s.callWithRetry(msg, context, nil)
	if err != nil {
		return "Maaf, saya tidak mengerti. Bisa jelaskan lagi?"
	}
	return resp
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

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Ollama connection error: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

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

// callWithRetry - Generic AI call with retry and backoff
func (s *ChatbotService) callWithRetry(message string, context map[string]interface{}, history []models.ChatMessage) (string, error) {
	maxRetries := 2
	backoffMs := []int{500, 2000} // 500ms, 2s

	// Try external AI first if configured
	if s.aiAPIKey != "" {
		for attempt := 0; attempt <= maxRetries; attempt++ {
			resp, err := s.callExternalAIWithHistory(message, context, history)
			if err == nil {
				return resp, nil
			}
			if attempt < maxRetries {
				time.Sleep(time.Duration(backoffMs[attempt]) * time.Millisecond)
			}
		}
	}

	// Fallback to Ollama
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := s.callOllamaWithHistory(message, context, history)
		if err == nil {
			return resp, nil
		}
		if attempt < maxRetries {
			time.Sleep(time.Duration(backoffMs[attempt]) * time.Millisecond)
		}
	}

	return "", fmt.Errorf("all AI services failed after %d retries", maxRetries+1)
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

	// Add RAG knowledge base context if available
	if s.ragService != nil {
		ragContext := s.ragService.BuildContext(message)
		if ragContext != "" {
			systemPrompt += "\n\n" + ragContext
			systemPrompt += "\nGunakan informasi dari knowledge base di atas untuk menjawab pertanyaan jika relevan."
		}
	}

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

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Ollama connection error: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

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

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.openAIAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

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

// callExternalAIWithHistory - Call external OpenAI-compatible API with conversation history
func (s *ChatbotService) callExternalAIWithHistory(message string, context map[string]interface{}, history []models.ChatMessage) (string, error) {
	if s.aiAPIKey == "" {
		return "", fmt.Errorf("External AI API key not configured")
	}

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

	// Add RAG knowledge base context if available
	if s.ragService != nil {
		ragContext := s.ragService.BuildContext(message)
		if ragContext != "" {
			systemPrompt += "\n\n" + ragContext
			systemPrompt += "\nGunakan informasi dari knowledge base di atas untuk menjawab pertanyaan jika relevan."
		}
	}

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

	// Build request - check if using Gemini or OpenAI-compatible API
	var reqBody map[string]interface{}

	if strings.Contains(s.aiAPIURL, "generativelanguage.googleapis.com") || s.aiModel != "" && strings.HasPrefix(s.aiModel, "gemini") {
		// Gemini API format
		parts := []map[string]string{}
		for _, m := range messages {
			parts = append(parts, map[string]string{
				"text": m["content"].(string),
			})
		}
		reqBody = map[string]interface{}{
			"contents": []map[string]interface{}{
				{"parts": parts},
			},
			"generationConfig": map[string]interface{}{
				"maxOutputTokens": 500,
				"temperature":     0.7,
			},
		}
		// Add API key as query param for Gemini
		s.aiAPIURL = strings.TrimSuffix(s.aiAPIURL, "?key="+s.aiAPIKey) + "?key=" + s.aiAPIKey
	} else {
		// OpenAI-compatible format
		reqBody = map[string]interface{}{
			"model":      s.aiModel,
			"max_tokens": 500,
			"messages":   messages,
		}
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", s.aiAPIURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	if !strings.Contains(s.aiAPIURL, "generativelanguage.googleapis.com") {
		req.Header.Set("Authorization", "Bearer "+s.aiAPIKey)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("External AI connection error: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("External AI API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// Handle Gemini response format
	if strings.Contains(s.aiAPIURL, "generativelanguage.googleapis.com") {
		if candidates, ok := result["candidates"].([]interface{}); ok && len(candidates) > 0 {
			if c, ok := candidates[0].(map[string]interface{}); ok {
				if content, ok := c["content"].(map[string]interface{}); ok {
					if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
						if p, ok := parts[0].(map[string]interface{}); ok {
							if text, ok := p["text"].(string); ok {
								return text, nil
							}
						}
					}
				}
			}
		}
		return "Maaf, ada masalah dengan respons AI.", nil
	}

	// Handle OpenAI-compatible response format
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

	// 5. Enhanced Spending Patterns (Personal Advisor Context)
	if s.financeProfileService != nil {
		profileContext := s.financeProfileService.GetProfileContext(userID)
		if len(profileContext) > 0 {
			context["spending_patterns"] = profileContext["spending_patterns"]
			context["budget_adherence"] = profileContext["budget_adherence"]
		}
	}

	// 6. Proactive Insights
	if s.proactiveInsightService != nil {
		insights := s.proactiveInsightService.GetInsightsForChatbot(userID)
		if len(insights) > 0 {
			context["proactive_insights"] = insights
		}
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
