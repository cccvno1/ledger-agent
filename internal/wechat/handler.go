package wechat

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Chatter processes a user message and returns a reply.
// sessionID is stable per WeChat user (derived from from_user_id).
type Chatter interface {
	Chat(ctx context.Context, sessionID string, message string) (string, error)
}

type handler struct {
	chatter  Chatter
	logger   *slog.Logger
	seenMsgs sync.Map // map[int64]time.Time — dedup by message_id
}

func newHandler(chatter Chatter, logger *slog.Logger) *handler {
	return &handler{chatter: chatter, logger: logger}
}

func (h *handler) handle(ctx context.Context, c *client, msg weixinMessage) {
	// Only process completed user messages.
	if msg.MessageType != msgTypeUser || msg.MessageState != msgStateDone {
		return
	}

	// Deduplicate by message_id (voice messages can fire multiple finish-state events).
	if msg.MessageID != 0 {
		if _, loaded := h.seenMsgs.LoadOrStore(msg.MessageID, time.Now()); loaded {
			return
		}
		go h.cleanSeenMsgs()
	}

	text := extractText(msg)
	if text == "" {
		return
	}

	h.logger.Info("wechat: received", "from", msg.FromUserID, "text", truncate(text, 80))

	reply, err := h.chatter.Chat(ctx, msg.FromUserID, text)
	if err != nil {
		h.logger.Error("wechat: chat error", "err", err)
		return
	}

	h.logger.Info("wechat: sending reply", "to", msg.FromUserID, "reply", truncate(reply, 120))

	// WeChat doesn't render markdown — convert before sending.
	if err := c.sendText(ctx, msg.FromUserID, msg.ContextToken, markdownToPlain(reply)); err != nil {
		h.logger.Error("wechat: send failed", "err", err)
	} else {
		h.logger.Info("wechat: send ok", "to", msg.FromUserID)
	}
}

func extractText(msg weixinMessage) string {
	for _, item := range msg.ItemList {
		switch item.Type {
		case itemTypeText:
			if item.TextItem != nil {
				return item.TextItem.Text
			}
		case itemTypeVoice:
			if item.VoiceItem != nil && item.VoiceItem.Text != "" {
				return item.VoiceItem.Text // WeChat speech-to-text
			}
		}
	}
	return ""
}

func (h *handler) cleanSeenMsgs() {
	cutoff := time.Now().Add(-5 * time.Minute)
	h.seenMsgs.Range(func(k, v any) bool {
		if t, ok := v.(time.Time); ok && t.Before(cutoff) {
			h.seenMsgs.Delete(k)
		}
		return true
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// markdownToPlain converts markdown to readable plain text for WeChat display.
// WeChat renders plain text only; markdown syntax shows as raw characters.
var (
	reCodeBlock  = regexp.MustCompile("(?s)```[^\n]*\n?(.*?)```")
	reInlineCode = regexp.MustCompile("`([^`]+)`")
	reImage      = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	reLink       = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	reTableSep   = regexp.MustCompile(`(?m)^\|[\s:|\-]+\|$`)
	reTableRow   = regexp.MustCompile(`(?m)^\|(.+)\|$`)
	reHeader     = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	reBold       = regexp.MustCompile(`\*\*(.+?)\*\*|__(.+?)__`)
	reStrike     = regexp.MustCompile(`~~(.+?)~~`)
	reBlockquote = regexp.MustCompile(`(?m)^>\s?`)
	reHR         = regexp.MustCompile(`(?m)^[-*_]{3,}\s*$`)
	reUL         = regexp.MustCompile(`(?m)^(\s*)[-*+]\s+`)
)

func markdownToPlain(text string) string {
	s := text

	// Code blocks: strip fences, keep content
	s = reCodeBlock.ReplaceAllStringFunc(s, func(m string) string {
		parts := reCodeBlock.FindStringSubmatch(m)
		if len(parts) > 1 {
			return strings.TrimSpace(parts[1])
		}
		return m
	})

	s = reInlineCode.ReplaceAllString(s, "$1")
	s = reImage.ReplaceAllString(s, "")
	s = reLink.ReplaceAllString(s, "$1")
	s = reTableSep.ReplaceAllString(s, "")
	s = reTableRow.ReplaceAllStringFunc(s, func(m string) string {
		parts := reTableRow.FindStringSubmatch(m)
		if len(parts) > 1 {
			cells := strings.Split(parts[1], "|")
			for i := range cells {
				cells[i] = strings.TrimSpace(cells[i])
			}
			return strings.Join(cells, "  ")
		}
		return m
	})
	s = reHeader.ReplaceAllString(s, "")
	s = reBold.ReplaceAllStringFunc(s, func(m string) string {
		parts := reBold.FindStringSubmatch(m)
		if parts[1] != "" {
			return parts[1]
		}
		return parts[2]
	})
	s = reStrike.ReplaceAllString(s, "$1")
	s = reBlockquote.ReplaceAllString(s, "")
	s = reHR.ReplaceAllString(s, "")
	s = reUL.ReplaceAllString(s, "$1• ")

	return strings.TrimSpace(s)
}
