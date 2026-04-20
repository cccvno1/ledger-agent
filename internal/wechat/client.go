package wechat

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	ilinkBaseURL    = "https://ilinkai.weixin.qq.com"
	longPollTimeout = 35 * time.Second
	sendTimeout     = 15 * time.Second

	msgTypeUser  = 1
	msgTypeBot   = 2
	msgStateDone = 2

	itemTypeText  = 1
	itemTypeVoice = 3

	errCodeSessionExpired = -14
)

// Credentials holds iLink Bot API session credentials obtained after QR login.
type Credentials struct {
	BotToken   string `json:"bot_token"`
	ILinkBotID string `json:"ilink_bot_id"`
	BaseURL    string `json:"base_url"`
}

type client struct {
	baseURL    string
	botToken   string
	botID      string
	httpClient *http.Client
	wechatUIN  string
}

func newClient(creds *Credentials) *client {
	baseURL := creds.BaseURL
	if baseURL == "" {
		baseURL = ilinkBaseURL
	}
	return &client{
		baseURL:    baseURL,
		botToken:   creds.BotToken,
		botID:      creds.ILinkBotID,
		httpClient: &http.Client{},
		wechatUIN:  generateWechatUIN(),
	}
}

func newUnauthClient() *client {
	return &client{
		baseURL:    ilinkBaseURL,
		httpClient: &http.Client{Timeout: 40 * time.Second},
		wechatUIN:  generateWechatUIN(),
	}
}

// --- iLink message types ---

type weixinMessage struct {
	MessageID    int64         `json:"message_id"`
	FromUserID   string        `json:"from_user_id"`
	MessageType  int           `json:"message_type"`
	MessageState int           `json:"message_state"`
	ItemList     []messageItem `json:"item_list"`
	ContextToken string        `json:"context_token"`
}

type messageItem struct {
	Type      int        `json:"type"`
	TextItem  *textItem  `json:"text_item,omitempty"`
	VoiceItem *voiceItem `json:"voice_item,omitempty"`
}

type textItem struct {
	Text string `json:"text"`
}

type voiceItem struct {
	Text string `json:"text,omitempty"` // WeChat speech-to-text transcription
}

type baseInfo struct {
	ChannelVersion string `json:"channel_version,omitempty"`
}

type getUpdatesReq struct {
	GetUpdatesBuf string   `json:"get_updates_buf"`
	BaseInfo      baseInfo `json:"base_info"`
}

type getUpdatesResp struct {
	Ret           int             `json:"ret"`
	ErrCode       int             `json:"errcode,omitempty"`
	ErrMsg        string          `json:"errmsg,omitempty"`
	Msgs          []weixinMessage `json:"msgs"`
	GetUpdatesBuf string          `json:"get_updates_buf"`
}

type sendMsgReq struct {
	Msg      sendMsg  `json:"msg"`
	BaseInfo baseInfo `json:"base_info"`
}

type sendMsg struct {
	FromUserID   string        `json:"from_user_id"`
	ToUserID     string        `json:"to_user_id"`
	ClientID     string        `json:"client_id"`
	MessageType  int           `json:"message_type"`
	MessageState int           `json:"message_state"`
	ItemList     []messageItem `json:"item_list"`
	ContextToken string        `json:"context_token"`
}

type sendMsgResp struct {
	Ret    int    `json:"ret"`
	ErrMsg string `json:"errmsg,omitempty"`
}

// --- API calls ---

func (c *client) getUpdates(ctx context.Context, buf string) (*getUpdatesResp, error) {
	ctx, cancel := context.WithTimeout(ctx, longPollTimeout+5*time.Second)
	defer cancel()

	var resp getUpdatesResp
	if err := c.post(ctx, "/ilink/bot/getupdates", getUpdatesReq{
		GetUpdatesBuf: buf,
		BaseInfo:      baseInfo{ChannelVersion: "1.0.0"},
	}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *client) sendText(ctx context.Context, toUserID, contextToken, text string) error {
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	req := sendMsgReq{
		Msg: sendMsg{
			FromUserID:   c.botID,
			ToUserID:     toUserID,
			ClientID:     generateClientID(),
			MessageType:  msgTypeBot,
			MessageState: msgStateDone,
			ItemList: []messageItem{
				{Type: itemTypeText, TextItem: &textItem{Text: text}},
			},
			ContextToken: contextToken,
		},
	}
	var resp sendMsgResp
	if err := c.post(ctx, "/ilink/bot/sendmessage", req, &resp); err != nil {
		return err
	}
	if resp.Ret != 0 {
		return fmt.Errorf("wechat: sendmessage ret=%d: %s", resp.Ret, resp.ErrMsg)
	}
	return nil
}

// --- HTTP helpers ---

func (c *client) post(ctx context.Context, path string, body, result interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.botToken != "" {
		req.Header.Set("AuthorizationType", "ilink_bot_token")
		req.Header.Set("Authorization", "Bearer "+c.botToken)
		req.Header.Set("X-WECHAT-UIN", c.wechatUIN)
	}
	return c.do(req, result)
}

func (c *client) get(ctx context.Context, url string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	return c.do(req, result)
}

func (c *client) do(req *http.Request, result interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	return nil
}

func generateWechatUIN() string {
	var n uint32
	_ = binary.Read(rand.Reader, binary.LittleEndian, &n)
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", n)))
}

func generateClientID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
