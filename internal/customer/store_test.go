package customer

import (
	"context"
	"testing"
	"time"

	"github.com/cccvno1/goplate/pkg/testkit"
	"github.com/google/uuid"
)

func TestStore_CreateAndGetByID(t *testing.T) {
	db := testkit.PG(t)
	store := NewStore(db)

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	c := &Customer{
		ID:        uuid.NewString(),
		Name:      "测试客户_" + uuid.NewString()[:8],
		CreatedAt: time.Now().UTC(),
	}
	if err := store.Create(context.Background(), tx, c); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	got, err := store.GetByID(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil")
	}
	if got.Name != c.Name {
		t.Errorf("Name = %q, want %q", got.Name, c.Name)
	}
}

func TestStore_ListAll(t *testing.T) {
	db := testkit.PG(t)
	store := NewStore(db)

	customers, err := store.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	// Just verify the call succeeds; content depends on existing data.
	_ = customers
}
