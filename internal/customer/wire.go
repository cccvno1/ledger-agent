package customer

import "database/sql"

// Wire assembles the customer feature's dependencies.
func Wire(db *sql.DB) *Service {
	store := NewStore(db)
	return NewService(db, store)
}
