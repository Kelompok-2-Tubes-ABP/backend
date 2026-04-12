package routes

import (
	authh "financeapi/essentials/auth"
	"financeapi/essentials/handler"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
)

func RegisterChatbotRoutes(r *gin.Engine, chatbotService *services.ChatbotService) {
	chatbotProtected := r.Group("/chatbot", authh.AuthMiddleware())
	{
		chatbotProtected.POST("/message", handler.NewChatbotHandler(chatbotService).ProcessMessage())
		chatbotProtected.GET("/history", handler.NewChatbotHandler(chatbotService).GetChatHistory())
		chatbotProtected.GET("/summary", handler.NewChatbotHandler(chatbotService).GetFinancialSummary())
		chatbotProtected.GET("/ask", handler.NewChatbotHandler(chatbotService).QuickAsk())
		chatbotProtected.DELETE("/clear", handler.NewChatbotHandler(chatbotService).ClearConversation())
	}
}
