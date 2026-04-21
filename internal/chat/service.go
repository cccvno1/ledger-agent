package chat

import (
	"context"
	"fmt"
	"regexp"
	"strings"

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

// Service orchestrates multi-turn chat with the agent.
type Service struct {
	sessions           SessionStorer
	agent              *react.Agent
	maxHistoryMessages int
}

// NewService creates a Service and builds the underlying agent.
func NewService(ctx context.Context, cfg conf.MiniMax, sessions SessionStorer, searcher CustomerSearcher, writer LedgerWriter, querier LedgerQuerier, products ProductSearcher, payments PaymentRecorder) (*Service, error) {
	tools := buildTools(sessions, searcher, writer, querier, products, payments)
	agent, err := buildAgent(ctx, cfg, sessions, tools)
	if err != nil {
		return nil, fmt.Errorf("chat: new service: %w", err)
	}
	maxHist := cfg.MaxHistoryMessages
	if maxHist <= 0 {
		maxHist = 50
	}
	return &Service{
		sessions:           sessions,
		agent:              agent,
		maxHistoryMessages: maxHist,
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

	ctx = context.WithValue(ctx, sessionIDKey{}, in.SessionID)
	sess := s.sessions.GetOrCreate(in.SessionID)

	// Lock the session for the entire generate cycle to prevent concurrent
	// mutations from rapid messages by the same user.
	sess.mu.Lock()
	defer sess.mu.Unlock()

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

	// Persist full history (not the windowed version).
	newMsgs := make([]*schema.Message, 0, len(sess.Messages)+1+len(intermediate)+1)
	newMsgs = append(newMsgs, sess.Messages...)
	newMsgs = append(newMsgs, userMsg)
	newMsgs = append(newMsgs, intermediate...)
	newMsgs = append(newMsgs, reply)
	sess.Messages = newMsgs
	s.sessions.Set(sess)

	return &ChatOutput{
		SessionID: in.SessionID,
		Reply:     reply.Content,
		Draft:     sess.Draft,
	}, nil
}

// thinkTagRe matches <think>...</think> blocks produced by reasoning models.
var thinkTagRe = regexp.MustCompile(`(?s)<think>.*?</think>\s*`)

func stripThinkTags(s string) string {
	return strings.TrimSpace(thinkTagRe.ReplaceAllString(s, ""))
}
