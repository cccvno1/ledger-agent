package ledger

import (
	"context"
	"testing"
	"time"

	"github.com/cccvno1/ledger-agent/testutil"
	"github.com/google/uuid"
)

// Idempotency: re-creating with the same key must return the existing row's
// id and not insert a duplicate.
func TestStore_Create_Idempotent(t *testing.T) {
	db := testutil.Postgres(t)
	store := NewStore(db)
	ctx := context.Background()

	custID := uuid.NewString()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO customers (id, name, created_at) VALUES ($1, $2, now())`,
		custID, "幂等客户_"+custID[:8],
	); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	key := "idem-" + uuid.NewString()
	now := time.Now().UTC()

	first := &Entry{
		ID: uuid.NewString(), CustomerID: custID, CustomerName: "幂等客户",
		ProductName: "苹果", UnitPrice: 5, Quantity: 2, Unit: "斤", Amount: 10,
		EntryDate: now, IdempotencyKey: key, CreatedAt: now, UpdatedAt: now,
	}
	tx, _ := db.BeginTx(ctx, nil)
	inserted, err := store.Create(ctx, tx, first)
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if !inserted {
		t.Fatal("first Create should report inserted=true")
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	// Retry with same key but a different proposed id; should resolve to the
	// surviving row and report inserted=false.
	second := &Entry{
		ID: uuid.NewString(), CustomerID: custID, CustomerName: "幂等客户",
		ProductName: "苹果", UnitPrice: 5, Quantity: 2, Unit: "斤", Amount: 10,
		EntryDate: now, IdempotencyKey: key, CreatedAt: now, UpdatedAt: now,
	}
	tx2, _ := db.BeginTx(ctx, nil)
	inserted2, err := store.Create(ctx, tx2, second)
	if err != nil {
		t.Fatalf("retry Create: %v", err)
	}
	if inserted2 {
		t.Error("retry Create should report inserted=false")
	}
	if second.ID != first.ID {
		t.Errorf("retry should resolve to original id %s, got %s", first.ID, second.ID)
	}
	tx2.Commit()

	// Verify only one row exists.
	var n int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM entries WHERE idempotency_key = $1`, key,
	).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("row count for idempotency_key = %d, want 1", n)
	}
}
