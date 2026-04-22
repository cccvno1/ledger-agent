package chat

import (
	"testing"
	"time"
)

func TestTokenStore_IssueAndConsume(t *testing.T) {
	s := NewTokenStore()
	op := s.Issue("sess-A", OpDeleteEntry, map[string]any{"entry_id": "e1"}, "preview")
	if op.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if op.SessionID != "sess-A" {
		t.Fatalf("session id = %q, want sess-A", op.SessionID)
	}

	got := s.Consume(op.Token, "sess-A")
	if got == nil {
		t.Fatal("Consume returned nil for valid token")
	}
	if got.Payload["entry_id"] != "e1" {
		t.Errorf("payload entry_id = %v", got.Payload["entry_id"])
	}

	// Second consume must fail (single-use).
	if again := s.Consume(op.Token, "sess-A"); again != nil {
		t.Fatal("token must be single-use")
	}
}

func TestTokenStore_RejectsWrongSession(t *testing.T) {
	s := NewTokenStore()
	op := s.Issue("sess-A", OpSettleAccount, nil, "preview")
	if got := s.Consume(op.Token, "sess-B"); got != nil {
		t.Fatal("token must not cross sessions")
	}
}

func TestTokenStore_Expiry(t *testing.T) {
	s := NewTokenStore()
	op := s.Issue("sess-A", OpDeleteEntry, nil, "preview")
	// Force expiry by rewinding ExpiresAt.
	s.mu.Lock()
	s.ops[op.Token].ExpiresAt = time.Now().Add(-1 * time.Second)
	s.mu.Unlock()

	if got := s.Consume(op.Token, "sess-A"); got != nil {
		t.Fatal("expired token must be rejected")
	}
}
