package ledger

import (
	"context"
	"testing"
	"time"

	"github.com/cccvno1/goplate/pkg/errkit"
)

func TestCreate_InvalidInput(t *testing.T) {
	svc := &Service{}

	tests := []struct {
		name string
		in   CreateInput
	}{
		{name: "missing customer id", in: CreateInput{CustomerName: "张三", ProductName: "苹果", UnitPrice: 5, Quantity: 1, EntryDate: time.Now()}},
		{name: "missing customer name", in: CreateInput{CustomerID: "cust-1", ProductName: "苹果", UnitPrice: 5, Quantity: 1, EntryDate: time.Now()}},
		{name: "missing product name", in: CreateInput{CustomerID: "cust-1", CustomerName: "张三", UnitPrice: 5, Quantity: 1, EntryDate: time.Now()}},
		{name: "non-positive unit price", in: CreateInput{CustomerID: "cust-1", CustomerName: "张三", ProductName: "苹果", UnitPrice: 0, Quantity: 1, EntryDate: time.Now()}},
		{name: "non-positive quantity", in: CreateInput{CustomerID: "cust-1", CustomerName: "张三", ProductName: "苹果", UnitPrice: 5, Quantity: 0, EntryDate: time.Now()}},
		{name: "missing entry date", in: CreateInput{CustomerID: "cust-1", CustomerName: "张三", ProductName: "苹果", UnitPrice: 5, Quantity: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), tt.in)
			if err == nil {
				t.Fatal("expected error")
			}
			if e := errkit.AsError(err); e == nil || e.Code != errkit.InvalidInput {
				t.Fatalf("expected invalid input error, got %v", err)
			}
		})
	}
}

func TestUpdate_InvalidInput(t *testing.T) {
	svc := &Service{}

	tests := []struct {
		name string
		in   UpdateInput
	}{
		{name: "missing id", in: UpdateInput{ProductName: "苹果"}},
		{name: "no changes", in: UpdateInput{ID: "entry-1"}},
		{name: "non-positive unit price", in: UpdateInput{ID: "entry-1", UnitPrice: -1}},
		{name: "non-positive quantity", in: UpdateInput{ID: "entry-1", Quantity: -1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Update(context.Background(), tt.in)
			if err == nil {
				t.Fatal("expected error")
			}
			if e := errkit.AsError(err); e == nil || e.Code != errkit.InvalidInput {
				t.Fatalf("expected invalid input error, got %v", err)
			}
		})
	}
}
