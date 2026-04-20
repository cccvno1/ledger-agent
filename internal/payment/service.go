package payment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
	p := &Payment{
		ID:          uuid.NewString(),
		CustomerID:  in.CustomerID,
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
