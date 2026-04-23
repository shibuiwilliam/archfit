---
id: 0001
title: Architecture overview and Phase 1 boundary
status: accepted
date: 2026-04-24
tags: [architecture, foundation]
---

# ADR 0001 — Architecture overview and Phase 1 boundary

## Context

archfit evaluates whether a repository is shaped for coding agents to work
safely and quickly. It must (a) do real work, and (b) follow its own
principles — because if archfit itself drifts, its verdicts cannot be trusted.

`PROJECT.md` lists the seven principles and the full intended feature set.
`CLAUDE.md` makes the meta-consistency rule binding: "archfit must pass its
own scan at a high score."

This ADR records the foundational architecture established in Phase 1 and the
trade-offs that drove it.

## Decision

### 1. Two-layer codebase: collectors and resolvers

`internal/collector/*` packages are the **only** places that touch the
filesystem, git, or the network. They gather facts and hand them to the
scheduler.

`packs/<pack>/resolvers/*` packages are **pure functions of `FactStore`**.
They never perform I/O. Adding a new fact type requires adding a collector,
not widening a pack's capabilities.

This is archfit's own aggregation principle (P5) made concrete.

### 2. Explicit registration, no `init()` magic

The registry is populated only from `cmd/archfit/main.go` via direct calls to
each pack's `Register` function. There is no reflection, no plugin discovery,
no package-level side effect.

Adding a pack is a two-line diff in `main.go`. Removing a pack is a one-line
diff. This enforces archfit's own shallow-explicitness principle (P3).

### 3. Schema-first types

JSON Schemas under `schemas/` are the authoritative contract for rule
definitions, configuration, and output. Go types in `internal/model` and
`internal/config` track those schemas by hand in Phase 1 and via `go generate`
from Phase 2 onward (with committed output — never implicit at build time).

### 4. Config format: JSON-in-YAML for Phase 1

`.archfit.yaml` is parsed as JSON in Phase 1. YAML 1.2 is a strict superset of
JSON, so a JSON document written into `.archfit.yaml` is a valid YAML document
that any YAML-aware tooling round-trips correctly. Phase 2 adds `yaml.v3` as an
approved dependency and unlocks full YAML syntax. The decision to stay
dependency-free in Phase 1 is worth the short-term restriction.

### 5. Scoring is weight-based and normalized

`internal/score` computes scores as `1 - (weighted penalty / total weight)` per
principle and overall. Multiple findings for the same rule do not compound —
the rule's weight is fully spent at the first failure. Adding new rules to an
existing repo does not mechanically lower its score if those rules do not fire.

This preserves CLAUDE.md §13's rule: "Adding rules must not make the score
artificially go down for existing repositories."

### 6. What Phase 1 does **not** do

- SARIF / HTML output.
- `archfit fix` (auto-remediation).
- `archfit diff` (baseline comparison).
- `archfit init` (config scaffold).
- Packs other than `core`.
- LLM-assisted explanation (`--with-llm`).
- Code generation from schemas at build time.
- Network-based rule registry.

Each of the above is explicitly reserved for later phases in
`DEVELOPMENT_PLAN.md`.

## Consequences

**Positive**

- The tool does real work on its own code from day one — self-scan passes
  with 4 `strong`-evidence rules.
- The collector/resolver boundary is enforced by the package graph itself: a
  resolver that tried to reach the filesystem would have to import `os`,
  which is a visible code-review signal. Phase 2 adds `go-arch-lint` to
  enforce this by tooling.
- A contributor can add a rule without touching the engine.

**Negative**

- Phase 1's config syntax is JSON-in-`.archfit.yaml`. This may surprise users
  expecting full YAML. The constraint is documented in `docs/configuration.md`
  and lifts in Phase 2.
- The model types and the JSON Schemas are hand-kept in sync. Phase 2's
  `make generate` step will fix this.
- Only `core` pack rules exist at the Phase 1 boundary. A user looking for,
  e.g., `web-saas` checks will see `archfit list-packs` return only `core`.
  This is intentional — breadth is Phase 2's job.

## Status

Accepted. This ADR governs Phase 1 and will be revised by ADR 0002 when
Phase 2 is opened.
