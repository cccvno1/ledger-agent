package chat

import (
	"context"
	"sync"
	"time"
)

// SessionStore is a thread-safe in-memory session store.
type SessionStore struct {
	mu   sync.RWMutex
	data map[string]*Session
}

// NewSessionStore creates an empty SessionStore.
func NewSessionStore() *SessionStore {
	return &SessionStore{data: make(map[string]*Session)}
}

// Get returns the session or nil if it does not exist.
func (s *SessionStore) Get(id string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[id]
}

// GetOrCreate returns an existing session or creates one with the given ID.
func (s *SessionStore) GetOrCreate(id string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.data[id]; ok {
		sess.LastActiveAt = time.Now()
		return sess
	}
	now := time.Now()
	sess := &Session{ID: id, CreatedAt: now, LastActiveAt: now}
	s.data[id] = sess
	return sess
}

// Set writes the session, replacing any existing entry.
func (s *SessionStore) Set(sess *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess.LastActiveAt = time.Now()
	s.data[sess.ID] = sess
}

// Delete removes a session.
func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
}

// Len returns the number of active sessions.
func (s *SessionStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// StartCleanup runs a background goroutine that evicts sessions older than ttl.
// It stops when ctx is cancelled.
func (s *SessionStore) StartCleanup(ctx context.Context, ttl, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.evict(ttl)
			}
		}
	}()
}

func (s *SessionStore) evict(ttl time.Duration) {
	cutoff := time.Now().Add(-ttl)
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, sess := range s.data {
		if sess.LastActiveAt.Before(cutoff) {
			delete(s.data, id)
		}
	}
}
