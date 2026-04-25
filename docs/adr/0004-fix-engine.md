---
id: 0004
title: Agent-in-the-loop remediation engine (archfit fix)
status: accepted
date: 2026-04-25
tags: [architecture, fix, remediation, agent]
---

# ADR 0004 — Agent-in-the-loop remediation engine (`archfit fix`)

## Context

archfit diagnoses architectural fitness issues but does not fix them. The
`archfit fix` command closes the scan-fix-verify loop: it applies
deterministic file changes that resolve findings, then re-scans to prove
the fix worked. This is *verifiable remediation* — a property no other
architecture checker currently offers.

Two classes of fixes exist:

1. **Static fixers** — deterministic file scaffolding for `strong`-evidence
   rules. Example: P1.LOC.001 creates a `CLAUDE.md` from a template.
2. **LLM-assisted fixers** — context-dependent content generation that wraps
   a static fixer and enriches its output via the `llm.Client` adapter.
   Falls back to the static template when the LLM is unavailable.

## Decision

### 1. Scan → Fix → Verify loop

The `FixEngine` orchestrates three phases:

1. **Plan**: for each targeted finding, call the matching `Fixer.Plan()` to
   get proposed `[]Change` (file creations/modifications).
2. **Apply**: write changes to disk, snapshotting original content first.
3. **Verify**: re-scan using the injected `Scanner`. If the finding persists
   or new findings appear, roll back all changes.

The engine receives a `Scanner func(ctx) (ScanResult, error)` — it never
imports collectors or adapters directly. This keeps `internal/fix/` decoupled
from I/O concerns.

### 2. Fixer interface

```go
type Fixer interface {
    RuleID() string
    Plan(ctx context.Context, finding model.Finding, facts model.FactStore) ([]Change, error)
    NeedsLLM() bool
}
```

Static fixers return `NeedsLLM() == false` and are safe for `--all` without
confirmation. LLM fixers return `true` and default to plan-only mode.

### 3. Safety model

- Static fixers: auto-apply with `--dry-run` option.
- LLM fixers: require `--with-llm`; show plan first by default.
- All fixes are atomic: rolled back if verify shows regressions.
- Fix history is appended to `.archfit-fix-log.json` for audit.

### 4. Registration

Fixers are registered explicitly in `cmd/archfit/main.go` via
`buildFixEngine()`, following the same pattern as `buildRegistry()`.
No reflection, no `init()`.

## Consequences

**Positive**
- archfit becomes the first tool that doesn't just diagnose agent-readiness
  but *creates* it in a single command.
- The verify step ensures fixes are correct — no silent regressions.
- Static fixers are deterministic and testable without API keys.

**Negative**
- The engine writes files to disk — this is intentional (the fix *is* the
  file change), but it means `archfit fix` is a mutating command unlike
  `scan`.
- Rollback relies on in-memory snapshots; if the process is killed during
  apply, partial changes may remain. This is acceptable for a CLI tool.

## Status

Accepted.
