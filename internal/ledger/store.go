package ledger

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Store handles database persistence for ledger entries.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store backed by the given database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Create inserts a new entry row.
func (s *Store) Create(ctx context.Context, tx *sql.Tx, e *Entry) error {
	const q = `
		INSERT INTO entries
			(id, customer_id, customer_name, product_name, unit_price, quantity, unit, amount,
			 entry_date, is_settled, settled_at, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`
	_, err := tx.ExecContext(ctx, q,
		e.ID, e.CustomerID, e.CustomerName, e.ProductName,
		e.UnitPrice, e.Quantity, e.Unit, e.Amount,
		e.EntryDate.Format("2006-01-02"), e.IsSettled, e.SettledAt,
		e.Notes, e.CreatedAt, e.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("ledger store: create: %w", err)
	}
	return nil
}

// GetByID returns the entry with the given ID, or nil if not found.
func (s *Store) GetByID(ctx context.Context, id string) (*Entry, error) {
	const q = `
		SELECT id, customer_id, customer_name, product_name, unit_price, quantity, unit, amount,
		       entry_date, is_settled, settled_at, notes, created_at, updated_at
		FROM entries WHERE id = $1`
	row := s.db.QueryRowContext(ctx, q, id)
	return scanEntry(row)
}

// ListFilter controls which entries to fetch.
type ListFilter struct {
	CustomerID   string
	CustomerName string
	DateFrom     *time.Time
	DateTo       *time.Time
	IsSettled    *bool
}

// List returns entries matching the filter.
func (s *Store) List(ctx context.Context, f ListFilter) ([]*Entry, error) {
	var (
		where []string
		args  []any
		idx   = 1
	)
	addArg := func(cond string, val any) {
		where = append(where, fmt.Sprintf(cond, idx))
		args = append(args, val)
		idx++
	}

	if f.CustomerID != "" {
		addArg("customer_id = $%d", f.CustomerID)
	}
	if f.CustomerName != "" {
		addArg("customer_name ILIKE $%d", "%"+f.CustomerName+"%")
	}
	if f.DateFrom != nil {
		addArg("entry_date >= $%d", f.DateFrom.Format("2006-01-02"))
	}
	if f.DateTo != nil {
		addArg("entry_date <= $%d", f.DateTo.Format("2006-01-02"))
	}
	if f.IsSettled != nil {
		addArg("is_settled = $%d", *f.IsSettled)
	}

	q := `SELECT id, customer_id, customer_name, product_name, unit_price, quantity, unit, amount,
		         entry_date, is_settled, settled_at, notes, created_at, updated_at
		  FROM entries`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY entry_date DESC, created_at DESC"

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("ledger store: list: %w", err)
	}
	defer rows.Close()

	var entries []*Entry
	for rows.Next() {
		e, err := scanEntryRow(rows)
		if err != nil {
			return nil, fmt.Errorf("ledger store: list scan: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ledger store: list rows: %w", err)
	}
	return entries, nil
}

// Update updates mutable fields of an entry.
func (s *Store) Update(ctx context.Context, tx *sql.Tx, e *Entry) error {
	const q = `
		UPDATE entries SET
			customer_name=$2, product_name=$3, unit_price=$4, quantity=$5, unit=$6, amount=$7,
			entry_date=$8, notes=$9, updated_at=$10
		WHERE id=$1`
	_, err := tx.ExecContext(ctx, q,
		e.ID, e.CustomerName, e.ProductName, e.UnitPrice, e.Quantity, e.Unit, e.Amount,
		e.EntryDate.Format("2006-01-02"), e.Notes, e.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("ledger store: update: %w", err)
	}
	return nil
}

// MarkSettled sets is_settled=true and settled_at=now for all unsettled entries of a customer.
func (s *Store) MarkSettled(ctx context.Context, tx *sql.Tx, customerID string, now time.Time) error {
	const q = `
		UPDATE entries SET is_settled=true, settled_at=$2, updated_at=$2
		WHERE customer_id=$1 AND is_settled=false`
	_, err := tx.ExecContext(ctx, q, customerID, now)
	if err != nil {
		return fmt.Errorf("ledger store: mark settled: %w", err)
	}
	return nil
}

