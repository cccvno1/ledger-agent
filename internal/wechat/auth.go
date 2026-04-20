package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	qrCodeURL   = "https://ilinkai.weixin.qq.com/ilink/bot/get_bot_qrcode?bot_type=3"
	qrStatusURL = "https://ilinkai.weixin.qq.com/ilink/bot/get_qrcode_status?qrcode="

	qrStatusWait      = "wait"
	qrStatusScanned   = "scaned"
	qrStatusConfirmed = "confirmed"
	qrStatusExpired   = "expired"
)

type qrCodeResp struct {
	QRCode           string `json:"qrcode"`
	QRCodeImgContent string `json:"qrcode_img_content"` // QR code content string to be rendered
}

type qrStatusResp struct {
	Status     string `json:"status"`
	BotToken   string `json:"bot_token"`
	ILinkBotID string `json:"ilink_bot_id"`
	BaseURL    string `json:"baseurl"`
}

// FetchQRCode retrieves a new QR code for login.
// Returns the opaque qrcode token and the QR code content string (to be rendered as QR).
func FetchQRCode(ctx context.Context) (qrcode string, imgContent string, err error) {
	c := newUnauthClient()
	var resp qrCodeResp
	if err := c.get(ctx, qrCodeURL, &resp); err != nil {
		return "", "", fmt.Errorf("wechat: fetch QR code: %w", err)
	}
	return resp.QRCode, resp.QRCodeImgContent, nil
}

// PollQRStatus polls for QR code scan status until confirmed or expired.
// onStatus is called on each status update so the caller can display progress.
func PollQRStatus(ctx context.Context, qrcode string, onStatus func(status string)) (*Credentials, error) {
	c := newUnauthClient()
	url := qrStatusURL + qrcode

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		pollCtx, cancel := context.WithTimeout(ctx, 40*time.Second)
		var resp qrStatusResp
		err := c.get(pollCtx, url, &resp)
		cancel()

		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			continue // timeout is normal for long-poll, retry
		}

		if onStatus != nil {
			onStatus(resp.Status)
		}

		switch resp.Status {
		case qrStatusConfirmed:
			return &Credentials{
				BotToken:   resp.BotToken,
				ILinkBotID: resp.ILinkBotID,
				BaseURL:    resp.BaseURL,
			}, nil
		case qrStatusExpired:
			return nil, fmt.Errorf("wechat: QR code expired")
		case qrStatusWait, qrStatusScanned:
			// continue polling
		}
	}
}

// credsPath returns the path where credentials are stored.
func credsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ledger-agent", "wechat.json"), nil
}

// SaveCredentials saves credentials to ~/.ledger-agent/wechat.json.
func SaveCredentials(creds *Credentials) error {
	path, err := credsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("wechat: create dir: %w", err)
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("wechat: marshal creds: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("wechat: write creds: %w", err)
	}
	return nil
}

// LoadCredentials loads credentials from ~/.ledger-agent/wechat.json.
// Returns nil, nil if no credentials file exists yet.
func LoadCredentials() (*Credentials, error) {
	path, err := credsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("wechat: read creds: %w", err)
	}
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("wechat: parse creds: %w", err)
	}
	return &creds, nil
}
