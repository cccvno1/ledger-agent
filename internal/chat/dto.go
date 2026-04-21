package chat

import "strings"

// ChatRequest is the HTTP request body for POST /api/v1/chat.
type ChatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// Validate checks request-level input formatting.
func (r *ChatRequest) Validate() error {
	if strings.TrimSpace(r.Message) == "" {
		return errMissingMessage
	}
	return nil
}

// ToInput converts the request DTO to the service input.
func (r *ChatRequest) ToInput() ChatInput {
	return ChatInput{
		SessionID: r.SessionID,
		Message:   strings.TrimSpace(r.Message),
	}
}

// ChatResponse is the HTTP response body for POST /api/v1/chat.
type ChatResponse struct {
	SessionID string       `json:"session_id"`
	Reply     string       `json:"reply"`
	Draft     []DraftEntry `json:"draft,omitempty"`
}

// NewChatResponse builds the response DTO from the service output.
func NewChatResponse(out *ChatOutput) ChatResponse {
	if out == nil {
		return ChatResponse{}
	}
	return ChatResponse{
		SessionID: out.SessionID,
		Reply:     out.Reply,
		Draft:     out.Draft,
	}
}
