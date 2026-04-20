package wechat

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

const (
	maxFailures           = 5
	initialBackoff        = 3 * time.Second
	maxBackoff            = 60 * time.Second
	sessionExpiredBackoff = 5 * time.Second
)

type monitor struct {
	c       *client
	handler *handler
	buf     string
	bufPath string
	logger  *slog.Logger
}

func newMonitor(c *client, h *handler, logger *slog.Logger) (*monitor, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	m := &monitor{
		c:       c,
		handler: h,
		bufPath: filepath.Join(home, ".ledger-agent", "wechat-sync.json"),
		logger:  logger,
	}
	m.loadBuf()
	return m, nil
}

// run starts the long-poll loop. It blocks until ctx is cancelled.
func (m *monitor) run(ctx context.Context) {
	m.logger.Info("wechat: monitor started")
	failures := 0

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("wechat: monitor stopped")
			return
		default:
		}

		resp, err := m.c.getUpdates(ctx, m.buf)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			failures++
			backoff := calcBackoff(failures)
			m.logger.Warn("wechat: getUpdates failed", "failures", failures, "backoff", backoff, "err", err)
			if failures >= maxFailures {
				m.logger.Error("wechat: too many consecutive failures — run ledger-wechat-login to re-authenticate")
			}
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			continue
		}

		failures = 0

		if resp.ErrCode == errCodeSessionExpired {
			if m.buf != "" {
				m.logger.Warn("wechat: session expired, resetting sync buf")
				m.buf = ""
				m.saveBuf()
			} else {
				m.logger.Error("wechat: WeChat session fully expired — run ledger-wechat-login to re-authenticate")
			}
			select {
			case <-time.After(sessionExpiredBackoff):
			case <-ctx.Done():
				return
			}
			continue
		}

		if resp.GetUpdatesBuf != "" {
			m.buf = resp.GetUpdatesBuf
			m.saveBuf()
		}

		for _, msg := range resp.Msgs {
			go m.handler.handle(ctx, m.c, msg)
		}
	}
}

func calcBackoff(failures int) time.Duration {
	d := initialBackoff
	for i := 1; i < failures; i++ {
		d *= 2
		if d > maxBackoff {
			return maxBackoff
		}
	}
	return d
}

type syncData struct {
	Buf string `json:"buf"`
}

func (m *monitor) loadBuf() {
	data, err := os.ReadFile(m.bufPath)
	if err != nil {
		return
	}
	var s syncData
	if json.Unmarshal(data, &s) == nil {
		m.buf = s.Buf
	}
}

func (m *monitor) saveBuf() {
	if err := os.MkdirAll(filepath.Dir(m.bufPath), 0o700); err != nil {
		return
	}
	data, _ := json.Marshal(syncData{Buf: m.buf})
	_ = os.WriteFile(m.bufPath, data, 0o600)
}
