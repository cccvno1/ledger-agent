package main

import (
	"context"
	"os"

	"github.com/cccvno1/ledger-agent/internal/base/boot"
	"github.com/cccvno1/ledger-agent/internal/base/conf"
	"github.com/cccvno1/goplate/pkg/confkit"
	"github.com/cccvno1/goplate/pkg/logkit"
)

func main() {
	var cfg conf.Config
	if err := confkit.Load(&cfg,
		confkit.WithConfigRoot("./configs"),
		confkit.WithEnv(os.Getenv("APP_ENV")),
	); err != nil {
		panic(err)
	}

	logger := logkit.New(cfg.Log, logkit.Fields{
		Service: "ledger-agent",
		Env:     os.Getenv("APP_ENV"),
	})

	if err := boot.Run(context.Background(), logger, &cfg); err != nil {
		logger.Error("fatal", "error", err)
		os.Exit(1)
	}
}
