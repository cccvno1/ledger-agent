---
description: "Review code for bugs, error handling gaps, and concurrency hazards. Convention compliance is handled by instructions — this prompt focuses on what automation cannot catch."
---

Review the current changes and look for problems that automated rules miss.

## What to look for

**Logic correctness**
- Missing code paths: unhandled enum values, nil cases, empty collections.
- Wrong conditions: off-by-one, inverted boolean, short-circuit that skips side effects.
- Boundary errors: zero, max int, empty string, time.Zero.

**Error handling completeness**
- Errors returned but never checked by the caller.
- Resources acquired before an error check — will they leak on the error path?
- `defer Close()` that silently discards the error on a write path.

**Concurrency safety**
- Shared mutable state accessed without synchronization.
- Goroutines that outlive their parent context or never get cancelled.
- Channel operations that can deadlock under specific ordering.

## What NOT to review

Architecture rules, import organization, naming conventions, method signatures,
and file structure are enforced by `.github/instructions/`. Do not duplicate
that work here. If a file compiles and passes `go vet`, assume structural
conventions are met.

## Output format

For each finding:

```
file: <path>
line: <number or range>
issue: <one sentence>
severity: error | warning
fix: <concrete suggestion>
```

If no issues found, say so. Do not invent problems to fill the report.
