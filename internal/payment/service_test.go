package payment

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
		{name: "missing customer id", in: CreateInput{Amount: 10, PaymentDate: time.Now()}},
		{name: "non-positive amount", in: CreateInput{CustomerID: "cust-1", Amount: 0, PaymentDate: time.Now()}},
		{name: "missing payment date", in: CreateInput{CustomerID: "cust-1", Amount: 10}},
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
