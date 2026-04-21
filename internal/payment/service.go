package payment

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/cccvno1/goplate/pkg/errkit"
	"github.com/google/uuid"
)

// Service implements payment business logic.
type Service struct {
	db    *sql.DB
	store *Store
}

// NewService creates a Service.
func NewService(db *sql.DB, store *Store) *Service {
	return &Service{db: db, store: store}
}

// CreateInput carries the intent to record a payment.
type CreateInput struct {
	CustomerID  string
	Amount      float64
	PaymentDate time.Time
	Notes       string
}

// Create records a new payment.
func (s *Service) Create(ctx context.Context, in CreateInput) (*Payment, error) {
	customerID := strings.TrimSpace(in.CustomerID)
	if customerID == "" {
		return nil, errkit.New(errkit.InvalidInput, "customer_id is required")
	}
	if in.Amount <= 0 {
		return nil, errkit.New(errkit.InvalidInput, "amount must be greater than 0")
	}
	if in.PaymentDate.IsZero() {
		return nil, errkit.New(errkit.InvalidInput, "payment_date is required")
	}

	pending, err := s.currentPending(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("payment: create: pending: %w", err)
	}
	if pending <= 0 {
		return nil, errkit.New(errkit.InvalidInput, "customer has no pending balance")
	}
	if in.Amount > pending {
		return nil, errkit.New(errkit.InvalidInput, "amount exceeds pending balance")
	}

	p := &Payment{
		ID:          uuid.NewString(),
		CustomerID:  customerID,
		Amount:      in.Amount,
		PaymentDate: in.PaymentDate,
		Notes:       in.Notes,
		CreatedAt:   time.Now().UTC(),
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("payment: create: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.store.Create(ctx, tx, p); err != nil {
		return nil, fmt.Errorf("payment: create: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("payment: create: commit: %w", err)
	}
	return p, nil
}

func (s *Service) currentPending(ctx context.Context, customerID string) (float64, error) {
	var totalEntries float64
	if err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM entries WHERE customer_id = $1`,
		customerID,
	).Scan(&totalEntries); err != nil {
		return 0, fmt.Errorf("query entries total: %w", err)
	}

	var totalPayments float64
	if err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM payments WHERE customer_id = $1`,
		customerID,
	).Scan(&totalPayments); err != nil {
		return 0, fmt.Errorf("query payments total: %w", err)
	}

	pending := totalEntries - totalPayments
	if pending < 0 {
		pending = 0
	}
	return pending, nil
}

// ListByCustomer returns all payments for a customer.
func (s *Service) ListByCustomer(ctx context.Context, customerID string) ([]*Payment, error) {
	payments, err := s.store.ListByCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("payment: list by customer: %w", err)
	}
	return payments, nil
}

// TotalByCustomer returns the cumulative payment total for a customer.
func (s *Service) TotalByCustomer(ctx context.Context, customerID string) (float64, error) {
	total, err := s.store.TotalByCustomer(ctx, customerID)
	if err != nil {
		return 0, fmt.Errorf("payment: total by customer: %w", err)
	}
	return total, nil
}
