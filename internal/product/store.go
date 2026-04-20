package product

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

// Store handles database persistence for products.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store backed by the given database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Create inserts a new product row.
func (s *Store) Create(ctx context.Context, tx *sql.Tx, p *Product) error {
	const q = `
		INSERT INTO products (id, name, aliases, default_unit, reference_price, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := tx.ExecContext(ctx, q,
		p.ID, p.Name, pq.Array(p.Aliases), p.DefaultUnit, nilIfZero(p.ReferencePrice),
		p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("product store: create: %w", err)
	}
	return nil
}

// GetByName returns the product with the given exact name, or nil if not found.
func (s *Store) GetByName(ctx context.Context, name string) (*Product, error) {
	const q = `SELECT id, name, aliases, default_unit, reference_price, created_at, updated_at
		FROM products WHERE name = $1`
	return s.scanOne(s.db.QueryRowContext(ctx, q, name))
}

// GetByAlias returns the product whose aliases array contains the given value.
func (s *Store) GetByAlias(ctx context.Context, alias string) (*Product, error) {
	const q = `SELECT id, name, aliases, default_unit, reference_price, created_at, updated_at
		FROM products WHERE $1 = ANY(aliases)`
	return s.scanOne(s.db.QueryRowContext(ctx, q, alias))
}

// ListAll returns every product ordered by name.
func (s *Store) ListAll(ctx context.Context) ([]*Product, error) {
	const q = `SELECT id, name, aliases, default_unit, reference_price, created_at, updated_at
		FROM products ORDER BY name`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("product store: list all: %w", err)
	}
	defer rows.Close()

	var products []*Product
	for rows.Next() {
		p, err := s.scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("product store: list all scan: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("product store: list all rows: %w", err)
	}
	return products, nil
}

// AddAlias appends an alias to the product's aliases array.
func (s *Store) AddAlias(ctx context.Context, tx *sql.Tx, productID, alias string) error {
	const q = `UPDATE products SET aliases = array_append(aliases, $2), updated_at = now() WHERE id = $1`
	_, err := tx.ExecContext(ctx, q, productID, alias)
	if err != nil {
		return fmt.Errorf("product store: add alias: %w", err)
	}
	return nil
}

func (s *Store) scanOne(row *sql.Row) (*Product, error) {
	p := &Product{}
	var refPrice sql.NullFloat64
	err := row.Scan(&p.ID, &p.Name, pq.Array(&p.Aliases), &p.DefaultUnit, &refPrice, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("product store: scan: %w", err)
	}
	if refPrice.Valid {
		p.ReferencePrice = refPrice.Float64
	}
	return p, nil
}

func (s *Store) scanRow(rows *sql.Rows) (*Product, error) {
	p := &Product{}
	var refPrice sql.NullFloat64
	err := rows.Scan(&p.ID, &p.Name, pq.Array(&p.Aliases), &p.DefaultUnit, &refPrice, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("product store: row scan: %w", err)
	}
	if refPrice.Valid {
		p.ReferencePrice = refPrice.Float64
	}
	return p, nil
}

func nilIfZero(f float64) any {
	if f == 0 {
		return nil
	}
	return f
}