// SumByCustomer returns customer totals derived from entries and payments.
func (s *Store) SumByCustomer(ctx context.Context, customerID string) ([]*CustomerSummary, error) {
	q := `
		WITH entry_sums AS (
			SELECT customer_id, customer_name,
			       COALESCE(SUM(amount), 0) AS total,
			       COUNT(*)                  AS cnt
			FROM entries`
	var args []any
	if customerID != "" {
		q += " WHERE customer_id = $1"
		args = append(args, customerID)
	}
	q += `
			GROUP BY customer_id, customer_name
		),
		payment_sums AS (
			SELECT customer_id, COALESCE(SUM(amount), 0) AS paid
			FROM payments
			GROUP BY customer_id
		)
		SELECT e.customer_id,
		       e.customer_name,
		       e.total,
		       LEAST(e.total, COALESCE(p.paid, 0))      AS settled,
		       GREATEST(e.total - COALESCE(p.paid, 0), 0) AS pending,
		       e.cnt
		FROM entry_sums e
		LEFT JOIN payment_sums p ON p.customer_id = e.customer_id
		ORDER BY e.customer_name`

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("ledger store: sum by customer: %w", err)
	}
	defer rows.Close()

	var summaries []*CustomerSummary
	for rows.Next() {
		cs := &CustomerSummary{}
		if err := rows.Scan(&cs.CustomerID, &cs.CustomerName,
			&cs.TotalAmount, &cs.SettledAmount, &cs.PendingAmount, &cs.EntryCount); err != nil {
			return nil, fmt.Errorf("ledger store: sum scan: %w", err)
		}
		summaries = append(summaries, cs)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ledger store: sum rows: %w", err)
	}
	return summaries, nil
}

// scanEntry scans a single row from *sql.Row.
func scanEntry(row *sql.Row) (*Entry, error) {
	e := &Entry{}
	var entryDate string
	err := row.Scan(
		&e.ID, &e.CustomerID, &e.CustomerName, &e.ProductName,
		&e.UnitPrice, &e.Quantity, &e.Unit, &e.Amount,
		&entryDate, &e.IsSettled, &e.SettledAt,
		&e.Notes, &e.CreatedAt, &e.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ledger store: scan: %w", err)
	}
	t, err := time.Parse("2006-01-02", entryDate)
	if err != nil {
		// Postgres may return full timestamp; try that.
		t, err = time.Parse("2006-01-02T15:04:05Z07:00", entryDate)
		if err != nil {
			return nil, fmt.Errorf("ledger store: parse entry_date %q: %w", entryDate, err)
		}
	}
	e.EntryDate = t
	return e, nil
}

// scanEntryRow scans a single row from *sql.Rows.
func scanEntryRow(rows *sql.Rows) (*Entry, error) {
	e := &Entry{}
	var entryDate string
	err := rows.Scan(
		&e.ID, &e.CustomerID, &e.CustomerName, &e.ProductName,
		&e.UnitPrice, &e.Quantity, &e.Unit, &e.Amount,
		&entryDate, &e.IsSettled, &e.SettledAt,
		&e.Notes, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("ledger store: row scan: %w", err)
	}
	t, err := time.Parse("2006-01-02", entryDate)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05Z07:00", entryDate)
		if err != nil {
			return nil, fmt.Errorf("ledger store: parse entry_date %q: %w", entryDate, err)
		}
	}
	e.EntryDate = t
	return e, nil
}

// Delete removes an entry by ID.
func (s *Store) Delete(ctx context.Context, tx *sql.Tx, id string) error {
	const q = `DELETE FROM entries WHERE id = $1`
	_, err := tx.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("ledger store: delete: %w", err)
	}
	return nil
}
