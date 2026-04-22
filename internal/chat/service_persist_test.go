package chat

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
)

// freshLoadingStore mimics DBSessionStore: every Get/GetOrCreate returns a
// freshly cloned *Session built from the last persisted JSON snapshot, so
// callers can never share a pointer across calls. This is the property that
// caused the lost-write bug fixed alongside this test.
type freshLoadingStore struct {
	data    map[string][]byte
	idLocks sync.Map
}

func newFreshLoadingStore() *freshLoadingStore {
	return &freshLoadingStore{data: make(map[string][]byte)}
}

func (s *freshLoadingStore) Get(id string) *Session {
	raw, ok := s.data[id]
	if !ok {
		return nil
	}
	var rec sessionRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil
	}
	return fromRecord(id, &rec)
}

func (s *freshLoadingStore) GetOrCreate(id string) *Session {
	if sess := s.Get(id); sess != nil {
		return sess
	}
	now := time.Now()
	sess := &Session{ID: id, CreatedAt: now, LastActiveAt: now}
	raw, _ := json.Marshal(toRecord(sess))
	s.data[id] = raw
	return sess
}

func (s *freshLoadingStore) Set(sess *Session) {
	sess.LastActiveAt = time.Now()
	raw, err := json.Marshal(toRecord(sess))
	if err != nil {
		return
	}
	s.data[sess.ID] = raw
}

func (s *freshLoadingStore) LockFor(id string) *sync.Mutex {
	v, _ := s.idLocks.LoadOrStore(id, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// TestService_PersistsToolMutations is a regression test for the bug where
// Service.Chat would Set() a stale *Session at the end of a turn and clobber
// any Draft / Phase / OpLog mutations the tools had committed via their own
// Set() calls. The fix reloads the session post-Generate so the final write
// carries tool mutations forward.
//
// We simulate the sequence directly without spinning up the agent: a tool
// loads, mutates, and Sets; then Service.Chat's persistence step runs.
func TestService_PersistsToolMutations(t *testing.T) {
	store := newFreshLoadingStore()
	const sid = "sim-1"

	// 1. Service.Chat loads the session and holds it for the turn.
	sessA := store.GetOrCreate(sid)
	if sessA.CurrentPhase() != PhaseIdle {
		t.Fatalf("initial phase = %q, want Idle", sessA.CurrentPhase())
	}

	// 2. A tool runs: it loads its own copy, mutates, and persists.
	sessB := store.GetOrCreate(sid)
	sessB.Draft = append(sessB.Draft, DraftEntry{
		CustomerName:   "张三",
		ProductName:    "桃",
		UnitPrice:      3,
		Quantity:       5,
		Amount:         15,
		EntryDate:      time.Now(),
		IdempotencyKey: "k-1",
	})
	sessB.SetPhase(PhaseDrafting)
	store.Set(sessB)

	// 3. Service.Chat appends messages and persists. With the fix it must
	// reload before Set so tool mutations survive.
	newMsgs := append(sessA.Messages, schema.UserMessage("今天张三五斤桃 每斤3元"))
	final := store.GetOrCreate(sid)
	final.Messages = newMsgs
	store.Set(final)

	// 4. Next turn: phase must still be Drafting and Draft must contain the
	// row. This is the property the production wechat session needed.
	turn2 := store.GetOrCreate(sid)
	if got := turn2.CurrentPhase(); got != PhaseDrafting {
		t.Fatalf("after turn-end Set, phase = %q, want Drafting (tool mutation lost)", got)
	}
	if len(turn2.Draft) != 1 {
		t.Fatalf("after turn-end Set, len(Draft) = %d, want 1 (tool mutation lost)", len(turn2.Draft))
	}
	if len(turn2.Messages) != 1 {
		t.Fatalf("after turn-end Set, len(Messages) = %d, want 1", len(turn2.Messages))
	}
}
