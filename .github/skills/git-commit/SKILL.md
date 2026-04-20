---
name: git-commit
description: "Commit changes following Conventional Commits with pre-commit checks. Use when: git commit, commit changes, save progress, 提交代码."
argument-hint: "Optional: describe what changed, or leave blank to auto-detect"
---

# Git Commit

Analyze changes, run pre-commit checks, and commit using Conventional Commits.

## Procedure

### 1. Gather Changes

Run `git status` and `git diff --stat` (include `--cached` for staged).
Classify changed files into logical units — a logical unit serves one purpose.

### 2. Commit Strategy

| Scenario | Action |
|----------|--------|
| One logical unit | Single commit |
| Unrelated concerns | Split into multiple commits |
| Source + test for same feature | Single commit |
| Formatting / config only | Separate commit (`style` / `chore`) |

### 3. Pre-Commit Checks

Before each commit, run checks relevant to changed files:

```bash
go vet ./...
go test -count=1 ./...
```

If checks fail: **stop and report**. Do NOT commit broken code.

### 4. Compose Commit Message

```
<type>(<scope>): <subject>

[optional body]
```

#### Type

| Type | When |
|------|------|
| `feat` | New feature, new API |
| `fix` | Bug fix |
| `refactor` | Neither fix nor feature |
| `test` | Tests only |
| `docs` | Documentation only |
| `chore` | Build, deps, config |

#### Scope

Derive from the primary area: feature package name (`order`, `user`),
infrastructure area (`boot`, `conf`, `migration`), or cross-cutting (`deps`).

#### Subject

- Imperative mood, lowercase, no period, max 72 chars.
- Describe *what* changed: `add user creation endpoint`.

#### Body (non-trivial changes)

- Explain *why* the change was made.
- Wrap at 80 characters.

### 5. Execute

```bash
git add <files>
git commit -m "<type>(<scope>): <subject>"
```

### 6. Verify

```bash
git log --oneline -3
```
