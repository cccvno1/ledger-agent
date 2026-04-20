# Architecture

## Overview

This service follows a layered architecture with strict dependency rules
enforced by `architecture_test.go`.

## Layers

```
cmd/server/          Entry point — config load, logger init, boot.Run()
internal/
  base/
    boot/            Wiring + lifecycle orchestration
    conf/            Config struct (all domains merged)
  domain/            Shared business types + error codes
  <feature>/         One package per business feature
```

### HTTP Transport

- **Middleware**: `internal/base/middleware/` — RequestID, Logging, Recover
- **Router**: `internal/base/router/` — shared route registration
- **Handlers**: `internal/<feature>/handler.go` — per-feature HTTP handlers
- Config: `configs/base/http.yaml`


## Feature Package Layout

Each feature is self-contained in `internal/<name>/`.

Fixed files (every feature):

| File | Responsibility |
|------|----------------|
| `model.go` | Domain model — no framework imports |
| `service.go` | Business logic — `(ctx, Input) → (Output, error)` |
| `wire.go` | Dependency assembly, called from `base/boot/boot.go` |

Additional files are requirement-driven. Common patterns:

| `handler.go` | HTTP request → service call → response |
| `dto.go` | Request / Response types (JSON serialization) |

Other files named by role: `sender.go`, `indexer.go`, `client.go`, `worker.go`, etc.

## Dependency Rules

```
domain/       → pkg/* only (no internal imports)
<feature>/    → domain/, pkg/* (no other features, no base/)
base/boot/    → everything (orchestration point)
base/conf/    → pkg/* only
```

Enforced by `architecture_test.go` using archkit.

## Config

- **Base**: `configs/base/*.yaml` — shared across environments
- **Override**: `configs/<env>/*.yaml` — environment-specific values
- **Sensitive**: environment variables only — never in YAML files

## Error Model

| Code | HTTP | When |
|------|------|------|
| `NOT_FOUND` | 404 | Resource does not exist |
| `INVALID_INPUT` | 400 | Request validation failed |
| `CONFLICT` | 409 | Duplicate or version conflict |
| `UNAUTHORIZED` | 401 | Missing credentials |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `INTERNAL` | 500 | Unexpected server error |

## Decisions

Non-obvious architectural decisions are recorded in `docs/decisions/`.
Each decision file explains context, options considered, and rationale.
