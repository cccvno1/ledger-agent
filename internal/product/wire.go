package product

import (
	"database/sql"
	"net/http"
)

// Wire assembles the product feature's dependencies and registers routes.
// Returns *Service for use by other features (e.g., chat adapters).
func Wire(mux *http.ServeMux, db *sql.DB) *Service {
	store := NewStore(db)
	svc := NewService(db, store)
	h := NewHandler(svc)
	mux.HandleFunc("GET /api/v1/products", h.List)
	return svc
}
