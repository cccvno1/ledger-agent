package chat

import (
	"net/http"

	"github.com/cccvno1/goplate/pkg/httpkit"
)

// Handler handles HTTP requests for the chat feature.
type Handler struct {
	svc *Service
}

// NewHandler creates a Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Chat handles POST /api/v1/chat.
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := httpkit.Decode(r, &req); err != nil {
		httpkit.Error(w, err)
		return
	}
	if req.Message == "" {
		httpkit.Error(w, errMissingMessage)
		return
	}

	out, err := h.svc.Chat(r.Context(), ChatInput{
		SessionID: req.SessionID,
		Message:   req.Message,
	})
	if err != nil {
		httpkit.Error(w, err)
		return
	}

	httpkit.JSON(w, http.StatusOK, ChatResponse{
		SessionID: out.SessionID,
		Reply:     out.Reply,
		Draft:     out.Draft,
	})
}
