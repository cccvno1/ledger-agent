---
name: project-review
description: "Project health audit — automated checks, cross-feature analysis, and release readiness verdict. Use when: review project, health check, pre-release, audit, 项目审查, 发版检查."
argument-hint: "Optional: 'release' to include release-readiness verdict"
---

# Project Review

Two-phase audit: run automated checks first, then analyze what automation misses.

## Phase 1 — Automated Checks

Run all of these. Every command must pass.

```bash
go vet ./...
go test -race -count=1 -coverprofile=coverage.out ./...
go test -run TestArchitecture -v ./...
go build -o /dev/null ./cmd/server
make tidy && git diff --exit-code go.mod go.sum
```

Report results:

| Check | Result |
|-------|--------|
| `go vet` | ✓ clean / ✗ findings |
| Tests | ✓ pass / ✗ N failures |
| Race detector | ✓ clean / ✗ races found |
| Architecture | ✓ clean / ✗ N violations |
| Build | ✓ success / ✗ error |
| Module tidy | ✓ clean / ✗ go.mod or go.sum out of date |

```bash
go tool cover -func=coverage.out | tail -1
```

Report overall coverage percentage.

If any automated check fails, stop here and report the failures.
Do not proceed to Phase 2 until Phase 1 is fully green.

## Phase 2 — Analysis

These require reading code, not running commands.

### Cross-feature dependency audit

For each `internal/<feature>/wire.go`, check what it injects.
Flag any feature that depends on another feature's concrete type
instead of an interface.

### Error handling scan

Scan `internal/` for:
- `if err != nil` blocks that log but don't return (swallowed errors).
- Functions returning `error` whose callers use `_` to discard it.
- `defer f.Close()` on write paths where the close error matters.

### Documentation freshness

Compare `docs/ARCHITECTURE.md` against the actual `internal/` directory.
Flag any feature that exists in code but not in docs, or vice versa.

### Operational readiness (if argument includes "release")

- Health endpoint exists and returns 200 with structured response.
- Graceful shutdown: `appkit.App` receives signal, calls Stop on all components.
- Sensitive config fields come from env vars, not YAML files.
- No `fmt.Println`, `panic()` outside main, or hardcoded secrets in code.

## Verdict

```
## Project Review

Phase 1: ✓ all automated checks pass | ✗ blocked (see failures)
Phase 2:
  Dependencies:  ✓ clean | ⚠ N findings
  Error handling: ✓ clean | ⚠ N findings
  Documentation:  ✓ current | ⚠ drift detected
  Ops readiness:  ✓ ready | ⚠ N gaps (release mode only)

Overall: READY / NOT READY
```

Categorize each finding as **blocking** (must fix before release) or
**advisory** (should fix, not a blocker).
