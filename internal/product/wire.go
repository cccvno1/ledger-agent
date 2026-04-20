package product

import "database/sql"

// Wire assembles the product feature's dependencies.
func Wire(db *sql.DB) *Service {
	store := NewStore(db)
	return NewService(db, store)
}
