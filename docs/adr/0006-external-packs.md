---
id: "0006"
title: "External pack validation and SDK"
status: accepted
date: 2026-04-25
tags: [architecture, packs, validation]
---

# ADR 0006: External pack validation and SDK

## Context

archfit organises rules into packs. Each pack is a vertical slice with a
well-defined structure: `AGENTS.md`, `INTENT.md`, a Go file exposing
`Register()` and `Rules()`, a `resolvers/` directory, and a `fixtures/`
directory with at least one test fixture. Optional but recommended items
include a `rules/` directory with YAML rule definitions and a `context.yaml`
metadata file.

As the number of packs grows and external contributors create their own, we
need a machine-checkable contract for what constitutes a valid pack. Without
this, malformed packs surface as confusing import-time or runtime errors
rather than clear validation messages.

## Decision

Introduce a `validate-pack` CLI command and a supporting `internal/packman`
package that checks whether a directory satisfies the pack structure contract.

### Pack structure requirements

A directory is a valid archfit pack when it contains:

1. **AGENTS.md** — describes the pack for coding agents (required).
2. **INTENT.md** — states the pack's purpose and scope (required).
3. **At least one `.go` file** in the pack root (required). This file is
   expected to expose `Register()` and `Rules()` functions, though the
   validator checks file presence only — it does not import or parse Go code.
4. **`resolvers/` directory** — contains the resolver functions (required).
5. **`fixtures/` directory** with at least one subdirectory containing an
   `input/` directory (required). Fixtures are the primary correctness bar
   for pack tests.

Recommended but not required:

- `rules/` directory with YAML rule definitions.
- `context.yaml` for pack metadata.

### Validation without importing

The `validate-pack` command checks structure by inspecting the filesystem. It
does **not** import or compile Go code. This means it can run against packs
that target a different Go version or have unresolved dependencies. The
trade-off is that it cannot verify function signatures — that remains the job
of `go build` and `go vet`.

### CLI surface

```
archfit validate-pack <path>    check pack structure
```

Exit code 0 when the pack is valid, exit code 1 when validation errors exist.
Warnings (missing optional items) do not affect the exit code.

## Consequences

- External pack authors get fast feedback on structural compliance without
  needing to wire the pack into the main binary.
- CI pipelines can gate on `archfit validate-pack` before attempting to build
  or test a new pack.
- The validator is intentionally conservative: it checks presence, not
  content. Content validation (schema conformance of YAML rules, resolver
  function signatures) is left to existing tools (`go build`, JSON Schema
  validation, pack tests).
