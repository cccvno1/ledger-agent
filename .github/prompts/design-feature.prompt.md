---
description: "Design one feature's internals — model invariants, service contracts, error taxonomy, and testing strategy."
---

Design the feature `{{name}}` using this method:

## 1. Model invariants — what must ALWAYS be true

Define the domain model in `model.go`. Beyond listing fields, state the **rules**
that must never be violated:
- Valid state transitions (if the model has a lifecycle).
- Fields that must never be zero/nil after construction.
- Relationships that must remain consistent (e.g., "a refund cannot exceed the
  original payment amount").

## 2. Service method contracts

For each method in `service.go`, specify:
- **Signature**: `(ctx context.Context, in VerbInput) (*VerbOutput, error)`
- **Behavior**: one sentence describing the happy path.
- **Error classification**: which errors are **transient** (retry may help) vs.
  **permanent** (caller must change input or give up). Use `errkit` codes.
- **What can go wrong**: one sentence on the worst realistic failure scenario
  and how the method handles it (rollback, compensate, or fail loudly).

## 3. Files needed

List every file this feature requires. Fixed:
- `model.go` — domain types + invariants
- `service.go` — business logic
- `wire.go` — dependency assembly

Additional files only if the requirement demands them (store, handler, dto,
client, worker, etc.). For each additional file, one sentence on its role.

If new database tables are needed, include the migration SQL.

## 4. Test strategy

Choose the approach that fits this feature's nature:

| Feature type | Strategy |
|-------------|----------|
| State machine | Test every valid transition + reject every invalid one |
| Algorithm / calculation | Property-based: output satisfies invariants for any valid input |
| CRUD + validation | Boundary cases: empty, max length, duplicate, not found |
| External integration | Contract tests: mock the dependency, verify request/response shape |
| Concurrent operations | Race tests: parallel calls must not corrupt state |

State which strategy applies and outline the key test cases (not full code —
just the scenario names and what each verifies).

## Output

Produce code skeletons with `// TODO` comments for business logic.
The skeletons must compile and satisfy `go vet`.
