# archfit — Development Plan

> Living document. Describes the phased implementation strategy for archfit.
> Each phase delivers a working end-to-end slice; no phase leaves the tool broken.

## North star

Deliver the CLI and skill described in `PROJECT.md` and `CLAUDE.md` without violating the repository's own architectural principles — locality, spec-first, shallow explicitness, verifiability, aggregation, reversibility, machine-readability.

The self-scan is the forcing function: if `archfit scan ./` flags archfit's own code, the change is wrong.

## Phases

### Phase 1 — Foundation and first rules (this iteration)

Goal: a working `archfit scan .` that runs against itself and produces deterministic JSON with real findings from 3–4 `strong`-evidence rules.

Deliverables:

1. **Tooling**: `go.mod`, `Makefile`, `.golangci.yaml`, `.go-arch-lint.yaml`, `.archfit.yaml`.
2. **Schemas** (spec-first — Go types track these): `schemas/rule.schema.json`, `schemas/config.schema.json`, `schemas/output.schema.json`.
3. **Core types** (`internal/model/`): `Principle`, `Severity`, `EvidenceStrength`, `Stability`, `Rule`, `Finding`, `Metric`, `Remediation`, `FactStore`.
4. **Adapter boundary** (`internal/adapter/`): `exec` (fake-able command runner), `fswrite` (contained write surface). Unused by Phase 1 resolvers but scaffolded so later collectors have somewhere to land.
5. **Collectors** (read-only fact gatherers):
   - `internal/collector/fs`: walk the repo, record file presence, sizes, line counts, globs.
   - `internal/collector/git`: sample `git log`, compute PR-size distribution stand-in.
6. **Rule engine** (`internal/rule/`): `Registry`, `Engine`, `FactStoreView`. No reflection, no `init()` registration across packages.
7. **Config** (`internal/config/`): load and validate `.archfit.yaml` against the JSON Schema.
8. **Scoring** (`internal/score/`): weight-based, normalized-per-applicable-rule scoring.
9. **Renderers** (`internal/report/`): `terminal`, `json`, `markdown`. Deterministic ordering.
10. **Core pack** (`packs/core/`) with these rules (all `strong` evidence, `experimental` stability):
    - `P1.LOC.001` — Top-level `AGENTS.md` or `CLAUDE.md` present at repo root.
    - `P2.SPC.001` — At least one executable contract (OpenAPI / JSON Schema / protobuf) if the repo declares an API surface.
    - `P4.VER.001` — Repo has a declared fast verification loop (`Makefile`, `justfile`, `Taskfile`, or `package.json` scripts) that names a `test` target.
    - `P7.MRD.001` — If the repo ships a CLI, it advertises `--json` or an equivalent structured-output contract (detected heuristically from `cmd/` or `bin/` names plus README evidence).
11. **CLI** (`cmd/archfit/`): `scan`, `score`, `explain`, `list-rules`, `list-packs`, `validate-config`. `--json`, `--format`, `-C`, `--fail-on` implemented. Stubs for `fix` / `diff` / `init` / `report` / `check` that return a clear "not implemented in 0.1.0" finding with exit code documented.
12. **Tests**: unit tests for `model`, `config`, `score`, each resolver; pack-level table tests with fixtures; exactly one end-to-end test that runs `scan` on the project itself and diffs canonicalized JSON.
13. **Self-scan cleanliness**: `archfit scan ./` exits 0 under `--fail-on=error`.
14. **Skill stub**: `.claude/skills/archfit/SKILL.md` (under 400 lines, canonical project-scope location per the Agent Skills docs) plus one remediation doc per Phase 1 rule.
15. **ADR**: `docs/adr/0001-architecture-overview.md`.
16. **Exit codes**: `docs/exit-codes.md` (contract).

**Non-goals for Phase 1**: SARIF/HTML output, auto-fix, diff mode, remote rule registry, LLM integration, the `web-saas` / `iac` / `mobile` / `data-event` / `agent-tool` packs, code generation from schemas (hand-written Go types validated against schema in tests is acceptable for Phase 1), Docker/Homebrew packaging.

### Phase 2 — CLI completion, SARIF, dogfooding pack (this iteration)

Committed scope:

