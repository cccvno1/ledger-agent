package ledger

import (
	"context"
	"testing"
	"time"

	"github.com/cccvno1/ledger-agent/testutil"
	"github.com/google/uuid"
)

func newTestEntry(customerID string) *Entry {
	now := time.Now().UTC()
	return &Entry{
		ID:           uuid.NewString(),
		CustomerID:   customerID,
		CustomerName: "测试客户",
		ProductName:  "商品A",
		UnitPrice:    10.5,
		Quantity:     3,
		Amount:       31.5,
		EntryDate:    now,
		Notes:        "",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestStore_CreateAndGetByID(t *testing.T) {
	db := testutil.Postgres(t)
	store := NewStore(db)

	// Ensure customer exists.
	custID := uuid.NewString()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO customers (id, name, created_at) VALUES ($1, $2, now())`,
		custID, "测试客户_"+custID[:8],
	)
	if err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	e := newTestEntry(custID)
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	if err := store.Create(context.Background(), tx, e); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	got, err := store.GetByID(context.Background(), e.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil")
	}
	if got.ProductName != e.ProductName {
		t.Errorf("ProductName = %q, want %q", got.ProductName, e.ProductName)
	}
}

func TestStore_MarkSettled(t *testing.T) {
	db := testutil.Postgres(t)
	store := NewStore(db)

	custID := uuid.NewString()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO customers (id, name, created_at) VALUES ($1, $2, now())`,
		custID, "清账客户_"+custID[:8],
	)
	if err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	e := newTestEntry(custID)
	tx, _ := db.BeginTx(context.Background(), nil)
	store.Create(context.Background(), tx, e)
	tx.Commit()

	tx2, _ := db.BeginTx(context.Background(), nil)
	defer tx2.Rollback()
	if err := store.MarkSettled(context.Background(), tx2, custID, time.Now().UTC()); err != nil {
		t.Fatalf("MarkSettled: %v", err)
	}
	tx2.Commit()

	got, _ := store.GetByID(context.Background(), e.ID)
	if !got.IsSettled {
		t.Error("expected is_settled = true after MarkSettled")
	}
}
