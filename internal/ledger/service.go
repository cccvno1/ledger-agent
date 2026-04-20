package ledger

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cccvno1/ledger-agent/internal/domain"
	"github.com/google/uuid"
)

// Service implements ledger business logic.
type Service struct {
	db    *sql.DB
	store *Store
}

// NewService creates a Service.
func NewService(db *sql.DB, store *Store) *Service {
	return &Service{db: db, store: store}
}

// CreateInput carries the fields for a new entry.
type CreateInput struct {
	CustomerID   string
	CustomerName string
	ProductName  string
	UnitPrice    float64
	Quantity     float64
	Unit         string
	EntryDate    time.Time
	Notes        string
}

// Create inserts a new ledger entry.
func (s *Service) Create(ctx context.Context, in CreateInput) (*Entry, error) {
	now := time.Now().UTC()
	unit := in.Unit
	if unit == "" {
		unit = "个"
	}
	e := &Entry{
		ID:           uuid.NewString(),
		CustomerID:   in.CustomerID,
		CustomerName: in.CustomerName,
		ProductName:  in.ProductName,
		UnitPrice:    in.UnitPrice,
		Quantity:     in.Quantity,
		Unit:         unit,
		Amount:       in.UnitPrice * in.Quantity,
		EntryDate:    in.EntryDate,
		Notes:        in.Notes,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("ledger: create: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.store.Create(ctx, tx, e); err != nil {
		return nil, fmt.Errorf("ledger: create: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("ledger: create: commit: %w", err)
	}
	return e, nil
}

// GetByID returns an entry or ErrNotFound.
func (s *Service) GetByID(ctx context.Context, id string) (*Entry, error) {
	e, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ledger: get by id: %w", err)
	}
	if e == nil {
		return nil, domain.ErrNotFound
	}
	return e, nil
}

// ListInput controls query parameters.
type ListInput struct {
	CustomerID   string
	CustomerName string
	DateFrom     *time.Time
	DateTo       *time.Time
	IsSettled    *bool
}

// List returns entries matching the input filter.
func (s *Service) List(ctx context.Context, in ListInput) ([]*Entry, error) {
	entries, err := s.store.List(ctx, ListFilter(in))
	if err != nil {
		return nil, fmt.Errorf("ledger: list: %w", err)
	}
	return entries, nil
}

// UpdateInput carries mutable fields. Only non-zero fields are considered.
type UpdateInput struct {
	ID           string
	CustomerName string
	ProductName  string
	UnitPrice    float64
	Quantity     float64
	Unit         string
	EntryDate    time.Time
	Notes        string
}

// Update modifies an existing entry.
func (s *Service) Update(ctx context.Context, in UpdateInput) (*Entry, error) {
	e, err := s.store.GetByID(ctx, in.ID)
	if err != nil {
		return nil, fmt.Errorf("ledger: update: fetch: %w", err)
	}
	if e == nil {
		return nil, domain.ErrNotFound
	}

	if in.CustomerName != "" {
		e.CustomerName = in.CustomerName
	}
	if in.ProductName != "" {
		e.ProductName = in.ProductName
	}
	if in.UnitPrice > 0 {
		e.UnitPrice = in.UnitPrice
	}
	if in.Quantity > 0 {
		e.Quantity = in.Quantity
	}
	if in.Unit != "" {
		e.Unit = in.Unit
	}
	if !in.EntryDate.IsZero() {
		e.EntryDate = in.EntryDate
	}
	if in.Notes != "" {
		e.Notes = in.Notes
	}
	e.Amount = e.UnitPrice * e.Quantity
	e.UpdatedAt = time.Now().UTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("ledger: update: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.store.Update(ctx, tx, e); err != nil {
		return nil, fmt.Errorf("ledger: update: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("ledger: update: commit: %w", err)
	}
	return e, nil
}

// SettleByCustomer marks all unsettled entries for a customer as settled.
func (s *Service) SettleByCustomer(ctx context.Context, customerID string) error {
	if customerID == "" {
		return fmt.Errorf("ledger: settle: customer_id is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ledger: settle: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.store.MarkSettled(ctx, tx, customerID, time.Now().UTC()); err != nil {
		return fmt.Errorf("ledger: settle: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ledger: settle: commit: %w", err)
	}
	return nil
}

// SummaryByCustomer returns outstanding totals. If customerID is empty, returns all customers.
func (s *Service) SummaryByCustomer(ctx context.Context, customerID string) ([]*CustomerSummary, error) {
	summaries, err := s.store.SumByCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("ledger: summary: %w", err)
	}
	return summaries, nil
}

// DeleteInput carries the ID of the entry to delete.
type DeleteInput struct {
	ID string
}

// Delete removes a ledger entry by ID.
func (s *Service) Delete(ctx context.Context, in DeleteInput) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ledger: delete: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.store.Delete(ctx, tx, in.ID); err != nil {
		return fmt.Errorf("ledger: delete: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ledger: delete: commit: %w", err)
	}
	return nil
}
