package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/cccvno1/goplate/pkg/errkit"
	"github.com/cccvno1/goplate/pkg/httpkit"
)

// New returns the middleware chain for the HTTP server.
// token is the expected Bearer token for the BearerAuth middleware.
func New(logger *slog.Logger, token string) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		httpkit.RequestID,
		httpkit.Logging(logger),
		httpkit.Recover(logger),
		BearerAuth(token),
	}
}

// BearerAuth returns a middleware that requires a valid Bearer token in the
// Authorization header. The /health endpoint is exempt from authentication.
// WeChat QR endpoints are also exempt (prefix /api/v1/wechat/qrcode).
func BearerAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if path == "/health" ||
				strings.HasPrefix(path, "/api/v1/wechat/qrcode") ||
				!strings.HasPrefix(path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			// No AUTH_TOKEN configured — skip authentication.
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}
			auth := r.Header.Get("Authorization")
			expected := "Bearer " + token
			if auth != expected {
				httpkit.Error(w, errkit.New(errkit.Unauthorized, "invalid or missing token"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
