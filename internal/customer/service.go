package customer

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cccvno1/goplate/pkg/errkit"
	"github.com/cccvno1/ledger-agent/internal/domain"
	"github.com/google/uuid"
)

// Service implements customer business logic.
type Service struct {
	db    *sql.DB
	store *Store
}

// NewService creates a Service.
func NewService(db *sql.DB, store *Store) *Service {
	return &Service{db: db, store: store}
}

// CreateInput carries the intent to create a new customer.
type CreateInput struct {
	Name string
}

// Create creates a new customer. Returns ErrConflict if the name already exists.
func (s *Service) Create(ctx context.Context, in CreateInput) (*Customer, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, errkit.New(errkit.InvalidInput, "customer name is required")
	}

	existing, err := s.store.GetByName(ctx, in.Name)
	if err != nil {
		return nil, fmt.Errorf("customer: create: check existing: %w", err)
	}
	if existing != nil {
		return existing, nil // idempotent: return existing customer
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("customer: create: begin tx: %w", err)
	}
	defer tx.Rollback()

	c := &Customer{
		ID:        uuid.NewString(),
		Name:      in.Name,
		Aliases:   []string{},
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.Create(ctx, tx, c); err != nil {
		return nil, fmt.Errorf("customer: create: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("customer: create: commit: %w", err)
	}
	return c, nil
}

// GetByID returns the customer by ID, or ErrNotFound.
func (s *Service) GetByID(ctx context.Context, id string) (*Customer, error) {
	c, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("customer: get by id: %w", err)
	}
	if c == nil {
		return nil, domain.ErrNotFound
	}
	return c, nil
}

// ListAll returns all customers.
func (s *Service) ListAll(ctx context.Context) ([]*Customer, error) {
	customers, err := s.store.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("customer: list all: %w", err)
	}
	return customers, nil
}

// SearchResult is a candidate customer with a similarity score.
type SearchResult struct {
	Customer   *Customer
	Score      int    // lower is more similar (Levenshtein distance)
	MatchedVia string // "exact", "alias", "fuzzy"
}

// SearchInput carries the search query.
type SearchInput struct {
	Query string
	TopN  int // maximum candidates to return (default 5)
}

// Search performs multi-strategy matching: exact name → alias → Levenshtein.
func (s *Service) Search(ctx context.Context, in SearchInput) ([]*SearchResult, error) {
	if in.TopN <= 0 {
		in.TopN = 5
	}
	q := strings.TrimSpace(in.Query)

	// 1. Exact name match
	c, err := s.store.GetByName(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("customer: search: exact: %w", err)
	}
	if c != nil {
		return []*SearchResult{{Customer: c, Score: 0, MatchedVia: "exact"}}, nil
	}

	// 2. Alias match
	c, err = s.store.GetByAlias(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("customer: search: alias: %w", err)
	}
	if c != nil {
		return []*SearchResult{{Customer: c, Score: 0, MatchedVia: "alias"}}, nil
	}

	// 3. Fuzzy match via Levenshtein
	all, err := s.store.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("customer: search: %w", err)
	}

	query := strings.ToLower(q)
	results := make([]*SearchResult, 0, len(all))
	for _, cust := range all {
		dist := domain.Levenshtein(query, strings.ToLower(cust.Name))
		// Also check aliases and keep best score
		for _, alias := range cust.Aliases {
			if ad := domain.Levenshtein(query, strings.ToLower(alias)); ad < dist {
				dist = ad
			}
		}
		results = append(results, &SearchResult{Customer: cust, Score: dist, MatchedVia: "fuzzy"})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score < results[j].Score
	})

	if len(results) > in.TopN {
		results = results[:in.TopN]
	}
	return results, nil
}

// AddAlias appends an alias to a customer.
func (s *Service) AddAlias(ctx context.Context, customerID, alias string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("customer: add alias: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.store.AddAlias(ctx, tx, customerID, alias); err != nil {
		return fmt.Errorf("customer: add alias: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("customer: add alias: commit: %w", err)
	}
	return nil
}
