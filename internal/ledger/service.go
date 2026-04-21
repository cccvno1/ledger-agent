package ledger

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/cccvno1/goplate/pkg/errkit"
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
	customerID := strings.TrimSpace(in.CustomerID)
	customerName := strings.TrimSpace(in.CustomerName)
	productName := strings.TrimSpace(in.ProductName)
	if customerID == "" {
		return nil, errkit.New(errkit.InvalidInput, "customer_id is required")
	}
	if customerName == "" {
		return nil, errkit.New(errkit.InvalidInput, "customer_name is required")
	}
	if productName == "" {
		return nil, errkit.New(errkit.InvalidInput, "product_name is required")
	}
	if in.UnitPrice <= 0 {
		return nil, errkit.New(errkit.InvalidInput, "unit_price must be greater than 0")
	}
	if in.Quantity <= 0 {
		return nil, errkit.New(errkit.InvalidInput, "quantity must be greater than 0")
	}
	if in.EntryDate.IsZero() {
		return nil, errkit.New(errkit.InvalidInput, "entry_date is required")
	}

	now := time.Now().UTC()
	unit := strings.TrimSpace(in.Unit)
	if unit == "" {
		unit = "个"
	}
	e := &Entry{
		ID:           uuid.NewString(),
		CustomerID:   customerID,
		CustomerName: customerName,
		ProductName:  productName,
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
	if strings.TrimSpace(in.ID) == "" {
		return nil, errkit.New(errkit.InvalidInput, "id is required")
	}

	hasChanges := false
	if in.CustomerName != "" {
		if strings.TrimSpace(in.CustomerName) == "" {
			return nil, errkit.New(errkit.InvalidInput, "customer_name must not be blank")
		}
		hasChanges = true
	}
	if in.ProductName != "" {
		if strings.TrimSpace(in.ProductName) == "" {
			return nil, errkit.New(errkit.InvalidInput, "product_name must not be blank")
		}
		hasChanges = true
	}
	if in.UnitPrice != 0 {
		if in.UnitPrice <= 0 {
			return nil, errkit.New(errkit.InvalidInput, "unit_price must be greater than 0")
		}
		hasChanges = true
	}
	if in.Quantity != 0 {
		if in.Quantity <= 0 {
			return nil, errkit.New(errkit.InvalidInput, "quantity must be greater than 0")
		}
		hasChanges = true
	}
	if in.Unit != "" {
		if strings.TrimSpace(in.Unit) == "" {
			return nil, errkit.New(errkit.InvalidInput, "unit must not be blank")
		}
		hasChanges = true
	}
	if !in.EntryDate.IsZero() {
		hasChanges = true
	}
	if in.Notes != "" {
		hasChanges = true
	}
	if !hasChanges {
		return nil, errkit.New(errkit.InvalidInput, "at least one field must be updated")
	}

	e, err := s.store.GetByID(ctx, in.ID)
	if err != nil {
		return nil, fmt.Errorf("ledger: update: fetch: %w", err)
	}
	if e == nil {
		return nil, domain.ErrNotFound
	}

	if in.CustomerName != "" {
		e.CustomerName = strings.TrimSpace(in.CustomerName)
	}
	if in.ProductName != "" {
		e.ProductName = strings.TrimSpace(in.ProductName)
	}
	if in.UnitPrice > 0 {
		e.UnitPrice = in.UnitPrice
	}
	if in.Quantity > 0 {
		e.Quantity = in.Quantity
	}
	if in.Unit != "" {
		e.Unit = strings.TrimSpace(in.Unit)
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
	customerID = strings.TrimSpace(customerID)
	if customerID == "" {
		return errkit.New(errkit.InvalidInput, "customer_id is required")
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

// DashboardOutput holds aggregate metrics for the management dashboard.
type DashboardOutput struct {
	TotalPending     float64 `json:"total_pending"`
	TotalCustomers   int     `json:"total_customers"`
	EntriesThisMonth int     `json:"entries_this_month"`
	AmountThisMonth  float64 `json:"amount_this_month"`
}

// Dashboard returns aggregated metrics. It queries both entries and payments tables directly.
func (s *Service) Dashboard(ctx context.Context) (*DashboardOutput, error) {
	now := time.Now().UTC()
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var out DashboardOutput

	row := s.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(amount), 0),
			COUNT(DISTINCT customer_id),
			COUNT(CASE WHEN entry_date >= $1 THEN 1 END),
			COALESCE(SUM(CASE WHEN entry_date >= $1 THEN amount ELSE 0 END), 0)
		FROM entries`, firstOfMonth)
	if err := row.Scan(&out.TotalPending, &out.TotalCustomers, &out.EntriesThisMonth, &out.AmountThisMonth); err != nil {
		return nil, fmt.Errorf("ledger: dashboard: entries: %w", err)
	}

	// Subtract payments from pending (payments apply against unsettled balance).
	var totalPaid float64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount), 0) FROM payments`).Scan(&totalPaid); err != nil {
		return nil, fmt.Errorf("ledger: dashboard: payments: %w", err)
	}
	out.TotalPending -= totalPaid
	if out.TotalPending < 0 {
		out.TotalPending = 0
	}

	return &out, nil
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
