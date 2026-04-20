---
description: "Break down a business requirement into a feature map, surface unknowns, and identify capability gaps before writing code."
---

Analyze the requirement using this method:

## 1. Decompose — what exists, what happens

Identify the core **entities** (nouns with identity and lifecycle) and **behaviors**
(verbs that change state or produce output). For each entity, state its lifecycle
stages. For each behavior, state who triggers it and what it changes.

## 2. Draw boundaries — one feature per bounded context

Group entities and behaviors into `internal/<name>/` feature packages.
For each feature, list:
- Fixed files: `model.go`, `service.go`, `wire.go`
- Additional files the feature actually needs (store, handler, client, worker, etc.)
- Dependencies on other features (injected via interface at wire time)

Do NOT assume all features need HTTP, database, or DTOs. Let the requirement decide.

## 3. Check framework capability

For each infrastructure need, state whether the project already supports it
(HTTP, PostgreSQL, Redis, etc.) or whether it requires manual extension
(gRPC, WebSocket, message queue, object storage, etc.).

Flag any gap clearly: "**Requires manual extension**: [what] — framework does not
provide a module/runtime for this."

## 4. Surface implicit assumptions

For each key behavior, answer this question:

> "What does this behavior **assume** about time, state, ordering, identity,
> or visibility? Under what conditions does that assumption break?"

Examples of what this question uncovers:
- A payment assumes idempotency — breaks if the same request is processed twice.
- An auction assumes a time boundary — breaks if the scheduler fires late.
- A query assumes tenant isolation — breaks if a filter is missing.
- A cache assumes freshness — breaks if the source changes without notification.

Do NOT enumerate a fixed checklist. Derive assumptions from the specific requirement.

## 5. Output

Produce:
1. **Feature map** — table of features, their files, and dependencies.
2. **Infrastructure gaps** — what the framework supports vs. what needs manual work.
3. **Open questions** — things that must be answered before implementation starts.
   For each question, suggest a default if reasonable.
4. **Implementation order** — which features to build first, based on dependencies.
