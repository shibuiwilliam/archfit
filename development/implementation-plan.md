# Implementation Plan

This document breaks the remaining development work into PR-sized units. Each step must pass `make lint test self-scan` before merge. Steps within a phase are sequential; phases can partially overlap.

## Phase A — Fix Engine Enhancements

### A.1 — Fix Conflict Resolution

**Status**: Not started
**Packages**: `internal/fix/`
**Effort**: ~200 lines

When `--all` is used and multiple fixers target files that share content (e.g., both P1.LOC.001 and P7.MRD.001 want to create docs), the engine must detect conflicts and either merge changes or skip the later fixer with a warning.

- Add conflict detection to `engine.go`: if two fixers produce `ActionCreate` for the same path, the second is skipped with a log message
- Add table tests for conflict scenarios
- Update `FixResult` to include `Skipped []SkippedFix` with reason

### A.2 — Disk-Backed Fix Audit Log

**Status**: Not started
**Packages**: `internal/fix/`
**Effort**: ~150 lines

Currently `log.go` exists but audit entries are not persisted across runs.

- Implement `AppendLog()` that writes to `.archfit-fix-log.json`
- Each entry: timestamp, rule ID, changes applied, verification result, rollback status
- Add `--no-log` flag to skip writing (for CI where the log is noise)
- Log file is append-only JSON Lines format

### A.3 — Richer LLM Fixer Prompts

**Status**: Not started
**Packages**: `internal/fix/llmfix/`
**Effort**: ~200 lines

