package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
)

// SessionStorer is the interface that any session store must implement.
type SessionStorer interface {
	Get(id string) *Session
	GetOrCreate(id string) *Session
	Set(sess *Session)
}

// sessionRecord is the JSON-serializable shape stored in the database.
// It omits the sync.Mutex that Session carries at runtime.
type sessionRecord struct {
	Messages     []*schema.Message `json:"messages"`
	Draft        []DraftEntry      `json:"draft"`
	OpLog        []OpEntry         `json:"op_log"`
	Stats        SessionStats      `json:"stats"`
	CreatedAt    time.Time         `json:"created_at"`
	LastActiveAt time.Time         `json:"last_active_at"`
}

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

// DBSessionStore is a Postgres-backed session store.
// Sessions are stored as JSONB in the sessions table.
type DBSessionStore struct {
	db *sql.DB
}

// NewDBSessionStore creates a DBSessionStore.
func NewDBSessionStore(db *sql.DB) *DBSessionStore {
	return &DBSessionStore{db: db}
}

// Get returns the session by ID, or nil if not found.
func (s *DBSessionStore) Get(id string) *Session {
	sess, _ := s.load(context.Background(), id)
	return sess
}

// GetOrCreate returns an existing session or creates a new one.
func (s *DBSessionStore) GetOrCreate(id string) *Session {
	ctx := context.Background()
	if sess, err := s.load(ctx, id); err == nil && sess != nil {
		sess.LastActiveAt = time.Now()
		return sess
	}
	now := time.Now()
	sess := &Session{ID: id, CreatedAt: now, LastActiveAt: now}
	rec, _ := json.Marshal(toRecord(sess))
	_, _ = s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, data, created_at, updated_at) VALUES ($1, $2, $3, $3)
		 ON CONFLICT (id) DO NOTHING`,
		id, rec, now)
	return sess
}

// Set persists the session to the database.
func (s *DBSessionStore) Set(sess *Session) {
	sess.LastActiveAt = time.Now()
	rec, err := json.Marshal(toRecord(sess))
	if err != nil {
		return
	}
	_, _ = s.db.ExecContext(context.Background(),
		`INSERT INTO sessions (id, data, created_at, updated_at) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data, updated_at = EXCLUDED.updated_at`,
		sess.ID, rec, sess.CreatedAt, sess.LastActiveAt)
}

func (s *DBSessionStore) load(ctx context.Context, id string) (*Session, error) {
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM sessions WHERE id = $1`, id).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("chat: session load: %w", err)
	}
	var rec sessionRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, fmt.Errorf("chat: session unmarshal: %w", err)
	}
	return fromRecord(id, &rec), nil
}

func toRecord(sess *Session) *sessionRecord {
	return &sessionRecord{
		Messages:     sess.Messages,
		Draft:        sess.Draft,
		OpLog:        sess.OpLog,
		Stats:        sess.Stats,
		CreatedAt:    sess.CreatedAt,
		LastActiveAt: sess.LastActiveAt,
	}
}

func fromRecord(id string, rec *sessionRecord) *Session {
	return &Session{
		ID:           id,
		Messages:     rec.Messages,
		Draft:        rec.Draft,
		OpLog:        rec.OpLog,
		Stats:        rec.Stats,
		CreatedAt:    rec.CreatedAt,
		LastActiveAt: rec.LastActiveAt,
	}
}
