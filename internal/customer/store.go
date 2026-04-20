package customer

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

// Store handles database persistence for customers.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store backed by the given database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Create inserts a new customer row. Uses tx for transactional writes.
func (s *Store) Create(ctx context.Context, tx *sql.Tx, c *Customer) error {
	const q = `INSERT INTO customers (id, name, aliases, created_at) VALUES ($1, $2, $3, $4)`
	_, err := tx.ExecContext(ctx, q, c.ID, c.Name, pq.Array(c.Aliases), c.CreatedAt)
	if err != nil {
		return fmt.Errorf("customer store: create: %w", err)
	}
	return nil
}

// GetByID returns the customer with the given ID.
func (s *Store) GetByID(ctx context.Context, id string) (*Customer, error) {
	const q = `SELECT id, name, aliases, created_at FROM customers WHERE id = $1`
	row := s.db.QueryRowContext(ctx, q, id)
	c := &Customer{}
	if err := row.Scan(&c.ID, &c.Name, pq.Array(&c.Aliases), &c.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("customer store: get by id: %w", err)
	}
	return c, nil
}

// GetByName returns the customer with the given exact name, or nil if not found.
func (s *Store) GetByName(ctx context.Context, name string) (*Customer, error) {
	const q = `SELECT id, name, aliases, created_at FROM customers WHERE name = $1`
	row := s.db.QueryRowContext(ctx, q, name)
	c := &Customer{}
	if err := row.Scan(&c.ID, &c.Name, pq.Array(&c.Aliases), &c.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("customer store: get by name: %w", err)
	}
	return c, nil
}

// ListAll returns every customer ordered by name.
func (s *Store) ListAll(ctx context.Context) ([]*Customer, error) {
	const q = `SELECT id, name, aliases, created_at FROM customers ORDER BY name`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("customer store: list all: %w", err)
	}
	defer rows.Close()

	var customers []*Customer
	for rows.Next() {
		c := &Customer{}
		if err := rows.Scan(&c.ID, &c.Name, pq.Array(&c.Aliases), &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("customer store: list all scan: %w", err)
		}
		customers = append(customers, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("customer store: list all rows: %w", err)
	}
	return customers, nil
}

// GetByAlias returns the customer whose aliases array contains the given value.
func (s *Store) GetByAlias(ctx context.Context, alias string) (*Customer, error) {
	const q = `SELECT id, name, aliases, created_at FROM customers WHERE $1 = ANY(aliases)`
	row := s.db.QueryRowContext(ctx, q, alias)
	c := &Customer{}
	if err := row.Scan(&c.ID, &c.Name, pq.Array(&c.Aliases), &c.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("customer store: get by alias: %w", err)
	}
	return c, nil
}

// AddAlias appends an alias to the customer's aliases array.
func (s *Store) AddAlias(ctx context.Context, tx *sql.Tx, customerID, alias string) error {
	const q = `UPDATE customers SET aliases = array_append(aliases, $2) WHERE id = $1`
	_, err := tx.ExecContext(ctx, q, customerID, alias)
	if err != nil {
		return fmt.Errorf("customer store: add alias: %w", err)
	}
	return nil
}
