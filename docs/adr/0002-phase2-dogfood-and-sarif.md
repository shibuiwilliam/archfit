---
id: 0002
title: Phase 2 — dogfooding, SARIF, and CLI completion
status: accepted
date: 2026-04-24
tags: [architecture, cli, output, packs]
---

# ADR 0002 — Phase 2: dogfooding, SARIF, and CLI completion

## Context

ADR 0001 established the Phase 1 architecture: collectors, resolvers, an
explicit registry, three renderers, and a `core` pack of four rules. Phase 2
had to widen the surface without compromising the principles Phase 1 paid
for in design effort.

Three pressures shaped the Phase 2 scope:

1. **Consumer-facing CI integration**: `PROJECT.md` promises SARIF output for
   GitHub Code Scanning and a `diff` mode for tracking repositories over time.
   Neither was present at the end of Phase 1.
2. **Self-dogfooding**: archfit is an agent-tool, but the Phase 1 self-scan
   only exercised `core`. A pack that encoded the agent-tool–specific
   concerns (schema versioning, changelog, ADR discipline) would both be
   useful to consumers *and* pull archfit itself toward better shape.
3. **Boundary enforcement**: `CLAUDE.md` §4 declares a strict package
   graph — packs cannot import collectors or adapters. Phase 1 enforced this
   by convention; Phase 2 had to encode it.

## Decision

### 1. CLI surface completed to match `CLAUDE.md` §8

`archfit init`, `check`, `report`, and `diff` are now real subcommands.
`fix` remains deferred because its design (per-rule auto-remediation) is
substantial and unlocks more value once rules stabilize.

`diff` uses a `(rule_id, path, message)` identity for each finding. That is
the weakest identity that remains stable across scans — it does not depend
on confidence scores, evidence blobs, or ordering. A finding whose message
changes becomes a `Fixed`+`New` pair, which is arguably correct: the old
claim was retracted, a new claim was made.

### 2. SARIF 2.1.0 output is a first-class renderer

SARIF has a large spec surface. We emit only what GitHub Code Scanning
actually consumes plus what is strictly required by validators: `$schema`,
`version`, `runs[].tool.driver.{name,version,informationUri,rules[]}`, and
`runs[].results[]` with `ruleId`, `level`, `message`, `locations`, and
`properties`.

Severity maps as `info→note`, `warn→warning`, `error/critical→error`.
`critical` collapses to `error` because SARIF has no four-level scale and
GitHub dashboards treat them identically.

The SARIF renderer takes the list of *rules that ran*, not the full
registry, so the `tool.driver.rules[]` reflects only the evaluated set. This
is important for the `check <rule-id>` case where a single-rule run should
not advertise the rest of the pack.

### 3. `agent-tool` pack — opt-in, archfit enables it on itself

Three rules at `strong` / `experimental` / `warn`:

- `P2.SPC.010` — versioned JSON output schema.
- `P7.MRD.002` — `CHANGELOG.md` at root.
- `P7.MRD.003` — `docs/adr/` exists when a CLI ships.

These only apply when the consumer declares `project_type: [agent-tool]` or
enables the pack explicitly in `.archfit.yaml`. Running them by default
would generate noise on repositories that never intended to ship a tool.

### 4. The `schema` collector — the first step beyond `fs`+`git`

`P2.SPC.010` needed to inspect schema file content. The cleanest way
without widening pack capabilities was to add a `schema` collector that
produces `SchemaFacts` the resolver consumes. A tempting shortcut — letting
the resolver call `os.ReadFile` directly — would have violated the pack
boundary. The collector also surfaces parse errors as `SchemaFile.ParseError`
which resolvers convert into `ParseFailure` findings per `CLAUDE.md` §13.

This is the first use of the "collector recorded a problem → resolver
decides whether it's a finding" pattern. The same shape will apply to
future YAML-config parse failures.

### 5. Boundary as configuration

`.go-arch-lint.yaml` now encodes the rule "packs may depend on `model` and
`rule` and nothing else I/O-adjacent." When the tool is installed in CI, it
fails on violations. The config is the contract; whether CI runs it in
Phase 2 is not essential — the contract being machine-checkable is.

### 6. Output JSON schema stays at `0.1.0`

All Phase 2 additions are additive to the scan-side output (they do not
change `findings[]` or `scores{}` shape). SARIF is a separate document with
its own schema. `schema_version` bumps to `1.0.0` when Phase 3 freezes the
contract.

## Consequences

**Positive**

- GitHub Code Scanning is now a supported integration out of the box.
- archfit dogfoods its own rules; the self-scan score (7 rules, 100.0) is a
  non-trivial claim.
- The package boundary is visible to static analysis, not just to reviewers.

**Negative**

- `FactStore` gained a method (`Schemas()`), a minor compatibility event for
  anyone implementing the interface directly. Documented in the CHANGELOG.
- The `schema` collector reads file contents during collection; we accept
  this complexity to keep resolvers pure. The alternative (lazy read-through
  collectors) was rejected as premature.
- The SARIF renderer's signature (`RenderSARIF` rather than a case in the
  generic `Render` switch) is slightly awkward — SARIF needs the rule list,
  generic `Render` does not. We chose clarity at the call site over
  uniformity; documented at the top of `sarif.go`.

## Status

Accepted. Phase 3 opens with either the metrics pipeline
(`context_span_p50`, `verification_latency_s`) or the next pack
(`web-saas`), to be decided by the next iteration's priorities.
