package models

type ChatRequest struct {
	Message     string `json:"message"`
	SessionID   string `json:"session_id,omitempty"`
	ContextType string `json:"context_type"`
}
