package handler

import (
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

type ChatbotHandler struct {
	chatbotService *services.ChatbotService
}

func NewChatbotHandler(chatbotService *services.ChatbotService) *ChatbotHandler {
	return &ChatbotHandler{
		chatbotService: chatbotService,
	}
}

type ChatMessageRequest struct {
	Message   string `json:"message" binding:"required"`
	SessionID string `json:"session_id,omitempty"`
}

type ChatResponse struct {
	Response  string                 `json:"response"`
	SessionID string                 `json:"session_id"`
	Timestamp string                 `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func (h *ChatbotHandler) ProcessMessage() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			return
		}

		var req ChatMessageRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body: " + err.Error()})
			return
		}

		sessionID := req.SessionID
		if sessionID == "" {
			sessionID = userID.(string)
		}

		response, err := h.chatbotService.ProcessMessage(userID.(string), req.Message, sessionID)
		if err != nil {
			c.JSON(500, gin.H{"error": "Error processing message: " + err.Error()})
			return
		}

		c.JSON(200, ChatResponse{
			Response:  response,
			SessionID: sessionID,
		})
	}
}

func (h *ChatbotHandler) GetChatHistory() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			return
		}

		sessionID := c.Query("session_id")
		if sessionID == "" {
			sessionID = userID.(string)
		}

		messages, err := h.chatbotService.GetChatHistory(userID.(string), sessionID)
		if err != nil {
			c.JSON(500, gin.H{"error": "Error retrieving chat history: " + err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"session_id": sessionID,
			"messages":   messages,
		})
	}
}

func (h *ChatbotHandler) GetFinancialSummary() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			return
		}

		context := h.chatbotService.GetFinancialContext(userID.(string))

		c.JSON(200, gin.H{
			"total_income":       context["total_income"],
			"total_outcome":      context["total_outcome"],
			"net_balance":        context["net_balance"],
			"transaction_count":  context["transaction_count"],
			"category_breakdown": context["category_breakdown"],
		})
	}
}

func (h *ChatbotHandler) QuickAsk() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			return
		}

		question := c.Query("q")
		if question == "" {
			c.JSON(400, gin.H{"error": "Question parameter 'q' is required"})
			return
		}

		response, err := h.chatbotService.ProcessMessage(userID.(string), question, userID.(string))
		if err != nil {
			c.JSON(500, gin.H{"error": "Error processing question: " + err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"question": question,
			"response": response,
		})
	}
}

func (h *ChatbotHandler) ClearConversation() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			return
		}

		sessionID := c.Query("session_id")
		if sessionID == "" {
			sessionID = userID.(string)
		}

		err := h.chatbotService.ClearConversationMemory(userID.(string), sessionID)
		if err != nil {
			c.JSON(500, gin.H{"error": "Error clearing conversation: " + err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message":    "Conversation memory cleared",
			"session_id": sessionID,
		})
	}
}
