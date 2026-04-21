package payment

import (
	"context"
	"testing"
	"time"

	"github.com/cccvno1/goplate/pkg/errkit"
	"github.com/cccvno1/ledger-agent/testutil"
	"github.com/google/uuid"
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

func TestCreate_AmountExceedsPending(t *testing.T) {
	db := testutil.Postgres(t)
	store := NewStore(db)
	svc := NewService(db, store)

	customerID := uuid.NewString()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO customers (id, name, created_at) VALUES ($1, $2, now())`,
		customerID, "测试客户_"+customerID[:8],
	)
	if err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	_, err = db.ExecContext(context.Background(),
		`INSERT INTO entries (
			id, customer_id, customer_name, product_name, unit_price, quantity, unit, amount,
			entry_date, is_settled, notes, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,now(),now())`,
		uuid.NewString(), customerID, "测试客户", "苹果", 10.0, 5.0, "斤", 50.0,
		time.Now().UTC().Format("2006-01-02"), false, "",
	)
	if err != nil {
		t.Fatalf("seed entry: %v", err)
	}

	_, err = svc.Create(context.Background(), CreateInput{
		CustomerID:  customerID,
		Amount:      60,
		PaymentDate: time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	e := errkit.AsError(err)
	if e == nil || e.Code != errkit.InvalidInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
}
