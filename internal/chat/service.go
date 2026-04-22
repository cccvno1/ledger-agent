package chat

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"

	"github.com/cccvno1/ledger-agent/internal/base/conf"
)

type sessionIDKey struct{}

func sessionIDFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(sessionIDKey{}).(string); ok {
		return v
	}
	return ""
}

// lockedSession acquires the per-id mutex on the store, loads or creates the
// session, and returns the live *Session together with an unlock func that
// callers must defer. All tool handlers that perform a load \u2192 mutate \u2192 save
// sequence MUST wrap their body with this helper, so that parallel tool calls
// inside a single ReAct iteration cannot clobber each other's writes (each
// goroutine reads a fresh copy from the DB-backed store; without serialisation
// the last writer wins).
func lockedSession(ctx context.Context, store SessionStorer) (*Session, func()) {
	sid := sessionIDFromCtx(ctx)
	mu := store.LockFor(sid)
	mu.Lock()
	return store.GetOrCreate(sid), mu.Unlock
}

// Service orchestrates multi-turn chat with the agent.
type Service struct {
	sessions           SessionStorer
	agent              *react.Agent
	registry           *Registry
	maxHistoryMessages int
	chatTimeout        time.Duration
}

// NewService creates a Service and builds the underlying agent.
func NewService(ctx context.Context, logger *slog.Logger, cfg conf.MiniMax, sessions SessionStorer, searcher CustomerSearcher, writer LedgerWriter, querier LedgerQuerier, products ProductSearcher, payments PaymentRecorder) (*Service, error) {
	if logger == nil {
		logger = slog.Default()
	}
	registry := buildRegistry(logger, sessions, searcher, writer, querier, products, payments)
	einoTools, err := registry.BuildEinoTools()
	if err != nil {
		return nil, fmt.Errorf("chat: new service: %w", err)
	}
	agent, err := buildAgent(ctx, cfg, sessions, einoTools)
	if err != nil {
		return nil, fmt.Errorf("chat: new service: %w", err)
	}
	maxHist := cfg.MaxHistoryMessages
	if maxHist <= 0 {
		maxHist = 50
	}
	chatTimeout := time.Duration(cfg.ChatTimeoutSeconds) * time.Second
	if chatTimeout <= 0 {
		chatTimeout = 60 * time.Second
	}
	return &Service{
		sessions:           sessions,
		agent:              agent,
		registry:           registry,
		maxHistoryMessages: maxHist,
		chatTimeout:        chatTimeout,
	}, nil
}

// ChatInput carries a single user turn.
type ChatInput struct {
	SessionID string
	Message   string
}

// ChatOutput carries the assistant reply and current draft state.
type ChatOutput struct {
	SessionID string
	Reply     string
	Draft     []DraftEntry
}

// Chat processes one user message and returns the assistant reply.
func (s *Service) Chat(ctx context.Context, in ChatInput) (*ChatOutput, error) {
	if in.SessionID == "" {
		in.SessionID = uuid.NewString()
	}

	// Cap the entire generate cycle so a stalled LLM/tool can never block a
	// caller indefinitely. Per-tool timeouts inside the Registry are stricter
	// budgets carved out of this envelope.
	ctx, cancel := context.WithTimeout(ctx, s.chatTimeout)
	defer cancel()

	ctx = context.WithValue(ctx, sessionIDKey{}, in.SessionID)

	// Hold the per-id store mutex for the entire turn. This serialises
	// concurrent HTTP/WeChat turns for the same session_id at the Service
	// boundary AND prevents tool handlers (which acquire the same mutex via
	// lockedSession) from re-entering it. We acquire here, do all reads and
	// the agent Generate (which spawns tool calls in goroutines that try to
	// re-lock — see note below), then release after persistence.
	//
	// IMPORTANT: tool handlers must NOT call lockedSession themselves while
	// this lock is held, otherwise they deadlock. Instead, we release the
	// turn-lock for the duration of Generate() so each tool invocation can
	// briefly acquire+release it around its own load\u2192mutate\u2192save sequence,
	// then re-acquire it for the final Messages write.
	turnLock := s.sessions.LockFor(in.SessionID)
	turnLock.Lock()
	sess := s.sessions.GetOrCreate(in.SessionID)
	turnLock.Unlock()

	userMsg := schema.UserMessage(in.Message)

	// Apply sliding window: send only the most recent messages to the LLM.
	// The structured context block (draft, op log, stats) is injected via
	// dynamicModifier, so older messages can be safely dropped.
	history := sess.Messages
	if len(history) > s.maxHistoryMessages {
		history = history[len(history)-s.maxHistoryMessages:]
	}
	window := make([]*schema.Message, 0, len(history)+1)
	window = append(window, history...)
	window = append(window, userMsg)

	// Use WithMessageFuture to capture all intermediate messages
	// (tool-call, tool-response) that Generate() produces internally.
	opt, future := react.WithMessageFuture()
	reply, err := s.agent.Generate(ctx, window, opt)
	if err != nil {
		return nil, fmt.Errorf("chat: generate: %w", err)
	}

	reply.Content = stripThinkTags(reply.Content)

	// Drain the future to collect all intermediate messages.
	var intermediate []*schema.Message
	iter := future.GetMessages()
	for {
		msg, hasNext, iterErr := iter.Next()
		if iterErr != nil || !hasNext {
			break
		}
		intermediate = append(intermediate, msg)
	}

	// Persist full history (not the windowed version). Tools have already
	// committed their Draft / Phase / OpLog mutations atomically via
	// lockedSession; we re-acquire the turn-lock here, reload, and write
	// once more to attach the assistant turn's messages without clobbering
	// any tool-side state.
	turnLock.Lock()
	defer turnLock.Unlock()
	final := s.sessions.GetOrCreate(in.SessionID)
	newMsgs := make([]*schema.Message, 0, len(final.Messages)+1+len(intermediate)+1)
	newMsgs = append(newMsgs, final.Messages...)
	newMsgs = append(newMsgs, userMsg)
	newMsgs = append(newMsgs, intermediate...)
	newMsgs = append(newMsgs, reply)
	final.Messages = newMsgs
	s.sessions.Set(final)

	return &ChatOutput{
		SessionID: in.SessionID,
		Reply:     reply.Content,
		Draft:     final.Draft,
	}, nil
}

// thinkTagRe matches <think>...</think> blocks produced by reasoning models.
var thinkTagRe = regexp.MustCompile(`(?s)<think>.*?</think>\s*`)

func stripThinkTags(s string) string {
	return strings.TrimSpace(thinkTagRe.ReplaceAllString(s, ""))
}
