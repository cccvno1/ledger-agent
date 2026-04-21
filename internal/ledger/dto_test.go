package ledger

import "testing"

func TestCreateRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateRequest
		wantErr bool
	}{
		{
			name: "valid",
			req: CreateRequest{
				CustomerID: "c1", CustomerName: "张三", ProductName: "苹果",
				UnitPrice: 5, Quantity: 10, EntryDate: "2026-01-15",
			},
			wantErr: false,
		},
		{"missing customer_id", CreateRequest{CustomerName: "张三", ProductName: "苹果", UnitPrice: 5, Quantity: 10, EntryDate: "2026-01-15"}, true},
		{"missing product_name", CreateRequest{CustomerID: "c1", CustomerName: "张三", UnitPrice: 5, Quantity: 10, EntryDate: "2026-01-15"}, true},
		{"zero unit_price", CreateRequest{CustomerID: "c1", CustomerName: "张三", ProductName: "苹果", UnitPrice: 0, Quantity: 10, EntryDate: "2026-01-15"}, true},
		{"zero quantity", CreateRequest{CustomerID: "c1", CustomerName: "张三", ProductName: "苹果", UnitPrice: 5, Quantity: 0, EntryDate: "2026-01-15"}, true},
		{"missing entry_date", CreateRequest{CustomerID: "c1", CustomerName: "张三", ProductName: "苹果", UnitPrice: 5, Quantity: 10}, true},
		{"invalid date format", CreateRequest{CustomerID: "c1", CustomerName: "张三", ProductName: "苹果", UnitPrice: 5, Quantity: 10, EntryDate: "2026/01/15"}, true},
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

func TestUpdateRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateRequest
		wantErr bool
	}{
		{"empty update", UpdateRequest{}, false},
		{"valid date", UpdateRequest{EntryDate: "2026-03-01"}, false},
		{"invalid date", UpdateRequest{EntryDate: "not-a-date"}, true},
		{"negative price", UpdateRequest{UnitPrice: -1}, true},
		{"negative quantity", UpdateRequest{Quantity: -1}, true},
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
		CustomerID: "c1", CustomerName: " 张三 ", ProductName: "苹果",
		UnitPrice: 5.5, Quantity: 3, Unit: "斤", EntryDate: "2026-04-01", Notes: "备注",
	}
	in := req.ToInput()
	if in.CustomerName != "张三" {
		t.Errorf("CustomerName = %q, want trimmed", in.CustomerName)
	}
	if in.UnitPrice != 5.5 {
		t.Errorf("UnitPrice = %v, want 5.5", in.UnitPrice)
	}
	if in.EntryDate.IsZero() {
		t.Error("EntryDate should not be zero")
	}
}
