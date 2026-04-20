package router

import (
	"log/slog"
	"net/http"

	"github.com/cccvno1/goplate/pkg/httpkit"
)

// Register adds routes to the given mux.
func Register(mux *http.ServeMux, logger *slog.Logger) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		httpkit.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
}
