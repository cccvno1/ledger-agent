---
description: "Systematically diagnose a runtime issue — classify the symptom, isolate the cause, fix, and prevent recurrence."
---

Diagnose the reported issue using this method:

## 1. Classify the symptom

| Category | Signals |
|----------|---------|
| Startup failure | Process exits immediately, config/wire error in logs |
| Request failure | HTTP 4xx/5xx, unexpected response body |
| Background job failure | Worker error, stuck job, missed schedule |
| Panic | Stack trace with goroutine dump |
| Performance | Slow response, high memory, connection exhaustion, goroutine leak |

State which category this issue falls into. If unclear, list the top two candidates.

## 2. Isolate

Before tracing code, narrow the scope:
- **Environment**: does it reproduce locally? Only in staging? Only under load?
- **Timing**: always, intermittent, or only after a recent change?
- **Blast radius**: one endpoint, one feature, or everything?

## 3. Trace to root cause

Follow the error wrapping chain (`fmt.Errorf("pkg: op: %w", err)`) from the
symptom back to the origin. At each hop, check:
- Is the input valid? (config, request, dependent data)
- Is the dependency reachable? (database, cache, external API)
- Is there a concurrency issue? (race, deadlock, context cancelled too early)

For **performance** issues: check connection pool metrics, query plans
(`EXPLAIN ANALYZE`), goroutine counts, and allocation profiles.

## 4. Fix

Propose the **smallest change** that resolves the root cause.
- Show the exact file and location.
- If config: show the corrected value.
- If code: show the diff.

## 5. Prevent

- Add a test that reproduces this failure.
- If the failure came from a missing validation, add the validation.
- If the failure came from unclear conventions, note what documentation
  should be updated (but do not update it now — fix the bug first).
