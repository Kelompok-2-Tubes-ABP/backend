package chatbot_commands

// ChatbotCommand defines the interface for all chatbot intention handlers.
type ChatbotCommand interface {
	Handle(userID string, message string) string
}
