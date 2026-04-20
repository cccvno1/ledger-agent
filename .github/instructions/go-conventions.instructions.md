---
applyTo: "**/*.go"
description: "Go code conventions for generated service. Covers naming, errors, testing, imports, and struct design."
---

# Go Code Conventions

## Naming

- Packages: short, lowercase, single-word (`domain`, `boot`, `conf`)
- Exported: PascalCase, preserve acronyms (`HTTPStatus`, `ID`, `DSN`)
- Constructors: `New()` for primary type, `New<Type>()` when package has multiple
- Constants: PascalCase for exported, camelCase for unexported — never ALL_CAPS
- Single-letter vars only in tight scopes: `i`, `ctx`, `t`, `err`, `r`/`w`

## Imports

Three groups separated by blank lines:

```go
import (
    "context"          // 1. stdlib
    "github.com/lib/pq" // 2. third-party
    "ledger-agent/internal/domain" // 3. internal
)
```

## Errors

- Wrap with context: `fmt.Errorf("package: operation: %w", err)`
- Business errors: `errkit.New(code, msg)` — codes from `domain/errors.go`
- Never discard errors from I/O, DB, or HTTP operations
- Check errors immediately after the call, before using the result

## Structs

- Group fields logically: config, dependencies, state
- Unexported fields by default; export only what callers need
- Use functional options or config struct for constructors with >3 params

## Functions

- Accept interfaces, return concrete types
- First param is `context.Context` when the function does I/O or may block
- Return `error` as the last return value
- Keep functions under 40 lines; extract helpers when logic branches

## Testing

- Standard library `testing` only — no testify, gomega
- Table-driven for multiple inputs: `tests := []struct{...}`
- Test helpers must call `t.Helper()` first
- Use `t.TempDir()`, `t.Setenv()` for isolation
- Name: `Test<Function>` or `Test<Function>_<Scenario>`

## Concurrency

- Never start goroutines without a shutdown path
- Protect shared state with `sync.Mutex` or channels, not both
- Pass `context.Context` to all blocking operations
