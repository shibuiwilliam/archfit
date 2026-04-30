# Changelog

All notable changes to archfit are documented in this file.

The format follows [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/),
and archfit adheres to [SemVer 2.0](https://semver.org/spec/v2.0.0.html) from
1.0 onward. Pre-1.0 releases may renumber rule IDs and extend the output
schema additively; breaking changes to the CLI, exit codes, or output JSON
are called out explicitly below with migration notes.

## [Unreleased]

### Changed

- **Output schema bumped to `0.2.0`**: added `summary.rules_with_findings`
  (integer) to `schemas/output.schema.json`. This field was already emitted
  by the renderer but was not declared in the schema, causing strict validation
  to reject all archfit output. Also declared the optional
  `findings[].llm_suggestion` object for `--with-llm` runs.
  See [ADR 0009](./docs/adr/0009-output-schema-rules-with-findings.md).

### Added

- **Schema conformance test** (`internal/report/schema_test.go`): validates
  every `expected.json` golden file against `schemas/output.schema.json` at
  build time. Prevents future drift between renderer and schema.
- New dependency: `github.com/santhosh-tekuri/jsonschema/v6` (pure-Go JSON
  Schema validator, test-only usage). Documented in `docs/dependencies.md`.

## [0.3.0] — 2026-04-24

### Added

- **LLM-assisted explanation (`--with-llm`)**: opt-in enrichment of `scan`,
  `check`, and `explain` with Google Gemini. Produces a short,
  evidence-specific follow-up to each finding's static remediation.
  Requires `GOOGLE_API_KEY` (or `GEMINI_API_KEY`) in the environment.
  Documented in [`docs/llm.md`](./docs/llm.md) and
  [ADR 0003](./docs/adr/0003-llm-explanation.md).
- **`internal/adapter/llm/`**: the single network boundary for LLM calls.
  Exposes a provider-agnostic `Client` interface with three implementations:
  `Real` (backed by `google.golang.org/genai`), `Fake` (tests), plus
  `Cached` and `Budget` decorators for cost control.
- **`--llm-budget N`** flag (default `5`) caps the number of LLM calls per
  run. In-memory response cache makes repeated prompts free within a run.
- **`Finding.LLMSuggestion`** — optional field emitted only when
  `--with-llm` is used. Purely additive; `schema_version` stays `0.1.0`.
  SARIF results include it under `properties.llm_suggestion`.

### Changed

- **Go toolchain minimum: `1.24`** (was `1.23`). Required by
  `google.golang.org/genai v1.54.0`. Noted in `docs/dependencies.md` and
  ADR 0003. Cross-compile targets unchanged.
- **First non-stdlib runtime dependency**: `google.golang.org/genai`.
  Used only inside `internal/adapter/llm/real.go`; every other package
  depends on the local `llm.Client` interface, never on the SDK.

### Non-breaking

- `archfit scan .` without `--with-llm` is byte-identical to 0.2.0. The
  end-to-end golden tests under `testdata/e2e/` continue to pin the
  non-LLM output.
- LLM failures (missing key with `--with-llm` set, network error, budget
  exhausted mid-run) never fail the scan — static remediation is the
  fallback and base exit codes are preserved.

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
