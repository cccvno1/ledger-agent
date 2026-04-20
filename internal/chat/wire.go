package chat

import (
	"context"
	"net/http"
	"time"

	"github.com/cccvno1/ledger-agent/internal/base/conf"
)

// Wire assembles the chat feature and registers its routes on mux.
func Wire(ctx context.Context, mux *http.ServeMux, cfg conf.MiniMax, searcher CustomerSearcher, writer LedgerWriter, querier LedgerQuerier, products ProductSearcher, payments PaymentRecorder) (*Service, error) {
	sessions := NewSessionStore()
	sessions.StartCleanup(ctx, 24*time.Hour, 10*time.Minute)

	svc, err := NewService(ctx, cfg, sessions, searcher, writer, querier, products, payments)
	if err != nil {
		return nil, err
	}
	h := NewHandler(svc)
	mux.HandleFunc("POST /api/v1/chat", h.Chat)
	return svc, nil
}
