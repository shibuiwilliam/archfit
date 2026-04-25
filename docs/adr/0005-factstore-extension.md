---
id: "0005"
title: "FactStore extension for command timing and dependency graph"
status: accepted
date: 2026-04-25
tags: [architecture, factstore, metrics]
---

# ADR 0005: FactStore extension for command timing and dependency graph

## Context

archfit needs to compute metrics such as verification latency (wall-clock test
time) and blast radius (transitive dependency reach). These require facts from
the command collector (Step 8) and the dependency-graph collector (Step 9), both
of which already exist under `internal/collector/`.

The current `FactStore` interface exposes three methods: `Repo()`, `Git()`, and
`Schemas()`. Adding `Commands()` and `DepGraph()` widens the interface, which
touches every implementation (scheduler, CLI wrapper, test fakes).

The key design question is whether these new fact sources should be always
available or opt-in.

## Decision

Add two new methods to the `FactStore` interface using the same optional-return
pattern as `Git()`:

```go
Commands() (CommandFacts, bool)   // opt-in, only populated with --depth=deep
DepGraph() (DepGraphFacts, bool)  // available when source is parseable
```

Both return `(T, bool)` so that callers can distinguish "not collected" from
"collected but empty". This is the same pattern already established by
`Git() (GitFacts, bool)`.

**Command timing** is expensive (it runs `make test`, `go test`, etc.) so it is
gated behind `--depth=deep`. At `shallow` or `standard` depth, `Commands()`
returns `(zero, false)`.

**Dependency graph** collection is lightweight (parsing import statements) so it
runs unconditionally for Go projects. `DepGraph()` returns `(zero, false)` only
when the repo has no parseable source.

New model types (`CommandFacts`, `CommandResult`, `DepGraphFacts`) are thin
projections of the collector output — they carry only what resolvers and metric
functions need, not raw stdout/stderr.

## Consequences

- Every `FactStore` implementation (scheduler, CLI, test fakes) must add the two
  new methods. Fakes return `(zero, false)`.
- The `ScanInput` struct gains a `Depth` field so the scheduler knows whether to
  run the command collector.
- Metric functions in `internal/score/` can consume the new facts to compute
  `verification_latency_s`, `blast_radius_score`, and related metrics.
- Future collectors follow the same pattern: add a method to `FactStore`, update
  all implementations, gate on depth if the collector is expensive.