Current LLM fixers use minimal prompts. Enrich with:
- Repository language composition from `RepoFacts.Languages`
- Existing file structure for context (e.g., when generating AGENTS.md, include the slice's file listing)
- Project type from `.archfit.yaml`
- Prompt templates in `internal/fix/llmfix/prompts.go`

## Phase B — Metrics and CI Completion

### B.1 — TypeScript/Python Dependency Graph

**Status**: Not started
**Packages**: `internal/collector/depgraph/`
**Effort**: ~300 lines

The depgraph collector currently supports Go only. Add:
- `typescript.go`: parse `import` statements from `.ts`/`.tsx`/`.js`/`.jsx` files using regex (not a full parser)
- `python.go`: parse `import` and `from ... import` from `.py` files
- Dispatcher in `collector.go` that selects parser based on `RepoFacts.Languages`
- Tests with fixture files under `internal/collector/depgraph/testdata/`

### B.2 — GitHub Action PR Comment Integration

**Status**: Scaffolded (`.github/archfit-action/`)
**Packages**: `.github/archfit-action/`
**Effort**: ~200 lines

Complete the GitHub Action:
- `entrypoint.sh`: install archfit, run scan on PR branch, run scan on base branch (cached), run diff, post PR comment via `gh pr comment`
- Handle edge cases: no baseline (first scan), no changes (skip comment), fork PRs (no comment permission)
- PR comment format: score delta table, new findings, fixed findings
- SARIF upload step using `github/codeql-action/upload-sarif@v3`

### B.3 — Trend Visualization Export

**Status**: Not started
**Packages**: `cmd/archfit/main.go`
**Effort**: ~100 lines

Extend `archfit trend` with:
- `--format=html`: generate a standalone HTML file with a score-over-time chart (inline SVG, no JS dependencies)
- Per-principle breakdown in the trend data

## Phase C — Ecosystem Expansion

### C.1 — Remote Pack Installation

**Status**: Not started
**Packages**: `internal/packman/`
**Effort**: ~400 lines

Implement `archfit pack install <module>`:
- Generate `archfit.plugins.go` with Go imports for external packs
- Rebuild the binary with the external pack compiled in
- Add pack to `.archfit.yaml` under `packs.external`
- `archfit pack remove <module>`: remove import and rebuild
- Validate pack structure before installation

ADR required: `docs/adr/0006-external-packs.md` (exists, may need update).

### C.2 — Cross-Stack Detection Expansion

**Status**: Not started (see `development/cross-stack-improvements.md`)
**Packages**: `packs/core/resolvers/`
**Effort**: ~200 lines across 3 PRs

Three targeted PRs:
1. P4.VER.001: add Java/Ruby/PHP/Elixir/Scala build tool detection
2. P1.LOC.002: expand slice container list + make configurable
3. P7.MRD.001: widen CLI detection beyond `cmd/` and `bin/`

### C.3 — Community Pack Registry (post-1.0)

**Status**: Design phase
**Effort**: Large

A curated index of community packs, similar to Homebrew taps. Design decisions:
- Registry is a Git repo with metadata files (not a database)
- `archfit pack search` queries the registry
- `archfit pack publish` submits a PR to the registry
- Quality gate: must pass `archfit validate-pack`

## Phase D — Fitness Contract as Code

See [fitness-contract.md](./fitness-contract.md) for full design.

### D.1 — Contract Schema and Types (Steps 2.1-2.2)

**Status**: Done
**Packages**: `internal/contract/`

- `Contract` type with hard constraints, soft targets, area budgets, agent directives
- `Check()` pure function evaluating scan results against contract
- 14 table tests covering all constraint types, scopes, and edge cases
- JSON Schema at `schemas/contract.schema.json`

### D.2 — CLI Wiring (Step 2.3)

**Status**: Not started
**Packages**: `cmd/archfit/`, `internal/contract/`
**Effort**: ~250 lines

- `archfit contract check [path]` — check against contract (exit 0/1/5)
- `archfit contract status [path]` — show contract dashboard
- `archfit contract init [path]` — scaffold `.archfit-contract.yaml`
- ADR `docs/adr/0008-contract-exit-codes.md` for new exit code 5

### D.3 — Agent Directive Support (Step 2.4)

**Status**: Not started
**Packages**: `.claude/skills/archfit/`
**Effort**: ~100 lines

- Contract-aware workflow in SKILL.md
- Reference doc: `.claude/skills/archfit/reference/contract-workflow.md`

## Phase E — Agent Behavior Observatory

See [agent-observatory.md](./agent-observatory.md) for full design.

### E.1 — Trace Schema and Types (Step 1.1)

**Status**: Not started
**Packages**: `internal/observer/`
**Effort**: ~150 lines

### E.2 — Behavioral Metrics (Step 1.2)

**Status**: Not started
**Packages**: `internal/observer/`
**Effort**: ~200 lines

### E.3 — Hotspot Analysis (Step 1.3)

**Status**: Not started
**Packages**: `internal/observer/`
**Effort**: ~150 lines

### E.4 — CLI Wiring (Step 1.4)

**Status**: Not started
**Packages**: `cmd/archfit/`
**Effort**: ~200 lines
**ADR**: `docs/adr/0007-agent-behavior-observatory.md`

## Phase F — Adaptive Rule Engine

See [adaptive-engine.md](./adaptive-engine.md) for full design.

### F.1 — Fix Outcome Tracking (Step 3.1)

**Status**: Not started
**Packages**: `internal/fix/`
**Effort**: ~100 lines

### F.2 — Adaptive Confidence (Step 3.2)

**Status**: Not started
**Packages**: `internal/adaptive/`
**Effort**: ~200 lines

### F.3 — Threshold Adaptation (Step 3.3)

**Status**: Not started
**Packages**: `internal/adaptive/`
**Effort**: ~150 lines
**ADR**: `docs/adr/0009-adaptive-thresholds.md`

### F.4 — CLI Wiring (Step 3.4)

**Status**: Not started
**Packages**: `cmd/archfit/`
**Effort**: ~100 lines

## Implementation Order

```
A.1 (fix conflicts)      ──→ A.2 (audit log) ──→ A.3 (LLM prompts)
B.1 (TS/Py depgraph)     ──→ B.2 (GH Action)  ──→ B.3 (trend viz)
C.2 (cross-stack)        ──→ C.1 (remote packs)

D.1 (contract types) ✓   ──→ D.2 (contract CLI) ──→ D.3 (skill)
E.1 (trace types)         ──→ E.2 (metrics) ──→ E.3 (hotspots) ──→ E.4 (CLI)
F.1 (fix tracking)        ──→ F.2 (confidence) ──→ F.3 (thresholds) ──→ F.4 (CLI)

A-C can run in parallel.
D starts immediately (contract types done).
E can start in parallel with D.
F benefits from running after D and E have been in use.
```

## Definition of Done (per step)

- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] `make self-scan` exits 0
- [ ] PR ≤ 500 lines, ≤ 5 packages
- [ ] New types have ≥ 3 table-test cases
- [ ] No new deps without justification
- [ ] ADR filed when required
