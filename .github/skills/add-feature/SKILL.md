---
name: add-feature
description: "Add a new business feature package with model, service, wire, and requirement-driven files. Use when: add feature, new feature, 添加功能."
argument-hint: "Feature name in singular lowercase (e.g., 'order', 'payment', 'notification')"
---

# Add Feature

Add a self-contained feature package to `internal/<name>/` following project conventions.

## Prerequisites

Before starting, verify:
1. Feature name is singular, lowercase, no underscores: `order`, `payment`, `notification`.
2. No existing directory at `internal/<name>/`.
3. The domain model is understood — know what the core entity looks like.

## Procedure

### 1. Create Feature Directory

Create `internal/<name>/` with these fixed files:

#### model.go (always)

```go
package <name>

// <Name> represents... (describe the domain entity)
type <Name> struct {
	ID        string
	// ... domain fields
	CreatedAt time.Time
	UpdatedAt time.Time
}
```

- No framework imports (`net/http`, `database/sql`).
- Types represent the business domain, not the database schema or API shape.

#### service.go (always)

```go
package <name>

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type Service struct {
	db    *sql.DB
	store *Store
}

func NewService(db *sql.DB, store *Store) *Service {
	return &Service{db: db, store: store}
}

// CreateInput carries the caller's intent for creation.
type CreateInput struct {
	Name string
}

// Single-identifier lookups may take a bare primitive:
// func (s *Service) GetBySlug(ctx context.Context, slug string) (*<Name>, error)
// Delete with no return value returns just error:
// func (s *Service) Delete(ctx context.Context, in DeleteInput) error

func (s *Service) Create(ctx context.Context, in CreateInput) (*<Name>, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("<name>: create: begin tx: %w", err)
	}
	defer tx.Rollback()

	entity := &<Name>{
		ID:   uuid.NewString(),
		Name: in.Name,
	}
	if err := s.store.Create(ctx, tx, entity); err != nil {
		return nil, fmt.Errorf("<name>: create: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("<name>: create: commit: %w", err)
	}
	return entity, nil
}
```

- Method signature: `(ctx context.Context, in VerbInput) (*<Name>, error)` — returns domain model directly for simple CRUD. Use a dedicated `VerbOutput` struct only when the return shape differs from the model.
- Service owns transaction boundaries: `BeginTx` → store ops → `Commit`.
- Generate IDs in the service with `uuid.NewString()` before insert — never rely on database defaults.
- `NewService` takes `db` (for transactions) and `store` (for queries). Omit `db` if the feature has no write operations.
- No transport or infrastructure types in method parameters.

#### wire.go (always)

```go
package <name>

import (
	"database/sql"
	"net/http"
)

// Wire assembles the feature's dependency graph and registers routes.
func Wire(mux *http.ServeMux, db *sql.DB) {
	store := NewStore(db)
	svc := NewService(db, store)
	h := NewHandler(svc)
	mux.HandleFunc("POST /<name>s", h.Create)
	mux.HandleFunc("GET /<name>s/{id}", h.Get)
}
```

Adjust the signature based on which infrastructure the feature needs. A feature
without HTTP would take only `db *sql.DB`. A feature without persistence would
take only `mux *http.ServeMux`.

### 2. Add Requirement-Driven Files

Determine what else this feature needs based on its responsibility:

| Need | File | Pattern |
|------|------|---------|
| Database persistence | `store.go` | `*sql.DB` for reads, `*sql.Tx` for writes |
| HTTP exposure | `handler.go` + `dto.go` | parse → delegate → respond |
| Background processing | `worker.go` or `processor.go` | job execution logic |
| External API calls | `client.go` or `gateway.go` | third-party integration |
| Event handling | `subscriber.go` or `listener.go` | event consumption |
| File/stream processing | `reader.go` or `writer.go` | I/O operations |

**Only create files the feature actually requires.** A notification feature may only need `service.go` + `sender.go` + `wire.go`. A search indexer may need `service.go` + `indexer.go` + `wire.go`.

### 3. Wire Into Boot

In `internal/base/boot/boot.go`, find the comment `// Wire feature packages here.`
and replace the placeholder `_ = db` / `_ = mux` lines with the feature's Wire call.

```go
// Wire feature packages here.
<name>.Wire(mux, db) // pass only what the feature needs
```

The `mux` and `db` variables are already declared above in boot.go. Import the
feature package: `"<module>/internal/<name>"`.

If the feature does not need HTTP (no handler.go), omit `mux`. If it does not need
a database (no store.go), omit `db`.

### 4. Add Migration (only if new tables are needed)

Create `migrations/postgres/NNNN_create_<name>.sql`:

```sql
CREATE TABLE IF NOT EXISTS <name>s (
    id         TEXT        PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

IDs are generated by the application (see service.go), so the column has no default.

Skip this step if the feature doesn't need its own tables.

### 5. Add Architecture Rule

In `architecture_test.go`, add a rule to prevent the feature from importing `base/`.
**Every new feature package must have this rule** — the architecture test is the only
guardrail preventing violations:

```go
{
    Package: "<name>",
    MustNot: []string{"base"},
},
```

### 6. Write Tests

Create test files alongside the source files that exist:
- `dto_test.go` — DTO validation and conversion tests (always, no dependencies)
- `service_test.go` — business logic tests (always)
- `store_test.go` — SQL correctness tests using `testkit.PG(t)` (when store.go exists)
- `handler_test.go` — request parsing, error paths, response shape (when handler.go exists)

**DB-dependent tests** (`store_test.go`, integration `service_test.go`) use `testkit.PG(t)`,
which auto-skips when Docker is unavailable. This ensures `go test ./...` always passes.
**Non-DB tests** (DTO validation, handler error paths, pure helpers) always run.

### 7. Verify

```bash
go vet ./...
go test -count=1 ./...
```

All tests must pass, including `TestArchitecture`.

## Cross-Feature Communication

If feature A needs to call feature B, do NOT import B directly.
Define an interface in A and inject it at wire time in `boot.go`:

```go
// In internal/order/service.go
type PaymentChecker interface {
    HasPaid(ctx context.Context, orderID string) (bool, error)
}

type Service struct {
    db      *sql.DB
    store   *Store
    payment PaymentChecker // injected, not imported
}
```

```go
// In internal/base/boot/boot.go
paySvc := payment.NewService(db, payStore)
orderSvc := order.NewService(db, orderStore, paySvc) // paySvc satisfies order.PaymentChecker
```

This preserves the "feature packages must not import other features" boundary.

## Checklist

- [ ] `internal/<name>/model.go` — domain types defined
- [ ] `internal/<name>/service.go` — business logic with typed Input structs and transaction control
- [ ] `internal/<name>/wire.go` — wiring function created
- [ ] Requirement-driven files created (only what's needed)
- [ ] Feature wired in `boot.go`
- [ ] Architecture rule added
- [ ] Migration created (if storing data)
- [ ] Tests written and passing
- [ ] `go vet ./...` clean
