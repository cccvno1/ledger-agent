package chat

// ChatRequest is the HTTP request body for POST /api/v1/chat.
type ChatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// ChatResponse is the HTTP response body for POST /api/v1/chat.
type ChatResponse struct {
	SessionID string       `json:"session_id"`
	Reply     string       `json:"reply"`
	Draft     []DraftEntry `json:"draft,omitempty"`
}
