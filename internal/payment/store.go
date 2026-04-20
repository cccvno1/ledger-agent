package payment

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store handles database persistence for payments.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store backed by the given database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Create inserts a new payment row.
func (s *Store) Create(ctx context.Context, tx *sql.Tx, p *Payment) error {
	const q = `INSERT INTO payments (id, customer_id, amount, payment_date, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := tx.ExecContext(ctx, q,
		p.ID, p.CustomerID, p.Amount, p.PaymentDate.Format("2006-01-02"), p.Notes, p.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("payment store: create: %w", err)
	}
	return nil
}

// ListByCustomer returns all payments for a customer ordered by date descending.
func (s *Store) ListByCustomer(ctx context.Context, customerID string) ([]*Payment, error) {
	const q = `SELECT id, customer_id, amount, payment_date, notes, created_at
		FROM payments WHERE customer_id = $1 ORDER BY payment_date DESC, created_at DESC`
	rows, err := s.db.QueryContext(ctx, q, customerID)
	if err != nil {
		return nil, fmt.Errorf("payment store: list by customer: %w", err)
	}
	defer rows.Close()

	var payments []*Payment
	for rows.Next() {
		p, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("payment store: list scan: %w", err)
		}
		payments = append(payments, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("payment store: list rows: %w", err)
	}
	return payments, nil
}

// TotalByCustomer returns the cumulative payment total for a customer.
func (s *Store) TotalByCustomer(ctx context.Context, customerID string) (float64, error) {
	const q = `SELECT COALESCE(SUM(amount), 0) FROM payments WHERE customer_id = $1`
	var total float64
	if err := s.db.QueryRowContext(ctx, q, customerID).Scan(&total); err != nil {
		return 0, fmt.Errorf("payment store: total by customer: %w", err)
	}
	return total, nil
}

func scanRow(rows *sql.Rows) (*Payment, error) {
	p := &Payment{}
	var payDate time.Time
	err := rows.Scan(&p.ID, &p.CustomerID, &p.Amount, &payDate, &p.Notes, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("payment store: row scan: %w", err)
	}
	p.PaymentDate = payDate
	return p, nil
}
