package testutil

import (
	"database/sql"
	"fmt"
	"testing"
)

func TestPostgres_AppliesMigrations(t *testing.T) {
	db := Postgres(t)

	for _, table := range []string{"customers", "entries", "products", "payments"} {
		var got sql.NullString
		if err := db.QueryRow(fmt.Sprintf("SELECT to_regclass('public.%s')", table)).Scan(&got); err != nil {
			t.Fatalf("lookup table %s: %v", table, err)
		}
		if !got.Valid {
			t.Fatalf("expected table %s to exist after migrations", table)
		}
	}
}
