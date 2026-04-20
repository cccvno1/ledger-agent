package conf

import (
	"github.com/cccvno1/goplate/pkg/logkit"
)

// Config holds all application configuration.
type Config struct {
	Log     logkit.Config `yaml:"log"`
	HTTP    HTTP          `yaml:"http"`
	DB      DB            `yaml:"db"`
	MiniMax MiniMax       `yaml:"minimax"`
}

// HTTP holds HTTP server configuration.
type HTTP struct {
	Addr         string `yaml:"addr"`
	ReadTimeout  int64  `yaml:"read_timeout"`
	WriteTimeout int64  `yaml:"write_timeout"`
}

// DB holds PostgreSQL connection configuration.
// DSN must be provided via the DATABASE_URL environment variable.
type DB struct {
	MaxOpenConns int `yaml:"max_open_conns"`
	MaxIdleConns int `yaml:"max_idle_conns"`
}

// MiniMax holds MiniMax LLM configuration.
// APIKey must be provided via the MINIMAX_API_KEY environment variable.
type MiniMax struct {
	BaseURL            string `yaml:"base_url"`
	Model              string `yaml:"model"`
	MaxHistoryMessages int    `yaml:"max_history_messages"`
}
