package chat

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// OperationKind names a destructive operation that requires user confirmation
// via the propose / commit token protocol.
type OperationKind string

const (
	OpDeleteEntry   OperationKind = "delete_entry"
	OpSettleAccount OperationKind = "settle_account"
)

// Operation is a destructive action staged via propose_X and executed via
// commit_operation. The token is opaque to the LLM; it must echo it back
// verbatim. Tokens expire after tokenTTL or after one successful commit.
type Operation struct {
	Token     string
	Kind      OperationKind
	SessionID string
	Payload   map[string]any // arguments captured at propose time
	Preview   string         // human-readable summary surfaced to the LLM
	CreatedAt time.Time
	ExpiresAt time.Time
}

const tokenTTL = 5 * time.Minute

// TokenStore holds short-lived destructive-operation tokens scoped per session.
// Concurrent access is safe; expired tokens are reaped on access.
type TokenStore struct {
	mu  sync.Mutex
	ops map[string]*Operation // token -> operation
}

// NewTokenStore returns a fresh, empty store.
func NewTokenStore() *TokenStore {
	return &TokenStore{ops: make(map[string]*Operation)}
}

// Issue records a new operation and returns its token.
func (s *TokenStore) Issue(sessionID string, kind OperationKind, payload map[string]any, preview string) *Operation {
	now := time.Now()
	op := &Operation{
		Token:     uuid.NewString(),
		Kind:      kind,
		SessionID: sessionID,
		Payload:   payload,
		Preview:   preview,
		CreatedAt: now,
		ExpiresAt: now.Add(tokenTTL),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ops[op.Token] = op
	return op
}

// Consume looks up and removes a token, but only if it belongs to sessionID
// and has not expired. Returns nil on any failure.
func (s *TokenStore) Consume(token, sessionID string) *Operation {
	s.mu.Lock()
	defer s.mu.Unlock()
	op, ok := s.ops[token]
	if !ok {
		return nil
	}
	delete(s.ops, token)
	if op.SessionID != sessionID {
		return nil
	}
	if time.Now().After(op.ExpiresAt) {
		return nil
	}
	return op
}

// Reap removes expired tokens. Callers may invoke this opportunistically;
// not currently scheduled, since the active set is tiny.
func (s *TokenStore) Reap() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, op := range s.ops {
		if now.After(op.ExpiresAt) {
			delete(s.ops, k)
		}
	}
}
