package middleware

import (
	"log/slog"
	"net/http"

	"github.com/cccvno1/goplate/pkg/httpkit"
)

// New returns the middleware chain for the HTTP server.
func New(logger *slog.Logger) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		httpkit.RequestID,
		httpkit.Logging(logger),
		httpkit.Recover(logger),
	}
}
