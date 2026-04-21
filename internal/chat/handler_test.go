package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cccvno1/goplate/pkg/errkit"
)

type stubChatService struct {
	out   *ChatOutput
	err   error
	input ChatInput
}

func (s *stubChatService) Chat(_ context.Context, in ChatInput) (*ChatOutput, error) {
	s.input = in
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

func TestHandlerChat_InvalidJSON(t *testing.T) {
	h := NewHandler(&stubChatService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(`{"message":`))
	rec := httptest.NewRecorder()

	h.Chat(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var resp struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Success {
		t.Fatal("expected success=false")
	}
	if resp.Error.Code == "" {
		t.Fatal("expected error code to be set")
	}
}

func TestHandlerChat_MissingMessage(t *testing.T) {
	h := NewHandler(&stubChatService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(`{"session_id":"demo","message":"   "}`))
	rec := httptest.NewRecorder()

	h.Chat(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandlerChat_ServiceError(t *testing.T) {
	h := NewHandler(&stubChatService{
		err: errkit.New(errkit.InvalidInput, "bad request"),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(`{"session_id":"demo","message":"查账"}`))
	rec := httptest.NewRecorder()

	h.Chat(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandlerChat_Success(t *testing.T) {
	svc := &stubChatService{
		out: &ChatOutput{
			SessionID: "demo",
			Reply:     "好的",
			Draft: []DraftEntry{
				{CustomerName: "张三", ProductName: "苹果"},
			},
		},
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(`{"session_id":"demo","message":"  查账  "}`))
	rec := httptest.NewRecorder()

	h.Chat(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if svc.input.Message != "查账" {
		t.Fatalf("service input message = %q, want %q", svc.input.Message, "查账")
	}

	var resp struct {
		Success bool         `json:"success"`
		Data    ChatResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if resp.Data.SessionID != "demo" {
		t.Errorf("SessionID = %q, want %q", resp.Data.SessionID, "demo")
	}
	if resp.Data.Reply != "好的" {
		t.Errorf("Reply = %q, want %q", resp.Data.Reply, "好的")
	}
	if len(resp.Data.Draft) != 1 {
		t.Fatalf("Draft len = %d, want 1", len(resp.Data.Draft))
	}
}
