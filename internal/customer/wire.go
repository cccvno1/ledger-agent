package customer

import (
	"database/sql"
	"net/http"
)

// Wire assembles the customer feature's dependencies and registers routes.
// Returns *Service for use by other features (e.g., chat adapters).
func Wire(mux *http.ServeMux, db *sql.DB) *Service {
	store := NewStore(db)
	svc := NewService(db, store)
	h := NewHandler(svc)
	mux.HandleFunc("GET /api/v1/customers", h.List)
	mux.HandleFunc("GET /api/v1/customers/{id}", h.Get)
	mux.HandleFunc("POST /api/v1/customers", h.Create)
	return svc
}
