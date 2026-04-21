package chat

import "testing"

func TestChatRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     ChatRequest
		wantErr bool
	}{
		{name: "valid", req: ChatRequest{Message: "查一下张三的账"}},
		{name: "empty", req: ChatRequest{Message: ""}, wantErr: true},
		{name: "whitespace", req: ChatRequest{Message: "   \n\t"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChatRequest_ToInput(t *testing.T) {
	req := ChatRequest{
		SessionID: "demo",
		Message:   "  查一下张三的账  ",
	}

	got := req.ToInput()

	if got.SessionID != "demo" {
		t.Errorf("SessionID = %q, want %q", got.SessionID, "demo")
	}
	if got.Message != "查一下张三的账" {
		t.Errorf("Message = %q, want %q", got.Message, "查一下张三的账")
	}
}
