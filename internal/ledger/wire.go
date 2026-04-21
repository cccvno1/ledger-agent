package ledger

import (
	"database/sql"
	"net/http"
)

// Wire assembles the ledger feature's dependencies and registers routes.
// Returns *Service for use by other features (e.g., chat adapters).
func Wire(mux *http.ServeMux, db *sql.DB) *Service {
	store := NewStore(db)
	svc := NewService(db, store)
	h := NewHandler(svc)
	mux.HandleFunc("GET /api/v1/entries", h.List)
	mux.HandleFunc("POST /api/v1/entries", h.Create)
	mux.HandleFunc("PUT /api/v1/entries/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/entries/{id}", h.Delete)
	mux.HandleFunc("GET /api/v1/customers/{customer_id}/summary", h.Summary)
	mux.HandleFunc("POST /api/v1/customers/{customer_id}/settle", h.Settle)
	mux.HandleFunc("GET /api/v1/dashboard", h.Dashboard)
	return svc
}
