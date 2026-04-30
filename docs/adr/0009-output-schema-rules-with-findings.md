---
id: 0009
title: Add rules_with_findings to output schema and bump to 0.2.0
status: accepted
date: 2026-04-30
---

## Context

The renderer in `internal/report/report.go` emits `summary.rules_with_findings`
in JSON output. However, `schemas/output.schema.json` declares
`additionalProperties: false` on the `summary` object and does not list the
field. Any strict JSON Schema validation of archfit output therefore fails.

This was tracked as CLAUDE.md §7.2 and PROJECT.md §3.2.

## Decision

1. Add `rules_with_findings` (integer, minimum 0) to `summary.properties` in
   `schemas/output.schema.json` and mark it as required.
2. Add `llm_suggestion` (optional object) to `findings.items.properties` to
   cover the `--with-llm` enrichment that was also undeclared.
3. Bump `OutputSchemaVersion` from `0.1.0` to `0.2.0`. Per the pre-1.0
   stability rules (CLAUDE.md §12), additive field changes are a minor bump.
4. Add a CI-runnable schema conformance test (`internal/report/schema_test.go`)
   that validates every golden `expected.json` against the schema. This
   prevents future drift.
5. Use `github.com/santhosh-tekuri/jsonschema/v6` as the validator — a
   pure-Go, zero-dependency JSON Schema implementation supporting draft
   2020-12. This was already listed in `docs/dependencies.md` as planned.

## Consequences

- Every current and future `expected.json` golden file must validate against
  the schema. Adding a field to the renderer without updating the schema will
  break the build.
- Consumers of `--json` output that parse `schema_version` should accept
  `0.2.0`. The only change is the new `rules_with_findings` field in
  `summary` — no fields were renamed, removed, or retyped.
- One new test-time dependency: `github.com/santhosh-tekuri/jsonschema/v6`.
  It has zero transitive dependencies of its own.
