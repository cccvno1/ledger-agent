package ledger

import "database/sql"

// Wire assembles the ledger feature's dependencies.
func Wire(db *sql.DB) *Service {
	store := NewStore(db)
	return NewService(db, store)
}
