package boot

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/cccvno1/ledger-agent/internal/base/conf"
	"github.com/cccvno1/ledger-agent/internal/base/middleware"
	"github.com/cccvno1/ledger-agent/internal/base/router"
	"github.com/cccvno1/goplate/pkg/appkit"
)

type httpServer struct {
	srv *http.Server
}

func (h *httpServer) Start(_ context.Context) error {
	return h.srv.ListenAndServe()
}

func (h *httpServer) Stop(ctx context.Context) error {
	return h.srv.Shutdown(ctx)
}

func wireHTTP(_ context.Context, app *appkit.App, logger *slog.Logger, cfg *conf.Config, mux *http.ServeMux) error {
	router.Register(mux, logger)

	chain := middleware.New(logger)
	var handler http.Handler = mux
	for i := len(chain) - 1; i >= 0; i-- {
		handler = chain[i](handler)
	}

	srv := &httpServer{
		srv: &http.Server{
			Addr:         cfg.HTTP.Addr,
			Handler:      handler,
			ReadTimeout:  time.Duration(cfg.HTTP.ReadTimeout) * time.Second,
			WriteTimeout: time.Duration(cfg.HTTP.WriteTimeout) * time.Second,
		},
	}
	app.Add(srv)
	return nil
}
