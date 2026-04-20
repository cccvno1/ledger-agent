package wechat

import (
	"context"
	"fmt"
	"log/slog"
)

// Wire loads saved WeChat credentials and starts the iLink message bridge in
// the background. If no credentials are found it logs a message and returns nil
// — the server starts normally without WeChat integration.
//
// To obtain credentials run: go run ./cmd/wechat-login
func Wire(ctx context.Context, logger *slog.Logger, chatter Chatter) error {
	creds, err := LoadCredentials()
	if err != nil {
		return fmt.Errorf("wechat: load credentials: %w", err)
	}
	if creds == nil {
		logger.Info("wechat: no credentials found — run 'go run ./cmd/wechat-login' to connect WeChat")
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
