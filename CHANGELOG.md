# Changelog

All notable changes to archfit are documented in this file.

The format follows [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/),
and archfit adheres to [SemVer 2.0](https://semver.org/spec/v2.0.0.html) from
1.0 onward. Pre-1.0 releases may renumber rule IDs and extend the output
schema additively; breaking changes to the CLI, exit codes, or output JSON
are called out explicitly below with migration notes.

## [Unreleased]

## [0.2.0] — 2026-04-24

### Added

- **CLI completion**: `archfit init`, `archfit check <rule-id>`,
  `archfit report`, `archfit diff <baseline.json> [current.json]`. The diff
  subcommand is the supported way to gate pull requests on new findings.
- **SARIF 2.1.0 output**: `archfit scan --format=sarif .` emits a document
  consumable by GitHub Code Scanning. Severity maps as
  info→note, warn→warning, error/critical→error.
- **`agent-tool` pack** (opt-in via `.archfit.yaml`): three `strong`-evidence
  rules at `experimental` stability:
  - `P2.SPC.010` — tool ships a versioned JSON output schema (checks
    `schemas/*.schema.json` for a top-level `$id`).
  - `P7.MRD.002` — repository has a `CHANGELOG.md` at the root.
  - `P7.MRD.003` — repository with a CLI has `docs/adr/`.
- **`internal/collector/schema`**: parses JSON Schema files under `schemas/`
  and surfaces parse errors as `SchemaFile.ParseError` for resolvers to
  convert into `ParseFailure` findings (per `CLAUDE.md` §13).
- **`model.ParseFailure`** helper — the canonical way for resolvers to
  surface malformed input as a finding rather than returning an error.
- **End-to-end golden tests** under `testdata/e2e/`. Run with `make e2e`;
  regenerate with `make update-golden`.
- **`.golangci.yaml`** and **`.go-arch-lint.yaml`** — the boundary rule
  from `CLAUDE.md` §4 encoded as enforceable configuration.
- **Documentation**: `CONTRIBUTING.md`, `SECURITY.md`, ADR 0002 covering
  Phase 2 decisions, `docs/rules/P2.SPC.010.md`, `docs/rules/P7.MRD.002.md`,
  `docs/rules/P7.MRD.003.md`, and the matching skill remediation docs.

### Changed

- `model.FactStore` gained a `Schemas() SchemaFacts` method. Downstream
  callers that implemented `FactStore` directly must add the method. The
  canonical implementation in `internal/core` already does.
- Self-scan enables both the `core` and `agent-tool` packs (7 rules total).

### Non-breaking

- The JSON output schema is unchanged at `schema_version: 0.1.0`. Additive
  evolutions (e.g., new optional fields on `target`) remain permitted within
  the current major; renames/removals will bump to `1.0.0`.

## [0.1.0] — 2026-04-24

### Added

- Initial Phase 1 release. Working end-to-end slice: `archfit scan` runs
  four `strong`-evidence rules from the `core` pack, emits terminal / JSON /
  Markdown output, and passes its own self-scan at score 100.0.
- Core pack: `P1.LOC.001`, `P1.LOC.002`, `P4.VER.001`, `P7.MRD.001`.
- CLI: `scan`, `score`, `explain`, `list-rules`, `list-packs`,
  `validate-config`, `version`.
- Collectors: `fs`, `git` (via `internal/adapter/exec`).
- JSON Schemas for rule, config, and output under `schemas/`.
- Agent skill `SKILL.md` with per-rule remediation docs.
- ADR 0001 documenting the Phase 1 architectural decisions.
- Exit codes 0/1/2/3/4 as the stability contract (see `docs/exit-codes.md`).
