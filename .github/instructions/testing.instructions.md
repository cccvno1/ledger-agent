---
applyTo: "**/*_test.go"
description: "Testing conventions. Behavioral tests with standard library only."
---

# Testing Conventions

## Framework

Standard library `testing` package only. No testify, gomega, or assertion libraries.

## Naming

- `Test<Function>` for the happy path.
- `Test<Function>_<Scenario>` for edge cases: `TestGetUser_NotFound`, `TestCreate_DuplicateEmail`.
- Test file shares name with source: `service.go` → `service_test.go`.

## Structure

```go
func TestCreate(t *testing.T) {
    // arrange
    store := NewStore(testDB)
    svc := NewService(store)
    in := CreateInput{Name: "alice"}

    // act
    out, err := svc.Create(context.Background(), in)

    // assert
    if err != nil {
        t.Fatalf("Create() error = %v", err)
    }
    if out.Name != "alice" {
        t.Errorf("Name = %q, want %q", out.Name, "alice")
    }
}
```

## Table-Driven Tests

Use for multiple inputs with the same assertion logic:

```go
func TestValidate(t *testing.T) {
    tests := []struct {
        name    string
        input   ValidateInput
        wantErr bool
    }{
        {"valid", ValidateInput{Name: "ok"}, false},
        {"empty name", ValidateInput{Name: ""}, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Validate(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## What to Test

| Layer | Test Focus | Dependencies |
|-------|-----------|--------------|
| Service | Business logic, error paths, validation | Real store (or test double) |
| Store | SQL correctness, edge cases | Real database via testkit |
| Handler | Request parsing, status codes, response shape | Real service + store |
| DTO | Validate(), ToInput() | None — always testable |

## Database-Dependent Tests

Store tests and integration tests require a running database (`make docker-up`).
Guard them so `go test ./...` passes without infrastructure:

```go
func TestCreate(t *testing.T) {
    db := testkit.PG(t) // skips automatically when DB is unavailable
    store := NewStore(db)
    // ...
}
```

`testkit.PG(t)` calls `t.Skip` when the database connection cannot be established.
This means:
- `go test ./...` always passes (DB tests are skipped when infra is down).
- `make test` with `make docker-up` runs all tests including DB tests.
- CI runs `make docker-up && make test` to get full coverage.

For tests that do NOT need a database (DTO validation, handler error paths,
pure business logic helpers), write them as regular tests with no skip guard —
they always run.

## What NOT to Test

- Do not test getter/setter methods or trivial constructors.
- Do not test the Go standard library (e.g., JSON marshaling of built-in types).
- Do not test third-party library behavior.

## Test Isolation

- Use `t.TempDir()` for file system tests.
- Use `t.Setenv()` for environment variable tests.
- Each test must be independent — no shared mutable state.
- Clean up database state per test (transaction rollback or truncate).

## Assertions

- Use `t.Fatalf` when continuing is meaningless (nil pointer would follow).
- Use `t.Errorf` to report and continue — allows collecting multiple failures.
- Compare with `==` for primitives, `reflect.DeepEqual` for structs only when necessary.
- For error checks: `if err != nil` (existence), `errors.Is(err, target)` (specific error).

## Behavioral Focus

Test **what** the code does, not **how** it does it:

- GOOD: "Create returns the user with a generated ID"
- BAD: "Create calls store.Insert exactly once with the right SQL"

Do not assert on internal implementation details, call counts, or argument capture.
