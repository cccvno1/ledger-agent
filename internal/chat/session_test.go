package chat

import (
	"sync"
	"testing"
	"time"
)

func TestSessionStore_GetOrCreate(t *testing.T) {
	s := NewSessionStore()
	sess1 := s.GetOrCreate("user-1")
	sess2 := s.GetOrCreate("user-1")

	if sess1 != sess2 {
		t.Error("GetOrCreate returned different pointers for same ID")
	}
	if s.Len() != 1 {
		t.Errorf("Len() = %d, want 1", s.Len())
	}
}

func TestSessionStore_Evict(t *testing.T) {
	s := NewSessionStore()

	// Create a session with old LastActiveAt
	sess := s.GetOrCreate("old-user")
	sess.LastActiveAt = time.Now().Add(-48 * time.Hour)
	s.data["old-user"] = sess

	// Recent session
	s.GetOrCreate("new-user")

	s.evict(24 * time.Hour)

	if s.Len() != 1 {
		t.Errorf("after evict: Len() = %d, want 1", s.Len())
	}
	if s.Get("old-user") != nil {
		t.Error("old session should have been evicted")
	}
	if s.Get("new-user") == nil {
		t.Error("new session should not have been evicted")
	}
}

func TestSessionStore_ConcurrentAccess(t *testing.T) {
	s := NewSessionStore()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sess := s.GetOrCreate("shared")
			sess.mu.Lock()
			sess.AppendOp("query", "q", nil)
			sess.mu.Unlock()
			s.Set(sess)
		}()
	}
	wg.Wait()

	sess := s.Get("shared")
	if sess == nil {
		t.Fatal("session should exist")
	}
	if sess.Stats.Queried != 100 {
		t.Errorf("Stats.Queried = %d, want 100", sess.Stats.Queried)
	}
}
