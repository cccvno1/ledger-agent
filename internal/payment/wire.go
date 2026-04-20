package payment

import "database/sql"

// Wire assembles the payment feature's dependencies.
func Wire(db *sql.DB) *Service {
	store := NewStore(db)
	return NewService(db, store)
}
