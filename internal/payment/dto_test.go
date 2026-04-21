package payment

import "testing"

func TestCreateRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateRequest
		wantErr bool
	}{
		{"valid", CreateRequest{CustomerID: "c1", Amount: 100, PaymentDate: "2026-01-15"}, false},
		{"missing customer_id", CreateRequest{Amount: 100, PaymentDate: "2026-01-15"}, true},
		{"zero amount", CreateRequest{CustomerID: "c1", Amount: 0, PaymentDate: "2026-01-15"}, true},
		{"negative amount", CreateRequest{CustomerID: "c1", Amount: -50, PaymentDate: "2026-01-15"}, true},
		{"missing payment_date", CreateRequest{CustomerID: "c1", Amount: 100}, true},
		{"invalid date format", CreateRequest{CustomerID: "c1", Amount: 100, PaymentDate: "01/15/2026"}, true},
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

func TestCreateRequest_ToInput(t *testing.T) {
	req := CreateRequest{
		CustomerID:  " c1 ",
		Amount:      250.5,
		PaymentDate: "2026-04-01",
		Notes:       "月结",
	}
	in := req.ToInput()
	if in.CustomerID != "c1" {
		t.Errorf("CustomerID = %q, want trimmed %q", in.CustomerID, "c1")
	}
	if in.Amount != 250.5 {
		t.Errorf("Amount = %v, want 250.5", in.Amount)
	}
	if in.PaymentDate.IsZero() {
		t.Error("PaymentDate should not be zero")
	}
}
