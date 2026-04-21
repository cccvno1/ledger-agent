package wechat

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
)

// Wire registers QR login endpoints, loads saved WeChat credentials, and
// starts the iLink message bridge in the background.
// If no credentials are found it logs a message and returns nil — the server
// starts normally without WeChat integration.
//
// To obtain credentials via the web UI, use POST /api/v1/wechat/qrcode and
// GET /api/v1/wechat/qrcode/status.
// Alternatively, run: go run ./cmd/wechat-login
func Wire(ctx context.Context, mux *http.ServeMux, logger *slog.Logger, chatter Chatter) error {
	// Register QR login endpoints (no auth required — see BearerAuth whitelist).
	qr := newQRHandler(logger)
	mux.HandleFunc("POST /api/v1/wechat/qrcode", qr.GenerateQRCode)
	mux.HandleFunc("GET /api/v1/wechat/qrcode/status", qr.CheckStatus)

	creds, err := LoadCredentials()
	if err != nil {
		return fmt.Errorf("wechat: load credentials: %w", err)
	}
	if creds == nil {
		logger.Info("wechat: no credentials found — use the web UI or run 'go run ./cmd/wechat-login' to connect WeChat")
		return nil
	}

	c := newClient(creds)
	h := newHandler(chatter, logger)
	mon, err := newMonitor(c, h, logger)
	if err != nil {
		return fmt.Errorf("wechat: create monitor: %w", err)
	}

	go mon.run(ctx)
	logger.Info("wechat: bridge started", "bot_id", creds.ILinkBotID)
	return nil
}