1. **CLI completion**: `archfit init` (scaffold `.archfit.yaml`), `archfit check <rule-id>` (single-rule scan), `archfit report` (markdown convenience wrapper), `archfit diff <baseline.json> <current.json>` (structured baseline→current comparison with new/fixed/unchanged buckets).
2. **SARIF 2.1.0 renderer**: `--format=sarif` emits a conformant SARIF log consumable by GitHub Code Scanning. `tool.driver.rules` populated from the registry; findings mapped to `results[]` with `ruleId`, `level`, `message`, `locations`, `properties.evidence`.
3. **`agent-tool` pack — 3 rules** targeting archfit's own concerns:
   - `P2.SPC.010` — Tool ships a versioned JSON output schema (checks `schemas/output.schema.json` with a top-level `$id` and an output `schema_version`).
   - `P7.MRD.002` — Repository has a `CHANGELOG.md` at the root (supports a machine-readable change log).
   - `P7.MRD.003` — ADR directory `docs/adr/` exists when the repo has a `cmd/` binary (ADRs are how irreversible design decisions are surfaced to agents).
4. **End-to-end golden tests**: `testdata/e2e/` with at least one controlled fixture repo. The test pins the full canonicalized JSON output byte-for-byte, updated via `make update-golden`.
5. **Parse-failure surface**: a `model.ParseFailureFinding` helper emitted when a collector or resolver encounters malformed input it was asked to interpret (CLAUDE.md §13). In Phase 2 this is infrastructure — the first concrete use-site lands with YAML config parsing in Phase 3.
6. **Tooling configs**: `.golangci.yaml` (minimum-but-opinionated) and `.go-arch-lint.yaml` (encoding the boundary rule from CLAUDE.md §4 as enforceable configuration). They serve as the contract even when not executed locally.
7. **Documentation**: `CHANGELOG.md` for 0.1.0 → 0.2.0, `CONTRIBUTING.md`, `SECURITY.md`, ADR 0002 (Phase 2 decisions), updated `README.md` status table.

Deliberately deferred to later phases (each a design decision, not an oversight):

- Full YAML config (`yaml.v3`): requires a network-fetched dependency; the JSON-in-`.archfit.yaml` compromise from Phase 1 remains valid.
- `archfit fix`: per-rule auto-remediation is a large, rule-by-rule surface. Done when the `agent-tool` pack's rules stabilize at `stable`.
- `web-saas`, `iac`, `mobile`, `data-event`, `desktop` packs: each is its own Phase in its own right.
- Metrics (`context_span_p50`, `verification_latency_s`, etc.): requires the `command` and `depgraph` collectors.
- HTML renderer: deferred until SARIF is certified end-to-end.
- `ast`, `depgraph`, `command`, `schema` collectors: added when the first rule requires each.

### Phase 3 — Remaining packs and operationalization

- `iac`, `mobile`, `data-event` packs.
- Metric pipeline: `context_span_p50`, `verification_latency_s`, `invariant_coverage`, `parallel_conflict_rate`, `rollback_signal`, `blast_radius_score`.
- `archfit fix` for rules with `strong` evidence and safe auto-fixes.
- CI workflow, cross-platform release binaries, Docker image, Homebrew tap.

### Phase 4 — 1.0

- Freeze rule IDs in `core` and `web-saas`.
- JSON schema v1 certified.
- SARIF output certified against GitHub Code Scanning.
- Public API stability statement in `docs/stability.md`.

## Meta-rules that apply across all phases

- PR size budget: ≤ 500 changed lines, ≤ 5 packages per logical PR. Longer phases are split.
- Every new rule ships with fixture + `expected.json` + remediation doc + `docs/rules/<id>.md`.
- Every change runs `make lint test self-scan` before being declared done.
- No `init()` cross-package registration. No reflection-based rule discovery. Registration is explicit in `cmd/archfit/main.go`.
- No dependency added without a justification comment at the import site and an entry in `docs/dependencies.md`.
- Resolvers are pure functions of `FactStore`. Any I/O lives in `internal/collector` (read) or `internal/adapter` (write).

## Review checklist (applies to every phase)

- [ ] Schema first, code matches schema.
- [ ] Self-scan passes at `--fail-on=error`.
- [ ] `make test` under 30s, `make lint` under 5s.
- [ ] Every new rule has fixture, expected.json, table test, remediation doc.
- [ ] Deterministic output verified with `-race` and shuffled input.
- [ ] No new I/O inside `packs/*`.
- [ ] CLI flag / exit-code / JSON-schema changes documented and versioned.
