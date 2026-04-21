package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Postgres starts a fresh Postgres testcontainer, applies all repo migrations,
// and returns a ready-to-use database connection.
func Postgres(t *testing.T) *sql.DB {
	t.Helper()

	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "test",
				"POSTGRES_PASSWORD": "test",
				"POSTGRES_DB":       "test",
			},
			WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Skipf("testutil.Postgres: docker unavailable: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Skipf("testutil.Postgres: container host unavailable: %v", err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Skipf("testutil.Postgres: container port unavailable: %v", err)
	}

	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=test sslmode=disable", host, port.Port())
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("testutil.Postgres: open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	for i := 0; i < 30; i++ {
		if err := db.Ping(); err == nil {
			if err := applyMigrations(ctx, db); err != nil {
				t.Fatalf("testutil.Postgres: apply migrations: %v", err)
			}
			return db
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Skip("testutil.Postgres: database not ready after 3s")
	return nil
}

func applyMigrations(ctx context.Context, db *sql.DB) error {
	entries, err := os.ReadDir(filepath.Join(repoRoot(), "migrations", "postgres"))
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("no migration files found")
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		path := filepath.Join(repoRoot(), "migrations", "postgres", entry.Name())
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		if _, err := db.ExecContext(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("exec %s: %w", entry.Name(), err)
		}
	}

	return nil
}

func repoRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Dir(filepath.Dir(filename))
}
