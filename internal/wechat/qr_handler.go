package wechat

import (
	"log/slog"
	"net/http"

	"github.com/cccvno1/goplate/pkg/httpkit"
)

// qrHandler serves the WeChat QR code login flow over HTTP.
type qrHandler struct {
	logger      *slog.Logger
	onConnected func(*Credentials) // called after credentials are saved; may be nil
}

func newQRHandler(logger *slog.Logger, onConnected func(*Credentials)) *qrHandler {
	return &qrHandler{logger: logger, onConnected: onConnected}
}

// GenerateQRCode handles POST /api/v1/wechat/qrcode.
// Returns a qrcode token and the raw content string that the frontend renders as a QR image.
func (h *qrHandler) GenerateQRCode(w http.ResponseWriter, r *http.Request) {
	qrcode, imgContent, err := FetchQRCode(r.Context())
	if err != nil {
		h.logger.Error("wechat: fetch QR code", "err", err)
		httpkit.Error(w, err)
		return
	}
	httpkit.JSON(w, http.StatusOK, map[string]string{
		"qrcode":      qrcode,
		"img_content": imgContent,
	})
}

// CheckStatus handles GET /api/v1/wechat/qrcode/status?qrcode=<token>.
// When status is "confirmed", credentials are persisted to disk.
func (h *qrHandler) CheckStatus(w http.ResponseWriter, r *http.Request) {
	qrcode := r.URL.Query().Get("qrcode")
	if qrcode == "" {
		httpkit.JSON(w, http.StatusOK, map[string]string{"status": qrStatusWait})
		return
	}

	status, creds, err := CheckQRStatus(r.Context(), qrcode)
	if err != nil {
		h.logger.Error("wechat: check QR status", "err", err)
		httpkit.Error(w, err)
		return
	}

	if status == qrStatusConfirmed && creds != nil {
		if saveErr := SaveCredentials(creds); saveErr != nil {
			h.logger.Error("wechat: save credentials", "err", saveErr)
		} else if h.onConnected != nil {
			go h.onConnected(creds)
		}
	}

	httpkit.JSON(w, http.StatusOK, map[string]string{"status": status})
}
