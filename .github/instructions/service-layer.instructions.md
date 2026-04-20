---
applyTo: "**/service.go,**/service_test.go"
description: "Service layer conventions. Business logic lives here with strict input/output contracts."
---

# Service Layer

## Method Signature

Every public method follows this contract:

```go
func (s *Service) Verb(ctx context.Context, in VerbInput) (*VerbOutput, error)
```

- First param: `context.Context` — always.
- Second param: typed input struct (`VerbInput`) — never primitives spread across params.
- Returns: pointer to result + error.
- Method name is a verb: `Create`, `Get`, `List`, `Update`, `Delete`, `Execute`.

### Single-Parameter Lookups

When a method takes only ONE identifier (e.g., get-by-slug, get-by-ID), a bare
primitive is acceptable — the "no spread" rule targets methods with 2+ business params:

```go
// OK — single lookup key
func (s *Service) GetBySlug(ctx context.Context, slug string) (*Article, error)

// OK — multiple params bundled
func (s *Service) Update(ctx context.Context, in UpdateInput) (*Article, error)
```

### Delete Methods

Delete methods that have no meaningful return value may return just `error`:

```go
func (s *Service) Delete(ctx context.Context, in DeleteInput) error
```

### Return Type

- Return the **domain model** directly (`*User`, `*Order`) when the output IS the entity.
- Use a dedicated `VerbOutput` struct only when the return shape differs from the model
  (e.g., `ListOutput` with items + total count, or an aggregate response).
- The handler is responsible for converting the domain model to a response DTO
  before sending it over the wire (see transport instructions).

## Input / Output

- Define `VerbInput` and `VerbOutput` structs in same file or `dto.go`.
- Input carries the caller's intent. Output carries the result.
- Never pass framework types (`*http.Request`, `*sql.Tx`) as input.
- Validate business preconditions inside the method, not in the caller.

## Transaction Control

Service owns transaction boundaries:

```go
func (s *Service) Transfer(ctx context.Context, in TransferInput) (*TransferOutput, error) {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("transfer: begin tx: %w", err)
    }
    defer tx.Rollback()
    // ... operations using tx ...
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("transfer: commit: %w", err)
    }
    return &TransferOutput{...}, nil
}
```

- Only service starts and commits transactions.
- Store methods accept `*sql.Tx` for writes, never start their own.
- If a method does not need a transaction, use `s.db` directly.

## Error Handling

- Return `errkit` errors for business failures: `errkit.New(errkit.NotFound, "user not found")`.
- Wrap infrastructure errors: `fmt.Errorf("service: create user: %w", err)`.
- Never log-and-return. Either log (at handler level) or return — not both.

## Dependencies

- Inject via constructor: `func NewService(db *sql.DB, store *Store) *Service`.
  - Pass `db` when the service needs transaction control (`BeginTx`).
  - Omit `db` if the service only reads (no writes).
- Store struct types, not interfaces, unless testing requires substitution.
- Service must NOT import transport packages (`net/http`, gRPC stubs).

## ID Generation

- Generate IDs in the service layer before insert: `uuid.NewString()` (from `github.com/google/uuid`).
- Never rely on database-generated IDs (`gen_random_uuid()`, `SERIAL`).
- This lets the service know the entity ID before the DB round-trip — simpler
  response construction, idempotency keys, and testability.
