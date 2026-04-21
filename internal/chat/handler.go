package chat

import (
	"context"
	"net/http"

	"github.com/cccvno1/goplate/pkg/httpkit"
)

type chatService interface {
	Chat(ctx context.Context, in ChatInput) (*ChatOutput, error)
}

// Handler handles HTTP requests for the chat feature.
type Handler struct {
	svc chatService
}

// NewHandler creates a Handler.
func NewHandler(svc chatService) *Handler {
	return &Handler{svc: svc}
}

// Chat handles POST /api/v1/chat.
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := httpkit.Decode(r, &req); err != nil {
		httpkit.Error(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		httpkit.Error(w, err)
		return
	}

	out, err := h.svc.Chat(r.Context(), req.ToInput())
	if err != nil {
		httpkit.Error(w, err)
		return
	}

	httpkit.JSON(w, http.StatusOK, NewChatResponse(out))
}
