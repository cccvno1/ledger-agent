package product

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cccvno1/ledger-agent/internal/domain"
	"github.com/google/uuid"
)

// Service implements product catalog business logic.
type Service struct {
	db    *sql.DB
	store *Store
}

// NewService creates a Service.
func NewService(db *sql.DB, store *Store) *Service {
	return &Service{db: db, store: store}
}

// CreateInput carries the intent to create a new product.
type CreateInput struct {
	Name           string
	DefaultUnit    string
	ReferencePrice float64
}

// Create creates a new product.
func (s *Service) Create(ctx context.Context, in CreateInput) (*Product, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("product: create: name is required")
	}
	unit := in.DefaultUnit
	if unit == "" {
		unit = "个"
	}
	now := time.Now().UTC()
	p := &Product{
		ID:             uuid.NewString(),
		Name:           in.Name,
		Aliases:        []string{},
		DefaultUnit:    unit,
		ReferencePrice: in.ReferencePrice,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("product: create: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.store.Create(ctx, tx, p); err != nil {
		return nil, fmt.Errorf("product: create: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("product: create: commit: %w", err)
	}
	return p, nil
}

// SearchResult is a candidate product with a similarity score.
type SearchResult struct {
	Product    *Product
	Score      int    // Levenshtein distance (0 = exact)
	MatchedVia string // "exact", "alias", "fuzzy"
}

// Search performs multi-strategy matching: exact name → alias → Levenshtein.
func (s *Service) Search(ctx context.Context, query string, topN int) ([]*SearchResult, error) {
	if topN <= 0 {
		topN = 5
	}
	q := strings.TrimSpace(query)

	// 1. Exact name match
	p, err := s.store.GetByName(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("product: search: exact: %w", err)
	}
	if p != nil {
		return []*SearchResult{{Product: p, Score: 0, MatchedVia: "exact"}}, nil
	}

	// 2. Alias match
	p, err = s.store.GetByAlias(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("product: search: alias: %w", err)
	}
	if p != nil {
		return []*SearchResult{{Product: p, Score: 0, MatchedVia: "alias"}}, nil
	}

	// 3. Fuzzy match via Levenshtein across all products
	all, err := s.store.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("product: search: list: %w", err)
	}

	qLower := strings.ToLower(q)
	results := make([]*SearchResult, 0, len(all))
	for _, prod := range all {
		dist := domain.Levenshtein(qLower, strings.ToLower(prod.Name))
		// Also check aliases and keep best score
		for _, alias := range prod.Aliases {
			if ad := domain.Levenshtein(qLower, strings.ToLower(alias)); ad < dist {
				dist = ad
			}
		}
		results = append(results, &SearchResult{Product: prod, Score: dist, MatchedVia: "fuzzy"})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score < results[j].Score
	})
	if len(results) > topN {
		results = results[:topN]
	}
	return results, nil
}

// FindOrCreate looks up a product by name/alias, or creates it if not found.
func (s *Service) FindOrCreate(ctx context.Context, name string) (*Product, error) {
	results, err := s.Search(ctx, name, 1)
	if err != nil {
		return nil, fmt.Errorf("product: find or create: search: %w", err)
	}
	// Accept exact name or alias match only (score 0).
	if len(results) > 0 && results[0].Score == 0 {
		return results[0].Product, nil
	}
	return s.Create(ctx, CreateInput{Name: name})
}

// ListAll returns all products.
func (s *Service) ListAll(ctx context.Context) ([]*Product, error) {
	products, err := s.store.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("product: list all: %w", err)
	}
	return products, nil
}

// AddAlias appends an alias to a product.
func (s *Service) AddAlias(ctx context.Context, productID, alias string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("product: add alias: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.store.AddAlias(ctx, tx, productID, alias); err != nil {
		return fmt.Errorf("product: add alias: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("product: add alias: commit: %w", err)
	}
	return nil
}
